package chat

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/app"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/message"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/pubsub"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/session"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/components/dialog"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/util"
)

type cacheItem struct {
	width    int
	role     string
	finished bool // tracks IsFinished() — flipping from false→true invalidates
	content  []uiMessage
}

// markdownRenderedMsg is the async result from a Glamour render. The
// rendering itself runs in a goroutine off the Bubbletea Update loop so the
// seconds-long CommonMark parse + chroma highlight never blocks the UI.
// When this msg arrives we install the rendered uiMessages into
// cachedContent and trigger a rerender; the sync renderView path renders
// plain text for the same msgID in the meantime.
type markdownRenderedMsg struct {
	sessionID string
	msgID     string
	width     int
	role      string
	rendered  []uiMessage
}

type messagesCmp struct {
	app           *app.App
	width, height int
	viewport      viewport.Model
	session       session.Session
	messages      []message.Message
	uiMessages    []uiMessage
	currentMsgID  string
	cachedContent map[string]cacheItem
	// asyncGlamour marks msgIDs currently being rendered through Glamour in a
	// background goroutine. While a msgID is in this map, renderView renders
	// plain text for it (useMarkdown=false) so the sync path stays fast.
	// Cleared when markdownRenderedMsg arrives.
	asyncGlamour map[string]bool
	spinner      spinner.Model
	attachments  viewport.Model
}

type MessageKeys struct {
	PageDown     key.Binding
	PageUp       key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
}

var messageKeys = MessageKeys{
	PageDown: key.NewBinding(
		key.WithKeys("pgdown"),
		key.WithHelp("f/pgdn", "page down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup"),
		key.WithHelp("b/pgup", "page up"),
	),
	HalfPageUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "½ page up"),
	),
	HalfPageDown: key.NewBinding(
		key.WithKeys("ctrl+d", "ctrl+d"),
		key.WithHelp("ctrl+d", "½ page down"),
	),
}

func (m *messagesCmp) Init() tea.Cmd {
	return tea.Batch(m.viewport.Init(), m.spinner.Tick)
}

