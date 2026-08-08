# ACT Pitch Page — Project Plan

*AI Fluency course project (Delegation lesson, 2026-07-07). Return here for the Description, Discernment, and Diligence lessons.*

## Vision

A single scrolling HTML page that a developer reads in ~2 minutes, understands what ACT is, and installs the alpha. Adopter-first, with a "build this with us" contributor CTA at the end.

- **Audience:** both adopters and collaborators, adopter-first.
- **Primary ask:** try the alpha.
- **Tone:** honest builder pitch — thesis, what works, what's broken, come help. d34d voice.
- **Format:** single self-contained HTML page, shareable as a link.
- **Success bar:** 5 outside devs install the alpha and report back.

## Tasks + delegation

| # | Task | Delegation | Why |
|---|------|-----------|-----|
| 1 | Positioning/story — thesis ("structure substitutes reasoning"), hook, differentiation vs CrewAI / AutoGen / claude-flow | **Human-led, AI sparring** | Only d34d knows why ACT exists; AI supplies competitor landscape + pushback |
| 2 | Page copy — problem → thesis → two-tier model → works today → honest gaps → roadmap → ask | **AI drafts in d34d voice (style-clone skill), d34d edits** | Deliberate experiment: is voice delegable? |
| 3 | Demo evidence — real TUI screenshots | **Human only** | Needs d34d's machine + live LLM session; this is the proof the honest pitch stands on |
| 4 | HTML page build — layout, styling, single file | **Near-full AI** | Pure execution once copy + assets exist |
| 5 | Claim verification — every "works today" line greped against code | **AI-strong** | Anti-trust rule applies to marketing hardest; one false claim burns the exact audience we want |
| 6 | Landing path — "try the alpha" → quickstart that actually works | **Collaboration** | AI checks README commands vs code; d34d confirms on a clean run |

## Decisions made

- Screenshots over video (cheaper, still real) and over HTML mockup (mockup fails the honest-pitch test).
- Full-draft voice delegation via the style-clone skill, with a human edit pass.
- Success = outcome (5 installs), not just output (page shipped).

## Open / blocking

- ~~Screenshots (task 3)~~ **DELIVERED 2026-07-07** — two real TUI shots from a live "finance" expense-tracker run:
  - **Shot 1 (intake):** human brief → Planner structures it (description/stack/constraints/success criteria) → agent-mix question + recommendation → Observer/Assurance/QA online pings. Anchors the "one window, coordinated team" section.
  - **Shot 2 (build+validation):** SPIL task creation, Assurance verdict JSON with per-criterion reasoning (9/9 tests pass, score 100), QA synthesis. Anchors the "evidence-based validation" section.
  - **Nits:** title bar shows `--debug` (crop or re-shoot); arc ends at synthesis — one closing shot of Planner's final report to human would complete brief → build → validate → report.
  - **TODO:** drop the PNG files into `docs/pitch/assets/` when we build the page (currently only pasted in chat).
- Quickstart must be verified before the page links to it (README is on the stale list).

## Insights from the planning conversation (for course reflection)

- Adopters and collaborators want different decks; adopter-first with a contributor outro resolves it.
- The honest-pitch choice constrains other choices downstream: it killed the mockup option and made claim-verification (task 5) mandatory rather than nice-to-have.
- Voice — normally the least delegable thing — became delegable here because a style-clone skill exists. Delegation boundaries move when you've invested in tooling.
- Hardest part ahead: task 5. Verifying marketing claims against a moving branch is the same anti-trust discipline the repo already enforces on docs.
