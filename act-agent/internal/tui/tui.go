package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/harmonica"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/app"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/agent"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/permission"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/pubsub"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/session"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/anim"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/components/chat"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/components/core"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/components/dialog"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/layout"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/page"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/util"
)

type keyMap struct {
	Logs          key.Binding
	Quit          key.Binding
	Help          key.Binding
	SwitchSession key.Binding
	Commands      key.Binding
	Filepicker    key.Binding
	Models        key.Binding
	SwitchTheme   key.Binding
	RenameSession key.Binding
}

type startCompactSessionMsg = dialog.StartCompactSessionMsg

// runDirectCommandMsg is dispatched by palette commands that bypass the
// Planner and shell out directly to the act CLI via Orchestrator.RunDirectCommand.
type runDirectCommandMsg struct {
	label string
	argv  []string
}

const (
	quitKey = "q"
)

var keys = keyMap{
	Logs: key.NewBinding(
		key.WithKeys("ctrl+l"),
		key.WithHelp("ctrl+l", "logs"),
	),

	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("ctrl+_", "ctrl+h", "ctrl+?"),
		key.WithHelp("ctrl+?", "toggle help"),
	),

	SwitchSession: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "switch session"),
	),

	Commands: key.NewBinding(
		key.WithKeys("ctrl+k"),
		key.WithHelp("ctrl+k", "commands"),
	),
	Filepicker: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("ctrl+f", "select files to upload"),
	),
	Models: key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("ctrl+o", "model selection"),
	),

	SwitchTheme: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("ctrl+t", "switch theme"),
	),
	RenameSession: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "rename session"),
	),
}

var helpEsc = key.NewBinding(
	key.WithKeys("?"),
	key.WithHelp("?", "toggle help"),
)

var logsKeyReturnKey = key.NewBinding(
	key.WithKeys("esc", "backspace", quitKey),
	key.WithHelp("esc/q", "go back"),
)

type appModel struct {
	width, height   int
	currentPage     page.PageID
	previousPage    page.PageID
	pages           map[page.PageID]tea.Model
	loadedPages     map[page.PageID]bool
	status          core.StatusCmp
	app             *app.App
	selectedSession session.Session

	// modalStack is the single source of truth for which overlay is open
	// and in what z-order (index 0 = bottom). The 13 discrete show* booleans
	// this replaces allowed contradictory states (two dialogs "open" with
	// esc unwinding in an order that didn't match draw order — audit H2).
	// Key routing goes to stack top; esc pops the top. Permissions is
	// push-only from the UI's perspective: it closes only via an explicit
	// PermissionResponseMsg, never via the esc cascade.
	modalStack []ModalID

	permissions   dialog.PermissionDialogCmp
	help          dialog.HelpCmp
	quit          dialog.QuitDialog
	sessionDialog dialog.SessionDialog
	commandDialog dialog.CommandDialog
	commands      []dialog.Command
	// slide-in spring for the command palette (positive offset = shifted right, off-screen)
	cmdSlideOffset float64
	cmdSlideVel    float64
	cmdSlideSpring harmonica.Spring
	cmdSliding     bool

	modelDialog          dialog.ModelDialog
	onboardingDialog     dialog.OnboardingCmp
	filepicker           dialog.FilepickerCmp
	themeDialog          dialog.ThemeDialog
	renameDialog         dialog.RenameDialog
	multiArgumentsDialog *dialog.MultiArgumentsDialogCmp
	infoDialog           dialog.InfoDialog

	isCompacting      bool
	compactingMessage string
}

// ModalID identifies one overlay dialog.
type ModalID uint8

const (
	ModalPermissions ModalID = iota
	ModalHelp
	ModalQuit
	ModalSession
	ModalCommand
	ModalModel
	ModalOnboarding
	ModalFilepicker
	ModalTheme
	ModalRename
	ModalMultiArguments
	ModalInfo
)

// pushModal marks the dialog open, top of the z-order.
func (a *appModel) pushModal(id ModalID) {
	// Idempotent: re-pushing an active modal is a no-op (a second pubsub
	// PermissionRequest while the prompt is already up must not double-stack).
	for _, active := range a.modalStack {
		if active == id {
			return
		}
	}
	a.modalStack = append(a.modalStack, id)
}

// popModal removes and returns the top modal. ok=false when the stack is
// empty. Callers that need side effects (filepicker toggle-off, palette
// spring reset) handle them at their Close*Msg sites; popModal itself only
// mutates the stack.
func (a *appModel) popModal() (ModalID, bool) {
	n := len(a.modalStack)
	if n == 0 {
		return 0, false
	}
	id := a.modalStack[n-1]
	a.modalStack = a.modalStack[:n-1]
	return id, true
}

// isModalActive reports whether the given dialog is currently open.
func (a *appModel) isModalActive(id ModalID) bool {
	for _, m := range a.modalStack {
		if m == id {
			return true
		}
	}
	return false
}

// topModal returns the topmost modal, ok=false when none.
func (a *appModel) topModal() (ModalID, bool) {
	n := len(a.modalStack)
	if n == 0 {
		return 0, false
	}
	return a.modalStack[n-1], true
}

// removeModal closes a specific modal wherever it sits in the stack (used
// by explicit Close*Msg handlers so a dialog's own close message can't
// strand it in the stack if something above it was popped first). Returns
// true when the modal was found and removed.
func (a *appModel) removeModal(id ModalID) bool {
	for i, m := range a.modalStack {
		if m == id {
			a.modalStack = append(a.modalStack[:i], a.modalStack[i+1:]...)
			return true
		}
	}
	return false
}

