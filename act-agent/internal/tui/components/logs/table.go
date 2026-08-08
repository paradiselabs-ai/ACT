package logs

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/pubsub"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/layout"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/styles"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/theme"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/util"
)

type TableComponent interface {
	tea.Model
	layout.Sizeable
	layout.Bindings
}

type tableCmp struct {
	table table.Model
}

type SelectedLogMsg logging.LogMessage

func (i *tableCmp) Init() tea.Cmd {
	i.setRows()
	return nil
}

func (i *tableCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg.(type) {
	case pubsub.Event[logging.LogMessage]:
		i.setRows()
		return i, nil
	}
	prevSelectedRow := i.table.SelectedRow()
	t, cmd := i.table.Update(msg)
	cmds = append(cmds, cmd)
	i.table = t
	selectedRow := i.table.SelectedRow()
	if selectedRow != nil {
		if prevSelectedRow == nil || selectedRow[0] == prevSelectedRow[0] {
			var log logging.LogMessage
			for _, row := range logging.List() {
				if row.ID == selectedRow[0] {
					log = row
					break
				}
			}
			if log.ID != "" {
				cmds = append(cmds, util.CmdHandler(SelectedLogMsg(log)))
			}
		}
	}
	return i, tea.Batch(cmds...)
}

func (i *tableCmp) View() tea.View {
	t := theme.CurrentTheme()
	defaultStyles := table.DefaultStyles()
	defaultStyles.Selected = defaultStyles.Selected.Foreground(t.Primary())
	i.table.SetStyles(defaultStyles)
	return tea.NewView(styles.ForceReplaceBackgroundWithLipgloss(i.table.View(), t.Background()))
}

func (i *tableCmp) GetSize() (int, int) {
	return i.table.Width(), i.table.Height()
}

func (i *tableCmp) SetSize(width int, height int) tea.Cmd {
	i.table.SetWidth(width)
	i.table.SetHeight(height)
	columns := i.table.Columns()
	if len(columns) == 5 {
		columns[0].Width = 10 // ID
		columns[1].Width = 10 // Time
		columns[2].Width = 7  // Level

		rem := width - 10 - 10 - 7 - 8
		if rem < 30 {
			rem = 30
		}
		msgW := rem * 55 / 100
		attrW := rem - msgW

		columns[3].Width = msgW  // Message
		columns[4].Width = attrW // Attributes
	}
	i.table.SetColumns(columns)
	return nil
}

func (i *tableCmp) BindingKeys() []key.Binding {
	return layout.KeyMapToSlice(i.table.KeyMap)
}

func (i *tableCmp) setRows() {
	rows := []table.Row{}

	logs := logging.List()
	slices.SortFunc(logs, func(a, b logging.LogMessage) int {
		if a.Time.Before(b.Time) {
			return 1
		}
		if a.Time.After(b.Time) {
			return -1
		}
		return 0
	})

	for _, log := range logs {
		shortID := log.ID
		if len(shortID) > 8 {
			shortID = shortID[len(shortID)-8:]
		}

		attrSummary := ""
		if len(log.Attributes) > 0 {
			var parts []string
			for _, a := range log.Attributes {
				parts = append(parts, a.Key+"="+a.Value)
			}
			attrSummary = strings.Join(parts, " ")
		}

		row := table.Row{
			shortID,
			log.Time.Format("15:04:05"),
			strings.ToUpper(log.Level),
			log.Message,
			attrSummary,
		}
		rows = append(rows, row)
	}
	i.table.SetRows(rows)
}

func NewLogsTable() TableComponent {
	columns := []table.Column{
		{Title: "ID", Width: 10},
		{Title: "Time", Width: 10},
		{Title: "Level", Width: 8},
		{Title: "Message", Width: 40},
		{Title: "Attributes", Width: 25},
	}

	tableModel := table.New(
		table.WithColumns(columns),
	)
	tableModel.Focus()
	return &tableCmp{
		table: tableModel,
	}
}
