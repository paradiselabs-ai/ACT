---
id: "skills-progressive-expertise-files-2026-07-07"
status: "todo"
priority: "medium"
assignee: null
dueDate: null
created: "2026-07-07T08:33:31.000Z"
modified: "2026-07-07T08:33:31.000Z"
completedAt: null
labels: ["prompts", "orchestrator"]
order: "z2"
---
# Skills: file-based progressive expertise via the section registry (orchestration plan Phase 3)

## Spec

The on-demand prompt-section machinery exists and is live for the Planner only:
`internal/llm/prompt/sections.go` (registry) + `expand_prompt_section` tool, sections hardcoded
in Go. Generalize it into file-based Skills — reusable expertise that outlives a session and
loads only when pulled (the Skill Graph concept from CLAUDE.md, minimal form):

1. Registry additionally loads `.act/skills/*.md` (project) and `~/.act/skills/*.md` (global);
   frontmatter `name:` + `description:` feed the tool's enum and description.
2. `expand_prompt_section` becomes available to Tier 2 swarm roles (add to their toolset).
3. Name collision: project file wins over global file wins over built-in Go section.

## Success Criteria

- [ ] A `.act/skills/db-migrations.md` file appears in the tool's section enum on next agent spawn without rebuild
- [ ] Pulling it returns the file body; built-in Planner sections still work unchanged
- [ ] Swarm roles can pull skills (visible in a task session's tool schema)
- [ ] Drift test in `sections_test.go` still locks Planner prompt enumeration to the registry
- [ ] `go build/vet/test` clean

## Constraints

- Reuse the existing registry + tool — no new subsystem, no new tool name.
- No wikilink traversal / graph loading in this pass (that is the full Skill Graph, later).
- No auto-injection: skills load ONLY via the tool call (token diet holds).

## Code-level Invariants

- Base prompts must not grow: skill content ships only in the tool RESULT, never the system prompt.
- Missing/empty skills dirs = zero behavior change from today.
