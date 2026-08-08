package page

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/pubsub"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/components/logs"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/layout"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
)

var LogsPage PageID = "logs"

type LogPage interface {
	tea.Model
	layout.Sizeable
	layout.Bindings
}

type logsPage struct {
	width, height int
	table         layout.Container
	details       layout.Container
	focusedPane   int // 0 = table, 1 = details viewport
}

func padLine(line string, width int, padStyle lipgloss.Style) string {
	w := lipgloss.Width(line)
	if w < width {
		return line + padStyle.Render(strings.Repeat(" ", width-w))
	}
	return line
}

func (p *logsPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		return p, p.SetSize(msg.Width, msg.Height)

	case tea.KeyPressMsg:
		k := msg.String()
		if k == "tab" || k == "shift+tab" {
			p.focusedPane = (p.focusedPane + 1) % 2
			return p, nil
		}
		if k == "left" || k == "right" || k == "h" || k == "l" {
			details, cmd := p.details.Update(msg)
			cmds = append(cmds, cmd)
			p.details = details.(layout.Container)
			return p, tea.Batch(cmds...)
		}
	}

	// Always forward log events (pubsub) to table
	if _, ok := msg.(pubsub.Event[logging.LogMessage]); ok {
		table, cmd := p.table.Update(msg)
		cmds = append(cmds, cmd)
		p.table = table.(layout.Container)
		return p, tea.Batch(cmds...)
	}

	// Forward SelectedLogMsg to details
	if _, ok := msg.(logs.SelectedLogMsg); ok {
		details, cmd := p.details.Update(msg)
		cmds = append(cmds, cmd)
		p.details = details.(layout.Container)
		return p, tea.Batch(cmds...)
	}

	// Otherwise route keypresses to active focused pane
	if p.focusedPane == 0 {
		table, cmd := p.table.Update(msg)
		cmds = append(cmds, cmd)
		p.table = table.(layout.Container)
	} else {
		details, cmd := p.details.Update(msg)
		cmds = append(cmds, cmd)
		p.details = details.(layout.Container)
	}

	return p, tea.Batch(cmds...)
}

func (p *logsPage) View() tea.View {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle().Background(t.Background())
	padStyle := baseStyle.Background(t.Background())

	totalLogs := len(logging.List())

	leftTitle := baseStyle.Bold(true).Foreground(t.Primary()).Render(" SYSTEM LOGS ")
	countStr := baseStyle.Foreground(t.TextMuted()).Render(fmt.Sprintf("(%d entries) ", totalLogs))

	focusText := "[Table Focused]"
	if p.focusedPane == 1 {
		focusText = "[Details Focused]"
	}
	focusStr := baseStyle.Bold(true).Foreground(t.Info()).Render(focusText)

	left := leftTitle + countStr + focusStr

	right := baseStyle.Foreground(t.TextMuted()).Render("Tab: switch pane | ↑/↓/←/→: navigate | esc/q: chat ")

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)

	gapW := p.width - leftW - rightW
	if gapW < 2 {
		gapW = 2
	}

	headerLine := left + padStyle.Render(strings.Repeat(" ", gapW)) + right
	headerLine = padLine(headerLine, p.width, padStyle)

	// Format table and details view lines with strict line padding
	tableLines := strings.Split(p.table.View().Content, "\n")
	for i, l := range tableLines {
		tableLines[i] = padLine(l, p.width, padStyle)
	}

	detailsLines := strings.Split(p.details.View().Content, "\n")
	for i, l := range detailsLines {
		detailsLines[i] = padLine(l, p.width, padStyle)
	}

	var allLines []string
	allLines = append(allLines, headerLine)
	allLines = append(allLines, tableLines...)
	allLines = append(allLines, detailsLines...)

	rendered := strings.Join(allLines, "\n")
	bgSeq := padStyle.Render("")
	if bgSeq != "" {
		rendered = strings.ReplaceAll(rendered, "\x1b[0m", "\x1b[0m"+bgSeq)
	}

	return tea.NewView(rendered)
}

func (p *logsPage) BindingKeys() []key.Binding {
	return p.table.BindingKeys()
}

// GetSize implements LogPage.
func (p *logsPage) GetSize() (int, int) {
	return p.width, p.height
}

// SetSize implements LogPage.
func (p *logsPage) SetSize(width int, height int) tea.Cmd {
	p.width = width
	p.height = height
	availH := height - 1
	if availH < 4 {
		availH = 4
	}
	tableH := availH / 2
	detailsH := availH - tableH
	return tea.Batch(
		p.table.SetSize(width, tableH),
		p.details.SetSize(width, detailsH),
	)
}

func (p *logsPage) Init() tea.Cmd {
	return tea.Batch(
		p.table.Init(),
		p.details.Init(),
	)
}

func NewLogsPage() LogPage {
	return &logsPage{
		table:   layout.NewContainer(logs.NewLogsTable(), layout.WithBorderAll()),
		details: layout.NewContainer(logs.NewLogsDetails(), layout.WithBorderAll()),
	}
}
