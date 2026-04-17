package dialog

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/util"
)

type ShowOnboardingDialogMsg struct {
	Show bool
}

// CloseOnboardingMsg is sent when the onboarding wizard finishes (confirmed
// or cancelled). The TUI uses it to hide the overlay and mark the project
// as initialized. Initialize=true means the user completed the wizard;
// Initialize=false means they cancelled out of it.
type CloseOnboardingMsg struct {
	Initialize bool
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

	// Per-Tier-2-role backend selection ("act-agent" or "claude-code").
	// Index matches o.tier2 by position.
	tier2Backends []string

	// Nomik state — detected at construction time
	nomikAvailable bool
	nomikEnabled   bool

	// Claude Code availability — detected at construction time
	claudeAvailable bool

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

	tier2 := []onboardingRole{
		{Label: "Developer", AgentKey: "developer", Options: tierOptions(detected), Selected: defaultTier2},
		{Label: "Frontend", AgentKey: "frontend_dev", Options: tierOptions(detected), Selected: defaultTier2},
		{Label: "Backend", AgentKey: "backend_dev", Options: tierOptions(detected), Selected: defaultTier2},
		{Label: "QA Engineer", AgentKey: "qa_engineer", Options: tierOptions(detected), Selected: defaultTier2},
		{Label: "Researcher", AgentKey: "researcher", Options: tierOptions(detected), Selected: defaultTier2},
	}

	// Default every Tier 2 role to act-agent backend
	tier2Backends := make([]string, len(tier2))
	for i := range tier2Backends {
		tier2Backends[i] = "act-agent"
	}

	nomikAvail := detectNomik()
	claudeAvail := detectClaudeCode()

	return &onboardingCmp{
		step: 0,
		tier1: []onboardingRole{
			{Label: "Planner", AgentKey: "planner", Options: tierOptions(detected), Selected: defaultTier1},
			{Label: "Observer", AgentKey: "observer", Options: tierOptions(detected), Selected: defaultTier1},
			{Label: "Assurance", AgentKey: "assurance", Options: tierOptions(detected), Selected: defaultTier1},
			{Label: "QA", AgentKey: "qa_synthesizer", Options: tierOptions(detected), Selected: defaultTier1},
		},
		tier2:           tier2,
		tier2Backends:   tier2Backends,
		nomikAvailable:  nomikAvail,
		nomikEnabled:    nomikAvail, // default ON if available
		claudeAvailable: claudeAvail,
		detected:        detected,
	}
}

// detectNomik returns true if both `nomik` is in PATH and a TCP connection
// to localhost:7687 (Neo4j Bolt) succeeds within 1 second.
func detectNomik() bool {
	if _, err := exec.LookPath("nomik"); err != nil {
		return false
	}
	conn, err := net.DialTimeout("tcp", "localhost:7687", 1*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// detectClaudeCode returns true if the `claude` binary is in PATH.
func detectClaudeCode() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

func (o *onboardingCmp) Init() tea.Cmd {
	return nil
}

// Wizard step constants. The step machine skips backend (4) and nomik (5)
// when their respective binaries aren't available, so the user only sees
// steps that are actually meaningful on their machine.
const (
	stepWelcome      = 0
	stepKeys         = 1
	stepTier1Models  = 2
	stepTier2Models  = 3
	stepTier2Backend = 4 // skipped if claude not in PATH
	stepNomik        = 5 // skipped if nomik unavailable
	stepSave         = 6
)

// nextStep advances past steps that should be skipped (backend if no claude,
// nomik if no nomik). Always lands on a meaningful step.
func (o *onboardingCmp) nextStep(from int) int {
	next := from + 1
	for {
		if next == stepTier2Backend && !o.claudeAvailable {
			next++
			continue
		}
		if next == stepNomik && !o.nomikAvailable {
			next++
			continue
		}
		return next
	}
}

func (o *onboardingCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		o.width = msg.Width
		o.height = msg.Height
		return o, nil
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			return o, util.CmdHandler(CloseOnboardingMsg{Initialize: false})
		}

		switch o.step {
		case stepWelcome, stepKeys:
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				o.step = o.nextStep(o.step)
				o.cursor = 0
			}
		case stepTier1Models:
			o.handleRoleSelection(msg, &o.tier1)
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				o.step = o.nextStep(o.step)
				o.cursor = 0
			}
		case stepTier2Models:
			o.handleRoleSelection(msg, &o.tier2)
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				o.step = o.nextStep(o.step)
				o.cursor = 0
			}
		case stepTier2Backend:
			o.handleBackendSelection(msg)
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				o.step = o.nextStep(o.step)
				o.cursor = 0
			}
		case stepNomik:
			o.handleNomikToggle(msg)
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				o.step = o.nextStep(o.step)
				o.saveErr = o.writeConfig()
			}
		case stepSave:
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				if o.saveErr != nil {
					return o, util.ReportError(o.saveErr)
				}
				return o, util.CmdHandler(CloseOnboardingMsg{Initialize: true})
			}
		}
	}
	return o, nil
}

