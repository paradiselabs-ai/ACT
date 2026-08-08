package dialog

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/layout"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/util"
)

// ShowInfoDialogMsg displays title and content in a styled modal dialog overlay
type ShowInfoDialogMsg struct {
	Title   string
	Content string
}

// CloseInfoDialogMsg closes the info dialog overlay
type CloseInfoDialogMsg struct{}

type InfoDialog interface {
	tea.Model
	layout.Bindings
	SetContent(title, content string)
	SetSize(width, height int)
}

type infoDialogCmp struct {
	title   string
	content string
	width   int
	height  int
}

func (i *infoDialogCmp) Init() tea.Cmd {
	return nil
}

func (i *infoDialogCmp) SetContent(title, content string) {
	i.title = title
	i.content = content
}

func (i *infoDialogCmp) SetSize(width, height int) {
	i.width = width
	i.height = height
}

func (i *infoDialogCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "esc" || msg.String() == "enter" || msg.String() == "q" || msg.String() == "space" {
			return i, util.CmdHandler(CloseInfoDialogMsg{})
		}
	}
	return i, nil
}

func parseMarkdownBold(line string, t theme.Theme) string {
	baseStyle := styles.BaseStyle().Background(t.Background())
	for {
		start := strings.Index(line, "**")
		if start == -1 {
			break
		}
		end := strings.Index(line[start+2:], "**")
		if end == -1 {
			break
		}
		end += start + 2
		boldText := line[start+2 : end]
		styled := baseStyle.Bold(true).Foreground(t.Primary()).Render(boldText)
		line = line[:start] + styled + line[end+2:]
	}
	return line
}

func (i *infoDialogCmp) View() tea.View {
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle().Background(t.Background())
	padStyle := baseStyle.Background(t.Background())

	dlgWidth := 64
	if i.width > 0 && i.width < 70 {
		dlgWidth = i.width - 6
	}
	if dlgWidth < 35 {
		dlgWidth = 35
	}

	titleText := i.title
	if titleText == "" {
		titleText = "Information"
	}

	titleStyle := baseStyle.
		Bold(true).
		Foreground(t.Primary()).
		Background(t.Background())

	renderedTitle := titleStyle.Render(titleText)

	bodyLines := strings.Split(i.content, "\n")
	var formattedLines []string

	for idx, line := range bodyLines {
		l := strings.TrimSpace(line)

		// Skip duplicate header on first line if it matches titleText
		if idx == 0 || (idx == 1 && len(formattedLines) == 0) {
			headerClean := strings.TrimPrefix(strings.TrimPrefix(l, "## "), "# ")
			if strings.EqualFold(headerClean, titleText) {
				continue
			}
		}

		if strings.HasPrefix(l, "## ") {
			l = strings.TrimPrefix(l, "## ")
			l = parseMarkdownBold(l, t)
			lineStr := baseStyle.Bold(true).Foreground(t.Primary()).Render(l)
			formattedLines = append(formattedLines, lineStr)
			continue
		}
		if strings.HasPrefix(l, "# ") {
			l = strings.TrimPrefix(l, "# ")
			l = parseMarkdownBold(l, t)
			lineStr := baseStyle.Bold(true).Foreground(t.Primary()).Render(l)
			formattedLines = append(formattedLines, lineStr)
			continue
		}

		l = parseMarkdownBold(l, t)
		formattedLines = append(formattedLines, baseStyle.Foreground(t.Text()).Render(l))
	}

	padLine := func(line string, width int) string {
		w := lipgloss.Width(line)
		if w < width {
			return line + padStyle.Render(strings.Repeat(" ", width-w))
		}
		return line
	}

	// Calculate max width for manual padding
	maxW := lipgloss.Width(renderedTitle)
	for _, l := range formattedLines {
		if w := lipgloss.Width(l); w > maxW {
			maxW = w
		}
	}
	if maxW < dlgWidth {
		maxW = dlgWidth
	}

	var allLines []string
	allLines = append(allLines, padLine(renderedTitle, maxW))
	allLines = append(allLines, padLine("", maxW))

	for _, l := range formattedLines {
		allLines = append(allLines, padLine(l, maxW))
	}

	footerHint := baseStyle.Foreground(t.TextMuted()).Render("Press esc or enter to close")
	allLines = append(allLines, padLine("", maxW))
	allLines = append(allLines, padLine(footerHint, maxW))

	body := strings.Join(allLines, "\n")

	boxStyle := baseStyle.
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderBackground(t.Background()).
		BorderForeground(t.BorderFocused())

	rendered := boxStyle.Render(body)
	return tea.NewView(styles.ForceBackgroundOnAllLines(rendered, t.Background()))
}

func (i *infoDialogCmp) BindingKeys() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("esc", "enter", "q"), key.WithHelp("esc/enter", "close")),
	}
}

func NewInfoDialogCmp() InfoDialog {
	return &infoDialogCmp{}
}
