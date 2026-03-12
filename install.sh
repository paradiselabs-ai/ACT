#!/usr/bin/env bash
# =============================================================================
# ACT — Agent Coordination Toolkit
# Installer script
#
# Usage:
#   bash install.sh
#   or:
#   curl -fsSL https://raw.githubusercontent.com/your-org/act/main/install.sh | bash
# =============================================================================

set -euo pipefail

# ─── Colors ──────────────────────────────────────────────────────────────────

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
RESET='\033[0m'

ok()   { echo -e "  ${GREEN}✓${RESET} $1"; }
info() { echo -e "  ${CYAN}→${RESET} $1"; }
warn() { echo -e "  ${YELLOW}⚠${RESET}  $1"; }
fail() { echo -e "  ${RED}✗${RESET} $1"; exit 1; }

# ─── Banner ──────────────────────────────────────────────────────────────────

echo ""
echo -e "${BOLD}╔══════════════════════════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}║          Agent Coordination Toolkit (ACT) — Installer        ║${RESET}"
echo -e "${BOLD}╚══════════════════════════════════════════════════════════════╝${RESET}"
echo ""

# ─── Resolve ACT root directory ──────────────────────────────────────────────

# If run via curl | bash, BASH_SOURCE[0] won't help — use PWD fallback
if [[ -n "${BASH_SOURCE[0]:-}" && "${BASH_SOURCE[0]}" != "bash" ]]; then
  ACT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
else
  ACT_ROOT="$(pwd)"
fi

if [[ ! -f "${ACT_ROOT}/CLAUDE.md" ]]; then
  fail "Cannot find ACT root directory. Please run this script from the ACT repo root."
fi

ok "ACT root: ${ACT_ROOT}"

# ─── Prerequisites ───────────────────────────────────────────────────────────

echo ""
echo -e "${BOLD}Checking prerequisites...${RESET}"

# Node.js
if ! command -v node &>/dev/null; then
  fail "Node.js is required but not installed. Install from https://nodejs.org (v18+)"
fi
NODE_VERSION=$(node --version | sed 's/v//' | cut -d. -f1)
if [[ "$NODE_VERSION" -lt 18 ]]; then
  fail "Node.js v18+ required. Found: $(node --version)"
fi
ok "Node.js $(node --version)"

# npm
if ! command -v npm &>/dev/null; then
  fail "npm is required but not installed."
fi
ok "npm $(npm --version)"

# claude CLI
if ! command -v claude &>/dev/null; then
  fail "Claude Code CLI ('claude') is required but not found in PATH. Install from https://claude.ai/code"
fi
ok "claude CLI $(claude --version 2>/dev/null | head -1 || echo '(found)')"

# curl
if ! command -v curl &>/dev/null; then
  fail "curl is required but not installed."
fi
ok "curl"

# python3 (used by hook script)
if ! command -v python3 &>/dev/null; then
  fail "python3 is required by the hook script but not found in PATH."
fi
ok "python3 $(python3 --version 2>&1 | awk '{print $2}')"

# ─── Build server ────────────────────────────────────────────────────────────

echo ""
echo -e "${BOLD}Building ACT server...${RESET}"
cd "${ACT_ROOT}/server"
info "Installing server dependencies..."
npm install --silent
info "Compiling TypeScript..."
npm run build --silent
ok "Server built"

# ─── Build MCP bridge ────────────────────────────────────────────────────────

echo ""
echo -e "${BOLD}Building MCP bridge...${RESET}"
cd "${ACT_ROOT}/mcp-servers/act-mcp-bridge"
info "Installing bridge dependencies..."
npm install --silent
info "Compiling TypeScript..."
npm run build --silent
ok "MCP bridge built"

# ─── Install CLI dependencies ─────────────────────────────────────────────────

echo ""
echo -e "${BOLD}Installing CLI dependencies...${RESET}"
cd "${ACT_ROOT}/cli"
npm install --silent
ok "CLI ready"

# ─── Write MCP config for Claude Code CLI ────────────────────────────────────

echo ""
echo -e "${BOLD}Configuring Claude Code CLI MCP...${RESET}"

# Claude Code CLI stores user-scoped MCP servers in ~/.claude.json
# This is separate from Claude Desktop's config and is what the `claude` CLI reads.
CLAUDE_CODE_CONFIG="${HOME}/.claude.json"
BRIDGE_PATH="${ACT_ROOT}/mcp-servers/act-mcp-bridge/dist/index.js"

python3 - <<PYEOF
import json, os

config_file = """${CLAUDE_CODE_CONFIG}"""
bridge_path = """${BRIDGE_PATH}"""

