# Phase 5 Implementation Roadmap
## Weeks 1-4: Semantic Coordination Intelligence MVP

**Timeline:** November 16 - December 14, 2025
**Goal:** Complete MCP-ready ACT with PVM, FLUX State, and PAIR reasoning
**Deliverable:** Production-ready coordination layer for hackathon + standalone deployment

---

## Overview: Parallel Track Architecture

**Instance #1 (Claude Code)** focuses on:
- Core coordination engine architecture
- Memory system infrastructure (chronological log + vector DB)
- FLUX State reasoning engine
- PAIR retrieval system
- Event streaming patterns

**Instance #2 (Windsurf)** focuses on:
- MCP server bridge implementation
- Self-improvement engine
- SSE widget visualization
- Vector database integration
- Performance optimization

**Both instances coordinate through act-coordination.json:**
- Document decisions, blockers, dependencies
- Share status updates every major milestone
- Escalate conflicts to user for arbitration

---

## Week 1: Foundation (Nov 16-22)

### Goal
Build the core memory system and decision infrastructure that everything else depends on.

### Instance #1 Tasks: Core Architecture

**Task 1.1: ACT Server Refactor (16 hours)**
- [ ] Create new modular server structure
  - `src/services/ACTMemorySystem.ts` - Core memory interface
  - `src/services/ChronologicalLog.ts` - Append-only log persistence
  - `src/services/VectorMemoryStore.ts` - Vector DB interface
  - `src/services/TaskCoordinator.ts` - Updated with PVM context
  - `src/services/ConflictResolver.ts` - Uses PVM for conflict patterns
- [ ] Add TypeScript interfaces for all major types
- [ ] Implement configuration system (vector DB selection, embedding model, persistence backend)
- [ ] Create logger with structured coordination event logging
- [ ] Database: Set up SQLite for embedded testing, PostgreSQL connection for production testing
- [ ] No breaking changes to existing working coordination

**Deliverable:** Modular server structure with PVM infrastructure ready

**Task 1.2: Chronological Log Implementation (12 hours)**
- [ ] Implement ChronologicalLog class
  - In-memory buffer for writes (100 events)
  - Append-only semantics (never overwrite)
  - Persistent storage: JSONL file format + SQLite fallback
  - Efficient query interface (recent events, range queries)
- [ ] Event serialization format
- [ ] Disk persistence strategy (background flush every 10 events or 5 seconds)
- [ ] Compaction strategy (for long-running servers)
- [ ] Unit tests for append-only guarantees

**Deliverable:** Production-ready chronological log that's human-readable and auditable

**Task 1.3: Vector Memory Store Interface (12 hours)**
- [ ] Design VectorMemoryStore abstract interface
  - `embed(text: string) -> number[]`
  - `store(event: CoordinationEvent, embedding: number[]) -> void`
  - `search(query: string, limit: number) -> SearchResult[]`
  - `batchEmbed(texts: string[]) -> number[][]`
- [ ] Implement Qdrant adapter
  - Connection management
  - Error handling + retries
  - Health checks
- [ ] Implement mock adapter for testing
- [ ] Unit tests for embedding + search
- [ ] Performance benchmarks

**Deliverable:** Pluggable vector DB layer that works with Qdrant, testable with mocks

**Completion Check (Week 1 End):**
- [ ] Server compiles without errors
- [ ] ACTMemorySystem interface complete and tested
- [ ] ChronologicalLog persists events durably
- [ ] VectorMemoryStore integrates Qdrant (even if local/mock for testing)
- [ ] All three systems communicate without errors
- [ ] No regression: existing coordination still works

### Instance #2 Tasks: Embedding + Vector Setup

**Task 2.1: Embedding Model Integration (12 hours)**
- [ ] Add sentence-transformers to project
- [ ] Create EmbeddingProvider interface
  - `embed(text: string) -> number[]`
  - `batchEmbed(texts: string[]) -> number[][]`
