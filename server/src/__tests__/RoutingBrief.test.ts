/**
 * RoutingBrief aggregation tests
 *
 * Verifies getProjectOutcomes() derives per-project composition + pass/kickback
 * counts from the event log, and getRoutingBrief() emits confidence-labeled
 * per-role and role-pair evidence. embed() is stubbed so the test doesn't load
 * the real model — the aggregation reads message fields, not vectors.
 */

import { LocalEmbeddingVectorStore } from '../services/LocalEmbeddingVectorStore.js';
import { PVMIndexer } from '../services/PVMIndexer.js';
import { CoordinationMessage } from '../types/coordination.js';

// Each event needs a unique (timestamp, agent) — store() keys points on
// `${timestamp}_${agent}`, so a shared stamp would collapse them into one point.
let _seq = 0;
function ev(partial: Partial<CoordinationMessage> & { type: string }): CoordinationMessage {
  _seq++;
  return {
    timestamp: new Date(Date.UTC(2026, 0, 1, 0, 0, _seq)).toISOString(),
    agent: 'system',
    message: '',
    ...partial,
  } as CoordinationMessage;
}

describe('RoutingBrief aggregation', () => {
  let store: LocalEmbeddingVectorStore;

  beforeEach(async () => {
    store = new LocalEmbeddingVectorStore();
    // Stub embedding so store() is fast + offline; vectors are irrelevant here.
    (store as any).embed = async () => new Array(384).fill(0);

    const events: CoordinationMessage[] = [
      // Project "alpha": composition 1×developer + 1×backend_dev, 1 pass / 1 kickback
      ev({ type: 'agent_registered', projectName: 'alpha', data: { agentId: 'dev-1', capabilities: ['node', 'javascript'] } } as any),
      ev({ type: 'agent_registered', projectName: 'alpha', data: { agentId: 'backend-1', capabilities: ['api', 'db'] } } as any),
      ev({ type: 'task_created', projectName: 'alpha', data: { taskId: 'A1', requiredCapabilities: ['node'] } } as any),
      ev({ type: 'task_assigned', projectName: 'alpha', data: { taskId: 'A1', agentId: 'dev-1' } } as any),
      ev({ type: 'task_validated', projectName: 'alpha', data: { taskId: 'A1' } } as any),
      ev({ type: 'task_created', projectName: 'alpha', data: { taskId: 'A2', requiredCapabilities: ['api'] } } as any),
      ev({ type: 'task_assigned', projectName: 'alpha', data: { taskId: 'A2', agentId: 'backend-1' } } as any),
      ev({ type: 'task_validation_failed', projectName: 'alpha', data: { taskId: 'A2', gaps: 'missing auth check' } } as any),

      // Project "beta": composition 3×developer, 3 pass / 0 kickback
      ev({ type: 'agent_registered', projectName: 'beta', data: { agentId: 'dev-1', capabilities: ['node'] } } as any),
      ev({ type: 'agent_registered', projectName: 'beta', data: { agentId: 'dev-2', capabilities: ['node'] } } as any),
      ev({ type: 'agent_registered', projectName: 'beta', data: { agentId: 'dev-3', capabilities: ['node'] } } as any),
      ev({ type: 'task_assigned', projectName: 'beta', data: { taskId: 'B1', agentId: 'dev-1' } } as any),
      ev({ type: 'task_validated', projectName: 'beta', data: { taskId: 'B1' } } as any),
      ev({ type: 'task_assigned', projectName: 'beta', data: { taskId: 'B2', agentId: 'dev-2' } } as any),
      ev({ type: 'task_validated', projectName: 'beta', data: { taskId: 'B2' } } as any),
      ev({ type: 'task_assigned', projectName: 'beta', data: { taskId: 'B3', agentId: 'dev-3' } } as any),
      ev({ type: 'task_validated', projectName: 'beta', data: { taskId: 'B3' } } as any),
    ];
    for (const e of events) await store.store(e);
  });

  afterEach(async () => { await store.close(); });

  it('derives per-project composition + pass/kickback counts', () => {
    const outcomes = store.getProjectOutcomes();
    const alpha = outcomes.find(o => o.project === 'alpha')!;
    const beta = outcomes.find(o => o.project === 'beta')!;

    expect(alpha.composition).toEqual({ developer: 1, backend_dev: 1 });
    expect(alpha.taskCount).toBe(2);
    expect(alpha.passed).toBe(1);
    expect(alpha.kickbacks).toBe(1);
    expect(alpha.perRole.developer).toEqual({ tasks: 1, passed: 1 });
    expect(alpha.perRole.backend_dev).toEqual({ tasks: 1, passed: 0 });
    expect(alpha.topGaps[0]).toContain('missing auth');

    expect(beta.composition).toEqual({ developer: 3 });
    expect(beta.taskCount).toBe(3);
    expect(beta.passed).toBe(3);
    expect(beta.kickbacks).toBe(0);
  });

  it('emits confidence-labeled per-role + role-pair evidence', async () => {
    const brief = await store.getRoutingBrief('node cli tool', ['node']);

    // per-role: developer ran 4 tasks (1 alpha + 3 beta), all passed
    const dev = brief.perRole.find((r: any) => r.role === 'developer');
    expect(dev.tasks).toBe(4);
    expect(dev.passRate).toBe(1);
    expect(dev.signal).toBe('low'); // <5 samples

    // role-pair from alpha: developer + backend_dev ran together
    const pair = brief.rolePairs.find((p: any) => p.pair === 'backend_dev + developer');
    expect(pair).toBeDefined();
    expect(pair.kickbacks).toBe(1);

    // text block carries signal labels + the failure note
    expect(brief.text).toContain('signal');
    expect(brief.text).toContain('Per-role track record');
    expect(brief.text).toContain('missing auth');
  });
});

