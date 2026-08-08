// Package config manages application configuration from various sources.
package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/models"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
	"github.com/spf13/viper"
)

// MCPType defines the type of MCP (Model Control Protocol) server.
type MCPType string

// Supported MCP types
const (
	MCPStdio MCPType = "stdio"
	MCPSse   MCPType = "sse"
)

// MCPServer defines the configuration for a Model Control Protocol server.
type MCPServer struct {
	Command string            `json:"command"`
	Env     []string          `json:"env"`
	Args    []string          `json:"args"`
	Type    MCPType           `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type AgentName string

// Auxiliary helper agents (session title generation, conversation
// summarization). These are not user-facing ACT roles
// and only wire up when the user explicitly configures
// agents.title / agents.summarizer in ~/.act.json. ACT
// dispatches strictly by explicit role (Planner / Observer / Assurance /
// QA / developer / frontend_dev / backend_dev / qa_engineer / researcher),
// never by a generic fallback.
const (
	AgentSummarizer AgentName = "summarizer"
	AgentTitle      AgentName = "title"
)

// ACT role names — map to agent configs for role-based model selection.
// When --role <role> is used, the system looks up agents.<role> first,
// and falls back to agents.developer. There is no other fallback —
// unconfigured roles must be routed to a real ACT role, not a generic
// catch-all.
//
// Tier 1 (Interactive — NesTTY window):
const (
	RolePlanner       AgentName = "planner"
	RoleObserver      AgentName = "observer"
	RoleAssurance     AgentName = "assurance"
	RoleQASynthesizer AgentName = "qa_synthesizer"
)

// Tier 2 (Headless — task execution):
const (
	RoleFrontendDev AgentName = "frontend_dev"
	RoleBackendDev  AgentName = "backend_dev"
	RoleQAEngineer  AgentName = "qa_engineer"
	RoleResearcher  AgentName = "researcher"
	RoleDeveloper   AgentName = "developer" // default Tier 2 role
)

// Agent defines configuration for different LLM models and their token limits.
type Agent struct {
	Provider        models.ModelProvider `json:"provider"`                  // routes to providers.<name> in ~/.act.json
	Model           models.ModelID       `json:"model"`                     // upstream model string, passed verbatim
	MaxTokens       int64                `json:"maxTokens"`
	ReasoningEffort string               `json:"reasoningEffort,omitempty"` // low|medium|high — passed through if set

	// Backend selects which CLI agent (or in-process LLM) backs this role.
	// One symmetric vocabulary across Tier 1 and Tier 2:
	//   "act-agent" (default) — in-process LLM, configured by Provider/Model above
	//   "claude-code"         — Claude Code (Tier 1 via ACP; Tier 2 via direct subprocess)
	//   "gemini"              — Gemini CLI (Tier 1 via native ACP `gemini --acp`; Tier 2 via `gemini -p`)
	//   future: "codex", "opencode" — added incrementally
	// An unset Backend means the in-process LLM path, unchanged from pre-ACP behaviour.
	//
	// The wire-level mechanism (ACP for Tier 1, direct CLI for Tier 2) is an
	// implementation detail — the user just names the agent they want.
	Backend string `json:"backend,omitempty"`

	// ACP carries optional power-user overrides for ACP-backed Tier 1 roles
	// (Command/Args/Env/Cwd for the subprocess spawn). The host is derived
	// from Backend — this struct exists only so advanced users can pin a
	// custom binary path, swap to a forked adapter, or set env vars. Empty
	// in the common case. Ignored for Tier 2.
	ACP *ACPConfig `json:"acp,omitempty"`
}

// ACPConfig — optional overrides for ACP subprocess spawn. The default spawn
// argv is derived from the Agent.Backend value (see internal/acp/claude_code.go
// for the claude-code preset). Most users leave this nil.
type ACPConfig struct {
	// Command overrides the spawn binary. If empty, derived from Backend.
	Command string `json:"command,omitempty"`

	// Args overrides the spawn argv tail. If empty (and Command is empty),
	// derived from Backend. If Command is set, Args is used verbatim — no
	// implicit defaults — so users can fully control the spawn.
	Args []string `json:"args,omitempty"`

	// Env adds environment variables to the subprocess. PATH is augmented
	// separately by app.go to expose the role's act-tier1-<role> shim.
	Env map[string]string `json:"env,omitempty"`

	// Cwd sets the subprocess working directory. Defaults to ACT's project
	// root (the directory ACT itself was launched from).
	Cwd string `json:"cwd,omitempty"`
}

// Provider defines configuration for an LLM provider. BaseURL overrides
// the hardcoded provider URL in provider.NewProvider — useful for
// self-hosted endpoints, custom proxies, and local OpenAI-compatible
// servers (LM Studio, Ollama, vLLM). For "local", BaseURL replaces the
// LM Studio default (http://localhost:1234/v1) and is the canonical place
// to point at Ollama's "http://localhost:11434/v1".
// A provider is "usable" iff it has the credentials its agent needs —
// apiKey for the cloud providers, baseURL (or LM Studio default) for local.
// We don't carry an explicit disabled flag: missing-credentials is the
// natural disable signal, and an explicit flag invited "I set apiKey but
// forgot to flip disabled=false" footguns.
type Provider struct {
	APIKey  string `json:"apiKey"`
	BaseURL string `json:"baseURL,omitempty"`
}

// Usable reports whether a provider has the credentials its API needs.
// Local talks to a self-hosted endpoint and only needs baseURL (which has a
// default). Bedrock/VertexAI authenticate via the surrounding cloud SDK —
// AWS_PROFILE / GOOGLE_APPLICATION_CREDENTIALS — and surface their own
// credential errors at request time, so they're always considered usable
// at config-load time when present in the providers map.
func (p Provider) Usable(id models.ModelProvider) bool {
	switch id {
	case models.ProviderLocal, models.ProviderBedrock, models.ProviderVertexAI:
		return true
	default:
		return p.APIKey != ""
	}
}

// AgentConfigForRole returns the agent config for an ACT role. Looks up
// agents.<role> first, falls back to agents.developer. There is no deeper
// fallback — the developer role is the project-wide default and must be
// configured. If it isn't, agent construction will fail with a clear error
// rather than silently landing on some generic catch-all.
//
// This is a MODEL/BACKEND-config lookup ONLY. NEVER use its result to pick a
// role's system prompt: the developer fallback would turn an unconfigured
// Planner into a Tier 2 swarm developer. Prompt selection must always key on
// the true role (see createAgentProvider / GetAgentPrompt).
func AgentConfigForRole(role string) AgentName {
	roleName := AgentName(role)
	cfg := Get()
	if _, ok := cfg.Agents[roleName]; ok {
		return roleName
	}
	return RoleDeveloper
}

// Tier1AgentNames returns the canonical ordered list of Tier 1 NesTTY roles
// — the 4 agents bound to the TUI window. There is no "default" agent in ACT;
// these four all share the conversation and each has its own LLM model.
func Tier1AgentNames() []AgentName {
	return []AgentName{RolePlanner, RoleObserver, RoleAssurance, RoleQASynthesizer}
}

// Tier1Configs returns the configured Agent struct for each Tier 1 role.
// Roles that aren't in the user's config are returned as zero-value Agents.
// Used by the status bar (to show all 4 models) and the model selection
// dialog (to let users pick which Tier 1 role to edit).
func Tier1Configs() map[AgentName]Agent {
	cfg := Get()
	out := make(map[AgentName]Agent, 4)
	if cfg == nil {
		return out
	}
	for _, name := range Tier1AgentNames() {
		if a, ok := cfg.Agents[name]; ok {
			out[name] = a
		}
	}
	return out
}

// Tier1ShortLabel returns a single-character abbreviation for a Tier 1 role,
// used by compact status bar displays.
func Tier1ShortLabel(name AgentName) string {
	switch name {
	case RolePlanner:
		return "P"
	case RoleObserver:
		return "O"
	case RoleAssurance:
		return "A"
	case RoleQASynthesizer:
		return "Q"
	default:
		return string(name)
	}
}

// Data defines storage configuration.
type Data struct {
	Directory string `json:"directory,omitempty"`
}

// LSPConfig defines configuration for Language Server Protocol integration.
type LSPConfig struct {
	Disabled bool     `json:"enabled"`
	Command  string   `json:"command"`
	Args     []string `json:"args"`
	Options  any      `json:"options"`
}

// TUIConfig defines the configuration for the Terminal User Interface.
type TUIConfig struct {
	Theme           string         `json:"theme,omitempty"`
	AutoFit         *AutoFitConfig `json:"autoFit,omitempty"`
	MaxMessageLines int            `json:"maxMessageLines,omitempty"` // max lines per assistant message before truncation; 0 = default (80)
}

// AutoFitConfig controls the startup terminal-window resize. On by default —
// the TUI needs ~160x58 cells to display the full AGENT/COORDINATION/TOOLKIT
// banner alongside the context navigator without clipping, and a default
// 80x24 window leaves most of it invisible. Set Disabled to skip, or tune
// MinCols/MinRows to your preferred minimum. The resize only grows the
// window (never shrinks), and is skipped in known-hostile environments
// (tmux, screen, vscode integrated terminal).
type AutoFitConfig struct {
	Disabled bool `json:"disabled,omitempty"`
	MinCols  int  `json:"minCols,omitempty"`
	MinRows  int  `json:"minRows,omitempty"`
}

// ShellConfig defines the configuration for the shell used by the bash tool.
type ShellConfig struct {
	Path string   `json:"path,omitempty"`
	Args []string `json:"args,omitempty"`
}

// Config is the main configuration structure for the application.
type Config struct {
	Data         Data                              `json:"data"`
	WorkingDir   string                            `json:"wd,omitempty"`
	MCPServers   map[string]MCPServer              `json:"mcpServers,omitempty"`
	Providers    map[models.ModelProvider]Provider `json:"providers,omitempty"`
	LSP          map[string]LSPConfig              `json:"lsp,omitempty"`
	Agents       map[AgentName]Agent               `json:"agents,omitempty"`
	Debug        bool                              `json:"debug,omitempty"`
	DebugLSP     bool                              `json:"debugLSP,omitempty"`
	ContextPaths []string                          `json:"contextPaths,omitempty"`
	TUI          TUIConfig                         `json:"tui"`
	Shell        ShellConfig                       `json:"shell,omitempty"`
	AutoCompact  bool                              `json:"autoCompact,omitempty"`
	// AutoCompactTokens is the conversation token total at which Tier 1
	// auto-compaction fires (when AutoCompact is true). This is ACT's own
	// hygiene threshold, NOT the model's context window — compaction in
	// ACT exists to keep context engineering / system prompts / project
	// data effective and prevent drift, regardless of which model is in
	// use. Whatever the conversation has accumulated still gets sent to
	// whichever model is current. Default: 120000 tokens.
	AutoCompactTokens int64 `json:"autoCompactTokens,omitempty"`
}

// Application constants
const (
	defaultDataDirectory = ".act"
	defaultLogLevel      = "info"
	appName              = "act"

	MaxTokensFallbackDefault = 4096

	// DefaultAutoCompactTokens is the conversation total above which
	// auto-compaction fires when AutoCompact is enabled. Picked to keep
	// Tier 1 prompts focused — system prompts + AGENTS.md + recent
	// turns rarely exceed this in healthy operation, so crossing it
	// signals drift worth summarizing.
	DefaultAutoCompactTokens int64 = 120000
)

// defaultContextPaths are files auto-loaded into every agent's system prompt
// at startup. AGENTS.md is the Planner-authored project brief (cross-vendor
// spec, ~1-2K tokens, regenerated from the server brief on every PROJECT_BRIEF
// parse). ACT.md/ACT.local.md are user-authored project memory. CLAUDE.md is
// deliberately NOT in this list — it's the AGENTS.md mirror for the claude-code
// swarm backend's own file-discovery and would double-load for Tier 1 agents
// that already read AGENTS.md here. Other-agent conventions (.cursorrules,
// copilot-instructions.md, etc.) are also excluded — they bloat prompts by
// tens of thousands of tokens without adding ACT-shaped context.
var defaultContextPaths = []string{
	"AGENTS.md",
	"ACT.md",
	"ACT.local.md",
}

// Global configuration instance
var cfg *Config

// ResetForTests clears the global config singleton and the underlying viper
// state. Test-only — call before re-invoking Load with a different fixture.
// Production code never touches this; the singleton is intentionally
// one-shot for the lifetime of an act-agent process.
func ResetForTests() {
	cfg = nil
	viper.Reset()
}

// Load initializes the configuration from environment variables and config files.
// If debug is true, debug mode is enabled and log level is set to debug.
// It returns an error if configuration loading fails.
func Load(workingDir string, debug bool) (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	loadDotEnv(workingDir)

	cfg = &Config{
		WorkingDir: workingDir,
		MCPServers: make(map[string]MCPServer),
		Providers:  make(map[models.ModelProvider]Provider),
		LSP:        make(map[string]LSPConfig),
	}

	configureViper()
	setDefaults(debug)

	// Read global config. readSanitizedGlobalConfig strips _comment keys
	// shipped in .act.example.json so users who copy the template verbatim
	// don't trip the mapstructure unmarshal (a "_comment" string value can't
	// be decoded into the Agent struct expected by the agents map).
	if err := readSanitizedGlobalConfig(); err != nil {
		return cfg, fmt.Errorf("failed to read config: %w", err)
	}

	// Load and merge local config
	mergeLocalConfig(workingDir)

	setProviderDefaults()

	// Apply configuration to the struct
	if err := viper.Unmarshal(cfg); err != nil {
		return cfg, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	applyDefaultValues()
	defaultLevel := slog.LevelInfo
	if cfg.Debug {
		defaultLevel = slog.LevelDebug
	}
	if os.Getenv("ACT_DEV_DEBUG") == "true" {
		loggingFile := fmt.Sprintf("%s/%s", cfg.Data.Directory, "debug.log")
		messagesPath := fmt.Sprintf("%s/%s", cfg.Data.Directory, "messages")

		// if file does not exist create it
		if _, err := os.Stat(loggingFile); os.IsNotExist(err) {
			if err := os.MkdirAll(cfg.Data.Directory, 0o755); err != nil {
				return cfg, fmt.Errorf("failed to create directory: %w", err)
			}
			if _, err := os.Create(loggingFile); err != nil {
				return cfg, fmt.Errorf("failed to create log file: %w", err)
			}
		}

		if _, err := os.Stat(messagesPath); os.IsNotExist(err) {
			if err := os.MkdirAll(messagesPath, 0o756); err != nil {
				return cfg, fmt.Errorf("failed to create directory: %w", err)
			}
		}
		logging.MessageDir = messagesPath

		sloggingFileWriter, err := os.OpenFile(loggingFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			return cfg, fmt.Errorf("failed to open log file: %w", err)
		}
		// Configure logger
		logger := slog.New(slog.NewTextHandler(sloggingFileWriter, &slog.HandlerOptions{
			Level: defaultLevel,
		}))
		slog.SetDefault(logger)
	} else {
		// Configure logger
		logger := slog.New(slog.NewTextHandler(logging.NewWriter(), &slog.HandlerOptions{
			Level: defaultLevel,
		}))
		slog.SetDefault(logger)
	}

	// Validate configuration
	if err := Validate(); err != nil {
		return cfg, fmt.Errorf("config validation failed: %w", err)
	}

	if cfg.Agents == nil {
		cfg.Agents = make(map[AgentName]Agent)
	}
	return cfg, nil
}

// configureViper sets the env-var prefix. File reads are handled by
// readSanitizedGlobalConfig/readSanitizedLocalConfig which strip _comment
// keys before handing bytes to viper — the AddConfigPath/ReadInConfig path
// is bypassed entirely because it would unmarshal the comment values into
// typed structs and fail.
func configureViper() {
	viper.SetConfigType("json")
	viper.SetEnvPrefix(strings.ToUpper(appName))
	viper.AutomaticEnv()
}

// setDefaults configures default values for configuration options.
func setDefaults(debug bool) {
	viper.SetDefault("data.directory", defaultDataDirectory)
	viper.SetDefault("contextPaths", defaultContextPaths)
	viper.SetDefault("tui.theme", "act")
	viper.SetDefault("autoCompact", true)

	// Set default shell from environment or fallback to /bin/bash
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		shellPath = "/bin/bash"
	}
	viper.SetDefault("shell.path", shellPath)
	viper.SetDefault("shell.args", []string{"-l"})

	if debug {
		viper.SetDefault("debug", true)
		viper.Set("log.level", "debug")
	} else {
		viper.SetDefault("debug", false)
		viper.SetDefault("log.level", defaultLogLevel)
	}
}

// setProviderDefaults configures LLM provider defaults based on provider provided by
// environment variables and configuration file.
func setProviderDefaults() {
	// Set all API keys we can find in the environment.
	// We use viper.Set instead of SetDefault because if the JSON config contains
	// a provider block with an empty apiKey (e.g. "openrouter": {}), the parsed
	// empty string takes precedence over SetDefault.
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		viper.Set("providers.anthropic.apiKey", apiKey)
	}
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		viper.Set("providers.openai.apiKey", apiKey)
	}
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		viper.Set("providers.gemini.apiKey", apiKey)
	}
	if apiKey := os.Getenv("GROQ_API_KEY"); apiKey != "" {
		viper.Set("providers.groq.apiKey", apiKey)
	}
	if apiKey := os.Getenv("NVIDIA_API_KEY"); apiKey != "" {
		viper.Set("providers.nvidia.apiKey", apiKey)
	}
	if apiKey := os.Getenv("OPENROUTER_API_KEY"); apiKey != "" {
		viper.Set("providers.openrouter.apiKey", apiKey)
	}
	if apiKey := os.Getenv("XAI_API_KEY"); apiKey != "" {
		viper.Set("providers.xai.apiKey", apiKey)
	}
	if apiKey := os.Getenv("AZURE_OPENAI_ENDPOINT"); apiKey != "" {
		// api-key may be empty when using Entra ID credentials – that's okay
		viper.Set("providers.azure.apiKey", os.Getenv("AZURE_OPENAI_API_KEY"))
	}

	// NesTTY uses its own four Tier 1 agents (planner/observer/assurance/
	// qa_synthesizer) plus a configurable Tier 2 swarm. OpenCode-inherited
	// helper agents (summarizer/task/title) are only wired up when the user
	// explicitly configures them in ~/.act.json. Default-injecting a model
	// per provider caused spurious "invalid max tokens" warnings at every
	// launch, so we don't set any agent defaults here anymore.
}

// mergeLocalConfig loads and merges configuration from the local directory.
// Local config is sanitized identically to global so per-project .act.json
// files can use the same _comment convention as the example template.
func mergeLocalConfig(workingDir string) {
	if local := readSanitizedLocalConfig(workingDir); local != nil {
		viper.MergeConfigMap(local.AllSettings())
	}
}

// applyDefaultValues sets default values for configuration fields that need processing.
func applyDefaultValues() {
	// Set default MCP type if not specified
	for k, v := range cfg.MCPServers {
		if v.Type == "" {
			v.Type = MCPStdio
			cfg.MCPServers[k] = v
		}
	}
}

// validateAgent performs lightweight, declarative checks on an agent's
// config block. There is NO model registry — ACT does not know or care
// which model strings are "supported". The upstream provider validates
// the model name when the first request is made. We only verify that
// the agent points at a configured, enabled provider with credentials.
func validateAgent(cfg *Config, name AgentName, agent Agent) error {
	// ACP-backed agents run an external CLI subprocess; model/provider are
	// the external agent's concern, not ACT's in-process config.
	switch agent.Backend {
	case "claude-code", "gemini", "antigravity", "agy":
		return nil
	}
	if agent.Provider == "" {
		return fmt.Errorf("agent %s missing required field 'provider' — set it to one of: anthropic, openai, gemini, groq, nvidia, openrouter, xai, azure, vertexai, bedrock, local", name)
	}
	if agent.Model == "" {
		return fmt.Errorf("agent %s missing required field 'model' — write the upstream model string verbatim (e.g. \"claude-sonnet-4-20250514\", \"z-ai/glm-4.5-air:free\")", name)
	}
	providerCfg, ok := cfg.Providers[agent.Provider]
	if !ok {
		return fmt.Errorf("agent %s configured for provider %q which is not present under \"providers\" in ~/.act.json", name, agent.Provider)
	}
	// Local provider needs only a baseURL (or the LM Studio default);
	// every other provider needs an API key. Bedrock/VertexAI surface
	// their own credential errors at request time when the provider
	// SDK probes the environment.
	switch agent.Provider {
	case models.ProviderLocal, models.ProviderBedrock, models.ProviderVertexAI:
		// no apiKey requirement at config-load time
	default:
		if providerCfg.APIKey == "" {
			return fmt.Errorf("provider %q has no apiKey set in ~/.act.json (used by agent %s)", agent.Provider, name)
		}
	}
	if agent.MaxTokens <= 0 {
		updated := cfg.Agents[name]
		updated.MaxTokens = MaxTokensFallbackDefault
		cfg.Agents[name] = updated
	}
	return nil
}

// Validate checks if the configuration is valid and applies defaults where needed.
func Validate() error {
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	// Validate agent models
	for name, agent := range cfg.Agents {
		if err := validateAgent(cfg, name, agent); err != nil {
			return err
		}
	}

	// Validate LSP configurations
	for language, lspConfig := range cfg.LSP {
		if lspConfig.Command == "" && !lspConfig.Disabled {
			logging.Warn("LSP configuration has no command, marking as disabled", "language", language)
			lspConfig.Disabled = true
			cfg.LSP[language] = lspConfig
		}
	}

	return nil
}

// (removed: getProviderAPIKey, setDefaultModelForAgent, isTier1Agent.
// The model registry they propped up is gone — validateAgent fails
// loudly with an actionable error instead of silently reverting.)

func updateCfgFile(updateCfg func(config *Config)) error {
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	// Get the config file path
	configFile := viper.ConfigFileUsed()
	var configData []byte
	if configFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configFile = filepath.Join(homeDir, fmt.Sprintf(".%s.json", appName))
		logging.Info("config file not found, creating new one", "path", configFile)
		configData = []byte(`{}`)
	} else {
		// Read the existing config file
		data, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		configData = data
	}

	// Parse the JSON
	var userCfg *Config
	if err := json.Unmarshal(configData, &userCfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	updateCfg(userCfg)

	// Write the updated config back to file
	updatedData, err := json.MarshalIndent(userCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, updatedData, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Get returns the current configuration.
// It's safe to call this function multiple times.
func Get() *Config {
	return cfg
}

// WorkingDirectory returns the current working directory from the configuration.
func WorkingDirectory() string {
	if cfg == nil {
		panic("config not loaded")
	}
	return cfg.WorkingDir
}

// UpdateAgentModel changes the model string for a configured agent. The
// provider stays as-is; only the upstream model identifier changes. The
// upstream API validates the model on the next request — ACT does not
// pre-check it against any registry.
func UpdateAgentModel(agentName AgentName, modelID models.ModelID) error {
	if cfg == nil {
		panic("config not loaded")
	}

	existingAgentCfg := cfg.Agents[agentName]

	maxTokens := existingAgentCfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = MaxTokensFallbackDefault
	}

	newAgentCfg := Agent{
		Provider:        existingAgentCfg.Provider,
		Model:           modelID,
		MaxTokens:       maxTokens,
		ReasoningEffort: existingAgentCfg.ReasoningEffort,
		Backend:         existingAgentCfg.Backend,
	}
	cfg.Agents[agentName] = newAgentCfg

	if err := validateAgent(cfg, agentName, newAgentCfg); err != nil {
		cfg.Agents[agentName] = existingAgentCfg
		return fmt.Errorf("failed to update agent model: %w", err)
	}

	return updateCfgFile(func(config *Config) {
		if config.Agents == nil {
			config.Agents = make(map[AgentName]Agent)
		}
		config.Agents[agentName] = newAgentCfg
	})
}

// UpdateAgentProvider changes the provider for an agent. Used by the
// model dialog when switching providers.
func UpdateAgentProvider(agentName AgentName, providerID models.ModelProvider, modelID models.ModelID) error {
	if cfg == nil {
		panic("config not loaded")
	}
	existing := cfg.Agents[agentName]
	maxTokens := existing.MaxTokens
	if maxTokens <= 0 {
		maxTokens = MaxTokensFallbackDefault
	}
	newCfg := Agent{
		Provider:        providerID,
		Model:           modelID,
		MaxTokens:       maxTokens,
		ReasoningEffort: existing.ReasoningEffort,
		Backend:         existing.Backend,
	}
	cfg.Agents[agentName] = newCfg
	if err := validateAgent(cfg, agentName, newCfg); err != nil {
		cfg.Agents[agentName] = existing
		return fmt.Errorf("failed to update agent provider/model: %w", err)
	}
	return updateCfgFile(func(c *Config) {
		if c.Agents == nil {
			c.Agents = make(map[AgentName]Agent)
		}
		c.Agents[agentName] = newCfg
	})
}

// UpdateTheme updates the theme in the configuration and writes it to the config file.
func UpdateTheme(themeName string) error {
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	// Update the in-memory config
	cfg.TUI.Theme = themeName

	// Update the file config
	return updateCfgFile(func(config *Config) {
		config.TUI.Theme = themeName
	})
}

// loadDotEnv attempts to find and parse a .env file from the working directory,
// parent directories, or the configured actRoot, injecting keys into the environment.
func loadDotEnv(workingDir string) {
	candidates := []string{
		filepath.Join(workingDir, ".env"),
	}

	// Walk up parent directories to find a .env file
	curr := workingDir
	for {
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		candidates = append(candidates, filepath.Join(parent, ".env"))
		curr = parent
	}

	// Try actRoot from ~/.act/config.json
	home, _ := os.UserHomeDir()
	if home != "" {
		if data, err := os.ReadFile(filepath.Join(home, ".act", "config.json")); err == nil {
			var actRootCfg struct {
				ActRoot string `json:"actRoot"`
			}
			if err := json.Unmarshal(data, &actRootCfg); err == nil && actRootCfg.ActRoot != "" {
				candidates = append(candidates, filepath.Join(actRootCfg.ActRoot, ".env"))
			}
		}
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])
				// Strip surrounding quotes
				if (strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"")) ||
					(strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'")) {
					if len(v) >= 2 {
						v = v[1 : len(v)-1]
					}
				}
				if os.Getenv(k) == "" {
					os.Setenv(k, v)
				}
			}
		}
		// Stop after loading the first found .env file
		break
	}
}


