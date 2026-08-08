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

## Design (settled 2026-08-08 — direction A only; B dropped as token-waste)
Current matching (verified): AgentRegistry.getOptimalAgent filters
`requiredCapabilities.some(cap => agent.capabilities.includes(cap))` — exact
string overlap with the runner's registered tags, no semantics.

SHIP NOW: code-level rejection at task creation. Task with any capability not in
the registered union is NOT created; Planner gets a rejection message naming the
invalid entries + the valid vocabulary, and MUST re-emit the task VERBATIM except
the capability fix (the rejection prompt includes the original task JSON verbatim
to prevent sloppy/hurried lower-quality re-creation). The stepwise self-reasoning
protocol (define→tools→roles→necessary?) was considered and REJECTED for now:
token cost, negligible benefit.

FLAG + MINE (cheap, ship with the gate): every rejection logs a distinctive
event (capability_rejected, with the requested capability + task title) to the
ChronLog. Post-release telemetry aggregates these: capabilities the Planner
repeatedly reaches for (e.g. "documentation") become CANDIDATES for real new
swarm roles — design the role by analyzing the tasks that wanted it. Lean into
what the model wants to specify; test whether a dedicated role outperforms.

UNFINISHED — note for later: whether Planner-side edit-in-place of a REJECTED
(never-created) task beats reject+re-emit — see
task-mutability-by-state-2026-08-08. The re-emit flow ships first because it
needs no new server surface.

## Constraints
- Deterministic orchestrator/server code, not prompt-only (Observer rescue stays
  as the backstop, not the mechanism).
- Do not silently rewrite the capabilities — reject + inform; the Planner decides.

## Invariants
- No task reaches the server with capabilities matching zero configured roles.
- Observer unservable_task anomaly logic untouched (still the net for runtime
  agent-offline cases, e.g. all agents of a capable role crashed).
