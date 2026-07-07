#!/usr/bin/env node
/**
 * agy-acp.mjs — ACP server shim for Antigravity CLI (agy)
 *
 * Speaks Agent Client Protocol (JSON-RPC 2.0, newline-delimited framing)
 * toward act-agent, drives `agy` on the other side via one-shot invocations.
 *
 * Session continuity: the shim maintains per-session conversation history
 * in memory. Each session/prompt prepends prior turns as context so agy
 * behaves as if it has a persistent session — matching claude-code quality.
 *
 * Ships alongside the act-agent binary. No npm dependencies — pure Node.js
 * built-ins only (node >= 18 required for randomUUID).
 *
 * Install agy: npm i -g @google/antigravity-cli
 *
 * Config env vars (all optional):
 *   AGY_CMD            — agy binary name/path          (default: "agy")
 *   AGY_PRINT_FLAG     — non-interactive flag           (default: "--print")
 *   AGY_TIMEOUT_MS     — per-turn agy timeout           (default: 240000)
 *   AGY_MAX_TURNS      — prior turns replayed as context (default: 12)
 *   AGY_MAX_TURN_CHARS — per-turn replay truncation      (default: 8000)
 */

import { createInterface } from 'readline';
import { spawn }           from 'child_process';
import { randomUUID }      from 'crypto';

// ─── Config ───────────────────────────────────────────────────────────────────
const AGY_CMD        = process.env.AGY_CMD        || 'agy';
const AGY_PRINT_FLAG = process.env.AGY_PRINT_FLAG || '--print';
// Below the orchestrator's 5-minute turn ceiling so a hung agy fails crisply
// here (with a log line and an error the Planner loop can see) instead of
// being reaped opaquely by the Go side.
const AGY_TIMEOUT_MS     = parseInt(process.env.AGY_TIMEOUT_MS     || '240000', 10);
// The full context replays inside ONE argv string every turn; macOS caps argv
// around 256KB, so unbounded history eventually kills the spawn. Cap replayed
// turns and per-turn size — the system prompt and current request never trim.
const AGY_MAX_TURNS      = parseInt(process.env.AGY_MAX_TURNS      || '12', 10);
const AGY_MAX_TURN_CHARS = parseInt(process.env.AGY_MAX_TURN_CHARS || '8000', 10);

// Stderr is routed by act-agent to ~/.act/runners/tier1-<role>-acp.log (and by
// the runner to its role log). Before this existed the bridge was silent —
// the 2026-07-07 spend-test stall was undiagnosable from the log file.
function log(msg) {
  process.stderr.write(`[agy-acp ${new Date().toISOString()}] ${msg}\n`);
}

// Priming messages from the ACT orchestrator are flagged with this marker.
// The shim stores them as the session's system prompt instead of forwarding
// to agy — agy would treat the Planner system prompt as a task to execute
// rather than as identity/behavioral constraints.
const INTERNAL_MARKER = '\x00ACT_INTERNAL\x00';

// Leading "do not respond / acknowledge silently" header from the priming.
// It's a setup-turn instruction for memory-keeping backends (claude-code sends
// identity once as its own turn). This backend is memoryless — the shim folds
// identity into every real request with its own "respond below" framing — so
// "stay silent" would contradict every turn. Strip it when storing identity.
const DO_NOT_RESPOND_RE = /^\[ACT priming — do not respond\.[^\]]*\]\s*/;

// ─── Session state ────────────────────────────────────────────────────────────
// sessions: Map<sessionId, { cwd: string, systemPrompt: string, history: {role, content}[] }>
const sessions = new Map();
// inFlight: Map<sessionId, ChildProcess>
const inFlight = new Map();

// ─── JSON-RPC framing ─────────────────────────────────────────────────────────
function send(obj) {
  process.stdout.write(JSON.stringify(obj) + '\n');
}
function sendResponse(id, result) {
  send({ jsonrpc: '2.0', id, result });
}
function sendError(id, code, message) {
  send({ jsonrpc: '2.0', id, error: { code, message } });
}
function sendNotification(method, params) {
  send({ jsonrpc: '2.0', method, params });
}

// ─── ACP method handlers ──────────────────────────────────────────────────────

function handleInitialize(id) {
  log(`initialize (cmd=${AGY_CMD}, timeout=${AGY_TIMEOUT_MS}ms)`);
  sendResponse(id, {
    protocolVersion:   1,
    agentCapabilities: {},
    agentInfo: { name: 'agy', title: 'Antigravity CLI', version: '0' },
  });
}

