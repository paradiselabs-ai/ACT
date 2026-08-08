package dialog

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/util"
)

// ShowMultiArgumentsDialogMsg is a message that is sent to show the multi-arguments dialog.
type ShowMultiArgumentsDialogMsg struct {
	CommandID string
	Content   string
	ArgNames  []string
}

// CloseMultiArgumentsDialogMsg is a message that is sent when the multi-arguments dialog is closed.
type CloseMultiArgumentsDialogMsg struct {
	Submit    bool
	CommandID string
	Content   string
	Args      map[string]string
}

// MultiArgumentsDialogCmp is a component that asks the user for multiple command arguments.
type MultiArgumentsDialogCmp struct {
	width, height int
	form          *huh.Form
	values        []string
	commandID     string
	content       string
	argNames      []string
}

// NewMultiArgumentsDialogCmp creates a new MultiArgumentsDialogCmp.
func NewMultiArgumentsDialogCmp(commandID, content string, argNames []string) MultiArgumentsDialogCmp {
	values := make([]string, len(argNames))
	fields := make([]huh.Field, len(argNames))
	for i, name := range argNames {
		fields[i] = huh.NewInput().
			Title(name).
			Value(&values[i])
	}
	form := huh.NewForm(huh.NewGroup(fields...)).WithShowHelp(false).WithTheme(actHuhTheme())
	return MultiArgumentsDialogCmp{
		form:      form,
		values:    values,
		commandID: commandID,
		content:   content,
		argNames:  argNames,
	}
}

// Init implements tea.Model.
func (m MultiArgumentsDialogCmp) Init() tea.Cmd {
	return m.form.Init()
}

// Update implements tea.Model.
func (m MultiArgumentsDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	_, cmd := m.form.Update(msg)
	switch m.form.State {
	case huh.StateCompleted:
		args := make(map[string]string, len(m.argNames))
		for i, name := range m.argNames {
			args[name] = m.values[i]
		}
		return m, util.CmdHandler(CloseMultiArgumentsDialogMsg{
			Submit:    true,
			CommandID: m.commandID,
			Content:   m.content,
			Args:      args,
		})
	case huh.StateAborted:
		return m, util.CmdHandler(CloseMultiArgumentsDialogMsg{
			Submit:    false,
			CommandID: m.commandID,
			Content:   m.content,
			Args:      nil,
		})
	}
	return m, cmd
}

// View implements tea.Model.
func (m MultiArgumentsDialogCmp) View() tea.View {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()
	return tea.NewView(baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.BorderFocused()).
		Render(m.form.View()))
}

// SetSize sets the size of the component.
func (m *MultiArgumentsDialogCmp) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// BindingKeys implements layout.Bindings.
func (m MultiArgumentsDialogCmp) BindingKeys() []key.Binding {
	return m.form.KeyBinds()
}


