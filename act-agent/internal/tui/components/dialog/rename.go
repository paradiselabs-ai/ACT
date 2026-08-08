package dialog

import (
	"errors"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/layout"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/util"
)

// RenameCompletedMsg is sent when a rename is successfully completed with a non-empty string.
type RenameCompletedMsg struct {
	NewTitle string
}

// CloseRenameDialogMsg is sent when the rename dialog is closed/cancelled.
type CloseRenameDialogMsg struct{}

// RenameDialog interface for the session renaming dialog
type RenameDialog interface {
	tea.Model
	layout.Bindings
	SetTitle(title string)
}

type renameDialogCmp struct {
	form  *huh.Form
	title string
}

func validateTitle(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("Title cannot be blank")
	}
	return nil
}

func (r *renameDialogCmp) SetTitle(title string) {
	r.title = title
	r.form = huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("Rename Session").
			Validate(validateTitle).
			Value(&r.title),
	)).WithShowHelp(false).WithTheme(actHuhTheme()).WithWidth(40)
	_ = r.form.Init()
}

func (r *renameDialogCmp) Init() tea.Cmd {
	return nil
}

func (r *renameDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if r.form == nil {
		return r, nil
	}
	m, cmd := r.form.Update(msg)
	if f, ok := m.(*huh.Form); ok {
		r.form = f
	}
	switch r.form.State {
	case huh.StateCompleted:
		return r, util.CmdHandler(RenameCompletedMsg{NewTitle: strings.TrimSpace(r.title)})
	case huh.StateAborted:
		return r, util.CmdHandler(CloseRenameDialogMsg{})
	}
	return r, cmd
}

func (r *renameDialogCmp) View() tea.View {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()
	if r.form == nil {
		return tea.NewView(baseStyle.Render(""))
	}
	return tea.NewView(baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.BorderFocused()).
		Width(46).
		Render(r.form.View()))
}

func (r *renameDialogCmp) BindingKeys() []key.Binding {
	if r.form == nil {
		return nil
	}
	return r.form.KeyBinds()
}

// NewRenameCmp creates a new session renaming dialog
func NewRenameCmp(initialTitle string) RenameDialog {
	r := &renameDialogCmp{}
	r.SetTitle(initialTitle)
	return r
}