// modalActiveBelow reports whether any modal OTHER than the given one is
// active. Used by close handlers that must not fully tear down state while
// another overlay is still up.
func (a *appModel) modalActiveBelow(id ModalID) bool {
	for _, m := range a.modalStack {
		if m != id {
			return true
		}
	}
	return false
}

func (a appModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmd := a.pages[a.currentPage].Init()
	a.loadedPages[a.currentPage] = true
	cmds = append(cmds, cmd)
	cmd = a.status.Init()
	cmds = append(cmds, cmd)
	cmd = a.quit.Init()
	cmds = append(cmds, cmd)
	cmd = a.help.Init()
	cmds = append(cmds, cmd)
	cmd = a.sessionDialog.Init()
	cmds = append(cmds, cmd)
	cmd = a.commandDialog.Init()
	cmds = append(cmds, cmd)
	cmd = a.modelDialog.Init()
	cmds = append(cmds, cmd)
	cmd = a.onboardingDialog.Init()
	cmds = append(cmds, cmd)
	cmd = a.filepicker.Init()
	cmds = append(cmds, cmd)
	cmd = a.themeDialog.Init()
	cmds = append(cmds, cmd)

	// Show onboarding wizard on first run. The legacy InitDialog (Yes/No
	// "scan codebase for ACT.md") is gone — new-project intake is now driven
	// by the orchestrator's INTAKE mode (see orchestrator.detectProjectState).
	// The Planner runs a structured 5-question conversation and emits
	// PROJECT_BRIEF when ready, which the orchestrator POSTs to the server.
	cmds = append(cmds, func() tea.Msg {
		if config.IsFirstRun() && !config.HasConfigFile() {
			return dialog.ShowOnboardingDialogMsg{Show: true}
		}
		return nil
	})

	cmds = append(cmds, func() tea.Msg {
		sessions, err := a.app.Sessions.List(context.Background())
		if err == nil && len(sessions) > 0 {
			return chat.SessionSelectedMsg(sessions[0])
		}
		return nil
	})

	return tea.Batch(cmds...)
}

