package agent

import (
	"context"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/history"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/tools"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/lsp"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/permission"
)

// DeveloperTools returns the full toolbox used by building Tier 2 swarm
// agents (developer, frontend_dev, backend_dev, qa_engineer). These roles
// do the actual building so they get bash, edit/patch/write, view, grep,
// glob, ls, fetch, and sourcegraph. researcher gets ResearcherTools instead.
func DeveloperTools(
	permissions permission.Service,
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
		}, otherTools...,
	)
}

// ResearcherTools returns the read-only toolbox for the researcher swarm
// role. Its prompt says "analysis, not code" — least privilege makes the
// tools match: search/read/fetch plus MCP and diagnostics, no bash, no
// edit/write/patch.
func ResearcherTools(
	permissions permission.Service,
	lspClients map[string]*lsp.Client,
) []tools.BaseTool {
	ctx := context.Background()
	otherTools := GetMcpTools(ctx, permissions)
	if len(lspClients) > 0 {
		otherTools = append(otherTools, tools.NewDiagnosticsTool(lspClients))
	}
	return append(
		[]tools.BaseTool{
			tools.NewFetchTool(permissions),
			tools.NewGlobTool(),
			tools.NewGrepTool(),
			tools.NewLsTool(),
			tools.NewSourcegraphTool(),
			tools.NewViewTool(lspClients),
		}, otherTools...,
	)
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
// Planner uses act_cli (subcommand-whitelisted) plus expand_prompt_section
// (to pull deeper guidance on demand without bloating the base prompt).
// Per KI-02: no raw bash — Planner's surface is narrowed to the act CLI to
// prevent exploratory shell commands (ls, go version, view main.go etc.)
// that waste tokens and turn latency.
func PlannerTools(permissions permission.Service) []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewActCLITool("planner", permissions),
		tools.NewExpandPromptSectionTool(),
	}
}

// ObserverTools returns the minimum tool set for the Observer agent.
// Observer uses act_cli with status/log/graph/context subcommands for
// anomaly detection. No bash — Observer has never needed shell access.
func ObserverTools(permissions permission.Service) []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewActCLITool("observer", permissions),
	}
}

// AssuranceTools returns the minimum tool set for the Assurance agent.
// Assurance uses act_cli (validation queue, log, status) plus view/grep
// to read submitted work against @success_criteria.
func AssuranceTools(permissions permission.Service, lspClients map[string]*lsp.Client) []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewActCLITool("assurance", permissions),
		tools.NewViewTool(lspClients),
		tools.NewGrepTool(),
	}
}

// QASynthesizerTools returns the minimum tool set for the QA/Synthesizer agent.
// QA uses act_cli (validation, log, status, codebase) plus view/grep to read
// validated outputs while assembling the final deliverable.
func QASynthesizerTools(permissions permission.Service, lspClients map[string]*lsp.Client) []tools.BaseTool {
	return []tools.BaseTool{
		tools.NewActCLITool("qa_synthesizer", permissions),
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
		return DeveloperTools(permissions, history, lspClients)
	}
}
