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
## Live evidence (2026-07-07, finance run)

Exactly this gap, observed end-to-end. dev-1's completion note for "Build
local report server" had no file paths; QA refused to synthesize and emitted
`NEED_CLARIFICATION: @dev-1 ...` twice (04:21 + 09:38 after resume). The
question sat in the inbox because workers only read messages when they pick
up a task — dev-1 was idle/dead between sessions. It was finally DELIVERED
when the budget tasks spawned dev-1 again (runner log: "[inbox] 1 pending
message(s) — including in task context"), but the ANSWER never routed back
to QA's pending synthesis: task ed645d3a is still `validated` with no
synthesizedAt and will be re-polled by every future finance session.

Fix shape confirmed by this run: (a) liveness/timeout on the addressee —
if the agent is offline or the question ages out, fall back to the Planner
autoroute (decision: retry vs accept vs answer); (b) an answer path that
completes the pending synthesis instead of relying on the agent's next
unrelated task; (c) worker completion notes must carry file paths (prompt
discipline — the root trigger).
