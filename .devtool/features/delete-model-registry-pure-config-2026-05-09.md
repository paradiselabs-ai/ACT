---
id: "delete-model-registry-pure-config-2026-05-09"
status: "todo"
priority: "high"
assignee: "d34d"
dueDate: null
created: "2026-05-09T10:00:00.000Z"
modified: "2026-05-09T10:00:00.000Z"
completedAt: null
labels: ["refactor", "architecture", "alpha-priority", "config", "ux"]
order: "a0"
---
# Delete the model registry; provider+model are pure config

## Problem

ACT's model registry (`act-agent/internal/llm/models/*.go`) is architecturally indefensible. It maintains hardcoded entries for every "supported" model across every provider — a list that goes stale weekly. Users who write the actual upstream model string in `~/.act.json` get reverse-lookup-mapped to a synthetic ID; if their model isn't registered, ACT silently reverts to a default and warns. New models = code change. LM Studio models = require env-var discovery hack. This is the passport-registered-under-an-entity-name absurdity.

The right design: provider + model are pure config. ACT routes to the provider's base URL with the user's API key and the user's model string. The upstream API validates the model. ACT validates nothing about the model — provider config validity is enough.

## What gets deleted

- `act-agent/internal/llm/models/anthropic.go` — model list
- `act-agent/internal/llm/models/openai.go` — model list
- `act-agent/internal/llm/models/openrouter.go` — model list
- `act-agent/internal/llm/models/groq.go` — model list
- `act-agent/internal/llm/models/azure.go` — model list
- `act-agent/internal/llm/models/gemini.go` — model list
- `act-agent/internal/llm/models/vertexai.go` — model list
- `act-agent/internal/llm/models/xai.go` — model list
- `act-agent/internal/llm/models/local.go::init()` — env-var-driven model discovery hack
- `models.SupportedModels` map
- `models.ProviderPopularity` map
- `config.resolveModelAlias()` — reverse lookup
- `config.validateAgent()`'s registry-based model check (lines ~510-565)
- `models.Model` struct's `APIModel`, `CostPer1MIn`, `CostPer1MInCached`, `CostPer1MOut`, `CostPer1MOutCached`, `CanReason`, `ToolsUnsupported`, `Name` fields

## What stays

- `models.ModelProvider` enum (`anthropic`, `openai`, `openrouter`, `groq`, `local`, etc.)
- `models.ModelID` type alias = `string`
- A slim `models.Model` struct with `Provider`, `MaxTokens` only — used internally for the per-turn config object
- `provider.NewProvider()` switch case routing to per-provider clients

## What gets added

### 1. `Provider` field on `Agent` struct

```go
type Agent struct {
    Provider        models.ModelProvider `json:"provider"`            // NEW — required
    Model           string               `json:"model"`               // was ModelID, now plain string
    MaxTokens       int64                `json:"maxTokens"`
    ReasoningEffort string               `json:"reasoningEffort,omitempty"`
    Backend         string               `json:"backend,omitempty"`
}
```

### 2. `BaseURL` field on `Provider` struct

```go
type Provider struct {
    APIKey   string `json:"apiKey"`
    BaseURL  string `json:"baseURL,omitempty"`  // NEW — overrides hardcoded provider URL
    Disabled bool   `json:"disabled"`
}
```

Each provider in `provider.go::NewProvider()` already has a hardcoded base URL (e.g., `https://api.groq.com/openai/v1`). The `baseURL` field overrides that — useful for local, self-hosted endpoints, custom proxies.

### 3. New `validateAgent` (much shorter)

```go
func validateAgent(cfg *Config, name AgentName, agent Agent) error {
    if agent.Backend == "claude-code" {
        return nil  // claude-code uses its own config
    }
    if agent.Provider == "" {
        return fmt.Errorf("agent %s missing provider field", name)
    }
    if agent.Model == "" {
        return fmt.Errorf("agent %s missing model field", name)
    }
    providerCfg, ok := cfg.Providers[agent.Provider]
    if !ok || providerCfg.Disabled {
        return fmt.Errorf("agent %s configured for provider %s which is not configured/enabled", name, agent.Provider)
    }
    if agent.Provider != models.ProviderLocal && providerCfg.APIKey == "" {
        return fmt.Errorf("provider %s missing apiKey", agent.Provider)
    }
    return nil
}
```

