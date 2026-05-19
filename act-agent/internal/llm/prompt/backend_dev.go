package prompt

import (
	"fmt"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/models"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/spil"
)

// BackendDevPrompt returns the system prompt for the Backend Developer swarm role.
// Role identity content lives in act-agent/internal/spil/prompts/backend_dev.spil.
func BackendDevPrompt(_ models.ModelProvider) string {
	identity := swarmIdentity("Backend Developer", "APIs, databases, server architecture, and backend systems.")
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s",
		identity,
		spil.Body("backend_dev"),
		actCLICommands("backend_dev"),
		swarmWorkflow(),
		coordinationConstraints("backend_dev"),
		getEnvironmentInfo())
}
