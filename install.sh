#!/usr/bin/env bash
# =============================================================================
# ACT — Agent Coordination Toolkit
# Installer (macOS / Linux / WSL)
#
# Usage:
#   bash install.sh
#
# Windows: use install.ps1 instead.
# =============================================================================

set -euo pipefail

# ─── Colors ──────────────────────────────────────────────────────────────────

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
DIM='\033[2m'
RESET='\033[0m'

ok()   { echo -e "  ${GREEN}✓${RESET} $1"; }
info() { echo -e "  ${CYAN}→${RESET} $1"; }
warn() { echo -e "  ${YELLOW}⚠${RESET}  $1"; }
fail() { echo -e "  ${RED}✗${RESET} $1"; exit 1; }
dim()  { echo -e "  ${DIM}$1${RESET}"; }

# ─── Banner ──────────────────────────────────────────────────────────────────

echo ""
echo -e "${BOLD}╔══════════════════════════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}║          Agent Coordination Toolkit (ACT) — Installer        ║${RESET}"
echo -e "${BOLD}╚══════════════════════════════════════════════════════════════╝${RESET}"
echo ""

# ─── Resolve ACT root directory ──────────────────────────────────────────────

if [[ -n "${BASH_SOURCE[0]:-}" && "${BASH_SOURCE[0]}" != "bash" ]]; then
  ACT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
else
  ACT_ROOT="$(pwd)"
fi

if [[ ! -f "${ACT_ROOT}/CLAUDE.md" ]]; then
  fail "Cannot find ACT root directory. Please run this script from the ACT repo root."
fi

ok "ACT root: ${ACT_ROOT}"

# ─── Detect platform ─────────────────────────────────────────────────────────

PLATFORM="$(uname -s)"
ARCH="$(uname -m)"
ok "Platform: ${PLATFORM} ${ARCH}"

# ─── Prerequisites ───────────────────────────────────────────────────────────

echo ""
echo -e "${BOLD}Checking prerequisites...${RESET}"

# Node.js
if ! command -v node &>/dev/null; then
  fail "Node.js v18+ required. Install from https://nodejs.org or via nvm."
fi
NODE_VERSION=$(node --version | sed 's/v//' | cut -d. -f1)
if [[ "$NODE_VERSION" -lt 18 ]]; then
  fail "Node.js v18+ required. Found: $(node --version)"
fi
ok "Node.js $(node --version)"

# npm
if ! command -v npm &>/dev/null; then
  fail "npm required (ships with Node.js)."
fi
ok "npm $(npm --version)"

# Go
GO_BIN=""
if command -v go &>/dev/null; then
  GO_BIN="$(command -v go)"
elif [[ -x "/opt/homebrew/bin/go" ]]; then
  GO_BIN="/opt/homebrew/bin/go"
elif [[ -x "/usr/local/go/bin/go" ]]; then
  GO_BIN="/usr/local/go/bin/go"
fi
if [[ -z "${GO_BIN}" ]]; then
  fail "Go required. Install from https://go.dev/dl (v1.21+)."
fi
ok "Go $(${GO_BIN} version | awk '{print $3}')"

# curl (used for fetching things if needed; soft-required)
if command -v curl &>/dev/null; then
  ok "curl"
else
  warn "curl not found (most platforms ship it; only needed for some optional features)"
fi

# claude CLI (optional)
if command -v claude &>/dev/null; then
  ok "claude CLI $(claude --version 2>/dev/null | head -1 || echo '(found)')"
else
  warn "Claude Code CLI not found. ACT will use act-agent (default) for swarm. Install Claude Code if you want subscription-backed swarm: https://docs.claude.com/claude-code"
fi

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
cd "${ACT_ROOT}/act-agent/cli"
npm install --silent
ok "CLI ready"

# ─── Build the act-agent Go binary ───────────────────────────────────────────

echo ""
echo -e "${BOLD}Building act-agent Go binary...${RESET}"
cd "${ACT_ROOT}/act-agent"
"${GO_BIN}" build -o act-agent .
ok "act-agent built → ${ACT_ROOT}/act-agent/act-agent"

# ─── Create ~/.act directory ─────────────────────────────────────────────────

echo ""
echo -e "${BOLD}Setting up ~/.act/...${RESET}"
mkdir -p "${HOME}/.act/sessions"
ok "Created ~/.act/"

# Write ACT_ROOT to ~/.act/config.json (used by 'act' CLI to find server scripts)
node - <<NODEEOF
const fs = require('fs');
const path = require('path');
const file = path.join(process.env.HOME, '.act', 'config.json');
let config = {};
try { config = JSON.parse(fs.readFileSync(file, 'utf8')); } catch (_) {}
config.actRoot = "${ACT_ROOT}";
fs.writeFileSync(file, JSON.stringify(config, null, 2));
console.log("  ACT root saved → " + file);
NODEEOF
ok "ACT root saved to ~/.act/config.json"

