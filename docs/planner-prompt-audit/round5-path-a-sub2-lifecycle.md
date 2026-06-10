# Round 5 — Path A — sub2 — Lifecycle, On-Demand Sections, Tool Surfaces

Scope: 4 lifecycle triggers + 5 on-demand section bodies + 3 tool/CLI invocation surfaces (12 entries).

---

## 1. Per-entry close read

### Lifecycle (4)

**1. `system_prompt_rebind`** — Not a prompt; side-effect call that invalidates `contextLoaded` and `RebindSystemPrompt()`s all 4 Tier 1 agents. Post-Round-4: Fix 5.1 wired ACP discard, Fix 11 kept `contextContent+contextHash` across invalidation so hash-compare skips rebuild. CORRECT. SUSPICIOUS: `IsBusy` guard silently no-ops when Planner is mid-turn during the PROJECT_BRIEF handoff — audit notes *"Failure is logged at WARN; not retried."* A busy Planner that misses the rebind keeps intake-era worldview through the next BUILD turn.

**2. `human_input_passthrough`** — `text` goes verbatim through `runAgentTurn`; `resumeContext` prepended on first turn; Planner bypasses `InternalPromptMarker`. CORRECT for cap-reset behavior. SUSPICIOUS: *"Attachments forwarded raw to agentSvc.Run with no size limit or truncation"* — a multi-MB drop becomes input-token cost on every cached cascade. No bound, no warning.

**3. `resume_context_prepended`** — Now `renderBriefContext("resume", brief, tasks)`: SPIL `@brief / @inFlightTasks / @completedTasks` block, closes with *"Do NOT re-emit task-creation directives for the task IDs above — they already exist on the server."* Fix 3.3 CLOSED. STILL SUSPICIOUS: *"[SYSTEM] Resuming project %q. A project brief already exists on the server — do NOT run intake."* literal lands in chat (7.1 ACTIVE). If `Description` AND `TechStack` are both empty, the `@brief` block may emit nothing — Planner infers project nature from task titles alone.

**4. `build_mode_trigger`** — *"[SYSTEM] Project %q has been created. Switch to BUILD mode now: decompose the brief below into tasks and emit task-creation directives for each one (you know the shape). Do not ask for confirmation — start creating tasks immediately."* Fix 3.2 + 7.2 CLOSED. SUSPICIOUS: *"you know the shape"* is a referential dependency on `basePlannerPrompt`'s CREATE_TASK shape block. Combined with the rebind that just fired, the Planner reloads its base — but for smaller models this is exactly when placeholder `CREATE_TASK: {}` is likeliest.

### On-demand sections (5)

**5. `section_evidence_routing`** — Solid PVM-routing guide. Two confirmed drift items: line 22 reads literally *"If you recieve information that a task fails"* (typo, low). Line 27 reads *"`act pvm search "<query>"`"* — neither backend accepts the bare `act` form. In-process: `{"subcommand":"pvm","args":["search","..."]}`. ACP: `act-tier1-planner pvm search ...`. A Planner that copies verbatim into a tool call fails.

**6. `section_success_criteria`** — Tight rules, good/bad examples, no "etc." rule. No drift. Only gap: when Assurance returns 100% pass on empty criteria (fail-open), this section is about *writing* not *interpreting verdicts*.

**7. `section_nomik`** — Same `act codebase X` shorthand drift as evidence_routing — 5 occurrences confirmed in source. Wrapped in backticks for prose readability but neither backend executes that form. SUSPICIOUS: *"Nomik disabled or error → fall back to file-level reasoning"* is correct but doesn't say HOW to fall back.

**8. `section_validation`** — Captures FAIL once/twice/three-times decision tree clearly. Audit correctly flags: section doesn't address the **fail-open** failure mode (empty `@success_criteria` → 100% auto-pass). A Planner suspicious of a perfect verdict has nowhere to turn here.

**9. `section_examples`** — **HIGH severity drift** confirmed in source: `planner_section_examples.go:33` shows `@dependencies` INSIDE the EXAMPLE_TASK's description string. Base prompt explicitly forbids this: *"Do NOT use @context, @dependencies, or any other @-section in the description string — they break the JSON parser silently."* The examples section is exactly what a Planner pulls when shape is unclear — and it demonstrates the prohibited pattern.

### Tool surfaces (3)

**10. `expand_prompt_section_tool_inprocess`** — Native Go tool via `Tier1ToolsForRole`. Enum from `prompt.SectionNames()`. `TestPromptSectionAdvertisementMatchesRegistry` closes drift risk. Description carries tight WHEN/WHEN-NOT-TO-USE — exemplary tool surface. CORRECT.

**11. `prompt_section_cli_acp`** — Mirrors in-process tool for ACP via `act-agent prompt-section <name>` CLI, gated by `tools.IsAllowed('planner', 'prompt-section')`. Reads same `sectionRegistry`. CORRECT. SUSPICIOUS: ACP path adds shim Bash framing around the same content — for the Planner LLM that envelope difference is real.

**12. `act_cli_tool_schema`** — Enum from `AllowedSubcommandHeads('planner')` collapses `task retry`+`task abandon` to one `task` head. CORRECT per Fix 1.1+1.2. STILL SUSPICIOUS (audit notes): enum still includes `status`, `context`, `log` — the very commands base prompt's 1.3 ACTIVE tells the Planner not to use for human queries. Schema offers, prose forbids ~5K tokens away.

