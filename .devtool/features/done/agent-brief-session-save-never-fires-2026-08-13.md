---
id: "agent-brief-session-save-never-fires-2026-08-13"
status: "done"
priority: "high"
assignee: null
dueDate: null
created: "2026-08-13T17:30:00.000Z"
modified: "2026-08-19T21:45:00.000Z"
completedAt: "2026-08-19T21:45:00.000Z"
labels: ["runner", "memory", "server"]
order: "a2"
---
# Agent brief persistence has never fired: 0 brief_stored events in all history

## Spec
`POST /api/projects/:name/briefs` stores an agent brief and appends a `brief_stored` event;
`ChronologicalLog.restoreFromLog` replays that event type back into `project.briefs`. Both
halves work. **Neither has ever run:** the 1785-event production log contains **0**
`brief_stored` events, and boot reports `0 briefs` restored across 15 projects.

Consequence: the documented "session save before exit" loop (`act-agent brief update`) and the
Runner's HTTP brief injection (`fetch /briefs/:agentId`, non-fatal 404) are documented memory
that does not exist at runtime. Every swarm agent starts from zero project memory on every task,
and the `claude-code` backend path skips the HTTP inject entirely when an on-disk AGENTS.md
exists — so the server-side brief is doubly unused.

## Success Criteria
- After a swarm agent completes a task, a `brief_stored` event exists for that agent+project.
- Restarting the server restores a non-zero brief count.
- `act-agent context <agent> --project <p>` renders the stored brief section.
- Decide and record: does the Runner write the brief, does the agent, or is the feature cut?
  A cut is an acceptable outcome — but then the docs and the CLI command must go with it.

## Constraints
- Do not add an LLM call to the write path.
- Do not make brief write failures fatal to task completion.
- If the feature is cut, remove it from CLAUDE.md/README rather than leaving dead docs.

## Invariants (code-level)
- Brief content stays a plain string keyed by `{projectName, agentId}`.
- `brief_stored` replay stays idempotent (last write wins on replay).

## Repro / Evidence
```
grep -c '"type":"brief_stored"' server/data/coordination-log.jsonl   # → 0
# boot: "Restored from ChronLog: 15 projects, 42 tasks, 0 briefs, 38 agents"
```
`docs/audits/memory-system-audit-2026-08-13.md` §3.2.

## Resolution note (2026-08-19)
Decision recorded: **the Runner writes the brief** (swarm agents stay stateless one-shots; the Runner owns lifecycle writes). After each successful completion it saves a deterministic "Recent Work" summary (last 5 task titles + one-line results, ≤2000 chars, no LLM) to the server brief endpoint. Live-verified: brief_stored events appear, restart restores briefs, CLI context renders them, forced HTTP 500 on the brief write does not fail task completion (opus-task-b).
