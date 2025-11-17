# Phase 5 Critical Pre-Implementation Decisions

**Date:** November 16, 2025
**Status:** Decision Point - Block Until Resolved
**Purpose:** Define the 5 critical architectural decisions before Instance #1 and Instance #2 begin coding

---

## The 5 Blocking Decisions

### 1. Vector Database Choice

**Decision Point:** Which vector DB for PVM semantic indexing?

**Options:**
- **Qdrant** (Lightweight, open-source, easy to embed, excellent performance)
  - Pros: Battle-tested, minimal dependencies, can run in-process or standalone
  - Cons: Less mature than commercial options
  - Typical deployment: Local docker-compose for dev, managed Qdrant for production

- **Milvus** (Scalable, enterprise-grade, distributed)
  - Pros: Massive scale support, advanced replication, proven at enterprise
  - Cons: Higher operational complexity, more dependencies
  - Typical deployment: Kubernetes cluster for production

- **Weaviate** (Purpose-built for RAG, managed SaaS available)
  - Pros: Purpose-built for semantic search, great docs
  - Cons: Younger project, less battle-tested than Qdrant
  - Typical deployment: Managed SaaS cloud

- **Pinecone** (Managed cloud vector DB)
  - Pros: Zero operational overhead, fast to get started
  - Cons: Vendor lock-in, monthly costs, less control
  - Typical deployment: Cloud only

**Recommendation for MCP Hackathon:** **Qdrant**
- Reason: Can embed in ACT server, zero external dependencies for MVP
- Can scale to managed Qdrant later if needed
- Perfect for hackathon: works immediately, no cloud setup required

**Decision Required:** Choice of vector DB + deployment strategy (embedded vs. standalone vs. cloud)

---

### 2. Embedding Model Choice

**Decision Point:** How do we convert coordination events → vectors for semantic search?

**Options:**

**Lightweight Open-Source (Embedded in ACT):**
- **sentence-transformers/all-MiniLM-L6-v2** (384-dim, 22MB)
  - Pros: Small, open-source, free, instant deployment
  - Cons: Slightly lower quality than commercial models (but sufficient for coordination events)
  - Use case: Perfect for MVP, hackathon, self-hosted

- **sentence-transformers/all-mpnet-base-v2** (768-dim, 420MB)
  - Pros: Better quality, still open-source and free
  - Cons: Larger model, slower inference
  - Use case: If quality is critical

**Cloud API (Requires Keys):**
- **OpenAI text-embedding-3-small** (1536-dim, $0.02 per million tokens)
  - Pros: Excellent quality, most mature
  - Cons: Requires API key, costs money, vendor lock-in
  - Use case: Production deployments where quality matters

- **Anthropic embeddings** (when available)
  - Pros: Native integration with Claude
  - Cons: Not yet available
  - Use case: Future option

- **Google Vertex AI embeddings**
  - Pros: Good quality, integrated with Google ecosystem
  - Cons: Requires Google Cloud account
  - Use case: Companies already using GCP

**Recommendation for MCP Hackathon:** **sentence-transformers/all-MiniLM-L6-v2**
- Reason: Zero dependencies, runs immediately, sufficient quality for coordination event vectors
- No API keys required
- Can be swapped out later without changing architecture

**Decision Required:** Embedding model + whether to support pluggable embeddings (OpenAI fallback, etc.)

---

### 3. Chronological Log Persistence

**Decision Point:** How do we persist the append-only coordination log?

**Options:**

**File-Based (Simple, portable):**
- **JSONL (JSON Lines)** - One JSON object per line, append-only
  - Pros: Human-readable, universal, simple, perfect for git history
  - Cons: Slower for large files, needs occasional compaction
  - Use case: Perfect for MVP, local development

- **SQLite** - Single-file embedded database
  - Pros: Transactional safety, queryable, standard ACID
  - Cons: Single-file limitation for clustering
  - Use case: Good for local deployments

**Database-Based (Scalable):**
- **PostgreSQL** - Mature, powerful, distributed
  - Pros: Battle-tested, replication, complex queries
  - Cons: Requires external service, more ops complexity
  - Use case: Production deployments

- **DynamoDB** (AWS) or **Firestore** (GCP)
  - Pros: Serverless, auto-scaling, managed
  - Cons: Vendor lock-in, costs, less control
  - Use case: Cloud-first deployments

**Hybrid Approach:**
- **Local:** JSONL for development, git-tracked
- **Production:** PostgreSQL for reliability
- **Sync:** Simple migration script between formats

**Recommendation for MCP Hackathon:** **JSONL with SQLite option**
- Reason: JSONL is human-readable, appendable, perfect for MVPs
- Can be git-tracked to show coordination history
- SQLite for production readiness if needed
- Migration path to PostgreSQL clear

**Decision Required:** Primary persistence + whether to support swappable storage backends

---

### 4. MCP Server Implementation

**Decision Point:** How do we expose ACT through MCP?

**Options:**

**Simple Approach (Fast):**
- Build MCP bridge that exposes ACT capabilities as tools
- ACT server already exists (port 8080)
- MCP server connects to ACT server, wraps capabilities as tools
- Agents connect to MCP, which routes to ACT

