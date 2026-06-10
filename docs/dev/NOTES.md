---
title: Developer Notes & Ideas
status: current
verified_against: bc0673e
owner: project-owner
last_verified: 2026-06-10
---

# Developer Notes & Ideas

Unordered parking lot. Promote to [ROADMAP.md](ROADMAP.md) when sequenced, or to a kanban ticket when task-shaped. Personal long-form thinking can live in `docs/Vault/` (gitignored) — but anything load-bearing must be promoted here or further.

---

- `e2e-api.sh:122` POSTs a `passed:true` verdict with `criteriaResults:[]` as a happy-path fixture — when the Fix-23 Layer-1 server gate lands, that fixture starts failing; update the fixture together with the gate.
- Empty-criteria force-fail surfaces in chat as "❌ validation failed (score: 100/100)" — confusing UX; the ticket's open "chat system message" item should also fix the rendered wording.
- `QdrantVectorStore.ts` is excluded in `server/tsconfig.json` — `tsc` runs clean; the TS2345 (line 87) reappears only if the exclude is removed when wiring Qdrant.
- `go.mod` declares `go 1.25.8`, toolchain is 1.26.1 — harmless skew, just don't "fix" it casually.
- The `internal/llm/tools` test failure is a **panic** (config not loaded), so tools tests after `TestLsTool_Run` never run — true pass state of that package is unknown until the harness config issue is fixed.
- All commits on `feat/remove-nomik` share one author identity; "who wrote what" rests on handoff say-so + timestamps. If multi-writer sessions continue, consider distinct `user.name` suffixes per session.
