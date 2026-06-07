package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/act"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/acp"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/db"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/format"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/history"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/agent"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/prompt"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/tools"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/lsp"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/message"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/permission"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/runner"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/session"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
)

type App struct {
	Sessions    session.Service
	Messages    message.Service
	History     history.Service
	Permissions permission.Service

	Agents       map[string]agent.Service // Tier 1: "planner", "observer", "assurance", "qa_synthesizer"
	SwarmSpecs   []runner.SwarmRoleSpec   // Tier 2: one spec per swarm role to spawn
	Orchestrator *Orchestrator

	LSPClients map[string]*lsp.Client

	clientsMutex sync.RWMutex

	watcherCancelFuncs []context.CancelFunc
	cancelFuncsMutex   sync.Mutex
	watcherWG          sync.WaitGroup
}

func New(ctx context.Context, conn *sql.DB) (*App, error) {
	q := db.New(conn)
	sessions := session.NewService(q)
	messages := message.NewService(q)
	files := history.NewService(q, conn)

	app := &App{
		Sessions:    sessions,
		Messages:    messages,
		History:     files,
		Permissions: permission.NewPermissionService(),
		LSPClients:  make(map[string]*lsp.Client),
	}

	// Initialize theme based on configuration
	app.initTheme()

	// Initialize LSP clients in the background
	go app.initLSPClients(ctx)

	// Create Tier 1 agents (each with role-specific model from .act.json AND
	// a role-specific tool subset). Per-role subsets are critical for free-tier
	// providers — the full developer toolbox ships ~16K tokens of tool schemas
	// per request, blowing Groq's 12K TPM cap. Planner and Observer only need
	// bash; Assurance and QA need bash + view + grep. See agent.Tier1ToolsForRole
	// for the rationale.
	tier1Roles := []string{"planner", "observer", "assurance", "qa_synthesizer"}
	app.Agents = make(map[string]agent.Service, len(tier1Roles))

	cfg := config.Get()
	for _, role := range tier1Roles {
		agentName := config.AgentConfigForRole(role)

		// Per-role backend dispatch. Backend value IS the host name —
		// "claude-code" (alpha), future "codex" / "gemini" / "opencode" —
		// or the default "act-agent" (empty == in-process LLM). The wire-level
		// mechanism (ACP for Tier 1 here) is invisible to the user.
		var backendChoice string
		var acpCfg *config.ACPConfig
		if cfg != nil {
			if ac, ok := cfg.Agents[agentName]; ok {
				backendChoice = ac.Backend
				acpCfg = ac.ACP
			}
		}

		var (
			agentSvc agent.Service
			err      error
			toolsN   int
		)
		switch backendChoice {
		case "claude-code", "codex", "gemini", "opencode":
			// External CLI agent → drive over ACP. The acp package decides
			// the spawn argv from the backend name; codex/gemini/opencode
			// return explicit unimplemented errors for the alpha.
			withShim := withTier1ShimPath(role, acpCfg)
			agentSvc, err = acp.NewACPAgent(role, backendChoice, withShim, app.Sessions, app.Messages, makePrimingInjector(role))
		default:
			roleTools := agent.Tier1ToolsForRole(
				role,
				app.Permissions,
				app.Sessions,
				app.Messages,
				app.History,
				app.LSPClients,
			)
			toolsN = len(roleTools)
			// Per-agent notebooks. Every in-process role stamps its own threadID (its
			// role name) on the messages it creates, in the one shared display session.
			// History mode keys on role, not backend, so flipping a role to in-process
			// later scopes it automatically:
			//   - Planner → HistoryThread: feeds only its own thread (human turns +
			//     autoroute prompts + its replies), NOT the workers' shared-session
			//     traffic. The conversational partner, but scoped to its conversation.
			//   - Observer/Assurance/QA → HistoryNone: stateless, self-contained
			//     snapshots; no prior history at all.
			histMode := agent.HistoryNone
			if role == "planner" {
				histMode = agent.HistoryThread
			}
			agentSvc, err = agent.NewAgent(agentName, app.Sessions, app.Messages, roleTools, role, histMode)
		}
		if err != nil {
			logging.Warn("tier1_agent_wire_failed", "role", role, "config_key", string(agentName), "backend", backendChoice, "error", err)
			continue
		}
		app.Agents[role] = agentSvc

		// Log the resolved config so a silent fallback (role key missing from
		// .act.json → falls back to developer) is visible at startup. Without
		// this, a broken Planner config looks identical to a working one until
		// the first turn produces unexpected behavior.
		fallback := string(agentName) != role
		var modelID, maxTokens string
		if cfg != nil {
			if ac, ok := cfg.Agents[agentName]; ok {
				modelID = string(ac.Model)
				maxTokens = fmt.Sprintf("%d", ac.MaxTokens)
			}
		}
		resolvedBackend := backendChoice
		if resolvedBackend == "" {
			resolvedBackend = "act-agent"
		}
		logging.Info("tier1_agent_wired",
			"role", role,
			"config_key", string(agentName),
			"config_fallback", fallback,
			"backend", resolvedBackend,
			"model", modelID,
			"max_tokens", maxTokens,
			"tools", toolsN,
		)
		if role == "planner" && fallback {
			logging.Warn("planner_config_fallback",
				"resolved_key", string(agentName),
				"reason", "no agents.planner entry in ~/.act.json — using developer config",
			)
		}
	}

	// The Planner is the canonical human-facing agent — non-interactive mode
	// and the TUI both route through it. If it failed to construct above,
	// there's no usable app.
	if _, ok := app.Agents["planner"]; !ok {
		return nil, fmt.Errorf("planner agent construction failed; cannot start ACT")
	}

	// Build the Tier 2 swarm specs from .act.json. One spec per known swarm
	// role that's configured (or defaulted). The Spawner will start one
	// Runner subprocess per spec.
	app.SwarmSpecs = buildSwarmSpecs(config.Get())

	// Orchestrator coordinates the agents
	app.Orchestrator = NewOrchestrator(app)

	return app, nil
}

