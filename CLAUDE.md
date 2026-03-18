# CLAUDE.md

This file provides guidance to Claude Code when working on the ACT codebase.

***

## Project Overview

**ACT (Agent Coordination Toolkit)** — the agentic harness for multi-agent CLI coordination. The harness IS the product: coordination patterns, context engineering, quality gates, memory architecture.

**Current Phase**: NesTTY branch — two-tier role hierarchy.

**Branch**: `NesTTY`

***

## Architecture: Two-Tier Role Hierarchy

### Tier 1 — Interactive (NesTTY Window)
| Role | Responsibility |
|------|---------------|
| **Planner** | ONLY decision-maker. Decomposes projects into SNLP task specs. Evidence-based routing via PVM. Assigns tasks to role IDs. |
| **Observer** | Monitors ChronLog/PVM. Surfaces bottlenecks, file conflicts, stuck tasks. No decisions. |
| **Assurance** | Two-layer validation: verifies agent's Ralph Wiggum Loop worked + independently scores @success_criteria (95% gate). |
| **QA/Synthesizer** | Assembles Assurance-validated outputs into final deliverable. Consults agents via targeted --print. |

### Tier 2 — The Swarm (Headless Agents)
The swarm agents execute tasks headlessly with role specializations: `frontend-dev`, `backend-dev`, `qa-engineer`, `researcher`, `developer` (default).
- Planner picks role mix per project → writes tasks to role IDs → Runner spawns swarm agents
- Self-bootstrap on spawn: `act context <agent-id> --project <name>`
- Session save: `act brief update` before exit
- Ralph Wiggum Loop: iterative self-verification before `act task complete` (Layer 1 validation)
- Role-based model selection: each role can use a different LLM via `.opencode.json` (e.g., cheap local model for routine coding, stronger model for research)

### Three-Layer Separation
```
Planner (LLM decisions) → ACT Server (deterministic state) → Runner (thin spawner) → Swarm
```
Planner never talks to Runner. Runner never asks Planner. Planner writes to ACT; Runner reacts.

***

## CRITICAL: Multi-Agent Coordination Protocol

### act-coordination.json
**ALWAYS READ** last 50-100 lines at session start. Append-only. Never modify existing entries.

```json
{
  "timestamp": "2026-03-16T00:00:00Z",
  "agent": "your_agent_id",
  "message": "What was done",
  "type": "feature_complete|architecture_decision|etc"
}
```

***

## Project Structure

```
ACT/
├── act-coordination.json          # CRITICAL: multi-agent coordination log (append-only)
├── CLAUDE.md                      # ← You are here
├── docs/nestty/                   # Architecture docs (READ FIRST for any design work)
├── docs/Harness/                  # Harness engineering principles + knowledge graphs
├── server/                        # ACT Coordination Server (TypeScript/Express/Socket.io)
│   ├── src/index.ts              # REST endpoints + Socket.io handlers
│   ├── src/services/
│   │   ├── AgentRegistry.ts      # Agent tracking
│   │   ├── TaskCoordinator.ts    # Task lifecycle
│   │   ├── EventHub.ts           # Message classification + routing
│   │   ├── ChronologicalLog.ts   # Append-only JSONL event log
│   │   ├── LocalEmbeddingVectorStore.ts  # Real embeddings (all-MiniLM-L6-v2, ACTIVE)
│   │   └── PVMIndexer.ts        # Indexes events into vector store
│   └── src/types/
│       └── roles.ts              # Role taxonomy (AgentRole enum, tiers, constraints)
├── runner/act-runner.mjs          # Headless agent spawner (thin launcher)
├── cli/                           # REPL with guided create project wizard
│   ├── act-repl.ts               # Main REPL
│   ├── act-cli.ts                # Agent CLI (act context, act task, etc.)
│   └── act-client.ts             # HTTP client
├── act-agent/                     # OpenCode fork (Go) — --agent + --nestty modes
└── nestty/                        # NesTTY orchestrator (future — Block 4)
```

***

## What Works

### ACT Server
- Agent registration (REST + Socket.io)
- Task creation, assignment, progress, completion (with result in `task.metadata.result`)
- Project + brief storage (in-memory, event sourcing replay rebuilds on restart)
- Message classification + routing (EventHub) with rate limiting (3/30s per agent)
- ChronologicalLog: append-only JSONL at `./data/coordination-log.jsonl`
- PVM search via LocalEmbeddingVectorStore (real all-MiniLM-L6-v2 embeddings)
- A2A protocol: Agent Cards + task push endpoints
- File locking: claim/release

### Runner
- Spawns agent CLI subprocesses (configurable via `AGENT_CLI` env var)
- AGENT.md brief injection from ACT server
- liveProcesses Map — no-clone enforcement (singleton per agent-ID)
- Complexity-triggered PVM context injection (heuristic score > 4)
- Parallel agent awareness + proactive coordination messages
- Session lifecycle logging on process exit

### CLI
- REPL: `create project`, `list agents/projects`, `show project`, `default agent`, `status`, `help`
- `act` CLI: 21 commands — `register`, `context`, `task complete/progress/retry/submit-for-validation`, `brief update`, `pvm search`, `validation queue`, `files claim/release`, `message`, `log`, `graph task/unverified/conflicts`, `status`, `codebase impact/rules/communities/onboard`

***

## Key Concepts

