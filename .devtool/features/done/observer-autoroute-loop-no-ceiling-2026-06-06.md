---
id: "observer-autoroute-loop-no-ceiling-2026-06-06"
status: "done"
priority: "medium"
assignee: "d34d"
dueDate: null
created: "2026-06-06T16:38:58.000Z"
modified: "2026-06-06T22:17:31.000Z"
completedAt: "2026-06-06T22:17:31.000Z"
labels: ["orchestrator", "observer", "reliability"]
order: "a2"
---
# Observer re-flags unfixable tasks every cycle → autoroute loop with no escalation ceiling

**Symptom (act-e2e, 2026-06-06):** after the project completed, two tasks stayed stuck — one
`failed` (`1d17c872`) and one blocked behind it (`8e155ba9`). The Observer re-detected the same
anomalies every 2-min cycle (`anomalies=2`) and auto-routed to the Planner each time (09:56, 09:58,
10:00… `autoroute_from_observer`), but the Planner had **no tool to resolve them** (no retry/abandon
in hand for a failed task it didn't own). The loop starved the Planner of real work and helped
trigger the fabricated-`Human:` behavior.

**The gap:** the Observer keeps surfacing the *same unresolved* anomaly forever with no ceiling and
no state change. There's no "already reported, no new info, and the responsible party has no
action" terminal state.

**Fix direction (pick during planning):**
- Don't re-autoroute an anomaly whose signature is unchanged AND was already surfaced — escalate
  once, then go quiet until it changes or a human acts (similar to the existing noop_gate hash, but
  applied to the *autoroute*, not just the Observer LLM call).
- For a `failed`/blocked task the Planner can't act on, surface it to the **human** as a decision
  (retry / abandon) instead of re-pinging the Planner indefinitely.
- Consider a per-anomaly escalation ceiling within the sliding window.

**Verify:** a persistently-stuck task should produce one escalation, not an endless 2-min
Planner-trigger loop.
