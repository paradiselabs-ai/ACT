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
 *   POLL_INTERVAL_MS Milliseconds between polls (default: 5000)
 *   MAX_ITERATIONS   Max loops before exiting (default: 100, 0 = unlimited)
 *   TASK_TIMEOUT_MS  Max ms for a single claude invocation (default: 120000)
 */

import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { parseArgs } from 'node:util';

const execFileAsync = promisify(execFile);

// ─── Config ──────────────────────────────────────────────────────────────────

const ACT_SERVER_URL = (process.env.ACT_SERVER_URL || 'http://localhost:8080').replace(/\/$/, '');
const CLAUDE_PATH    = process.env.CLAUDE_PATH    || 'claude';
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
  --backend <name>         Agent execution backend: act-agent (default) or claude-code
  --capabilities <list>    Comma-separated capabilities (e.g. typescript,react,testing)
  --max-iterations <n>     Max coordination loops before exit (default: 100, 0 = unlimited)
  --poll-interval <ms>     Milliseconds between polls (default: 5000)
  --help, -h               Show this help

Environment variables:
  ACT_SERVER_URL           ACT server URL (default: http://localhost:8080)
  ACT_BACKEND              Default backend if --backend not passed (default: act-agent)
  ACTOR_CLI / AGENT_CLI    Agent CLI binary (fallback: ./act-agent/act-agent)
  CLAUDE_PATH              Path to claude binary (default: claude)
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
  const body = {
    agentId: AGENT_ID,
    name: AGENT_NAME,
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
  const data = await get('/api/tasks/assigned', { agent_id: AGENT_ID });
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
  const data = await get(`/api/agents/${encodeURIComponent(AGENT_ID)}/messages`, params);
  return data.messages || [];
}

async function sendMessage(message) {
  await post('/api/messages', { sender: AGENT_ID, message });
}

async function sendSessionEnded(taskId, code) {
  await post('/api/messages', {
    sender: AGENT_ID,
    message: `status: session ended for task ${taskId} (exit code: ${code})`
  });
}

// ─── Agent CLI invocation ─────────────────────────────────────────────────────

async function runAgent(prompt) {
  if (BACKEND === 'claude-code') {
    return runAgentClaudeCode(prompt);
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

async function runAgentClaudeCode(prompt) {
  log(`  Invoking Claude Code (${CLAUDE_PATH})...`);
  try {
    const { stdout, stderr } = await execFileAsync(
      CLAUDE_PATH,
      ['--print', '--dangerously-skip-permissions', prompt],
      { timeout: TASK_TIMEOUT, maxBuffer: 10 * 1024 * 1024 }
    );
    if (stderr) log(`  [claude stderr] ${stderr.trim()}`);
    return {
      success: true,
      output: stdout.trim(),
      code: 0
    };
  } catch (err) {
    const output = err.stdout?.trim() || '';
    const errMsg = err.stderr?.trim() || err.message;
    const code = Number.isInteger(err.code) ? err.code : 1;
    return {
      success: false,
      output,
      error: errMsg,
      code
    };
  }
}

// ─── Wire 2: Complexity-triggered PVM search ─────────────────────────────────

// Keywords that signal a task will benefit from past coordination context
const COMPLEXITY_KEYWORDS = [
  'refactor', 'architect', 'implement', 'integrate', 'design', 'migrate',
  'optimize', 'overhaul', 'rewrite', 'scaffold', 'system', 'pipeline',
  'coordinate', 'orchestrate', 'framework', 'infrastructure', 'strategy'
];
const COMPLEXITY_THRESHOLD = 4; // score must exceed this to trigger PVM search

/**
 * Score task complexity — higher means more likely to benefit from PVM context.
 * Heuristic: description length + capability count + keyword hits.
 */
function scoreComplexity(task) {
  let score = 0;
  const text = `${task.title || ''} ${task.description || ''}`.toLowerCase();

  // Description length: every 100 chars = 1 point (cap at 5)
  score += Math.min(5, Math.floor(text.length / 100));

  // Each required capability = 1 point (cap at 3)
  score += Math.min(3, (task.requiredCapabilities || []).length);

  // Each complexity keyword found = 1 point
  for (const kw of COMPLEXITY_KEYWORDS) {
    if (text.includes(kw)) score += 1;
  }

  return score;
}

/**
 * Fetch semantically similar past coordination patterns from PVM.
 * Returns a formatted string ready for prompt injection, or null if nothing useful.
 */
async function fetchPVMContext(task) {
  try {
    const query = [task.title, task.description].filter(Boolean).join(' ').substring(0, 300);
    const data = await get('/api/pvm/search', { query, limit: 5 });
    const results = data.results || [];
    if (results.length === 0) return null;

    const formatted = results
      .map((r, i) => {
        const msg = r.message || r;
        return `[${i + 1}] ${msg.agent || 'unknown'} (${msg.type || 'event'}): ${(msg.message || '').substring(0, 300)}`;
      })
      .join('\n\n');

    log(`  [PVM] Injecting ${results.length} related pattern(s) into task prompt.`);
    return formatted;
  } catch (err) {
    log(`  [PVM] Search unavailable (non-fatal): ${err.message}`);
    return null;
  }
}

// ─── Parallel awareness ───────────────────────────────────────────────────────

/**
 * Fetch all active agents and their in-progress tasks.
 * Returns a formatted string describing the parallel work landscape.
 */
async function fetchParallelContext() {
  try {
    const [agentsData, tasksData] = await Promise.all([
      get('/api/agents'),
      get('/api/tasks'),
    ]);

    const agents = (agentsData.agents || []).filter(a => a.id !== AGENT_ID);
    const tasks  = (tasksData.tasks  || []).filter(t =>
      t.status === 'in_progress' || t.status === 'assigned'
    );

    if (agents.length === 0) return null;

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
 */
async function broadcastCompletion(task, output) {
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

  const { success, output: summary } = await runAgent(prompt);
  if (!success) return;

  const cleaned = summary.trim();
  if (cleaned) {
    await sendMessage(cleaned.startsWith('status:') ? cleaned : `status: ${cleaned}`);
    log(`  [coord] Broadcast completion summary to team`);
  }
}

// ─── Task execution ───────────────────────────────────────────────────────────

/**
 * Fetch the AGENT.md brief for this agent from the ACT server.
 * Returns the brief content string, or null if not found.
 */
async function fetchAgentBrief(projectName) {
  if (!projectName) return null;
  try {
    const data = await get(`/api/projects/${encodeURIComponent(projectName)}/briefs/${encodeURIComponent(AGENT_ID)}`);
    const content = data.brief?.content || data.content || null;
    if (content) log(`  [brief] Loaded AGENT.md brief for project "${projectName}"`);
    return content;
  } catch {
    return null; // no brief stored yet — non-fatal
  }
}

async function executeTask(task) {
  log(`Task: ${task.id} — ${task.title || task.description}`);

  // Block 1a: inject AGENT.md brief from ACT server
  const projectName = task.projectName || task.metadata?.projectName || null;
  const agentBrief = await fetchAgentBrief(projectName);

  // Wire 2: inject PVM context for complex tasks
  let pvmContext = null;
  const complexity = scoreComplexity(task);
  if (complexity > COMPLEXITY_THRESHOLD) {
    log(`  [PVM] Task complexity score ${complexity} > ${COMPLEXITY_THRESHOLD} — searching coordination memory...`);
    pvmContext = await fetchPVMContext(task);
  }

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

  const prompt = buildTaskPrompt(task, pvmContext, parallelContext, agentBrief, inboxContext, gapContext);
  await reportProgress(task.id, 10, 'Starting task execution');

  const { success, output, error, code } = await runAgent(prompt);

  if (success) {
    log(`  Task complete. Output length: ${output.length} chars`);
    await reportComplete(task.id, true, output.slice(0, 2000));
    // Broadcast what was built so parallel agents can wire against it
    await broadcastCompletion(task, output);
  } else {
    log(`  Task failed: ${error}`);
    await reportComplete(task.id, false, `Error: ${error}\n\n${output}`.slice(0, 2000));
  }

  return { exitCode: code ?? (success ? 0 : 1) };
}

function buildTaskPrompt(task, pvmContext = null, parallelContext = null, agentBrief = null, inboxContext = null, gapContext = null) {
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

  // Wire 2: inject PVM patterns when available (complexity-triggered)
  if (pvmContext) {
    lines.push(
      ``,
      `## Related Past Work`,
      `The following coordination patterns from similar past tasks may be relevant.`,
      `Use them to avoid known pitfalls and build on approaches that worked:`,
      ``,
      pvmContext,
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
  const lines = text.split('\\n');
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

main().catch(err => {
  console.error('Fatal error:', err);
  process.exit(1);
});
