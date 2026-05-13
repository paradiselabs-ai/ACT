---
id: "block14-claude-code-swarm-hooks-wizard-2026-04-21"
status: "backlog"
priority: "low"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:30:00.000Z"
completedAt: null
labels: ["hooks", "wizard", "TUI", "block-14"]
order: "b09"
---
# Block 14 — Claude Code Swarm Hooks Wizard

Replace hand-editing `settings.json` with guided wizard (`/swarm hooks <role>` in TUI).

**Steps**:
1. Pick hook events (`PreToolUse`, `PostToolUse`, `Stop`...) with plain-English descriptions
2. Build matcher interactively ("only Bash", "only writes to src/", "any git op") + live preview
3. Pick action template — gates, context injection, perf patterns, coordination patterns
4. Save scope: per-role or swarm-wide
5. Test mode: dry-run against sample tool call

**Output**: `.claude/hooks/swarm/<role>/` scripts + `settings.json` injected at Runner spawn.

See BUILD_ORDER.md Block 14.