func (a appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		windowWidth := msg.Width
		windowHeight := msg.Height

		a.width = windowWidth
		a.height = windowHeight

		pageHeight := max(1, windowHeight-1)
		pageMsg := tea.WindowSizeMsg{Width: windowWidth, Height: pageHeight}

		statusMsg := tea.WindowSizeMsg{Width: windowWidth, Height: 1}
		s, _ := a.status.Update(statusMsg)
		a.status = s.(core.StatusCmp)

		a.pages[a.currentPage], cmd = a.pages[a.currentPage].Update(pageMsg)
		cmds = append(cmds, cmd)

		prm, permCmd := a.permissions.Update(pageMsg)
		a.permissions = prm.(dialog.PermissionDialogCmp)
		cmds = append(cmds, permCmd)

		help, helpCmd := a.help.Update(pageMsg)
		a.help = help.(dialog.HelpCmp)
		cmds = append(cmds, helpCmd)

		session, sessionCmd := a.sessionDialog.Update(pageMsg)
		a.sessionDialog = session.(dialog.SessionDialog)
		cmds = append(cmds, sessionCmd)

		command, commandCmd := a.commandDialog.Update(pageMsg)
		a.commandDialog = command.(dialog.CommandDialog)
		cmds = append(cmds, commandCmd)

		filepicker, filepickerCmd := a.filepicker.Update(pageMsg)
		a.filepicker = filepicker.(dialog.FilepickerCmp)
		cmds = append(cmds, filepickerCmd)

		if a.isModalActive(ModalMultiArguments) && a.multiArgumentsDialog != nil {
			a.multiArgumentsDialog.SetSize(windowWidth, pageHeight)
			args, argsCmd := a.multiArgumentsDialog.Update(pageMsg)
			a.multiArgumentsDialog = args.(*dialog.MultiArgumentsDialogCmp)
			cmds = append(cmds, argsCmd)
		}

		onboard, onboardCmd := a.onboardingDialog.Update(pageMsg)
		a.onboardingDialog = onboard.(dialog.OnboardingCmp)
		cmds = append(cmds, onboardCmd)

		return a, tea.Batch(cmds...)
	// Status
	case util.InfoMsg:
		s, cmd := a.status.Update(msg)
		a.status = s.(core.StatusCmp)
		cmds = append(cmds, cmd)
		return a, tea.Batch(cmds...)
	case pubsub.Event[logging.LogMessage]:
		if msg.Payload.Persist {
			switch msg.Payload.Level {
			case "error":
				s, cmd := a.status.Update(util.InfoMsg{
					Type: util.InfoTypeError,
					Msg:  msg.Payload.Message,
					TTL:  msg.Payload.PersistTime,
				})
				a.status = s.(core.StatusCmp)
				cmds = append(cmds, cmd)
			case "info":
				s, cmd := a.status.Update(util.InfoMsg{
					Type: util.InfoTypeInfo,
					Msg:  msg.Payload.Message,
					TTL:  msg.Payload.PersistTime,
				})
				a.status = s.(core.StatusCmp)
				cmds = append(cmds, cmd)

			case "warn":
				s, cmd := a.status.Update(util.InfoMsg{
					Type: util.InfoTypeWarn,
					Msg:  msg.Payload.Message,
					TTL:  msg.Payload.PersistTime,
				})

				a.status = s.(core.StatusCmp)
				cmds = append(cmds, cmd)
			default:
				s, cmd := a.status.Update(util.InfoMsg{
					Type: util.InfoTypeInfo,
					Msg:  msg.Payload.Message,
					TTL:  msg.Payload.PersistTime,
				})
				a.status = s.(core.StatusCmp)
				cmds = append(cmds, cmd)
			}
		}
	case util.ClearStatusMsg:
		s, _ := a.status.Update(msg)
		a.status = s.(core.StatusCmp)

		// Permission
	case pubsub.Event[permission.PermissionRequest]:
		a.pushModal(ModalPermissions)
		return a, a.permissions.SetPermissions(msg.Payload)
	case dialog.PermissionResponseMsg:
		var cmd tea.Cmd
		switch msg.Action {
		case dialog.PermissionAllow:
			a.app.Permissions.Grant(msg.Permission)
		case dialog.PermissionAllowForSession:
			a.app.Permissions.GrantPersistant(msg.Permission)
		case dialog.PermissionDeny:
			a.app.Permissions.Deny(msg.Permission)
		}
		a.removeModal(ModalPermissions)
		return a, cmd

	case page.PageChangeMsg:
		return a, a.moveToPage(msg.ID)

	case dialog.CloseQuitMsg:
		a.removeModal(ModalQuit)
		return a, nil

	case dialog.CloseSessionDialogMsg:
		a.removeModal(ModalSession)
		return a, nil

	case dialog.CloseRenameDialogMsg:
		a.removeModal(ModalRename)
		return a, nil

	case dialog.RenameCompletedMsg:
		a.removeModal(ModalRename)
		newTitle := strings.TrimSpace(msg.NewTitle)
		if newTitle != "" && a.selectedSession.ID != "" {
			a.selectedSession.Title = newTitle
			// Save the renamed session to the database
			_, err := a.app.Sessions.Save(context.Background(), a.selectedSession)
			if err != nil {
				return a, util.ReportError(err)
			}
			// Update the session in the session dialog list so it shows the new name
			sessions, err := a.app.Sessions.List(context.Background())
			if err == nil {
				a.sessionDialog.SetSessions(sessions)
			}
		}
		return a, nil

	case dialog.CloseCommandDialogMsg:
		a.removeModal(ModalCommand)
		a.cmdSlideOffset = 0
		a.cmdSliding = false
		return a, nil

	case startCompactSessionMsg:
		// Start compacting the current session
		a.isCompacting = true
		a.compactingMessage = "Starting summarization..."

		if a.selectedSession.ID == "" {
			a.isCompacting = false
			return a, util.ReportWarn("No active session to summarize")
		}

		// Start the summarization process — Planner is the canonical
		// human-facing agent so it owns summarization of the shared session.
		return a, func() tea.Msg {
			ctx := context.Background()
			a.app.Agents["planner"].Summarize(ctx, a.selectedSession.ID)
			return nil
		}

	case pubsub.Event[agent.AgentEvent]:
		payload := msg.Payload
		if payload.Error != nil {
			a.isCompacting = false
			return a, util.ReportError(payload.Error)
		}

		a.compactingMessage = payload.Progress

		if payload.Done && payload.Type == agent.AgentEventTypeSummarize {
			a.isCompacting = false
			return a, util.ReportInfo("Session summarization complete")
		} else if payload.Done && payload.Type == agent.AgentEventTypeResponse && a.selectedSession.ID != "" {
			// Auto-compaction is ACT hygiene, not model arithmetic. Whatever
			// the conversation accumulates still gets sent to whichever model
			// is current; compaction exists to keep system prompts + project
			// context + recent turns coherent and prevent drift. The trigger
			// is an ACT-defined token total, configurable via autoCompactTokens
			// in ~/.act.json.
			cfg := config.Get()
			if cfg != nil && cfg.AutoCompact {
				threshold := cfg.AutoCompactTokens
				if threshold <= 0 {
					threshold = config.DefaultAutoCompactTokens
				}
				tokens := a.selectedSession.CompletionTokens + a.selectedSession.PromptTokens
				if tokens >= threshold {
					return a, util.CmdHandler(startCompactSessionMsg{})
				}
			}
		}
		// Continue listening for events
		return a, nil

	case dialog.CloseThemeDialogMsg:
		a.removeModal(ModalTheme)
		return a, nil

	case dialog.ThemeChangedMsg:
		a.pages[a.currentPage], cmd = a.pages[a.currentPage].Update(msg)
		a.removeModal(ModalTheme)
		return a, tea.Batch(cmd, util.ReportInfo("Theme changed to: "+msg.ThemeName))

	case dialog.CloseModelDialogMsg:
		a.removeModal(ModalModel)
		return a, nil

	case dialog.ModelSelectedMsg:
		a.removeModal(ModalModel)

		targetRole := msg.Role
		if targetRole == "" {
			targetRole = string(config.RolePlanner)
		}
		roleName := config.AgentConfigForRole(targetRole)
		if err := config.UpdateAgentProvider(roleName, msg.Model.Provider, msg.Model.ID); err != nil {
			return a, util.ReportError(err)
		}
		if agent, ok := a.app.Agents[targetRole]; ok {
			if _, err := agent.Update(roleName, msg.Model.ID); err != nil {
				return a, util.ReportError(err)
			}
		}
		return a, util.ReportInfo(fmt.Sprintf("%s model changed to %s (%s)", targetRole, msg.Model.ID, msg.Model.Provider))

	case dialog.ShowOnboardingDialogMsg:
		if msg.Show {
			a.pushModal(ModalOnboarding)
		} else {
			a.removeModal(ModalOnboarding)
		}
		return a, nil

	case dialog.CloseOnboardingMsg:
		a.removeModal(ModalOnboarding)
		// Onboarding finished (confirmed or cancelled). Mark the project as
		// initialized either way — the wizard wrote the config it needed.
		if err := config.MarkProjectInitialized(); err != nil {
			return a, util.ReportError(err)
		}
		return a, nil

	case runDirectCommandMsg:
		sid := a.selectedSession.ID
		if sid == "" || a.app.Orchestrator == nil {
			return a, util.ReportWarn("No active session — start a conversation first")
		}
		label := msg.label
		argv := append([]string(nil), msg.argv...)
		go a.app.Orchestrator.RunDirectCommand(context.Background(), sid, label, argv)
		return a, nil

	case chat.SessionSelectedMsg:
		a.selectedSession = msg
		a.sessionDialog.SetSelectedSession(msg.ID)
		if msg.ID != "" && a.app.Orchestrator != nil {
			a.app.Orchestrator.Start(context.Background(), msg.ID)
		}

	case pubsub.Event[session.Session]:
		if msg.Type == pubsub.UpdatedEvent && msg.Payload.ID == a.selectedSession.ID {
			a.selectedSession = msg.Payload
		}
	case dialog.SessionSelectedMsg:
		a.removeModal(ModalSession)
		if a.currentPage == page.ChatPage {
			return a, util.CmdHandler(chat.SessionSelectedMsg(msg.Session))
		}
		return a, nil

	case dialog.CreateNewSessionMsg:
		a.removeModal(ModalSession)
		newSess, err := a.app.Sessions.Create(context.Background(), "New Session")
		if err != nil {
			return a, util.ReportError(err)
		}
		a.selectedSession = newSess
		a.sessionDialog.SetSelectedSession(newSess.ID)
		if sessions, err := a.app.Sessions.List(context.Background()); err == nil {
			a.sessionDialog.SetSessions(sessions)
		}
		return a, tea.Batch(
			util.CmdHandler(chat.SessionSelectedMsg(newSess)),
			util.ReportInfo("Created new session"),
		)

	case dialog.ShowInfoDialogMsg:
		a.infoDialog.SetContent(msg.Title, msg.Content)
		a.pushModal(ModalInfo)
		return a, nil

	case dialog.CloseInfoDialogMsg:
		a.removeModal(ModalInfo)
		return a, nil

	case dialog.ToggleHelpMsg:
		if a.isModalActive(ModalHelp) {
			a.removeModal(ModalHelp)
		} else {
			a.pushModal(ModalHelp)
		}
		return a, nil

	case dialog.ShowLogsMsg:
		return a, a.moveToPage(page.LogsPage)

	case dialog.ShowModelDialogMsg:
		a.pushModal(ModalModel)
		a.modelDialog.SetRole(msg.Role)
		return a, a.modelDialog.Init()

	case dialog.CommandSelectedMsg:
		a.removeModal(ModalCommand)
		a.cmdSlideOffset = 0
		a.cmdSliding = false
		// Execute the command handler if available
		if msg.Command.Handler != nil {
			return a, msg.Command.Handler(msg.Command)
		}
		return a, util.ReportInfo("Command selected: " + msg.Command.Title)

	case dialog.ShowMultiArgumentsDialogMsg:
		// Show multi-arguments dialog
		m := dialog.NewMultiArgumentsDialogCmp(msg.CommandID, msg.Content, msg.ArgNames)
		a.multiArgumentsDialog = &m
		a.pushModal(ModalMultiArguments)
		return a, a.multiArgumentsDialog.Init()

	case dialog.CloseMultiArgumentsDialogMsg:
		// Close multi-arguments dialog
		a.removeModal(ModalMultiArguments)

		// If submitted, replace all named arguments and run the command
		if msg.Submit {
			content := msg.Content

			// Replace each named argument with its value
			for name, value := range msg.Args {
				placeholder := "$" + name
				content = strings.ReplaceAll(content, placeholder, value)
			}

			// Execute the command with arguments
			return a, util.CmdHandler(dialog.CommandRunCustomMsg{
				Content: content,
				Args:    msg.Args,
			})
		}
		return a, nil

	case tea.KeyPressMsg:
		if msg.String() == "esc" {
			// Single pop point: esc closes the TOP modal only. The stack IS
			// the z-order, so unwind order can never diverge from draw order
			// (audit H2). Permissions is intentionally not escapable — it is
			// answered via PermissionResponseMsg, not dismissed.
			top, ok := a.topModal()
			if ok && top != ModalPermissions {
				a.popModal()
				switch top {
				case ModalFilepicker:
					a.filepicker.ToggleFilepicker(false)
				case ModalCommand:
					a.cmdSlideOffset = 0
					a.cmdSliding = false
				}
				return a, nil
			}
			if a.currentPage == page.LogsPage {
				return a, a.moveToPage(page.ChatPage)
			}
		}

		// If multi-arguments dialog is open, let it handle the key press first
		if a.isModalActive(ModalMultiArguments) && a.multiArgumentsDialog != nil {
			args, argsCmd := a.multiArgumentsDialog.Update(msg)
			a.multiArgumentsDialog = args.(*dialog.MultiArgumentsDialogCmp)
			return a, argsCmd
		}

		switch {

		case key.Matches(msg, keys.Quit):
			if a.currentPage == page.LogsPage {
				return a, a.moveToPage(page.ChatPage)
			}
			// A pending permission prompt must never be buried by the quit
			// cascade: an agent is blocked on it, and dismissing every other
			// dialog around it leaves the user staring at a prompt whose
			// context (chat, palette) just vanished. Quit is refused while
			// one is up — answer or deny it first. Esc intentionally cannot
			// dismiss it either (see esc cascade).
			if a.isModalActive(ModalPermissions) {
				return a, util.ReportWarn("Answer the permission prompt first (a/s/d)")
			}
			if a.isModalActive(ModalQuit) {
				a.removeModal(ModalQuit)
			} else {
				a.pushModal(ModalQuit)
			}
			// ctrl+c clears every other overlay except permissions — the
			// user is expressing intent to exit, so stale dialogs shouldn't
			// survive into the quit confirmation.
			a.removeModal(ModalHelp)
			a.removeModal(ModalSession)
			if a.removeModal(ModalCommand) {
				a.cmdSlideOffset = 0
				a.cmdSliding = false
			}
			if a.removeModal(ModalFilepicker) {
				a.filepicker.ToggleFilepicker(false)
			}
			a.removeModal(ModalModel)
			a.removeModal(ModalMultiArguments)
			return a, nil
		case key.Matches(msg, keys.SwitchSession):
			if a.currentPage == page.ChatPage && !a.isModalActive(ModalQuit) && !a.isModalActive(ModalPermissions) && !a.isModalActive(ModalCommand) {
				// Load sessions and show the dialog
				sessions, err := a.app.Sessions.List(context.Background())
				if err != nil {
					return a, util.ReportError(err)
				}
				if len(sessions) == 0 {
					return a, util.ReportWarn("No sessions available")
				}
				a.sessionDialog.SetSessions(sessions)
				a.pushModal(ModalSession)
				return a, nil
			}
			return a, nil
		case key.Matches(msg, keys.Commands):
			if a.currentPage == page.ChatPage && !a.isModalActive(ModalQuit) && !a.isModalActive(ModalPermissions) && !a.isModalActive(ModalSession) && !a.isModalActive(ModalTheme) && !a.isModalActive(ModalFilepicker) {
				// Show commands dialog
				if len(a.commands) == 0 {
					return a, util.ReportWarn("No commands available")
				}
				animCmd := a.commandDialog.SetCommands(a.commands)
				a.pushModal(ModalCommand)
				// Start slide-in from +40 cols to the right (stiffness=14, damping=0.6)
				a.cmdSlideSpring = anim.NewSpring(14, 0.6)
				a.cmdSlideOffset = 40
				a.cmdSlideVel = 0
				a.cmdSliding = true
				return a, tea.Batch(animCmd, anim.Frame())
			}
			return a, nil
		case key.Matches(msg, keys.Models):
			if a.isModalActive(ModalModel) {
				a.removeModal(ModalModel)
				return a, nil
			}
			if a.currentPage == page.ChatPage && !a.isModalActive(ModalQuit) && !a.isModalActive(ModalPermissions) && !a.isModalActive(ModalSession) && !a.isModalActive(ModalCommand) {
				a.pushModal(ModalModel)
				return a, nil
			}
			return a, nil
		case key.Matches(msg, keys.SwitchTheme):
			if !a.isModalActive(ModalQuit) && !a.isModalActive(ModalPermissions) && !a.isModalActive(ModalSession) && !a.isModalActive(ModalCommand) {
				// Show theme switcher dialog
				a.pushModal(ModalTheme)
				// Theme list is dynamically loaded by the dialog component
				return a, a.themeDialog.Init()
			}
			return a, nil
		case key.Matches(msg, keys.RenameSession):
			if a.currentPage == page.ChatPage && !a.isModalActive(ModalQuit) && !a.isModalActive(ModalPermissions) && !a.isModalActive(ModalSession) && !a.isModalActive(ModalCommand) && !a.isModalActive(ModalTheme) && !a.isModalActive(ModalFilepicker) {
				if a.selectedSession.ID != "" {
					a.renameDialog.SetTitle(a.selectedSession.Title)
					a.pushModal(ModalRename)
					return a, nil
				}
			}
			return a, nil
		// Catchall for "any key not matched above". The old form was
		// `key.Matches(msg, returnKey) || key.Matches(msg)` — the second
		// operand has zero bindings so it always evaluated false, making it
		// dead (audit M8). Plain `default:` semantics via a bare condition.
		case len(msg.String()) > 0:
			if msg.String() == quitKey || msg.String() == "esc" || msg.String() == "q" {
				if a.currentPage == page.LogsPage {
					return a, a.moveToPage(page.ChatPage)
				}
			} else if !a.filepicker.IsCWDFocused() {
				// Dismiss the TOP modal only — same single-pop semantics as
				// the esc handler above.
				if top, ok := a.topModal(); ok {
					switch top {
					case ModalQuit:
						a.removeModal(ModalQuit)
						return a, nil
					case ModalHelp:
						a.removeModal(ModalHelp)
						return a, nil
					case ModalOnboarding:
						a.removeModal(ModalOnboarding)
						// Onboarding dismissed without completing — mark the
						// project initialized; the wizard wrote what it needed.
						if err := config.MarkProjectInitialized(); err != nil {
							return a, util.ReportError(err)
						}
						return a, nil
					case ModalFilepicker:
						a.removeModal(ModalFilepicker)
						a.filepicker.ToggleFilepicker(false)
						return a, nil
					case ModalRename:
						a.removeModal(ModalRename)
						return a, nil
					case ModalPermissions:
						return a, nil // answered via PermissionResponseMsg only
					}
				}
				if a.currentPage == page.LogsPage {
					return a, a.moveToPage(page.ChatPage)
				}
			}
		case key.Matches(msg, keys.Logs):
			return a, a.moveToPage(page.LogsPage)
		case key.Matches(msg, keys.Help):
			if a.isModalActive(ModalQuit) {
				return a, nil
			}
			if a.isModalActive(ModalHelp) {
				a.removeModal(ModalHelp)
			} else {
				a.pushModal(ModalHelp)
			}
			return a, nil
		case key.Matches(msg, helpEsc):
			if a.app.Orchestrator.IsAnyBusy("") {
				if a.isModalActive(ModalQuit) {
					return a, nil
				}
				if a.isModalActive(ModalHelp) {
					a.removeModal(ModalHelp)
				} else {
					a.pushModal(ModalHelp)
				}
				return a, nil
			}
		case key.Matches(msg, keys.Filepicker):
			if a.isModalActive(ModalFilepicker) {
				a.removeModal(ModalFilepicker)
				a.filepicker.ToggleFilepicker(false)
			} else {
				a.pushModal(ModalFilepicker)
				a.filepicker.ToggleFilepicker(true)
			}
			return a, nil
		}
	case anim.FrameMsg:
		// Advance the command palette slide-in spring.
		if a.cmdSliding {
			newOff, newVel := a.cmdSlideSpring.Update(a.cmdSlideOffset, a.cmdSlideVel, 0)
			a.cmdSlideOffset = newOff
			a.cmdSlideVel = newVel
			if a.cmdSlideOffset <= 0.5 {
				a.cmdSlideOffset = 0
				a.cmdSlideVel = 0
				a.cmdSliding = false
				return a, nil
			}
			return a, anim.Frame()
		}
		// Pass other FrameMsgs (e.g. splash fade) through to child pages.
		a.pages[a.currentPage], cmd = a.pages[a.currentPage].Update(msg)
		cmds = append(cmds, cmd)
		return a, tea.Batch(cmds...)

	default:
		f, filepickerCmd := a.filepicker.Update(msg)
		a.filepicker = f.(dialog.FilepickerCmp)
		cmds = append(cmds, filepickerCmd)

	}

	// Modal key routing — stack-driven. Only the TOP modal receives key
	// messages; non-key messages flow to every open modal (so pubsub events
	// still reach e.g. the filepicker while a help overlay is up). This
	// replaces eight sequential `if a.showX` blocks whose routing order was
	// maintained by hand and could diverge from draw order (audit H2).
	if _, isKey := msg.(tea.KeyPressMsg); isKey {
		if top, ok := a.topModal(); ok {
			var modalCmd tea.Cmd
			switch top {
			case ModalFilepicker:
				f, cmd := a.filepicker.Update(msg)
				a.filepicker = f.(dialog.FilepickerCmp)
				modalCmd = cmd
			case ModalQuit:
				q, cmd := a.quit.Update(msg)
				a.quit = q.(dialog.QuitDialog)
				modalCmd = cmd
			case ModalPermissions:
				d, cmd := a.permissions.Update(msg)
				a.permissions = d.(dialog.PermissionDialogCmp)
				modalCmd = cmd
			case ModalInfo:
				d, cmd := a.infoDialog.Update(msg)
				a.infoDialog = d.(dialog.InfoDialog)
				modalCmd = cmd
			case ModalSession:
				d, cmd := a.sessionDialog.Update(msg)
				a.sessionDialog = d.(dialog.SessionDialog)
				modalCmd = cmd
			case ModalCommand:
				d, cmd := a.commandDialog.Update(msg)
				a.commandDialog = d.(dialog.CommandDialog)
				modalCmd = cmd
			case ModalModel:
				d, cmd := a.modelDialog.Update(msg)
				a.modelDialog = d.(dialog.ModelDialog)
				modalCmd = cmd
			case ModalOnboarding:
				d, cmd := a.onboardingDialog.Update(msg)
				a.onboardingDialog = d.(dialog.OnboardingCmp)
				modalCmd = cmd
			case ModalTheme:
				d, cmd := a.themeDialog.Update(msg)
				a.themeDialog = d.(dialog.ThemeDialog)
				modalCmd = cmd
			case ModalRename:
				d, cmd := a.renameDialog.Update(msg)
				a.renameDialog = d.(dialog.RenameDialog)
				modalCmd = cmd
			case ModalMultiArguments:
				if a.multiArgumentsDialog != nil {
					args, cmd := a.multiArgumentsDialog.Update(msg)
					a.multiArgumentsDialog = args.(*dialog.MultiArgumentsDialogCmp)
					modalCmd = cmd
				}
			case ModalHelp:
				// HelpCmp has no key handling of its own (esc/? handled above).
				return a, nil
			}
			return a, modalCmd
		}
	} else {
		// Non-key messages: broadcast to all open modals in stack order.
		for _, id := range a.modalStack {
			switch id {
			case ModalFilepicker:
				f, filepickerCmd := a.filepicker.Update(msg)
				a.filepicker = f.(dialog.FilepickerCmp)
				cmds = append(cmds, filepickerCmd)
			case ModalPermissions:
				d, permissionsCmd := a.permissions.Update(msg)
				a.permissions = d.(dialog.PermissionDialogCmp)
				cmds = append(cmds, permissionsCmd)
			case ModalSession:
				d, sessionCmd := a.sessionDialog.Update(msg)
				a.sessionDialog = d.(dialog.SessionDialog)
				cmds = append(cmds, sessionCmd)
			case ModalOnboarding:
				d, onboardCmd := a.onboardingDialog.Update(msg)
				a.onboardingDialog = d.(dialog.OnboardingCmp)
				cmds = append(cmds, onboardCmd)
			}
		}
	}

	s, _ := a.status.Update(msg)
	a.status = s.(core.StatusCmp)
	a.pages[a.currentPage], cmd = a.pages[a.currentPage].Update(msg)
	cmds = append(cmds, cmd)
	return a, tea.Batch(cmds...)
}

