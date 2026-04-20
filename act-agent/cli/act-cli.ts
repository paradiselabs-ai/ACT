#!/usr/bin/env npx tsx
/**
 * ACT CLI - Agent interface to ACT coordination layer
 *
 * Headless agents use this CLI instead of MCP (saves ~47K tokens of schema overhead).
 * Each command is a thin HTTP wrapper to the ACT server.
 */

import { parseArgs } from 'node:util';
import { ACTClient } from './act-client.js';

const DEFAULT_SERVER_URL = process.env.ACT_SERVER_URL || 'http://localhost:8080';

function getAgentId(args: Record<string, any>): string | null {
  if (args['agent-id']) return args['agent-id'];
  if (process.env.ACT_AGENT_ID) return process.env.ACT_AGENT_ID;
  return null;
}

function printError(msg: string): void {
  console.error(msg);
}

async function cmdContext(client: ACTClient, args: Record<string, any>): Promise<void> {
  const agentId = args['agent-id'] || args['<agent-id>'];
  const projectName = args['--project'];

  if (!agentId) {
    printError('Error: agent-id is required (--agent-id flag or ACT_AGENT_ID env var)');
    process.exit(1);
  }
  if (!projectName) {
    printError('Error: --project is required');
    process.exit(1);
  }

  const serverUrl = client.getServerUrl();

  // Fetch brief + task + parallel context + messages in parallel
  const [brief, task, allTasksRes, allAgentsRes, locksRes, messages] = await Promise.all([
    client.getBrief(projectName, agentId),
    client.getAssignedTask(agentId),
    fetch(`${serverUrl}/api/tasks`).then(r => r.json() as any).catch(() => null),
    fetch(`${serverUrl}/api/agents`).then(r => r.json() as any).catch(() => null),
    fetch(`${serverUrl}/api/files/locks`).then(r => r.json() as any).catch(() => null),
    client.getMessages(agentId, 5),
  ]);

  // Agent brief
  if (brief) {
    console.log('## Agent Brief');
    console.log(brief);
    console.log();
  }

  // Current task with enriched context
  if (task) {
    console.log('## Current Task');
    console.log(`ID: ${task.id}`);
    console.log(`Title: ${task.title || task.metadata?.title || task.description?.substring(0, 60) || 'Untitled'}`);
    console.log(`Priority: ${task.priority || 'medium'}`);
    console.log(`Description: ${task.description}`);

    // Extract SPIL success criteria from description
    const criteria = extractSuccessCriteriaSimple(task.description || '');
    if (criteria.length > 0) {
      console.log();
      console.log('Success Criteria:');
      criteria.forEach((c: string, i: number) => console.log(`  ${i + 1}. ${c}`));
    }

    // Dependencies
    if (task.dependencies?.length) {
      const allTasks = allTasksRes?.tasks || (Array.isArray(allTasksRes) ? allTasksRes : []);
      const taskMap = new Map<string, any>(allTasks.map((t: any) => [t.id, t]));
      console.log();
      console.log('Dependencies (must be done before this task):');
      for (const depId of task.dependencies) {
        const dep = taskMap.get(depId);
        if (dep) {
          console.log(`  - ${dep.title || dep.id} [${dep.status}]`);
        }
      }
    }

    // What depends on this task (downstream)
    const allTasks = allTasksRes?.tasks || (Array.isArray(allTasksRes) ? allTasksRes : []);
    const downstream = allTasks.filter((t: any) => t.dependencies?.includes(task.id));
    if (downstream.length > 0) {
      console.log();
      console.log('Blocked by this task (deliver to unblock):');
      for (const t of downstream) {
        console.log(`  - ${t.title || t.id} [${t.status}]`);
      }
    }

    console.log();
  } else {
    console.log(`No task currently assigned to ${agentId}`);
    console.log();
  }

  // Parallel agent awareness
  const allAgents = Array.isArray(allAgentsRes) ? allAgentsRes : allAgentsRes?.agents || [];
  const otherAgents = allAgents.filter((a: any) => a.id !== agentId && a.status !== 'offline');
  if (otherAgents.length > 0) {
    const allTasks = allTasksRes?.tasks || (Array.isArray(allTasksRes) ? allTasksRes : []);
    console.log('## Parallel Agents');
    for (const agent of otherAgents) {
      const agentTask = allTasks.find((t: any) => t.assignedAgent === agent.id && (t.status === 'assigned' || t.status === 'in_progress'));
      const taskInfo = agentTask ? `working on: ${agentTask.title || agentTask.id}` : 'idle';
      console.log(`  ${agent.id} (${agent.status}) — ${taskInfo}`);
    }
    console.log();
  }

  // Active file locks relevant to this agent
  const locks = locksRes?.locks || [];
  if (locks.length > 0) {
    const myLocks = locks.filter((l: any) => l.agentId === agentId);
    const otherLocks = locks.filter((l: any) => l.agentId !== agentId);
    if (myLocks.length > 0) {
      console.log('## Your File Locks');
      for (const l of myLocks) console.log(`  ${l.filePath || l.file}`);
      console.log();
    }
    if (otherLocks.length > 0) {
      console.log('## Files Locked by Others (do not edit)');
      for (const l of otherLocks) console.log(`  ${l.filePath || l.file} — ${l.agentId}`);
      console.log();
    }
  }

  // Pending messages
  if (messages.length > 0) {
    console.log(`## Messages (${messages.length})`);
    for (const msg of messages) {
      console.log(`  [${msg.from || msg.sender}]: ${msg.message}`);
    }
    console.log();
    console.log('Respond with: act message "your reply" --agent-id ' + agentId);
    console.log();
  }
}

