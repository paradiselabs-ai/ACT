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
// Safety cap: flushTickCap (240 = 60s @ 250ms) prevents an infinite loop if
// IsAnyBusy ever fails to transition to false.
type flushTickMsg struct{}

const flushTickCap = 240 // 60 seconds @ 250ms intervals

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
	scrollFocused        bool
	flushTickCount       int
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
	case flushTickMsg:
		// While any Tier 1 role is busy, keep nudging the event loop so the
		// synchronized-output buffer flushes mid-stream. Stops when all roles
		// idle or after flushTickCap iterations (safety).
		if p.flushTickCount >= flushTickCap {
			p.flushTickCount = 0
			return p, nil
		}
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
	// This wires Observer monitoring, validation polling, QA polling, swarm
	// spawn, and Nomik project init to the active NesTTY session.
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
	cg := completions.NewFileAndFolderContextGroup()
	completionDialog := dialog.NewCompletionDialogCmp(cg)

	messagesContainer := layout.NewContainer(
		chat.NewMessagesCmp(app),
		layout.WithPadding(2, 1, 1, 1),
	)
	editorContainer := layout.NewContainer(
		chat.NewEditorCmp(app),
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
