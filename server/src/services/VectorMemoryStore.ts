/**
 * VectorMemoryStore - Abstract interface for semantic search over PVM coordination logs
 *
 * Task 1.3: Vector Memory Store Interface (12 hours)
 *
 * Purpose:
 * - Semantic search over coordination history
 * - PAIR retrieval (find relevant past patterns)
 * - Agent profile derivation (evidence-based performance tracking)
 * - Support dual-purpose PVM: team coordination + individual agent memory
 */

import {
  CoordinationMessage,
  AgentProfile,
  CoordinationPattern,
  SearchResult,
  VectorSearchQuery
} from '../types/coordination.js';

/**
 * Abstract interface for vector-based semantic memory
 *
 * Implementations:
 * - QdrantVectorStore: Production Qdrant adapter
 * - MockVectorStore: In-memory testing adapter
 */
export abstract class VectorMemoryStore {
  /**
   * Embed text into vector representation
   * @param text - Text to embed
   * @returns Vector embedding (number array)
   */
  abstract embed(text: string): Promise<number[]>;

  /**
   * Batch embed multiple texts (more efficient)
   * @param texts - Array of texts to embed
   * @returns Array of vector embeddings
   */
  abstract batchEmbed(texts: string[]): Promise<number[][]>;

  /**
   * Store coordination message with embedding
   * @param message - Coordination message to store
   * @param embedding - Pre-computed embedding (optional, will compute if not provided)
   */
  abstract store(message: CoordinationMessage, embedding?: number[]): Promise<void>;

  /**
   * Batch store multiple messages (more efficient)
   * @param messages - Array of coordination messages
   */
  abstract batchStore(messages: CoordinationMessage[]): Promise<void>;

  /**
   * Search for similar coordination messages
   * @param query - Search query (text or vector search parameters)
   * @param limit - Maximum results to return
   * @returns Array of search results with similarity scores
   */
  abstract search(query: string | VectorSearchQuery, limit?: number): Promise<SearchResult[]>;

  /**
   * Find relevant coordination patterns for PAIR retrieval
   * @param context - Current coordination context
   * @param limit - Maximum patterns to return
   * @returns Array of relevant past coordination patterns
   */
  abstract findRelevantPatterns(context: string, limit?: number): Promise<CoordinationPattern[]>;

  /**
   * Derive agent profile from coordination history
   * Evidence-based: analyzes actual outcomes, not self-reported data
   *
   * @param agentId - Agent identifier
   * @returns Comprehensive agent profile with capabilities, patterns, synergies
   */
  abstract getAgentProfile(agentId: string): Promise<AgentProfile>;

  /**
   * Get synergy metrics between two agents
   * Analyzes how well agents collaborate
   *
   * @param agent1 - First agent ID
   * @param agent2 - Second agent ID
   * @returns Synergy metrics
   */
  abstract getAgentSynergy(agent1: string, agent2: string): Promise<{
    collaborationCount: number;
    successRate: number;
    strengthAreas: string[];
    weaknessAreas: string[];
  }>;

  /**
   * Compare agents for a specific task type
   * Evidence-based recommendation for task assignment
   *
   * @param agentIds - Array of agent IDs to compare
   * @param taskType - Type of task (e.g., "react_development", "database_optimization")
   * @returns Ranked agents with recommendation
   */
  abstract compareAgents(agentIds: string[], taskType: string): Promise<{
    agentId: string;
    matchScore: number;
    successRate: number;
    taskCount: number;
    recommendation: string;
  }[]>;

  /**
   * Health check - verify vector store is operational
   * @returns True if healthy, false otherwise
   */
  abstract healthCheck(): Promise<boolean>;

  /**
   * Close connections and cleanup resources
   */
  abstract close(): Promise<void>;
}

/**
 * Configuration for VectorMemoryStore implementations
 */
export interface VectorStoreConfig {
  // Embedding configuration
  embeddingProvider: 'local' | 'openai' | 'custom';
  embeddingModel?: string;
  embeddingDimension?: number;

  // Vector DB configuration
  vectorDbType: 'qdrant' | 'mock';
  qdrantUrl?: string;
  qdrantApiKey?: string;
  collectionName?: string;

  // Performance tuning
  batchSize?: number;
  cacheEmbeddings?: boolean;
  maxCacheSize?: number;

  // Agent profile derivation
  minTasksForProfile?: number; // Minimum tasks before deriving profile
  profileRefreshInterval?: number; // Seconds between profile updates
}

export const DEFAULT_VECTOR_CONFIG: VectorStoreConfig = {
  embeddingProvider: 'local',
  embeddingModel: 'all-MiniLM-L6-v2',
  embeddingDimension: 384,
  vectorDbType: 'qdrant',
  qdrantUrl: 'http://localhost:6333',
  collectionName: 'act_coordination',
  batchSize: 100,
  cacheEmbeddings: true,
  maxCacheSize: 10000,
  minTasksForProfile: 3,
  profileRefreshInterval: 300 // 5 minutes
};
