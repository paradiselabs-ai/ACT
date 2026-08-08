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

## Design (owner input, 2026-08-08)
Current matching (verified): AgentRegistry.getOptimalAgent filters
`requiredCapabilities.some(cap => agent.capabilities.includes(cap))` — exact
string overlap with the runner's registered tag list, no semantics ("documentation"
matches nothing even though any file-writing agent can document). Two-layer fix:
- A (default, cheap): closed vocabulary — Planner picks ONLY from the registered
  capability union, code gate rejects everything else (this ticket's criteria).
- B (escape hatch, prompt-level): when the Planner believes a capability outside
  the vocab is genuinely needed, it must reason stepwise BEFORE emitting: define
  the capability → what tools does it reduce to → which roles have those tools →
  "is a new capability term necessary, or does an existing role serve this?" —
  usually collapsing to an existing tag. Keeps door open for future specialized
  roles (e.g. a real documentation agent) without letting casual synonyms strand
  tasks.

## Constraints
- Deterministic orchestrator/server code, not prompt-only (Observer rescue stays
  as the backstop, not the mechanism).
- Do not silently rewrite the capabilities — reject + inform; the Planner decides.

## Invariants
- No task reaches the server with capabilities matching zero configured roles.
- Observer unservable_task anomaly logic untouched (still the net for runtime
  agent-offline cases, e.g. all agents of a capable role crashed).
