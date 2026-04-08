package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/prompt"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
)

// ExpandPromptSectionToolName is the public tool name. Kept short because
// it appears in every Tier 1 tool catalog the LLM sees.
const ExpandPromptSectionToolName = "expand_prompt_section"

// expandSectionRegistry maps section names to their content provider.
// Adding a new section is a one-line entry — no schema changes.
var expandSectionRegistry = map[string]func() string{
	"evidence_routing":  prompt.PlannerSectionEvidenceRouting,
	"success_criteria":  prompt.PlannerSectionSuccessCriteria,
	"nomik":             prompt.PlannerSectionNomik,
	"validation":        prompt.PlannerSectionValidation,
	"examples":          prompt.PlannerSectionExamples,
}

// expandPromptSectionTool returns reference material on demand. The
// Planner's base prompt stays small; deeper guidance is pulled only
// when the Planner actually needs it. This is the receiver-side
// equivalent of context tiering — most turns never call this tool.
type expandPromptSectionTool struct{}

// ExpandPromptSectionParams is the JSON input shape.
type ExpandPromptSectionParams struct {
	Section string `json:"section"`
}

// NewExpandPromptSectionTool constructs the tool. No dependencies — the
// section content is compiled in via the prompt package.
func NewExpandPromptSectionTool() BaseTool {
	return &expandPromptSectionTool{}
}

func (t *expandPromptSectionTool) Info() ToolInfo {
	names := make([]string, 0, len(expandSectionRegistry))
	for k := range expandSectionRegistry {
		names = append(names, k)
	}
	sort.Strings(names)

	return ToolInfo{
		Name: ExpandPromptSectionToolName,
		Description: `Returns deeper reference guidance for the Planner on a specific topic. Use this only when you actually need it — most turns don't.

WHEN TO USE:
- "evidence_routing" — before assigning a task whose role isn't obvious, or when you want PVM-backed routing rationale
- "success_criteria" — when writing or repairing @success_criteria for a CREATE_TASK
- "nomik" — at decomposition start for an existing codebase, or when sequencing tightly-coupled tasks
- "validation" — when reacting to an Assurance failure or a stuck validation queue
- "examples" — when you need to confirm the exact shape of a CREATE_TASK or PROJECT_BRIEF directive

WHEN NOT TO USE:
- Routine task creation. The skeleton prompt has everything needed for normal flow.
- Status checks. Use act CLI commands instead.
- Multiple sections at once. Pull one, act on it, pull another only if still needed.

The returned content is reference text — read it, apply it, then proceed with your normal directive output.`,
		Parameters: map[string]any{
			"section": map[string]any{
				"type":        "string",
				"description": "Name of the section to expand. One of: " + fmt.Sprint(names),
				"enum":        names,
			},
		},
		Required: []string{"section"},
	}
}

func (t *expandPromptSectionTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	sessionID, messageID := GetContextValues(ctx)
	var params ExpandPromptSectionParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		logging.Error("expand_prompt_section.parse_error",
			"session_id", sessionID,
			"message_id", messageID,
			"input", call.Input,
			"error", err.Error(),
		)
		return NewTextErrorResponse("Failed to parse expand_prompt_section parameters: " + err.Error()), nil
	}
	if params.Section == "" {
		logging.Warn("expand_prompt_section.missing_section",
			"session_id", sessionID,
			"message_id", messageID,
		)
		return NewTextErrorResponse("section parameter is required"), nil
	}

	provider, ok := expandSectionRegistry[params.Section]
	if !ok {
		names := make([]string, 0, len(expandSectionRegistry))
		for k := range expandSectionRegistry {
			names = append(names, k)
		}
		sort.Strings(names)
		logging.Warn("expand_prompt_section.unknown",
			"session_id", sessionID,
			"section", params.Section,
			"available", names,
		)
		return NewTextErrorResponse(fmt.Sprintf("unknown section %q. Available: %v", params.Section, names)), nil
	}

	content := provider()
	if content == "" {
		logging.Warn("expand_prompt_section.empty",
			"session_id", sessionID,
			"section", params.Section,
		)
		return NewTextErrorResponse(fmt.Sprintf("section %q is currently empty", params.Section)), nil
	}
	logging.Info("expand_prompt_section.served",
		"session_id", sessionID,
		"message_id", messageID,
		"section", params.Section,
		"bytes", len(content),
	)
	return NewTextResponse(content), nil
}
