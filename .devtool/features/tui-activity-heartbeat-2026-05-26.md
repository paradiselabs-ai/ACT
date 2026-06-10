---
id: "tui-activity-heartbeat-2026-05-26"
status: "backlog"
priority: "high"
assignee: null
dueDate: null
created: "2026-05-26T22:00:00.000Z"
modified: "2026-05-26T22:00:00.000Z"
completedAt: null
labels: ["ux", "tui", "polish"]
order: "u01"
---
# TUI activity heartbeat — swarm work invisible while running

## Problem

When the swarm is mid-task, the TUI looks frozen / hung from the user's
perspective. The Tier 2 agents (claude-code via runner.mjs) write files to
the project directory and stream their thinking to `~/.act/runners/<role>.log`,
but none of that surfaces in the chat window. Users see the last chat message
and an empty input prompt for minutes at a time. Natural interpretation: ACT
crashed.

Observed live in the wordtally test session (2026-05-26):
- Files appearing under the project dir prove the swarm is working.
- Chat window static.
- User assumes ACT is hung and is about to ctrl+c — but then a swarm message
  finally lands and proves it was alive the whole time.

This is purely UX — the harness works, the user just can't tell.

## Goal

Make it obvious in the TUI that real work is happening, without polluting
the chat. The bar for "obvious" is: a user who has never seen ACT before
can tell within 5 seconds that something is alive.

## Possible surfaces (pick one or two, not all)

1. **Status line / footer activity indicator.** Reuse the existing footer
   area (the one that shows `pwd` / model / etc). Add a small spinner +
   "swarm: dev-1 working on `<task title>` (3m12s)" line that updates from
   server polling. Cheap, non-intrusive, mirrors what `npm` / `cargo` do.

2. **Per-agent "online with task" badge in `/swarm status`.** Already
   partially there but the user has to type a slash command — passive
   surface is what's missing.

3. **Subtle chat-side ambient line every ~30s** during long swarm work:
   `📝 dev-1 still working… (4 files touched, last activity 12s ago)`.
   The orchestrator already polls `/api/log` every 3s via
   `coordinationEventLoop`; this just synthesises a heartbeat when the
   gap between events exceeds N seconds.

4. **Pulse the role banner color** in chat for the currently-active agent.
   Low-effort visual cue, zero new text.

5. **Stream the runner's stdout one-liner.** Right now stdout/stderr go
   to `~/.act/runners/<role>.log`. Tail the last non-empty line and show
   it as an ephemeral status under the chat input ("dev-1: editing
   tally.py"). High-info, very Claude-Code-like.

## Constraints

- Do not pollute the chat transcript with progress noise. The chat is
  decisions + deliverables; status is its own surface.
- Heartbeat must come from real activity (file writes, server events,
  runner output) — never a synthetic "I'm still here" timer that lies
  if the swarm has actually crashed.
- Cost: heartbeat polling stays well under 1 RPS to the server. Reuse
  existing `coordinationEventLoop` rather than adding a parallel poll.

## Success criteria

- During a swarm-busy phase (Tier 2 agent mid-task), the TUI shows a
  visible "alive + working" indicator that updates at least every 15s.
- When the swarm genuinely IS hung (subprocess died, no file activity for
  >2 min during a claimed in-progress task), the indicator transitions
  to a warning state.
- Indicator clears within 5s of the swarm becoming idle.
- No new lines added to the chat transcript while implementing this.

## Notes

- The Tier 1 ACP-backed agents (Planner, Assurance, QA) have a similar
  problem during their own turns — the user types a message, Planner
  goes silent for 20-90s while Claude Code thinks. Spinner there too.
- Tier 1 ACP `session/update` notifications already stream chunks; the
  visible-streaming is fine for the Planner's reply itself but doesn't
  cover the pre-stream pause. A "Planner is thinking…" indicator
  bridges that gap.
- Related: the existing `currentSpeaker` field in the orchestrator
  already knows which Tier 1 role is mid-turn — easy hook for a Tier 1
  indicator.
