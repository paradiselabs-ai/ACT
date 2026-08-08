/**
 * GraphIndexer - derives coordination-graph edges from ChronologicalLog events
 *
 * Mirrors PVMIndexer: constructed with the log plus its target store, drains
 * new events on an interval. Derivation is RULE-BASED — no LLM calls, no
 * heuristics beyond the event shapes the server already writes. Every derived
 * edge carries the reference key(s) of the event(s) that produced it, so the
 * whole graph is re-derivable from the log alone.
 *
 * Events have no id field, so the reference key is "<timestamp>|<agent>|<type>".
 */

import { ChronologicalLog } from './ChronologicalLog';
import { GraphStore, nodeKey } from './GraphStore';
import { extractProjectName } from './PVMIndexer';
import { logger } from '../utils/logger';

/** Reference key for an event — events carry no id of their own. */
export function eventKey(event: any): string {
  return `${event?.timestamp}|${event?.agent ?? 'system'}|${event?.type}`;
}

// Verdict node names embed the verdict's event time so repeated validation
// rounds on one task produce distinct verdict nodes.
function verdictName(taskId: string, timestamp: string): string {
  return `${taskId}-${String(timestamp).replace(/[^0-9]/g, '')}`;
}

function truncate(text: unknown, max: number): string {
  const s = typeof text === 'string' ? text : '';
  return s.length > max ? `${s.slice(0, max)}…` : s;
}

export class GraphIndexer {
  private chronologicalLog: ChronologicalLog;
  private graphStore: GraphStore;
  private lastIndexedTimestamp: string | null = null;
  private lastIndexedKey: string | null = null;
  private indexingInterval: NodeJS.Timeout | null = null;
  private isIndexing: boolean = false;

  constructor(chronologicalLog: ChronologicalLog, graphStore: GraphStore) {
    this.chronologicalLog = chronologicalLog;
    this.graphStore = graphStore;
  }

  /**
   * Start background derivation of graph edges
   * @param intervalMs - How often to check for new events (default: 10 seconds)
   */
  startIndexing(intervalMs: number = 10000): void {
    if (this.indexingInterval) {
      logger.warn('GraphIndexer already running, stopping previous instance');
      this.stopIndexing();
    }

    logger.info(`🕸  GraphIndexer started - deriving edges every ${intervalMs}ms`);

    this.indexingInterval = setInterval(() => {
      this.indexNewEvents().catch(err => {
        logger.error(`GraphIndexer failed to derive edges: ${err.message}`);
      });
    }, intervalMs);
  }

  stopIndexing(): void {
    if (this.indexingInterval) {
      clearInterval(this.indexingInterval);
      this.indexingInterval = null;
      logger.info('🛑 GraphIndexer stopped');
    }
  }

  /** Derive edges from events appended since the last pass. */
  async indexNewEvents(): Promise<void> {
    if (this.isIndexing) {
      logger.debug('GraphIndexer already deriving, skipping this cycle');
      return;
    }
    this.isIndexing = true;

    try {
      // ChronologicalLog.query filters `since` with >=, so bump by 1ms to avoid
      // re-reading the last seen event (same trick as PVMIndexer).
      const query: any = { limit: 100000 };
      if (this.lastIndexedTimestamp) {
        query.since = new Date(new Date(this.lastIndexedTimestamp).getTime() + 1).toISOString();
      }

      const { events } = await this.chronologicalLog.query(query);
      if (events.length === 0) return;

      const before = this.graphStore.getEdgeCount();
      for (const event of events) {
        await this.deriveFrom(event);
      }
      this.advanceCursor(events);

      const added = this.graphStore.getEdgeCount() - before;
      logger.info(`🕸  GraphIndexer processed ${events.length} events (+${added} edges)`);
    } catch (error: any) {
      logger.error(`❌ GraphIndexer failed during derivation: ${error.message}`);
      throw error;
    } finally {
      this.isIndexing = false;
    }
  }

  /**
   * Replay the entire event log through the derivation rules. This is also the
   * rebuild path: against a fresh (or removed) graph-edges.jsonl it reproduces
   * the whole projection. Safe to run over a populated store — GraphStore.addEdge
   * is idempotent on (src, rel, dst, episode key).
   */
  async indexAllEvents(): Promise<void> {
    const { events } = await this.chronologicalLog.query({ limit: 100000 });
    if (events.length === 0) {
      logger.info('🕸  GraphIndexer: no events to derive from');
      return;
    }

    for (const event of events) {
      await this.deriveFrom(event);
    }
    this.advanceCursor(events);

    const stats = this.graphStore.stats();
    logger.info(`🕸  GraphIndexer replayed ${events.length} events → ${stats.edges} edges / ${stats.nodes} nodes`);
  }

  getStatus(): {
    isRunning: boolean;
    isIndexing: boolean;
    lastIndexedKey: string | null;
    lastIndexedTimestamp: string | null;
  } {
    return {
      isRunning: !!this.indexingInterval,
      isIndexing: this.isIndexing,
      lastIndexedKey: this.lastIndexedKey,
      lastIndexedTimestamp: this.lastIndexedTimestamp
    };
  }

  private advanceCursor(events: any[]): void {
    let latest = events[0];
    for (const e of events) {
      if (e.timestamp > latest.timestamp) latest = e;
    }
    if (this.lastIndexedTimestamp === null || latest.timestamp > this.lastIndexedTimestamp) {
      this.lastIndexedTimestamp = latest.timestamp;
    }
    this.lastIndexedKey = eventKey(events[events.length - 1]);
  }

