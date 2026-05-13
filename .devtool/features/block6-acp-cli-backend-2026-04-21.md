---
id: "block6-acp-cli-backend-2026-04-21"
status: "todo"
priority: "critical"
assignee: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-04-21T17:30:00.000Z"
completedAt: null
labels: ["v1-gate", "backend", "architecture", "block-6"]
order: "b00"
---
# Block 6 — ACP CLI Backend for Tier 1

Add `ACPBackend` alongside `OpenRouterBackend` so Tier 1 (Planner/Observer/Assurance/QA) can be backed by ACP-compatible CLIs — Gemini CLI (`gemini --acp`), Claude Code via `@zed-industries/claude-agent-acp`. Users bring subscription; no per-token billing; no free-tier churn.

**New interface**: `AgentBackend` — uniform contract; orchestrator never knows which backend.

**Two invariants**:
1. `rolePrompt` injected **once per session** (ACP) vs **every turn** (API). Never swap.
2. ACP = long-lived subprocess; API = stateless. Backend owns lifecycle.

**Context injection asymmetry**: `ACPBackend.Run()` skips per-turn system-prompt injection; `OpenRouterBackend.Run()` keeps it.

**Files to create**:
- `act-agent/internal/llm/backend.go` — `AgentBackend` interface
- `act-agent/internal/llm/acp/backend.go` — JSON-RPC 2.0 over stdio: spawn → `initialize` → `session/new` → streaming `session/update` on `session/prompt`

**Unblocks**: v1 release as "Claude Code multi-agent harness." See BUILD_ORDER.md Block 6 + FUTURE_VISION.md "ACP CLI Backend for Tier 1 Agents".
