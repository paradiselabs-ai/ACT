---
id: "tui-status-legend-shows-unused-model-2026-08-08"
status: "todo"
priority: "medium"
assignee: null
dueDate: null
created: "2026-08-08T15:40:00.000Z"
completedAt: null
labels: ["TUI", "bug"]
order: "a0"
---
# Status legend shows the in-process model for roles running on external backends

## Describe
`internal/tui/components/core/status.go` (~L392-405) renders each Tier-1 role's
`cfg.Model` badge with no awareness of `cfg.Backend`. When a role runs on an
external backend (claude-code / gemini / antigravity via ACP), the model field is
UNUSED — the external CLI picks its own model — so the legend shows either a
model that is not actually running (misleading) or `:-` when the field is empty.
Observed live 2026-08-08 (link-dock run, planner+observer on claude-code,
assurance+qa on gemini).

## Success Criteria
- Legend shows the backend name for externally-backed roles (e.g.
  `Planner:claude-code`), model badge only for in-process roles.
- `:-` only when a role has neither model nor backend configured.

## Constraints
- TUI domain (Kareem). Config truth: `agents.<role>.backend` in ~/.act.json —
  non-empty backend ⇒ model field is not in use for that role.
