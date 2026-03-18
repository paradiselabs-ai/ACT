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

# claude CLI (optional — ACT supports multiple agent shells)
if command -v claude &>/dev/null; then
  ok "claude CLI $(claude --version 2>/dev/null | head -1 || echo '(found)')"
else
  warn "Claude Code CLI not found. ACT will use act-agent (OpenCode fork) or other configured agent shells."
fi

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

# ─── Install CLI dependencies ─────────────────────────────────────────────────

echo ""
echo -e "${BOLD}Installing CLI dependencies...${RESET}"
cd "${ACT_ROOT}/cli"
npm install --silent
ok "CLI ready"

# ─── Install NesTTY dependencies ──────────────────────────────────────────────

echo ""
echo -e "${BOLD}Installing NesTTY dependencies...${RESET}"
cd "${ACT_ROOT}/nestty"
npm install --silent
ok "NesTTY ready"

# ─── Register hooks (SessionStart + Stop) ────────────────────────────────────

echo ""
echo -e "${BOLD}Checking for ACT hooks...${RESET}"

SESSION_HOOK="${ACT_ROOT}/hooks/act-session-start.sh"
STOP_HOOK="${ACT_ROOT}/hooks/act-stop-hook.sh"

if [[ ! -f "${SESSION_HOOK}" || ! -f "${STOP_HOOK}" ]]; then
  info "Hook scripts not found. ACT uses NesTTY orchestrator for coordination."
  info "Skipping hook registration."
else
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
fi

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
echo "  1. Start the ACT server:"
echo -e "     ${CYAN}cd server && npm run dev${RESET}"
echo ""
echo "  2. Open REPL and create a project:"
echo -e "     ${CYAN}act${RESET}"
echo ""
echo "  3. Launch NesTTY (multi-agent coordination):"
echo -e "     ${CYAN}nestty${RESET}                      (from REPL, all 4 Tier 1 roles)"
echo -e "     ${CYAN}nestty --mock${RESET}               (mock agents for testing)"
echo ""
echo "  4. Configure providers in .opencode.json (see .opencode.example.json)"
echo ""
echo -e "${BOLD}Commands:${RESET}"
echo "  act                          REPL (interactive)"
echo "  act context <id> --project   Agent context (headless)"
echo "  act status                   System status"
echo "  act graph task/unverified/conflicts  Observability"
echo ""