/** Simple SPIL success criteria extraction (no server dependency) */
function extractSuccessCriteriaSimple(text: string): string[] {
  const lines = text.split('\n');
  let inCriteria = false;
  const criteria: string[] = [];
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed.match(/^@success_criteria:?\s*$/i)) { inCriteria = true; continue; }
    if (trimmed.startsWith('@') && inCriteria) break; // next section
    if (inCriteria && trimmed.startsWith('- ')) {
      criteria.push(trimmed.substring(2));
    }
  }
  return criteria;
}

async function cmdTaskComplete(client: ACTClient, args: Record<string, any>): Promise<void> {
  const taskId = args['<task-id>'];
  const agentId = getAgentId(args);
  const result = args['--result'];

  if (!taskId) {
    printError('Error: task-id is required');
    process.exit(1);
  }
  if (!agentId) {
    printError('Error: agent-id is required (--agent-id flag or ACT_AGENT_ID env var)');
    process.exit(1);
  }

  const response = await client.completeTask(taskId, agentId, true, result);

  if (response.success) {
    console.log(`Task ${taskId} marked complete`);
  } else {
    printError(response.error || 'Failed to complete task');
    process.exit(1);
  }
}

async function cmdTaskProgress(client: ACTClient, args: Record<string, any>): Promise<void> {
  const taskId = args['<task-id>'];
  const agentId = getAgentId(args);
  const percent = args['--percent'];

  if (!taskId) {
    printError('Error: task-id is required');
    process.exit(1);
  }
  if (!agentId) {
    printError('Error: agent-id is required (--agent-id flag or ACT_AGENT_ID env var)');
    process.exit(1);
  }
  if (percent === undefined) {
    printError('Error: --percent is required');
    process.exit(1);
  }

  const percentNum = parseInt(String(percent), 10);
  if (isNaN(percentNum) || percentNum < 0 || percentNum > 100 || !Number.isInteger(percentNum)) {
    printError('Error: --percent must be an integer between 0 and 100');
    process.exit(1);
  }

  const response = await client.updateTaskProgress(taskId, agentId, percentNum, 'in_progress');

  if (response.success) {
    console.log(`Task ${taskId} progress updated to ${percentNum}%`);
  } else {
    printError(response.error || 'Failed to update task progress');
    process.exit(1);
  }
}

async function cmdBriefUpdate(client: ACTClient, args: Record<string, any>): Promise<void> {
  const agentId = args['<agent-id>'];
  const projectName = args['--project'];
  const session = args['--session'];

  if (!agentId) {
    printError('Error: agent-id is required');
    process.exit(1);
  }
  if (!projectName) {
    printError('Error: --project is required');
    process.exit(1);
  }
  if (!session) {
    printError('Error: --session is required');
    process.exit(1);
  }

  try {
    await client.storeBrief(projectName, agentId, session);
    console.log(`Brief updated for agent ${agentId} in project ${projectName}`);
  } catch (error: any) {
    if (error.message.includes('404') || error.message.includes('not found')) {
      printError(`Error: Project "${projectName}" not found`);
    } else {
      printError(error.message);
    }
    process.exit(1);
  }
}

async function cmdFilesClaim(client: ACTClient, args: Record<string, any>): Promise<void> {
  const agentId = getAgentId(args);
  const taskId = args['--task-id'];
  const paths = args['<paths>'];

  if (!agentId) {
    printError('Error: agent-id is required (--agent-id flag or ACT_AGENT_ID env var)');
    process.exit(1);
  }
  if (!taskId) {
    printError('Error: --task-id is required');
    process.exit(1);
  }
  if (!paths || paths.length === 0) {
    printError('Error: at least one file path is required');
    process.exit(1);
  }

  const response = await client.claimFiles(agentId, taskId, paths);

  if (response.success && response.claimed) {
    console.log(`Claimed ${response.claimed.length} file(s): ${response.claimed.join(', ')}`);
  } else if (response.conflict) {
    printError(`Error: File conflict - ${response.conflicts?.length} file(s) already locked`);
    response.conflicts?.forEach((c: any) => {
      printError(`  - ${c.filePath} locked by ${c.lockedBy} (task: ${c.taskId})`);
    });
    process.exit(1);
  } else {
    printError(response.error || 'Failed to claim files');
    process.exit(1);
  }
}

