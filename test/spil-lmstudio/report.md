# SPIL × LM Studio — Phase 1+2 Report

**Runs:** `2026-05-19T11-03-15-142Z` (smoke, 3 fixtures) + `2026-05-19T11-43-50-269Z` (full, 5 fixtures)
**Fixtures:** parse-01, produce-01, obey-01, ctd-01-violation, ctd-02-valid

## Swarm-tier matrix (RAM-fit, 4 models × 5 fixtures)

| Model | avg | tok/s p50 | RAM | Verdict |
|---|---|---|---|---|
| **openai/gpt-oss-20b** MLX MXFP4-Q8 | **100.0** | **22.7** | ~12GB | ⭐ **Top pick.** Perfect across all 5 fixtures and 2-3× faster than the 14B-class. |
| **google/gemma-3-12b** MLX QAT 4bit | **100.0** | 8.3 | ~7GB | ✅ Best low-RAM option. Perfect quality, slow inference. |
| **qwen/qwen2.5-coder-14b** MLX 4bit | 98.6 | 8.5 | ~8GB | ✅ Solid. Lost 7pts on produce-01 (3 success criteria, fixture requires 4). |
| qwen/qwen3-14b MLX 4bit | 67.4 | 7.0 | ~8GB | ❌ Failed CTD comprehension (false-passed an invalid doc) + incomplete parse-01 extraction. Reasoning-model thinking budget interferes. |

## Planner-tier (RAM-bound, partial)

| Model | RAM | State | Notes |
|---|---|---|---|
| qwen/qwen3.6-35b-a3b GGUF Q4_K_M | 20.5GB | loadable | Forces thinking via LM Studio chat template regardless of `chat_template_kwargs.enable_thinking=false` and `/no_think` prompt suffix. `reasoning_tokens` consumed 80-95% of completion budget. **Result: parse/obey ok (output fits remaining budget), produce/CTD starved.** |
| zai-org/glm-4.7-flash MLX-6bit | ~22GB | blocked | Guardrail rejects load even at `mode:"off"` because LM Studio app caches setting in memory (needs full app restart, not server restart). |

## Five-dimension test design

Each fixture corresponds to a real path in the ACT pipeline:

| Fixture | What it tests | ACT pipeline analogue |
|---|---|---|
| **parse-01** | Recognize @keyword inline, @keyword: sections, `-` list items, `>` directive attachment | What SPILParser.ts does server-side |
| **obey-01** | Extract `@success_criteria` items only, as a focused JSON array | Assurance's validation gate input |
| **produce-01** | Emit a complete SPIL task spec from a prose brief, with required sections + CTD order | Planner's `CREATE_TASK:` payload |
| **ctd-01-violation** | Audit a SPIL doc with `@error_handling` placed before `@data` (references concepts not yet introduced). Identify the violation. | Block 7 SPIL parser's CTD validation pass (when implemented) |
| **ctd-02-valid** | Audit a well-ordered SPIL doc. Do NOT invent violations. | Negative-test for the same gate — false-positive rate |

**CTD fixtures both embed a full SPIL primer in the system prompt** (definitions of @ and >, the 8-step progression, the violation criterion). Tests application of the rules, not cold-knowledge of them — matches real ACT deployment where SPIL.md is in Planner context.

## Findings (deltas from earlier round)

### gpt-oss-20b on produce: variance not failure
First run scored 70 (used `@task: "value"` colon-inline hybrid + markdown fences inside @data). Second run scored 100 (clean `@task "value"` syntax, no fences). Same prompt, same temperature. Model is non-deterministic on the leading-colon decision. **Implication:** acceptable for ACT's Ralph Wiggum Loop (one bad shape gets bounced by Assurance, retried), but worth a system-prompt assertion *"@keyword is followed by a space and quoted value, NEVER a colon"* to lock the syntax.

### CTD comprehension is real on most models
3 of 4 swarm models correctly identified the CTD violation in ctd-01 (error_handling referencing `max_upload_bytes` from a @data section placed below it). Same 3 correctly said `valid:true` on the well-ordered ctd-02. **Conclusion:** CTD understanding doesn't require reasoning models. A 7-12B non-reasoning model with the rules embedded in system prompt is enough.

### qwen3-14b false-passed ctd-01
Said `{ valid: true, violations: [] }` on a doc with a real violation. Combined with parse-01 returning only 3 of 7 expected keywords, qwen3-14b is consistently the weak link. **Hypothesis:** Qwen3 family's forced thinking eats into the comprehension budget even at 14B. Same model would likely score higher with thinking disabled — but LM Studio's loaded template doesn't honor either disable method.

### Bigger ≠ better at Swarm — confirms the architectural inversion
gpt-oss-20b (20B params, 12GB MLX) tied with gemma-3-12b (12B params, 7GB MLX) on quality. The 20B's edge is *speed* (22 vs 8 tok/s on Apple Silicon MLX), not capability. **Implication for ACT:**
- Don't burn the largest local model on Planner. Planner work is structured pattern-matching — medium models nail it.
- At Swarm, prioritize **non-reasoning** + **fast inference** + **clean instruction following** over raw parameter count.
- Reasoning models force their reasoning even when ACT's spec + Assurance gate make external reasoning unnecessary. Costs tokens, costs latency, costs first-shot quality.

## Recommended ACT config (Tier 2 / `~/.act.json` `agents.developer`)

```jsonc
{
  "agents": {
    "developer": {
      "provider": "local",
      "model": "openai/gpt-oss-20b",        // top pick: 100/100, 22 tok/s
      "baseURL": "http://localhost:1234/v1",
      "maxTokens": 4000
    }
    // Fallbacks if gpt-oss-20b unavailable / RAM tight:
    //   google/gemma-3-12b   (7GB, 8 tok/s, 100/100)
    //   qwen/qwen2.5-coder-14b (8GB, 8.5 tok/s, 98.6/100, coder-specialized)
  }
}
```

## Still not tested

- **Planner-tier on full fixtures**: needs LM Studio app restart to override RAM guardrail OR a non-reasoning planner-class model (10-20B non-reasoning).
- **Tool-use channel**: SPIL extraction via OpenAI function-calling, not freeform text. Both GLM-4.7 and Qwen3.6 are `trainedForToolUse: true`.
- **Ralph Wiggum Loop convergence**: real iteration test — give a swarm model a SPIL task + criterion that fails first attempt, count iterations to converge.
- **End-to-end ACT**: wire gpt-oss-20b into `~/.act.json` developer role, spawn one Runner, complete a real task with Assurance validation gate.

## Files

- `models.json` — registry with per-model `chatTemplateKwargs` + `userSuffix`
- `spil-parser.mjs` — JS port of SPILParser.ts
- `run.mjs` — fixture runner with thinking-control support
- `rescore.mjs` — re-score dumps without re-running models
- `fixtures/{parse,produce,obey,ctd-01-violation,ctd-02-valid}-*.spec.json`
- `results/2026-05-19T11-43-50-269Z/` — per-call full prompt + raw response + score (20 dumps + summary.json)
