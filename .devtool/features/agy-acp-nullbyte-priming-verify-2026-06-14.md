---
id: "agy-acp-nullbyte-priming-verify-2026-06-14"
status: "todo"
priority: "high"
assignee: null
dueDate: null
created: "2026-06-14T05:00:00.000Z"
modified: "2026-06-14T05:00:00.000Z"
completedAt: null
labels: ["acp", "orchestrator", "antigravity"]
order: "a1"
---
# Verify + commit the agy/antigravity null-byte priming fix (shim strips ACT_INTERNAL marker)

## Spec
On the 2026-06-11 mosh ACP test, the Tier-1 Planner priming **never reached** the antigravity
(`agy`) backend. ACT prepends the internal marker `\x00ACT_INTERNAL\x00` (`InternalPromptMarker`,
`internal/app/orchestrator.go:682`) to every hidden Tier-1 prompt and sends the role identity to
the backend. The old antigravity path passed that text as a process **command-line argument**, and
POSIX argv cannot contain null bytes — so the spawn was rejected and priming silently failed. The
antigravity agent then ran as a plain agent with its own native sub-agents, bypassing ACT entirely.

A fix now exists in the **uncommitted** working tree:
- `act-agent/runner/agy-acp.mjs` (untracked) — an ACP-server shim wrapping `agy`. On a prompt
  starting with the marker (line ~92) it **strips the marker**, stores the remainder as the
  session's system prompt, and returns `end_turn` **without** spawning `agy`. Real task prompts
  are then spawned with the identity re-injected as plain framed text (`buildContextPrompt`,
  line ~186), so no null bytes ever hit argv, and `agy` no longer mistakes the role prompt for a
  task to build.
- `act-agent/internal/acp/antigravity_cli.go` (untracked) — spawns `node agy-acp.mjs`; priming is
  delivered over the ACP wire channel (`client.Prompt`, `internal/acp/agent.go:359`), not argv.

This ticket: **commit the fix and prove it end-to-end.** Code reads correct, but it is untracked
and there is NO transcript of a successful ACT-primed antigravity planner run (the Jun-14 antigravity
sessions show either zero priming markers or a debugging session, not a clean primed planner turn).

## Success Criteria
- Live antigravity run produces a planner trajectory whose system prompt = the ACT Planner identity
  (no `\x00` bytes anywhere in what `agy` receives), confirmed from the antigravity store decode.
- `.act/debug.log` shows `acp_priming_completed role=planner` (NOT `acp_priming_failed`) for backend=antigravity.
- The planner emits at least one valid `PROJECT_BRIEF`/`CREATE_TASK` marker that the orchestrator parses.
- `agy-acp.mjs` + `antigravity_cli.go` (and the `app.go`/`slash.go`/`config.go` glue) are committed, not untracked.
- No regression on the claude-code backend (its priming already works over the channel).

## Constraints
- Do NOT change `InternalPromptMarker` or how claude-code receives priming — the marker is correct for
  the channel transport; only the argv-bound backend needed the strip. Fix stays at the shim boundary.
- No new dependency; the shim is plain Node stdio JSON-RPC.
- Coordinate with `[[document-antigravity-agy-backends-2026-06-11]]` — that ticket lands the backends
  broadly; this one is the priming-correctness slice. Don't double-commit the same files.

## Invariants (code-level)
- `internal/app/app.go:278` still composes priming as `InternalPromptMarker + doNotRespondHeader + base + renderShimNote(role)`.
- `runner/agy-acp.mjs` marker check uses the exact literal `'\x00ACT_INTERNAL\x00'` (must equal the Go constant).
- Marker strip happens BEFORE any `spawn(AGY_CMD, [..., fullPrompt])`.

## Repro / Evidence
- Failure: `/Users/user/Documents/Developer/dev/AI/mosh/.act/debug.log:173` —
  `acp_priming_failed … failed to spawn agy: The argument 'args[1]' must be a string without null bytes.
  Received '\x00ACT_INTERNAL\x00[ACT priming …'`.
- Antigravity ran un-primed: trajectory `339950ed-…` in `~/.gemini/antigravity-cli/conversations/`
  has 0 ACT-priming / 0 CREATE_TASK markers in the raw db.
- Full writeup + decoded transcripts (gitignored): `docs/antigravity acp/FINDINGS.md`.
