#!/usr/bin/env node
/**
 * ACT Agent Runner
 *
 * Wraps the `claude` CLI in a coordination loop:
 *   1. register_with_act
 *   2. get_task  →  run claude -p "<task prompt>"  →  report_task_complete
 *   3. get_messages  →  if messages, process with claude  →  send_message reply
 *   4. sleep POLL_INTERVAL_MS, repeat
 *
 * Usage:
 *   node act-runner.mjs --agent-id myagent --name "MyAgent" --capabilities typescript,react
 *   node act-runner.mjs --agent-id myagent --name "MyAgent" --max-iterations 20
 *
 * Environment variables:
 *   ACT_SERVER_URL   Override ACT server (default: http://localhost:8080)
 *   CLAUDE_PATH      Path to `claude` binary (default: claude, must be in PATH)
 *   GEMINI_PATH      Path to `gemini` binary (default: gemini, must be in PATH)
 *   AGY_PATH         Path to `agy` binary (default: agy, must be in PATH)
 *   POLL_INTERVAL_MS Milliseconds between polls (default: 5000)
 *   MAX_ITERATIONS   Max loops before exiting (default: 100, 0 = unlimited)
 *   TASK_TIMEOUT_MS  Max ms for a single claude invocation (default: 120000)
 */

import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { parseArgs } from 'node:util';
import { existsSync } from 'node:fs';

const execFileAsync = promisify(execFile);

// ─── Config ──────────────────────────────────────────────────────────────────

const ACT_SERVER_URL = (process.env.ACT_SERVER_URL || 'http://localhost:8080').replace(/\/$/, '');
const ACT_PROJECT    = process.env.ACT_PROJECT || '';
const CLAUDE_PATH    = process.env.CLAUDE_PATH    || 'claude';
const GEMINI_PATH    = process.env.GEMINI_PATH    || 'gemini';
const AGY_PATH       = process.env.AGY_PATH       || 'agy';
const DEVIN_PATH     = process.env.DEVIN_PATH     || 'devin';
const AGENT_CLI      = process.env.ACTOR_CLI || process.env.AGENT_CLI || './act-agent';
const POLL_INTERVAL  = parseInt(process.env.POLL_INTERVAL_MS || '5000',  10);
const TASK_TIMEOUT   = parseInt(process.env.TASK_TIMEOUT_MS  || '120000', 10);

let maxIterations = parseInt(process.env.MAX_ITERATIONS || '100', 10);

// ─── CLI args ─────────────────────────────────────────────────────────────────

const { values: args } = parseArgs({
  options: {
    'agent-id':       { type: 'string' },
    'name':           { type: 'string' },
    'role':           { type: 'string' },
    'backend':        { type: 'string' },
    'capabilities':   { type: 'string' },  // comma-separated
    'max-iterations': { type: 'string' },
    'poll-interval':  { type: 'string' },
    'help':           { type: 'boolean', short: 'h' },
  },
  allowPositionals: false,
  strict: false,
});

if (args.help) {
  console.log(`
ACT Agent Runner

Usage:
  node act-runner.mjs --agent-id <id> --name <name> [options]

Required:
  --agent-id <id>          Unique agent identifier (e.g. claude_code_1)
  --name <name>            Human-readable agent name (e.g. "Claude Code 1")

Optional:
  --role <role>           Agent specialization (e.g. frontend-dev, backend-dev, developer)
  --backend <name>         Agent execution backend: act-agent (default), claude-code, gemini, antigravity, or devin
  --capabilities <list>    Comma-separated capabilities (e.g. typescript,react,testing)
  --max-iterations <n>     Max coordination loops before exit (default: 100, 0 = unlimited)
  --poll-interval <ms>     Milliseconds between polls (default: 5000)
  --help, -h               Show this help

Environment variables:
  ACT_SERVER_URL           ACT server URL (default: http://localhost:8080)
  ACT_BACKEND              Default backend if --backend not passed (default: act-agent)
  ACTOR_CLI / AGENT_CLI    Agent CLI binary (fallback: ./act-agent/act-agent)
  CLAUDE_PATH              Path to claude binary (default: claude)
  GEMINI_PATH              Path to gemini binary (default: gemini)
  AGY_PATH                 Path to agy binary (default: agy)
  DEVIN_PATH               Path to devin binary (default: devin)
  POLL_INTERVAL_MS         Polling interval in ms
  MAX_ITERATIONS           Max loops
  TASK_TIMEOUT_MS          Timeout for a single claude invocation (default: 120000)
`);
  process.exit(0);
}

const AGENT_ID     = args['agent-id'];
const AGENT_NAME   = args['name'] || AGENT_ID;
const AGENT_ROLE   = args['role'] || process.env.AGENT_ROLE || undefined;
const BACKEND      = args['backend'] || process.env.ACT_BACKEND || 'act-agent';
const CAPABILITIES = (args['capabilities'] || '').split(',').map(s => s.trim()).filter(Boolean);
const liveProcesses = new Map(); // agentId -> { pid, startedAt, taskId }

if (!AGENT_ID) {
  console.error('Error: --agent-id is required');
  process.exit(1);
}

