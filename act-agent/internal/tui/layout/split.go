package layout

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
)

type SplitPaneLayout interface {
	tea.Model
	Sizeable
	Bindings
	SetLeftPanel(panel Container) tea.Cmd
	SetRightPanel(panel Container) tea.Cmd
	SetBottomPanel(panel Container) tea.Cmd

	ClearLeftPanel() tea.Cmd
	ClearRightPanel() tea.Cmd
	ClearBottomPanel() tea.Cmd

	// N-column API
	SetPanel(index int, panel Container) tea.Cmd
	ClearPanel(index int) tea.Cmd
	PanelCount() int
}

type splitPaneLayout struct {
	width         int
	height        int
	ratios        []float64
	verticalRatio float64

	panels      []Container
	bottomPanel Container
}

type SplitPaneOption func(*splitPaneLayout)

func (s *splitPaneLayout) Init() tea.Cmd {
	var cmds []tea.Cmd

	for _, panel := range s.panels {
		if panel != nil {
			cmds = append(cmds, panel.Init())
		}
	}

	if s.bottomPanel != nil {
		cmds = append(cmds, s.bottomPanel.Init())
	}

	return tea.Batch(cmds...)
}

func (s *splitPaneLayout) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return s, s.SetSize(msg.Width, msg.Height)
	}

	for i, panel := range s.panels {
		if panel != nil {
			u, cmd := panel.Update(msg)
			s.panels[i] = u.(Container)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	if s.bottomPanel != nil {
		u, cmd := s.bottomPanel.Update(msg)
		s.bottomPanel = u.(Container)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return s, tea.Batch(cmds...)
}

func (s *splitPaneLayout) View() tea.View {
	var topSection string

	// Collect non-nil panel views
	var panelViews []string
	for _, panel := range s.panels {
		if panel != nil {
			panelViews = append(panelViews, panel.View().Content)
		}
	}

	if len(panelViews) > 1 {
		topSection = lipgloss.JoinHorizontal(lipgloss.Top, panelViews...)
	} else if len(panelViews) == 1 {
		topSection = panelViews[0]
	}

	var finalView string

	if s.bottomPanel != nil && topSection != "" {
		bottomView := s.bottomPanel.View().Content
		finalView = lipgloss.JoinVertical(lipgloss.Left, topSection, bottomView)
	} else if s.bottomPanel != nil {
		finalView = s.bottomPanel.View().Content
	} else {
		finalView = topSection
	}

	if finalView != "" {
		t := theme.CurrentTheme()
		style := lipgloss.NewStyle().
			Width(s.width).
			Height(s.height).
			Background(t.Background())
		return tea.NewView(style.Render(finalView))
	}

	return tea.NewView(finalView)
}

func (s *splitPaneLayout) SetSize(width, height int) tea.Cmd {
	s.width = width
	s.height = height

	var topHeight, bottomHeight int
	if s.bottomPanel != nil {
		topHeight = int(float64(height) * s.verticalRatio)
		bottomHeight = height - topHeight
	} else {
		topHeight = height
		bottomHeight = 0
	}

	// Count non-nil panels for width distribution
	nonNilCount := 0
	for _, panel := range s.panels {
		if panel != nil {
			nonNilCount++
		}
	}

	// Calculate widths based on ratios
	panelWidths := s.distributePanelWidths(width, nonNilCount)

	var cmds []tea.Cmd
	widthIdx := 0
	for i, panel := range s.panels {
		if panel != nil {
			w := panelWidths[i]
			cmd := panel.SetSize(w, topHeight)
			cmds = append(cmds, cmd)
			widthIdx++
		}
	}

	if s.bottomPanel != nil {
		cmd := s.bottomPanel.SetSize(width, bottomHeight)
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// distributePanelWidths distributes the total width among panels according
// to their ratios, with remainder correction on the last non-nil panel.
func (s *splitPaneLayout) distributePanelWidths(totalWidth int, nonNilCount int) []int {
	widths := make([]int, len(s.panels))

	if nonNilCount <= 1 {
		// Single or zero panels — give full width to whichever is non-nil
		for i, panel := range s.panels {
			if panel != nil {
				widths[i] = totalWidth
			}
		}
		return widths
	}

	// Sum ratios of non-nil panels for normalization
	var ratioSum float64
	for i, panel := range s.panels {
		if panel != nil && i < len(s.ratios) {
			ratioSum += s.ratios[i]
		}
	}
	if ratioSum == 0 {
		ratioSum = 1.0
	}

	// Assign widths proportionally
	allocated := 0
	lastNonNilIdx := -1
	for i, panel := range s.panels {
		if panel != nil {
			ratio := 0.0
			if i < len(s.ratios) {
				ratio = s.ratios[i]
			}
			w := int(float64(totalWidth) * (ratio / ratioSum))
			widths[i] = w
			allocated += w
			lastNonNilIdx = i
		}
	}

	// Give remainder to the last non-nil panel
	if lastNonNilIdx >= 0 && allocated != totalWidth {
		widths[lastNonNilIdx] += totalWidth - allocated
	}

	return widths
}

func (s *splitPaneLayout) GetSize() (int, int) {
	return s.width, s.height
}

// --- N-column API ---

func (s *splitPaneLayout) SetPanel(index int, panel Container) tea.Cmd {
	s.ensurePanelSlot(index)
	s.panels[index] = panel
	if s.width > 0 && s.height > 0 {
		return s.SetSize(s.width, s.height)
	}
	return nil
}

func (s *splitPaneLayout) ClearPanel(index int) tea.Cmd {
	if index < len(s.panels) {
		s.panels[index] = nil
	}
	if s.width > 0 && s.height > 0 {
		return s.SetSize(s.width, s.height)
	}
	return nil
}

func (s *splitPaneLayout) PanelCount() int {
	count := 0
	for _, p := range s.panels {
		if p != nil {
			count++
		}
	}
	return count
}

func (s *splitPaneLayout) ensurePanelSlot(index int) {
	for len(s.panels) <= index {
		s.panels = append(s.panels, nil)
	}
	for len(s.ratios) <= index {
		// Default ratio for new slots
		s.ratios = append(s.ratios, 0.3)
	}
}

// --- Backward-compatible left/right wrappers ---

func (s *splitPaneLayout) SetLeftPanel(panel Container) tea.Cmd {
	return s.SetPanel(0, panel)
}

func (s *splitPaneLayout) SetRightPanel(panel Container) tea.Cmd {
	return s.SetPanel(1, panel)
}

func (s *splitPaneLayout) SetBottomPanel(panel Container) tea.Cmd {
	s.bottomPanel = panel
	if s.width > 0 && s.height > 0 {
		return s.SetSize(s.width, s.height)
	}
	return nil
}

func (s *splitPaneLayout) ClearLeftPanel() tea.Cmd {
	return s.ClearPanel(0)
}

func (s *splitPaneLayout) ClearRightPanel() tea.Cmd {
	return s.ClearPanel(1)
}

func (s *splitPaneLayout) ClearBottomPanel() tea.Cmd {
	s.bottomPanel = nil
	if s.width > 0 && s.height > 0 {
		return s.SetSize(s.width, s.height)
	}
	return nil
}

func (s *splitPaneLayout) BindingKeys() []key.Binding {
	keys := []key.Binding{}
	for _, panel := range s.panels {
		if panel != nil {
			if b, ok := panel.(Bindings); ok {
				keys = append(keys, b.BindingKeys()...)
			}
		}
	}
	if s.bottomPanel != nil {
		if b, ok := s.bottomPanel.(Bindings); ok {
			keys = append(keys, b.BindingKeys()...)
		}
	}
	return keys
}

func NewSplitPane(options ...SplitPaneOption) SplitPaneLayout {

	layout := &splitPaneLayout{
		panels:        make([]Container, 2),
		ratios:        []float64{0.7, 0.3},
		verticalRatio: 0.9, // Default 90% for top section, 10% for bottom
	}
	for _, option := range options {
		option(layout)
	}
	return layout
}

func WithLeftPanel(panel Container) SplitPaneOption {
	return func(s *splitPaneLayout) {
		s.ensurePanelSlot(0)
		s.panels[0] = panel
	}
}

func WithRightPanel(panel Container) SplitPaneOption {
	return func(s *splitPaneLayout) {
		s.ensurePanelSlot(1)
		s.panels[1] = panel
	}
}

func WithRatio(ratio float64) SplitPaneOption {
	return func(s *splitPaneLayout) {
		s.ratios = []float64{ratio, 1.0 - ratio}
	}
}

func WithBottomPanel(panel Container) SplitPaneOption {
	return func(s *splitPaneLayout) {
		s.bottomPanel = panel
	}
}

func WithVerticalRatio(ratio float64) SplitPaneOption {
	return func(s *splitPaneLayout) {
		s.verticalRatio = ratio
	}
}
