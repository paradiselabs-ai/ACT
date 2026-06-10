# ⛔ NEVER TRUST SINGLE-FILE SOURCES — THE CODEBASE IS THE ONLY TRUTH

**NEVER trust single-file sources** (`architecture-flows.json`, this CLAUDE.md, `planner-prompts.json`, any one or few `docs/` files, HANDOFF files, kanban frontmatter) **to give reliable data about the current state of fixes, implementations, identified bugs, broken features, or desired features** — or any other information that directly affects steps moving forward.

Why this is non-negotiable:
- Assuming a bug is **un**fixed when it IS fixed → a duplicate fix lands on top of the real one → new bugs.
- Assuming a feature is **un**implemented when it IS implemented (e.g. ACP) → a second implementation → two half-broken, conflicting, destructive systems that are very hard to synthesize.
- This failure mode is not hypothetical: the Round-6 dual-path audit had BOTH independent analysis paths rank HIGH a "bug" that was already fixed, because both trusted a stale map file over the code. Only re-grepping the code caught it.

**ALWAYS pull directly from the codebase itself**: grep and validate everything, and do manual file reads of the most vital parts. A doc citing `file:line` is NOT evidence — line numbers drift within commits on active branches; re-grep before trusting any citation.

**When a doc and the code disagree: the code is the truth. Fix or omit the doc statement. NEVER change the codebase to make a doc statement true.**

> `architecture-flows.html/.json` are currently **known stale** — do not consult them until they are rebuilt against the current codebase.

***

## 🚧 TEMPORARY FOCUS (active 2026-06-10 — remove when the cleanup system has proven itself)

Current priority: **project cleanup + standardizing the methodology and constitution of progress tracking.** Master plan: `/Users/user/.claude/plans/cleanup-constitution.md`. Work happens on `feat/cleanup-constitution` (off `feat/remove-nomik`); merges back when scaffolding is clean and current; TUI e2e runs on the merged branch; PR → `NesTTY` only when alpha-worthy.

The system being built (lives in `docs/constitution/` when done):

1. **One always-current picture of development state.** Standards + workflows for: updating docs, reporting bugs, keeping a growing task log ordered by importance — every task scoped with a spec, Success Criteria, constraints, and **code-level invariants**.
2. **A real update loop** keeping these in sync with the code: `architecture-flows.json/html`, `planner-prompts.json/html`, kanban (`.devtool/`), `CLAUDE.md`, `README.md`, `HANDOFF.md`/`F-handoff.md`.
3. **Better doc organization**: upkeep, pruning, developer notes, ideas, future roadmap, dev log; public-facing docs (GitHub / DeepWiki) separated from internal dev-state docs.
4. **Discoverability for every actor** — agents, users, AI editors (Windsurf / Devin Desktop): where to find, write, update, and read docs; how to structure a doc page; which loops (manual + hooked) keep each artifact fresh.
5. **Where to publish feature implementations when completed** — a defined publication flow so finished work is recorded in one predictable place.

Verification pipeline for this effort (in order): `F-handoff.md` → verify handoff → subagents with explicit **anti-trust operating mentality** (grep-only, never assume) → dual-path-analysis on the aggregated result → verify dual-analysis results → identify where statements don't match the codebase → identify the truth → fix or omit false info **in the scaffolding only** — never by changing the codebase to match.

***

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
- **Coordination flow**: Human → Planner (INTAKE: 5-question conversation → `PROJECT_BRIEF:` → BUILD: `CREATE_TASK:` directives) → ACT server → Runner spawns swarm agents → swarm executes → Runner calls `act-agent task complete` then `act-agent task submit-for-validation` → Assurance validates → QA assembles → Planner reports to human.
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

Users change backends with `/swarm <role> <backend>` (in the TUI) or `act-agent swarm set <role> <backend>` (CLI). The bulk form is `/swarm all claude-code` or `act-agent swarm set all claude-code`. **Backend selection only applies to Tier 2** — Tier 1 agents are in-process goroutines and have no executable to swap.

