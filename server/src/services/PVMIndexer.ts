import { ChronologicalLog } from './ChronologicalLog';
import { VectorMemoryStore } from './VectorMemoryStore';
import { CoordinationMessage } from '../types/coordination';
import { logger } from '../utils/logger';

// classifyScope tags events as "meta" (harness/tooling state — CLI behavior, server
// errors, runner issues) or "project" (real coordination work). Meta-scope events
// are excluded from default PVM search so stale "CLI broken" claims from a prior
// session don't feed back into new agent context after the bug is fixed.
const META_TEXT_PATTERNS = [
  /\bact\s+task\b/i,
  /\bact\s+files\b/i,
  /\bact\s+cli\b/i,
  /\bact-agent\b/i,
  /\bact-runner\b/i,
  /\bact_cli\b/i,
  /Unknown command/i,
  /HTTP 5\d\d\b/,
  /server error/i,
  /\.getTime is not a function/i,
  /TERMINAL_STATE_TRANSITION/,
  /subcommand (un)?available/i
];
const META_EVENT_TYPES = new Set<string>([
  'server_error', 'runner_error', 'cli_error', 'harness_diagnostic'
]);

function classifyScope(eventType: string | undefined, text: string): 'project' | 'meta' {
  if (eventType && META_EVENT_TYPES.has(eventType)) return 'meta';
  for (const re of META_TEXT_PATTERNS) {
    if (re.test(text)) return 'meta';
  }
  return 'project';
}

export function extractProjectName(event: any): string {
  const d = event?.data;
  if (!d) return '__global__';
  // Order: most specific to least. task.metadata covers task lifecycle events;
  // metadata.projectName covers Planner-emitted project context; projectName
  // covers direct agent messages. d.name is ONLY a project name when the
  // event itself is the project_created announcement — otherwise d.name is
  // typically an agent display name (e.g., "Developer", "Backend Dev") and
  // must NOT be treated as a project.
  const candidate =
    d.task?.metadata?.projectName ||
    d.metadata?.projectName ||
    d.projectName ||
    (event?.type === 'project_created' ? d.name : undefined);
  return (typeof candidate === 'string' && candidate.length > 0)
    ? candidate
    : '__global__';
}

/**
 * Build the text that gets embedded for a given ChronLog event. The default
 * (event.message) is bare for typed lifecycle events — e.g., a task_created
 * event's message is just "task created: <uuid>", which has no semantic
 * content to embed. The richer content lives in event.data.title /
 * description / task.description / task.title. Concatenate them so a search
 * for "markdown tree CLI" actually matches the typed event for that task,
 * not just its coordination-duplicate.
 *
 * Falls back to event.message alone when no enrichable data fields exist.
 */
export function buildEmbeddingText(event: any): string {
  const base = event?.message || (event as any)?.content || '';
  const d = event?.data || {};
  const parts: string[] = [];
  if (base) parts.push(base);
  if (typeof d.title === 'string' && d.title) parts.push(d.title);
  if (typeof d.description === 'string' && d.description) parts.push(d.description);
  if (d.task) {
    if (typeof d.task.title === 'string' && d.task.title) parts.push(d.task.title);
    if (typeof d.task.description === 'string' && d.task.description) parts.push(d.task.description);
  }
  if (typeof d.techStack === 'string' && d.techStack) parts.push(d.techStack);
  if (typeof d.constraints === 'string' && d.constraints) parts.push(d.constraints);
  if (typeof d.successCriteria === 'string' && d.successCriteria) parts.push(d.successCriteria);
  const joined = parts.join(' — ');
  return joined.length > 0 ? joined : 'Unknown event';
}

/** Join key for task lifecycle events — same precedence the analytics joins use. */
export function taskIdOfEvent(event: any): string | undefined {
  const d = event?.data;
  if (!d) return undefined;
  const id = d.taskId || d.task?.id || d.id;
  return typeof id === 'string' && id.length > 0 ? id : undefined;
}

export class PVMIndexer {
  private chronologicalLog: ChronologicalLog;
  private vectorStore: VectorMemoryStore;
  private lastIndexedTimestamp: string | null = null;
  private indexingInterval: NodeJS.Timeout | null = null;
  private isIndexing: boolean = false;
  // taskId -> owning project, learned from whichever of a task's events carries
  // the tag (task_created always does). Legacy outcome events written before the
  // emit sites were tagged carry no project of their own; without this map they
  // stay in __global__, which getRoutingBrief excludes from evidence.
  private taskProjectMap: Map<string, string> = new Map();

  constructor(chronologicalLog: ChronologicalLog, vectorStore: VectorMemoryStore) {
    this.chronologicalLog = chronologicalLog;
    this.vectorStore = vectorStore;
  }

