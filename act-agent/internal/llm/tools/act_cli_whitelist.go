// Sourced from cli/act-cli.ts dispatch (lines 915-1059).
package tools

import "strings"

// RoleSubcommands lists the `act` CLI subcommands each Tier 1 role is allowed
// to invoke. Sourced from cli/act-cli.ts dispatch (lines 915-1059).
// Restrictive-by-design — add entries only with a clear per-role rationale.
//
// Two encodings are supported:
//   - bare subcommand           — e.g. "status" (matches any args)
//   - compound "cmd subcmd"     — e.g. "task retry" (matches only when
//     args[0] == "retry")
//
// Compound keys exist so we can grant the Planner narrow subsets of a
// multi-purpose command. The `task` command, for example, has subcommands
// that belong to swarm agents (`task complete`, `task submit-for-validation`)
// — granting bare `"task"` would let the Planner spoof those. `"task retry"`
// + `"task abandon"` keeps the surface narrow.
var RoleSubcommands = map[string][]string{
	"planner": {
		"status",         // high-level overview
		"context",        // project snapshot (agents, tasks, locks)
		"log",            // recent coordination events
		"graph",          // graph unverified | graph conflicts (routing evidence)
		"pvm",            // pvm search — past coordination patterns
		"message",        // send a coord message to another agent
		"task retry",     // re-dispatch a failed task to a new agent
		"task abandon",   // mark a task permanently failed (skips retry)
		"prompt-section", // fetch on-demand Planner reference section (ACP parity for expand_prompt_section)
	},
	// Swarm roles report their own failures so the server's retry ladder can
	// re-dispatch (audit H8: previously only abandon existed, which skips
	// retry — a worker that failed mid-attempt had no way to say so).
	"developer": {
		"status",
		"context",
		"log",
		"message",
		"task fail", // retryable failure report (reason required)
	},
	"frontend_dev": {
		"status",
		"context",
		"log",
		"message",
		"task fail",
	},
	"backend_dev": {
		"status",
		"context",
		"log",
		"message",
		"task fail",
	},
	"qa_engineer": {
		"status",
		"context",
		"log",
		"message",
		"task fail",
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

// AllowedSubcommandHeads returns the deduplicated set of first-token
// subcommands for the role, suitable for the JSON-schema enum exposed to the
// LLM. Compound entries collapse to their head (`"task retry"` → `"task"`).
func AllowedSubcommandHeads(role string) []string {
	src, ok := RoleSubcommands[role]
	if !ok {
		return nil
	}
	seen := make(map[string]struct{}, len(src))
	out := make([]string, 0, len(src))
	for _, entry := range src {
		head := entry
		if i := strings.IndexByte(entry, ' '); i >= 0 {
			head = entry[:i]
		}
		if _, ok := seen[head]; ok {
			continue
		}
		seen[head] = struct{}{}
		out = append(out, head)
	}
	return out
}

// IsAllowed reports whether the given role may invoke the given subcommand
// with the given args. Unknown roles return false.
//
// Matching rules:
//   - bare entry `"status"` matches subcommand=="status" with any args.
//   - compound entry `"task retry"` matches subcommand=="task" AND
//     len(args) >= 1 AND args[0] == "retry".
//
// args is variadic so existing call sites that pass no args still compile.
// Builds the lookup on each call — this is a rare hot path (one Tier 1 tool
// invocation per LLM turn), so the extra allocation is cheaper than the lock
// coordination a package-level cache would need.
func IsAllowed(role, subcommand string, args ...string) bool {
	src, ok := RoleSubcommands[role]
	if !ok {
		return false
	}
	// Pre-compute the compound probe so we don't do it inside the loop on
	// every entry.
	var compound string
	if len(args) > 0 && args[0] != "" {
		compound = subcommand + " " + args[0]
	}
	for _, entry := range src {
		if entry == subcommand {
			return true
		}
		if compound != "" && entry == compound {
			return true
		}
	}
	return false
}
