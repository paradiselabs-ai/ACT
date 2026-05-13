---
id: "block15-planner-decision-feedback-loop-2026-04-21"
status: "backlog"
priority: "low"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:30:00.000Z"
completedAt: null
labels: ["planner", "learning", "block-15"]
order: "b10"
---
# Block 15 — Planner Decision Feedback Loop

Three orthogonal signals:
1. **Outcome regression** — same failure class after Planner intervention → `planner_intervention_no_resolution` tag
2. **Observer sub-counter** — "retries after intervention" count → `planner_call_ineffective` on threshold
3. **`/correct` slash command** — human override, annotates last Planner turn, writes `decision_correction` to ChronLog

**Four archetypes** tagged post-hoc: reassign-when-rewrite-needed, rewrite-when-reassign-needed, split-when-not-needed, no-split-when-needed.

**Three feedback tiers**:
- Runtime — tagged mis-calls re-enter Planner's next turn as `@recent_decision_debt`
- Cross-session — PVM indexed; `act pvm search --type decision --gap <sig>` returns historical resolution rates
- Offline — ChronLog exports bad-call corpus; prompt revisions tested against archetype resolution rates; winning revision ships as version bump

Depends on Block 13. See BUILD_ORDER.md Block 15 + FUTURE_VISION.md "Planner Decision Feedback Loop".
