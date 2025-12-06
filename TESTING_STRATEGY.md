# ACT Testing Strategy

**Created:** December 6, 2025
**Owner:** Claude Code Instance #2
**Status:** Active Development

---

## Overview

As ACT grows in complexity (Phase 5 PVM features), we need a comprehensive testing strategy to ensure:
1. **Baseline coordination continues working** as new features are added
2. **New PVM features work correctly** (memory, FLUX State, PAIR, /improve)
3. **Regression prevention** - changes don't break existing functionality
4. **Confidence in deployment** - tests pass = safe to ship

---

## Current State

### ✅ What Works (Verified Dec 6, 2025)

**Test File:** `examples/communicating_ai_demo.py`

**Verified Functionality:**
- ✅ Agent registration (WebSocket connection)
- ✅ Capability-based task assignment (scoring algorithm)
- ✅ Real-time task progress tracking (0% → 25% → 50% → 75% → 100%)
- ✅ Agent-to-agent communication (@mentions)
- ✅ Rate limiting with graceful fallbacks
- ✅ Task completion workflow (analysis → planning → implementation → completion)
- ✅ Zero human intervention required

**Test Results:**
- 2 agents registered (Alex: designer, Morgan: analyst)
- 2 tasks completed (dashboard wireframe, API documentation)
- Perfect capability matching (1.00 vs 0.40 scores)
- Real AI API calls (OpenRouter: Mistral 7B + Gemini 2 9B)

---

## Test Architecture

### Pyramid Structure

```
                    /\
                   /  \
                  / E2E \          ← End-to-end (slowest, most realistic)
                 /______\
                /        \
               / INTEGRA- \        ← Integration (medium speed)
              /    TION    \
             /_____________ \
            /                \
           /    UNIT TESTS    \   ← Unit (fastest, most isolated)
          /____________________\
```

### Test Levels

#### 1. Unit Tests (Fast, Isolated)
**Purpose:** Test individual functions/classes in isolation

**Examples:**
- Capability matching algorithm
- Task scoring calculation
- Message routing logic
- PVM log appending
- Vector embedding generation

**Location:** `tests/unit/`

**Speed:** Milliseconds per test

---

#### 2. Integration Tests (Medium, Realistic)
**Purpose:** Test multiple components working together

**Examples:**
- PVM log → vector store → RAG retrieval flow
- FLUX State evaluation → PAIR reasoning → improvement loop
- AgentProfileBuilder deriving profiles from coordination history
- /improve command parsing → execution → output generation

**Location:** `tests/integration/`

**Speed:** Seconds per test

---

#### 3. End-to-End Tests (Slow, Comprehensive)
**Purpose:** Test entire system with real agents and API calls

**Examples:**
- 2-agent coordination demo (current baseline)
- 5-agent swarm coordination
- PVM improvement workflow (task → FLUX → PAIR → retry)
- Full /improve command with all parameters

**Location:** `tests/e2e/`

**Speed:** Minutes per test

---

## Test Suite Plan

### Phase 1: Preserve Baseline (ASAP)

**Goal:** Ensure current functionality never regresses

**Tasks:**
1. ✅ Run existing demo (`examples/communicating_ai_demo.py`) - DONE
2. Create `tests/e2e/test_baseline_coordination.py`
   - Automated version of current demo
   - Asserts task completion
   - Asserts capability matching works
   - Asserts agent communication works
3. Create `tests/unit/test_task_assignment.py`
   - Test capability scoring algorithm
   - Test workload balancing
   - Test conflict resolution
4. Add CI/CD workflow (GitHub Actions)
   - Run tests on every commit
   - Fail PR if tests break

**Success Criteria:**
- Baseline tests pass consistently
- Can run with `pytest tests/e2e/test_baseline_coordination.py`
- Takes < 2 minutes to complete

---

### Phase 2: PVM Core Tests (Next)

**Goal:** Test PVM append-only log and vector indexing

**Unit Tests:**
```python
# tests/unit/test_pvm_log.py
def test_append_coordination_minute():
    """Test PVM log appends without modifying existing entries"""
    log = ChronologicalLog()
    minute1 = create_test_minute("task_assigned", agent="alex")
    minute2 = create_test_minute("task_completed", agent="alex")

    log.append(minute1)
    log.append(minute2)

    assert len(log.get_all()) == 2
    assert log.get_all()[0] == minute1  # Immutable
    assert log.get_all()[1] == minute2

def test_immutability():
    """Test that log entries can't be modified after appending"""
    log = ChronologicalLog()
    minute = create_test_minute("task_assigned", agent="alex")

    log.append(minute)
    minute["agent_id"] = "hacker"  # Try to modify

    assert log.get_all()[0]["agent_id"] == "alex"  # Original preserved
```

