package dialog

import (
	"testing"
)

func TestRenameDialog_Validation(t *testing.T) {
	// 1. Valid title validation
	if err := validateTitle("New Session Name"); err != nil {
		t.Errorf("expected nil error for valid title, got %v", err)
	}

	// 2. Empty title validation
	if err := validateTitle("   "); err == nil {
		t.Error("expected error for whitespace title, got nil")
	}

	if err := validateTitle(""); err == nil {
		t.Error("expected error for empty title, got nil")
	}
}
