# Architecture-flows method

How to build a single-file, offline-openable visual map of this codebase (or to update the one that already exists) without bluffing.

The artifacts this method produces are:

- `architecture-flows.html` — a single HTML file at repo root that renders an interactive diagram. All CSS, JS, SVG, and the JSON data live inline. No external requests.
- `architecture-flows.json` — a sibling JSON file at repo root, byte-identical to the inline `<script type="application/json">` block in the HTML. Used for machine verification and for re-rendering.
- `flows-explainer.html` — a companion HTML at repo root that opens with a Findings headline at the top. Plain prose describing what is real, what is prompt-wished, and what is a gap.

This method file is the contract. Every later phase of the work must satisfy it.

## Status taxonomy

The diagram must distinguish four states. The first three are the only ones shipped; the fourth is a construction-time scratch marker that must be zero at completion.

| Status | Visual | Meaning |
|--------|--------|---------|
| `ok` | solid gold arrow + gold badge | Code-verified. The `detail` field must cite `file:line` for the implementing code. |
| `prompt-only` | solid orange arrow + ⌕ badge | Behavior lives in prompt text only. LLM compliance, no code enforcement. |
| `gap-found` | dashed red arrow + ⚠ badge | Documented but no implementing code, OR partially implemented. Frame the gap precisely (what works, what does not). |
| `unverified` | (forbidden in shipped artifact) | Scratch marker used during construction. Must be replaced before completion. |

The binary `ok` / `unverified` taxonomy used in the prior attempt was the root cause of the bluffs that prompted this rewrite — `unverified` was being used to mean "I did not bother to check," and `ok` was being assigned to choreography that had only been grep-hinted, not opened. The four-status taxonomy forces an honest answer per step.

## The amended §4c: how to assign status honestly

The handoff that produced the prior diagram permitted "source-only inference" under the `ok` flag — meaning the author could see that a REST endpoint existed and then extend confidence to the surrounding choreography (e.g. `fetch context`, `markCompleted()`, `LLM turns during work`) without opening those files. That rule is rescinded.

The new rule is:

1. **Grep upfront, not after.** Before assigning a status to a step, run the grep that would falsify the claim. If the grep returns the expected hit, open the file at the line number and confirm the surrounding context matches the claim. Only then assign `ok` and write the `file:line` into `detail`.
2. **`unverified` is forbidden in shipped artifacts.** It exists during construction so that you can write down a claim you have not yet checked. Before shipping, every `unverified` step must be promoted to `ok` with a `file:line` cite, demoted to `prompt-only` if the behavior is only in prompts, demoted to `gap-found` if there is no implementing code, or removed entirely if the step does not belong on the diagram.
3. **Sub-agent claims are second-hand.** If you delegate exploration to a sub-agent (Explore, general-purpose, etc.), the sub-agent's claims must be re-grep'd by the parent before being encoded into the JSON. This is the rule the prior session learned the hard way when a sub-agent claimed "QA persistence = full gap" using bad grep terms (`qaOutput|finalDeliverable|synthesizedOutput`) and the user caught the bluff because the actual endpoint at `server/src/index.ts:795` was named `synthesis`. No second-hand assertions in shipped JSON.
4. **Partial gaps are first-class.** When a system does part of what its prose claims and skips the rest, that is `gap-found`, not `ok`. Frame the gap precisely: which half works, which half does not, and what evidence supports each claim.

## JSON schema

The JSON is shaped to make verification easy. Stable keys, no opaque blobs.

```jsonc
{
  "meta": {
    "version": "n",
    "generatedAt": "ISO timestamp",
    "branch": "NesTTY",
    "headCommit": "ab159d4...",
    "findings": [
      { "id": "F1", "title": "...", "summary": "...", "evidence": "file:line, file:line" }
    ]
  },
  "categories": [
    { "id": "server", "label": "ACT Server", "color": "#hex" }
  ],
  "columns": [
    { "id": "tier1", "label": "Tier 1 (interactive)" }
  ],
  "components": [
    { "id": "...", "label": "...", "categoryId": "...", "columnId": "...", "detail": "...", "tags": ["..."] }
  ],
  "flows": [
    {
      "id": "...",
      "label": "...",
      "summary": "...",
      "steps": [
        { "from": "componentId", "to": "componentId", "label": "...", "status": "ok|prompt-only|gap-found", "detail": "...with file:line if status==ok..." }
      ]
    }
  ]
}
```

Constraints on the JSON:

