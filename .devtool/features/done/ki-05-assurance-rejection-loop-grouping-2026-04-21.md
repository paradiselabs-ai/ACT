---
id: "ki-05-assurance-rejection-loop-grouping-2026-04-21"
status: "done"
priority: "medium"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:52:26.107Z"
completedAt: "2026-04-21T17:52:26.107Z"
labels: ["bug", "assurance", "planner"]
order: "a04"
---
# KI-05: Assurance rejection loop

Assurance rejecting → Planner autoroute → Planner emits new CREATE_TASK → Assurance rejects → repeat until `autoTurnCap=5` drop. Cap masks symptom but wastes LLM quota.

Two-part fix:

1. **Rejection grouping**: if Assurance rejects 2+ tasks within 60s, collapse into one autoroute with all contexts.
2. **Planner-to-Assurance handoff markers**: give Planner a `RETRY_TASK: <id>` directive (distinct from CREATE_TASK) and an explicit "surface to human" marker, so it can respond to rejections without recursively looping.

See KNOWN_ISSUES.md KI-05.