# Load existing config or start fresh
if os.path.exists(config_file):
    try:
        with open(config_file) as f:
            config = json.load(f)
    except:
        config = {}
else:
    config = {}

if 'mcpServers' not in config:
    config['mcpServers'] = {}

config['mcpServers']['act'] = {
    'command': 'node',
    'args': [bridge_path],
    'env': {'ACT_SERVER_URL': 'http://localhost:8080'}
}

with open(config_file, 'w') as f:
    json.dump(config, f, indent=2)

print(f"  Written to: {config_file}")
PYEOF

ok "ACT MCP bridge registered in ~/.claude.json (Claude Code CLI)"

# ─── Register hooks (SessionStart + Stop) ────────────────────────────────────

echo ""
echo -e "${BOLD}Registering ACT hooks...${RESET}"

SESSION_HOOK="${ACT_ROOT}/hooks/act-session-start.sh"
STOP_HOOK="${ACT_ROOT}/hooks/act-stop-hook.sh"
chmod +x "${SESSION_HOOK}" "${STOP_HOOK}"

CLAUDE_SETTINGS="${HOME}/.claude/settings.json"
mkdir -p "${HOME}/.claude"

python3 - <<PYEOF
import json, os

settings_file  = """${CLAUDE_SETTINGS}"""
session_hook   = """${SESSION_HOOK}"""
stop_hook      = """${STOP_HOOK}"""

if os.path.exists(settings_file):
    try:
        with open(settings_file) as f:
            settings = json.load(f)
    except:
        settings = {}
else:
    settings = {}

if 'hooks' not in settings:
    settings['hooks'] = {}

# SessionStart: exports CLAUDE_SESSION_ID via CLAUDE_ENV_FILE so
# register_with_act can write a per-session identity file.
settings['hooks']['SessionStart'] = [
    {
        'hooks': [
            {
                'type': 'command',
                'command': session_hook,
                'timeout': 5
            }
        ]
    }
]

# Stop: autonomous coordination loop — checks inbox, tasks, proactive coordination.
# Uses session_id from stdin JSON to resolve per-instance agent identity.
# Non-ACT windows exit silently with zero overhead.
settings['hooks']['Stop'] = [
    {
        'hooks': [
            {
                'type': 'command',
                'command': stop_hook,
                'timeout': 10
            }
        ]
    }
]

with open(settings_file, 'w') as f:
    json.dump(settings, f, indent=2)

print(f"  Written to: {settings_file}")
PYEOF

ok "Hooks registered (SessionStart + Stop)"

# ─── Create ~/.act directory ──────────────────────────────────────────────────

mkdir -p "${HOME}/.act/sessions"
ok "Created ~/.act directory"

# Write ACT_ROOT to ~/.act/config.json so the 'act' CLI can find the server
python3 - <<PYEOF
import json, os
config_file = os.path.expanduser("~/.act/config.json")
try:
    with open(config_file) as f:
        config = json.load(f)
except:
    config = {}
config["actRoot"] = """${ACT_ROOT}"""
with open(config_file, "w") as f:
    json.dump(config, f, indent=2)
print(f"  ACT root saved → {config_file}")
PYEOF
ok "ACT root path saved to ~/.act/config.json"

# ─── Build + install 'act' CLI command globally ───────────────────────────────

echo ""
echo -e "${BOLD}Installing 'act' command...${RESET}"
cd "${ACT_ROOT}/cli"
info "Building CLI..."
npm run build --silent
info "Installing globally via npm..."
npm install -g --force . --silent
ACT_BIN="$(which act 2>/dev/null || echo 'not found — ensure npm global bin is in PATH')"
ok "'act' command installed → ${ACT_BIN}"

# ─── Done ────────────────────────────────────────────────────────────────────

echo ""
echo -e "${BOLD}╔══════════════════════════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}║                    ACT installed! 🎉                         ║${RESET}"
echo -e "${BOLD}╚══════════════════════════════════════════════════════════════╝${RESET}"
echo ""
echo -e "${BOLD}Getting started:${RESET}"
echo ""
echo "  1. Open a terminal in your project directory and run:"
echo ""
echo -e "     ${CYAN}act${RESET}"
echo ""
echo "  2. In each Claude Code window you want to coordinate:"
echo "     — Claude Code will register with ACT automatically via the MCP bridge"
echo "     — The Stop hook will start the autonomous coordination loop"
echo "     — No additional configuration needed"
echo ""
echo -e "${BOLD}Commands:${RESET}"
echo "  act                  Start server + open REPL"
echo "  act server start     Start server in background"
echo "  act server stop      Stop server"
echo "  act server status    Check server health"
echo ""
echo -e "  Server logs: ${CYAN}~/.act/server.log${RESET}"
echo ""
