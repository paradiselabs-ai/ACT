---
id: "pvm-role-attribution-joins-wrong-event-2026-08-13"
status: "done"
priority: "high"
assignee: null
dueDate: null
created: "2026-08-13T17:30:00.000Z"
modified: "2026-08-19T21:45:00.000Z"
completedAt: "2026-08-19T21:45:00.000Z"
labels: ["server", "PVM", "memory"]
order: "a1"
---
# PVM: outcome→worker attribution joins task_assigned, which is mostly payload-less

## Spec
`LocalEmbeddingVectorStore.getProjectOutcomes()` resolves "who did this task" by joining
`task_validated` / `task_validation_failed` to `task_assigned` on `taskId`. In real history
that join almost always misses: **24 validations, 5 assignment records with a payload, 1
joinable task.** Unattributable tasks become the `unknown` role and are then dropped from the
routing brief, so real outcomes silently vanish from evidence.

`task_completed` carries `data.agentId` on **all 37** historical events and is not consulted.

## Success Criteria
- Attribution resolves from `task_completed` (authoritative worker) with `task_assigned` as a
  fallback, not the reverse.
- On real history, the count of validated-but-unattributed tasks drops to ~0.
- `perRole` totals equal the number of validation events for the bucket (no silent drops).
- A one-line log or status field reports how many validations were unattributable, so a future
  regression is visible instead of silent.

## Constraints
- Read-only change to the analytics path; do not change event emission in this ticket
  (that is `pvm-outcome-events-untagged-by-project-2026-08-13`).
- Keep dropping the `unknown` bucket from the *brief text* — but count it in status output.
- No new event types.

## Invariants (code-level)
- The join key stays `data.taskId || data.task?.id || data.id`.
- `roleOfAgent(agentId, capabilities)` keeps capability-based classification as the fallback
  after the ID-prefix map; do not add project-name prefixes to the prefix table (agent IDs like
  `authsvc-beta-backend-1` must classify from capabilities, not from their first segment).

## Repro / Evidence
`docs/audits/memory-system-audit-2026-08-13.md` §2.3 contains the counting script.
Note the ID-prefix trap found live: `roleOfAgent` takes `agentId.split('-')[0]`, so a
project-prefixed ID (`authsvc-beta-backend-1` → `authsvc`) never matches the prefix table.

## Resolution note (2026-08-19)
Fixed by opus-task-a: attribution now resolves worker from task_completed.agentId first, task_assigned fallback; unattributable count exposed in PVM status (+1 log line). Real-history check: 24 validations, 0 unattributed (was 1 joinable). roleOfAgent prefix table untouched; project-prefixed IDs classify via capabilities.
