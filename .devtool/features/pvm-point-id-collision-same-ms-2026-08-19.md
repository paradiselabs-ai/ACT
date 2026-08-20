---
id: "pvm-point-id-collision-same-ms-2026-08-19"
status: "todo"
priority: "medium"
assignee: null
dueDate: null
created: "2026-08-19T21:30:00.000Z"
modified: "2026-08-19T21:30:00.000Z"
completedAt: null
labels: ["server", "PVM", "memory"]
order: "a5"
---
# PVM: indexed points collide on `${timestamp}_${agent}` — same-millisecond events silently overwritten

## Spec
`LocalEmbeddingVectorStore.store()` upserts by an id built from timestamp + agent name.
Two distinct events from the same agent inside one millisecond overwrite each other.
Measured (2026-08-19, during Task A of the memory-fix batch): real production log has
1786 events → only 1614 indexed points; an unthrottled test run lost a `task_completed`,
a `task_created`, and a `task_assigned`, which produced an empty capability bucket for
one agent's profile. Found by opus-task-a; deliberately NOT fixed in that batch because
the fix changes the sidecar cache key, invalidating the 19 MB `data/pvm-vectors.jsonl`
embedding cache and forcing a full re-embed — an owner call.

## Success Criteria
- Two events, same agent, same millisecond → two distinct indexed points.
- Reindexing the full production log yields point count == event count minus the
  deliberately-skipped coordination duplicates (no silent loss).
- A migration story for the sidecar cache is stated (re-embed once, or key translation).

## Constraints
- Id must stay deterministic across replays (event-sourcing invariant) — derive from
  event content/position, not a random uuid at index time.
- Decide sidecar handling explicitly; do not silently orphan the old cache file.

## Invariants (code-level)
- No `Math.random`/`Date.now()`-at-index-time in the id derivation.

## Repro / Evidence
opus-task-a final report (Task A, plan `/Users/user/.claude/plans/pvm-memory-fixes-2026-08-13.md`):
1786 events → 1614 points on a copy of the real log. Harness workaround: 50 ms spacing
between API calls in `memtest.py`.
