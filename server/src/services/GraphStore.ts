/**
 * GraphStore - append-only temporal edge store for the coordination graph
 *
 * The graph is a DERIVED projection of the ChronologicalLog: every edge is
 * produced by a rule in GraphIndexer from one or more log events, referenced
 * back through `episodeKeys`. Removing data/graph-edges.jsonl loses nothing —
 * replaying the event log through the same rules reproduces it exactly.
 *
 * Bi-temporal model (Graphiti-shaped, no graph database):
 * - createdAt / expiredAt — ingestion time: when ACT learned / unlearned the fact
 * - validAt   / invalidAt — event time: when the fact became / stopped being true
 *
 * Contradictions are handled by invalidate-never-delete: a superseding fact
 * stamps the old edge instead of removing it, so "what did we believe at time T"
 * stays answerable. Invalidations are appended as their own JSONL records
 * referencing the edge id; an original line is never rewritten.
 */

import fs from 'fs/promises';
import path from 'path';
import { logger } from '../utils/logger';

export type GraphNodeType = 'agent' | 'task' | 'project' | 'file' | 'verdict';

const NODE_TYPES = new Set<string>(['agent', 'task', 'project', 'file', 'verdict']);

export interface GraphEdge {
  id: string;
  src: string;
  rel: string;
  dst: string;
  fact: string;
  episodeKeys: string[];
  createdAt: string;
  expiredAt?: string;
  validAt?: string;
  invalidAt?: string;
}

export interface GraphNode {
  key: string;
  type: GraphNodeType;
  name: string;
}

export interface EdgeInput {
  src: string;
  rel: string;
  dst: string;
  fact: string;
  episodeKeys: string[];
  validAt?: string;
  createdAt?: string;
}

export interface EdgeFilter {
  src?: string;
  rel?: string;
  dst?: string;
  at?: string;
}

export interface GraphStoreConfig {
  jsonlPath: string;
}

export const DEFAULT_GRAPH_STORE_CONFIG: GraphStoreConfig = {
  jsonlPath: './data/graph-edges.jsonl'
};

interface InvalidationRecord {
  invalidates: string;
  at: string;
  expiredAt?: string;
}

export function nodeKey(type: GraphNodeType, name: string): string {
  return `${type}:${name}`;
}

/**
 * Split "<type>:<name>" on the FIRST colon only — names legitimately contain
 * colons (file paths, ISO timestamps in verdict names).
 */
export function parseNodeKey(key: string): GraphNode | null {
  const i = key.indexOf(':');
  if (i <= 0) return null;
  const type = key.slice(0, i);
  const name = key.slice(i + 1);
  if (!NODE_TYPES.has(type) || name.length === 0) return null;
  return { key, type: type as GraphNodeType, name };
}

/** Point-in-time visibility: (validAt ?? createdAt) <= at < (invalidAt ?? ∞). */
export function isVisibleAt(edge: GraphEdge, at?: string): boolean {
  if (!at) return true;
  const from = edge.validAt ?? edge.createdAt;
  if (from > at) return false;
  return edge.invalidAt === undefined || at < edge.invalidAt;
}

export class GraphStore {
  private config: GraphStoreConfig;
  private edgesById = new Map<string, GraphEdge>();
  private outgoing = new Map<string, GraphEdge[]>();
  private incoming = new Map<string, GraphEdge[]>();
  private nodes = new Map<string, GraphNode>();
  // (src|rel|dst|first episode key) -> edge id. Makes addEdge idempotent so a
  // restart that re-derives the whole log doesn't double every edge.
  private bySignature = new Map<string, string>();
  private seq = 0;
  private initialized = false;
  // Serializes appends so two concurrent addEdge calls can't interleave a line.
  private writeQueue: Promise<void> = Promise.resolve();

  constructor(config: Partial<GraphStoreConfig> = {}) {
    this.config = { ...DEFAULT_GRAPH_STORE_CONFIG, ...config };
  }

