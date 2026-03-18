// Package act provides a thin client for the ACT coordination server.
// It shells out to the `act` CLI (cli/act-cli.ts) rather than implementing
// HTTP calls directly, avoiding code duplication with the existing 21-command CLI.
package act

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/opencode-ai/opencode/internal/logging"
)

// Client wraps the ACT CLI for coordination server communication.
type Client struct {
	// CLIPath is the path to act-cli.ts (resolved at init from ACT_CLI_PATH or default)
	CLIPath string
	// ServerURL is the ACT server URL (from ACT_SERVER_URL env var)
	ServerURL string
	// AgentID is this agent's registered ID
	AgentID string
	// Project is the current project name
	Project string
}

// NewClient creates a new ACT client. It resolves the CLI path from
// ACT_CLI_PATH env var, falling back to ../cli/act-cli.ts relative to the binary.
func NewClient(agentID, project string) *Client {
	cliPath := os.Getenv("ACT_CLI_PATH")
	if cliPath == "" {
		// Default: act-cli.ts relative to the act-agent binary's parent
		cliPath = "cli/act-cli.ts"
	}
	serverURL := os.Getenv("ACT_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	return &Client{
		CLIPath:   cliPath,
		ServerURL: serverURL,
		AgentID:   agentID,
		Project:   project,
	}
}

// run executes an act CLI command and returns stdout output.
func (c *Client) run(args ...string) (string, error) {
	cmd := exec.Command("npx", append([]string{"tsx", c.CLIPath}, args...)...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("ACT_SERVER_URL=%s", c.ServerURL),
		fmt.Sprintf("ACT_AGENT_ID=%s", c.AgentID),
	)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		logging.Warn("ACT CLI command failed", "args", args, "error", err, "output", output)
		return output, fmt.Errorf("act %s: %w (%s)", strings.Join(args, " "), err, output)
	}
	return output, nil
}

// Register registers this agent with the ACT coordination server.
func (c *Client) Register() error {
	_, err := c.run("register", c.AgentID)
	if err != nil && strings.Contains(err.Error(), "already registered") {
		// Already registered is fine — idempotent
		logging.Info("Agent already registered", "agent_id", c.AgentID)
		return nil
	}
	return err
}

// GetContext fetches the full agent context (brief, task, parallel agents, messages).
func (c *Client) GetContext() (string, error) {
	return c.run("context", c.AgentID, "--project", c.Project)
}

// ReportProgress updates task progress percentage.
func (c *Client) ReportProgress(taskID string, percent int) error {
	_, err := c.run("task", "progress", taskID,
		"--agent-id", c.AgentID,
		"--percent", fmt.Sprintf("%d", percent))
	return err
}

// ReportComplete marks a task as complete with a result summary.
func (c *Client) ReportComplete(taskID string, result string) error {
	_, err := c.run("task", "complete", taskID,
		"--agent-id", c.AgentID,
		"--result", result)
	return err
}

// ClaimFiles claims exclusive editing rights on files.
func (c *Client) ClaimFiles(taskID string, files []string) error {
	args := []string{"files", "claim", "--agent-id", c.AgentID, "--task-id", taskID}
	args = append(args, files...)
	_, err := c.run(args...)
	return err
}

// ReleaseFiles releases file locks.
func (c *Client) ReleaseFiles(files []string) error {
	args := []string{"files", "release", "--agent-id", c.AgentID}
	args = append(args, files...)
	_, err := c.run(args...)
	return err
}

// SendMessage sends a coordination message to other agents.
func (c *Client) SendMessage(text string) error {
	_, err := c.run("message", text, "--agent-id", c.AgentID)
	return err
}

// SubmitForValidation submits a completed task for Assurance review.
func (c *Client) SubmitForValidation(taskID string) error {
	_, err := c.run("task", "submit-for-validation",
		"--task-id", taskID,
		"--agent-id", c.AgentID)
	return err
}

// Status fetches the ACT system status.
func (c *Client) Status() (string, error) {
	return c.run("status")
}

// PVMSearch searches coordination memory for relevant patterns.
func (c *Client) PVMSearch(query string) (string, error) {
	return c.run("pvm", "search", query)
}

// IsAvailable returns true if the ACT server is reachable.
func (c *Client) IsAvailable() bool {
	_, err := c.run("status")
	return err == nil
}
