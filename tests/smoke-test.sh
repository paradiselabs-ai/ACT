#!/bin/bash
# ACT Smoke Test — verifies core server endpoints work
# Requires: ACT server running on localhost:8080
# Usage: bash tests/smoke-test.sh

BASE="http://localhost:8080"
PASS=0
FAIL=0

check() {
  local desc="$1" expected="$2" actual="$3"
  if echo "$actual" | grep -q "$expected"; then
    echo "  ✓ $desc"
    ((PASS++))
  else
    echo "  ✗ $desc (expected: $expected)"
    echo "    got: $actual"
    ((FAIL++))
  fi
}

echo "ACT Smoke Test"
echo "=============="

# 1. Register agent
echo ""
echo "1. Agent Registration"
R=$(curl -s -X POST "$BASE/api/agents/register" \
  -H "Content-Type: application/json" \
  -d '{"agentId":"test-agent-1","name":"Test Agent","capabilities":["testing"]}')
check "register agent" "success" "$R"

# 2. Create task
echo ""
echo "2. Task Creation"
R=$(curl -s -X POST "$BASE/api/tasks" \
  -H "Content-Type: application/json" \
  -d '{"description":"Test task for smoke test","title":"Smoke Test Task","assignedAgent":"test-agent-1","priority":"medium"}')
