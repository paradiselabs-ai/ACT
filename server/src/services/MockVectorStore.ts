/**
 * MockVectorStore - In-memory testing adapter for VectorMemoryStore
 *
 * For unit tests and local development without Qdrant dependency
 */

import {
  VectorMemoryStore,
  VectorStoreConfig,
  DEFAULT_VECTOR_CONFIG
} from './VectorMemoryStore.js';
import {
  CoordinationMessage,
  AgentProfile,
  CoordinationPattern,
  SearchResult,
  VectorSearchQuery
} from '../types/coordination.js';

interface StoredPoint {
  id: string;
  vector: number[];
  message: CoordinationMessage;
}

export class MockVectorStore extends VectorMemoryStore {
  private config: VectorStoreConfig;
  private points: StoredPoint[] = [];
  private embeddingCache: Map<string, number[]> = new Map();

  constructor(config: Partial<VectorStoreConfig> = {}) {
    super();
    this.config = { ...DEFAULT_VECTOR_CONFIG, ...config, vectorDbType: 'mock' };
  }

  async embed(text: string): Promise<number[]> {
    // Check cache
    if (this.config.cacheEmbeddings && this.embeddingCache.has(text)) {
      return this.embeddingCache.get(text)!;
    }

    // Mock embedding: simple hash-based deterministic embedding
    const embedding = this.generateMockEmbedding(text);

    // Cache
    if (this.config.cacheEmbeddings) {
      if (this.embeddingCache.size >= this.config.maxCacheSize!) {
        const firstKey = this.embeddingCache.keys().next().value;
        if (firstKey !== undefined) {
          this.embeddingCache.delete(firstKey);
        }
      }
      this.embeddingCache.set(text, embedding);
    }

    return embedding;
  }

  async batchEmbed(texts: string[]): Promise<number[][]> {
    return Promise.all(texts.map(text => this.embed(text)));
  }

  async store(message: CoordinationMessage, embedding?: number[]): Promise<void> {
    const vector = embedding || await this.embed(
      `${message.agent}: ${message.message} [${message.type}]`
    );

    const point: StoredPoint = {
      id: `${message.timestamp}_${message.agent}`,
      vector,
      message
    };

    // Replace if exists, otherwise append
    const existingIndex = this.points.findIndex(p => p.id === point.id);
    if (existingIndex >= 0) {
      this.points[existingIndex] = point;
    } else {
      this.points.push(point);
    }
  }

  async batchStore(messages: CoordinationMessage[]): Promise<void> {
    for (const message of messages) {
      await this.store(message);
    }
  }

  async search(
    query: string | VectorSearchQuery,
    limit: number = 10
  ): Promise<SearchResult[]> {
    // Parse query
    const searchQuery = typeof query === 'string'
      ? { query, limit }
      : { ...query, limit: query.limit || limit };

    // Embed query
    const queryVector = await this.embed(searchQuery.query);

    // Filter points
    let filtered = this.points;

    if (searchQuery.agentFilter) {
      filtered = filtered.filter(p => p.message.agent === searchQuery.agentFilter);
    }

    if (searchQuery.typeFilter) {
      filtered = filtered.filter(p => p.message.type === searchQuery.typeFilter);
    }

    if (searchQuery.timeframe) {
      filtered = filtered.filter(p =>
        p.message.timestamp >= searchQuery.timeframe!.start &&
        p.message.timestamp <= searchQuery.timeframe!.end
      );
    }

    // Calculate cosine similarity
    const results = filtered.map(point => ({
      point,
      similarity: this.cosineSimilarity(queryVector, point.vector)
    }));

    // Filter by threshold if specified
    let thresholdFiltered = searchQuery.threshold
      ? results.filter(r => r.similarity >= searchQuery.threshold!)
      : results;

    // Sort by similarity descending
    thresholdFiltered.sort((a, b) => b.similarity - a.similarity);

    // Limit results
    const limited = thresholdFiltered.slice(0, searchQuery.limit || limit);

    // Convert to SearchResult
    return limited.map(r => ({
      message: r.point.message,
      similarity: r.similarity,
      context: undefined
    }));
  }

  async findRelevantPatterns(
    context: string,
    limit: number = 5
  ): Promise<CoordinationPattern[]> {
    const results = await this.search(context, limit);

    return results.map((result, idx) => ({
      id: `pattern_${Date.now()}_${idx}`,
      pattern: result.message.message,
      context: `${result.message.agent} coordination`,
      outcome: 'success' as const,
      participants: [result.message.agent],
      timestamp: result.message.timestamp,
      similarity: result.similarity
    }));
  }

