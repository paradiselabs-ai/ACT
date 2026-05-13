---
id: "swarm-pvm-task-decision-event-and-filter-2026-05-12"
status: "todo"
priority: "medium"
assignee: "d34d"
epic: "swarm-context"
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: null
labels: ["swarm", "pvm", "context", "chronlog"]
order: "b2"
---
# PVM swarm-side enrichment — task_decision event type + re-injection filter

Two related issues:

## Issue 1 — PVM-for-swarm is currently marginal value

ChronLog indexes coordination metadata (`task_created`, `task_completed`, `task_failed`, `agent_registered`, `brief_stored`). NOT technical decisions or code artifacts. Vector search returns "dev-3 completed similar task on 2026-04-12" — coordination noise, not actionable code/pattern signal. Planner-side PVM works (Planner queries coordination patterns by design). Swarm wants code/decision patterns, gets the wrong thing.

**Fix:** new event type `task_decision` distinct from `task_completed`:
```
{
  agentId: "dev-3",
  taskId: "...",
  decision: "used bcrypt with cost=12 for password hashing",
  context: "auth flow / signup endpoint",
  reusable: true
}
```
Capture mode: either (a) Tier 2 prompt teaches the agent to emit a `task_decision` via new `act-agent decision record --decision <text> --context <text>` CLI, called during task execution when a non-obvious choice is made; or (b) Assurance, when scoring success criteria, extracts decisions and writes them. Recommend (a) — agent is the source of truth for what was a "decision" vs incidental code.

Indexer (`PVMIndexer.ts`) treats `task_decision` as the primary swarm-PVM retrieval source. `task_completed` stays for coordination/Planner-side queries.

## Issue 2 — PVM re-injection without dedup signal

PVM has no view into the agent's writable memory. Re-injects the same top-K patterns every spawn even if the agent already absorbed them. Not a "thread" problem (swarm agents have no thread); a "PVM-doesn't-know-what-agent-knows" problem.

**Fix:** server endpoint `POST /api/pvm/retrieve` adds `excludePatterns: [...]` param. Runner extracts "applied pattern X" entries from the agent's writable brief, passes IDs to exclude. Eliminates the same-K-every-spawn pathology.

**Depends on:** `swarm-next-task-preamble-readonly-brief-2026-05-12` (uses the writable brief section to track applied patterns).

**Files:**
- `server/src/services/ChronologicalLog.ts` (new event type).
- `server/src/services/PVMIndexer.ts` (index `task_decision`).
- `server/src/index.ts` (`/api/pvm/retrieve` filter param).
- `act-agent/cli/act-cli.ts` (new `decision record` subcommand).
- `act-agent/internal/llm/prompt/{developer,frontend_dev,backend_dev,qa_engineer,researcher}.go` (teach the protocol).
- `act-agent/runner/act-runner.mjs` (`fetchPVMContext` passes excludePatterns).

**Success criteria:**
- New event type round-trips through ChronLog replay.
- Vector retrieval returns `task_decision` entries before `task_completed` for swarm queries.
- Manual: run two consecutive tasks with PVM hits; verify second task's PVM context does not re-list patterns the first task's brief recorded as applied.
- Build + vet + jest clean.
