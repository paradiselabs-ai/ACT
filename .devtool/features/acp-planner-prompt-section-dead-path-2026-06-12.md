---
id: "acp-planner-prompt-section-dead-path-2026-06-12"
status: "todo"
priority: "high"
assignee: null
dueDate: null
created: "2026-06-12T07:30:00.000Z"
modified: "2026-06-12T07:30:00.000Z"
completedAt: null
labels: ["prompts", "acp", "cli"]
order: "a2"
---
# ACP-backed Planner's `prompt-section` is a dead path — whitelisted + prompt-instructed, but no CLI branch exists

## Spec
Found by the 2026-06-12 architecture-flows rebuild (finding FT1-1, encoded gap-found in
`architecture-flows.json` meta.findings). The Planner act_cli whitelist grants `prompt-section`
(`internal/llm/tools/act_cli_whitelist.go`) and the ACP planner prompt instructs
`act-tier1-planner prompt-section <name>` — but `cli/act-cli.ts` has NO `prompt-section` dispatch
branch; the command falls through to "Unknown command" (act-cli.ts:~1288). In-process Planners are
fine (native `expand_prompt_section` tool); ACP-backed Planners (claude-code/antigravity) silently
lose all on-demand prompt sections (evidence_routing, success_criteria, validation, examples).

## Success Criteria
- `act-agent prompt-section validation` (via the shim) prints the section content from the same
  registry the in-process tool uses (`sections.go` / `SectionNames()`); unknown names error cleanly.
- An ACP-backed Planner can retrieve a section end-to-end (shim allowlist → act-cli.ts → output).
- Regression test: a test asserting every whitelist-granted subcommand has an act-cli.ts dispatch
  branch (this class of drift is exactly what it would catch).

## Constraints
- One source of truth for section content — the CLI must read/serve the same registry, not a copy
  (no second sections list in TS; fetch via the Go binary or generate at build time — decide).
- No changes to the in-process `expand_prompt_section` tool.

## Invariants (code-level)
- The whitelist (`RoleSubcommands`) and the act-cli.ts dispatch surface stay in sync —
  every granted subcommand resolves (the new regression test enforces it).

## Repro/Evidence
`grep -n "prompt-section" act-agent/cli/act-cli.ts` → no dispatch branch;
`grep -n "prompt-section" act-agent/internal/llm/tools/act_cli_whitelist.go` → granted;
planner ACP prompt fragment instructs the call (`common.go` actCLICommandsACP path).
