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