  async initialize(): Promise<void> {
    if (this.initialized) return;

    await fs.mkdir(path.dirname(this.config.jsonlPath), { recursive: true });

    try {
      const content = await fs.readFile(this.config.jsonlPath, 'utf-8');
      const lines = content.split('\n').filter(l => l.trim().length > 0);
      const invalidations: InvalidationRecord[] = [];
      let skipped = 0;

      for (const line of lines) {
        try {
          const record = JSON.parse(line);
          if (record && typeof record.invalidates === 'string') {
            invalidations.push(record as InvalidationRecord);
          } else {
            this.indexEdge(record as GraphEdge);
          }
        } catch {
          skipped++; // tolerate a truncated line (crash mid-append)
        }
      }

      // Applied after every edge is indexed: an invalidation record may precede
      // its target if edge files were concatenated out of order.
      for (const inv of invalidations) {
        const edge = this.edgesById.get(inv.invalidates);
        if (!edge) continue;
        edge.invalidAt = inv.at;
        edge.expiredAt = inv.expiredAt ?? inv.at;
      }

      if (skipped > 0) {
        logger.warn(`GraphStore: skipped ${skipped} unparseable line(s) in ${this.config.jsonlPath}`);
      }
      logger.info(`GraphStore: loaded ${this.edgesById.size} edges across ${this.nodes.size} nodes`);
    } catch (error: any) {
      if (error.code !== 'ENOENT') {
        logger.error(`GraphStore failed to load ${this.config.jsonlPath}: ${error.message}`);
      }
      // Missing file is the rebuild path: the caller replays the event log.
    }

    this.initialized = true;
  }

  /**
   * Add an edge. Idempotent on (src, rel, dst, first episode key) — the same
   * event re-derived returns the existing edge instead of a duplicate.
   */
  async addEdge(input: EdgeInput): Promise<GraphEdge> {
    if (!this.initialized) await this.initialize();

    for (const key of [input.src, input.dst]) {
      if (!parseNodeKey(key)) {
        throw new Error(`GraphStore: invalid node key "${key}" (expected <${[...NODE_TYPES].join('|')}>:<name>)`);
      }
    }

    const signature = this.signatureOf(input.src, input.rel, input.dst, input.episodeKeys[0]);
    const existingId = this.bySignature.get(signature);
    if (existingId) return this.edgesById.get(existingId)!;

    const edge: GraphEdge = {
      id: `e${this.seq}`,
      src: input.src,
      rel: input.rel,
      dst: input.dst,
      fact: input.fact,
      episodeKeys: [...input.episodeKeys],
      createdAt: input.createdAt ?? new Date().toISOString(),
      ...(input.validAt ? { validAt: input.validAt } : {})
    };

    this.indexEdge(edge);
    await this.appendLine(edge);
    return edge;
  }

  /**
   * Stamp an edge as no longer true as of `at`. The edge stays in the adjacency
   * maps and in the file; only the temporal bounds change. Idempotent.
   */
  async invalidateEdge(edgeId: string, at: string): Promise<GraphEdge | null> {
    const edge = this.edgesById.get(edgeId);
    if (!edge) return null;
    if (edge.invalidAt) return edge;

    const expiredAt = new Date().toISOString();
    edge.invalidAt = at;
    edge.expiredAt = expiredAt;
    await this.appendLine({ invalidates: edgeId, at, expiredAt } as InvalidationRecord);
    return edge;
  }

  getEdge(id: string): GraphEdge | undefined {
    return this.edgesById.get(id);
  }

  allEdges(): GraphEdge[] {
    return [...this.edgesById.values()];
  }

  findEdges(filter: EdgeFilter = {}): GraphEdge[] {
    const candidates = filter.src
      ? this.outgoing.get(filter.src) ?? []
      : filter.dst
        ? this.incoming.get(filter.dst) ?? []
        : this.allEdges();

    return candidates.filter(e =>
      (!filter.src || e.src === filter.src) &&
      (!filter.dst || e.dst === filter.dst) &&
      (!filter.rel || e.rel === filter.rel) &&
      isVisibleAt(e, filter.at)
    );
  }

