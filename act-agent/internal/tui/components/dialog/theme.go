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

// ThemeChangedMsg is sent when the theme is changed
type ThemeChangedMsg struct {
	ThemeName string
}

// CloseThemeDialogMsg is sent when the theme dialog is closed
type CloseThemeDialogMsg struct{}

// ThemeDialog interface for the theme switching dialog
type ThemeDialog interface {
	tea.Model
	layout.Bindings
}

type themeDialogCmp struct {
	form     *huh.Form
	selected string
}

func (d *themeDialogCmp) buildForm() tea.Cmd {
	themes := theme.AvailableThemes()
	current := theme.CurrentThemeName()
	opts := make([]huh.Option[string], len(themes))
	for i, name := range themes {
		label := name
		if name == current {
			label = "▶ " + label
		}
		opts[i] = huh.NewOption(label, name)
	}
	d.selected = current
	d.form = huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select Theme").
			Options(opts...).
			Value(&d.selected),
	)).WithShowHelp(false).WithTheme(actHuhTheme())
	return d.form.Init()
}

func (d *themeDialogCmp) Init() tea.Cmd {
	return d.buildForm()
}

func (d *themeDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if d.form == nil {
		return d, nil
	}
	m, cmd := d.form.Update(msg)
	if f, ok := m.(*huh.Form); ok {
		d.form = f
	}
	switch d.form.State {
	case huh.StateCompleted:
		if err := theme.SetTheme(d.selected); err != nil {
			return d, util.ReportError(err)
		}
		styles.InvalidateMarkdownRendererCache()
		return d, util.CmdHandler(ThemeChangedMsg{ThemeName: d.selected})
	case huh.StateAborted:
		return d, util.CmdHandler(CloseThemeDialogMsg{})
	}
	return d, cmd
}

func (d *themeDialogCmp) View() tea.View {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()
	if d.form == nil {
		return tea.NewView(baseStyle.Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderBackground(t.Background()).
			BorderForeground(t.BorderFocused()).
			Width(40).
			Render("Loading themes..."))
	}
	return tea.NewView(baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.BorderFocused()).
		Render(d.form.View()))
}

func (d *themeDialogCmp) BindingKeys() []key.Binding {
	if d.form == nil {
		return nil
	}
	return d.form.KeyBinds()
}

// NewThemeDialogCmp creates a new theme switching dialog
func NewThemeDialogCmp() ThemeDialog {
	return &themeDialogCmp{}
}