async function cmdFilesRelease(client: ACTClient, args: Record<string, any>): Promise<void> {
  const agentId = getAgentId(args);
  const paths = args['<paths>'];
  const taskId = args['--task-id'];

  if (!agentId) {
    printError('Error: agent-id is required (--agent-id flag or ACT_AGENT_ID env var)');
    process.exit(1);
  }
  if (!paths || paths.length === 0) {
    printError('Error: at least one file path is required');
    process.exit(1);
  }

  const response = await client.releaseFiles(agentId, paths, taskId);

  if (response.success && response.released) {
    console.log(`Released ${response.released.length} file(s): ${response.released.join(', ')}`);
  } else {
    printError(response.error || 'Failed to release files');
    process.exit(1);
  }
}

async function cmdMessage(client: ACTClient, args: Record<string, any>): Promise<void> {
  const agentId = getAgentId(args);
  const text = args['<text>'];

  if (!agentId) {
    printError('Error: agent-id is required (--agent-id flag or ACT_AGENT_ID env var)');
    process.exit(1);
  }

  // No text = read inbox
  if (!text) {
    const messages = await client.getMessages(agentId);
    if (messages.length === 0) {
      console.log('No messages.');
      return;
    }
    for (const msg of messages) {
      const time = msg.timestamp ? new Date(msg.timestamp).toLocaleTimeString() : '';
      console.log(`[${time}] ${msg.from || msg.sender}: ${msg.message}`);
    }
    return;
  }

  // Text provided = send message
  try {
    await client.sendMessage(agentId, text);
    console.log('Message sent');
  } catch (error: any) {
    printError(error.message);
    process.exit(1);
  }
}

async function cmdTaskSubmitValidation(client: ACTClient, args: Record<string, any>): Promise<void> {
  const taskId = args['<task-id>'] || args['task-id'];
  const agentId = getAgentId(args);
  const selfVerification = args['result'] || undefined;

  if (!taskId) { printError('Error: task-id is required'); process.exit(1); }
  if (!agentId) { printError('Error: agent-id is required'); process.exit(1); }

  const result = await client.submitForValidation(taskId, agentId, selfVerification);
  if (result.success) {
    console.log(`Task ${taskId} submitted for Assurance validation`);
  } else {
    printError(`Failed: ${result.error}`);
    process.exit(1);
  }
}

async function cmdLog(client: ACTClient, args: Record<string, any>): Promise<void> {
  const tail = args['--tail'] || 20;
  // Scope to active project. Tier 1 agents call this via act_cli and expect
  // their own project's event stream — without scoping they see stale
  // cross-project events and incorrectly decide other projects are active.
  const project = args['--project'] || process.env.ACT_PROJECT;
  const events = await client.getRecentLog(tail, project);

  for (const event of events) {
    console.log(`[${event.timestamp}] [${event.agent}] ${event.type}: ${event.message}`);
  }
}

async function cmdGraphTask(client: ACTClient, taskId: string): Promise<void> {
  if (!taskId) {
    printError('Error: task-id is required');
    printError('Usage: act graph task <task-id>');
    process.exit(1);
  }

  try {
    const [target, allTasks] = await Promise.all([
      client.getTask(taskId),
      client.getTasks()
    ]);

    if (!target) {
      printError(`Task not found: ${taskId}`);
      process.exit(1);
    }

    const taskMap = new Map<string, any>(allTasks.map((t: any) => [t.id, t]));
    const seen = new Set<string>();
    printTaskTree(target, taskMap, 0, seen);
  } catch (error: any) {
    printError(`Failed to render task graph: ${error.message}`);
    process.exit(1);
  }
}

function printTaskTree(task: any, taskMap: Map<string, any>, depth: number, seen: Set<string>): void {
  if (seen.has(task.id)) return;
  seen.add(task.id);

  const indent = depth === 0 ? '' : '  '.repeat(depth - 1) + '└─';
  const status = task.status || 'unknown';
  const title = task.title || task.metadata?.title || task.description?.substring(0, 60) || task.id;
  const agent = task.assignedAgent || 'unassigned';
  const deps = Array.isArray(task.dependencies) ? task.dependencies : [];
  const depsDone = deps.filter((depId: string) => {
    const dep = taskMap.get(depId);
    return dep && (dep.status === 'completed' || dep.status === 'validated');
  }).length;
  const depsInfo = deps.length > 0 ? ` deps:${depsDone}/${deps.length}` : ' deps:0/0';
  console.log(`${indent}[${status}] ${title} (${agent})${depsInfo}`);

  for (const [, child] of taskMap) {
    if (Array.isArray(child.dependencies) && child.dependencies.includes(task.id)) {
      printTaskTree(child, taskMap, depth + 1, seen);
    }
  }
}

