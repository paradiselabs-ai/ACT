package navigator

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

func (n *ContextNavigator) SetApp(app *app.App) {
	n.app = app
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
		return "Intake"
	case app.PhaseBrownfieldAnalysis:
		return "Analyzing Codebase"
	case app.PhasePlanning:
		return "Planning"
	case app.PhaseExecuting:
		return "Executing"
	case app.PhaseValidating:
		return "Validating"
	case app.PhaseAwaitingInput:
		return "Awaiting input"
	default:
		return "Unknown"
	}
}

func getBadge(modelStr string) string {
	if modelStr == "" {
		return "H3"
	}
	lower := strings.ToLower(modelStr)
	switch {
	case strings.Contains(lower, "hermes"):
		return "H3"
	case strings.Contains(lower, "claude-3-7-sonnet"):
		return "S7"
	case strings.Contains(lower, "sonnet"):
		return "SN"
	case strings.Contains(lower, "gpt-4") || strings.Contains(lower, "gpt4"):
		return "G4"
	case strings.Contains(lower, "claude-code") || strings.Contains(lower, "claude"):
		return "CC"
	case strings.Contains(lower, "gemini-2.0-flash"):
		return "G2"
	case strings.Contains(lower, "gemini-1.5-pro"):
		return "G1P"
	case strings.Contains(lower, "deepseek-reasoner"), strings.Contains(lower, "deepseek-r1"):
		return "R1"
	case strings.Contains(lower, "deepseek-chat"), strings.Contains(lower, "deepseek-v3"):
		return "V3"
	case strings.Contains(lower, "llama-3.3-70b"):
		return "L70"
	case strings.Contains(lower, "qwen2.5-coder"):
		return "QW"
	default:
		parts := strings.Split(modelStr, "-")
		if len(parts) >= 2 && len(parts[0]) > 0 && len(parts[1]) > 0 {
			return strings.ToUpper(parts[0][:1] + parts[1][:1])
		}
		if len(modelStr) >= 2 {
			return strings.ToUpper(modelStr[:2])
		}
		return "M1"
	}
}

func (n *ContextNavigator) agentStatusLine(role string, phase app.Phase) (string, string, string) {
	t := theme.CurrentTheme()
	c := styles.AgentColor(role)

	actualRole := role
	if role == "qa" {
		actualRole = "qa_synthesizer"
	}

	isOnline := n.app != nil && (n.app.Agents[role] != nil || n.app.Agents[actualRole] != nil)

	state := app.AgentStateIdle
	if n.app != nil && n.app.Orchestrator != nil {
		state = n.app.Orchestrator.AgentState(actualRole, phase)
	}

	bgStyle := lipgloss.NewStyle().Background(t.Background())
	var glyph string
	if !isOnline {
		glyph = bgStyle.Foreground(t.Error()).Render("✕")
	} else {
		switch state {
		case app.AgentStateActive:
			glyph = bgStyle.Foreground(t.Success()).Bold(true).Render("●")
		case app.AgentStateWaiting:
			glyph = bgStyle.Foreground(t.Warning()).Render("◐")
		case app.AgentStateFailed:
			glyph = bgStyle.Foreground(t.Error()).Bold(true).Render("✕")
		default:
			glyph = bgStyle.Foreground(t.TextMuted()).Render("○")
		}
	}

	nameStr := bgStyle.Foreground(c).Render(role)
	if state == app.AgentStateActive {
		nameStr = bgStyle.Foreground(c).Bold(true).Render(role)
	} else if state == app.AgentStateFailed {
		nameStr = bgStyle.Foreground(t.Error()).Bold(true).Render(role)
	}

	modelStr := "hermes-3-llama-3.1-8b"
	cfg := config.Get()
	if cfg != nil {
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

	badge := getBadge(modelStr)
	badgeStyle := bgStyle.Foreground(t.TextMuted()).Render(fmt.Sprintf("[%s]", badge))

	sp := bgStyle.Render(" ")
	line := fmt.Sprintf("%s%s%s%s%s%s", sp, glyph, sp, nameStr, sp, badgeStyle)
	return line, badge, modelStr
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
	b := lipgloss.NewStyle()
	headerStyle := b.Bold(true).Foreground(t.Primary())
	mutedStyle := b.Foreground(t.TextMuted())
	textStyle := b.Foreground(t.Text())

	var lines []string
	lines = append(lines, headerStyle.Render("Context"), "")

	// Current Phase
	phaseStr := "Idle"
	var currentPhase app.Phase = app.PhaseIdle
	if n.app != nil && n.app.Orchestrator != nil {
		currentPhase = n.app.Orchestrator.CurrentPhase()
		phaseStr = phaseString(currentPhase)
	}
	lines = append(lines, mutedStyle.Render("Phase"))
	lines = append(lines, textStyle.Render(phaseStr), "")

	// Unified Last run: [time] role · event
	nowStr := time.Now().Format("15:04")
	lastRunStr := fmt.Sprintf("%s · idle", nowStr)
	if n.app != nil && n.app.Orchestrator != nil {
		speaker := n.app.Orchestrator.CurrentSpeaker()
		if speaker != "" {
			label := speaker
			if speaker == "qa_synthesizer" {
				label = "qa"
			}
			lastRunStr = fmt.Sprintf("%s · %s %s", nowStr, label, strings.ToLower(phaseStr))
		}
	}
	lines = append(lines, mutedStyle.Render("Last run"))
	lines = append(lines, textStyle.Render(lastRunStr), "")

	// Registered agents & collecting badge legend map
	lines = append(lines, mutedStyle.Render("Agents"))
	badgeMap := make(map[string]string)
	var badgeOrder []string

	for _, role := range []string{"planner", "observer", "assurance", "qa"} {
		statusLine, badge, modelStr := n.agentStatusLine(role, currentPhase)
		lines = append(lines, statusLine)
		if _, exists := badgeMap[badge]; !exists {
			badgeMap[badge] = modelStr
			badgeOrder = append(badgeOrder, badge)
		}
	}

	// Unconditional Legend at bottom
	lines = append(lines, "", mutedStyle.Render("Legend"))
	for _, badge := range badgeOrder {
		modelStr := badgeMap[badge]
		budget := n.width - len(badge) - 6
		if budget < 1 {
			modelStr = ""
		} else {
			modelStr = ansi.Truncate(modelStr, budget, "..")
		}
		var legendLine string
		if modelStr != "" {
			legendLine = fmt.Sprintf(" [%s] %s", badge, modelStr)
		} else {
			legendLine = fmt.Sprintf(" [%s]", badge)
		}
		lines = append(lines, textStyle.Render(legendLine))
	}

	content := strings.Join(lines, "\n")
	rendered := baseStyle.Render(content)
	if n.height > 0 {
		rLines := strings.Split(rendered, "\n")
		if len(rLines) > n.height {
			rendered = strings.Join(rLines[:n.height], "\n")
		}
	}
	bgSeq := lipgloss.NewStyle().Background(t.Background()).Render("")
	if bgSeq != "" {
		rendered = strings.ReplaceAll(rendered, "\x1b[0m", "\x1b[0m"+bgSeq)
	}
	return tea.NewView(rendered)
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
