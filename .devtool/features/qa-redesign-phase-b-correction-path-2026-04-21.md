---
id: "qa-redesign-phase-b-correction-path-2026-04-21"
status: "todo"
priority: "high"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:55:11.098Z"
completedAt: null
labels: ["QA", "architecture"]
order: "a01"
---
# QA Redesign — Phase B: Correction path (Option A vs B)

Decide and implement QA's correction mechanism when integration drift is detected:

- **Option A**: QA edits codebase directly via tightly-constrained glue-only tool (routing/imports/module decls only). Requires new narrow edit tool.
- **Option B**: QA dispatches code-fix task to qa_engineer swarm agent (or new `integrator` role). QA stays read-only.

Option B is lower-risk (zero blast radius, reuses existing task infra). Option A has faster iteration loop. Pick one after Phase A lands and we have real signal.

Depends on Phase A.