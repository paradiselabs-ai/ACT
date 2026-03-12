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
const POLL_INTERVAL  = parseInt(process.env.POLL_INTERVAL_MS || '5000',  10);
const TASK_TIMEOUT   = parseInt(process.env.TASK_TIMEOUT_MS  || '120000', 10);

let maxIterations = parseInt(process.env.MAX_ITERATIONS || '100', 10);

// ─── CLI args ─────────────────────────────────────────────────────────────────

const { values: args } = parseArgs({
  options: {
    'agent-id':       { type: 'string' },
    'name':           { type: 'string' },
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
  --capabilities <list>    Comma-separated capabilities (e.g. typescript,react,testing)
  --max-iterations <n>     Max coordination loops before exit (default: 100, 0 = unlimited)
  --poll-interval <ms>     Milliseconds between polls (default: 5000)
  --help, -h               Show this help

Environment variables:
  ACT_SERVER_URL           ACT server URL (default: http://localhost:8080)
  CLAUDE_PATH              Path to claude binary (default: claude)
  POLL_INTERVAL_MS         Polling interval in ms
  MAX_ITERATIONS           Max loops
  TASK_TIMEOUT_MS          Timeout for a single claude invocation (default: 120000)
`);
  process.exit(0);
}

const AGENT_ID     = args['agent-id'];
const AGENT_NAME   = args['name'] || AGENT_ID;
const CAPABILITIES = (args['capabilities'] || '').split(',').map(s => s.trim()).filter(Boolean);

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
  await post('/api/agents/register', {
    agentId: AGENT_ID,
    name: AGENT_NAME,
    capabilities: CAPABILITIES,
  });
  log(`Registered as "${AGENT_NAME}" [${CAPABILITIES.join(', ') || 'no capabilities listed'}]`);
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
  await post(`/api/tasks/${taskId}/complete`, {
    agentId: AGENT_ID,
    success,
    result,
  });
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

// ─── Claude invocation ────────────────────────────────────────────────────────

async function runClaude(prompt) {
  log(`  Invoking claude CLI...`);
  try {
    const { stdout, stderr } = await execFileAsync(
      CLAUDE_PATH,
      ['--print', '--dangerously-skip-permissions', prompt],
      { timeout: TASK_TIMEOUT, maxBuffer: 10 * 1024 * 1024 }
    );
    if (stderr) log(`  [claude stderr] ${stderr.trim()}`);
    return { success: true, output: stdout.trim() };
  } catch (err) {
    const output = err.stdout?.trim() || '';
    const errMsg = err.stderr?.trim() || err.message;
    return { success: false, output, error: errMsg };
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
 * Ask Claude to identify interface points with parallel agents and send
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

  const { success, output } = await runClaude(prompt);
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

  const { success, output: summary } = await runClaude(prompt);
  if (!success) return;

  const cleaned = summary.trim();
  if (cleaned) {
    await sendMessage(cleaned.startsWith('status:') ? cleaned : `status: ${cleaned}`);
    log(`  [coord] Broadcast completion summary to team`);
  }
}

// ─── Task execution ───────────────────────────────────────────────────────────

async function executeTask(task) {
  log(`Task: ${task.id} — ${task.title || task.description}`);

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

  const prompt = buildTaskPrompt(task, pvmContext, parallelContext);
  await reportProgress(task.id, 10, 'Starting task execution');

  const { success, output, error } = await runClaude(prompt);

  if (success) {
    log(`  Task complete. Output length: ${output.length} chars`);
    await reportComplete(task.id, true, output.slice(0, 2000));
    // Broadcast what was built so parallel agents can wire against it
    await broadcastCompletion(task, output);
  } else {
    log(`  Task failed: ${error}`);
    await reportComplete(task.id, false, `Error: ${error}\n\n${output}`.slice(0, 2000));
  }
}

function buildTaskPrompt(task, pvmContext = null, parallelContext = null) {
  const lines = [
    `You are ${AGENT_NAME}, an autonomous AI agent connected to the ACT coordination system.`,
    ``,
    `## Task`,
    `Title: ${task.title || '(untitled)'}`,
    `Description: ${task.description || '(no description)'}`,
  ];

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

  lines.push(
    ``,
    `## Instructions`,
    `Complete this task to the best of your ability. Be thorough and precise.`,
    `Return your output as a clear, well-structured response.`,
    `If this is a planning task, return valid JSON as specified in the task description.`,
  );

  return lines.join('\n');
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

    const { success, output, error } = await runClaude(prompt);
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
  log(`Claude: ${CLAUDE_PATH}`);
  log(`Poll interval: ${POLL_INTERVAL}ms`);
  log(`Max iterations: ${maxIterations === 0 ? 'unlimited' : maxIterations}`);
  log(``);

  // Register with ACT server
  try {
    await register();
  } catch (err) {
    console.error(`Fatal: Could not register with ACT server at ${ACT_SERVER_URL}`);
    console.error(`  ${err.message}`);
    console.error(`  Is the server running? cd server && npm run dev`);
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
        await executeTask(task);
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
