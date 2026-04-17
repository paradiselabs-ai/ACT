package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/app"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/db"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/format"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/agent"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/pubsub"
	actserver "github.com/paradiselabs-ai/ACT/act-agent/internal/server"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "act",
	Short: "ACT — Agent Coordination Toolkit",
	Long: `ACT — the multi-agent coordination TUI.
Launches a single window hosting four Tier 1 agents (Planner, Observer,
Assurance, QA/Synthesizer) over a parallel swarm of headless workers.
The TUI is the harness; there is no separate orchestrator process.`,
	Example: `
  # Launch the ACT TUI in the current directory
  act

  # Launch the TUI for a specific project (cd into the project dir first)
  act --project my-app

  # Headless worker mode (spawned by the swarm runner — not for users)
  act --agent dev-1 --role developer -p "implement auth"

  # Single non-interactive prompt (returns the response on stdout and exits)
  act -p "Explain this codebase"
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Route CLI subcommands (e.g., `act context`, `act task complete`, `act files claim`,
		// `act swarm set`, `act nomik enable`) to the TypeScript CLI. This prevents agents
		// from accidentally launching the TUI when they run `act <subcommand>` via their
		// bash tool.
		if len(args) > 0 && isCLISubcommand(args[0]) {
			return routeToCLI(args)
		}

		// If the help flag is set, show the help message
		if cmd.Flag("help").Changed {
			cmd.Help()
			return nil
		}
		if cmd.Flag("version").Changed {
			fmt.Println(version.Version)
			return nil
		}

		// Load the config
		debug, _ := cmd.Flags().GetBool("debug")
		cwd, _ := cmd.Flags().GetString("cwd")
		prompt, _ := cmd.Flags().GetString("prompt")
		outputFormat, _ := cmd.Flags().GetString("output-format")
		quiet, _ := cmd.Flags().GetBool("quiet")
		projectFlag, _ := cmd.Flags().GetString("project")

		// Validate format option
		if !format.IsValid(outputFormat) {
			return fmt.Errorf("invalid format option: %s\n%s", outputFormat, format.GetHelpText())
		}

		if cwd != "" {
			err := os.Chdir(cwd)
			if err != nil {
				return fmt.Errorf("failed to change directory: %v", err)
			}
		}
		if cwd == "" {
			c, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %v", err)
			}
			cwd = c
		}
		startupT := time.Now()
		stepT := startupT

		_, err := config.Load(cwd, debug)
		if err != nil {
			return err
		}
		logging.InfoPersist("startup:config.Load", "elapsed", time.Since(stepT))
		stepT = time.Now()

		// Resolve project name: explicit --project flag wins, else basename of cwd.
		// All orchestrator/client code reads this via os.Getenv("ACT_PROJECT").
		if os.Getenv("ACT_PROJECT") == "" {
			projectName := projectFlag
			if projectName == "" {
				projectName = filepath.Base(cwd)
			}
			os.Setenv("ACT_PROJECT", projectName)
		}

		// Ensure the ACT coordination server is running. Idempotent: returns
		// immediately if a server is already healthy at ACT_SERVER_URL,
		// otherwise spawns one in the background and waits for /health.
		// Non-fatal — agents can still partially work without a server, but
		// the swarm Runners will fail to register and we'll log it.
		if err := actserver.EnsureServerRunning(""); err != nil {
			logging.Warn("ACT server auto-start failed", "error", err)
		}
		logging.InfoPersist("startup:EnsureServerRunning", "elapsed", time.Since(stepT))
		stepT = time.Now()

		// Connect DB, this will also run migrations
		conn, err := db.Connect()
		if err != nil {
			return err
		}
		logging.InfoPersist("startup:db.Connect", "elapsed", time.Since(stepT))
		stepT = time.Now()

		// Create main context for the application
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		app, err := app.New(ctx, conn)
		if err != nil {
			logging.Error("Failed to create app: %v", err)
			return err
		}
		logging.InfoPersist("startup:app.New", "elapsed", time.Since(stepT))
		stepT = time.Now()

		// Defer shutdown here so it runs for both interactive and non-interactive modes
		defer app.Shutdown()

		// Initialize MCP tools early for both modes
		initMCPTools(ctx, app)
		logging.InfoPersist("startup:initMCPTools", "elapsed", time.Since(stepT))
		logging.InfoPersist("startup:total-pre-mode", "elapsed", time.Since(startupT))

		// ACT agent mode (headless, JSON stdout, wired to act CLI)
		agentID, _ := cmd.Flags().GetString("agent")
		agentRole, _ := cmd.Flags().GetString("role")
		if agentID != "" {
			if prompt == "" {
				return fmt.Errorf("--agent requires --prompt")
			}
			return app.RunAgent(ctx, prompt, agentID, agentRole)
		}

		// Non-interactive mode
		if prompt != "" {
			// Run non-interactive flow using the App method
			return app.RunNonInteractive(ctx, prompt, outputFormat, quiet)
		}

		// Interactive mode — launch the ACT TUI.
		// The TUI hosts the 4 Tier 1 agents as in-process goroutines and the
		// orchestrator spawns the Tier 2 swarm via runner.Spawner. The TUI
		// is the harness; there is no separate orchestrator process.
		return runTUI(app, ctx)
	},
}

// runTUI launches the ACT TUI (the Bubble Tea interface that hosts the
// Tier 1 agents and the orchestrator).
func runTUI(a *app.App, ctx context.Context) error {
	stepT := time.Now()
	autoFitTerminal(config.Get())
	logging.InfoPersist("startup:autoFitTerminal", "elapsed", time.Since(stepT))
	stepT = time.Now()

	zone.NewGlobal()
	program := tea.NewProgram(
		tui.New(a),
	)
	logging.InfoPersist("startup:tui.New+NewProgram", "elapsed", time.Since(stepT))
	stepT = time.Now()

	ch, cancelSubs := setupSubscriptions(a, ctx)
	logging.InfoPersist("startup:setupSubscriptions", "elapsed", time.Since(stepT))

	tuiCtx, tuiCancel := context.WithCancel(ctx)
	var tuiWg sync.WaitGroup
	tuiWg.Add(1)

	go func() {
		defer tuiWg.Done()
		defer logging.RecoverPanic("TUI-message-handler", func() {
			attemptTUIRecovery(program)
		})

		for {
			select {
			case <-tuiCtx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				program.Send(msg)
			}
		}
	}()

	cleanup := func() {
		a.Shutdown()
		cancelSubs()
		tuiCancel()
		tuiWg.Wait()
	}

	// macOS window-close (red X) and most terminal-close paths send SIGHUP;
	// container/service shutdowns send SIGTERM. Without trapping these the
	// process dies before deferred app.Shutdown runs, leaving dirty DB WAL
	// and stale PID locks that starve the next launch. Translate them into
	// a normal Bubbletea quit so program.Run() returns and cleanup executes.
	// SIGINT stays with Bubbletea's built-in handler (ctrl+c → quit).
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		sig, ok := <-sigCh
		if !ok {
			return
		}
		logging.Info("signal received, quitting TUI cleanly", "signal", sig.String())
		program.Quit()
	}()

	_, err := program.Run()
	cleanup()
	return err
}

// autoFitTerminal grows the user's terminal window to at least the minimum
// size the ACT TUI needs to render the banner, side panel, and chat without
// clipping. Default on; users can opt out via tui.autoFit.disabled in
// ~/.act.json.
//
// Mechanism: emit CSI 8 ; rows ; cols t (XTWINOPS DECSLPP) to stdout. Honored
// by iTerm2, Kitty, Ghostty, WezTerm, and macOS Terminal.app. Silently
// ignored elsewhere. We skip the resize entirely in environments where
// intercepting it breaks more than it helps (multiplexers, remote VS Code
// terminals — those trap/rewrite escapes).
//
// We only ever grow the window. If the user already has a larger terminal,
// we leave it alone.
func autoFitTerminal(cfg *config.Config) {
	if cfg == nil {
		return
	}
	fit := cfg.TUI.AutoFit
	if fit != nil && fit.Disabled {
		return
	}
	if !isTerminal(os.Stdout) {
		return
	}
	if os.Getenv("TMUX") != "" {
		return
	}
	if term := os.Getenv("TERM"); strings.HasPrefix(term, "screen") {
		return
	}
	if os.Getenv("TERM_PROGRAM") == "vscode" {
		return
	}

	minCols, minRows := 160, 58
	if fit != nil {
		if fit.MinCols > 0 {
			minCols = fit.MinCols
		}
		if fit.MinRows > 0 {
			minRows = fit.MinRows
		}
	}

	curCols, curRows := terminalSize(os.Stdout)
	if curCols == 0 || curRows == 0 {
		return
	}

	targetCols, targetRows := curCols, curRows
	if curCols < minCols {
		targetCols = minCols
	}
	if curRows < minRows {
		targetRows = minRows
	}
	if targetCols == curCols && targetRows == curRows {
		return
	}
	fmt.Fprintf(os.Stdout, "\x1b[8;%d;%dt", targetRows, targetCols)
}

// terminalSize returns (cols, rows) via TIOCGWINSZ. Returns 0,0 on failure.
func terminalSize(f *os.File) (int, int) {
	ws := struct {
		Row, Col, Xpixel, Ypixel uint16
	}{}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		f.Fd(),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 {
		return 0, 0
	}
	return int(ws.Col), int(ws.Row)
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// attemptTUIRecovery tries to recover the TUI after a panic
func attemptTUIRecovery(program *tea.Program) {
	logging.Info("Attempting to recover TUI after panic")

	// We could try to restart the TUI or gracefully exit
	// For now, we'll just quit the program to avoid further issues
	program.Quit()
}

func initMCPTools(ctx context.Context, app *app.App) {
	go func() {
		defer logging.RecoverPanic("MCP-goroutine", nil)

		// Create a context with timeout for the initial MCP tools fetch
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Set this up once with proper error handling
		agent.GetMcpTools(ctxWithTimeout, app.Permissions)
		logging.Info("MCP message handling goroutine exiting")
	}()
}

func setupSubscriber[T any](
	ctx context.Context,
	wg *sync.WaitGroup,
	name string,
	subscriber func(context.Context) <-chan pubsub.Event[T],
	outputCh chan<- tea.Msg,
) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer logging.RecoverPanic(fmt.Sprintf("subscription-%s", name), nil)

		subCh := subscriber(ctx)

		for {
			select {
			case event, ok := <-subCh:
				if !ok {
					logging.Info("subscription channel closed", "name", name)
					return
				}

				var msg tea.Msg = event

				select {
				case outputCh <- msg:
				case <-time.After(2 * time.Second):
					logging.Warn("message dropped due to slow consumer", "name", name)
				case <-ctx.Done():
					logging.Info("subscription cancelled", "name", name)
					return
				}
			case <-ctx.Done():
				logging.Info("subscription cancelled", "name", name)
				return
			}
		}
	}()
}

func setupSubscriptions(app *app.App, parentCtx context.Context) (chan tea.Msg, func()) {
	ch := make(chan tea.Msg, 100)

	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(parentCtx) // Inherit from parent context

	setupSubscriber(ctx, &wg, "logging", logging.Subscribe, ch)
	setupSubscriber(ctx, &wg, "sessions", app.Sessions.Subscribe, ch)
	setupSubscriber(ctx, &wg, "messages", app.Messages.Subscribe, ch)
	setupSubscriber(ctx, &wg, "permissions", app.Permissions.Subscribe, ch)

	for role, agentSvc := range app.Agents {
		if agentSvc != nil {
			setupSubscriber(ctx, &wg, "agent-"+role, agentSvc.Subscribe, ch)
		}
	}

	cleanupFunc := func() {
		logging.Info("Cancelling all subscriptions")
		cancel() // Signal all goroutines to stop

		waitCh := make(chan struct{})
		go func() {
			defer logging.RecoverPanic("subscription-cleanup", nil)
			wg.Wait()
			close(waitCh)
		}()

		select {
		case <-waitCh:
			logging.Info("All subscription goroutines completed successfully")
			close(ch) // Only close after all writers are confirmed done
		case <-time.After(5 * time.Second):
			logging.Warn("Timed out waiting for some subscription goroutines to complete")
			close(ch)
		}
	}
	return ch, cleanupFunc
}

// CLI subcommands recognized by the act CLI.
// When `act <subcommand>` is called, route to the TypeScript CLI instead of the TUI.
var cliSubcommands = map[string]bool{
	"register": true, "context": true, "task": true, "brief": true,
	"pvm": true, "validation": true, "files": true, "message": true,
	"log": true, "graph": true, "status": true, "codebase": true,
	"swarm": true, "nomik": true,
}

func isCLISubcommand(arg string) bool {
	return cliSubcommands[arg]
}

// runReset asks for explicit confirmation then POSTs to /api/dev/reset.
func runReset() error {
	fmt.Println()
	fmt.Println("  ⚠  This will delete all your ACT project history including tasks,")
	fmt.Println("     agents, briefs, and file lock state. This means that if you want")
	fmt.Println("     to use ACT for any previous projects they must be imported as if")
	fmt.Println("     they are new, and agents will have to re-analyze the codebase.")
	fmt.Println("     The Planner will not be familiar with the deeper nuances of those")
	fmt.Println("     projects and may need to be re-explained.")
	fmt.Println()
	fmt.Println("     PVM coordination memory (learned patterns, agent skill profiles)")
	fmt.Println("     is NOT cleared — only live project/task state.")
	fmt.Println()
	fmt.Print(`  If you are sure, type "remove everything" and press Enter.`)
	fmt.Println()
	fmt.Print("  Otherwise press Ctrl-C to cancel: ")

	fmt.Println()
	fmt.Print("  Confirmation: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	phrase := strings.TrimSpace(scanner.Text())

	if phrase != "remove everything" {
		fmt.Println()
		fmt.Println("  Cancelled — nothing was changed.")
		return nil
	}

	if err := actserver.EnsureServerRunning(""); err != nil {
		return fmt.Errorf("server not reachable: %w", err)
	}

	serverURL := os.Getenv("ACT_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	resp, err := http.Post(serverURL+"/api/dev/reset", "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to reach server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	fmt.Println()
	fmt.Println("  Done. All project and task state has been cleared.")
	fmt.Println("  PVM coordination memory was preserved.")
	return nil
}

// routeToCLI finds and execs into the TypeScript act CLI.
func routeToCLI(args []string) error {
	cliScript := findCLIScript()
	if cliScript == "" {
		return fmt.Errorf("act CLI not found — expected cli/act-cli.ts or act-agent/cli/act-cli.ts")
	}

	bin, err := exec.LookPath("npx")
	if err != nil {
		return fmt.Errorf("npx not found: %w", err)
	}

	execArgs := append([]string{"npx", "tsx", cliScript}, args...)
	return syscall.Exec(bin, execArgs, os.Environ())
}

// findCLIScript locates act-cli.ts. Search order:
//  1. ~/.act/config.json `actRoot` field (canonical install path)
//  2. Walk up from the resolved binary location, sibling `act-agent/cli/`
//  3. cwd-relative fallbacks (only useful when running inside the ACT repo)
//
// This is the same problem-shape as findRunnerScript and findServerScript:
// `act` is a globally-symlinked binary that users invoke from arbitrary cwds,
// so the .ts file lookup must NOT depend on cwd.
func findCLIScript() string {
	// Strategy 1: ~/.act/config.json
	if home, err := os.UserHomeDir(); err == nil {
		cfgPath := filepath.Join(home, ".act", "config.json")
		if data, err := os.ReadFile(cfgPath); err == nil {
			if root := extractActRoot(string(data)); root != "" {
				for _, rel := range []string{
					filepath.Join("act-agent", "cli", "act-cli.ts"),
					filepath.Join("cli", "act-cli.ts"),
				} {
					full := filepath.Join(root, rel)
					if _, err := os.Stat(full); err == nil {
						return full
					}
				}
			}
		}
	}

	// Strategy 2: walk up from the resolved binary
	if execPath, err := os.Executable(); err == nil {
		dir := filepath.Dir(execPath)
		if resolved, rerr := filepath.EvalSymlinks(execPath); rerr == nil {
			dir = filepath.Dir(resolved)
		}
		for i := 0; i < 5; i++ {
			for _, rel := range []string{
				filepath.Join("cli", "act-cli.ts"),
				filepath.Join("act-agent", "cli", "act-cli.ts"),
			} {
				full := filepath.Join(dir, rel)
				if _, err := os.Stat(full); err == nil {
					return full
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Strategy 3: cwd-relative fallback
	for _, c := range []string{"cli/act-cli.ts", "act-agent/cli/act-cli.ts"} {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

// extractActRoot pulls the actRoot string field from a minimal JSON document.
// Avoids importing encoding/json into cmd/ for one field.
func extractActRoot(doc string) string {
	const key = `"actRoot"`
	idx := strings.Index(doc, key)
	if idx == -1 {
		return ""
	}
	rest := doc[idx+len(key):]
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
	closeIdx := strings.Index(rest, `"`)
	if closeIdx == -1 {
		return ""
	}
	return rest[:closeIdx]
}

