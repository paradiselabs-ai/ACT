# ACT Production Integration & Implementation Guide
## Connecting Memory Architecture + Prompt Engineering Into Working System

---

## Executive Summary

**Goal:** Single cohesive ACT system where:
- Memory retrieval feeds agent prompts with relevant context
- Prompt engineering generates rich coordination signals
- Those signals update PVM log (closes learning loop)
- System improves continuously without human tuning

**Scope (ManusAI):** One isolated sandbox task—implement the memory + context injection system. This can be built independently and integrated later.

**Why Isolated?** ManusAI cannot directly edit coordination.json or affect other agents. But it CAN build the memory subsystem that feeds all agents.

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    ACT Coordination Server                      │
│                   (Main team + ManusAI)                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Agent Input                                                    │
│  ├─ "Build the payment API"                                    │
│  └─ Agent: Backend Specialist                                  │
│           │                                                    │
│           ▼                                                    │
│  ┌───────────────────────────────────────────┐                │
│  │   CONTEXT INJECTION (ManusAI builds)      │                │
│  ├───────────────────────────────────────────┤                │
│  │ 1. Query PVM: "payment API tasks"         │                │
│  │ 2. Retrieve: CompressedPatterns[] (50)    │                │
│  │ 3. Inject: Into system prompt             │                │
│  └───┬────────────────────────────────────────┘                │
│      │                                                         │
│      ▼                                                         │
│  ┌────────────────────────────────────────────┐               │
│  │   ENHANCED AGENT PROMPT                    │               │
│  ├────────────────────────────────────────────┤               │
│  │ System: \"You are Backend Specialist\"       │               │
│  │         \"Similar past work: [context]\"     │               │
│  │         \"Make decisions, log reasoning\"    │               │
│  │ User: \"Build payment API\"                 │               │
│  └───┬─────────────────────────────────────────┘               │
│      │                                                         │
│      ▼                                                         │
│  ┌────────────────────────────────────────────┐               │
│  │   AGENT EXECUTION                          │               │
│  │   (Existing: Claude, GPT, Llama, etc.)    │               │
│  └───┬─────────────────────────────────────────┘               │
│      │                                                         │
│      ▼ Output with embedded signals:                          │
│  ┌────────────────────────────────────────────┐               │
│  │   RICH COORDINATION OUTPUT                 │               │
│  ├────────────────────────────────────────────┤               │
│  │ DECISION: Implement Payment API...         │               │
│  │ REASONING:                                 │               │
│  │   - Capability match: 87/100               │               │
│  │   - Past pattern: 85% success              │               │
│  │ ALTERNATIVES:...                          │               │
│  │ STATUS: Complete + needs QA review        │               │
│  │ NEXT: Request Backend review...           │               │
│  └───┬─────────────────────────────────────────┘               │
│      │                                                         │
│      ▼ (Main team handles)                                    │
│  ┌────────────────────────────────────────────┐               │
│  │   PVM CHRONOLOGICAL LOG UPDATES            │               │
│  │   (ChronologicalLog appends new event)     │               │
│  └────────────────────────────────────────────┘               │
│           │                                                   │
│           ▼                                                   │
│  ┌────────────────────────────────────────────┐               │
│  │   MEMORY COMPRESSION (ManusAI handles)     │               │
│  │   (Background: recompress patterns)        │               │
│  └────────────────────────────────────────────┘               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

↑ Next Coordination Cycle ↑
Agent receives UPDATED patterns based on latest decisions
```

---

## Implementation: Three Integration Points

### Integration Point 1: Context Injection (Request → Prompt)

**Location:** `server/src/index.ts` - where agents register and receive tasks

**Current Code (Example):**
```typescript
// OLD: No context
socket.on('register_agent', async (data) => {
  await agentRegistry.registerAgent(agentId, { ...data });
  
  // Agent gets no context here
  // Just registers and waits for tasks
});

socket.on('create_task', async (data) => {
  const task = await taskCoordinator.createTask(data);
  // Task assigned but no PVM context provided
});
```

**New Code (With ManusAI Subsystem):**
```typescript
import { PVMRetriever } from './services/PVMRetriever';
import { ContextBuilder } from './services/ContextBuilder';

// Initialize ManusAI's retriever on startup
const pvmRetriever = new PVMRetriever();
await pvmRetriever.initialize(process.env.PVM_LOG_PATH || './data/coordination-log.jsonl');