- [ ] Implement local sentence-transformers adapter
  - Load model on startup
  - Cache embeddings
  - Handle out-of-memory errors
- [ ] Implement OpenAI fallback adapter (for future production use)
- [ ] Performance testing: measure embedding latency
- [ ] Unit tests for embedding consistency

**Deliverable:** Portable embedding system that defaults to local, supports cloud APIs

**Task 2.2: Qdrant Setup + Testing (12 hours)**
- [ ] Docker compose setup for local Qdrant (development)
- [ ] Connection configuration (localhost for dev, env variable for production)
- [ ] Collection creation for coordination events
- [ ] Health check implementation
- [ ] Integration tests: embed + store + search cycle
- [ ] Performance testing: search latency with realistic event volumes
- [ ] Documentation: "How to set up Qdrant locally"

**Deliverable:** Working local Qdrant setup with integration tests

**Task 2.3: MCP Server Skeleton (12 hours)**
- [ ] Create new MCP server project (separate from ACT server)
  - TypeScript/Node.js MCP implementation
  - Connects to ACT server on port 8080
  - Routes agent requests to ACT
- [ ] Implement basic MCP tools:
  - `register_with_act` - Register agent capabilities
  - `get_task` - Receive next task
  - `report_task_progress` - Update progress
  - `report_task_complete` - Mark done
- [ ] Tool input/output schemas defined
- [ ] Error handling stubs
- [ ] Does NOT need to be fully functional yet, structure is key

**Deliverable:** MCP server that compiles and connects to ACT (functions added in Week 2)

**Completion Check (Week 1 End):**
- [ ] Embedding model loads and produces consistent vectors
- [ ] Qdrant runs locally in Docker
- [ ] Store + search roundtrip works
- [ ] MCP server structure complete
- [ ] Both instances have reviewed each other's work
- [ ] No integration blockers identified

---

## Week 2: Intelligence Layer (Nov 23-29)

### Goal
Build FLUX State reasoning and PAIR retrieval - the core innovation that makes coordination intelligent.

### Instance #1 Tasks: FLUX State + PAIR Reasoning

**Task 1.4: FLUX State Evaluation Engine (16 hours)**
- [ ] Implement FluxStateEvaluator class
  - `evaluateTask(task: Task, deliverables: string[], originalCriteria: string[])`
  - Steps:
    1. Wipe agent memory (create fresh agent instance)
    2. Provide original task + success criteria (no context of original execution)
    3. Show deliverables
    4. Fresh agent evaluates: "Do these deliverables meet the criteria?"
    5. Return: evaluation_score (0-100), gaps_identified, improvement_suggestions
- [ ] Implementation: Works with ACT's supported models (Claude, GPT, etc.)
- [ ] Handles long-running evaluations (may take minutes for complex tasks)
- [ ] Error handling: Graceful degradation if evaluation fails
- [ ] Unit tests: Test with mock tasks, verify gap identification
- [ ] Integration: Can call evaluation for any completed task

**Deliverable:** Unbiased task evaluation mechanism that identifies improvement opportunities

**Task 1.5: PAIR Retrieval Workflow (16 hours)**
- [ ] Implement PAIRController class
  - Triggered when FLUX State finds gaps (evaluation_score < 95)
  - Steps:
    1. Identify gaps from FLUX State evaluation
    2. RAG search: "Find coordination patterns for similar task type"
    3. Retrieve top 5 relevant past coordination decisions
    4. Format as context: "Here's why similar decisions were made..."
    5. Provide to agent for informed revision
    6. Agent either validates original or suggests improvement
    7. Loop until 95%+ success OR max 3 iterations
- [ ] RAG query builder: Convert gaps → semantic search queries
- [ ] Context formatter: Present patterns clearly to agent
- [ ] Loop controller: Iteration tracking, convergence detection
- [ ] Database: Store evaluation + PAIR results back to chronological log
- [ ] Unit tests: Simulate gap identification and retrieval
- [ ] Integration tests: Full FLUX+PAIR cycle on sample tasks

