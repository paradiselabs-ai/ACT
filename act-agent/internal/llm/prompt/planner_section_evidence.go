package prompt

// PlannerSectionEvidenceRouting returns the expandable Planner section
// describing evidence-based routing via PVM. Loaded on-demand by the
// Planner via expand_prompt_section("evidence_routing") when it needs
// to choose between roles or check whether a similar task has been
// done before.
func PlannerSectionEvidenceRouting() string {
	return `# Evidence-Based Routing (PVM)

PVM (PAIRed Vector Minutes) is the coordination memory. Every task event
(creation, progress, validation, failure) is embedded and indexed. Use it
to make routing decisions backed by what has actually worked before, not
intuition.

## When to invoke

- Before assigning a task whose role isn't obvious — search for who has
  succeeded on similar work.
- When two roles could plausibly do the same task — pick the one with
  the stronger track record on that domain.
- If you recieve information that a task fails — search for prior failures of the same shape so
  you can identify the root cause class.

## How to invoke

Call the act_cli tool: ` + "`{\"subcommand\":\"pvm\",\"args\":[\"search\",\"<query>\"]}`" + `.
It returns top matches by semantic similarity, each with the agent ID, task
title, outcome, and timestamp.
(ACP backend: ` + "`act-tier1-planner pvm search \"<query>\"`" + ` via Bash — same result.)

## How to read the results

- Look for outcome=completed AND high success_criteria scores together.
  Either alone is weak signal.
- Recency matters less than role+domain match. A 2-week-old success on
  the same domain beats a 1-day-old success on a different one.
- If every match is for a single agent ID, that's a specialization
  signal — prefer that agent.
- If results are mixed (some passes, some fails) on the same role, the
  task class itself is hard. Add a tighter @success_criteria and
  consider sequencing it earlier.

## What NOT to do

- Don't search for every task. Routine work doesn't need evidence.
- Don't pick a role just because it has the most history — pick the one
  whose history matches the current task domain.
- Don't override a clear capability tag match with PVM results unless
  the PVM evidence is overwhelming.`
}