**Integration Tests:**
```python
# tests/integration/test_pvm_rag.py
def test_vector_search_retrieves_similar_patterns():
    """Test RAG retrieval finds semantically similar coordination patterns"""
    memory = ACTMemorySystem()

    # Record past coordination patterns
    memory.append(create_minute("react_task_succeeded", agent="alex"))
    memory.append(create_minute("vue_task_failed", agent="alex"))
    memory.append(create_minute("react_task_succeeded", agent="morgan"))

    # Query for similar patterns
    results = memory.query("React frontend task assignment")

    assert len(results) >= 2
    assert "react" in results[0].description.lower()
    assert results[0].metadata.composite_score > 0.7
```

**Success Criteria:**
- PVM log is truly append-only (no modifications possible)
- Vector search retrieves semantically relevant coordination patterns
- Context window manager balances recent + referenced + RAG

---

### Phase 3: FLUX State + PAIR Tests (After PVM Core)

**Goal:** Test memory-wipe evaluation and RAG-guided improvement

**Integration Tests:**
```python
# tests/integration/test_flux_state.py
async def test_flux_state_unbiased_evaluation():
    """Test FLUX State provides unbiased evaluation via memory wipe"""

    # Agent completes task (with memory)
    original_task = "Build React dashboard"
    agent_output = await alex.complete_task(original_task)

    # FLUX State evaluation (memory wiped)
    evaluation = await flux_state_evaluate(
        task=original_task,
        success_criteria="Responsive, accessible, follows design system",
        output=agent_output,
        agent_id=None  # No memory of who created it
    )

    # Should identify gaps critically
    assert evaluation.success_percentage < 95  # Not perfect
    assert len(evaluation.identified_gaps) > 0
    assert "accessibility" in str(evaluation.identified_gaps).lower()

# tests/integration/test_pair_reasoning.py
async def test_pair_improves_outcome():
    """Test PAIR retrieval improves coordination outcomes"""

    # Initial attempt (no context)
    attempt1 = await coordinator.assign_task(
        "Complex React state management refactor",
        context_enabled=False
    )

    # PAIR-guided attempt (with RAG context)
    attempt2 = await coordinator.assign_task(
        "Complex React state management refactor",
        context_enabled=True
    )

    # PAIR should improve assignment quality
    assert attempt2.confidence_score > attempt1.confidence_score
    assert len(attempt2.reasoning) > len(attempt1.reasoning)
```

**Success Criteria:**
- FLUX State evaluates critically without bias
- PAIR retrieves relevant coordination patterns
- Improvement loop converges to 95%+ success criteria

---

### Phase 4: AgentProfileBuilder Tests (After FLUX/PAIR)

**Goal:** Test evidence-based agent profiling

**Integration Tests:**
```python
# tests/integration/test_agent_profiling.py
async def test_builds_evidence_based_profile():
    """Test AgentProfileBuilder derives accurate profiles from coordination history"""

    profiler = AgentProfileBuilder(memory_system)

    # Simulate coordination history
    await simulate_tasks([
        ("React hooks refactor", agent="alex", success=True),
        ("React class components", agent="alex", success=False),
        ("Vue.js dashboard", agent="alex", success=False),
        ("React hooks optimization", agent="alex", success=True),
    ])

    # Build profile
    profile = await profiler.build_profile("alex")

    # Should identify React hooks as strength
    assert profile.skills["react_hooks"].success_rate > 0.90
    assert profile.skills["react_class_components"].success_rate < 0.50

    # Should recommend React hooks assignments
    assert "React hooks" in profile.recommendations
```

**Success Criteria:**
- Profiles derived from actual outcomes (not self-reported)
- Skill progression tracked over time
- Recommendations accurate for future assignments

---

### Phase 5: /improve Command Tests (Final)

**Goal:** Test surgical precision user-controlled improvement

