---
id: "swarm-recovery-observer-zombie-detection-2026-05-12"
status: "backlog"
priority: "high"
assignee: null
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: null
labels: ["swarm-recovery", "observer", "tier1", "epic-swarm-recovery"]
order: "a6"
---

# Observer: detect zombie subprocesses

Add a 7th anomaly rule to `detectAnomalies` in `act-agent/internal/app/orchestrator.go` so Observer flags the specific failure mode that motivates this whole epic: an agent whose `liveProcess` heartbeat is stale while its assigned task has already been retried/reassigned.

## Why

Discovered 2026-05-12 — see `act-coordination.json`. Today's 6 rules (stuck task, stale lock, idle agent, unvalidated work, conflicts, etc.) don't catch the zombie case. Without detection, the Planner has no signal to fire the new abort tool. Observer is the eyes; this rule gives it the right vocabulary.

## Acceptance criteria

- [ ] New `Category` constant: `CategoryZombieAgent`
- [ ] New rule in `detectAnomalies` (`orchestrator.go:1314+`):
  ```
  For each agent A with liveProcess registered:
    If A.lastHeartbeat > zombieThreshold (suggest 4 min) AND
       A.currentTaskId is null OR A.currentTask.status in {pending, aborting, aborted, completed, validated}:
      emit Anomaly{Severity: critical, Category: CategoryZombieAgent,
        Message: "Agent A's subprocess (PID ?) is still registered but its task moved to <status>. Likely zombie."}
  ```
- [ ] New constant `zombieThresholdMinutes = 4` colocated with `stuckTaskMinutes`
- [ ] Anomaly auto-routes to Planner via existing path so Planner sees: "zombie detected on dev-2, task X already moved to pending — recommend POST /api/agents/dev-2/abort"
- [ ] Unit test in `orchestrator_test.go`: synthesize StatusSnapshot with zombie scenario, assert anomaly emitted; healthy scenario asserts no anomaly
- [ ] Observer prompt (`observer.go`) updated to mention rule #7 in the listed detection rules

## Files

- `act-agent/internal/app/orchestrator.go`
- `act-agent/internal/app/orchestrator_types.go` (Category constant)
- `act-agent/internal/app/orchestrator_test.go`
- `act-agent/internal/llm/prompt/observer.go` (rule list update)

## Depends on

- `swarm-recovery-task-states-2026-05-12` (uses `aborting`/`aborted` in the rule)

## Blocks

Nothing — last task in the epic.

## Notes

The "stale heartbeat" check assumes Runner already sends periodic registration heartbeats. Verify in `act-runner.mjs` registration loop; if absent, this task expands to add a 60s heartbeat ping. Either way the surface is small.