**Deliverable:** Intelligent context-guided improvement system via semantic memory

**Task 1.6: Memory Evaluation & Pruning (12 hours)**
- [ ] Implement MemoryEvaluator class
  - Score coordination events by relevance + freshness
  - Identify outdated/low-confidence memories
  - Pruning strategy: Remove bottom 10% lowest-confidence events monthly
- [ ] Scoring algorithm:
  - Relevance: How often retrieved in RAG searches?
  - Freshness: Recency decay over time
  - Success: Did coordination decisions lead to successful tasks?
  - Impact: How many future tasks were influenced?
- [ ] Batch processing: Run during idle periods (no performance impact)
- [ ] Safety: Never delete from chronological log, only vector index
- [ ] Unit tests: Verify pruning doesn't break audit trail

**Deliverable:** Intelligent memory hygiene to prevent context explosion

**Completion Check (Week 2 End):**
- [ ] FLUX State evaluation works end-to-end
- [ ] PAIR retrieval finds relevant patterns
- [ ] Loop converges on real evaluation tasks
- [ ] Memory evaluation identifies low-confidence events
- [ ] All three components integrated
- [ ] No performance regressions

### Instance #2 Tasks: MCP Completion + Self-Improvement

**Task 2.4: MCP Tools Implementation (16 hours)**
- [ ] Complete all 6 MCP tools (started in Week 1):
  - `register_with_act` → Calls ACT /register endpoint
  - `get_task` → Polls ACT task queue
  - `report_task_progress` → Sends progress updates
  - `report_task_complete` → Marks task done
  - `query_coordination_memory` → RAG search on PVM
  - `evaluate_coordination` → Trigger FLUX State evaluation
- [ ] Implement error handling + retries for each tool
- [ ] Tool schemas: Comprehensive input/output documentation
- [ ] Integration tests: Test each tool against ACT server
- [ ] Performance: All tools return within 2 seconds (except long-running evaluation)

**Deliverable:** Fully functional MCP bridge between agents and ACT

**Task 2.5: Self-Improvement Engine (16 hours)**
- [ ] Implement SelfImprovementEngine class
  - Explicit trigger: `/improve` command
    - User-initiated post-session analysis
    - Comprehensive FLUX+PAIR evaluation
    - Generates improvement recommendations
    - Updates agent performance profiles
  - Implicit trigger: Background learning
    - Runs during coordination idle periods
    - Analyzes recent sessions
    - Updates vector store with new patterns
    - Prunes low-confidence memories
- [ ] Endpoint: `POST /improve` - Start improvement analysis
- [ ] Background task queue: Scheduler for idle-time learning
- [ ] Results storage: Save evaluations + recommendations
- [ ] Agent profile updates: Track improvement over time
- [ ] Unit tests: Mock FLUX+PAIR, verify engine flow

**Deliverable:** ACT learns and improves autonomously over time

**Task 2.6: SSE Widget System Basics (12 hours)**
- [ ] Create /server/src/widgets/ directory structure
- [ ] Define widget interfaces:
  - `TaskAssignmentWidget` - Shows task + why assigned
  - `AgentStatusWidget` - Current workload + capabilities
  - `ProgressWidget` - Task completion tracking
  - `ConflictResolutionWidget` - How conflicts were resolved
  - `CoordinationInsightWidget` - PVM semantic learnings
- [ ] SSE event emitter: Broadcast widgets via `/coordinate` endpoint
- [ ] Widget serialization: JSON format for HTTP streaming
- [ ] Does NOT need rendering yet (HTML client is Week 3)
- [ ] Unit tests: Verify widget data format

**Deliverable:** Widget system ready for visualization in Week 3

**Completion Check (Week 2 End):**
- [ ] MCP tools are functional and tested
- [ ] Self-improvement engine runs without errors
- [ ] `/improve` endpoint can trigger evaluation
- [ ] SSE widget formats are defined
- [ ] Instance #1 FLUX+PAIR integrated successfully
- [ ] Both instances tested integration points

