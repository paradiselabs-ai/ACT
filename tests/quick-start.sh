#!/bin/bash
# ACT Quick Start — runs a complete pipeline test
#
# Usage:
#   bash tests/quick-start.sh              # mock agents (no API key needed)
#   bash tests/quick-start.sh --live       # real LLM agents (needs API key)
#
# Prerequisites:
#   npm install in server/, cli/, nestty/
#   For --live: ANTHROPIC_API_KEY or GROQ_API_KEY set, act-agent compiled

set -euo pipefail

ACT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${1:-mock}"
BASE="http://localhost:8080"

echo ""
echo "╔═══════════════════════════════════════╗"
echo "║     ACT Quick Start Test              ║"
echo "║     Mode: ${MODE}                         ║"
echo "╚═══════════════════════════════════════╝"
echo ""

# ─── Start server ────────────────────────────────────────────────────────────

echo "Starting ACT server..."
# Kill any existing server
lsof -ti:8080 2>/dev/null | xargs kill 2>/dev/null || true
sleep 1

# Clear log for clean state
> "${ACT_ROOT}/server/data/coordination-log.jsonl"

cd "${ACT_ROOT}/server"
npx tsx src/index.ts > /tmp/act-quickstart.log 2>&1 &
SERVER_PID=$!

# Wait for health
for i in $(seq 1 15); do
  if curl -s "${BASE}/health" > /dev/null 2>&1; then
    echo "  Server ready (${i}s)"
    break
  fi
  sleep 2
done

if ! curl -s "${BASE}/health" > /dev/null 2>&1; then
  echo "  ERROR: Server failed to start. Check /tmp/act-quickstart.log"
  exit 1
fi

# ─── Create project ─────────────────────────────────────────────────────────

echo ""
echo "Creating test project..."
curl -s -X POST "${BASE}/api/projects" \
  -H "Content-Type: application/json" \
  -d '{"name":"quickstart","workspace":"/tmp/quickstart","description":"ACT quick start test project"}' > /dev/null
echo "  Project: quickstart"

# ─── Run smoke tests ────────────────────────────────────────────────────────

echo ""
echo "Running smoke tests..."
bash "${ACT_ROOT}/tests/smoke-test.sh" 2>&1 | tail -3

# ─── Launch NesTTY ───────────────────────────────────────────────────────────

echo ""
if [[ "$MODE" == "--live" ]]; then
  echo "Launching NesTTY with LIVE agents..."
  echo "  (Press Ctrl+C or type /quit to stop)"
  echo ""
  cd "${ACT_ROOT}"
  npx tsx nestty/index.ts --project quickstart --server "${BASE}"
else
  echo "Launching NesTTY with MOCK agents (15s test)..."
  cd "${ACT_ROOT}"
  (
    sleep 8
    echo "Create a simple hello world app"
    sleep 5
    echo "/quit"
  ) | MOCK_AGENT=1 npx tsx nestty/index.ts --project quickstart --roles planner,observer,assurance,qa --server "${BASE}" 2>/dev/null

  echo ""
  echo "─── Final system state ───"
  echo ""
  curl -s "${BASE}/api/status" | python3 -c "
import sys, json
s = json.load(sys.stdin)
print(f'Tasks: {s[\"tasks\"][\"total\"]} ({dict(s[\"tasks\"][\"byStatus\"])})')
print(f'Agents: {s[\"agents\"][\"total\"]}')
print(f'Locks: {s[\"fileLocks\"]}')
print(f'Projects: {s[\"projects\"]}')
" 2>/dev/null || echo "(could not reach server)"
fi

# ─── Cleanup ─────────────────────────────────────────────────────────────────

echo ""
kill $SERVER_PID 2>/dev/null || true
echo "Server stopped. Quick start complete."