async function cmdGraphUnverified(client: ACTClient): Promise<void> {
  try {
    const allTasks = await client.getTasks();
    const unverified = allTasks.filter((t: any) => t.status === 'completed');

    if (unverified.length === 0) {
      console.log('No unverified completed tasks.');
      return;
    }

    console.log(`Unverified completed tasks (${unverified.length}):`);
    for (const t of unverified) {
      const id = t.id || 'unknown';
      const title = t.title || t.metadata?.title || t.description?.substring(0, 60) || 'Untitled';
      const agent = t.assignedAgent || 'unassigned';
      const completedAt = t.completedAt || 'unknown';
      console.log(`- ${id} | ${title}`);
      console.log(`  agent: ${agent} | completed: ${completedAt}`);
    }
  } catch (error: any) {
    printError(`Failed to list unverified tasks: ${error.message}`);
    process.exit(1);
  }
}

async function cmdGraphConflicts(client: ACTClient): Promise<void> {
  const serverUrl = client.getServerUrl();
  try {
    const [locksRes, agentsRes, allTasks] = await Promise.all([
      fetch(`${serverUrl}/api/files/locks`),
      fetch(`${serverUrl}/api/agents`),
      client.getTasks()
    ]);

    if (!locksRes.ok) {
      throw new Error(`locks endpoint returned HTTP ${locksRes.status}`);
    }
    if (!agentsRes.ok) {
      throw new Error(`agents endpoint returned HTTP ${agentsRes.status}`);
    }

    const locksData = await locksRes.json() as any;
    const agentsData = await agentsRes.json() as any;
    const locks = locksData.locks || [];
    const agents = Array.isArray(agentsData) ? agentsData : agentsData.agents || [];

    if (locks.length === 0) {
      console.log('No file locks active.');
      return;
    }

    const agentMap = new Map<string, any>(agents.map((a: any) => [a.id, a]));
    const taskMap = new Map<string, any>(allTasks.map((t: any) => [t.id, t]));

    console.log(`Active file locks (${locks.length}):`);
    for (const lock of locks) {
      const filePath = lock.filePath || lock.file || 'unknown-file';
      const agentId = lock.agentId || 'unknown-agent';
      const taskId = lock.taskId || 'unknown-task';
      const agent = agentMap.get(agentId);
      const task = taskMap.get(taskId);
      const reasons: string[] = [];

      if (!agent || agent.status === 'offline') reasons.push('agent offline');
      if (task && (task.status === 'completed' || task.status === 'validated')) reasons.push(`task ${task.status}`);
      if (!task) reasons.push('task missing');

      const staleTag = reasons.length > 0 ? ` [STALE: ${reasons.join(', ')}]` : '';
      console.log(`- ${filePath}`);
      console.log(`  locked by: ${agentId} | task: ${taskId}${staleTag}`);
    }
  } catch (error: any) {
    printError(`Failed to inspect file lock conflicts: ${error.message}`);
    process.exit(1);
  }
}

async function cmdStatus(client: ACTClient): Promise<void> {
  const serverUrl = client.getServerUrl();
  // Scope to current project when ACT_PROJECT is set — default agent/runner
  // invocations always have it set, so `act status` in a session shows the
  // current project's state. Unset only in detached ops tooling.
  const project = process.env.ACT_PROJECT;
  const projectParam = project ? `?project=${encodeURIComponent(project)}` : '';
  try {
    const res = await fetch(`${serverUrl}/api/status${projectParam}`);
    const s = await res.json() as any;

    console.log(`ACT System Status — ${s.timestamp}`);
    console.log();

    // Tasks
    console.log(`Tasks: ${s.tasks.total}`);
    for (const [status, count] of Object.entries(s.tasks.byStatus)) {
      console.log(`  ${status}: ${count}`);
    }
    console.log();

    // Agents
    console.log(`Agents: ${s.agents.total}`);
    for (const agent of s.agents.list || []) {
      const task = agent.currentTask ? ` → ${agent.currentTask.substring(0, 8)}...` : '';
      const role = agent.role ? ` [${agent.role}]` : '';
      console.log(`  ${agent.id}${role}: ${agent.status}${task}`);
    }
    console.log();

    // Other
    console.log(`File locks: ${s.fileLocks}`);
    console.log(`Projects: ${s.projects}`);
    if (s.pvm) {
      console.log(`PVM: ${s.pvm.indexed || 0} events indexed`);
    }
  } catch {
    printError('Cannot reach ACT server at ' + serverUrl);
    process.exit(1);
  }
}

