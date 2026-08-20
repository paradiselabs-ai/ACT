# DISC — openloop v1 ("reminders with receipts")

**Dual-consumer spec.** This one document drives two independent builds of the same product:
(A) a direct Claude Code session, (B) ACT's full Planner-intake → swarm pipeline, as a
deliberate stress test. Neither builder receives instructions beyond this document. Do not
edit this spec mid-run — if it's wrong, both runs finish on the wrong spec and the comparison
stays valid.

Codename: **openloop**. A macOS CLI agent that reads your PiecesOS long-term memory (the
AI-generated summaries of everything you did on your machine), finds YOUR open commitments
in them, and reminds you — and every reminder carries a receipt: the Pieces summary ID,
annotation ID, timestamp, and the exact quote that caused it. You can mark a reminder wrong,
and that specific mistake never comes back.

***

## Describe

**What's missing and why it matters.** Personal memory/reminder agents (Mem Agent et al.)
infer commitments from your activity and nag you, but they are black boxes: no reminder
shows its source, and "wrong" feedback is a vague thumbs-down at best. The differentiator
here is provenance + correction: every surfaced commitment is traceable to a verbatim quote
in a real PiecesOS memory record, and corrections are enforced in code (a suppressed item
cannot be re-extracted), not just wished into a prompt.

**The data source (live-verified 2026-08-19 on the target machine).** PiecesOS runs a local
HTTP API on port 39300. No auth, no OAuth, plain GET requests:

- `GET http://localhost:39300/.well-known/health` → `ok:<uuid>` (liveness check)
- `GET http://localhost:39300/workstream_summaries` → `{iterable:[…]}` — all workstream
  summaries. Each item is a SHELL: `id`, `created.value` / `updated.value` (ISO8601), and
  (on the single-item endpoint) `annotations.indices` — a dict whose KEYS are annotation
  UUIDs. The list endpoint may return shells with empty annotation indices for
  still-processing recent summaries; the single-item fetch is authoritative.
- `GET http://localhost:39300/workstream_summary/{id}` → one summary incl.
  `annotations.indices`.
- `GET http://localhost:39300/annotation/{annotationId}` → `{id, type, text, created, updated}`.
  The narrative content lives here; relevant `type` values observed: `SUMMARY`, `DESCRIPTION`.

So the read path is: list summaries → filter to a recent window by `updated.value` →
single-fetch each → collect annotation UUIDs from `annotations.indices` keys → fetch each
annotation's `text`. That text is what gets scanned for commitments.

**What v1 is.** A TypeScript CLI + optional polling daemon:

1. **Scan** — pull workstream summaries updated within the last `windowDays` (default 3)
   via the read path above. Only annotations not yet processed (tracked by annotation ID +
   text hash, since summaries update in place) go forward. If PiecesOS is unreachable, fail
   loudly with the health-check URL in the error — never proceed silently empty.
