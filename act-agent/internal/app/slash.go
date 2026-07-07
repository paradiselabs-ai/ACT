package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/runner"
)

// HandleSlashCommand parses and executes a slash command typed at the TUI
// prompt. Returns (response, true) if the input was a known slash command
// (in which case the TUI should display the response as a system message
// instead of routing to the Planner), or ("", false) if the input was not a
// recognized slash command (in which case the TUI falls through to the
// normal Planner-routing path).
//
// Unknown commands starting with `/` (e.g. `/usr/bin/...`) return false so
// the user can still type literal text containing slashes without triggering
// command parsing.
func (a *App) HandleSlashCommand(input string) (response string, handled bool) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return "", false
	}

	// Tokenize on whitespace
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", false
	}
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "/help":
		return slashHelp(), true
	case "/status":
		return a.slashStatus(), true
	case "/swarm":
		return a.slashSwarm(args), true
	case "/backend":
		return a.slashBackend(args), true
	case "/quit", "/exit":
		// Soft signal — the TUI key handler already maps ctrl+c to clean shutdown
		return "Use ctrl+c to quit the TUI cleanly.", true
	default:
		return "", false
	}
}

func slashHelp() string {
	return strings.Join([]string{
		"## Slash Commands",
		"",
		"  /help                                  Show this help",
		"  /status                                Show ACT system state (Tier 1 + swarm)",
		"",
		"  /swarm                                 List swarm roles, backends, models",
		"  /swarm list                            (alias)",
		"  /swarm <role> <act-agent|claude-code>      Set backend for one swarm role (Tier 2 only)",
		"  /swarm all <act-agent|claude-code>         Set backend for ALL swarm roles",
		"  /swarm restart <role>                  Restart one runner",
		"  /swarm restart all                     Restart the whole swarm",
		"  /swarm status                          Show live runner PIDs and state",
		"",
		"  /backend                               List Tier 1 role backends",
		"  /backend list                          (alias)",
		"  /backend <role> <act-agent|claude-code|antigravity>  Switch one Tier 1 role's backend",
		"  /backend all <act-agent|claude-code|antigravity>     Switch all four Tier 1 roles",
		"",
		"  /quit | /exit                          Hint: use ctrl+c to quit cleanly",
	}, "\n")
}

func (a *App) slashStatus() string {
	var sb strings.Builder
	sb.WriteString("## ACT System Status\n\n")

	sb.WriteString("**Tier 1 (in-process)**:\n")
	t1 := []string{"planner", "observer", "assurance", "qa_synthesizer"}
	for _, r := range t1 {
		if _, ok := a.Agents[r]; ok {
			sb.WriteString(fmt.Sprintf("  ✓ %s\n", r))
		} else {
			sb.WriteString(fmt.Sprintf("  ✗ %s (not configured)\n", r))
		}
	}

	sb.WriteString("\n**Tier 2 (swarm)**:\n")
	if a.Orchestrator != nil && a.Orchestrator.runnerSpawner != nil {
		statuses := a.Orchestrator.runnerSpawner.Status()
		if len(statuses) == 0 {
			sb.WriteString("  (no runners alive)\n")
		}
		sort.Slice(statuses, func(i, j int) bool { return statuses[i].Role < statuses[j].Role })
		for _, s := range statuses {
			alive := "alive"
			if !s.Alive {
				alive = "dead"
			}
			sb.WriteString(fmt.Sprintf("  ✓ %-15s pid=%-7d backend=%-12s model=%s [%s]\n",
				s.Role, s.PID, s.Backend, s.Model, alive))
		}
	}

	return sb.String()
}

// ─── /swarm subcommands ────────────────────────────────────────────────────────