/** Run a nomik command and print output. Returns false if nomik unavailable. */
async function runNomik(args: string[]): Promise<boolean> {
  const { execFileSync } = await import('child_process');
  try {
    const output = execFileSync('nomik', args, {
      encoding: 'utf-8',
      timeout: 30000,
      cwd: process.cwd(),
    });
    console.log(output);
    return true;
  } catch (err: any) {
    if (err.code === 'ENOENT') {
      printError('nomik is not installed. Install from https://nomik.co');
      return false;
    }
    // nomik ran but returned non-zero — still print output
    if (err.stdout) console.log(err.stdout);
    if (err.stderr) console.error(err.stderr);
    return true;
  }
}

async function cmdCodebaseImpact(symbol: string): Promise<void> {
  if (!symbol) { printError('Usage: act codebase impact <symbol>'); process.exit(1); }
  await runNomik(['impact', symbol]);
}

async function cmdCodebaseRules(): Promise<void> {
  await runNomik(['rules']);
}

async function cmdCodebaseCommunities(): Promise<void> {
  await runNomik(['communities']);
}

async function cmdCodebaseOnboard(): Promise<void> {
  await runNomik(['onboard']);
}

// ─── /swarm and /nomik command surface (mirrors the TUI slash commands) ──────

const SWARM_ROLES = ['developer', 'frontend_dev', 'backend_dev', 'qa_engineer', 'researcher'];
const VALID_BACKENDS = ['act-agent', 'claude-code'];

async function cmdSwarm(args: string[]): Promise<void> {
  // args[0] is the subcommand: list, status, set, restart
  const sub = args[0] || 'list';

  if (sub === 'list' || sub === 'status') {
    const home = process.env.HOME || '';
    const cfgPath = `${home}/.act.json`;
    const fs = await import('fs');
    let agents: Record<string, any> = {};
    try {
      const raw = fs.readFileSync(cfgPath, 'utf-8');
      const parsed = JSON.parse(raw);
      agents = parsed.agents || {};
    } catch {
      printError(`Could not read ${cfgPath} — using defaults.`);
    }
    console.log('ROLE              BACKEND       MODEL');
    for (const role of SWARM_ROLES) {
      const cfg = agents[role] || {};
      const backend = cfg.backend || 'act-agent';
      const model = cfg.model || '(unset)';
      console.log(`${role.padEnd(17)} ${backend.padEnd(13)} ${model}`);
    }
    return;
  }

  if (sub === 'set') {
    const role = args[1];
    const backend = args[2];
    if (!role || !backend) {
      printError('Usage: act swarm set <role|all> <act-agent|claude-code>');
      process.exit(1);
    }
    if (!VALID_BACKENDS.includes(backend)) {
      printError(`Invalid backend "${backend}". Valid: ${VALID_BACKENDS.join(', ')}`);
      process.exit(1);
    }

    if (role === 'all') {
      let updated = 0;
      for (const r of SWARM_ROLES) {
        if (writeAgentBackend(r, backend)) updated++;
      }
      console.log(`Set backend=${backend} for ${updated} swarm role(s).`);
      console.log('Restart `act` for changes to take effect (or use /swarm all in the running TUI).');
      return;
    }

    if (!SWARM_ROLES.includes(role)) {
      printError(`backend selection only applies to Tier 2 swarm agents (${SWARM_ROLES.join(', ')}). "${role}" is not a swarm role.`);
      process.exit(1);
    }

    if (writeAgentBackend(role, backend)) {
      console.log(`Set ${role} backend = ${backend}.`);
      console.log('Restart `act` for changes to take effect (or use /swarm in the running TUI).');
    }
    return;
  }

  if (sub === 'restart') {
    printError('act swarm restart is only available inside the running TUI as /swarm restart.');
    process.exit(1);
  }

  printError(`Unknown swarm subcommand "${sub}". Try: list, set, status, restart`);
  process.exit(1);
}

function writeAgentBackend(role: string, backend: string): boolean {
  const fs = require('fs');
  const path = require('path');
  const home = process.env.HOME || '';
  const cfgPath = path.join(home, '.act.json');

  let data: any = {};
  try {
    const raw = fs.readFileSync(cfgPath, 'utf-8');
    data = JSON.parse(raw);
  } catch (err: any) {
    if (err.code !== 'ENOENT') {
      printError(`Failed to read ${cfgPath}: ${err.message}`);
      return false;
    }
  }

  if (!data.agents) data.agents = {};
  if (!data.agents[role]) data.agents[role] = {};
  data.agents[role].backend = backend;

  // Atomic write
  const tmp = cfgPath + '.tmp';
  fs.writeFileSync(tmp, JSON.stringify(data, null, 2) + '\n');
  fs.renameSync(tmp, cfgPath);
  return true;
}