// withTier1ShimPath returns a copy of cfg with cfg.Env["PATH"] prepended with
// a directory that exposes only this role's act-tier1-<role> symlink. The
// ACP-backed Tier 1 agent's native Bash tool can then invoke
// `act-tier1-planner status` (etc.) and the call is gated by the shim's
// role-aware whitelist — same enforcement the in-process tool has via
// internal/llm/tools.IsAllowed.
//
// Locating the install dir: walk from this process's binary path. The build
// script (per the plan) lands act-tier1-shim alongside the act-agent binary,
// so os.Executable's parent dir is where the symlinks live.
func withTier1ShimPath(role string, base *config.ACPConfig) *config.ACPConfig {
	var out config.ACPConfig
	if base != nil {
		out = *base // shallow copy — Env is a map, we re-make below.
	}
	srcEnv := map[string]string(nil)
	if base != nil {
		srcEnv = base.Env
	}
	out.Env = make(map[string]string, len(srcEnv)+1)
	maps.Copy(out.Env, srcEnv)

	exe, err := os.Executable()
	if err != nil {
		logging.Warn("acp_tier1_shim_path_skipped",
			"role", role,
			"reason", "os.Executable failed",
			"error", err,
		)
		return &out
	}
	binDir := filepath.Dir(exe)

	// Carry the parent process's PATH through (it already has node/npx and the
	// system tools the agent needs), and prepend the act-agent install dir so
	// `act` and `act-tier1-<role>` resolve.
	parentPath := os.Getenv("PATH")
	if existing, ok := out.Env["PATH"]; ok {
		parentPath = existing
	}
	out.Env["PATH"] = binDir + string(os.PathListSeparator) + parentPath
	return &out
}