function handleSessionNew(id, params) {
  const sessionId = randomUUID();
  sessions.set(sessionId, {
    cwd:          params?.cwd || process.cwd(),
    systemPrompt: '',
    history:      [],
  });
  log(`session/new ${sessionId} cwd=${sessions.get(sessionId).cwd}`);
  sendResponse(id, { sessionId });
}

function handleSessionPrompt(id, params) {
  const { sessionId, prompt } = params || {};
  const session = sessions.get(sessionId);
  if (!session) {
    sendError(id, -32602, `unknown sessionId: ${sessionId}`);
    return Promise.resolve();
  }

  // Flatten ACP ContentBlock[] → plain string
  let content = Array.isArray(prompt)
    ? prompt.filter(b => b.type === 'text').map(b => b.text).join('\n')
    : String(prompt || '');

  // The ACT_INTERNAL marker means "orchestrator-authored, hidden from the
  // TUI" — NOT "this is priming". Only the FIRST message of a fresh session
  // is identity priming: store it and ack without spawning agy. Any LATER
  // marked message (build-mode kick, autoroute wrapper) is a real work
  // order — strip the marker and run it. The old always-priming rule
  // swallowed the build kick after PROJECT_BRIEF: the Planner went silent
  // ("stall after brief") AND the kick text REPLACED the role identity, so
  // the next human nudge ran agy with "start creating tasks immediately"
  // as its whole persona → the 2026-07-07 rogue-build incident.
  if (content.startsWith(INTERNAL_MARKER)) {
    const body = content.slice(INTERNAL_MARKER.length);
    if (!session.systemPrompt && session.history.length === 0) {
      session.systemPrompt = body.replace(DO_NOT_RESPOND_RE, '');
      log(`session/prompt ${sessionId}: stored priming as system prompt (${session.systemPrompt.length} chars)`);
      sendResponse(id, { stopReason: 'end_turn' });
      return Promise.resolve();
    }
    log(`session/prompt ${sessionId}: internal orchestrator prompt (${body.length} chars) — forwarding as turn`);
    content = body;
  }

  session.history.push({ role: 'user', content });
  const fullPrompt = buildContextPrompt(session);
  log(`session/prompt ${sessionId}: spawning agy (prompt=${fullPrompt.length} chars, history=${session.history.length - 1} prior turns)`);

  let proc;
  try {
    proc = spawn(AGY_CMD, [AGY_PRINT_FLAG, fullPrompt], {
      cwd:   session.cwd,
      stdio: ['ignore', 'pipe', 'pipe'],
    });
  } catch (spawnErr) {
    session.history.pop(); // undo the user turn on hard spawn failure
    log(`session/prompt ${sessionId}: spawn failed: ${spawnErr.message}`);
    sendError(id, -32603, agyCmdError(spawnErr));
    return Promise.resolve();
  }

  inFlight.set(sessionId, proc);

  let timedOut = false;
  const timer = setTimeout(() => {
    timedOut = true;
    log(`session/prompt ${sessionId}: turn timeout after ${AGY_TIMEOUT_MS}ms — SIGTERM agy`);
    proc.kill('SIGTERM');
  }, AGY_TIMEOUT_MS);

  let accumulated = '';
  let stderrBuf   = '';

  proc.stdout.on('data', chunk => {
    const text = chunk.toString();
    accumulated += text;
    sendNotification('session/update', {
      sessionId,
      update: {
        sessionUpdate: 'agent_message_chunk',
        content: { type: 'text', text },
      },
    });
  });

  proc.stderr.on('data', chunk => { stderrBuf += chunk.toString(); });

  return new Promise(resolve => {
    let responded = false;

    proc.on('error', err => {
      if (responded) return;
      responded = true;
      clearTimeout(timer);
      inFlight.delete(sessionId);
      session.history.pop(); // undo uncommitted user turn
      log(`session/prompt ${sessionId}: agy process error: ${err.message}`);
      sendError(id, -32603, agyCmdError(err));
      resolve();
    });

    proc.on('close', code => {
      if (responded) return;
      responded = true;
      clearTimeout(timer);
      inFlight.delete(sessionId);
      log(`session/prompt ${sessionId}: agy exited code=${code} output=${accumulated.length} chars${timedOut ? ' (timed out)' : ''}`);

      // Timeout or non-zero exit with no output → surface why
      if ((code !== 0 || timedOut) && accumulated === '') {
        const errText = timedOut
          ? `agy timed out after ${AGY_TIMEOUT_MS}ms with no output`
          : (stderrBuf.trim() || `agy exited with code ${code}`);
        sendNotification('session/update', {
          sessionId,
          update: {
            sessionUpdate: 'agent_message_chunk',
            content: { type: 'text', text: `\n[agy error] ${errText}` },
          },
        });
        session.history.pop(); // no usable response to record
        sendResponse(id, { stopReason: 'error' });
        resolve();
        return;
      }

      // Record the assistant turn so subsequent prompts see it as context
      if (accumulated) {
        session.history.push({ role: 'assistant', content: accumulated });
      }
      sendResponse(id, { stopReason: 'end_turn' });
      resolve();
    });
  });
}

