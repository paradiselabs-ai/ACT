package runner

import "fmt"

// Tier 2 swarm role identifiers. These match the keys in `~/.act.json` under
// `agents.<role>` and the prompt dispatcher in internal/llm/prompt/prompt.go.
const (
	RoleDeveloper   = "developer"
	RoleFrontendDev = "frontend_dev"
	RoleBackendDev  = "backend_dev"
	RoleQAEngineer  = "qa_engineer"
	RoleResearcher  = "researcher"
)

// AllSwarmRoles is the canonical ordered list of Tier 2 swarm roles.
// The Spawner walks this list when starting the swarm. Order is intentional:
// developer first (the default fallback), then specialists, then researcher.
var AllSwarmRoles = []string{
	RoleDeveloper,
	RoleFrontendDev,
	RoleBackendDev,
	RoleQAEngineer,
	RoleResearcher,
}

// IsSwarmRole returns true if the given role name is a known Tier 2 swarm role.
// Used by slash commands and the CLI to reject Tier 1 role names with a clear
// error message.
func IsSwarmRole(role string) bool {
	for _, r := range AllSwarmRoles {
		if r == role {
			return true
		}
	}
	return false
}

// Backend identifiers for the SwarmRoleSpec.Backend field.
const (
	BackendActAgent    = "act-agent"
	BackendClaudeCode  = "claude-code"
	BackendGemini      = "gemini"
	BackendAntigravity = "antigravity" // Tier 1 via ACP; Tier 2 via a direct `agy --print` one-shot in the Runner
)

// IsValidBackend returns true if the given backend name is supported.
func IsValidBackend(backend string) bool {
	return backend == BackendActAgent || backend == BackendClaudeCode ||
		backend == BackendGemini || backend == BackendAntigravity
}

// BackendAllowedForRole reports whether a backend may host a given swarm role.
// Backends that cannot honor a role's privilege contract are rejected here —
// at config-set time — rather than silently running with more privilege than
// the role allows.
//
// researcher + antigravity: the researcher is read-only by contract on every
// other backend (ResearcherTools on act-agent, --disallowedTools on claude-code,
// --approval-mode plan on gemini). The agy CLI has no read-only or plan mode —
// its only restriction flag is --sandbox, which limits terminal access, not file
// writes — so an antigravity researcher would run with full write privilege.
func BackendAllowedForRole(role, backend string) error {
	if role == RoleResearcher && backend == BackendAntigravity {
		return fmt.Errorf("backend %q is not allowed for the %s role: agy has no read-only/plan mode "+
			"(--sandbox restricts the terminal only), so the researcher's read-only contract cannot be enforced; "+
			"use act-agent, claude-code, or gemini for researcher", backend, RoleResearcher)
	}
	return nil
}

// SwarmRoleSpec is the per-Runner configuration consumed by the Spawner.
// One spec produces one Runner subprocess (one swarm agent).
type SwarmRoleSpec struct {
	// Role is the Tier 2 role name (e.g. "developer", "frontend_dev"). Must
	// be one of the constants above.
	Role string

	// AgentID is the unique identifier this Runner registers with on the ACT
	// server. By convention: <role-prefix>-1 (e.g. "dev-1", "frontend-1").
	AgentID string

	// Name is a human-readable label shown in logs and the TUI swarm panel.
	Name string

	// Backend selects the agent execution path: "act-agent" (default),
	// "claude-code", "gemini", or "antigravity". The Runner script reads
	// --backend to dispatch.
	Backend string

	// Capabilities is the list of capability tags this Runner registers with.
	// The ACT server uses these to route tasks via assignOptimalAgent.
	Capabilities []string

	// Model is the LLM model identifier (informational; the act-agent binary
	// reads its own model config from ~/.act.json based on Role).
	Model string
}

// DefaultAgentID returns the conventional agent ID for a swarm role.
// Examples: developer → dev-1, frontend_dev → frontend-1, qa_engineer → qa-1.
func DefaultAgentID(role string) string {
	switch role {
	case RoleDeveloper:
		return "dev-1"
	case RoleFrontendDev:
		return "frontend-1"
	case RoleBackendDev:
		return "backend-1"
	case RoleQAEngineer:
		return "qa-1"
	case RoleResearcher:
		return "researcher-1"
	default:
		return role + "-1"
	}
}

// DefaultName returns the human-readable name for a swarm role.
func DefaultName(role string) string {
	switch role {
	case RoleDeveloper:
		return "Developer"
	case RoleFrontendDev:
		return "Frontend Dev"
	case RoleBackendDev:
		return "Backend Dev"
	case RoleQAEngineer:
		return "QA Engineer"
	case RoleResearcher:
		return "Researcher"
	default:
		return role
	}
}
