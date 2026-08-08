// Package dialog — overlay components rendered via PlaceOverlay.
//
// IMPORTANT: See STYLING_GUIDE.md for the full explanation of why
// lipgloss.JoinHorizontal/JoinVertical cause black strips in overlays
// and the correct manual-padding pattern to avoid them.
//
// help.go is the REFERENCE IMPLEMENTATION of the correct pattern.
package dialog

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
)

type helpCmp struct {
	width  int
	height int
	keys   []key.Binding
}

func (h *helpCmp) Init() tea.Cmd {
	return nil
}

func (h *helpCmp) SetBindings(k []key.Binding) {
	h.keys = k
}

func (h *helpCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = 90
		h.height = msg.Height
	}
	return h, nil
}

func removeDuplicateBindings(bindings []key.Binding) []key.Binding {
	seen := make(map[string]struct{})
	result := make([]key.Binding, 0, len(bindings))

	// Process bindings in reverse order
	for i := len(bindings) - 1; i >= 0; i-- {
		b := bindings[i]
		k := strings.Join(b.Keys(), " ")
		if _, ok := seen[k]; ok {
			// duplicate, skip
			continue
		}
		seen[k] = struct{}{}
		// Add to the beginning of result to maintain original order
		result = append([]key.Binding{b}, result...)
	}

	return result
}

func (h *helpCmp) render() string {
	t := theme.CurrentTheme()
	bg := t.Background()

	helpKeyStyle := styles.Bold().
		Background(bg).
		Foreground(t.Text()).
		Padding(0, 1, 0, 0)

	helpDescStyle := styles.Regular().
		Background(bg).
		Foreground(t.TextMuted())

	padStyle := lipgloss.NewStyle().Background(bg)

	bindings := removeDuplicateBindings(h.keys)

	rows := 10
	// Build columns: each column is a pair of (key, desc) arrays
	type colPair struct {
		keys  []string
		descs []string
		keyW  int
		descW int
	}
	var cols []colPair
	for i := 0; i < len(bindings); i += rows {
		cp := colPair{}
		for j := i; j < min(i+rows, len(bindings)); j++ {
			k := helpKeyStyle.Render(bindings[j].Help().Key)
			d := helpDescStyle.Render(bindings[j].Help().Desc)
			cp.keys = append(cp.keys, k)
			cp.descs = append(cp.descs, d)
			if kw := lipgloss.Width(k); kw > cp.keyW {
				cp.keyW = kw
			}
			if dw := lipgloss.Width(d); dw > cp.descW {
				cp.descW = dw
			}
		}
		cols = append(cols, cp)
	}

	// Build each row line-by-line with explicit padding
	var lines []string
	for row := 0; row < rows; row++ {
		var line strings.Builder
		for ci, cp := range cols {
			if ci > 0 {
				line.WriteString(padStyle.Render("   "))
			}
			if row < len(cp.keys) {
				k := cp.keys[row]
				kPad := cp.keyW - lipgloss.Width(k)
				line.WriteString(k)
				if kPad > 0 {
					line.WriteString(padStyle.Render(strings.Repeat(" ", kPad)))
				}
				d := cp.descs[row]
				dPad := cp.descW - lipgloss.Width(d)
				line.WriteString(d)
				if dPad > 0 {
					line.WriteString(padStyle.Render(strings.Repeat(" ", dPad)))
				}
			} else {
				// Empty row in this column — fill with background
				line.WriteString(padStyle.Render(strings.Repeat(" ", cp.keyW+cp.descW)))
			}
		}
		lines = append(lines, line.String())
	}

	return strings.Join(lines, "\n")
}

// padLine pads a rendered line to exactly targetWidth using bg-styled spaces.
func padLine(line string, targetWidth int, padStyle lipgloss.Style) string {
	w := lipgloss.Width(line)
	if w < targetWidth {
		return line + padStyle.Render(strings.Repeat(" ", targetWidth-w))
	}
	return line
}

func (h *helpCmp) View() tea.View {
	t := theme.CurrentTheme()
	bg := t.Background()
	padStyle := lipgloss.NewStyle().Background(bg)

	titleStyle := styles.BaseStyle().
		Background(bg).
		Bold(true).
		Foreground(t.Primary())

	mutedStyle := styles.BaseStyle().
		Background(bg).
		Foreground(t.TextMuted())

	// Keyboard Shortcuts
	shortcutsContent := h.render()

	// Palette Commands — build each line manually
	palEntries := []struct{ left, right string }{
		{"act-agent:status   System overview", "act-agent:validation  Pending queue"},
		{"act-agent:log      Recent ChronLog", "act-agent:conflicts   File conflicts"},
		{"act-agent:tasks    Unverified task graph", "act-agent:swarm       Swarm overview"},
	}

	var palLines []string
	for _, e := range palEntries {
		left := mutedStyle.Render("  " + e.left)
		mid := padStyle.Render("    ")
		right := mutedStyle.Render(e.right)
		palLines = append(palLines, left+mid+right)
	}

	// Assemble all body lines
	var bodyLines []string
	bodyLines = append(bodyLines, titleStyle.Render("Keyboard Shortcuts"))
	bodyLines = append(bodyLines, "")
	bodyLines = append(bodyLines, strings.Split(shortcutsContent, "\n")...)
	bodyLines = append(bodyLines, "")
	bodyLines = append(bodyLines, titleStyle.Render("Palette Commands (`:` in prompt)"))
	bodyLines = append(bodyLines, "")
	bodyLines = append(bodyLines, palLines...)

	// Find max width of all lines
	maxW := 0
	for _, l := range bodyLines {
		if w := lipgloss.Width(l); w > maxW {
			maxW = w
		}
	}

	// Pad every line to maxW with background-styled spaces
	for i, l := range bodyLines {
		bodyLines[i] = padLine(l, maxW, padStyle)
	}

	body := strings.Join(bodyLines, "\n")

	boxStyle := styles.BaseStyle().
		Background(bg).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocused()).
		BorderBackground(bg)

	rendered := boxStyle.Render(body)

	// Final pass: pad every rendered line (including border lines) to the
	// full box width, ensuring no bare spaces leak through.
	finalLines := strings.Split(rendered, "\n")
	finalW := 0
	for _, l := range finalLines {
		if w := lipgloss.Width(l); w > finalW {
			finalW = w
		}
	}
	for i, l := range finalLines {
		finalLines[i] = padLine(l, finalW, padStyle)
	}
	rendered = strings.Join(finalLines, "\n")

	return tea.NewView(rendered)
}

type ToggleHelpMsg struct{}
type ShowLogsMsg struct{}
type ShowModelDialogMsg struct {
	Role string
}
type StartCompactSessionMsg struct{}

type HelpCmp interface {
	tea.Model
	SetBindings([]key.Binding)
}

func NewHelpCmp() HelpCmp {
	return &helpCmp{}
}
