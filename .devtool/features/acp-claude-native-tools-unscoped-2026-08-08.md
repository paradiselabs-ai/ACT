---
id: "acp-claude-native-tools-unscoped-2026-08-08"
status: "todo"
priority: "high"
assignee: null
dueDate: null
created: "2026-08-08T09:30:00.000Z"
modified: "2026-08-08T09:30:00.000Z"
completedAt: null
labels: ["acp", "tier1", "security"]
order: "a1"
---
# Claude-backed Tier 1 roles get Claude Code's FULL native toolset (KI-02 gap)

## Describe
KI-02 says no Tier-1 role gets raw bash, and in-process roles enforce per-role
tool subsets. But the claude ACP bridge defaults `tools` to the full claude_code
preset (verified in bridge 0.37 source: `tools = userProvidedOptions?.tools ??
{type:"preset", preset:"claude_code"}`), so an ACP claude Planner/Observer has
native Write/Edit/Bash — the act-tier1-shim only governs the act-agent CLI, not
claude's own tools. Evidence of the risk class (lido run, 2026-08-08): the
Planner offered to persist rules to memory on its own initiative. A Planner that
can edit files or run bash violates the role contract (Planner decides, never
executes).

## Success Criteria
- `_meta.claudeCode.options.tools` set per role, mirroring Tier1ToolsForRole
  intent: Planner/Observer → read-only-or-none native toolset; Assurance/QA →
  read-only (view/grep equivalents). Exact preset/array syntax verified against
  the bridge source before wiring.
- Wire test asserting the tools restriction serializes on session/new per role.
- Live e2e: claude-Planner cannot Write a file (tool absent or rejected),
  evidence quoted here before done.

## Constraints
- internal/acp only; same _meta.claudeCode.options channel as settingSources.
- Verify the bridge's expected shape for tools (array of names vs preset object)
  from source, not guessed.

## Invariants
- No claude-backed Tier-1 role has Bash/Write/Edit natively.
- Tier-2 claude-code swarm agents are NOT restricted by this change (they must
  write code).
