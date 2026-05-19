package prompt

import (
	"fmt"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/models"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/spil"
)

// FrontendDevPrompt returns the system prompt for the Frontend Developer swarm role.
// Role identity content lives in act-agent/internal/spil/prompts/frontend_dev.spil.
func FrontendDevPrompt(_ models.ModelProvider) string {
	identity := swarmIdentity("Frontend Developer", "UI/UX implementation, component architecture, and frontend systems.")
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s",
		identity,
		spil.Body("frontend_dev"),
		actCLICommands("frontend_dev"),
		swarmWorkflow(),
		coordinationConstraints("frontend_dev"),
		getEnvironmentInfo())
}
