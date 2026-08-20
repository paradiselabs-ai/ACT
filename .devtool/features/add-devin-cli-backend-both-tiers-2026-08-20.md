---
id: "add-devin-cli-backend-both-tiers-2026-08-20"
status: "review"
priority: "high"
assignee: null
dueDate: null
created: "2026-08-20T00:00:00.000Z"
modified: "2026-08-20T09:20:00.000Z"
completedAt: null
labels: ["orchestrator", "runner", "acp"]
order: "a0"
---
# Add Devin CLI as a backend (Tier 1 via ACP, Tier 2 via one-shot)

## Describe
Devin CLI (Windsurf's Cascade agent, post-merger) is an ACP-capable agent CLI the user has Pro access to. Adding it as an ACT backend spreads token cost off Claude. It is not wired anywhere today — `grep -rn "devin" act-agent/ server/` returns nothing backend-related (verified 2026-08-20).

Devin's two surfaces (docs.devin.ai/cli/reference/commands):
- **ACP server**: `devin acp` — stdio newline JSON-RPC, Zed spec, built to be spawned by ACP hosts. Auth order: `WINDSURF_API_KEY` env → stored `devin auth login` creds → ACP authenticate request. Optional `--model <name>` (fuzzy, e.g. `opus`).
- **One-shot**: `devin -p "<prompt>"` with `--permission-mode <normal|accept-edits|smart|dangerous|autonomous>` and `--respect-workspace-trust false` for headless/CI.

ACT's integration surfaces (all freshly code-verified; full map in scratchpad handoff `acp-architecture-handoff.md`):
- Tier 1 dispatch: single switch `act-agent/internal/app/app.go` (~:105) → `acp.NewACPAgent`; spawn argv from `internal/acp/agent.go::buildCommand` (~:596); validator `internal/app/slash.go::isValidTier1Backend` (~:347) + hardcoded help strings.
- Tier 2: `internal/runner/swarm_roles.go::IsValidBackend`/`BackendAllowedForRole`, `internal/runner/spawner.go` PATH pre-check (~:109), `runner/act-runner.mjs::runAgent` dispatch (~:243) + per-backend one-shot fn, `cli/act-cli.ts::VALID_BACKENDS` (~:595) + `backendDisallowedReason`.

The fix: wire backend string `"devin"` through both tiers, pattern-matching `gemini` for Tier 1 (native ACP, no bridge needed) and `claude-code`/`antigravity` for the Tier-2 one-shot.

## Success Criteria
1. `cd act-agent && /opt/homebrew/bin/go build -o act-agent .` clean; `cd server && npx tsc --noEmit` clean; existing Go tests pass (`go test ./internal/...`).
2. Tier 1: `~/.act.json` `agents.planner.backend: "devin"` → TUI starts, `buildCommand` spawns `devin acp` (newline JSON-RPC transport), initialize succeeds, a prompt turn streams chunks into the chat. `/backend planner devin` accepted; `/backend planner bogus` still rejected.
3. Tier 2: `act-agent swarm set developer devin` accepted and written to config; Runner spawn for that role passes a PATH pre-check for `devin`; a dispatched task executes via a single `execFileAsync('devin', [...bool flags first..., '-p', prompt], { timeout, maxBuffer, input: '' })` and returns non-empty output (one retry on empty, antigravity pattern).
4. Researcher gate resolved one of two ways, decided by verifying the actual CLI at implementation time: (a) `devin -p` has a working read-only/plan mode → wire it in the runner's restriction args, or (b) it does not → researcher×devin rejected at all three enforcement points (`swarm_roles.go::BackendAllowedForRole`, `act-cli.ts::backendDisallowedReason`, `act-runner.mjs` startup gate) with an error naming the reason, antigravity pattern. Which path was taken is stated in the completion report with the command output that proved it.
5. All manually-mirrored lists updated in the same commit: Tier 1 (app.go switch, buildCommand, isValidTier1Backend, slash.go help strings ~:78-79, :284, :290) and Tier 2 (IsValidBackend, VALID_BACKENDS, runner dispatch, spawner PATH check, `/swarm` help strings slash.go ~:67-70, act-runner.mjs help text).
6. Priming: devin uses the default user-message priming path (`planPriming` unchanged unless devin ACP is verified to expose a system-prompt channel). First-turn StopReason logged like other hosts.
7. Live smoke: one Tier-1 turn and one Tier-2 task completed against the real `devin` binary, evidence quoted (transcript lines or `~/.act/runners/*.log` excerpts). If the binary is not installed/authenticated on this machine, the ticket stays in `review` with everything else done and this criterion listed as owed.

## Constraints
- Touch ONLY the files named above plus (optionally) a devin case in `internal/acp/` comments. NO new bridge file — devin speaks ACP natively; do not clone `agy-acp.mjs`.
- Tier-1 dispatch stays in app.go's single switch (architectural principle: one dispatch point). Runner stays stateless one-shot — NO ACP in `act-runner.mjs` (server is the swarm's memory; Three-Layer Separation).
- No config schema changes: `agents.<role>.backend: "devin"` + existing `acp: {...}` override already suffice (`internal/config/config.go::ACPConfig`).
- Flag order in the one-shot: boolean flags BEFORE `-p` (agy `--print` bug precedent). Close stdin with `input: ''`.
- Headless perms: use `--permission-mode dangerous --respect-workspace-trust false` (mirror of `--dangerously-skip-permissions`); verify exact flag spelling against `devin --help` before hardcoding — docs may lag the binary.
- Do not add clean-room/settings-isolation flags unless devin has a documented equivalent; note absence in a code comment instead.
- No side-effect refactors of the acp package, no compat shims, no doc rewrites beyond: CLAUDE.md backend lists ARE allowed to gain "devin" where lists are quoted as drift-prone (keep the "re-grep, don't trust" caveats), one DEV_LOG line.
- Append coordination entries to act-coordination.json (append-only) at start/complete.

