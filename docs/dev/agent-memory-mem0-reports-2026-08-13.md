# Agent Memory — Source Reports (mem0.ai, read 2026-08-13)

> Two background research passes over four mem0.ai articles. Each idea is classified
> DIRECTLY IMPLEMENTABLE / CONCEPTUAL (needs research) / MARKETING, so nobody
> fills a spec gap with guesswork. Companion analysis:
> `docs/dev/agent-memory-swap-analysis-2026-08-13.md`. Audit of ACT's own memory:
> `docs/audits/memory-system-audit-2026-08-13.md`.

---

# Agent Memory: Two Mem0 Articles, Read for Engineering Content

**Fetch note.** Both pages were retrieved with WebFetch. Article 1 returned *empty body* on the first fetch (metadata only) and full body on a retry with a trailing-slash URL — content below is from the successful retry. Article 2 fetched cleanly twice. Neither is paywalled. Everything below comes from the fetched text; where I mark something "not in the article," that is a genuine gap in the source, not a gap in retrieval. Nothing is filled in from prior knowledge except where explicitly labeled as my own analysis (the Fellegi-Sunter critique and the research pointers).

---

## Article 1 — "Mem0 vs. Building Your Own Vector Store for Agent Memory"

### What problem it claims to solve

That a vector store is storage, not memory. The article's decomposition: a production memory system must handle **identity** (profiles, shared vs. private), **extraction** (conversation logs → structured retrievable facts), **lifecycle** (retention, summarization, growth), **retrieval** (ranking, filtering, weighting per task), and **tooling/governance** (debugging, monitoring, privacy, migration). A vector DB gives you one of five; the other four become application code that "tends to expand over time."

### The mechanism

The article specifies a roll-your-own stack concretely enough to build:

**Row schema (single table/collection):** `id`, `user_id`, `agent_id`, `embedding`, `text`, `type`, `created_at`, `updated_at`, plus metadata `source`, `confidence`, `tags`. Candidate backends named: Postgres+pgvector, Qdrant, Milvus. No index type, distance metric, or dimension is specified.

**Write pipeline** (four stages): decide whether the turn is worth storing → LLM fact-extraction → embed each extracted fact → insert. The extraction prompt is given nearly verbatim: from a dialog turn, extract "stable user facts, preferences, and long-term relevant information," respond as a JSON list of strings. Note the granularity choice this encodes — one row per extracted *fact*, not per turn.

**Read pipeline** (four stages): build a retrieval query from the user message plus context → vector search *within scope* → filter and rank → inject into prompt. The worked Mem0 example shows retrieval as `query(user_id, query, limit=5, filters={"type": ["user_fact", "preference"]})` and injection as a second system message containing a bulleted memory block, with the literal fallback string "No prior memory." when the set is empty.

**Six components** of the DIY stack, enumerated: storage, embedding pipeline (model selection, preprocessing, normalization, **versioning**), extraction, identity/scoping ID schemes, retrieval strategy (similarity + filtering + dedup + merging), lifecycle (expiration, archival, compression, re-embedding on model change).

**Scoping in the sample code** is done by namespacing the external ID: `f"app-user:{raw_user_id}"` — a prefix convention, not a schema.

### Numbers

Almost none, and the ones present are unmeasured. **"2-3 weeks initial, then ongoing maintenance"** for DIY engineering effort — no basis, no team size, no scope definition. **`limit=5`** as the retrieval k in the sample. **`gpt-4o-mini`** as the chat model in the sample. **"Sub-10 ms"** is named as the latency regime where a hosted memory layer's extra network hop is disqualifying and inline caching is required — this is the single most useful number in the piece and it is a stated boundary condition, not a measurement. There are zero recall, accuracy, token-count, or cost figures. There is no benchmark of any kind.

### Failure modes named

- **Prompt drift** — extraction prompts silently degrade when the underlying model changes. The article correctly frames extraction prompts as "critical product artifacts."
- **Over-extraction** — storing low-value facts; **under-extraction** — missing implicit preferences. Both asserted, neither measured.
- **Storage grows linearly with traffic**, embedding costs climb, and **retrieval quality degrades as old information competes with new**. This last one is the real mechanism claim: unbounded memory is a *precision* problem, not just a cost problem.
- **Identity coupling makes schema migrations risky** — `user_id`/`agent_id` hardcoded across the codebase means identity leaks everywhere and can't be refactored.
- **Retrieval config becomes an embedded DSL** — k-values, thresholds, per-type/recency/source/confidence filters, rerankers, and per-agent overrides accumulate into an unmaintainable configuration language inside application code.
- **Stated limits of the whole pattern** (unusually honest for vendor content): hard factual knowledge (docs, policies) needs a separate RAG path; highly structured domains (trading, medical) should derive memory from canonical DBs, not from conversation; sub-10ms paths need inline caching; compliance environments still owe their own storage-footprint and approval work.

