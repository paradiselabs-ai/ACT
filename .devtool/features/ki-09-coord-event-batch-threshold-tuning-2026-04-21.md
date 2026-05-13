---
id: "ki-09-coord-event-batch-threshold-tuning-2026-04-21"
status: "backlog"
priority: "low"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:30:00.000Z"
completedAt: null
labels: ["tuning", "coordination"]
order: "a07"
---
# KI-09: Coordination event batching threshold tuning

`coordEventFloodThreshold = 8` per poll cycle. With 5 swarm agents the per-tick count will grow; may need drop to 5 or make dynamic (compact if `render_stats.slow_pct > 15%`). Tunable, not a bug — observe across several sessions before adjusting.

See KNOWN_ISSUES.md KI-09.