  /**
   * Learn taskId -> project from every task event that already carries a tag.
   * Restricted to `task_*` events so a non-task `data.id` (e.g. a project
   * record) can never claim a task's slot.
   */
  private learnTaskProjects(events: any[]): void {
    for (const event of events) {
      if (typeof event?.type !== 'string' || !event.type.startsWith('task_')) continue;
      const project = extractProjectName(event);
      if (project === '__global__') continue;
      const taskId = taskIdOfEvent(event);
      if (taskId && !this.taskProjectMap.has(taskId)) this.taskProjectMap.set(taskId, project);
    }
  }

  /**
   * Project bucket for one event: its own tag when it has one, else the tag of
   * the task it belongs to. `extractProjectName` stays a pure function of the
   * event; the map lookup lives here.
   */
  private resolveProjectName(event: any): string {
    const direct = extractProjectName(event);
    if (direct !== '__global__') return direct;
    const taskId = taskIdOfEvent(event);
    return (taskId && this.taskProjectMap.get(taskId)) || '__global__';
  }

  /**
   * Shared event -> CoordinationMessage mapping for both index paths.
   * Drops coordination-type duplicates of typed lifecycle events in the same
   * batch (their rich content is captured via buildEmbeddingText on the typed
   * variant) and tags each message with its resolved project bucket.
   */
  private toCoordinationMessages(events: any[]): CoordinationMessage[] {
    // Identify which taskIds have a typed lifecycle event in this batch, so we
    // can skip the coordination-duplicates that mirror them.
    const typedTaskIds = new Set<string>();
    const TYPED_LIFECYCLE_TYPES = new Set([
      'task_created', 'task_assigned', 'task_completed', 'task_failed',
      'task_validated', 'task_validation_failed', 'task_submitted_for_validation',
      'synthesis_complete'
    ]);
    for (const event of events) {
      if (TYPED_LIFECYCLE_TYPES.has(event.type)) {
        const taskId = taskIdOfEvent(event);
        if (taskId) typedTaskIds.add(taskId);
      }
    }

    this.learnTaskProjects(events);

    return events
      .filter(event => {
        // Standalone coordination events (no taskId or no typed pair) are preserved.
        if (event.type !== 'coordination') return true;
        const taskId = event.data?.taskId || event.data?.task?.id;
        if (!taskId) return true;
        return !typedTaskIds.has(taskId);
      })
      .map(event => {
        const text = buildEmbeddingText(event);
        return {
          timestamp: event.timestamp,
          agent: event.agent || event.agentId || 'system',
          message: text,
          type: event.type || 'coordination',
          // Preserve raw event.data so analytics readers (lookupTaskOutcomes,
          // getAgentSynergy, SelfImprovementEngine) can JOIN on data.taskId
          // across task_assigned / task_completed / task_validated events.
          // Without this, the indexed point loses its lifecycle JOIN key and
          // every agent reports completedTasks:0.
          data: event.data,
          scope: classifyScope(event.type, text),
          projectName: this.resolveProjectName(event)
        } as CoordinationMessage;
      });
  }

  /**
   * Start background indexing of coordination events
   * @param intervalMs - How often to check for new events (default: 10 seconds)
   */
  startIndexing(intervalMs: number = 10000): void {
    if (this.indexingInterval) {
      logger.warn('PVMIndexer already running, stopping previous instance');
      this.stopIndexing();
    }
    
    logger.info(`🚀 PVMIndexer started - checking for new events every ${intervalMs}ms`);
    
    this.indexingInterval = setInterval(() => {
      this.indexNewEvents().catch(err => {
        logger.error(`PVMIndexer failed to index events: ${err.message}`);
      });
    }, intervalMs);
  }
  
  /**
   * Stop background indexing
   */
  stopIndexing(): void {
    if (this.indexingInterval) {
      clearInterval(this.indexingInterval);
      this.indexingInterval = null;
      logger.info('🛑 PVMIndexer stopped');
    }
  }
  
