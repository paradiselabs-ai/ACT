# Round 5 — Path A sub1 — Static composition + identity (8 entries)

Subset: the always-on system-prompt fragments + the once-per-session ACP priming. Branch state: post-Round-4 (14 fix-commits). Focus: drift surfaces the prior 27-entry audit missed, or new drift the recent fixes introduced.

---

## 1. Per-entry close read

### 1.1 `base_planner_prompt_fragment` (planner.go:24)
~6K tokens — INTAKE/BUILD modes, role-count guidance, SPIL shape, JSON rules, act_cli envelope.
**Correct post-R4:** role-count loophole closed (*"ALWAYS \`developer\`, unless the script is explicitly an HTTP server or a DB-backed API"*); dependencies pinned (*"ALWAYS an array of strings ... Use [] when none — do NOT use null, do NOT use \"\", do NOT omit the field"*); ACP `prompt-section` fallback present.
**Still suspicious:** *"DO NOT run act_cli to answer the human's status/log/swarm queries"* sits ~3K tokens AFTER the offered affordances in the separate fragment — never co-located. Also: this fragment names `{"subcommand":"prompt-section","args":["<name>"]}` as ACP-side, but `act_cli_commands_fragment` never lists `prompt-section` — a self-contradiction within the composed prompt.

### 1.2 `act_cli_commands_fragment` (common.go:79)
Canonical command list; header *"do NOT shell out to send messages. CREATE_TASK and PROJECT_BRIEF are markers in your reply text, not shell commands."*
**Correct post-R4:** `task retry` / `task abandon` added; enumeration is the single source after Fix 2.3.
**Still suspicious:** `act-agent prompt-section <name>` is NOT enumerated despite being on `AllowedFor('planner')` (the ACP `renderShimNote` DOES list it). In-process and ACP planners disagree on a real allowed subcommand. The *"do NOT shell out"* line is also directly contradicted by ACP priming's *"Use it via Bash for all ACT-coordination subcommands"*.

### 1.3 `coordination_constraints_fragment` (common.go:193)
6-bullet NEVER list (no spawning, no monitoring, no validation, no assembly, SPIL+@success_criteria).
**Correct post-R4:** single source after Fix 2.2; `TestBasePlannerPromptNoFragmentDuplication` guards.
**Still suspicious:** *"NEVER monitor the ChronLog yourself"* is in tension with the fragment's offer of `act-agent log --tail 20` — neither fragment explains what `log` is FOR. Reader has to cross-reference base prompt's "routing evidence during decomposition" clause living several thousand tokens away.

### 1.4 `env_block_fragment` (common.go:26)
`<env>cwd: %s\ngit: %s\nplatform: %s\ndate: %s\n</env>`.
**Correct post-R4:** date is ISO 8601 UTC via `time.RFC3339` (Fix 8.1).
**Still suspicious:** `git` is a binary Yes/No, not a branch/commit/dirty signal. For a coordination-aware Planner that's expected to reason about in-flight work, "git: Yes" carries near-zero signal. Pre-existing but never flagged.

### 1.5 `project_context_fragment` (prompt.go:53)
`"\n\n# Project-Specific Context\nFollow the instructions in the context below:\n%s"` with ACT.md + ACT.local.md inlined.
**Correct post-R4:** deterministic walk + SHA-256 hash skip preserves Anthropic ephemeral cache key.
**Still suspicious:** *"Follow the instructions in the context below"* is unconditional — no precedence vs base rules (7.3 STILL-ACTIVE). New concern: `InvalidateContextCache` no longer clears `contextContent` — if a path is REMOVED from `contextPaths` config (not just edited), cached content may persist until the hash differs, and the absent file can't shift it.

### 1.6 `acp_priming_wrapper` (app.go:261)
`InternalPromptMarker + doNotRespondHeader + base + renderShimNote("planner")`. Shim note enumerates 10 allowed subcommands.
**Correct post-R4:** allowed list live from `AllowedFor("planner")` (Fix 5.4); do-not-respond header present; stopReason switch routes non-end_turn primes to WARN.
**Still suspicious:** *"Use it via Bash for all ACT-coordination subcommands"* directly contradicts the embedded fragment's *"do NOT shell out to send messages"* — both ship in the SAME composed text. Also: the embedded base prompt advertises `expand_prompt_section` as a TOOL; ACP Planners have no such tool and must use `act_cli prompt-section`. The composed text never tells the ACP Planner the in-process tool is unavailable; the ACP-fallback note is buried at the end of `# On-demand reference material`.

### 1.7 `static_system_prompt_inprocess` (prompt.go:18)
Assembly recipe — no raw text of its own.
**Correct post-R4:** ACP rebind (Fix 5.1) propagates via session discard; hash-skip keeps cache key stable.
**Still suspicious:** `provider` arg to `PlannerPrompt(provider)` is **ignored** (runtime_substitutions: *"ignored by PlannerPrompt (arg present for signature parity only)"*). ACP and in-process get byte-identical composed bodies. Any future backend-aware composition has nowhere to branch — latent foot-gun.

### 1.8 `acp_priming_prompt` (acp/agent.go:340)
Wrapped prompt sent as FIRST user-role message in a fresh ACP session.
**Correct post-R4:** marker + do-not-respond header in place; stopReason switch logs hallucinations.
**Still suspicious:** entry notes *"ACP host UI may show as first chat bubble."* — the do-not-respond header is English INSIDE the bubble, so host UI still renders the full ~6K-token prompt verbatim to the user. Mitigated, not fixed.

