---
id: "swarm-next-task-preamble-readonly-brief-2026-05-12"
status: "todo"
priority: "high"
assignee: "d34d"
epic: "swarm-context"
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: null
labels: ["swarm", "context", "brief", "persistence"]
order: "b1"
---
# Swarm brief redesign — next-task preamble + read-only / writable split

Two coupled changes that together replace agent-self-disciplined memory with structured persistence. Land in one PR.

## Part A — Next-task preamble + forced brief-update directive

Swarm agents currently rely on themselves to remember to call `act-agent brief update` before exit. If they forget, prior state is lost. There's also no signal about what's coming next, so they can't write the brief with the next task in mind.

**Fix:**
- New endpoint `GET /api/tasks/next?agentRole=X&afterTask=Y` returns `{ id, title, oneLineDescription }` or `null`.
- Runner reads this in `buildTaskPrompt` and appends a section:
  ```
  ## Next Task Queued For You
  After this one: "{title}" — {desc}.

  As your FINAL ACTION, call `act-agent brief update` with information
  from this task the next one needs. Remove anything no longer relevant.
  ```
- Server soft-enforces: if `task_complete` fires without a matching `brief_updated` event in the same task window, log a `brief_update_skipped` warning event in ChronLog.

## Part B — Read-only / writable brief split

Current `project.briefs[agentId]` is a single string the agent can overwrite freely. Critical info (role, project domain, tech stack, hard constraints) can be erased during a cleanup pass.

**Fix:**
- Change `project.briefs[agentId]: string → { readonly: string, writable: string }`.
- `readonly` is set once at project-brief creation by the Planner (from `PROJECT_BRIEF` JSON). Never mutated by `brief update`. Contents: project domain, tech stack, role definition, success-criteria boilerplate, hard constraints (e.g. "no Co-Authored-By in commits"), glossary for caveman-protocol abbreviations (see separate kanban).
- `writable` is what `brief update` edits.
- Runner injects both: `## Project Constants (immutable)` block then `## Working Notes (your scratchpad)` block.
- Schema migration: existing string briefs → `writable`, `readonly` empty until Planner re-emits `PROJECT_BRIEF`.

**Files:**
- `server/src/index.ts` (brief storage shape, new endpoint, soft-enforce warning).
- `server/src/services/ChronologicalLog.ts` (`brief_updated` / `brief_update_skipped` event types).
- `act-agent/runner/act-runner.mjs` (`fetchAgentBrief` returns both blocks; `buildTaskPrompt` renders both + next-task preamble).
- `act-agent/cli/act-cli.ts` (`brief update` writes to `writable` field only; reject attempts on `readonly`).
- `act-agent/internal/app/orchestrator.go` (`handleProjectBrief` populates `readonly` from `PROJECT_BRIEF` JSON).

**Success criteria:**
- Fresh project end-to-end: Planner emits brief, swarm agent receives both blocks, calls `brief update`, only writable changes.
- `task_complete` without brief update logs warning event (visible via `act-agent log`).
- ChronLog replay reconstitutes both blocks correctly.
- Build + vet + jest clean.