// Least-privilege gate, mirroring runner/swarm_roles.go::BackendAllowedForRole.
// agy has no read-only/plan mode (--sandbox restricts the terminal only), so an
// antigravity researcher would run with full write privilege. The Go spawner
// rejects this pair before spawn; this catches hand-rolled invocations too.
if (AGENT_ROLE === 'researcher' && BACKEND === 'antigravity') {
  console.error('Error: backend "antigravity" is not allowed for the researcher role — agy has no read-only/plan mode.');
  process.exit(1);
}
// Same gate for devin: its one-shot mode has no read-only/plan mode and no
// tool-restriction flag (--permission-mode auto still leaves write tools
// available), so a devin researcher would run with full write privilege.
if (AGENT_ROLE === 'researcher' && BACKEND === 'devin') {
  console.error('Error: backend "devin" is not allowed for the researcher role — devin has no read-only/plan mode.');
  process.exit(1);
}

if (args['max-iterations']) maxIterations = parseInt(args['max-iterations'], 10);

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

async function request(method, path, body) {
  const url = `${ACT_SERVER_URL}${path}`;
  const opts = {
    method,
    headers: { 'Content-Type': 'application/json' },
  };
  if (body) opts.body = JSON.stringify(body);

  const res = await fetch(url, opts);
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(`HTTP ${res.status} ${method} ${path}: ${text}`);
  }
  return res.json();
}

const get  = (path, params) => {
  const qs = params ? '?' + new URLSearchParams(params).toString() : '';
  return request('GET', path + qs);
};
const post = (path, body) => request('POST', path, body);

// ─── ACT API calls ────────────────────────────────────────────────────────────

async function register() {
  if (!ACT_PROJECT) {
    throw new Error('ACT_PROJECT env var is required to register a swarm runner (per-project isolation)');
  }
  const body = {
    agentId: AGENT_ID,
    name: AGENT_NAME,
    projectName: ACT_PROJECT,
    role: AGENT_ROLE,
    capabilities: CAPABILITIES,
  };
  try {
    await post('/api/agents/register', body);
  } catch (err) {
    // 409 = stale registration from a previous run that never cleaned up.
    // The server's in-memory registry survives runner restarts. Self-heal:
    // DELETE the stale entry and retry once. If the second attempt also
    // fails, propagate the error.
    if (String(err.message).includes('HTTP 409')) {
      log(`Stale registration for "${AGENT_ID}" — deregistering and retrying`);
      try {
        await request('DELETE', `/api/agents/${encodeURIComponent(AGENT_ID)}`);
      } catch (delErr) {
        // Ignore — the retry will surface any remaining problem.
      }
      await post('/api/agents/register', body);
    } else {
      throw err;
    }
  }
  log(`Registered as "${AGENT_NAME}"${AGENT_ROLE ? ` (role: ${AGENT_ROLE})` : ''} [${CAPABILITIES.join(', ') || 'no capabilities listed'}]`);
}

async function getTask() {
  // Scope to current project so this runner only pulls tasks the active
  // TUI session dispatched — not stale tasks from prior projects that
  // happened to use the same agent ID (dev-1, backend-1, etc. are shared).
  const params = { agent_id: AGENT_ID };
  if (ACT_PROJECT) params.project = ACT_PROJECT;
  const data = await get('/api/tasks/assigned', params);
  return data.task || null;
}

async function reportProgress(taskId, progress, message) {
  await post(`/api/tasks/${taskId}/progress`, {
    agentId: AGENT_ID,
    progress,
    status: 'in_progress',
    message,
  });
}

async function reportComplete(taskId, success, result) {
  // Always mark the task complete first so it has a result body.
  await post(`/api/tasks/${taskId}/complete`, {
    agentId: AGENT_ID,
    success,
    result,
  });
  // Then, on success, route through the validation pipeline so Assurance
  // can score against @success_criteria. Without this step, completed work
  // is accepted on the swarm agent's word alone and Assurance/QA never run.
  // Failures skip validation — there's nothing valid to score.
  if (success) {
    try {
      await post(`/api/tasks/${taskId}/submit-for-validation`, {
        agentId: AGENT_ID,
      });
    } catch (err) {
      // Non-fatal: the task is still marked complete, validation just won't
      // run for this one. Log and move on.
      log(`  submit-for-validation failed for ${taskId}: ${err.message}`);
    }
  }
}

async function getMessages(since) {
  const params = { limit: 20 };
  if (since) params.since = since;
  if (ACT_PROJECT) params.project = ACT_PROJECT;
  const data = await get(`/api/agents/${encodeURIComponent(AGENT_ID)}/messages`, params);
  return data.messages || [];
}

async function sendMessage(message) {
  await post('/api/messages', { sender: AGENT_ID, projectName: ACT_PROJECT, message });
}

async function sendSessionEnded(taskId, code) {
  await post('/api/messages', {
    sender: AGENT_ID,
    projectName: ACT_PROJECT,
    message: `status: session ended for task ${taskId} (exit code: ${code})`
  });
}

// ─── Agent CLI invocation ─────────────────────────────────────────────────────

async function runAgent(prompt) {
  if (BACKEND === 'claude-code') {
    return runAgentClaudeCode(prompt);
  }
  if (BACKEND === 'gemini') {
    return runAgentGemini(prompt);
  }
  if (BACKEND === 'antigravity') {
    return runAgentAntigravity(prompt);
  }
  if (BACKEND === 'devin') {
    return runAgentDevin(prompt);
  }
  return runAgentActAgent(prompt);
}

