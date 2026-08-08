package dialog

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestInfoDialog(t *testing.T) {
	d := NewInfoDialogCmp()
	d.SetContent("Test Status", "System is running\nRole: planner")
	d.SetSize(80, 24)

	view := d.View()
	content := view.Content

	if !strings.Contains(content, "Test Status") {
		t.Errorf("expected view to contain title 'Test Status'")
	}
	if !strings.Contains(content, "System is running") {
		t.Errorf("expected view to contain content 'System is running'")
	}
	if !strings.Contains(content, "Press esc or enter to close") {
		t.Errorf("expected view to contain footer hint")
	}

	// Test Esc keypress closes dialog
	escMsg := tea.KeyPressMsg{
		Text: "esc",
	}
	_, cmd := d.Update(escMsg)
	if cmd == nil {
		t.Fatalf("expected Cmd on esc keypress")
	}
	msg := cmd()
	if _, ok := msg.(CloseInfoDialogMsg); !ok {
		t.Errorf("expected CloseInfoDialogMsg on esc keypress, got %T", msg)
	}
}