socket.on('create_task', async (data) => {
  const task = await taskCoordinator.createTask(data);
  const assignment = await taskCoordinator.assignOptimalAgent(task.id);
  
  if (assignment) {
    // NEW: Inject PVM context
    const context = await pvmRetriever.search({
      query: task.description,
      agentId: assignment.agentId,
      taskType: task.type,
      limit: 3  // Top 3 similar patterns
    });
    
    // Build enhanced prompt
    const enhancedPrompt = new ContextBuilder()
      .withRole(assignment.agentId)
      .withSimilarPatterns(context)
      .withDecisionFramework()
      .build();
    
    // Send to agent with context
    io.to(agents[assignment.agentId].socketId).emit('task_assigned', {
      task,
      context: enhancedPrompt,  // Agent receives this
      pvm_patterns: context     // Debugging visibility
    });
  }
});
```

**ContextBuilder Implementation:**
```typescript
class ContextBuilder {
  private role: string;
  private patterns: RetrievalResult[] = [];
  private includeDecisionFramework = true;
  
  withRole(agentId: string): ContextBuilder {
    this.role = agentId;
    return this;
  }
  
  withSimilarPatterns(patterns: RetrievalResult[]): ContextBuilder {
    this.patterns = patterns;
    return this;
  }
  
  withDecisionFramework(): ContextBuilder {
    this.includeDecisionFramework = true;
    return this;
  }
  
  build(): string {
    return `
You are the ${this.role} Specialist.

## Your Recent Performance
[Agent role + specialties from registry]

## Similar Past Work
${this.patterns
  .map(p => `- ${p.reasoning} (confidence: ${p.relevanceScore}%)`)
  .join('\n')}

## Decision Framework
${this.includeDecisionFramework ? `
Before executing:
1. Log your reasoning (capability match, past patterns, confidence)
2. Consider alternatives and why you rejected them
3. Identify if you need to involve other specialists
4. Report progress every 4 hours
5. Request reviews before marking complete
` : ''}

Execute the task with your best judgment.
Leverage past patterns and your expertise.
    `;
  }
}
```

**What ManusAI Needs to Provide:**
- ✅ `PVMRetriever.search()` - returns relevant patterns
- ✅ `ContextBuilder` - formats them into prompt injection
- Nothing more. The rest (agent registry, task coordinator) already exists.

---

### Integration Point 2: Signal Parsing (Output → Log)

**Location:** `server/src/services/ChronologicalLog.ts` - where coordination events are logged

**Current Code:**
```typescript
// OLD: Simple append
async append(event: CoordinationMessage): Promise<void> {
  this.buffer.push(event);
  if (this.buffer.length >= this.bufferSize) {
    await this.flush();
  }
}
```

**New Code (With Signal Extraction):**
```typescript
import { SignalExtractor } from './SignalExtractor';

class ChronologicalLog {
  private extractor = new SignalExtractor();
  
  async append(event: CoordinationMessage): Promise<void> {
    // Extract structured signals from agent output
    const signals = this.extractor.extract(event.message);
    
    // Enrich event with parsed signals
    const enrichedEvent = {
      ...event,
      parsed_decision: signals.decision,
      parsed_reasoning: signals.reasoning,
      parsed_alternatives: signals.alternatives,
      parsed_status: signals.status,
      parsed_requests: signals.requests,
      outcome: signals.outcome,
      confidence: signals.confidence
    };
    
    this.buffer.push(enrichedEvent);
    if (this.buffer.length >= this.bufferSize) {
      await this.flush();
    }
  }
}
```

**SignalExtractor Implementation:**
```typescript
interface ParsedSignals {
  decision: string;
  reasoning: { [key: string]: string };
  alternatives: string[];
  status: 'in_progress' | 'complete' | 'blocked' | 'waiting_review';
  requests: Array<{
    type: 'critique' | 'delegation' | 'help';
    target: string;
    reason: string;
  }>;
  outcome: 'success' | 'failure' | 'in_progress';
  confidence: number;  // 0-100
}

class SignalExtractor {
  extract(message: string): ParsedSignals {
    // Use regex + LLM parsing to extract structured signals
    
    const decision = this.extractSection(message, 'DECISION');
    const reasoning = this.extractReasoningDict(message, 'REASONING');
    const alternatives = this.extractList(message, 'ALTERNATIVES CONSIDERED');
    const status = this.extractStatus(message);
    const requests = this.extractRequests(message);
    const outcome = this.inferOutcome(message);
    const confidence = this.extractConfidence(message);
    
    return {
      decision: decision || 'Unknown',
      reasoning: reasoning || {},
      alternatives: alternatives || [],
      status: status || 'in_progress',
      requests: requests || [],
      outcome: outcome || 'in_progress',
      confidence: confidence || 50
    };
  }
  