async function runAgentActAgent(prompt) {
  log(`  Invoking agent CLI (${AGENT_CLI})...`);
  try {
    const { stdout, stderr } = await execFileAsync(
      AGENT_CLI,
      ['--agent', AGENT_ID, ...(AGENT_ROLE ? ['--role', AGENT_ROLE] : []), '--prompt', prompt],
      { timeout: TASK_TIMEOUT, maxBuffer: 10 * 1024 * 1024 }
    );
    if (stderr) log(`  [agent stderr] ${stderr.trim()}`);

    const raw = stdout.trim();
    try {
      const parsed = JSON.parse(raw);
      const status = parsed?.status;
      const result = typeof parsed?.result === 'string' ? parsed.result : raw;
      return {
        success: status === 'completed',
        output: result,
        error: status && status !== 'completed' ? `agent status: ${status}` : undefined,
        code: 0
      };
    } catch {
      return { success: true, output: raw, code: 0 };
    }
  } catch (err) {
    const output = err.stdout?.trim() || '';
    const errMsg = err.stderr?.trim() || err.message;
    const code = Number.isInteger(err.code) ? err.code : 1;
    if (output) {
      try {
        const parsed = JSON.parse(output);
        const status = parsed?.status;
        const result = typeof parsed?.result === 'string' ? parsed.result : output;
        return {
          success: status === 'completed',
          output: result,
          error: status && status !== 'completed' ? `agent status: ${status}` : errMsg,
          code
        };
      } catch {
        // Backward-compatible fallback for plain text stdout from claude --print
      }
    }
    return { success: false, output, error: errMsg, code };
  }
}

// Least-privilege tool restriction per role for the claude-code backend.
// Mirrors ResearcherTools on the act-agent backend: the researcher's prompt
// says "analysis, not code", so its tools must not mutate files or run shell
// commands. Names claude doesn't know are ignored, so this is version-safe.
const ROLE_DISALLOWED_TOOLS = {
  researcher: ['Bash', 'Edit', 'MultiEdit', 'Write', 'NotebookEdit'],
};

function claudeToolRestrictionArgs() {
  const denied = ROLE_DISALLOWED_TOOLS[AGENT_ROLE];
  return denied ? ['--disallowedTools', denied.join(',')] : [];
}

async function runAgentClaudeCode(prompt, attempt = 1) {
  const restrictionArgs = claudeToolRestrictionArgs();
  log(`  [claude invoke] path=${CLAUDE_PATH} attempt=${attempt} prompt_bytes=${prompt.length}${restrictionArgs.length ? ` disallowed=${restrictionArgs[1]}` : ''}`);
  try {
    // `input: ''` closes stdin with EOF immediately. Without this Node leaves
    // stdin as a piped-but-empty stream and claude's "--print" mode waits 3s
    // for user input, prints a warning to stderr, then hangs until TASK_TIMEOUT
    // kills it. Passing empty input cleanly tells claude "no piped input".
    const { stdout, stderr } = await execFileAsync(
      CLAUDE_PATH,
      // --setting-sources '' clean-rooms the one-shot: without it claude loads
      // the OPERATOR's user/project/local config (global CLAUDE.md, persona
      // plugins, hooks) into every swarm task — same leak class as the Tier-1
      // ACP settingSources:[] fix. Verified live: flag accepts empty string.
      ['--print', '--dangerously-skip-permissions', '--setting-sources', '', ...restrictionArgs, prompt],
      { timeout: TASK_TIMEOUT, maxBuffer: 10 * 1024 * 1024, input: '' }
    );
    if (stderr) log(`  [claude stderr] ${stderr.trim().split('\n')[0]}`);
    log(`  [claude result] success=true code=0 out_bytes=${stdout.length}`);
    return {
      success: true,
      output: stdout.trim(),
      code: 0
    };
  } catch (err) {
    const output = err.stdout?.trim() || '';
    const errMsg = err.stderr?.trim() || err.message;
    const code = Number.isInteger(err.code) ? err.code : 1;
    const firstErrLine = (errMsg || '').split('\n')[0];
    log(`  [claude result] success=false code=${code} stderr="${firstErrLine}" out_bytes=${output.length}`);

    // One-shot retry on transient failure: no output AND (timeout OR non-zero code).
    // Transient here means the process died without producing a response — most
    // commonly a stdin-wait timeout, a 5xx from the API, or a dropped connection.
    // A second attempt with a fresh process + closed stdin almost always resolves it.
    if (attempt === 1 && !output) {
      log(`  [claude retry] first attempt had no output; retrying once`);
      return runAgentClaudeCode(prompt, 2);
    }

    return {
      success: false,
      output,
      error: errMsg,
      code
    };
  }
}

// Least-privilege for the gemini backend: no per-tool deny flag in headless
// mode, but --approval-mode plan is a native read-only mode — exactly the
// researcher contract. Everything else runs --yolo (gemini's equivalent of
// claude's --dangerously-skip-permissions; headless auto-rejects tool calls
// without it).
function geminiApprovalArgs() {
  return AGENT_ROLE === 'researcher' ? ['--approval-mode', 'plan'] : ['--yolo'];
}

