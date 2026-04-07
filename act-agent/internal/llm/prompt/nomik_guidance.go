package prompt

// NomikGuidance returns role-specific guidance on when to use Nomik
// (the codebase knowledge graph, exposed via `act codebase` commands).
//
// Each role gets a "## When to use the codebase graph" section that tells
// the agent the concrete triggers for invoking `act codebase impact|rules|
// communities|onboard`. The goal is to make agents use these tools
// proactively without being asked.
//
// If the role doesn't benefit from codebase context, returns an empty string.
func NomikGuidance(role string) string {
	switch role {

	case "planner":
		return "## Codebase graph (Nomik, optional)\n" +
			"- `act codebase onboard` once at decomposition start to learn existing architecture\n" +
			"- `act codebase communities` before assigning tasks to detect file groups that must be sequenced (not parallelized)"

	case "observer":
		return `## When to use the codebase graph
You have access to a Nomik-powered codebase knowledge graph via these commands:
- ` + "`act codebase rules`" + ` — architecture rule violations (e.g. cross-module imports, circular deps)
- ` + "`act codebase impact <symbol>`" + ` — blast radius of changing a function/class

USE THEM:
- During your monitoring loop, run ` + "`act codebase rules`" + ` periodically (every few cycles, not every cycle). If new violations appear, escalate to the Planner.
- When you see an agent modifying a widely-used symbol, run ` + "`act codebase impact <symbol>`" + ` to estimate blast radius. If the impact is large (many callers), flag it as a potential bottleneck or risk.
- If Nomik returns "disabled" or an error, skip the check silently and continue monitoring other signals.`

	case "assurance":
		return `## When to use the codebase graph
You have access to a Nomik-powered codebase knowledge graph via these commands:
- ` + "`act codebase impact <symbol>`" + ` — blast radius of changing a function/class
- ` + "`act codebase rules`" + ` — architecture rule violations introduced by recent changes

USE THEM:
- BEFORE approving any task that modified code, run ` + "`act codebase impact <symbol>`" + ` on the changed symbols. If the agent claims a "small change" but the impact is enormous, that's a red flag — score the validation lower and request a tighter scope.
- Run ` + "`act codebase rules`" + ` to check for new architecture violations introduced by the change. Architecture violations are an automatic deduction in your overall score.
- If Nomik returns "disabled" or an error, fall back to manual code review only.`

	case "qa_synthesizer", "qa":
		return `## When to use the codebase graph
You have access to a Nomik-powered codebase knowledge graph via these commands:
- ` + "`act codebase communities`" + ` — functional clusters / module boundaries
- ` + "`act codebase onboard`" + ` — overall architecture summary

USE THEM:
- BEFORE assembling validated outputs into a deliverable, run ` + "`act codebase communities`" + ` to find integration points between modules. This tells you where to look for interface mismatches between independently-produced outputs.
- AFTER assembly, run ` + "`act codebase onboard`" + ` for a final sanity check that the deliverable matches the architecture the Planner intended.
- If Nomik returns "disabled" or an error, do the integration check manually using the validated outputs alone.`

	case "developer":
		return `## When to use the codebase graph
You have access to a Nomik-powered codebase knowledge graph via these commands:
- ` + "`act codebase impact <symbol>`" + ` — blast radius of changing a function/class
- ` + "`act codebase onboard`" + ` — high-level architecture overview

USE THEM:
- BEFORE refactoring or modifying any function/class that might be widely used, run ` + "`act codebase impact <symbol>`" + `. If the impact is large, narrow your change or coordinate with peer agents.
- At the START of any complex task, run ` + "`act codebase onboard`" + ` for a quick architecture refresher. This is faster than reading through the codebase manually.
- If Nomik returns "disabled" or an error, fall back to ` + "`act codebase`" + ` skipped — proceed using grep/glob/read.`

	case "frontend_dev":
		return `## When to use the codebase graph
You have access to a Nomik-powered codebase knowledge graph via these commands:
- ` + "`act codebase communities`" + ` — UI component clusters and what depends on what
- ` + "`act codebase impact <symbol>`" + ` — blast radius of changing a component or shared util

USE THEM:
- BEFORE changing a shared component (button, layout, theme), run ` + "`act codebase impact <component-name>`" + ` to see every page using it. Don't break others.
- Run ` + "`act codebase communities`" + ` to understand which UI components are tightly coupled — these typically should be modified together.
- If Nomik returns "disabled" or an error, fall back to grep for component imports manually.`

	case "backend_dev":
		return `## When to use the codebase graph
You have access to a Nomik-powered codebase knowledge graph via these commands:
- ` + "`act codebase impact <symbol>`" + ` — blast radius of changing an API endpoint, model, or service
- ` + "`act codebase rules`" + ` — architecture rule violations (e.g. controllers calling models directly)

USE THEM:
- BEFORE changing any API endpoint signature or shared service method, run ` + "`act codebase impact <method-name>`" + `. If clients depend on the old signature, you must version the change or coordinate.
- Run ` + "`act codebase rules`" + ` after adding new endpoints to confirm you didn't violate any conventions (e.g. endpoint naming, middleware ordering).
- If Nomik returns "disabled" or an error, manually grep for callers and check existing endpoints for conventions.`

	case "qa_engineer":
		return `## When to use the codebase graph
You have access to a Nomik-powered codebase knowledge graph via these commands:
- ` + "`act codebase impact <symbol>`" + ` — which tests are affected when a function changes
- ` + "`act codebase communities`" + ` — module clusters for scoping integration tests

USE THEM:
- When writing tests for a function, run ` + "`act codebase impact <function>`" + ` to find existing tests that depend on it. Don't duplicate coverage; extend instead.
- When scoping integration tests, run ` + "`act codebase communities`" + ` to identify which modules should be tested together as a unit.
- If Nomik returns "disabled" or an error, fall back to grep for existing test files manually.`

	case "researcher":
		return `## When to use the codebase graph
You have FULL access to the Nomik-powered codebase knowledge graph:
- ` + "`act codebase onboard`" + ` — architecture overview
- ` + "`act codebase impact <symbol>`" + ` — blast radius
- ` + "`act codebase rules`" + ` — architecture violations
- ` + "`act codebase communities`" + ` — functional clusters

USE THEM AGGRESSIVELY:
- These tools are your primary investigation surface. Run ` + "`act codebase onboard`" + ` first on any new project to ground your analysis.
- Use ` + "`act codebase impact`" + ` and ` + "`act codebase communities`" + ` to map dependencies before recommending refactors.
- Use ` + "`act codebase rules`" + ` to identify pre-existing architectural debt that should be addressed.
- If Nomik returns "disabled" or an error, the project hasn't enabled the graph — fall back to file-level analysis (grep, read, glob).`

	default:
		return ""
	}
}
