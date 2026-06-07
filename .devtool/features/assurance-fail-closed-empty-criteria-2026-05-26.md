---
id: "assurance-fail-closed-empty-criteria-2026-05-26"
status: "in-progress"
priority: "high"
assignee: null
dueDate: null
created: "2026-05-26T23:10:00.000Z"
modified: "2026-06-07T05:00:00.000Z"
completedAt: null
labels: ["assurance", "validation", "correctness"]
order: "v01"
---
# Assurance fails open on empty `@success_criteria`

## Status update (2026-06-07, Fix 23 / commit 578d280)

**The CRITICAL fail-open is closed at the verdict-parser layer (the "Where to fix"
item #3).** `parseValidationVerdict` now sets
`Passed = OverallScore >= passThreshold && len(CriteriaResults) > 0`, so an empty
criteriaResults array can no longer produce a pass, and an otherwise-passing
verdict force-failed for empty criteria gets an explanatory `gaps` message.
Regression tests: `TestParseValidationVerdict_EmptyCriteriaFailsClosed` (the exact
wordtallies repro) + `_EmptyCriteriaBrokenVerdict`.

This is the ticket's **option 2 (fail-closed at the gate)** — the minimal safety
net that unblocks alpha. The ticket *recommended* **option 1 (refuse + re-queue +
immediate Planner re-emit)**; that self-healing UX is NOT yet built.

**Still open (kept this ticket in-progress, not done):**
- Option-1 re-queue loop: orchestrator immediately routes the Planner to re-emit
  with criteria, instead of the current burn-retries-then-`validation_stuck` path.
- Layer #1 — server-side gate (`server/src/index.ts`): reject zero-criteria
  submissions with a specific 400 before Assurance is invoked.
- Layer #2 — Assurance prompt (`assurance.go`): make the refusal clause explicit
  for the zero-criteria case (defense in depth; the LLM should also refuse).
- Success-criteria items not yet met: an orchestrator chat **system message**
  stating "no criteria attached", and the **e2e-api.sh** assertion.
- Separate follow-up: reject a criteria-less CREATE_TASK at dispatch (Planner-
  output-quality; the under-specified 45-byte directive).

## Problem

When a task is submitted for validation with no `@success_criteria` block,
Assurance scores it **100% by default** instead of failing or refusing.
Observed live during the wordtallies test session:

```
task_id  : 9c7bdb39
title    : '' (empty)
criteria : 0
score    : 100
feedback : "Task submitted with no @success_criteria block (title
            'Untitled task', empty criteria list). With nothing to score
            against, there is no basis to fail; agent reports wordtally.py
            and test_tally.py were created and tests pass."
```

Planner correctly observed the deliverable deviated from the original spec.
Assurance had no spec to check against, so it rubber-stamped. The 95%
validation gate became a 100% bypass via a malformed input.

This breaks the whole point of two-layer validation. The work might be
right, might be wrong — but Assurance is supposed to verify against
criteria, and "no criteria" should be a verification *failure*, not a
verification *pass*.

## Upstream context

The empty-criteria submission came from a 45-byte CREATE_TASK that the
Planner emitted between two valid task batches (wordtallies session,
18:00:05). That's a separate Planner-output-quality issue (see followup
ticket), but Assurance should be the safety net — even if Planner drifts,
Assurance should not auto-pass.

## Fail-closed semantics

Pick one (orchestrator should make this explicit, not let Assurance LLM
decide case-by-case):

1. **Refuse the submission.** Assurance returns an empty string (already
   the documented behavior for "no pending task" — extend to "task lacks
   criteria"). Orchestrator re-queues the task and asks Planner to re-emit
   with explicit criteria.

2. **Score 0, gate fails.** Task transitions to `failed`, retry counter
   increments, planner gets the gap report.

3. **Escalate to human.** "Cannot validate — no criteria attached. Original
   spec was: ... Submitted result was: ... Pass / Reject?"

Recommended: **#1 (refuse + re-queue)** for the alpha. The re-queue path
already exists for other validation failure modes, and the planner-side
re-emission gives the system a self-healing loop instead of a hard stop.

## Where to fix

Two places, both should be hardened:

1. **Server-side gate** (`server/src/index.ts` validation submission
   handler): reject submissions whose task has zero criteria with a
   400 / specific error so the runner / planner sees the failure
   *before* Assurance gets the chance.

2. **Assurance prompt** (`act-agent/internal/llm/prompt/assurance.go`):
   add to the refusal clause — if `@success_criteria` count is zero, the
   verdict MUST be either an empty response or `{"passed": false, "score":
   0, "gaps": "missing success criteria", ...}`. Right now the prompt
   tolerates the empty-criteria case implicitly.

3. **Verdict parser** (`parseValidationVerdict`): if `criteriaResults` is
   empty AND no `gaps` text is present, treat the verdict as malformed —
   do not write it to the task's metadata as a "pass".

## Success criteria

- Submitting a task with no `@success_criteria` block produces a
  `validation_failed` event in the chronlog, not a `validation_passed`.
- The orchestrator surfaces a system message that says exactly why the
  validation failed (no criteria attached), not just a generic failure.
- The Planner is auto-routed to re-emit the task with explicit criteria
  (existing autoRoutePlanner path).
- A regression test in `parseValidationVerdict_test.go` covers
  "verdict with empty criteriaResults and score=100 is rejected as
  malformed."
- e2e-api.sh extended with one assertion: POST a submission with no
  criteria, verify the verdict is recorded as failed not passed.

## Related

- Planner-side: under-specified CREATE_TASK emission (45-byte directive,
  no title, no criteria) is a separate Planner-output-quality issue.
  See followup ticket if filed.
- The W11 assurance-fix (refusal-clause + watchdog-route-through-
  buildValidationPrompt, commit 35e8d94) closed the "Assurance freestyle
  when no task is pending" case. This ticket closes the symmetric
  "Assurance over-confident when criteria are missing" case.