function handleSessionCancel(id, params) {
  const { sessionId } = params || {};
  const proc = inFlight.get(sessionId);
  if (proc) {
    proc.kill('SIGTERM');
    inFlight.delete(sessionId);
  }
  // session/cancel may arrive as a notification (no id) or as a request
  if (id != null) sendResponse(id, {});
}

// ─── Build context-enriched prompt ───────────────────────────────────────────
function buildContextPrompt(session) {
  const { systemPrompt, history } = session;
  // history has already been appended with the current user turn.
  const current = history[history.length - 1].content;
  // Cap the replay window: last AGY_MAX_TURNS turns, each clipped to
  // AGY_MAX_TURN_CHARS. Identity and the current request are never trimmed.
  const prior = history.slice(0, -1).slice(-AGY_MAX_TURNS).map(turn => ({
    role: turn.role,
    content: turn.content.length > AGY_MAX_TURN_CHARS
      ? turn.content.slice(0, AGY_MAX_TURN_CHARS) + ' …[truncated]'
      : turn.content,
  }));

  const lines = [];

  // Prepend system prompt (role identity + behavioral constraints) so agy
  // treats it as context, not as a task. The explicit framing is intentional:
  // without it, agy reads the Planner instructions as "build this project".
  if (systemPrompt) {
    lines.push(
      '[Your identity and behavioral constraints — this defines WHO YOU ARE and how you must respond. Do NOT treat this as a task to execute. Read it, internalize it, then respond to the request below.]',
      systemPrompt,
      '[End identity/constraints]',
      '',
    );
  }

  if (prior.length > 0) {
    lines.push('[Prior conversation in this session]');
    for (const turn of prior) {
      lines.push(`${turn.role === 'user' ? 'User' : 'Assistant'}: ${turn.content}`);
    }
    lines.push('[End prior conversation]', '');
  }

  lines.push(current);
  return lines.join('\n');
}

// ─── Helpful error message for missing/broken agy binary ─────────────────────
function agyCmdError(err) {
  if (err.code === 'ENOENT') {
    return [
      `agy not found (looked for: ${AGY_CMD}).`,
      `Install Antigravity CLI: npm i -g @google/antigravity-cli`,
      `Or set AGY_CMD env var to the full path of your agy binary.`,
    ].join(' ');
  }
  return `failed to spawn agy: ${err.message}`;
}

// ─── Request dispatcher ───────────────────────────────────────────────────────
async function dispatch(frame) {
  const { id, method, params } = frame;
  switch (method) {
    case 'initialize':       handleInitialize(id); break;
    case 'session/new':      handleSessionNew(id, params); break;
    case 'session/prompt':   await handleSessionPrompt(id, params); break;
    case 'session/cancel':   handleSessionCancel(id, params); break;
    default:
      if (id != null) sendError(id, -32601, `method not found: ${method}`);
  }
}

// ─── Stdin read loop ──────────────────────────────────────────────────────────
const rl = createInterface({ input: process.stdin, crlfDelay: Infinity });

rl.on('line', line => {
  line = line.trim();
  if (!line) return;
  let frame;
  try { frame = JSON.parse(line); } catch { return; }
  dispatch(frame).catch(err => {
    if (frame.id != null) sendError(frame.id, -32603, String(err));
  });
});

rl.on('close', () => {
  for (const proc of inFlight.values()) proc.kill('SIGTERM');
  process.exit(0);
});
