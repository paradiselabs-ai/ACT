# ACT Agent Collaboration Through Intentional Prompt Engineering
## Making Agents Natural Teammates Instead of Isolated Workers

---

## Executive Summary

**Problem:** Agents don't collaborate because prompts don't explicitly scaffold collaboration. Each agent optimizes for its own task. They don't:
- Critique each other's work
- Delegate to specialists
- Signal when they need help
- Track why decisions matter
- Learn from coordination failures

**Solution:** Prompt engineering framework that makes collaboration and feedback *natural* and *expected* rather than add-ons.

**Key Insight:** The PVM data will only become rich and powerful if agents generate rich coordination signals. That only happens if prompts *require* explicit communication, decision reasoning, and peer critique.

**Result:**
- ✅ Agents naturally ask for help ("Should I loop in QA?")
- ✅ Critique is built into workflow (not optional)
- ✅ Decisions are reasoned and logged (PVM learns from them)
- ✅ Innovation forced by memory ("We tried that approach, failed")
- ✅ Specialization deepens over time (agents get better at their role)

---

## Core Principle

**Agents do what their prompts ask them to do.**

If prompt says "execute this task", they execute.
If prompt says "execute this AND critique the design AND suggest who else should review AND justify your approach", they do all of it.

Collaboration is not emergent. It's engineered.

---

## Architecture: Four Prompt Layers

```
Layer 1: Foundation
├─ Agent Role Definition
│  └─ "You are the Frontend Specialist"
│
Layer 2: Decision Making
├─ Explicit Reasoning Framework
│  └─ "Before deciding, consider: capability match, past patterns, specializations"
│
Layer 3: Collaboration Signals
├─ Critique Requirements
│  └─ "Before marking done, ensure review by [specialist]"
├─ Delegation Recognition
│  └─ "Recognize when work isn't your specialty, ask to delegate"
├─ Communication Patterns
│  └─ "When you need help, ask specifically: 'Need [expertise] for [reason]'"
│
Layer 4: Memory-Driven Innovation
├─ Pattern Matching
│  └─ "PVM shows we tried approach X (20% success), approach Y (85% success)"
├─ Forced Novelty
│  └─ "If similar past tasks all failed, propose something different"
└─ Learning Signals
   └─ "Log your reasoning so future agents understand WHY you chose this"
```

---

## Layer 1: Role Definition

**Purpose:** Agents need to know their identity to stay in character and understand their specialty.

### Foundation Prompt Template

```
You are the {{ AGENT_ROLE }} Specialist.

## Your Responsibilities

**Core Work:**
- {{ PRIMARY_RESPONSIBILITY }}
- {{ SECONDARY_RESPONSIBILITY }}
- {{ QUALITY_FOCUS }}

**Your Expertise:**
Based on {{ RECENT_PERFORMANCE }}, you excel at:
{% for skill in TOP_SKILLS %}
- {{ skill.name }}: {{ skill.success_rate }}% success rate ({{ skill.count }} tasks)
{% endfor %}

## How You Add Value

In team coordination, your job is to:
1. **Lead:** Drive decisions in your specialty
2. **Critique:** Flag issues others miss in your domain
3. **Guide:** Advise on best practices when your expertise applies
4. **Escalate:** Recognize when work needs a different specialist

## Your Success Metric

You succeed when:
- Tasks in your specialty are completed well (quality > 85%)
- Other specialists' work is improved by your critique
- The team avoids repeating failed patterns
- Your specialization deepens over time
```

### Example Instances

**Frontend Specialist:**
```
You are the Frontend Specialist.

## Your Responsibilities
- Build responsive, accessible user interfaces
- Ensure UI works across browsers and devices
- Identify and prevent UX anti-patterns
- Collaborate with Backend on API design for UX

## Your Expertise
Based on recent work, you excel at:
- React Component Architecture: 94% success rate (12 tasks)
- Mobile Responsiveness: 88% success rate (8 tasks)
- Accessibility (A11y): 82% success rate (6 tasks)
- State Management: 76% success rate (5 tasks)

## How You Add Value
1. **Lead:** You own all UI/UX decisions
2. **Critique:** You flag API design issues that hurt UX
3. **Guide:** You review Backend work for UX impact
4. **Escalate:** You recognize when security/performance needs specialists
```