async function runAgentGemini(prompt, attempt = 1) {
  const approvalArgs = geminiApprovalArgs();
  log(`  [gemini invoke] path=${GEMINI_PATH} attempt=${attempt} prompt_bytes=${prompt.length} approval=${approvalArgs.join(' ')}`);
  try {
    // --skip-trust: gemini 0.50+ hard-refuses headless runs in untrusted
    // folders (verified live); swarm agents run in arbitrary project dirs.
    const { stdout, stderr } = await execFileAsync(
      GEMINI_PATH,
      ['--skip-trust', ...approvalArgs, '-p', prompt],
      { timeout: TASK_TIMEOUT, maxBuffer: 10 * 1024 * 1024, input: '' }
    );
    if (stderr) log(`  [gemini stderr] ${stderr.trim().split('\n')[0]}`);
    log(`  [gemini result] success=true code=0 out_bytes=${stdout.length}`);
    return {
      success: true,
      output: stdout.trim(),
      code: 0
    };
  } catch (err) {
    const output = err.stdout?.trim() || '';
    const errMsg = err.stderr?.trim() || err.message;
    const code = Number.isInteger(err.code) ? err.code : 1;
    const firstErrLine = (errMsg || '').split('\n')[0];
    log(`  [gemini result] success=false code=${code} stderr="${firstErrLine}" out_bytes=${output.length}`);

    // Same one-shot retry contract as claude-code: process died with no
    // output → transient (API 5xx, dropped connection); retry once fresh.
    if (attempt === 1 && !output) {
      log(`  [gemini retry] first attempt had no output; retrying once`);
      return runAgentGemini(prompt, 2);
    }

    return {
      success: false,
      output,
      error: errMsg,
      code
    };
  }
}

// Antigravity (agy) backend: a plain one-shot, same shape as claude-code.
// `agy --print <prompt>` runs a single prompt non-interactively and exits; no
// --continue/--conversation, because the swarm is stateless per task (the Runner
// rebuilds full context into every prompt and the ACT server is the memory).
// No least-privilege flag exists — agy's only restriction is --sandbox, which
// limits the terminal, not file writes — so the researcher role is rejected up
// front (see the startup gate) rather than run with more privilege than it has
// on every other backend.
async function runAgentAntigravity(prompt, attempt = 1) {
  log(`  [agy invoke] path=${AGY_PATH} attempt=${attempt} prompt_bytes=${prompt.length}`);
  try {
    // `input: ''` closes stdin with EOF immediately — same reason as claude:
    // a piped-but-empty stdin makes print mode wait for input it never gets.
    const { stdout, stderr } = await execFileAsync(
      AGY_PATH,
      // Flag order matters: agy's --print consumes the NEXT arg as its prompt
      // value. Boolean flags must come first or --print swallows them as the
      // prompt and agy answers the flag name instead of the task (live bug,
      // 2026-08-08 LinkDock e2e: every swarm task returned an explanation of
      // --dangerously-skip-permissions and Assurance 0/100'd all of them).
      ['--dangerously-skip-permissions', '--print', prompt],
      { timeout: TASK_TIMEOUT, maxBuffer: 10 * 1024 * 1024, input: '' }
    );
    if (stderr) log(`  [agy stderr] ${stderr.trim().split('\n')[0]}`);
    log(`  [agy result] success=true code=0 out_bytes=${stdout.length}`);
    return {
      success: true,
      output: stdout.trim(),
      code: 0
    };
  } catch (err) {
    const output = err.stdout?.trim() || '';
    const errMsg = err.stderr?.trim() || err.message;
    const code = Number.isInteger(err.code) ? err.code : 1;
    const firstErrLine = (errMsg || '').split('\n')[0];
    log(`  [agy result] success=false code=${code} stderr="${firstErrLine}" out_bytes=${output.length}`);

    // Same one-shot retry contract as claude-code/gemini: process died with no
    // output → transient (API 5xx, dropped connection); retry once fresh.
    if (attempt === 1 && !output) {
      log(`  [agy retry] first attempt had no output; retrying once`);
      return runAgentAntigravity(prompt, 2);
    }

    return {
      success: false,
      output,
      error: errMsg,
      code
    };
  }
}

// `devin -p <prompt>` runs a single prompt non-interactively and exits; no
// --continue/--resume, because the swarm is stateless per task (the Runner
// rebuilds full context into every prompt and the ACT server is the memory).
// Verified against devin 3000.1.27: --permission-mode values are
// auto|accept-edits|smart|dangerous (the docs' "normal"/"autonomous" do not
// exist), and --respect-workspace-trust already defaults to false in print
// mode — passed explicitly so the behavior survives a default flip.
// devin has no --setting-sources equivalent, so the spawned agent still loads
// the operator's own rules/skills; nothing to clean-room with.
// No least-privilege flag exists (no --allowed-tools/--disallowed-tools; the
// read-only --agent-type review is `devin acp`-only) — so the researcher role
// is rejected up front (see the startup gate).
async function runAgentDevin(prompt, attempt = 1) {
  log(`  [devin invoke] path=${DEVIN_PATH} attempt=${attempt} prompt_bytes=${prompt.length}`);
  try {
    // `input: ''` closes stdin with EOF immediately — same reason as claude:
    // a piped-but-empty stdin makes print mode wait for input it never gets.
    const { stdout, stderr } = await execFileAsync(
      DEVIN_PATH,
      // Flag order matters (agy precedent): value/boolean flags first, -p last
      // so its optional inline value is the prompt and nothing else.
      ['--permission-mode', 'dangerous', '--respect-workspace-trust', 'false', '-p', prompt],
      { timeout: TASK_TIMEOUT, maxBuffer: 10 * 1024 * 1024, input: '' }
    );
    if (stderr) log(`  [devin stderr] ${stderr.trim().split('\n')[0]}`);
    log(`  [devin result] success=true code=0 out_bytes=${stdout.length}`);
    return {
      success: true,
      output: stdout.trim(),
      code: 0
    };
  } catch (err) {
    const output = err.stdout?.trim() || '';
    const errMsg = err.stderr?.trim() || err.message;
    const code = Number.isInteger(err.code) ? err.code : 1;
    const firstErrLine = (errMsg || '').split('\n')[0];
    log(`  [devin result] success=false code=${code} stderr="${firstErrLine}" out_bytes=${output.length}`);

    // Same one-shot retry contract as claude-code/gemini/agy: process died with
    // no output → transient (API 5xx, dropped connection); retry once fresh.
    if (attempt === 1 && !output) {
      log(`  [devin retry] first attempt had no output; retrying once`);
      return runAgentDevin(prompt, 2);
    }

    return {
      success: false,
      output,
      error: errMsg,
      code
    };
  }
}

