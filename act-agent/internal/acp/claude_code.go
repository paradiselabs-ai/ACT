package acp

// claudeCodeDefaults returns the spawn argv for the Claude Code ACP host.
//
// Verified against the published package @agentclientprotocol/claude-agent-acp
// (Apache-2.0, latest 0.37.0 as of May 2026). Two spawn paths:
//
//  1. Globally installed: `claude-agent-acp` is on PATH after
//     `npm i -g @agentclientprotocol/claude-agent-acp`. One-process spawn.
//  2. On-demand via npx: `npx --yes -p @agentclientprotocol/claude-agent-acp@^0.37 claude-agent-acp`
//     — works for users who have not installed it, at the cost of a cold-start
//     hit on the first launch. After npx caches the package, subsequent spawns
//     are fast.
//
// We default to the npx form because it requires zero setup. Users who want
// the snappy path can override ACPConfig.Command to "claude-agent-acp" after
// `npm i -g`.
//
// Framing verified empirically: newline-delimited JSON-RPC (not LSP-style
// Content-Length). The agent rejects Content-Length framing with a JSON
// parser error.
func claudeCodeDefaults() (command string, args []string) {
	return "npx", []string{
		"--yes",
		"-p", "@agentclientprotocol/claude-agent-acp@^0.37",
		"claude-agent-acp",
	}
}
