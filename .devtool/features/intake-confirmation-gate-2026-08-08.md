---
id: "intake-confirmation-gate-2026-08-08"
status: "todo"
priority: "high"
assignee: null
dueDate: null
created: "2026-08-08T08:50:00.000Z"
modified: "2026-08-08T08:50:00.000Z"
completedAt: null
labels: ["orchestrator", "prompts", "intake", "bug"]
order: "a0"
---
# Intake confirmation gate — PROJECT_BRIEF accepted after a non-confirmation reply

## Describe
The "Ready to start?" hard stop is prompt-wished only. Live evidence (2026-08-08
LinkDock e2e, session db `linkdock/.act/act.db` messages table): after the Planner's
intake summary, the human's reply was a pasted description paragraph — not any form of
yes — and the Planner emitted `PROJECT_BRIEF:` in the next turn. Orchestrator parsed
it, created the project, and dispatched 4 tasks. Any non-empty reply after the summary
is effectively treated as confirmation; an accidental Enter or stray paste spins up
the swarm and burns backend quota on an unapproved brief.

## Success Criteria
- Code gate in the orchestrator's PROJECT_BRIEF parse path: before POSTing the
  project, check the human turn immediately preceding the Planner's brief-emitting
  turn against an affirmative pattern (yes/y/yep/go/ready/start/confirm/do it/ship it,
  case-insensitive, allowing surrounding words). Non-match → do NOT create the
  project; emit a system message + re-prompt the Planner to re-ask for confirmation,
  folding the human's actual reply in as updated intake info.
- Prompt line added to the intake section (planner.go): a reply that is not an
  explicit yes is new intake information — restate the summary and ask again.
- Live e2e: pasting a paragraph at the confirm step does NOT create the project;
  replying "yes" after the re-ask does. Evidence quoted here before done.
- Unit test on the affirmative matcher (table-driven: yes-variants pass, the LinkDock
  paste + empty string + questions fail).

## Constraints
- Gate lives at the orchestrator boundary (same philosophy as the fail-closed
  empty-criteria verdict gate and the build-contract auto-scold): never left to the
  LLM's judgment. Prompt line is support, not the enforcement.
- Touch only orchestrator.go (brief parse path) + planner.go intake section + tests.
- Brownfield intake path uses the same gate (shared confirmation rule).

## Invariants (code-level)
- Project creation from PROJECT_BRIEF is unreachable without a preceding
  affirmative-matching human turn in the same session.
- Matcher is pure function, unit-testable, no LLM call.