**Architecture:**
```
Agent -> MCP Server -> ACT Server (port 8080)
```

**Integrated Approach (Cleaner):**
- Build MCP server that IS the ACT server
- Agents connect directly via MCP protocol
- No separate HTTP/WebSocket connection needed
- Single unified interface

**Architecture:**
```
Agent -> MCP Server (which IS ACT)
```

**Hybrid Approach (Most Flexible):**
- Keep ACT server as-is (HTTP/WebSocket)
- Build MCP server as optional distribution layer
- Agents can use either MCP or direct HTTP
- Both work, support both

**Architecture:**
```
Agent -> (MCP Server) -> ACT Server
  or
Agent -> (HTTP) -> ACT Server (both work)
```

**Recommendation for MCP Hackathon:** **Hybrid Approach**
- Reason: ACT server stays clean and focused
- MCP becomes distribution mechanism, not core
- Backward compatible with existing integrations
- Proven pattern: OpenAI also has both API and MCP

**Decision Required:** Whether to integrate MCP into ACT server or build as bridge

---

### 5. MVP Scope for Phase 5 (Weeks 1-4)

**Decision Point:** What MUST work for Phase 5 MVP vs. what's future work?

**MUST HAVE (Week 1-2):**
- ✅ Chronological log persistence (append-only)
- ✅ Vector DB integration (semantic search)
- ✅ FLUX State reasoning engine (memory wipe + evaluation)
- ✅ PAIR retrieval workflow (RAG context injection)
- ✅ MCP server basics (expose core coordination capabilities)
- ✅ Task assignment with PVM context
- ✅ Agent registration with capability matching

**SHOULD HAVE (Week 3):**
- ✅ Self-improvement engine (explicit /improve command)
- ✅ Background learning tasks
- ✅ SSE widget visualization
- ✅ Conflict resolution using PVM
- ✅ Agent message broadcasting

**NICE TO HAVE (Week 4+, after MVP):**
- ⏳ Enterprise multi-team security (namespace isolation)
- ⏳ Federated coordination (multiple ACT servers)
- ⏳ Advanced ML optimization of task assignment
- ⏳ Custom evaluation metrics
- ⏳ Agent-specific skills/plugins

**EXPLICITLY OUT OF SCOPE (Phase 5):**
- ❌ Agent execution frameworks (ACT is coordination only)
- ❌ Custom agent SDKs (documentation first, SDKs if demand)
- ❌ Production deployment automation (Phase 6)
- ❌ Compliance frameworks (Phase 6)
- ❌ Advanced security (Phase 6)

**Recommendation:** Scope strictly to MUST HAVE + SHOULD HAVE
- Reason: MCP hackathon deadline is likely coming
- Core features demonstrate semantic coordination intelligence
- Rest builds on solid foundation

**Decision Required:** Exact scope agreement + definition of "MVP success criteria"

---

## Decision Process

### Before Instance #1 and Instance #2 Begin Coding

1. **User approval** of all 5 decisions
2. **Document rationale** in act-coordination.json
3. **Instance #1 and #2 acknowledge** decisions in coordination file
4. **Begin implementation** only after consensus

### Decision Documentation Template

```json
{
  "decision_id": "decision_vector_db_001",
  "title": "Vector Database Selection",
  "timestamp": "2025-11-16T12:00:00Z",
  "decision": "Qdrant embedded for MVP, PostgreSQL for production",
  "reasoning": "...",
  "approved_by": ["user", "claude_code_instance_1"],
  "blocks": ["vector_memory_store_implementation", "pvm_semantic_search"],
  "depends_on": ["embedding_model_choice"]
}
```

---

## Why These 5 Matter

1. **Vector DB** - Affects query performance, deployment complexity, scalability
2. **Embeddings** - Affects semantic search quality, API costs, portability
3. **Persistence** - Affects audit trail reliability, scalability, deployment options
4. **MCP** - Affects distribution, agent adoption, integration complexity
5. **Scope** - Affects timeline, quality, MVP definition, success metrics

**Any wrong choice here cascades through all implementation work.**

That's why we decide BEFORE coding starts.

---

## Recommended Decisions (Summary)

| Decision | Recommendation | Why |
|----------|---------------|-----|
| Vector DB | Qdrant embedded | Zero deps, fast to MVP, scales later |
| Embeddings | sentence-transformers/all-MiniLM-L6-v2 | Free, portable, sufficient quality |
| Persistence | JSONL + SQLite | Simple, git-trackable, production-ready option |
| MCP | Hybrid (bridge approach) | Clean separation, backward compatible |
| Scope | MUST + SHOULD only | MCP hackathon focus, quality foundation |

**All recommendations support fast shipping + long-term quality.**

---

## Next: User Decision

Please confirm:
1. Accept these 5 decisions or propose alternatives?
2. Any blocking concerns with recommendations?
3. Ready to document in coordination file and begin Phase 5 implementation?

Once approved → Instance #1 + Instance #2 begin parallel implementation.