2. **Extract** — send new annotation texts to the Claude API. The model returns candidate
   commitments as structured JSON: commitment text, optional due date, and evidence (the
   verbatim quote from the annotation). Commitments belonging to *other people* ("Kareem
   said he'd cut the demo") are explicitly NOT the user's commitments and must not be
   extracted — the fixture set tests this. Pieces summaries are third-person narrations of
   the user's own activity ("Updated the Paradise Labs profile…"), so the prompt must treat
   the narrated actor as "me" while still excluding other named people's obligations.
3. **Store** — append every occurrence to an append-only JSONL event log. Current state
   (open / done / suppressed) is derived by replaying the log, never by mutating it. Event
   types: `commitment_detected`, `reminder_fired`, `commitment_done`, `correction`.
4. **Remind** — items that are due/overdue (or have no due date and are older than a
   configurable staleness threshold) fire a macOS notification via `osascript`. The
   notification body includes a truncated quote. `openloop list` shows the full receipt for
   every open item.
5. **Correct** — `openloop wrong <id> [reason]` appends a `correction` event. Two effects,
   both required: (a) code-level: a hash of the evidence quote goes on a suppression list
   checked BEFORE any future commitment is persisted — re-scanning the same Pieces data
   cannot resurrect the item; (b) prompt-level: recent correction reasons are injected into
   future extraction prompts as negative guidance. (a) is the guarantee; (b) is the tuning.
6. **Done** — `openloop done <id>` appends `commitment_done`; the item leaves `list` and
   never reminds again.

**What v1 is NOT** (explicitly out of scope; see Constraints): no GUI/menubar app, no
Obsidian/Notion/browser/calendar integrations, no vector store, no database, no OAuth, no
Pieces SDK, no "capture layer" services, no plugin system. Those are v2+ candidates that
only exist if v1 proves the loop is worth living with.

**Evidence duties for a cold builder.** This is greenfield — there is no existing
implementation to collide with. Verification owed before coding: (1) the target directory
is empty/new and NOT inside the ACT repository; (2) `curl http://localhost:39300/.well-known/health`
answers (PiecesOS is running on the machine — it is, verified).

**CLI surface (complete list):**

| Command | Effect |
|---|---|
| `openloop init` | Interactive/flagged config: pieces URL, window days, staleness days, interval, model. Writes `~/.openloop.json`. |
| `openloop scan` | One scan-extract-store-remind pass. `--dry-run` prints would-be events without writing. |
| `openloop list` | All open commitments with full receipts (summary ID, annotation ID, timestamp, quote, age, due). |
| `openloop done <id>` | Mark complete. |
| `openloop wrong <id> [reason]` | Correction: suppress + learn. |
| `openloop daemon` | Loop `scan` on an interval (default 30 min) until killed. |

**Tech decisions (fixed, not open for redesign):**
- Node 20+ / TypeScript, strict mode. Runtime dependency: `@anthropic-ai/sdk` ONLY —
  PiecesOS is reached with Node's built-in `fetch`, no Pieces SDK. Dev deps: `typescript`,
  `tsx`, `@types/node`. Tests use `node:test` (built-in).
- Default extraction model: `claude-sonnet-5`, overridable in `~/.openloop.json`. API key
  from `ANTHROPIC_API_KEY`.
- Two test seams, both env-driven:
  `OPENLOOP_SOURCE=fake:<dir>` reads canned Pieces-shaped JSON fixtures instead of hitting
  localhost:39300; `OPENLOOP_EXTRACTOR=fake:<path.json>` returns canned extraction JSON
  instead of calling the Claude API. Together the entire pipeline is testable with no
  PiecesOS and no API key.
- Notifications: `osascript -e 'display notification …'` via child_process. No notifier deps.
- Data at `~/.openloop/events.jsonl` (append-only log) and `~/.openloop/scan-state.json`
  (processed annotation-ID → text-hash map — this one IS mutable; it's a cache, not history).

**Receipt shape (the load-bearing type):** every commitment carries
`{ summaryId, annotationId, capturedAt, quote }` — all four non-optional. `capturedAt` is
the annotation's `updated.value`; `quote` is verbatim from the annotation text.

***

## Success Criteria

All testable; a reviewer verdicts pass/fail with no judgment calls.

1. `npx tsc --noEmit` clean and `npm test` green in the project root.
2. Fixture set at `fixtures/pieces/` mimicking the real shapes (a summaries list file +
   per-annotation files), containing ≥6 annotations, including at minimum: 2 planted
   first-person commitments (one with a parseable due date, one without), 1 decoy
   commitment belonging to another named person, and 1 annotation with no commitments.
   A matching `fixtures/extractions.json` drives the fake extractor.
3. With both fakes on, `openloop scan` against the fixtures produces `commitment_detected`
   events for exactly the planted items; a test asserts the decoy is absent.
4. Every line of `events.jsonl` validates against the event schemas; a test feeds a
   malformed event and asserts replay rejects it loudly (no silent skip).
5. `openloop list` renders, for EVERY open item: commitment text, summary ID, annotation
   ID, captured-at timestamp, and verbatim quote. A test asserts rendering throws (not
   omits) if any receipt field is missing.
6. Correction is code-enforced: test runs scan → `wrong <id>` → scan again on unchanged
   fixtures → asserts the suppressed item does not reappear as a new commitment.
7. `done <id>` → item absent from `list` and from due-reminder selection; test asserts.
8. Reminder selection is pure and tested: a function takes (open commitments, now, config)
   and returns items to fire; tests cover due, overdue, stale-no-date, and already-fired-
   today (no duplicate nag within one day).
9. The osascript notification command is built by a pure function; a test asserts the
   constructed string for a sample item (actual notification firing is NOT asserted in tests).
10. Unreachable PiecesOS (real-source mode, no fake) exits non-zero with an error message
    containing `http://localhost:39300/.well-known/health`; a test asserts this using an
    unroutable URL in config.
11. Re-scan idempotence: scanning unchanged fixtures twice produces no duplicate
    `commitment_detected` events (annotation hash tracking works); test asserts.
12. Append-only holds: the events log is only ever opened in append mode; no code path
    rewrites or truncates it (see Invariants for the grep).
13. `README.md` exists with a quickstart ≤15 lines: install, init, scan, the wrong/done loop.
14. One end-to-end happy path documented and demonstrated: scan fixtures with both fakes →
    list shows receipts → wrong one item → done another → re-scan → list reflects both.

## Constraints

- **Standalone repo.** Build in the empty working directory the run provides. Nothing is
  written inside the ACT repository. No imports from ACT code.
- **Dependency ceiling:** one runtime dep (`@anthropic-ai/sdk`). PiecesOS access is built-in
  `fetch` against the endpoints pinned in Describe — adding the Pieces SDK, an HTTP client
  lib, or any other runtime dependency is a spec violation, not a judgment call.
- **No speculative architecture.** No "normalizer service", no event bus, no plugin
  interfaces, no abstraction with a single implementation. Flat `src/` with single-purpose
  modules is the expected shape. Shortest working version wins.
- **Provenance is load-bearing, not decorative.** The four receipt fields (`summaryId`,
  `annotationId`, `capturedAt`, `quote`) are non-optional in the commitment type. Rendering
  or persisting a commitment without them must be a type error / thrown error, never a
  fallback.
- **Append-only event log.** State is replay-derived. Mutating history to change state is
  forbidden (same principle as ACT's coordination log).
- **Corrections enforced in code.** The suppression check runs in the scan pipeline before
  persistence. Prompt-side hinting alone does not satisfy the correction requirement.
- **No API-key- or PiecesOS-dependent tests.** Everything in Success Criteria must pass
  offline via the two fake seams. (Live PiecesOS may be used for manual verification only.)
- **Read-only toward Pieces.** The CLI only ever GETs from PiecesOS. No POST/PUT/DELETE to
  any Pieces endpoint, ever.
- **Scope freeze.** The CLI surface above is complete for v1. No extra commands, no config
  options beyond pieces URL / window days / staleness / interval / model, no "while I'm
  here" features.

## Invariants (code-level)

Greppable assertions that must hold after the build; these are the reviewer's checklist:

1. `grep -rn "truncate\|unlink\|writeFileSync" src/` shows no hits touching the events-log
   path; log writes use append (`appendFile` / flag `"a"`) only.
2. `grep -rn "Math.random" src/` → zero hits (no fabricated confidence/scoring).
3. The commitment type declares `summaryId`, `annotationId`, `capturedAt`, `quote` as
   required (non-optional) fields — verifiable by reading the single type definition file.
4. A suppression function (named `isSuppressed` or equivalent single greppable name) is
   called in the scan pipeline before any `commitment_detected` is appended.
5. The default model string `claude-sonnet-5` and the default Pieces URL
   `http://localhost:39300` each appear in exactly one defaults/config module.
6. `grep -rn "method:" src/` (and equivalent fetch options) shows no non-GET request to any
   Pieces URL.
7. No file under the ACT repo tree is created or modified by the build.
8. `package.json` lists exactly one entry under `dependencies`.

***

## Appendix A — feeding this to ACT (test-run protocol)

This spec doubles as the payload for the ACT stress test. Operator notes (not part of the
product spec):

**Setup:** rebuild the `act-agent` binary from branch tip, validate `~/.act.json` parses,
start the server, then launch the TUI in a NEW empty directory (e.g. `~/dev/openloop-act/`).
Greenfield → the Planner runs the 5-question intake. Answer from this mapping:

- **description:** paragraph 2 of this spec's header + the numbered loop in Describe.
- **techStack:** the "Tech decisions" block + the pinned PiecesOS endpoints, verbatim.
- **constraints:** the Constraints section, verbatim.
- **successCriteria:** the Success Criteria section, verbatim (these become
  `@success_criteria` — the Assurance 95% gate will score against them).
- **agentsInvolved:** `backend_dev`, `qa_engineer`. (No frontend work exists; a Planner that
  spawns `frontend_dev` anyway is itself a finding.)

**Pre-tagged known bugs** — rediscoveries of these do NOT count as new findings (they're
already ticketed on the board): the five `pvm-*-2026-08-13` tickets (routing evidence broken:
untagged project bucket, wrong role join, event-type pollution, no relevance floor) and
`agent-brief-session-save-never-fires-2026-08-13`. Expect the Planner's evidence-based
routing brief to be empty/fabricated; that's the known state.

**Evidence to capture during the run:** the chat transcript, `~/.act/**/debug.log`,
`~/.act/runners/*.log`, `server/data/coordination-log.jsonl`. File anything new as kanban
tickets per TASK_TRACKING §6 with a Repro/Evidence section.

**What to watch specifically:** intake→brief→task fidelity (does the task breakdown
preserve the constraints — especially the dependency ceiling, the two fake seams, and
read-only-toward-Pieces?), cross-task file conflicts in one small repo, whether validation
actually scores against these criteria or paraphrases them soft, and whether qa_engineer
work lands before the code it tests exists.

## Appendix B — comparison protocol (after both runs)

Score each build against the 14 Success Criteria (pass/fail each), then diff: criteria
passed, wall-clock time, token spend, human interventions needed, and count of spec
violations (Constraints breached, Invariants failed). ACT findings become tickets; product
learnings (if either build makes the loop feel useful) feed the openloop v2 decision.
