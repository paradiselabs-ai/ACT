# =============================================================================
# ACT — Agent Coordination Toolkit
# Installer (Windows / PowerShell 5.1+)
#
# Usage (from PowerShell):
#   cd path\to\act
#   .\install.ps1
#
# If you get "running scripts is disabled":
#   Set-ExecutionPolicy -Scope CurrentUser -ExecutionPolicy RemoteSigned
#
# WSL / Git Bash users: use install.sh instead.
# =============================================================================

#Requires -Version 5.1

$ErrorActionPreference = 'Stop'
$InformationPreference  = 'Continue'

# ── Helpers ─────────────────────────────────────────────────────────────────

function Write-Ok    ($msg) { Write-Host "  ✓ $msg" -ForegroundColor Green }
function Write-Info  ($msg) { Write-Host "  → $msg" -ForegroundColor Cyan }
function Write-Warn  ($msg) { Write-Host "  ⚠ $msg" -ForegroundColor Yellow }
function Write-Fail  ($msg) { Write-Host "  ✗ $msg" -ForegroundColor Red; exit 1 }
function Write-Dim   ($msg) { Write-Host "  $msg" -ForegroundColor DarkGray }
function Write-Bold  ($msg) { Write-Host "$msg" -ForegroundColor White }

function Test-Command($cmd) {
    return $null -ne (Get-Command $cmd -ErrorAction SilentlyContinue)
}

# ── Banner ──────────────────────────────────────────────────────────────────

Write-Host ""
Write-Bold "╔══════════════════════════════════════════════════════════════╗"
Write-Bold "║          Agent Coordination Toolkit (ACT) — Installer        ║"
Write-Bold "║                          Windows                              ║"
Write-Bold "╚══════════════════════════════════════════════════════════════╝"
Write-Host ""

# ── Resolve ACT root ─────────────────────────────────────────────────────────

$ActRoot = $PSScriptRoot
if ([string]::IsNullOrEmpty($ActRoot)) { $ActRoot = (Get-Location).Path }

if (-not (Test-Path (Join-Path $ActRoot "CLAUDE.md"))) {
    Write-Fail "Cannot find ACT root directory. Run this script from the ACT repo root."
}
Write-Ok "ACT root: $ActRoot"
$archStr = if ([System.Environment]::Is64BitOperatingSystem) { 'x64' } else { 'x86' }
Write-Ok "Platform: Windows ($archStr)"

# ── Prerequisites ────────────────────────────────────────────────────────────

Write-Host ""
Write-Bold "Checking prerequisites..."

# Node.js
if (-not (Test-Command "node")) {
    Write-Fail "Node.js v18+ required. Install from https://nodejs.org or via 'winget install OpenJS.NodeJS'"
}
$nodeVersion = (node --version) -replace '^v',''
$nodeMajor = [int]($nodeVersion.Split('.')[0])
if ($nodeMajor -lt 18) {
    Write-Fail "Node.js v18+ required. Found: v$nodeVersion"
}
Write-Ok "Node.js v$nodeVersion"

# npm
if (-not (Test-Command "npm")) { Write-Fail "npm required (ships with Node.js)." }
Write-Ok "npm $(npm --version)"

# Go
if (-not (Test-Command "go")) {
    Write-Fail "Go required. Install from https://go.dev/dl or via 'winget install GoLang.Go' (v1.21+)."
}
$goVersionStr = (go version) -split ' '
Write-Ok "Go $($goVersionStr[2])"

# claude CLI (optional)
if (Test-Command "claude") {
    Write-Ok "claude CLI (found)"
} else {
    Write-Warn "Claude Code CLI not found. ACT will use act-agent for swarm. To install Claude Code: see https://docs.claude.com/claude-code"
}

# ── Build server ─────────────────────────────────────────────────────────────

Write-Host ""
Write-Bold "Building ACT server..."
Push-Location (Join-Path $ActRoot "server")
try {
    Write-Info "Installing server dependencies..."
    npm install --silent | Out-Null
    Write-Info "Compiling TypeScript..."
    npm run build --silent | Out-Null
    Write-Ok "Server built"
} finally {
    Pop-Location
}

