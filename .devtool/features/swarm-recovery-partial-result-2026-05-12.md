---
id: "swarm-recovery-partial-result-2026-05-12"
status: "backlog"
priority: "high"
assignee: null
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: null
labels: ["swarm-recovery", "server", "epic-swarm-recovery"]
order: "a2"
---

# Server: persist partial results across abort/retry

When a swarm agent is aborted mid-task, store whatever progress it had into `task.metadata.partialResult` so the next agent assigned to that task can resume instead of restarting from zero.

## Why

Discovered 2026-05-12 — see `act-coordination.json`. Today `task.metadata.result` only gets set on `complete`. `act task progress` writes a chat-line message, never a checkpoint blob. An abort + retry today = full restart, all prior tokens wasted, file conflicts likely. CLAUDE.md "Multi-Agent Coordination Protocol" requires append-only coord log; partial results fit that pattern (each progress checkpoint is an append).

## Acceptance criteria

- [ ] New endpoint: `POST /api/tasks/:id/abort-finalize` accepting `{ partialResult: string, reason: string, completedSteps: string[] }`
- [ ] Persists into `task.metadata.partialResult`, `task.metadata.abortReason`, `task.metadata.completedSteps`
- [ ] Transitions task `aborting → aborted`
- [ ] `TaskCoordinator.retryTask()` preserves `partialResult` (does NOT clear `metadata`) when resetting to `pending`
- [ ] New Task type field: `metadata.partialResult?: string`, `metadata.completedSteps?: string[]`
- [ ] Existing `act task progress` extended to optionally accept `--checkpoint <blob>` writing to `metadata.partialResult` even without abort (defensive — survives crashes too)
- [ ] Jest tests: abort-finalize sets fields, retry preserves them, normal complete clears them

## Files

- `server/src/index.ts`
- `server/src/services/TaskCoordinator.ts`
- `server/src/types/` (Task interface)
- `server/test/TaskCoordinator.test.ts`
- `act-agent/cli/act-cli.ts` — extend `task progress` with `--checkpoint`

## Depends on

`swarm-recovery-task-states-2026-05-12` (needs `aborting`/`aborted` to exist)

## Blocks

- `swarm-recovery-brief-injection-2026-05-12`
