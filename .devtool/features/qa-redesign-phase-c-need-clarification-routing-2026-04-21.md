---
id: "qa-redesign-phase-c-need-clarification-routing-2026-04-21"
status: "todo"
priority: "medium"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:55:17.097Z"
completedAt: null
labels: ["QA", "routing"]
order: "a02"
---
# QA Redesign — Phase C: NEED_CLARIFICATION routing

`NEED_CLARIFICATION: @agent` is currently parsed + ChronLogged but never routed — the named agent never receives the question. Wire it to `/api/messages` with mention so the clarification loop actually closes.

Depends on Phases A + B.