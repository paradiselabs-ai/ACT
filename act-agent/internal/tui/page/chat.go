package page

import (
	"context"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/app"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/completions"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/message"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/session"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/components/chat"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/components/dialog"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/components/navigator"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/layout"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/util"
)

// flushTickMsg nudges the bubbletea event loop while a Tier 1 agent is
// streaming a response. The synchronized-output buffer can hold mid-stream
// frames until a tea.Msg arrives to flush — manifests as the response
// pausing mid-sentence until the user hits a key.
//
// Mechanism: on every chat send, dispatch a flushTickMsg. The Update handler
// for flushTickMsg checks whether any Tier 1 role is still busy. If yes,
// schedule another flushTickMsg in 250ms. If no, stop. Each dispatch forces
// Update→View→render → BSU/ESU emission → terminal commits.
//
// Why this is needed: the harmonica anim.Frame() loop in messagesCmp only
// runs while splashAnimating is true (~1s after launch). After splash
// settles, no continuous tick exists. Without one, mid-stream frames buffer
// indefinitely on slow LLM streams.
//
// Safety cap removed: the loop's stop condition is IsAnyBusy transitioning
// to false, which is authoritative. The old flushTickCap (240 ticks = 60s)
// permanently ended nudging after one minute since the last send — any
// Tier-1 run longer than that reinstated the exact freeze this loop exists
// to fix (frames buffered until a keypress).
type flushTickMsg struct{}

var ChatPage PageID = "chat"

type chatPage struct {
	app                     *app.App
	editor                  layout.Container
	messages                layout.Container
	layout                  layout.SplitPaneLayout
	navigator               layout.Container
	session                 session.Session
	completionDialog        dialog.CompletionDialog
	fileCompletionProvider  dialog.CompletionProvider
	slashCompletionProvider dialog.CompletionProvider
	showCompletionDialog    bool
	showNavigator           bool
	firstSendDone           bool
	scrollFocused           bool
	flushTickCount          int
}

type ChatKeyMap struct {
	ShowCompletionDialog key.Binding
	ShowSlashCommands    key.Binding
	NewSession           key.Binding
	Cancel               key.Binding
	ToggleNavigator      key.Binding
}

var keyMap = ChatKeyMap{
	ShowCompletionDialog: key.NewBinding(
		key.WithKeys("@"),
		key.WithHelp("@", "Complete context"),
	),
	ShowSlashCommands: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "Slash actions"),
	),
	NewSession: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", "new session"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	ToggleNavigator: key.NewBinding(
		key.WithKeys("ctrl+p"),
		key.WithHelp("ctrl+p", "toggle context"),
	),
}

func (p *chatPage) Init() tea.Cmd {
	cmds := []tea.Cmd{
		p.layout.Init(),
		p.completionDialog.Init(),
	}
	return tea.Batch(cmds...)
}

