// Package server provides auto-launch for the ACT coordination server.
// The `act` command checks if the server is running and starts it if needed.
package server

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	// Try common locations relative to cwd
	candidates := []string{
		"server/src/index.ts",
		"../server/src/index.ts",
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	for _, c := range candidates {
		full := filepath.Join(cwd, c)
		if _, err := os.Stat(full); err == nil {
			return full
		}
	}
	return ""
}
