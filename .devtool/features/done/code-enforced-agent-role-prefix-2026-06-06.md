---
id: "code-enforced-agent-role-prefix-2026-06-06"
status: "done"
priority: "high"
assignee: "d34d"
dueDate: null
created: "2026-06-06T16:38:58.000Z"
modified: "2026-06-06T22:17:31.000Z"
completedAt: "2026-06-06T22:17:31.000Z"
labels: ["orchestrator", "tui", "invariants", "reliability"]
order: "a0"
---
# Code-enforced agent role prefix (who-is-speaking is an invariant, not inferred)

**Symptom (act-e2e, 2026-06-06):** the Planner (claude-code backend) emitted output starting with
`Human:` — e.g. `Human: stop. why is the synthesizer being triggered again ?` (debug.log:410,
`role:"assistant"`). The developer read it as the Planner *addressing* them and replied, ending up
in a confused human↔Planner exchange. Compounded by the TUI sometimes not rendering the sender at
all, forcing you to deduce it from content.

**Fix:** ACT prepends the authoritative role label to every Tier 1 message **in code** — agents
write normally, ACT owns the label:
```
Planner:   …
Observer:  …
Assurance: …
QA:        …
```
- **Strip then prepend:** remove any leading role-label / `Human:` the model emitted, then prepend
  the real one — so a hallucinated prefix can't survive and you never get `Planner: Human: …`.
- **Code-only, no LLM:** never left for the model to obey/disobey; zero extra tokens.
- **Doubles as the TUI fallback:** the role is in the message text, so even when the banner doesn't
  render the sender, who-spoke is unambiguous.

**Why it matters:** *who is speaking* should be a code-asserted invariant tied to the authoritative
role tag — not something the human infers from text the model wrote. (See
`assert-code-behavior-invariants` and the `docs/Vault/future-post-act-builds-act.md` writeup.)

**Where:** message construction/rendering for Tier 1 agents — `internal/app` (message ownership
tagging) + `internal/tui/components/chat` (render). Likely prepend at the ownership-tagging layer so
it's backend-agnostic (act-agent and claude-code/ACP both).
