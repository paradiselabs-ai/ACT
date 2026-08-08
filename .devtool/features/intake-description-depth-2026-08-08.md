---
id: "intake-description-depth-2026-08-08"
status: "todo"
priority: "medium"
assignee: null
dueDate: null
created: "2026-08-08T08:50:00.000Z"
modified: "2026-08-08T08:50:00.000Z"
completedAt: null
labels: ["prompts", "intake", "planner"]
order: "a1"
---
# Intake never asks for a detailed description — one-liners silently become the brief

## Describe
`planner.go` intake instruction says "Acknowledge whatever the user already gave; ask
only for what's missing." A one-line opener ("I want to build X") therefore counts as
the description answered, and the Planner never probes deeper. Live evidence
(2026-08-08 LinkDock e2e): opening line became the brief's description verbatim; the
Planner asked stack → criteria → roles but never asked for project detail. With a
well-prepared user the gaps were covered by pasted success criteria; with a vague user
the Planner fills gaps with its own guesswork and the swarm builds the guess.

## Success Criteria
- Intake section gains an explicit description-sufficiency step: unless the opening
  message already pins down what's being built, for whom, core behaviors, and rough
  scope, the Planner's FIRST intake question is "describe the project in as much
  detail as possible" (with 2-3 concrete prompts for what detail means: core flows,
  data, what done looks like).
- The sufficiency bar is written in the prompt as a checklist the Planner tests the
  opener against — not left to vibes.
- Mentions the /brainstorm escape hatch once it exists (see
  brainstorm-intake-mode-2026-08-08): "or type /brainstorm to work it out together".
- Live intake with a one-line opener produces the detail question; a rich opener
  (covers the checklist) skips it. Transcript evidence quoted here before done.

## Constraints
- planner.go intake section only; no orchestrator changes (this is prompt behavior).
- Keep the addition tight — intake prompt is on the token diet; a few lines, not a
  page.
- planner-prompts freshness artifact: this change stales it further — regenerate per
  UPDATE_LOOPS §3 in the same effort or log why not.

## Invariants
- 5-question order preserved (description first); brownfield 2-question variant
  untouched except the shared confirmation rule from the gate ticket.
