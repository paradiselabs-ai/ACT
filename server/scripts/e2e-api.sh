#!/bin/bash
# E2E for PVM rebuild chain (W01-10 + W10-followup-a):
# Drives the server HTTP API through a realistic 2-project, 3-agent, 6-task workflow,
# then asserts every PVM endpoint returns evidence-backed numbers (not placeholders).
set +e

SERVER_DIR=/Users/user/Documents/Developer/dev/AI/act/server
LOG=/tmp/pvm-e2e.log
SERVER_PID=""
BASE=http://localhost:8080
PASS=0; FAIL=0

assert() {
  local label="$1" expected="$2" actual="$3"
  if [ "$actual" = "$expected" ]; then
    echo "  ✅ $label  [$actual]"
    PASS=$((PASS+1))
  else
    echo "  ❌ $label  expected=[$expected] actual=[$actual]"
    FAIL=$((FAIL+1))
  fi
}
assert_gt() {
  local label="$1" threshold="$2" actual="$3"
  if [ -n "$actual" ] && [ "$actual" != "null" ] && [ "$actual" -gt "$threshold" ] 2>/dev/null; then
    echo "  ✅ $label  [$actual > $threshold]"
    PASS=$((PASS+1))
  else
    echo "  ❌ $label  expected > $threshold actual=[$actual]"
    FAIL=$((FAIL+1))
  fi
}

cleanup() { [ -n "$SERVER_PID" ] && kill -9 $SERVER_PID 2>/dev/null; pkill -9 -f "tsx watch.*server/src/index.ts" 2>/dev/null; }
trap cleanup EXIT

# ────────────────────────────────────────────────────────────────
echo "═══════════════════════════════════════════════════════"
echo " PVM E2E — 2 projects × 3 agents × 6 task lifecycles"
echo "═══════════════════════════════════════════════════════"

pkill -9 -f "tsx watch.*server/src/index.ts" 2>/dev/null
sleep 2
cd "$SERVER_DIR"
rm -f "$LOG"
npm run dev > "$LOG" 2>&1 &
SERVER_PID=$!

for i in $(seq 1 30); do
  if curl -s --max-time 1 "$BASE/health" >/dev/null 2>&1; then break; fi
  sleep 1
done
sleep 3

# ────────────────────────────────────────────────────────────────
echo
echo "── Step 1: Create 2 projects ──"
for P in alpha beta; do
  R=$(curl -s -X POST -H "Content-Type: application/json" \
    -d "{\"name\":\"$P\",\"workspace\":\"/tmp/$P\",\"description\":\"E2E project $P\",\"techStack\":\"node\",\"successCriteria\":\"all tasks pass validation\"}" \
    "$BASE/api/projects")
  S=$(echo "$R" | jq -r '.success // false')
  assert "project $P created" "true" "$S"
done

# ────────────────────────────────────────────────────────────────
echo
echo "── Step 2: Register 5 project-prefixed agents ──"
# Post-bleed-fix: registry keys globally by agentId, so cross-project reuse of
# "dev-1" collides. Prefix per project to keep identities distinct (matches
# the pattern in /tmp/bleed_e2e.sh).
for SPEC in dev-1-alpha:alpha:coding backend-1-alpha:alpha:coding,backend qa-1-alpha:alpha:testing dev-1-beta:beta:coding backend-1-beta:beta:coding,backend; do
  IFS=':' read -r A P CAPLIST <<< "$SPEC"
  CAP="[\"$(echo $CAPLIST | sed 's/,/","/g')\"]"
  R=$(curl -s -X POST -H "Content-Type: application/json" \
    -d "{\"agentId\":\"$A\",\"name\":\"$A\",\"projectName\":\"$P\",\"capabilities\":$CAP}" \
    "$BASE/api/agents/register")
  S=$(echo "$R" | jq -r '.success // false')
  assert "agent $A registered in $P" "true" "$S"
done

