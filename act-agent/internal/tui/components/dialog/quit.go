package dialog

import (
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

// quitDialogCmp is a minimal custom Yes/No dialog that avoids huh's
// internal lipgloss.NewStyle() (no-background) button alignment, which
// left a terminal-black rectangle next to the focused button.
type quitDialogCmp struct {
	// cursor: false = No focused (default/safe), true = Yes focused
	cursor bool
}

func (q *quitDialogCmp) Init() tea.Cmd { return nil }

func (q *quitDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "left", "h", "tab":
			q.cursor = !q.cursor
		case "right", "l":
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
	b := styles.BaseStyle()
	const innerWidth = 36

	title := b.Width(innerWidth).Foreground(t.Text()).Bold(true).Render("Are you sure you want to quit?")

	yesStyle := b.Foreground(t.TextMuted()).Padding(0, 1)
	noStyle := b.Foreground(t.TextMuted()).Padding(0, 1)
	if q.cursor {
		yesStyle = b.Background(t.Primary()).Foreground(t.Background()).Padding(0, 1)
	} else {
		noStyle = b.Background(t.Primary()).Foreground(t.Background()).Padding(0, 1)
	}

	buttons := b.Width(innerWidth).Render(lipgloss.JoinHorizontal(lipgloss.Top,
		yesStyle.Render("Yes"),
		b.Render("  "),
		noStyle.Render("No"),
	))

	inner := lipgloss.JoinVertical(lipgloss.Left,
		title,
		b.Width(innerWidth).Render(""),
		buttons,
	)

	dialog := b.
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.BorderFocused()).
		Render(inner)

	return tea.NewView(dialog)
}

func (q *quitDialogCmp) BindingKeys() []key.Binding { return nil }

func NewQuitCmp() QuitDialog {
	return &quitDialogCmp{}
}