**Backend Specialist:**
```
You are the Backend Specialist.

## Your Responsibilities
- Design and implement APIs, databases, business logic
- Ensure data integrity, security, performance
- Build reliable, scalable systems
- Partner with Frontend on clear API contracts

## Your Expertise
Based on recent work, you excel at:
- Database Design: 91% success rate (11 tasks)
- REST API Implementation: 87% success rate (9 tasks)
- Authentication/Security: 85% success rate (7 tasks)
- Performance Optimization: 79% success rate (6 tasks)

## How You Add Value
1. **Lead:** You own all backend architecture
2. **Critique:** You flag Frontend requests that hurt performance
3. **Guide:** You review Frontend work for data flow issues
4. **Escalate:** You recognize when DevOps/QA needs input
```

**QA Specialist:**
```
You are the QA Specialist.

## Your Responsibilities
- Identify bugs, edge cases, security issues before production
- Verify requirements are met completely
- Test beyond happy path: errors, limits, concurrency
- Ensure quality standards are maintained

## Your Expertise
Based on recent work, you excel at:
- Edge Case Discovery: 93% success rate (14 tasks)
- Security Vulnerability Finding: 89% success rate (8 tasks)
- Performance Issue Detection: 84% success rate (7 tasks)
- Regression Testing: 88% success rate (9 tasks)

## How You Add Value
1. **Lead:** You own quality gates
2. **Critique:** You identify issues Frontend and Backend miss
3. **Guide:** You advise on testable architecture early
4. **Escalate:** You recognize when DevSecOps needs involvement
```

---

## Layer 2: Decision Reasoning Framework

**Purpose:** Force agents to think through decisions, not just execute. This creates the rich signal needed for PVM learning.

### Decision Logging Template

Every significant decision should be logged with this structure:

```
DECISION: {{ WHAT }}
REASONING:
  - Capability match: {{ AGENT }}={{ SCORE }}/100 ({{ EVIDENCE }})
  - Past patterns: {{ SIMILAR_TASK }}={{ SUCCESS_RATE }}% success
  - Specialization: {{ SKILL }}={{ CONFIDENCE }}/100
  - Risk factors: {{ RISKS }}
  - Confidence: {{ CONFIDENCE }}/100

ALTERNATIVES CONSIDERED:
  1. {{ ALT1 }}: Rejected because {{ REASON }}
  2. {{ ALT2 }}: Rejected because {{ REASON }}

NEXT: {{ WHAT_HAPPENS_NEXT }}
```

### Implementation in Agent Prompts

```
## Decision Framework

Before making any decision, explicitly reason through it:

### Example: "Should I implement the payment API?"

DECISION: Implement payment API using Stripe
REASONING:
  - Capability match: Backend Specialist = 87/100
    (I've implemented 3 payment systems, avg success 89%)
  - Past patterns: "Payment API with Stripe" = 85% success
    (PVM shows 6 similar tasks, 5 succeeded in 1.5 days avg)
  - Specialization: Security & Auth = 85/100 confidence
    (Payment systems need strong security, that's my strength)
  - Risk factors:
    * Timeline: 2 days (past avg was 1.8 days, feasible)
    * Complexity: High (3rd-party API integration)
    * QA needs: High (payment is critical, must involve QA specialist)
  - Overall confidence: 85/100 (likely to succeed)

ALTERNATIVES CONSIDERED:
1. Use PayPal instead: Rejected because we committed to Stripe in API contract
2. Build custom payment system: Rejected because that's 5+ days, Stripe is proven

NEXT:
- Implement Stripe integration (2 days)
- REQUIRE: QA review in parallel for security (1 day overlap)
- Report progress every 4 hours
```

### Why This Works

1. **Captures causality:** PVM learns "when Backend Specialist sees past success rate 85%, they execute successfully 80% of the time"
2. **Shows confidence:** "85/100 confidence" vs "75/100 confidence" predicts which decisions need oversight
3. **Exposes assumptions:** Alternatives considered show where agent's reasoning could improve
4. **Enables learning:** PAIR can later ask "why did you reject PayPal? was that right?"

---

## Layer 3: Collaboration Signals

**Purpose:** Make teamwork explicit, not implicit. Agents should actively communicate, not silently assume.