// makePrimingInjector returns the priming-prompt closure for a Tier 1 role.
// It is invoked once per ACP session (lazy, at first Run for each ACT
// sessionID). The text is the role's static system prompt — the same one
// shipped to the in-process LLM today — plus the shim-binary instructions.
//
// The role prompts live in internal/llm/prompt/<role>.go but are wired up
// through the prompt dispatcher (GetAgentPrompt). We use that here so the
// ACP backend stays in lockstep with the in-process backend on prompt
// content — no second copy to maintain.
func makePrimingInjector(role string) acp.SystemPromptInjector {
	return func(_ string) string {
		// Provider doesn't shape Tier 1 prompts (the role prompts are
		// provider-agnostic prose). Pass the synthetic ACP provider so the
		// dispatcher takes a deterministic branch.
		base := prompt.GetAgentPrompt(config.AgentName(role), acp.ProviderACP)
		if base == "" {
			return ""
		}
		return wrapPriming(role, base)
	}
}

// doNotRespondHeader leads the ACP priming so the LLM has an explicit
// "this is configuration, do not reply" instruction. ACP has no system
// channel (acp/types.go:147-161 — ContentBlock.Type is text/image/
// resource only), so the priming arrives as a user message and the
// model would otherwise treat it as conversation. Smaller models may
// still ack — unavoidable without a real system channel — but this
// shifts the noise-to-signal ratio. Audit Fix 8 (entries 5.2 + 5.3).
const doNotRespondHeader = "[ACT priming — do not respond. This is one-time configuration injected by the orchestrator. Acknowledge silently by emitting no text.]\n\n"

// wrapPriming composes the final ACP priming text from the role prompt
// base + shim discoverability note, fronted by InternalPromptMarker and
// the do-not-respond header. Extracted from makePrimingInjector so unit
// tests can exercise the wrapping shape without needing a loaded
// config (prompt.GetAgentPrompt indirectly calls
// config.WorkingDirectory which panics in test contexts).
func wrapPriming(role, base string) string {
	return InternalPromptMarker + doNotRespondHeader + base + renderShimNote(role)
}

// renderShimNote builds the [ACT] discoverability footer appended to the
// ACP priming text. The allowed list is read live from the canonical
// whitelist so this note can't drift out of sync. Bare entries render as
// "<name>"; compound entries (e.g. "task retry") render verbatim so the
// LLM sees the full sub-subcommand form it must use.
func renderShimNote(role string) string {
	allowed := tools.AllowedFor(role)
	var b strings.Builder
	fmt.Fprintf(&b, "\n\n[ACT] The CLI `act-tier1-%s` is on your PATH. Use it via Bash for all ACT-coordination subcommands. It enforces this role's allowed subcommand set:\n", role)
	for _, entry := range allowed {
		fmt.Fprintf(&b, "  - %s\n", entry)
	}
	b.WriteString("Anything outside this list will be rejected at the shim boundary.")
	return b.String()
}

// buildSwarmSpecs walks the configured agents and produces a SwarmRoleSpec
// for each Tier 2 swarm role. Roles not present in the config still get a
// default spec — the developer role is always included as the fallback.
func buildSwarmSpecs(cfg *config.Config) []runner.SwarmRoleSpec {
	specs := make([]runner.SwarmRoleSpec, 0, len(runner.AllSwarmRoles))

	for _, role := range runner.AllSwarmRoles {
		spec := runner.SwarmRoleSpec{
			Role:         role,
			AgentID:      runner.DefaultAgentID(role),
			Name:         runner.DefaultName(role),
			Backend:      runner.BackendActAgent,
			Capabilities: runner.DefaultCapabilities[role],
		}

		if cfg != nil {
			if agentCfg, ok := cfg.Agents[config.AgentName(role)]; ok {
				if agentCfg.Backend != "" && runner.IsValidBackend(agentCfg.Backend) {
					spec.Backend = agentCfg.Backend
				}
				spec.Model = string(agentCfg.Model)
			}
		}

		// Skip roles whose backend can't be supported on this machine.
		// (StartSwarm will also check, but we filter early so /swarm list
		// shows the truth.)
		specs = append(specs, spec)
	}

	return specs
}

