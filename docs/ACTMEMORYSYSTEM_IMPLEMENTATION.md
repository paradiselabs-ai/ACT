# ACTMemorySystem Implementation Specification

**Date:** October 22, 2025
**Component:** PAIRed Vector Minutes (PVM) - Core Memory Layer
**Status:** Design Phase
**Priority:** Critical Path Item #3

---

## Overview

ACTMemorySystem is the core memory engine for ACT, implementing PAIRed Vector Minutes (PVM). It combines:
- **Chronological append-only coordination log** (complete audit trail)
- **Vector-indexed semantic memory** (efficient RAG retrieval)
- **PAIR reasoning workflow** (intelligent re-injection)
- **Context window management** (recent + referenced + RAG)
- **Memory evaluation & pruning** (continuous optimization)

---

## Architecture

```typescript
class ACTMemorySystem {
  // Storage layers
  private chronologicalLog: ChronologicalLog;
  private vectorStore: VectorMemoryStore;
  private memoryIndex: MemoryIndex;

  // Memory categories
  private agentPreferences: Map<AgentID, AgentMemory>;
  private coordinationPatterns: CoordinationPatternDB;
  private projectKnowledge: ProjectContextDB;

  // Processing
  private contextWindowManager: ContextWindowManager;
  private memoryEvaluator: MemoryEvaluator;
  private pairReasoningEngine: PAIRReasoningEngine;
}
```

---

## Component 1: Chronological Coordination Log

### Purpose
Complete, immutable record of every coordination event. Never deleted, always auditable.

### Structure
```typescript
interface CoordinationMinute {
  id: string;                    // Unique identifier
  timestamp: ISO8601;            // When event occurred
  event_type: CoordinationEventType;
  agent_id?: string;             // Agent involved
  project_id: string;            // Which project

  // Event-specific data
  data: {
    task_id?: string;
    assignment_reasoning?: string;
    conflict_type?: string;
    resolution?: any;
    success_criteria_met?: number; // 0-100%
  };

  // Metadata for context
  context_references?: string[]; // IDs of related past events
  memory_note?: string;          // Human-readable summary
}
```

### Implementation
```typescript
class ChronologicalLog {
  private logFile: AppendOnlyFile;
  private inMemoryBuffer: CoordinationMinute[] = [];

  async append(event: CoordinationMinute): Promise<void> {
    // 1. Add to in-memory buffer
    this.inMemoryBuffer.push(event);

    // 2. Persist to disk (append-only)
    await this.logFile.append(JSON.stringify(event) + '\n');

    // 3. Trigger downstream processing
    await this.onEventRecorded(event);
  }

  async getRecent(count: number): Promise<CoordinationMinute[]> {
    // Get last N events from buffer (fast)
    return this.inMemoryBuffer.slice(-count);
  }

  async getByIds(ids: string[]): Promise<CoordinationMinute[]> {
    // Get specific events by reference (for context reconstruction)
    return this.inMemoryBuffer.filter(e => ids.includes(e.id));
  }

  async getAllHistory(): Promise<CoordinationMinute[]> {
    // Full history (for initial load, FLUX State evaluation)
    return this.inMemoryBuffer;
  }
}
```

### Key Properties
- **Append-only:** No modifications, no deletions
- **Immutable:** Once written, never changed
- **Auditable:** Complete trail of every decision
- **Efficient:** In-memory buffer for recent events, disk-backed for durability

---

## Component 2: Vector-Indexed Memory Store

### Purpose
Semantic indexing of coordination events for fast RAG retrieval. Enables semantic search without scanning entire log.

