package prompt

// PlannerSectionNomik returns the expandable Planner section describing
// extended Nomik (codebase knowledge graph) usage for the Planner.
// Loaded on-demand via expand_prompt_section("nomik") when the Planner
// is decomposing a project that touches an existing codebase and needs
// to understand structure before assigning tasks.
func PlannerSectionNomik() string {
	return `# Nomik (Codebase Knowledge Graph) — Extended Planner Guide

Nomik builds a Neo4j graph of the project's code (functions, classes,
imports, call edges) from AST analysis. It's exposed via ` + "`act codebase`" + `
commands. Use it BEFORE decomposing a project that touches an existing
codebase — never after, when you've already painted yourself into a corner.

## Commands worth knowing

- ` + "`act codebase onboard`" + ` — high-level architecture summary. Run ONCE
  at the start of decomposition for any non-greenfield project.
- ` + "`act codebase communities`" + ` — functional clusters / module
  boundaries. Use this to find which file groups are tightly coupled and
  must be edited together (sequence them, don't parallelize).
- ` + "`act codebase rules`" + ` — architecture rule violations (cross-module
  imports, circular deps, god files). If a project starts with a lot of
  violations, factor cleanup into your task plan instead of letting the
  swarm pile more on top.
- ` + "`act codebase impact <symbol>`" + ` — blast radius of changing a
  function/class. Useful when the user asks for "a small change" to a
  function that turns out to have 200 callers.

## Routing decisions Nomik should inform

1. **Sequencing.** If communities shows two clusters that share a
   handful of files, those tasks must run sequentially or coordinate
   via file claims. Never parallelize them.
2. **Capability matching.** If a cluster is mostly TypeScript+React,
   route to frontend_dev. If mostly Express+SQL, backend_dev.
3. **Refactor scope.** If rules shows N violations clustered in one
   module, scope a single refactor task instead of spreading fixes
   across feature work.

## Failure modes

- If ` + "`act codebase`" + ` returns "Nomik disabled" or an error, the project
  hasn't enabled the graph. Don't block on it — fall back to file-level
  reasoning (grep, glob, read by the swarm).
- Nomik does NOT know about runtime behavior, only static structure.
  Don't ask it about race conditions or perf hotspots.`
}
