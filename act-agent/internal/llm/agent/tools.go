package agent

import (
	"context"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/history"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/tools"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/lsp"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/message"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/permission"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/session"
)

// DeveloperTools returns the full toolbox used by Tier 2 swarm agents
// (developer, frontend_dev, backend_dev, qa_engineer, researcher). These
// roles do the actual building so they get bash, edit/patch/write, view,
// grep, glob, ls, fetch, sourcegraph, and the sub-agent dispatcher.
func DeveloperTools(
	permissions permission.Service,
	sessions session.Service,
	messages message.Service,
	history history.Service,
	lspClients map[string]*lsp.Client,
) []tools.BaseTool {
	ctx := context.Background()
	otherTools := GetMcpTools(ctx, permissions)
	if len(lspClients) > 0 {
		otherTools = append(otherTools, tools.NewDiagnosticsTool(lspClients))
	}
	return append(
		[]tools.BaseTool{
			tools.NewBashTool(permissions),
			tools.NewEditTool(lspClients, permissions, history),
			tools.NewFetchTool(permissions),
			tools.NewGlobTool(),
			tools.NewGrepTool(),
			tools.NewLsTool(),
			tools.NewSourcegraphTool(),
			tools.NewViewTool(lspClients),
			tools.NewPatchTool(lspClients, permissions, history),
			tools.NewWriteTool(lspClients, permissions, history),
			NewAgentTool(sessions, messages, lspClients),
		}, otherTools...,
	)
}

func TaskAgentTools(lspClients map[string]*lsp.Client) []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewGlobTool(),
		tools.NewGrepTool(),
		tools.NewLsTool(),
		tools.NewSourcegraphTool(),
		tools.NewViewTool(lspClients),
	}
}

// ─── ACT Tier 1 per-role tool subsets ─────────────────────────────────────────
//
// Why these exist: every tool ships its full Description + JSONSchema in every
// LLM request. With the full DeveloperTools roster (11 tools), each request
// is ~16-18K tokens of tool definitions BEFORE any system prompt or user
// message — which blows free-tier rate limits (Groq's 12K TPM cap) on the
// first turn.
//
// The Tier 1 ACT roles only need a tiny slice of the toolbox:
//   - Planner / Observer talk to the system exclusively via the `act` CLI,
//     which is invoked through bash. They never read or write code directly.
//   - Assurance / QA Synthesizer also use `act` CLI but additionally need to
//     read submitted task outputs and codebase fragments for validation /
//     synthesis — so they get view + grep on top of bash.
//
// Tier 2 swarm agents (developer, frontend_dev, etc.) still get the full
// DeveloperTools roster — they're the ones actually building things.

// PlannerTools returns the minimum tool set for the Planner agent.
// Planner needs bash (for `act` CLI commands) plus expand_prompt_section
// (to pull deeper guidance on demand without bloating the base prompt).
func PlannerTools(permissions permission.Service) []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewBashTool(permissions),
		tools.NewExpandPromptSectionTool(),
	}
}

// ObserverTools returns the minimum tool set for the Observer agent.
// Observer only needs bash to run `act log`, `act graph`, `act status`, etc.
func ObserverTools(permissions permission.Service) []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewBashTool(permissions),
	}
}

// AssuranceTools returns the minimum tool set for the Assurance agent.
// Assurance needs bash for `act validation queue` and view/grep to read
// the submitted work being validated against @success_criteria.
func AssuranceTools(permissions permission.Service, lspClients map[string]*lsp.Client) []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewBashTool(permissions),
		tools.NewViewTool(lspClients),
		tools.NewGrepTool(),
	}
}

// QASynthesizerTools returns the minimum tool set for the QA/Synthesizer agent.
// QA needs bash for `act` CLI and view/grep to read validated outputs while
// assembling the final deliverable.
func QASynthesizerTools(permissions permission.Service, lspClients map[string]*lsp.Client) []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewBashTool(permissions),
		tools.NewViewTool(lspClients),
		tools.NewGrepTool(),
	}
}

// Tier1ToolsForRole returns the per-role tool subset for a Tier 1 agent.
// Falls back to DeveloperTools (the full roster) for any role that isn't
// one of the four canonical Tier 1 roles — that's the right behavior for
// Tier 2 swarm agents (developer, frontend_dev, etc.) which DO need the
// full toolbox to build things.
func Tier1ToolsForRole(
	role string,
	permissions permission.Service,
	sessions session.Service,
	messages message.Service,
	history history.Service,
	lspClients map[string]*lsp.Client,
) []tools.BaseTool {
	switch role {
	case "planner":
		return PlannerTools(permissions)
	case "observer":
		return ObserverTools(permissions)
	case "assurance":
		return AssuranceTools(permissions, lspClients)
	case "qa_synthesizer":
		return QASynthesizerTools(permissions, lspClients)
	default:
		return DeveloperTools(permissions, sessions, messages, history, lspClients)
	}
}