  /** Edges still believed true (no invalidAt stamp). */
  activeEdges(filter: EdgeFilter = {}): GraphEdge[] {
    return this.findEdges(filter).filter(e => e.invalidAt === undefined);
  }

  /** All edges touching `key` in either direction, optionally as-of a time. */
  edgesFor(key: string, at?: string): GraphEdge[] {
    const out = this.outgoing.get(key) ?? [];
    const inc = this.incoming.get(key) ?? [];
    return [...out, ...inc].filter(e => isVisibleAt(e, at));
  }

  /**
   * BFS expansion around a node. hops=1 (default) returns the node's own edges;
   * hops=2 also returns the edges of its direct neighbours.
   */
  neighborhood(key: string, options: { at?: string; hops?: number } = {}): { node: GraphNode | null; edges: GraphEdge[] } {
    const hops = options.hops && options.hops > 1 ? 2 : 1;
    const collected = new Map<string, GraphEdge>();
    const visited = new Set<string>([key]);
    let frontier = [key];

    for (let h = 0; h < hops; h++) {
      const next: string[] = [];
      for (const current of frontier) {
        for (const edge of this.edgesFor(current, options.at)) {
          if (!collected.has(edge.id)) collected.set(edge.id, edge);
          const other = edge.src === current ? edge.dst : edge.src;
          if (!visited.has(other)) {
            visited.add(other);
            next.push(other);
          }
        }
      }
      frontier = next;
    }

    return {
      node: this.nodes.get(key) ?? parseNodeKey(key),
      edges: [...collected.values()]
    };
  }

  getEdgeCount(): number {
    return this.edgesById.size;
  }

  getNodeCount(): number {
    return this.nodes.size;
  }

  stats(): { nodes: number; edges: number; activeEdges: number; path: string } {
    let active = 0;
    for (const edge of this.edgesById.values()) {
      if (edge.invalidAt === undefined) active++;
    }
    return {
      nodes: this.nodes.size,
      edges: this.edgesById.size,
      activeEdges: active,
      path: this.config.jsonlPath
    };
  }

  private signatureOf(src: string, rel: string, dst: string, episodeKey?: string): string {
    return `${src}|${rel}|${dst}|${episodeKey ?? ''}`;
  }

  private indexEdge(edge: GraphEdge): void {
    if (!edge || typeof edge.id !== 'string' || !edge.src || !edge.dst) return;

    this.edgesById.set(edge.id, edge);
    this.registerNode(edge.src);
    this.registerNode(edge.dst);

    const out = this.outgoing.get(edge.src);
    if (out) out.push(edge); else this.outgoing.set(edge.src, [edge]);

    const inc = this.incoming.get(edge.dst);
    if (inc) inc.push(edge); else this.incoming.set(edge.dst, [edge]);

    this.bySignature.set(this.signatureOf(edge.src, edge.rel, edge.dst, edge.episodeKeys?.[0]), edge.id);

    // Keep id generation monotonic across a reload of previously written ids.
    const n = parseInt(edge.id.replace(/^e/, ''), 10);
    if (!Number.isNaN(n) && n >= this.seq) this.seq = n + 1;
    else if (Number.isNaN(n)) this.seq++;
  }

  private registerNode(key: string): void {
    if (this.nodes.has(key)) return;
    const node = parseNodeKey(key);
    if (node) this.nodes.set(key, node);
  }

  private async appendLine(record: unknown): Promise<void> {
    const line = JSON.stringify(record) + '\n';
    const write = this.writeQueue.catch(() => {}).then(async () => {
      const fh = await fs.open(this.config.jsonlPath, 'a');
      try {
        await fh.write(line, null, 'utf-8');
      } finally {
        await fh.close();
      }
    });
    // A failed write must not poison the chain for later callers.
    this.writeQueue = write.catch(() => {});
    return write;
  }
}

export default GraphStore;
