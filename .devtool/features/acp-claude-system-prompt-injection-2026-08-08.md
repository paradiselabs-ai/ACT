---
id: "acp-claude-system-prompt-injection-2026-08-08"
status: "in-progress"
priority: "high"
assignee: null
dueDate: null
created: "2026-08-08T09:40:00.000Z"
modified: "2026-08-08T09:40:00.000Z"
completedAt: null
labels: ["acp", "prompts", "tier1", "bug"]
order: "a0"
---
# Claude-code Tier 1 roles ignore user-message priming — inject role prompt as real system prompt

## Describe
ACP has no system channel, so Tier-1 role prompts are injected as a first user
message ("priming") with a do-not-respond header. For claude-code this is too weak:
Claude Code's own system prompt (clarify, propose options, act like a coding CLI)
outranks a user message. Live evidence (2026-08-08 linkdocker run, fresh binary,
priming completed cleanly): claude-Planner ignored the new intake script — no
description-depth question, jumped to design consulting, offered SQLite right after
the human said "no database".

The claude ACP bridge (@agentclientprotocol/claude-agent-acp 0.37, verified in
dist/acp-agent.js ~L1479-1495) accepts `_meta.systemPrompt` on session/new:
an object form is spread with type/preset locked to claude_code, forwarding
`append` — i.e. `_meta: {"systemPrompt": {"append": "<role prompt>"}}` appends our
text to Claude Code's REAL system prompt.

## Success Criteria
- For the claude-code backend only: session/new carries
  `_meta.systemPrompt.append` = the full role payload (role prompt + shim
  discoverability note — same content wrapPriming assembles, WITHOUT the
  InternalPromptMarker and do-not-respond header, which exist only because priming
  was a user message).
- The user-message priming turn is SKIPPED for claude-code (one fewer API call per
  role at startup). gemini/antigravity/agy priming paths byte-identical to before.
- Unit test asserting: claude-code session/new params include the append text and
  no priming turn is issued; gemini path unchanged (still primes, no _meta).
- go build clean; go vet clean; full ./... test suite green.
- Live e2e (next run): claude-Planner's first intake move on a thin opener is the
  description-depth question. Evidence quoted here before done.

## Constraints
- Touch only `internal/acp/` (+ its tests). No orchestrator changes; no prompt-text
  changes; no bridge-package pinning changes.
- Do not alter the priming content builders in app.go — the acp layer decides
  transport (system-append vs priming turn) per backend.

## Invariants (code-level)
- Non-claude backends: zero behavior change (their bridges may ignore or choke on
  unknown _meta — do not send it to them).
- InternalPromptMarker never appears inside the system-prompt append.
