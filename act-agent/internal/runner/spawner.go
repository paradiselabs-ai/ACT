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
	"syscall"
	"time"

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

	// Defensive: kill any orphaned runners from a previous abnormal exit
	// before spawning fresh ones. This is what prevents the "two runners
	// for the same role registering at the same time" 409 conflict storm.
	s.SweepOrphans()

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

	// External-CLI backends need their binary on PATH — fail before spawn.
	if backend == BackendClaudeCode {
		if _, err := exec.LookPath("claude"); err != nil {
			return fmt.Errorf("claude-code backend requested but `claude` not in PATH: %w", err)
		}
	}
	if backend == BackendGemini {
		if _, err := exec.LookPath("gemini"); err != nil {
			return fmt.Errorf("gemini backend requested but `gemini` not in PATH: %w", err)
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

	// Put each runner in its own process group so the parent (act TUI) can
	// kill the entire subtree by signaling the negative pgid. Without this,
	// if the TUI dies abnormally (terminal close, force quit), the runner
	// subprocesses orphan and keep polling the server forever.
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	setProcGroup(cmd.SysProcAttr)

	// Redirect stdout/stderr to a per-runner log file at ~/.act/runners/<role>.log
	// instead of inheriting the parent's stdio. The parent here is the Bubble
	// Tea TUI — letting child processes write to its terminal corrupts the
	// rendered chat with raw `[role] ...` lines from the runner.
	//
	// Failures opening the log file are non-fatal: we fall back to discarding
	// the runner's output (the runner still works, you just can't see its logs).
	var logFile *os.File
	if f, err := openRunnerLog(spec.Role); err == nil {
		_, _ = f.WriteString(fmt.Sprintf("\n=== runner session start: %s role=%s ===\n", time.Now().Format(time.RFC3339), spec.Role))
		cmd.Stdout = f
		cmd.Stderr = f
		logFile = f
	} else {
		logging.Warn("Could not open runner log file, discarding output", "role", spec.Role, "error", err)
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	if err := cmd.Start(); err != nil {
		if logFile != nil {
			_ = logFile.Close()
		}
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
	// Spawner state reflects reality. Also close the log file fd we passed
	// to the child so it isn't held open indefinitely.
	go func(role string, c *exec.Cmd, lf *os.File) {
		_ = c.Wait()
		if lf != nil {
			_, _ = lf.WriteString(fmt.Sprintf("=== runner session end: %s role=%s ===\n", time.Now().Format(time.RFC3339), role))
			_ = lf.Close()
		}
		s.mu.Lock()
		if existing, ok := s.runners[role]; ok && existing.cmd == c {
			delete(s.runners, role)
		}
		s.mu.Unlock()
		logging.Info("Swarm runner exited", "role", role)
	}(spec.Role, cmd, logFile)

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
// Each runner was started with Setpgid:true so we can kill its entire process
// group (the runner + any swarm-agent subprocesses it spawned) with one signal.
func (s *Spawner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for role, rp := range s.runners {
		if rp.cmd != nil && rp.cmd.Process != nil {
			pid := rp.cmd.Process.Pid
			// Negative pid = kill the whole process group.
			if err := killProcessGroup(pid); err != nil {
				// Fall back to single-process kill if pgkill fails.
				_ = rp.cmd.Process.Kill()
			}
		}
		delete(s.runners, role)
	}
}

// SweepOrphans kills any leftover act-runner.mjs processes from previous
// sessions before spawning new ones. Defensive cleanup against the case where
// a previous act TUI exited abnormally and left runners running. Safe to call
// even when no orphans exist — pkill returns non-zero with no matches, which
// we ignore.
func (s *Spawner) SweepOrphans() {
	cmd := exec.Command("pkill", "-f", "act-runner.mjs")
	_ = cmd.Run()
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

// openRunnerLog opens (creating if needed) the per-role runner log file at
// ~/.act/runners/<role>.log in append mode. Caller is responsible for closing
// the returned file when the subprocess exits.
func openRunnerLog(role string) (*os.File, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}
	dir := filepath.Join(home, ".act", "runners")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir runners: %w", err)
	}
	path := filepath.Join(dir, role+".log")
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
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
