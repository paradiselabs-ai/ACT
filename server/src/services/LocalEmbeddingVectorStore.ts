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
  private mode: 'real' | 'hash' | 'unknown' = 'unknown';
  private modeAnnounced: boolean = false;

  constructor(config: Partial<VectorStoreConfig> = {}) {
    super();
    this.config = { ...DEFAULT_VECTOR_CONFIG, ...config, vectorDbType: 'mock' };
    // Warm up the model in the background so first embed() is fast
    getEmbeddingPipeline()
      .then(() => {
        this.modelReady = true;
        if (this.mode !== 'real') {
          this.mode = 'real';
          if (!this.modeAnnounced) {
            logger.info('[PVM] real embeddings active (model=Xenova/all-MiniLM-L6-v2, dim=384)');
            this.modeAnnounced = true;
          }
        }
      })
      .catch((err: unknown) => {
        if (this.mode !== 'hash') {
          this.mode = 'hash';
          if (!this.modeAnnounced) {
            const reason = (err instanceof Error ? err.message : String(err)).slice(0, 200);
            logger.warn(`[PVM] HASH FALLBACK ACTIVE — embeddings are NOT semantic. Reason: ${reason}`);
            logger.warn('[PVM] Search results will rank by hashCode proximity, not meaning. Install Xenova/all-MiniLM-L6-v2 ONNX model in ~/.cache/huggingface/hub to fix.');
            this.modeAnnounced = true;
          }
        }
      });
  }

  public getMode(): 'real' | 'hash' | 'unknown' {
    return this.mode;
  }

  async embed(text: string): Promise<number[]> {
    if (this.config.cacheEmbeddings && this.embeddingCache.has(text)) {
      return this.embeddingCache.get(text)!;
    }

    let embedding: number[];

    if (this.modelReady) {
      embedding = await this.realEmbed(text);
      if (this.mode !== 'real') {
        this.mode = 'real';
        if (!this.modeAnnounced) {
          logger.info('[PVM] real embeddings active (model=Xenova/all-MiniLM-L6-v2, dim=384)');
          this.modeAnnounced = true;
        }
      }
    } else {
      // Model not yet loaded — try loading now, fall back to hash if unavailable
      try {
        await getEmbeddingPipeline();
        this.modelReady = true;
        embedding = await this.realEmbed(text);
        if (this.mode !== 'real') {
          this.mode = 'real';
          if (!this.modeAnnounced) {
            logger.info('[PVM] real embeddings active (model=Xenova/all-MiniLM-L6-v2, dim=384)');
            this.modeAnnounced = true;
          }
        }
      } catch (err) {
        embedding = this.hashEmbed(text);
        if (this.mode !== 'hash') {
          this.mode = 'hash';
          if (!this.modeAnnounced) {
            const reason = (err instanceof Error ? err.message : String(err)).slice(0, 200);
            logger.warn(`[PVM] HASH FALLBACK ACTIVE — embeddings are NOT semantic. Reason: ${reason}`);
            logger.warn('[PVM] Search results will rank by hashCode proximity, not meaning. Install Xenova/all-MiniLM-L6-v2 ONNX model in ~/.cache/huggingface/hub to fix.');
            this.modeAnnounced = true;
          }
        }
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

    logger.debug(`[PVM] search mode=${this.mode} query="${searchQuery.query.slice(0, 60)}" limit=${searchQuery.limit ?? 'default'}`);

    const queryVector = await this.embed(searchQuery.query);

    // Project filter — opt-in. Caller passes searchQuery.projectName to narrow
    // results to one project plus all __global__ events. When omitted (the
    // default), search returns cross-project results.
    const projectScope = (searchQuery as any).projectName as string | undefined;
    let filtered = projectScope
      ? this.points.filter(p =>
          (p.message as any).projectName === projectScope ||
          (p.message as any).projectName === '__global__')
      : this.points;
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

    const outcomes = this.lookupTaskOutcomes(agentId);

    // Per-task-type capability metrics derived from real outcomes
    const outcomesByType: Record<string, typeof outcomes> = {};
    for (const o of outcomes) {
      const key = o.type ?? 'unknown';
      (outcomesByType[key] ||= []).push(o);
    }

    // Also include event-type counts so types without lifecycle data still show
    const typeCount: Record<string, number> = {};
    for (const point of agentMessages) {
      typeCount[point.message.type] = (typeCount[point.message.type] || 0) + 1;
    }

    const capabilities: Record<string, any> = {};
    // Capability per task-type for which we have completed-task outcomes
    for (const [type, taskOutcomes] of Object.entries(outcomesByType)) {
      const validated = taskOutcomes.filter(o => o.validated);
      const passed = validated.filter(o => o.passed);
      const durations = taskOutcomes.map(o => o.durationMs).filter((d): d is number => d !== null);
      capabilities[type] = {
        successRate: validated.length > 0 ? passed.length / validated.length : 0,
        taskCount: taskOutcomes.length,
        avgCompletionTime: durations.length > 0
          ? Math.round(durations.reduce((s, d) => s + d, 0) / durations.length / 1000)
          : 0,
        confidenceScore: Math.min(1.0, taskOutcomes.length / 10),
        evidenceQuality: taskOutcomes.length >= 10 ? 'strong'
                       : taskOutcomes.length >= 5  ? 'moderate'
                       : 'weak'
      };
    }
    // Event-type counts for types without lifecycle data
    for (const [type, count] of Object.entries(typeCount)) {
      if (!capabilities[type]) {
        capabilities[type] = {
          successRate: 0,
          taskCount: count,
          avgCompletionTime: 0,
          confidenceScore: count >= 5 ? 0.5 : 0.2,
          evidenceQuality: count >= 10 ? 'moderate' : 'weak'
        };
      }
    }

    // Overall performance from the same outcomes
    const completedTasks = outcomes.length;
    const validatedTasks = outcomes.filter(o => o.validated);
    const passedTasks = validatedTasks.filter(o => o.passed);
    const avgDurationMs = (() => {
      const d = outcomes.map(o => o.durationMs).filter((x): x is number => x !== null);
      return d.length > 0 ? d.reduce((s, x) => s + x, 0) / d.length : 0;
    })();

    return {
      agentId,
      capabilities,
      communicationPatterns: [],
      toolUsage: {},
      synergies: {},
      overallPerformance: {
        totalTasks: agentMessages.filter(p => p.message.type?.startsWith('task_')).length,
        completedTasks,
        successRate: validatedTasks.length > 0 ? passedTasks.length / validatedTasks.length : 0,
        avgTaskTime: Math.round(avgDurationMs / 1000),
        reliability: completedTasks > 0 ? passedTasks.length / completedTasks : 0
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
    // Collect taskIds each agent participated in
    const tasksOf = (agentId: string): Set<string> => {
      const set = new Set<string>();
      for (const p of this.points) {
        const m = p.message as any;
        const data = m.data || {};
        const taskId = data.taskId || data.task?.id;
        if (!taskId) continue;
        if (m.agent === agentId || data.agentId === agentId || data.assignedAgent === agentId) {
          set.add(taskId);
        }
      }
      return set;
    };
    const t1 = tasksOf(agent1);
    const t2 = tasksOf(agent2);
    const shared: string[] = [];
    for (const id of t1) if (t2.has(id)) shared.push(id);

    // Outcomes for the shared task set
    const validationByTask: Map<string, boolean> = new Map();
    const typeByTask: Map<string, string | undefined> = new Map();
    for (const p of this.points) {
      const m = p.message as any;
      const data = m.data || {};
      const taskId = data.taskId || data.task?.id;
      if (!taskId || !shared.includes(taskId)) continue;
      if (m.type === 'task_validated') validationByTask.set(taskId, true);
      if (m.type === 'task_validation_failed') validationByTask.set(taskId, false);
      if (m.type === 'task_created') typeByTask.set(taskId, data.task?.type || data.requiredCapabilities?.[0]);
    }
    const validatedShared = shared.filter(id => validationByTask.has(id));
    const passedShared = validatedShared.filter(id => validationByTask.get(id));

    // Strength / weakness from per-type pass rate
    const perTypePass: Record<string, { pass: number; total: number }> = {};
    for (const id of validatedShared) {
      const type = typeByTask.get(id) ?? 'unknown';
      (perTypePass[type] ||= { pass: 0, total: 0 }).total++;
      if (validationByTask.get(id)) perTypePass[type].pass++;
    }
    const strengthAreas = Object.entries(perTypePass)
      .filter(([_, v]) => v.total >= 2 && v.pass / v.total >= 0.75)
      .map(([k]) => k);
    const weaknessAreas = Object.entries(perTypePass)
      .filter(([_, v]) => v.total >= 2 && v.pass / v.total < 0.5)
      .map(([k]) => k);

    return {
      collaborationCount: shared.length,
      successRate: validatedShared.length > 0 ? passedShared.length / validatedShared.length : 0,
      strengthAreas,
      weaknessAreas
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
      const cap = profile.capabilities[taskType] || { successRate: 0, taskCount: 0, confidenceScore: 0 };
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

  /**
   * For an agent, walk this.points and pair task_completed events with their
   * task_validated counterparts (joined by taskId in event.data). Returns
   * per-task outcomes for use by getAgentProfile / getAgentSynergy.
   */
  private lookupTaskOutcomes(agentId: string): {
    taskId: string;
    type: string | undefined;
    completed: boolean;
    validated: boolean;
    passed: boolean;
    durationMs: number | null;
  }[] {
    const assignedByTask: Map<string, number> = new Map();
    const completedByTask: Map<string, number> = new Map();
    const validationByTask: Map<string, boolean> = new Map();
    // Type lives on task_created (data IS the Task object). task_completed/assigned
    // events only carry { taskId, agentId } and have no type field, so without
    // this join every outcome's type defaulted to 'unknown' and compareAgents
    // returned taskCount=0 for any taskType filter.
    const typeByTask: Map<string, string | undefined> = new Map();

    for (const p of this.points) {
      const m = p.message as any;
      const data = m.data || {};
      const taskId = data.taskId || data.task?.id || data.id;
      if (!taskId) continue;

      const ms = new Date(m.timestamp).getTime();
      if (isNaN(ms)) continue;

      if (m.type === 'task_created') {
        const t = data.metadata?.taskType
              || data.task?.metadata?.taskType
              || data.type
              || data.task?.type
              || data.requiredCapabilities?.[0]
              || data.task?.requiredCapabilities?.[0];
        if (t && !typeByTask.has(taskId)) typeByTask.set(taskId, t);
      } else if (m.type === 'task_assigned' && (data.assignedAgent === agentId || data.agentId === agentId)) {
        assignedByTask.set(taskId, ms);
      } else if (m.type === 'task_completed' && (data.agentId === agentId || m.agent === agentId)) {
        completedByTask.set(taskId, ms);
      } else if (m.type === 'task_validated' || m.type === 'task_validation_failed') {
        validationByTask.set(taskId, m.type === 'task_validated');
      }
    }

    const outcomes = [];
    for (const [taskId, completedMs] of completedByTask.entries()) {
      const assigned = assignedByTask.get(taskId);
      const validated = validationByTask.has(taskId);
      const passed = validationByTask.get(taskId) ?? false;
      outcomes.push({
        taskId,
        type: typeByTask.get(taskId),
        completed: true,
        validated,
        passed,
        durationMs: assigned !== undefined ? completedMs - assigned : null
      });
    }
    return outcomes;
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
