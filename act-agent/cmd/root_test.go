package cmd

import (
	"strings"
	"testing"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/tools"
)

// Locks the Tier-1 whitelist to the CLI router: every subcommand granted in
// RoleSubcommands must resolve either natively in root.go or through the TS
// CLI routing table. Catches the drift class where a whitelist grant + prompt
// advertisement falls through to "Unknown command" (the ACP Planner
// prompt-section dead path, kanban 2026-06-12).
func TestWhitelistSubcommandsAllRoute(t *testing.T) {
	nativeSubcommands := map[string]bool{"prompt-section": true, "reset": true}
	for role, entries := range tools.RoleSubcommands {
		for _, entry := range entries {
			head := entry
			if i := strings.IndexByte(entry, ' '); i >= 0 {
				head = entry[:i]
			}
			if !nativeSubcommands[head] && !isCLISubcommand(head) {
				t.Errorf("role %s: whitelisted subcommand %q has no route (not native, not in cliSubcommands)", role, entry)
			}
		}
	}
}