# ────────────────────────────────────────────────────────────────
echo
echo "── Step 3: Create + complete + validate 6 tasks ──"
# 4 in alpha, 2 in beta. Mix of agents + task types.
TASKS=(
  "alpha:dev-1-alpha:python:Write a CSV parser"
  "alpha:dev-1-alpha:python:Add JSON serialization"
  "alpha:backend-1-alpha:api:Build REST endpoint"
  "alpha:backend-1-alpha:api:Add auth middleware"
  "beta:dev-1-beta:typescript:Migrate config to TS"
  "beta:backend-1-beta:api:Database schema migration"
)

TASK_IDS=()
for spec in "${TASKS[@]}"; do
  IFS=':' read -r PROJ AGENT TYPE DESC <<< "$spec"
  # Create task scoped to project + agent
  R=$(curl -s -X POST -H "Content-Type: application/json" \
    -d "{\"description\":\"$DESC\",\"requiredCapabilities\":[\"coding\"],\"priority\":\"medium\",\"assignedAgent\":\"$AGENT\",\"metadata\":{\"projectName\":\"$PROJ\",\"taskType\":\"$TYPE\"}}" \
    "$BASE/api/tasks")
  TID=$(echo "$R" | jq -r '.task.id // .id // empty')
  if [ -z "$TID" ]; then
    echo "  ❌ task create failed: $R"
    FAIL=$((FAIL+1))
    continue
  fi
  TASK_IDS+=("$TID:$AGENT:$PROJ:$TYPE")

  # Complete
  curl -s -X POST -H "Content-Type: application/json" \
    -d "{\"agentId\":\"$AGENT\",\"success\":true,\"result\":\"done: $DESC\"}" \
    "$BASE/api/tasks/$TID/complete" >/dev/null

  # Submit for validation
  curl -s -X POST -H "Content-Type: application/json" \
    -d "{\"agentId\":\"$AGENT\",\"selfVerification\":{\"checks\":[\"compiles\",\"tested\"]}}" \
    "$BASE/api/tasks/$TID/submit-for-validation" >/dev/null

  # Validation verdict (pass)
  curl -s -X POST -H "Content-Type: application/json" \
    -d "{\"agentId\":\"assurance\",\"passed\":true,\"score\":95,\"criteriaResults\":[],\"gaps\":[],\"feedback\":\"all good\"}" \
    "$BASE/api/tasks/$TID/validation-verdict" >/dev/null

  echo "  task $PROJ/$AGENT/$TYPE/${TID:0:8} ✓ lifecycle complete"
done

# Wait for PVM indexer to ingest (10s polling loop)
echo
echo "── Step 4: Wait for indexer ──"
for i in $(seq 1 20); do
  CT=$(curl -s "$BASE/api/pvm/status" | jq -r '.indexedEventCount // 0')
  if [ "$CT" -gt 20 ] 2>/dev/null; then
    echo "  indexer caught up: $CT events"
    break
  fi
  sleep 2
done

# ────────────────────────────────────────────────────────────────
echo
echo "── Step 5: Cursor + reindex assertions ──"
B1=$(curl -s "$BASE/api/pvm/status" | jq -r '.lastIndexedTimestamp // empty')
curl -s -X POST "$BASE/api/pvm/reindex" >/dev/null
A1=$(curl -s "$BASE/api/pvm/status" | jq -r '.lastIndexedTimestamp // empty')
echo "  cursor before: $B1"
echo "  cursor after:  $A1"
if [ -n "$B1" ] && [ "$A1" \< "$B1" ]; then
  echo "  ❌ cursor regressed"
  FAIL=$((FAIL+1))
else
  echo "  ✅ cursor held or advanced"
  PASS=$((PASS+1))
fi

# ────────────────────────────────────────────────────────────────
echo
echo "── Step 6: Scoped search (project filter) ──"
R=$(curl -s "$BASE/api/pvm/search?query=REST+endpoint&limit=5&project=alpha")
SCOPE=$(echo "$R" | jq -r '.scope')
N=$(echo "$R" | jq -r '.results | length')
assert "scoped search returns scope=alpha" "alpha" "$SCOPE"
assert_gt "scoped search result count > 0" 0 "$N"
# Make sure no beta-only events leak in
BETA_LEAK=$(echo "$R" | jq -r '[.results[] | select(.message.projectName == "beta")] | length')
assert "no beta leakage in alpha-scoped search" "0" "$BETA_LEAK"