// Outcome -> worker attribution. Real history has task_completed on every
// completion but a payload on only a fraction of task_assigned events, so
// completion is authoritative and assignment is the fallback.
describe('outcome attribution', () => {
  const mkStore = async (events: CoordinationMessage[]) => {
    const store = new LocalEmbeddingVectorStore();
    (store as any).embed = async () => new Array(384).fill(0);
    for (const e of events) await store.store(e);
    return store;
  };

  it('attributes a validation via task_completed when task_assigned has no payload', async () => {
    const store = await mkStore([
      ev({ type: 'agent_registered', projectName: 'gamma', data: { agentId: 'authsvc-backend-1', capabilities: ['api', 'postgres'] } } as any),
      ev({ type: 'task_assigned', projectName: 'gamma', data: { taskId: 'G1' } } as any),
      ev({ type: 'task_completed', projectName: 'gamma', data: { taskId: 'G1', agentId: 'authsvc-backend-1' } } as any),
      ev({ type: 'task_validated', projectName: 'gamma', data: { taskId: 'G1' } } as any),
    ]);
    const gamma = store.getProjectOutcomes().find(o => o.project === 'gamma')!;
    // Project-prefixed id misses the prefix table; capabilities classify it.
    expect(gamma.perRole.backend_dev).toEqual({ tasks: 1, passed: 1 });
    expect(gamma.perRole.unknown).toBeUndefined();
    expect(store.getAttributionStats()).toMatchObject({ validations: 1, unattributed: 0 });
    await store.close();
  });

  it('counts a validation with no resolvable worker instead of dropping it silently', async () => {
    const store = await mkStore([
      ev({ type: 'task_validated', projectName: 'delta', data: { taskId: 'D1' } } as any),
    ]);
    const delta = store.getProjectOutcomes().find(o => o.project === 'delta')!;
    expect(delta.perRole.unknown).toEqual({ tasks: 1, passed: 1 });
    expect(store.getAttributionStats()).toMatchObject({ validations: 1, unattributed: 1 });
    await store.close();
  });
});