# ─── Bootstrap ~/.act.json from template if missing ──────────────────────────

if [[ -f "${HOME}/.act.json" ]]; then
  ok "~/.act.json already exists — leaving it alone"
else
  if [[ -f "${ACT_ROOT}/.act.example.json" ]]; then
    cp "${ACT_ROOT}/.act.example.json" "${HOME}/.act.json"
    chmod 600 "${HOME}/.act.json"
    ok "~/.act.json created from template"
    warn "→ Edit ~/.act.json and add your API keys (Anthropic / Groq / OpenRouter)"
  else
    warn "~/.act.json template not found at .act.example.json — skipping"
  fi
fi

# ─── Bootstrap .env from template if missing ─────────────────────────────────

if [[ -f "${ACT_ROOT}/.env" ]]; then
  ok ".env already exists — leaving it alone"
else
  if [[ -f "${ACT_ROOT}/.env.example" ]]; then
    cp "${ACT_ROOT}/.env.example" "${ACT_ROOT}/.env"
    ok ".env created from template (most users won't need to edit it)"
  fi
fi

# ─── Symlink 'act-agent' into PATH ───────────────────────────────────────────
#
# Note: the binary is named 'act-agent' (NOT 'act') to avoid collision with
# nektos/act, the popular GitHub Actions local runner. If you previously had
# an 'act' symlink from an older install, this script removes it.
#
# Strategy: prefer ~/.local/bin (per-user, no sudo, modern XDG-aligned).
# Falls back to /opt/homebrew/bin (Apple Silicon Homebrew) or /usr/local/bin
# only if ~/.local/bin can't be created/written.

echo ""
echo -e "${BOLD}Installing 'act-agent' command...${RESET}"

USER_BIN="${HOME}/.local/bin"
SYMLINK_DIR=""

# 1) Try ~/.local/bin first — create if missing
mkdir -p "${USER_BIN}" 2>/dev/null || true
if [[ -d "${USER_BIN}" && -w "${USER_BIN}" ]]; then
  SYMLINK_DIR="${USER_BIN}"
fi

# 2) Fallbacks if ~/.local/bin isn't usable
if [[ -z "${SYMLINK_DIR}" ]]; then
  if [[ "${PLATFORM}" == "Darwin" ]]; then
    if [[ "${ARCH}" == "arm64" && -d "/opt/homebrew/bin" && -w "/opt/homebrew/bin" ]]; then
      SYMLINK_DIR="/opt/homebrew/bin"
    elif [[ -d "/usr/local/bin" && -w "/usr/local/bin" ]]; then
      SYMLINK_DIR="/usr/local/bin"
    fi
  elif [[ "${PLATFORM}" == "Linux" ]]; then
    if [[ -d "/usr/local/bin" && -w "/usr/local/bin" ]]; then
      SYMLINK_DIR="/usr/local/bin"
    fi
  fi
fi

if [[ -z "${SYMLINK_DIR}" ]]; then
  warn "Could not find a writable PATH directory to symlink into."
  warn "Add ${ACT_ROOT}/act-agent to your PATH manually:"
  dim "  export PATH=\"${ACT_ROOT}/act-agent:\$PATH\"   # add to ~/.zshrc or ~/.bashrc"
  ACT_BIN="${ACT_ROOT}/act-agent/act-agent"
