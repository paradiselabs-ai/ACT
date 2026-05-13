---
id: "ki-03-assurance-verdict-non-determinism-2026-04-21"
status: "done"
priority: "medium"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:52:36.096Z"
completedAt: "2026-04-21T17:52:36.096Z"
labels: ["bug", "assurance", "validation"]
order: "a03"
---
# KI-03: Assurance verdict non-determinism

Same task, same Assurance model produces opposite verdicts (pass 100 → fail 0) on consecutive runs. Three-part fix:

1. Log full Assurance response body (not 300-char truncation) on every verdict → visibility into *why* the model flipped.
2. Set temperature=0 on Assurance model config where supported.
3. Second-opinion pattern — if two consecutive runs disagree by &gt;50 score delta, escalate to Planner with both verdicts.

Evidence: task `e56d6740` (11:35) flipped on retry. 3/3 tasks in that session failed post-cancel retries. See KNOWN_ISSUES.md KI-03.