### Asserted without evidence

Every claim in the comparison table's Mem0 column ("built-in scopes," "centralized retention," "native shared memory support," "memory-centric introspection views") is a feature assertion with no described mechanism. The "2-3 weeks" figure is invented. "Mem0 automatically extracts useful memories" — *useful* is undefined and unmeasured. The FAQ is pure positioning. The framing "memory layer that sits on top of storage" (Mem0 FAQ) is a category claim, not an architecture.

### Implementation Readiness — Article 1

**DIRECTLY IMPLEMENTABLE (4)**

1. **The memory row schema.** Build the table exactly as listed. Acceptance test: insert 100 extracted facts across 3 `user_id`s × 2 `agent_id`s, confirm scoped query returns only the matching partition.
2. **The write pipeline with the given extraction prompt.** Order: extraction → embed → insert. Acceptance test: feed 20 hand-labeled dialog turns with known ground-truth facts, assert JSON parses on all 20 and extracted-fact recall against labels is reported (not thresholded — you have no target, see Open Questions).
3. **The read pipeline with type-filtered scoped search.** Order: build query → scoped ANN search → filter by `type` → inject as a distinct system message with an explicit empty-state string. Acceptance test: with an empty store the prompt contains the fallback line and the model does not hallucinate prior context.
4. **Metadata-tagged writes** (`source`, `direction`, `type`) as first-class filter keys rather than free-form JSON. Acceptance test: a query filtered to `type in {user_fact, preference}` excludes raw `chat` turns.

**CONCEPTUAL — NEEDS FURTHER RESEARCH (5)**

1. **Lifecycle / retention.** The article names TTLs, summarization jobs, and re-embedding scripts as things people build — it specifies none of them. Missing: what triggers summarization (count? age? token budget?), what the summarization prompt is, whether summarized originals are deleted or tombstoned, and the decay function (there is none). Next step: read the Mem0 OSS repo's `memory/main.py` ADD/UPDATE/DELETE decision logic and the Letta/MemGPT paper's core-vs-archival tiering and eviction policy — those are the two published designs that actually specify eviction.
2. **Retrieval tuning.** Named variables (k, similarity threshold, recency/confidence filters, reranker, hybrid scoring) with zero values or formulas. Missing: the fusion formula combining similarity with recency and confidence. Next step: implement Reciprocal Rank Fusion over dense + BM25 as the baseline and A/B against pure-dense on your own logged queries; there is no number in this article to target.
3. **Extraction quality control.** "Custom taxonomies, validation jobs, manual inspection tools" is a list of nouns. Missing: the taxonomy itself, what a validation job asserts, and any quality metric. Next step: build a labeled eval set of turns→facts before writing extraction code; measure over/under-extraction rates as your own baseline. LoCoMo is the benchmark Mem0 publishes against elsewhere — it is *not* cited in this article.
4. **Cross-agent shared memory and access levels.** "Shared versus private data" and "teams need shared knowledge bases plus private memories" — no ACL model, no join semantics, no conflict rule when shared and private memories disagree. Next step: specify a read-scope resolution order (private > team > global) and test contradiction handling explicitly; nothing in the article helps.
5. **Embedding-model versioning and re-embed migration.** Named as a requirement, unspecified as a procedure. Missing: whether to dual-write, how to serve during backfill, how to compare score distributions across model versions. Next step: design a `embedding_model_version` column plus shadow-index backfill; validate by measuring recall@k on a frozen query set before and after.

**MARKETING / NOT ACTIONABLE (3)**
The full Mem0 column of the comparison table; the "2-3 weeks" effort figure; the six-question FAQ.

---

## Article 2 — "Cross-Session Identity Resolution in Agent Memory"

### What problem it claims to solve