func (p *chatPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmd := p.layout.SetSize(msg.Width, msg.Height)
		cmds = append(cmds, cmd)
	case dialog.CompletionDialogCloseMsg:
		p.showCompletionDialog = false
	case flushTickMsg:
		// While any Tier 1 role is busy, keep nudging the event loop so the
		// synchronized-output buffer flushes mid-stream. Stops when all roles
		// idle. (No wall-clock cap — see the flushTickMsg doc comment: a cap
		// reinstates the freeze on runs longer than the cap.)
		if p.app.Orchestrator != nil && !p.app.Orchestrator.IsAnyBusy(p.session.ID) {
			p.flushTickCount = 0
			return p, nil
		}
		p.flushTickCount++
		return p, tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
			return flushTickMsg{}
		})
	case chat.SendMsg:
		cmd := p.sendMessage(msg.Text, msg.Attachments)
		if cmd != nil {
			return p, cmd
		}
	case dialog.CommandRunCustomMsg:
		// Check if the agent is busy before executing custom commands
		if p.app.Orchestrator.IsAnyBusy(p.session.ID) {
			return p, util.ReportWarn("Agent is busy, please wait before executing a command...")
		}

		// Process the command content with arguments if any
		content := msg.Content
		if msg.Args != nil {
			// Replace all named arguments with their values
			for name, value := range msg.Args {
				placeholder := "$" + name
				content = strings.ReplaceAll(content, placeholder, value)
			}
		}

		// Handle custom command execution
		cmd := p.sendMessage(content, nil)
		if cmd != nil {
			return p, cmd
		}
	case chat.SessionSelectedMsg:
		p.session = msg
		if msg.ID != "" && p.app.Orchestrator != nil {
			p.app.Orchestrator.Start(context.Background(), msg.ID)
		}
	case tea.KeyPressMsg:
		// Tab toggles scroll-focus mode — gives arrow keys to the viewport.
		// Available even while Tier 1 agents are busy so the user can scroll
		// back through history during long Planner responses.
		if msg.String() == "tab" {
			p.scrollFocused = !p.scrollFocused
			return p, util.CmdHandler(chat.ScrollFocusMsg{On: p.scrollFocused})
		}
		// When scroll-focused, intercept directional keys and esc so the
		// editor never sees them. Emit ScrollMsg so messagesCmp can move the
		// viewport without needing a key event.
		if p.scrollFocused {
			switch msg.String() {
			case "esc", "tab":
				p.scrollFocused = false
				return p, util.CmdHandler(chat.ScrollFocusMsg{On: false})
			case "up":
				return p, util.CmdHandler(chat.ScrollMsg{Lines: -1})
			case "down":
				return p, util.CmdHandler(chat.ScrollMsg{Lines: 1})
			case "pgup":
				return p, util.CmdHandler(chat.ScrollMsg{Lines: -20})
			case "pgdown":
				return p, util.CmdHandler(chat.ScrollMsg{Lines: 20})
			case "ctrl+u":
				return p, util.CmdHandler(chat.ScrollMsg{Lines: -10})
			case "ctrl+d":
				return p, util.CmdHandler(chat.ScrollMsg{Lines: 10})
			}
			// Swallow all other keys while scroll-focused.
			return p, nil
		}
		if msg.String() == "ctrl+up" || msg.String() == "alt+up" {
			return p, util.CmdHandler(chat.ScrollMsg{Lines: -1})
		}
		if msg.String() == "ctrl+down" || msg.String() == "alt+down" {
			return p, util.CmdHandler(chat.ScrollMsg{Lines: 1})
		}
		switch {
		case key.Matches(msg, keyMap.ShowCompletionDialog):
			p.completionDialog = dialog.NewCompletionDialogCmp(p.fileCompletionProvider)
			p.showCompletionDialog = true
		case key.Matches(msg, keyMap.ShowSlashCommands):
			p.completionDialog = dialog.NewCompletionDialogCmp(p.slashCompletionProvider)
			p.showCompletionDialog = true
		case key.Matches(msg, keyMap.NewSession):
			return p, util.CmdHandler(dialog.CreateNewSessionMsg{})
		case key.Matches(msg, keyMap.Cancel):
			if p.session.ID != "" {
				// Cancel the current session's generation process
				// This allows users to interrupt long-running operations
				p.app.Orchestrator.CancelActive(p.session.ID)
				return p, nil
			}
		case key.Matches(msg, keyMap.ToggleNavigator):
			p.showNavigator = !p.showNavigator
			if p.showNavigator {
				return p, p.layout.SetRightPanel(p.navigator)
			}
			return p, p.layout.ClearRightPanel()
		}
	}
	if p.showCompletionDialog {
		context, contextCmd := p.completionDialog.Update(msg)
		p.completionDialog = context.(dialog.CompletionDialog)
		cmds = append(cmds, contextCmd)

		// Doesn't forward event if enter key is pressed
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			if keyMsg.String() == "enter" {
				return p, tea.Batch(cmds...)
			}
		}
	}

	u, cmd := p.layout.Update(msg)
	cmds = append(cmds, cmd)
	p.layout = u.(layout.SplitPaneLayout)

	return p, tea.Batch(cmds...)
}