  /**
   * The derivation rules. Each case maps one event type to edges; `validAt` is
   * the event's own timestamp (when the fact became true), while GraphStore
   * stamps `createdAt` with ingestion time.
   */
  private async deriveFrom(event: any): Promise<void> {
    const data = event?.data;
    if (!data) return;

    const key = eventKey(event);
    // causedBy, when an emitter sets it, links the derived edges back to the
    // event that triggered this one (both keys land in episodeKeys).
    const episodeKeys = event.causedBy ? [key, event.causedBy] : [key];
    const validAt = event.timestamp;
    const actor = event.agent || 'system';
    const store = this.graphStore;

    switch (event.type) {
      case 'task_created': {
        // task_created carries the whole task object as data.
        const taskId = data.id || data.taskId;
        if (!taskId) return;
        const title = truncate(data.title || data.description || taskId, 80);

        await store.addEdge({
          src: nodeKey('agent', actor),
          rel: 'created',
          dst: nodeKey('task', taskId),
          fact: `${actor} created task ${taskId}: ${title}`,
          episodeKeys,
          validAt
        });

        const projectName = extractProjectName(event);
        if (projectName !== '__global__') {
          await store.addEdge({
            src: nodeKey('task', taskId),
            rel: 'belongs_to',
            dst: nodeKey('project', projectName),
            fact: `task ${taskId} belongs to project ${projectName}`,
            episodeKeys,
            validAt
          });
        }
        return;
      }

      case 'task_assigned': {
        const taskId = data.taskId;
        const agentId = data.agentId;
        if (!taskId || !agentId) return;

        const src = nodeKey('task', taskId);
        const dst = nodeKey('agent', agentId);
        // Reassignment supersedes rather than overwrites: the previous holder's
        // edge is stamped so "who owned this task at time T" stays answerable.
        for (const prior of store.activeEdges({ src, rel: 'assigned_to' })) {
          if (prior.dst !== dst) await store.invalidateEdge(prior.id, validAt);
        }

        await store.addEdge({
          src,
          rel: 'assigned_to',
          dst,
          fact: `task ${taskId} assigned to ${agentId}`,
          episodeKeys,
          validAt
        });
        return;
      }

      case 'task_completed':
      case 'task_failed': {
        const taskId = data.taskId;
        if (!taskId) return;
        const agentId = data.agentId || actor;
        // The server splits completion into task_completed / task_failed, but
        // pre-split logs signalled failure via data.success === false.
        const failed = event.type === 'task_failed' || data.success === false;

        await store.addEdge({
          src: nodeKey('agent', agentId),
          rel: failed ? 'failed' : 'completed',
          dst: nodeKey('task', taskId),
          fact: `${agentId} ${failed ? 'failed' : 'completed'} task ${taskId}${data.result ? `: ${truncate(data.result, 120)}` : ''}`,
          episodeKeys,
          validAt
        });
        return;
      }

      case 'task_submitted_for_validation': {
        const taskId = data.taskId;
        if (!taskId) return;

        await store.addEdge({
          src: nodeKey('task', taskId),
          rel: 'submitted',
          dst: nodeKey('verdict', `pending-${taskId}`),
          fact: `task ${taskId} submitted for validation by ${data.agentId || actor}`,
          episodeKeys,
          validAt
        });
        return;
      }

      case 'task_validated':
      case 'task_validation_failed': {
        const taskId = data.taskId;
        if (!taskId) return;
        const passed = event.type === 'task_validated' && data.passed !== false;
        const judge = data.agentId || actor;

        // The pending marker stops being true the moment a verdict lands.
        for (const pending of store.activeEdges({
          src: nodeKey('task', taskId),
          rel: 'submitted',
          dst: nodeKey('verdict', `pending-${taskId}`)
        })) {
          await store.invalidateEdge(pending.id, validAt);
        }

        const scoreText = data.score !== undefined ? ` score ${data.score}` : '';
        const gapsText = data.gaps ? ` gaps: ${truncate(data.gaps, 120)}` : '';
        await store.addEdge({
          src: nodeKey('verdict', verdictName(taskId, validAt)),
          rel: 'judged',
          dst: nodeKey('task', taskId),
          fact: `${passed ? 'pass' : 'fail'}${scoreText} by ${judge}${gapsText}`,
          episodeKeys,
          validAt
        });
        return;
      }

      case 'file_claim': {
        const agentId = data.agentId || actor;
        const paths: string[] = Array.isArray(data.filePaths) ? data.filePaths : [];
        for (const filePath of paths) {
          await store.addEdge({
            src: nodeKey('agent', agentId),
            rel: 'holds',
            dst: nodeKey('file', filePath),
            fact: `${agentId} holds ${filePath}${data.taskId ? ` for task ${data.taskId}` : ''}`,
            episodeKeys,
            validAt
          });
        }
        return;
      }

      case 'file_release': {
        const agentId = data.agentId || actor;
        const paths: string[] = Array.isArray(data.filePaths) ? data.filePaths : [];
        for (const filePath of paths) {
          for (const hold of store.activeEdges({
            src: nodeKey('agent', agentId),
            rel: 'holds',
            dst: nodeKey('file', filePath)
          })) {
            await store.invalidateEdge(hold.id, validAt);
          }
        }
        return;
      }
    }
  }
}

export default GraphIndexer;
