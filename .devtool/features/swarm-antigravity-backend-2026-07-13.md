---
id: "swarm-antigravity-backend-2026-07-13"
status: "review"
priority: "medium"
assignee: null
dueDate: null
created: "2026-07-13T00:00:00.000Z"
modified: "2026-07-13T00:00:00.000Z"
completedAt: null
labels: ["runner", "swarm", "backend"]
order: "a0"
---
# Antigravity (agy) as a Tier 2 swarm backend — direct one-shot, NO ACP bridge

## Describe
The swarm has no antigravity backend. Verified in code 2026-07-13:
`internal/runner/swarm_roles.go::IsValidBackend` accepts only `act-agent`,
`claude-code`, `gemini`; the `BackendAntigravity` constant exists but is commented
"Tier 1 ACP only". The agy CLI is memoryless one-shot (`agy --print <prompt>`), which
is exactly the shape every swarm backend already has — the Runner rebuilds full context
(identity, brief, inbox, gaps, parallel awareness) into each task prompt and no swarm
backend keeps memory across tasks.

Therefore the fix is a direct one-shot spawn branch in `runner/act-runner.mjs`
mirroring the gemini pattern (`runAgentGemini`). Do NOT route swarm agy through the
ACP bridge (`agy-acp.mjs`): the bridge's only job is faking session continuity for
Tier 1's long conversation. For one-shot task execution it would add a persistent Node
process, protocol framing, and replay of prior-task history — pure overhead, zero
benefit, and token bloat from irrelevant replayed turns.

## Success Criteria
- `runner/act-runner.mjs` has an agy branch: one-shot spawn with print flag, closed
  stdin (`input: ''`), TASK_TIMEOUT, 10MB maxBuffer, and the same one-shot-retry
  contract as claude-code/gemini (retry once only when process died with no output).
- `IsValidBackend` accepts the antigravity backend; `/swarm <role> antigravity` (TUI)
  and `act-agent swarm set <role> antigravity` (CLI) both accept it end-to-end.
- Researcher least-privilege: verify agy's actual flag surface first. If agy has a
  read-only/plan mode, wire it for `researcher` (mirroring gemini's
  `--approval-mode plan`). If it has none, `researcher`+antigravity is REJECTED at
  config-set time with a clear error — no silent full-privilege researcher.
- `cd act-agent && /opt/homebrew/bin/go build -o act-agent .` clean; existing runner
  tests green.
- Live e2e evidence captured per CLAUDE.md "TUI Verification & Bug Reporting":
  one task dispatched to an antigravity-backed role, `~/.act/runners/<role>.log`
  excerpt quoted in this ticket before moving to done.

## Constraints
- Touch only: `runner/act-runner.mjs`, `internal/runner/swarm_roles.go`, the config +
  slash/CLI validation plumbing for backend names. Leave `internal/acp/` and
  `agy-acp.mjs` untouched — the bridge stays Tier-1-only.
- No shared-abstraction refactor of the three existing backend functions ("while I'm
  here" ban) — add the fourth branch in the same flat style.
- Server untouched (Three-Layer Separation: Runner is a thin spawner; the server never
  knows which backend ran a task).
- Swarm stays stateless one-shot per task — no session/memory layer for agy swarm.

## Invariants (code-level)
- grep "agy-acp" in `runner/act-runner.mjs` → 0 hits (bridge never spawned from the
  swarm task-execution path).
- `BackendAntigravity` constant's "Tier 1 ACP only" comment updated to reflect Tier 2
  support in the same commit.
- Researcher restriction exists as code (flag args or config-time rejection), not as
  prompt text only.
- No new import of `internal/acp` outside `internal/app/app.go`.
