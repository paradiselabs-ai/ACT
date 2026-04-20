// Sourced from cli/act-cli.ts dispatch (lines 915-1059).
package tools

// RoleSubcommands lists the `act` CLI subcommands each Tier 1 role is allowed
// to invoke. Sourced from cli/act-cli.ts dispatch (lines 915-1059).
// Restrictive-by-design — add entries only with a clear per-role rationale.
var RoleSubcommands = map[string][]string{
	"planner": {
		"status",   // high-level overview
		"context",  // project snapshot (agents, tasks, locks)
		"log",      // recent coordination events
		"graph",    // graph unverified | graph conflicts (routing evidence)
		"pvm",      // pvm search — past coordination patterns
		"message",  // send a coord message to another agent
		"codebase", // codebase rules | communities | onboard (read-only intel)
	},
	"observer": {
		"status",
		"log",
		"graph",   // unverified + conflicts only — anomaly detection
		"context", // project-state awareness
	},
	"assurance": {
		"validation", // validation queue
		"log",        // trace of what happened before submission
		"status",
	},
	"qa_synthesizer": {
		"validation",
		"log",
		"status",
		"codebase", // onboard to understand the deliverable structure
	},
}

// AllowedFor returns a copy of the subcommand slice for the given role.
// Returns nil for unknown roles. Callers may mutate the result freely —
// the underlying map is not aliased.
func AllowedFor(role string) []string {
	src, ok := RoleSubcommands[role]
	if !ok {
		return nil
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}

// IsAllowed reports whether the given role may invoke the given subcommand.
// Unknown roles return false. Builds the lookup set on each call — this is
// a rare hot path (one Tier 1 tool invocation per LLM turn), so the extra
// allocation is cheaper than the lock coordination a package-level cache
// would need.
func IsAllowed(role, subcommand string) bool {
	src, ok := RoleSubcommands[role]
	if !ok {
		return false
	}
	lookup := make(map[string]struct{}, len(src))
	for _, s := range src {
		lookup[s] = struct{}{}
	}
	_, ok = lookup[subcommand]
	return ok
}