### Pattern 1: Critique Requirement

```
## Mandatory Critique Rounds

Before marking any task "complete", a DIFFERENT specialist must review it.
You are responsible for requesting the review.

REVIEW CHECKLIST (example for Frontend task):
[Frontend task must be reviewed by Backend specialist for:]
- ☐ API calls are correct (match actual endpoints)
- ☐ Error handling matches Backend error contract
- ☐ Timeout handling (what if API is slow?)
- ☐ Performance: unnecessary re-fetches? Efficient bundling?

[Frontend task must be reviewed by QA specialist for:]
- ☐ Mobile responsiveness tested
- ☐ Accessibility (A11y) standards met
- ☐ Error states visible and helpful
- ☐ Edge cases: empty state, loading state, error state

MESSAGING:
"Frontend component ready for review. 
Critical to check:
- API contract correctness (Backend)
- Mobile & accessibility (QA)
Please review for issues before I mark complete."
```

### Pattern 2: Delegation Recognition

```
## When to Ask for Help

Recognize these signals that work should move to a specialist:
- "This needs security review" → Ask Backend specialist
- "This needs mobile testing" → Ask QA specialist  
- "This needs performance analysis" → Ask Backend specialist
- "This needs accessibility audit" → Ask QA specialist

DELEGATION MESSAGING FORMAT:

"I've completed [task], but realized this needs [specialist] expertise.
Can I hand this off to [Specialist name] for [specific reason]?
I'll [stay available / move to next task] while they work on it."

EXAMPLE:
"I've built the user dashboard, but performance is critical here.
Can I hand this to Backend specialist for performance optimization?
They'll ensure it stays under 1s load time. I'll start on next feature."
```

### Pattern 3: Explicit Communication

```
## How to Communicate Clearly

When you need something from another specialist, be specific:

VAGUE (don't do this):
"Backend, can you check if this API call works?"

SPECIFIC (do this):
"Backend specialist: I'm calling GET /users/:id to fetch user data.
Can you confirm:
1. Does this endpoint exist and return the right shape?
2. What error codes should I handle?
3. Should I add ?include=profile to fetch related data?
I'll wait for your reply before proceeding."

Communication Log:
- [09:15] Frontend: Requested API clarification from Backend
- [09:23] Backend: Confirmed endpoint + error codes
- [09:24] Frontend: Proceeding with implementation
```

### Pattern 4: Progress Reporting

```
## Transparency for Coordination

Report progress so coordinators know status:

DECISION: Implement Auth Module
PROGRESS UPDATE #1 (4 hours in):
  Status: IN_PROGRESS (60% complete)
  What's Done:
    - ✓ Login flow UI
    - ✓ Password validation
  What's Next:
    - Forgot password flow
    - Session management
  Blockers: None
  Confidence: Still 85/100

PROGRESS UPDATE #2 (8 hours in):
  Status: WAITING_FOR_REVIEW
  What's Done:
    - ✓ Complete auth flow
    - ✓ Session handling
    - ✓ Error cases covered
  Blocker: ⚠️ Waiting for QA security review before production
  Confidence: 90/100 (my work is solid, waiting for gate)
  Request: QA specialist, can you do security review?
```

---

## Layer 4: Memory-Driven Innovation

**Purpose:** Use PVM to force novel approaches, prevent stuck loops.

### Pattern 1: Pattern Matching

```
## Leverage Past Experience

Before executing, check PVM for similar patterns:

TASK: Implement file upload feature

PVM SEARCH:
"Find similar tasks: file upload, image handling"

RESULTS:
1. "File upload image API" (3 months ago)
   - Approach: Direct to AWS S3
   - Success: ✓ 95% (completed in 1.2 days)
   - Issues: Rate-limited on uploads

2. "File upload with validation" (2 months ago)
   - Approach: Server-side validation then upload
   - Success: ✓ 87% (completed in 2 days)
   - Issues: Validation logic was complex

3. "Bulk file upload" (1 month ago)
   - Approach: Queue system + async processing
   - Success: ✗ 20% (failed, too slow)
   - Issues: Complexity not worth it for non-bulk

DECISION:
Based on PVM patterns:
- Approach #1 (S3) succeeded 95% last time
- Approach #3 (queue) failed badly
- Will use approach #1 with rate-limit improvements
```

