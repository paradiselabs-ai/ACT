*Shared Message from Pieces (https://pieces.app) by d34d (paradiselabs.ai@gmail.com) on Saturday Mar 14, 2026 - 3:19 AM*
---
# Deep Dive: The Three Engines of MCO

## Progressive Revelation, Context Feedback Looping & The Validation Loop

---

## Preface: The Problem Space

Before dissecting the three mechanisms, it's worth naming the exact failure modes they were designed to kill — because each one targets a specific, observable pathology in how LLMs handle autonomous work.

| Failure Mode | Observable Behavior | Root Cause |
|:---|:---|:---|
| **Context Saturation** | Agent loops, contradicts itself, forgets constraints mid-task | Too much information loaded at once exceeds effective attention |
| **Silent Drift** | Agent slowly deviates from spec without signaling | No persistent anchor point to re-align against |
| **Premature Completion** | Agent claims "done" with placeholder logic, mocked APIs, hardcoded outputs | No objective gate — agent's self-assessment is unchecked |
| **Context Amnesia** | Agent forgets features, styles, or requirements it acknowledged earlier | Important-but-not-critical context evicts from effective window |

As you put it on [GitHub Copilot](https://copilot.github.com) around August 25, 2025:

> *"LLMs typically get to a point where they are just gonna go, 'OK, done,' but if you ask, 'OK, so what's the current state of the project?' they'll say something like, 'Well, actually nothing actually works right now. It's all just placeholder logic and API calls that are mocked.'"*

MCO's three engines — **Progressive Revelation**, **Context Feedback Looping**, and the **Validation Loop** — don't just mitigate these problems. They form an interlocking system where each mechanism covers the blind spots of the other two.

---

## I. Progressive Revelation

### The Principle

Progressive Revelation is a **context management architecture** built on one insight: **an LLM's effective reasoning capacity degrades non-linearly as context volume increases.** It's not just that the agent has "too much to read" — it's that token attention mechanisms create interference patterns where unrelated context actively degrades performance on the task at hand.

The traditional approach treats context like a database dump:

```
Agent receives: Core + Features + Styles + Success Criteria + Examples + Edge Cases
Result: Overwhelmed → Loops → Failures → "Done" (nothing works)
```

Progressive Revelation treats context like **medication dosing** — the right information, in the right amount, at the right time:

```
Phase 1: Agent receives ONLY mco.core + mco.sc (the vital organs)
Phase 2: Agent receives mco.features (~33% progress) 
Phase 3: Agent receives mco.styles (~66% progress)
```

### The Two-Tier Memory Model

This is where Progressive Revelation becomes architecturally distinct from just "giving less context." MCO establishes two explicit memory tiers:

**Tier 1: Persistent Memory (Never Leaves Context)**
- `mco.core` — The workflow definition, agent roles, data structures, fundamental architecture
- `mco.sc` — Success criteria, target audience, completion goals

These are the agent's **permanent consciousness**. They serve as the gravitational anchor that prevents drift. No matter what gets injected or evicted, the agent always knows *what it's building* and *what "done" looks like*.

**Tier 2: Semi-Persistent Memory (Strategically Injected & Allowed to Decay)**
- `mco.features` — Feature requirements, functionality specs
- `mco.styles` — UI/UX guidelines, formatting, visual presentation

These are allowed to **enter and exit** the agent's effective context window. MCO doesn't care if the agent "forgets" that the button should be blue during the database schema phase. It cares that the agent *remembers* it when it's time to render the button.

### Why This Works (Cognitive Load Theory Applied to LLMs)

The parallel to human cognitive science is not accidental. Cognitive Load Theory (Sweller, 1988) distinguishes between:

- **Intrinsic load** — complexity inherent to the task itself
- **Extraneous load** — complexity from *how* the information is presented
- **Germane load** — effort spent building useful mental models

When you dump everything on an LLM at once, you're maximizing **extraneous load**. The agent spends attention tokens processing style guidelines while it should be reasoning about data architecture. Progressive Revelation minimizes extraneous load by ensuring the agent's context window contains **only information germane to its current phase of work**.

### The Dosing Algorithm

As you described on [GitHub Copilot](https://copilot.github.com) (~August 25, 2025, 4:14 PM):

> *"You don't want to overwhelm the agent, so you keep the important stuff persistent in their memory, and then you allow them to forget the non-important stuff, but you smartly loop it back into the workflow — not only intelligently WHAT gets looped back, but WHEN in the workflow process it should get looped back."*

This reveals that Progressive Revelation isn't a simple timer. It's a **contextual dosing algorithm** with two dimensions:

1. **WHAT** — Which file(s) or portions of files get injected. Maybe only three of twelve features are relevant to the current step. MCO can inject a subset.
2. **WHEN** — At what point in the workflow progression. This isn't purely percentage-based in practice — it's **evaluation-driven**. The injection decision is informed by the Validation Loop's assessment of what the agent is missing.

This creates a feedback relationship between Progressive Revelation and the Validation Loop (more on this below).

---

## II. Context Feedback Looping (Prompt Injection Context Looping)

### The Principle

Context Feedback Looping is the **execution mechanism** that makes Progressive Revelation dynamic rather than static. If Progressive Revelation is the *philosophy* (dose information strategically), Context Feedback Looping is the *circulatory system* that actually moves that information through the agent's working memory.

The core loop:

```
┌─────────────────────────────────────────────────────┐
│                                                     │
│   ┌───────────┐     ┌───────────┐     ┌──────────┐ │
│   │ PERSISTENT│     │ AGENT     │     │ EVALUATE │ │
│   │ CONTEXT   │────►│ WORKS     │────►│ OUTPUT   │ │
│   │ .core+.sc │     │           │     │ vs .sc   │ │
│   └───────────┘     └───────────┘     └────┬─────┘ │
│        ▲                                    │       │
│        │         ┌───────────────┐          │       │
│        │         │ INJECT        │          │       │
│        └─────────│ relevant SPIL │◄─────────┘       │
│                  │ (.features/   │                   │
│                  │  .styles/     │   What's missing? │
│                  │  partial)     │                   │
│                  └───────────────┘                   │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### How It Differs From Simple Retry

A naive retry loop says: *"Agent failed. Give it the same prompt again."*

A context feedback loop says: *"Agent produced an incomplete result. Evaluate what's missing against the success criteria. Determine which specific SPIL context addresses the gap. Inject that context on top of the persistent foundation. Agent continues with enriched understanding."*

The critical distinction: **each loop iteration changes the agent's context in a targeted way.** The agent isn't just retrying — it's receiving new, relevant information that it didn't have before. This turns what would be a frustrating infinite loop into a **convergent refinement process**.

### The Three Context States

Context Feedback Looping operates on three distinct context states across each iteration:

| State | Contents | Behavior |
|:---|:---|:---|
| **Always Present** | `mco.core` + `mco.sc` | Never evicted. The "DNA" of the workflow. |
| **Injected This Loop** | Relevant portions of `.features` / `.styles` | Fresh context targeting identified gaps. |
| **Accumulated** | Results from previous steps + injection history | The agent builds on its own prior work. |

### The Intelligence of the Loop

As Iqra summarized in [Discord](https://discord.com) (~August 22, 2025, 11:41 PM):

> *"The looping prompt injections are the less vital things like 'style, colors, features, font, visual presentations, animations, etc.' And that continues until the final result meets all the requirements and your project is finished!"*

What makes this "intelligent" rather than "mechanical":

1. **Gap Analysis** — After each evaluation, MCO identifies *which* success criteria are unsatisfied. This determines injection content.

2. **Partial Injection** — MCO doesn't re-inject the entire `.features` file. If the agent nailed 8 of 10 features but missed responsive layout and error handling, only those specific SPIL blocks get injected.

3. **Temporal Relevance** — Style information isn't injected during a backend logic phase, even if styles are technically "missing." The loop respects the natural progression of development work.

4. **Convergence Guarantee** — Because each loop iteration adds targeted context that addresses specific failures, the score should monotonically increase. If it doesn't, that's a signal that the problem isn't contextual — it's architectural (the agent needs a different approach, not more information).

### Relationship to FLUX+PAIR Origins

Context Feedback Looping evolved directly from your earlier **PAIR (Past Archived Information Re-injection)** concept. PAIR used vector databases (Qdrant) to store and retrieve context fragments. MCO stripped out the vector DB dependency and replaced it with **file-based SPIL parsing** — making the same conceptual operation (evaluate → identify gap → retrieve relevant context → re-inject) work through standard file I/O and MCP tooling instead of requiring embedding infrastructure.

The intellectual lineage:

```
PAIR (2025, early)              →    MCO Context Feedback Loop (2025, mid)
━━━━━━━━━━━━━━━━━━━                  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Vector DB storage                    SPIL file storage
Embedding-based retrieval            Parser-based injection
Qdrant dependency                    Framework-agnostic (MCP native)
Max 3 iterations                     Loop until ≥95% (convergent)
Context minimization                 Progressive Revelation (dosing)
```

The key upgrade: PAIR had a **hard iteration cap** (3 loops). MCO replaced this with a **quality-based exit condition** (the Validation Loop's 95% gate), which is more principled — you don't stop because you've tried three times, you stop because the work is actually good enough.

---

## III. The Validation Loop (95%+ Success Gate)

### The Principle

The Validation Loop is **MCO's immune system**. It exists because of one fundamental truth about LLMs:

> **An agent's self-assessment of completion has near-zero correlation with actual completion.**

Every developer who's worked with AI coding assistants has experienced the agent confidently declaring "Done! Here's your complete implementation" when what it produced is scaffolding with `// TODO: implement actual logic` comments scattered throughout.

The Validation Loop weaponizes this distrust into a formal mechanism:

### The Mechanism (Step by Step)

```
Step 1: Agent claims "done" with current step
            │
            ▼
Step 2: MCO triggers validation function
        (not optional — the agent doesn't get to skip this)
            │
            ▼
Step 3: Agent is FORCED to evaluate its own output against:
        ├── mco.core  (persistent: what am I building?)
        └── mco.sc    (persistent: what does "done" look like?)
            │
            ▼
Step 4: Agent produces a percentage-based score
        with justification for each criterion
            │
            ├── Score ≥ 95% → PASS → Orchestration completes
            │                        Present result to user
            │
            └── Score < 95% → FAIL → Context Feedback Loop activates
                                      Identify gaps
                                      Inject relevant SPIL
                                      Return to agent work phase
```

### Why 95% and Not 100%

The 95% threshold is a **pragmatic design choice**, not an arbitrary number. It accounts for:

1. **Diminishing returns** — Getting from 95% to 100% often requires exponentially more work than getting from 0% to 95%. The last 5% might involve edge cases that are better handled by a human.

2. **Evaluation imprecision** — The agent's self-scoring, even when forced, isn't perfectly calibrated. A strict 100% requirement would cause false-negative failures where work is actually complete but the agent can't score itself perfectly.

3. **Convergence risk** — At 100%, the loop might never terminate. At 95%, you get a convergent process that stops at "production-quality" rather than chasing theoretical perfection.

### The Structural Honesty Mechanism

Here's the subtlety that makes the Validation Loop more than a simple retry gate. When MCO forces the agent to evaluate its output, it's not asking *"are you done?"* — it's asking *"score yourself against these specific, persistent criteria that you cannot modify."*

The `mco.sc` file is in **persistent memory**. The agent can't forget the success criteria. It can't redefine what "done" means to match what it actually produced. The goalposts are bolted to the floor.

As you described on [GitHub Copilot](https://copilot.github.com) (~August 25, 2025, 4:07 PM):

> *"When an agent says 'OK, I'm done' then the MCO server triggers a function that causes the agent to evaluate and compare its 'finished' result to the success criteria and core vital features that it has in its persistent memory. Then it must give a score. Scores are percentage based — so if literally every core feature was implemented and the success criteria was met perfectly, then that would be 100%."*

The key phrase: **"triggers a function."** This isn't a suggestion or a prompt — it's a **server-side enforcement**. The MCO server interceptsthe completion claim and forces the evaluation. The agent doesn't choose to self-evaluate; it's *made* to.

### Validation as Injection Trigger

This is where the three systems **interlock**:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                    THE INTERLOCKING SYSTEM                                │
│                                                                          │
│  PROGRESSIVE REVELATION                                                  │
│  decides WHAT context exists and HOW it's tiered                        │
│         │                                                                │
│         │ feeds structure to                                             │
│         ▼                                                                │
│  CONTEXT FEEDBACK LOOP                                                   │
│  executes the injection/evaluation cycle                                │
│         │                                                                │
│         │ receives gap analysis from                                     │
│         ▼                                                                │
│  VALIDATION LOOP                                                         │
│  determines IF the cycle continues and WHAT gaps exist                  │
│         │                                                                │
│         │ gap analysis determines WHAT to inject next                    │
│         │ (feeds back into Progressive Revelation's dosing logic)        │
│         ▼                                                                │
│  PROGRESSIVE REVELATION                                                  │
│  selects targeted SPIL content based on identified gaps                 │
│         │                                                                │
│         └──────► CONTEXT FEEDBACK LOOP (next iteration)                 │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

The Validation Loop doesn't just say pass/fail. Its **gap analysis becomes the input** to Progressive Revelation's dosing decision. If the agent scored 72% and the missing criteria are all feature-related, then `mco.features` content gets injected. If it scored 88% and the gaps are all UI/presentation, then `mco.styles` gets injected. The validation output *drives* the injection logic.

This creates a **closed-loop control system** in the engineering sense:

- **Set point:** 95% satisfaction of `mco.sc`
- **Sensor:** Validation Loop (measures current state against criteria)
- **Error signal:** Gap between current score and 95%
- **Controller:** Progressive Revelation dosing algorithm
- **Actuator:** Context Feedback Loop (performs the injection)
- **Plant:** The LLM agent

---

## IV. The Three Systems as One: Convergent Orchestration

### Why They Must Be Together

Any one of these systems in isolation is insufficient:

| System Alone | What Goes Wrong |
|:---|:---|
| **Progressive Revelation only** | Agent gets information at the right time, but has no mechanism to check if it used it correctly. Premature completion persists. |
| **Context Feedback Loop only** | Agent receives re-injections, but without a quality gate, it doesn't know when to stop. Without tiered memory, injections are unfocused. |
| **Validation Loop only** | Agent is forced to self-evaluate, but without progressive context management, it's evaluating work that was done in a saturated context. Without feedback looping, a "fail" has no recovery mechanism. |

Together, they form a **convergent system** — one that mathematically trends toward completion:

1. **Progressive Revelation** ensures each work phase starts with a clean, focused context (minimize interference)
2. **Validation Loop** objectively measures progress against immutable criteria (detect gaps)
3. **Context Feedback Loop** closes the gap by injecting precisely what's missing (targeted correction)

Each iteration should score higher than the last. If it doesn't, that's a diagnostic signal — not a system failure, but an indication that the problem requires architectural intervention (different approach, decomposed subtasks, human input).

### The Anti-Pattern This Replaces

Before MCO, the standard agentic workflow was:

```
1. Give agent everything
2. Say "build this"
3. Agent says "done"
4. Developer checks → nothing works
5. Developer manually identifies problems
6. Developer re-prompts with corrections
7. Agent fixes some things, breaks others
8. Repeat steps 3-7 for hours/days
```

After MCO:

```
1. Load mco.core + mco.sc into persistent memory
2. Agent works on current step with focused context
3. Agent claims "done"
4. MCO forces evaluation against persistent success criteria
5. Score < 95% → MCO identifies gaps, injects targeted SPIL
6. Agent continues with enriched context
7. Score ≥ 95% → Orchestration completes
```

The human is removed from the correction loop. The **protocol itself** handles the forensic work of identifying what's missing and providing the right context to fix it.

---

## V. Comparison: MCO vs. GSD vs. Raw Agentic

Given the [GSD article you were reading](https://freedium.cfd) earlier today (~2:21 AM), it's worth placing MCO's three engines in the broader landscape:

| Dimension | Raw Agent Chat | GSD | MCO Protocol |
|:---|:---|:---|:---|
| **Context management** | Single thread, everything in one window | Fresh context per task, externalized to files | Progressive Revelation with persistent/semi-persistent tiers |
| **Memory** | Chat history (degrades) | File artifacts (`STATE.md`, `PROJECT.md`) | SPIL files with explicit memory tier assignments |
| **Quality gate** | None (trust agent) | Manual UAT + verify commands per task | Automated 95% validation loop against persistent `mco.sc` |
| **Failure recovery** | Manual re-prompting | Debug agents + fix plans | Automated context feedback loop with targeted SPIL injection |
| **Drift prevention** | None | Discuss phase locks decisions into files | `mco.core` + `mco.sc` in permanent persistent memory |
| **Granularity** | One long session | Phase → Plan → Task | Workflow → Step → Validation iteration |

The interesting overlap: GSD's **Nyquist Validation** concept (map test coverage to requirements *before* code is written) is conceptually adjacent to MCO's approach of making `mco.sc` persistent — both are saying *"define what 'done' looks like before you start, and never let the agent lose sight of it."*

The key difference: GSD achieves this through **workflow discipline and file conventions** (human-operated). MCO achieves it through **protocol-level enforcement** (server-triggered, automated). GSD is a methodology. MCO is a runtime.

---

## VI. Implications for ACT

Given your work on the [Agent Coordination Toolkit](file:///Users/cooperlyle/act), these three MCO engines have direct relevance to the multi-agent coordination problem you're solving:

- **Progressive Revelation** maps to how a coordinator agent should dose context to worker agents — AXIOM doesn't need NOVA's full reasoning history, just the relevant conclusions.
- **Context Feedback Looping** maps to how agents should re-check shared state — which is exactly what the `/loop` mechanism in Claude Code enables (as painfully rediscovered on [Thursday morning](file:///Users/cooperlyle/act) ~5:00 AM).
- **Validation Loop** maps to the FLUX-style self-verification you designed for ACT — agents that evaluate their own work against explicit criteria before claiming completion.

MCO proved these patterns work for single-agent orchestration. ACT is the multiplayer version of the same principles.

---

*This deep dive reconstructed from your MCO Protocol documentation, GitHub Copilot conversations (~August 25, 2025), Discord explanations (~August 22, 2025), and NotebookLM sessions (~August 22, 2025). Want me to format this as a standalone document for the repo, or break it into three separate reference docs?*