## Invariants (code-level)
- `grep -n '"devin"' act-agent/internal/app/app.go act-agent/internal/app/slash.go act-agent/internal/acp/agent.go act-agent/internal/runner/swarm_roles.go act-agent/cli/act-cli.ts act-agent/runner/act-runner.mjs` hits every file (all mirrors updated).
- `grep -rn "acp\|json-rpc\|initialize" act-agent/runner/act-runner.mjs` shows no ACP protocol code (Tier 2 still one-shot).
- `grep -c "case" <buildCommand region>`: devin case spawns `devin` with args containing `acp`; `codex`,`opencode` still return the not-implemented error (unchanged).
- In `act-runner.mjs`, the devin exec call contains `input: ''` and `-p` (or `--print`) as the LAST flag before the prompt.
- `BackendAllowedForRole` either contains a `RoleResearcher && BackendDevin` rejection OR the runner's devin fn contains a read-only restriction arg for researcher — exactly one of the two, never neither.
- No new file under `act-agent/` matching `*devin*.mjs` (no bridge).

## Implementation notes (2026-08-20, uncommitted)

**Researcher gate: path (b) — rejected.** Verified against the installed binary,
devin 3000.1.27 (0d4bf12e), not the docs:
- `devin --help`: `--permission-mode` possible values are `auto|accept-edits|smart|dangerous`
  (the ticket's "normal"/"autonomous" do not exist). `auto` only *auto-approves* read-only
  tools; it does not remove the write tools.
- No `--allowed-tools` / `--disallowed-tools` flag exists on the one-shot path.
- The only read-only agent devin ships is `devin acp --agent-type review`, which is ACP-only —
  Tier 2 never uses ACP.
- `--agent-config <FILE>` can declare tool visibility, but that is a config file ACT does not author.
So researcher×devin is rejected at all three enforcement points (`swarm_roles.go::BackendAllowedForRole`,
`act-cli.ts::backendDisallowedReason`, `act-runner.mjs` startup gate).

**Other doc-vs-binary corrections:** `--respect-workspace-trust` already defaults to `false` in
`-p`/print mode; it is passed explicitly anyway so the behavior survives a default flip.

**Live verification (criterion 7): both tiers done.**
- Tier 1 ACP: `devin acp` driven through ACT's own `NewlineTransport`/`Client` (a temporary
  `acplive`-tagged copy of `integration_live_test.go`, deleted after the run):
  `initialize` → `agent=affogato version=0.0.0-dev protocolVersion=1`; `session/new` →
  `sessionId=toothsome-knuckle`; `session/prompt` → `stopReason=end_turn elapsed=25.3s chunks=1 assembled="OK"`.
- Tier 2: real ACT server (port 8137, scratchpad data dir) + real Runner subprocess with
  `--backend devin`: `[devin invoke] path=devin attempt=1 prompt_bytes=1049` →
  `[devin result] success=true code=0 out_bytes=15` → task `completed` with
  `metadata.result = "DEVIN_TIER2_OK"`.
- CLI: `swarm set researcher devin` rejected with the reason; `swarm set developer bogus` rejected
  listing devin as valid; `swarm set developer devin` written to `~/.act.json` (config restored after).

**Fixed in passing (same file, pre-existing, all backends):** `act-cli.ts::writeAgentBackend` used
CommonJS `require('fs')`/`require('path')` inside an ESM module, so every
`act-agent swarm set <role> <backend>` write failed with "require is not defined"
(present since `ca13adf4`, 2026-04-07 — reproduced identically with `gemini`). Now uses the
existing top-level `node:fs`/`node:path` imports. Without this, criterion 3's "written to config"
could not be met.

**Found, NOT fixed (out of this ticket's constraints — no side-effect refactors of the acp package):**
`ACPAgent.Close()` calls `client.Close()` before killing the process group, and `Client.Close()`
blocks on `<-c.done` until the host closes stdout. devin's ACP server does not exit on stdin EOF
(measured: read loop still blocked in `ReadFrame` 3 min after stdin close), so TUI shutdown /
`/backend restart` on a devin-backed Tier-1 role can hang. Needs its own ticket.
