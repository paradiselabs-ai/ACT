package logs

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/layout"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
)

type DetailComponent interface {
	tea.Model
	layout.Sizeable
	layout.Bindings
}

type detailCmp struct {
	width, height int
	hOffset       int
	currentLog    logging.LogMessage
	viewport      viewport.Model
}

func (i *detailCmp) Init() tea.Cmd {
	messages := logging.List()
	if len(messages) == 0 {
		return nil
	}
	i.currentLog = messages[0]
	i.hOffset = 0
	i.updateContent()
	return nil
}

func (i *detailCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SelectedLogMsg:
		if msg.ID != i.currentLog.ID {
			i.currentLog = logging.LogMessage(msg)
			i.hOffset = 0
			i.updateContent()
		}
		return i, nil

	case tea.KeyPressMsg:
		k := msg.String()
		switch k {
		case "right", "l":
			i.hOffset += 4
			i.updateContent()
			return i, nil
		case "left", "h":
			i.hOffset -= 4
			if i.hOffset < 0 {
				i.hOffset = 0
			}
			i.updateContent()
			return i, nil
		case "home":
			i.hOffset = 0
			i.updateContent()
			return i, nil
		}
	}

	var cmd tea.Cmd
	i.viewport, cmd = i.viewport.Update(msg)
	return i, cmd
}

func (i *detailCmp) updateContent() {
	var content strings.Builder
	t := theme.CurrentTheme()
	baseStyle := styles.BaseStyle().Background(t.Background())

	if i.currentLog.ID == "" {
		i.viewport.SetContent(baseStyle.Foreground(t.TextMuted()).Render("No log entry selected"))
		return
	}

	// Format Level Badge
	levelBg := t.Info()
	switch strings.ToLower(i.currentLog.Level) {
	case "warn", "warning":
		levelBg = t.Warning()
	case "error", "err":
		levelBg = t.Error()
	case "debug":
		levelBg = t.Primary()
	}

	badge := baseStyle.Background(levelBg).Foreground(t.Background()).Bold(true).Padding(0, 1).Render(strings.ToUpper(i.currentLog.Level))
	timeStr := baseStyle.Foreground(t.TextMuted()).Render(i.currentLog.Time.Format("2006-01-02 15:04:05"))
	idStr := baseStyle.Foreground(t.TextMuted()).Render("ID: " + i.currentLog.ID)

	scrollInfo := ""
	if i.hOffset > 0 {
		scrollInfo = baseStyle.Foreground(t.Warning()).Bold(true).Render(fmt.Sprintf(" ← [H-Scroll +%d cols]", i.hOffset))
	}

	header := badge + "   " + timeStr + "   " + idStr + scrollInfo
	content.WriteString(header)
	content.WriteString("\n\n")

	// Message
	msgHeader := baseStyle.Bold(true).Foreground(t.Primary()).Render("MESSAGE:")
	content.WriteString(msgHeader)
	content.WriteString("\n")

	msgStr := i.currentLog.Message
	if i.hOffset > 0 {
		if len(msgStr) > i.hOffset {
			msgStr = "..." + msgStr[i.hOffset:]
		} else {
			msgStr = ""
		}
		content.WriteString(baseStyle.Foreground(t.Text()).Padding(0, 2).Render(msgStr))
	} else {
		w := i.width - 4
		if w < 15 {
			w = 15
		}
		msgStyle := baseStyle.Foreground(t.Text()).Width(w)
		content.WriteString(msgStyle.Render("  " + msgStr))
	}
	content.WriteString("\n\n")

	// Attributes section
	if len(i.currentLog.Attributes) > 0 {
		attrHeader := baseStyle.Bold(true).Foreground(t.Primary()).Render("ATTRIBUTES:")
		content.WriteString(attrHeader)
		content.WriteString("\n")

		keyStyle := baseStyle.Foreground(t.Primary()).Bold(true)
		valStyle := baseStyle.Foreground(t.Text())

		for _, attr := range i.currentLog.Attributes {
			val := attr.Value
			if i.hOffset > 0 {
				if len(val) > i.hOffset {
					val = "..." + val[i.hOffset:]
				} else {
					val = ""
				}
				attrLine := fmt.Sprintf("  %-16s %s", keyStyle.Render(attr.Key+":"), valStyle.Render(val))
				content.WriteString(attrLine)
				content.WriteString("\n")
			} else {
				kStr := keyStyle.Render("  " + attr.Key + ": ")
				valW := i.width - lipgloss.Width(kStr) - 2
				vStr := valStyle.Render(val)
				if valW > 15 {
					vStr = baseStyle.Foreground(t.Text()).Width(valW).Render(val)
				}
				content.WriteString(kStr)
				content.WriteString(vStr)
				content.WriteString("\n")
			}
		}
	}

	i.viewport.SetContent(content.String())
}

func (i *detailCmp) View() tea.View {
	t := theme.CurrentTheme()
	return tea.NewView(styles.ForceReplaceBackgroundWithLipgloss(i.viewport.View(), t.Background()))
}

func (i *detailCmp) GetSize() (int, int) {
	return i.width, i.height
}

func (i *detailCmp) SetSize(width int, height int) tea.Cmd {
	i.width = width
	i.height = height
	i.viewport.SetWidth(i.width)
	i.viewport.SetHeight(i.height)
	i.updateContent()
	return nil
}

func (i *detailCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(i.viewport.KeyMap)
}

func NewLogsDetails() DetailComponent {
	return &detailCmp{
		viewport: viewport.New(viewport.WithWidth(0), viewport.WithHeight(0)),
	}
}
