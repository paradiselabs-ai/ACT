package prompt

import (
	"fmt"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/models"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/spil"
)

// PlannerPrompt returns the system prompt for the Planner role.
//
// Tight by design: Planner runs on free-tier providers (Groq, OpenRouter free)
// where the per-minute token budget is the binding constraint. Every line here
// has to earn its place. If you find yourself wanting to add more guidance,
// consider whether the Planner can derive it from the existing rules + tool
// outputs instead.
//
// As of experiment/spil-internalize: role-identity content is in
// act-agent/internal/spil/prompts/planner.spil (embedded via go:embed).
// Dynamic blocks (actCLICommands, coordinationConstraints, getEnvironmentInfo)
// remain in Go because they depend on runtime config + env state.
func PlannerPrompt(_ models.ModelProvider) string {
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
		spil.Body("planner"),
		actCLICommands("planner"),
		coordinationConstraints("planner"),
		getEnvironmentInfo())
}
