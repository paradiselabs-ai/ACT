---
id: "pvm-search-no-relevance-floor-2026-08-13"
status: "done"
priority: "medium"
assignee: null
dueDate: null
created: "2026-08-13T17:30:00.000Z"
modified: "2026-08-19T21:45:00.000Z"
completedAt: "2026-08-19T21:45:00.000Z"
labels: ["server", "PVM", "memory"]
order: "a4"
---
# PVM search returns noise as if it were memory (no relevance floor)

## Spec
`LocalEmbeddingVectorStore.search` supports a `threshold`, but it is opt-in and
`GET /api/pvm/search` never passes one. The endpoint always returns `limit` rows sorted by
cosine similarity, however low. Live: query `"build a kanban board UI"` against real history
returned a top hit at **0.22** about an unrelated README synthesis. A Planner (or the Runner
context builder) cannot distinguish "nothing relevant is remembered" from "here is the
relevant memory".

## Success Criteria
- `/api/pvm/search` applies a default minimum similarity, overridable per request.
- A query with no semantically related history returns an empty result set, not filler.
- The response carries the similarity score for every row (it already does — keep it).
- Threshold value is chosen from measurement on the real store, not guessed: sample ~20 queries
  with known-relevant/known-irrelevant targets and pick the separating value; record it in the
  ticket before shipping.

## Constraints
- No reranker, no hybrid BM25 in this ticket — floor only.
- Do not change the embedding model (changing it invalidates `data/pvm-vectors.jsonl`).

## Invariants (code-level)
- `search()` stays pure over `this.points`; the default lives at the route, not in the store.
- Empty results stay a 200 with `results: []`, never an error.

## Repro / Evidence
`docs/audits/memory-system-audit-2026-08-13.md` §3.4.

## Resolution note (2026-08-19)
Default floor = **0.28**, measured, not guessed: 10 on-topic vs 10 off-topic queries against BOTH the seeded 3-role test project (on-topic rank-1 0.449–0.828, off-topic 0.025–0.164) and a copy of the real 1786-event history (0.310–0.664 vs 0.109–0.228). Populations separate in (0.228, 0.310); 0.28 sits inside with margin on the noise side. The audit's noise query "build a kanban board UI" measures 0.221 and now returns `[]`. Overridable per request via `?threshold=`. Rationale comment lives at the route constant in `server/src/index.ts`.
