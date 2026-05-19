package prompt

import (
	"fmt"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/models"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/spil"
)

// QAEngineerPrompt returns the system prompt for the QA Engineer swarm role.
// Role identity content lives in act-agent/internal/spil/prompts/qa_engineer.spil.
func QAEngineerPrompt(_ models.ModelProvider) string {
	identity := swarmIdentity("QA Engineer", "Test strategy, test implementation, and quality verification.")
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s",
		identity,
		spil.Body("qa_engineer"),
		actCLICommands("qa_engineer"),
		swarmWorkflow(),
		coordinationConstraints("qa_engineer"),
		getEnvironmentInfo())
}