  private extractSection(text: string, sectionName: string): string {
    // Find "DECISION: ..." and extract content
    const regex = new RegExp(`${sectionName}:\\s*(.+?)(?=\\n[A-Z]|$)`, 's');
    const match = text.match(regex);
    return match?.[1]?.trim() || '';
  }
  
  private extractReasoningDict(text: string, sectionName: string): { [key: string]: string } {
    // Extract "- Capability match: 87/100" format
    const section = this.extractSection(text, sectionName);
    const lines = section.split('\n').filter(l => l.trim().startsWith('-'));
    
    const result: { [key: string]: string } = {};
    for (const line of lines) {
      const [key, ...value] = line.replace('-', '').split(':');
      if (key && value) {
        result[key.trim()] = value.join(':').trim();
      }
    }
    return result;
  }
  
  private extractList(text: string, sectionName: string): string[] {
    const section = this.extractSection(text, sectionName);
    return section
      .split('\n')
      .filter(l => /^\d+\./.test(l.trim()))
      .map(l => l.replace(/^\d+\./, '').trim());
  }
  
  private extractStatus(text: string): ParsedSignals['status'] {
    if (text.includes('STATUS: COMPLETE')) return 'complete';
    if (text.includes('BLOCKED:')) return 'blocked';
    if (text.includes('waiting for') || text.includes('WAITING FOR')) return 'waiting_review';
    return 'in_progress';
  }
  
  private extractRequests(text: string): ParsedSignals['requests'] {
    const requests: ParsedSignals['requests'] = [];
    
    // Parse critique requests
    const critiqueMatch = text.match(/(?:critique|review).*?([A-Z][a-z]+\s+specialist)/gi);
    if (critiqueMatch) {
      requests.push({
        type: 'critique',
        target: critiqueMatch[0],
        reason: 'Code review required'
      });
    }
    
    // Parse delegation requests
    const delegateMatch = text.match(/(?:can i hand off|delegate.*?to|loop in)\s+([A-Z][a-z]+\s+specialist)/gi);
    if (delegateMatch) {
      requests.push({
        type: 'delegation',
        target: delegateMatch[0],
        reason: 'Specialization mismatch'
      });
    }
    
    return requests;
  }
  
  private inferOutcome(text: string): ParsedSignals['outcome'] {
    if (text.includes('Complete:') || text.includes('SUCCESS:')) return 'success';
    if (text.includes('FAILURE:') || text.includes('BLOCKED:')) return 'failure';
    return 'in_progress';
  }
  
  private extractConfidence(text: string): number {
    // Find "confidence: 85/100" pattern
    const match = text.match(/confidence:\s*(\d+)/i);
    if (match) {
      return Math.min(100, Math.max(0, parseInt(match[1], 10)));
    }
    return 50;  // Default: assume medium confidence
  }
}
```

**What ManusAI Needs to Provide:**
- ✅ `SignalExtractor` - parses agent output into structured signals
- Nothing else. ChronologicalLog already appends to JSONL.

---

### Integration Point 3: Memory Refresh (Log → Cache)

**Location:** Background service that keeps compressed patterns fresh

**Implementation:**
```typescript
import { MemoryCompressor } from './MemoryCompressor';

class BackgroundMemoryService {
  private compressor: MemoryCompressor;
  private pvmRetriever: PVMRetriever;
  
  async startPeriodicRefresh(intervalMs = 3600000): Promise<void> {
    // Refresh cached patterns every 1 hour
    setInterval(() => this.refreshPatterns(), intervalMs);
  }
  
  private async refreshPatterns(): Promise<void> {
    try {
      console.log('[Memory] Refreshing compressed patterns...');
      
      // 1. Load full log
      const fullLog = await this.chronologicalLog.loadAll();
      
      // 2. Recompress top patterns
      const freshPatterns = await this.compressor.compress(fullLog, 50);
      
      // 3. Prune stale patterns
      const pruned = await this.compressor.pruneStalePatterns(freshPatterns);
      
      // 4. Update retriever cache
      this.pvmRetriever.updateCache(pruned);
      
      console.log(
        `[Memory] Refreshed: ${pruned.length} patterns, ` +
        `${fullLog.length} total events, ` +
        `${this.pvmRetriever.getCacheSize()}MB cache`
      );
    } catch (error) {
      console.error('[Memory] Refresh failed:', error);
      // Continue with stale cache if error (don't crash)
    }
  }
}