func Execute() {
	// Allow positional args (for CLI subcommand routing)
	rootCmd.Args = cobra.ArbitraryArgs

	// Pre-cobra CLI subcommand routing. Cobra parses flags BEFORE RunE runs,
	// so a call like `act log --tail 20` would die on "unknown flag: --tail"
	// before reaching routeToCLI inside RunE. Detect the subcommand straight
	// from os.Args and exec into the TS CLI here, skipping cobra entirely
	// for those calls. This also keeps the bash tool's act-CLI invocations
	// flag-permissive (the TS CLI parses its own flags).
	if len(os.Args) > 1 {
		first := os.Args[1]

		// `act reset` — handled natively (HTTP POST to server, no TS CLI needed)
		if first == "reset" {
			if err := runReset(); err != nil {
				fmt.Fprintf(os.Stderr, "reset failed: %v\n", err)
				os.Exit(1)
			}
			return
		}

		if isCLISubcommand(first) {
			// Ensure server is running before routing to TS CLI
			if err := actserver.EnsureServerRunning(""); err != nil {
				logging.Warn("ACT server auto-start failed", "error", err)
			}
			if err := routeToCLI(os.Args[1:]); err != nil {
				os.Exit(1)
			}
			return
		}
	}

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("help", "h", false, "Help")
	rootCmd.Flags().BoolP("version", "v", false, "Version")
	rootCmd.Flags().BoolP("debug", "d", false, "Debug")
	rootCmd.Flags().StringP("cwd", "c", "", "Current working directory")
	rootCmd.Flags().StringP("prompt", "p", "", "Prompt to run in non-interactive mode")

	// Add format flag with validation logic
	rootCmd.Flags().StringP("output-format", "f", format.Text.String(),
		"Output format for non-interactive mode (text, json)")

	// Add quiet flag to hide spinner in non-interactive mode
	rootCmd.Flags().BoolP("quiet", "q", false, "Hide spinner in non-interactive mode")

	// ACT coordination modes
	rootCmd.Flags().String("agent", "", "ACT agent ID — headless worker mode (used by the swarm runner)")
	rootCmd.Flags().String("role", "", "ACT role — selects model config (developer|frontend_dev|backend_dev|qa_engineer|researcher)")
	rootCmd.Flags().String("project", "", "Project name for the ACT session (defaults to the current directory's basename)")

	// Register custom validation for the format flag
	rootCmd.RegisterFlagCompletionFunc("output-format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return format.SupportedFormats, cobra.ShellCompDirectiveNoFileComp
	})
}
