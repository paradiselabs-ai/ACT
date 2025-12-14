import { ChronologicalLog } from './ChronologicalLog';
import { MockVectorStore } from './MockVectorStore';
import { CoordinationMessage } from '../types/coordination';
import { logger } from '../utils/logger';

export class PVMIndexer {
  private chronologicalLog: ChronologicalLog;
  private vectorStore: MockVectorStore;
  private lastIndexedTimestamp: string | null = null;
  private indexingInterval: NodeJS.Timeout | null = null;
  private isIndexing: boolean = false;
  
  constructor(chronologicalLog: ChronologicalLog, vectorStore: MockVectorStore) {
    this.chronologicalLog = chronologicalLog;
    this.vectorStore = vectorStore;
  }
  
  /**
   * Start background indexing of coordination events
   * @param intervalMs - How often to check for new events (default: 10 seconds)
   */
  startIndexing(intervalMs: number = 10000): void {
    if (this.indexingInterval) {
      logger.warn('PVMIndexer already running, stopping previous instance');
      this.stopIndexing();
    }
    
    logger.info(`🚀 PVMIndexer started - checking for new events every ${intervalMs}ms`);
    
    this.indexingInterval = setInterval(() => {
      this.indexNewEvents().catch(err => {
        logger.error(`PVMIndexer failed to index events: ${err.message}`);
      });
    }, intervalMs);
  }
  
  /**
   * Stop background indexing
   */
  stopIndexing(): void {
    if (this.indexingInterval) {
      clearInterval(this.indexingInterval);
      this.indexingInterval = null;
      logger.info('🛑 PVMIndexer stopped');
    }
  }
  
  /**
   * Index new events that haven't been indexed yet
   */
  async indexNewEvents(): Promise<void> {
    // Prevent concurrent indexing
    if (this.isIndexing) {
      logger.debug('PVMIndexer already indexing, skipping this cycle');
      return;
    }
    
    this.isIndexing = true;
    
    try {
      logger.debug('🔍 PVMIndexer checking for new events to index...');
      
      // Query for events after the last indexed timestamp
      const query: any = {};
      if (this.lastIndexedTimestamp) {
        // Note: This assumes the log is queried by timestamp
        // The actual implementation may need adjustment based on ChronologicalLog API
        query.since = this.lastIndexedTimestamp;
      }
      
      // For now, let's read all events and filter
      // In a production implementation, we'd use a more efficient approach
      const result = await this.chronologicalLog.query(query);
      const events = result.events;
      
      if (events.length === 0) {
        logger.debug('🔍 PVMIndexer: No new events to index');
        return;
      }
      
      logger.info(`📥 PVMIndexer found ${events.length} events to index`);
      
      // Convert events to CoordinationMessages and index them
      const coordinationMessages: CoordinationMessage[] = events.map(event => ({
        timestamp: event.timestamp,
        agent: (event as any).agentId || 'system',
        message: event.message || (event as any).content || 'Unknown event',
        type: event.type || 'coordination'
      }));
      
      // Store in vector store
      await this.vectorStore.batchStore(coordinationMessages);
      
      // Update last indexed timestamp
      if (events.length > 0) {
        const latestEvent = events[events.length - 1];
        this.lastIndexedTimestamp = latestEvent.timestamp;
      }
      
      logger.info(`✅ PVMIndexer successfully indexed ${events.length} events`);
      
    } catch (error: any) {
      logger.error(`❌ PVMIndexer failed during indexing: ${error.message}`);
      throw error;
    } finally {
      this.isIndexing = false;
    }
  }
  
  /**
   * Force index all events (useful for initial setup)
   */
  async indexAllEvents(): Promise<void> {
    logger.info('🔄 PVMIndexer force indexing all events...');
    
    try {
      // Query for all events
      const result = await this.chronologicalLog.query({});
      const events = result.events;
      
      if (events.length === 0) {
        logger.info('🔍 PVMIndexer: No events found to index');
        return;
      }
      
      logger.info(`📥 PVMIndexer found ${events.length} events to index`);
      
      // Convert events to CoordinationMessages and index them
      const coordinationMessages: CoordinationMessage[] = events.map(event => ({
        timestamp: event.timestamp,
        agent: (event as any).agentId || 'system',
        message: event.message || (event as any).content || 'Unknown event',
        type: event.type || 'coordination'
      }));
      
      // Store in vector store
      await this.vectorStore.batchStore(coordinationMessages);
      
      // Update last indexed timestamp
      if (events.length > 0) {
        const latestEvent = events[events.length - 1];
        this.lastIndexedTimestamp = latestEvent.timestamp;
      }
      
      logger.info(`✅ PVMIndexer successfully indexed ${events.length} events`);
      
    } catch (error: any) {
      logger.error(`❌ PVMIndexer failed during full indexing: ${error.message}`);
      throw error;
    }
  }
  
  /**
   * Search for similar coordination patterns
   * @param query - Search query text
   * @param limit - Maximum number of results to return
   */
  async search(query: string, limit: number = 10): Promise<any[]> {
    try {
      const results = await this.vectorStore.search(query, limit);
      logger.info(`🔍 PVMIndexer search returned ${results.length} results`);
      return results;
    } catch (error: any) {
      logger.error(`❌ PVMIndexer search failed: ${error.message}`);
      throw error;
    }
  }
  
  /**
   * Get indexer status
   */
  getStatus(): {
    isRunning: boolean;
    isIndexing: boolean;
    lastIndexedTimestamp: string | null;
    indexedEventCount: number;
  } {
    return {
      isRunning: !!this.indexingInterval,
      isIndexing: this.isIndexing,
      lastIndexedTimestamp: this.lastIndexedTimestamp,
      indexedEventCount: (this.vectorStore as any).points ? (this.vectorStore as any).points.length : -1
    };
  }
}

export default PVMIndexer;