---

## 2. Cross-cutting themes

**A. Same-content, two-shorthand-paths drift in section bodies.** Three sections (`evidence_routing`, `nomik`, and the prose around `examples`) write CLI invocations as `act X Y` — user-facing CLI shorthand. Neither Planner backend uses that form: in-process is JSON tool call; ACP is `act-tier1-planner X Y` via Bash. Sections were written as if a human were reading them; actual audience is the Planner LLM which cannot translate `act pvm search` to the right envelope without an extra inference step. Risk: Planner pulls a section, copies the suggestion, gets a tool-validation error.

**B. Lifecycle text is loud about state but silent about *how*.** Resume + BUILD triggers restate the brief well (Fix 3.2/3.3) but defer "decompose into tasks" mechanics to "you know the shape" or recall. The `examples` section holds shape — but lifecycle triggers don't *invite* the Planner to pull `examples` on first BUILD turn. Combined with theme C's `examples`-shape contradiction, first BUILD turn is unusually risky.

**C. Tool schema offers > prose constraints.** `act_cli` schema lists `status/context/log` despite prose forbidding their use for human queries (1.3). `expand_prompt_section` advertises all 5 sections, but `examples` violates the base prompt's `@dependencies` rule. Pattern: schema/section breadth is uncoupled from prompt-narrative constraints. Schema/section narrowing would shrink hallucination surface more than any prose tightening.

**D. In-process vs ACP parity at the content layer, divergence at the envelope layer.** Fix 1.2 made `sectionRegistry` the single source — content drift impossible. But in-process gets a structured tool result; ACP gets a Bash tool result with shim framing. A section saying "Use this output to make a decision" reads differently when wrapped in shell output. No mitigation today.

---

## 3. Drift surfaces NEW since prior audit

- **`section_examples` `@dependencies`-in-description shape contradiction** — HIGH. The examples section actively teaches the wrong shape; highest-leverage finding in this subset.
- **`section_evidence_routing` and `section_nomik` use `act` shorthand the Planner can't execute** — MEDIUM. Won't crash; burns turns on retries, erodes Planner confidence in pulling sections.
- **`section_validation` silent on Assurance fail-open** — MEDIUM. 100%-on-empty edge case is exactly when guidance is most needed.
- **`section_evidence_routing` typo "recieve"** — LOW. Signal of insufficient review of extracted section bodies.
- **Resume's `[SYSTEM]` literal visible to user (7.1 ACTIVE)** — MEDIUM. Hits more chats now because Round 2 made resume far more verbose.
- **`build_mode_trigger` "you know the shape" deferral** — MEDIUM. Combined with bad `@dependencies` example, this is where first-BUILD-turn placeholder CREATE_TASKs originate.
- **`human_input_passthrough` attachments unbounded** — LOW for normal use, but a single 50MB drop blows out cache window indefinitely.

---

## 4. Verification of closed entries

- **3.2 + 3.3 (Fix 5 / 3f0e8dd) — `renderBriefContext`** — VERIFIED. `orchestrator.go:1171` defines the helper with `kind in {resume, build, default}`. Resume at `:331` passes `tasks` from `client.ListTasks()`. Build at `:1329` passes `nil`. All 5 brief fields emitted conditionally. Closing guard line present.
- **1.2 (Fix 4 / c853932) — Section registry + ACP CLI parity** — VERIFIED. `prompt/sections.go` defines `sectionRegistry`, `SectionNames`, `GetSection`. `cmd/root.go::runPromptSection` reads same registry. Allowlist entry confirmed. Both backends genuinely converge on content.
- **7.2 (Fix 13 / 805fd4e) — `CREATE_TASK:` literal scrub** — VERIFIED. `build_mode_trigger` at `:1178` reads *"emit task-creation directives for each one (you know the shape)"*. Literal `CREATE_TASK:` no longer in `renderBriefContext`.
- **5.1 (Fix 1 / 770a290) — ACP rebind discard-sessions** — Not directly re-verified at code level (out of subset code-read scope), but `system_prompt_rebind` entry text correctly describes the flow.

All [FIXED] closures in this subset match resolution-note claims.

---

## 5. Top 3 ranked drift surfaces

1. **`section_examples` shows `@dependencies` inside the description string** — directly contradicts the base prompt's shape rule, and `examples` is exactly what a confused Planner pulls. HIGH leverage, scoped fix (move the `@dependencies` block to a top-level JSON `dependencies` property).
2. **`section_evidence_routing` and `section_nomik` use the bare `act` user-CLI shorthand neither backend executes** — Planner pulls section, copies command, tool-validation fails. Cluster fix: rewrite CLI examples in the in-process JSON-call shape with a note that ACP users prepend `act-tier1-`.
3. **`build_mode_trigger`'s "you know the shape" + missing invitation to pull `examples`** — first BUILD turn is the high-leverage moment for placeholder CREATE_TASK regressions. Adding an explicit *"If unsure of shape, call expand_prompt_section('examples') first"* clause to the trigger is one-line and defuses it.
