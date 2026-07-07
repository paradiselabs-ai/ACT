package agent

import (
	"testing"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
)

// TestResearcherToolsReadOnly locks the least-privilege contract: the
// researcher role must never hold a tool that mutates files or runs shell
// commands (its prompt says "analysis, not code" — the tools must agree).
func TestResearcherToolsReadOnly(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	config.ResetForTests()
	if _, err := config.Load(tmp, false); err != nil {
		t.Fatalf("config load: %v", err)
	}
	t.Cleanup(config.ResetForTests)

	banned := map[string]bool{"bash": true, "edit": true, "write": true, "patch": true}
	seen := map[string]bool{}
	for _, tool := range ResearcherTools(nil, nil) {
		name := tool.Info().Name
		seen[name] = true
		if banned[name] {
			t.Errorf("researcher toolset contains mutating tool %q", name)
		}
	}
	for _, want := range []string{"glob", "grep", "ls", "view", "fetch", "sourcegraph"} {
		if !seen[want] {
			t.Errorf("researcher toolset missing read tool %q", want)
		}
	}
}
