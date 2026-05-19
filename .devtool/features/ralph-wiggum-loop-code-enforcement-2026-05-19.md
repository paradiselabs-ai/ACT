---
id: "ralph-wiggum-loop-code-enforcement-2026-05-19"
status: "todo"
priority: "medium"
assignee: null
dueDate: null
created: "2026-05-19T05:50:00.000Z"
modified: "2026-05-19T05:50:00.000Z"
completedAt: null
labels: ["validation", "swarm", "architecture-mapping-finding"]
order: "m01"
---
# Code-enforce the Ralph Wiggum self-verification loop on swarm agents

## Finding

Architecture mapping pass (2026-05-19, see `architecture-flows.html` and `.claude/architecture-flows-method.md`) confirmed that the Ralph Wiggum self-verification loop lives entirely in prompt text. References appear in:

- `act-agent/internal/llm/prompt/common.go`
- `act-agent/internal/llm/prompt/developer.go`
- `act-agent/internal/llm/prompt/frontend_dev.go`
- `act-agent/internal/llm/prompt/backend_dev.go`
- `act-agent/internal/llm/prompt/qa_engineer.go`
- `act-agent/internal/llm/prompt/researcher.go`
- `act-agent/internal/llm/prompt/assurance.go`

No non-prompt enforcement exists. Grepping for `Ralph` across `server/src/` and `act-agent/internal/` excluding `/prompt/` and `_test` files returns zero hits.

Enforcement is statistical: Assurance Layer 1 looks for evidence of self-verification in the agent's submitted result, but an agent that skipped Ralph would only be caught by another LLM, not by a compiler or middleware. This is the difference between code-enforced and prompt-only behavior, and it is the kind of gap the new four-status taxonomy (`ok` / `prompt-only` / `gap-found` / `unverified`) was designed to surface.

## Proposed remediation

Add a gate at task complete time. Two possible shapes:

1. **Lightweight server-side gate.** When a Tier 2 agent calls `POST /api/tasks/:taskId/complete`, the handler checks that the submitted result contains at minimum a recognizable self-verification trace (e.g. a `## Self-Verification` heading, or a JSON field `selfVerificationPassed: true` on the request). If absent, return 422 with a message describing what is missing. The runner then surfaces the failure to the agent and asks it to re-run with the verification.
2. **Layer-1 strict-mode in Assurance.** Already partially exists: Assurance is meant to verify the Ralph loop ran. Tighten the prompt at `act-agent/internal/llm/prompt/assurance.go:29-57` to refuse to pass any submission that does not show explicit self-verification reasoning. Cost: this still relies on an LLM judge, just with a stricter prompt.

Option 1 is real code enforcement and is preferred. Option 2 is statistical enforcement and is what currently exists.

## Success criteria

- Skipping the self-verification step on a Tier 2 task results in either a 422 response from the complete endpoint or a deterministic Assurance failure with a specific reason ("no self-verification trace found").
- An agent that DID run Ralph passes through unchanged.
- Existing prompt text continues to instruct agents to run Ralph; only the enforcement is new.

## Out of scope

- Tier 1 agents are not part of this change. Tier 1 turns are short-lived in-process goroutines and the value of a Ralph loop there is different.
- Refactoring the prompt text itself.

## References

- Architecture mapping: `architecture-flows.html` flow `runner-poll`, step `runner_proc → ralph_loop` marked `prompt-only`.
- Finding F-Ralph surfaced in `flows-explainer.html`.
- Coord entry: `act-coordination.json` 2026-05-19T05:50:01Z.