# ── Install CLI dependencies ─────────────────────────────────────────────────

Write-Host ""
Write-Bold "Installing CLI dependencies..."
Push-Location (Join-Path $ActRoot "act-agent\cli")
try {
    npm install --silent | Out-Null
    Write-Ok "CLI ready"
} finally {
    Pop-Location
}

# ── Build the act-agent Go binary ────────────────────────────────────────────

Write-Host ""
Write-Bold "Building act-agent Go binary..."
Push-Location (Join-Path $ActRoot "act-agent")
try {
    & go build -o "act-agent.exe" .
    Write-Ok "act-agent.exe built → $ActRoot\act-agent\act-agent.exe"
} finally {
    Pop-Location
}

# ── Create %USERPROFILE%\.act\ ───────────────────────────────────────────────

Write-Host ""
Write-Bold "Setting up $env:USERPROFILE\.act\ ..."
$ActDir       = Join-Path $env:USERPROFILE ".act"
$SessionsDir  = Join-Path $ActDir "sessions"
New-Item -ItemType Directory -Path $SessionsDir -Force | Out-Null
Write-Ok "Created $ActDir"

# Save ACT_ROOT to ~/.act/config.json
# (Read-modify-write to preserve any other fields from prior runs.)
$configFile = Join-Path $ActDir "config.json"
$config = @{}
if (Test-Path $configFile) {
    try {
        $existing = Get-Content $configFile -Raw | ConvertFrom-Json
        # ConvertFrom-Json on PS 5.1 returns PSCustomObject — convert to hashtable
        $existing.PSObject.Properties | ForEach-Object { $config[$_.Name] = $_.Value }
    } catch { $config = @{} }
}
$config["actRoot"] = $ActRoot
$config | ConvertTo-Json -Depth 10 | Set-Content $configFile -Encoding UTF8
Write-Ok "ACT root saved → $configFile"

# ── Bootstrap ~/.act.json from template if missing ───────────────────────────

$ActJsonTarget   = Join-Path $env:USERPROFILE ".act.json"
$ActJsonTemplate = Join-Path $ActRoot ".act.example.json"

if (Test-Path $ActJsonTarget) {
    Write-Ok "$ActJsonTarget already exists — leaving it alone"
} elseif (Test-Path $ActJsonTemplate) {
    Copy-Item $ActJsonTemplate $ActJsonTarget
    Write-Ok "$ActJsonTarget created from template"
    Write-Warn "→ Edit $ActJsonTarget and add your API keys (Anthropic / Groq / OpenRouter)"
} else {
    Write-Warn "Template .act.example.json not found — skipping ~/.act.json bootstrap"
}

# ── Bootstrap .env from template if missing ──────────────────────────────────

$EnvTarget   = Join-Path $ActRoot ".env"
$EnvTemplate = Join-Path $ActRoot ".env.example"
if (Test-Path $EnvTarget) {
    Write-Ok ".env already exists — leaving it alone"
} elseif (Test-Path $EnvTemplate) {
    Copy-Item $EnvTemplate $EnvTarget
    Write-Ok ".env created from template (most users won't need to edit it)"
}

# ── Install 'act-agent' command ──────────────────────────────────────────────
#
# Note: the binary is named 'act-agent' (NOT 'act') to avoid collision with
# nektos/act, the popular GitHub Actions local runner.

Write-Host ""
Write-Bold "Installing 'act-agent' command..."

# Strategy: copy act-agent.exe to %LOCALAPPDATA%\Programs\act-agent\ and add to user PATH
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\act-agent"
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null

$SourceBin = Join-Path $ActRoot "act-agent\act-agent.exe"
$TargetBin = Join-Path $InstallDir "act-agent.exe"

# Try a hardlink first (no admin needed, survives source rebuilds since binary
# is rebuilt in place). If that fails, fall back to a copy.
Remove-Item -Path $TargetBin -ErrorAction SilentlyContinue
$linked = $false
try {
    New-Item -ItemType HardLink -Path $TargetBin -Target $SourceBin -ErrorAction Stop | Out-Null
    $linked = $true
    Write-Ok "Hardlinked act-agent → $TargetBin"
} catch {
    Copy-Item $SourceBin $TargetBin -Force
    Write-Ok "Copied act-agent → $TargetBin"
    Write-Dim "(Note: hardlink failed — to update 'act-agent' after rebuilds, re-run install.ps1)"
}