// ─── Parallel awareness ───────────────────────────────────────────────────────

/**
 * Fetch all active agents and their in-progress tasks.
 * Returns a formatted string describing the parallel work landscape.
 */
async function fetchParallelContext() {
  try {
    const params = ACT_PROJECT ? { project: ACT_PROJECT } : undefined;
    const [agentsData, tasksData] = await Promise.all([
      get('/api/agents', params),
      get('/api/tasks', params),
    ]);

    const agents = (agentsData.agents || []).filter(a => a.id !== AGENT_ID);
    const tasks  = (tasksData.tasks  || []).filter(t =>
      t.status === 'in_progress' || t.status === 'assigned'
    );

    if (agents.length === 0) return null;

    // Only worth a coordination model call if a PEER actually has work in flight.
    // Every runner registers at spawn, so agent presence alone is not signal.
    const peerIds = new Set(agents.map(a => a.id));
    if (!tasks.some(t => peerIds.has(t.assignedAgent))) {
      log(`  [coord skip] no peer tasks in flight`);
      return null;
    }

    const agentLines = agents.map(a => {
      const agentTasks = tasks.filter(t => t.assignedAgent === a.id);
      const taskSummary = agentTasks.length > 0
        ? agentTasks.map(t => `"${t.title || t.description?.substring(0, 80)}"`).join(', ')
        : 'no active tasks';
      return `  • ${a.id} (${a.name || a.id}): ${taskSummary}`;
    });

    return agentLines.join('\n');
  } catch {
    return null; // non-fatal
  }
}

/**
 * Ask the agent CLI to identify interface points with parallel agents and send
 * proactive coordination messages where needed. Fire-and-forget.
 */
async function proactiveCoordination(task, parallelContext) {
  if (!parallelContext) return;

  const prompt = [
    `You are ${AGENT_NAME}, an AI agent in the ACT coordination system.`,
    ``,
    `You are about to start this task:`,
    `Title: ${task.title || '(untitled)'}`,
    `Description: ${task.description || '(no description)'}`,
    ``,
    `These other agents are working in parallel right now:`,
    parallelContext,
    ``,
    `Your job: identify which agents are building something that will directly interface`,
    `with your task (shared APIs, data structures, modules, files, or contracts).`,
    ``,
    `For each agent whose work will interface with yours, send them a short message`,
    `using this EXACT format on its own line:`,
    `SEND @agentId: <your message>`,
    ``,
    `Your message should tell them: what you are building, what you need from them`,
    `(interface, contract, type, API shape), and ask them to flag any conflicts early.`,
    ``,
    `If no agents are building anything that interfaces with your task, respond with:`,
    `NO_COORDINATION_NEEDED`,
    ``,
    `Be concise. Only reach out if there is a genuine interface dependency.`,
  ].join('\n');

  const { success, output } = await runAgent(prompt);
  if (!success) return;

  // Parse and send any SEND directives
  const lines = output.split('\n');
  for (const line of lines) {
    const match = line.match(/^SEND\s+(@\S+):\s*(.+)/);
    if (match) {
      const [, target, message] = match;
      await sendMessage(`${target} [pre-task coordination from ${AGENT_NAME}]: ${message.trim()}`);
      log(`  [coord] Sent pre-task message to ${target}`);
    }
  }
}

/**
 * Broadcast a completion summary so parallel agents know what was built
 * and what interfaces/outputs are now available.
 *
 * The task has already been marked complete on the server before this runs.
 * A failure here is a broadcast failure, not a task outcome — the claude
 * subprocess exit code is deliberately not propagated to any task-state
 * endpoint. See KI-10.
 */
async function broadcastCompletion(task, output, deps = {}) {
  const invokeAgent = deps.runAgent || runAgent;
  const send = deps.sendMessage || sendMessage;
  const logFn = deps.log || log;

  const prompt = [
    `You are ${AGENT_NAME}, an AI agent in the ACT coordination system.`,
    ``,
    `You just completed this task:`,
    `Title: ${task.title || '(untitled)'}`,
    ``,
    `Your output summary (first 800 chars):`,
    output.substring(0, 800),
    ``,
    `Write a single short status broadcast (2-4 sentences) for other agents telling them:`,
    `1. What you built`,
    `2. What interfaces, APIs, types, or outputs are now available for them to use`,
    `3. Any assumptions or constraints they should know about`,
    ``,
    `Start the message with "status:" — this tells the system it is a broadcast, not a request.`,
    `Keep it under 200 words.`,
  ].join('\n');

  let result;
  try {
    result = await invokeAgent(prompt);
  } catch (err) {
    logFn(`  [coord] Broadcast invocation threw (${err.message}); task already complete, ignoring`);
    return;
  }

  const { success, output: summary } = result;
  if (!success) {
    logFn(`  [coord] Broadcast invocation failed; task already complete, ignoring`);
    return;
  }

  const cleaned = (summary || '').trim();
  if (cleaned) {
    await send(cleaned.startsWith('status:') ? cleaned : `status: ${cleaned}`);
    logFn(`  [coord] Broadcast completion summary to team`);
  }
}