---

## Week 3: Integration & Visualization (Nov 30-Dec 6)

### Goal
Connect all pieces, build the visualization layer, and achieve full end-to-end coordination with intelligence.

### Instance #1 Tasks: Server Architecture Completion

**Task 1.7: ChatKit-Style Endpoint Design (12 hours)**
- [ ] Refactor server endpoints:
  - Primary: `POST /coordinate` (handles task assignment + coordination, returns SSE stream)
  - Auxiliary: `POST /agents/register` - Fast, non-blocking agent join
  - Auxiliary: `POST /agents/{id}/heartbeat` - Health checks
  - Auxiliary: `GET /coordination/history` - Query past events
  - Auxiliary: `POST /coordination/improve` - Trigger improvement
  - Auxiliary: `GET /projects` - List projects
- [ ] Queue system: `/coordinate` uses internal queue to prevent blocking
- [ ] Background workers: FLUX+PAIR evaluation runs async
- [ ] Error isolation: Auxiliary endpoints never fail due to primary endpoint issues
- [ ] Load testing: Verify no blocking under 100 concurrent agents

**Deliverable:** Robust hybrid endpoint design that prevents bottlenecks

**Task 1.8: Event Streaming Design (12 hours)**
- [ ] SSE event streaming implementation
  - Server → Client: Coordination events as they happen
  - Structured format: event_type, data, metadata
  - Heartbeat: Keep-alive events every 30 seconds
  - Reconnect: Client auto-reconnects on disconnect
  - Ordering: Maintain chronological ordering through network
- [ ] Integration: Connect SSE to widget system (Task 2.6)
- [ ] Error handling: Network failures, client disconnects
- [ ] Unit tests: Verify event ordering under load
- [ ] Performance: Test 1000s of concurrent subscribers

**Deliverable:** Reliable, scalable event streaming for real-time coordination

**Task 1.9: Store Abstraction & Persistence (12 hours)**
- [ ] Design Store interface:
  - Implementations: FilesystemStore (JSONL), SQLiteStore, PostgreSQLStore
  - All implement same interface: `append()`, `query()`, `search()`
  - Runtime selection: Environment variable `ACT_STORE_TYPE`
- [ ] Migration strategy: Tools to convert between stores
- [ ] Backup/restore: Export/import full coordination log
- [ ] Unit tests: Each store type validated independently
- [ ] Integration: All three stores produce identical results

**Deliverable:** Pluggable storage that supports MVP through enterprise scale

**Completion Check (Week 3 End):**
- [ ] Server handles 100+ agents without blocking
- [ ] Event streaming works reliably
- [ ] All endpoints integrated with new architecture
- [ ] Storage abstraction working
- [ ] FLUX+PAIR fully integrated into coordination flow

### Instance #2 Tasks: Visualization + Polish

**Task 2.7: HTML Client + Widget Rendering (16 hours)**
- [ ] Create `/client/index.html` v2 (replace Week 1 placeholder)
  - No mock data (show reality)
  - Responsive grid layout
  - Live connection status
  - Widget sections for all widget types
- [ ] Create `/client/widgets.js` v2
  - ACTWidgetSystem manager class
  - EventSource connection to `/coordinate`
  - Event handlers for all 8+ event types
  - Widget update logic (receive JSON, render)
- [ ] Styling: Professional dark theme
- [ ] Error handling: Connection failures show troubleshooting
- [ ] Performance: Render 1000s of events without slowdown
- [ ] Mobile responsive

**Deliverable:** Professional client that streams real coordination intelligence

**Task 2.8: Integration Testing (12 hours)**
- [ ] E2E test suite: Agent register → Task assign → Progress → Complete → Improve
- [ ] Test with multiple agents (3-5 coordinating)
- [ ] Test conflict scenarios
- [ ] Test PVM retrieval mid-coordination
- [ ] Test FLUX State evaluation
- [ ] Test MCP tools
- [ ] Performance benchmarks: Latency, throughput, memory
- [ ] Load testing: 50+ agents, 100+ tasks

