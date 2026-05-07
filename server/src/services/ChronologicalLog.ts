/**
 * ChronologicalLog - Append-only coordination event log for PVM
 *
 * Task 1.2: Chronological Log Implementation (12 hours)
 *
 * Purpose:
 * - Human-readable audit trail of all coordination events
 * - Append-only semantics (never overwrite, never delete)
 * - Efficient recent event queries for context windows
 * - Persistent storage (JSONL format) with SQLite fallback
 * - Background flush strategy (every 10 events or 5 seconds)
 * - Compaction for long-running servers
 *
 * Integration:
 * - VectorMemoryStore reads from this log to build semantic memory
 * - FLUX State uses this for unbiased evaluation context
 * - PAIR retrieval searches this for patterns
 * - AgentProfileBuilder analyzes this for evidence-based profiles
 */

import fs from 'fs/promises';
import path from 'path';
import { CoordinationMessage } from '../types/coordination.js';

export interface ChronologicalLogConfig {
  // Storage configuration
  storageType: 'jsonl' | 'sqlite' | 'both';
  jsonlPath?: string;
  sqlitePath?: string;

  // Performance tuning
  bufferSize?: number; // In-memory buffer size before flush
  flushIntervalMs?: number; // Max time between flushes
  compactionThreshold?: number; // Events before compaction triggers
  compactionKeepRecent?: number; // Keep this many recent events uncompacted

  // Query optimization
  indexTimestamps?: boolean;
  indexAgents?: boolean;
}

export const DEFAULT_CHRONOLOGICAL_CONFIG: ChronologicalLogConfig = {
  storageType: 'jsonl',
  jsonlPath: './data/coordination-log.jsonl',
  bufferSize: 100,
  flushIntervalMs: 5000,
  compactionThreshold: 10000,
  compactionKeepRecent: 1000,
  indexTimestamps: true,
  indexAgents: true
};

export interface LogQuery {
  // Time-based queries
  since?: string; // ISO timestamp
  until?: string; // ISO timestamp

  // Agent-based queries
  agent?: string;

  // Type-based queries
  type?: string;

  // Pagination
  limit?: number;
  offset?: number;
}

export interface LogQueryResult {
  events: CoordinationMessage[];
  total: number;
  hasMore: boolean;
}

/**
 * ChronologicalLog - Append-only coordination event log
 *
 * Design principles:
 * - Append-only: Never modify or delete events
 * - Human-readable: JSONL format for easy inspection
 * - Durable: Flush to disk regularly
 * - Efficient: In-memory buffer for recent events
 * - Auditable: Complete history with timestamps
 */
export class ChronologicalLog {
  private config: ChronologicalLogConfig;
  private buffer: CoordinationMessage[] = [];
  private flushTimer: NodeJS.Timeout | null = null;
  private totalEvents: number = 0;
  private initialized: boolean = false;

  constructor(config: Partial<ChronologicalLogConfig> = {}) {
    this.config = { ...DEFAULT_CHRONOLOGICAL_CONFIG, ...config };
  }

  /**
   * Initialize the log (create directories, load existing events)
   */
  async initialize(): Promise<void> {
    if (this.initialized) return;

    // Ensure storage directory exists
    if (this.config.jsonlPath) {
      const dir = path.dirname(this.config.jsonlPath);
      await fs.mkdir(dir, { recursive: true });
    }

    // Count existing events
    if (this.config.jsonlPath) {
      try {
        const content = await fs.readFile(this.config.jsonlPath, 'utf-8');
        const lines = content.trim().split('\n').filter(l => l.length > 0);
        this.totalEvents = lines.length;
        console.log(`ChronologicalLog: Loaded ${this.totalEvents} existing events`);
      } catch (error: any) {
        if (error.code !== 'ENOENT') {
          console.error('Error loading existing log:', error);
        }
        // File doesn't exist yet - that's fine
        this.totalEvents = 0;
      }
    }

    // Start flush timer
    this.startFlushTimer();

    this.initialized = true;
  }

  /**
   * Append a coordination event to the log
   * This is the ONLY way to add events (append-only)
   *
   * @param event - Coordination event to append
   */
  async append(event: CoordinationMessage): Promise<void> {
    if (!this.initialized) {
      await this.initialize();
    }

    // Add to in-memory buffer
    this.buffer.push(event);
    this.totalEvents++;

    // Flush if buffer is full
    if (this.buffer.length >= this.config.bufferSize!) {
      await this.flush();
    }

    // Check if compaction is needed
    if (this.totalEvents >= this.config.compactionThreshold!) {
      // Trigger compaction in background (don't await)
      this.compact().catch(error => {
        console.error('Background compaction failed:', error);
      });
    }
  }