// ─── Task execution ───────────────────────────────────────────────────────────

/**
 * Fetch the AGENTS.md/CLAUDE.md brief for this agent from the ACT server.
 * Returns the brief content string, or null if not found.
 *
 * Short-circuits for the claude-code backend when an on-disk AGENTS.md or
 * CLAUDE.md exists in cwd (which is the project working directory via Go→Node
 * cwd inheritance). Claude-code auto-discovers CLAUDE.md walking up from cwd,
 * so HTTP-injecting the same content would double-load it in the prompt and
 * waste tokens. For the act-agent backend we still inject — that binary's
 * contextPaths loader only picks up the file after the Planner writes it, and
 * runners may be executing a task before the first PROJECT_BRIEF lands.
 */
async function fetchAgentBrief(projectName) {
  if (!projectName) return null;
  if (BACKEND === 'claude-code' && (existsSync('CLAUDE.md') || existsSync('AGENTS.md'))) {
    log(`  [brief] claude-code backend + on-disk AGENTS.md/CLAUDE.md detected — skipping HTTP inject (auto-loaded)`);
    return null;
  }
  try {
    const data = await get(`/api/projects/${encodeURIComponent(projectName)}/briefs/${encodeURIComponent(AGENT_ID)}`);
    const content = data.brief?.content || data.content || null;
    if (content) log(`  [brief] Loaded brief for project "${projectName}" via HTTP`);
    return content;
  } catch {
    return null; // no brief stored yet — non-fatal
  }
}

// ─── Session brief (write path) ───────────────────────────────────────────────
//
// The server stores one plain-string brief per {projectName, agentId} and
// replays `brief_stored` back into project.briefs on boot. Nothing ever wrote
// one (0 brief_stored events in all history — ticket
// agent-brief-session-save-never-fires-2026-08-13), so fetchAgentBrief above
// always 404'd and swarm agents started every task with zero project memory.
//
// The Runner owns this write, not the agent: swarm agents are stateless
// one-shots, the Runner is the process with the task lifecycle. The content is
// deterministic (no LLM call) — the agent's last MAX_BRIEF_ENTRIES completed
// task titles plus a one-line result each, newest first.

const BRIEF_SECTION_HEADER = '## Recent Work (most recent first)';
const MAX_BRIEF_ENTRIES = 5;
const MAX_BRIEF_CHARS = 2000;

function oneLine(text, max = 200) {
  const flat = String(text ?? '').replace(/\s+/g, ' ').trim();
  if (flat.length <= max) return flat;
  return flat.slice(0, max - 1) + '…';
}

/**
 * Read-modify-write of the Runner's own brief section.
 *
 * Anything before the section header is preserved verbatim (a Planner- or
 * CLI-authored preamble stays put); the section itself is rebuilt from the
 * parsed `- ` entry lines with the new entry prepended. Capped at
 * MAX_BRIEF_ENTRIES entries and MAX_BRIEF_CHARS characters, trimming oldest
 * entries first. Pure function — no I/O, no clock, no randomness.
 */
function buildRecentWorkBrief(existing, entry) {
  const content = String(existing ?? '');
  const idx = content.indexOf(BRIEF_SECTION_HEADER);
  const prefix = (idx >= 0 ? content.slice(0, idx) : content).trimEnd();
  const section = idx >= 0 ? content.slice(idx + BRIEF_SECTION_HEADER.length) : '';
  const previous = section
    .split('\n')
    .map(l => l.trim())
    .filter(l => l.startsWith('- '));

  let entries = [entry, ...previous].slice(0, MAX_BRIEF_ENTRIES);

  const render = (es) => {
    const parts = [];
    if (prefix) parts.push(prefix, '');
    parts.push(BRIEF_SECTION_HEADER, '', ...es);
    return parts.join('\n');
  };

  let out = render(entries);
  while (out.length > MAX_BRIEF_CHARS && entries.length > 1) {
    entries = entries.slice(0, -1); // drop oldest
    out = render(entries);
  }
  return out.length > MAX_BRIEF_CHARS ? out.slice(0, MAX_BRIEF_CHARS) : out;
}

/**
 * Store the agent's "recent work" brief on the ACT server after a successful
 * completion. Fully wrapped: every failure mode (project 404, server 500,
 * unreachable server) is swallowed and logged, so this can never affect the
 * task-completion path that ran before it.
 */
async function saveAgentBrief(projectName, task, result, deps = {}) {
  const getFn  = deps.get  || get;
  const postFn = deps.post || post;
  const logFn  = deps.log  || log;
  const nowFn  = deps.now  || (() => new Date().toISOString());

  if (!projectName) return false;

  try {
    let existing = '';
    try {
      const data = await getFn(`/api/projects/${encodeURIComponent(projectName)}/briefs/${encodeURIComponent(AGENT_ID)}`);
      existing = data.brief?.content || data.content || '';
    } catch {
      existing = ''; // 404 on the first completion — expected
    }

    const title  = oneLine(task?.title || task?.description || task?.id || 'untitled task', 120);
    const detail = oneLine(result) || '(no result recorded)';
    const entry  = `- [${nowFn()}] ${title} — ${detail}`;
    const content = buildRecentWorkBrief(existing, entry);

    await postFn(`/api/projects/${encodeURIComponent(projectName)}/briefs`, {
      agentId: AGENT_ID,
      content,
    });
    logFn(`  [brief] Saved session brief for "${projectName}" (${content.length} chars)`);
    return true;
  } catch (err) {
    logFn(`  [brief] Brief save failed (non-fatal): ${err.message}`);
    return false;
  }
}