func (p *chatPage) sendMessage(text string, attachments []message.Attachment) tea.Cmd {
	var cmds []tea.Cmd

	trimmed := strings.TrimSpace(text)

	// Slash command intercept — handle TUI actions (/help, /log, /role, /clear, /init, /compact, /status, /swarm, /backend)
	if strings.HasPrefix(trimmed, "/") {
		cmdLower := strings.ToLower(trimmed)
		fields := strings.Fields(cmdLower)
		var firstWord string
		if len(fields) > 0 {
			firstWord = fields[0]
		}

		switch firstWord {
		case "/help":
			return util.CmdHandler(dialog.ToggleHelpMsg{})
		case "/log", "/logs":
			return util.CmdHandler(dialog.ShowLogsMsg{})
		case "/role", "/model":
			role := ""
			if len(fields) > 1 {
				role = fields[1]
			}
			return util.CmdHandler(dialog.ShowModelDialogMsg{Role: role})
		case "/plan":
			task := strings.TrimSpace(trimmed[len(fields[0]):])
			if task == "" {
				return util.CmdHandler(util.ReportInfo("Usage: /plan <task description> — create an implementation plan before executing"))
			}
			return p.sendMessage("Create a detailed implementation plan for: "+task, nil)
		case "/run":
			task := strings.TrimSpace(trimmed[len(fields[0]):])
			if task == "" {
				return util.CmdHandler(util.ReportInfo("Usage: /run <task description> — execute a task directly"))
			}
			return p.sendMessage("Execute this task directly: "+task, nil)
		case "/clear":
			p.session = session.Session{}
			return util.CmdHandler(chat.SessionClearedMsg{})
		case "/compact":
			return func() tea.Msg { return dialog.StartCompactSessionMsg{} }
		case "/init":
			initPrompt := `Please analyze this codebase and create an ACT.md file for multi-agent coordination. Include:

1. Build/lint/test commands (especially for running a single test)
2. Code style guidelines (imports, formatting, types, naming conventions, error handling)
3. Key architecture decisions and patterns that agents should follow
4. File ownership or areas of concern (which directories map to which functionality)

This file will be read by ACT swarm agents (developer, frontend, backend, QA, researcher) operating in this repository. Keep it about 20-30 lines.
If there's already an ACT.md or CLAUDE.md, improve it — don't overwrite important context.
If there are Cursor rules (.cursor/rules/) or Copilot rules (.github/copilot-instructions.md), incorporate them.`
			return p.sendMessage(initPrompt, nil)
		}

		if response, handled := p.app.HandleSlashCommand(text); handled {
			title := "Command Output"
			switch firstWord {
			case "/status":
				title = "ACT System Status"
			case "/swarm":
				title = "Swarm Status & Configuration"
			case "/backend":
				title = "Tier 1 Backend Configuration"
			}
			return util.CmdHandler(dialog.ShowInfoDialogMsg{
				Title:   title,
				Content: response,
			})
		}
	}

	// Palette-ID intercept — if the user typed a palette command literally
	// (e.g. "act-agent:status"), dispatch the deterministic CLI handler instead of
	// routing to the Planner. Keep this argv map in sync with tui.go.
	if argv, ok := paletteCmdArgv(trimmed); ok {
		if p.session.ID == "" {
			sess, err := p.app.Sessions.Create(context.Background(), "New Session")
			if err != nil {
				return util.ReportError(err)
			}
			p.session = sess
			cmds = append(cmds, util.CmdHandler(chat.SessionSelectedMsg(sess)))
		}
		sid := p.session.ID
		go p.app.Orchestrator.RunDirectCommand(context.Background(), sid, trimmed, argv)
		return tea.Batch(cmds...)
	}

	if p.session.ID == "" {
		session, err := p.app.Sessions.Create(context.Background(), "New Session")
		if err != nil {
			return util.ReportError(err)
		}

		p.session = session
		cmds = append(cmds, util.CmdHandler(chat.SessionSelectedMsg(session)))
	}

	// Start orchestrator background loops on first message (idempotent).
	// This wires Observer monitoring, validation polling, QA polling, and swarm
	// spawn to the active NesTTY session.
	p.app.Orchestrator.Start(context.Background(), p.session.ID)

	go p.app.Orchestrator.HandleHumanInput(context.Background(), p.session.ID, text, attachments...)

	// Flush nudge — see flushTickMsg comment. Kicks off a self-extending tick
	// loop that runs while any Tier 1 role is busy, flushing the synchronized-
	// output buffer mid-stream so the response renders incrementally instead
	// of in one block when the user happens to keypress.
	//
	// Two early ticks (50ms, 250ms) catch the cold-start window before the
	// orchestrator has marked the agent busy. Subsequent ticks self-extend
	// from the flushTickMsg handler in Update.
	p.flushTickCount = 0
	flushKick := func(time.Time) tea.Msg { return flushTickMsg{} }
	cmds = append(cmds,
		tea.Tick(50*time.Millisecond, flushKick),
		tea.Tick(250*time.Millisecond, flushKick),
	)
	if !p.firstSendDone {
		p.firstSendDone = true
	}

	return tea.Batch(cmds...)
}

