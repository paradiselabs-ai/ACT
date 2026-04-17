package navigator

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/app"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/layout"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
)

// ContextNavigator shows session info and registered agents
type ContextNavigator struct {
	app    *app.App
	width  int
	height int
}

// NewContextNavigator creates a new context navigator component
func NewContextNavigator(app *app.App) *ContextNavigator {
	return &ContextNavigator{
		app: app,
	}
}

// Init implements tea.Model
func (n *ContextNavigator) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (n *ContextNavigator) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return n, nil
}

// View implements tea.Model
func (n *ContextNavigator) View() tea.View {
	t := theme.CurrentTheme()
	baseStyle := lipgloss.NewStyle().
		Width(n.width).
		Height(n.height).
		Padding(1, 1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(t.BorderDim())

	// Header
	headerStyle := styles.Bold().Foreground(t.Primary())
	content := headerStyle.Render("Context") + "\n\n"

	// Active agent
	content += lipgloss.NewStyle().Foreground(t.TextMuted()).Render("Active") + "\n"
	if n.app.Orchestrator != nil {
		speaker := n.app.Orchestrator.CurrentSpeaker()
		if speaker != "" {
			content += fmt.Sprintf("• %s\n", speaker)
		} else {
			content += "• None\n"
		}
	}
	content += "\n"

	// Registered agents (placeholder for now)
	content += lipgloss.NewStyle().Foreground(t.TextMuted()).Render("Agents") + "\n"
	content += "• planner\n"
	content += "• observer\n"
	content += "• assurance\n"
	content += "• qa\n"

	return tea.NewView(baseStyle.Render(content))
}

// SetSize implements layout.Sizeable
func (n *ContextNavigator) SetSize(width, height int) tea.Cmd {
	n.width = width
	n.height = height
	return nil
}

// GetSize implements layout.Sizeable
func (n *ContextNavigator) GetSize() (int, int) {
	return n.width, n.height
}

// BindingKeys implements layout.Bindings
func (n *ContextNavigator) BindingKeys() []key.Binding {
	return nil
}

// Ensure interfaces are implemented
var (
	_ tea.Model       = (*ContextNavigator)(nil)
	_ layout.Sizeable = (*ContextNavigator)(nil)
	_ layout.Bindings = (*ContextNavigator)(nil)
)
