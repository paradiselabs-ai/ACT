package models

const (
	ProviderOpenRouter ModelProvider = "openrouter"

	OpenRouterGPT41          ModelID = "openrouter.gpt-4.1"
	OpenRouterGPT41Mini      ModelID = "openrouter.gpt-4.1-mini"
	OpenRouterGPT41Nano      ModelID = "openrouter.gpt-4.1-nano"
	OpenRouterGPT45Preview   ModelID = "openrouter.gpt-4.5-preview"
	OpenRouterGPT4o          ModelID = "openrouter.gpt-4o"
	OpenRouterGPT4oMini      ModelID = "openrouter.gpt-4o-mini"
	OpenRouterO1             ModelID = "openrouter.o1"
	OpenRouterO1Pro          ModelID = "openrouter.o1-pro"
	OpenRouterO1Mini         ModelID = "openrouter.o1-mini"
	OpenRouterO3             ModelID = "openrouter.o3"
	OpenRouterO3Mini         ModelID = "openrouter.o3-mini"
	OpenRouterO4Mini         ModelID = "openrouter.o4-mini"
	OpenRouterGemini25Flash  ModelID = "openrouter.gemini-2.5-flash"
	OpenRouterGemini25       ModelID = "openrouter.gemini-2.5"
	OpenRouterClaude35Sonnet ModelID = "openrouter.claude-3.5-sonnet"
	OpenRouterClaude3Haiku   ModelID = "openrouter.claude-3-haiku"
	OpenRouterClaude37Sonnet ModelID = "openrouter.claude-3.7-sonnet"
	OpenRouterClaude35Haiku  ModelID = "openrouter.claude-3.5-haiku"
	OpenRouterClaude3Opus    ModelID = "openrouter.claude-3-opus"
	OpenRouterNemotron3SuperFree    ModelID = "openrouter.nemotron-3-super-free"
	OpenRouterGPTOSS120BFree       ModelID = "openrouter.gpt-oss-120b-free"
	OpenRouterQwen3Next80BFree     ModelID = "openrouter.qwen3-next-80b-free"
	OpenRouterMiniMaxM25Free       ModelID = "openrouter.minimax-m2.5-free"
	OpenRouterLlama33_70BFree      ModelID = "openrouter.llama-3.3-70b-free"
	OpenRouterGemma3_27BFree       ModelID = "openrouter.gemma-3-27b-free"
	OpenRouterQwen3CoderFree       ModelID = "openrouter.qwen3-coder-free"
)

