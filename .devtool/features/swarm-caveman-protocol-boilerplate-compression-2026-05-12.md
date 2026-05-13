---
id: "swarm-caveman-protocol-boilerplate-compression-2026-05-12"
status: "backlog"
priority: "medium"
assignee: "d34d"
epic: "swarm-context"
dueDate: null
created: "2026-05-12T00:00:00.000Z"
modified: "2026-05-12T00:00:00.000Z"
completedAt: null
labels: ["swarm", "context", "tokens", "prompt-engineering"]
order: "b3"
---
# Caveman protocol for swarm boilerplate — compress prompts via glossary

Swarm Tier-2 prompts are token-heavy. Most of the bloat is BOILERPLATE — `act_cli` call shapes, role-prelude, environmental info, Ralph Wiggum self-check instructions, success-criteria framing. Repeated verbatim across `developer.go`, `frontend_dev.go`, `backend_dev.go`, `qa_engineer.go`, `researcher.go`.

**Idea:** caveman-compress the boilerplate while keeping task-specific directives in full English.

Define a glossary at the top of the brief's read-only block (see `swarm-next-task-preamble-readonly-brief-2026-05-12`):
```
GLOSSARY
AC = @success_criteria
RW = Ralph Wiggum self-check loop
BU = act-agent brief update
CT = act-agent task complete
SV = act-agent task submit-for-validation
DR = act-agent decision record
```

Use codes in repeated boilerplate, expand in directives. Example boilerplate transform:
- Before (~80 tokens): "Before calling `act-agent task complete`, run a Ralph Wiggum self-check: re-read your `@success_criteria`, verify each one is met, fix gaps, then submit for validation via `act-agent task submit-for-validation`."
- After (~25 tokens): "Pre-CT: RW vs AC, fix gaps, SV."

**Risk:** free-tier swarm models (the cheapest configs) are the ones most likely to fall apart on compressed instructions. Mitigation:
- Glossary lives in the read-only brief block (can't be deleted by agent).
- Only compress BOILERPLATE. Task descriptions stay verbose.
- A/B test on free-tier models before rollout.

**Files:**
- `act-agent/internal/llm/prompt/common.go` (shared boilerplate snippets).
- `act-agent/internal/llm/prompt/{developer,frontend_dev,backend_dev,qa_engineer,researcher}.go` (per-role specifics).
- `act-agent/internal/app/agents_md.go` (writeAgentsMd includes glossary in read-only section).

**Depends on:** `swarm-next-task-preamble-readonly-brief-2026-05-12` (read-only block is where the glossary lives).

**Success criteria:**
- Measurable token reduction in average swarm prompt (target: -25% to -40% on boilerplate sections).
- No regression in task completion rate on free-tier models (A/B vs current prompts on the same project seed).
- Glossary visible at top of every spawn's brief.
- Build + vet clean.
