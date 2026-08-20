---
title: Roadmap
status: current
verified_against: bc0673e
owner: project-owner
last_verified: 2026-06-10
---

# Roadmap

Ordered by importance. Task-shaped items live on the kanban (`.devtool/features/`) — this doc holds the arc, not the tickets. Unordered ideas go to [NOTES.md](NOTES.md).

---

## Now (alpha path)

1. **Cleanup + constitution effort** (this branch) — scaffolding truth restored, docs system + freshness loops installed. Then merge → `feat/remove-nomik`.
2. **TUI e2e matrix** on merged `feat/remove-nomik` — varying project types/complexity, greenfield + brownfield onboarding, Tier-1 streams, swarm on free models (OpenRouter + NVIDIA NIM). Includes the pending **Phase-4 live verification** (Planner prepared-messages thread scoping, Observer token drop).
3. **Alpha-blockers from Round 6** (kanban-tracked): brownfield prompt-injection fence (#2 — before any untrusted-repo onboarding); Fix 22 half-close for non-Planner ACP roles (#3 — cheap); Fix 23 completion decision (option-1 re-queue loop vs option-2-is-enough).
4. **Alpha tag**: `feat/remove-nomik` → PR → `NesTTY` when the e2e matrix is alpha-worthy; `NesTTY` → `main` with `v0.1.0-alpha.1` per branch protocol.

## Next

- NVIDIA NIM provider support in act-agent (config block exists in `~/.act.json`, marked NOT YET IMPLEMENTED; needs Go provider wiring)
- Round-7 prompt re-audit (prompt produced in the 2026-06-07 session; run against post-cleanup state)
- SPIL Stage 1 (parser/AST/evaluator + proof-criteria gate — kanban `spil-stage1-*` tickets)
- Remaining Round-6 MED findings (#4–#10)

## Later

- SPIL evolution to agentic language (kanban `spil-evolve-to-agentic-language`)
- FLUX State (needs causal PVM edges / Coordination KG)
- PVM analytics runtime-quality — **validated 2026-08-13 and it FAILED** (no longer "unverified"). Analytics are computed, not faked, but their inputs are broken: task lifecycle events carry no project tag, and outcome→worker attribution joins the wrong event. A live 6-task/3-role project produced a routing brief of "developer: 100% pass over 3 tasks" with zero similar projects. Fix tickets: `pvm-outcome-events-untagged-by-project-2026-08-13` (critical), `pvm-role-attribution-joins-wrong-event-2026-08-13`. Evidence: [docs/audits/memory-system-audit-2026-08-13.md](../audits/memory-system-audit-2026-08-13.md). `SelfImprovementEngine` internals not separately re-verified — check before relying on it.
- Architecture patterns backlog (`docs/ARCHITECTURE_PATTERNS.md`: compaction, deferred tool discovery, hooks, caching split, autoDream)
