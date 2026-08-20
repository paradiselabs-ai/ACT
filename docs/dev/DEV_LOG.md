---
title: Dev Log
status: current
verified_against: bc0673e
owner: project-owner
last_verified: 2026-06-10
---

# Dev Log

One line per landing, newest first. Format ([TASK_TRACKING.md](../constitution/TASK_TRACKING.md) §5):
`YYYY-MM-DD · <ticket-id or effort> · <commit(s)> · <what/where>`

This is the canonical "what shipped" record. Entries below the seed line were back-filled on 2026-06-10 from git history and **code-verified** by the anti-trust verification pass (`docs/audits/handoff-verification-2026-06-10.md`).

---

- 2026-08-20 · acp-close-hangs-on-non-exiting-host-2026-08-20 · uncommitted (main) · ACP shutdown hang fixed: `ACPAgent.Close` SIGTERMs the process group BEFORE waiting on the client, and `Client.Close` bounds its reader-loop wait at 5s (`closeWait`); added `TestClient_CloseBoundedWhenHostNeverExits`. Live-verified `devin acp` exits on SIGTERM → shutdown immediate on the normal path.
- 2026-08-20 · add-devin-cli-backend-both-tiers-2026-08-20 · uncommitted (main) · Devin CLI wired as a backend on both tiers: Tier 1 native ACP (`devin acp`, no bridge) via `internal/acp/agent.go::buildCommand` + app.go switch + `isValidTier1Backend`; Tier 2 one-shot `devin --permission-mode dangerous --respect-workspace-trust false -p` in `act-runner.mjs` (+ `BackendDevin`, spawner PATH check, `VALID_BACKENDS`). researcher×devin REJECTED at all three enforcement points (no read-only/plan mode, no tool-restriction flag on the one-shot path — verified against devin 3000.1.27 `--help`). Live-verified: Tier-1 ACP wire smoke through ACT's own client (initialize agent=affogato protocolVersion=1 → session/new → prompt `stopReason=end_turn chunks=1`), Tier-2 task completed end-to-end on a real server (result `DEVIN_TIER2_OK`). Fixed in passing: `act-cli.ts::writeAgentBackend` used CommonJS `require` in an ESM module, so EVERY `act-agent swarm set <role> <backend>` write threw "require is not defined" (pre-existing since 2026-04-07, all backends). Known: `ACPAgent.Close()` blocks on `client.Close()` before killing the process group, and devin's ACP server does not exit on stdin EOF — see ticket.
- 2026-08-19 · agent-brief-session-save-never-fires-2026-08-13 · uncommitted (main) · Runner now writes a deterministic "Recent Work" brief (last 5 completed tasks, ≤2000 chars, no LLM) to the server after each successful completion (`act-agent/runner/act-runner.mjs`, +8 node tests, 11/11); live-verified: brief_stored events, restart restores briefs, CLI context renders, forced 500 non-fatal. Decision: Runner (not agent) owns the write.
- 2026-08-19 · pvm-evidence-pipeline (tickets pvm-outcome-events-untagged / pvm-role-attribution / pvm-agent-profile-event-type-pollution / pvm-search-no-relevance-floor, all 2026-08-13) · uncommitted (main) · outcome events project-tagged at emit + taskId→project legacy fallback in indexer; attribution joins task_completed first (real history: 24/24 attributed, was 1); profile buckets capability-only; search floor 0.28 (measured) with `?threshold=` override; jest 87/87 (+11), tsc clean; live-verified routing brief now returns real similar projects + 3-role breakdown. New tickets filed: pvm-point-id-collision-same-ms-2026-08-19, pvm-synergy-shared-project-definition-2026-08-19.
- 2026-08-13 · memory-system-audit · uncommitted (main) · live audit of PVM/memory: writes real, recall structurally broken (untagged outcomes, wrong attribution join, no relevance floor, brief save never fired); `docs/audits/memory-system-audit-2026-08-13.md` + 5 tickets; CLAUDE.md/README/ROADMAP corrected; mem0 research reports + swap analysis in docs/dev/.
- 2026-08-08 · coordination-graph-mvp-2026-08-08 · uncommitted (feat/coordination-graph) · temporal edge layer over ChronLog: `GraphStore`/`GraphIndexer` (server/src/services/), rule-derived bi-temporal edges → append-only `data/graph-edges.jsonl`, invalidate-never-delete, `/api/graph/node/:key` + `/api/graph/status`, CLI `graph node`, optional `causedBy` on events; 22 new jest tests, suite 76/76, tsc clean; replay evidence: 1158 events → 164 edges/104 nodes; live HTTP probe owed (needs server restart)
- 2026-06-12 · arch-flows-v7 + auto-refresh loop · `f729330` (+ loop commit prior) · baseline rebuild (183 components/36 flows/0 bluffed-ok, findings ×15, new ticket acp-planner-prompt-section-dead-path); auto_refresh loop live: stale-transition spawns debounced headless Opus delta-refresh (`scripts/freshness-autorefresh.sh`, UPDATE_LOOPS §2a)
- 2026-06-12 · janitorial-sweep · `6b5c13c` · untracked 205 sandbox run-outputs (gitignore path fix), root HTMLs → docs/, architecture-diagrams.md → _archive (superseded by flows artifacts), code_improvements audit → docs/audits/, tests/ swarm junk → ../act-archive/, pub/ + docs/refactor/ removed
- 2026-06-11 · nvidia-nim-provider · `e56daaa` · provider `nvidia` (OpenAI-compatible, base integrate.api.nvidia.com/v1) in models.go/provider.go/config.go; live-verified incl. `moonshotai/kimi-k2.6` completion; keys in ~/.act.json only
- 2026-06-11 · cleanup-constitution merge · `90f7230` (ff) · feat/cleanup-constitution merged into feat/remove-nomik — constitution, freshness loops, reconciled scaffolding, audits, review fixes
- 2026-06-11 · scaffolding-reconciliation · `d7ef63b` · all verified-stale/false doc statements fixed (CLAUDE.md Tier-1 backends/pitfalls, AGENTS/GEMINI→pointers, combined-analysis 3.5, planner-prompts gate claim, kanban re-scopes); claude-md/readme/combined-analysis marked fresh
- 2026-06-10 · cleanup-constitution (this effort) · (branch `feat/cleanup-constitution`) · docs constitution + freshness system: `docs/constitution/`, `scripts/git-hooks/post-commit`, `scripts/freshness-*.sh`, `docs/dev/` seeds
- 2026-06-07 · per-agent-notebooks Phase 4 · `bc0673e` · ThreadID-scoped logical notebooks: `internal/message/`, `internal/llm/agent/agent.go` (HistoryMode Full/None/Thread), migration `20260607000000_add_message_thread_id.sql`, wiring in `app.go`
- 2026-06-07 · assurance-fail-closed-empty-criteria (PARTIAL — ticket stays in-progress) · `578d280`, `f2c8d78` · `parseValidationVerdict` fails closed on empty `@success_criteria` (`orchestrator.go`, both ingestion routes); server gate / prompt clause / re-queue loop NOT shipped
- 2026-06-06 · dogfood Phase 3 · `7021488` · scope in-process event-agent input to current prompt (`scopeHistory`, superseded by Phase 4's HistoryMode)
- 2026-06-06 · 4 dogfood bug tickets → done · `1919b06` · code-enforced-agent-role-prefix, planner-prompts-render-as-human, qa-synth-queue-never-drains, observer-autoroute-loop-no-ceiling
- 2026-06-06 · dogfood Phase 2 · `9aa8417` · QA watchdog honors `synthesizedAt`; Observer no-op gate hashes stable `anomalySignature` (`orchestrator.go`)
- 2026-06-06 · dogfood Phase 1 · `e06f273` · code-stamped Tier-1 role labels (`applyRoleLabel`) + `fromHuman` gating of injected prompts (`orchestrator.go`, `message/content.go`)
- 2026-06-06 · server lockKey NUL byte · `26f2c3d` · strip stray `\0` from file-lock key delimiter (`server/src/index.ts`)
- 2026-06-06 · UNDOCUMENTED in handoffs (recovered from git 2026-06-10) · `b03ef50`, `7f439ca`, `1e33bc8` · Planner "Ready to start?" hard stop (`orchestrator.go`, `prompt/planner.go`); Observer orphaned/unservable-task detection (+`observer_anomaly_test.go`); server `/api/tasks/assigned` self-healing heartbeat (`server/src/index.ts`)
- 2026-06-06 · planner-prompt audit Round 5b · `ac241e0`, `8249a19`, `8e3d3a8`, `ffb51e4`, `2326d0b`, `3de2163` · Fixes 18–22 in `internal/llm/prompt/` + autoroute variants (`orchestrator.go`); audit log updates
- 2026-06-05/06 · planner-prompt audit Round 5a · `4f7fc3e`, `e1adc85`, `6d934d2` · Fixes 15–17 (examples-`@dependencies`, ValidationScore-always-0, fragment-missing-`prompt-section`)
