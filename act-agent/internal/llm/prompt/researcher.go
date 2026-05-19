package prompt

import (
	"fmt"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/models"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/spil"
)

// ResearcherPrompt returns the system prompt for the Researcher swarm role.
// Role identity content lives in act-agent/internal/spil/prompts/researcher.spil.
func ResearcherPrompt(_ models.ModelProvider) string {
	identity := swarmIdentity("Researcher", "Information gathering, analysis, and written deliverables.")
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s",
		identity,
		spil.Body("researcher"),
		actCLICommands("researcher"),
		swarmWorkflow(),
		coordinationConstraints("researcher"),
		getEnvironmentInfo())
}
