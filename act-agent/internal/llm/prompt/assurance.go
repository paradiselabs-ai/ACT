package prompt

import (
	"fmt"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/models"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/spil"
)

// AssurancePrompt returns the system prompt for the Assurance role.
// Role identity content lives in act-agent/internal/spil/prompts/assurance.spil.
func AssurancePrompt(_ models.ModelProvider) string {
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n%s",
		spil.Body("assurance"),
		actCLICommands("assurance"),
		communicationProtocol(),
		coordinationConstraints("assurance"),
		getEnvironmentInfo())
}
