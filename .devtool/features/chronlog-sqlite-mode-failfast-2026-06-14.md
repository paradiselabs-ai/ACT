---
id: "chronlog-sqlite-mode-failfast-2026-06-14"
status: "todo"
priority: "low"
assignee: null
dueDate: null
created: "2026-06-14T12:00:00.000Z"
modified: "2026-06-14T12:00:00.000Z"
completedAt: null
labels: ["server", "persistence", "safety", "data-loss"]
order: "z1"
---
# ChronLog — fail fast on unimplemented SQLite storage mode

## Spec
`ChronologicalLog` accepts `storageType: 'jsonl' | 'sqlite' | 'both'` (`server/src/services/ChronologicalLog.ts:28`),
but `flushToSQLite` (:242) is an empty TODO. The flush path (:188-193) routes `'sqlite'` and `'both'`
to it. So `storageType:'sqlite'` makes every flush a **silent no-op = total event-log data loss**;
`'both'` is safe only because JSONL is still written. Default is `'jsonl'` and no caller sets
otherwise, so this is dormant — but it is a config value the type accepts that silently destroys data.
Until `block13-pvm-phase1-lancedb-sqlite` implements real SQLite, the constructor (or `initialize`)
must **reject `'sqlite'`/`'both'` loudly** so the config cannot silently lose data.

## Success Criteria
- Constructing/initializing `ChronologicalLog` with `storageType:'sqlite'` or `'both'` throws a clear
  error (e.g. `storageType 'sqlite' not implemented — see block13; use 'jsonl'`) at construction time,
  NOT silently at first flush.
- `storageType:'jsonl'` and the default are unchanged.
- A test asserts the throw for `'sqlite'` and `'both'` and the no-throw for `'jsonl'`.

## Constraints
- Touch only `server/src/services/ChronologicalLog.ts` (+ its test). Do NOT implement SQLite — that is
  `block13`. No new npm dependencies.
- Keep `'sqlite'|'both'` in the type union (block13 will implement them); only guard against selecting
  them at runtime.

## Invariants (code-level)
- `flushToSQLite` is never reached in normal operation (guarded upstream); if retained it throws rather
  than silently returning.
- `DEFAULT_CHRONOLOGICAL_CONFIG.storageType` stays `'jsonl'` (ChronologicalLog.ts:44).

## Repro/Evidence
`flushToSQLite` empty body (ChronologicalLog.ts:242) under the `:188-193` dispatch; type at :28; default
at :44. `rg -n "storageType\s*[:=]\s*'(sqlite|both)'" server/src` → no caller sets it (dormant). Sweep:
`docs/audits/unwired-code-sweep-2026-06-14.md`.
