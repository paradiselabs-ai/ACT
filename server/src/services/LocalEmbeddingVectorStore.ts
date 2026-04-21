/**
 * LocalEmbeddingVectorStore
 *
 * Drop-in replacement for MockVectorStore that uses real semantic embeddings
 * via @xenova/transformers + all-MiniLM-L6-v2 (~25MB, runs fully locally, no API key).
 *
 * The model is downloaded to ~/.cache/huggingface/hub on first use (Node.js only).
 * After the first download, startup takes ~1-2 seconds to load the model.
 *
 * Keeps everything from MockVectorStore except generateMockEmbedding() —
 * cosine similarity, filtering, agent profiles, and storage are unchanged.
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
import { logger } from '../utils/logger.js';

// Dynamically import transformers.js (ESM-only package)
// We lazy-load it on first embed() call to avoid blocking server startup.
let pipeline: any = null;
let embeddingPipeline: any = null;

async function getEmbeddingPipeline(): Promise<any> {
  if (embeddingPipeline) return embeddingPipeline;

  if (!pipeline) {
    const { pipeline: pipelineFn, env } = await import('@xenova/transformers');
    // Use local cache (default: ~/.cache/huggingface/hub)
    env.allowLocalModels = true;
    // env.cacheDir left as default (~/.cache/huggingface/hub)
    pipeline = pipelineFn;
  }

  logger.info('[PVM] Loading all-MiniLM-L6-v2 embedding model...');
  embeddingPipeline = await pipeline('feature-extraction', 'Xenova/all-MiniLM-L6-v2', {
    quantized: true // ~25MB quantized vs ~90MB fp32
  });
  logger.info('[PVM] Embedding model loaded ✅');
  return embeddingPipeline;
}

interface StoredPoint {
  id: string;
  vector: number[];
  message: CoordinationMessage;
}

export class LocalEmbeddingVectorStore extends VectorMemoryStore {
  private config: VectorStoreConfig;
  private points: StoredPoint[] = [];
  private embeddingCache: Map<string, number[]> = new Map();
  private modelReady = false;

  constructor(config: Partial<VectorStoreConfig> = {}) {
    super();
    this.config = { ...DEFAULT_VECTOR_CONFIG, ...config, vectorDbType: 'mock' };
    // Warm up the model in the background so first embed() is fast
    this.warmup();
  }

  private async warmup(): Promise<void> {
    try {
      await getEmbeddingPipeline();
      this.modelReady = true;
    } catch (err: any) {
      logger.warn(`[PVM] Could not load embedding model: ${err.message}. Falling back to hash embeddings.`);
    }
  }

  async embed(text: string): Promise<number[]> {
    if (this.config.cacheEmbeddings && this.embeddingCache.has(text)) {
      return this.embeddingCache.get(text)!;
    }

    let embedding: number[];

    if (this.modelReady) {
      embedding = await this.realEmbed(text);
    } else {
      // Model not yet loaded — try loading now, fall back to hash if unavailable
      try {
        await getEmbeddingPipeline();
        this.modelReady = true;
        embedding = await this.realEmbed(text);
      } catch {
        embedding = this.hashEmbed(text);
      }
    }

    if (this.config.cacheEmbeddings) {
      if (this.embeddingCache.size >= (this.config.maxCacheSize ?? 1000)) {
        const firstKey = this.embeddingCache.keys().next().value;
        if (firstKey !== undefined) this.embeddingCache.delete(firstKey);
      }
      this.embeddingCache.set(text, embedding);
    }

    return embedding;
  }

  private async realEmbed(text: string): Promise<number[]> {
    const pipe = await getEmbeddingPipeline();
    // The model returns a Tensor; we mean-pool across the token dimension
    const output = await pipe(text, { pooling: 'mean', normalize: true });
    // output.data is a Float32Array
    return Array.from(output.data as Float32Array);
  }

  /** Deterministic hash-based fallback (same as MockVectorStore, only used if model fails) */
  private hashEmbed(text: string): number[] {
    const dimension = this.config.embeddingDimension ?? 384;
    let hash = 0;
    for (let i = 0; i < text.length; i++) {
      hash = ((hash << 5) - hash) + text.charCodeAt(i);
      hash = hash & hash;
    }
    const embedding = new Array(dimension);
    for (let i = 0; i < dimension; i++) {
      embedding[i] = Math.sin(hash + i) * 0.5 + 0.5;
    }
    const mag = Math.sqrt(embedding.reduce((s, v) => s + v * v, 0));
    return embedding.map(v => v / mag);
  }

  async batchEmbed(texts: string[]): Promise<number[][]> {
    return Promise.all(texts.map(t => this.embed(t)));
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
    const existingIdx = this.points.findIndex(p => p.id === point.id);
    if (existingIdx >= 0) {
      this.points[existingIdx] = point;
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
    const searchQuery = typeof query === 'string'
      ? { query, limit }
      : { ...query, limit: query.limit || limit };

    const queryVector = await this.embed(searchQuery.query);

    let filtered = this.points;
    if (!searchQuery.includeMeta) {
      // Default: exclude harness/tooling-state events so stale "CLI broken"
      // claims don't leak into agent task context. Events without an explicit
      // scope are treated as project (pre-existing indexed events).
      filtered = filtered.filter(p => p.message.scope !== 'meta');
    }
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

    const results = filtered.map(point => ({
      point,
      similarity: this.cosineSimilarity(queryVector, point.vector)
    }));

    let thresholdFiltered = searchQuery.threshold
      ? results.filter(r => r.similarity >= searchQuery.threshold!)
      : results;

    thresholdFiltered.sort((a, b) => b.similarity - a.similarity);

    return thresholdFiltered.slice(0, searchQuery.limit || limit).map(r => ({
      message: r.point.message,
      similarity: r.similarity,
      context: undefined
    }));
  }

  async findRelevantPatterns(context: string, limit: number = 5): Promise<CoordinationPattern[]> {
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
    const agentMessages = this.points.filter(p => p.message.agent === agentId);

    if (agentMessages.length === 0) {
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

    const typeCount: Record<string, number> = {};
    for (const point of agentMessages) {
      typeCount[point.message.type] = (typeCount[point.message.type] || 0) + 1;
    }

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
    const a1 = this.points.filter(p => p.message.agent === agent1).length;
    const a2 = this.points.filter(p => p.message.agent === agent2).length;
    return {
      collaborationCount: Math.min(a1, a2),
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
      const cap = profile.capabilities[taskType] || { successRate: 0.5, taskCount: 0, confidenceScore: 0 };
      comparisons.push({
        agentId,
        matchScore: cap.confidenceScore,
        successRate: cap.successRate,
        taskCount: cap.taskCount,
        recommendation: cap.taskCount >= 5
          ? `Strong match (${cap.taskCount} tasks, ${(cap.successRate * 100).toFixed(0)}% success)`
          : `Limited evidence (${cap.taskCount} tasks)`
      });
    }
    return comparisons.sort((a, b) => b.matchScore - a.matchScore);
  }

  async healthCheck(): Promise<boolean> {
    return true;
  }

  async close(): Promise<void> {
    this.points = [];
    this.embeddingCache.clear();
  }

  getAllPoints(): StoredPoint[] {
    return [...this.points];
  }

  clear(): void {
    this.points = [];
    this.embeddingCache.clear();
  }

  private cosineSimilarity(a: number[], b: number[]): number {
    if (a.length !== b.length) {
      // Dimension mismatch (hash fallback vs real embeddings) — return 0
      return 0;
    }
    let dot = 0, magA = 0, magB = 0;
    for (let i = 0; i < a.length; i++) {
      dot  += a[i] * b[i];
      magA += a[i] * a[i];
      magB += b[i] * b[i];
    }
    magA = Math.sqrt(magA);
    magB = Math.sqrt(magB);
    return (magA === 0 || magB === 0) ? 0 : dot / (magA * magB);
  }
}