async function executeTask(task) {
  log(`Task: ${task.id} — ${task.title || task.description}`);

  // Block 1a: inject AGENT.md brief from ACT server
  const projectName = task.projectName || task.metadata?.projectName || null;
  const agentBrief = await fetchAgentBrief(projectName);

  // Parallel awareness: identify interface dependencies and reach out proactively
  const parallelContext = await fetchParallelContext();
  if (parallelContext) {
    log(`  [coord] Checking interface dependencies with parallel agents...`);
    await proactiveCoordination(task, parallelContext);
  }

  // Check inbox for pending messages (gap analysis from Assurance, questions from peers)
  let inboxContext = null;
  try {
    const msgs = await getMessages(new Date(Date.now() - 3600000).toISOString()); // last hour
    if (msgs.length > 0) {
      log(`  [inbox] ${msgs.length} pending message(s) — including in task context`);
      inboxContext = msgs
        .map(m => `[${m.from || m.sender}]: ${m.message}`)
        .join('\n');
    }
  } catch { /* non-fatal */ }

  // Check for validation gaps attached to task metadata (from Assurance rejection)
  let gapContext = null;
  if (task.metadata?.validationGaps) {
    log(`  [assurance] Task was previously rejected — including gap analysis`);
    gapContext = `VALIDATION FEEDBACK (attempt ${task.metadata.validationAttempts || '?'}):\n${task.metadata.validationGaps}`;
  }

  const prompt = buildTaskPrompt(task, parallelContext, agentBrief, inboxContext, gapContext);
  await reportProgress(task.id, 10, 'Starting task execution');

  const { success, output, error, code } = await runAgent(prompt);

  if (success) {
    log(`  Task complete. Output length: ${output.length} chars`);
    await reportComplete(task.id, true, output.slice(0, 2000));
    // Session save: persist this completion into the agent's server-side brief
    // so the next task on this agent+project starts with recent-work memory.
    // saveAgentBrief never throws — the task is already complete.
    await saveAgentBrief(projectName, task, output.slice(0, 2000));
    // Broadcast what was built so parallel agents can wire against it
    await broadcastCompletion(task, output);
  } else {
    log(`  Task failed: ${error}`);
    await reportComplete(task.id, false, `Error: ${error}\n\n${output}`.slice(0, 2000));
  }

  return { exitCode: code ?? (success ? 0 : 1) };
}

function buildTaskPrompt(task, parallelContext = null, agentBrief = null, inboxContext = null, gapContext = null) {
  const lines = [];

  // AGENT.md brief goes first — establishes agent identity and project context
  if (agentBrief) {
    lines.push(`## Project Brief`, ``, agentBrief, ``, `---`, ``);
  }

  lines.push(
    `You are ${AGENT_NAME}, an autonomous AI agent connected to the ACT coordination system.`,
    AGENT_ROLE ? `Role: ${AGENT_ROLE}` : null,
    ``,
    `## Task`,
    `Title: ${task.title || '(untitled)'}`,
    `Description: ${task.description || '(no description)'}`,
  );

  if (task.metadata?.projectContext) {
    lines.push(``, `## Project Context`, task.metadata.projectContext);
  }

  if (task.requiredCapabilities?.length) {
    lines.push(``, `## Required Capabilities`, task.requiredCapabilities.join(', '));
  }

  // Parallel awareness: show agent who else is working and on what
  if (parallelContext) {
    lines.push(
      ``,
      `## Parallel Agents`,
      `These agents are working concurrently on this project:`,
      parallelContext,
      `Design your implementation to be compatible with their work.`,
      `If you discover a conflict or dependency mid-task, use send_message to reach out directly.`,
    );
  }


  // Pending messages (gap analysis from Assurance, peer questions)
  if (inboxContext) {
    lines.push(
      ``,
      `## Pending Messages`,
      `These messages were sent to you by other agents. Read them carefully — they may contain`,
      `gap analysis from Assurance (what failed validation) or questions from peers:`,
      ``,
      inboxContext,
    );
  }

  // Validation gap analysis (from Assurance rejection — attached to task metadata)
  if (gapContext) {
    lines.push(
      ``,
      `## IMPORTANT: Previous Validation Failed`,
      `This task was previously completed but REJECTED by Assurance. Fix the specific issues below:`,
      ``,
      gapContext,
      ``,
      `Focus on fixing these gaps. Do not rewrite everything — address the specific failures.`,
    );
  }

  // Extract success criteria from SPIL if present
  const criteria = extractSuccessCriteria(task.description || '');
  if (criteria.length > 0) {
    lines.push(
      ``,
      `## Success Criteria`,
      `Your output will be validated against these criteria (95% gate):`,
    );
    criteria.forEach((c, i) => lines.push(`  ${i + 1}. ${c}`));
  }

  lines.push(
    ``,
    `## Workflow`,
    `1. CLAIM FILES: Before editing any files, run \`act files claim <paths...>\` to prevent conflicts.`,
    `2. IMPLEMENT: Complete the task. Be thorough and precise.`,
    `3. SELF-VERIFY (Ralph Wiggum Loop): Before reporting complete, verify your own work:`,
    `   - Re-read your output critically. Does it actually satisfy each success criterion?`,
    `   - Run tests/linters if available. Check for obvious errors.`,
    `   - If you find gaps, fix them before continuing.`,
    `4. REPORT: Run \`act task complete <task-id> --result "<summary>"\` with a concise summary.`,
    `5. RELEASE FILES: Run \`act files release <paths...>\` (or they auto-release on complete).`,
    `6. CHECK MESSAGES: Run \`act message\` to see if other agents need your help.`,
    ``,
    `If this is a planning task, return valid JSON as specified in the task description.`,
  );

  return lines.join('\n');
}

