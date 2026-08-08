package completions

import (
	"testing"
)

func TestSlashCommandsContextGroup(t *testing.T) {
	provider := NewSlashCommandsContextGroup()

	if provider.GetId() != "slash" {
		t.Errorf("expected id 'slash', got %q", provider.GetId())
	}

	// Test getting all entries with empty query
	entries, err := provider.GetChildEntries("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) < 10 {
		t.Errorf("expected at least 10 slash command entries, got %d", len(entries))
	}

	// Test searching for specific commands
	planEntries, err := provider.GetChildEntries("/plan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(planEntries) == 0 {
		t.Errorf("expected matches for '/plan', got 0")
	}

	firstValue := planEntries[0].GetValue()
	if firstValue != "/plan " {
		t.Errorf("expected completion value '/plan ', got %q", firstValue)
	}

	// Test query without slash prefix (e.g. "help")
	helpEntries, err := provider.GetChildEntries("help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundHelp := false
	for _, entry := range helpEntries {
		if entry.GetValue() == "/help " {
			foundHelp = true
			break
		}
	}

	if !foundHelp {
		t.Errorf("expected '/help ' in results for query 'help'")
	}
}
