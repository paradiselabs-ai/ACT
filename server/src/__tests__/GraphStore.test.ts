/**
 * GraphStore + GraphIndexer Unit Tests
 *
 * Covers the four properties the coordination graph is built on:
 * derivation rules, invalidate-never-delete, point-in-time queries,
 * and rebuild-from-log equivalence.
 */

import fs from 'fs/promises';
import path from 'path';
import { GraphStore, nodeKey, parseNodeKey, isVisibleAt, GraphEdge } from '../services/GraphStore.js';
import { GraphIndexer, eventKey } from '../services/GraphIndexer.js';
import { ChronologicalLog } from '../services/ChronologicalLog.js';
import { CoordinationMessage } from '../types/coordination.js';

const TEST_DATA_DIR = './test-data-graph';
const TEST_LOG_PATH = path.join(TEST_DATA_DIR, 'test-coordination.jsonl');
const TEST_EDGE_PATH = path.join(TEST_DATA_DIR, 'test-graph-edges.jsonl');

const PROJECT = 'demo-app';
const TASK = 'task-1';

function ev(partial: Partial<CoordinationMessage> & { type: string; timestamp: string }): CoordinationMessage {
  return {
    agent: 'system',
    message: partial.type,
    ...partial
  } as CoordinationMessage;
}

// The canonical lifecycle: create → assign → reassign → complete → submit →
// verdict, plus a file claim/release pair.
function lifecycleEvents(): CoordinationMessage[] {
  return [
    ev({
      timestamp: '2026-08-01T00:00:00.000Z',
      agent: 'planner',
      type: 'task_created',
      data: { id: TASK, title: 'Build the thing', metadata: { projectName: PROJECT } }
    }),
    ev({
      timestamp: '2026-08-01T01:00:00.000Z',
      agent: 'dev-1',
      type: 'task_assigned',
      data: { taskId: TASK, agentId: 'dev-1' }
    }),
    ev({
      timestamp: '2026-08-01T02:00:00.000Z',
      agent: 'dev-1',
      type: 'file_claim',
      data: { filePaths: ['src/app.ts'], projectName: PROJECT, agentId: 'dev-1', taskId: TASK }
    }),
    ev({
      timestamp: '2026-08-01T03:00:00.000Z',
      agent: 'dev-2',
      type: 'task_assigned',
      data: { taskId: TASK, agentId: 'dev-2' }
    }),
    ev({
      timestamp: '2026-08-01T04:00:00.000Z',
      agent: 'dev-1',
      type: 'file_release',
      data: { filePaths: ['src/app.ts'], projectName: PROJECT, agentId: 'dev-1', taskId: TASK }
    }),
    ev({
      timestamp: '2026-08-01T05:00:00.000Z',
      agent: 'dev-2',
      type: 'task_completed',
      data: { taskId: TASK, agentId: 'dev-2', success: true, result: 'shipped' }
    }),
    ev({
      timestamp: '2026-08-01T06:00:00.000Z',
      agent: 'dev-2',
      type: 'task_submitted_for_validation',
      data: { taskId: TASK, agentId: 'dev-2' }
    }),
    ev({
      timestamp: '2026-08-01T07:00:00.000Z',
      agent: 'assurance',
      type: 'task_validated',
      data: { taskId: TASK, agentId: 'assurance', score: 97, passed: true }
    })
  ];
}

/** Ingestion-time fields are wall-clock, so compare only the derived facts. */
function adjacency(store: GraphStore): string[] {
  return store
    .allEdges()
    .map(e => [e.src, e.rel, e.dst, e.fact, e.validAt ?? '', e.invalidAt ?? '', e.episodeKeys.join(',')].join(' :: '))
    .sort();
}

async function freshStore(suffix = ''): Promise<GraphStore> {
  const store = new GraphStore({ jsonlPath: TEST_EDGE_PATH.replace('.jsonl', `${suffix}.jsonl`) });
  await store.initialize();
  return store;
}

async function logWith(events: CoordinationMessage[]): Promise<ChronologicalLog> {
  const log = new ChronologicalLog({ jsonlPath: TEST_LOG_PATH, bufferSize: 1000, flushIntervalMs: 60000 });
  await log.initialize();
  for (const e of events) await log.append(e);
  return log;
}

