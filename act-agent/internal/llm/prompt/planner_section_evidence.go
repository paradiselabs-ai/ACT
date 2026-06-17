package prompt

// PlannerSectionEvidenceRouting returns the expandable Planner section
// describing evidence-based routing via PVM. Loaded on-demand by the
// Planner via expand_prompt_section("evidence_routing") when it needs
// to choose between roles or check whether a similar task has been
// done before.
func PlannerSectionEvidenceRouting() string {
	return `# Evidence-Based Routing (PVM)

PVM (PAIRed Vector Minutes) is the coordination memory: every task event
(creation, assignment, validation, failure) is recorded. You use it to pick the
swarm composition backed by what has actually worked, not intuition.

## You are handed the evidence — you don't fetch it

When a project brief is accepted, the orchestrator injects a
"## Routing evidence from past projects" block into your first BUILD turn. Every
line is confidence-labeled. It contains:
- similar past projects — their swarm composition, pass rate, and kickback count;
- per-role track records — pass rate per role across all projects;
- role-pair history — how role combinations (e.g. frontend_dev + backend_dev) fared together.

Read it before choosing your role mix. Example lines:
- "3×developer — 95% pass, 1 kickback (high signal: 41 tasks)" → trust it.
- "frontend_dev + backend_dev — 55% pass, 4 kickbacks (low signal: 3 tasks)" → weak hint only.

## How to read the signal labels

- high signal (≥10 samples): trust it; let it drive the composition.
- moderate signal (≥5): a real lean, but keep your own judgment.
- low signal (<5): a weak hint only. Do NOT override a clear capability-tag
  match on low-signal evidence — a fresh install shows low signal everywhere,
  so reason from first principles until history accrues.
- At comparable pass rates, the composition with fewer kickbacks is the cheaper bet.

## Digging deeper (optional, secondary)

The injected block is your primary evidence. To inspect a SPECIFIC past
situation (e.g. how auth middleware was wired before), do a raw pattern lookup
with the act_cli tool: ` + "`{\"subcommand\":\"pvm\",\"args\":[\"search\",\"<query>\"]}`" + `.
That returns the closest past event snippets by similarity — raw text, NOT
scored outcomes — so treat it as background reading, not routing proof.
(ACP backend: ` + "`act-tier1-planner pvm search \"<query>\"`" + ` via Bash — same result.)

## What NOT to do

- Don't route on a hunch when the injected evidence is high-signal.
- Don't treat low-signal lines as proof.
- Don't override a clear capability-tag match unless high-signal evidence
  clearly contradicts it.`
}
