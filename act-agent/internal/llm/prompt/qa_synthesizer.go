package prompt

import (
	"fmt"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/models"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/spil"
)

// QASynthesizerPrompt returns the system prompt for the QA/Synthesizer role.
// Role identity content lives in act-agent/internal/spil/prompts/qa_synthesizer.spil.
func QASynthesizerPrompt(_ models.ModelProvider) string {
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n%s",
		spil.Body("qa_synthesizer"),
		actCLICommands("qa_synthesizer"),
		communicationProtocol(),
		coordinationConstraints("qa_synthesizer"),
		getEnvironmentInfo())
}