# Cross-project (no filter)
R=$(curl -s "$BASE/api/pvm/search?query=REST+endpoint&limit=10")
SCOPE=$(echo "$R" | jq -r '.scope')
assert "cross-project search scope" "cross-project" "$SCOPE"

# ────────────────────────────────────────────────────────────────
echo
echo "── Step 7: Profile JOIN (W10-followup-a fix) ──"
# PVM aggregates by agentId globally; per-project prefixes mean dev-1-alpha
# and dev-1-beta are separate identities, matching the architecture.
for A in dev-1-alpha backend-1-alpha; do
  R=$(curl -s "$BASE/api/pvm/profile?agentId=$A")
  CT=$(echo "$R" | jq -r '.profile.overallPerformance.completedTasks')
  SR=$(echo "$R" | jq -r '.profile.overallPerformance.successRate')
  TT=$(echo "$R" | jq -r '.profile.overallPerformance.totalTasks')
  REL=$(echo "$R" | jq -r '.profile.overallPerformance.reliability')
  echo "  $A: totalTasks=$TT completedTasks=$CT successRate=$SR reliability=$REL"
  assert_gt "$A completedTasks > 0" 0 "$CT"
  assert "$A successRate=1" "1" "$SR"
  assert "$A reliability=1" "1" "$REL"
done

# ────────────────────────────────────────────────────────────────
echo
echo "── Step 8: Synergy (cross-agent JOIN) ──"
R=$(curl -s "$BASE/api/pvm/synergy?agent1=dev-1-alpha&agent2=backend-1-alpha")
CC=$(echo "$R" | jq -r '.synergy.collaborationCount')
echo "  synergy: $R"
# dev-1 + backend-1 don't share taskIds in this E2E, so collab=0 is correct.
# Just assert the endpoint responds with a numeric field.
if [ -n "$CC" ] && [ "$CC" != "null" ]; then
  echo "  ✅ synergy endpoint returns numeric collaborationCount=$CC"
  PASS=$((PASS+1))
else
  echo "  ❌ synergy endpoint broken"
  FAIL=$((FAIL+1))
fi

# ────────────────────────────────────────────────────────────────
echo
echo "── Step 9: Compare ──"
R=$(curl -s "$BASE/api/pvm/compare?agents=dev-1-alpha,backend-1-alpha&taskType=python")
echo "  compare: $R" | head -c 400
echo
S=$(echo "$R" | jq -r '.success // false')
assert "compare endpoint responds" "true" "$S"

# ────────────────────────────────────────────────────────────────
echo
echo "── Step 10: Per-project ChronLog (W08 split) ──"
PROJ_DIRS=$(ls "$SERVER_DIR/data/projects/" 2>/dev/null)
echo "  project dirs: $PROJ_DIRS"
echo "$PROJ_DIRS" | grep -q "alpha" && { echo "  ✅ alpha dir exists"; PASS=$((PASS+1)); } || { echo "  ❌ alpha dir missing"; FAIL=$((FAIL+1)); }
echo "$PROJ_DIRS" | grep -q "beta"  && { echo "  ✅ beta dir exists";  PASS=$((PASS+1)); } || { echo "  ❌ beta dir missing";  FAIL=$((FAIL+1)); }

# Per-project log via /api/log?projectName=alpha
R=$(curl -s "$BASE/api/log?projectName=alpha&limit=200")
ALPHA_N=$(echo "$R" | jq -r '.events | length')
assert_gt "/api/log?projectName=alpha returns events" 0 "$ALPHA_N"

R=$(curl -s "$BASE/api/log?projectName=beta&limit=200")
BETA_N=$(echo "$R" | jq -r '.events | length')
assert_gt "/api/log?projectName=beta returns events" 0 "$BETA_N"

# ────────────────────────────────────────────────────────────────
echo
echo "═══════════════════════════════════════════════════════"
echo " RESULT: $PASS passed, $FAIL failed"
echo "═══════════════════════════════════════════════════════"
exit $FAIL
