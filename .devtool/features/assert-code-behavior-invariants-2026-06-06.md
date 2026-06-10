---
id: "assert-code-behavior-invariants-2026-06-06"
status: "backlog"
priority: "high"
assignee: "d34d"
dueDate: null
created: "2026-06-06T14:00:11.000Z"
modified: "2026-06-06T14:00:11.000Z"
completedAt: null
labels: ["architecture", "invariants", "reliability", "north-star"]
order: "a0"
---
# Assert code behavior — program invariants in to guarantee behavior

> **Captured verbatim:** "ACT needs to assert that code behavior exists, effectively
> programming in invariance in the code to guarantee behavior."

The principle: ACT should **mechanically assert** that a behavior holds, not *assume* it
or *wish* it via prompt text. Build the guarantee into the code (invariants, runtime
assertions, design-by-contract) so the behavior can't silently not-happen.

This is the through-line of the 2026-06-06 session — every bug was a place where behavior
was assumed/prompt-wished but never asserted:

- **Agent liveness** was assumed ("the runner polls, so it's alive") but never asserted →
  the agent went offline mid-life → assignment deadlock. Fix made the poll *assert*
  liveness. (`server/src/index.ts` `/api/tasks/assigned`)
- **Pipeline health** was claimed by the Observer ("all clear") without asserting the
  pending→assigned hop → it reported healthy on a dead pipeline. Fix added code-enforced
  detection. (`detectAnomalies`)
- **The "Ready to start?" confirmation** is *still* prompt-only — a soft nudge the model
  can ignore. The invariant version: the orchestrator **refuses** a `PROJECT_BRIEF` that
  arrives in the same turn as the question. (Currently `planner.go` prompt text, not a
  code gate.)
- **Verification of my own claims** — I asserted "the re-dispatch fell flat" from one log
  line without asserting (checking) the success path existed. Same failure mode, human side.

Maps directly onto the architecture-flows taxonomy: push everything from `prompt-only ⌕`
(wished) toward `ok` (code-verified). And it's the same idea as the project thesis —
**"structure substitutes reasoning"**: a code invariant is structure doing the work instead
of hoping the LLM follows an instruction.

## What this could concretely become (to develop, not yet decided)
- An invariant/assertion layer in the orchestrator + server: liveness, assignment
  serviceability, confirmation gates, one-task-per-agent, deps-satisfied-before-assign, etc.
  — each enforced in code with a loud failure, not a silent wrong state.
- Convert the highest-value prompt-only behaviors into code gates (start: the confirmation
  hard-stop; the QA double-fire/cancel; runner-on-resume spawn).
- Possibly a `@invariant` convention in SPIL / success_criteria so a task's guaranteed
  behaviors are declared and then *checked*, not trusted.

Next step: turn this into a short design note (one invariant catalog + where each is
enforced) before building anything.
