package navigator

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/app"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
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

func phaseString(p app.Phase) string {
	switch p {
	case app.PhaseIdle:
		return "Idle"
	case app.PhaseIntake:
		return "Intake questions"
	case app.PhaseBrownfieldAnalysis:
		return "Analyzing codebase"
	case app.PhasePlanning:
		return "Planning tasks"
	case app.PhaseExecuting:
		return "Executing"
	case app.PhaseValidating:
		return "Validating changes"
	case app.PhaseAwaitingInput:
		return "Awaiting input"
	default:
		return "Idle"
	}
}

func (n *ContextNavigator) agentStatusLine(role string, phase app.Phase) string {
	t := theme.CurrentTheme()

	c := styles.AgentColor(role)

	isOnline := n.app.Agents[role] != nil
	if role == "qa" {
		isOnline = n.app.Agents["qa_synthesizer"] != nil
	}

	if !isOnline {
		redDot := lipgloss.NewStyle().Foreground(t.Error()).Render("●")
		mutedName := lipgloss.NewStyle().Foreground(t.TextMuted()).Render(role)
		return fmt.Sprintf(" %s %s (offline)", redDot, mutedName)
	}

	state := app.AgentStateIdle
	if n.app.Orchestrator != nil {
		actualRole := role
		if role == "qa" {
			actualRole = "qa_synthesizer"
		}
		state = n.app.Orchestrator.AgentState(actualRole, phase)
	}

	var dot, nameStr string
	switch state {
	case app.AgentStateActive:
		dot = lipgloss.NewStyle().Foreground(t.Success()).Render("●")
		nameStr = lipgloss.NewStyle().Foreground(c).Bold(true).Render(role)
	case app.AgentStateWaiting:
		dot = lipgloss.NewStyle().Foreground(t.Warning()).Render("●")
		nameStr = lipgloss.NewStyle().Foreground(t.TextMuted()).Render(role)
	default:
		dot = lipgloss.NewStyle().Foreground(t.BorderDim()).Render("●")
		nameStr = lipgloss.NewStyle().Foreground(t.TextMuted()).Render(role)
	}

	modelStr := ""
	cfg := config.Get()
	if cfg != nil {
		actualRole := role
		if role == "qa" {
			actualRole = "qa_synthesizer"
		}
		roleConfigName := config.AgentConfigForRole(actualRole)
		if agentCfg, ok := cfg.Agents[roleConfigName]; ok {
			if agentCfg.Backend == "claude-code" {
				modelStr = "claude-code"
			} else if agentCfg.Model != "" {
				parts := strings.Split(string(agentCfg.Model), "/")
				modelStr = parts[len(parts)-1]
			}
		}
	}

	modelSuffix := ""
	if truncated := truncateModelName(modelStr, n.width); truncated != "" {
		modelSuffix = lipgloss.NewStyle().Foreground(t.TextMuted()).Render(fmt.Sprintf(" (%s)", truncated))
	}

	return fmt.Sprintf(" %s %s%s", dot, nameStr, modelSuffix)
}

// truncateModelName computes how much of a model name can fit in the sidebar
// after accounting for all fixed-width overhead:
//   - Padding: 1 left + 1 right = 2
//   - Left border: 1
//   - Line prefix " ● rolename": space(1) + dot(1) + space(1) + longest_role("assurance"=9) = 12
//   - Suffix wrapper " (" + model + ")": 3 chars
//
// Total overhead = 2 + 1 + 12 + 3 = 18 characters.
// Using the longest role name ("assurance") as a fixed budget ensures all rows
// truncate to the same visual width, avoiding per-row inconsistency.
func truncateModelName(modelStr string, width int) string {
	if modelStr == "" {
		return ""
	}
	const overhead = 18 // padding(2) + border(1) + " ● assurance"(12) + " ()"(3)
	avail := width - overhead
	if avail > 3 {
		if len(modelStr) > avail {
			return modelStr[:avail-2] + ".."
		}
		return modelStr
	}
	return ""
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

	// Current Phase
	phaseStr := "Idle"
	var currentPhase app.Phase = app.PhaseIdle
	if n.app.Orchestrator != nil {
		currentPhase = n.app.Orchestrator.CurrentPhase()
		phaseStr = phaseString(currentPhase)
	}
	lines = append(lines, mutedStyle.Render("Phase"))
	lines = append(lines, textStyle.Render("• "+phaseStr), "")

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
	for _, role := range []string{"planner", "observer", "assurance", "qa"} {
		lines = append(lines, n.agentStatusLine(role, currentPhase))
	}

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