// RegisterCommand adds a command to the command dialog
func (a *appModel) RegisterCommand(cmd dialog.Command) {
	a.commands = append(a.commands, cmd)
}

func (a *appModel) moveToPage(pageID page.PageID) tea.Cmd {
	// Read-only pages stay reachable while agents are running — the logs
	// pane is exactly where a user wants to look during a long Tier-1 run
	// (audit M9: the blanket busy-lock blocked ALL navigation). Only
	// blocking the move out of the chat page would strand the conversation,
	// and ChatPage is where ctrl+l's counterpart (esc from Logs) returns to;
	// chat remains implicitly available since moveToPage is only called for
	// non-chat targets today.
	if pageID != page.LogsPage && a.app.Orchestrator.IsAnyBusy("") {
		return util.ReportWarn("Agent is busy — only the logs view is available during a run")
	}

	var cmds []tea.Cmd
	if _, ok := a.loadedPages[pageID]; !ok {
		cmd := a.pages[pageID].Init()
		cmds = append(cmds, cmd)
		a.loadedPages[pageID] = true
	}
	a.previousPage = a.currentPage
	a.currentPage = pageID
	if sizable, ok := a.pages[a.currentPage].(layout.Sizeable); ok {
		pageHeight := a.height
		if pageHeight > 1 {
			pageHeight -= 1
		}
		cmd := sizable.SetSize(a.width, pageHeight)
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (a appModel) View() tea.View {
	components := []string{
		a.pages[a.currentPage].View().Content,
		a.status.View().Content,
	}

	appView := lipgloss.JoinVertical(lipgloss.Top, components...)

	// placeOverlayCentered is the single centered-overlay primitive. Every
	// modal draws through the stack loop below via this one helper, so z-order
	// is exactly stack order — later stack entries land visually on top
	// (audit H2: draw order and unwind order can no longer diverge).
	placeOverlayCentered := func(content string) {
		row := lipgloss.Height(appView)/2 - lipgloss.Height(content)/2
		col := lipgloss.Width(appView)/2 - lipgloss.Width(content)/2
		if row < 0 {
			row = 0
		}
		if col < 0 {
			col = 0
		}
		appView = layout.PlaceOverlay(col, row, content, appView, true)
	}

	// Draw every open modal in stack order (bottom → top).
	for _, id := range a.modalStack {
		switch id {
		case ModalPermissions:
			placeOverlayCentered(a.permissions.View().Content)
		case ModalOnboarding:
			placeOverlayCentered(a.onboardingDialog.View().Content)
		case ModalFilepicker:
			placeOverlayCentered(a.filepicker.View().Content)
		case ModalHelp:
			bindings := layout.KeyMapToSlice(keys)
			if p, ok := a.pages[a.currentPage].(layout.Bindings); ok {
				bindings = append(bindings, p.BindingKeys()...)
			}
			for _, m := range a.modalStack {
				if m == ModalPermissions {
					bindings = append(bindings, a.permissions.BindingKeys()...)
					break
				}
			}
			if a.currentPage == page.LogsPage {
				bindings = append(bindings, logsKeyReturnKey)
			}
			if !a.app.Orchestrator.IsAnyBusy("") {
				bindings = append(bindings, helpEsc)
			}
			a.help.SetBindings(bindings)
			placeOverlayCentered(a.help.View().Content)
		case ModalQuit:
			placeOverlayCentered(a.quit.View().Content)
		case ModalInfo:
			placeOverlayCentered(a.infoDialog.View().Content)
		case ModalSession:
			placeOverlayCentered(a.sessionDialog.View().Content)
		case ModalModel:
			placeOverlayCentered(a.modelDialog.View().Content)
		case ModalCommand:
			content := a.commandDialog.View().Content
			row := lipgloss.Height(appView)/2 - lipgloss.Height(content)/2
			col := lipgloss.Width(appView)/2 - lipgloss.Width(content)/2 + int(a.cmdSlideOffset)
			if row < 0 {
				row = 0
			}
			if col < 0 {
				col = 0
			}
			appView = layout.PlaceOverlay(col, row, content, appView, true)
		case ModalTheme:
			placeOverlayCentered(a.themeDialog.View().Content)
		case ModalRename:
			placeOverlayCentered(a.renameDialog.View().Content)
		case ModalMultiArguments:
			if a.multiArgumentsDialog != nil {
				placeOverlayCentered(a.multiArgumentsDialog.View().Content)
			}
		}
	}

	// Compacting indicator draws on top of everything (it is not part of
	// the dismissible modal stack — it reflects background work).
	if a.isCompacting {
		t := theme.CurrentTheme()
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderFocused()).
			BorderBackground(t.Background()).
			Padding(1, 2).
			Background(t.Background()).
			Foreground(t.Text())

		overlay := style.Render("Summarizing\n" + a.compactingMessage)
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay,
			appView,
			true,
		)
	}

	t := theme.CurrentTheme()
	bgColor := t.Background()
	bgCol := bgColor.Dark
	if bgCol == nil {
		bgCol = lipgloss.Color("#212121")
	}
	sBg := lipgloss.NewStyle().Background(bgCol).Render(" ")
	bgAnsi := ""
	if idx := strings.Index(sBg, " "); idx > 0 {
		bgAnsi = sBg[:idx]
	}

	if bgAnsi != "" && a.width > 0 {
		lines := strings.Split(appView, "\n")
		lineStyle := lipgloss.NewStyle().Width(a.width).Background(bgColor)
		for i, line := range lines {
			renderedLine := lineStyle.Render(line)
			renderedLine = styles.RepaintBackground(renderedLine, bgAnsi)
			lines[i] = renderedLine + "\x1b[0m"
		}
		appView = strings.Join(lines, "\n")
	}

	v := tea.NewView(appView)
	v.AltScreen = true
	// Enable mouse cell-motion mode so MouseWheelMsg events reach
	// messagesCmp.Update for chat-history scroll. Without this the wheel-
	// scroll handler at list.go is dead code; the terminal handles wheel
	// events natively (which alt-screen blocks). Cell-motion is enough for
	// click + wheel; AllMotion is overkill.
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func New(app *app.App) tea.Model {
	startPage := page.ChatPage
	model := &appModel{
		currentPage:      startPage,
		loadedPages:      make(map[page.PageID]bool),
		status:           core.NewStatusCmp(app),
		help:             dialog.NewHelpCmp(),
		quit:             dialog.NewQuitCmp(),
		sessionDialog:    dialog.NewSessionDialogCmp(),
		commandDialog:    dialog.NewCommandDialogCmp(),
		modelDialog:      dialog.NewModelDialogCmp(),
		permissions:      dialog.NewPermissionDialogCmp(),
		onboardingDialog: dialog.NewOnboardingCmp(),
		themeDialog:      dialog.NewThemeDialogCmp(),
		renameDialog:     dialog.NewRenameCmp(""),
		app:              app,
		commands:         []dialog.Command{},
		pages: map[page.PageID]tea.Model{
			page.ChatPage: page.NewChatPage(app),
			page.LogsPage: page.NewLogsPage(),
		},
		filepicker: dialog.NewFilepickerCmp(app),
		infoDialog: dialog.NewInfoDialogCmp(),
	}

	model.RegisterCommand(dialog.Command{
		ID:          "init",
		Title:       "Initialize Project",
		Description: "Create/Update the ACT.md memory file",
		Handler: func(cmd dialog.Command) tea.Cmd {
			prompt := `Please analyze this codebase and create an ACT.md file for multi-agent coordination. Include:

1. Build/lint/test commands (especially for running a single test)
2. Code style guidelines (imports, formatting, types, naming conventions, error handling)
3. Key architecture decisions and patterns that agents should follow
4. File ownership or areas of concern (which directories map to which functionality)

This file will be read by ACT swarm agents (developer, frontend, backend, QA, researcher) operating in this repository. Keep it about 20-30 lines.
If there's already an ACT.md or CLAUDE.md, improve it — don't overwrite important context.
If there are Cursor rules (.cursor/rules/) or Copilot rules (.github/copilot-instructions.md), incorporate them.`
			return tea.Batch(
				util.CmdHandler(chat.SendMsg{
					Text: prompt,
				}),
			)
		},
	})

	model.RegisterCommand(dialog.Command{
		ID:          "compact",
		Title:       "Compact Session",
		Description: "Summarize the current session and create a new one with the summary",
		Handler: func(cmd dialog.Command) tea.Cmd {
			return func() tea.Msg {
				return startCompactSessionMsg{}
			}
		},
	})

	model.RegisterCommand(dialog.Command{
		ID:          "act-agent:new-session",
		Title:       "New Session",
		Description: "Start a fresh ACT agent session",
		Handler: func(cmd dialog.Command) tea.Cmd {
			return util.CmdHandler(dialog.CreateNewSessionMsg{})
		},
	})
	// ACT coordination commands (HITL — bypass Planner, shell out direct).
	// Each command execs the act binary and renders stdout as a System message.
	directCmd := func(label string, argv ...string) func(dialog.Command) tea.Cmd {
		args := argv
		return func(cmd dialog.Command) tea.Cmd {
			return util.CmdHandler(runDirectCommandMsg{label: label, argv: args})
		}
	}
	model.RegisterCommand(dialog.Command{
		ID:          "/plan",
		Title:       "/plan <task>",
		Description: "Plan a task with Tier 1 Planner before executing",
		Handler: func(cmd dialog.Command) tea.Cmd {
			return util.ReportInfo("Type '/plan <task>' in the chat prompt to plan a task")
		},
	})
	model.RegisterCommand(dialog.Command{
		ID:          "/run",
		Title:       "/run <task>",
		Description: "Execute a task directly with Tier 1 Planner",
		Handler: func(cmd dialog.Command) tea.Cmd {
			return util.ReportInfo("Type '/run <task>' in the chat prompt to execute a task directly")
		},
	})
	model.RegisterCommand(dialog.Command{
		ID:          "act-agent:status",
		Title:       "ACT Status",
		Description: "Server health, registered agents, projects",
		Handler:     directCmd("act-agent:status", "status"),
	})
	model.RegisterCommand(dialog.Command{
		ID:          "act-agent:log",
		Title:       "ACT Log",
		Description: "Last 10 coordination log entries",
		Handler:     directCmd("act-agent:log", "log", "--tail", "10"),
	})
	model.RegisterCommand(dialog.Command{
		ID:          "act-agent:tasks",
		Title:       "ACT Tasks",
		Description: "Tasks awaiting validation",
		Handler:     directCmd("act-agent:tasks", "graph", "unverified"),
	})
	model.RegisterCommand(dialog.Command{
		ID:          "act-agent:validation",
		Title:       "ACT Validation Queue",
		Description: "Assurance queue (tasks pending validation)",
		Handler:     directCmd("act-agent:validation", "validation", "queue"),
	})
	model.RegisterCommand(dialog.Command{
		ID:          "act-agent:conflicts",
		Title:       "ACT File Conflicts",
		Description: "File lock conflicts between agents",
		Handler:     directCmd("act-agent:conflicts", "graph", "conflicts"),
	})
	model.RegisterCommand(dialog.Command{
		ID:          "act-agent:swarm",
		Title:       "ACT Swarm",
		Description: "Per-role backend selection (act-agent vs claude-code)",
		Handler:     directCmd("act-agent:swarm", "swarm"),
	})

	// Load custom commands
	customCommands, err := dialog.LoadCustomCommands()
	if err != nil {
		logging.Warn("Failed to load custom commands", "error", err)
	} else {
		for _, cmd := range customCommands {
			model.RegisterCommand(cmd)
		}
	}

	return model
}