  async getAgentProfile(agentId: string): Promise<AgentProfile> {
    // Get all messages from this agent
    const agentMessages = this.points.filter(p => p.message.agent === agentId);

    if (agentMessages.length === 0) {
      // Return empty profile
      return {
        agentId,
        capabilities: {},
        communicationPatterns: [],
        toolUsage: {},
        synergies: {},
        overallPerformance: {
          totalTasks: 0,
          completedTasks: 0,
          successRate: 0,
          avgTaskTime: 0,
          reliability: 0
        },
        lastUpdated: new Date().toISOString()
      };
    }

    // Analyze message types
    const typeCount: Record<string, number> = {};
    for (const point of agentMessages) {
      typeCount[point.message.type] = (typeCount[point.message.type] || 0) + 1;
    }

    // Build capabilities from message types
    const capabilities: Record<string, any> = {};
    for (const [type, count] of Object.entries(typeCount)) {
      capabilities[type] = {
        successRate: 0.85 + Math.random() * 0.15,
        taskCount: count,
        avgCompletionTime: 3600,
        confidenceScore: count >= 5 ? 0.9 : 0.5,
        evidenceQuality: count >= 10 ? 'strong' : count >= 5 ? 'moderate' : 'weak'
      };
    }

    return {
      agentId,
      capabilities,
      communicationPatterns: [],
      toolUsage: {},
      synergies: {},
      overallPerformance: {
        totalTasks: agentMessages.length,
        completedTasks: agentMessages.length,
        successRate: 0.9,
        avgTaskTime: 3600,
        reliability: 0.95
      },
      lastUpdated: new Date().toISOString()
    };
  }

  async getAgentSynergy(agent1: string, agent2: string): Promise<{
    collaborationCount: number;
    successRate: number;
    strengthAreas: string[];
    weaknessAreas: string[];
  }> {
    // Simple mock implementation
    const agent1Count = this.points.filter(p => p.message.agent === agent1).length;
    const agent2Count = this.points.filter(p => p.message.agent === agent2).length;

    return {
      collaborationCount: Math.min(agent1Count, agent2Count),
      successRate: 0.88,
      strengthAreas: ['coordination'],
      weaknessAreas: []
    };
  }

  async compareAgents(agentIds: string[], taskType: string): Promise<{
    agentId: string;
    matchScore: number;
    successRate: number;
    taskCount: number;
    recommendation: string;
  }[]> {
    const comparisons = [];

    for (const agentId of agentIds) {
      const profile = await this.getAgentProfile(agentId);
      const capability = profile.capabilities[taskType] || {
        successRate: 0.5,
        taskCount: 0,
        confidenceScore: 0
      };

      comparisons.push({
        agentId,
        matchScore: capability.confidenceScore,
        successRate: capability.successRate,
        taskCount: capability.taskCount,
        recommendation: capability.taskCount >= 5
          ? `Strong match (${capability.taskCount} tasks, ${(capability.successRate * 100).toFixed(0)}% success)`
          : `Limited evidence (${capability.taskCount} tasks)`
      });
    }

    return comparisons.sort((a, b) => b.matchScore - a.matchScore);
  }

  async healthCheck(): Promise<boolean> {
    return true; // Mock is always healthy
  }

  async close(): Promise<void> {
    this.points = [];
    this.embeddingCache.clear();
  }

  /**
   * Generate deterministic mock embedding from text
   */
  private generateMockEmbedding(text: string): number[] {
    const dimension = this.config.embeddingDimension!;
    const embedding = new Array(dimension);

    // Simple hash
    let hash = 0;
    for (let i = 0; i < text.length; i++) {
      hash = ((hash << 5) - hash) + text.charCodeAt(i);
      hash = hash & hash;
    }

    // Generate pseudo-random but deterministic embedding
    for (let i = 0; i < dimension; i++) {
      const seed = hash + i;
      embedding[i] = Math.sin(seed) * 0.5 + 0.5;
    }

    // Normalize to unit vector
    const magnitude = Math.sqrt(embedding.reduce((sum, val) => sum + val * val, 0));
    return embedding.map(val => val / magnitude);
  }

  /**
   * Calculate cosine similarity between two vectors
   */
  private cosineSimilarity(a: number[], b: number[]): number {
    if (a.length !== b.length) {
      throw new Error('Vectors must have same dimension');
    }

    let dotProduct = 0;
    let magnitudeA = 0;
    let magnitudeB = 0;

    for (let i = 0; i < a.length; i++) {
      dotProduct += a[i] * b[i];
      magnitudeA += a[i] * a[i];
      magnitudeB += b[i] * b[i];
    }

    magnitudeA = Math.sqrt(magnitudeA);
    magnitudeB = Math.sqrt(magnitudeB);

    if (magnitudeA === 0 || magnitudeB === 0) {
      return 0;
    }

    return dotProduct / (magnitudeA * magnitudeB);
  }

  /**
   * Get all stored points (for testing)
   */
  getAllPoints(): StoredPoint[] {
    return [...this.points];
  }

  /**
   * Clear all stored data (for testing)
   */
  clear(): void {
    this.points = [];
    this.embeddingCache.clear();
  }
}
