---
id: "tier1-agents-md-cached-pre-intake-2026-05-19"
status: "in-progress"
priority: "high"
assignee: "d34d"
dueDate: null
created: "2026-05-19T00:00:00.000Z"
modified: "2026-05-19T00:00:00.000Z"
completedAt: null
labels: ["orchestrator", "prompts", "bug", "tier1"]
order: "a0"
---
# Tier 1 system prompts cached pre-INTAKE — AGENTS.md never reaches Observer/Assurance/QA in the same session

## Symptom (corrected scope)

In the session where INTAKE happens and `PROJECT_BRIEF:` is first parsed, three of the four Tier 1 agents (Observer, Assurance, QA) never see the AGENTS.md content the Planner just authored. The Planner itself is **not** symptomatic — its INTAKE conversation IS the brief, and that conversation lives in its message thread. Ask the Planner "what is this project" mid-session and it answers from history.

The other three Tier 1 agents were not party to INTAKE. They were supposed to receive project context via AGENTS.md auto-injection. They don't, because their system prompts were frozen before AGENTS.md existed.

Edge case where Planner also bites: after context compaction, if the summary loses brief fields, AGENTS.md would have been the safety net — and isn't loaded for Tier 1.

## Root cause

Two interacting design points:

1. `GetAgentPrompt` is invoked **once per agent at construction time** (`act-agent/internal/llm/agent/agent.go:730`), not per-turn. The result is passed to `provider.WithSystemMessage(...)` and frozen into the provider client.
2. `getContextFromPaths()` (`act-agent/internal/llm/prompt/prompt.go:62`) uses `sync.Once` to cache the context content for the lifetime of the process.

Timeline for a fresh project:
- T+0: TUI launches → `app.go::New()` constructs 4 Tier 1 agents → each calls `GetAgentPrompt` → `sync.Once` fires → AGENTS.md does not exist yet → empty context string → baked into system message.
- T+30s: User finishes INTAKE → Planner emits `PROJECT_BRIEF:` → `orchestrator.go:956` calls `writeAgentsMd` → AGENTS.md + CLAUDE.md materialize on disk.
- T+31s+: Every subsequent Tier 1 turn uses the system message frozen at T+0. The `sync.Once` cache still holds the empty string. AGENTS.md is invisible to Tier 1 until the user restarts `act`.

Tier 2 swarm agents are unaffected — each task spawn is a separate subprocess with its own `sync.Once`. They read AGENTS.md fresh on construction, which by then exists.

## Fix options

### (a) Rebuild Tier 1 system prompts on PROJECT_BRIEF parse
After `writeAgentsMd` succeeds in `orchestrator.go`, reset the `sync.Once` (or expose an invalidation hook on the `prompt` package), then reconstruct the four Tier 1 agents. Cheapest in tokens. **Cost:** breaks any provider-side prompt caching mid-session for Tier 1 — the next turn after INTAKE pays full prompt-input price. Acceptable, since that turn is going to be the first `CREATE_TASK` and the conversation has already drifted from any cached prefix.

### (b) Move AGENTS.md out of the system message, inject per-turn
Strip AGENTS.md from `getContextFromPaths` for Tier 1. Inject the rendered AGENTS.md content into each Tier 1 turn as a leading user/system message instead. Always-fresh, but costs ~1-2K extra tokens **every turn** for the entire session lifetime.

### (c) Bake-once-with-late-binding
Keep system message as-is but on PROJECT_BRIEF parse, send a one-shot system message to each Tier 1 agent ("Project brief updated: <inline AGENTS.md>") that they treat as the canonical context going forward. Cheapest in steady state, ugliest in protocol — Tier 1 prompts need to mention this is possible.

## Recommendation

Option (a). The cache invalidation is bounded (fires once per session per PROJECT_BRIEF event, which is typically exactly once). The token cost is a single re-prefill of four agents, not a per-turn overhead. (b) is wrong for the Phase 3 token diet goals. (c) muddles the prompt protocol.

## Implementation (landed 2026-05-19)

- `act-agent/internal/llm/prompt/prompt.go` — replaced `sync.Once` with `sync.Mutex` + `bool` guard; added `InvalidateContextCache()` that clears both. Next `getContextFromPaths` call re-reads every contextPath file.
- `act-agent/internal/llm/agent/agent.go` — stored `agentName` on the `agent` struct; added `RebindSystemPrompt()` to the `Service` interface + `*agent` impl. Reuses `createAgentProvider(agentName)` (same path `Update` uses) to build a fresh provider with the refreshed system message. Idle-check via `IsBusy()` to avoid stomping an in-flight turn.
- `act-agent/internal/app/orchestrator.go` — after `writeAgentsMd` succeeds, call `prompt.InvalidateContextCache()` then iterate `{planner, observer, assurance, qa_synthesizer}` and call `RebindSystemPrompt()` on each. Failures are logged, not fatal.

Total: ~50 lines across 3 files. No provider-level changes — `createAgentProvider` already does the right thing once the prompt cache is invalidated.

## Verification

E2E test: launch `act` against a fresh project name, run INTAKE, send a follow-up message that should reference the brief (e.g., "what tech stack are we using on this project?"). Without the fix, Planner replies vaguely (it's reading the user's recent conversation, not AGENTS.md). With the fix, Planner replies with the exact tech stack from the brief. Confirm by inspecting the Planner's last LLM request payload — system message should contain the AGENTS.md content.

## Related

- Originally surfaced during contextPaths Q&A on 2026-05-19. ACT.md and ACT.local.md have the same caching issue but matter less since they're user-authored and stable across restarts.
- `compaction-all-tier1-agents-not-just-planner-2026-05-12.md` touches the same Tier 1 agent map but is a different bug (post-construction state drift, not pre-construction file timing).
