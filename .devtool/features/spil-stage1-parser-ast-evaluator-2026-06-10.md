---
id: "spil-stage1-parser-ast-evaluator-2026-06-10"
status: "todo"
priority: "medium"
assignee: null
dueDate: null
created: "2026-06-10T00:00:00.000Z"
modified: "2026-06-10T00:00:00.000Z"
completedAt: null
labels: ["SPIL", "language", "orchestrator", "parser"]
order: "a2"
---
# SPIL stage-1 full: parser → AST + evaluator (run the `@` forms)

Replace the regex MVP (`SPILParser.ts`) with a real tokenizer → AST, and add an **evaluator**
in the deterministic layer that executes each `@form` against existing orchestrator calls.
This is what turns SPIL from "a note the model reads" into "a contract the machine runs."

Depends on the proof slice landing first (`spil-stage1-proof-criteria-gate-2026-06-10`) so the
gate pattern is established. Parent epic: `spil-evolve-to-agentic-language-2026-06-10`.

**Do NOT start before the proof ticket is marked done** — it establishes the fail-closed gate
shape this builds on.

## Success criteria
- `@depends_on` becomes a real task-graph edge: a task does not start until its named upstream
  tasks are complete (verify with a 2-task chain — downstream stays pending until upstream done).
- `@context` paths are auto-injected into the agent's turn before it fires (verify the
  referenced file content appears in the turn input without the agent fetching it).
- `@error_handling` conditions are represented as checked guards, not prose (at minimum:
  parsed into structured `{condition, response}` the evaluator can act on).
- Parser emits a typed AST consumed by the evaluator; `@success_criteria` gate (from proof
  ticket) is re-expressed through the evaluator, not a separate code path.
- `go build` clean; server build clean; existing Assurance flow still passes.

## Code constraints
- Parser changes in `server/src/services/SPILParser.ts` (or a new module beside it).
- Evaluator lives in `act-agent/internal/app/orchestrator.go` (deterministic middle layer,
  per Three-Layer rule). Planner/Runner untouched.
- `>` directives stay model-interpreted — the evaluator passes them through to the turn,
  it does NOT execute them.
- No s-expression / lisp surface yet (that's stage 2, not ticketed). No speculative macro
  system. Keep the AST shape minimal — only the forms above.

## Docs to reference
- `docs/Vault/Agent Coordination Toolkit/nestty/SPIL-as-language-evolution.md` — §"What
  changes" table (BUILD rows) + §"The split that makes it safe"
- `docs/Vault/Agent Coordination Toolkit/nestty/SPIL.md` — manifest/lazy-load + `spil get`
  (informs `@context` injection design)
- `docs/Vault/Agent Coordination Toolkit/nestty/plans/SPIL Integration Plan.md` — parser
  roadmap phases + what is NOT parsed today
- `server/src/services/SPILParser.ts`, `server/src/index.ts` (task storage),
  `act-agent/internal/app/orchestrator.go`