**Other swarm details:**
- Planner picks role mix per project → writes tasks to role IDs → Runner spawns swarm agents
- Self-bootstrap on spawn: `act-agent context <agent-id> --project <name>`
- Session save: `act-agent brief update` before exit
- Ralph Wiggum Loop: iterative self-verification before `act-agent task complete` (Layer 1 validation)
- Role-based model selection: each role can use a different LLM via `~/.act.json` (e.g., cheap local model for routine coding, stronger model for research)
- Capability-based routing: each Runner registers with capability tags from `runner.DefaultCapabilities[role]` so the server's `assignOptimalAgent` matches tasks to the right role

### Three-Layer Separation
```
Planner (LLM decisions) → ACT Server (deterministic state) → Runner (thin spawner) → Swarm
```
Planner never talks to Runner. Runner never asks Planner. Planner writes to ACT; Runner reacts.

***

## Team & Dual-Development Workflow

ACT is a two-founder project as of 2026-05-07. Domain ownership splits the codebase to prevent merge conflicts and eliminate cross-bottlenecks. **Both founders ship in parallel from day one — neither waits on the other to finish "their part" first.**

### Domain ownership

| Owner | Domain | Subdirectories | First-authority decisions |
|-------|--------|----------------|---------------------------|
| **Project owner** | Architecture / Orchestrator / Server | `act-agent/internal/app/`, `act-agent/internal/llm/`, `act-agent/cmd/`, `server/`, `act-agent/cli/`, `act-agent/runner/` | Coordination protocol, Tier 1 prompts, server contracts, runner behavior, validation pipeline, model registry |
| **Cofounder (Domain C)** | TUI + UX | `act-agent/internal/tui/` | Bubbletea v2 rendering, keyboard handling, scroll/viewport, message layout, demo videos, all `tui-*` kanban items |

**The boundary between them is `app.Service` (the orchestrator API surface).** TUI consumes orchestrator events; orchestrator emits events. Neither side reaches into the other's internals.

### Branch protocol

- **`main`** — untouched until alpha tag (`v0.1.0-alpha.1`).
- **`NesTTY`** — shared integration branch. Both founders pull from it, branch off it, PR back to it.
- **Feature branches:** `feat/<short-task-name>` per task. Examples: `feat/tui-scroll`, `feat/observer-watchdog`. Branch off `NesTTY`, not `main`.
- **PRs target `NesTTY`**, not `main`. When all alpha items merge into NesTTY and tests pass, NesTTY merges to main with the alpha tag attached.
- **No direct commits to `NesTTY`.** Always via PR.
- **Branch protection (set on GitHub before alpha):** require PR before merge to main; require linear history (squash-merge); no force pushes.

### Review etiquette

- **Inside a domain:** owner opens, owner merges. Other founder doesn't review unless tagged.
- **At the boundary (API contract changes — `app.Service` signature, event schemas, REST contracts):** owner opens, other founder reviews within 24h, merge after sign-off.
- Reviews focus on **contract correctness, blast radius, test coverage**. Not style — domain owner's call inside the domain.

### Communication rhythms

- **Daily 60-second async ping** in Discord/Slack: "yesterday: X, today: Y, blocking: Z." Not a status report.
- **Weekly 30-min sync** — high-level only. Calendar invite, recurring same day each week.
- **Architecture decisions affecting the boundary** — write a `decisions/<short-name>.md` in `docs/Vault/decisions/` (gitignored), drop link in Discord, wait 24h for ack or pushback before merging.
- **Async-first.** Synchronous calls only when async fails three times.

### Commit conventions

Conventional Commits: `<type>(<scope>): <description>`. Types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `perf`. Scope = roughly the domain (`server`, `tui`, `runner`, `orchestrator`, etc.).

Examples:
- `feat(server): add per-project ChronLog rotation`
- `fix(tui): wire scroll-up keybinding to chat list viewport`
- `docs: update CLAUDE.md with team workflow`

### Anti-pattern check

If either founder catches themselves doing any of these, stop and recalibrate:
- "I should wait for the other founder to finish before starting" — NO. Parallel from day one.
- "Should I ask permission before merging this internal-domain change?" — NO. Inside your domain, you decide.
- "I should review every PR they open even on internal-domain stuff" — NO. Boundary changes only.
- "We should pair-program on this" — RARE. Only for specific cross-domain knowledge transfer. Otherwise async > sync.

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
- `act` CLI: `register`, `context`, `task complete/progress/retry/submit-for-validation`, `brief update`, `pvm search`, `validation queue`, `files claim/release`, `message`, `log`, `graph task/unverified/conflicts`, `status`

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
~50-100 tokens vs 47K for MCP schemas. Commands: `register`, `context`, `task complete/progress/retry/submit-for-validation`, `brief update`, `pvm search`, `validation queue`, `files claim/release`, `message`, `log`, `graph task/unverified/conflicts`, `status`.