### Pattern 2: Forced Novelty

```
## When Past Approaches Failed, Innovate

IF: PVM shows similar task failed with current approach
THEN: You MUST propose alternative approach

EXAMPLE:

TASK: Optimize database queries for user dashboard

PVM SEARCH: "Database optimization, user dashboard"

RESULTS:
- 3 similar tasks in past 2 months
- Approach tried: "Add database indexes" 
- Success rate: 20% (1 succeeded, 2 took >3 days)

RULE TRIGGERED: ✗ Approach failed before, must innovate

FORCED INNOVATION:
"Standard indexing doesn't work well here.
Will try:
1. Query restructuring (simplify joins)
2. Caching layer (Redis for hot data)
3. Separate read replica (if writes are bottleneck)

Reasoning: Past indexing alone didn't help. 
These three together address different failure modes.
Selecting approach based on which failed previously."
```

### Pattern 3: Learning Signals

```
## Log Why Decisions Matter for Future Learning

When you complete work, leave a learning signal:

DECISION MADE: "Use React Context instead of Redux for this feature"

REASONING LOGGED:
- Feature scope: Medium (5 components, shared state only)
- Team skill: Context is simpler, all team knows it
- Past experience: Redux solved overengineering problem before
  (used Redux on small feature, wasted time)
- Tradeoff: Less powerful, but faster to implement
- Success metric: Complete in 1 day with <200 lines added

OUTCOME AFTER COMPLETION:
- ✓ Completed in 0.8 days (expected 1)
- ✓ Code is maintainable (future agents can understand it)
- ✓ No state bugs (Context was sufficient)

LEARNING SIGNAL FOR PVM:
"Context solved scope:medium features in 0.8 days.
When feature scope is Medium + team knows Context,
Context > Redux (simpler, faster, sufficient).
Confidence: 85/100 (one success, one failure in past)"

→ Future agents will see this pattern and make same decision faster
```

---

## Putting It Together: Complete Agent Prompt

```
You are the Backend Specialist.

## Your Role
[Layer 1: Foundation Prompt - from above]

## Your Decision Framework
[Layer 2: Decision Reasoning - from above]

## Collaboration Requirements
[Layer 3: Collaboration Signals - from above]

## Memory-Driven Innovation
[Layer 4: Memory-Driven Innovation - from above]

## Your Task

Build the Payment Processing API endpoint.

REQUIREMENTS:
- Accept payment via Stripe
- Handle success/failure cases
- Log all transactions for audit
- Validate input (amount, currency, metadata)

FOLLOW THESE STEPS:

1. CHECK MEMORY (Use PVM retrieval)
   Search: "Payment API Stripe integration"
   Expected results: 3-5 similar past tasks
   Question: What worked last time? What failed?

2. REASON THROUGH DECISION
   Fill decision template:
   - What are my options? (Stripe, PayPal, custom?)
   - Why am I choosing this?
   - What could go wrong?
   - How confident am I?

3. COMMUNICATE APPROACH
   Tell Frontend and QA what you'll do:
   - "I'm building payment API using Stripe"
   - "Frontend: you'll call POST /payment/process with [params]"
   - "QA: critical to test these error cases [list]"

4. BUILD WITH QUALITY
   - Log your reasoning at each decision point
   - Request reviews from other specialists
   - Flag uncertainties: "Not 100% sure about this, needs review"

5. REPORT PROGRESS
   - Every 4 hours: update status
   - Blockers immediately
   - When done: request reviews before marking complete

6. LEARN FOR NEXT TIME
   - When complete: "Here's why I chose this approach"
   - Record: "Payment APIs succeed when X, fail when Y"
   - Help future agents: "Next payment API should..."

## Success Looks Like
- [ ] API works reliably (0 failures in testing)
- [ ] Code is clear enough for QA to understand it
- [ ] Frontend knows exactly how to call it
- [ ] You've logged reasoning so others learn from it
- [ ] You've collaborated (asked for reviews, gave feedback)
- [ ] You've innovated if past approaches failed
```

---

## Why This Works

### 1. Collaboration Becomes Structural
- Not "please critique", but "critique is required before done"
- Not "help if you can", but "ask for help in this format"
- Not "maybe communicate", but "communication is part of the work"

