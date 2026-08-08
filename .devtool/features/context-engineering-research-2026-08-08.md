---
id: "context-engineering-research-2026-08-08"
status: "backlog"
priority: "medium"
created: "2026-08-08T15:40:00.000Z"
completedAt: null
labels: ["research", "prompts", "architecture"]
order: "a4"
---
# Research: scalable prompt/context engineering — stop growing system prompts per-bug

## Describe
Owner direction (2026-08-08): patching each discovered failure by appending to
system prompts bloats them and decays instruction-following. Need principled
alternatives. Research sweep over: (1) invariant-style enforcement — minimal
prompt statement of invariants + code-level gates as the enforcement (ACT
already trends here: confirmation gate, fail-closed verdicts, capability gate);
(2) structured self-check protocols at decision points (the owner's stepwise
capability-reasoning pattern) vs closed vocabularies — cost/quality tradeoff;
(3) prompt architecture: stable core + per-turn injected context vs monolith;
what the strongest agent harnesses do (Claude Code's own patterns in
docs/ARCHITECTURE_PATTERNS.md); (4) measurement: token cost + adherence
regression per prompt addition. NOT a graph-engineering problem (closed-set
matching + gates are deterministic code; PVM/graph helps evidence-based routing,
a different axis).

## Success Criteria
- A docs/audits note ranking: which current prompt sections could move to code
  gates; a rule for when a new failure gets prompt text vs a gate; proposed
  prompt architecture for Tier-1 roles with est. token budgets.
