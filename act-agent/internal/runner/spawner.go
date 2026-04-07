// Package runner provides a helper for spawning and managing Runner
// subprocesses that poll the ACT server for tasks and dispatch them to
// swarm agents (Tier 2).
//
// The Spawner manages a SET of Runner processes — one per swarm role —
// so multiple agents can work in parallel. This is the difference between
// "queue" and "swarm".
package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
)

// runnerProcess tracks a single live Runner subprocess.
type runnerProcess struct {
	spec    SwarmRoleSpec
	cmd     *exec.Cmd
	pid     int
	started bool
}

// Spawner manages the swarm Runner processes' lifecycle. Each Runner is a
// Node.js process (act-runner.mjs) that polls the ACT server for assigned
// tasks and spawns swarm agent subprocess (either act-agent or claude-code).
//
// Multi-runner: one process per swarm role, all alive concurrently.
type Spawner struct {
	mu      sync.Mutex
	runners map[string]*runnerProcess // role → process
}

// NewSpawner creates an idle Spawner. Call StartSwarm() to launch processes.
func NewSpawner() *Spawner {
	return &Spawner{
		runners: make(map[string]*runnerProcess),
	}
}

// StartSwarm launches one Runner subprocess per spec. Idempotent per-role:
// if a runner for that role is already alive, it's left alone.
//
// Each runner is started with --agent-id, --name, --role, --capabilities, and
// --backend flags. The Runner script self-registers with the ACT server using
// these values.
//
// Errors are logged but non-fatal — the orchestrator continues even if some
// or all runners fail to start. The TUI is still usable; the swarm just won't
// execute tasks.
func (s *Spawner) StartSwarm(specs []SwarmRoleSpec) error {
	if len(specs) == 0 {
		return fmt.Errorf("StartSwarm: no specs provided")
	}

	scriptPath, err := findRunnerScript()
	if err != nil {
		return err
	}

	nodeBin, err := exec.LookPath("node")
	if err != nil {
		return fmt.Errorf("node not found in PATH: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, spec := range specs {
		if _, exists := s.runners[spec.Role]; exists {
			logging.Debug("Swarm runner already running, skipping", "role", spec.Role)
			continue
		}
		if err := s.startOneLocked(nodeBin, scriptPath, spec); err != nil {
			logging.Warn("Failed to start swarm runner", "role", spec.Role, "error", err)
			continue
		}
	}
	return nil
}

// startOneLocked spawns a single Runner. Caller must hold s.mu.
func (s *Spawner) startOneLocked(nodeBin, scriptPath string, spec SwarmRoleSpec) error {
	// Validate backend
	backend := spec.Backend
	if backend == "" {
		backend = BackendActAgent
	}
	if !IsValidBackend(backend) {
		return fmt.Errorf("invalid backend %q for role %q", backend, spec.Role)
	}

	// If claude-code requested, verify the binary is available
	if backend == BackendClaudeCode {
		if _, err := exec.LookPath("claude"); err != nil {
			return fmt.Errorf("claude-code backend requested but `claude` not in PATH: %w", err)
		}
	}

	args := []string{
		scriptPath,
		"--agent-id", spec.AgentID,
		"--name", spec.Name,
		"--role", spec.Role,
		"--backend", backend,
	}
	if len(spec.Capabilities) > 0 {
		args = append(args, "--capabilities", strings.Join(spec.Capabilities, ","))
	}

	cmd := exec.Command(nodeBin, args...)
	cmd.Env = os.Environ()
	// Inherit stdout/stderr so Runner logs are visible in the same terminal
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start runner: %w", err)
	}

	rp := &runnerProcess{
		spec:    spec,
		cmd:     cmd,
		pid:     cmd.Process.Pid,
		started: true,
	}
	s.runners[spec.Role] = rp

	logging.Info("Spawned swarm runner",
		"role", spec.Role,
		"agent_id", spec.AgentID,
		"backend", backend,
		"pid", rp.pid)

	// Reap the process when it exits so we don't leak zombies and the
	// Spawner state reflects reality.
	go func(role string, c *exec.Cmd) {
		_ = c.Wait()
		s.mu.Lock()
		if existing, ok := s.runners[role]; ok && existing.cmd == c {
			delete(s.runners, role)
		}
		s.mu.Unlock()
		logging.Info("Swarm runner exited", "role", role)
	}(spec.Role, cmd)

	return nil
}

// RestartRole kills the runner for a given role and starts a new one with
// the new spec. If no runner exists for that role, just starts a new one.
// Used by the slash command /swarm <role> <backend>.
func (s *Spawner) RestartRole(spec SwarmRoleSpec) error {
	s.mu.Lock()
	if existing, ok := s.runners[spec.Role]; ok && existing.cmd != nil && existing.cmd.Process != nil {
		_ = existing.cmd.Process.Kill()
		delete(s.runners, spec.Role)
	}
	s.mu.Unlock()

	scriptPath, err := findRunnerScript()
	if err != nil {
		return err
	}
	nodeBin, err := exec.LookPath("node")
	if err != nil {
		return fmt.Errorf("node not found in PATH: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startOneLocked(nodeBin, scriptPath, spec)
}

// Stop kills all runner subprocesses. Safe to call when no runners are alive.
func (s *Spawner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for role, rp := range s.runners {
		if rp.cmd != nil && rp.cmd.Process != nil {
			_ = rp.cmd.Process.Kill()
		}
		delete(s.runners, role)
	}
}

// IsRunning returns true if any runner is currently alive.
func (s *Spawner) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.runners) > 0
}

// RunnerStatus is a snapshot of one runner process for /swarm status display.
type RunnerStatus struct {
	Role    string
	AgentID string
	Backend string
	Model   string
	PID     int
	Alive   bool
}

// Status returns a snapshot of all runners' state. Used by /swarm status,
// /swarm list, and the act swarm CLI command.
func (s *Spawner) Status() []RunnerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]RunnerStatus, 0, len(s.runners))
	for _, rp := range s.runners {
		out = append(out, RunnerStatus{
			Role:    rp.spec.Role,
			AgentID: rp.spec.AgentID,
			Backend: rp.spec.Backend,
			Model:   rp.spec.Model,
			PID:     rp.pid,
			Alive:   rp.started,
		})
	}
	return out
}

func findRunnerScript() (string, error) {
	// 1. Explicit env override
	if p := os.Getenv("ACT_RUNNER_PATH"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// 2. Relative to the executable
	execPath, err := os.Executable()
	if err == nil {
		// Resolve symlinks (the act binary is typically symlinked to /opt/homebrew/bin/act)
		resolved, rerr := filepath.EvalSymlinks(execPath)
		if rerr != nil {
			resolved = execPath
		}
		dir := filepath.Dir(resolved)

		candidates := []string{
			filepath.Join(dir, "runner", "act-runner.mjs"),
			filepath.Join(dir, "..", "runner", "act-runner.mjs"),
			filepath.Join(dir, "..", "act-agent", "runner", "act-runner.mjs"),
			filepath.Join(dir, "internal", "runner", "act-runner.mjs"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				abs, _ := filepath.Abs(c)
				return abs, nil
			}
		}
	}

	// 3. Relative to cwd (last resort)
	for _, c := range []string{"runner/act-runner.mjs", "act-agent/runner/act-runner.mjs", "act-agent/internal/runner/act-runner.mjs"} {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs, nil
		}
	}

	return "", fmt.Errorf("act-runner.mjs not found (set ACT_RUNNER_PATH to override)")
}