else
  SYMLINK_TARGET="${SYMLINK_DIR}/act-agent"
  if [[ -L "${SYMLINK_TARGET}" || ! -e "${SYMLINK_TARGET}" ]]; then
    if ln -sf "${ACT_ROOT}/act-agent/act-agent" "${SYMLINK_TARGET}" 2>/dev/null; then
      ACT_BIN="${SYMLINK_TARGET}"
      ok "'act-agent' symlinked → ${ACT_BIN}"
    else
      warn "Permission denied symlinking to ${SYMLINK_DIR}. Try:"
      dim "  sudo ln -sf \"${ACT_ROOT}/act-agent/act-agent\" \"${SYMLINK_TARGET}\""
      ACT_BIN="${ACT_ROOT}/act-agent/act-agent"
    fi
  else
    warn "${SYMLINK_TARGET} already exists and is not a symlink. Skipping."
    ACT_BIN="${ACT_ROOT}/act-agent/act-agent"
  fi

  # If we installed to ~/.local/bin and it isn't on PATH, add it to the user's shell rc
  if [[ "${SYMLINK_DIR}" == "${USER_BIN}" ]]; then
    case ":${PATH}:" in
      *":${USER_BIN}:"*)
        : # already on PATH
        ;;
      *)
        RC=""
        if [[ "${SHELL}" == */zsh ]]; then
          RC="${HOME}/.zshrc"
        elif [[ "${SHELL}" == */bash ]]; then
          RC="${HOME}/.bashrc"
        fi
        if [[ -n "${RC}" ]]; then
          if ! grep -qs '\.local/bin' "${RC}"; then
            {
              echo ''
              echo '# Added by ACT installer'
              echo 'export PATH="$HOME/.local/bin:$PATH"'
            } >> "${RC}"
            warn "Added ~/.local/bin to PATH in ${RC}"
            dim "  Open a new shell or run: source ${RC}"
          fi
        else
          warn "Add ~/.local/bin to PATH manually in your shell rc:"
          dim "  export PATH=\"\$HOME/.local/bin:\$PATH\""
        fi
        ;;
    esac
  fi

  # Sweep legacy 'act' symlinks from any of the candidate install dirs
  for legacy_dir in "${USER_BIN}" "/opt/homebrew/bin" "/usr/local/bin"; do
    legacy="${legacy_dir}/act"
    if [[ -L "${legacy}" ]]; then
      legacy_target="$(readlink "${legacy}" 2>/dev/null || echo "")"
      if [[ "${legacy_target}" == *"act-agent"* ]]; then
        info "Removing legacy '${legacy}' (clashed with nektos/act GitHub Actions runner)"
        rm -f "${legacy}" 2>/dev/null || warn "Could not remove ${legacy} — sudo rm if you want it gone"
      fi
    fi
  done
fi

# ─── Optional: register Claude Code hooks ────────────────────────────────────

SESSION_HOOK="${ACT_ROOT}/hooks/act-session-start.sh"
STOP_HOOK="${ACT_ROOT}/hooks/act-stop-hook.sh"

if [[ -f "${SESSION_HOOK}" && -f "${STOP_HOOK}" ]]; then
  echo ""
  echo -e "${BOLD}Registering Claude Code hooks...${RESET}"
  chmod +x "${SESSION_HOOK}" "${STOP_HOOK}"
  CLAUDE_SETTINGS="${HOME}/.claude/settings.json"
  mkdir -p "${HOME}/.claude"
  node - <<NODEEOF
const fs = require('fs');
const file = "${CLAUDE_SETTINGS}";
let settings = {};
try { settings = JSON.parse(fs.readFileSync(file, 'utf8')); } catch (_) {}
settings.hooks = settings.hooks || {};
settings.hooks.SessionStart = [{ hooks: [{ type: 'command', command: "${SESSION_HOOK}", timeout: 5 }] }];
settings.hooks.Stop = [{ hooks: [{ type: 'command', command: "${STOP_HOOK}", timeout: 10 }] }];
fs.writeFileSync(file, JSON.stringify(settings, null, 2));
console.log("  Hooks written → " + file);
NODEEOF
  ok "Hooks registered (SessionStart + Stop)"
fi

# ─── Done ────────────────────────────────────────────────────────────────────

echo ""
echo -e "${BOLD}╔══════════════════════════════════════════════════════════════╗${RESET}"
echo -e "${BOLD}║                    ACT installed!                            ║${RESET}"
echo -e "${BOLD}╚══════════════════════════════════════════════════════════════╝${RESET}"
echo ""
echo -e "${BOLD}Next steps:${RESET}"
echo ""
echo -e "  1. Edit ${CYAN}~/.act.json${RESET} and add your API keys"
echo -e "     ${DIM}Get keys: Anthropic (https://console.anthropic.com), Groq (https://console.groq.com), OpenRouter (https://openrouter.ai)${RESET}"
echo ""
echo -e "  2. Launch ACT in any project directory:"
echo -e "     ${CYAN}cd ~/my-project && act-agent${RESET}"
echo ""
echo -e "     The ACT server auto-starts on first launch (port 8080)."
echo -e "     The TUI (NesTTY) opens with Planner / Observer / Assurance / QA running in one window."
echo ""
echo -e "${BOLD}Useful commands:${RESET}"
echo -e "  ${CYAN}act-agent${RESET}                          Launch the multi-agent TUI"
echo -e "  ${CYAN}act-agent --project <name>${RESET}         Launch into a specific project"
echo -e "  ${CYAN}act-agent status${RESET}                   Show project / system status"
echo -e "  ${CYAN}act-agent task complete <id>${RESET}       Mark a task complete (swarm-side)"
echo -e "  ${CYAN}act-agent --help${RESET}                   List all CLI subcommands"
echo ""
echo -e "${BOLD}Docs:${RESET} ${ACT_ROOT}/README.md and ${ACT_ROOT}/CLAUDE.md"
echo ""