**Deliverable:** Comprehensive test suite proving MVP works reliably

**Task 2.9: Documentation & Setup (12 hours)**
- [ ] Quickstart guide: "Get ACT running in 5 minutes"
- [ ] Architecture diagram: Full system overview
- [ ] API documentation: All endpoints + message formats
- [ ] MCP setup: "Add 7 lines of JSON to claude.json"
- [ ] Vector DB setup: How to configure Qdrant
- [ ] Development guide: How to extend ACT
- [ ] Deployment guide: From local to production

**Deliverable:** Clear documentation for hackathon judges and future developers

**Completion Check (Week 3 End):**
- [ ] Full end-to-end coordination works with visualization
- [ ] MCP tools tested and documented
- [ ] All integration tests pass
- [ ] Performance meets targets
- [ ] Client loads and streams events
- [ ] Documentation complete

---

## Week 4: Optimization & Deployment (Dec 7-14)

### Goal
Polish, optimize, and prepare for hackathon submission + standalone deployment.

### Instance #1 Tasks: Hardening

**Task 1.10: Error Handling & Resilience (12 hours)**
- [ ] Comprehensive error handling
  - Graceful degradation: Coordination fails ≠ agent crashes
  - Retry logic: Exponential backoff for transient failures
  - Circuit breakers: Prevent cascade failures
  - Fallback modes: Coordination reduced but functional
- [ ] Health checks: Server reports status continuously
- [ ] Self-healing: Automatic recovery from common failures
- [ ] Error logging: Structured logs for debugging
- [ ] Unit tests: All error paths tested

**Deliverable:** Production-grade reliability

**Task 1.11: Performance Optimization (12 hours)**
- [ ] Profile server under load (100+ agents, 1000+ tasks)
  - Identify bottlenecks
  - Optimize hot paths
  - Memory usage analysis
  - CPU usage analysis
- [ ] Vector search optimization
  - Index optimization
  - Query batching
  - Result caching
- [ ] Event streaming optimization
  - Batch events when possible
  - Compression for large events
- [ ] Unit tests: Performance regression suite

**Deliverable:** Proven scalability metrics

**Task 1.12: Security Hardening (12 hours)**
- [ ] API key validation: ACT_API_KEY required for all endpoints
- [ ] Input validation: Sanitize all agent input
- [ ] Rate limiting: Prevent agent from spamming
- [ ] CORS: Proper origin restrictions
- [ ] Secrets management: API keys not logged
- [ ] SQL injection prevention: Parameterized queries
- [ ] Security tests: Attempt known attacks

**Deliverable:** Production-ready security posture

### Instance #2 Tasks: Deployment + Polish

**Task 2.10: Docker + Cloud Deployment (12 hours)**
- [ ] Dockerfile: Self-contained ACT server
- [ ] Docker Compose: ACT + Qdrant + PostgreSQL
- [ ] Environment configuration: Externalize all secrets
- [ ] Health checks: Liveness + readiness probes
- [ ] Logging: Structured JSON logs
- [ ] Deployment guides:
  - Local: `docker-compose up`
  - AWS: CloudFormation template
  - GCP: Kubernetes manifest
  - Heroku: One-click deploy button
  - Render: Simple config
- [ ] Testing: Verify each deployment method works

**Deliverable:** Production deployment options

**Task 2.11: Hackathon Submission Prep (12 hours)**
- [ ] README: Project overview + quick start
- [ ] Demo scenario: Step-by-step instructions
- [ ] Video: 3-5 min demo walkthrough
- [ ] Judges packet: Architecture diagrams + key insights
- [ ] GitHub: Clean repo with comprehensive docs
- [ ] License: Clear license for judge review
- [ ] Submission: Upload to hackathon platform

**Deliverable:** Polished hackathon submission

