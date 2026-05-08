package dialog

import (
	"fmt"
	"slices"

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

// ModelSelectedMsg is sent when a model is selected
type ModelSelectedMsg struct {
	Model models.Model
}

// CloseModelDialogMsg is sent when a model is selected
type CloseModelDialogMsg struct{}

// ModelDialog interface for the model selection dialog
type ModelDialog interface {
	tea.Model
	layout.Bindings
}

type modelDialogCmp struct {
	form     *huh.Form
	selected models.Model
}

func (m *modelDialogCmp) buildForm() tea.Cmd {
	cfg := config.Get()
	allModels := getAllModels(cfg)
	current := GetSelectedModel(cfg)

	opts := make([]huh.Option[models.Model], len(allModels))
	for i, model := range allModels {
		label := fmt.Sprintf("%s (%s)", model.Name, model.Provider)
		opts[i] = huh.NewOption(label, model)
	}
	m.selected = current
	m.form = huh.NewForm(huh.NewGroup(
		huh.NewSelect[models.Model]().
			Title("Select Model").
			Filtering(true).
			Options(opts...).
			Value(&m.selected),
	)).WithShowHelp(false).WithTheme(actHuhTheme())
	return m.form.Init()
}

func (m *modelDialogCmp) Init() tea.Cmd {
	return m.buildForm()
}

func (m *modelDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Reset the form if the dialog is being reopened after a previous use.
	if m.form.State != huh.StateNormal {
		return m, m.buildForm()
	}
	updated, cmd := m.form.Update(msg)
	if f, ok := updated.(*huh.Form); ok {
		m.form = f
	}
	switch m.form.State {
	case huh.StateCompleted:
		return m, util.CmdHandler(ModelSelectedMsg{Model: m.selected})
	case huh.StateAborted:
		return m, util.CmdHandler(CloseModelDialogMsg{})
	}
	return m, cmd
}

func (m *modelDialogCmp) View() tea.View {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle()
	if m.form == nil {
		return tea.NewView(baseStyle.Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderBackground(t.Background()).
			BorderForeground(t.BorderFocused()).
			Width(40).
			Render("Loading models..."))
	}
	return tea.NewView(baseStyle.Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.BorderFocused()).
		Render(m.form.View()))
}

func (m *modelDialogCmp) BindingKeys() []key.Binding {
	if m.form == nil {
		return nil
	}
	return m.form.KeyBinds()
}

func NewModelDialogCmp() ModelDialog {
	return &modelDialogCmp{}
}

// GetSelectedModel returns the currently configured model for the Planner role.
func GetSelectedModel(cfg *config.Config) models.Model {
	// TODO Phase 3: this dialog should show a Tier 1 role picker first
	// (Planner / Observer / Assurance / QA). For now we read the Planner
	// config since that's the agent the human is actively talking to.
	if cfg != nil {
		if a, ok := cfg.Agents[config.RolePlanner]; ok {
			return models.SupportedModels[a.Model]
		}
	}
	return models.Model{}
}

// getAllModels returns all models from enabled providers, sorted by provider popularity then name.
func getAllModels(cfg *config.Config) []models.Model {
	enabledProviders := getEnabledProviders(cfg)
	enabled := make(map[models.ModelProvider]bool, len(enabledProviders))
	for _, p := range enabledProviders {
		enabled[p] = true
	}

	var result []models.Model
	for _, model := range models.SupportedModels {
		if enabled[model.Provider] {
			result = append(result, model)
		}
	}

	slices.SortFunc(result, func(a, b models.Model) int {
		rA := models.ProviderPopularity[a.Provider]
		rB := models.ProviderPopularity[b.Provider]
		if rA == 0 {
			rA = 999
		}
		if rB == 0 {
			rB = 999
		}
		if rA != rB {
			return rA - rB
		}
		if a.Name > b.Name {
			return -1
		} else if a.Name < b.Name {
			return 1
		}
		return 0
	})
	return result
}

func getEnabledProviders(cfg *config.Config) []models.ModelProvider {
	var providers []models.ModelProvider
	for providerID, provider := range cfg.Providers {
		if !provider.Disabled {
			providers = append(providers, providerID)
		}
	}
	slices.SortFunc(providers, func(a, b models.ModelProvider) int {
		rA := models.ProviderPopularity[a]
		rB := models.ProviderPopularity[b]
		if rA == 0 {
			rA = 999
		}
		if rB == 0 {
			rB = 999
		}
		return rA - rB
	})
	return providers
}




