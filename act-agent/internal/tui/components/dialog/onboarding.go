package dialog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opencode-ai/opencode/internal/tui/styles"
	"github.com/opencode-ai/opencode/internal/tui/theme"
	"github.com/opencode-ai/opencode/internal/tui/util"
)

type ShowOnboardingDialogMsg struct {
	Show bool
}

type onboardingRole struct {
	Label    string
	AgentKey string
	Options  []onboardingModel
	Selected int
}

type onboardingModel struct {
	Provider string
	Model    string
	Note     string
}

type detectedProvider struct {
	EnvVar   string
	Provider string
	Detail   string
	Found    bool
}

type onboardingCmp struct {
	width  int
	height int

	step   int
	cursor int

	tier1 []onboardingRole
	tier2 []onboardingRole

	detected []detectedProvider
	saveErr  error
}

type OnboardingCmp interface {
	tea.Model
}

func NewOnboardingCmp() OnboardingCmp {
	detected := detectProviders()
	defaultTier1 := bestTier1(detected)
	defaultTier2 := bestTier2(detected)

	return &onboardingCmp{
		step: 0,
		tier1: []onboardingRole{
			{Label: "Planner", AgentKey: "planner", Options: tierOptions(detected), Selected: defaultTier1},
			{Label: "Observer", AgentKey: "observer", Options: tierOptions(detected), Selected: defaultTier1},
			{Label: "Assurance", AgentKey: "assurance", Options: tierOptions(detected), Selected: defaultTier1},
			{Label: "QA", AgentKey: "qa_synthesizer", Options: tierOptions(detected), Selected: defaultTier1},
		},
		tier2: []onboardingRole{
			{Label: "Developer", AgentKey: "developer", Options: tierOptions(detected), Selected: defaultTier2},
			{Label: "Frontend", AgentKey: "frontend_dev", Options: tierOptions(detected), Selected: defaultTier2},
			{Label: "Backend", AgentKey: "backend_dev", Options: tierOptions(detected), Selected: defaultTier2},
			{Label: "QA Engineer", AgentKey: "qa_engineer", Options: tierOptions(detected), Selected: defaultTier2},
			{Label: "Researcher", AgentKey: "researcher", Options: tierOptions(detected), Selected: defaultTier2},
		},
		detected: detected,
	}
}

func (o *onboardingCmp) Init() tea.Cmd {
	return nil
}

func (o *onboardingCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		o.width = msg.Width
		o.height = msg.Height
		return o, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			return o, util.CmdHandler(CloseInitDialogMsg{Initialize: false})
		}

		switch o.step {
		case 0, 1:
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				o.step++
				o.cursor = 0
			}
		case 2:
			o.handleRoleSelection(msg, &o.tier1)
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				o.step = 3
				o.cursor = 0
			}
		case 3:
			o.handleRoleSelection(msg, &o.tier2)
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				o.step = 4
				o.saveErr = o.writeConfig()
			}
		case 4:
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				if o.saveErr != nil {
					return o, util.ReportError(o.saveErr)
				}
				return o, util.CmdHandler(CloseInitDialogMsg{Initialize: true})
			}
		}
	}
	return o, nil
}

