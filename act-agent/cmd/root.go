package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/app"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/db"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/format"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/agent"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/pubsub"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "act",
	Short: "ACT — Agent Coordination Toolkit",
	Long:  `ACT launches the multi-agent coordination harness with NesTTY.`,
	Example: `
  # Launch NesTTY (default — interactive project selection)
  act

  # Launch with a specific project
  act --project my-app

  # Headless agent mode (used by runner)
  act --agent dev-1 --role developer -p "implement auth"

  # Single non-interactive prompt
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
		_, err := config.Load(cwd, debug)
		if err != nil {
			return err
		}

		// Connect DB, this will also run migrations
		conn, err := db.Connect()
		if err != nil {
			return err
		}

		// Create main context for the application
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		app, err := app.New(ctx, conn)
		if err != nil {
			logging.Error("Failed to create app: %v", err)
			return err
		}
		// Defer shutdown here so it runs for both interactive and non-interactive modes
		defer app.Shutdown()

		// Initialize MCP tools early for both modes
		initMCPTools(ctx, app)

		// ACT agent mode (headless, JSON stdout, wired to act CLI)
		agentID, _ := cmd.Flags().GetString("agent")
		agentRole, _ := cmd.Flags().GetString("role")
		if agentID != "" {
			if prompt == "" {
				return fmt.Errorf("--agent requires --prompt")
			}
			return app.RunAgent(ctx, prompt, agentID, agentRole)
		}

		// NesTTY mode (persistent session, stdin/stdout conversation relay)
		nesttyRole, _ := cmd.Flags().GetString("nestty")
		if nesttyRole != "" {
			return app.RunNesTTY(ctx, nesttyRole, prompt)
		}

		// Non-interactive mode
		if prompt != "" {
			// Run non-interactive flow using the App method
			return app.RunNonInteractive(ctx, prompt, outputFormat, quiet)
		}

		// Interactive mode — launch the ACT TUI
		// The TUI provides onboarding, help, project management, sessions, and stats.
		// NesTTY orchestration is triggered from within the TUI or via `act orchestrate`.
		return runTUI(app, ctx)
	},
}

// runTUI launches the ACT TUI (the OpenCode-fork Bubble Tea interface).
// This IS NesTTY — the TUI hosts the 4 Tier 1 agents as in-process goroutines
// and the orchestrator spawns the Tier 2 swarm via runner.Spawner.
// There is no separate "NesTTY launcher" — the deprecated TypeScript orchestrator
// at nestty/ is reference material only.
func runTUI(a *app.App, ctx context.Context) error {
	zone.NewGlobal()
	program := tea.NewProgram(
		tui.New(a),
		tea.WithAltScreen(),
	)

	ch, cancelSubs := setupSubscriptions(a, ctx)

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

	_, err := program.Run()
	cleanup()
	return err
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

	if len(app.Agents) > 0 {
		for role, agentSvc := range app.Agents {
			if agentSvc != nil {
				setupSubscriber(ctx, &wg, "agent-"+role, agentSvc.Subscribe, ch)
			}
		}
	} else {
		setupSubscriber(ctx, &wg, "coderAgent", app.CoderAgent.Subscribe, ch)
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

func findCLIScript() string {
	candidates := []string{
		"cli/act-cli.ts",
		"act-agent/cli/act-cli.ts",
	}
	if execPath, err := os.Executable(); err == nil {
		dir := filepath.Dir(execPath)
		candidates = append(candidates,
			filepath.Join(dir, "cli", "act-cli.ts"),
			filepath.Join(dir, "..", "cli", "act-cli.ts"),
		)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

func Execute() {
	// Allow positional args (for CLI subcommand routing)
	rootCmd.Args = cobra.ArbitraryArgs

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
	rootCmd.Flags().String("agent", "", "ACT agent ID — headless mode with JSON stdout, wired to act CLI")
	rootCmd.Flags().String("role", "", "ACT role — selects model config (developer|planner|observer|assurance|qa)")
	rootCmd.Flags().String("nestty", "", "NesTTY role (planner|observer|assurance|qa) — PTY split mode")
	rootCmd.Flags().String("project", "", "Project name for NesTTY session")

	// Register custom validation for the format flag
	rootCmd.RegisterFlagCompletionFunc("output-format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return format.SupportedFormats, cobra.ShellCompDirectiveNoFileComp
	})
}
