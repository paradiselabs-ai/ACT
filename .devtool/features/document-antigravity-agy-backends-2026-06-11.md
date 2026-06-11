---
id: "document-antigravity-agy-backends-2026-06-11"
status: "todo"
priority: "high"
assignee: null
dueDate: null
created: "2026-06-11T17:30:00.000Z"
modified: "2026-06-11T17:30:00.000Z"
completedAt: null
labels: ["orchestrator", "acp", "docs"]
order: "a1"
---
# Document + land the antigravity/agy Tier-1 backends (currently uncommitted, in zero docs)

## Spec
The working tree on `feat/cleanup-constitution` carries uncommitted work adding two Tier-1 ACP
backends: `antigravity` (Devin Desktop / former Windsurf) and `agy`, while dropping `gemini` from
the dispatch. Touched (uncommitted): `internal/app/app.go` (switch now
`"claude-code", "antigravity", "agy", "codex", "opencode"`), `internal/app/slash.go` (`/backend …
antigravity`), `internal/acp/agent.go`, `internal/acp/types.go`, `internal/config/config.go`,
`internal/llm/agent/agent.go`, `internal/llm/prompt/prompt.go`, `internal/runner/swarm_roles.go`,
`cli/act-cli.ts`; untracked: `internal/acp/antigravity_cli.go`, `agy-acp.mjs`, `runner/agy-acp.mjs`.
**No document anywhere mentions these backends** — the 2026-06-10/11 dual-path recon ranked this
the apex dual-implementation hazard (CV1 in `docs/audits/dual-path-recon-2026-06-10.md`).
`~/.act.json` already sets `planner.backend=antigravity` on the dev machine.

This ticket = finish/commit that work AND publish it per the constitution.

## Success Criteria
- The antigravity/agy changes are committed (by their author) with a Conventional Commit message.
- CLAUDE.md's Tier-1 backend paragraph names the live dispatch location (`app.go` backend switch)
  and the `/backend` command, WITHOUT hardcoding a member list (anti-trust: lists self-stale;
  point at the switch).
- README backend mention updated the same way.
- DEV_LOG entry appended per TASK_TRACKING.md §5.
- `gemini`'s removal from the Tier-1 dispatch is either intentional-and-documented or restored.

## Constraints
- Doc + commit hygiene only here; no behavior redesign of the backends themselves.
- Do NOT re-implement any ACP machinery — it ships at `internal/acp/` via `acp.NewACPAgent`
  (NOT `internal/llm/backend.go`, which never existed — see block6 ticket).

## Invariants (code-level)
- `acp.NewACPAgent` remains the single Tier-1 external-backend constructor (no parallel backend layer).
- `grep -rn "antigravity" CLAUDE.md README.md` returns ≥1 hit each once published.
- The Tier-1 backend dispatch remains in exactly one switch in `internal/app/app.go`.

## Repro/Evidence
`git status --short` on 2026-06-11 shows the 9 modified + 3 untracked files; `sed -n '103,112p'
internal/app/app.go` shows the live switch; `grep -rln "antigravity" CLAUDE.md README.md
F-handoff.md .devtool/` (before this ticket) → zero hits.
