---
title: Session Handoff — cleanup-constitution effort complete (scaffolding phase)
status: current
verified_against: 70344e3
owner: project-owner
last_verified: 2026-06-11
---

# Session Handoff — 2026-06-11

> **Trust rule (Constitution Art. 7):** if this file's `verified_against` is older than the
> branch tip, everything below is advisory only — re-verify in code before acting.

## TL;DR

The cleanup-constitution effort's scaffolding phase is DONE and merged into `feat/remove-nomik`:
project truth system live (`docs/constitution/` + freshness loops), every doc statement proven
stale/false by the verification pipeline fixed, adversarial review passed with all approved fixes
landed. **Anti-trust probation is active** (CLAUDE.md top section): code-verify everything until
3 trustworthy cycles.

## What happened (full chain, all verified)

1. **Anti-trust verification** — 10 agents, 137 claims from F-handoff/CLAUDE.md/kanban vs live
   code: 122 confirmed, 12 stale, 2 false (`docs/audits/handoff-verification-2026-06-10.md`).
2. **Dual-path recon** — 6 agents verified the verification; zero verdict-errors; apex coverage
   gap = the uncommitted antigravity/agy backends (`docs/audits/dual-path-recon-2026-06-10.md`).
3. **Constitution + freshness system** — `docs/constitution/` (5 docs), `freshness.json` registry,
   post-commit + post-merge stalers, session-start check, refresh marker. Bootstraps on fresh
   clones (`.claude/settings.json` + hooks tracked, `$CLAUDE_PROJECT_DIR` paths,
   `scripts/install-hooks.sh` in README/AGENTS/GEMINI).
4. **Reconciliation (`d7ef63b`)** — all stale/false statements fixed; AGENTS/GEMINI → pointers;
   combined-analysis 3.5 un-struck (Planner-only); planner-prompts false gate claim fixed;
   block6 + qa-redesign tickets re-scoped; antigravity documentation ticket filed.
5. **Adversarial review** — `docs/audits/review-plan-2026-06-11.md` (disposition inline): C1
   bootstrap + C2 merge-blindness fixed (incl. my refinement: git fires post-merge, not
   post-commit — wrapper added, end-to-end tested).
6. **planner-prompts refreshed** (36 entries re-grepped, html synced) and marked fresh.

## Freshness state at handoff

`architecture-flows` = STALE (intentional — rebuild deferred to its own scoped session, method:
`.claude/architecture-flows-method.md`; it predates ACP/notebooks/act_cli). Everything else fresh.
Run `./scripts/freshness-check.sh` — don't trust this paragraph.

## OPEN / NEXT

1. **NVIDIA NIM provider** — `~/.act.json` has an inert `nvidia` block ("NOT YET IMPLEMENTED");
   no `nvidia` case exists in `internal/llm/provider/provider.go` NewProvider switch. Plan: add
   `ProviderNVIDIA` to `models/models.go` + an OpenAI-compatible case with default base
   `https://integrate.api.nvidia.com/v1` (mirror the groq/openrouter cases). Keys live in
   `~/.act.json` ONLY — never in the repo.
2. **TUI e2e matrix** (gates the alpha PR) — methodology + preflight now in CLAUDE.md
   "TUI Verification & Bug Reporting". Includes the owed **Phase-4 live verification**
   (debug.log Prepared-messages thread scoping; Observer token drop). REBUILD THE BINARY FIRST.
3. **antigravity/agy backends** — uncommitted working-tree code (9 modified + 3 untracked files),
   in flight by the project owner. Ticket: `document-antigravity-agy-backends-2026-06-11`.
   Do NOT touch those files; do NOT re-implement ACP anything.
4. **architecture-flows rebuild** — scoped session, pre-alpha.
5. Round-6 open findings #2 (brownfield injection fence) and #3 (ACP fragment for non-Planner
   roles, re-scoped into block6 ticket) remain the named alpha-hardening items.

## Where things live now (one home per fact)

Shipped log: `docs/dev/DEV_LOG.md` · tasks: `.devtool/features/` (spec format:
`docs/constitution/TASK_TRACKING.md`) · audits: `docs/audits/` · process: `docs/constitution/` ·
ideas/arc: `docs/dev/NOTES.md` / `ROADMAP.md`. Legacy `F-handoff.md` (repo root, gitignored) and
`.claude/HANDOFF.md` are frozen history with correction banners.