Memory keyed to a *session or machine* is not memory keyed to a *person*. The article's opening case: two people share a computer, A prefers dark mode, B prefers light; if scope is the machine, B overwrites A. The load-bearing sentence: passing a `user_id` "isn't the same as knowing who the user is" (Mem0). The store resolves lookups by identifier; deciding *which* identifier a human maps to across sessions is explicitly left to the application.

### The mechanism

**Three distinct jobs**, which the article usefully separates because they have different failure costs:
- **Unify** — collapse a fragmented trail (guest token → email → SSO subject → second device) into one canonical identity.
- **Isolate** — guarantee two humans never collapse into one identity.
- **Merge** — fold duplicate identity graphs without loss or duplication once discovered.

**Scoping schema — four partition keys:** `user_id` (persistent account), `agent_id` (agent persona), `app_id` (product surface), `run_id` (single session or ticket).

**Scorer** — presented as Fellegi-Sunter (1969) probabilistic record linkage, given in full:

```
W_EMAIL, W_DEVICE, W_NAME = 1.0, 0.5, 0.3
LINK, REVIEW = 0.8, 0.4

def resolve(signals, profile):
    if signals.get("email") in profile["emails"]:
        return "link"                      # deterministic short-circuit
    score = 0.0
    if signals.get("device") in profile["devices"]: score += W_DEVICE
    if signals.get("name")   == profile["name"]:    score += W_NAME
    if score >= LINK:   return "link"
    if score >= REVIEW: return "review"
    return "new"
```

Three outcomes: `link`, `review`, `new`. Signal tiers: **deterministic** (verified email, authenticated SSO) short-circuit to link; **fuzzy** (device, IP, name, fingerprint) accumulate additively toward a threshold; the **review band** (0.4 ≤ score < 0.8) holds ambiguous cases for human approval. Device alone (0.5) lands in review by construction; device+name (0.8) links.

**Threshold rationale — asymmetric costs.** False negatives (missed unification) degrade UX silently and are recoverable. False positives (wrong merge) expose one user's private context to another and are a security incident. Hence: set the link bar high, let the review band absorb ambiguity, and "eat the occasional missed unification as the cheap mistake" (Mem0).

**Provisional memory.** Before resolution, an anonymous session writes under `run_id`, not `user_id`. Memories are **promoted** to the canonical identity only after the match clears the link threshold. Unmatched sessions **expire with the session** rather than persisting under a wrong identity. This is the single best idea in either article: it makes the default outcome data loss rather than data contamination.

**Merge:**

```
def merge(from_id, into_id):
    for m in mem.get_all(user_id=from_id)["results"]:
        mem.add(m["memory"], user_id=into_id)
    mem.delete_all(user_id=from_id)
```

Read-all, re-add under target, delete source. The article immediately qualifies it: keep **provenance** on what moved and which signals justified it, "so a bad merge can be traced and undone" (Mem0). The code as written has no provenance and is not undoable — the prose contradicts the snippet.

### Numbers

Model: **gpt-4o-mini**. Five checks against a live Mem0 store:

| Check | Result |
|---|---|
| Fuzzy recall, naive session ID | 0 memories |
| Fuzzy recall, resolved (device+name, score 0.8) | 1 memory recovered |
| Device-only session (score 0.5) | held for review, not linked |
| Cross-user leakage | 0 hits |
| Duplicate-graph merge | 1 memory moved, 0 stranded |

These are **n=1 smoke tests**, not benchmarks. "1 memory recovered" and "0 leakage over one trial" carry no statistical weight. No latency, no cost, no dataset size, no precision/recall over a population.

### Failure modes named

Shared-device collision (the opening scenario); **returning-user amnesia** (unresolved session returns 0 memories — demonstrated, at n=1); **cross-user leakage** from an over-eager merge; **stranded memories** after a partial merge; **bad merges quietly poisoning a real account** when provenance is absent.

### Asserted without evidence

That fuzzy matching prevents returning-user amnesia at scale (demonstrated once). That weights `1.0/0.5/0.3` and thresholds `0.8/0.4` are correct — they are hand-picked, never calibrated, and the article admits tuning is required without saying against what. That the review band is an effective safety valve — unquantified, and no queue exists. Calling the additive scorer "Fellegi-Sunter" is, in my assessment, a misattribution: real F-S weights are log-likelihood ratios `log2(m_i/u_i)` derived from match/non-match probabilities per field, usually fit by EM; this snippet is arbitrary additive weighting wearing the name.

