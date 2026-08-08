---
id: "token-switch-budget-reroute-2026-08-08"
status: "backlog"
priority: "low"
dueDate: null
created: "2026-08-08T10:20:00.000Z"
completedAt: null
labels: ["runner", "server", "TUI", "cost", "feature"]
order: "a5"
---
# /token-switch — per-role token budgets with mid-flight model/agent reroute

## Describe
User-proposed: set a token limit for swarm roles at any time; when a role hits
its budget, either (a) redirect its current/queued tasks to another agent, or
(b) swap the model/backend running that role. Deterministic code, no LLM calls.

Honest scoping (why this is backlog, not todo): ACT currently tracks ZERO token
usage — no per-call token counts from any backend reach the runner or server.
Prerequisites before the command is possible:
1. Usage capture per invocation (claude/gemini/agy one-shots report usage in
   different formats; the in-process path has provider usage in responses).
2. A per-role running counter (server-side, so it survives runner restarts —
   ChronLog event or agent metadata).
3. THEN /token-switch <role> <limit> + the reroute: on breach, runner declines
   further tasks for that role (server reassigns via existing capability
   routing) or the role's backend/model is flipped (existing /swarm machinery).

Related existing mechanics to reuse, not duplicate: /swarm <role> <backend>
(backend swap), task abandon/retry (reassignment), capability routing
(assignOptimalAgent).

## Success Criteria (phase-gated)
- P1: every swarm invocation logs prompt+completion token counts to the server.
- P2: /status shows per-role cumulative tokens for the session.
- P3: /token-switch sets a budget; breach visibly stops new tasks for the role
  and reroutes via existing assignment; optional auto-swap config.

## Constraints
- Zero model calls in the mechanism itself; counters and gates are code.
- Runner stays thin: counting at invocation site, policy on the server.
