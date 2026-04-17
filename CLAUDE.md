
## Project Overview

**ACT (Agent Coordination Toolkit)** — the agentic harness for multi-agent CLI coordination. The harness IS the product: coordination patterns, context engineering, quality gates, memory architecture.

**Current Phase**: NesTTY branch — two-tier role hierarchy.

**Branch**: `NesTTY`

***

## NesTTY (Nested TTY)

> **THE TUI IS NesTTY.** When you run `act`, you launch the OpenCode-fork TUI, and that TUI _is_ the multi-agent window. There is no separate orchestrator process. There is no separate "NesTTY mode" to launch. The 4 Tier 1 agents run as goroutines inside the same Go binary, sharing one chat session, with their messages interleaved and color-coded by role. If you find yourself thinking "I need to launch the NesTTY orchestrator," stop — you're already inside it.

NesTTY = multiple agent REPLs sharing one terminal window. The TUI IS NesTTY.

- **One window, four agents**: Planner, Observer, Assurance, QA all have their chat messages interleaved in a single conversation view, color-coded by role.
- **Background execution**: Each agent's actual work (tool calls, file I/O, bash commands, LLM API calls) runs in a background process/goroutine. Only chat-level responses surface into the shared window.
- **Agent behaviors**:
  - **Planner** — the ONLY agent that responds to human input. The human's conversational partner.
  - **Observer** — silent background watchdog on a ~120s loop. Only injects a message when anomalies detected (stuck tasks, file conflicts, idle agents with pending work, unresponsive agents, bottlenecks, duplicate assignments).
  - **Assurance** — event-driven. Activates when a swarm agent submits work for validation. Scores `@success_criteria` items (95% gate).
  - **QA/Synthesizer** — event-driven. Activates when Assurance passes validated work. Assembles into final deliverable.
- **Coordination flow**: Human → Planner (INTAKE: 5-question conversation → `PROJECT_BRIEF:` → BUILD: `CREATE_TASK:` directives) → ACT server → Runner spawns swarm agents → swarm executes → Runner calls `act task complete` then `act task submit-for-validation` → Assurance validates → QA assembles → Planner reports to human.
- **INTAKE mode**: When the Planner detects a new project (server returns 404 for the project name), it runs a 5-question intake conversation (description, techStack, constraints, successCriteria, agentsInvolved), summarizes, asks "Ready to start?", then emits `PROJECT_BRIEF: {json}` on confirmation. The orchestrator parses it and POSTs to `/api/projects`. Only after that does it switch to BUILD mode and start creating tasks.

***

## Architecture: Two-Tier Role Hierarchy

### Tier 1 — Interactive (NesTTY Window)
| Role | Responsibility |
|------|---------------|
| **Planner** | ONLY decision-maker. Decomposes projects into SPIL task specs. Evidence-based routing via PVM. Assigns tasks to role IDs. |
| **Observer** | Monitors ChronLog/PVM. Surfaces bottlenecks, file conflicts, stuck tasks. No decisions. |
| **Assurance** | Two-layer validation: verifies agent's Ralph Wiggum Loop worked + independently scores @success_criteria (95% gate). |
| **QA/Synthesizer** | Assembles Assurance-validated outputs into final deliverable. Consults agents via targeted --print. |

### Tier 2 — The Swarm (Headless Agents)
The swarm agents execute tasks headlessly with role specializations: `frontend_dev`, `backend_dev`, `qa_engineer`, `researcher`, `developer` (default).

**Parallel from startup.** When the orchestrator starts (first message in the TUI), it spawns ONE Runner subprocess per swarm role — five Node.js processes by default, each polling the ACT server for tasks matching its role/capabilities. Tasks dispatched by the Planner execute concurrently, not sequentially. This is the difference between "swarm" and "queue".

**Per-role backend selection.** Each swarm role can use one of two backends:
- `act-agent` (default) — the local Go binary, configured via per-role model in `~/.act.json`
- `claude-code` — the official Claude Code CLI (`claude --print --dangerously-skip-permissions`)

Users change backends with `/swarm <role> <backend>` (in the TUI) or `act swarm set <role> <backend>` (CLI). The bulk form is `/swarm all claude-code` or `act swarm set all claude-code`. **Backend selection only applies to Tier 2** — Tier 1 agents are in-process goroutines and have no executable to swap.

