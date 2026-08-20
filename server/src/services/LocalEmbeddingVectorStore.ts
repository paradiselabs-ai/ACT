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

import fs from 'fs/promises';
import path from 'path';
import { createHash } from 'crypto';
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

const EMBEDDING_MODEL = 'Xenova/all-MiniLM-L6-v2';

// Persistent embedding cache entry. The sidecar JSONL is a derived,
// rebuildable artifact — the ChronLog remains the single source of truth;
// deleting the sidecar just means re-embedding on the next drain.
interface SidecarEntry {
  key: string;      // `${timestamp}_${agent}` — same identity as StoredPoint.id
  textHash: string; // sha1 of the exact embedded string
  model: string;    // invalidates the cache wholesale on model swap
  dim: number;
  vector: number[];
}

function sha1(text: string): string {
  return createHash('sha1').update(text).digest('hex');
}

export class LocalEmbeddingVectorStore extends VectorMemoryStore {
  private config: VectorStoreConfig;
  private points: StoredPoint[] = [];
  private embeddingCache: Map<string, number[]> = new Map();
  private modelReady = false;
  private mode: 'real' | 'hash' | 'unknown' = 'unknown';
  private modeAnnounced: boolean = false;
  private sidecarPath: string;
  private sidecar: Map<string, SidecarEntry> | null = null; // lazy-loaded on first store()
  private sidecarStats = { fromCache: 0, freshEmbeds: 0 };
  private attributionStats: { validations: number; unattributed: number; computedAt: string } | null = null;

