---
id: "unservable-task-capabilities-gate-2026-08-08"
status: "todo"
priority: "high"
assignee: null
dueDate: null
created: "2026-08-08T15:20:00.000Z"
completedAt: null
labels: ["orchestrator", "server", "planner", "bug"]
order: "a0"
---
# Planner can emit requiredCapabilities no agent provides — task strands unservable

## Describe
Live (2026-08-08 link-dock run): Planner created "README documentation" with
requiredCapabilities ["documentation"]. No swarm role registers that capability
(runner DefaultCapabilities), so assignment never matched and the task sat
pending 22+ minutes. It took the Observer escalating CRITICAL three times and a
Planner autoroute turn ("abandoning and re-dispatching with capabilities dev-1
actually has") to self-heal — burning Tier-1 turns on something deterministic
code should have rejected at creation.

## Success Criteria
- CREATE_TASK validation (orchestrator side, at parse/POST time): every entry in
  requiredCapabilities must be provided by at least one CONFIGURED swarm role
  (union of runner DefaultCapabilities across spawnable roles). On mismatch: task
  NOT created; system message + a re-prompt to the Planner listing the valid
  capability vocabulary and the offending entries.
- The valid capability vocabulary is included in the Planner's task-creation
  prompt section, so the mismatch is rare, not routine.
- Unit test: task with ["documentation"] rejected with the vocab in the error;
  task with ["typescript"] accepted.
- Live e2e: same README task shape gets rejected at creation and the Planner
  re-emits with valid capabilities in the next turn.

## Constraints
- Deterministic orchestrator/server code, not prompt-only (Observer rescue stays
  as the backstop, not the mechanism).
- Do not silently rewrite the capabilities — reject + inform; the Planner decides.

## Invariants
- No task reaches the server with capabilities matching zero configured roles.
- Observer unservable_task anomaly logic untouched (still the net for runtime
  agent-offline cases, e.g. all agents of a capable role crashed).
