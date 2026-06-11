---
id: "block6-acp-cli-backend-2026-04-21"
status: "in-progress"
priority: "critical"
assignee: null
epic: null
dueDate: null
created: "2026-04-21T17:30:00.000Z"
modified: "2026-06-11T17:35:00.000Z"
completedAt: null
labels: ["v1-gate", "backend", "architecture", "block-6"]
order: "b00"
---
# Block 6 — ACP CLI Backend for Tier 1

Tier 1 (Planner/Observer/Assurance/QA) backed by ACP-compatible CLIs. Users bring subscription; no per-token billing; no free-tier churn.

> **⚠ BODY RE-SCOPED 2026-06-11 (anti-trust verification + dual-path recon, CV3).**
> The original body said "create `act-agent/internal/llm/backend.go` + `AgentBackend` interface" —
> **that design was never built and must NOT be built now.** The implementation SHIPPED at
> **`act-agent/internal/acp/`** (agent.go, client.go, session.go, transport.go, types.go,
> claude_code.go + tests), exposed as `acp.NewACPAgent` returning the existing `agent.Service`
> interface — no new interface needed. Tier-1 dispatch is the backend switch in
> `internal/app/app.go` (re-grep it for the live member set); `/backend <role|all> <backend>` is the
> TUI command (`slash.go`); `act-tier1-shim` (cmd/act-tier1-shim) enforces the per-role CLI allowlist.
> Executing the old body as written would create a second, conflicting ACP layer — the exact
> dual-implementation failure CLAUDE.md's banner warns about.

## Spec (remaining work)
The shipped layer covers the claude-code host; `codex`/`opencode` return explicit unimplemented
errors; `antigravity`/`agy` are in flight (see `document-antigravity-agy-backends-2026-06-11`).
Remaining scope for this ticket:

1. **ACP-priming parity for non-Planner roles** — `actCLICommandsACP` branches only for the Planner
   (`common.go`; sole caller `planner.go`). ACP-backed Observer/Assurance/QA still receive the
   in-process "do NOT shell out" fragment while their shim note says "use the shim via Bash"
   (Round-6 finding #3 / combined-analysis 3.5 — fix is Planner-only today).
2. Decide + implement the unimplemented hosts (`codex`, `opencode`) or remove them from the switch.

## Success Criteria
- `actCLICommandsACP` (or successor) branches correctly for all four Tier-1 roles; the
  contradiction between the CLI fragment and `renderShimNote` is gone for ACP-backed
  Observer/Assurance/QA.
- `TestPlannerPromptBranchesOnProvider` extended to observer/assurance/qa (or equivalent tests).
- combined-analysis 3.5 can be marked fully FIXED (cross-role) only after this lands.

## Constraints
- Extend the existing `internal/acp/` layer only. No new backend interface, no `internal/llm/backend.go`.
- No orchestrator dispatch redesign — the single switch in `app.go` stays the only dispatch point.

## Invariants (code-level)
- `acp.NewACPAgent` remains the only external-backend constructor; no `act-agent/internal/llm/backend.go` exists and no `AgentBackend` *type* is introduced (`grep -rnE '\bAgentBackend\b\s*(struct|interface)' act-agent/internal/` stays empty — the plain-substring grep false-hits the live `WriteAgentBackend` config writer).
- Original invariants still hold: rolePrompt injected once per ACP session (priming injector) vs every turn in-process; ACP = long-lived subprocess whose lifecycle the acp package owns.

**Unblocks**: v1 release as "Claude Code multi-agent harness." See BUILD_ORDER.md Block 6 + FUTURE_VISION.md "ACP CLI Backend for Tier 1 Agents".
