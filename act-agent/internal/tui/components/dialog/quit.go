package dialog

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
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

type quitDialogCmp struct {
	form      *huh.Form
	confirmed bool
}

func newQuitForm(confirmed *bool) *huh.Form {
	return huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title("Are you sure you want to quit?").
			Affirmative("Yes").
			Negative("No").
			Value(confirmed),
	)).WithShowHelp(false)
}

func (q *quitDialogCmp) Init() tea.Cmd {
	return q.form.Init()
}

func (q *quitDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Reset the form if the dialog is being reopened after a previous use.
	if q.form.State != huh.StateNormal {
		q.confirmed = false
		q.form = newQuitForm(&q.confirmed)
		return q, q.form.Init()
	}
	m, cmd := q.form.Update(msg)
	if f, ok := m.(*huh.Form); ok {
		q.form = f
	}
	switch q.form.State {
	case huh.StateCompleted:
		if q.confirmed {
			return q, tea.Quit
		}
		return q, util.CmdHandler(CloseQuitMsg{})
	case huh.StateAborted:
		return q, util.CmdHandler(CloseQuitMsg{})
	}
	return q, cmd
}

func (q *quitDialogCmp) View() tea.View {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()
	return tea.NewView(baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.TextMuted()).
		Render(q.form.View()))
}

func (q *quitDialogCmp) BindingKeys() []key.Binding {
	return q.form.KeyBinds()
}

func NewQuitCmp() QuitDialog {
	d := &quitDialogCmp{}
	d.form = newQuitForm(&d.confirmed)
	return d
}