// Start on server init
const memoryService = new BackgroundMemoryService(
  chronologicalLog,
  pvmRetriever,
  memoryCompressor
);
await memoryService.startPeriodicRefresh();
```

**What ManusAI Needs to Provide:**
- ✅ Background service glue (simple wrapper)
- Everything else already exists

---

## Code Changes Summary

**Files ManusAI Creates:**
1. `server/src/services/PVMRetriever.ts` - Main retrieval class
2. `server/src/services/KeywordIndexer.ts` - Fast keyword indexing
3. `server/src/services/RelevanceRanker.ts` - Heuristic ranking
4. `server/src/services/MemoryCompressor.ts` - Pattern compression
5. `server/src/services/ContextBuilder.ts` - Prompt injection
6. `server/src/services/SignalExtractor.ts` - Output parsing
7. `server/src/index.ts` - Integration + initialization (MINIMAL CHANGES)
8. Tests + documentation

**Files Main Team Updates:**
1. `server/src/services/ChronologicalLog.ts` - Add signal enrichment (small)
2. `server/src/index.ts` - Use new retriever (integration point)
3. Agent prompt templates - Include context injection

**No Changes Needed To:**
- TaskCoordinator
- AgentRegistry
- EventHub
- Socket.IO server
- Existing coordination logic

---

## Data Flow Example

**Scenario:** Backend Specialist receives payment API task

**Step 1: Task Assignment**
```javascript
// server/src/index.ts
socket.on('create_task', async (data) => {
  // Create task
  const task = await taskCoordinator.createTask({
    description: "Implement payment API with Stripe",
    requiredCapabilities: ["backend", "api_design"],
    priority: "high"
  });
  
  // Assign to Backend specialist
  const assignment = await taskCoordinator.assignOptimalAgent(task.id);
  // → assignment = { agentId: "claude_backend", confidence: 0.87 }
```

**Step 2: Context Retrieval**
```javascript
  // NEW: Retrieve similar patterns from ManusAI's system
  const patterns = await pvmRetriever.search({
    query: "Implement payment API with Stripe",
    agentId: "claude_backend",
    taskType: "api_implementation",
    limit: 3
  });
  // → patterns = [
  //   {
  //     type: 'pattern',
  //     content: CompressedPattern {
  //       pattern: { taskType: 'payment_api', agentRole: 'claude_backend' },
  //       outcome: { success: true, duration_ms: 5400000, qualityScore: 92 },
  //       confidence: { occurrences: 5, successRate: 0.8 }
  //     },
  //     relevanceScore: 87,
  //     reasoning: "Payment API by Backend specialist (80% success, seen 5x, 1.5h avg)"
  //   },
  //   // ... 2 more patterns
  // ]
```

**Step 3: Prompt Injection**
```javascript
  // NEW: Build enhanced system prompt
  const systemPrompt = new ContextBuilder()
    .withRole("claude_backend")
    .withSimilarPatterns(patterns)
    .withDecisionFramework()
    .build();
  // → "You are the Backend Specialist...
  //    Similar past work:
  //    - Payment API by Backend specialist (80% success, seen 5x)
  //    - Stripe integration (87% success, recent)
  //    - API design with validation (92% quality)
  //
  //    Decision Framework: Before executing,
  //    1. Log your reasoning...
  //    2. Consider alternatives...
  //    etc."
```

**Step 4: Agent Execution**
```javascript
  // Send to agent
  io.to(agents[assignment.agentId].socketId).emit('task_assigned', {
    taskId: task.id,
    task: task,
    systemPrompt: systemPrompt,  // Context injected here
    patterns: patterns
  });
  
  // Agent receives task with context
  // Agent sees similar successes, understands their specialty
  // Agent reasons through decision and logs it
```

**Step 5: Output Parsing**
```javascript
  // Agent responds (in 'task_progress' event)
  socket.on('task_progress', async (data) => {
    // Agent output includes reasoning:
    // "DECISION: Implement using Stripe"
    // "REASONING: ... 85/100 confidence..."
    // "STATUS: In progress, will complete in 2 days"
    
    // NEW: Extract structured signals
    const signals = signalExtractor.extract(data.message);
    // → signals = {
    //   decision: "Implement using Stripe",
    //   confidence: 85,
    //   status: "in_progress",
    //   requests: [
    //     { type: "critique", target: "QA specialist", reason: "Security review" }
    //   ]
    // }
```

**Step 6: Event Logging**
```javascript
    // Enrich and log
    const event = {
      timestamp: new Date().toISOString(),
      type: "task_progress",
      agentId: assignment.agentId,
      taskId: task.id,
      message: data.message,
      // NEW: Parsed signals
      parsed_decision: signals.decision,
      parsed_confidence: signals.confidence,
      parsed_requests: signals.requests,
      // ...
    };
    
    await chronologicalLog.append(event);
    // → Written to coordination-log.jsonl
```

**Step 7: Memory Refresh (Background)**
```javascript
// Every hour, background service runs:
memoryService.refreshPatterns():
  1. Load full log (all events since last refresh)
  2. Compress: find high-signal patterns
     → "Stripe API: 5 occurrences, 80% success, 1.5h avg"
  3. Prune: remove patterns older than 30 days with <60% success
  4. Update cache: pvmRetriever.updateCache(newPatterns)

// Next time a Backend specialist gets a payment task:
// → They see the updated pattern from THIS task
```

---

## Integration Workflow

**For ManusAI (Isolated Development):**

1. **Implement Core (Weeks 1-2)**
   - Build all four services (Retriever, Indexer, Ranker, Compressor)
   - Unit tests on existing coordination-log.jsonl
   - No integration needed yet

2. **Local Testing (Week 2)**
   - Test: Can retrieve patterns from 6+ weeks of real data
   - Test: Latency <100ms on 10k events
   - Test: Memory footprint <200MB
   - Test: Patterns compress/decompress correctly

3. **Integration Stubs (Week 2-3)**
   - Create ContextBuilder (formats context for injection)
   - Create SignalExtractor (parses agent output)
   - Both integrate into existing server code

4. **Deployment Prep (Week 3)**
   - Zero external dependencies check
   - Docker build test
   - Documentation

**For Main Team (When Ready):**
- Wire up retriever calls in task assignment
- Wire up signal extraction in log appending
- Start background refresh service
- Update agent prompts to inject context

**No Blocking Dependencies:**
- ManusAI doesn't need FLUX, PAIR, or any main team work
- Main team doesn't need to wait for ManusAI to start
- Both work in parallel, integrate at the end

---

## Success Checklist

✅ **Functional**
- [ ] PVMRetriever.search() returns relevant patterns in <100ms
- [ ] Patterns compress from 10k events to 50 high-signal items
- [ ] ContextBuilder formats patterns into valid system prompts
- [ ] SignalExtractor parses agent output into structured signals

✅ **Performance**
- [ ] Memory footprint <200MB on 6+ weeks data
- [ ] No external model dependencies (pure TypeScript)
- [ ] Startup <500ms
- [ ] Cache refresh completes in <5 minutes

✅ **Integration Ready**
- [ ] All code properly typed (TypeScript)
- [ ] Error handling for missing/malformed data
- [ ] Logging for debugging (what did retriever return?)
- [ ] Documentation for main team integration

✅ **Deployment**
- [ ] Docker image builds
- [ ] No GPU/CUDA required
- [ ] Works on test edge device
- [ ] Zero embedding errors

---

## Future Enhancements (Main Team)

Once PAIR reasoning is implemented:
1. **Learn better patterns** - PAIR identifies which pattern types predict success
2. **Tune ranking weights** - PAIR adjusts recency/quality/specialty balance
3. **Identify missing patterns** - PAIR finds "we needed pattern type X"
4. **Generalize patterns** - PAIR extracts principles from individual decisions

ManusAI's memory system becomes the data engine for ACT's continuous improvement.

---

## References

**Architecture Patterns:**
- Retrieval Augmented Generation (RAG): Context injection into LLM prompts
- Adaptive Expertise (AX): Compress noise, keep principles
- Background Refresh: Staleness tolerance for large-scale systems

**Integration Design:**
- Observer pattern: Events (task completion) trigger pattern recompression
- Dependency injection: Retriever is injected into coordinator
- Graceful degradation: If refresh fails, continue with stale cache

**Production Considerations:**
- Stateless services: Each component works independently
- Observable: Logging at each step for debugging
- Resilient: Failures don't cascade (cache fallback, retry logic)
- Scalable: O(log N) lookups, batch operations where possible
