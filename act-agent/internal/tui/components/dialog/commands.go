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

// Command represents a command that can be executed
type Command struct {
	ID          string
	Title       string
	Description string
	Handler     func(cmd Command) tea.Cmd
}

// CommandSelectedMsg is sent when a command is selected
type CommandSelectedMsg struct {
	Command Command
}

// CloseCommandDialogMsg is sent when the command dialog is closed
type CloseCommandDialogMsg struct{}

// CommandDialog interface for the command selection dialog
type CommandDialog interface {
	tea.Model
	layout.Bindings
	SetCommands(commands []Command)
}

type commandDialogCmp struct {
	form        *huh.Form
	commands    []Command
	selectedIdx int
}

func (c *commandDialogCmp) buildForm() {
	opts := make([]huh.Option[int], len(c.commands))
	for i, cmd := range c.commands {
		label := cmd.Title
		if cmd.Description != "" {
			label = cmd.Title + " — " + cmd.Description
		}
		opts[i] = huh.NewOption(label, i)
	}
	c.form = huh.NewForm(huh.NewGroup(
		huh.NewSelect[int]().
			Title("Commands").
			Filtering(true).
			Height(12).
			Options(opts...).
			Value(&c.selectedIdx),
	)).WithShowHelp(false)
	_ = c.form.Init()
}

func (c *commandDialogCmp) Init() tea.Cmd {
	return nil
}

func (c *commandDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if c.form == nil {
		return c, nil
	}
	m, cmd := c.form.Update(msg)
	if f, ok := m.(*huh.Form); ok {
		c.form = f
	}
	switch c.form.State {
	case huh.StateCompleted:
		if c.selectedIdx >= 0 && c.selectedIdx < len(c.commands) {
			return c, util.CmdHandler(CommandSelectedMsg{Command: c.commands[c.selectedIdx]})
		}
		return c, util.CmdHandler(CloseCommandDialogMsg{})
	case huh.StateAborted:
		return c, util.CmdHandler(CloseCommandDialogMsg{})
	}
	return c, cmd
}

func (c *commandDialogCmp) View() tea.View {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()
	if c.form == nil || len(c.commands) == 0 {
		return tea.NewView(baseStyle.Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderBackground(t.Background()).
			BorderForeground(t.TextMuted()).
			Width(40).
			Render("No commands available"))
	}
	return tea.NewView(baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.TextMuted()).
		Render(c.form.View()))
}

func (c *commandDialogCmp) BindingKeys() []key.Binding {
	if c.form == nil {
		return nil
	}
	return c.form.KeyBinds()
}

func (c *commandDialogCmp) SetCommands(commands []Command) {
	c.commands = commands
	c.selectedIdx = 0
	c.buildForm()
}

// NewCommandDialogCmp creates a new command selection dialog
func NewCommandDialogCmp() CommandDialog {
	return &commandDialogCmp{}
}