// CreateAgentForRole creates a new agent using role-specific model config.
// Falls back to the developer role's config if no role-specific config exists
// (see config.AgentConfigForRole). Used by --agent mode to select the model
// per swarm role with the full Tier 2 toolbox.
func (a *App) CreateAgentForRole(role string) (agent.Service, error) {
	agentName := config.AgentConfigForRole(role)
	return agent.NewAgent(
		agentName,
		a.Sessions,
		a.Messages,
		agent.DeveloperTools(
			a.Permissions,
			a.Sessions,
			a.Messages,
			a.History,
			a.LSPClients,
		),
		"",                 // no thread tag — Tier 2 runs alone in its own task session
		agent.HistoryFull,  // keeps full history of that session
	)
}

// initTheme sets the application theme based on the configuration
func (app *App) initTheme() {
	cfg := config.Get()
	if cfg == nil || cfg.TUI.Theme == "" {
		return // Use default theme
	}

	// Try to set the theme from config
	err := theme.SetTheme(cfg.TUI.Theme)
	if err != nil {
		logging.Warn("Failed to set theme from config, using default theme", "theme", cfg.TUI.Theme, "error", err)
	} else {
		logging.Debug("Set theme from config", "theme", cfg.TUI.Theme)
	}
}

// RunNonInteractive handles the execution flow when a prompt is provided via CLI flag.
func (a *App) RunNonInteractive(ctx context.Context, prompt string, outputFormat string, quiet bool) error {
	logging.Info("Running in non-interactive mode")

	// Start spinner if not in quiet mode
	var spinner *format.Spinner
	if !quiet {
		spinner = format.NewSpinner("Thinking...")
		spinner.Start()
		defer spinner.Stop()
	}

	const maxPromptLengthForTitle = 100
	titlePrefix := "Non-interactive: "
	var titleSuffix string

	if len(prompt) > maxPromptLengthForTitle {
		titleSuffix = prompt[:maxPromptLengthForTitle] + "..."
	} else {
		titleSuffix = prompt
	}
	title := titlePrefix + titleSuffix

	sess, err := a.Sessions.Create(ctx, title)
	if err != nil {
		return fmt.Errorf("failed to create session for non-interactive mode: %w", err)
	}
	logging.Info("Created session for non-interactive run", "session_id", sess.ID)

	// Automatically approve all permission requests for this non-interactive session
	a.Permissions.AutoApproveSession(sess.ID)

	// Route non-interactive single-shot prompts to the Planner — ACT's canonical
	// human entry point. New() guarantees Agents["planner"] is non-nil or it
	// would have errored out of construction.
	done, err := a.Agents["planner"].Run(ctx, sess.ID, prompt)
	if err != nil {
		return fmt.Errorf("failed to start agent processing stream: %w", err)
	}

	result := <-done
	if result.Error != nil {
		if errors.Is(result.Error, context.Canceled) || errors.Is(result.Error, agent.ErrRequestCancelled) {
			logging.Info("Agent processing cancelled", "session_id", sess.ID)
			return nil
		}
		return fmt.Errorf("agent processing failed: %w", result.Error)
	}

	// Stop spinner before printing output
	if !quiet && spinner != nil {
		spinner.Stop()
	}

	// Get the text content from the response
	content := "No content available"
	if result.Message.Content().String() != "" {
		content = result.Message.Content().String()
	}

	fmt.Println(format.FormatOutput(content, outputFormat))

	logging.Info("Non-interactive run completed", "session_id", sess.ID)

	return nil
}