- Schema keys stay stable. Adding fields is fine; renaming or removing keys is not.
- **Columns ARE flow-stage lanes, not subsystems.** The diagram exists so humans read a flow's path left-to-right. If columns are subsystems (server/orchestrator/tier1…), intra-subsystem flows collapse into one vertical column = arc spaghetti (the v7 failure). `columns` MUST be these 10 ordered lanes: `entry, intake, plan, dispatch, execute, coord, validate, synth, store, infra`. Assign each component's `columnId` to the lane of its PRIMARY role (what it most does, not every flow it touches): REST endpoints / in-memory maps / data files / vector stores → `store`; config / bootstrap / server-launch / HTTP client / process-group / generic wiring → `infra`; everything else → the stage it performs. Target: median distinct-lanes-per-flow ≥ 3. Single-stage flows (registration, config resolution) staying in one lane is acceptable — they ARE one stage's internals.
- **Every component MUST carry a `slot` integer** — its 0-based row position within its `columnId` lane. The renderer sets `card.style.gridRow = slot + 2`; omitting `slot` defaults every card in a column to row 2, so dense columns overlap into unreadable mush (the v7 rebuild shipped without it and had to be patched). Assign slots densely per column (0,1,2,… in display order). Verify post-render: no two cards in the same column share a top coordinate.
- **After writing the JSON, re-render the HTML and actually look at it** (headless browser or a screenshot): confirm cards don't overlap, filter chips toggle, and selecting a flow draws arrows. Data-block parity passing is necessary but NOT sufficient — it does not prove the visual renders.
- `meta.findings[]` surfaces the gap-found items as standalone records, so the explainer page and any later automation can pick them up without walking the entire `flows` tree.
- Component count ≥ 70, flow count between 15 and 22. These are not hard inventory targets; they reflect the actual moving-part count of this codebase. If the real count comes out outside the band, write down why before shipping.

## HTML rendering rules

- Single file. All CSS in a `<style>` block, all JS in a `<script>` block, the JSON in a `<script type="application/json" id="data">…</script>` block. SVG is inline.
- Zero network requests. No external fonts, CDN scripts, images, or analytics. Open in a browser via `file://` and DevTools Network tab must be empty.
- Filter chips toggle visibility per category. Clicking a chip hides or shows every component in that category.
- Multi-select flow overlays: clicking a flow assigns it the next color in a fixed 5-color palette (gold, blue, green, magenta, cyan). Shift-clicking a second flow adds it to the overlay set; clicking an already-selected flow deselects it. A "Clear" button drops all overlays.
- Step badges include a flow-color border on the side closer to the overlay so a chain of handoffs reads spatially.
- The `architecture-flows.json` sibling file is byte-identical to the content of the `<script type="application/json">` block. Verify with a round-trip diff before shipping.

## Explainer page rules

- `flows-explainer.html` opens with the Findings headline at the top. Each finding gets a short paragraph: what was claimed, what is actually true, and what the implications are.
- Below the headline, the explainer can describe individual flows in prose for readers who want narrative rather than a clickable diagram.
- Same single-file rule applies. Inline CSS, no external requests.

## Code constraints

- **No edits to existing source.** Read-only against `server/`, `act-agent/`, `install.sh`, `install.ps1`, `.act.example.json`, and `nomik/`. The `git status` post-completion shows only the create/modify list documented in the plan file.
- **No defensive scaffolding, no "while I'm here" cleanup.** Strict surface.
- **No emojis in code or commit messages.** UI badge glyphs (⚠ ⌕) are exceptions because they convey status.
- **`act-coordination.json` is append-only.** Never modify existing entries.
- **Files written must be readable prose.** Caveman mode is chat-only; the artifacts ship for human readers.

## Success criteria (verified post-completion)

The plan file ships with a 22-row criteria table. Walk every row, mark pass/fail, fix any failure before declaring done. The non-negotiable rows are:

- Zero `"status": "unverified"` in shipped JSON.
- Every `ok` step has `file:line` in its `detail`.
- Six findings rendered on the explainer headline (Ralph prompt-only, QA partial-deliverable persistence, Socket.io vestigial, no-auth, Qdrant build-excluded, kanban-doc-mismatch).
- `git status` matches the create/modify allow-list. No source files altered.

## Verification — single command

```bash
python3 -c "
import json, re
d = json.load(open('architecture-flows.json'))
bluffed = [s for f in d['flows'] for s in f['steps']
           if s.get('status') == 'ok' and not re.search(r'\.(go|ts|mjs|json|sh|ps1):\d+', s.get('detail',''))]
print('bluffed ok count (should be 0):', len(bluffed))
print('unverified count (should be 0):', sum(1 for f in d['flows'] for s in f['steps'] if s.get('status') == 'unverified'))
print('prompt-only count:', sum(1 for f in d['flows'] for s in f['steps'] if s.get('status') == 'prompt-only'))
print('gap-found count:', sum(1 for f in d['flows'] for s in f['steps'] if s.get('status') == 'gap-found'))
print('components:', len(d['components']))
print('flows:', len(d['flows']))
"
```

If `bluffed ok count` is not zero, the artifact is dishonest — fix before shipping. If `unverified count` is not zero, the artifact is incomplete — promote, demote, or remove every `unverified` step before shipping.

## When to redo this work

The diagram goes stale fast. Re-run this method when any of the following changes:

- A REST endpoint is added, removed, or renamed in `server/src/index.ts`.
- A new Tier 1 or Tier 2 role file appears in `act-agent/internal/llm/prompt/`.
- A coordination protocol step changes (e.g. the validation pipeline grows a new gate, a new ChronLog event type is added, the QA endpoint starts persisting full deliverables).
- A claim in this method doc is contradicted by reality.

Each rebuild starts by walking the 22-row criteria table from scratch. The previous diagram is not authoritative — the codebase is.