func paletteCmdArgv(s string) ([]string, bool) {
	switch s {
	case "act-agent:status":
		return []string{"status"}, true
	case "act-agent:log":
		return []string{"log", "--tail", "10"}, true
	case "act-agent:tasks":
		return []string{"graph", "unverified"}, true
	case "act-agent:validation":
		return []string{"validation", "queue"}, true
	case "act-agent:conflicts":
		return []string{"graph", "conflicts"}, true
	case "act-agent:swarm":
		return []string{"swarm"}, true
	}
	return nil, false
}

func (p *chatPage) SetSize(width, height int) tea.Cmd {
	return p.layout.SetSize(width, height)
}

func (p *chatPage) GetSize() (int, int) {
	return p.layout.GetSize()
}

func (p *chatPage) View() tea.View {
	layoutViewContent := p.layout.View().Content

	if p.showCompletionDialog {
		_, layoutHeight := p.layout.GetSize()
		editorWidth, editorHeight := p.editor.GetSize()

		p.completionDialog.SetWidth(editorWidth)
		overlay := p.completionDialog.View()

		layoutViewContent = layout.PlaceOverlay(
			0,
			layoutHeight-editorHeight-lipgloss.Height(overlay.Content),
			overlay.Content,
			layoutViewContent,
			false,
		)
	}

	return tea.NewView(layoutViewContent)
}

func (p *chatPage) BindingKeys() []key.Binding {
	bindings := layout.KeyMapToSlice(keyMap)
	bindings = append(bindings, p.messages.BindingKeys()...)
	bindings = append(bindings, p.editor.BindingKeys()...)
	return bindings
}

func NewChatPage(app *app.App) tea.Model {
	fileCg := completions.NewFileAndFolderContextGroup()
	slashCg := completions.NewSlashCommandsContextGroup()
	completionDialog := dialog.NewCompletionDialogCmp(fileCg)

	messagesContainer := layout.NewContainer(
		chat.NewMessagesCmp(app),
		layout.WithPadding(1, 1, 0, 1),
	)
	editorContainer := layout.NewContainer(
		chat.NewEditorCmp(app),
	)
	navigatorContainer := layout.NewContainer(
		navigator.NewContextNavigator(app),
	)
	return &chatPage{
		app:                     app,
		editor:                  editorContainer,
		messages:                messagesContainer,
		navigator:               navigatorContainer,
		completionDialog:        completionDialog,
		fileCompletionProvider:  fileCg,
		slashCompletionProvider: slashCg,
		showNavigator:           true,
		layout: layout.NewSplitPane(
			layout.WithLeftPanel(messagesContainer),
			layout.WithRightPanel(navigatorContainer),
			layout.WithBottomPanel(editorContainer),
			layout.WithRatio(0.7),
		),
	}
}
