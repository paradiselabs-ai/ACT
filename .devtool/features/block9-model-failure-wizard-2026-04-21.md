---
id: "block9-model-failure-wizard-2026-04-21"
status: "backlog"
priority: "medium"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:30:00.000Z"
completedAt: null
labels: ["models", "TUI", "block-9"]
order: "b04"
---
# Block 9 — Model Failure Wizard + Dynamic Registry

**Note**: partially superseded by Model Registry Simplification. If that lands first, this narrows to just the failure-recovery wizard.

9a. Dynamic model registry — fetch `GET /api/v1/models` + equivalents on startup, cache 24h TTL at `~/.act/cache/models.json`. TUI picker (`ctrl+o`) shows live catalog.

9b. Failure wizard — catch model-not-found/404/provider-unavailable in `AgentBackend`. TUI modal shows failed role + live replacement candidates. User picks → write to `~/.act.json` → hot-swap → re-trigger turn. No TUI restart.

Depends on Block 6. See BUILD_ORDER.md Block 9.