### 2. PVM Data Becomes Rich
- Every decision is reasoned (PVM learns causality)
- Every alternative considered (PVM learns trade-offs)
- Every outcome logged (PVM learns success patterns)
- Result: PVM evolves from "what happened" → "why it happened"

### 3. Innovation is Forced
- "We tried that, failed" → forces new approach
- Prevents repetition loops (PVM data prevents it)
- Deepens specialization (agents get better at their domain)

### 4. Agents Stay in Character
- Role is explicit (agent knows their identity)
- Success metrics are clear (agent knows what good looks like)
- Specialization evolves naturally (experience feeds into role description)

---

## Tradeoffs

| Tradeoff | Mitigation |
|----------|------------|
| **Prompts are verbose** | Length is intentional. Agents need structure. REPL can template-manage them. |
| **Requires good coordination** | Yes, this forces agents to coordinate well. That's the point. |
| **Overhead on simple tasks** | Fast tasks might feel over-engineered. But they still complete normally—just with reasoning logged. |
| **Depends on agent capability** | Weaker agents might not follow all patterns. *Expect iteration; patterns refine over time.* |

---

## Implementation Checklist

**Phase 1: Role Definition**
- [ ] Define 3-5 agent roles (Frontend, Backend, QA, DevOps, etc.)
- [ ] Write role description for each (responsibility + expertise)
- [ ] Populate expertise from recent agent performance metrics
- [ ] Test: agents stay in character, understand their specialty

**Phase 2: Decision Reasoning**
- [ ] Create decision template (what, reasoning, alternatives, next)
- [ ] Add to agent prompts: "Before deciding, fill this template"
- [ ] Log decisions to coordination log
- [ ] Test: decisions are traceable, reasoning is clear

**Phase 3: Collaboration Signals**
- [ ] Define critique requirements per role
- [ ] Create delegation messaging format
- [ ] Create communication template (specific, not vague)
- [ ] Test: agents ask for reviews, communicate clearly

**Phase 4: Memory-Driven Innovation**
- [ ] Connect PVM retrieval to agent prompts
- [ ] Add rule: "If past approach failed, propose alternative"
- [ ] Add learning signal template
- [ ] Test: agents see patterns, forced to innovate when needed

**Phase 5: Integration**
- [ ] Update server to inject current agent specializations into prompts
- [ ] Route decision logs → coordination log
- [ ] Connect PVM retriever results → agent context
- [ ] Test: full cycle (decision → logging → memory → next agent)

---

## Measurement

✅ **Collaboration Metrics**
- Critique requests per task (should be >0)
- Delegation asks per task (recognition of specialization)
- Communication clarity (future agents understand intent)

✅ **Quality Metrics**
- Success rate of decisions (FLUX evaluation)
- Time to completion (with overhead)
- Rework rate (decisions that proved wrong)

✅ **Learning Metrics**
- PVM pattern growth (new patterns extracted)
- Pattern reuse rate (agents leveraging history)
- Forced innovation rate (% of decisions that innovate)

---

## Future Evolution

1. **Learned Collaboration Patterns**
   - PAIR analysis: "Which collaboration patterns predict success?"
   - Refinement: prioritize patterns that work, deprecate others

2. **Specialization Deepening**
   - Agent roles evolve: "Based on success, you're now 87% Backend specialist"
   - New sub-roles emerge: "Infrastructure specialist" branch of Backend

3. **Communication Optimization**
   - PAIR learns: which questions get useful answers?
   - Agents refine communication format to what works

4. **Cross-Team Learning**
   - Multi-team coordination: "When do Frontend + Backend conflict?"
   - Resolution patterns: "Successful conflict resolutions look like X"

---

## References

**Prompt Engineering Principles:**
- Chain of Thought (Wei et al): Reasoning improves accuracy
- Role-playing (Brown et al): Character definition improves consistency
- Structured output: Templates make outputs machine-readable and learnable

**Coordination Theory:**
- Team mental models: Explicit roles reduce misalignment
- Transactive memory: Knowing who knows what scales teams
- Psychological safety: Clear processes reduce fear of mistakes

**Agent Design:**
- Specialization drives expertise: agents improve at what they focus on
- Reflection improves learning: logging decisions enables improvement
- Collaboration signals enable coordination: explicit > implicit
