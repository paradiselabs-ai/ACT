---
title: Known Limitations
status: current
owner: project-owner
last_verified: 2026-08-08
---

# Known Limitations (v0.1.0-alpha)

The alpha contract: the happy path works, the edges are known. Everything here
is a known edge — loud where possible, documented where not. Ticket references
point into `.devtool/features/`.

## Swarm recovery — no kill switch for hung agents

If a Tier 2 swarm agent hangs, `task retry` spawns a replacement but cannot
kill the zombie process. The zombie and the replacement may race on the same
files, and both may call `task complete` for the same task ID. There is no
partial-progress persistence — abort + retry restarts from zero.

**Workaround:** restart `act-agent` (the Runner's process-group cleanup kills
the whole subtree on shutdown).
Tickets: epic `swarm-recovery` (`swarm-recovery-*-2026-05-12`, 7 tasks).

## Observer anomalies have no resolution trail

Observer anomalies surface as chat messages but carry no `anomaly_id` and no
`resolved` event — there is no audit trail connecting a flagged anomaly to its
fix. Ticket: `compaction-anomaly-lifecycle-chronlog-ids-2026-05-12`.

## Compaction is Planner-only

Auto-compaction (AutoCompactTokens threshold) fires only for the Planner
thread. Observer / Assurance / QA threads grow unbounded in long sessions.
There is no manual `/compact` command yet. The summarizer falls back to the
Planner's provider/model — but an ACP-backed Planner (backend:
claude-code/antigravity) has none, so compaction is disabled unless you
configure `agents.summarizer` explicitly (a startup warning says so).
Tickets: `compaction-all-tier1-agents-not-just-planner-2026-05-12`,
`compaction-slash-and-palette-command-2026-05-12`.

## QA synthesis persists a marker, not the deliverable

When QA/Synthesizer assembles validated work, the server stores a
`synthesizedAt` timestamp plus a short summary — not the full assembled
deliverable text. The deliverable lives in the chat transcript only.
Ticket: `qa-deliverable-text-persistence-2026-05-19`.

## Free-tier Tier 1 models drift

ACT's contract (JSON tool calls, `CREATE_TASK:` / `PROJECT_BRIEF:` markers,
JSON verdicts) is more than most free-tier models reliably produce: observed
failures include duplicate task-list re-emission, XML envelopes inside JSON
args, and rate-limit retry storms. This is why Tier 1 is backend-selectable
via ACP — run Planner/Observer/Assurance/QA on an agent CLI you already trust
(`"backend": "claude-code"`, `"gemini"`, or `"antigravity"` per role in
`~/.act.json`), or use paid models for in-process Tier 1.

## Researcher cannot run on the antigravity backend

The researcher role is read-only by contract on every backend (tool subset on
act-agent, `--disallowedTools` on claude-code, `--approval-mode plan` on
gemini). The agy CLI has no read-only or plan mode — its only restriction flag
limits terminal access, not file writes — so `researcher` + `antigravity` is
rejected at config-set time (`/swarm`, `act-agent swarm set`) and again at
runner startup. Use any other backend for researcher. This is a deliberate
guard, not a bug.

## Validation gate hardening is partial

The 95% gate fails closed on the known holes (zero-criteria submissions are
rejected server-side; evidence-free `passed:true` verdicts are rejected;
the verdict parser refuses empty `criteriaResults`). Residual: a verdict with
non-empty junk criteria items (e.g. `[{}]`) still passes the length check.
Ticket: `assurance-fail-closed-empty-criteria-2026-05-26`.

## Task timeout default

The Runner kills any single swarm invocation after 120s by default. Raise
`TASK_TIMEOUT_MS` for heavyweight work.

## Misc

- Brownfield codebase notes are fenced and directive-scrubbed before prompt
  injection, but onboarding a hostile repo is still an LLM-mediated trust
  decision — review the analysis the Planner presents.
- ACP has no system-prompt channel, so Tier 1 role priming arrives as a user
  message with a do-not-respond header. Smaller models may still emit a short
  acknowledgment as their first message — cosmetic noise, not a malfunction.
- On task completion each swarm agent makes one extra model call to write its
  completion broadcast. (The pre-task coordination call is now skipped when no
  peer has work in flight; the broadcast call remains — replacing it with a
  deterministic template is an open decision,
  `swarm-coordination-call-overhead-2026-07-13`.)
- Socket.io dashboard handlers exist but have no client consumer (clients use
  REST `/api/log`).
- `flushToSQLite` is a no-op stub under the default `jsonl` storage.
- ChronLog replay is O(events) per server restart; snapshots are a backlog
  item (`docs/ARCHITECTURE_PATTERNS.md`).
