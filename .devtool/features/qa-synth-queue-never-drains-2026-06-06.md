---
id: "qa-synth-queue-never-drains-2026-06-06"
status: "todo"
priority: "high"
assignee: "d34d"
dueDate: null
created: "2026-06-06T16:38:58.000Z"
modified: "2026-06-06T16:38:58.000Z"
completedAt: null
labels: ["orchestrator", "qa", "validation-pipeline", "reliability"]
order: "a1"
---
# QA synthesizer queue never drains → watchdog re-triggers QA forever

**Symptom (act-e2e, 2026-06-06):** the Planner correctly diagnosed this itself
(debug.log:531, 09:08:03):

> "QA synth is reporting its own queue isn't draining — no new validated outputs exist, but it
> keeps getting re-triggered on the same two. That's a synth-side infra issue, not a planning
> issue. I have no command to clear its queue. Nothing for me to do; project still complete."

The QA tier1 watchdog (`tier1_watchdog.fire role=qa_synthesizer pending=2`) keeps re-firing QA on
the **same already-synthesized** tasks because the synthesis queue is never marked
drained/cleared. Each re-fire auto-routes to the Planner, contributing to the starvation loop.

**Fix direction:** once a validated task has been synthesized (`synthesis_emitted kind=complete`),
it must be removed from QA's pending-synthesis set so the watchdog's "queue non-empty" condition
goes false. Audit the QA poll/watchdog source-of-truth (`qaPollLoop` / `tier1Watchdog` in
`internal/app/orchestrator.go`) — it's re-counting tasks that are already done.

**Verify:** after a task is synthesized, `qa_poll_start` / `tier1_watchdog.fire` for
`qa_synthesizer` must stop firing for that task. A completed+validated+synthesized project should
reach steady silence, not a 2-min re-trigger loop.