// Agent profile capability buckets. The vocabulary is the agent's registered
// capabilities and nothing else — ChronLog event-type names used to leak in
// ("coordination: 0% success over 460 tasks").
describe('agent profile capability buckets', () => {
  const build = async () => {
    const store = new LocalEmbeddingVectorStore();
    (store as any).embed = async () => new Array(384).fill(0);
    const events: CoordinationMessage[] = [
      ev({ type: 'agent_registered', projectName: 'eps', data: { agentId: 'be-1', capabilities: ['api', 'postgres', 'react'] } } as any),
      ev({ type: 'coordination', agent: 'be-1', projectName: 'eps', message: 'chatter', data: {} } as any),
      ev({ type: 'progress_report', agent: 'be-1', projectName: 'eps', message: 'halfway', data: {} } as any),
      ev({ type: 'task_created', projectName: 'eps', data: { id: 'E1', requiredCapabilities: ['api'] } } as any),
      ev({ type: 'task_assigned', projectName: 'eps', data: { taskId: 'E1', agentId: 'be-1' } } as any),
      ev({ type: 'task_completed', projectName: 'eps', data: { taskId: 'E1', agentId: 'be-1' } } as any),
      ev({ type: 'task_validated', projectName: 'eps', data: { taskId: 'E1' } } as any),
      // Second task: no assignment record, so its duration is unknowable.
      ev({ type: 'task_created', projectName: 'eps', data: { id: 'E2', requiredCapabilities: ['postgres'] } } as any),
      ev({ type: 'task_completed', projectName: 'eps', data: { taskId: 'E2', agentId: 'be-1' } } as any),
      ev({ type: 'task_validation_failed', projectName: 'eps', data: { taskId: 'E2', gaps: 'no migration' } } as any),
    ];
    for (const e of events) await store.store(e);
    return store;
  };

  it('keys buckets by registered capability only — no event-type keys', async () => {
    const store = await build();
    const profile = await store.getAgentProfile('be-1');
    expect(Object.keys(profile.capabilities).sort()).toEqual(['api', 'postgres']);
    expect(profile.capabilities['coordination']).toBeUndefined();
    expect(profile.capabilities['progress_report']).toBeUndefined();
    expect(profile.capabilities['agent_registered']).toBeUndefined();
    // 'react' is registered but no task required it — no evidence, no bucket.
    expect(profile.capabilities['react']).toBeUndefined();
    expect(profile.capabilities['api'].successRate).toBe(1);
    expect(profile.capabilities['postgres'].successRate).toBe(0);
    await store.close();
  });

  it('omits avgCompletionTime when no assignment timestamp exists', async () => {
    const store = await build();
    const profile = await store.getAgentProfile('be-1');
    expect(profile.capabilities['api']).toHaveProperty('avgCompletionTime');
    expect(profile.capabilities['postgres']).not.toHaveProperty('avgCompletionTime');
    await store.close();
  });
});

// The indexer resolves the project bucket for outcome events that carry no tag
// of their own (legacy history) from the taskId -> project map it learns while
// indexing. extractProjectName itself stays a pure function of one event.
describe('PVMIndexer project resolution', () => {
  const mkIndexer = (events: any[]) => {
    const store = new LocalEmbeddingVectorStore({ sidecarPath: './test-data-indexer/pvm-vectors.jsonl' } as any);
    (store as any).embed = async () => new Array(384).fill(0);
    const log = { query: async () => ({ events }) } as any;
    return { store, indexer: new PVMIndexer(log, store) };
  };

  const legacyEvents = [
    { timestamp: '2026-01-01T00:00:01Z', agent: 'system', type: 'task_created', message: 'task created: L1',
      data: { id: 'L1', metadata: { projectName: 'legacy-proj' }, requiredCapabilities: ['api'] } },
    // Untagged, exactly as the old emit sites wrote them:
    { timestamp: '2026-01-01T00:00:02Z', agent: 'be-1', type: 'task_completed', message: 'task completed: L1',
      data: { taskId: 'L1', agentId: 'be-1' } },
    { timestamp: '2026-01-01T00:00:03Z', agent: 'assurance', type: 'task_validated', message: 'task validated: L1',
      data: { taskId: 'L1' } },
  ];

  it('buckets untagged outcome events under the owning project', async () => {
    const { store, indexer } = mkIndexer(legacyEvents);
    await indexer.indexAllEvents();
    const projects = store.getAllPoints().map(p => (p.message as any).projectName);
    expect(projects).toEqual(['legacy-proj', 'legacy-proj', 'legacy-proj']);
    expect(indexer.getStatus().taggedTaskCount).toBe(1);
    await store.close();
  });

  it('is deterministic across a full re-index (replay rebuilds the map)', async () => {
    const { store, indexer } = mkIndexer(legacyEvents);
    await indexer.indexAllEvents();
    await indexer.indexAllEvents();
    const projects = store.getAllPoints().map(p => (p.message as any).projectName);
    expect(projects).toEqual(['legacy-proj', 'legacy-proj', 'legacy-proj']);
    await store.close();
  });

  it('leaves events with no task and no project in __global__', async () => {
    const { store, indexer } = mkIndexer([
      { timestamp: '2026-01-01T00:00:04Z', agent: 'dev-1', type: 'coordination', message: 'general chatter', data: {} },
    ]);
    await indexer.indexAllEvents();
    expect((store.getAllPoints()[0].message as any).projectName).toBe('__global__');
    await store.close();
  });
});