/** Extract @success_criteria items from SPIL-formatted text */
function extractSuccessCriteria(text) {
  const lines = text.split('\n');
  let inCriteria = false;
  const criteria = [];
  for (const line of lines) {
    const trimmed = line.trim();
    if (/^@success_criteria:?\s*$/i.test(trimmed)) { inCriteria = true; continue; }
    if (trimmed.startsWith('@') && inCriteria) break;
    if (inCriteria && trimmed.startsWith('- ')) {
      criteria.push(trimmed.substring(2));
    }
  }
  return criteria;
}

// ─── Message handling ─────────────────────────────────────────────────────────

async function processMessages(messages) {
  if (messages.length === 0) return;
  log(`  ${messages.length} message(s) in inbox`);

  for (const msg of messages) {
    // Only respond to direct mentions to avoid broadcast storms
    if (msg.type !== 'direct_mention') continue;

    log(`  → Message from ${msg.from}: ${msg.message.slice(0, 80)}...`);

    const prompt = [
      `You are ${AGENT_NAME}, an AI agent in the ACT coordination system.`,
      `Another agent (${msg.from}) sent you this message:`,
      ``,
      msg.message,
      ``,
      `Respond helpfully and concisely. Your response will be sent back as a message.`,
      `Keep your reply under 500 words.`,
    ].join('\n');

    const { success, output, error } = await runAgent(prompt);
    const reply = success ? output : `I encountered an error: ${error}`;

    await sendMessage(`@${msg.from} ${reply.slice(0, 1000)}`);
    log(`  ↩ Replied to ${msg.from}`);
  }
}

// ─── Utilities ────────────────────────────────────────────────────────────────

function log(msg) {
  const ts = new Date().toISOString().split('T')[1].slice(0, 8);
  console.log(`[${ts}] [${AGENT_ID}] ${msg}`);
}

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

// ─── Main loop ────────────────────────────────────────────────────────────────

async function main() {
  log(`ACT Agent Runner starting`);
  log(`Server: ${ACT_SERVER_URL}`);
  log(`Agent CLI: ${AGENT_CLI}`);
  if (AGENT_ROLE) log(`Role: ${AGENT_ROLE}`);
  log(`Poll interval: ${POLL_INTERVAL}ms`);
  log(`Max iterations: ${maxIterations === 0 ? 'unlimited' : maxIterations}`);
  log(``);

  // Register with ACT server
  try {
    await register();
  } catch (err) {
    const msg = String(err.message);
    console.error(`Fatal: Could not register with ACT server at ${ACT_SERVER_URL}`);
    console.error(`  ${msg}`);
    if (msg.includes('ECONNREFUSED') || msg.includes('fetch failed')) {
      console.error(`  Server is unreachable. Is it running? cd server && npm run dev`);
    } else if (msg.includes('HTTP 409')) {
      console.error(`  Stale agent registration that self-heal could not clear.`);
      console.error(`  Try: curl -X DELETE ${ACT_SERVER_URL}/api/agents/${AGENT_ID}`);
    }
    process.exit(1);
  }

  let iteration   = 0;
  let lastMsgTime = new Date().toISOString();

  // Graceful shutdown
  process.on('SIGINT',  () => { log('Shutting down (SIGINT)');  process.exit(0); });
  process.on('SIGTERM', () => { log('Shutting down (SIGTERM)'); process.exit(0); });

  while (maxIterations === 0 || iteration < maxIterations) {
    iteration++;

    try {
      // 1. Check for assigned task
      const task = await getTask();
      if (task) {
        if (liveProcesses.has(AGENT_ID)) {
          const live = liveProcesses.get(AGENT_ID);
          log(`Agent ${AGENT_ID} already has a live process (pid ${live.pid}), skipping`);
          continue;
        }

        liveProcesses.set(AGENT_ID, {
          pid: process.pid,
          startedAt: new Date().toISOString(),
          taskId: task.id,
        });

        let taskExitCode = -1;
        try {
          const taskResult = await executeTask(task);
          taskExitCode = taskResult?.exitCode ?? 0;
        } finally {
          liveProcesses.delete(AGENT_ID);
          log(`Agent ${AGENT_ID} process exited`);
          try {
            await sendSessionEnded(task.id, taskExitCode);
          } catch (err) {
            log(`  [lifecycle] failed to report session end: ${err.message}`);
          }
        }

        // After finishing a task, skip the sleep and immediately check for the next
        continue;
      }

      // 2. Check inbox for messages (only when idle)
      const messages = await getMessages(lastMsgTime);
      if (messages.length > 0) {
        lastMsgTime = new Date().toISOString();
        await processMessages(messages);
      }
    } catch (err) {
      log(`Error in coordination loop: ${err.message}`);
    }

    // 3. Sleep before next poll
    await sleep(POLL_INTERVAL);
  }

  log(`Max iterations (${maxIterations}) reached. Exiting.`);
}

const isDirectInvocation = process.argv[1] && import.meta.url === new URL(`file://${process.argv[1]}`).href;
if (isDirectInvocation) {
  main().catch(err => {
    console.error('Fatal error:', err);
    process.exit(1);
  });
}

export { broadcastCompletion, saveAgentBrief, buildRecentWorkBrief };
