---
id: "block10-planner-resume-bundle-2026-04-21"
status: "backlog"
priority: "medium"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:30:00.000Z"
completedAt: null
labels: ["planner", "resume", "block-10"]
order: "b05"
---
# Block 10 — Planner Resume Bundle

Replace today's one-line resume message with a structured bundle.

**New server endpoint**: `GET /api/projects/:name/resume-context` — full brief, open tasks, in-flight validations, last N ChronLog events, active file locks, swarm roster.

**Orchestrator changes**: on resume, fetch bundle → CTD-ordered context block (brief → tasks → coordination state). PVM seeded retrieval (top-K coordination patterns + skill-profile snapshots). Token budget: bundle ≤5K, role prompt cached, working context ~15K headroom. Backend-aware (Block 6): API backends re-emit on resume; ACP backends inject once via session init.

Depends on Block 6, Block 7. See BUILD_ORDER.md Block 10.
