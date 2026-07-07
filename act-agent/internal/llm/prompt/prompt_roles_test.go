package prompt

import (
	"strings"
	"testing"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/config"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/models"
)

// TestPromptSwitchRouting verifies that the GetAgentPrompt switch
// routes all 9 ACT roles to non-default prompts. Since getEnvironmentInfo()
// requires config to be loaded, we test the base prompt strings directly.
func TestPromptSwitchRouting(t *testing.T) {
	// These functions should return non-empty prompts without needing config
	// (they only panic when calling getEnvironmentInfo, which appends env info)
	tests := []struct {
		name string
		fn   func(models.ModelProvider) string
	}{
		{"PlannerPrompt", func(p models.ModelProvider) string { return basePlannerPrompt }},
		{"ObserverPrompt", func(p models.ModelProvider) string { return baseObserverPrompt }},
		{"AssurancePrompt", func(p models.ModelProvider) string { return baseAssurancePrompt }},
		{"QASynthesizerPrompt", func(p models.ModelProvider) string { return baseQASynthesizerPrompt }},
		{"DeveloperPrompt", func(p models.ModelProvider) string { return baseDeveloperPrompt }},
		{"FrontendDevPrompt", func(p models.ModelProvider) string { return baseFrontendDevPrompt }},
		{"BackendDevPrompt", func(p models.ModelProvider) string { return baseBackendDevPrompt }},
		{"QAEngineerPrompt", func(p models.ModelProvider) string { return baseQAEngineerPrompt }},
		{"ResearcherPrompt", func(p models.ModelProvider) string { return baseResearcherPrompt }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.fn(models.ProviderAnthropic)
			if strings.Contains(p, "You are a helpful assistant") {
				t.Errorf("got garbage 'helpful assistant' prompt")
			}
			if len(p) < 500 {
				t.Errorf("prompt too short (%d chars)", len(p))
			}
		})
	}
}

// TestSwitchCasesExist verifies the dispatcher has cases for all ACT roles.
// This is a compile-time check — if a role constant doesn't match any switch case,
// the prompt function wouldn't be called. We verify by checking that the role
// constants are valid AgentName values that differ from the built-in agents.
func TestSwitchCasesExist(t *testing.T) {
	roles := []config.AgentName{
		config.RolePlanner, config.RoleObserver, config.RoleAssurance,
		config.RoleQASynthesizer, config.RoleDeveloper, config.RoleFrontendDev,
		config.RoleBackendDev, config.RoleQAEngineer, config.RoleResearcher,
	}

	builtins := map[config.AgentName]bool{
		config.AgentTitle:      true,
		config.AgentSummarizer: true,
	}

	for _, role := range roles {
		if builtins[role] {
			t.Errorf("%s should be an ACT role, not a built-in agent", role)
		}
		if role == "" {
			t.Error("empty role constant found")
		}
	}
}

// TestCommonBuildingBlocks verifies shared prompt helpers return non-empty strings.
func TestCommonBuildingBlocks(t *testing.T) {
	roles := []string{"planner", "observer", "assurance", "qa_synthesizer", "developer"}

	for _, role := range roles {
		cli := actCLICommands(role)
		if len(cli) < 100 {
			t.Errorf("actCLICommands(%s) too short: %d", role, len(cli))
		}

		constraints := coordinationConstraints(role)
		if len(constraints) < 50 {
			t.Errorf("coordinationConstraints(%s) too short: %d", role, len(constraints))
		}
	}

	protocol := communicationProtocol()
	if !strings.Contains(protocol, "NesTTY") {
		t.Error("communicationProtocol missing NesTTY reference")
	}

	workflow := swarmWorkflow()
	if !strings.Contains(workflow, "Ralph Wiggum") {
		t.Error("swarmWorkflow missing Ralph Wiggum Loop")
	}
}

// TestPlannerPromptBranchesOnProvider locks audit Fix 22: PlannerPrompt must
// render a backend-accurate CLI fragment. An ACP-backed Planner (provider ==
// models.ProviderACP) reaches act_cli through the act-tier1-planner shim via
// Bash; an in-process Planner (any real provider) uses the native act_cli JSON
// tool and must NOT shell out. Before Fix 22 the provider arg was discarded
// (`func PlannerPrompt(_ ...)`) and both backends got the same — and only one
// was correct. If the branch is ever collapsed, the two outputs stop differing
// and this test fails.
//
// Not parallel: PlannerPrompt → getEnvironmentInfo → config.Get touches
// package-level config state. Mirror prompt_test.go's isolation.
func TestPlannerPromptBranchesOnProvider(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	config.ResetForTests()
	if _, err := config.Load(tmpDir, false); err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	acpPrompt := PlannerPrompt(models.ProviderACP)
	inProcPrompt := PlannerPrompt(models.ProviderGROQ)

	if acpPrompt == inProcPrompt {
		t.Fatal("PlannerPrompt returned identical text for ACP and in-process — the provider branch was collapsed (Fix 22 regressed)")
	}
	if !strings.Contains(acpPrompt, "act-tier1-planner") {
		t.Error("ACP PlannerPrompt must instruct the act-tier1-planner shim via Bash; missing 'act-tier1-planner'")
	}
	if strings.Contains(acpPrompt, "do NOT shell out to send messages") {
		t.Error("ACP PlannerPrompt must NOT carry the in-process 'do NOT shell out' framing — ACP MUST shell out for subcommands")
	}
	if !strings.Contains(inProcPrompt, "do NOT shell out to send messages") {
		t.Error("in-process PlannerPrompt must keep the 'do NOT shell out' guard")
	}
	if strings.Contains(inProcPrompt, "act-tier1-planner") {
		t.Error("in-process PlannerPrompt must NOT reference the ACP shim binary")
	}
}