  /**
   * Batch append multiple events
   * More efficient than multiple append() calls
   *
   * @param events - Array of coordination events
   */
  async batchAppend(events: CoordinationMessage[]): Promise<void> {
    if (!this.initialized) {
      await this.initialize();
    }

    // Add all to buffer
    this.buffer.push(...events);
    this.totalEvents += events.length;

    // Flush if buffer exceeds threshold
    if (this.buffer.length >= this.config.bufferSize!) {
      await this.flush();
    }
  }

  /**
   * Flush in-memory buffer to persistent storage
   */
  async flush(): Promise<void> {
    if (this.buffer.length === 0) return;

    try {
      if (this.config.storageType === 'jsonl' || this.config.storageType === 'both') {
        await this.flushToJSONL();
      }

      if (this.config.storageType === 'sqlite' || this.config.storageType === 'both') {
        await this.flushToSQLite();
      }

      // Clear buffer after successful flush
      this.buffer = [];
    } catch (error) {
      console.error('Flush failed:', error);
      // Keep buffer intact for retry
      throw error;
    }
  }

  /**
   * Flush to JSONL file (human-readable format).
   * Uses open+write+fsync+close for durability — ensures data reaches disk
   * even if the process crashes immediately after this call returns.
   */
  private async flushToJSONL(): Promise<void> {
    if (!this.config.jsonlPath) return;

    const lines = this.buffer.map(event => JSON.stringify(event)).join('\n') + '\n';

    const fh = await fs.open(this.config.jsonlPath, 'a');
    try {
      await fh.write(lines, null, 'utf-8');
      await fh.sync(); // fsync — ensure data is on disk, not just in OS cache
    } finally {
      await fh.close();
    }
  }

  /**
   * Flush to SQLite database (queryable format)
   * TODO: Implement SQLite storage
   */
  private async flushToSQLite(): Promise<void> {
    // TODO: Implement SQLite storage for efficient queries
    // For now, JSONL is sufficient
  }

  /**
   * Query recent events
   * Efficient for common use case: "get last N events"
   *
   * @param limit - Maximum number of events to return
   * @returns Recent events in reverse chronological order (newest first)
   */
  async getRecent(limit: number = 100): Promise<CoordinationMessage[]> {
    if (!this.initialized) {
      await this.initialize();
    }

    // Flush buffer to ensure we have latest
    await this.flush();

    // Read from JSONL file
    if (!this.config.jsonlPath) {
      return [];
    }

    try {
      const content = await fs.readFile(this.config.jsonlPath, 'utf-8');
      const lines = content.trim().split('\n').filter(l => l.length > 0);

      // Get last N lines
      const recentLines = lines.slice(-limit);

      // Parse and reverse (newest first)
      const events = recentLines.map(line => JSON.parse(line) as CoordinationMessage);
      return events.reverse();
    } catch (error: any) {
      if (error.code === 'ENOENT') {
        return [];
      }
      throw error;
    }
  }

  /**
   * Query events with filters
   *
   * @param query - Query parameters
   * @returns Matching events
   */
  async query(query: LogQuery): Promise<LogQueryResult> {
    if (!this.initialized) {
      await this.initialize();
    }

    // Flush to ensure we query latest data
    await this.flush();

    // Read all events
    // TODO: Optimize with SQLite for large logs
    const allEvents = await this.getAll();

    // Filter events
    let filtered = allEvents;

    if (query.since) {
      filtered = filtered.filter(e => e.timestamp >= query.since!);
    }

    if (query.until) {
      filtered = filtered.filter(e => e.timestamp <= query.until!);
    }

    if (query.agent) {
      filtered = filtered.filter(e => e.agent === query.agent);
    }

    if (query.type) {
      filtered = filtered.filter(e => e.type === query.type);
    }

    // Apply pagination
    const offset = query.offset || 0;
    const limit = query.limit || 100;
    const total = filtered.length;
    const paginated = filtered.slice(offset, offset + limit);

    return {
      events: paginated,
      total,
      hasMore: offset + limit < total
    };
  }

  /**
   * Get all events (use with caution for large logs)
   *
   * @returns All events in chronological order
   */
  async getAll(): Promise<CoordinationMessage[]> {
    if (!this.initialized) {
      await this.initialize();
    }

    await this.flush();

    if (!this.config.jsonlPath) {
      return [];
    }

    try {
      const content = await fs.readFile(this.config.jsonlPath, 'utf-8');
      const lines = content.trim().split('\n').filter(l => l.length > 0);
      return lines.map(line => JSON.parse(line) as CoordinationMessage);
    } catch (error: any) {
      if (error.code === 'ENOENT') {
        return [];
      }
      throw error;
    }
  }

