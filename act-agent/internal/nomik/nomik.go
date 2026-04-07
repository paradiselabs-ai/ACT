// Package nomik wraps the `nomik` codebase knowledge graph CLI.
//
// Nomik is an external tool (https://nomik.co) that scans a project's source
// code and stores a graph of functions, classes, imports, call sites, etc.
// in Neo4j. It exposes commands like `nomik impact <symbol>`, `nomik rules`,
// `nomik communities`, `nomik onboard` for querying the graph.
//
// ACT agents can call these via `act codebase impact|rules|communities|onboard`
// (wired in cli/act-cli.ts). This package provides the orchestrator side:
// detect availability, run initial scans, and trigger incremental rescans
// after code changes.
//
// All operations are non-fatal — Nomik failure must NEVER block the
// orchestrator. If `nomik` isn't installed or Neo4j isn't reachable, every
// function returns gracefully and ACT continues without graph features.
package nomik

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
)

// neo4jBoltAddress is the default Neo4j Bolt port checked by IsAvailable.
// Nomik defaults to localhost:7687 (configurable via .env on the user's side).
const neo4jBoltAddress = "localhost:7687"

// IsAvailable returns true if both:
//  1. The `nomik` binary is in PATH
//  2. A TCP connection to localhost:7687 (Neo4j Bolt) succeeds within 1s
//
// Returns false silently for any failure — no errors are surfaced. Used by
// the orchestrator to decide whether to attempt project initialization.
func IsAvailable() bool {
	if _, err := exec.LookPath("nomik"); err != nil {
		return false
	}
	conn, err := net.DialTimeout("tcp", neo4jBoltAddress, 1*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// EnsureProject runs `nomik init` (idempotent) followed by `nomik scan ./` in
// the given project directory. Captures combined stdout/stderr to logs at INFO
// level on success and WARN level on failure.
//
// Non-fatal: returns the error but the orchestrator should log and continue.
func EnsureProject(ctx context.Context, projectDir string) error {
	if projectDir == "" {
		return fmt.Errorf("EnsureProject: empty projectDir")
	}

	// `nomik init` creates .nomik/project.json if missing. Idempotent — safe
	// to run on every startup.
	initCmd := exec.CommandContext(ctx, "nomik", "init")
	initCmd.Dir = projectDir
	if out, err := initCmd.CombinedOutput(); err != nil {
		// init may fail if already initialized — that's fine, fall through to scan
		logging.Debug("nomik init returned non-zero (possibly already initialized)",
			"dir", projectDir, "output", strings.TrimSpace(string(out)))
	}

	// `nomik scan ./` indexes the project. This may take a few seconds for
	// medium-sized projects. We block on it intentionally — caller is expected
	// to dispatch this in a goroutine.
	scanCmd := exec.CommandContext(ctx, "nomik", "scan", "./")
	scanCmd.Dir = projectDir
	out, err := scanCmd.CombinedOutput()
	if err != nil {
		logging.Warn("nomik scan failed", "dir", projectDir, "error", err, "output", strings.TrimSpace(string(out)))
		return fmt.Errorf("nomik scan: %w", err)
	}
	logging.Info("nomik scan complete", "dir", projectDir)
	return nil
}

// Onboard runs `nomik onboard` and returns its stdout — a high-level
// architecture summary suitable for injection into agent briefs.
func Onboard(ctx context.Context, projectDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "nomik", "onboard")
	if projectDir != "" {
		cmd.Dir = projectDir
	}
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("nomik onboard: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Rescan runs `nomik scan:incremental` to update the graph after code changes.
// Faster than a full scan; intended to be called after task completion.
func Rescan(ctx context.Context, projectDir string) error {
	cmd := exec.CommandContext(ctx, "nomik", "scan:incremental")
	if projectDir != "" {
		cmd.Dir = projectDir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		logging.Debug("nomik scan:incremental failed", "dir", projectDir, "error", err, "output", strings.TrimSpace(string(out)))
		return fmt.Errorf("nomik scan:incremental: %w", err)
	}
	logging.Debug("nomik incremental rescan complete", "dir", projectDir)
	return nil
}

// Status is a snapshot of the Nomik graph for /nomik status display.
type Status struct {
	Available    bool
	Nodes        int
	Edges        int
	LastScanTime string
	Raw          string // raw output of `nomik status` for fallback display
}

// GetStatus runs `nomik status` and parses out the headline numbers.
// Returns a Status with Available=false if Nomik isn't reachable.
func GetStatus(ctx context.Context, projectDir string) Status {
	if !IsAvailable() {
		return Status{Available: false}
	}

	cmd := exec.CommandContext(ctx, "nomik", "status")
	if projectDir != "" {
		cmd.Dir = projectDir
	}
	out, err := cmd.Output()
	if err != nil {
		return Status{Available: true, Raw: fmt.Sprintf("error: %v", err)}
	}

	raw := strings.TrimSpace(string(out))
	// Best-effort parse of "X nodes, Y edges" — Nomik's output format may vary
	// across versions, so we keep the raw output as a fallback.
	st := Status{Available: true, Raw: raw}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if n, ok := parseCount(line, "nodes"); ok {
			st.Nodes = n
		}
		if n, ok := parseCount(line, "edges"); ok {
			st.Edges = n
		}
	}
	return st
}

// parseCount extracts the integer before the first occurrence of `unit` in
// the line. Returns (0, false) if no match. Used by GetStatus to scrape
// "1583 functions, 18 communities" style output.
func parseCount(line, unit string) (int, bool) {
	idx := strings.Index(line, unit)
	if idx == -1 {
		return 0, false
	}
	// Walk backwards from idx to find the start of the number
	end := idx
	for end > 0 && line[end-1] == ' ' {
		end--
	}
	start := end
	for start > 0 && line[start-1] >= '0' && line[start-1] <= '9' {
		start--
	}
	if start == end {
		return 0, false
	}
	n := 0
	for i := start; i < end; i++ {
		n = n*10 + int(line[i]-'0')
	}
	return n, true
}