func (m *messagesCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case dialog.ThemeChangedMsg:
		// Theme change invalidates every Glamour-rendered body (chroma
		// palette is theme-derived). Drop cache + any in-flight marks and
		// re-queue off the Update loop.
		m.asyncGlamour = make(map[string]bool)
		m.rerender()
		return m, m.queueAsyncGlamour()
	case SessionSelectedMsg:
		if msg.ID != m.session.ID {
			cmd := m.SetSession(msg)
			return m, cmd
		}
		return m, nil
	case SessionClearedMsg:
		m.session = session.Session{}
		m.messages = make([]message.Message, 0)
		m.currentMsgID = ""
		return m, nil

	case tea.KeyPressMsg:
		if key.Matches(msg, messageKeys.PageUp) || key.Matches(msg, messageKeys.PageDown) ||
			key.Matches(msg, messageKeys.HalfPageUp) || key.Matches(msg, messageKeys.HalfPageDown) {
			u, cmd := m.viewport.Update(msg)
			m.viewport = u
			cmds = append(cmds, cmd)
		}
	case pubsub.Event[session.Session]:
		if msg.Type == pubsub.UpdatedEvent && msg.Payload.ID == m.session.ID {
			m.session = msg.Payload
			if m.session.SummaryMessageID == m.currentMsgID {
				delete(m.cachedContent, m.currentMsgID)
				m.renderView()
			}
		}
	case markdownRenderedMsg:
		// Stale result — session switched or window resized between kickoff
		// and completion. Drop it and re-queue so the msg doesn't stay
		// stuck in plain-text forever.
		if msg.sessionID != m.session.ID || msg.width != m.width {
			delete(m.asyncGlamour, msg.msgID)
			return m, m.queueAsyncGlamour()
		}
		delete(m.asyncGlamour, msg.msgID)
		m.cachedContent[msg.msgID] = cacheItem{
			width:    msg.width,
			role:     msg.role,
			finished: true,
			content:  msg.rendered,
		}
		m.renderView()
		return m, nil

	case pubsub.Event[message.Message]:
		needsRerender := false
		if msg.Type == pubsub.CreatedEvent {
			if msg.Payload.SessionID == m.session.ID {

				messageExists := false
				for _, v := range m.messages {
					if v.ID == msg.Payload.ID {
						messageExists = true
						break
					}
				}

				if !messageExists {
					// Only invalidate the previous last-message cache if the
					// new message is a tool response matching a tool_use in
					// that message. Blindly invalidating every time meant a
					// finished assistant message re-rendered through Glamour
					// on every subsequent message creation — one of the
					// dominant freeze contributors per KI-01.
					if len(m.messages) > 0 && msg.Payload.Role == message.Tool {
						prev := m.messages[len(m.messages)-1]
						toolResults := msg.Payload.ToolResults()
						for _, tr := range toolResults {
							for _, tc := range prev.ToolCalls() {
								if tc.ID == tr.ToolCallID {
									delete(m.cachedContent, prev.ID)
									break
								}
							}
						}
					}

					m.messages = append(m.messages, msg.Payload)
					delete(m.cachedContent, m.currentMsgID)
					m.currentMsgID = msg.Payload.ID
					needsRerender = true
				}
			}
			// There are tool calls from the child task
			for _, v := range m.messages {
				for _, c := range v.ToolCalls() {
					if c.ID == msg.Payload.SessionID {
						delete(m.cachedContent, v.ID)
						needsRerender = true
					}
				}
			}
		} else if msg.Type == pubsub.UpdatedEvent && msg.Payload.SessionID == m.session.ID {
			for i, v := range m.messages {
				if v.ID == msg.Payload.ID {
					m.messages[i] = msg.Payload
					delete(m.cachedContent, msg.Payload.ID)
					needsRerender = true
					break
				}
			}
		}
		if needsRerender {
			// Queue async Glamour BEFORE renderView so asyncGlamour marks
			// are set and renderView renders plain text for pending msgs.
			// Without this the sync path would still run Glamour once
			// before the async result lands.
			if cmd := m.queueAsyncGlamour(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			m.renderView()
			if len(m.messages) > 0 {
				if (msg.Type == pubsub.CreatedEvent) ||
					(msg.Type == pubsub.UpdatedEvent && msg.Payload.ID == m.messages[len(m.messages)-1].ID) {
					m.viewport.GotoBottom()
				}
			}
		}
	}

	spinner, cmd := m.spinner.Update(msg)
	m.spinner = spinner
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *messagesCmp) IsAgentWorking() bool {
	return m.app.Orchestrator.IsAnyBusy(m.session.ID)
}

