package dialog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOnboardingNextStepReachesSave locks the step machine after the Nomik
// step was removed. The previous wizard wrote config in stepNomik, which was
// skipped when Nomik was unavailable — so save had to migrate to "whichever
// step lands on stepSave". This verifies both arrival paths:
//   - claude in PATH:    Tier2Backend -> Save
//   - claude not in PATH: Tier2Backend skipped, Tier2Models -> Save
func TestOnboardingNextStepReachesSave(t *testing.T) {
	withClaude := &onboardingCmp{claudeAvailable: true}
	if got := withClaude.nextStep(stepTier2Models); got != stepTier2Backend {
		t.Errorf("claude available: nextStep(Tier2Models) = %d, want stepTier2Backend(%d)", got, stepTier2Backend)
	}
	if got := withClaude.nextStep(stepTier2Backend); got != stepSave {
		t.Errorf("claude available: nextStep(Tier2Backend) = %d, want stepSave(%d)", got, stepSave)
	}

	noClaude := &onboardingCmp{claudeAvailable: false}
	if got := noClaude.nextStep(stepTier2Models); got != stepSave {
		t.Errorf("no claude: nextStep(Tier2Models) = %d, want stepSave(%d) — save would never fire", got, stepSave)
	}
}

// TestOnboardingWriteConfigNoNomik confirms the generated ~/.act.json carries
// no nomik key after removal, and that writeConfig still produces a valid
// config. HOME is redirected to a temp dir so the real user config is untouched.
func TestOnboardingWriteConfigNoNomik(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	role := func(key string) onboardingRole {
		return onboardingRole{
			AgentKey: key,
			Options:  []onboardingModel{{Provider: "anthropic", Model: "claude-sonnet-4-20250514"}},
			Selected: 0,
		}
	}
	o := &onboardingCmp{
		tier1: []onboardingRole{role("planner"), role("observer"), role("assurance"), role("qa_synthesizer")},
		tier2: []onboardingRole{role("developer")},
		// tier2Backends shorter than tier2 is tolerated by writeConfig (defaults to act-agent)
	}

	if err := o.writeConfig(); err != nil {
		t.Fatalf("writeConfig: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, ".act.json"))
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	if strings.Contains(strings.ToLower(string(data)), "nomik") {
		t.Errorf("written config still mentions nomik:\n%s", data)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("written config is not valid JSON: %v", err)
	}
	if _, ok := parsed["nomik"]; ok {
		t.Errorf("written config has a nomik key, want none")
	}
	if _, ok := parsed["agents"]; !ok {
		t.Errorf("written config missing agents block")
	}
}