  constructor(config: Partial<VectorStoreConfig> & { sidecarPath?: string } = {}) {
    super();
    this.config = { ...DEFAULT_VECTOR_CONFIG, ...config, vectorDbType: 'mock' };
    this.sidecarPath = config.sidecarPath ?? './data/pvm-vectors.jsonl';
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

    // Only cache real embeddings — a cached hash vector would be served on a
    // later cache hit even after the model recovers (and could then leak into
    // the persistent sidecar).
    if (this.config.cacheEmbeddings && this.mode === 'real') {
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
    const id = `${message.timestamp}_${message.agent}`;
    let vector = embedding;

    if (!vector) {
      const text = `${message.agent}: ${message.message} [${message.type}]`;
      await this.ensureSidecarLoaded();
      const hash = sha1(text);
      const cached = this.sidecar!.get(id);
      if (cached && cached.textHash === hash && cached.model === EMBEDDING_MODEL) {
        vector = cached.vector;
        this.sidecarStats.fromCache++;
      } else {
        vector = await this.embed(text);
        this.sidecarStats.freshEmbeds++;
        // Persist only real embeddings — a hash-fallback vector written to
        // disk would permanently poison the index even after the model heals.
        if (this.mode === 'real') {
          const entry: SidecarEntry = { key: id, textHash: hash, model: EMBEDDING_MODEL, dim: vector.length, vector };
          this.sidecar!.set(id, entry);
          await this.sidecarAppend(entry);
        }
      }
    }

    const point: StoredPoint = {
      id,
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

  /** Cumulative sidecar counters; callers diff around batchStore for per-drain numbers. */
  public getSidecarStats(): { fromCache: number; freshEmbeds: number } {
    return { ...this.sidecarStats };
  }

  private async ensureSidecarLoaded(): Promise<void> {
    if (this.sidecar) return;
    const map = new Map<string, SidecarEntry>();
    try {
      const content = await fs.readFile(this.sidecarPath, 'utf-8');
      for (const line of content.split('\n')) {
        if (!line.trim()) continue;
        try {
          const e = JSON.parse(line) as SidecarEntry;
          // last-write-wins per key
          if (e.key && e.textHash && e.model && Array.isArray(e.vector)) map.set(e.key, e);
        } catch {
          // tolerate a truncated final line (crash mid-append)
        }
      }
    } catch (err: any) {
      if (err.code !== 'ENOENT') {
        logger.warn(`[PVM] sidecar load failed (${err.message}) — starting with empty cache`);
      }
    }
    this.sidecar = map;
    if (map.size > 0) {
      logger.info(`[PVM] loaded ${map.size} cached embeddings from ${this.sidecarPath}`);
    }
  }

  private async sidecarAppend(entry: SidecarEntry): Promise<void> {
    try {
      await fs.mkdir(path.dirname(this.sidecarPath), { recursive: true });
      await fs.appendFile(this.sidecarPath, JSON.stringify(entry) + '\n', 'utf-8');
    } catch (err: any) {
      // Cache-only artifact — never fatal, worst case is a re-embed next boot
      logger.warn(`[PVM] sidecar append failed: ${err.message}`);
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

    // Capability vocabulary comes from what the agent REGISTERED with, and from
    // nothing else. Bucketing on the indexed event's `type` used to leak ChronLog
    // event names into this map ("coordination: 0% success over 460 tasks"),
    // which reads to a router as an agent that failed 460 tasks.
    const capabilities: Record<string, any> = {};
    for (const cap of this.registeredCapabilities(agentId)) {
      const relevant = outcomes.filter(o => o.capabilities.some(c => c.toLowerCase() === cap));
      // No outcomes touching this capability => no bucket. An empty bucket would
      // be a synthetic "0% over 0 tasks", which is worse than saying nothing.
      if (relevant.length === 0) continue;
      const validated = relevant.filter(o => o.validated);
      const passed = validated.filter(o => o.passed);
      const durations = relevant.map(o => o.durationMs).filter((d): d is number => d !== null);
      capabilities[cap] = {
        successRate: validated.length > 0 ? passed.length / validated.length : 0,
        taskCount: relevant.length,
        // Seconds. Duration needs an assignment timestamp to subtract from; with
        // none the field is omitted entirely — a 0 here would read as "instant"
        // rather than "unknown". Sub-second averages keep two decimals for the
        // same reason: a measured 0.04s must not print as a bare 0.
        ...(durations.length > 0
          ? { avgCompletionTime: Number((durations.reduce((s, d) => s + d, 0) / durations.length / 1000).toFixed(2)) }
          : {}),
        confidenceScore: Math.min(1.0, relevant.length / 10),
        evidenceQuality: relevant.length >= 10 ? 'strong'
                       : relevant.length >= 5  ? 'moderate'
                       : 'weak'
      };
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

  /**
   * The capability tags an agent registered with, lowercased and de-duplicated.
   * This is the ONLY vocabulary allowed into an agent profile's capability map.
   */
  private registeredCapabilities(agentId: string): string[] {
    const caps: string[] = [];
    const seen = new Set<string>();
    for (const p of this.points) {
      const m = p.message as any;
      if (m.type !== 'agent_registered') continue;
      const data = m.data || {};
      if ((data.agentId || data.name || m.agent) !== agentId) continue;
      for (const c of (data.capabilities || [])) {
        const key = String(c).toLowerCase();
        if (key && !seen.has(key)) { seen.add(key); caps.push(key); }
      }
    }
    return caps;
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
      // Capability buckets are keyed by lowercased registered capability tag.
      const cap = profile.capabilities[taskType]
        || profile.capabilities[taskType.toLowerCase()]
        || { successRate: 0, taskCount: 0, confidenceScore: 0 };
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

  // ─── Routing brief ───────────────────────────────────────────────────────
  // Always-on coordination evidence for the Planner: which swarm COMPOSITIONS
  // worked on past projects, per-role track records, and role-pair history.
  // Every fact carries a sample count + signal label (high ≥10 / moderate ≥5 /
  // low <5 — the same thresholds getAgentProfile uses for evidenceQuality), so
  // thin data is harmless (labeled low) and rich data is trusted. The Planner
  // reasons from these facts; we do NOT synthesize prose here.

  private roleOfAgent(agentId: string, capabilities: string[] = []): string {
    const prefix = (agentId || '').split('-')[0].toLowerCase();
    const byPrefix: Record<string, string> = {
      dev: 'developer', developer: 'developer',
      backend: 'backend_dev', be: 'backend_dev',
      frontend: 'frontend_dev', fe: 'frontend_dev',
      qa: 'qa_engineer', researcher: 'researcher',
    };
    if (byPrefix[prefix]) return byPrefix[prefix];
    const caps = capabilities.map(c => String(c).toLowerCase());
    if (caps.some(c => ['react', 'vue', 'svelte', 'css', 'tailwind', 'html', 'a11y'].includes(c))) return 'frontend_dev';
    if (caps.some(c => ['api', 'rest', 'db', 'sql', 'postgres', 'auth', 'middleware'].includes(c))) return 'backend_dev';
    if (caps.some(c => ['documentation', 'analysis', 'research'].includes(c))) return 'researcher';
    if (caps.some(c => ['testing', 'pytest', 'jest', 'playwright', 'cypress'].includes(c))) return 'qa_engineer';
    return 'developer';
  }

  private signal(count: number): 'high' | 'moderate' | 'low' {
    return count >= 10 ? 'high' : count >= 5 ? 'moderate' : 'low';
  }

  private projectOf(m: any): string {
    return m.projectName || m.data?.projectName || m.scope || '__global__';
  }

  /**
   * Roll up indexed events by project into {composition, outcomes} views.
   * Composition (role counts) comes from agent_registered; outcomes from
   * task_assigned (who) joined to task_validated/failed (pass/fail) by taskId.
   */
  getProjectOutcomes(): {
    project: string;
    composition: Record<string, number>;
    capabilities: string[];
    taskCount: number;
    passed: number;
    kickbacks: number;
    passRate: number;
    perRole: Record<string, { tasks: number; passed: number }>;
    topGaps: string[];
  }[] {
    const byProject: Map<string, StoredPoint[]> = new Map();
    for (const p of this.points) {
      const proj = this.projectOf(p.message as any);
      let arr = byProject.get(proj);
      if (!arr) { arr = []; byProject.set(proj, arr); }
      arr.push(p);
    }

    const out = [];
    let totalValidations = 0;
    let totalUnattributed = 0;
    for (const [project, pts] of byProject.entries()) {
      const roleOfAgentId: Map<string, string> = new Map();
      const composition: Record<string, number> = {};
      const capSet = new Set<string>();

      for (const p of pts) {
        const m = p.message as any;
        const data = m.data || {};
        if (m.type === 'agent_registered') {
          const aid = data.agentId || data.name || m.agent;
          const caps: string[] = data.capabilities || [];
          caps.forEach(c => capSet.add(c));
          if (aid && !roleOfAgentId.has(aid)) {
            const role = this.roleOfAgent(aid, caps);
            roleOfAgentId.set(aid, role);
            composition[role] = (composition[role] || 0) + 1;
          }
        } else if (m.type === 'task_created') {
          const rc: string[] = data.requiredCapabilities || data.task?.requiredCapabilities || [];
          rc.forEach(c => capSet.add(c));
        }
      }

      const assignedAgent: Map<string, string> = new Map();
      // Who actually did the work. task_completed carries data.agentId on every
      // event; task_assigned is frequently payload-less in real history (24
      // validations vs 5 usable assignment records), so completion is the
      // authoritative source and assignment is only a fallback.
      const completedAgent: Map<string, string> = new Map();
      const validation: Map<string, boolean> = new Map();
      const gaps: string[] = [];
      for (const p of pts) {
        const m = p.message as any;
        const data = m.data || {};
        const taskId = data.taskId || data.task?.id || data.id;
        if (!taskId) continue;
        if (m.type === 'task_completed' || m.type === 'task_failed') {
          const worker = data.agentId || (m.agent !== 'system' ? m.agent : undefined);
          if (worker && !completedAgent.has(taskId)) completedAgent.set(taskId, worker);
        } else if (m.type === 'task_assigned') {
          assignedAgent.set(taskId, data.assignedAgent || data.agentId || m.agent);
        } else if (m.type === 'task_validated') {
          validation.set(taskId, true);
        } else if (m.type === 'task_validation_failed') {
          validation.set(taskId, false);
          if (data.gaps) gaps.push(String(data.gaps).slice(0, 120));
        }
      }

      const perRole: Record<string, { tasks: number; passed: number }> = {};
      let passed = 0, kickbacks = 0, validatedCount = 0;
      for (const [taskId, didPass] of validation.entries()) {
        validatedCount++;
        if (didPass) passed++; else kickbacks++;
        const aid = completedAgent.get(taskId) ?? assignedAgent.get(taskId);
        const role = aid ? (roleOfAgentId.get(aid) || this.roleOfAgent(aid)) : 'unknown';
        (perRole[role] ||= { tasks: 0, passed: 0 }).tasks++;
        if (didPass) perRole[role].passed++;
        if (!aid) totalUnattributed++;
        totalValidations++;
      }

      out.push({
        project,
        composition,
        capabilities: Array.from(capSet),
        taskCount: validatedCount,
        passed,
        kickbacks,
        passRate: validatedCount > 0 ? passed / validatedCount : 0,
        perRole,
        topGaps: gaps.slice(0, 2),
      });
    }

    // Unattributable validations are dropped from the brief text (the 'unknown'
    // role is filtered there), so without this counter they vanish silently —
    // exactly how the attribution bug stayed invisible. Surfaced via
    // getAttributionStats() -> /api/pvm/status.
    this.attributionStats = {
      validations: totalValidations,
      unattributed: totalUnattributed,
      computedAt: new Date().toISOString(),
    };
    if (totalUnattributed > 0) {
      logger.info(`[PVM] attribution: ${totalUnattributed}/${totalValidations} validated tasks have no resolvable worker (no task_completed and no task_assigned payload)`);
    }
    return out;
  }

  /** Attribution health from the last getProjectOutcomes() pass (null before the first). */
  public getAttributionStats(): { validations: number; unattributed: number; computedAt: string } | null {
    return this.attributionStats;
  }

  /**
   * Build the Planner's routing brief for a new project. Picks similar past
   * projects by capability overlap, plus cross-project per-role track records
   * and role-pair history. Returns structured data + a ready-to-inject text
   * block where every line is confidence-labeled.
   */
  async getRoutingBrief(description: string, capabilities: string[] = []): Promise<{
    text: string;
    similarProjects: any[];
    perRole: any[];
    rolePairs: any[];
  }> {
    const projects = this.getProjectOutcomes();
    const reqCaps = new Set(capabilities.map(c => String(c).toLowerCase()));

    // Similar projects: named projects with data, ranked by capability overlap
    // then task volume. (__global__ is the indexer's catch-all, not a project.)
    const named = projects.filter(p => p.project !== '__global__' && p.taskCount > 0);
    const scored = named
      .map(p => ({ p, overlap: p.capabilities.filter(c => reqCaps.has(String(c).toLowerCase())).length }))
      .sort((a, b) => (b.overlap - a.overlap) || (b.p.taskCount - a.p.taskCount));
    const hasOverlap = scored.some(s => s.overlap > 0);
    const top = (hasOverlap ? scored.filter(s => s.overlap > 0) : scored).slice(0, 5).map(s => s.p);

    // Per-role pass rates across ALL events (named + global)
    const roleAgg: Record<string, { tasks: number; passed: number }> = {};
    for (const p of projects) {
      for (const [role, v] of Object.entries(p.perRole)) {
        (roleAgg[role] ||= { tasks: 0, passed: 0 }).tasks += v.tasks;
        roleAgg[role].passed += v.passed;
      }
    }

    // Role-pair history: projects where both roles were present together
    const pairAgg: Record<string, { tasks: number; passed: number; kickbacks: number; projects: number }> = {};
    for (const p of projects) {
      const roles = Object.keys(p.composition);
      for (let i = 0; i < roles.length; i++) {
        for (let j = i + 1; j < roles.length; j++) {
          const key = [roles[i], roles[j]].sort().join(' + ');
          const a = (pairAgg[key] ||= { tasks: 0, passed: 0, kickbacks: 0, projects: 0 });
          a.tasks += p.taskCount; a.passed += p.passed; a.kickbacks += p.kickbacks; a.projects++;
        }
      }
    }

    const pct = (n: number, d: number) => (d > 0 ? `${Math.round((n / d) * 100)}%` : 'n/a');
    const compStr = (c: Record<string, number>) =>
      Object.entries(c).map(([r, n]) => `${n}×${r}`).join(', ') || 'unknown mix';

    const lines: string[] = [];
    if (top.length) {
      lines.push('Similar past projects (by capability overlap):');
      for (const p of top) {
        const gap = p.topGaps.length ? `; common failure: "${p.topGaps[0]}"` : '';
        lines.push(`- "${p.project}" [${compStr(p.composition)}] — ${p.taskCount} tasks, ${pct(p.passed, p.taskCount)} pass, ${p.kickbacks} kickbacks (${this.signal(p.taskCount)} signal: ${p.taskCount} tasks)${gap}`);
      }
    }
    // Drop the 'unknown' bucket: tasks whose worker can't be attributed to a
    // role (old/messy history) are not actionable for composition decisions.
    const roleEntries = Object.entries(roleAgg).filter(([role, v]) => v.tasks > 0 && role !== 'unknown').sort((a, b) => b[1].tasks - a[1].tasks);
    if (roleEntries.length) {
      lines.push('', 'Per-role track record (all projects):');
      for (const [role, v] of roleEntries) {
        lines.push(`- ${role}: ${pct(v.passed, v.tasks)} pass over ${v.tasks} tasks (${this.signal(v.tasks)} signal)`);
      }
    }
    const pairEntries = Object.entries(pairAgg).filter(([, v]) => v.tasks > 0).sort((a, b) => b[1].tasks - a[1].tasks).slice(0, 4);
    if (pairEntries.length) {
      lines.push('', 'Role-pair history (when run together):');
      for (const [pair, v] of pairEntries) {
        lines.push(`- ${pair}: ${pct(v.passed, v.tasks)} combined pass over ${v.tasks} tasks, ${v.kickbacks} kickbacks (${this.signal(v.tasks)} signal)`);
      }
    }
    const text = lines.length
      ? lines.join('\n') + '\n\nRead the signal labels: trust high-signal lines, treat low-signal as a weak hint and lean on your own judgment.'
      : '';

    return {
      text,
      similarProjects: top,
      perRole: roleEntries.map(([role, v]) => ({ role, tasks: v.tasks, passRate: v.tasks ? v.passed / v.tasks : 0, signal: this.signal(v.tasks) })),
      rolePairs: pairEntries.map(([pair, v]) => ({ pair, tasks: v.tasks, kickbacks: v.kickbacks, passRate: v.tasks ? v.passed / v.tasks : 0, signal: this.signal(v.tasks) })),
    };
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
    capabilities: string[];
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
    // requiredCapabilities of the task, used to bucket outcomes under the
    // agent's registered capability tags in getAgentProfile.
    const capsByTask: Map<string, string[]> = new Map();

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
        const caps: any[] = data.requiredCapabilities || data.task?.requiredCapabilities || [];
        if (caps.length > 0 && !capsByTask.has(taskId)) {
          capsByTask.set(taskId, caps.map(c => String(c)));
        }
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
        capabilities: capsByTask.get(taskId) ?? [],
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