func formatTimeDifference(unixTime1, unixTime2 int64) string {
	diffSeconds := float64(math.Abs(float64(unixTime2 - unixTime1)))

	if diffSeconds < 60 {
		return fmt.Sprintf("%.1fs", diffSeconds)
	}

	minutes := int(diffSeconds / 60)
	seconds := int(diffSeconds) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

func (m *messagesCmp) renderView() {
	start := time.Now()
	defer func() {
		app.BumpRender(time.Since(start), len(m.messages))
	}()

	m.uiMessages = make([]uiMessage, 0)
	pos := 0
	baseStyle := styles.BaseStyle()

	if m.width == 0 {
		return
	}
	for inx, msg := range m.messages {
		switch msg.Role {
		case message.User:
			// Hide orchestrator-internal user messages — these are the prompts
			// the orchestrator generates for Observer/Assurance/QA/Synthesizer
			// (anomaly reports, validation requests, synthesis instructions).
			// They're tagged with app.InternalPromptMarker; the LLM still
			// sees the full content but the human shouldn't.
			if strings.HasPrefix(msg.Content().String(), app.InternalPromptMarker) {
				continue
			}
			if cache, ok := m.cachedContent[msg.ID]; ok && cache.width == m.width {
				m.uiMessages = append(m.uiMessages, cache.content...)
				continue
			}
			userMsg := renderUserMessage(
				msg,
				msg.ID == m.currentMsgID,
				m.width,
				pos,
			)
			m.uiMessages = append(m.uiMessages, userMsg)
			m.cachedContent[msg.ID] = cacheItem{
				width:   m.width,
				content: []uiMessage{userMsg},
			}
			pos += userMsg.height + 1 // + 1 for spacing
		case message.Assistant:
			isSummary := m.session.SummaryMessageID == msg.ID
			role := m.app.Orchestrator.GetOwner(msg.ID)
			finished := msg.IsFinished()

			// Use cache only for finished messages. Unfinished messages are
			// streaming — their content changes every token, so caching them
			// would show stale output. Cost is low because renderAssistantMessage
			// uses plain-text rendering (not Glamour) while the message is
			// unfinished. See KI-01.
			if finished {
				if cache, ok := m.cachedContent[msg.ID]; ok && cache.width == m.width && cache.role == role && cache.finished {
					m.uiMessages = append(m.uiMessages, cache.content...)
					continue
				}
			}

			// useMarkdown decouples Glamour from msg.IsFinished(). If an
			// async Glamour cmd is in flight for this msgID (asyncGlamour
			// marked), render plain text now; the async result will populate
			// the cache when it lands. Otherwise render through Glamour as
			// before. queueAsyncGlamour (called from Update before renderView
			// runs) populates asyncGlamour so the first sync render after a
			// finish-transition is already fast.
			useMarkdown := finished && !m.asyncGlamour[msg.ID]

			assistantMessages := renderAssistantMessage(
				msg,
				role,
				inx,
				m.messages,
				m.app.Messages,
				m.currentMsgID,
				isSummary,
				m.width,
				pos,
				useMarkdown,
			)
			for _, msg := range assistantMessages {
				m.uiMessages = append(m.uiMessages, msg)
				pos += msg.height + 1 // + 1 for spacing
			}
			// Cache only Glamour-rendered output. Plain-text output produced
			// while the async Glamour cmd is in flight must NOT shadow the
			// richer result that lands shortly after.
			if finished && useMarkdown {
				m.cachedContent[msg.ID] = cacheItem{
					width:    m.width,
					role:     role,
					finished: true,
					content:  assistantMessages,
				}
			}
		case message.System:
			// Coordination events injected by the orchestrator (task created,
			// agent completed work, validation passed/failed, etc). Rendered
			// as a single muted line, no header, no avatar — they're status
			// updates, not conversation turns.
			if cache, ok := m.cachedContent[msg.ID]; ok && cache.width == m.width {
				m.uiMessages = append(m.uiMessages, cache.content...)
				continue
			}
			sysMsg := renderSystemMessage(msg, m.width, pos)
			m.uiMessages = append(m.uiMessages, sysMsg)
			m.cachedContent[msg.ID] = cacheItem{
				width:   m.width,
				content: []uiMessage{sysMsg},
			}
			pos += sysMsg.height + 1
		}
	}

	messages := make([]string, 0)
	spacer := baseStyle.Width(m.width).Render("")
	for _, v := range m.uiMessages {
		messages = append(messages,
			lipgloss.JoinVertical(lipgloss.Left, v.content),
			spacer,
			spacer, // extra line for breathing room between messages
		)
	}

	m.viewport.SetContent(
		baseStyle.
			Width(m.width).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Top,
					messages...,
				),
			),
	)
}

func (m *messagesCmp) View() tea.View {
	baseStyle := styles.BaseStyle()

	if len(m.messages) == 0 {
		// Hard-truncate the splash to fit in m.height-2 rows so the editor
		// pane below it stays visible. lipgloss Height() pads up but does
		// NOT clip content that's already taller — on windows shorter than
		// the full splash (~42 rows), the overflow was pushing the editor
		// off-screen entirely. Manual line-slice gives guaranteed fit.
		splashHeight := m.height - 2
		if splashHeight < 1 {
			splashHeight = 1
		}
		splash := m.initialScreen()
		lines := strings.Split(splash, "\n")
		if len(lines) > splashHeight {
			lines = lines[:splashHeight]
		}
		content := baseStyle.
			Width(m.width).
			Render(strings.Join(lines, "\n"))

		return tea.NewView(baseStyle.
			Width(m.width).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Top,
					content,
					"",
					m.help(),
				),
			))
	}

	return tea.NewView(baseStyle.
		Width(m.width).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Top,
				m.viewport.View(),
				m.working(),
				m.help(),
			),
		))
}