// RunAgent handles headless ACT agent mode — JSON stdout, no TUI, no spinner.
// The runner provides the task prompt; this method returns structured JSON.
// If role is provided, uses role-specific model config (e.g., "developer" uses cheaper model).
func (a *App) RunAgent(ctx context.Context, prompt string, agentID string, role string) error {
	logging.Info("Running in ACT agent mode", "agent_id", agentID, "role", role)

	// Build the agent for this role. Swarm roles (developer, frontend_dev, etc.)
	// get their configured model + full DeveloperTools. If role is empty, fall
	// back to the Planner — the human-facing canonical agent.
	var runAgent agent.Service
	if role != "" {
		roleAgent, err := a.CreateAgentForRole(role)
		if err != nil {
			return a.agentError(agentID, fmt.Errorf("failed to create agent for role %q: %w", role, err))
		}
		runAgent = roleAgent
	} else {
		runAgent = a.Agents["planner"]
	}

	// ACT coordination: register with server and fetch context
	project := os.Getenv("ACT_PROJECT")
	actClient := act.NewClient(agentID, project)
	if actClient.IsAvailable() {
		if err := actClient.Register(); err != nil {
			logging.Warn("ACT registration failed (continuing without coordination)", "error", err)
		} else {
			logging.Info("Registered with ACT server", "agent_id", agentID)
		}
		// Prepend ACT context (brief, task, parallel agents) to the prompt
		if project != "" {
			if actContext, err := actClient.GetContext(); err == nil && actContext != "" {
				prompt = fmt.Sprintf("## ACT Coordination Context\n%s\n\n## Task\n%s", actContext, prompt)
			}
		}
	}

	sess, err := a.Sessions.Create(ctx, fmt.Sprintf("act-agent:%s", agentID))
	if err != nil {
		return a.agentError(agentID, fmt.Errorf("failed to create session: %w", err))
	}

	// Auto-approve all permissions — headless agents can't prompt
	a.Permissions.AutoApproveSession(sess.ID)

	done, err := runAgent.Run(ctx, sess.ID, prompt)
	if err != nil {
		return a.agentError(agentID, fmt.Errorf("failed to start agent: %w", err))
	}

	result := <-done
	if result.Error != nil {
		if errors.Is(result.Error, context.Canceled) || errors.Is(result.Error, agent.ErrRequestCancelled) {
			return a.agentOutput(agentID, "cancelled", "Agent processing was cancelled")
		}
		// Report failure to ACT server
		if actClient.IsAvailable() {
			_ = actClient.SendMessage(fmt.Sprintf("Task failed: %s", result.Error.Error()))
		}
		return a.agentError(agentID, result.Error)
	}

	content := "No content available"
	if result.Message.Content().String() != "" {
		content = result.Message.Content().String()
	}

	// Report completion to ACT server
	if actClient.IsAvailable() {
		summary := content
		if len(summary) > 500 {
			summary = summary[:500] + "..."
		}
		_ = actClient.SendMessage(fmt.Sprintf("Task completed: %s", summary))
	}

	return a.agentOutput(agentID, "completed", content)
}

// agentOutput writes structured JSON to stdout for the runner to parse.
func (a *App) agentOutput(agentID string, status string, result string) error {
	out := struct {
		AgentID   string `json:"agent_id"`
		Status    string `json:"status"`
		Result    string `json:"result"`
		Timestamp string `json:"timestamp"`
	}{
		AgentID:   agentID,
		Status:    status,
		Result:    result,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent output: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// agentError writes a structured error JSON to stdout (not stderr) so the runner can parse it.
func (a *App) agentError(agentID string, err error) error {
	_ = a.agentOutput(agentID, "error", err.Error())
	return err
}

// Shutdown performs a clean shutdown of the application
func (app *App) Shutdown() {
	// Stop orchestrator background loops and reap Runner subprocess
	if app.Orchestrator != nil {
		app.Orchestrator.Stop()
	}

	// Cancel all watcher goroutines
	app.cancelFuncsMutex.Lock()
	for _, cancel := range app.watcherCancelFuncs {
		cancel()
	}
	app.cancelFuncsMutex.Unlock()
	app.watcherWG.Wait()

	// Perform additional cleanup for LSP clients
	app.clientsMutex.RLock()
	clients := make(map[string]*lsp.Client, len(app.LSPClients))
	maps.Copy(clients, app.LSPClients)
	app.clientsMutex.RUnlock()

	for name, client := range clients {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := client.Shutdown(shutdownCtx); err != nil {
			logging.Error("Failed to shutdown LSP client", "name", name, "error", err)
		}
		cancel()
	}
}
