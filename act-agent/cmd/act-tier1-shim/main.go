// Command act-tier1-shim is the per-role `act` CLI wrapper exposed to
// ACP-backed Tier 1 agents (Planner / Observer / Assurance / QA-Synthesizer).
//
// It is invoked indirectly, by symlink — one symlink per role:
//
//	act-tier1-planner        →  act-tier1-shim
//	act-tier1-observer       →  act-tier1-shim
//	act-tier1-assurance      →  act-tier1-shim
//	act-tier1-qa_synthesizer →  act-tier1-shim
//
// The shim:
//
//  1. Reads its own argv[0] to determine the calling role
//     (act-tier1-planner → planner).
//  2. Validates argv[1] (the subcommand) against the role's allowlist —
//     same RoleSubcommands map the in-process act_cli tool uses (one source
//     of truth in internal/llm/tools/act_cli_whitelist.go).
//  3. Validates argv[2:] for the banned shell metacharacters from the
//     in-process tool (no ;, |, &&, ||, $(, `). The shim does not run a
//     shell — these can't inject — but rejecting them keeps the agent's
//     usage pattern aligned with the in-process contract.
//  4. exec.Commands the real `act` binary with the remaining args.
//
// Hard enforcement preserved: the shim, not the LLM, decides what runs.
// The ACP-backed agent's PATH is configured to expose only its role's
// symlink, so a Planner cannot invoke Assurance-only subcommands.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/tools"
)

const (
	exitUsage      = 2
	exitNotAllowed = 2
	exitBadArg     = 2
	exitExecFail   = 127
)

// bannedSubstrings mirrors actCLIBannedArgSubstrings in
// internal/llm/tools/act_cli.go. Kept in sync manually — the in-process
// tool's list is unexported and Go does not allow importing unexported
// identifiers; both lists exist for the same reason (LLM-misuse signal,
// not actual shell-injection defence, since neither call site runs a
// shell).
var bannedSubstrings = []string{";", "|", "&&", "||", "$(", "`"}

func main() {
	role, err := roleFromArgv0(os.Args[0])
	if err != nil {
		fail(exitUsage, "act-tier1-shim: %v", err)
	}

	if len(os.Args) < 2 {
		fail(exitUsage, "act-tier1-%s: usage: act-tier1-%s <subcommand> [args...]", role, role)
	}
	subcommand := os.Args[1]
	args := os.Args[2:]

	if !tools.IsAllowed(role, subcommand, args...) {
		got := subcommand
		if len(args) > 0 && args[0] != "" {
			got = subcommand + " " + args[0]
		}
		fail(exitNotAllowed,
			"act-tier1-%s: %q is not allowed for role %s.\nAllowed: %s",
			role, got, role, strings.Join(tools.AllowedFor(role), ", "))
	}

	for _, a := range args {
		for _, bad := range bannedSubstrings {
			if strings.Contains(a, bad) {
				fail(exitBadArg,
					"act-tier1-%s: argument %q contains banned substring %q (use the act CLI's own argument shape, not shell composition)",
					role, a, bad)
			}
		}
	}

	if err := runAct(subcommand, args); err != nil {
		// runAct already wrote the child's stderr to ours; surface a one-line
		// signal and inherit the child's exit code if we have it.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fail(exitExecFail, "act-tier1-%s: exec act: %v", role, err)
	}
}

// roleFromArgv0 extracts the role name from the invoking argv[0]. Tolerates
// absolute paths, relative paths, and a bare basename. Examples:
//
//	"/usr/local/bin/act-tier1-planner"  →  "planner"
//	"./act-tier1-observer"              →  "observer"
//	"act-tier1-qa_synthesizer"          →  "qa_synthesizer"
//
// Returns an error for any name that doesn't start with the expected
// prefix — protects against the shim being copied to a non-conventional
// name and silently running with no role at all.
func roleFromArgv0(argv0 string) (string, error) {
	base := filepath.Base(argv0)
	const prefix = "act-tier1-"
	if !strings.HasPrefix(base, prefix) {
		return "", fmt.Errorf("expected argv0 basename to start with %q, got %q", prefix, base)
	}
	role := strings.TrimPrefix(base, prefix)
	if role == "" {
		return "", fmt.Errorf("empty role in argv0 %q", base)
	}
	// Sanity-check against the allowlist map. Unknown roles are a
	// configuration error, not a security issue per se — but we'd rather
	// fail loud here than dispatch to an empty allowlist later.
	if tools.AllowedFor(role) == nil {
		return "", fmt.Errorf("unknown role %q (no entries in RoleSubcommands)", role)
	}
	return role, nil
}

// runAct execs the real `act` binary (whatever's on PATH, typically the
// symlink at /opt/homebrew/bin/act → act-agent). stdin/stdout/stderr are
// passed through so the calling ACP-backed agent sees the same output it
// would running `act` directly.
func runAct(subcommand string, args []string) error {
	full := append([]string{subcommand}, args...)
	cmd := exec.Command("act", full...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func fail(code int, format string, args ...any) {
	fmt.Fprintln(io.Writer(os.Stderr), fmt.Sprintf(format, args...))
	os.Exit(code)
}
