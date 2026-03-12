/**
 * ACT Services - Phase 5 Semantic Coordination Intelligence
 */

export {
  VectorMemoryStore,
  VectorStoreConfig,
  DEFAULT_VECTOR_CONFIG
} from './VectorMemoryStore.js';

// QdrantVectorStore has a pre-existing TypeScript error — excluded from build (MockVectorStore used instead)
// export { QdrantVectorStore } from './QdrantVectorStore.js';
export { MockVectorStore } from './MockVectorStore.js';

export {
  ChronologicalLog,
  ChronologicalLogConfig,
  DEFAULT_CHRONOLOGICAL_CONFIG,
  LogQuery,
  LogQueryResult
} from './ChronologicalLog.js';