// handleBackendSelection drives the Tier 2 backend picker step.
// ↑/↓ moves between roles, ←/→ toggles act-agent vs claude-code.
func (o *onboardingCmp) handleBackendSelection(msg tea.KeyPressMsg) {
	if len(o.tier2Backends) == 0 {
		return
	}
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if o.cursor > 0 {
			o.cursor--
		} else {
			o.cursor = len(o.tier2Backends) - 1
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		o.cursor = (o.cursor + 1) % len(o.tier2Backends)
	case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h", "right", "l", " "))):
		if o.tier2Backends[o.cursor] == "act-agent" {
			o.tier2Backends[o.cursor] = "claude-code"
		} else {
			o.tier2Backends[o.cursor] = "act-agent"
		}
	}
}

// handleNomikToggle drives the Nomik enable/disable step.
// Space, ←, →, y, n all toggle. Enter advances.
func (o *onboardingCmp) handleNomikToggle(msg tea.KeyPressMsg) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys(" ", "h", "l", "left", "right"))):
		o.nomikEnabled = !o.nomikEnabled
	case key.Matches(msg, key.NewBinding(key.WithKeys("y", "Y"))):
		o.nomikEnabled = true
	case key.Matches(msg, key.NewBinding(key.WithKeys("n", "N"))):
		o.nomikEnabled = false
	}
}

func (o *onboardingCmp) handleRoleSelection(msg tea.KeyPressMsg, roles *[]onboardingRole) {
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

func (o *onboardingCmp) View() tea.View {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()
	maxWidth := min(90, max(70, o.width-12))

	titleStyle := baseStyle.Foreground(t.Primary()).Bold(true).Width(maxWidth)
	bodyStyle := baseStyle.Foreground(t.Text()).Width(maxWidth)
	mutedStyle := baseStyle.Foreground(t.TextMuted()).Width(maxWidth)

	var content string
	switch o.step {
	case stepWelcome:
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
	case stepKeys:
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
	case stepTier1Models:
		content = o.renderRoleStep("Tier 1 - NesTTY Roles (stronger models recommended)", o.tier1, "Enter to continue ->")
	case stepTier2Models:
		content = o.renderRoleStep("Tier 2 - Swarm Agents (cheaper/local models work well)", o.tier2, "Enter to continue ->")
	case stepTier2Backend:
		content = o.renderBackendStep()
	case stepNomik:
		content = o.renderNomikStep()
	case stepSave:
		tier1Summary, tier2Summary := o.selectedTierSummary()
		status := "Configuration saved to ~/.act.json"
		if o.saveErr != nil {
			status = "Failed to save ~/.act.json: " + o.saveErr.Error()
		}
		summaryLines := []string{
			titleStyle.Render(status),
			"",
			bodyStyle.Render("  Tier 1: " + tier1Summary),
			bodyStyle.Render("  Tier 2: " + tier2Summary),
		}
		if o.claudeAvailable {
			summaryLines = append(summaryLines,
				bodyStyle.Render("  Backends: "+o.backendSummary()),
			)
		}
		if o.nomikAvailable {
			nomikState := "disabled"
			if o.nomikEnabled {
				nomikState = "enabled"
			}
			summaryLines = append(summaryLines,
				bodyStyle.Render("  Nomik: "+nomikState),
			)
		}
		summaryLines = append(summaryLines,
			"",
			mutedStyle.Render("You can change these later via /swarm and /nomik. Press Enter to start ->"),
		)
		content = lipgloss.JoinVertical(lipgloss.Left, summaryLines...)
	}

	return tea.NewView(baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.TextMuted()).
		Width(maxWidth + 4).
		Render(content))
}

