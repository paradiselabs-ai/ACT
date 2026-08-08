---
id: "validation-seen-key-strands-resubmissions-2026-08-08"
status: "review"
priority: "critical"
assignee: null
dueDate: null
created: "2026-08-08T08:30:00.000Z"
modified: "2026-08-08T08:30:00.000Z"
completedAt: null
labels: ["orchestrator", "assurance", "bug", "critical"]
order: "a0"
---
# Validation seen-key strands resubmitted tasks — Assurance never re-validates rejected work

## Describe
`orchestrator.go::routeToAssurance` marked `validation:<taskID>` seen after EVERY
submitted verdict, pass or fail. The in-code comment claimed "the seen-key is per
submission" / "new key lifecycle" — false: the key is the bare task ID, stable across
resubmissions, and `seenTasks` entries are never deleted. Consequence: a task rejected
by Assurance, reworked by the swarm agent, and resubmitted is permanently skipped by
`checkPendingValidation` (`alreadySeen` hit) and strands in `submitted_for_validation`.
Never surfaced before because no prior live run had Assurance mass-reject work.

## Repro / Evidence (2026-08-08 LinkDock e2e, agy-all-roles run)
- `server/data/coordination-log.jsonl`: 4 tasks rejected 0/100 (08:09–08:11Z), all 4
  resubmitted (e.g. `d993cc4d` at 08:12:17Z), zero re-validation events after.
- `linkdock/.act/debug.log` 03:19:11 local: `tier1_watchdog.fire role=assurance
  pending=4 last_turn_ago=7m29s` — poller alive but skipping all pending via seen-keys
  (assurance_poll_start fired only for the NEW 03:17 task batch).

## Fix (applied same day)
`markSeen("validation:"+t.ID)` only when `verdict.Passed`. Failed verdicts leave the
key unseen: the server routes the task back to `assigned` immediately (leaves the
pending list), and on resubmission `incAttempt` still bounds total validation cycles
at `maxValidationAttempts` before escalating to the Planner — no unbounded loop.
False comment replaced with the true lifecycle.

## Success Criteria
- A task rejected then resubmitted appears in a subsequent `assurance_poll_start` and
  receives a second verdict (live e2e evidence quoted here before `done`).
- Escalation path intact: >3 validation cycles → `⚠ validation stuck` + Planner escalation.
- `go build` clean, `go test ./internal/app/` green (both verified at fix time).

## Constraints / Invariants
- Touch only `routeToAssurance` seen-marking; poller loop, attempt cap, QA path unchanged.
- `markSeen` for validation keys reachable ONLY on `verdict.Passed == true` or the
  `maxValidationAttempts` escalation branch in `checkPendingValidation`.
