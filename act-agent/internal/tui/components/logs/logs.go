package logs

import (
	tea "github.com/charmbracelet/bubbletea"
)

// logsTable is a stub for the logs table component.
// The original package was missing from the NesTTY branch.
type logsTable struct{}

func (l logsTable) Init() tea.Cmd                           { return nil }
func (l logsTable) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return l, nil }
func (l logsTable) View() string                            { return "" }

func NewLogsTable() tea.Model { return logsTable{} }

// logsDetails is a stub for the logs detail component.
type logsDetails struct{}

func (l logsDetails) Init() tea.Cmd                           { return nil }
func (l logsDetails) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return l, nil }
func (l logsDetails) View() string                            { return "" }

func NewLogsDetails() tea.Model { return logsDetails{} }
