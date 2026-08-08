# Architecture-flows method

> ## ⛔ THE MODEL: GROUPED COMPONENTS IN SUBSYSTEM BOXES (rewritten 2026-06-14, v9)
> The diagram exists so a HUMAN can read **how ACT's architecture flows**. Three earlier rebuilds failed:
> (1) every function/endpoint as a node — a call-graph, granular and agent-invisible; (2) one blob per
> subsystem — a README-surface mermaid that hid everything; (3) "actors only" (~14 flat nodes) — still
> too coarse to show the real moving parts. The correct model is the **MIDDLE**.
>
> **Nodes = real architectural COMPONENTS** (~40-50), laid out in **COLUMNS — one column per subsystem**
> (a component carries `parent` = its subsystem id). Subsystems (7, left→right):
> `external orchestrator prompts execution swarm server memory`. Each column = one color, **same-size**
> nodes **stacked top-down**, a column header label at the top; columns all start at the top and run
> **independent lengths** (a column ends when it runs out of nodes). **NO literal boxes** around a column —
> color + header is the grouping. A component is a real moving part with a distinct job in a flow (a loop,
> a parser, an endpoint, a store, a fork twin, a convergence node) — NOT every function (collapse those)
> and NOT a whole subsystem. Each node shows **two lines**: the friendly name + a `src` code anchor
> (file + function/symbol, e.g. `orchestrator.go runAgentTurn`) so the node carries its code identity.
> **Edges = coordination handoffs**, flowing **horizontally across columns**. Label = the REAL MECHANISM
> (a CLI subcommand, a REST route, a parsed marker like `CREATE_TASK:`, a poll, a spawn, an interface
> call); `file:line` evidence in `detail`. Edges are faint until a flow is selected.
> **Config FORKS are first-class**: twin component nodes + a shared convergence node, tag each step
> with `branch` (e.g. `in-process|acp|converge`). **Documented-but-unbuilt = a GAP**: a single
> `gap-found` self-loop on the node where it would live, never a working flow.
>
> **Build from TRACED code, not imagination.** The flows are produced by tracing each process through the
> live code (file:line per step), verified by re-grep, then collapsed onto the fixed component set.
> **Renderer = a custom vanilla HTML+CSS+SVG engine — NO libraries** (ported from the NesTTY-branch
> build, which is the canonical look; do NOT reintroduce Cytoscape/dagre — they auto-zoom, allow
> node-dragging, and scattered the layout). The structure: **CSS-grid columns** (one per subsystem,
> header on top), **dark cards** with a subsystem-colored **left-accent** + name + the `src` code anchor
> as a subtitle; a card's `detail` expands on click. An **SVG overlay** draws the selected flow's
> handoffs as colored arcs (status: solid=ok, dashed=prompt-only, dotted=gap) each carrying a **numbered
> circle badge** (step order); non-member cards dim to ~13%. **Pan by scrolling the canvas — no zoom, no
> node-dragging** (cards are fixed grid items). Hovering a card shows a floating tooltip = its `hover`
> field (plain-English explanation ending in a gold "From a user perspective…" line). **Column order =
> the order of the `subsystems[]` array** (reorder that array to reorder columns). Top chips filter by subsystem. The card row within a
> column (`slot`) is computed at render from component order — not stored. Up to 5 flows can overlay at
> once (shift-click), each a different palette color.

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
    "version": 9,
    "model": "subsystem-columns",
    "generatedAt": "ISO date",
    "branch": "feat/remove-nomik",
    "headCommit": "d59785a...",
    "summary": "one-line orientation, rendered in the sidebar",
    "findings": [
      { "id": "F1", "title": "...", "status": "gap-found|prompt-only", "summary": "...", "evidence": "file:line, file:line" }
    ]
  },
  "subsystems": [                                  // the 7 columns (color + header, NO box); left-to-right order
    { "id": "orchestrator", "label": "ORCHESTRATOR · Go TUI brain", "color": "#hex" }
  ],
  "components": [                                   // real components, one per row in its subsystem column
    { "id": "...", "label": "...", "src": "file + symbol (card's 2nd line)", "hover": "plain-English: what it is · where in code · ends 'From a user perspective, this is ...'", "parent": "<subsystem id>", "kind": "engine|parser|endpoint|store|prompt", "detail": "..." }
  ],
  "flows": [
    {
      "id": "...",
      "label": "...",
      "summary": "...",
      "family": "lifecycle|coordination|memory|config-fork|gap",
      "steps": [
        { "from": "componentId", "to": "componentId", "label": "...mechanism...", "status": "ok|prompt-only|gap-found",
          "detail": "...with file:line if status==ok...", "branch": "optional fork tag e.g. in-process|acp|converge" }
      ]
    }
  ]
}
```

Constraints on the JSON:

- Schema keys stay stable within a version. The v8→v9 change renamed the node role (`nodes`→`components` + `parent`), replaced `categories`/`columns` with `subsystems`, and added `kind` + `src` (component), `family` + `branch` (flow/step). Positions are computed at render (column = subsystem index, row = order within the column), NOT stored per-node — the old v7 CSS-grid `slot`/`columnId` machinery stays gone.
- **Every flow step's `from`/`to` MUST be a component id, never a subsystem id.** The renderer finds each endpoint by `.card[data-id=...]`; an unknown id means no card is found and the arc is silently dropped (`drawOverlay` skips it). The build's validation step asserts zero bad edges so this never ships.
- **Config forks render as twin nodes + a convergence node** (both children of the same box), with a `branch` tag per step. **Gaps render as one `gap-found` self-loop** on the node where the capability would live — labelled so it reads as a hole, not a path.
- **After writing the JSON, re-render the HTML and actually look at it** (headless browser screenshot): confirm the subsystem boxes draw with their components nested, no box overlap, selecting a flow highlights its path while boxes stay framed, a fork shows its two branches, the gap shows as a self-loop, and DevTools Network is empty. Data-block byte-parity is necessary but NOT sufficient — it does not prove the visual renders.
- `meta.findings[]` surfaces gap-found / prompt-only items as standalone records, so the explainer page and any later automation can pick them up without walking the entire `flows` tree.
- Component count ~40-50, flow count ~20-26. Not hard targets — they reflect the real moving-part count. Collapse two candidates into one component when no flow has a handoff between them; keep a component distinct when it's a flow target, a fork twin/convergence, or a loop with its own interval. If the count lands outside the band, write down why before shipping.

## HTML rendering rules

- Single file. All CSS in a `<style>` block, all JS in a `<script>` block, the JSON in a `<script type="application/json" id="data">…</script>` block. SVG is inline.
- Zero network requests. No external fonts, CDN scripts, images, or analytics. Open in a browser via `file://` and DevTools Network tab must be empty.
- The left sidebar lists every flow (label + summary). Clicking a flow highlights its path and dims the rest; shift-clicking adds another flow to the selection; clicking a selected flow deselects it. "Clear" resets; "Fit" re-frames.
- Selecting a flow fills the steps panel with its ordered handoffs — `from → to`, the mechanism label, and the `detail` (with file:line for code-enforced steps). Edge color encodes status (gold=ok, orange=prompt-only, red=gap); fork branches are tinted.
- The column header labels and node `src` second-lines stay visible at all times so the map reads as code even before a flow is picked.
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

Walk these before declaring done; fix any failure first. The non-negotiable rows:

- Zero `"status": "unverified"` and zero bluffed `ok` (every `ok` step has `file:line` in its `detail`).
- Zero bad edges: every step `from`/`to` is a component id (run the build validation).
- `architecture-flows.json` is byte-identical to the HTML's inline `<script id="data">` block.
- The render check passed (boxes draw + nest, a flow highlights while boxes stay framed, a fork shows two branches, the gap shows as a self-loop, Network tab empty).
- The `meta.findings[]` items render on the explainer headline (currently: deferred-tools gap, unwired client methods, SQLite stub, file-locking prompt-only, QA marker-only, Socket.io vestigial, no-auth, Qdrant build-excluded).
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
