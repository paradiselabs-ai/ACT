---
title: Unwired-code sweep — "built then never wired"
status: current
verified_against: d59785a
owner: project-owner
analyzed: 2026-06-14
method: |
  Triggered by an observation: internal/act/client.go was written recently yet a method on it
  appeared uncalled. "Dead code" two weeks old is usually built-then-never-wired, not abandoned —
  so the whole class was swept. Every claim re-grepped against the live tree at d59785a (anti-trust;
  one grep-loop gave an all-false-positive result and was discarded — never trusted a result that
  contradicted a known fact). Each finding bucketed, then reconciled against the kanban board BEFORE
  filing anything (Constitution Art. 2 — no duplicate fixes).
---

# Unwired-code sweep — "built then never wired"

## Thesis

Code that is written but never connected looks "done" in git history while doing nothing — or
silently failing — at runtime. This sweep finds that class across the codebase, labels each item,
and tracks only the genuinely-new and actionable ones. The headline result is reassuring: **most
suspected items are already on the board or are deliberate; only two are genuinely new, both LOW.**

## Buckets

- **never-built** — documented/designed, zero implementing code.
- **built-unwired** — code exists, nothing calls it.
- **half-implemented** — interface/config exists, implementation is a stub; may silently no-op.
- **intentional-deferred** — disconnected on purpose, documented.

## Findings (all re-grepped at d59785a)

| # | Item | Where | Bucket | Sev | Disposition |
|---|------|-------|--------|-----|-------------|
| 1 | `internal/act/client.go` — 9 of 25 methods unwired in Go | client.go (see below) | built-unwired | LOW | **NEW ticket** `client-go-unwired-methods-2026-06-14` |
| 2 | `flushToSQLite` empty stub; `storageType:'sqlite'` silently drops data | ChronologicalLog.ts:242 / dispatch :188-193 | half-implemented | LOW–MED (dormant) | **NEW ticket** `chronlog-sqlite-mode-failfast-2026-06-14` |
| 3 | Context compaction (the 4-tier strategy) | agent.go:34/57/70 (summarize primitive only) | half-implemented | — | already tracked (`pi-compaction-first-class` + 8-ticket `compaction-*` cluster) |
| 4 | Pre/Post tool-use hooks | none in `internal/llm/` | never-built | — | already tracked (`pi-extension-hook-system`, `block14-...hooks-wizard`, `pi-file-mutation-queue`) |
| 5 | Deferred tool discovery (Claude-Code-style `ToolSearch`) | docs/ARCHITECTURE_PATTERNS.md §2 only | never-built | — (intentional, Low) | no ticket — diagram `gap-found` |
| 6 | autoDream memory consolidation | docs/ARCHITECTURE_PATTERNS.md §5 only | never-built | — (intentional, Low) | no ticket |
| 7 | Static/dynamic prompt-cache split | prompt.go:70-119 | **BUILT — not a finding** | — | verified wired (Audit Fix 14; keeps the Anthropic ephemeral-cache breakpoint stable, anthropic.go:190) |
| 8 | Persistent vector store (Qdrant) | QdrantVectorStore.ts (build-excluded) | intentional-deferred | — | no ticket — forward plan is `block13`; cold-start re-embeds the whole log each boot |

### Detail — finding #1 (the 9 unwired client methods)

`internal/act/client.go` is the Go native HTTP client for the coordination server. Its own header
says it "replaces the previous shell-out implementation." It mirrors the *full* server API, but 9
methods have **zero non-test Go callers** — only their definition lines exist:

- `ReportProgress` (:136), `ReportComplete` (:151), `SubmitForValidation` (:212) — the **runner**
  does these via direct HTTP (act-runner.mjs:173/183/194).
- `ClaimFiles` (:166), `ReleaseFiles` (:182) — only the **TS CLI** (`cmdFilesClaim`/`cmdFilesRelease`)
  touches locking; Go never locks (consistent with file-locking being prompt-only per HANDOFF).
- `RetryTask` (:403), `AbandonTask` (:424) — the Planner affordance runs through the `act_cli` tool →
  TS CLI, not Go.
- `PVMSearch` (:230) — Planner PVM search runs through the `act_cli` tool → CLI → `/api/pvm/search`.
  This native method (the very thing client.go exists to replace shell-outs with) was never swapped in.
- `Status` (:225) — no caller (the `/status` slash command reads `runnerSpawner.Status()`, a different method).

**Honest read:** this is a *complete client surface built ahead of need*, not a functional bug. The
orchestrator legitimately does not perform task execution (the runner does) or file locking (the
swarm does). It is dead code — a wire-or-delete cleanup decision, not broken behavior. Wired subset
(16, leave intact): Register, GetContext, CreateProject, CreateTask, SetTaskDependencies, SubmitVerdict,
SubmitSynthesis, SendMessage, IsAvailable, ListTasks, GetPendingValidation, GetValidatedTasks,
ListAgents, GetFileLocks, GetLog, GetProject.

### Detail — finding #2 (the SQLite stub)

`ChronologicalLog` accepts `storageType: 'jsonl' | 'sqlite' | 'both'` (ChronologicalLog.ts:28).
`flushToSQLite` (:242) is an empty TODO. The flush path (:188-193) routes `'sqlite'`/`'both'` to it.
So `storageType:'sqlite'` makes every flush a silent no-op = total event-log data loss; `'both'` is
safe only because JSONL is still written. Default is `'jsonl'` and no caller sets otherwise, so it is
**dormant**. Distinct from `block13` (which *builds* the real SQLite/LanceDB stack): this is a
fail-fast safety guard so the config can't silently destroy data until block13 lands.

## Disposition summary

- **New tickets:** 2 (both LOW) — `client-go-unwired-methods-2026-06-14`, `chronlog-sqlite-mode-failfast-2026-06-14`.
- **Already tracked:** compaction (9 tickets), tool-hooks (3), persistent store (block13).
- **Intentional / no action:** deferred tools, autoDream, Qdrant build-exclusion.
- **Not a finding (verified built):** static/dynamic prompt-cache split.

## Takeaway

The "built-then-unwired" pattern is real but mostly benign — dead-code cleanup or already on the
board. The only latent *data* risk is the SQLite stub, and it is dormant behind a non-default config.
The board is more complete than the codebase "feels": the overwhelm is largely already-tracked work,
not undiscovered bugs.