func hasToolsWithoutResponse(messages []message.Message) bool {
	toolCalls := make([]message.ToolCall, 0)
	toolResults := make([]message.ToolResult, 0)
	for _, m := range messages {
		toolCalls = append(toolCalls, m.ToolCalls()...)
		toolResults = append(toolResults, m.ToolResults()...)
	}

	for _, v := range toolCalls {
		found := false
		for _, r := range toolResults {
			if v.ID == r.ToolCallID {
				found = true
				break
			}
		}
		if !found && v.Finished {
			return true
		}
	}
	return false
}

func hasUnfinishedToolCalls(messages []message.Message) bool {
	toolCalls := make([]message.ToolCall, 0)
	for _, m := range messages {
		toolCalls = append(toolCalls, m.ToolCalls()...)
	}
	for _, v := range toolCalls {
		if !v.Finished {
			return true
		}
	}
	return false
}

func (m *messagesCmp) working() string {
	text := ""
	if m.IsAgentWorking() && len(m.messages) > 0 {
		t := theme.CurrentTheme()
		baseStyle := styles.BaseStyle()

		task := "Thinking..."
		lastMessage := m.messages[len(m.messages)-1]
		if hasToolsWithoutResponse(m.messages) {
			task = "Waiting for tool response..."
		} else if hasUnfinishedToolCalls(m.messages) {
			task = "Building tool call..."
		} else if !lastMessage.IsFinished() {
			task = "Generating..."
		}
		if task != "" {
			text += baseStyle.
				Width(m.width).
				Foreground(t.Primary()).
				Bold(true).
				Render(fmt.Sprintf("%s %s ", m.spinner.View(), task))
		}
	}
	return text
}

func (m *messagesCmp) help() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()

	text := ""

	if m.app.Orchestrator.IsAnyBusy("") {
		text += lipgloss.JoinHorizontal(
			lipgloss.Left,
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render("press "),
			baseStyle.Foreground(t.Text()).Bold(true).Render("esc"),
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render(" to exit cancel"),
		)
	} else {
		text += lipgloss.JoinHorizontal(
			lipgloss.Left,
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render("press "),
			baseStyle.Foreground(t.Text()).Bold(true).Render("enter"),
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render(" to send the message,"),
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render(" write"),
			baseStyle.Foreground(t.Text()).Bold(true).Render(" \\"),
			baseStyle.Foreground(t.TextMuted()).Bold(true).Render(" and enter to add a new line"),
		)
	}
	return baseStyle.
		Width(m.width).
		Render(text)
}

func (m *messagesCmp) initialScreen() string {
	baseStyle := styles.BaseStyle()

	return baseStyle.Width(m.width).Render(
		lipgloss.JoinVertical(
			lipgloss.Top,
			actBanner(m.width),
			"",
			welcomeGuide(m.width),
		),
	)
}

func (m *messagesCmp) rerender() {
	for _, msg := range m.messages {
		delete(m.cachedContent, msg.ID)
	}
	m.renderView()
}

func (m *messagesCmp) SetSize(width, height int) tea.Cmd {
	if m.width == width && m.height == height {
		return nil
	}
	m.width = width
	m.height = height
	m.viewport.SetWidth(width)
	m.viewport.SetHeight(height - 2)
	m.attachments.SetWidth(width + 40)
	m.attachments.SetHeight(3)
	// Width change invalidates every cached render (word-wrap is
	// width-dependent). Clear in-flight async marks so queueAsyncGlamour
	// re-dispatches at the new width.
	m.asyncGlamour = make(map[string]bool)
	m.rerender()
	return m.queueAsyncGlamour()
}

func (m *messagesCmp) GetSize() (int, int) {
	return m.width, m.height
}