### Knowledge Graphs
1. **Code KG** — no persistent graph. ACT matches Claude Code's bet: codebase intelligence is agentic search (Grep/Glob/Read) + context discipline, not an index. (The Nomik/Neo4j integration was removed — the Docker dependency was an ops burden and never on the alpha critical path.)
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
act-agent                      # in any project directory
act-agent --project my-app     # for a specific project

# Headless/internal modes (not for users)
act-agent --agent dev-1 --role developer -p "..."  # spawned by Runner for Tier 2 swarm agents
act-agent -p "single query"                         # OpenCode single-turn mode (legacy)
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

**Supported providers:** Anthropic, OpenAI, Gemini, Groq (free tier), OpenRouter (free models), Bedrock, Azure, VertexAI, xAI, Local (Ollama / LM Studio / vLLM).

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
7. **LocalEmbeddingVectorStore is already active.** PVM embeddings are NOT a build priority. **Caveat:** the embedding pipeline is real; the analytics layer on top of it (`getAgentProfile`, `compareAgents`, `getAgentSynergy`, `SelfImprovementEngine`) currently returns placeholder data (`successRate: 0.85 + Math.random() * 0.15`, four `// placeholder` methods). When a user asks "is PVM real," answer scoped: embeddings real, analytics fake. Do not conflate them.
8. **Verifying "is X real" requires running X, not finding X.** When the user asks whether code is real or uses placeholder data, the verification owed is: run the method, inspect what it returns, AND grep the implementation for `Math.random`, `placeholder`, `// TODO`, `// FIXME`, `hardcoded`, `mock`, `stub`, `return 0\.[0-9]`. A file that exists with real imports underneath can still have fake methods on its surface. Past sessions failed this by reporting "PVM is real" after finding the embedding layer was real, without checking the analytics methods downstream. State explicitly which layers were verified and which weren't.

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

## Delegated /plans (multi-Claude execution)

When writing a `/plan` that the user intends to execute across multiple Claude instances in parallel, every delegated task block must include:

1. **Dependency block** — explicit list of upstream tasks (by ID or name) this task depends on. Include a line: *"Do NOT attempt to fulfill dependency requirements yourself — wait until the dependency is marked complete in `act-coordination.json` before starting."* The goal is strict serialization of dependent work; parallel instances must not race to fill each other's gaps.
2. **Success criteria** — concrete, testable outcomes. Not "works correctly" — actual conditions: files exist at paths X/Y/Z, `go build` clean, function returns shape Q, endpoint responds with schema R. The criteria must be precise enough that a reviewer can verdict pass/fail without judgment calls.
3. **Code constraints** — exact shape of the fix: which files to touch, which to leave alone, no side-effect refactors, no speculative abstractions, no "while I'm here" cleanup. Constraints exist to prevent drift from ACT's philosophy: minimal surface area, trust the framework, no defensive scaffolding, no re-exports/compat shims, comments only for non-obvious *why*, no backwards-compat hacks (see CLAUDE.md "Doing tasks" + "Executing actions with care").
4. **Philosophy alignment** — success criteria and constraints together must make it obvious *how* best practices apply for this specific task. Reference the relevant ACT architectural principle (Three-Layer Separation, `act-coordination.json` append-only, Tier 1 in-process, `~/.act.json` as config truth, etc.) so the delegated instance doesn't re-derive it.
5. **Coordination protocol** — each delegated instance must:
   - Read the last 50–100 lines of `act-coordination.json` before starting any task.
   - Append a `task_start` entry with its task ID and timestamp before beginning work.
   - Append `task_progress` entries at meaningful checkpoints (not every file save — at decision points, blockers, dependency resolutions).
   - Append a `task_complete` entry with a summary of what was done and which success criteria were verified, before moving on.
   - Check for dependency-fulfillment entries from other instances at every checkpoint — not on a timer.
   - Never edit existing `act-coordination.json` entries; only append.

A delegated `/plan` task that omits any of these produces drift, races, or duplicated work across instances. All five are required, even for "small" tasks — the coordination overhead is the point.

