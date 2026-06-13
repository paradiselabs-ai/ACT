---
title: Targeted Handoff — open code issues (non-flow-html)
status: current
verified_against: c569034
owner: project-owner
last_verified: 2026-06-12
---

# Targeted Handoff — open code issues

Ranked most-vital first. Each line: what / where / evidence / ticket. All code-verified this session.
The architecture-flows lane-layout work is tracked separately (its own kanban ticket + in-flight) — not here.

## 1. ACP-backed Planner `prompt-section` is a dead path — HIGH
- Whitelist grants `prompt-section` and the ACP planner prompt instructs `act-tier1-planner prompt-section <name>`, but `cli/act-cli.ts` has NO such branch → falls to "Unknown command" (~act-cli.ts:1288).
- In-process Planner is fine (native `expand_prompt_section`). ACP Planners (claude-code/antigravity) silently lose all on-demand sections.
- Ticket: `acp-planner-prompt-section-dead-path-2026-06-12`.

## 2. antigravity/agy backends — uncommitted, undocumented, half-wired — HIGH
- Working tree: 9 modified + 3 untracked (`internal/acp/antigravity_cli.go`, `agy-acp.mjs`, `runner/agy-acp.mjs`). In no doc until this session's ticket.
- `gemini` dropped from the Tier-1 dispatch switch (app.go) — intentional? unverified.
- CLI accepts `antigravity` as a **swarm** (Tier-2) backend (act-cli.ts VALID_BACKENDS) but the spawner rejects it (`IsValidBackend`, swarm_roles.go) → silently settable where it never starts.
- Ticket: `document-antigravity-agy-backends-2026-06-11`. Owner's in-flight work — do not edit those files; finish + commit + doc.

## 3. Fix 23 (empty-criteria fail-open) is PARTIAL — MED/HIGH, decision pending
- Shipped: verdict-parser fail-closed (`parseValidationVerdict`, `len(CriteriaResults)>0`).
- NOT shipped: server-side zero-criteria gate (verdict endpoint trusts caller `passed`/`score` verbatim), assurance.go refusal clause, refuse+re-queue loop, the "no criteria" chat message.
- `server/scripts/e2e-api.sh:122` actively POSTs `passed:true, criteriaResults:[]` — encodes the fail-open; will break when the server gate lands (update fixture with it).
- Residual: `criteriaResults:[{}]` (len>0 junk) still passes.
- DECISION: option-2-is-enough for alpha vs build full option-1. Ticket: `assurance-fail-closed-empty-criteria-2026-05-26` (in-progress).

## 4. Brownfield prompt-injection — no fence — MED (security; before any untrusted repo)
- `agents_md.go` concatenates researcher `brief.CodebaseNotes` verbatim into AGENTS.md → every Tier 1+2 system prompt. No `<codebase_analysis>` fence, no `CREATE_TASK:`/`PROJECT_BRIEF:`/`[SYSTEM]` scrub.
- Round-6 finding #2. Needs its own ticket.

## 5. ACP non-Planner CLI fragment contradiction — MED
- `actCLICommandsACP` branches Planner-only (`common.go`, sole caller planner.go). ACP-backed Observer/Assurance/QA still get the in-process "do NOT shell out" fragment while their shim note says to shell out.
- Round-6 #3. Re-scoped into `block6-acp-cli-backend-2026-04-21`. combined-analysis 3.5 marked Planner-only.

## 6. RebindSystemPrompt skipped on AGENTS.md write failure — MED
- On AGENTS.md write failure the rebind loop is skipped (it's in the `else`); all 4 Tier-1 run stale intake-era prompts, never retried. `orchestrator.go` brief-accept path.

## 7. QA synthesis persists a marker, not a deliverable — gap (design)
- Synthesis stamps `synthesizedAt` + summary string; never persists the full assembled deliverable (server index.ts synthesis handler).

## 8. TUI e2e matrix — OWED (task 12)
- Methodology: CLAUDE.md "TUI Verification & Bug Reporting". Rebuild binary first (PATH one predates Phase 4). Includes owed Phase-4 live check (debug.log Prepared-messages thread scoping; Observer token drop). Swarm on free OpenRouter + NVIDIA models. Gates the NesTTY PR.

## Low / notes
- Socket.io dashboard handlers vestigial (exist, no client consumer; clients use REST `/api/log`).
- `TASK_TIMEOUT` 120s default can kill long swarm tasks.
- `flushToSQLite` is a no-op stub under default `jsonl` storage.
- 95% validation gate is server-side prompt-only (overlaps #3).

## Caught false-finding (anti-trust working)
- An arch-flows inventory agent claimed `act-runner.mjs extractSuccessCriteria` splits on a literal `\n`. FALSE — live is `text.split('\n')` (real newline, matches the CLI sibling). Re-grep caught it; no ticket. A `spawn_task` chip for it is a false alarm.
