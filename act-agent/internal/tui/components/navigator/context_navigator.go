package navigator

import (
	"strings"

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
		Background(t.Background()).
		Padding(1, 1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(t.BorderDim()).
		BorderBackground(t.Background())

	// Header
	b := styles.BaseStyle()
	headerStyle := styles.Bold().Foreground(t.Primary()).Background(t.Background())
	mutedStyle := b.Foreground(t.TextMuted())
	textStyle := b.Foreground(t.Text())

	var lines []string
	lines = append(lines, headerStyle.Render("Context"), "")

	// Active agent
	lines = append(lines, mutedStyle.Render("Active"))
	if n.app.Orchestrator != nil {
		speaker := n.app.Orchestrator.CurrentSpeaker()
		if speaker != "" {
			lines = append(lines, textStyle.Render("• "+speaker))
		} else {
			lines = append(lines, textStyle.Render("• None"))
		}
	}
	lines = append(lines, "")

	// Registered agents
	lines = append(lines, mutedStyle.Render("Agents"))
	lines = append(lines,
		textStyle.Render("• planner"),
		textStyle.Render("• observer"),
		textStyle.Render("• assurance"),
		textStyle.Render("• qa"),
	)

	content := strings.Join(lines, "\n")
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