### SNLP (Syntactic Natural Language Programming)
Task/project specification language. `@` for structure, `>` for NL directives. CTD progression (each section depends on what's above). `@success_criteria` = Assurance's validation checklist.

### PVM (PAIRed Vector Minutes)
Dual-purpose semantic memory: team coordination patterns + individual agent skill profiles. ChronLog + vector store. Evidence-based routing.

### A2A Protocol
Task delegation: Planner pushes tasks to ACT targeting specific role IDs. Agent Cards expose capabilities. NOT for conversation.

### `act` CLI (Agent Interface)
~50-100 tokens vs 47K for MCP schemas. 21 commands: `register`, `context`, `task complete/progress/retry/submit-for-validation`, `brief update`, `pvm search`, `validation queue`, `files claim/release`, `message`, `log`, `graph task/unverified/conflicts`, `status`, `codebase impact/rules/communities/onboard`.

### Three Knowledge Graphs
1. **Code KG (Nomik)** — ships with ACT as runtime capability. Agents get project codebase graph.
2. **Skill Graph** — markdown + wikilinks replacing AGENT.md monolith. Progressive loading.
3. **Coordination KG** — deferred. In-memory maps for now. Neo4j/Kuzu when FLUX needs causal traversal.

### FLUX State (future)
Unbiased self-evaluation via selective memory suppression + causal PVM re-injection. Requires Coordination KG causal edges. NOT the same as Assurance validation.

***

## Development Commands

```bash
# ACT Server
cd server && npm install && npm run dev     # port 8080, tsx watch

# CLI REPL
cd cli && npx tsx act-repl.ts

# NesTTY (multi-agent conversation — requires server running)
npx tsx nestty/index.ts --project <name> --server http://localhost:8080
MOCK_AGENT=1 npx tsx nestty/index.ts --project <name>  # mock agents for testing

# Or from REPL:
#   nestty                         — launch with all 4 roles
#   nestty --roles planner,observer — specific roles
#   nestty --mock                   — mock agents

# Nomik (codebase KG — requires Docker + Neo4j on port 7687)
nomik rules              # architecture violations
nomik communities        # functional clusters
nomik scan:incremental . # update graph after changes
```

### Environment Variables
- `ACT_SERVER_URL` — server URL (default: `http://localhost:8080`)
- `PORT` — server port (default: `8080`)
- `AGENT_CLI` — agent binary override (default: `./act-agent/act-agent`)
- `ANTHROPIC_API_KEY` / `GROQ_API_KEY` / `OPENROUTER_API_KEY` — LLM provider keys
- `LOCAL_ENDPOINT` — local model server URL (Ollama, LM Studio, vLLM)

### Provider Configuration

Each role can use a different LLM model. Configure in `.opencode.json` (see `.opencode.example.json`):

```json
{
  "agents": {
    "planner":      { "model": "claude-opus-4-20250514", "maxTokens": 8000 },
    "observer":     { "model": "claude-sonnet-4-20250514", "maxTokens": 2000 },
    "assurance":    { "model": "claude-sonnet-4-20250514", "maxTokens": 5000 },
    "qa_synthesizer": { "model": "claude-sonnet-4-20250514", "maxTokens": 5000 },
    "developer":    { "model": "claude-sonnet-4-20250514", "maxTokens": 5000 }
  }
}
```

**Supported providers:** Anthropic, OpenAI, Gemini, Groq (free tier), OpenRouter (free models), Bedrock, Azure, VertexAI, xAI, Copilot, Local (Ollama/LM Studio).

**Cost strategy:** Don't skimp on Planner (use strongest model). Swarm agents can use cheaper/local models. Groq free tier (Llama 3.3 70B) works for routine coding tasks.

Any role not configured in `.opencode.json` falls back to the `coder` default.

***

## Common Pitfalls

1. **act-coordination.json is append-only.** Never edit existing entries.
2. **Task creation is REPL-only.** Agents consume via `get_task`, never create.
3. **QdrantVectorStore.ts has a pre-existing TypeScript error.** Don't fix unless wiring Qdrant.
4. **tsx must be installed locally** in `server/` (`npm install -D tsx`).
5. **MCP bridge removed.** ACT CLI (`act`) replaces all MCP tools with ~50-100 tokens vs 47K schema overhead.
6. **LocalEmbeddingVectorStore is already active.** PVM embeddings are NOT a build priority.

***

## Build Order (Current)

See `docs/nestty/BUILD_ORDER.md` for full details.

**Gate 0**: `claude --print "test"` works ✅
**Block 1**: Brief injection ✅, CLAUDE.md update ✅, Role taxonomy ✅
**Block 2**: ChronLog replay ✅, `act` CLI ✅, A2A protocol ✅, Runner improvements ✅, Task title field ✅
**Block 3**: act-agent opencode fork (--agent + --nestty modes)
**Block 4**: NesTTY orchestrator
**Block 5**: Assurance validation + QA/Synthesizer assembly
**Block 6**: SNLP parser
**Future**: FLUX State

***

## Terminology — Standard Multi-Agent Names

| CORRECT | DO NOT USE |
|---------|-----------|
| Planner | Director |
| Observer | Operator |
| Assurance / Judge | Production Assistant / PA |
| QA / Synthesizer | Producer |
| Swarm / Swarm agent | Actor / Headless agent |
| Tier 1 / Tier 2 (The Swarm) | Upper hierarchy / Execution layer |

***

## Handoff Protocol

If the user says **"this is a handoff"** or **"handoff session"**, read `.claude/HANDOFF.md` for session continuity — it contains what was done, what's in progress, and what's next. **Do NOT read HANDOFF.md otherwise** — it may be stale and is only relevant when explicitly invoked.

***

## Documentation

- `docs/nestty/` — **Primary architecture docs** (9 files covering all roles, protocols, harness, SNLP, infrastructure, build order)
- `docs/Harness/` — Harness engineering principles, knowledge graphs, Deep Agents research, MCO concept
- Full concept alignment: see plan file `serialized-weaving-spring.md`
