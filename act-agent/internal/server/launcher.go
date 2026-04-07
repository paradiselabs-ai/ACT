// Package server provides auto-launch for the ACT coordination server.
// The `act` command checks if the server is running and starts it if needed.
package server

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
func EnsureServerRunning(serverURL string) error {
	if serverURL == "" {
		serverURL = os.Getenv("ACT_SERVER_URL")
		if serverURL == "" {
			serverURL = defaultServerURL
		}
	}

	// Check if already running
	if isHealthy(serverURL) {
		logging.Info("ACT server already running", "url", serverURL)
		return nil
	}

	// Find server entry point relative to the working directory
	serverScript := findServerScript()
	if serverScript == "" {
		logging.Warn("ACT server script not found — server must be started manually")
		return nil // Not fatal — agent can still work without server
	}

	logging.Info("Starting ACT server", "script", serverScript)

	// Start server as detached background process
	cmd := exec.Command("npx", "tsx", serverScript)
	cmd.Dir = filepath.Dir(serverScript)
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

	// Poll for health
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
