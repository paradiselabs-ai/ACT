---
id: "autonomy-dial-checkpoint-vs-full-2026-07-07"
status: "backlog"
priority: "medium"
assignee: null
dueDate: null
created: "2026-07-07T08:33:31.000Z"
modified: "2026-07-07T08:33:31.000Z"
completedAt: null
labels: ["orchestrator", "prompts"]
order: "z1"
---
# Autonomy dial: checkpoint vs full HOTL mode (orchestration plan Phase 4)

## Spec

ACT's shape is already human-on-the-loop (human talks only to the Planner; Observer watchdog;
Assurance gate). Add the missing knob: a per-project `autonomy` setting —

- `checkpoint` (default, current behavior): Planner pauses for human confirmation at phase
  transitions.
- `full`: after the INTAKE "Ready to start?" confirmation (which stays mandatory), the Planner
  proceeds phase-to-phase unattended; the human supervises via coordination events + Observer
  escalations and can interrupt at any time.

Mostly a Planner-prompt clause + one branch in the orchestrator; safe only AFTER Phase 1
(least privilege) and Phase 2 (hooks + evidence-gated completion) are in.

## Success Criteria

- [ ] `autonomy: "full"` in project config/brief → Planner emits successive CREATE_TASK phases without waiting for human ack (observable in a live TUI e2e run)
- [ ] `checkpoint` (or unset) → behavior identical to today
- [ ] INTAKE hard stop unaffected in both modes
- [ ] Autoroute sliding-window cap (≤5/10min) still applies in full mode

## Constraints

- No new agent, no new process, no TUI surface beyond maybe a status-line indicator.
- Deferred with this ticket, NOT in scope: custom role definitions from files (real friction: role enums in server `roles.ts`, capabilities map, Go prompt dispatcher). Split into its own ticket when picked up.

## Code-level Invariants

- Planner remains the ONLY agent that responds to human input, in both modes.
- The Three-Layer rule holds: autonomy affects when the Planner acts, never lets the Runner or server make decisions.

## Dependencies

- swarm-least-privilege-tool-scoping-2026-07-07 (Phase 1)
- pi-extension-hook-system-2026-04-21 + ralph-wiggum-loop-code-enforcement-2026-05-19 (Phase 2)
