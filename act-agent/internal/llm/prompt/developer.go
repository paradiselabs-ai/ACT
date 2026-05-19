package prompt

import (
	"fmt"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/models"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/spil"
)

// DeveloperPrompt returns the system prompt for the default Developer swarm role.
// Role identity content lives in act-agent/internal/spil/prompts/developer.spil.
func DeveloperPrompt(_ models.ModelProvider) string {
	identity := swarmIdentity("Developer", "General-purpose full-stack development.")
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s",
		identity,
		spil.Body("developer"),
		actCLICommands("developer"),
		swarmWorkflow(),
		coordinationConstraints("developer"),
		getEnvironmentInfo())
}