# Clean up legacy 'act.exe' from previous installs (if it exists in this dir or the old install dir)
$LegacyTargets = @(
    (Join-Path $InstallDir "act.exe"),
    (Join-Path $env:LOCALAPPDATA "Programs\act\act.exe")
)
foreach ($legacy in $LegacyTargets) {
    if (Test-Path $legacy) {
        try {
            Remove-Item $legacy -Force
            Write-Info "Removed legacy '$legacy' (clashed with nektos/act GitHub Actions runner)"
        } catch {
            Write-Warn "Could not remove legacy '$legacy' — delete manually if you want it gone"
        }
    }
}

# Add InstallDir to user PATH if not already there
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($null -eq $userPath) { $userPath = "" }
$pathEntries = $userPath -split ';' | Where-Object { $_ }
if ($pathEntries -notcontains $InstallDir) {
    $newPath = if ($userPath) { "$userPath;$InstallDir" } else { $InstallDir }
    [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
    Write-Ok "Added $InstallDir to user PATH"
    Write-Warn "→ Restart your terminal for PATH changes to take effect"
} else {
    Write-Ok "$InstallDir already in user PATH"
}

# ── Done ─────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Bold "╔══════════════════════════════════════════════════════════════╗"
Write-Bold "║                    ACT installed!                            ║"
Write-Bold "╚══════════════════════════════════════════════════════════════╝"
Write-Host ""
Write-Bold "Next steps:"
Write-Host ""
Write-Host "  1. Edit " -NoNewline; Write-Host "$ActJsonTarget" -ForegroundColor Cyan -NoNewline; Write-Host " and add your API keys"
Write-Dim    "     Get keys: Anthropic (https://console.anthropic.com), Groq (https://console.groq.com), OpenRouter (https://openrouter.ai)"
Write-Host ""
Write-Host "  2. Restart your terminal (so PATH update takes effect)"
Write-Host ""
Write-Host "  3. Launch ACT in any project directory:"
Write-Host "     " -NoNewline; Write-Host "cd ~/my-project; act-agent" -ForegroundColor Cyan
Write-Host ""
Write-Host "     The ACT server auto-starts on first launch (port 8080)."
Write-Host "     The TUI (NesTTY) opens with Planner / Observer / Assurance / QA running in one window."
Write-Host ""
Write-Bold "Useful commands:"
Write-Host "  " -NoNewline; Write-Host "act-agent"                        -ForegroundColor Cyan -NoNewline; Write-Host "                       Launch the multi-agent TUI"
Write-Host "  " -NoNewline; Write-Host "act-agent --project <name>"       -ForegroundColor Cyan -NoNewline; Write-Host "      Launch into a specific project"
Write-Host "  " -NoNewline; Write-Host "act-agent status"                 -ForegroundColor Cyan -NoNewline; Write-Host "                Show project / system status"
Write-Host "  " -NoNewline; Write-Host "act-agent task complete <id>"     -ForegroundColor Cyan -NoNewline; Write-Host "    Mark a task complete (swarm-side)"
Write-Host "  " -NoNewline; Write-Host "act-agent --help"                 -ForegroundColor Cyan -NoNewline; Write-Host "                List all CLI subcommands"
Write-Host ""
Write-Bold "Docs:"
Write-Host "  $ActRoot\README.md"
Write-Host "  $ActRoot\CLAUDE.md"
Write-Host ""
Write-Bold "Notes for Windows:"
Write-Dim "  • Terminal: Windows Terminal or any modern alternative recommended."
Write-Dim "    Avoid the legacy Console Host (cmd.exe in default mode) — TUI rendering may glitch."
Write-Dim "  • Bubbletea + UTF-8: if you see boxy characters in the TUI, run 'chcp 65001' before 'act'."
Write-Dim "  • If 'act-agent' is not found after install, restart your shell (PATH update needs a fresh shell)."
Write-Host ""
