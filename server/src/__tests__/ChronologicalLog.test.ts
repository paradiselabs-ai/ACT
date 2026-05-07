/**
 * ChronologicalLog Unit Tests
 *
 * Tests for append-only coordination event log
 */

import { ChronologicalLog } from '../services/ChronologicalLog.js';
import { CoordinationMessage } from '../types/coordination.js';
import fs from 'fs/promises';
import path from 'path';

const TEST_DATA_DIR = './test-data';
const TEST_LOG_PATH = path.join(TEST_DATA_DIR, 'test-coordination.jsonl');

describe('ChronologicalLog', () => {
  let log: ChronologicalLog;

  beforeEach(async () => {
    // Clean up test directory
    try {
      await fs.rm(TEST_DATA_DIR, { recursive: true, force: true });
    } catch (error) {
      // Directory doesn't exist, that's fine
    }

    // Create fresh log
    log = new ChronologicalLog({
      jsonlPath: TEST_LOG_PATH,
      bufferSize: 5, // Small buffer for testing
      flushIntervalMs: 1000
    });

    await log.initialize();
  });

  afterEach(async () => {
    await log.close();

    // Clean up
    try {
      await fs.rm(TEST_DATA_DIR, { recursive: true, force: true });
    } catch (error) {
      // Ignore cleanup errors
    }
  });

  describe('append', () => {
    it('should append events to the log', async () => {
      const event: CoordinationMessage = {
        timestamp: '2025-12-06T00:00:00Z',
        agent: 'test_agent',
        message: 'Test message',
        type: 'coordination'
      };

      await log.append(event);
      await log.flush(); // Ensure written

      const recent = await log.getRecent(10);
      expect(recent.length).toBe(1);
      expect(recent[0]).toEqual(event);
    });

    it('should maintain append-only semantics (never overwrite)', async () => {
      const events: CoordinationMessage[] = [
        {
          timestamp: '2025-12-06T00:00:00Z',
          agent: 'agent1',
          message: 'Message 1',
          type: 'coordination'
        },
        {
          timestamp: '2025-12-06T00:01:00Z',
          agent: 'agent2',
          message: 'Message 2',
          type: 'coordination'
        },
        {
          timestamp: '2025-12-06T00:02:00Z',
          agent: 'agent3',
          message: 'Message 3',
          type: 'coordination'
        }
      ];

      for (const event of events) {
        await log.append(event);
      }

      await log.flush();

      const all = await log.getAll();
      expect(all.length).toBe(3);
      expect(all[0]).toEqual(events[0]);
      expect(all[1]).toEqual(events[1]);
      expect(all[2]).toEqual(events[2]);
    });

    it('should auto-flush when buffer is full', async () => {
      const events: CoordinationMessage[] = [];

      // Create 6 events (buffer size is 5)
      for (let i = 0; i < 6; i++) {
        events.push({
          timestamp: `2025-12-06T00:0${i}:00Z`,
          agent: `agent${i}`,
          message: `Message ${i}`,
          type: 'coordination'
        });
      }

      for (const event of events) {
        await log.append(event);
      }

      // Should have auto-flushed after 5th event
      const fileContent = await fs.readFile(TEST_LOG_PATH, 'utf-8');
      const lines = fileContent.trim().split('\n');

      // At least 5 should be flushed
      expect(lines.length).toBeGreaterThanOrEqual(5);
    });
  });

  describe('batchAppend', () => {
    it('should append multiple events efficiently', async () => {
      const events: CoordinationMessage[] = [
        {
          timestamp: '2025-12-06T00:00:00Z',
          agent: 'agent1',
          message: 'Batch 1',
          type: 'coordination'
        },
        {
          timestamp: '2025-12-06T00:01:00Z',
          agent: 'agent2',
          message: 'Batch 2',
          type: 'coordination'
        },
        {
          timestamp: '2025-12-06T00:02:00Z',
          agent: 'agent3',
          message: 'Batch 3',
          type: 'coordination'
        }
      ];

      await log.batchAppend(events);
      await log.flush();

      const all = await log.getAll();
      expect(all.length).toBe(3);
      expect(all).toEqual(events);
    });
  });

  describe('getRecent', () => {
    it('should return recent events in reverse chronological order', async () => {
      const events: CoordinationMessage[] = [];

      for (let i = 0; i < 10; i++) {
        events.push({
          timestamp: `2025-12-06T00:${String(i).padStart(2, '0')}:00Z`,
          agent: `agent${i}`,
          message: `Message ${i}`,
          type: 'coordination'
        });
      }

      await log.batchAppend(events);
      await log.flush();

      const recent = await log.getRecent(5);

      expect(recent.length).toBe(5);
      // Should be newest first
      expect(recent[0].agent).toBe('agent9');
      expect(recent[1].agent).toBe('agent8');
      expect(recent[2].agent).toBe('agent7');
      expect(recent[3].agent).toBe('agent6');
      expect(recent[4].agent).toBe('agent5');
    });

    it('should handle limit larger than total events', async () => {
      const events: CoordinationMessage[] = [
        {
          timestamp: '2025-12-06T00:00:00Z',
          agent: 'agent1',
          message: 'Only event',
          type: 'coordination'
        }
      ];

      await log.batchAppend(events);
      await log.flush();

      const recent = await log.getRecent(100);
      expect(recent.length).toBe(1);
    });
  });

  describe('query', () => {
    beforeEach(async () => {
      const events: CoordinationMessage[] = [
        {
          timestamp: '2025-12-06T00:00:00Z',
          agent: 'agent1',
          message: 'Message 1',
          type: 'coordination'
        },
        {
          timestamp: '2025-12-06T00:01:00Z',
          agent: 'agent2',
          message: 'Message 2',
          type: 'progress_report'
        },
        {
          timestamp: '2025-12-06T00:02:00Z',
          agent: 'agent1',
          message: 'Message 3',
          type: 'coordination'
        },
        {
          timestamp: '2025-12-06T00:03:00Z',
          agent: 'agent2',
          message: 'Message 4',
          type: 'feature_complete'
        }
      ];

      await log.batchAppend(events);
      await log.flush();
    });

    it('should filter by agent', async () => {
      const result = await log.query({ agent: 'agent1' });

      expect(result.events.length).toBe(2);
      expect(result.events[0].agent).toBe('agent1');
      expect(result.events[1].agent).toBe('agent1');
      expect(result.total).toBe(2);
    });

    it('should filter by type', async () => {
      const result = await log.query({ type: 'coordination' });

      expect(result.events.length).toBe(2);
      expect(result.events[0].type).toBe('coordination');
      expect(result.events[1].type).toBe('coordination');
    });

    it('should filter by time range', async () => {
      const result = await log.query({
        since: '2025-12-06T00:01:00Z',
        until: '2025-12-06T00:02:30Z'
      });

      expect(result.events.length).toBe(2);
      expect(result.events[0].timestamp).toBe('2025-12-06T00:01:00Z');
      expect(result.events[1].timestamp).toBe('2025-12-06T00:02:00Z');
    });

    it('should support pagination', async () => {
      const page1 = await log.query({ limit: 2, offset: 0 });
      const page2 = await log.query({ limit: 2, offset: 2 });

      expect(page1.events.length).toBe(2);
      expect(page2.events.length).toBe(2);
      expect(page1.hasMore).toBe(true);
      expect(page2.hasMore).toBe(false);
      expect(page1.total).toBe(4);
    });

    it('should combine multiple filters', async () => {
      const result = await log.query({
        agent: 'agent1',
        type: 'coordination'
      });

      expect(result.events.length).toBe(2);
      expect(result.events.every(e => e.agent === 'agent1' && e.type === 'coordination')).toBe(true);
    });
  });

  describe('persistence', () => {
    it('should persist events to JSONL file', async () => {
      const event: CoordinationMessage = {
        timestamp: '2025-12-06T00:00:00Z',
        agent: 'test_agent',
        message: 'Persistence test',
        type: 'coordination'
      };

      await log.append(event);
      await log.flush();

      // Read file directly
      const content = await fs.readFile(TEST_LOG_PATH, 'utf-8');
      const lines = content.trim().split('\n');

      expect(lines.length).toBe(1);
      const parsed = JSON.parse(lines[0]);
      expect(parsed).toEqual(event);
    });

    it('should load existing events on initialization', async () => {
      // Write events
      const events: CoordinationMessage[] = [
        {
          timestamp: '2025-12-06T00:00:00Z',
          agent: 'agent1',
          message: 'Existing 1',
          type: 'coordination'
        },
        {
          timestamp: '2025-12-06T00:01:00Z',
          agent: 'agent2',
          message: 'Existing 2',
          type: 'coordination'
        }
      ];

      await log.batchAppend(events);
      await log.flush();
      await log.close();

      // Create new log instance
      const log2 = new ChronologicalLog({
        jsonlPath: TEST_LOG_PATH,
        bufferSize: 5
      });

      await log2.initialize();

      // Should have loaded existing events
      expect(log2.getEventCount()).toBe(2);

      const all = await log2.getAll();
      expect(all.length).toBe(2);
      expect(all).toEqual(events);

      await log2.close();
    });

    it('should be human-readable JSONL format', async () => {
      const event: CoordinationMessage = {
        timestamp: '2025-12-06T00:00:00Z',
        agent: 'human_reader',
        message: 'This should be readable',
        type: 'documentation_update'
      };

      await log.append(event);
      await log.flush();

      const content = await fs.readFile(TEST_LOG_PATH, 'utf-8');

      // Should be valid JSON
      const parsed = JSON.parse(content.trim());
      expect(parsed.agent).toBe('human_reader');
      expect(parsed.message).toBe('This should be readable');
    });
  });

  describe('getEventCount', () => {
    it('should return accurate event count', async () => {
      expect(log.getEventCount()).toBe(0);

      await log.append({
        timestamp: '2025-12-06T00:00:00Z',
        agent: 'agent1',
        message: 'Event 1',
        type: 'coordination'
      });

      expect(log.getEventCount()).toBe(1);

      await log.batchAppend([
        {
          timestamp: '2025-12-06T00:01:00Z',
          agent: 'agent2',
          message: 'Event 2',
          type: 'coordination'
        },
        {
          timestamp: '2025-12-06T00:02:00Z',
          agent: 'agent3',
          message: 'Event 3',
          type: 'coordination'
        }
      ]);

      expect(log.getEventCount()).toBe(3);
    });
  });

  describe('append-only guarantees', () => {
    it('should never allow modification of existing events', async () => {
      const original: CoordinationMessage = {
        timestamp: '2025-12-06T00:00:00Z',
        agent: 'agent1',
        message: 'Original message',
        type: 'coordination'
      };

      await log.append(original);
      await log.flush();

      // Try to "modify" by appending with same timestamp
      const attempted: CoordinationMessage = {
        timestamp: '2025-12-06T00:00:00Z', // Same timestamp
        agent: 'agent1',
        message: 'Modified message', // Different message
        type: 'coordination'
      };

      await log.append(attempted);
      await log.flush();

      // Both should exist (append-only)
      const all = await log.getAll();
      expect(all.length).toBe(2);
      expect(all[0]).toEqual(original);
      expect(all[1]).toEqual(attempted);
    });

    it('should never delete events', async () => {
      const events: CoordinationMessage[] = [];

      for (let i = 0; i < 100; i++) {
        events.push({
          timestamp: `2025-12-06T00:${String(i).padStart(2, '0')}:00Z`,
          agent: `agent${i}`,
          message: `Message ${i}`,
          type: 'coordination'
        });
      }

      await log.batchAppend(events);
      await log.flush();

      // All events should still be there
      const all = await log.getAll();
      expect(all.length).toBe(100);
    });
  });

  describe('restoreFromLog (KI-13 regression)', () => {
    it('rehydrates task date fields as Date instances, not strings', async () => {
      const taskEvent: CoordinationMessage = {
        timestamp: '2026-04-19T12:00:00.000Z',
        agent: 'orchestrator',
        message: 'task created',
        type: 'task_created',
        data: {
          id: 'task-ki13',
          description: 'regression task',
          status: 'pending',
          priority: 'medium',
          requiredCapabilities: [],
          progress: 0,
          createdAt: '2026-04-19T12:00:00.000Z',
          updatedAt: '2026-04-19T12:00:00.000Z',
        },
      };
      const startedEvent: CoordinationMessage = {
        timestamp: '2026-04-19T12:01:00.000Z',
        agent: 'orchestrator',
        message: 'task started',
        type: 'task_assigned',
        data: { taskId: 'task-ki13', agentId: 'dev-1' },
      };
      await log.batchAppend([taskEvent, startedEvent]);
      await log.flush();

      const projects = new Map<string, any>();
      const tasks = new Map<string, any>();
      const briefs = new Map<string, Map<string, string>>();
      const agents = new Map<string, any>();
      await log.restoreFromLog(projects, tasks, briefs, agents);

      const t = tasks.get('task-ki13');
      expect(t).toBeDefined();
      expect(t.createdAt).toBeInstanceOf(Date);
      expect(t.updatedAt).toBeInstanceOf(Date);
      // simulate the call site that crashed pre-fix: TaskCoordinator.completeTask
      // line 245 — task.completedAt.getTime() - task.startedAt.getTime()
      t.startedAt = new Date('2026-04-19T12:01:00.000Z');
      t.completedAt = new Date('2026-04-19T12:05:00.000Z');
      expect(() => t.completedAt.getTime() - t.startedAt.getTime()).not.toThrow();
    });

    it('rehydrates agent lastSeen as a Date instance', async () => {
      const agentEvent: CoordinationMessage = {
        timestamp: '2026-04-19T12:00:00.000Z',
        agent: 'system',
        message: 'agent registered',
        type: 'agent_registered',
        data: { agentId: 'dev-1', name: 'dev-1', capabilities: ['go'] },
      };
      await log.batchAppend([agentEvent]);
      await log.flush();

      const projects = new Map<string, any>();
      const tasks = new Map<string, any>();
      const briefs = new Map<string, Map<string, string>>();
      const agents = new Map<string, any>();
      await log.restoreFromLog(projects, tasks, briefs, agents);

      const a = agents.get('dev-1');
      expect(a).toBeDefined();
      expect(a.lastSeen).toBeInstanceOf(Date);
      // matches AgentRegistry.ts:262 stale-check pattern
      expect(() => Date.now() - a.lastSeen.getTime()).not.toThrow();
    });

    it('rehydrates file lock lockedAt as a Date instance', async () => {
      const claimEvent: CoordinationMessage = {
        timestamp: '2026-04-19T12:00:00.000Z',
        agent: 'dev-1',
        message: 'file claim',
        type: 'file_claim',
        data: { agentId: 'dev-1', taskId: 'task-1', filePaths: ['src/foo.ts'] },
      };
      await log.batchAppend([claimEvent]);
      await log.flush();

      const projects = new Map<string, any>();
      const tasks = new Map<string, any>();
      const briefs = new Map<string, Map<string, string>>();
      const agents = new Map<string, any>();
      const fileLocks = new Map<string, any>();
      await log.restoreFromLog(projects, tasks, briefs, agents, fileLocks);

      const lock = fileLocks.get('src/foo.ts');
      expect(lock).toBeDefined();
      expect(lock.lockedAt).toBeInstanceOf(Date);
    });
  });
});
