---
id: "spil-evolve-to-agentic-language-2026-06-10"
status: "backlog"
priority: "high"
assignee: null
dueDate: null
created: "2026-06-10T00:00:00.000Z"
modified: "2026-06-10T00:00:00.000Z"
completedAt: null
labels: ["SPIL", "language", "orchestrator", "epic"]
order: "a0"
---
# EPIC: Evolve SPIL from spec framework → agentic expression language

**Thesis:** SPIL today is *data the model reads*. To become an agentic language it must
become *instructions the machine runs* — one new component: a parser that emits a tree +
an evaluator that executes the `@` forms while leaving `>` directives for the model.

This is the umbrella ticket. The value lands at **stage 1** (executable contract); the
lisp/s-expression step is **stage 2** and only adds composability. Do not start stage 2
until stage 1 is proven in dogfood.

**Child tickets:**
- `spil-stage1-proof-criteria-gate-2026-06-10` — smallest slice (evaluated fail-closed
  `@success_criteria` gate). Demo-able / HN-worthy.
- `spil-stage1-parser-ast-evaluator-2026-06-10` — full stage 1 (parser→AST + evaluator,
  wire `@depends_on` / `@context` / `@error_handling`).
- Stage 2 (s-expr + macros) — not yet ticketed; create only after stage 1 ships.

**The `@`/`>` split must be preserved:** `@` becomes machine-executed (deterministic layer);
`>` stays model-interpreted (the context-engineering layer). Do NOT make `>` executable.

## Docs to reference (read before any child ticket)
- `docs/Vault/Agent Coordination Toolkit/nestty/SPIL-as-language-evolution.md` — **primary
  design note**, stages + what-changes table + build order
- `docs/Vault/Agent Coordination Toolkit/nestty/SPIL.md` — SPIL spec (CTD, manifest, syntax)
- `docs/Vault/Agent Coordination Toolkit/nestty/plans/SPIL Integration Plan.md` — current
  integration inventory (parser scope, producer, consumer, gaps)
- `docs/spil-evolution.html` — the visual pitch / shared mental model

## Honest scope guard
SPIL-as-language is the **orchestration layer only** — ACT's internal wiring, NOT the
project code the swarm ships. Owning the full corpus is what makes a custom language safe
here. Do not extend this toward "agents write user deliverables in SPIL."
