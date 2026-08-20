# If ACT's Memory Were Swapped for These Architectures

**Date:** 2026-08-13
**Inputs:** [`agent-memory-mem0-reports-2026-08-13.md`](agent-memory-mem0-reports-2026-08-13.md)
(four mem0.ai articles, classified by implementation readiness) and
[`../audits/memory-system-audit-2026-08-13.md`](../audits/memory-system-audit-2026-08-13.md)
(live verification of what ACT's memory actually does).

---

## 0. The premise correction that governs everything below

ACT's memory is not failing because its architecture is weak. It is failing because of **two
scoping/join defects** in an otherwise-sound design:

1. outcome events carry no project tag, so they all pile into the bucket the evidence builder
   excludes;
2. outcome→worker attribution joins the one event type that lacks a payload.

**No memory architecture in these articles fixes either defect for you.** mem0's own event
article commits the same class of bug in its sample code (it jams `task_id` into `user_id` and
thereby disables mem0's own consolidation feature). Swapping stores would move the bug, not
remove it. Everything below therefore evaluates *patterns to adopt*, not *products to swap in*.

---

## 1. Straight swap: what survives, what dies

| ACT capability today | After a wholesale swap to a hosted mem0-style memory |
|---|---|
| Append-only JSONL event log | **Redundant** — mem0's "event-based memory" *is* this. ACT has had it since day one. |
| Event-sourced replay rebuilds full server state on restart | **LOST.** mem0 has no replay-as-state story; the reports flag retention and replay as unspecified in the source. This is ACT's strongest memory property and it would be traded away. |
| Local-first, offline, zero-vendor | **LOST.** A network hop enters the task write path. |
| Deterministic server (Three-Layer Separation) | **VIOLATED.** mem0's writer is an LLM (`infer=True` extraction). Putting an LLM inside ACT's deterministic state layer breaks the architecture's central rule. |
| Real local embeddings, cached, no per-write cost | **Traded** for hosted extraction + embedding: better recall potential, new per-event token cost and latency. |
| Bi-temporal edge log (`graph-edges.jsonl`, active/inactive edges) | **Redundant-and-better-already.** mem0's "supersede, don't delete" with a `latest_only` read flag is exactly what ACT's graph store already implements — and nothing in ACT reads it that way yet. |
| Broken evidence/routing layer | **STILL BROKEN**, in a new dialect. |

**Verdict:** a swap is a downgrade. ACT would give up replay and determinism to buy patterns it
can implement locally in a fraction of the code.

---

## 2. What ACT genuinely lacks, that these architectures name well

### 2.1 Outcomes, not attempts (highest value, lowest cost)
The event article's one real insight: an event that records *what was tried* without *what
happened* cannot prevent a repeat. ACT's `task_completed` has `success` plus 2000 chars of raw
agent output — attempt-and-transcript, not outcome-and-constraint. There is no field where
"`auth.py` needs pyjwt>=2.0" can live, so it never does.

**ACT-shaped version:** one optional `learned_constraint`-style field on completion/validation
events, written by the agent that hit the wall. No LLM extraction, no new store — one field on
an event ACT already emits.

### 2.2 Pre-retry recall exists in ACT, but only for one task, and it evaporates
ACT already implements mem0's best pattern and doesn't know it: on an Assurance kickback the
Runner injects `## IMPORTANT: Previous Validation Failed` with the gap text into the retry
prompt. That is precisely "query the failure history before retrying".

Its limits are what to fix:
- carried on **task metadata**, not in the memory system;
- scoped to **that one task**;
- **discarded** the moment the task passes.

Assurance kickback gaps are the highest-signal text ACT produces — a specific, independently
verified statement of what was wrong — and ACT throws every one of them away. Nothing in the
mem0 articles produces evidence this good; ACT's validation gate does it for free.

### 2.3 Compaction / consolidation — absent in ACT, and correctly unspecified everywhere
No summarization, no lessons, no eviction anywhere in ACT's pipeline. The event article
proposes batch-compacting raw receipts into lessons — but its trigger is a judgment sentence,
not a policy, and it never says what happens to the raw events afterward. Both source reports
class this **CONCEPTUAL — needs research**, pointing at Letta/MemGPT eviction and the tiered
consolidation already noted in ACT's references.

**Do not vibe-code autoDream from these articles.** They do not contain enough to build it.
The ACT-native trigger candidate that the articles do *not* have: compact a project's raw
events at **project completion / QA synthesis**, which is an event ACT already emits.

### 2.4 Scope keys as a first-class schema
mem0's four keys (`user_id` / `agent_id` / `app_id` / `run_id`) map onto ACT as
**project / role / agent-instance / task-run**. ACT currently has one of these applied
consistently and it is the wrong one. The lesson is not the key names — it is that scope is a
schema decision made once, not a field each emitter remembers to add. ACT's critical bug is
literally "an emitter forgot the scope field".

---

## 3. The non-obvious one: ACT has an identity-resolution problem

The cross-session identity article looks irrelevant (ACT has no end users). It is not.

ACT's real agent IDs across its own history: `dev-1`, `dev-1-alpha`, `dev-1-beta`, `backend-1`,
`backend-1-alpha`, `qa-1`, `test-dev-gate-1`, `gate-test-dev-real`, `authsvc-beta-backend-1`.
**Every project mints fresh agent identities.** Skill history is therefore fragmented across
throwaway IDs — the exact "returning user amnesia" the article describes, with agents instead of
humans. It compounds the join bug: even after the join is fixed, per-agent history cannot
accumulate across projects because the agent is a new person every time.

The resolution is easier for ACT than for the article's case, because ACT has a **deterministic
signal** where the article only had fuzzy ones: registered capabilities and the declared role.
- Deterministic short-circuit: role tag / capability set → canonical role identity.
- The stable identity for evidence is the **role**, not the agent instance. The agent ID is the
  run key.
- The fuzzy scorer, weights, thresholds, and review band from the article are **not needed** —
  and per the report, that scorer is misattributed statistics anyway (arbitrary additive
  weights wearing the Fellegi-Sunter name). Skip it.

**Provisional memory** (write under the run key, promote only after the match clears) has an
even cleaner ACT analogue — see §4.

---

## 4. What ACT could do that these architectures never considered

### 4.1 Validation-gated memory
mem0 promotes provisional memory once identity clears a threshold. ACT has something better
available: a **95% independent-verification gate** already sitting in the pipeline.

Write raw events always (cheap, append-only, unchanged); promote a record into *durable
evidence* only when Assurance validated the work it came from. Failed work is retained as a
**failure lesson**, never as a success pattern.

That inverts the failure mode of every architecture in these articles. Their default risk is
confidently remembering something wrong (mem0's own report flags this: a wrong extracted
constraint propagates silently and supersede will never fire on it, because nothing contradicts
it). ACT can make *verified* the precondition for *remembered*. No vendor memory layer can offer
this, because none of them own a verification gate — it is a property of ACT's coordination
shape, not of its storage.

**Is it actually useful for ACT?** Yes, and it is the one idea here worth building first:
it needs no new store, no LLM in the write path, no new dependency — just a promotion flag
written at the moment Assurance already emits a verdict.

### 4.2 Failure-keyed recall across tasks
Key kickback gaps by `(role, target-file-or-capability)` and inject matching prior failures into
any *new* task touching the same target. This is mem0's pre-retry query generalized from
one-task to cross-task — the natural extension of code ACT already runs.

---

## 5. Adopt / skip

**Adopt (in this order):**
1. Fix scope + join (the two audit tickets). Nothing else matters until evidence populates.
2. Outcome-bearing event field (`learned_constraint`) — one field, no LLM.
3. Persist and index Assurance kickback gaps as durable failure memory.
4. Validation-gated promotion (§4.1) — ACT's differentiator.
5. Read the graph store's existing active/inactive edges as `latest_only` semantics instead of
   building a supersede model.

**Skip:**
- Hosted memory as ACT's store (loses replay + determinism, adds vendor and latency).
- The fuzzy identity scorer (deterministic role/capability signal makes it unnecessary; the
  statistics in it are misattributed regardless).
- Cross-harness "shared memory spine" (ACT *is* the harness).
- Hierarchical code summarization (ACT's deliberate bet is agentic search, not an index — see
  CLAUDE.md "Code KG").

**Research before building (do not guess):**
- Compaction trigger + what happens to raw events after lesson extraction — both source reports
  class this unspecified; named next reads are Letta/MemGPT eviction and mem0 OSS's
  ADD/UPDATE/DELETE decision logic.
- Retrieval scoring beyond raw cosine — neither article gives a fusion formula. Measure a
  relevance floor on ACT's own store first (`pvm-search-no-relevance-floor-2026-08-13`).
- Extraction/lesson quality — needs a labeled eval set before any LLM-written memory is trusted.