### Structure
```typescript
interface VectorMemory {
  id: string;                    // References CoordinationMinute.id
  embedding: number[];           // Vector embedding (1536-dim for OpenAI)

  // Semantic categorization
  category: MemoryCategory;      // agent_capability, coordination_pattern, conflict_resolution, etc.

  // Extracted insights
  insights: Insight[];

  // Scoring for memory evaluation
  metadata: {
    recency_score: number;       // How recent (0-1)
    relevance_score: number;     // How relevant to typical queries (0-1)
    accuracy_score: number;      // How accurate was the outcome (0-1)
    impact_score: number;        // How much did this affect coordination (0-1)
    composite_score: number;     // Weighted combination for pruning
  };
}

enum MemoryCategory {
  AGENT_CAPABILITY = 'agent_capability',
  COORDINATION_PATTERN = 'coordination_pattern',
  CONFLICT_RESOLUTION = 'conflict_resolution',
  TASK_ASSIGNMENT = 'task_assignment',
  FAILURE_PATTERN = 'failure_pattern',
  PROJECT_KNOWLEDGE = 'project_knowledge'
}

interface Insight {
  type: InsightType;
  finding: string;              // What we learned
  confidence: number;           // 0-1 confidence score
  applicable_contexts?: string[]; // When this applies
}
```

### Implementation
```typescript
class VectorMemoryStore {
  private vectorDB: Qdrant;      // or Milvus, Pinecone, etc.
  private embedder: EmbeddingModel; // OpenAI embeddings

  async storeMemory(minute: CoordinationMinute, insights: Insight[]): Promise<void> {
    // 1. Generate embedding from event data
    const text = this.formatForEmbedding(minute);
    const embedding = await this.embedder.embed(text);

    // 2. Extract category
    const category = this.categorizeEvent(minute);

    // 3. Store in vector DB
    await this.vectorDB.upsert({
      id: minute.id,
      vector: embedding,
      payload: {
        category,
        insights,
        timestamp: minute.timestamp,
        agent_id: minute.agent_id,
        project_id: minute.project_id
      }
    });
  }

  async search(query: string, limit: number = 10): Promise<VectorMemory[]> {
    // 1. Embed the query
    const queryEmbedding = await this.embedder.embed(query);

    // 2. Search vector DB
    const results = await this.vectorDB.search({
      vector: queryEmbedding,
      limit,
      with_payload: true
    });

    // 3. Return scored results
    return results.map(r => this.reconstructMemory(r));
  }

  async pruneMemories(threshold: number = 0.3): Promise<void> {
    // Remove low-scoring memories from vector store
    // (Chronological log keeps everything)
    const allMemories = await this.vectorDB.scroll();
    const toDelete = allMemories.filter(m => m.metadata.composite_score < threshold);

    await Promise.all(toDelete.map(m => this.vectorDB.delete(m.id)));
  }
}
```

### Key Properties
- **Semantic search:** Find similar past coordination patterns
- **Efficient:** O(1) lookup via vector similarity, not O(n) scan
- **Scored:** Memory evaluation helps system learn what works
- **Prunable:** Low-confidence memories removed, high-confidence kept

---

## Component 3: Context Window Manager

### Purpose
Build optimal context for coordination decisions without overloading agents.

### Strategy

Following the manual JSON coordination pattern:
- Agents don't re-read entire history (prevents context explosion)
- But always get recent events (last 50)
- Plus referenced events mentioned in recent messages
- Plus RAG results for task-specific patterns
- Result: balanced context (informed but bounded)

