/**
 * Observer monitoring logic for NesTTY.
 *
 * The Observer is a Tier 1 role that monitors coordination state and surfaces
 * problems. It does NOT make decisions — it reports findings to the Planner.
 *
 * This module provides:
 * - buildStatusSnapshot(): polls ACT server for current state → structured summary
 * - detectAnomalies(): heuristic checks for stuck tasks, stale locks, idle agents
 *
 * The orchestrator calls these periodically and injects results into the Observer's turn.
 */

/** Snapshot of coordination state for the Observer to analyze */
export interface StatusSnapshot {
  tasks: TaskSummary[];
  agents: AgentSummary[];
  fileLocks: FileLockSummary[];
  recentEvents: EventSummary[];
  timestamp: string;
}

export interface TaskSummary {
  id: string;
  title?: string;
  status: string;
  assignedAgent?: string;
  priority?: string;
  ageMinutes: number;
}

export interface AgentSummary {
  id: string;
  status: string;
  currentTask?: string;
  lastSeenMinutes: number;
}

export interface FileLockSummary {
  file: string;
  agentId: string;
  taskId: string;
  ageMinutes: number;
}

export interface EventSummary {
  type: string;
  agent: string;
  message: string;
  minutesAgo: number;
}

/** Anomaly detected by the Observer heuristics */
export interface Anomaly {
  severity: 'info' | 'warning' | 'critical';
  category: 'stuck_task' | 'stale_lock' | 'idle_agent' | 'unvalidated' | 'bottleneck';
  message: string;
  taskId?: string;
  agentId?: string;
}

const STUCK_TASK_MINUTES = 30;
const STALE_LOCK_MINUTES = 20;
const IDLE_AGENT_MINUTES = 10;

/**
 * Build a status snapshot by polling ACT server endpoints.
 * Returns null if server is unreachable.
 */
export async function buildStatusSnapshot(serverUrl: string): Promise<StatusSnapshot | null> {
  try {
    const now = Date.now();
    const [tasksRes, agentsRes, locksRes, logRes] = await Promise.all([
      fetchJSON(`${serverUrl}/api/tasks`),
      fetchJSON(`${serverUrl}/api/agents`),
      fetchJSON(`${serverUrl}/api/files/locks`),
      fetchJSON(`${serverUrl}/api/log?limit=20`),
    ]);

    const tasks: TaskSummary[] = (Array.isArray(tasksRes) ? tasksRes : tasksRes?.tasks || []).map((t: any) => ({
      id: t.id,
      title: t.title,
      status: t.status,
      assignedAgent: t.assignedAgent,
      priority: t.priority,
      ageMinutes: t.createdAt ? Math.round((now - new Date(t.createdAt).getTime()) / 60000) : 0,
    }));

    const agents: AgentSummary[] = (Array.isArray(agentsRes) ? agentsRes : agentsRes?.agents || []).map((a: any) => ({
      id: a.id,
      status: a.status,
      currentTask: a.currentTask,
      lastSeenMinutes: a.lastSeen ? Math.round((now - new Date(a.lastSeen).getTime()) / 60000) : 999,
    }));

    const fileLocks: FileLockSummary[] = (locksRes?.locks || []).map((l: any) => ({
      file: l.filePath || l.file,
      agentId: l.agentId,
      taskId: l.taskId,
      ageMinutes: l.lockedAt ? Math.round((now - new Date(l.lockedAt).getTime()) / 60000) : 0,
    }));

    const recentEvents: EventSummary[] = (logRes?.events || []).map((e: any) => ({
      type: e.type,
      agent: e.agent,
      message: e.message?.substring(0, 120),
      minutesAgo: e.timestamp ? Math.round((now - new Date(e.timestamp).getTime()) / 60000) : 0,
    }));

    return { tasks, agents, fileLocks, recentEvents, timestamp: new Date().toISOString() };
  } catch {
    return null;
  }
}

/**
 * Detect anomalies from a status snapshot.
 * Returns a list of issues sorted by severity.
 */