describe('GraphStore', () => {
  beforeEach(async () => {
    await fs.rm(TEST_DATA_DIR, { recursive: true, force: true });
  });

  afterEach(async () => {
    await fs.rm(TEST_DATA_DIR, { recursive: true, force: true });
  });

  describe('node keys', () => {
    it('splits on the first colon so file paths and timestamps survive', () => {
      expect(parseNodeKey('file:src/a:b.ts')).toEqual({ key: 'file:src/a:b.ts', type: 'file', name: 'src/a:b.ts' });
      expect(parseNodeKey('task:t1')).toEqual({ key: 'task:t1', type: 'task', name: 't1' });
    });

    it('rejects unknown node types', async () => {
      expect(parseNodeKey('service:foo')).toBeNull();
      const store = await freshStore();
      await expect(
        store.addEdge({ src: 'service:foo', rel: 'x', dst: 'task:t1', fact: 'f', episodeKeys: ['k'] })
      ).rejects.toThrow(/invalid node key/);
    });
  });

  describe('derivation rules', () => {
    let store: GraphStore;
    let log: ChronologicalLog;

    beforeEach(async () => {
      store = await freshStore();
      log = await logWith(lifecycleEvents());
      await new GraphIndexer(log, store).indexAllEvents();
    });

    afterEach(async () => {
      await log.close();
    });

    it('task_created → (agent)-[created]->(task) and (task)-[belongs_to]->(project)', () => {
      const created = store.findEdges({ src: nodeKey('agent', 'planner'), rel: 'created' });
      expect(created).toHaveLength(1);
      expect(created[0].dst).toBe(nodeKey('task', TASK));
      expect(created[0].fact).toContain('Build the thing');
      expect(created[0].validAt).toBe('2026-08-01T00:00:00.000Z');

      const belongs = store.findEdges({ src: nodeKey('task', TASK), rel: 'belongs_to' });
      expect(belongs.map(e => e.dst)).toEqual([nodeKey('project', PROJECT)]);
    });

    it('task_assigned → (task)-[assigned_to]->(agent)', () => {
      const assigned = store.findEdges({ src: nodeKey('task', TASK), rel: 'assigned_to' });
      expect(assigned.map(e => e.dst).sort()).toEqual([nodeKey('agent', 'dev-1'), nodeKey('agent', 'dev-2')]);
    });

    it('task_completed → (agent)-[completed]->(task)', () => {
      const completed = store.findEdges({ rel: 'completed' });
      expect(completed).toHaveLength(1);
      expect(completed[0].src).toBe(nodeKey('agent', 'dev-2'));
      expect(completed[0].dst).toBe(nodeKey('task', TASK));
    });

    it('task_failed → (agent)-[failed]->(task)', async () => {
      const failStore = await freshStore('-fail');
      const failLog = await logWith([
        ev({
          timestamp: '2026-08-02T00:00:00.000Z',
          agent: 'dev-9',
          type: 'task_failed',
          data: { taskId: 'task-9', agentId: 'dev-9', success: false, result: 'boom' }
        })
      ]);
      await new GraphIndexer(failLog, failStore).indexAllEvents();
      await failLog.close();

      const failed = failStore.findEdges({ rel: 'failed' });
      expect(failed).toHaveLength(1);
      expect(failed[0].src).toBe(nodeKey('agent', 'dev-9'));
      expect(failed[0].fact).toContain('boom');
    });

    it('submit-for-validation → (task)-[submitted]->(verdict:pending-*)', () => {
      const submitted = store.findEdges({ src: nodeKey('task', TASK), rel: 'submitted' });
      expect(submitted).toHaveLength(1);
      expect(submitted[0].dst).toBe(nodeKey('verdict', `pending-${TASK}`));
    });

    it('validation verdict → (verdict)-[judged]->(task) with pass/fail in fact', () => {
      const judged = store.findEdges({ rel: 'judged' });
      expect(judged).toHaveLength(1);
      expect(judged[0].dst).toBe(nodeKey('task', TASK));
      expect(judged[0].fact).toMatch(/^pass score 97 by assurance/);
      expect(judged[0].src.startsWith('verdict:')).toBe(true);
    });

    it('file claim/release → (agent)-[holds]->(file), released hold goes inactive', () => {
      const holds = store.findEdges({ rel: 'holds' });
      expect(holds).toHaveLength(1);
      expect(holds[0].dst).toBe(nodeKey('file', 'src/app.ts'));
      expect(store.activeEdges({ rel: 'holds' })).toHaveLength(0);
      expect(holds[0].invalidAt).toBe('2026-08-01T04:00:00.000Z');
    });

    it('stamps every edge with the reference key of its source event', () => {
      const created = store.findEdges({ rel: 'created' })[0];
      expect(created.episodeKeys).toEqual([
        eventKey({ timestamp: '2026-08-01T00:00:00.000Z', agent: 'planner', type: 'task_created' })
      ]);
    });

    it('carries causedBy into episodeKeys when the emitter sets it', async () => {
      const causalStore = await freshStore('-causal');
      const cause = eventKey({ timestamp: '2026-08-03T00:00:00.000Z', agent: 'planner', type: 'task_created' });
      const causalLog = await logWith([
        ev({
          timestamp: '2026-08-03T01:00:00.000Z',
          agent: 'dev-3',
          type: 'task_assigned',
          data: { taskId: 'task-3', agentId: 'dev-3' },
          causedBy: cause
        })
      ]);
      await new GraphIndexer(causalLog, causalStore).indexAllEvents();
      await causalLog.close();

      expect(causalStore.findEdges({ rel: 'assigned_to' })[0].episodeKeys).toContain(cause);
    });
  });

  describe('invalidation', () => {
    let store: GraphStore;
    let log: ChronologicalLog;

    beforeEach(async () => {
      store = await freshStore();
      log = await logWith(lifecycleEvents());
      await new GraphIndexer(log, store).indexAllEvents();
    });

    afterEach(async () => {
      await log.close();
    });

    it('stamps expiredAt/invalidAt without removing the edge', () => {
      const toDev1 = store.findEdges({ src: nodeKey('task', TASK), rel: 'assigned_to', dst: nodeKey('agent', 'dev-1') });
      expect(toDev1).toHaveLength(1);
      expect(toDev1[0].invalidAt).toBe('2026-08-01T03:00:00.000Z');
      expect(typeof toDev1[0].expiredAt).toBe('string');
      // Still present in adjacency, just no longer believed.
      expect(store.getEdge(toDev1[0].id)).toBeDefined();
      expect(store.activeEdges({ src: nodeKey('task', TASK), rel: 'assigned_to' }).map(e => e.dst))
        .toEqual([nodeKey('agent', 'dev-2')]);
    });

    it('invalidates the pending verdict marker when a verdict lands', () => {
      const submitted = store.findEdges({ rel: 'submitted' })[0];
      expect(submitted.invalidAt).toBe('2026-08-01T07:00:00.000Z');
    });

    it('appends invalidation records instead of rewriting edge lines', async () => {
      const content = await fs.readFile(TEST_EDGE_PATH, 'utf-8');
      const records = content.trim().split('\n').map(l => JSON.parse(l));
      const invalidations = records.filter((r: any) => typeof r.invalidates === 'string');
      const edgeLines = records.filter((r: any) => typeof r.id === 'string');

      expect(invalidations.length).toBe(3); // dev-1 assignment, file hold, pending verdict
      // Original edge lines are untouched: none of them carries a stamp on disk.
      for (const line of edgeLines) {
        expect(line.invalidAt).toBeUndefined();
        expect(line.expiredAt).toBeUndefined();
      }
    });

    it('replays invalidations when the edge file is reloaded', async () => {
      const reloaded = new GraphStore({ jsonlPath: TEST_EDGE_PATH });
      await reloaded.initialize();
      expect(reloaded.getEdgeCount()).toBe(store.getEdgeCount());
      expect(reloaded.activeEdges({ rel: 'holds' })).toHaveLength(0);
      expect(reloaded.activeEdges({ src: nodeKey('task', TASK), rel: 'assigned_to' }).map(e => e.dst))
        .toEqual([nodeKey('agent', 'dev-2')]);
    });

    it('is idempotent — a second invalidation neither re-stamps nor re-appends', async () => {
      const edge = store.findEdges({ rel: 'holds' })[0];
      const before = (await fs.readFile(TEST_EDGE_PATH, 'utf-8')).trim().split('\n').length;
      await store.invalidateEdge(edge.id, '2026-08-01T09:00:00.000Z');
      const after = (await fs.readFile(TEST_EDGE_PATH, 'utf-8')).trim().split('\n').length;

      expect(edge.invalidAt).toBe('2026-08-01T04:00:00.000Z');
      expect(after).toBe(before);
    });
  });

  describe('point-in-time queries', () => {
    let store: GraphStore;
    let log: ChronologicalLog;

    beforeEach(async () => {
      store = await freshStore();
      log = await logWith(lifecycleEvents());
      await new GraphIndexer(log, store).indexAllEvents();
    });

    afterEach(async () => {
      await log.close();
    });

    it('returns the superseded edge before the reassignment and the new one after', () => {
      const owners = (at: string) =>
        store.findEdges({ src: nodeKey('task', TASK), rel: 'assigned_to', at }).map(e => e.dst);

      expect(owners('2026-08-01T02:00:00.000Z')).toEqual([nodeKey('agent', 'dev-1')]);
      expect(owners('2026-08-01T06:00:00.000Z')).toEqual([nodeKey('agent', 'dev-2')]);
    });

    it('hides edges whose validAt is in the future of the query time', () => {
      expect(store.findEdges({ src: nodeKey('task', TASK), rel: 'assigned_to', at: '2026-08-01T00:30:00.000Z' }))
        .toHaveLength(0);
    });

    it('treats the invalidation instant as exclusive and the valid instant as inclusive', () => {
      const hold = store.findEdges({ rel: 'holds' })[0];
      expect(isVisibleAt(hold, '2026-08-01T02:00:00.000Z')).toBe(true);  // validAt boundary
      expect(isVisibleAt(hold, '2026-08-01T04:00:00.000Z')).toBe(false); // invalidAt boundary
      expect(isVisibleAt(hold, undefined)).toBe(true);                    // no filter = full history
    });

    it('expands two hops from an agent to the task and its project', () => {
      const oneHop = store.neighborhood(nodeKey('agent', 'dev-2'), { hops: 1 });
      expect(oneHop.node).toEqual({ key: 'agent:dev-2', type: 'agent', name: 'dev-2' });
      expect(oneHop.edges.some(e => e.dst === nodeKey('project', PROJECT))).toBe(false);

      const twoHops = store.neighborhood(nodeKey('agent', 'dev-2'), { hops: 2 });
      expect(twoHops.edges.some(e => e.dst === nodeKey('project', PROJECT))).toBe(true);
      expect(new Set(twoHops.edges.map(e => e.id)).size).toBe(twoHops.edges.length); // no dupes
    });
  });

  describe('rebuild equivalence', () => {
    it('full replay produces the same adjacency as incremental indexing', async () => {
      const events = lifecycleEvents();

      // Incremental: drain the log in three passes as events arrive.
      const incrementalStore = await freshStore('-incremental');
      const log = await logWith([]);
      const incremental = new GraphIndexer(log, incrementalStore);
      for (const batch of [events.slice(0, 3), events.slice(3, 6), events.slice(6)]) {
        for (const e of batch) await log.append(e);
        await incremental.indexNewEvents();
      }

      // Rebuild: fresh store, one full replay of the same log.
      const rebuiltStore = await freshStore('-rebuilt');
      await new GraphIndexer(log, rebuiltStore).indexAllEvents();
      await log.close();

      expect(adjacency(rebuiltStore)).toEqual(adjacency(incrementalStore));
      expect(rebuiltStore.stats().edges).toBe(incrementalStore.stats().edges);
      expect(rebuiltStore.stats().nodes).toBe(incrementalStore.stats().nodes);
      expect(rebuiltStore.stats().activeEdges).toBe(incrementalStore.stats().activeEdges);
    });

    it('re-deriving the same events adds nothing (idempotent addEdge)', async () => {
      const store = await freshStore();
      const log = await logWith(lifecycleEvents());
      const indexer = new GraphIndexer(log, store);

      await indexer.indexAllEvents();
      const first: GraphEdge[] = store.allEdges();
      await indexer.indexAllEvents();
      await log.close();

      expect(store.allEdges()).toHaveLength(first.length);
      expect(adjacency(store)).toEqual(first
        .map(e => [e.src, e.rel, e.dst, e.fact, e.validAt ?? '', e.invalidAt ?? '', e.episodeKeys.join(',')].join(' :: '))
        .sort());
    });
  });
});