func (m *messagesCmp) SetSession(session session.Session) tea.Cmd {
	if m.session.ID == session.ID {
		return nil
	}
	m.session = session
	messages, err := m.app.Messages.List(context.Background(), session.ID)
	if err != nil {
		return util.ReportError(err)
	}
	m.messages = messages
	if len(m.messages) > 0 {
		m.currentMsgID = m.messages[len(m.messages)-1].ID
	}
	delete(m.cachedContent, m.currentMsgID)
	// Queue async Glamour for any finished assistant msgs loaded from the
	// DB. Without this a session with N finished assistant msgs would fire
	// N synchronous Glamour renders inside renderView and hang the Update
	// loop for tens of seconds on session switch.
	cmd := m.queueAsyncGlamour()
	m.renderView()
	m.viewport.GotoBottom()
	return cmd
}

// queueAsyncGlamour scans m.messages for finished assistant msgs that are
// neither cached nor already being rendered, marks them as pending, and
// returns a Cmd that dispatches one goroutine per msg to run Glamour off
// the Update loop. Each goroutine posts markdownRenderedMsg back to Update
// where the rendered uiMessages get installed into cachedContent.
// Safe to call multiple times — already-pending msgs are skipped.
func (m *messagesCmp) queueAsyncGlamour() tea.Cmd {
	var cmds []tea.Cmd
	for i, msg := range m.messages {
		if msg.Role != message.Assistant || !msg.IsFinished() {
			continue
		}
		if _, cached := m.cachedContent[msg.ID]; cached {
			continue
		}
		if m.asyncGlamour[msg.ID] {
			continue
		}
		m.asyncGlamour[msg.ID] = true
		cmds = append(cmds, m.asyncGlamourCmd(msg, i))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// asyncGlamourCmd returns a Cmd that renders the given msg through Glamour
// in a goroutine. All state read by the goroutine is snapshotted at cmd
// construction so mutation of m.messages on the Update loop during the
// render doesn't race. The result is posted back as markdownRenderedMsg;
// the Update handler reconciles stale results (session switch, resize) by
// dropping them.
func (m *messagesCmp) asyncGlamourCmd(msg message.Message, msgIndex int) tea.Cmd {
	width := m.width
	role := m.app.Orchestrator.GetOwner(msg.ID)
	sessionID := m.session.ID
	allMessages := append([]message.Message(nil), m.messages...)
	currentMsgID := m.currentMsgID
	messagesService := m.app.Messages
	isSummary := m.session.SummaryMessageID == msg.ID
	return func() tea.Msg {
		rendered := renderAssistantMessage(
			msg,
			role,
			msgIndex,
			allMessages,
			messagesService,
			currentMsgID,
			isSummary,
			width,
			0, // position recomputed by renderView; cacheItem.content uses it as a no-op
			true,
		)
		return markdownRenderedMsg{
			sessionID: sessionID,
			msgID:     msg.ID,
			width:     width,
			role:      role,
			rendered:  rendered,
		}
	}
}

func (m *messagesCmp) BindingKeys() []key.Binding {
	return []key.Binding{
		m.viewport.KeyMap.PageDown,
		m.viewport.KeyMap.PageUp,
		m.viewport.KeyMap.HalfPageUp,
		m.viewport.KeyMap.HalfPageDown,
	}
}

func NewMessagesCmp(app *app.App) tea.Model {
	s := spinner.New()
	s.Spinner = spinner.Pulse
	vp := viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))
	attachmets := viewport.New(viewport.WithWidth(0), viewport.WithHeight(0))
	vp.KeyMap.PageUp = messageKeys.PageUp
	vp.KeyMap.PageDown = messageKeys.PageDown
	vp.KeyMap.HalfPageUp = messageKeys.HalfPageUp
	vp.KeyMap.HalfPageDown = messageKeys.HalfPageDown
	return &messagesCmp{
		app:           app,
		cachedContent: make(map[string]cacheItem),
		asyncGlamour:  make(map[string]bool),
		viewport:      vp,
		spinner:       s,
		attachments:   attachmets,
	}
}
