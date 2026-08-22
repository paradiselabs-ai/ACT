package chat

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/app"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/message"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/session"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/components/dialog"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/layout"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/util"
)

type editorCmp struct {
	width       int
	height      int
	app         *app.App
	session     session.Session
	textarea    textarea.Model
	attachments []message.Attachment
	deleteMode  bool
}

type EditorKeyMaps struct {
	Send       key.Binding
	OpenEditor key.Binding
}

type DeleteAttachmentKeyMaps struct {
	AttachmentDeleteMode key.Binding
	Escape               key.Binding
	DeleteAllAttachments key.Binding
}

var editorMaps = EditorKeyMaps{
	Send: key.NewBinding(
		key.WithKeys("enter", "ctrl+s"),
		key.WithHelp("enter", "send message"),
	),
	// alt+e, NOT ctrl+e: ctrl+e is bound at the appModel layer (tui.go) to
	// "rename session" and its switch runs first — binding the editor there
	// made $EDITOR unreachable from the TUI. Keep this in sync with the
	// duplicate-binding check when adding more.
	OpenEditor: key.NewBinding(
		key.WithKeys("alt+e"),
		key.WithHelp("alt+e", "open editor"),
	),
}

var DeleteKeyMaps = DeleteAttachmentKeyMaps{
	AttachmentDeleteMode: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r+{i}", "delete attachment at index i"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel delete mode"),
	),
	DeleteAllAttachments: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("ctrl+r+r", "delete all attchments"),
	),
}

const (
	maxAttachments = 5
)

