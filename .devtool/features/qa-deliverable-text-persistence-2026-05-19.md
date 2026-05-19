---
id: "qa-deliverable-text-persistence-2026-05-19"
status: "todo"
priority: "medium"
assignee: null
dueDate: null
created: "2026-05-19T05:50:00.000Z"
modified: "2026-05-19T05:50:00.000Z"
completedAt: null
labels: ["server", "validation", "architecture-mapping-finding"]
order: "m02"
---
# Persist the QA/Synthesizer final deliverable text server-side

## Finding (partial gap)

Architecture mapping pass (2026-05-19, see `architecture-flows.html` and `.claude/architecture-flows-method.md`) confirmed that QA synthesis persistence is partial:

- `POST /api/tasks/:taskId/synthesis` at `server/src/index.ts:795` DOES persist a `synthesis_complete` event to ChronLog with a short summary string. The handler write is at `server/src/index.ts:805-811`.
- It does NOT persist the full synthesized deliverable text. The complete assembled artifact body lives only in TUI session memory.

So the event survives a server restart (replay rebuilds the in-memory store via `chronologicalLog.restoreFromLog` at `server/src/index.ts:1339`), but the deliverable does not. If the user closes the TUI between synthesis and reading the deliverable, the body is lost.

The prior session's exploration sub-agent had claimed this was a full gap using bad grep terms (`qaOutput|finalDeliverable|synthesizedOutput`); the actual endpoint at `:795` is named `synthesis` and the agent missed it. The partial-gap framing is encoded under `meta.findings[id=F-E]` in `architecture-flows.json`.

## Proposed remediation

Extend the synthesis endpoint's request body with an optional `deliverable` field carrying the full assembled text. When present, persist it alongside the event:

1. Either add a new ChronLog event type `synthesis_deliverable` with the full body, OR (preferred) attach the deliverable text to the existing `synthesis_complete` event's `data` object.
2. On task fetch, surface the deliverable in the `task.metadata.synthesizedDeliverable` field so it is replayable.
3. Update the QA/Synthesizer agent prompt at `act-agent/internal/llm/prompt/qa_synthesizer.go` to pass the assembled body to the endpoint.
4. Update the orchestrator's QA poll loop at `act-agent/internal/app/orchestrator.go:1774` to wire the deliverable through.

Open question: deliverables can be large. Consider a size cap or chunked storage if a single ChronLog event ends up over a megabyte. The current ChronLog has no per-event size limit; this is the moment to decide whether one is needed.

## Success criteria

- After QA finishes a synthesis turn and the server is restarted, the final deliverable body is still available via the task fetch endpoint.
- Existing `synthesis_complete` event consumers (Observer, coordination event loop) are unchanged or backward-compatible.
- TUI rendering of the deliverable in the chat works identically.

## Out of scope

- Making QA outputs queryable by content (that would be a PVM-layer feature, not a persistence-layer feature).
- Versioning of deliverables across multiple syntheses on the same task.

## References

- Architecture mapping: `architecture-flows.html` flow `qa-synthesis`, step `qa_synth → store_mem_vector` marked `gap-found` with the partial-gap note.
- Finding F-E surfaced in `flows-explainer.html`.
- Coord entry: `act-coordination.json` 2026-05-19T05:50:02Z.
- Endpoint handler: `server/src/index.ts:795-830`.