---

## 2. Cross-cutting themes in this subset

### Theme A — Fragment co-location is still ad-hoc
Every fragment that ships affordances (act_cli_commands_fragment) is structurally divorced from the constraints that govern them (base prompt's "decomposition-only" rule; coordination_constraints' NEVER list). Round 3 dedup collapsed duplication but **did not co-locate** — it just picked the canonical copy. Reader still sees affordance → ~3K tokens unrelated → prohibition. Pattern persists for `act-agent log`, `act-agent status`, and now `prompt-section`.

### Theme B — ACP/in-process drift is one-sided
ACP-specific surfaces (renderShimNote, doNotRespondHeader, prompt-section CLI fallback) have all been retrofitted to match in-process truth. But IN-PROCESS surfaces never branch on backend — `actCLICommands('planner')` returns identical text regardless of consumer. Result: the ACP wrapper has to inject corrections (the "Use it via Bash" line) that CONTRADICT the unmodified fragment it ships alongside. The `provider` arg in `PlannerPrompt(provider)` already exists for the branch but is unused.

### Theme C — Live-from-source drift prevention is partial
Round 3 fixes pioneered the pattern: `renderShimNote` reads `AllowedFor()` live, `TestPromptSectionAdvertisementMatchesRegistry` locks the prompt's section list to `SectionNames()`. But the pattern is NOT extended to `act_cli_commands_fragment` — its enumeration in `common.go:79` is hand-written. Adding a whitelisted subcommand updates `AllowedFor()` and `renderShimNote()` (drift-locked) but not the in-process command list. That's exactly how `prompt-section` reached the allowlist and the priming while staying missing from the fragment.

### Theme D — Context-write surfaces have no precedence guard
`project_context_fragment` lands LAST in the composed prompt with the unconditional *"Follow the instructions in the context below"*. The only surface in this subset where runtime content (the project's ACT.md) can countermand framework rules — and it's the surface with the strongest "obey" framing and the most LLM-anchored position.

---

## 3. Drift surfaces NEW since the prior audit

### D1. `prompt-section` advertised in base + priming but missing from `act_cli_commands_fragment` — HIGH
Same drift class as the now-fixed Fix 5.4 (allowlist↔advertisement parity), uncaught for the in-process fragment surface. ACP Planners reading the fragment for available commands won't find `prompt-section` even though the allowlist accepts it.

### D2. `renderShimNote` "Use it via Bash" contradicts fragment's "do NOT shell out" — HIGH
Both ship inside the SAME composed ACP priming. Head-on contradiction, not a separation-by-distance issue.

### D3. `InvalidateContextCache` doesn't handle path-set deletions — MEDIUM
Fix 2.4's hash-skip is correct for content-change detection but doesn't address `contextPaths` config shrinking. If a path is REMOVED, the cached content for it may persist until the hash differs.

### D4. `PlannerPrompt(provider)` ignores `provider` arg — MEDIUM
Signature implies backend-aware composition; impl is hard-coded. Latent foot-gun for any future ACP-vs-in-process branch.

### D5. `git: Yes/No` env field is signal-poor — LOW
For NesTTY's branch-awareness ambitions, "git: Yes" is missed observability surface.

---

## 4. Verification of closed entries

- **2.2 + 2.3 (basePlannerPrompt trim):** Verified. No "Reacting to other roles" block, no duplicate "Allowed subcommands:" line. Match.
- **2.4 + 8.2 (deterministic walk + hash skip):** Verified by runtime_substitutions note *"Sequential read order, sort.Strings within directory walks ... SHA-256 of rebuilt content compared to contextHash."* Match.
- **5.2 + 5.3 (ACP priming hygiene):** Verified — `InternalPromptMarker` and `doNotRespondHeader` present in raw_text. Match.
- **5.4 (priming from allowlist):** Verified — *"renderShimNote generates the allowed-subcommand list live from tools.AllowedFor('planner') — no hardcoded text."* Match.
- **1.2 (`expand_prompt_section` parity for ACP):** PARTIAL DRIFT. Base prompt names the act_cli fallback; allowlist includes `prompt-section`. **BUT** `act_cli_commands_fragment` never got the corresponding enumeration line. Drift surface D1 is the unclosed remainder.
- **7.2 (CREATE_TASK literal scrub):** Verified for orchestrator-authored surfaces (not in this subset). Within this subset, literals here are legitimately instructional. Match.
- **8.1 (date format):** Verified — `time.RFC3339` with `.UTC()`. Match.

---

## 5. Top 3 ranked drift surfaces

1. **D1 — `prompt-section` missing from `act_cli_commands_fragment`.** Trivial line-add; blocks ACP/in-process command-list parity. HIGH leverage because the live-from-registry pattern (`TestPromptSectionAdvertisementMatchesRegistry`) could be generalized to lock the fragment enumeration too.

2. **D2 / Finding 3.5 — "Use it via Bash" vs "do NOT shell out" in one composed prompt.** Direct contradiction in every ACP Planner turn. Fix: branch the fragment via the unused `provider` arg, or rewrite to be backend-agnostic.

3. **7.3 — `project_context_fragment` can countermand base rules.** Only surface in this subset where runtime content can override framework rules — and it's the strongest-positioned (last, with *"Follow the instructions"* framing). MEDIUM but unique.