// splitEditorCommand splits an $EDITOR value that may carry arguments
// (e.g. "code -w", "nvim -u NONE") into program + args. exec.Command cannot
// take a multi-word string as a single argv entry, so the old code failed to
// launch any editor whose value contained spaces (audit M7). Quotes are
// honored for paths with spaces ("C:\Program Files\...\vim.exe").
func splitEditorCommand(editor string) []string {
	var parts []string
	var cur strings.Builder
	inQuote := false
	for _, r := range editor {
		switch {
		case r == '"':
			inQuote = !inQuote
		case unicode.IsSpace(r) && !inQuote:
			if cur.Len() > 0 {
				parts = append(parts, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

func (m *editorCmp) openEditor() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nvim"
	}

	tmpfile, err := os.CreateTemp("", "msg_*.md")
	if err != nil {
		return util.ReportError(err)
	}
	tmpfile.Close()

	// Remove the temp file no matter how the editor session ends. The old
	// code only removed it after reading non-empty content — aborting with
	// an empty buffer or killing the editor leaked msg_*.md files in %TEMP%
	// forever (audit M7).
	removeTmp := func() {
		if rmErr := os.Remove(tmpfile.Name()); rmErr != nil && !os.IsNotExist(rmErr) {
			logging.Warn("editor temp file cleanup failed", "path", tmpfile.Name(), "error", rmErr)
		}
	}

	editorArgs := splitEditorCommand(editor)
	editorArgs = append(editorArgs, tmpfile.Name())
	c := exec.Command(editorArgs[0], editorArgs[1:]...) //nolint:gosec
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return tea.ExecProcess(c, func(err error) tea.Msg {
		defer removeTmp()
		if err != nil {
			return util.ReportError(err)
		}
		content, readErr := os.ReadFile(tmpfile.Name())
		if readErr != nil {
			return util.ReportError(readErr)
		}
		if len(content) == 0 {
			return util.ReportWarn("Message is empty")
		}
		attachments := m.attachments
		m.attachments = nil
		return SendMsg{
			Text:        string(content),
			Attachments: attachments,
		}
	})
}

func (m *editorCmp) Init() tea.Cmd {
	return textarea.Blink
}

func (m *editorCmp) send() tea.Cmd {
	if m.app.Orchestrator.IsAnyBusy(m.session.ID) {
		return util.ReportWarn("Agent is working, please wait...")
	}

	value := m.textarea.Value()
	m.textarea.Reset()
	attachments := m.attachments

	m.attachments = nil
	if value == "" {
		return nil
	}
	return tea.Batch(
		util.CmdHandler(SendMsg{
			Text:        value,
			Attachments: attachments,
		}),
	)
}

func (m *editorCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case dialog.ThemeChangedMsg:
		m.textarea = CreateTextArea(&m.textarea)
	case dialog.CompletionSelectedMsg:
		existingValue := m.textarea.Value()
		modifiedValue := strings.Replace(existingValue, msg.SearchString, msg.CompletionValue, 1)

		m.textarea.SetValue(modifiedValue)
		return m, nil
	case SessionSelectedMsg:
		if msg.ID != m.session.ID {
			m.session = msg
		}
		return m, nil
	case dialog.AttachmentAddedMsg:
		if len(m.attachments) >= maxAttachments {
			logging.ErrorPersist(fmt.Sprintf("cannot add more than %d images", maxAttachments))
			return m, cmd
		}
		m.attachments = append(m.attachments, msg.Attachment)
	case tea.KeyPressMsg:
		if key.Matches(msg, DeleteKeyMaps.AttachmentDeleteMode) {
			m.deleteMode = true
			return m, nil
		}
		if key.Matches(msg, DeleteKeyMaps.DeleteAllAttachments) && m.deleteMode {
			m.deleteMode = false
			m.attachments = nil
			return m, nil
		}
		if m.deleteMode && msg.Key().Code != 0 && unicode.IsDigit(msg.Key().Code) {
			num := int(msg.Key().Code - '0')
			m.deleteMode = false
			if num < 10 && len(m.attachments) > num {
				if num == 0 {
					m.attachments = m.attachments[num+1:]
				} else {
					m.attachments = slices.Delete(m.attachments, num, num+1)
				}
				return m, nil
			}
		}
		if key.Matches(msg, messageKeys.PageUp) || key.Matches(msg, messageKeys.PageDown) ||
			key.Matches(msg, messageKeys.HalfPageUp) || key.Matches(msg, messageKeys.HalfPageDown) {
			return m, nil
		}
		if key.Matches(msg, editorMaps.OpenEditor) {
			if m.app.Orchestrator.IsAnyBusy(m.session.ID) {
				return m, util.ReportWarn("Agent is working, please wait...")
			}
			return m, m.openEditor()
		}
		if key.Matches(msg, DeleteKeyMaps.Escape) {
			m.deleteMode = false
			return m, nil
		}
		// Hanlde Enter key
		if m.textarea.Focused() && key.Matches(msg, editorMaps.Send) {
			value := m.textarea.Value()
			if len(value) > 0 && value[len(value)-1] == '\\' {
				// If the last character is a backslash, remove it and add a newline
				m.textarea.SetValue(value[:len(value)-1] + "\n")
				return m, nil
			} else {
				// Otherwise, send the message
				return m, m.send()
			}
		}

	}
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m *editorCmp) View() tea.View {
	t := theme.CurrentTheme()
	bgColor := t.Background()
	bgSeq := lipgloss.NewStyle().Background(bgColor).Render("")

	baseStyle := lipgloss.NewStyle().
		Width(m.width).
		Background(bgColor)

	style := lipgloss.NewStyle().
		Background(bgColor).
		Padding(0, 0, 0, 1).
		Bold(true).
		Foreground(t.Primary())

	var content string
	if len(m.attachments) == 0 {
		content = lipgloss.JoinHorizontal(lipgloss.Top, style.Render(">"), m.textarea.View())
	} else {
		m.textarea.SetHeight(m.height - 1)
		content = lipgloss.JoinVertical(lipgloss.Top,
			m.attachmentsContent(),
			lipgloss.JoinHorizontal(lipgloss.Top, style.Render(">"), m.textarea.View()),
		)
	}

	rendered := baseStyle.Render(content)
	if bgSeq != "" {
		rendered = styles.RepaintBackground(rendered, bgSeq)
	}

	return tea.NewView(rendered)
}

func (m *editorCmp) SetSize(width, height int) tea.Cmd {
	m.width = width
	m.height = height
	m.textarea.SetWidth(width - 3) // account for the prompt and padding right
	m.textarea.SetHeight(1)
	return nil
}

func (m *editorCmp) GetSize() (int, int) {
	h := m.textarea.Height()
	if len(m.attachments) > 0 {
		h++
	}
	return m.textarea.Width(), max(1, h)
}

func (m *editorCmp) attachmentsContent() string {
	var styledAttachments []string
	t := theme.CurrentTheme()
	attachmentStyles := styles.BaseStyle().
		MarginLeft(1).
		Background(t.TextMuted()).
		Foreground(t.Text())
	for i, attachment := range m.attachments {
		var filename string
		if len(attachment.FileName) > 10 {
			filename = fmt.Sprintf(" %s %s...", styles.DocumentIcon, attachment.FileName[0:7])
		} else {
			filename = fmt.Sprintf(" %s %s", styles.DocumentIcon, attachment.FileName)
		}
		if m.deleteMode {
			filename = fmt.Sprintf("%d%s", i, filename)
		}
		styledAttachments = append(styledAttachments, attachmentStyles.Render(filename))
	}
	content := lipgloss.JoinHorizontal(lipgloss.Left, styledAttachments...)
	return content
}

func (m *editorCmp) BindingKeys() []key.Binding {
	bindings := []key.Binding{}
	bindings = append(bindings, layout.KeyMapToSlice(editorMaps)...)
	bindings = append(bindings, layout.KeyMapToSlice(DeleteKeyMaps)...)
	return bindings
}

func CreateTextArea(existing *textarea.Model) textarea.Model {
	t := theme.CurrentTheme()
	bgColor := t.Background()
	textColor := t.Text()
	textMutedColor := t.TextMuted()

	ta := textarea.New()
	baseStyle := lipgloss.NewStyle().Background(bgColor)
	ta.SetStyles(textarea.Styles{
		Blurred: textarea.StyleState{
			Base:        baseStyle.Foreground(textColor),
			CursorLine:  baseStyle,
			Placeholder: baseStyle.Foreground(textMutedColor),
			Text:        baseStyle.Foreground(textColor),
			Prompt:      baseStyle.Foreground(textColor),
		},
		Focused: textarea.StyleState{
			Base:        baseStyle.Foreground(textColor),
			CursorLine:  baseStyle,
			Placeholder: baseStyle.Foreground(textMutedColor),
			Text:        baseStyle.Foreground(textColor),
			Prompt:      baseStyle.Foreground(textColor),
		},
	})

	ta.Prompt = " "
	ta.Placeholder = "try /plan, /run, or @planner <task>"
	ta.ShowLineNumbers = false
	ta.CharLimit = -1

	if existing != nil {
		ta.SetValue(existing.Value())
		ta.SetWidth(existing.Width())
		ta.SetHeight(existing.Height())
	}

	ta.Focus()
	return ta
}

func NewEditorCmp(app *app.App) tea.Model {
	ta := CreateTextArea(nil)
	return &editorCmp{
		app:      app,
		textarea: ta,
	}
}
