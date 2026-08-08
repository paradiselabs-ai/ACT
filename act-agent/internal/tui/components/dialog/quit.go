package dialog

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/layout"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/util"
)

type CloseQuitMsg struct{}

type QuitDialog interface {
	tea.Model
	layout.Bindings
}

// quitDialogCmp is a minimal custom Yes/No dialog that strictly adheres
// to STYLING_GUIDE.md to prevent black/grey strip anomalies in overlays.
type quitDialogCmp struct {
	// cursor: true = Yes focused (default), false = No focused
	cursor bool
}

func (q *quitDialogCmp) Init() tea.Cmd { return nil }

func (q *quitDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "left", "h", "right", "l", "tab":
			q.cursor = !q.cursor
		case "y", "Y":
			return q, tea.Quit
		case "n", "N", "esc":
			return q, util.CmdHandler(CloseQuitMsg{})
		case "enter":
			if q.cursor {
				return q, tea.Quit
			}
			return q, util.CmdHandler(CloseQuitMsg{})
		}
	}
	return q, nil
}

func (q *quitDialogCmp) View() tea.View {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle().Background(t.Background())
	padStyle := baseStyle.Background(t.Background())
	const innerWidth = 36

	title := baseStyle.Foreground(t.Text()).Bold(true).Render("Are you sure you want to quit?")

	yesStyle := baseStyle.Foreground(t.TextMuted()).Padding(0, 1)
	noStyle := baseStyle.Foreground(t.TextMuted()).Padding(0, 1)

	if q.cursor {
		yesStyle = baseStyle.Background(t.Primary()).Foreground(t.Background()).Bold(true).Padding(0, 1)
	} else {
		noStyle = baseStyle.Background(t.Primary()).Foreground(t.Background()).Bold(true).Padding(0, 1)
	}

	buttons := yesStyle.Render("Yes") + padStyle.Render("  ") + noStyle.Render("No")

	padLine := func(line string, width int) string {
		w := lipgloss.Width(line)
		if w < width {
			return line + padStyle.Render(strings.Repeat(" ", width-w))
		}
		return line
	}

	maxW := innerWidth
	if w := lipgloss.Width(title); w > maxW {
		maxW = w
	}
	if w := lipgloss.Width(buttons); w > maxW {
		maxW = w
	}

	var bodyLines []string
	bodyLines = append(bodyLines, padLine(title, maxW))
	bodyLines = append(bodyLines, padLine("", maxW))
	bodyLines = append(bodyLines, padLine(buttons, maxW))

	body := strings.Join(bodyLines, "\n")

	boxStyle := baseStyle.
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.BorderFocused())

	rendered := boxStyle.Render(body)
	return tea.NewView(styles.ForceBackgroundOnAllLines(rendered, t.Background()))
}

func (q *quitDialogCmp) BindingKeys() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("y", "Y"), key.WithHelp("y", "confirm quit")),
		key.NewBinding(key.WithKeys("n", "N", "esc"), key.WithHelp("n / esc", "cancel")),
	}
}

func NewQuitCmp() QuitDialog {
	return &quitDialogCmp{cursor: true}
}