### Implementation
```typescript
class ContextWindowManager {
  private ACTIVE_WINDOW_SIZE = 50;  // Always include last 50 events

  async buildContextForAgent(
    agentId: string,
    task?: Task
  ): Promise<CoordinationContext> {
    // 1. Always: Recent chronological events
    const recent = await this.log.getRecent(this.ACTIVE_WINDOW_SIZE);

    // 2. Extract references from recent messages
    const referencedIds = this.extractReferences(recent);
    const referenced = await this.log.getByIds(referencedIds);

    // 3. Agent-specific memory (historical performance)
    const agentMemory = await this.getAgentMemory(agentId);

    // 4. Task-specific RAG (if task provided)
    const relevantExperience = task
      ? await this.vectorStore.search(
          `Similar task assignment: ${task.title} with ${task.requiredCapabilities.join(',')}`,
          limit: 5
        )
      : [];

    return {
      recent,              // Always fresh
      referenced,          // Mentioned in recent
      agentMemory,        // This agent's history
      relevantExperience, // Similar past outcomes
      fullLogAvailable: true  // Available if needed
    };
  }

  private extractReferences(events: CoordinationMinute[]): string[] {
    // Find references to past events ("as discussed in task_015...")
    const references = new Set<string>();

    for (const event of events) {
      if (event.context_references) {
        event.context_references.forEach(ref => references.add(ref));
      }
    }

    return Array.from(references);
  }
}
```

---

## Component 4: Memory Evaluation & Pruning

### Purpose
Ensure memory store stays lean and accurate. Learn what coordination patterns actually work.

### Evaluation Metrics

```typescript
interface MemoryScore {
  recency: number;       // How recent (higher = more recent)
  relevance: number;     // How often retrieved (higher = more relevant)
  accuracy: number;      // Did the outcome meet success criteria (0-1)
  impact: number;        // How much did this affect coordination success (0-1)

  composite: number;     // Weighted: 0.25*R + 0.25*Rel + 0.25*A + 0.25*I
}
```

### Implementation
```typescript
class MemoryEvaluator {
  async evaluateAndOrganize(): Promise<void> {
    // 1. Get all memories from vector store
    const memories = await this.vectorStore.getAllMemories();

    // 2. Score each
    const scored = memories.map(m => ({
      memory: m,
      score: this.scoreMemory(m)
    }));

    // 3. Deduplicate similar memories
    const deduped = this.deduplicateSimilar(scored);

    // 4. Resolve conflicts (contradicting patterns)
    const resolved = this.resolveConflicts(deduped);

    // 5. Prune low-scoring (keep ALL in chronological log)
    const toKeep = resolved.filter(m => m.score.composite > 0.3);
    await this.vectorStore.replace(toKeep);

    // 6. Log organization results
    await this.updateMemoryStats({
      totalInLog: memories.length,
      keptInVector: toKeep.length,
      pruned: memories.length - toKeep.length,
      avgScore: (toKeep.reduce((s, m) => s + m.score.composite, 0) / toKeep.length)
    });
  }

  private scoreMemory(memory: VectorMemory): MemoryScore {
    const recency = this.recencyScore(memory.metadata.timestamp);
    const relevance = this.relevanceScore(memory);
    const accuracy = this.accuracyScore(memory);
    const impact = this.impactScore(memory);

    return {
      recency,
      relevance,
      accuracy,
      impact,
      composite: (recency + relevance + accuracy + impact) / 4
    };
  }

  private recencyScore(timestamp: ISO8601): number {
    // Recent events score higher
    const ageSeconds = (Date.now() - new Date(timestamp).getTime()) / 1000;
    const ageWeeks = ageSeconds / (7 * 24 * 60 * 60);

    // Decay over time: week 0 = 1.0, week 4 = 0.0
    return Math.max(0, 1 - (ageWeeks / 4));
  }

  private accuracyScore(memory: VectorMemory): number {
    // Did the outcome meet success criteria?
    if (!memory.insights.some(i => i.type === 'outcome')) return 0.5;

    const outcome = memory.insights.find(i => i.type === 'outcome');
    return outcome?.confidence ?? 0.5;
  }
}
```

---

## Component 5: PAIR Reasoning Engine

### Purpose
Re-inject relevant past coordination patterns when agents need context.

### Workflow

```
1. Agent completes task
2. Memory wipe (FLUX State)
3. Fresh agent evaluates outcome
4. Gaps identified → PAIR triggers
5. RAG retrieves relevant past patterns
6. Agent revises understanding
7. Loop until 95%+ success criteria
```