func (o *onboardingCmp) handleRoleSelection(msg tea.KeyMsg, roles *[]onboardingRole) {
	if len(*roles) == 0 {
		return
	}
	current := &(*roles)[o.cursor]
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if o.cursor > 0 {
			o.cursor--
		} else {
			o.cursor = len(*roles) - 1
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		o.cursor = (o.cursor + 1) % len(*roles)
	case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
		if current.Selected > 0 {
			current.Selected--
		} else {
			current.Selected = len(current.Options) - 1
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
		current.Selected = (current.Selected + 1) % len(current.Options)
	}
}

func (o *onboardingCmp) View() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()
	maxWidth := min(90, max(70, o.width-12))

	titleStyle := baseStyle.Foreground(t.Primary()).Bold(true).Width(maxWidth)
	bodyStyle := baseStyle.Foreground(t.Text()).Width(maxWidth)
	mutedStyle := baseStyle.Foreground(t.TextMuted()).Width(maxWidth)

	var content string
	switch o.step {
	case 0:
		content = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("Welcome to ACT - Agent Coordination Toolkit"),
			"",
			bodyStyle.Render("ACT coordinates multiple AI agents working together on software projects. It uses a two-tier architecture:"),
			"",
			bodyStyle.Render("  Tier 1 (NesTTY)   Planner, Observer, Assurance, QA"),
			mutedStyle.Render("                     These roles need stronger models."),
			"",
			bodyStyle.Render("  Tier 2 (Swarm)    Developer agents that execute tasks."),
			mutedStyle.Render("                     Can use cheaper or local models."),
			"",
			mutedStyle.Render("Press Enter to configure your providers ->"),
		)
	case 1:
		lines := []string{
			titleStyle.Render("Detected API Keys:"),
			"",
		}
		for _, p := range o.detected {
			mark := "✗"
			status := "Not set"
			if p.Found {
				mark = "✓"
				status = p.Detail
			}
			lines = append(lines, bodyStyle.Render(fmt.Sprintf("  %s %-18s %s", mark, p.EnvVar, status)))
		}
		lines = append(lines,
			"",
			mutedStyle.Render("You can add keys later in ~/.act.json"),
			"",
			mutedStyle.Render("Press Enter to configure roles ->"),
		)
		content = lipgloss.JoinVertical(lipgloss.Left, lines...)
	case 2:
		content = o.renderRoleStep("Tier 1 - NesTTY Roles (stronger models recommended)", o.tier1, "Enter to continue ->")
	case 3:
		content = o.renderRoleStep("Tier 2 - Swarm Agents (cheaper/local models work well)", o.tier2, "Enter to save configuration ->")
	case 4:
		tier1Summary, tier2Summary := o.selectedTierSummary()
		status := "Configuration saved to ~/.act.json"
		if o.saveErr != nil {
			status = "Failed to save ~/.act.json: " + o.saveErr.Error()
		}
		content = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(status),
			"",
			bodyStyle.Render("  Tier 1: "+tier1Summary),
			bodyStyle.Render("  Tier 2: "+tier2Summary),
			"",
			mutedStyle.Render("You can edit this file anytime. Press Enter to start ->"),
		)
	}

	return baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.TextMuted()).
		Width(maxWidth + 4).
		Render(content)
}

func (o *onboardingCmp) renderRoleStep(title string, roles []onboardingRole, footer string) string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()
	maxWidth := min(90, max(70, o.width-12))

	titleStyle := baseStyle.Foreground(t.Primary()).Bold(true).Width(maxWidth)
	lineStyle := baseStyle.Foreground(t.Text()).Width(maxWidth)
	mutedStyle := baseStyle.Foreground(t.TextMuted()).Width(maxWidth)
	selectedStyle := baseStyle.Foreground(t.Background()).Background(t.Primary()).Bold(true).Width(maxWidth)

	lines := []string{titleStyle.Render(title), ""}
	for i, role := range roles {
		option := role.Options[role.Selected]
		line := fmt.Sprintf("  %-12s [%s] %s", role.Label+":", option.Provider, option.Model)
		if option.Note != "" {
			line += "  " + option.Note
		}
		if i == o.cursor {
			lines = append(lines, selectedStyle.Render(line))
		} else {
			lines = append(lines, lineStyle.Render(line))
		}
	}
	lines = append(lines,
		"",
		mutedStyle.Render("  ↑/↓ to select role, ←/→ to change provider/model"),
		mutedStyle.Render("  "+footer),
	)

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (o *onboardingCmp) selectedTierSummary() (string, string) {
	t1 := uniqueProviders(o.tier1)
	t2 := uniqueProviders(o.tier2)
	return strings.Join(t1, ", "), strings.Join(t2, ", ")
}

