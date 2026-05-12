---
id: "swarm-recovery-task-states-2026-05-12"
status: "backlog"
priority: "high"
assignee: null
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: null
labels: ["swarm-recovery", "server", "epic-swarm-recovery"]
order: "a1"
---

# Server: add 'aborting' and 'aborted' task states

Extend `TaskCoordinator`'s status state machine with two new terminal-adjacent states so the abort lifecycle is explicit and queryable.

## Why

Discovered 2026-05-12 — see `act-coordination.json`. Without dedicated states, an aborted task is indistinguishable from a stuck one (`in_progress` forever) or a normally-failed one (`failed`). Observer can't tell zombies from healthy work; retry logic can't decide whether partial progress is recoverable.

## State transitions to allow

```
in_progress  → aborting    (Planner-initiated)
aborting     → aborted     (Runner confirms kill + finalize)
aborting     → completed   (Runner finished naturally before kill landed — preserve)
aborted      → pending     (retryTask resets for re-dispatch)
```

## Transitions to block (return 409)

- `aborted → in_progress` (must go through pending first)
- `validated → aborting` (validated is terminal-good)
- `completed → aborting` (already done; aborting a completed task is meaningless)

## Acceptance criteria

- [ ] `Task.status` type union extended in `server/src/types/`
- [ ] `TaskCoordinator.transitionState()` permits the four allowed edges, blocks the rest
- [ ] `KI-10`-style 409 `TERMINAL_STATE_TRANSITION` for forbidden transitions (consistent with existing pattern)
- [ ] Jest tests covering all four allowed + all three blocked transitions
- [ ] `assignOptimalAgent` skips agents whose `currentTask` is in `aborting`
- [ ] Server log emits `task_aborting` and `task_aborted` events; ChronLog records both

## Files

- `server/src/services/TaskCoordinator.ts`
- `server/src/types/` (status union)
- `server/test/TaskCoordinator.test.ts`

## Depends on

`swarm-recovery-abort-endpoint-2026-05-12` (endpoint triggers the state changes)

## Blocks

- `swarm-recovery-partial-result-2026-05-12`
- `swarm-recovery-observer-zombie-detection-2026-05-12`
