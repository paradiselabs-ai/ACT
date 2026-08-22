package dialog

import (
	"fmt"
	"sort"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/models"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/layout"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/util"
)

// ModelSelectedMsg is sent when a model is updated for an agent.
type ModelSelectedMsg struct {
	Role  string
	Model models.Model
}

// CloseModelDialogMsg is sent when the dialog is dismissed.
type CloseModelDialogMsg struct{}

// ModelDialog is a two-step dialog that lets users edit the provider and
// model string for a given role (defaults to Planner).
// There is no model registry — the user types the upstream model string
// verbatim, and the upstream API validates it on the next request.
type ModelDialog interface {
	tea.Model
	layout.Bindings
	SetRole(role string)
}

type modelDialogCmp struct {
	form     *huh.Form
	role     string
	provider models.ModelProvider
	modelID  string
}

func (m *modelDialogCmp) SetRole(role string) {
	if role == "" {
		role = string(config.RolePlanner)
	}
	m.role = role
	m.form = nil
	m.provider = ""
	m.modelID = ""
}

func (m *modelDialogCmp) buildForm() tea.Cmd {
	if m.role == "" {
		m.role = string(config.RolePlanner)
	}
	cfg := config.Get()
	current := config.Agent{}
	if cfg != nil {
		roleName := config.AgentConfigForRole(m.role)
		if a, ok := cfg.Agents[roleName]; ok {
			current = a
		}
	}
	if m.provider == "" {
		m.provider = current.Provider
	}
	if m.modelID == "" {
		m.modelID = string(current.Model)
	}

	providerOpts := buildProviderOptions(cfg)

	title := fmt.Sprintf("Provider (%s)", m.role)

	m.form = huh.NewForm(huh.NewGroup(
		huh.NewSelect[models.ModelProvider]().
			Title(title).
			Options(providerOpts...).
			Value(&m.provider),
		huh.NewInput().
			Title("Model string (verbatim, as upstream API expects)").
			Description("Examples: google/gemini-2.0-flash-001 · anthropic/claude-3.5-sonnet · meta-llama/llama-3.3-70b-instruct · deepseek/deepseek-chat").
			Value(&m.modelID),
	)).WithShowHelp(false).WithTheme(actHuhTheme())
	return m.form.Init()
}

func (m *modelDialogCmp) Init() tea.Cmd {
	return m.buildForm()
}

func (m *modelDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Rebuild ONLY when the form doesn't exist yet. The old unconditional
	// rebuild fired on every message while State != Normal — including
	// arbitrary pubsub traffic between completion and dialog close — and
	// would nil-deref if a non-WindowSizeMsg arrived before any
	// SetRole/Init had run.
	if m.form == nil {
		return m, m.buildForm()
	}
	updated, cmd := m.form.Update(msg)
	if f, ok := updated.(*huh.Form); ok {
		m.form = f
	}
	switch m.form.State {
	case huh.StateCompleted:
		picked := models.Model{
			ID:       models.ModelID(m.modelID),
			Provider: m.provider,
		}
		return m, util.CmdHandler(ModelSelectedMsg{Role: m.role, Model: picked})
	case huh.StateAborted:
		return m, util.CmdHandler(CloseModelDialogMsg{})
	}
	return m, cmd
}

func (m *modelDialogCmp) View() tea.View {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle().Background(t.Background())

	if m.form == nil {
		rendered := baseStyle.Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderBackground(t.Background()).
			BorderForeground(t.BorderFocused()).
			Width(40).
			Render("Loading…")
		return tea.NewView(styles.ForceBackgroundOnAllLines(rendered, t.Background()))
	}

	boxStyle := baseStyle.
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.BorderFocused())

	rendered := boxStyle.Render(m.form.View())
	return tea.NewView(styles.ForceBackgroundOnAllLines(rendered, t.Background()))
}

func (m *modelDialogCmp) BindingKeys() []key.Binding {
	if m.form == nil {
		return nil
	}
	return m.form.KeyBinds()
}

// NewModelDialogCmp constructs a freshly-initialized model dialog.
func NewModelDialogCmp() ModelDialog {
	return &modelDialogCmp{role: string(config.RolePlanner)}
}

// GetSelectedModel returns the configured Model for the Planner role.
// Used by callers that want to display "current model" before opening
// the dialog. The returned Model only has ID and Provider populated —
// MaxTokens is not relevant outside the per-turn provider call.
func GetSelectedModel(cfg *config.Config) models.Model {
	if cfg == nil {
		return models.Model{}
	}
	a, ok := cfg.Agents[config.RolePlanner]
	if !ok {
		return models.Model{}
	}
	return models.Model{ID: a.Model, Provider: a.Provider}
}

// buildProviderOptions returns the usable providers, sorted alphabetically.
// A provider is usable when it has the credentials its API needs: apiKey for
// cloud providers, baseURL (or LM Studio default) for local. Bedrock/VertexAI
// authenticate via the environment so they are always offered when configured.
// Without a popularity map, alphabetical is a stable, predictable default.
func buildProviderOptions(cfg *config.Config) []huh.Option[models.ModelProvider] {
	var enabled []models.ModelProvider
	if cfg != nil {
		for id, p := range cfg.Providers {
			if p.Usable(id) {
				enabled = append(enabled, id)
			}
		}
	}
	sort.Slice(enabled, func(i, j int) bool { return string(enabled[i]) < string(enabled[j]) })
	if len(enabled) == 0 {
		// No providers configured yet — let the user pick from the known
		// identifiers anyway. They'll get a clear error from validateAgent
		// if the provider isn't actually configured.
		enabled = []models.ModelProvider{
			models.ProviderAnthropic, models.ProviderOpenAI, models.ProviderOpenRouter,
			models.ProviderGROQ, models.ProviderGemini, models.ProviderXAI,
			models.ProviderAzure, models.ProviderVertexAI, models.ProviderBedrock,
			models.ProviderLocal,
		}
	}
	opts := make([]huh.Option[models.ModelProvider], len(enabled))
	for i, p := range enabled {
		opts[i] = huh.NewOption(fmt.Sprintf("%s", p), p)
	}
	return opts
}