var OpenRouterModels = map[ModelID]Model{
	OpenRouterGPT41: {
		ID:                 OpenRouterGPT41,
		Name:               "OpenRouter – GPT 4.1",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/gpt-4.1",
		CostPer1MIn:        OpenAIModels[GPT41].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[GPT41].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[GPT41].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[GPT41].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[GPT41].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[GPT41].DefaultMaxTokens,
	},
	OpenRouterGPT41Mini: {
		ID:                 OpenRouterGPT41Mini,
		Name:               "OpenRouter – GPT 4.1 mini",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/gpt-4.1-mini",
		CostPer1MIn:        OpenAIModels[GPT41Mini].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[GPT41Mini].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[GPT41Mini].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[GPT41Mini].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[GPT41Mini].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[GPT41Mini].DefaultMaxTokens,
	},
	OpenRouterGPT41Nano: {
		ID:                 OpenRouterGPT41Nano,
		Name:               "OpenRouter – GPT 4.1 nano",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/gpt-4.1-nano",
		CostPer1MIn:        OpenAIModels[GPT41Nano].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[GPT41Nano].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[GPT41Nano].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[GPT41Nano].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[GPT41Nano].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[GPT41Nano].DefaultMaxTokens,
	},
	OpenRouterGPT45Preview: {
		ID:                 OpenRouterGPT45Preview,
		Name:               "OpenRouter – GPT 4.5 preview",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/gpt-4.5-preview",
		CostPer1MIn:        OpenAIModels[GPT45Preview].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[GPT45Preview].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[GPT45Preview].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[GPT45Preview].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[GPT45Preview].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[GPT45Preview].DefaultMaxTokens,
	},
	OpenRouterGPT4o: {
		ID:                 OpenRouterGPT4o,
		Name:               "OpenRouter – GPT 4o",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/gpt-4o",
		CostPer1MIn:        OpenAIModels[GPT4o].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[GPT4o].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[GPT4o].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[GPT4o].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[GPT4o].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[GPT4o].DefaultMaxTokens,
	},
	OpenRouterGPT4oMini: {
		ID:                 OpenRouterGPT4oMini,
		Name:               "OpenRouter – GPT 4o mini",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/gpt-4o-mini",
		CostPer1MIn:        OpenAIModels[GPT4oMini].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[GPT4oMini].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[GPT4oMini].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[GPT4oMini].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[GPT4oMini].ContextWindow,
	},
	OpenRouterO1: {
		ID:                 OpenRouterO1,
		Name:               "OpenRouter – O1",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/o1",
		CostPer1MIn:        OpenAIModels[O1].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[O1].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[O1].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[O1].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[O1].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[O1].DefaultMaxTokens,
		CanReason:          OpenAIModels[O1].CanReason,
	},
	OpenRouterO1Pro: {
		ID:                 OpenRouterO1Pro,
		Name:               "OpenRouter – o1 pro",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/o1-pro",
		CostPer1MIn:        OpenAIModels[O1Pro].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[O1Pro].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[O1Pro].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[O1Pro].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[O1Pro].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[O1Pro].DefaultMaxTokens,
		CanReason:          OpenAIModels[O1Pro].CanReason,
	},
	OpenRouterO1Mini: {
		ID:                 OpenRouterO1Mini,
		Name:               "OpenRouter – o1 mini",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/o1-mini",
		CostPer1MIn:        OpenAIModels[O1Mini].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[O1Mini].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[O1Mini].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[O1Mini].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[O1Mini].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[O1Mini].DefaultMaxTokens,
		CanReason:          OpenAIModels[O1Mini].CanReason,
	},
	OpenRouterO3: {
		ID:                 OpenRouterO3,
		Name:               "OpenRouter – o3",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/o3",
		CostPer1MIn:        OpenAIModels[O3].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[O3].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[O3].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[O3].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[O3].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[O3].DefaultMaxTokens,
		CanReason:          OpenAIModels[O3].CanReason,
	},
	OpenRouterO3Mini: {
		ID:                 OpenRouterO3Mini,
		Name:               "OpenRouter – o3 mini",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/o3-mini-high",
		CostPer1MIn:        OpenAIModels[O3Mini].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[O3Mini].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[O3Mini].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[O3Mini].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[O3Mini].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[O3Mini].DefaultMaxTokens,
		CanReason:          OpenAIModels[O3Mini].CanReason,
	},
	OpenRouterO4Mini: {
		ID:                 OpenRouterO4Mini,
		Name:               "OpenRouter – o4 mini",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/o4-mini-high",
		CostPer1MIn:        OpenAIModels[O4Mini].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[O4Mini].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[O4Mini].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[O4Mini].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[O4Mini].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[O4Mini].DefaultMaxTokens,
		CanReason:          OpenAIModels[O4Mini].CanReason,
	},
	OpenRouterGemini25Flash: {
		ID:                 OpenRouterGemini25Flash,
		Name:               "OpenRouter – Gemini 2.5 Flash",
		Provider:           ProviderOpenRouter,
		APIModel:           "google/gemini-2.5-flash-preview:thinking",
		CostPer1MIn:        GeminiModels[Gemini25Flash].CostPer1MIn,
		CostPer1MInCached:  GeminiModels[Gemini25Flash].CostPer1MInCached,
		CostPer1MOut:       GeminiModels[Gemini25Flash].CostPer1MOut,
		CostPer1MOutCached: GeminiModels[Gemini25Flash].CostPer1MOutCached,
		ContextWindow:      GeminiModels[Gemini25Flash].ContextWindow,
		DefaultMaxTokens:   GeminiModels[Gemini25Flash].DefaultMaxTokens,
	},
	OpenRouterGemini25: {
		ID:                 OpenRouterGemini25,
		Name:               "OpenRouter – Gemini 2.5 Pro",
		Provider:           ProviderOpenRouter,
		APIModel:           "google/gemini-2.5-pro-preview-03-25",
		CostPer1MIn:        GeminiModels[Gemini25].CostPer1MIn,
		CostPer1MInCached:  GeminiModels[Gemini25].CostPer1MInCached,
		CostPer1MOut:       GeminiModels[Gemini25].CostPer1MOut,
		CostPer1MOutCached: GeminiModels[Gemini25].CostPer1MOutCached,
		ContextWindow:      GeminiModels[Gemini25].ContextWindow,
		DefaultMaxTokens:   GeminiModels[Gemini25].DefaultMaxTokens,
	},
	OpenRouterClaude35Sonnet: {
		ID:                 OpenRouterClaude35Sonnet,
		Name:               "OpenRouter – Claude 3.5 Sonnet",
		Provider:           ProviderOpenRouter,
		APIModel:           "anthropic/claude-3.5-sonnet",
		CostPer1MIn:        AnthropicModels[Claude35Sonnet].CostPer1MIn,
		CostPer1MInCached:  AnthropicModels[Claude35Sonnet].CostPer1MInCached,
		CostPer1MOut:       AnthropicModels[Claude35Sonnet].CostPer1MOut,
		CostPer1MOutCached: AnthropicModels[Claude35Sonnet].CostPer1MOutCached,
		ContextWindow:      AnthropicModels[Claude35Sonnet].ContextWindow,
		DefaultMaxTokens:   AnthropicModels[Claude35Sonnet].DefaultMaxTokens,
	},
	OpenRouterClaude3Haiku: {
		ID:                 OpenRouterClaude3Haiku,
		Name:               "OpenRouter – Claude 3 Haiku",
		Provider:           ProviderOpenRouter,
		APIModel:           "anthropic/claude-3-haiku",
		CostPer1MIn:        AnthropicModels[Claude3Haiku].CostPer1MIn,
		CostPer1MInCached:  AnthropicModels[Claude3Haiku].CostPer1MInCached,
		CostPer1MOut:       AnthropicModels[Claude3Haiku].CostPer1MOut,
		CostPer1MOutCached: AnthropicModels[Claude3Haiku].CostPer1MOutCached,
		ContextWindow:      AnthropicModels[Claude3Haiku].ContextWindow,
		DefaultMaxTokens:   AnthropicModels[Claude3Haiku].DefaultMaxTokens,
	},
	OpenRouterClaude37Sonnet: {
		ID:                 OpenRouterClaude37Sonnet,
		Name:               "OpenRouter – Claude 3.7 Sonnet",
		Provider:           ProviderOpenRouter,
		APIModel:           "anthropic/claude-3.7-sonnet",
		CostPer1MIn:        AnthropicModels[Claude37Sonnet].CostPer1MIn,
		CostPer1MInCached:  AnthropicModels[Claude37Sonnet].CostPer1MInCached,
		CostPer1MOut:       AnthropicModels[Claude37Sonnet].CostPer1MOut,
		CostPer1MOutCached: AnthropicModels[Claude37Sonnet].CostPer1MOutCached,
		ContextWindow:      AnthropicModels[Claude37Sonnet].ContextWindow,
		DefaultMaxTokens:   AnthropicModels[Claude37Sonnet].DefaultMaxTokens,
		CanReason:          AnthropicModels[Claude37Sonnet].CanReason,
	},
	OpenRouterClaude35Haiku: {
		ID:                 OpenRouterClaude35Haiku,
		Name:               "OpenRouter – Claude 3.5 Haiku",
		Provider:           ProviderOpenRouter,
		APIModel:           "anthropic/claude-3.5-haiku",
		CostPer1MIn:        AnthropicModels[Claude35Haiku].CostPer1MIn,
		CostPer1MInCached:  AnthropicModels[Claude35Haiku].CostPer1MInCached,
		CostPer1MOut:       AnthropicModels[Claude35Haiku].CostPer1MOut,
		CostPer1MOutCached: AnthropicModels[Claude35Haiku].CostPer1MOutCached,
		ContextWindow:      AnthropicModels[Claude35Haiku].ContextWindow,
		DefaultMaxTokens:   AnthropicModels[Claude35Haiku].DefaultMaxTokens,
	},
	OpenRouterClaude3Opus: {
		ID:                 OpenRouterClaude3Opus,
		Name:               "OpenRouter – Claude 3 Opus",
		Provider:           ProviderOpenRouter,
		APIModel:           "anthropic/claude-3-opus",
		CostPer1MIn:        AnthropicModels[Claude3Opus].CostPer1MIn,
		CostPer1MInCached:  AnthropicModels[Claude3Opus].CostPer1MInCached,
		CostPer1MOut:       AnthropicModels[Claude3Opus].CostPer1MOut,
		CostPer1MOutCached: AnthropicModels[Claude3Opus].CostPer1MOutCached,
		ContextWindow:      AnthropicModels[Claude3Opus].ContextWindow,
		DefaultMaxTokens:   AnthropicModels[Claude3Opus].DefaultMaxTokens,
	},

	// NVIDIA Nemotron 3 Super 120B — biggest free model on OpenRouter, hybrid
	// Mamba-Transformer, 262K context. Hosted on Venice (free tier). Supports
	// tool calls — confirmed working as of 2026-04. Different upstream provider
	// from OpenInference, so good for load-balancing across 4 Tier 1 agents.
	OpenRouterNemotron3SuperFree: {
		ID:                 OpenRouterNemotron3SuperFree,
		Name:               "OpenRouter – Nemotron 3 Super 120B Free",
		Provider:           ProviderOpenRouter,
		APIModel:           "nvidia/nemotron-3-super-120b-a12b:free",
		CostPer1MIn:        0,
		CostPer1MInCached:  0,
		CostPer1MOut:       0,
		CostPer1MOutCached: 0,
		ContextWindow:      262_144,
		DefaultMaxTokens:   8000,
	},

	// OpenAI OSS 120B — OpenAI's open-source 120B model served free via
	// OpenInference. Supports tool calls. Different upstream provider from
	// Venice (Nemotron), so using both distributes rate-limit pressure across
	// two independent free-tier quotas.
	OpenRouterGPTOSS120BFree: {
		ID:                 OpenRouterGPTOSS120BFree,
		Name:               "OpenRouter – OpenAI OSS 120B Free",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/gpt-oss-120b:free",
		CostPer1MIn:        0,
		CostPer1MInCached:  0,
		CostPer1MOut:       0,
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   8000,
	},

	// Llama 3.3 70B Instruct — proven, well-tested instruction follower.
	// Context reduced to 65K on OpenRouter free tier (was 131K).
	OpenRouterLlama33_70BFree: {
		ID:                 OpenRouterLlama33_70BFree,
		Name:               "OpenRouter – Llama 3.3 70B Free",
		Provider:           ProviderOpenRouter,
		APIModel:           "meta-llama/llama-3.3-70b-instruct:free",
		CostPer1MIn:        0,
		CostPer1MInCached:  0,
		CostPer1MOut:       0,
		CostPer1MOutCached: 0,
		ContextWindow:      65_536,
		DefaultMaxTokens:   5000,
	},

	// Gemma 3 27B — smaller / faster than 70B class, plenty for the Observer
	// role which only summarizes status snapshots into short messages.
	// Burns less of the 20-RPM free-tier budget per call.
	OpenRouterGemma3_27BFree: {
		ID:                 OpenRouterGemma3_27BFree,
		Name:               "OpenRouter – Gemma 3 27B Free",
		Provider:           ProviderOpenRouter,
		APIModel:           "google/gemma-3-27b-it:free",
		CostPer1MIn:        0,
		CostPer1MInCached:  0,
		CostPer1MOut:       0,
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   3000,
	},

	// Qwen3 Coder 480B A35B — coding-focused MoE, 262K context. Replaces
	// Qwen 2.5 72B (removed from free tier). Strong at structured output
	// and code validation — good fit for Assurance.
	OpenRouterQwen3CoderFree: {
		ID:                 OpenRouterQwen3CoderFree,
		Name:               "OpenRouter – Qwen3 Coder 480B Free",
		Provider:           ProviderOpenRouter,
		APIModel:           "qwen/qwen3-coder:free",
		CostPer1MIn:        0,
		CostPer1MInCached:  0,
		CostPer1MOut:       0,
		CostPer1MOutCached: 0,
		ContextWindow:      262_000,
		DefaultMaxTokens:   5000,
	},

	// Qwen3 Next 80B A3B — MoE instruct model via Venice free tier. Good
	// balance of size/quality for monitoring and validation roles. Same
	// upstream provider (Venice) as Nemotron — rate-limit pressure is shared
	// but the free quotas reset frequently.
	OpenRouterQwen3Next80BFree: {
		ID:                 OpenRouterQwen3Next80BFree,
		Name:               "OpenRouter – Qwen3 Next 80B Free",
		Provider:           ProviderOpenRouter,
		APIModel:           "qwen/qwen3-next-80b-a3b-instruct:free",
		CostPer1MIn:        0,
		CostPer1MInCached:  0,
		CostPer1MOut:       0,
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   5000,
	},

	// MiniMax M2.5 — MiniMax's large free model via OpenInference. Different
	// upstream provider from Venice. Useful when Venice models are saturated.
	OpenRouterMiniMaxM25Free: {
		ID:                 OpenRouterMiniMaxM25Free,
		Name:               "OpenRouter – MiniMax M2.5 Free",
		Provider:           ProviderOpenRouter,
		APIModel:           "minimax/minimax-m2.5:free",
		CostPer1MIn:        0,
		CostPer1MInCached:  0,
		CostPer1MOut:       0,
		CostPer1MOutCached: 0,
		ContextWindow:      1_000_000,
		DefaultMaxTokens:   5000,
	},
}
