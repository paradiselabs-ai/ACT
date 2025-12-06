/**
 * MockVectorStore Unit Tests
 *
 * Tests for in-memory vector store implementation
 */

import { MockVectorStore } from '../services/MockVectorStore.js';
import { CoordinationMessage } from '../types/coordination.js';

describe('MockVectorStore', () => {
  let store: MockVectorStore;

  beforeEach(() => {
    store = new MockVectorStore();
  });

  afterEach(async () => {
    await store.close();
  });

  describe('embed', () => {
    it('should generate deterministic embeddings', async () => {
      const text = 'test message';
      const embedding1 = await store.embed(text);
      const embedding2 = await store.embed(text);

      expect(embedding1).toEqual(embedding2);
      expect(embedding1.length).toBe(384); // default dimension
    });

    it('should cache embeddings when enabled', async () => {
      const text = 'cached test';
      await store.embed(text);
      const cached = await store.embed(text);

      expect(cached).toBeDefined();
    });

    it('should generate different embeddings for different texts', async () => {
      const embedding1 = await store.embed('text one');
      const embedding2 = await store.embed('text two');

      expect(embedding1).not.toEqual(embedding2);
    });
  });

  describe('store and search', () => {
    it('should store and retrieve coordination messages', async () => {
      const message: CoordinationMessage = {
        timestamp: '2025-12-06T00:00:00Z',
        agent: 'test_agent',
        message: 'Test coordination message',
        type: 'coordination'
      };

      await store.store(message);

      const results = await store.search('coordination', 10);
      expect(results.length).toBeGreaterThan(0);
      expect(results[0].message.agent).toBe('test_agent');
    });

    it('should rank results by similarity', async () => {
      const messages: CoordinationMessage[] = [
        {
          timestamp: '2025-12-06T00:00:00Z',
          agent: 'agent1',
          message: 'Implementing vector database',
          type: 'progress_report'
        },
        {
          timestamp: '2025-12-06T00:01:00Z',
          agent: 'agent2',
          message: 'Adding new features',
          type: 'feature_complete'
        },
        {
          timestamp: '2025-12-06T00:02:00Z',
          agent: 'agent3',
          message: 'Vector store implementation complete',
          type: 'progress_report'
        }
      ];

      await store.batchStore(messages);

      const results = await store.search('vector implementation', 3);
      expect(results.length).toBe(3);
      // Most relevant should be first
      expect(results[0].similarity).toBeGreaterThan(results[1].similarity);
    });

    it('should filter by agent', async () => {
      const messages: CoordinationMessage[] = [
        {
          timestamp: '2025-12-06T00:00:00Z',
          agent: 'agent1',
          message: 'Message from agent1',
          type: 'coordination'
        },
        {
          timestamp: '2025-12-06T00:01:00Z',
          agent: 'agent2',
          message: 'Message from agent2',
          type: 'coordination'
        }
      ];

      await store.batchStore(messages);

      const results = await store.search({
        query: 'message',
        agentFilter: 'agent1'
      });

      expect(results.length).toBe(1);
      expect(results[0].message.agent).toBe('agent1');
    });

    it('should filter by message type', async () => {
      const messages: CoordinationMessage[] = [
        {
          timestamp: '2025-12-06T00:00:00Z',
          agent: 'agent1',
          message: 'Progress update',
          type: 'progress_report'
        },
        {
          timestamp: '2025-12-06T00:01:00Z',
          agent: 'agent2',
          message: 'Feature done',
          type: 'feature_complete'
        }
      ];

      await store.batchStore(messages);

      const results = await store.search({
        query: 'update',
        typeFilter: 'progress_report'
      });

      expect(results.length).toBe(1);
      expect(results[0].message.type).toBe('progress_report');
    });
  });

  describe('getAgentProfile', () => {
    it('should return empty profile for unknown agent', async () => {
      const profile = await store.getAgentProfile('unknown_agent');

      expect(profile.agentId).toBe('unknown_agent');
      expect(profile.overallPerformance.totalTasks).toBe(0);
    });

    it('should derive profile from coordination history', async () => {
      const messages: CoordinationMessage[] = [
        {
          timestamp: '2025-12-06T00:00:00Z',
          agent: 'test_agent',
          message: 'Completed task 1',
          type: 'progress_report'
        },
        {
          timestamp: '2025-12-06T00:01:00Z',
          agent: 'test_agent',
          message: 'Completed task 2',
          type: 'progress_report'
        },
        {
          timestamp: '2025-12-06T00:02:00Z',
          agent: 'test_agent',
          message: 'Feature complete',
          type: 'feature_complete'
        }
      ];

      await store.batchStore(messages);

      const profile = await store.getAgentProfile('test_agent');

      expect(profile.agentId).toBe('test_agent');
      expect(profile.overallPerformance.totalTasks).toBe(3);
      expect(profile.capabilities['progress_report']).toBeDefined();
      expect(profile.capabilities['progress_report'].taskCount).toBe(2);
    });
  });

  describe('findRelevantPatterns', () => {
    it('should find coordination patterns', async () => {
      const messages: CoordinationMessage[] = [
        {
          timestamp: '2025-12-06T00:00:00Z',
          agent: 'agent1',
          message: 'Successfully implemented feature X using pattern Y',
          type: 'architecture_decision'
        },
        {
          timestamp: '2025-12-06T00:01:00Z',
          agent: 'agent2',
          message: 'Used pattern Y for feature Z',
          type: 'architecture_decision'
        }
      ];

      await store.batchStore(messages);

      const patterns = await store.findRelevantPatterns('pattern Y implementation', 5);

      expect(patterns.length).toBeGreaterThan(0);
      expect(patterns[0].pattern).toContain('pattern Y');
      expect(patterns[0].similarity).toBeDefined();
    });
  });

  describe('healthCheck', () => {
    it('should always return true for mock store', async () => {
      const healthy = await store.healthCheck();
      expect(healthy).toBe(true);
    });
  });

  describe('clear', () => {
    it('should remove all stored data', async () => {
      const message: CoordinationMessage = {
        timestamp: '2025-12-06T00:00:00Z',
        agent: 'agent1',
        message: 'Test',
        type: 'coordination'
      };

      await store.store(message);
      expect(store.getAllPoints().length).toBe(1);

      store.clear();
      expect(store.getAllPoints().length).toBe(0);
    });
  });
});