  /**
   * Get event count
   *
   * @returns Total number of events in log
   */
  getEventCount(): number {
    return this.totalEvents;
  }

  /**
   * Compact the log file
   * Removes duplicate events, optimizes storage
   * Keeps recent events intact for auditing
   */
  private async compact(): Promise<void> {
    if (!this.config.jsonlPath) return;

    console.log('ChronologicalLog: Starting compaction...');

    try {
      const allEvents = await this.getAll();

      // Keep recent events uncompacted
      const keepRecent = this.config.compactionKeepRecent!;
      const recentEvents = allEvents.slice(-keepRecent);

      // Older events can be deduplicated/compressed
      const olderEvents = allEvents.slice(0, -keepRecent);

      // For now, simple compaction: just keep all events
      // TODO: More sophisticated compaction (remove duplicates, compress, etc.)
      const compactedEvents = [...olderEvents, ...recentEvents];

      // Write compacted log
      const compactedPath = this.config.jsonlPath + '.compacted';
      const lines = compactedEvents.map(e => JSON.stringify(e)).join('\n') + '\n';
      await fs.writeFile(compactedPath, lines, 'utf-8');

      // Atomic replace
      await fs.rename(compactedPath, this.config.jsonlPath);

      console.log(`ChronologicalLog: Compaction complete (${allEvents.length} → ${compactedEvents.length} events)`);
    } catch (error) {
      console.error('Compaction failed:', error);
      throw error;
    }
  }

  /**
   * Start background flush timer
   */
  private startFlushTimer(): void {
    if (this.flushTimer) {
      clearInterval(this.flushTimer);
    }

    this.flushTimer = setInterval(() => {
      this.flush().catch(error => {
        console.error('Background flush failed:', error);
      });
    }, this.config.flushIntervalMs!);
  }

  /**
   * Close the log and flush any pending events
   */
  async close(): Promise<void> {
    // Stop flush timer
    if (this.flushTimer) {
      clearInterval(this.flushTimer);
      this.flushTimer = null;
    }

    // Final flush
    await this.flush();

    this.initialized = false;
  }