### 4. `provider.NewProvider()` reads `BaseURL` from config

Already partially done in commit `762fa51` for Local. Extend to all providers — let user override base URL for any of them via `providers.<name>.baseURL`.

### 5. Tool-call probe on first turn (replaces `ToolsUnsupported`)

When a Tier 1 agent makes its first tool call, if the upstream API returns "model does not support tools," ACT emits a clear error in the chat:

> ⚠️ Model `<model-id>` doesn't support tool calling, which is required for the {Planner|Observer|Assurance|QA} role. Edit `~/.act.json` and pick a tool-calling model. See https://act.example/models for known-good options.

Better than current behavior (silent revert to default with warning logged to file).

### 6. Cost tracking from API response

Anthropic returns usage in response. OpenAI returns usage. Anthropic returns input + output token counts. Use those — don't precompute from a hardcoded cost table. Format: read from response, multiply by per-provider rate (a tiny lookup, NOT per-model). For free providers, cost is $0.

If providers don't return usage (rare), display token count without cost.

## Migration / backward compat

User configs that omit `provider` will fail loudly with a clear error message: "agent X missing provider field — see migration guide." Provide a one-time migration helper:

```bash
act-agent migrate-config
```

Reads existing `~/.act.json`, infers provider from the model string using the OLD registry one last time (i.e., the migration helper still has the registry temporarily), writes the new config with explicit `provider:` fields. Then user deletes the migration helper or it auto-deletes on next run.

Or simpler: just print clear errors and let user manually update. The pre-alpha phase has small enough user count.

## TUI dialog model picker

`internal/tui/components/dialog/models.go` currently lists models from `SupportedModels`. Post-refactor: hits `<provider-baseURL>/v1/models` for the selected provider's currently-loaded models (LM Studio for local; OpenRouter has a public catalog endpoint; Anthropic has a list endpoint). Truly current, never stale.

## Out of scope (future work)

- Dynamic model rotation per task type via PVM evidence (the user's stated next step). Becomes trivial once registry is gone — PVM tracks model-string + task-type + outcome, picks the historically-best at decomposition time.
- Streaming/non-streaming preference per provider. Already handled per-client.

## Constraints

- Project-owner domain (cmd, internal/llm, internal/config). Not Kareem's.
- Big refactor — should be its own PR, no other work bundled.
- Will require updating examples in `.act.example.json`, README, CLAUDE.md.
- Tests: rewrite `config_test.go` for the new validateAgent shape; remove tests asserting registry presence.

## Success criteria

1. All 8 model-list files (`anthropic.go`, `openai.go`, etc.) deleted.
2. `validateAgent` is <50 LOC.
3. `~/.act.json` requires `provider` field on every agent block; ACT errors clearly if missing.
4. New model "supported" by writing the string in config — no code change required.
5. LM Studio works zero-config: just set `providers.local.baseURL` and add the model string. No `LOCAL_ENDPOINT` env var needed.
6. Cost shown in TUI status bar comes from API response usage fields, not from hardcoded cost tables.
7. `cd act-agent && go build && go vet ./...` clean.
8. Existing test suite passes (after expected updates to model-related tests).
9. End-to-end: fresh project, INTAKE, swarm spawn, validation, synthesis — all work with no registry.

## Estimate

1-2 days focused work. ~600 LOC deleted, ~150 LOC added (mostly the new validateAgent + tool-call probe error handler).

## Priority justification

**HIGH alpha-priority.** This is the kind of architectural simplification that makes ACT readable to outside contributors and demoable without explanation. The registry is the single weirdest thing in the codebase right now. Pre-alpha is the right time to tear it out — once external users are configuring against the current schema, migration becomes painful.