async function cmdNomik(args: string[]): Promise<void> {
  const sub = args[0] || 'status';

  if (sub === 'status') {
    const ok = await runNomik(['status']);
    if (!ok) process.exit(1);
    return;
  }
  if (sub === 'enable') {
    console.log('Running initial Nomik scan...');
    const ok = await runNomik(['scan', './']);
    if (!ok) {
      printError('Nomik scan failed. Make sure Neo4j is running on localhost:7687.');
      process.exit(1);
    }
    console.log('Nomik enabled. The graph will refresh after task completions when the TUI is running.');
    return;
  }
  if (sub === 'disable') {
    console.log('Nomik is per-project. To disable for the current project use `/nomik disable` inside the running TUI.');
    return;
  }
  if (sub === 'rescan') {
    const ok = await runNomik(['scan:incremental']);
    if (!ok) process.exit(1);
    return;
  }
  if (sub === 'onboard') {
    await cmdCodebaseOnboard();
    return;
  }
  printError(`Unknown nomik subcommand "${sub}". Try: status, enable, disable, rescan, onboard`);
  process.exit(1);
}

async function cmdRegister(client: ACTClient, args: Record<string, any>): Promise<void> {
  const agentId = args['<agent-id>'] || getAgentId(args);
  const sessionId = args['--session-id'] || process.env.CLAUDE_SESSION_ID;

  if (!agentId) {
    printError('Error: agent-id is required');
    printError('Usage: act register <agent-id> [--session-id <id>]');
    process.exit(1);
  }

  const result = await client.registerAgent(agentId);
  if (!result.success) {
    if (result.conflict) {
      printError(`Agent ID "${agentId}" already registered. Use a different ID.`);
    } else {
      printError(result.error || 'Registration failed');
    }
    process.exit(1);
  }

  console.log(`Registered agent: ${agentId}`);

  // Write session identity file for stop hook
  if (sessionId) {
    const { mkdirSync, writeFileSync } = await import('fs');
    const { join } = await import('path');
    const home = process.env.HOME || process.env.USERPROFILE || '~';
    const sessionsDir = join(home, '.act', 'sessions');
    try {
      mkdirSync(sessionsDir, { recursive: true });
      writeFileSync(join(sessionsDir, sessionId), agentId, 'utf8');
      console.log(`Session identity written: ~/.act/sessions/${sessionId}`);
    } catch { /* non-fatal */ }
  }
}

async function cmdPvmSearch(client: ACTClient, args: Record<string, any>): Promise<void> {
  const query = args['<query>'];
  const limit = parseInt(args['--limit'] || '10', 10);

  if (!query) {
    printError('Error: query is required');
    printError('Usage: act pvm search "query" [--limit N]');
    process.exit(1);
  }

  // PVM is cross-project by design — past patterns from any project are
  // legitimate routing evidence for the current one.
  const results = await client.searchPVM(query, limit);
  if (results.length === 0) {
    console.log('No results.');
    return;
  }

  console.log(`PVM search: "${query}" — ${results.length} result(s)`);
  console.log();
  for (const r of results) {
    // Results from LocalEmbeddingVectorStore are { message: CoordinationMessage, similarity: number }.
    const msg = r.message && typeof r.message === 'object' ? r.message : r;
    const score = (r.similarity ?? r.score) !== undefined
      ? ` (score: ${(r.similarity ?? r.score).toFixed(3)})`
      : '';
    const agent = msg.agent || msg.metadata?.agent || '';
    const type = msg.type || msg.metadata?.type || '';
    const time = msg.timestamp ? new Date(msg.timestamp).toLocaleString() : '';
    const text = typeof msg.message === 'string' ? msg.message : (msg.text || msg.content || '');
    console.log(`  [${time}] ${agent} ${type}${score}`);
    console.log(`    ${text}`);
  }
}

async function cmdTaskRetry(client: ACTClient, args: Record<string, any>): Promise<void> {
  const taskId = args['<task-id>'];

  if (!taskId) {
    printError('Error: task-id is required');
    printError('Usage: act task retry <task-id>');
    process.exit(1);
  }

  const result = await client.retryTask(taskId);
  if (result.permanentlyFailed) {
    printError(`Task ${taskId} permanently failed — max retries exceeded.`);
    process.exit(1);
  }
  if (!result.success) {
    printError(result.error || 'Failed to retry task');
    process.exit(1);
  }

  console.log(`Task ${taskId} reset for retry (attempt ${result.task?.retryCount ?? '?'}/3)`);
}

async function cmdValidationQueue(client: ACTClient): Promise<void> {
  const tasks = await client.getValidationQueue();

  if (tasks.length === 0) {
    console.log('No tasks pending validation.');
    return;
  }

  console.log(`Validation queue (${tasks.length} task(s)):`);
  for (const t of tasks) {
    const title = t.title || t.description?.substring(0, 60) || t.id;
    const agent = t.assignedAgent || 'unassigned';
    console.log(`  - ${t.id} | ${title}`);
    console.log(`    agent: ${agent} | self-verification: ${t.metadata?.selfVerification ? 'yes' : 'no'}`);
  }
}