func (o *onboardingCmp) renderBackendStep() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()
	maxWidth := min(90, max(70, o.width-12))

	titleStyle := baseStyle.Foreground(t.Primary()).Bold(true).Width(maxWidth)
	lineStyle := baseStyle.Foreground(t.Text()).Width(maxWidth)
	mutedStyle := baseStyle.Foreground(t.TextMuted()).Width(maxWidth)
	selectedStyle := baseStyle.Foreground(t.Background()).Background(t.Primary()).Bold(true).Width(maxWidth)

	lines := []string{
		titleStyle.Render("Tier 2 Backend - Choose how each swarm role executes"),
		"",
		mutedStyle.Render("  act-agent  = the local Go binary (default, fast, configured per-role above)"),
		mutedStyle.Render("  claude-code = the official Claude Code CLI (uses your Claude session)"),
		"",
	}
	for i, role := range o.tier2 {
		backend := o.tier2Backends[i]
		line := fmt.Sprintf("  %-12s [%s]", role.Label+":", backend)
		if i == o.cursor {
			lines = append(lines, selectedStyle.Render(line))
		} else {
			lines = append(lines, lineStyle.Render(line))
		}
	}
	lines = append(lines,
		"",
		mutedStyle.Render("  ↑/↓ to select role, ←/→ or space to toggle backend"),
		mutedStyle.Render("  Enter to continue ->"),
	)
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (o *onboardingCmp) renderNomikStep() string {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()
	maxWidth := min(90, max(70, o.width-12))

	titleStyle := baseStyle.Foreground(t.Primary()).Bold(true).Width(maxWidth)
	bodyStyle := baseStyle.Foreground(t.Text()).Width(maxWidth)
	mutedStyle := baseStyle.Foreground(t.TextMuted()).Width(maxWidth)
	selectedStyle := baseStyle.Foreground(t.Background()).Background(t.Primary()).Bold(true).Width(maxWidth)

	lines := []string{
		titleStyle.Render("Nomik - Codebase Knowledge Graph"),
		"",
		bodyStyle.Render("Detected nomik + Neo4j on localhost:7687."),
		"",
		mutedStyle.Render("Nomik scans your project's source code into a graph that agents can"),
		mutedStyle.Render("query for impact analysis, architecture rules, and module clusters."),
		mutedStyle.Render("Recommended: Yes. Disable for non-code projects or when Neo4j is busy."),
		"",
	}
	yesLine := "  [ ] Yes - enable Nomik for new projects"
	noLine := "  [ ] No  - disable (agents will skip codebase graph queries)"
	if o.nomikEnabled {
		yesLine = "  [x] Yes - enable Nomik for new projects"
	} else {
		noLine = "  [x] No  - disable (agents will skip codebase graph queries)"
	}
	if o.nomikEnabled {
		lines = append(lines, selectedStyle.Render(yesLine))
		lines = append(lines, bodyStyle.Render(noLine))
	} else {
		lines = append(lines, bodyStyle.Render(yesLine))
		lines = append(lines, selectedStyle.Render(noLine))
	}
	lines = append(lines,
		"",
		mutedStyle.Render("  Space / ←→ to toggle, y / n to set directly"),
		mutedStyle.Render("  Enter to save and start ->"),
	)
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (o *onboardingCmp) backendSummary() string {
	parts := []string{}
	for i, role := range o.tier2 {
		parts = append(parts, fmt.Sprintf("%s=%s", role.Label, o.tier2Backends[i]))
	}
	return strings.Join(parts, ", ")
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
		Backend   string `json:"backend,omitempty"` // Tier 2 only
	}
	type nomikCfg struct {
		Enabled bool `json:"enabled"`
	}
	type fileCfg struct {
		Schema    string                 `json:"$schema"`
		Providers map[string]providerCfg `json:"providers"`
		Agents    map[string]agentCfg    `json:"agents"`
		Nomik     *nomikCfg              `json:"nomik,omitempty"`
	}

	providers := map[string]providerCfg{}
	agents := map[string]agentCfg{}

	// Tier 1 — model only, no backend (in-process goroutines)
	for _, role := range o.tier1 {
		choice := role.Options[role.Selected]
		providers[choice.Provider] = providerCfg{Disabled: false}
		agents[role.AgentKey] = agentCfg{Model: choice.Model, MaxTokens: 5000}
	}

	// Tier 2 — model + backend per role
	for i, role := range o.tier2 {
		choice := role.Options[role.Selected]
		providers[choice.Provider] = providerCfg{Disabled: false}
		backend := "act-agent"
		if i < len(o.tier2Backends) {
			backend = o.tier2Backends[i]
		}
		agents[role.AgentKey] = agentCfg{
			Model:     choice.Model,
			MaxTokens: 5000,
			Backend:   backend,
		}
	}

	if planner, ok := agents["planner"]; ok {
		planner.MaxTokens = 8000
		agents["planner"] = planner
	}
	// The 4 Tier 1 agents (planner, observer, assurance, qa_synthesizer) and
	// 5 Tier 2 swarm roles each have their own model — there is no "default"
	// agent in ACT. UI surfaces that need to display "current model" use
	// config.Tier1Configs() to show all 4.

	payload := fileCfg{
		Schema:    "./act-schema.json",
		Providers: providers,
		Agents:    agents,
	}
	if o.nomikAvailable {
		payload.Nomik = &nomikCfg{Enabled: o.nomikEnabled}
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