export function detectAnomalies(snapshot: StatusSnapshot): Anomaly[] {
  const anomalies: Anomaly[] = [];

  // Stuck tasks: assigned or in_progress for too long
  for (const task of snapshot.tasks) {
    if ((task.status === 'assigned' || task.status === 'in_progress') && task.ageMinutes > STUCK_TASK_MINUTES) {
      anomalies.push({
        severity: task.ageMinutes > STUCK_TASK_MINUTES * 2 ? 'critical' : 'warning',
        category: 'stuck_task',
        message: `Task "${task.title || task.id}" has been ${task.status} for ${task.ageMinutes}min (assigned to ${task.assignedAgent || 'nobody'})`,
        taskId: task.id,
        agentId: task.assignedAgent,
      });
    }
  }

  // Unvalidated completed tasks
  const completedNotValidated = snapshot.tasks.filter(t => t.status === 'completed');
  if (completedNotValidated.length > 0) {
    anomalies.push({
      severity: completedNotValidated.length > 3 ? 'warning' : 'info',
      category: 'unvalidated',
      message: `${completedNotValidated.length} completed task(s) not yet submitted for validation: ${completedNotValidated.map(t => t.title || t.id).join(', ')}`,
    });
  }

  // Stale file locks
  for (const lock of snapshot.fileLocks) {
    if (lock.ageMinutes > STALE_LOCK_MINUTES) {
      anomalies.push({
        severity: 'warning',
        category: 'stale_lock',
        message: `File "${lock.file}" locked by ${lock.agentId} for ${lock.ageMinutes}min (task: ${lock.taskId})`,
        agentId: lock.agentId,
        taskId: lock.taskId,
      });
    }

    // Lock held by offline agent
    const lockAgent = snapshot.agents.find(a => a.id === lock.agentId);
    if (lockAgent && lockAgent.status === 'offline') {
      anomalies.push({
        severity: 'critical',
        category: 'stale_lock',
        message: `File "${lock.file}" locked by OFFLINE agent ${lock.agentId} — needs manual release`,
        agentId: lock.agentId,
      });
    }
  }

  // Idle agents (online but no task, while tasks are pending)
  const pendingTasks = snapshot.tasks.filter(t => t.status === 'pending');
  if (pendingTasks.length > 0) {
    for (const agent of snapshot.agents) {
      if (agent.status === 'online' && !agent.currentTask) {
        anomalies.push({
          severity: 'warning',
          category: 'idle_agent',
          message: `Agent ${agent.id} is idle while ${pendingTasks.length} task(s) are pending`,
          agentId: agent.id,
        });
      }
    }
  }

  // Bottleneck: many tasks assigned to one agent while others are idle
  const agentTaskCounts = new Map<string, number>();
  for (const task of snapshot.tasks) {
    if (task.assignedAgent && (task.status === 'assigned' || task.status === 'in_progress')) {
      agentTaskCounts.set(task.assignedAgent, (agentTaskCounts.get(task.assignedAgent) || 0) + 1);
    }
  }
  for (const [agentId, count] of agentTaskCounts) {
    if (count >= 3) {
      anomalies.push({
        severity: 'warning',
        category: 'bottleneck',
        message: `Agent ${agentId} has ${count} active tasks — possible bottleneck`,
        agentId,
      });
    }
  }

  // Sort: critical first, then warning, then info
  const severityOrder = { critical: 0, warning: 1, info: 2 };
  anomalies.sort((a, b) => severityOrder[a.severity] - severityOrder[b.severity]);

  return anomalies;
}

/**
 * Build a monitoring prompt for the Observer agent.
 * Includes status snapshot + detected anomalies.
 */
export function buildObserverPrompt(snapshot: StatusSnapshot, anomalies: Anomaly[]): string {
  const lines: string[] = [
    `OBSERVER STATUS UPDATE — ${snapshot.timestamp}`,
    '',
  ];

  // Anomalies first (most important)
  if (anomalies.length > 0) {
    lines.push(`## Detected Issues (${anomalies.length})`);
    for (const a of anomalies) {
      const icon = a.severity === 'critical' ? 'CRITICAL' : a.severity === 'warning' ? 'WARNING' : 'INFO';
      lines.push(`[${icon}] ${a.message}`);
    }
    lines.push('');
  } else {
    lines.push('## No Issues Detected');
    lines.push('All tasks progressing normally.');
    lines.push('');
  }

  // Task board
  const tasksByStatus = new Map<string, TaskSummary[]>();
  for (const t of snapshot.tasks) {
    const list = tasksByStatus.get(t.status) || [];
    list.push(t);
    tasksByStatus.set(t.status, list);
  }
  lines.push('## Task Board');
  for (const [status, tasks] of tasksByStatus) {
    lines.push(`${status}: ${tasks.length} task(s)`);
    for (const t of tasks.slice(0, 5)) {
      lines.push(`  - ${t.title || t.id} (${t.assignedAgent || 'unassigned'}, ${t.ageMinutes}min)`);
    }
    if (tasks.length > 5) lines.push(`  ... and ${tasks.length - 5} more`);
  }
  lines.push('');

  // Active file locks
  if (snapshot.fileLocks.length > 0) {
    lines.push(`## File Locks (${snapshot.fileLocks.length})`);
    for (const l of snapshot.fileLocks) {
      lines.push(`  ${l.file} — ${l.agentId} (${l.ageMinutes}min)`);
    }
    lines.push('');
  }

  // Agent status
  lines.push('## Agents');
  for (const a of snapshot.agents) {
    lines.push(`  ${a.id}: ${a.status}${a.currentTask ? ` (task: ${a.currentTask})` : ''} — last seen ${a.lastSeenMinutes}min ago`);
  }
  lines.push('');

  // Instructions
  lines.push('## Your Task');
  if (anomalies.length > 0) {
    lines.push('Report these issues to @planner with your assessment and suggested actions.');
    lines.push('Be specific: which tasks, which agents, what should change.');
  } else {
    lines.push('No action needed. Acknowledge the status if Planner asked, otherwise stay quiet.');
  }

  return lines.join('\n');
}

/** Simple fetch wrapper that returns parsed JSON */
async function fetchJSON(url: string): Promise<any> {
  const res = await fetch(url);
  return res.json();
}