TASK_ID=$(echo "$R" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
check "create task" "id" "$R"
echo "    task_id=$TASK_ID"

# 3. Get task by ID
echo ""
echo "3. Get Task by ID"
R=$(curl -s "$BASE/api/tasks/$TASK_ID")
check "get task by id" "$TASK_ID" "$R"

# 4. Update progress
echo ""
echo "4. Task Progress"
R=$(curl -s -X POST "$BASE/api/tasks/$TASK_ID/progress" \
  -H "Content-Type: application/json" \
  -d '{"agentId":"test-agent-1","progress":50,"status":"in_progress","message":"halfway done"}')
check "update progress" "success" "$R"

# 5. Complete task
echo ""
echo "5. Task Completion"
R=$(curl -s -X POST "$BASE/api/tasks/$TASK_ID/complete" \
  -H "Content-Type: application/json" \
  -d '{"agentId":"test-agent-1","success":true,"result":"Smoke test completed"}')
check "complete task" "success" "$R"

# 6. Submit for validation
echo ""
echo "6. Submit for Validation"
R=$(curl -s -X POST "$BASE/api/tasks/$TASK_ID/submit-for-validation" \
  -H "Content-Type: application/json" \
  -d '{"agentId":"test-agent-1","selfVerification":"Verified output matches spec"}')
check "submit for validation" "success" "$R"

# 7. Check validation queue
echo ""
echo "7. Validation Queue"
R=$(curl -s "$BASE/api/tasks/pending-validation")
check "pending validation" "$TASK_ID" "$R"

# 8. Submit validation verdict (pass)
echo ""
echo "8. Validation Verdict"
R=$(curl -s -X POST "$BASE/api/tasks/$TASK_ID/validation-verdict" \
  -H "Content-Type: application/json" \
  -d '{"agentId":"assurance-1","passed":true,"score":97,"criteriaResults":[{"criterion":"test","passed":true,"score":97}]}')
check "validation verdict" "validated" "$R"

# 9. Check validated queue
echo ""
echo "9. Validated Queue"
R=$(curl -s "$BASE/api/tasks/validated")
check "validated tasks" "$TASK_ID" "$R"

# 10. Send message
echo ""
echo "10. Messaging"
R=$(curl -s -X POST "$BASE/api/messages" \
  -H "Content-Type: application/json" \
  -d '{"sender":"test-agent-1","message":"status: smoke test completed"}')
check "send message" "success" "$R"

# 11. Get messages
echo ""
echo "11. Message Inbox"
R=$(curl -s "$BASE/api/agents/test-agent-1/messages?limit=5")
check "get messages" "messages" "$R"

# 12. PVM search
echo ""
echo "12. PVM Search"
R=$(curl -s "$BASE/api/pvm/search?query=test&limit=3")
check "pvm search" "results" "$R"

# 13. File locking
echo ""
echo "13. File Locking"
R=$(curl -s -X POST "$BASE/api/files/claim" \
  -H "Content-Type: application/json" \
  -d "{\"agent_id\":\"test-agent-1\",\"task_id\":\"$TASK_ID\",\"file_paths\":[\"src/test.ts\"]}")
check "claim file" "success" "$R"

# 14. File conflict detection
echo ""
echo "14. File Conflict"
R=$(curl -s -X POST "$BASE/api/files/claim" \
  -H "Content-Type: application/json" \
  -d "{\"agent_id\":\"other-agent\",\"task_id\":\"other\",\"file_paths\":[\"src/test.ts\"]}")
check "conflict detected" "conflict" "$R"

# 15. File release
echo ""
echo "15. File Release"
R=$(curl -s -X POST "$BASE/api/files/release" \
  -H "Content-Type: application/json" \
  -d "{\"agent_id\":\"test-agent-1\",\"task_id\":\"$TASK_ID\",\"file_paths\":[\"src/test.ts\"]}")
check "release file" "success" "$R"

# 16. SNLP success criteria extraction
echo ""
echo "16. SNLP Extraction"
SNLP_TASK=$(curl -s -X POST "$BASE/api/tasks" \
  -H "Content-Type: application/json" \
  -d '{"title":"SNLP Test","description":"@success_criteria\n- Code compiles\n- Tests pass\n- No regressions","requiredCapabilities":["testing"],"priority":"low"}')
SNLP_ID=$(echo "$SNLP_TASK" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
curl -s -X POST "$BASE/api/tasks/$SNLP_ID/complete" -H "Content-Type: application/json" \
  -d '{"agentId":"test-agent-1","success":true,"result":"done"}' > /dev/null
curl -s -X POST "$BASE/api/tasks/$SNLP_ID/submit-for-validation" -H "Content-Type: application/json" \
  -d '{"agentId":"test-agent-1"}' > /dev/null
R=$(curl -s "$BASE/api/tasks/pending-validation")
check "snlp criteria extracted" "successCriteria" "$R"

# 17. A2A Agent Card
echo ""
echo "17. A2A Agent Card"
R=$(curl -s "$BASE/.well-known/agent.json")
check "system agent card" "ACT" "$R"

# 18. Per-agent card
echo ""
echo "18. Per-Agent Card"
R=$(curl -s "$BASE/api/agents/test-agent-1/agent.json")
check "agent card" "test-agent-1" "$R"

# 19. Project CRUD
echo ""
echo "19. Project Creation"
R=$(curl -s -X POST "$BASE/api/projects" \
  -H "Content-Type: application/json" \
  -d '{"name":"smoke-proj","workspace":"/tmp/smoke","description":"test"}')
check "create project" "success" "$R"

# 20. Brief store/retrieve
echo ""
echo "20. Brief Store/Retrieve"
curl -s -X POST "$BASE/api/projects/smoke-proj/briefs" \
  -H "Content-Type: application/json" \
  -d '{"agentId":"test-agent-1","content":"# Test Brief"}' > /dev/null
R=$(curl -s "$BASE/api/projects/smoke-proj/briefs/test-agent-1")
check "brief retrieve" "Test Brief" "$R"

# 21. Dependency chain
echo ""
echo "21. Dependency Chain"
DEP1=$(curl -s -X POST "$BASE/api/tasks" \
  -H "Content-Type: application/json" \
  -d '{"title":"Dep Parent","description":"parent","requiredCapabilities":["testing"],"priority":"high"}' | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
DEP2=$(curl -s -X POST "$BASE/api/tasks" \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"Dep Child\",\"description\":\"child\",\"requiredCapabilities\":[\"testing\"],\"dependencies\":[\"$DEP1\"]}")
check "task with dependency" "id" "$DEP2"

# 22. System status
echo ""
echo "22. System Status"
R=$(curl -s "$BASE/api/status")
check "system status" "tasks" "$R"

# 23. Dev reset
echo ""
echo "23. Dev Reset"
R=$(curl -s -X POST "$BASE/api/dev/reset")
check "dev reset" "success" "$R"

# Verify reset worked
R=$(curl -s "$BASE/health")
check "post-reset clean" '"agents":0' "$R"

echo ""
echo "=============="
echo "Results: $PASS passed, $FAIL failed"
[ $FAIL -eq 0 ] && echo "ALL TESTS PASSED" || echo "SOME TESTS FAILED"
exit $FAIL