func uniqueProviders(roles []onboardingRole) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, r := range roles {
		p := r.Options[r.Selected].Provider
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func (o *onboardingCmp) writeConfig() error {
	type providerCfg struct {
		Disabled bool `json:"disabled"`
	}
	type agentCfg struct {
		Model     string `json:"model"`
		MaxTokens int    `json:"maxTokens"`
	}
	type fileCfg struct {
		Schema    string                 `json:"$schema"`
		Providers map[string]providerCfg `json:"providers"`
		Agents    map[string]agentCfg    `json:"agents"`
	}

	providers := map[string]providerCfg{}
	agents := map[string]agentCfg{}

	for _, role := range append([]onboardingRole{}, append(o.tier1, o.tier2...)...) {
		choice := role.Options[role.Selected]
		providers[choice.Provider] = providerCfg{Disabled: false}
		agents[role.AgentKey] = agentCfg{Model: choice.Model, MaxTokens: 5000}
	}

	if planner, ok := agents["planner"]; ok {
		planner.MaxTokens = 8000
		agents["planner"] = planner
	}
	if _, ok := agents["coder"]; !ok {
		if planner, ok := agents["observer"]; ok {
			agents["coder"] = agentCfg{Model: planner.Model, MaxTokens: 5000}
		}
	}

	payload := fileCfg{
		Schema:    "./act-schema.json",
		Providers: providers,
		Agents:    agents,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".act.json")
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func detectProviders() []detectedProvider {
	checks := []detectedProvider{
		{EnvVar: "ANTHROPIC_API_KEY", Provider: "anthropic", Detail: "Claude models available"},
		{EnvVar: "OPENAI_API_KEY", Provider: "openai", Detail: "OpenAI models available"},
		{EnvVar: "GEMINI_API_KEY", Provider: "google", Detail: "Gemini models available"},
		{EnvVar: "GROQ_API_KEY", Provider: "groq", Detail: "Free tier (Llama 3.3 70B)"},
		{EnvVar: "OPENROUTER_API_KEY", Provider: "openrouter", Detail: "OpenRouter models available"},
		{EnvVar: "XAI_API_KEY", Provider: "xai", Detail: "xAI models available"},
		{EnvVar: "GITHUB_TOKEN", Provider: "github", Detail: "Copilot available"},
	}

	for i := range checks {
		checks[i].Found = os.Getenv(checks[i].EnvVar) != ""
	}
	return checks
}

func tierOptions(detected []detectedProvider) []onboardingModel {
	hasAnthropic := false
	hasGroq := false
	for _, p := range detected {
		if p.Provider == "anthropic" && p.Found {
			hasAnthropic = true
		}
		if p.Provider == "groq" && p.Found {
			hasGroq = true
		}
	}

	options := []onboardingModel{}
	if hasAnthropic {
		options = append(options,
			onboardingModel{Provider: "anthropic", Model: "claude-opus-4-20250514", Note: "(strong)"},
			onboardingModel{Provider: "anthropic", Model: "claude-sonnet-4-20250514", Note: "(balanced)"},
		)
	}
	if hasGroq {
		options = append(options,
			onboardingModel{Provider: "groq", Model: "llama-3.3-70b-versatile", Note: "(free)"},
		)
	}
	if len(options) == 0 {
		options = append(options,
			onboardingModel{Provider: "anthropic", Model: "claude-sonnet-4-20250514", Note: "(set key later)"},
			onboardingModel{Provider: "groq", Model: "llama-3.3-70b-versatile", Note: "(set key later)"},
		)
	}
	return options
}

func bestTier1(detected []detectedProvider) int {
	options := tierOptions(detected)
	for i, option := range options {
		if option.Provider == "anthropic" && option.Model == "claude-opus-4-20250514" {
			return i
		}
	}
	return 0
}

func bestTier2(detected []detectedProvider) int {
	options := tierOptions(detected)
	for i, option := range options {
		if option.Provider == "groq" {
			return i
		}
	}
	return 0
}