**Task 2.12: Standalone MVP Verification (12 hours)**
- [ ] Test: Fresh clone → docker-compose up → working coordination
- [ ] Test: MCP client registration via JSON config
- [ ] Test: Full task assignment → completion → evaluation cycle
- [ ] Test: PVM semantic search works
- [ ] Test: FLUX State unbiased evaluation
- [ ] Test: PAIR loop converges to improvement
- [ ] Benchmark: Performance meets targets
- [ ] Cleanup: No debug code, no TODOs

**Deliverable:** Production-ready MVP proven working end-to-end

**Completion Check (Week 4 End):**
- [ ] All error handling complete
- [ ] Performance optimized
- [ ] Security hardened
- [ ] Deployments tested
- [ ] Hackathon submission ready
- [ ] Documentation final
- [ ] MVP proven working
- [ ] Both instances sign off on quality

---

## Critical Path & Dependencies

```
Week 1:
├─ Instance #1: ACT Server Refactor + Chronological Log + Vector Interface
├─ Instance #2: Embeddings + Qdrant + MCP Skeleton
└─ Dependencies: None (parallel work)

Week 2:
├─ Instance #1: FLUX State + PAIR + Memory Evaluation (depends on Week 1)
├─ Instance #2: MCP Tools + Self-Improvement + Widgets (depends on Week 1)
└─ Dependencies: Both need Week 1 complete

Week 3:
├─ Instance #1: Endpoint Design + Event Streaming + Store Abstraction (depends on Week 2)
├─ Instance #2: Client + Integration Tests + Documentation (depends on Week 2)
└─ Dependencies: Both need Week 2 complete

Week 4:
├─ Instance #1: Error Handling + Performance + Security (depends on Week 3)
├─ Instance #2: Deployment + Hackathon Prep + Verification (depends on Week 3)
└─ Dependencies: Both need Week 3 complete
```

**No blocking dependencies between instances per week** → True parallel development

---

## Success Criteria per Week

### Week 1 Success
- [ ] Server compiles and starts without errors
- [ ] Chronological log persists and retrieves events
- [ ] Vector DB (Qdrant) runs locally and stores embeddings
- [ ] MCP server structure complete
- [ ] Both instances reviewed each other's architecture
- [ ] No integration blockers identified

### Week 2 Success
- [ ] FLUX State evaluation works on sample tasks
- [ ] PAIR retrieval finds relevant patterns
- [ ] Loop converges to 95%+ success
- [ ] MCP tools connect to ACT server
- [ ] Self-improvement engine runs without errors
- [ ] Widget system ready for rendering

### Week 3 Success
- [ ] Full end-to-end coordination with 5 coordinating agents
- [ ] Real-time event visualization works
- [ ] MCP clients can register and receive tasks
- [ ] Performance meets targets (100+ agents, 1000+ tasks)
- [ ] All integration tests pass
- [ ] Documentation complete and accurate

### Week 4 Success
- [ ] Production deployments work (Docker, cloud platforms)
- [ ] Hackathon submission polished and submitted
- [ ] Performance optimized (benchmarks document improvements)
- [ ] Security hardening complete
- [ ] MVP verified working end-to-end
- [ ] Ready for judges and future development

---

## Decision Points Requiring User Input

1. **Week 1 kickoff:** Confirm all 5 critical decisions finalized?
2. **Week 2 mid-week:** Any architectural concerns after seeing code?
3. **Week 3 mid-week:** Performance acceptable or need optimization?
4. **Week 4 pre-submission:** Hackathon platform and submission requirements?

---

## Timeline Reality Check

- **Total effort:** ~280 hours (40 hours per week × 2 people × 4 weeks)
- **Instance #1:** 140 hours (coordination engine, memory system, reasoning)
- **Instance #2:** 140 hours (MCP bridge, visualization, deployment)
- **Risk:** Architecture changes, integration surprises, performance issues
- **Mitigation:** Weekly status updates via coordination file, early integration tests

---

**This roadmap is ambitious but achievable with disciplined execution and clear communication through the coordination file.**

Let's build the semantic coordination intelligence layer that changes how AI agents work together.
