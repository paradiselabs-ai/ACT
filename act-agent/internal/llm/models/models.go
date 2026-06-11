// Package models contains LLM provider+model identifiers.
//
// ACT does NOT maintain a registry of "supported" models. The user writes
// the upstream provider's model string verbatim in ~/.act.json under
// agents.<role>.model, along with agents.<role>.provider. ACT routes by
// provider name and passes the model string through to the upstream API.
// New models become "supported" by writing the string in config — no code
// change required.
package models

type (
	// ModelID is the upstream model string passed verbatim to the provider
	// (e.g. "claude-sonnet-4-20250514", "z-ai/glm-4.5-air:free",
	// "qwen2.5-coder-14b-instruct").
	ModelID string

	// ModelProvider identifies which provider client routes the request.
	ModelProvider string
)

// Model is a thin per-turn descriptor passed to provider clients. It
// carries only the data the client needs to make a request: the upstream
// model string, which provider, and the per-turn maxTokens budget.
type Model struct {
	ID        ModelID       `json:"id"`
	Provider  ModelProvider `json:"provider"`
	MaxTokens int64         `json:"maxTokens,omitempty"`
}

// Provider identifiers. Each value matches a key under "providers" in
// ~/.act.json and a case in provider.NewProvider.
const (
	ProviderAnthropic  ModelProvider = "anthropic"
	ProviderOpenAI     ModelProvider = "openai"
	ProviderGemini     ModelProvider = "gemini"
	ProviderGROQ       ModelProvider = "groq"
	ProviderNVIDIA     ModelProvider = "nvidia"
	ProviderOpenRouter ModelProvider = "openrouter"
	ProviderXAI        ModelProvider = "xai"
	ProviderAzure      ModelProvider = "azure"
	ProviderVertexAI   ModelProvider = "vertexai"
	ProviderBedrock    ModelProvider = "bedrock"
	ProviderLocal      ModelProvider = "local"
	ProviderMock       ModelProvider = "__mock"

	// ProviderACP is the synthetic provider identifier for ACP-backed Tier 1
	// agents. It is not a real LLM provider — no case in provider.NewProvider —
	// but the prompt dispatcher branches on it so a role's system prompt can be
	// rendered backend-accurately (ACP uses the act-tier1-* shim via Bash; the
	// in-process backend uses the native act_cli JSON tool). Lives here, not in
	// the acp package, so the prompt package can compare against it without an
	// import cycle (acp imports prompt).
	ProviderACP ModelProvider = "acp"
)
