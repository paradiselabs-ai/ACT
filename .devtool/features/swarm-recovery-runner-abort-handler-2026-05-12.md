---
id: "swarm-recovery-runner-abort-handler-2026-05-12"
status: "backlog"
priority: "high"
assignee: null
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: null
labels: ["swarm-recovery", "runner", "epic-swarm-recovery"]
order: "a3"
---

# Runner: handle abort_signal, kill child PID, finalize

Runner subscribes to the new `abort_signal` Socket.io event. When the event matches its own `AGENT_ID`, it kills the child subprocess, captures whatever partial output exists, and reports it back via `/api/tasks/:id/abort-finalize` before the runner loop continues.

## Why

Discovered 2026-05-12 — see `act-coordination.json`. The Runner's `liveProcesses` Map (`act-runner.mjs:93`) is per-process state with no external control surface. Planner cannot reach in to kill a hung subprocess. This task closes that loop.

## Acceptance criteria

- [ ] `act-runner.mjs` opens a Socket.io client at startup (was REST-poll-only)
- [ ] Subscribes to `abort_signal` events
- [ ] On match (`event.agentId === AGENT_ID`):
  - [ ] Read current `child.pid` from `liveProcesses.get(AGENT_ID)`
  - [ ] Send `SIGTERM` first; after 5s grace, escalate to `SIGKILL`
  - [ ] Capture `child.stdout` buffer accumulated so far as `partialResult` (string, capped at 16KB tail)
  - [ ] Capture which `act task progress` calls were made (already in liveProcess record) as `completedSteps`
  - [ ] POST to `/api/tasks/:taskId/abort-finalize` with the captured data
  - [ ] Remove from `liveProcesses` map
  - [ ] Continue runner loop normally
- [ ] If abort arrives between task assignment and child spawn → mark task aborted with empty partialResult, no kill needed
- [ ] If abort arrives for an `AGENT_ID` not currently running → log warning, no-op (covered by 409 in endpoint task)
- [ ] node:test cases: SIGTERM path, SIGKILL escalation path, no-process path, finalize POST failure handling

## Files

- `act-agent/runner/act-runner.mjs`
- `act-agent/runner/act-runner.test.mjs`

## Depends on

- `swarm-recovery-abort-endpoint-2026-05-12` (event source)
- `swarm-recovery-partial-result-2026-05-12` (finalize endpoint)

## Blocks

- `swarm-recovery-brief-injection-2026-05-12`
