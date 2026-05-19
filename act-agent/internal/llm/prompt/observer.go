package prompt

import (
	"fmt"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/models"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/spil"
)

// ObserverPrompt returns the system prompt for the Observer role.
// Role identity content lives in act-agent/internal/spil/prompts/observer.spil.
func ObserverPrompt(_ models.ModelProvider) string {
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n%s",
		spil.Body("observer"),
		actCLICommands("observer"),
		communicationProtocol(),
		coordinationConstraints("observer"),
		getEnvironmentInfo())
}