### Implementation Readiness — Article 2

**DIRECTLY IMPLEMENTABLE (4)**

1. **The `resolve()` scorer verbatim** as a *v0 placeholder*. Ship it behind a flag with the weights as config, not constants. Acceptance test: table-driven — email-match→link; device+name→link; device-only→review; nothing→new; assert no path auto-links on a single fuzzy signal.
2. **Four-key partitioning** (`user_id`/`agent_id`/`app_id`/`run_id`). Acceptance test: a query at each scope returns exactly its partition; a `run_id` query never returns another run's rows.
3. **Provisional memory with promote-on-link and expire-on-no-match.** Order: write anonymous under `run_id` → on resolve=link, promote to `user_id` → on session end without link, delete. Acceptance test: two anonymous sessions on one device, only one of which authenticates; assert the unauthenticated session's memories are gone and none landed on the authenticated account.
4. **The five-check harness itself**, as a regression suite. Acceptance test: it runs in CI, and the cross-user-leakage check is a hard gate.

**CONCEPTUAL — NEEDS FURTHER RESEARCH (6)**

1. **Actual probabilistic linkage.** Missing: the m/u probabilities per field, the EM fitting procedure, and how to convert log-likelihood weights into the link/review cutoffs. Next step: read the **Splink** repo (Ministry of Justice, UK) — it is the reference open-source F-S/EM implementation — and **Zingg**; also **fastLink** (R) for the EM derivation.
2. **Blocking / candidate generation.** Completely absent. `resolve(signals, profile)` compares against *one* profile; nothing says how you pick which profiles to compare. Naively this is O(N) per session. Next step: read Splink's blocking-rule design and the Papadakis et al. entity-resolution blocking survey; decide on blocking keys (email domain, device hash prefix, name phonetic key) before writing any scorer.
3. **Signal acquisition and stability.** "Device, IP, name, fingerprint" — no collection method, no normalization, no drift model. A device ID that rotates makes W_DEVICE meaningless. Missing: fingerprint entropy and stability half-life. Next step: measure your own device-ID persistence over 30 days before assigning it a weight.
4. **Review queue.** Named as the safety valve, entirely unspecified — no storage, no SLA, no reviewer UI, no what-happens-to-memories-while-pending. Note this interacts with provisional memory: a review-band session's memories must not expire before the reviewer acts, and the article never addresses that. Next step: design the pending state explicitly; this is a spec you must write, not research.
5. **Merge provenance and reversibility.** The prose demands traceable, undoable merges; the code deletes the source. Missing: the provenance record shape, and whether merge should be a tombstone/alias-graph operation rather than copy-and-delete. Next step: model identity as a union-find / alias graph with an append-only edge log (edge = {from, into, signals, score, timestamp}) so unmerge is edge removal, not restoration from backup.
6. **Post-merge dedup and contradiction.** Merging two graphs will produce near-duplicate and directly contradictory memories (A: dark mode; A': light mode). The article names neither dedup nor conflict resolution nor decay. Next step: read Mem0 OSS's ADD/UPDATE/DELETE memory-operation prompt, which is the closest published mechanism for contradiction handling, and decide separately whether recency alone should win.

**MARKETING / NOT ACTIONABLE (2)**
"Identity as a first-class concern" positioning; the "returning user amnesia" branding of an ordinary cache-miss.

---

## Open Questions — must be answered before writing code

1. **Extraction granularity and cost.** One row per extracted fact means an LLM call per turn. What is the per-turn token and dollar cost, and does extraction run inline (latency on the write path) or async (staleness on the next read)? Neither article says.
2. **What is the retrieval scoring function?** Both articles name similarity, recency, confidence, and rerankers; neither gives a formula combining them. This must be specified before the read pipeline is real.
3. **Contradiction policy.** When a new fact contradicts a stored one, do you overwrite, version, or store both with recency weighting? Unaddressed in both articles, and unavoidable the moment merge exists.
4. **Are the identity weights (1.0/0.5/0.3) and thresholds (0.8/0.4) defensible for your signal mix?** They are hand-picked. What labeled linkage set will you calibrate them against, and what false-merge rate is acceptable?
5. **Blocking strategy.** Against which candidate profiles does a new session get scored, and what is the cost per resolution at your user count?
6. **Provisional-memory TTL vs. review latency.** How long does an unresolved session's memory live, and what happens if human review outlasts it?
7. **Is a hosted memory layer viable at all in your latency budget?** The article concedes sub-10ms paths need inline caching. Measure your budget first.
8. **Re-embedding migration.** What is the procedure when the embedding model changes, and can you serve during backfill?
9. **Deletion semantics.** GDPR-style "delete this user" across a merged identity graph — which rows does it reach, and do merged-away aliases still resolve? Both articles claim privacy/retention as a strength; neither describes the deletion mechanism.

---

# mem0.ai — Two-Article Technical Report: Agent Memory

**Fetch status:** Both URLs retrieved successfully via WebFetch (two passes each, second pass targeting code samples and FAQ). No paywall, no truncation notice. Everything below comes from the fetched content; where the articles omit something, that is stated as an omission, not filled from prior knowledge.

**Overriding caveat, stated once and applying to the whole first article:** the harness comparison names **no file paths, no databases, no config filenames** (explicitly checked: no CLAUDE.md, AGENTS.md, `.cursorrules`, memories file, or knowledge-base path appears anywhere), and **no numbers of any kind** (no token counts, no context-window sizes, no compaction thresholds, no latency, no benchmarks). It describes four *archetypes* — it literally uses the phrasing "Claude Code style", "Cursor style" in its own comparison table — not four reverse-engineered implementations. As competitive analysis input it is a taxonomy, not evidence.

---

## Article 1 — Harness Comparison (Claude Code / Cursor / Devin / Antigravity)

### What problem it claims to solve
Coding harnesses have three memory layers: **ephemeral context** (prompt window, inline snippets, cursor location, recent edits), **session memory** (files opened, commands run, tests executed, scratchpads, intermediate summaries), and **long-term memory** (architecture decisions, past tickets, user preferences, recurring patterns). The claim: every harness solves layers 1–2 and none exposes layer 3 as a generic API. Its thesis sentence — the bottleneck is "no longer model capacity but the quality of the harness memory stack" (mem0.ai).

### The mechanism — per harness

**Claude Code archetype — hierarchical summarization.**
- *Persists:* block summaries, file summaries, project summaries, conversation summaries.
- *Structure:* a fixed ladder — raw code → block → file → project → conversation.
- *Written:* automatically and periodically, triggered when token budgets are hit.
- *Read:* at prompt time; the harness selects which rung of the ladder each component enters the prompt at, per task.
- *Scope:* per-session, coupled to a workspace snapshot.
- *Named failures:* codebase changes invalidate parts of the hierarchy; long-term decisions and their rationale get "compacted away"; the same project a week later must resummarize or re-explain design choices. Struggles specifically with continuous refactors and with capturing human *intent*.

**Cursor archetype — raw context stuffing + aggressive retrieval.**
- *Persists:* a code/symbol index (embeddings or symbol graphs) and retrieval traces. Deliberately minimal intermediate summarization; summarization is lazy and confined to conversation history.
- *Format:* raw code blocks annotated with file paths — kept close to source truth.
- *Written/read:* per query. Retrieval fetches the most relevant files or blocks each time.
- *Scope:* per-query/per-session.
- *Named failures:* prompt budget caps effective codebase size (does not scale to large monorepos); repeated context fetching for similar queries wastes compute; cross-session continuity depends entirely on re-indexing and retrieval quality; cannot store durable conventions (article's example: a team preferring dependency injection).

**Devin archetype — checkpointing / execution state.**
- *Persists:* files changed, commands executed, tests run and their results, artifacts (logs, screenshots, outputs), plus progress notes — as a sequence of checkpoints.
- *Written:* after each "meaningful step".
- *Read:* on failure or interruption, reloading the last checkpoint.
- *Scope:* task/ticket-scoped only.
- *Named failures:* checkpoints are heavy and rarely reused across unrelated tasks; not intrinsically queryable across tasks; long-term preferences do not bubble up out of checkpoints; cross-task continuity is manual; sharing between harnesses is nontrivial.

**Antigravity archetype — artifacts as memory.**
- *Persists:* planning docs, test reports, code diffs, design diagrams, transcript segments — as first-class, versioned, linked objects with metadata (tags, embeddings).
- *Storage shape:* a graph or workspace.
- *Written:* the agent emits an artifact per meaningful thought or deliverable.
- *Read:* tag search, embedding similarity, or structured queries over artifact metadata.
- *Scope:* workspace-persistent, but cross-session continuity is implicit — not enforced by a memory API.
- *Named failures:* retrieval quality is hostage to tagging/indexing discipline; artifacts are task-specific noise rather than normalized reusable facts; without a separate distillation layer important patterns slip through; not all harnesses can ingest arbitrary artifact formats.

**Comparison table (structure as published):** columns = harness style | primary memory unit | scope | persistence model | strength | weakness. Rows: hierarchical summaries / file+project / auto compaction per session / scales over large codebases / rationale compacted away — raw code context / file+snippet / retrieval per query / accurate current-code view / limited cross-session continuity — checkpoints+logs / task / task-scoped history / strong within-task / weak cross-task reuse — artifacts+documents / workspace / graph or workspace storage / human-aligned representation / retrieval depends on artifact quality.

### mem0's proposed layer
Write API for structured memories + metadata; read API by text, metadata filter, or vector similarity; identity/scope control (user, project, agent, shared); persistence and versioning across sessions. Integration is **system-prompt augmentation**: fetch relevant memories before each LLM call and prepend to the system/assistant message; alternatively "memory hooks on key events". Write at *milestones* — design accepted, refactor completed, bug diagnosed, deployment stabilized. Backfill an existing harness by scanning design docs and ADRs. Code surface shown: `mem0.search(query=..., filters={"project_id":..., "user_id":...}, top_k=10)` returning `.get("results", [])`, and `mem0.add(payload)` with `content` + `metadata{project_id, user_id, type}` where `type` is e.g. `"architecture_decision"`.

### Numbers
**None.** Zero quantitative claims in the entire article.

### What is asserted without evidence
Nearly all of it. The per-harness mechanisms are asserted with no citation, no source-code reference, no experiment, and no version. The four "weakness" claims are plausible-sounding but untested. The "bottleneck is the harness memory stack" thesis has no measurement behind it. Marketing density is high: three-plus CTAs for a free API key, "no credit card", positioning mem0 as "the shared memory spine that the harnesses currently lack" (mem0.ai), and a Further Reading block of five internal links.

### Implementation Readiness — Article 1

**DIRECTLY IMPLEMENTABLE (3)**
1. *Three-layer memory taxonomy as an explicit ACT design constraint.* Build: label every existing ACT persistence site as ephemeral / session / long-term, and assert every one has exactly one owner. Order: audit first, then close gaps. Acceptance test: an inventory where each of ACT's stores (prompt context, ChronLog, PVM, AGENTS.md brief) maps to exactly one layer with no store spanning two.
2. *Milestone-triggered writes rather than continuous writes.* Build: emit a durable memory only on the four named events (design accepted, refactor complete, bug diagnosed, deploy stabilized). Acceptance test: run a task producing N tool calls; durable-store growth is O(milestones), not O(tool calls).
3. *Metadata-scoped retrieval.* Build: every durable record carries `{project_id, user_id, agent/role, type}`; retrieval always filters before ranking. Acceptance test: a query scoped to project A returns zero project-B records even when B is semantically closer.

**CONCEPTUAL — NEEDS FURTHER RESEARCH (3)**
1. *Hierarchical summarization ladder (block→file→project→conversation).* Missing: the level-selection policy (what function picks the rung per component per task), the summarize trigger threshold, the invalidation rule when a file changes, and the summary size budget per rung. Next step: read Claude Code's actual auto-compact behavior and LangChain/LlamaIndex `SummaryIndex` + `DocumentSummaryIndex` retriever code for a concrete level-selection implementation; benchmark against LoCoMo or LongMemEval rather than trusting the ladder shape.
2. *Artifact-graph memory with tag+embedding retrieval.* Missing: the graph schema (node/edge types), what makes an artifact retrievable, and the "separate distillation layer" the article says is required but never specifies. Next step: read Graphiti's episode→entity extraction implementation (already flagged in ACT memory as the shape to steal) for concrete node/edge typing.
3. *Cross-harness shared memory spine.* Outcome described, mechanism absent — the "shared demo architecture" section is four sentences of who-writes-what with no protocol, no conflict rule, no schema. Bluntly: outcome without mechanism. Next step: define the record schema and write-conflict rule yourself; no research artifact in this article helps.

**MARKETING / NOT ACTIONABLE (2)**
1. "Mem0 treats memory as a first-class service" / "shared memory spine" positioning.
2. The FAQ block — six questions whose answers restate the pitch (use harness for transient, mem0 for durable) with no added engineering content.

---

## Article 2 — Event-Based Memory for Long-Running Agents

### What problem it claims to solve
Background agents running hours-to-days accumulate tool calls, failures, and corrections where task progress matters more than what a chat-memory extractor was built to pull out. Two concrete harms: token waste from re-deriving the same conclusion off raw session tails on every turn, and **repeating an already-failed action** because the log recorded the attempt but not the outcome. The article's sharpest line is the framing: knowing you applied the patch does not stop a second identical attempt; knowing it failed with an ImportError does.

### The mechanism
An append-only ledger of typed events. Schema (Pydantic, field names verbatim):

```python
class AgentEvent(BaseModel):
    event_id: str
    timestamp: str = Field(default_factory=lambda: datetime.utcnow().isoformat())
    actor: str                        # "coding_agent_v1"
    event_type: str                   # "tool_execution_failure"
    task_id: str
    action: str                       # "apply_git_patch"
    payload: Dict[str, Any]           # {"file": "auth.py", "error": "ImportError"}
    learned_constraint: Optional[str] # "auth.py requires pyjwt>=2.0"
```

The design point in the schema: `action` (what was attempted) is separate from `payload`/`event_type` (what actually occurred), plus an optional slot for a distilled durable fact.

Three patterns:

- **Write (append raw).** `mem0.add(messages=[...], user_id=event.task_id, metadata={"category": event_type, "action": action, "is_active": True}, infer=False)`. `infer=False` is the load-bearing flag — it stores the exact receipt with **no LLM extraction**, so the write stays off the latency path inside a retry loop. Note the scoping hack: `task_id` is passed as `user_id`.
- **Read (before retry).** `mem0.search(query=f"failures on {file_path}", user_id=task_id)` → `[e["memory"] for e in results]`. The same call transparently returns raw receipts pre-compaction or condensed lessons post-compaction; callers do not branch on pipeline stage.
- **Maintain (compaction).** `mem0.add(messages=raw_event_logs, user_id=task_id, metadata={"type": "compacted_lesson"}, infer=True)` — batch the raw events, let the LLM extract the persistent pattern, replace repetitive receipts with one consolidated memory.

**State derivation.** Current state = the set of non-superseded memories. **Supersede** marks a contradicted memory outdated-and-dated rather than deleting it; raw history is never rewritten. Retrieval flag: `latest_only=True` → current beliefs only; `latest_only=False` → full history including superseded entries.

**Synthesis** is a separate background feature (Pro plan and above), up to **one-day latency**, and works **only** on memories scoped to bare `user_id` — not `agent_id`, not `run_id`. That constraint collides directly with the article's own pattern of jamming `task_id` into `user_id`.

**Compaction trigger.** Not a threshold — an explicit judgment call keyed to the cost of a blind retry. Cheap idempotent actions tolerate a few blind repeats; an action with real side effects, a deploy, "should compact on the first failure, not the third" (mem0.ai).

### Numbers
One figure, and it is soft: ~**12,000 tokens** of raw session tail versus a "handful" of active constraints after compaction. No measurement conditions given — no model, no task, no methodology, no repeat count. No latency, recall, accuracy, or cost numbers anywhere. The one-day Synthesis latency bound is the only other quantity.

### Failure modes named
- **Stale constraints.** Supersede is reactive only — it fires when contradicting evidence arrives and never goes looking. Constraints that quietly stopped being true persist indefinitely; the article's remedy is "build in an occasional re-validation" with no cadence, no trigger, no mechanism.
- **Scope/feature collision.** Synthesis silently does nothing for `agent_id`/`run_id`-scoped memory.
- **Blind repeats** before the compaction point, by design, on cheap actions.
- **Unaddressed:** concurrent writes, contradiction between two simultaneously-live constraints, partition tolerance — none discussed.

### What is asserted without evidence
The 12K-token comparison (no conditions). That event-based memory beats chat-memory extraction for long-running agents (no A/B). That the ledger is "durable" and "lossless" — asserted, not demonstrated, and no retention policy or replay semantics are specified anywhere despite the append-only framing implying both. That the approach generalizes to any multi-session agent.

### Implementation Readiness — Article 2

**DIRECTLY IMPLEMENTABLE (4)**
1. *Outcome-bearing event schema.* Build the seven-field record with `action` separated from outcome, and require every tool invocation to emit one. This maps cleanly onto ACT's existing `coordination-log.jsonl`. Order: schema first, emitters second. Acceptance test: for every completed tool call the log contains both the attempt and its outcome; no event has an empty outcome field.
2. *`infer=False` raw append on the hot path.* Build: writes are pure serialization, zero LLM calls, inside retry loops. Acceptance test: p99 event-write latency is disk-bound (sub-10ms), and no LLM request is issued during a write.
3. *Pre-retry history query.* Build: before re-attempting a failed action, query prior events for that task+target and inject them. Acceptance test: an agent that failed `apply_patch` on `auth.py` with an ImportError does not issue a byte-identical retry; the second attempt's prompt provably contains the first failure.
4. *Supersede-not-delete with a `latest_only` read flag.* Build: contradiction marks the old record outdated-and-dated; two read modes over one store. Acceptance test: after superseding, `latest_only=True` excludes the old record while `latest_only=False` returns both in timestamp order — and the original bytes are unchanged.

**CONCEPTUAL — NEEDS FURTHER RESEARCH (4)**
1. *Compaction policy.* The article gives a heuristic sentence, not a policy. Missing entirely: how side-effect severity is classified (hand-annotated per tool? inferred?), the event-count/time/token trigger, batch-window boundaries, and what happens to raw events after a lesson is extracted (kept forever? tiered? the article says lossless but never says where they go). Next step: read Letta/MemGPT's memory-pressure eviction implementation for a concrete trigger, and the TencentDB-Agent-Memory tiered-consolidation design already noted in ACT's references as the closest analogue.
2. *Lesson extraction quality.* `infer=True` is a black box — no extraction prompt, no validation, no measure of whether the distilled constraint is correct. A wrong lesson is worse than no lesson because supersede will not catch it. Next step: run the extraction against LongMemEval or LoCoMo and measure constraint precision before trusting it.
3. *Re-validation of stale constraints.* The article names the failure and then says "occasional". Missing: TTL, trigger, and who pays for the re-check. Next step: design and A/B it locally; no cited work.
4. *Retention and replay.* Append-only is claimed; retention policy and replay semantics are simply absent. Missing: snapshot format, snapshot cadence, whether state is rebuilt by full replay or from a snapshot plus tail. ACT already does event-sourcing replay from `coordination-log.jsonl`, so this is a comparison, not a greenfield question. Next step: read the article's own repo (`mem0ai/mem0`) rather than the post — the post does not answer it.

**MARKETING / NOT ACTIONABLE (2)**
1. Synthesis as a Pro-plan background feature — a hosted product tier, one-day latency, scope-restricted; no engineering content.
2. The "scope beyond coding" generalization claim in the FAQ.

---

## Open Questions — answer before writing a line of code

1. **What triggers compaction?** Event count, token budget, wall-clock, or per-action severity class — and who assigns the severity class? Nothing in either article answers this.
2. **After a lesson is extracted, what happens to the raw events?** "Lossless" and "replaces repetitive receipts" are in tension. Cold tier, or same store with a flag?
3. **How is a constraint validated before it becomes durable?** An LLM-extracted wrong constraint propagates silently and supersede will never fire on it.
4. **What is the re-validation cadence for stale constraints, and what pays for it?**
5. **Two live non-contradictory-but-incompatible constraints — what resolves them?** Supersede handles direct contradiction only.
6. **Concurrent writers.** ACT has five parallel swarm Runners. Ordering, dedup, and read-your-writes are undiscussed in both articles.
7. **What is the actual scope key?** The article overloads `user_id` with `task_id`, which breaks the Synthesis feature. Does ACT need `{project, role, task, run}` as first-class dimensions instead?
8. **Is the 12K-token figure reproducible?** No model, no task, no method given — treat as unmeasured until re-run locally.
9. **Retrieval over events vs over state:** does a query hit raw events, compacted lessons, or both ranked together — and how are stale-but-unsuperseded receipts kept from outranking current lessons?
10. **Do the four harness descriptions survive contact with reality?** Every claim is uncited archetype. Before using as competitive input, verify at least the Claude Code and Antigravity rows against actual behavior.
