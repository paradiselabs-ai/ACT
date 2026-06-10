---
title: Anti-Trust Verification — F-handoff / CLAUDE.md / status docs vs live code
status: current
verified_against: bc0673e
owner: generated
last_verified: 2026-06-10
---

# Anti-Trust Verification Report — 2026-06-10

Every load-bearing claim in `F-handoff.md`, `CLAUDE.md`, `combined-analysis.md`, and the active kanban
was checked against the **live codebase** (branch `feat/cleanup-constitution` ≡ `feat/remove-nomik` @ `bc0673e`)
by 10 independent verification agents under the anti-trust operating rules (grep-only, docs are claims not
evidence, re-grep every citation, hunt for closures of anything claimed OPEN).

**Verdict counts: 137 claims — 122 CONFIRMED · 12 STALE · 2 FALSE · 1 UNVERIFIABLE-HEADLESS.**

Verdict key: CONFIRMED = seen in live code · STALE = was true, code moved · FALSE = doesn't match reality ·
UNVERIFIABLE-HEADLESS = needs a running TUI.

Line numbers cited below were live at `bc0673e` and WILL drift — re-grep symbols before reuse.

## 1. Problems (STALE / FALSE) — the reconciliation work list

### P1. [STALE] `push-date-wrong`  (slice: git-commits)
- **Source:** F-handoff.md L19 ('fast-forwarded 26f2c3d..bc0673e on 2026-06-07') and L23-24 ('they are now pushed to origin too')
- **Claim:** The push fast-forwarding origin 26f2c3d→bc0673e happened on 2026-06-07, so origin already had Phase 3/4 + Fix 23 when the handoff was written.
- **Live evidence:** git reflog show origin/feat/remove-nomik: 'bc0673e ...@{2026-06-10 08:33:20 -0500}: update by push' and '26f2c3d ...@{2026-06-06 13:37:01 -0500}: update by push'. The fast-forward RANGE (26f2c3d..bc0673e) is exactly right, but the push happened 2026-06-10, three days after the handoff date. At writing time origin was still at 26f2c3d — the 5 commits 7021488, 1919b06, 578d280, f2c8d78, bc0673e were local-only.
- **Impact if acted on:** Anyone trusting 'PUSHED, nothing outstanding' between 06-07 and 06-10 and pulling origin (e.g. the 'second writer' the handoff tells to git pull first, or another machine) would have based work on 26f2c3d, missing Phase 3, Phase 4, and Fix 23 — exactly the collision the handoff warns about. The state is now repaired; the claim is safe to trust today.