**Discovery convention.** Write the plan file at `/Users/user/.claude/plans/<name>.md` AND append a coord entry of type `plan_delegated` pointing the target instance at it (fields: `plan_file`, `target_agent`, `phase`). Other Claude instances discover their work by grepping `act-coordination.json` for their agent ID — not by the author remembering to paste the path. If the plan file isn't linked from coord, the delegation didn't happen.

***

## Kanban Board (`.devtool/`)

Bugs and feature requests are tracked as markdown files in `.devtool/`:
- `.devtool/features/` — active tasks (any status except `done`)
- `.devtool/features/done/` — completed tasks (note: subdirectory of `features/`, not a sibling)

Each task is a single `.md` file with YAML frontmatter. Example:

```markdown
---
id: "tui-scroll-up-to-see-current-session-history-2026-04-21"
status: "todo"
priority: "medium"
assignee: null
dueDate: null
created: "2026-04-21T17:16:00.885Z"
modified: "2026-04-21T17:16:00.885Z"
completedAt: null
labels: ["TUI", "design"]
order: "a0"
---
# TUI Scroll up to see current session history

<body — description of the bug or feature>
```

**Frontmatter fields:**
- `id` — kebab-case slug + ISO date suffix
- `status` — one of: `backlog`, `todo`, `in-progress`, `review`, `done`
- `priority` — one of: `low`, `medium`, `high`, `critical`
- `assignee` — default `null` unless user specifies
- `dueDate` — default `null` unless user specifies
- `created` / `modified` / `completedAt` — ISO timestamps (`completedAt` null until `done`)
- `labels` — custom tag array (e.g., `["TUI", "design"]`)
- `order` — lexicographic sort key within column (e.g., `"a0"`)

When a task reaches `done`, move the file from `.devtool/features/` → `.devtool/features/done/` and set `completedAt`.

***

## Handoff Protocol

If the user says **"this is a handoff"** or **"handoff session"**, read `.claude/HANDOFF.md` for session continuity — it contains what was done, what's in progress, and what's next. **Do NOT read HANDOFF.md otherwise** — it may be stale and is only relevant when explicitly invoked.

***

## Visual codebase mapping

Three artifacts at repo root map the entire NesTTY codebase honestly:

- `architecture-flows.html` — single-file interactive diagram. Open offline via `file://`. Filter chips per category, multi-select flow overlays in five colors (gold, blue, green, magenta, cyan).
- `architecture-flows.json` — sibling JSON, byte-identical to the inline `<script type="application/json">` block in the HTML. Used for machine verification.
- `flows-explainer.html` — companion with a Findings headline at top covering six gap items (Ralph prompt-only, QA partial-deliverable persistence, Socket.io vestigial, no-auth trust boundary, Qdrant build-excluded, kanban doc-only).

Status taxonomy distinguishes code-enforced from prompt-wished behavior:
- `ok` — code-verified. Every step's `detail` cites `file:line`.
- `prompt-only` ⌕ — behavior in prompt text only.
- `gap-found` ⚠ — documented but no implementing code, or partially implemented.
- `unverified` is forbidden in shipped artifacts; it exists as a construction-time scratch marker only.

**The method doc is `.claude/architecture-flows-method.md`.** Read it before rebuilding any of these artifacts. It contains the amended §4c (grep upfront, sub-agent claims must be re-grep'd by the parent before encoding), the JSON schema, the rendering rules, and the post-completion verification command. Rebuild when a REST endpoint changes, a Tier 1 or Tier 2 role file appears or disappears, a coordination protocol step changes, or a claim in the method doc is contradicted by reality.

***

## Documentation

- `docs/Vault/Agent Coordination Toolkit/nestty/` — **Primary architecture docs** (9 files covering all roles, protocols, harness, SPIL, infrastructure, build order)
- `docs/Harness/` — Harness engineering principles, knowledge graphs, Deep Agents research, MCO concept
- `docs/ARCHITECTURE_PATTERNS.md` — **5 patterns from Claude Code analysis** to implement independently (context compaction, deferred tool discovery, pre/post hooks, prompt caching split, autoDream memory consolidation). Includes references, priorities, and where each fits in ACT.
- Full concept alignment: see plan file `serialized-weaving-spring.md`