func (a *App) slashSwarm(args []string) string {
	if len(args) == 0 || args[0] == "list" {
		return a.swarmList()
	}
	if args[0] == "status" {
		return a.swarmStatus()
	}
	if args[0] == "restart" {
		if len(args) < 2 {
			return "usage: /swarm restart <role|all>"
		}
		return a.swarmRestart(args[1])
	}
	// /swarm <role> <backend>  OR  /swarm all <backend>
	if len(args) >= 2 {
		role := args[0]
		backend := args[1]
		return a.swarmSetBackend(role, backend)
	}
	return "unknown /swarm subcommand. Try /swarm list"
}

func (a *App) swarmList() string {
	if a.Orchestrator == nil || a.Orchestrator.runnerSpawner == nil {
		return "swarm not initialized"
	}
	statuses := a.Orchestrator.runnerSpawner.Status()
	sort.Slice(statuses, func(i, j int) bool { return statuses[i].Role < statuses[j].Role })

	var sb strings.Builder
	sb.WriteString("## Swarm\n\n")
	if len(statuses) == 0 {
		sb.WriteString("(no swarm runners alive)\n")
		return sb.String()
	}
	sb.WriteString(fmt.Sprintf("  %-15s %-12s %-25s %s\n", "ROLE", "BACKEND", "MODEL", "AGENT_ID"))
	for _, s := range statuses {
		sb.WriteString(fmt.Sprintf("  %-15s %-12s %-25s %s\n", s.Role, s.Backend, s.Model, s.AgentID))
	}
	return sb.String()
}

func (a *App) swarmStatus() string {
	return a.swarmList()
}

func (a *App) swarmRestart(target string) string {
	if a.Orchestrator == nil || a.Orchestrator.runnerSpawner == nil {
		return "swarm not initialized"
	}

	if target == "all" {
		a.Orchestrator.runnerSpawner.Stop()
		if err := a.Orchestrator.runnerSpawner.StartSwarm(a.SwarmSpecs); err != nil {
			return fmt.Sprintf("restart failed: %v", err)
		}
		return fmt.Sprintf("Restarted %d swarm runners", len(a.SwarmSpecs))
	}

	if !runner.IsSwarmRole(target) {
		return fmt.Sprintf("unknown swarm role %q (valid: %s)", target, strings.Join(runner.AllSwarmRoles, ", "))
	}

	for _, spec := range a.SwarmSpecs {
		if spec.Role == target {
			if err := a.Orchestrator.runnerSpawner.RestartRole(spec); err != nil {
				return fmt.Sprintf("restart failed: %v", err)
			}
			return fmt.Sprintf("Restarted swarm runner %q", target)
		}
	}
	return fmt.Sprintf("no spec found for role %q", target)
}

func (a *App) swarmSetBackend(role, backend string) string {
	if !runner.IsValidBackend(backend) {
		return fmt.Sprintf("invalid backend %q (valid: %s, %s)", backend, runner.BackendActAgent, runner.BackendClaudeCode)
	}

	// Bulk operation
	if role == "all" {
		updated := 0
		for i, spec := range a.SwarmSpecs {
			if !runner.IsSwarmRole(spec.Role) {
				continue
			}
			a.SwarmSpecs[i].Backend = backend
			if err := config.WriteAgentBackend(spec.Role, backend); err != nil {
				logging.Warn("Failed to persist backend change", "role", spec.Role, "error", err)
			}
			updated++
		}
		// Restart whole swarm to pick up changes
		if a.Orchestrator != nil && a.Orchestrator.runnerSpawner != nil {
			a.Orchestrator.runnerSpawner.Stop()
			_ = a.Orchestrator.runnerSpawner.StartSwarm(a.SwarmSpecs)
		}
		return fmt.Sprintf("Set backend=%s for %d swarm roles. Restarting swarm...", backend, updated)
	}

	// Tier 1 rejection
	if !runner.IsSwarmRole(role) {
		return fmt.Sprintf("backend selection only applies to Tier 2 swarm agents (%s). %q is not a swarm role.",
			strings.Join(runner.AllSwarmRoles, ", "), role)
	}

	// Single role
	for i, spec := range a.SwarmSpecs {
		if spec.Role != role {
			continue
		}
		a.SwarmSpecs[i].Backend = backend
		if err := config.WriteAgentBackend(role, backend); err != nil {
			return fmt.Sprintf("backend change failed to persist: %v", err)
		}
		if a.Orchestrator != nil && a.Orchestrator.runnerSpawner != nil {
			if err := a.Orchestrator.runnerSpawner.RestartRole(a.SwarmSpecs[i]); err != nil {
				return fmt.Sprintf("backend updated in config but runner restart failed: %v", err)
			}
		}
		return fmt.Sprintf("Swarm role %q backend set to %q. Restarting %s...", role, backend, spec.AgentID)
	}
	return fmt.Sprintf("no spec found for role %q", role)
}

