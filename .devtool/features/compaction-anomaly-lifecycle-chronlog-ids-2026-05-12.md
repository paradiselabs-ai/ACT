---
id: "compaction-anomaly-lifecycle-chronlog-ids-2026-05-12"
status: "todo"
priority: "high"
assignee: "d34d"
epic: "compaction"
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: null
labels: ["compaction", "observer", "chronlog", "anomalies"]
order: "a4"
---
# Observer anomaly lifecycle — flag/resolve with ChronLog-backed IDs

Observer flags anomalies via auto-route → Planner today, but there's no anomaly_id, no resolved-event, no way to query "what's open." On compaction, unresolved anomalies can vanish into prose memory.

**Fix:** introduce two new CLI commands + matching ChronLog event types.
- `act-agent observer flag --kind <stuck_task|file_conflict|idle_agent|...> --details <text>` → server emits `observer_flag` event with new `anomalyId`.
- `act-agent observer resolve --id <anomalyId>` → server emits `observer_resolved` event referencing the open ID.
- New endpoint `GET /api/observer/open?project=X` returns currently-open anomalies (flags without matching resolved events).

On compaction, the summarize prompt is augmented with the open-anomaly list pulled from `/api/observer/open`. Anomalies survive compaction by reference, not by prose.

**Files:** `server/src/index.ts`, `server/src/services/ChronologicalLog.ts` (new event types), `act-agent/cli/act-cli.ts` (new subcommands), `act-agent/internal/llm/prompt/observer.go` (teach Observer the flag/resolve protocol).

**Depends on:** `compaction-structured-summary-template-2026-05-12` (the structured summary is where the open-anomaly list lands).

**Success criteria:**
- New CLI subcommands work end-to-end.
- ChronLog replay reconstitutes open anomalies after server restart.
- Compaction summary includes open anomalies verbatim (no LLM re-derivation).
- Build + vet + jest clean.
