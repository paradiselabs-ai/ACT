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

var returnKey = key.NewBinding(
	key.WithKeys("esc"),
	key.WithHelp("esc", "close"),
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

	showPermissions bool
	permissions     dialog.PermissionDialogCmp

	showHelp bool
	help     dialog.HelpCmp

	showQuit bool
	quit     dialog.QuitDialog

	showSessionDialog bool
	sessionDialog     dialog.SessionDialog

	showCommandDialog bool
	commandDialog     dialog.CommandDialog
	commands          []dialog.Command
	// slide-in spring for the command palette (positive offset = shifted right, off-screen)
	cmdSlideOffset float64
	cmdSlideVel    float64
	cmdSlideSpring harmonica.Spring
	cmdSliding     bool

	showModelDialog bool
	modelDialog     dialog.ModelDialog

	showOnboardingDialog bool
	onboardingDialog     dialog.OnboardingCmp

	showFilepicker bool
	filepicker     dialog.FilepickerCmp

	showThemeDialog bool
	themeDialog     dialog.ThemeDialog

	showRenameDialog bool
	renameDialog     dialog.RenameDialog

	showMultiArgumentsDialog bool
	multiArgumentsDialog     *dialog.MultiArgumentsDialogCmp

	showInfoDialog bool
	infoDialog     dialog.InfoDialog

	isCompacting      bool
	compactingMessage string
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

		if a.showMultiArgumentsDialog {
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
		a.showPermissions = true
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
		a.showPermissions = false
		return a, cmd

	case page.PageChangeMsg:
		return a, a.moveToPage(msg.ID)

	case dialog.CloseQuitMsg:
		a.showQuit = false
		return a, nil

	case dialog.CloseSessionDialogMsg:
		a.showSessionDialog = false
		return a, nil

	case dialog.CloseRenameDialogMsg:
		a.showRenameDialog = false
		return a, nil

	case dialog.RenameCompletedMsg:
		a.showRenameDialog = false
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
		a.showCommandDialog = false
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
		a.showThemeDialog = false
		return a, nil

	case dialog.ThemeChangedMsg:
		a.pages[a.currentPage], cmd = a.pages[a.currentPage].Update(msg)
		a.showThemeDialog = false
		return a, tea.Batch(cmd, util.ReportInfo("Theme changed to: "+msg.ThemeName))

	case dialog.CloseModelDialogMsg:
		a.showModelDialog = false
		return a, nil

	case dialog.ModelSelectedMsg:
		a.showModelDialog = false

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
		a.showOnboardingDialog = msg.Show
		return a, nil

	case dialog.CloseOnboardingMsg:
		a.showOnboardingDialog = false
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
		a.showSessionDialog = false
		if a.currentPage == page.ChatPage {
			return a, util.CmdHandler(chat.SessionSelectedMsg(msg.Session))
		}
		return a, nil

	case dialog.CreateNewSessionMsg:
		a.showSessionDialog = false
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
		a.showInfoDialog = true
		return a, nil

	case dialog.CloseInfoDialogMsg:
		a.showInfoDialog = false
		return a, nil

	case dialog.ToggleHelpMsg:
		a.showHelp = !a.showHelp
		return a, nil

	case dialog.ShowLogsMsg:
		return a, a.moveToPage(page.LogsPage)

	case dialog.ShowModelDialogMsg:
		a.showModelDialog = true
		a.modelDialog.SetRole(msg.Role)
		return a, a.modelDialog.Init()

	case dialog.CommandSelectedMsg:
		a.showCommandDialog = false
		// Execute the command handler if available
		if msg.Command.Handler != nil {
			return a, msg.Command.Handler(msg.Command)
		}
		return a, util.ReportInfo("Command selected: " + msg.Command.Title)

	case dialog.ShowMultiArgumentsDialogMsg:
		// Show multi-arguments dialog
		m := dialog.NewMultiArgumentsDialogCmp(msg.CommandID, msg.Content, msg.ArgNames)
		a.multiArgumentsDialog = &m
		a.showMultiArgumentsDialog = true
		return a, a.multiArgumentsDialog.Init()

	case dialog.CloseMultiArgumentsDialogMsg:
		// Close multi-arguments dialog
		a.showMultiArgumentsDialog = false

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
			// Unwind in strict reverse z-order. View() overlays in this
			// order: permissions → onboarding → filepicker → help → quit →
			// info → session → model → command → theme → rename →
			// multiArgs (later = visually on top). Esc must close the TOP
			// layer first; the previous fixed order diverged from draw
			// order and could dismiss a background dialog while a
			// foreground one stayed up. Permissions intentionally absent:
			// it must be answered (allow/deny), not dismissed.
			if a.showMultiArgumentsDialog {
				a.showMultiArgumentsDialog = false
				return a, nil
			}
			if a.showRenameDialog {
				a.showRenameDialog = false
				return a, nil
			}
			if a.showThemeDialog {
				a.showThemeDialog = false
				return a, nil
			}
			if a.showCommandDialog {
				a.showCommandDialog = false
				return a, nil
			}
			if a.showModelDialog {
				a.showModelDialog = false
				return a, nil
			}
			if a.showSessionDialog {
				a.showSessionDialog = false
				return a, nil
			}
			if a.showInfoDialog {
				a.showInfoDialog = false
				return a, nil
			}
			if a.showQuit {
				a.showQuit = false
				return a, nil
			}
			if a.showHelp {
				a.showHelp = false
				return a, nil
			}
			if a.showFilepicker {
				a.showFilepicker = false
				a.filepicker.ToggleFilepicker(a.showFilepicker)
				return a, nil
			}
			if a.currentPage == page.LogsPage {
				return a, a.moveToPage(page.ChatPage)
			}
		}

		// If multi-arguments dialog is open, let it handle the key press first
		if a.showMultiArgumentsDialog {
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
			if a.showPermissions {
				return a, util.ReportWarn("Answer the permission prompt first (a/s/d)")
			}
			a.showQuit = !a.showQuit
			if a.showHelp {
				a.showHelp = false
			}
			if a.showSessionDialog {
				a.showSessionDialog = false
			}
			if a.showCommandDialog {
				a.showCommandDialog = false
			}
			if a.showFilepicker {
				a.showFilepicker = false
				a.filepicker.ToggleFilepicker(a.showFilepicker)
			}
			if a.showModelDialog {
				a.showModelDialog = false
			}
			if a.showMultiArgumentsDialog {
				a.showMultiArgumentsDialog = false
			}
			return a, nil
		case key.Matches(msg, keys.SwitchSession):
			if a.currentPage == page.ChatPage && !a.showQuit && !a.showPermissions && !a.showCommandDialog {
				// Load sessions and show the dialog
				sessions, err := a.app.Sessions.List(context.Background())
				if err != nil {
					return a, util.ReportError(err)
				}
				if len(sessions) == 0 {
					return a, util.ReportWarn("No sessions available")
				}
				a.sessionDialog.SetSessions(sessions)
				a.showSessionDialog = true
				return a, nil
			}
			return a, nil
		case key.Matches(msg, keys.Commands):
			if a.currentPage == page.ChatPage && !a.showQuit && !a.showPermissions && !a.showSessionDialog && !a.showThemeDialog && !a.showFilepicker {
				// Show commands dialog
				if len(a.commands) == 0 {
					return a, util.ReportWarn("No commands available")
				}
				animCmd := a.commandDialog.SetCommands(a.commands)
				a.showCommandDialog = true
				// Start slide-in from +40 cols to the right (stiffness=14, damping=0.6)
				a.cmdSlideSpring = anim.NewSpring(14, 0.6)
				a.cmdSlideOffset = 40
				a.cmdSlideVel = 0
				a.cmdSliding = true
				return a, tea.Batch(animCmd, anim.Frame())
			}
			return a, nil
		case key.Matches(msg, keys.Models):
			if a.showModelDialog {
				a.showModelDialog = false
				return a, nil
			}
			if a.currentPage == page.ChatPage && !a.showQuit && !a.showPermissions && !a.showSessionDialog && !a.showCommandDialog {
				a.showModelDialog = true
				return a, nil
			}
			return a, nil
		case key.Matches(msg, keys.SwitchTheme):
			if !a.showQuit && !a.showPermissions && !a.showSessionDialog && !a.showCommandDialog {
				// Show theme switcher dialog
				a.showThemeDialog = true
				// Theme list is dynamically loaded by the dialog component
				return a, a.themeDialog.Init()
			}
			return a, nil
		case key.Matches(msg, keys.RenameSession):
			if a.currentPage == page.ChatPage && !a.showQuit && !a.showPermissions && !a.showSessionDialog && !a.showCommandDialog && !a.showThemeDialog && !a.showFilepicker {
				if a.selectedSession.ID != "" {
					a.renameDialog.SetTitle(a.selectedSession.Title)
					a.showRenameDialog = true
					return a, nil
				}
			}
			return a, nil
		case key.Matches(msg, returnKey) || key.Matches(msg):
			if msg.String() == quitKey || msg.String() == "esc" || msg.String() == "q" {
				if a.currentPage == page.LogsPage {
					return a, a.moveToPage(page.ChatPage)
				}
			} else if !a.filepicker.IsCWDFocused() {
				if a.showQuit {
					a.showQuit = !a.showQuit
					return a, nil
				}
				if a.showHelp {
					a.showHelp = !a.showHelp
					return a, nil
				}
				if a.showOnboardingDialog {
					a.showOnboardingDialog = false
					if err := config.MarkProjectInitialized(); err != nil {
						return a, util.ReportError(err)
					}
					return a, nil
				}
				if a.showFilepicker {
					a.showFilepicker = false
					a.filepicker.ToggleFilepicker(a.showFilepicker)
					return a, nil
				}
				if a.showRenameDialog {
					a.showRenameDialog = false
					return a, nil
				}
				if a.currentPage == page.LogsPage {
					return a, a.moveToPage(page.ChatPage)
				}
			}
		case key.Matches(msg, keys.Logs):
			return a, a.moveToPage(page.LogsPage)
		case key.Matches(msg, keys.Help):
			if a.showQuit {
				return a, nil
			}
			a.showHelp = !a.showHelp
			return a, nil
		case key.Matches(msg, helpEsc):
			if a.app.Orchestrator.IsAnyBusy("") {
				if a.showQuit {
					return a, nil
				}
				a.showHelp = !a.showHelp
				return a, nil
			}
		case key.Matches(msg, keys.Filepicker):
			a.showFilepicker = !a.showFilepicker
			a.filepicker.ToggleFilepicker(a.showFilepicker)
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

	if a.showFilepicker {
		f, filepickerCmd := a.filepicker.Update(msg)
		a.filepicker = f.(dialog.FilepickerCmp)
		cmds = append(cmds, filepickerCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showQuit {
		q, quitCmd := a.quit.Update(msg)
		a.quit = q.(dialog.QuitDialog)
		cmds = append(cmds, quitCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}
	if a.showPermissions {
		d, permissionsCmd := a.permissions.Update(msg)
		a.permissions = d.(dialog.PermissionDialogCmp)
		cmds = append(cmds, permissionsCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showInfoDialog {
		d, infoCmd := a.infoDialog.Update(msg)
		a.infoDialog = d.(dialog.InfoDialog)
		cmds = append(cmds, infoCmd)
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showSessionDialog {
		d, sessionCmd := a.sessionDialog.Update(msg)
		a.sessionDialog = d.(dialog.SessionDialog)
		cmds = append(cmds, sessionCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showCommandDialog {
		d, commandCmd := a.commandDialog.Update(msg)
		a.commandDialog = d.(dialog.CommandDialog)
		cmds = append(cmds, commandCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showModelDialog {
		d, modelCmd := a.modelDialog.Update(msg)
		a.modelDialog = d.(dialog.ModelDialog)
		cmds = append(cmds, modelCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showOnboardingDialog {
		d, onboardCmd := a.onboardingDialog.Update(msg)
		a.onboardingDialog = d.(dialog.OnboardingCmp)
		cmds = append(cmds, onboardCmd)
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showThemeDialog {
		d, themeCmd := a.themeDialog.Update(msg)
		a.themeDialog = d.(dialog.ThemeDialog)
		cmds = append(cmds, themeCmd)
		// Only block key messages send all other messages down
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
		}
	}

	if a.showRenameDialog {
		d, renameCmd := a.renameDialog.Update(msg)
		a.renameDialog = d.(dialog.RenameDialog)
		cmds = append(cmds, renameCmd)
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return a, tea.Batch(cmds...)
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
	if a.app.Orchestrator.IsAnyBusy("") {
		// For now we don't move to any page if the agent is busy
		return util.ReportWarn("Agent is busy, please wait...")
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

	if a.showPermissions {
		overlay := a.permissions.View()
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay.Content) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay.Content) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay.Content,
			appView,
			true,
		)
	}

	if a.showOnboardingDialog {
		overlay := a.onboardingDialog.View()
		appView = layout.PlaceOverlay(
			a.width/2-lipgloss.Width(overlay.Content)/2,
			a.height/2-lipgloss.Height(overlay.Content)/2,
			overlay.Content,
			appView,
			true,
		)
	}

	if a.showFilepicker {
		overlay := a.filepicker.View()
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay.Content) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay.Content) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay.Content,
			appView,
			true,
		)

	}

	// Show compacting status overlay
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

	if a.showHelp {
		bindings := layout.KeyMapToSlice(keys)
		if p, ok := a.pages[a.currentPage].(layout.Bindings); ok {
			bindings = append(bindings, p.BindingKeys()...)
		}
		if a.showPermissions {
			bindings = append(bindings, a.permissions.BindingKeys()...)
		}
		if a.currentPage == page.LogsPage {
			bindings = append(bindings, logsKeyReturnKey)
		}
		if !a.app.Orchestrator.IsAnyBusy("") {
			bindings = append(bindings, helpEsc)
		}
		a.help.SetBindings(bindings)

		overlay := a.help.View()
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay.Content) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay.Content) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay.Content,
			appView,
			true,
		)
	}

	if a.showQuit {
		overlay := a.quit.View()
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay.Content) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay.Content) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay.Content,
			appView,
			true,
		)
	}

	if a.showInfoDialog {
		overlay := a.infoDialog.View()
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay.Content) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay.Content) / 2
		if row < 0 {
			row = 0
		}
		if col < 0 {
			col = 0
		}
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay.Content,
			appView,
			true,
		)
	}

	if a.showSessionDialog {
		overlay := a.sessionDialog.View()
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay.Content) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay.Content) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay.Content,
			appView,
			true,
		)
	}

	if a.showModelDialog {
		overlay := a.modelDialog.View()
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay.Content) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay.Content) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay.Content,
			appView,
			true,
		)
	}

	if a.showCommandDialog {
		overlay := a.commandDialog.View()
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay.Content) / 2
		col := lipgloss.Width(appView)/2 - lipgloss.Width(overlay.Content)/2
		col += int(a.cmdSlideOffset)
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay.Content,
			appView,
			true,
		)
	}

	if a.showThemeDialog {
		overlay := a.themeDialog.View()
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay.Content) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay.Content) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay.Content,
			appView,
			true,
		)
	}

	if a.showRenameDialog {
		overlay := a.renameDialog.View()
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay.Content) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay.Content) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay.Content,
			appView,
			true,
		)
	}

	if a.showMultiArgumentsDialog {
		overlay := a.multiArgumentsDialog.View()
		row := lipgloss.Height(appView) / 2
		row -= lipgloss.Height(overlay.Content) / 2
		col := lipgloss.Width(appView) / 2
		col -= lipgloss.Width(overlay.Content) / 2
		appView = layout.PlaceOverlay(
			col,
			row,
			overlay.Content,
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
			renderedLine = strings.ReplaceAll(renderedLine, "\x1b[0m", "\x1b[0m"+bgAnsi)
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