**Other swarm details:**
- Planner picks role mix per project → writes tasks to role IDs → Runner spawns swarm agents
- Self-bootstrap on spawn: `act context <agent-id> --project <name>`
- Session save: `act brief update` before exit
- Ralph Wiggum Loop: iterative self-verification before `act task complete` (Layer 1 validation)
- Role-based model selection: each role can use a different LLM via `~/.act.json` (e.g., cheap local model for routine coding, stronger model for research)
- Capability-based routing: each Runner registers with capability tags from `runner.DefaultCapabilities[role]` so the server's `assignOptimalAgent` matches tasks to the right role

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
├── docs/Vault/Agent Coordination Toolkit/nestty/  # Architecture docs (READ FIRST for design work)
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
├── act-agent/                     # OpenCode fork (Go) — the `act` command
│   ├── internal/llm/prompt/      # Role-specific system prompts (13 files)
│   │   ├── planner.go            # Tier 1: ONLY decision-maker
│   │   ├── observer.go           # Tier 1: Monitor + report
│   │   ├── assurance.go          # Tier 1: Validate @success_criteria (95% gate)
│   │   ├── qa_synthesizer.go     # Tier 1: Assemble validated outputs
│   │   ├── developer.go          # Tier 2: General full-stack
│   │   ├── frontend_dev.go       # Tier 2: UI/UX specialist
│   │   ├── backend_dev.go        # Tier 2: API/DB specialist
│   │   ├── qa_engineer.go        # Tier 2: Testing only
│   │   ├── researcher.go         # Tier 2: Analysis, not code
│   │   ├── common.go             # Shared prompt building blocks + env/LSP helpers
│   │   └── prompt.go             # Dispatcher (routes role → prompt)
│   ├── internal/act/client.go    # Native HTTP client for ACT server
│   ├── cli/                      # Agent CLI (21 commands, TS)
│   └── runner/act-runner.mjs     # Headless swarm agent spawner
```

> The TypeScript NesTTY prototype (the old `nestty/` directory) was deleted in the cleanup pass. All of its logic — turn management, CREATE_TASK parsing, validation routing — lives in **`act-agent/internal/app/orchestrator.go`** as Go goroutines inside the TUI process. There is no NesTTY directory to run; the TUI _is_ NesTTY.

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
- **`submit-for-validation` after every successful `task complete`** — this is what feeds Assurance + QA. Skipping it leaves the validation pipeline as dead code.
- 409 self-heal on registration (deletes stale agent + retries)
- Process groups (`Setpgid`) so the parent kills the entire subtree on shutdown
- `SweepOrphans()` at startup runs `pkill -f act-runner.mjs` defensively
- Subprocess stdout/stderr → `~/.act/runners/<role>.log` (no chat pollution)

### TUI / Orchestrator (Phase 3 deltas)
- **Token diet**: `prompt/common.go::getEnvironmentInfo` no longer runs `ls .`; `bash.go` description trimmed 2300→200 tokens; per-role tool subsets (`Tier1ToolsForRole`) — Planner/Observer get just `bash`, Assurance/QA get `bash + view + grep`. Tier 1 LLM requests dropped ~22K → ~5-7K tokens.
- **Context paths**: defaultContextPaths reduced to `["ACT.md", "ACT.local.md"]`. CLAUDE.md is no longer auto-injected (was injecting ~20K tokens when running `act` inside the ACT repo).
- **Lazy swarm spawn**: Runners spawn on the first `CREATE_TASK`, not on every `act` launch.
- **Coordination event surface**: `coordinationEventLoop` polls `/api/log` every 3s and surfaces task lifecycle events as system messages in the chat (`📝 task created`, `✓ dev-1 completed`, `📤 submitted for validation`, `✅ validation passed`, etc.).
- **Per-turn 5-min timeout**: `runAgentTurn` wraps `agentSvc.Run` with `context.WithTimeout`. On expiry: cancels the agent and emits a system message.
- **Internal prompt marker**: `app.InternalPromptMarker` (`\x00ACT_INTERNAL\x00`) prepended to non-Planner inputs so the TUI hides Observer/Assurance/QA prompts but still shows their outputs with role banners.
- **Auto-route Tier 1 → Planner**: Observer/Assurance/QA messages trigger a Planner turn via `autoRoutePlanner`. Recursion guard `consecutiveAutoTurns < 5`. Reset on every human input. Asymmetric — Planner replies don't auto-route.
- **Tier 1 visibility**: startup pings (`👁 Observer online` etc.), Observer health pings every ~8 min when no anomalies, role banners cached only after role tagging.
- **Server auto-start**: `EnsureServerRunning` wired into `cmd/root.go::RunE`. `findServerScript` and `findCLIScript` use `~/.act/config.json::actRoot` first, then walk from binary path, then cwd fallback.

### CLI
- REPL: `create project`, `list agents/projects`, `show project`, `default agent`, `status`, `help`
- `act` CLI: 21 commands — `register`, `context`, `task complete/progress/retry/submit-for-validation`, `brief update`, `pvm search`, `validation queue`, `files claim/release`, `message`, `log`, `graph task/unverified/conflicts`, `status`, `codebase impact/rules/communities/onboard`

### TUI (the NesTTY window)
- `act-agent/internal/app/orchestrator.go` — coordination logic: CREATE_TASK parsing, Observer monitoring loop, validation routing to Assurance, QA synthesis routing, message ownership tracking
- 4 Tier 1 agents created in `app.go::New()` — Planner, Observer, Assurance, QA — each with its own LLM provider (configurable per role in `~/.act.json`)
- Human input → Planner via `Orchestrator.HandleHumanInput()`
- Background goroutines for Observer polling, validation polling, QA polling
- Message rendering with role-based colors via `internal/tui/components/chat/message.go`

***

## Key Concepts

### SPIL (Structured Progressive Instruction Language)
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
# ACT Server (must be running before act TUI)
cd server && npm install && npm run dev     # port 8080, tsx watch

# Build the act TUI binary
cd act-agent && /opt/homebrew/bin/go build -o act-agent .

# Launch the TUI (the NesTTY window — Planner + Observer + Assurance + QA in one terminal)
act                      # in any project directory
act --project my-app     # for a specific project

# Headless/internal modes (not for users)
act --agent dev-1 --role developer -p "..."  # spawned by Runner for Tier 2 swarm agents
act -p "single query"                         # OpenCode single-turn mode (legacy)

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

Any role not configured in `~/.act.json` falls back to `agents.developer`. There is no further fallback — ACT dispatches by explicit role only.

***

## Common Pitfalls

1. **act-coordination.json is append-only.** Never edit existing entries.
2. **Tasks are created by the Planner agent inside the TUI.** It parses `CREATE_TASK:` directives from the Planner's responses and POSTs them to the server. Swarm agents consume tasks via the `act` CLI, never create them.
3. **There is no separate NesTTY orchestrator process.** The TUI _is_ NesTTY. The old `nestty/` directory was deleted — all coordination logic lives in `act-agent/internal/app/orchestrator.go`.
4. **QdrantVectorStore.ts has a pre-existing TypeScript error.** Don't fix unless wiring Qdrant.
5. **tsx must be installed locally** in `server/` (`npm install -D tsx`).
6. **MCP bridge removed.** ACT CLI (`act`) replaces all MCP tools with ~50-100 tokens vs 47K schema overhead.
7. **LocalEmbeddingVectorStore is already active.** PVM embeddings are NOT a build priority.

***

## Build Order (Current)

See `docs/Vault/Agent Coordination Toolkit/nestty/BUILD_ORDER.md` for full details.

**Gate 0**: `claude --print "test"` works ✅
**Block 1**: Brief injection ✅, CLAUDE.md update ✅, Role taxonomy ✅
**Block 2**: ChronLog replay ✅, `act` CLI ✅, A2A protocol ✅, Runner improvements ✅, Task title field ✅
**Block 3**: act-agent opencode fork (--agent + --nestty modes) ✅
**Block 4**: NesTTY orchestrator ✅ (ported to Go, lives in `act-agent/internal/app/orchestrator.go`)
**Block 5**: Assurance validation + QA/Synthesizer assembly ✅ (server-side complete, in-TUI routing in Phase 2)
**Block 6**: SPIL parser
**Future**: FLUX State, architecture patterns from `docs/ARCHITECTURE_PATTERNS.md`

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

- `docs/Vault/Agent Coordination Toolkit/nestty/` — **Primary architecture docs** (9 files covering all roles, protocols, harness, SPIL, infrastructure, build order)
- `docs/Harness/` — Harness engineering principles, knowledge graphs, Deep Agents research, MCO concept
- `docs/ARCHITECTURE_PATTERNS.md` — **5 patterns from Claude Code analysis** to implement independently (context compaction, deferred tool discovery, pre/post hooks, prompt caching split, autoDream memory consolidation). Includes references, priorities, and where each fits in ACT.
- Full concept alignment: see plan file `serialized-weaving-spring.md`
