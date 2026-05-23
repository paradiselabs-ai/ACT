// Package server provides auto-launch for the ACT coordination server.
// The `act` command checks if the server is running and starts it if needed.
// Uses a PID file + health check to prevent concurrent starts.
package server

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
)

const (
	defaultServerURL = "http://localhost:8080"
	healthPath       = "/health"
	pollInterval     = 500 * time.Millisecond
	startTimeout     = 15 * time.Second
)

// EnsureServerRunning checks if the ACT server is accessible.
// If not, it starts the server as a background process and waits for it to be healthy.
// Uses a PID file to detect stale processes and prevent concurrent start races.
func EnsureServerRunning(serverURL string) error {
	if serverURL == "" {
		serverURL = os.Getenv("ACT_SERVER_URL")
		if serverURL == "" {
			serverURL = defaultServerURL
		}
	}

	// Find server entry point relative to the working directory
	serverScript := findServerScript()
	if serverScript == "" {
		// Without a script we can't sweep duplicates by pattern. Fall back to a
		// pure health check and hope the caller has started something compatible.
		if isHealthy(serverURL) {
			logging.Debug("ACT server already running", "url", serverURL)
			return nil
		}
		logging.Warn("ACT server script not found — server must be started manually")
		return nil // Not fatal — agent can still work without server
	}

	// Sweep stale tsx watch processes for our script BEFORE accepting the
	// "already running" happy path. Zombie tsx servers from prior sessions can
	// keep responding to /health long after their in-memory state has drifted
	// (stale agents, stale tasks, lost ChronLog correlation). The PID file is
	// the source of truth: any tsx watch process for our script whose PID is
	// NOT the one written to the PID file is a leftover and must die.
	serverDir := filepath.Dir(filepath.Dir(serverScript)) // server/ (parent of src/)
	pidFile := filepath.Join(serverDir, "data", "act-server.pid")
	authoritativePID := readPIDFile(pidFile)
	killStaleServerProcesses(authoritativePID)

	// After the sweep, health-check again. If a legitimate server still owns
	// the port, we attach to it; otherwise we fall through to start fresh.
	if isHealthy(serverURL) {
		logging.Debug("ACT server already running", "url", serverURL, "authoritative_pid", authoritativePID)
		return nil
	}

	if isServerProcessAlive(pidFile) {
		// Process exists but health check failed — server is starting up.
		// Wait for it instead of spawning a second one.
		logging.Info("ACT server process exists (PID file found), waiting for health...")
		return waitForHealth(serverURL)
	}

	logging.Info("Starting ACT server", "script", serverScript)

	// Start server as detached background process
	cmd := exec.Command("npx", "tsx", serverScript)
	// Set cwd to server/ (parent of src/) so process.cwd() paths resolve correctly.
	cmd.Dir = filepath.Dir(filepath.Dir(serverScript))
	cmd.Stdout = nil // Discard output
	cmd.Stderr = nil
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ACT server: %w", err)
	}

	// Detach — don't wait for the process
	go func() {
		_ = cmd.Wait()
	}()

	return waitForHealth(serverURL)
}

// waitForHealth polls the server health endpoint until it responds or times out.
func waitForHealth(serverURL string) error {
	deadline := time.Now().Add(startTimeout)
	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)
		if isHealthy(serverURL) {
			logging.Info("ACT server is ready", "url", serverURL)
			return nil
		}
	}
	return fmt.Errorf("ACT server did not become healthy within %s", startTimeout)
}

// readPIDFile reads the PID file and returns the recorded PID, or 0 if the
// file is missing/unreadable/empty. Used as the "authoritative" PID for the
// stale-server sweep — any other tsx watch process for our script is fair game.
func readPIDFile(pidFile string) int {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0
	}
	return pid
}