  /**
   * Restore in-memory state from the event log (event sourcing replay)
   * Reads JSONL file and replays events to rebuild projects, tasks, briefs, agents
   */
  async restoreFromLog(
    projects: Map<string, any>,
    tasks: Map<string, any>,
    briefs: Map<string, Map<string, string>>,
    agents: Map<string, any>,
    fileLocks?: Map<string, any>
  ): Promise<{ projectCount: number; taskCount: number; briefCount: number; agentCount: number; fileLockCount: number }> {
    if (!this.initialized) {
      await this.initialize();
    }

    await this.flush();

    const counts = { projectCount: 0, taskCount: 0, briefCount: 0, agentCount: 0, fileLockCount: 0 };

    if (!this.config.jsonlPath) {
      return counts;
    }

    try {
      const content = await fs.readFile(this.config.jsonlPath, 'utf-8');
      const lines = content.trim().split('\n').filter(l => l.length > 0);

      for (const line of lines) {
        try {
          const event = JSON.parse(line) as { type: string; agent: string; message: string; timestamp: string; data?: any };
          const d = event.data;

          switch (event.type) {
            case 'project_created': {
              if (d && d.name) {
                projects.set(d.name, d);
                counts.projectCount++;
              }
              break;
            }
            case 'task_created': {
              if (d && d.id) {
                coerceTaskDates(d);
                tasks.set(d.id, d);
                counts.taskCount++;
              }
              break;
            }
            case 'task_assigned': {
              if (d && d.taskId) {
                const task = tasks.get(d.taskId);
                if (task) {
                  task.assignedAgent = d.agentId;
                  task.status = 'assigned';
                }
              }
              break;
            }
            case 'task_completed': {
              if (d && d.taskId) {
                const task = tasks.get(d.taskId);
                if (task) {
                  // Older log entries used a unified 'task_completed' event
                  // with d.success=false to signal failure. New entries use
                  // the dedicated 'task_failed' type below. Handle both so
                  // replay of pre-split logs still works.
                  task.status = d.success === false ? 'failed' : 'completed';
                  task.progress = d.success === false ? task.progress : 100;
                  if (d.result) {
                    if (!task.metadata) task.metadata = {};
                    task.metadata.result = d.result;
                  }
                }
              }
              break;
            }
            case 'task_failed': {
              if (d && d.taskId) {
                const task = tasks.get(d.taskId);
                if (task) {
                  task.status = 'failed';
                  if (d.result) {
                    if (!task.metadata) task.metadata = {};
                    task.metadata.result = d.result;
                  }
                }
              }
              break;
            }
            case 'brief_stored': {
              if (d && d.projectName && d.agentId && d.content) {
                let projectBriefs = briefs.get(d.projectName);
                if (!projectBriefs) {
                  projectBriefs = new Map<string, string>();
                  briefs.set(d.projectName, projectBriefs);
                }
                projectBriefs.set(d.agentId, d.content);
                // Also restore into the project object's briefs
                const project = projects.get(d.projectName);
                if (project) {
                  if (!project.briefs) project.briefs = {};
                  project.briefs[d.agentId] = d.content;
                }
                counts.briefCount++;
              }
              break;
            }
            case 'task_submitted_for_validation': {
              if (d && d.taskId) {
                const task = tasks.get(d.taskId);
                if (task) {
                  task.status = 'submitted_for_validation';
                }
              }
              break;
            }
            case 'task_validated': {
              if (d && d.taskId) {
                const task = tasks.get(d.taskId);
                if (task) {
                  task.status = 'validated';
                  if (d.score !== undefined) {
                    if (!task.metadata) task.metadata = {};
                    task.metadata.validationScore = d.score;
                  }
                }
              }
              break;
            }
            case 'task_validation_failed': {
              if (d && d.taskId) {
                const task = tasks.get(d.taskId);
                if (task) {
                  task.status = 'assigned'; // returned to agent
                  if (!task.metadata) task.metadata = {};
                  task.metadata.validationAttempts = (task.metadata.validationAttempts || 0) + 1;
                  if (d.gaps) task.metadata.validationGaps = d.gaps;
                }
              }
              break;
            }
            case 'agent_registered': {
              if (d && d.agentId) {
                agents.set(d.agentId, {
                  id: d.agentId,
                  name: d.name || d.agentId,
                  capabilities: d.capabilities || [],
                  status: 'offline',  // will re-register on connect
                  lastSeen: toDate(event.timestamp) ?? new Date(),
                  performanceScore: 1.0,
                  tasksCompleted: 0,
                  averageTaskTime: 0
                });
                counts.agentCount++;
              }
              break;
            }
            case 'dev_reset': {
              // A reset event means everything before this point is void.
              // Clear all maps so only post-reset events populate state.
              projects.clear();
              tasks.clear();
              briefs.clear();
              agents.clear();
              if (fileLocks) fileLocks.clear();
              counts.projectCount = 0;
              counts.taskCount = 0;
              counts.briefCount = 0;
              counts.agentCount = 0;
              counts.fileLockCount = 0;
              break;
            }
            case 'file_claim': {
              if (fileLocks && d && d.filePaths && d.agentId) {
                const lockedAt = toDate(event.timestamp) ?? new Date();
                for (const fp of d.filePaths) {
                  fileLocks.set(fp, {
                    filePath: fp,
                    agentId: d.agentId,
                    taskId: d.taskId || '',
                    lockedAt,
                  });
                  counts.fileLockCount++;
                }
              }
              break;
            }
            case 'file_release': {
              if (fileLocks && d && d.filePaths) {
                for (const fp of d.filePaths) {
                  fileLocks.delete(fp);
                  if (counts.fileLockCount > 0) counts.fileLockCount--;
                }
              }
              break;
            }
          }
        } catch (parseError) {
          // Skip unparseable lines (non-fatal)
        }
      }
    } catch (error: any) {
      if (error.code !== 'ENOENT') {
        console.error('Error restoring from log:', error);
      }
    }

    return counts;
  }

}

// JSONL replay deserializes ISO timestamp strings as plain strings; downstream
// callers expect Date instances and crash on .getTime(). Coerce at the boundary.
function toDate(v: any): Date | undefined {
  if (v === undefined || v === null) return undefined;
  if (v instanceof Date) return v;
  const d = new Date(v);
  return isNaN(d.getTime()) ? undefined : d;
}

function coerceTaskDates(t: any): void {
  if (!t) return;
  for (const k of ['createdAt', 'updatedAt', 'startedAt', 'completedAt', 'assignedAt'] as const) {
    if (t[k] !== undefined) t[k] = toDate(t[k]);
  }
}
