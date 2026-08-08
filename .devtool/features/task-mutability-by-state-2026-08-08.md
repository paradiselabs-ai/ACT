---
id: "task-mutability-by-state-2026-08-08"
status: "backlog"
priority: "medium"
created: "2026-08-08T16:10:00.000Z"
completedAt: null
labels: ["orchestrator", "server", "planner", "design"]
order: "a5"
---
# Task correction rules by lifecycle state (foundation for /pivot)

## Describe
Owner design (2026-08-08). When a task is wrong (bad spec, capability mismatch,
or a project pivot invalidates it), the correct handling depends ENTIRELY on how
far it got. Governing invariant: an edit must NEVER race an agent's pickup — an
agent that already read the old version will execute the old version; a task
already refused assignment will not be re-evaluated on silent edit (nothing
re-triggers matching). Four states:

1. CREATED, NOT DISPATCHED (rejected at gate / not yet queued): Planner may fix
   attributes and resubmit — ideally edit-in-place IF the server can guarantee
   no agent ever saw it; otherwise reject+verbatim-re-emit (shipping first, see
   unservable-task-capabilities-gate).
2. QUEUED, NOT PICKED UP (confirmed via ACT data, not vibes): allow edit ONLY
   with an atomic kill-and-replace — old task is dropped from the lineup in the
   same operation that queues the corrected one, so no agent can grab the stale
   version in the gap.
3. PICKED UP / IN PROGRESS: let it complete. New mechanism: Planner informs
   Assurance the task spec was wrong; Assurance rejects the submission WITH the
   correction, explicitly telling the swarm agent "rejected because the TASK was
   wrong, not your work — here is the corrected task."
4. COMPLETED + VALIDATED + SYNTHESIZED: corrective follow-up task: "previous
   task <desc> was completed but the instruction was wrong; correct instruction
   is <desc>; locate the original implementation and fix it" + full DISC.

State 4's pattern IS the project-pivot mechanism: a pivot is a batch of state-4
corrections plus cancellations in states 1-2.

## Success Criteria (design phase)
- Server audit: which states are distinguishable today from task status +
  assignment data; which transitions need new endpoints (kill-and-replace) vs
  exist already (abandon + create).
- The state-3 Assurance flow spec'd as a message type, not prompt vibes.

## Constraints
- Race-free by construction (server-side atomicity), never by timing luck.