// killStaleServerProcesses finds any tsx watch process running our server
// script and kills it UNLESS its PID matches the authoritative PID. Mirrors
// the runner-side SweepOrphans pattern at internal/runner/spawner.go:243.
//
// When the PID file is missing or stale (authoritativePID==0), every tsx
// watch process for our script is treated as stale. Cold-start path.
//
// Uses pgrep+kill rather than pkill because pkill on macOS lacks -P / process
// filtering and would match too broadly. We narrow to "tsx watch.*src/index.ts"
// which is specific to the ACT server entry point.
func killStaleServerProcesses(authoritativePID int) {
	out, err := exec.Command("pgrep", "-f", "tsx watch.*src/index.ts").Output()
	if err != nil {
		// pgrep exits 1 with no match — that's the no-op happy path.
		return
	}
	killed := 0
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		pid, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || pid <= 0 {
			continue
		}
		if authoritativePID > 0 && pid == authoritativePID {
			continue
		}
		// SIGKILL — these are zombies from prior sessions, SIGTERM may not stick.
		proc, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		_ = proc.Signal(syscall.SIGKILL)
		killed++
	}
	if killed > 0 {
		logging.Info("Swept stale ACT server processes", "killed", killed, "authoritative_pid", authoritativePID)
		// Give the kernel a moment to release the port before the next health check.
		time.Sleep(500 * time.Millisecond)
	}
}

// isServerProcessAlive reads the PID file and checks if that process is still running.
// Returns false if the PID file doesn't exist, is unreadable, or the process is dead.
func isServerProcessAlive(pidFile string) bool {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return false
	}
	// Signal 0 checks if the process exists without actually sending a signal.
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func isHealthy(serverURL string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(serverURL + healthPath)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func findServerScript() string {
	// Strategy 1: ~/.act/config.json stores `actRoot` — the canonical location
	// of the ACT repo on this machine. Set during install/first-run. This is
	// the authoritative source and works no matter what cwd `act` was launched
	// from (which is the common case — users run `act` in their own projects,
	// not inside the ACT repo).
	if home, err := os.UserHomeDir(); err == nil {
		cfgPath := filepath.Join(home, ".act", "config.json")
		if data, err := os.ReadFile(cfgPath); err == nil {
			// Lightweight parse: avoid pulling in encoding/json for one field.
			// Look for `"actRoot": "..."`.
			if root := extractJSONString(string(data), "actRoot"); root != "" {
				candidate := filepath.Join(root, "server", "src", "index.ts")
				if _, err := os.Stat(candidate); err == nil {
					return candidate
				}
			}
		}
	}

	// Strategy 2: walk up from the running binary's directory looking for
	// a sibling `server/src/index.ts`. This handles dev installs where the
	// binary lives in `<repo>/act-agent/act-agent`.
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		// Resolve symlinks (the global `act` is symlinked into homebrew bin)
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			dir = filepath.Dir(resolved)
		}
		for i := 0; i < 5; i++ {
			candidate := filepath.Join(dir, "server", "src", "index.ts")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Strategy 3: cwd-relative fallback (only useful when running from inside
	// the ACT repo itself, e.g. `cd /path/to/act && act`).
	if cwd, err := os.Getwd(); err == nil {
		for _, c := range []string{"server/src/index.ts", "../server/src/index.ts"} {
			full := filepath.Join(cwd, c)
			if _, err := os.Stat(full); err == nil {
				return full
			}
		}
	}

	return ""
}

// extractJSONString pulls a top-level string field from a JSON document
// without requiring encoding/json. Returns empty string if not found.
// Tolerates simple whitespace; does NOT handle escapes inside the value.
func extractJSONString(doc, key string) string {
	needle := `"` + key + `"`
	idx := strings.Index(doc, needle)
	if idx == -1 {
		return ""
	}
	rest := doc[idx+len(needle):]
	colon := strings.Index(rest, ":")
	if colon == -1 {
		return ""
	}
	rest = rest[colon+1:]
	open := strings.Index(rest, `"`)
	if open == -1 {
		return ""
	}
	rest = rest[open+1:]
	close := strings.Index(rest, `"`)
	if close == -1 {
		return ""
	}
	return rest[:close]
}