  /**
   * Index new events that haven't been indexed yet
   */
  async indexNewEvents(): Promise<void> {
    // Prevent concurrent indexing
    if (this.isIndexing) {
      logger.debug('PVMIndexer already indexing, skipping this cycle');
      return;
    }

    this.isIndexing = true;

    try {
      logger.debug('🔍 PVMIndexer checking for new events to index...');

      // Fetch all events after the last indexed timestamp.
      // Use a very high limit so we drain everything in one pass.
      // ChronologicalLog.query filters with >= on `since`, so we bump the
      // stored timestamp by 1ms to avoid re-indexing the last seen event.
      const query: any = { limit: 100000 };
      if (this.lastIndexedTimestamp) {
        const bumped = new Date(new Date(this.lastIndexedTimestamp).getTime() + 1).toISOString();
        query.since = bumped;
      }

      const result = await this.chronologicalLog.query(query);
      const events = result.events;

      if (events.length === 0) {
        logger.debug('🔍 PVMIndexer: No new events to index');
        return;
      }

      logger.info(`📥 PVMIndexer found ${events.length} new events to index`);

      const coordinationMessages = this.toCoordinationMessages(events);

      const globalCount = coordinationMessages.filter(m => m.projectName === '__global__').length;
      if (globalCount > 0) {
        logger.debug(`PVMIndexer: ${globalCount}/${events.length} events tagged as __global__ (no extractable project name)`);
      }

      // Store in vector store
      const statsBefore = (this.vectorStore as any).getSidecarStats?.();
      await this.vectorStore.batchStore(coordinationMessages);
      const statsAfter = (this.vectorStore as any).getSidecarStats?.();
      if (statsBefore && statsAfter) {
        logger.info(`PVM drain: ${statsAfter.fromCache - statsBefore.fromCache} from cache, ${statsAfter.freshEmbeds - statsBefore.freshEmbeds} embedded fresh`);
      }

      // Advance cursor to the latest event's timestamp
      this.lastIndexedTimestamp = events[events.length - 1].timestamp;

      logger.info(`✅ PVMIndexer successfully indexed ${events.length} events`);

    } catch (error: any) {
      logger.error(`❌ PVMIndexer failed during indexing: ${error.message}`);
      throw error;
    } finally {
      this.isIndexing = false;
    }
  }
  
  /**
   * Force index all events (useful for initial setup)
   */
  async indexAllEvents(): Promise<void> {
    logger.info('🔄 PVMIndexer force indexing all events...');
    
    try {
      // Query for all events
      const result = await this.chronologicalLog.query({ limit: 100000 });
      const events = result.events;
      
      if (events.length === 0) {
        logger.info('🔍 PVMIndexer: No events found to index');
        return;
      }
      
      logger.info(`📥 PVMIndexer found ${events.length} events to index`);

      // Full re-index rebuilds the taskId -> project map from scratch, so a
      // restart (or a manual /api/pvm/reindex) is deterministic — the same log
      // always produces the same buckets.
      this.taskProjectMap.clear();
      const coordinationMessages = this.toCoordinationMessages(events);

      // Store in vector store
      await this.vectorStore.batchStore(coordinationMessages);

      // Update last indexed timestamp
      if (events.length > 0) {
        // Don't trust event order — pick the actual max timestamp so the cursor
        // never regresses. (ChronologicalLog.query may return events in any order
        // depending on how the storage backend serves them.)
        let maxTs = events[0].timestamp;
        for (const e of events) {
          if (e.timestamp > maxTs) maxTs = e.timestamp;
        }
        // Belt-and-suspenders: even if callers provide a narrowed event set,
        // never allow cursor regression.
        if (this.lastIndexedTimestamp === null || maxTs > this.lastIndexedTimestamp) {
          this.lastIndexedTimestamp = maxTs;
        }
      }
      
      logger.info(`✅ PVMIndexer successfully indexed ${events.length} events`);
      
    } catch (error: any) {
      logger.error(`❌ PVMIndexer failed during full indexing: ${error.message}`);
      throw error;
    }
  }
  
  /**
   * Search for similar coordination patterns
   * @param query - Search query text
   * @param limit - Maximum number of results to return
   * @param projectName - Optional project scope
   * @param threshold - Optional minimum cosine similarity; callers own the
   *   default (see PVM_SEARCH_MIN_SIMILARITY at the /api/pvm/search route).
   */
  async search(query: string, limit: number = 10, projectName?: string, threshold?: number): Promise<any[]> {
    try {
      const searchQuery: any = { query, limit };
      if (projectName) searchQuery.projectName = projectName;
      if (typeof threshold === 'number') searchQuery.threshold = threshold;
      const results = await this.vectorStore.search(searchQuery);
      logger.info(`🔍 PVMIndexer search returned ${results.length} results${projectName ? ` (scope: ${projectName})` : ' (cross-project)'}`);
      return results;
    } catch (error: any) {
      logger.error(`❌ PVMIndexer search failed: ${error.message}`);
      throw error;
    }
  }
  
  /**
   * Get indexer status
   */
  getStatus(): {
    isRunning: boolean;
    isIndexing: boolean;
    lastIndexedTimestamp: string | null;
    indexedEventCount: number;
    taggedTaskCount: number;
    attribution?: { validations: number; unattributed: number; computedAt: string };
  } {
    // Attribution health: how many validated tasks could not be tied back to a
    // worker on the last getProjectOutcomes() pass. Silent drops here are what
    // made the routing brief lie; surfacing the count makes a regression visible.
    const attribution = (this.vectorStore as any).getAttributionStats?.();
    return {
      isRunning: !!this.indexingInterval,
      isIndexing: this.isIndexing,
      lastIndexedTimestamp: this.lastIndexedTimestamp,
      indexedEventCount: (this.vectorStore as any).points ? (this.vectorStore as any).points.length : -1,
      taggedTaskCount: this.taskProjectMap.size,
      ...(attribution ? { attribution } : {})
    };
  }
}

export default PVMIndexer;