**E2E Tests:**
```python
# tests/e2e/test_improve_command.py
async def test_improve_communication_scope():
    """Test /improve command with communication scope"""

    # Run coordination session with communication issues
    session = await run_coordination_session([
        ("Build dashboard", assigned_to="alex"),
        ("Write API docs", assigned_to="morgan")
    ])

    # Run /improve command
    result = await coordinator.improve(
        scope="communication",
        agents=["alex", "morgan"],
        session_id=session.id,
        filter="bad",
        output="detailed-report"
    )

    # Should identify communication gaps
    assert len(result.issues_found) > 0
    assert "clarity" in str(result.issues_found).lower()

    # Should provide actionable recommendations
    assert len(result.recommendations) > 0
    assert result.output_format == "detailed-report"

async def test_improve_all_scopes():
    """Test /improve command supports all 6 scopes"""

    scopes = ["communication", "tools", "assignments",
              "conflicts", "collaboration", "performance"]

    for scope in scopes:
        result = await coordinator.improve(scope=scope)
        assert result.scope == scope
        assert result.analysis_completed == True
```

**Success Criteria:**
- All 6 scopes work correctly
- All filters work (good/bad/all)
- All output formats work (summary/detailed-report/action-items)
- Agents + session filtering works
- Analysis is surgical (not generic)

---

## Test Data Strategy

### Mock vs. Real Data

**Unit Tests:** Use mock data (fast, deterministic)
```python
def create_test_minute(event_type, agent, **kwargs):
    return {
        "id": str(uuid.uuid4()),
        "timestamp": "2025-12-06T10:00:00Z",
        "event_type": event_type,
        "agent_id": agent,
        **kwargs
    }
```

**Integration Tests:** Use test database (realistic, isolated)
```python
@pytest.fixture
def test_memory_system():
    memory = ACTMemorySystem(db="test_pvm.db")
    yield memory
    memory.clear()  # Clean up after test
```

**E2E Tests:** Use real APIs with rate limiting (realistic, slow)
```python
@pytest.fixture
def openrouter_client():
    return OpenRouterClient(
        api_key=os.getenv("OPENROUTER_API_KEY"),
        rate_limit=3  # seconds between calls
    )
```

---

## CI/CD Integration

### GitHub Actions Workflow

```yaml
# .github/workflows/test.yml
name: ACT Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v3

    - name: Setup Python
      uses: actions/setup-python@v4
      with:
        python-version: '3.11'

    - name: Setup Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '18'

    - name: Install dependencies
      run: |
        pip install -r requirements.txt
        cd sdk/python/server && npm install

    - name: Run unit tests
      run: pytest tests/unit -v

    - name: Run integration tests
      run: pytest tests/integration -v

    - name: Run E2E tests (if API key available)
      if: ${{ secrets.OPENROUTER_API_KEY }}
      env:
        OPENROUTER_API_KEY: ${{ secrets.OPENROUTER_API_KEY }}
      run: pytest tests/e2e -v --slow
```

---

## Test Coverage Goals

**Phase 5 Completion Targets:**
- **Unit Tests:** 80% code coverage
- **Integration Tests:** All critical paths tested
- **E2E Tests:** 3+ realistic scenarios

**Minimum Coverage:**
- PVM append-only log: 100%
- FLUX State evaluation: 100%
- PAIR reasoning: 100%
- AgentProfileBuilder: 90%
- /improve command: 100%

---

## Running Tests

### Quick Start

```bash
# Run all tests
pytest

# Run specific test level
pytest tests/unit          # Fast (seconds)
pytest tests/integration   # Medium (30s-2min)
pytest tests/e2e          # Slow (2-5min)

# Run specific test file
pytest tests/unit/test_pvm_log.py

# Run with coverage
pytest --cov=src --cov-report=html

# Run only fast tests (skip E2E)
pytest -m "not slow"
```

### Test Markers

```python
@pytest.mark.slow  # E2E tests requiring API calls
@pytest.mark.integration  # Integration tests
@pytest.mark.unit  # Unit tests (fast)
```

---

## Next Steps

**Immediate (High Priority):**
1. Create `tests/` directory structure
2. Write baseline E2E test (`test_baseline_coordination.py`)
3. Write PVM core unit tests (`test_pvm_log.py`, `test_vector_store.py`)
4. Set up pytest configuration

**Soon After:**
5. Add FLUX State + PAIR integration tests
6. Add AgentProfileBuilder tests
7. Add /improve command E2E tests
8. Set up GitHub Actions CI/CD

**Ongoing:**
- Add tests for each new feature
- Maintain 80%+ code coverage
- Run tests before every commit
- Update tests when requirements change

---

## Success Metrics

**Testing is successful when:**
- ✅ All tests pass on main branch
- ✅ No regressions when adding new features
- ✅ Developers can run tests locally in < 5 minutes
- ✅ CI/CD catches bugs before deployment
- ✅ Coverage stays above 80%
- ✅ Tests are readable and maintainable

---

**Last Updated:** December 6, 2025
**Status:** Baseline coordination verified, ready to build test suite