async function main(): Promise<void> {
  const commands: Record<string, string> = {
    register: 'Register agent with ACT server',
    context: 'Get agent brief and current task',
    'task complete': 'Mark a task as complete',
    'task progress': 'Update task progress percentage',
    'task retry': 'Retry a failed task (resets to pending)',
    'task submit-for-validation': 'Submit completed task for Assurance review',
    'brief update': 'Update agent brief for a project',
    'pvm search': 'Search coordination memory (PVM)',
    'validation queue': 'List tasks pending Assurance validation',
    'files claim': 'Claim files for exclusive editing',
    'files release': 'Release file locks',
    message: 'Send a message or read inbox (no text = read)',
    log: 'View recent coordination log entries',
    'graph task': 'Show task dependency tree',
    'graph unverified': 'List completed tasks not yet validated',
    'graph conflicts': 'Show file lock conflicts',
    status: 'Show system status overview',
    'codebase impact': 'Analyze impact of changing a symbol (via Nomik)',
    'codebase rules': 'Check architecture rules (via Nomik)',
    'codebase communities': 'Show functional code communities (via Nomik)',
    'codebase onboard': 'Generate codebase overview (via Nomik)',
  };

  const args = process.argv.slice(2);

  if (args.length === 0 || args[0] === '--help' || args[0] === '-h') {
    console.log('ACT CLI - Agent interface to ACT coordination layer');
    console.log('');
    console.log('Usage: act <command> [options]');
    console.log('');
    console.log('Commands:');
    for (const [cmd, desc] of Object.entries(commands)) {
      console.log(`  ${cmd.padEnd(18)} ${desc}`);
    }
    console.log('');
    console.log('Options:');
    console.log('  --agent-id <id>        Agent ID (or set ACT_AGENT_ID env var)');
    console.log('  --project <name>       Project name');
    console.log('  --session <text>       Brief content text');
    console.log('  --session-id <id>      Claude session ID (or set CLAUDE_SESSION_ID)');
    console.log('  --task-id <id>         Task ID');
    console.log('  --result <json>        Task result as JSON string');
    console.log('  --percent <n>          Progress percentage (0-100)');
    console.log('  --limit <n>            Number of PVM search results (default: 10)');
    console.log('  --tail <n>             Number of log entries to show (default: 20)');
    console.log('');
    console.log('Environment:');
    console.log('  ACT_SERVER_URL         ACT server URL (default: http://localhost:8080)');
    console.log('  ACT_AGENT_ID           Default agent ID for commands');
    return;
  }

  const command = args[0];
  const subcommand = args[1];

  // Parse remaining args
  let parsed: { values: Record<string, any> };
  try {
    const remainingArgs = args.slice(2);
    parsed = parseArgs({
      options: {
        'agent-id': { type: 'string' },
        'project': { type: 'string' },
        'session': { type: 'string' },
        'task-id': { type: 'string' },
        'result': { type: 'string' },
        'percent': { type: 'string' },
        'tail': { type: 'string' }
      },
      strict: false,
      allowPositionals: true
    });
  } catch {
    // Fall back to manual parsing
    parsed = { values: {} };
    const remainingArgs = args.slice(2);
    for (let i = 0; i < remainingArgs.length; i++) {
      const arg = remainingArgs[i];
      if (arg.startsWith('--')) {
        const key = arg.slice(2);
        parsed.values[key] = remainingArgs[i + 1] || '';
        i++;
      } else if (!arg.startsWith('-')) {
        const pos = parsed.values['<paths>'] || [];
        if (Array.isArray(pos)) {
          pos.push(arg);
        } else {
          parsed.values['<paths>'] = [arg];
        }
      }
    }
  }

  const client = new ACTClient(DEFAULT_SERVER_URL);

  try {
    if (command === 'register') {
      // act register <agent-id> [--session-id <id>]
      const regArgs = parseArgs({
        options: {
          'session-id': { type: 'string' }
        },
        strict: false,
        allowPositionals: true
      });
      await cmdRegister(client, { '<agent-id>': subcommand, ...regArgs.values });
    } else if (command === 'context') {
      const contextArgs = parseArgs({
        options: {
          'agent-id': { type: 'string' },
          'project': { type: 'string', default: '' }
        },
        strict: false,
        allowPositionals: true
      });
      // act context <agent-id> --project <name> → agent-id is args[1] (subcommand position)
      await cmdContext(client, { 'agent-id': subcommand || contextArgs.values['agent-id'] || process.env.ACT_AGENT_ID, '--project': contextArgs.values['project'] });
    } else if (command === 'task' && subcommand === 'complete') {
      const taskArgs = parseArgs({
        options: {
          'agent-id': { type: 'string' },
          'result': { type: 'string' }
        },
        strict: false,
        allowPositionals: true
      });
      await cmdTaskComplete(client, { '<task-id>': taskArgs.values['<0>'], ...taskArgs.values });
    } else if (command === 'task' && subcommand === 'progress') {
      const progArgs = parseArgs({
        options: {
          'agent-id': { type: 'string' },
          'percent': { type: 'string' }
        },
        strict: false,
        allowPositionals: true
      });
      await cmdTaskProgress(client, { '<task-id>': progArgs.values['<0>'], ...progArgs.values });
    } else if (command === 'brief' && subcommand === 'update') {
      const briefArgs = parseArgs({
        options: {
          'project': { type: 'string' },
          'session': { type: 'string' }
        },
        strict: false,
        allowPositionals: true
      });
      await cmdBriefUpdate(client, { '<agent-id>': briefArgs.values['<0>'], ...briefArgs.values });
    } else if (command === 'task' && subcommand === 'retry') {
      // act task retry <task-id>
      const retryTaskId = args[2] || '';
      await cmdTaskRetry(client, { '<task-id>': retryTaskId });
    } else if (command === 'task' && subcommand === 'submit-for-validation') {
      const valArgs = parseArgs({
        options: {
          'agent-id': { type: 'string' },
          'task-id': { type: 'string' },
          'result': { type: 'string' }
        },
        strict: false,
        allowPositionals: true
      });
      await cmdTaskSubmitValidation(client, { '<task-id>': valArgs.values['task-id'], ...valArgs.values });
    } else if (command === 'files' && subcommand === 'claim') {
      const claimArgs = parseArgs({
        options: {
          'agent-id': { type: 'string' },
          'task-id': { type: 'string' }
        },
        strict: false,
        allowPositionals: true
      });
      await cmdFilesClaim(client, { '<paths>': claimArgs.values.positionals, ...claimArgs.values });
    } else if (command === 'files' && subcommand === 'release') {
      const releaseArgs = parseArgs({
        options: {
          'agent-id': { type: 'string' },
          'task-id': { type: 'string' }
        },
        strict: false,
        allowPositionals: true
      });
      await cmdFilesRelease(client, { '<paths>': releaseArgs.values.positionals, ...releaseArgs.values });
    } else if (command === 'pvm' && subcommand === 'search') {
      // act pvm search "query" [--limit N]
      const pvmArgs = parseArgs({
        options: {
          'limit': { type: 'string' }
        },
        strict: false,
        allowPositionals: true
      });
      // Query is everything after 'pvm search' that isn't a flag
      const queryParts: string[] = [];
      for (let i = 2; i < args.length; i++) {
        if (args[i] === '--limit' && args[i + 1]) { i++; continue; }
        if (!args[i].startsWith('--')) queryParts.push(args[i]);
      }
      await cmdPvmSearch(client, { '<query>': queryParts.join(' ') || undefined, ...pvmArgs.values });
    } else if (command === 'validation' && subcommand === 'queue') {
      await cmdValidationQueue(client);
    } else if (command === 'message') {
      // act message [text...] — no text = read inbox
      // Collect everything after 'message' that isn't a flag as text
      const msgParts: string[] = [];
      let msgAgentId: string | undefined;
      for (let i = 1; i < args.length; i++) {
        if (args[i] === '--agent-id' && args[i + 1]) { msgAgentId = args[++i]; }
        else if (!args[i].startsWith('--')) { msgParts.push(args[i]); }
      }
      const text = msgParts.join(' ');
      await cmdMessage(client, { '<text>': text || undefined, 'agent-id': msgAgentId });
    } else if (command === 'log') {
      const logArgs = parseArgs({
        options: {
          'tail': { type: 'string' }
        },
        strict: false
      });
      await cmdLog(client, logArgs.values);
    } else if (command === 'graph' && subcommand === 'task') {
      // act graph task <task-id> — task-id is args[2] (after 'graph' 'task')
      const taskId = args[2] || '';
      await cmdGraphTask(client, taskId);
    } else if (command === 'graph' && subcommand === 'unverified') {
      await cmdGraphUnverified(client);
    } else if (command === 'graph' && subcommand === 'conflicts') {
      await cmdGraphConflicts(client);
    } else if (command === 'status') {
      await cmdStatus(client);
    } else if (command === 'codebase' && subcommand === 'impact') {
      await cmdCodebaseImpact(args[2] || '');
    } else if (command === 'codebase' && subcommand === 'rules') {
      await cmdCodebaseRules();
    } else if (command === 'codebase' && subcommand === 'communities') {
      await cmdCodebaseCommunities();
    } else if (command === 'codebase' && subcommand === 'onboard') {
      await cmdCodebaseOnboard();
    } else if (command === 'swarm') {
      await cmdSwarm(args.slice(1));
    } else if (command === 'nomik') {
      await cmdNomik(args.slice(1));
    } else {
      printError(`Unknown command: ${command}${subcommand ? ' ' + subcommand : ''}`);
      printError('Run "act --help" for usage information');
      process.exit(1);
    }
  } catch (error: any) {
    printError(`Error: ${error.message}`);
    process.exit(1);
  }
}

main();