### Implementation
```typescript
class PAIRReasoningEngine {
  async reinjectionWorkflow(
    coordinationSession: CoordinationSession
  ): Promise<ImprovedCoordination> {
    // 1. FLUX State: Evaluate with memory wipe
    const evaluation = await this.fluxStateEvaluation(coordinationSession);

    // If meets criteria, done
    if (evaluation.success_criteria_met >= 95) {
      return { improved: false, reason: 'Already meets criteria' };
    }

    // 2. Identify gaps
    const gaps = evaluation.identified_gaps;

    // 3. PAIR: For each gap, retrieve relevant context
    const improvements = [];
    for (const gap of gaps) {
      const relevant = await this.vectorStore.search(
        `Coordination pattern: ${gap.description}`,
        limit: 5
      );

      improvements.push({
        gap,
        relevantPatterns: relevant,
        suggestedChanges: await this.generateSuggestions(gap, relevant)
      });
    }

    // 4. Apply improvements and re-evaluate
    const revised = await this.applyImprovements(coordinationSession, improvements);
    const reEvaluation = await this.fluxStateEvaluation(revised);

    // Loop until threshold
    if (reEvaluation.success_criteria_met < 95) {
      return this.reinjectionWorkflow(revised); // Recursive improvement
    }

    return { improved: true, finalSession: revised };
  }

  private async fluxStateEvaluation(
    session: CoordinationSession
  ): Promise<FluxStateEvaluation> {
    // Wipe working memory, give original task + success criteria + output
    const prompt = `
      Original Task: ${session.original_task}
      Success Criteria: ${session.success_criteria}
      Output Produced: ${session.output}

      Evaluate: How well does the output meet the success criteria?
      Be critical. Identify gaps.
    `;

    // Fresh agent (no memory of creating this) evaluates
    return await this.evaluationAgent.analyze(prompt);
  }
}
```

---

## Data Flow

```
Coordination Event Occurs
  ↓
ChronologicalLog.append()
  ├→ In-memory buffer update
  ├→ Disk persistence
  └→ onEventRecorded() trigger
      ↓
VectorMemoryStore.storeMemory()
  ├→ Generate embedding
  ├→ Extract insights
  ├→ Store in vector DB
  └→ Index for search
      ↓
Context requests use all layers:
  ├→ Recent from ChronologicalLog
  ├→ Referenced events from ChronologicalLog
  ├→ Agent memory from VectorMemoryStore
  └→ RAG results from VectorMemoryStore
      ↓
Periodic evaluation:
  └→ MemoryEvaluator scores all memories
      ├→ Keeps high-scoring in vector store
      ├→ Keeps ALL in chronological log
      └→ Prunes low-scoring from vector store
```

---

## Key Design Decisions

1. **Append-only chronological log** - Inspired by manual JSON coordination. Never loses history.
2. **Vector store is index, not source** - ChronologicalLog is the source of truth. Vector store is optimization.
3. **Bounded context windows** - Recent + Referenced + RAG prevents context explosion while staying informed.
4. **Continuous memory evaluation** - System learns which coordination patterns work.
5. **Non-destructive pruning** - Remove from vector store, keep in chronological log.

---

## Dependencies

- **Vector Database:** Qdrant (lightweight, open-source)
- **Embedding Model:** OpenAI embeddings or open-source equivalent
- **File System:** Append-only log files
- **In-Memory State:** CoordinationMinute[]

---

## Success Criteria

- ✅ Complete coordination history preserved (append-only)
- ✅ Fast semantic search without full-log scans
- ✅ Agents get informed context without overload
- ✅ System learns which patterns work over time
- ✅ Unbiased self-evaluation via FLUX State + PAIR

---

## Next Steps

1. Implement ChronologicalLog class
2. Implement VectorMemoryStore class
3. Integrate with ACT server event handling
4. Test with sample coordination sessions
5. Measure performance (query time, vector DB efficiency)
