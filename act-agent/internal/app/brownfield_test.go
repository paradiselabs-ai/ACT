package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasExistingCode(t *testing.T) {
	// Empty dir, no git → false.
	empty := t.TempDir()
	if hasExistingCode(empty) {
		t.Errorf("hasExistingCode(empty dir) = true, want false")
	}

	// Manifest present → true.
	withManifest := t.TempDir()
	if err := os.WriteFile(filepath.Join(withManifest, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasExistingCode(withManifest) {
		t.Errorf("hasExistingCode(dir with go.mod) = false, want true")
	}

	// A non-manifest file alone → false (no recognized manifest, no git).
	other := t.TempDir()
	if err := os.WriteFile(filepath.Join(other, "notes.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if hasExistingCode(other) {
		t.Errorf("hasExistingCode(dir with only notes.txt) = true, want false")
	}
}

func TestParseHeadlessResult(t *testing.T) {
	// Happy path — indented envelope like App.agentOutput produces.
	good := []byte(`{
  "agent_id": "onboard-researcher",
  "status": "completed",
  "result": "This is the analysis.",
  "timestamp": "2026-06-06T00:00:00Z"
}`)
	got, err := parseHeadlessResult(good)
	if err != nil {
		t.Fatalf("parseHeadlessResult(good) error: %v", err)
	}
	if got != "This is the analysis." {
		t.Errorf("result = %q, want %q", got, "This is the analysis.")
	}

	// Non-completed status → error.
	if _, err := parseHeadlessResult([]byte(`{"status":"error","result":"boom"}`)); err == nil {
		t.Errorf("expected error on status=error")
	}

	// Empty result → error.
	if _, err := parseHeadlessResult([]byte(`{"status":"completed","result":"  "}`)); err == nil {
		t.Errorf("expected error on empty result")
	}

	// Not JSON → error.
	if _, err := parseHeadlessResult([]byte(`not json at all`)); err == nil {
		t.Errorf("expected error on non-JSON output")
	}
}
