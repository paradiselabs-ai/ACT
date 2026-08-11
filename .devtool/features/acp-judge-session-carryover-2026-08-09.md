---
id: "acp-judge-session-carryover-2026-08-09"
status: "review"
priority: "high"
assignee: null
dueDate: null
created: "2026-08-09T00:00:00.000Z"
modified: "2026-08-09T00:00:00.000Z"
completedAt: null
labels: ["acp", "assurance", "validation", "bug"]
order: "a0"
---
# ACP-backed Assurance carries its own prior verdicts across tasks

## Describe
Judge isolation diverges by backend for the SAME role.

- **In-process Assurance is isolated.** `app.go` wires every Tier-1 role except
  Planner with `agent.HistoryNone`, and `agent.go::scopedHistory` returns `nil`
  for that mode — the model sees only the current validation prompt. Verdict N
  cannot anchor verdict N+1.
- **ACP-backed Assurance is not.** `acp/agent.go::runTurn` correctly sends only
  the current prompt (no shared-transcript replay — the comment at the top of
  `runTurn` is right about ACT's own input scoping). But `ensureACPSession`
  caches one ACP session per ACT session and reuses it for every turn, and the
  external host keeps its own conversation state across `session/prompt` calls.
  So a `claude-code`/`gemini`/`antigravity`-backed Assurance accumulates every
  earlier verdict it issued during the run.

This is the recommended config: CLAUDE.md's Provider Configuration example ships
`"assurance": { "backend": "claude-code" }`.

Failure mode is anchoring, not corruption — subtle by construction. The concrete
case: task fails validation → swarm reworks → resubmits (same task ID; the
seen-key comment in `routeToAssurance` documents that resubmission path). The
re-validation turn arrives at a judge that still remembers writing the failing
verdict and its own reasoning for it. Sibling issue KI-03 (verdict
non-determinism, done) treated verdict stability as a model-temperature problem;
this is the same surface from the session-state side.

Adjacent, NOT covered here: `backend-settings-isolation-audit-2026-08-08`
(operator config bleeding INTO a backend). This ticket is about the judge's own
turn-to-turn memory.

## Success Criteria
- An ACP-backed Assurance opens a NEW ACP session for every validation turn;
  no verdict text from a prior task can be in its context window.
- Non-judge ACP roles (Planner, Observer, QA, swarm) keep session reuse — no
  extra `session/new` + re-priming cost on paths that don't need isolation.
- `go build ./...` clean; `go test ./internal/acp/` green.
- Behavior is identical between the in-process and ACP Assurance paths.

## Constraints
- Do not add history filtering to the ACP input path — ACT already sends only
  the current prompt there. The leak is host-side session state; fix it at the
  session boundary.
- Do not change the validation prompt. It is already evidence-first: it labels
  the swarm's completion report "NOT evidence", requires a tool-verified
  reasoning line per criterion, and fails any criterion without one.
- No new config surface. Isolation is a correctness property, not a user knob.
- Assurance only. Observer/QA also run `HistoryNone` in-process, but memory
  there is a token-cost/consistency question, not a bias-in-the-verdict one.

## Invariants (code-level)
- `internal/acp/agent.go::judgeRoleNeedsFreshSession` returns true for
  `"assurance"` and false for every other role.
- `ensureACPSession` skips the `acpSessions` cache read when that returns true,
  and still writes the newest ACP session ID to the map so cancel/shutdown
  (`Cancel`, `Close`, `RebindSystemPrompt`) keep a live handle.
- `internal/app/app.go` keeps `histMode = agent.HistoryNone` for every non-
  Planner in-process Tier-1 role.

## Repro/Evidence
Code-verified 2026-08-09 (no live run yet):
- `act-agent/internal/app/app.go` — `histMode := agent.HistoryNone`, overridden
  to `HistoryThread` only for `planner`.
- `act-agent/internal/llm/agent/agent.go::scopedHistory` — `HistoryNone → nil`.
- `act-agent/internal/acp/agent.go::ensureACPSession` — cache hit returned the
  reused ACP session ID unconditionally before this fix.
- `act-agent/internal/app/orchestrator.go::buildValidationPrompt` — evidence-
  first protocol confirmed present.

## Status
Fix landed in `internal/acp/agent.go` + unit test
`TestJudgeRoleNeedsFreshSession`. Build + `internal/acp` suite green.

**Owed before done:** live e2e with `"assurance": {"backend": "claude-code"}` —
run two validations in one session, confirm a distinct `session/new` per
validation turn and that no prior verdict text appears in the second turn's
context. Fold into the TUI e2e matrix.