// ─── /backend subcommands (Tier 1 only) ────────────────────────────────────────

var tier1Roles = []string{"planner", "observer", "assurance", "qa_synthesizer"}

// slashBackend implements `/backend` for Tier 1 role backend selection.
// Same vocabulary as `/swarm` — the backend value IS the agent name.
//
// Forms:
//   /backend                              — list current backends
//   /backend list                         — alias
//   /backend <role> <act-agent|claude-code>  — switch one Tier 1 role
//   /backend all    <act-agent|claude-code>  — bulk switch all four Tier 1 roles
//
// On success the config write lands in ~/.act.json. The change applies to
// the NEXT TUI launch — hot-rebuild of Tier 1 agents is out of scope for
// the alpha (manual restart keeps the failure surface visible per plan).
func (a *App) slashBackend(args []string) string {
	if len(args) == 0 || args[0] == "list" {
		return a.backendList()
	}
	if len(args) < 2 {
		return "usage: /backend <role|all> <act-agent|claude-code|antigravity>"
	}
	role := args[0]
	backend := args[1]

	if !isValidTier1Backend(backend) {
		return fmt.Sprintf("invalid backend %q (valid: act-agent, claude-code, antigravity)", backend)
	}

	if role == "all" {
		updated := 0
		for _, r := range tier1Roles {
			if err := config.WriteAgentBackend(r, backend); err != nil {
				logging.Warn("Failed to persist Tier 1 backend", "role", r, "error", err)
				continue
			}
			updated++
		}
		return fmt.Sprintf("Set backend=%s for %d Tier 1 roles. Restart ACT to apply.",
			backend, updated)
	}

	if !isTier1Role(role) {
		return fmt.Sprintf("/backend only applies to Tier 1 roles (%s). For swarm roles, use /swarm.",
			strings.Join(tier1Roles, ", "))
	}
	if err := config.WriteAgentBackend(role, backend); err != nil {
		return fmt.Sprintf("backend change failed to persist: %v", err)
	}
	return fmt.Sprintf("Tier 1 role %q backend set to %s. Restart ACT to apply.",
		role, backend)
}

func (a *App) backendList() string {
	var sb strings.Builder
	sb.WriteString("## Tier 1 backends\n\n")
	sb.WriteString(fmt.Sprintf("  %-18s %s\n", "ROLE", "BACKEND"))
	cfg := config.Get()
	for _, r := range tier1Roles {
		backend := "act-agent"
		if cfg != nil {
			if ac, ok := cfg.Agents[config.AgentName(r)]; ok && ac.Backend != "" {
				backend = ac.Backend
			}
		}
		sb.WriteString(fmt.Sprintf("  %-18s %s\n", r, backend))
	}
	return sb.String()
}

func isTier1Role(s string) bool {
	for _, r := range tier1Roles {
		if r == s {
			return true
		}
	}
	return false
}

// isValidTier1Backend reports whether the given value is a known Tier 1
// backend identifier. Kept narrow on purpose — adding "codex" / "gemini" /
// "opencode" here is the same gesture as wiring them up in
// internal/acp/agent.go's buildCommand switch.
func isValidTier1Backend(s string) bool {
	switch s {
	case "act-agent", "claude-code", "antigravity", "agy":
		return true
	}
	return false
}

