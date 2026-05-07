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

// firstPromptFlushMsg nudges the bubbletea event loop after the first prompt
// of a fresh session. On a fresh `.act/` directory the splash→chat transition
// can sit in the synchronized-output buffer until a second tea.Msg arrives —
// previously the user had to press Enter twice. Three staggered ticks (50ms,
// 150ms, 400ms) cover the timing window across slow/fast LLM first-token
// arrivals. Update doesn't switch on this type; the act of dispatch is enough
// to trigger a render frame.
type firstPromptFlushMsg struct{}

var ChatPage PageID = "chat"

type chatPage struct {
	app                  *app.App
	editor               layout.Container
	messages             layout.Container
	layout               layout.SplitPaneLayout
	navigator            layout.Container
	session              session.Session
	completionDialog     dialog.CompletionDialog
	showCompletionDialog bool
	showNavigator        bool
	firstSendDone        bool
}

type ChatKeyMap struct {
	ShowCompletionDialog key.Binding
	NewSession           key.Binding
	Cancel               key.Binding
	ToggleNavigator      key.Binding
}

var keyMap = ChatKeyMap{
	ShowCompletionDialog: key.NewBinding(
		key.WithKeys("@"),
		key.WithHelp("@", "Complete"),
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
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keyMap.ShowCompletionDialog):
			p.showCompletionDialog = true
			// Continue sending keys to layout->chat
		case key.Matches(msg, keyMap.NewSession):
			p.session = session.Session{}
			return p, util.CmdHandler(chat.SessionClearedMsg{})
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

	// Slash command intercept — handle /swarm, /nomik, /status, /help, etc.
	// before routing to the Planner. Unknown commands fall through.
	if strings.HasPrefix(trimmed, "/") {
		if response, handled := p.app.HandleSlashCommand(text); handled {
			return util.ReportInfo(response)
		}
	}

	// Palette-ID intercept — if the user typed a palette command literally
	// (e.g. "act:status"), dispatch the deterministic CLI handler instead of
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
	// This wires Observer monitoring, validation polling, QA polling, swarm
	// spawn, and Nomik project init to the active NesTTY session.
	p.app.Orchestrator.Start(context.Background(), p.session.ID)

	go p.app.Orchestrator.HandleHumanInput(context.Background(), p.session.ID, text, attachments...)

	// First-prompt flush nudge — see firstPromptFlushMsg comment. Without these
	// the user had to press Enter twice on the very first prompt of a fresh
	// `.act/` to see the response. Three staggered ticks cover the LLM-first-
	// token timing variance.
	if !p.firstSendDone {
		p.firstSendDone = true
		flush := func(time.Time) tea.Msg { return firstPromptFlushMsg{} }
		cmds = append(cmds,
			tea.Tick(50*time.Millisecond, flush),
			tea.Tick(150*time.Millisecond, flush),
			tea.Tick(400*time.Millisecond, flush),
		)
	}

	return tea.Batch(cmds...)
}

func paletteCmdArgv(s string) ([]string, bool) {
	switch s {
	case "act:status":
		return []string{"status"}, true
	case "act:log":
		return []string{"log", "--tail", "10"}, true
	case "act:tasks":
		return []string{"graph", "unverified"}, true
	case "act:validation":
		return []string{"validation", "queue"}, true
	case "act:conflicts":
		return []string{"graph", "conflicts"}, true
	case "act:swarm":
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
	cg := completions.NewFileAndFolderContextGroup()
	completionDialog := dialog.NewCompletionDialogCmp(cg)

	messagesContainer := layout.NewContainer(
		chat.NewMessagesCmp(app),
		layout.WithPadding(2, 1, 1, 1),
	)
	editorContainer := layout.NewContainer(
		chat.NewEditorCmp(app),
		layout.WithBorder(true, false, false, false),
	)
	navigatorContainer := layout.NewContainer(
		navigator.NewContextNavigator(app),
	)
	return &chatPage{
		app:              app,
		editor:           editorContainer,
		messages:         messagesContainer,
		navigator:        navigatorContainer,
		completionDialog: completionDialog,
		showNavigator:    true,
		layout: layout.NewSplitPane(
			layout.WithLeftPanel(messagesContainer),
			layout.WithRightPanel(navigatorContainer),
			layout.WithBottomPanel(editorContainer),
			layout.WithRatio(0.7),
		),
	}
}
