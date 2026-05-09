package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderAgentsMd_ContainsBriefFields(t *testing.T) {
	brief := &ProjectBrief{
		Description:     "A simple word counter CLI.",
		TechStack:       "Node.js, no dependencies.",
		Constraints:     "Single file. No external packages.",
		SuccessCriteria: "Prints count of words in input.",
		AgentsInvolved:  []string{"backend_dev", "qa_engineer"},
	}
	out := renderAgentsMd("wordcount", brief)

	for _, want := range []string{
		"# AGENTS.md",
		"wordcount",
		"A simple word counter CLI.",
		"Node.js, no dependencies.",
		"Single file. No external packages.",
		"backend_dev, qa_engineer",
		"Validation",
		"95%",
		userNotesMarker,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered AGENTS.md missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderAgentsMd_DefaultsForEmptyOptionals(t *testing.T) {
	brief := &ProjectBrief{
		Description: "x",
		TechStack:   "y",
	}
	out := renderAgentsMd("p", brief)
	if !strings.Contains(out, "None specified.") {
		t.Errorf("empty constraints should render as 'None specified.'\n%s", out)
	}
	if !strings.Contains(out, "developer") {
		t.Errorf("empty agentsInvolved should default to 'developer'\n%s", out)
	}
}

func TestWriteAgentsMd_WritesBothFiles(t *testing.T) {
	dir := t.TempDir()
	brief := &ProjectBrief{Description: "d", TechStack: "t", AgentsInvolved: []string{"developer"}}
	if err := writeAgentsMd(dir, "proj", brief); err != nil {
		t.Fatalf("writeAgentsMd: %v", err)
	}
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if !strings.Contains(string(data), "# AGENTS.md") {
			t.Errorf("%s missing header", name)
		}
	}
}

func TestWriteAgentsMd_PreservesUserNotesBelowMarker(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	initial := "# Old content\n\nstale body\n\n" + userNotesMarker + "\n\n## My notes\nimportant hand-written text\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	brief := &ProjectBrief{Description: "new desc", TechStack: "new stack"}
	if err := writeAgentsMd(dir, "proj", brief); err != nil {
		t.Fatalf("writeAgentsMd: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if strings.Contains(got, "stale body") {
		t.Errorf("ACT-managed section should have been overwritten; got:\n%s", got)
	}
	if !strings.Contains(got, "new desc") {
		t.Errorf("new description missing; got:\n%s", got)
	}
	if !strings.Contains(got, "important hand-written text") {
		t.Errorf("user notes were dropped; got:\n%s", got)
	}
	if strings.Count(got, userNotesMarker) != 1 {
		t.Errorf("marker should appear exactly once; got %d\n%s", strings.Count(got, userNotesMarker), got)
	}
}

func TestWriteAgentsMd_TreatsPreMarkerFileAsUserContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(path, []byte("user wrote this before ACT existed"), 0o644); err != nil {
		t.Fatal(err)
	}

	brief := &ProjectBrief{Description: "d", TechStack: "t"}
	if err := writeAgentsMd(dir, "proj", brief); err != nil {
		t.Fatalf("writeAgentsMd: %v", err)
	}
	data, _ := os.ReadFile(path)
	got := string(data)
	if !strings.Contains(got, "user wrote this before ACT existed") {
		t.Errorf("pre-existing file without marker should be preserved as user content; got:\n%s", got)
	}
}

func TestWriteAgentsMd_ErrorsOnEmptyDir(t *testing.T) {
	brief := &ProjectBrief{Description: "d"}
	if err := writeAgentsMd("", "p", brief); err == nil {
		t.Error("expected error on empty projectDir")
	}
}