### P2. [FALSE] `commit-tables-complete`  (slice: git-commits)
- **Source:** F-handoff.md L28-38 ('This session's commits') + L294-298 (Coordination note, range 'b03ef50→bc0673e ... described in the section above')
- **Claim:** The two commit tables plus the coordination note account for all branch commits in the session window (ac241e0..bc0673e).
- **Live evidence:** Three commits inside the interleave window appear in NEITHER writer's table and are never described: b03ef50 (06-06 07:53, fix(planner): 'Ready to start?' hard stop — orchestrator.go + prompt/planner.go), 7f439ca (08:21, feat(observer): detect orphaned/unservable pending tasks — orchestrator.go +100, orchestrator_types.go, new observer_anomaly_test.go), 1e33bc8 (08:24, fix(server): /api/tasks/assigned self-healing heartbeat — server/src/index.ts +37/-6). The coordination note's range 'b03ef50→bc0673e' attributes them to the first effort, but that section's own table starts at e06f273 and calls e06f273/9aa8417 'prior session' — so a third uncredited writer/session produced them, or the first writer's table is incomplete.
- **Impact if acted on:** Anyone reconstructing 'who changed orchestrator.go and why' from the handoff will miss ~330 lines of Observer-anomaly, planner-prompt, and server-heartbeat changes sitting between the two documented efforts — the highest drift risk in this slice, since all three touch the exact orchestrator/prompt/server surfaces both writers are actively editing.

### P3. [STALE] `p1-prompt-files-13`  (slice: claude-md-architecture)
- **Source:** CLAUDE.md Project Structure (internal/llm/prompt/ — "13 files")
- **Claim:** The prompt directory contains 13 role-specific system prompt files.
- **Live evidence:** Live ls of act-agent/internal/llm/prompt/ shows 19 .go files: 16 non-test (the 11 listed in CLAUDE.md plus planner_section_evidence.go, planner_section_examples.go, planner_section_success_criteria.go, planner_section_validation.go, sections.go) and 3 test files (prompt_test.go, prompt_roles_test.go, sections_test.go).
- **Impact if acted on:** An agent counting files to detect drift or rebuilding architecture-flows would flag false discrepancies or miss the planner_section_* prompt-splitting system entirely.

### P4. [STALE] `p1-cli-21-commands`  (slice: claude-md-architecture)
- **Source:** CLAUDE.md Project Structure + Key Concepts ("21 commands")
- **Claim:** act-agent/cli/act-cli.ts implements 21 agent CLI commands.
- **Live evidence:** Live act-cli.ts:1096-1283 has 23 top-level dispatch branches. Commands not in CLAUDE.md's list: task abandon (:1163), pvm reindex (:1219), codebase / codebase onboard (:1271,:1278), swarm (:1281).
- **Impact if acted on:** Doc rebuilds or help text generated from the '21 commands' figure would omit swarm/codebase/abandon/reindex.

### P5. [STALE] `ww-autoroute-guard`  (slice: claude-md-architecture)
- **Source:** CLAUDE.md What Works (auto-route Tier 1 → Planner: 'Recursion guard consecutiveAutoTurns < 5')
- **Claim:** Autoroute recursion is guarded by a consecutiveAutoTurns < 5 counter, reset on every human input.
- **Live evidence:** consecutiveAutoTurns no longer exists. orchestrator.go:91-97: recentAutoRoutes sliding-window cap, comment 'Audit Fix 6 — replaces the prior consecutiveAutoTurns counter which... missed three documented loops'. Constants: autoTurnCap = 5 (orchestrator.go:1421), autoRouteWindow = 10 * time.Minute (:1427); enforcement at :1314-1325. Still cleared by HandleHumanInput. Replacement commit: 4cb1d26.
- **Impact if acted on:** An agent reasoning about loop protection (or re-adding a counter) would duplicate/conflict with the sliding-window mechanism and could reintroduce the three loop classes Fix 6 closed.

### P6. [STALE] `ww-tier1-tool-subsets`  (slice: claude-md-architecture)
- **Source:** CLAUDE.md What Works (token diet: 'Planner/Observer get just bash, Assurance/QA get bash + view + grep')
- **Claim:** Tier1ToolsForRole gives Planner/Observer bash only, and Assurance/QA bash + view + grep.
- **Live evidence:** Tier1ToolsForRole exists (act-agent/internal/llm/agent/tools.go:124) but the composition changed: PlannerTools = NewActCLITool + NewExpandPromptSectionTool, explicitly 'Per KI-02: no raw bash' (tools.go:96-105); ObserverTools = NewActCLITool only, 'No bash' (tools.go:107-113); Assurance/QA = NewActCLITool + view + grep — no bash (tools.go:115-135). A stale comment matching CLAUDE.md's wording survives in app.go:76-78.
- **Impact if acted on:** An agent could 'restore' bash to Tier 1 roles believing it was the design, undoing the KI-02 act_cli whitelisting.

### P7. [STALE] `ww-context-paths`  (slice: claude-md-architecture)
- **Source:** CLAUDE.md What Works (context paths: defaultContextPaths reduced to ["ACT.md", "ACT.local.md"])
- **Claim:** defaultContextPaths = ["ACT.md", "ACT.local.md"].
- **Live evidence:** act-agent/internal/config/config.go:303-307: defaultContextPaths = {"AGENTS.md", "ACT.md", "ACT.local.md"} — AGENTS.md added (Planner-authored project brief, regenerated on PROJECT_BRIEF parse). CLAUDE.md still correctly says CLAUDE.md itself is excluded (config.go:298 comment confirms).
- **Impact if acted on:** Token-budget reasoning and docs about what auto-loads into prompts would miss the AGENTS.md injection (~1-2K tokens) and its regeneration loop.

### P8. [STALE] `pitfall7-pvm-analytics-placeholder`  (slice: claude-md-architecture)
- **Source:** CLAUDE.md Common Pitfalls #7
- **Claim:** PVM analytics layer (getAgentProfile, compareAgents, getAgentSynergy, SelfImprovementEngine) currently returns placeholder data (successRate: 0.85 + Math.random() * 0.15, four // placeholder methods); 'embeddings real, analytics fake'.
- **Live evidence:** The active store is LocalEmbeddingVectorStore (server/src/index.ts:42). Its getAgentProfile (LocalEmbeddingVectorStore.ts:273-330) computes real per-task-type metrics from lookupTaskOutcomes (validated/passed ratios, real durations, evidence-quality tiers); getAgentSynergy (:366-425) computes shared-task pass rates from task_validated/task_validation_failed events; compareAgents (:428-450) derives from real profiles. SelfImprovementEngine analyze methods compute real aggregations from events (e.g., analyzeCommunicationPatterns at SelfImprovementEngine.ts:378-402). No Math.random/placeholder hits in either file. The 0.85 + Math.random() placeholders survive only in inactive code: MockVectorStore.ts:194 and QdrantVectorStore.ts:265 (Qdrant is build-excluded per the repo's own flows-explainer findings). index.ts:1223 comment also states profiles come from real lookupTaskOutcomes data. Caveat: statistical quality of outputs requires runtime data to assess — but the 'returns placeholder data' claim is code-level false today.
- **Impact if acted on:** Answering 'is PVM real' per CLAUDE.md would wrongly tell the user the analytics are fake, and an agent might re-implement the analytics layer that already exists.

### P9. [STALE] `tier1-backend-only-tier2`  (slice: claude-md-architecture)
- **Source:** CLAUDE.md NesTTY section ('Backend selection only applies to Tier 2 — Tier 1 agents are in-process goroutines and have no executable to swap')
- **Claim:** Tier 1 agents have no backend selection and no executable to swap.
- **Live evidence:** Live app.go:86-110 dispatches each Tier 1 role on cfg.Agents[agentName].Backend: case "claude-code", "codex", "gemini", "opencode" → acp.NewACPAgent(role, backendChoice, withTier1ShimPath(role, acpCfg), ...); default (empty/'act-agent') → in-process agent.NewAgent. The ACP machinery exists on this branch: act-agent/internal/acp/ (agent.go, claude_code.go, client.go, session.go, transport.go, types.go) and act-agent/cmd/act-tier1-shim/. Introduced by commit 5021848 'feat(acp): Tier 1 backend-selectable via Agent Client Protocol'. Note: 'acp' is NOT itself a backend value — backend values are host names (claude-code shipping; codex/gemini/opencode return explicit unimplemented errors); ACP is the invisible wire mechanism. Config struct documents this at config.go:91-93. Tier 1 backends are set via agents.<role>.backend in ~/.act.json, not via /swarm — act-cli.ts:603 still correctly scopes `swarm set` to Tier 2 roles.
- **Impact if acted on:** HIGH. An agent trusting CLAUDE.md could re-implement Tier 1 backend support from scratch — exactly the dual-implementation failure mode CLAUDE.md's own banner warns about (and it names ACP as the past example).

### P10. [STALE] `provider-config-opencode-json`  (slice: claude-md-architecture)
- **Source:** CLAUDE.md Provider Configuration ('Configure in .opencode.json (see .opencode.example.json)')
- **Claim:** Per-role provider config lives in .opencode.json, with an .opencode.example.json template in the repo.
- **Live evidence:** Internally inconsistent with the same section's later line 'Any role not configured in ~/.act.json...' and with the code: primary config is ~/.act.json (comments.go:106-125); .opencode.json is read only as a legacy fallback (global ~/.opencode.json last in search path; local <cwd>/.opencode.json second candidate, comments.go:82,109; writer.go:79-90 calls it legacy). find over the repo (excluding node_modules) finds NO .opencode.example.json anywhere.
- **Impact if acted on:** A user or agent following this section would create .opencode.json and look for a template that doesn't exist; role configs would only work via the legacy fallback path and would lose to any existing ~/.act.json keys.

### P11. [STALE] `block6-files-to-create-stale`  (slice: kanban-vs-reality)
- **Source:** .devtool/features/block6-acp-cli-backend-2026-04-21.md body ('Files to create: act-agent/internal/llm/backend.go, act-agent/internal/llm/acp/backend.go; New interface: AgentBackend')
- **Claim:** The ACP work will create internal/llm/backend.go and internal/llm/acp/backend.go with an AgentBackend interface.
- **Live evidence:** Neither file exists (ls act-agent/internal/llm/backend.go → no such file; internal/llm contains only agent/models/prompt/provider/tools). The implementation landed at act-agent/internal/acp/ instead, exposed as acp.NewACPAgent returning agent.Service (app.go:110). No AgentBackend interface in Go code — only config.WriteAgentBackend (internal/config/writer.go:16), a config writer, not the interface.
- **Impact if acted on:** Someone executing the ticket as written would create a second, parallel ACP backend layer at internal/llm/ on top of the existing internal/acp/ one — exactly the dual-implementation failure mode CLAUDE.md warns about.

### P12. [STALE] `qa-redesign-phase-a-half-implemented`  (slice: kanban-vs-reality)
- **Source:** .devtool/features/qa-redesign-phase-a-nomik-agentmd-ingest-2026-04-21.md (status: todo; 'Give QA/Synthesizer Grep + AGENT.md ingestion... Per-turn prompt stops forbidding tool use')
- **Claim:** QA/Synthesizer lacks Grep access and its prompt forbids tool use (status todo = none of this exists).
- **Live evidence:** The Grep half is ALREADY implemented: QA gets bash+view+grep via agent.Tier1ToolsForRole (act-agent/internal/llm/agent/tools.go:119-124, app.go:77,112), and the QA prompt actively instructs tool use: 'Use view and grep to read validated outputs directly' (act-agent/internal/llm/prompt/qa_synthesizer.go:62-63). The AGENT.md-ingestion-at-startup half is genuinely unimplemented (no AGENT.md grep hits in app/ or qa_synthesizer.go). Body was correctly scrubbed of Nomik in the uncommitted edit (nomik is gone from live code — only removal-lock tests in internal/tui/components/dialog/onboarding_test.go:11-33 reference it). Also: the body's dependency pointer to '.claude/HANDOFF.md QA/Synthesizer gap' is a stale-doc reference per the handoff protocol.
- **Impact if acted on:** An implementer would re-grant grep/view tools and rewrite a prompt clause that no longer forbids tools — duplicate work and possible prompt regression. Ticket should be re-scoped to AGENT.md ingestion only.

### P13. [STALE] `sessionid-35-sites`  (slice: config-env)
- **Source:** F-handoff.md L135-136
- **Claim:** ~35 orchestrator sites read the single o.sessionID.
- **Live evidence:** Live grep: exactly 14 occurrences of `o.sessionID` in act-agent/internal/app/orchestrator.go (2 assignments at :144/:149, 12 reads — mostly the `sid := o.sessionID` snapshot pattern at :183,348,723,1333,1611,1887,2063,2599,2615,2721,2740 plus :2862). Count is also 14 at e06f273, 9aa8417, b03ef50, 7021488, bc0673e~1 — so ~35 matches no recent revision. If you instead count downstream uses of the `sid` locals (46) the total is ~60. The architectural point (single shared session deeply baked in) IS true; only the number is wrong.
- **Impact if acted on:** Low — anyone sizing a 'split into per-agent physical sessions' refactor off the 35 figure gets the wrong effort estimate; re-grep gives 14 direct sites + 46 derived uses.

### P14. [FALSE] `server-dev-one-shot`  (slice: config-env)
- **Source:** F-handoff.md L145-146
- **Claim:** server `npm run dev` is one-shot `npx tsx`, no hot-reload — restart to load server changes.
- **Live evidence:** Live server/package.json:10 — `"dev": "tsx watch src/index.ts"` — watch mode, i.e. hot-reload on file change. git log -S 'tsx watch' shows this script has said `tsx watch` since 5fb565e (Phase 5 Foundation) with no intervening change; it was never one-shot in git history. tsx ^4.21.0 is in devDependencies (package.json:49). Project CLAUDE.md agrees ('tsx watch, port 8080'). Caveat: whether watch-restart fires correctly at runtime on this machine is UNVERIFIABLE-HEADLESS without starting the server, but the 'one-shot npx tsx' description of the script is simply wrong.
- **Impact if acted on:** A dev trusting the handoff will manually kill/restart the server after every edit (wasted time, and each restart re-replays coordination-log.jsonl), or worse, 'add' watch mode that already exists. Conversely they may not expect tsx-watch auto-restarts wiping in-memory-only state mid-test.

## 2. Confirmed claims (abbreviated — full evidence in the per-slice JSON)

**git-commits** (18 confirmed):
- `all-17-commits-exist` — All 17 named commits (bc0673e, f2c8d78, 578d280, 1919b06, 7021488, 26f2c3d, 9aa8417, e06f273, ac241e0, 8249a19, 8e3d3a8, ffb51e4, 2326d0b, 3de2163, 4f7fc3e, e1adc85, 6d934d2) exist on the branch with the described subjects.
- `head-is-bc0673e` — feat/remove-nomik HEAD = bc0673e.
- `origin-sync-state` — local == origin/feat/remove-nomik == bc0673e; tree in sync, nothing outstanding to push.
- `fix23-between-phase3-phase4` — 578d280 + f2c8d78 (Fix 23, 'other writer') landed between the Phase-3 commit (7021488) and the Phase-4 commit (bc0673e).
- `phase4-11-files` — bc0673e (Phase 4) changed exactly the 11 listed files: content.go, message.go, migration 20260607000000_add_message_thread_id.sql, sql/messages.sql, messages.sql.go, models.go, agent.go, agent-tool.go, acp/agent.go, app.go, scoped_history_test.go.
- `phase3-scopehistory` — 7021488 (Phase 3) added a scopeHistory bool to the in-process agent so event-driven workers feed the model only their current prompt.
- `phase1-rolelabels` — e06f273 (Phase 1) code-stamps Tier-1 role labels via applyRoleLabel and adds fromHuman bool to runAgentTurn.
- `phase2-qa-watchdog-observer-gate` — 9aa8417 (Phase 2) makes the QA watchdog honor synthesizedAt (taskSynthesized) and hashes a stable anomalySignature for the Observer no-op gate.
- `26f2c3d-nul-lockkey` — 26f2c3d strips a stray NUL byte from the server lockKey delimiter (spawned sub-task task_7a786c8e).
- `1919b06-kanban-moves` — 1919b06 moved exactly 4 dogfood-bug kanban cards to .devtool/features/done/: code-enforced-agent-role-prefix, planner-prompts-render-as-human, qa-synth-queue-never-drains, observer-autoroute-loop-no-ceiling.
- `f2c8d78-ticket-in-progress` — f2c8d78 moved the assurance-fail-closed-empty-criteria-2026-05-26 kanban ticket to in-progress (not done).
- `578d280-fix23-content` — 578d280 changes parseValidationVerdict (~orchestrator.go 3094-3105) to Passed = OverallScore>=95 && len(CriteriaResults)>0, with tests TestParseValidationVerdict_EmptyCriteriaFailsClosed and _EmptyCriteriaBrokenVerdict.
- `ac241e0-fix22-content` — ac241e0 (Fix 22) branches PlannerPrompt by backend, moves ProviderACP from acp → models pkg keeping an alias at acp/agent.go:30, adds actCLICommandsACP in common.go and test TestPlannerPromptBranchesOnProvider in prompt_roles_test.go.
- `8249a19-fix20-content` — 8249a19 (Fix 20+20b) rewrites the bare act-CLI shorthand in planner_section_evidence.go, rewords validation out of planner_section_validation.go, test TestNoSectionUsesBareActShorthand in sections_test.go.
- `8e3d3a8-ffb51e4-2326d0b-variants` — 8e3d3a8 adds variantSystemNoTask + 2 named tests; ffb51e4 drops the variantPassVerdict CREATE_TASK escape hatch; 2326d0b trims variantFailVerdict keeping the split.
- `round5b-ordering` — Round 5b commits ran ac241e0 → 8249a19 → 8e3d3a8 → ffb51e4 → 2326d0b → 3de2163 (docs-only), with Round 5a (4f7fc3e Fix 15, e1adc85 Fix 16, 6d934d2 Fix 17) made earlier and carried in; bc0673e landed AFTER 578d280.
- `worktree-clean-go-sql` — Working tree is clean of Go/SQL changes; pre-existing untracked junk at repo root (*.html, docs/refactor/, Nomik-removal kanban deletions) is unrelated.
- `linear-history-no-merges` — The two writers' commits interleave on a single linear branch (no merge commits), making a fast-forward push possible.

**phase4-notebooks** (11 confirmed):
- `p4-message-threadid-exists` — Message.ThreadID and CreateMessageParams.ThreadID exist and are plumbed through Create and fromDBItem in internal/message/.
- `p4-update-preserves-threadid` — message Update only writes parts/finished_at so ThreadID is preserved on update.
- `p4-migration-file` — Migration 20260607000000_add_message_thread_id.sql exists with ALTER TABLE messages ADD COLUMN thread_id TEXT NOT NULL DEFAULT '', embedded via //go:embed migrations/*.sql, applied by goose on start.
- `p4-handedited-db-threadid-last` — Hand-edited generated db code (messages.sql, messages.sql.go, models.go) has thread_id appended LAST in every struct and Scan, matching what sqlc would emit; sqlc is not installed.
- `p4-agent-historymode` — agent.go has threadID field + HistoryMode type with HistoryFull/HistoryNone/HistoryThread, scopedHistory() generalizing the Phase-3 scopeHistory bool which no longer exists.
- `p4-four-create-sites-stamped` — ALL FOUR message-create sites in agent.go stamp a.threadID: user (createUserMessage), assistant (streamAndHandleEvents), tool-result (~488), summary (~731).
- `p4-agent-tool-historyfull` — agent-tool.go creates the task sub-agent with ("", HistoryFull).
- `p4-acp-stamps-role` — acp/agent.go stamps ThreadID: a.role on its 2 display message creates while still not filtering its own input (external session is already per-agent).
- `p4-app-wiring-by-role` — app.go wires history mode per role, not backend: planner→HistoryThread, workers (Observer/Assurance/QA)→HistoryNone in the in-process branch, and CreateAgentForRole gives Tier 2 ("", HistoryFull).
- `p4-scoped-history-test` — scoped_history_test.go exists, unit-testing all 3 modes plus thread filtering, and is green.
- `p4-emitsystemmessage-empty-threadid` — emitSystemMessage creates system messages with ThreadID empty, so they are excluded from the Planner's HistoryThread input (intended behavior).

**phases123** (9 confirmed):
- `p1-applyrolelabel-strip-prepend-idempotent` — applyRoleLabel exists and is strip-then-prepend and idempotent: strips any leading role label (incl. hallucinated 'Human:') then prepends the authoritative Tier-1 label.
- `p1-applied-every-tier1-assistant-message` — Every Tier-1 assistant message gets the code-stamped Planner:/Observer:/etc. label.
- `p1-fromhuman-only-handlehumaninput` — runAgentTurn gained `fromHuman bool` and only HandleHumanInput passes true.
- `p1-internalpromptmarker-nonhuman` — Everything not fromHuman gets the InternalPromptMarker prepended, so injected Planner prompts no longer render as 'Human:'.
- `p2-qa-watchdog-synthesizedat` — QA watchdog honors synthesizedAt via taskSynthesized so it stops re-firing on already-synthesized tasks.
- `p2-server-poll-already-drained` — No server change was needed for Phase 2 because the server already drained the QA poll path (/api/tasks/validated filters synthesized tasks).
- `p2-observer-anomalysignature-gate` — Observer no-op gate hashes a stable anomalySignature (category+task+agent, not the volatile prompt); escalates an unchanged anomaly set once; all-clear resets the gate.
- `p3-scopehistory-added-then-superseded` — Phase 3 (7021488) added a `scopeHistory bool` to the in-process agent so event-driven workers feed the model only their current prompt, and this was superseded/generalized by Phase 4.
- `p3-scopehistory-gone-historymode-live` — scopeHistory is gone from agent.go; the live mechanism is HistoryMode (HistoryFull|None|Thread) + scopedHistory(msgs, mode, threadID).

**round5b-fixes** (14 confirmed):
- `f22-plannerprompt-branches-provider` — PlannerPrompt(provider) branches the CLI fragment by backend (ACP vs in-process).
- `f22-provideracp-models-pkg-alias` — ProviderACP moved from acp pkg to models pkg, with an alias kept at acp/agent.go:30 to avoid the acp→prompt import cycle.
- `f22-actclicommandsacp-common` — actCLICommandsACP was added in common.go as the ACP-backend CLI fragment variant.
- `f20-evidence-pvm-rewrite` — The bare `act pvm search` shorthand in planner_section_evidence.go was rewritten to the act_cli JSON shape plus an ACP note.
- `f20b-validation-queue-reworded` — `act validation queue` was reworded out of planner_section_validation.go because the Planner isn't allowlisted for the `validation` subcommand.
- `f20-test-bare-shorthand` — Test TestNoSectionUsesBareActShorthand exists in sections_test.go and locks Fix 20/20b.
- `f18-variantsystemnotask-routing` — variantSystemNoTask exists and both synthesis_stuck and dedup are routed to it instead of the retry/abandon menu.
- `f19-pass-verdict-no-escape-hatch` — variantPassVerdict no longer contains the option-(a) 'emit CREATE_TASK for the obvious next step' escape hatch.
- `f21-fail-verdict-prose-cadence` — variantFailVerdict was trimmed to prose, the pass/fail split was kept, and the once/twice/three-times cadence mirroring planner_section_validation.go was preserved.
- `f-roundb-tests-exist` — The named test files prompt_roles_test.go and sections_test.go exist with the handoff-named tests.
- `f15-examples-dependencies-shape` — Fix 15 is in: the examples section no longer teaches the forbidden inline @dependencies shape, using the top-level JSON dependencies array instead.
- `f16-validationscore-populated` — Fix 16 is in: routeToQA populates ValidatedOutput.ValidationScore (previously always 0) so the QA synthesis prompt shows the real score.
- `f17-planner-prompt-section-listed` — Fix 17 is in: the hand-written actCLICommands("planner") fragment now lists the prompt-section command.
- `f-roundb-commits-exist-unreverted` — All nine listed Round 5a/5b commit SHAs exist on the branch with matching subjects and none were reverted.

**round6-open-findings** (10 confirmed):
- `r6-2-brownfield-injection-open` — Brownfield researcher output flows unfenced/unscrubbed into every Tier 1+2 system prompt: brownfieldEnrichPrompt → brief.CodebaseNotes → AGENTS.md verbatim concat → context appended last in prompt.go.
- `r6-3-acp-cli-planner-only-open` — actCLICommandsACP only branches for the Planner; ACP-backed Observer/Assurance/QA still get the in-process 'do NOT shell out' fragment while renderShimNote tells them to use the shim via Bash.
- `r6-4-rebind-skipped-on-write-failure` — RebindSystemPrompt is skipped when the AGENTS.md write fails, leaving all four Tier 1 agents on stale intake-era prompts; the else-nesting needed re-verification.
- `r6-5-qa-synthesis-via-variantanomaly` — QA SYNTHESIS_COMPLETE messages are routed to the Planner through variantAnomaly ('React by taking action'), the wrong tone and the likeliest empty-CREATE_TASK trigger.
- `r6-6-failverdict-count-blind` — variantFailVerdict asks the Planner to act differently on 'Third+ failure' without supplying the failure count.
- `r6-7-systemnotask-generic` — variantSystemNoTask is too generic for synthesis_stuck — one template serves both synthesis_stuck and dedup with no synthesis-specific guidance.
- `r6-8-9-actcli-schema-vs-prose` — The act_cli tool schema/description offers status/log/context that the Planner prose forbids for human queries; the carve-out is not co-located in the tool description.
- `r6-10-burst-first-failure-only` — Burst (batch) mode surfaces only the first failed task's details to the Planner.
- `r6-28-fixed-entries-hold` — All 28 [FIXED] entries in docs/planner-prompt-audit/combined-analysis.md still hold in live code.
- `r6-chatleak-gate-fromhuman` — The live chat-leak gate is `if !fromHuman` (not `role != "planner"` as planner-prompts.json claims), so autoroute/[SYSTEM] prompts routed through fireWhenPlannerIdle are hidden from chat.

**fix23-assurance** (12 confirmed):
- `f23-guard-shipped` — parseValidationVerdict force-fails verdicts with empty criteriaResults: Passed = OverallScore>=95 && len(CriteriaResults)>0.
- `f23-gaps-injection` — An otherwise-passing verdict force-failed for empty criteria gets an explanatory gaps message injected.
- `f23-score-preserved` — The reported score is preserved as-is on force-fail (not zeroed).
- `f23-tests-exist` — Tests TestParseValidationVerdict_EmptyCriteriaFailsClosed (exact wordtallies repro: score 100, criteriaResults:[], 'tests pass') and _EmptyCriteriaBrokenVerdict exist.
- `f23-no-server-gate` — Layer #1 NOT shipped: server/src/index.ts validation handler has no zero-criteria rejection.
- `f23-no-prompt-refusal` — Layer #2 NOT shipped: assurance.go prompt has no explicit zero-criteria refusal clause.
- `f23-no-requeue-loop` — The ticket's recommended option 1 (refuse + re-queue + immediate Planner re-emit with criteria) is NOT built.
- `f23-no-system-message` — No orchestrator chat system message stating 'no criteria attached' was shipped.
- `f23-no-e2e-assertion` — No e2e-api.sh assertion covering zero-criteria rejection was shipped.
- `f23-no-dispatch-rejection` — Criteria-less CREATE_TASK is NOT rejected at dispatch (separate follow-up, unshipped).
- `f23-kanban-status` — Kanban ticket assurance-fail-closed-empty-criteria-2026-05-26 is status in-progress (not done) and its Status-update section records exactly the shipped/open split.
- `f23-residual-junk-criteria` — Residual: a verdict with criteriaResults:[{}] (non-empty junk, len>0) still passes the guard; only the literal [] case is closed.

**claude-md-architecture** (18 confirmed):
- `p1-client-go` — act-agent/internal/act/client.go is the native HTTP client for the ACT server.
- `p1-runner-mjs` — act-agent/runner/act-runner.mjs is the headless swarm agent spawner.
- `ww-eventhub-ratelimit` — EventHub rate-limits messages at 3 per 30s per agent.
- `ww-chronlog-path` — ChronologicalLog is append-only JSONL at ./data/coordination-log.jsonl.
- `ww-a2a-endpoints` — A2A Agent Card and task push endpoints exist on the server.
- `ww-file-locking` — File locking claim/release endpoints exist.
- `ww-runner-409-selfheal` — Runner self-heals 409 on registration by deleting the stale agent and retrying.
- `ww-setpgid` — Runner subprocesses get process groups via Setpgid so the parent kills the entire subtree.
- `ww-sweeporphans` — SweepOrphans() at startup runs pkill -f act-runner.mjs defensively.
- `ww-runner-logs` — Runner subprocess stdout/stderr go to ~/.act/runners/<role>.log.
- `ww-lazy-swarm-spawn` — Runners spawn on the first CREATE_TASK, not on every act launch.
- `ww-coordloop-3s` — coordinationEventLoop polls /api/log every 3s and surfaces task lifecycle events as system messages.
- `ww-5min-turn-timeout` — runAgentTurn wraps agentSvc.Run with a 5-minute context.WithTimeout; on expiry it cancels the agent and emits a system message.
- `cfg-act-json` — The binary's config truth is ~/.act.json.
- `cfg-developer-fallback` — AgentConfigForRole falls back to agents.developer only, with no deeper fallback.
- `cfg-swarm-commands` — /swarm exists in the TUI and `act-agent swarm set <role|all> <backend>` exists in the CLI.
- `cfg-tier2-backend-values` — Tier 2 swarm backends are act-agent (default) or claude-code.
- `dev-commands-paths` — Build/run paths exist: server/ with npm dev, and act-agent buildable with go build at its root.

**kanban-vs-reality** (14 confirmed):
- `block6-acp-status-inprogress` — Block 6 ACP CLI backend for Tier 1 is in-progress.
- `assurance-failclosed-fix23-landed` — Fix 23 closed the empty-criteria fail-open at the verdict parser, with regression tests; ticket stays in-progress for the remaining layers.
- `assurance-failclosed-open-items-still-open` — Server-side zero-criteria 400 gate, assurance.go zero-criteria refusal clause, orchestrator 'no criteria attached' system message, and the e2e-api.sh assertion are all still NOT built.
- `invariants-ticket-ready-to-start-prompt-only` — The 'Ready to start?' confirmation hard-stop is prompt text only; no code gate rejects a PROJECT_BRIEF arriving in the same turn as the question.
- `spil-stage1-no-go-code-yet` — No SPIL stage-1 parser/AST/evaluator code exists in Go yet; the only parser is the regex MVP SPILParser.ts.
- `tui-heartbeat-not-built` — No swarm-activity heartbeat/indicator exists in the TUI.
- `tui-truncation-still-open` — The TUI still truncates multi-line tool/status output with no expand affordance.
- `done-role-prefix-fix-exists` — ACT now code-prepends the authoritative role label to every Tier 1 message, stripping hallucinated prefixes like 'Human:'.
- `done-planner-marker-fix-exists` — Orchestrator-injected Planner prompts no longer render as Human: — InternalPromptMarker now applies to all non-human input including Planner.
- `done-observer-escalation-cap-exists` — The Observer now escalates an unchanged anomaly set once and stays quiet (no endless 2-min autoroute loop).
- `done-qa-queue-drain-fix-exists` — tier1Watchdog no longer re-fires QA on already-synthesized tasks — it applies the synthesizedAt exclusion.
- `done-model-registry-deleted` — The per-provider model-list files and registry machinery (SupportedModels, resolveModelAlias) are deleted; provider+model are pure config.
- `untracked-tier1-binaries-explained` — Untracked act-tier1-* files in act-agent/ are unexplained artifacts.
- `untracked-root-htmls` — Two untracked HTML files at repo root are unexplained.

**config-env** (11 confirmed):
- `actjson-backends` — ~/.act.json backends: planner=claude-code, observer=in-process (gpt-oss-120b), assurance=claude-code, qa_synthesizer=claude-code, developer=claude-code.
- `sqlc-not-on-path` — sqlc is NOT on PATH; config exists at act-agent/sqlc.yaml.
- `go-binary-version` — Go toolchain is /opt/homebrew/bin/go version 1.26.1.
- `act-symlink` — /opt/homebrew/bin/act is a symlink to the repo's act-agent binary.
- `acp-runturn-content-only` — ACP agent runTurn sends ONLY content via client.Prompt and writes display messages to the shared session.
- `acpsessions-map-per-agent` — Each ACP agent keeps a private external session per ACT-sessionID per agent via the acpSessions map.
- `tui-renders-by-sessionid` — TUI renders strictly by msg.Payload.SessionID == m.session.ID at internal/tui/components/chat/list.go:196,241.
- `ownership-loop-session-agnostic` — messageOwnershipLoop (orchestrator.go ~1145-1162) runs off o.currentSpeaker + a global pubsub subscription and is session-agnostic.
- `messageowners-assistant-only-inmemory` — messageOwners map[string]string tags assistant messages only, in-memory, built live, not persisted, not rebuilt on replay.
- `lstool-preexisting-failure` — Pre-existing test failure: TestLsTool_Run in internal/llm/tools panics 'config not loaded' — a test-harness config issue in a package untouched by this work.
- `build-test-commands` — Build with `cd act-agent && /opt/homebrew/bin/go build ./...`; test with `go test ./internal/llm/agent/... ./internal/app/...`.

**build-and-test** (5 confirmed):
- `build-clean` — go build ./... in act-agent compiles with no errors.
- `agent-tests-green` — go test on ./internal/llm/agent/..., ./internal/app/..., ./internal/llm/prompt/... all pass.
- `lstool-run-panics` — go test ./internal/llm/tools/... fails because TestLsTool_Run panics with 'config not loaded'.
- `qdrant-ts-error` — server/src/services/QdrantVectorStore.ts has a known pre-existing TypeScript error.
- `scoped-history-tests-pass` — The scoped_history_test.go tests in internal/llm/agent pass.

## 3. Unverifiable headless

- `p4-live-verify-planner-thread` (phase4-notebooks): End-to-end behavior — an in-process Planner's prepared messages contain only human turns + autoroute prompts + its own replies, with no worker traffic — still needs live verification.
  Runtime check needed: All static plumbing is confirmed (see other claims), but the end-to-end assertion requires runtime: set planner to an in-process model in ~/.act.json, run act-agent in a TUI, trigger Observer/Assurance output then a human turn, and inspect the Planner's 'Prepared messages' dump in ~/.act/.../debug.log to confirm no worker snapshots/outputs appear and tool_use/tool_result pairs are intact.

## 4. Extra observations (adjacent contradictions found en route)

**git-commits:** Unmentioned commits in the last ~30 that touch orchestrator/prompt/server code, beyond the three interleave-window ones already flagged (these predate the handoff's session and are referenced at most indirectly, never as commits): e49d197 feat(orchestrator) full Info-level brownfield-onboard tracing; 701fede fix(orchestrator) brownfield enrichment honors researcher backend; 045ad26 feat(orchestrator) persist brownfield analysis into AGENTS.md; c1e9bd5 feat(orchestrator) brownfield INTAKE auto-onboard; 033bec1 feat(cli) rebuild `act codebase onboard`; a90b010 refactor remove Nomik integration; 35f3bd7 fix(server) two ChronLog rehydration bugs; 805fd4e cluster of 5 prompt+normalization fixes; a8577d2 deterministic context-walk + content-hash cache skip; b3cc2b7 route NEED_CLARIFICATION to named addressee; 0b5129b surface dispatch-hash dedup events. Also note: every commit on the branch uses the identical author identity (dead-developers <paradiselabs.ai@gmail.com>), so the handoff's two-writer (arguably three-writer) attribution rests entirely on its own say-so plus timestamp gaps — git metadata cannot corroborate WHO; only the ordering claims are verifiable, and those all check out. Finally, the project-docs claim worth correcting: F-handoff.md L18-19's 'PUSHED on 2026-06-07' — the local reflog proves the push happened 2026-06-10 08:33; if any other doc repeats the 06-07 push date, it inherited the error.

**phase4-notebooks:** Commit bc0673e ('feat(orchestrator): per-agent notebooks — scope in-process input by thread') is HEAD of feat/cleanup-constitution, matching the handoff's Phase 4 commit reference — none of the verified code has been reverted since. Two precision nudges for doc reconciliation: (1) the handoff's tool-result line '~488' and summary '~731' are slightly off from the stamp lines themselves (stamps at 491 and 733; the Create calls open at 488 and 731 — effectively accurate); (2) the 'column appends LAST' wording is true for all structs, Scans, and SELECT/RETURNING lists, but in the INSERT column list of CreateMessage thread_id sits before created_at/updated_at — harmless and still what sqlc would emit, but a literal reader expecting thread_id last in the INSERT will see otherwise. Also note F-handoff is not on disk under .claude/ — it lives at repo root (/Users/user/Documents/Developer/dev/AI/act/F-handoff.md), untracked.

**phases123:** All four phase→commit attributions in the handoff table check out via `git log -S` symbol tracing: applyRoleLabel→e06f273, anomalySignature/taskSynthesized→9aa8417, scopeHistory→7021488, HistoryMode→bc0673e. Line numbers in the handoff have drifted slightly but symbols are intact (e.g. handoff says parseValidationVerdict ~3094-3105 and tool-result stamp ~488; live tool-result ThreadID stamp is agent.go:491 — drift, same code). One precision note on the 'every Tier-1 assistant message' claim: the label is applied at message-finalize keyed on the in-memory messageOwners tag set at CreatedEvent; an assistant message whose owner was never tagged in this process (e.g. created before a crash) would not be retro-labeled — but the label IS persisted via Messages.Update once applied, so replayed sessions keep labels. This matches, rather than contradicts, the handoff's own L134 statement that messageOwners is in-memory and not rebuilt on replay. No reverts or contradicting later commits found for any Phase 1-3 change on this branch.

**round5b-fixes:** Three minor drift/nuance notes, none rising to STALE: (1) the handoff cites actCLICommandsACP at common.go:147 — the live func declaration is common.go:146 (the planner guard is 147); one-line drift, consistent with the handoff's own 'line numbers drift fast' warning. (2) Fix 21's 'once/twice/three-times cadence' is preserved semantically but the variantFailVerdict prose phrases it as 'First or second failure / Third+ failure' (orchestrator.go:1373), while the literal once/twice/three-times wording lives only in planner_section_validation.go:29/32/36 — anyone grepping the variant for 'once' will miss it. (3) The handoff's open item #3 (Fix 22 half-closed: actCLICommandsACP only branches for planner, so ACP-backed Assurance/Observer/QA still get the in-process 'do NOT shell out' fragment) is confirmed still OPEN in live code at common.go:147-149 — the Round 5b fix claims themselves are accurate about what was and wasn't closed. Separately, CLAUDE.md's Provider Configuration section says roles are configured in '.opencode.json' in one paragraph and '~/.act.json' in the next — the memory file says ~/.act.json is config truth; CLAUDE.md's .opencode.json mention looks like leftover fork text (not part of this slice, flagging for the reconcile pass).

**round6-open-findings:** 1) The handoff's blanket framing 'autoroute/[SYSTEM] prompts leak to chat — fixed' is only two-thirds true: build-mode and brownfield-intake [SYSTEM] turns are hidden (fireWhenPlannerIdle → fromHuman=false → InternalPromptMarker), but the resume-context [SYSTEM] block is prepended to the human's own first message (orchestrator.go:236-238) and rides fromHuman=true, so it is still rendered in chat. Any doc regen (handoff next-step #4) must keep that surface marked ACTIVE. 2) common.go:138-145's own comment explicitly documents finding #3 as a known deliberate gap ('Only the planner case diverges enough...'), so the fix is a conscious deferral, not drift — the scaffolding correction is to amend combined-analysis.md 3.5 from FIXED to Planner-only, which matches the handoff. 3) None of the five commits landed after the handoff snapshot (7021488, 1919b06, 578d280, f2c8d78, bc0673e) touch any of findings #2-#10's code paths — the handoff's OPEN statuses have not been silently closed by the parallel notebooks/Fix-23 work. 4) CLAUDE.md's project-structure section still names the Go module path as github.com/opencode-ai/opencode (per MEMORY.md), but live imports use github.com/paradiselabs-ai/ACT/act-agent (e.g. act_cli.go:11) — an adjacent doc staleness worth folding into the scaffolding reconciliation.

**fix23-assurance:** 1) The e2e suite is load-bearing on the missing server gate: server/scripts/e2e-api.sh:122 posts passed:true with criteriaResults:[] and expects success — adding the ticket's Layer #1 server gate naively will BREAK the e2e, so that fixture must be updated in the same change. 2) Commit c237c0e already ships a server-side fail-closed guard at POST /api/tasks rejecting empty title/description (server/src/index.ts:516-531) — this closes the empty-body half of the '45-byte directive' repro; the remaining dispatch follow-up is strictly the missing-@success_criteria case. Whoever picks up the dispatch-rejection item should not re-implement the title/description part. 3) Handoff line-number cites for the guard ('orchestrator.go ~3094-3105', '#1 CRITICAL orchestrator.go:3097') have drifted a few lines (guard now 3094-3114) — immaterial, but consistent with the branch's known line drift.

**claude-md-architecture:** 1) app.go:76-78 contains a stale in-code comment ('Planner and Observer only need bash; Assurance and QA need bash + view + grep') that contradicts tools.go's actual act_cli-based subsets — when reconciling CLAUDE.md, note the code comment is also wrong, though per the rules only scaffolding docs should be fixed this pass. 2) ChronologicalLog now also writes per-project JSONL logs (ChronologicalLog.ts:250) — an addition no CLAUDE.md section mentions. 3) The Tier 1 history model changed materially and is undocumented in CLAUDE.md: per-agent notebooks (app.go ~122-135) — Planner uses HistoryThread scoped to its own thread; Observer/Assurance/QA use HistoryNone (stateless snapshots). 4) MEMORY.md's 'Config fallback: AgentConfigForRole → developer ONLY' is confirmed; its '21 commands' figure shares the staleness of CLAUDE.md's. 5) Untracked repo-root binaries act-agent/act-tier1-{shim,planner,observer,assurance,qa_synthesizer} in git status are build artifacts of the new ACP shim — consistent with the Tier 1 ACP finding. 6) The Provider Configuration JSON example in CLAUDE.md shows only model/maxTokens fields; live Agent struct also has provider, backend, and acp fields (config.go:84-110) — the example understates the schema.

**kanban-vs-reality:** 1) go.mod module path is `github.com/paradiselabs-ai/ACT/act-agent` (act-agent/go.mod:1) — this contradicts the project memory/CLAUDE.md-adjacent note that the module path was 'kept as github.com/opencode-ai/opencode (UI rebrand only)'. Any doc repeating that is stale. 2) server/scripts/e2e-api.sh:122 POSTs a passing verdict with empty criteriaResults as a normal expected step — the e2e script actively encodes the fail-open behavior Fix 23 closed at the orchestrator layer; when the server-side gate lands this line will break, and the ticket's planned assertion should replace it. 3) act-tier1-planner being a full binary copy instead of the documented symlink (and 35 min newer than the shim) suggests a stale ad-hoc build; if the shim is rebuilt, the planner role keeps executing the OLD allowlist code. 4) The act-tier1-* artifacts and the two root HTML files are not gitignored, so they permanently pollute git status — relevant to the cleanup-constitution effort's 'where do finished artifacts live' question. 5) CLAUDE.md 'Project Structure' still lists 13 prompt files incl. per-provider details — not re-verified in this slice, but the models/ directory collapse (claim done-model-registry-deleted) means any doc enumerating internal/llm/models/*.go per-provider files is now stale.

**config-env:** (1) MEMORY.md states "go.mod module path: Kept as github.com/opencode-ai/opencode (UI rebrand only)" — this is now FALSE: live act-agent/go.mod:1 is `module github.com/paradiselabs-ai/ACT/act-agent` (confirmed in both the file and the test panic trace). Memory/doc should be updated. (2) go.mod says `go 1.25.8` while the installed toolchain is 1.26.1 — fine, but docs citing "Go 1.26.1" describe the toolchain, not the module directive. (3) The act-agent binary the global `act` symlink points to was built 2026-06-06 08:21, likely predating the newest commits on this branch (bc0673e etc.) — rebuild before any TUI e2e verification or the runtime won't reflect Phase 4. (4) ~/.act.json contextPaths is ["AGENTS.md"], while project CLAUDE.md says defaultContextPaths is ["ACT.md", "ACT.local.md"] — user config overrides the default, so any doc claiming ACT.md is what gets injected on this machine is wrong at runtime. (5) F-handoff's path citation "acp/agent.go" actually lives at act-agent/internal/acp/agent.go; runTurn line ~219 still exact.

**build-and-test:** MEMORY.md ("Key Architecture") claims the go.mod module path was kept as `github.com/opencode-ai/opencode` (UI rebrand only). The live /Users/user/Documents/Developer/dev/AI/act/act-agent/go.mod line 1 says `module github.com/paradiselabs-ai/ACT/act-agent` (go 1.25.8) — the module path HAS been renamed. That memory note is STALE. Also: the only failing Go test in the audited packages is the single TestLsTool_Run subtest; everything else in tools also passes (the package FAIL is driven entirely by that one panic, which aborts the test binary — so other tests in the same binary that hadn't run yet are not reported; a targeted `-run` exclusion would be needed to certify the rest of the tools package green).
