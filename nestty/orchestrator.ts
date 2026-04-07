/**
 * NesTTY Orchestrator — core turn loop for multi-agent conversation.
 *
 * Decides who speaks next, injects context into each agent's PTY,
 * routes responses to the display, and enforces the turn protocol
 * to prevent infinite response loops.
 *
 * Architecture: orchestrator reads from a message queue (populated by
 * PTY manager parsing agent stdout) and writes to agent PTYs via
 * the PTY manager. The display layer subscribes to orchestrator events.
 */

import { EventEmitter } from 'events';
import type {
  NestTTYMessage,
  NestTTYRole,
  ConversationTurn,
  OrchestratorEvent,
  SessionAgent,
} from './types.js';
import { buildValidationPrompt, buildGapAnalysisPrompt, parseValidationResponse, type ValidationVerdict } from './assurance.js';
import { buildSynthesisPrompt, parseSynthesisResponse, type AssemblyState, type ValidatedOutput } from './synthesizer.js';
import { buildStatusSnapshot, detectAnomalies, buildObserverPrompt } from './observer.js';
import { buildKickoffPrompt, buildHumanRequestPrompt, parseCreateTaskDirectives } from './planner.js';

/** Minimal PTY manager interface — decoupled from node-pty implementation */
export interface PTYManagerInterface {
  writeTurn(role: string, input: string): void;
  kill(role: string): void;
  killAll(): void;
}

/** Configuration for the orchestrator */
export interface OrchestratorConfig {
  projectName: string;
  turnTimeoutMs?: number;    // default 60000 (60s)
  maxTurnReplies?: number;   // default 1
  roles?: NestTTYRole[];     // default: all four
  serverUrl?: string;        // default http://localhost:8080
  observerIntervalMs?: number; // default 120000 (2min)
}

const DEFAULT_TURN_TIMEOUT = 60_000;
const DEFAULT_MAX_REPLIES = 1;
const DEFAULT_OBSERVER_INTERVAL = 120_000; // 2 minutes

/**
 * Turn routing rules:
 * - Planner message → Observer responds (monitoring acknowledgment)
 * - Observer message → Planner responds (decision needed)
 * - Assurance message → Planner responds (validation result)
 * - QA message → Planner responds (synthesis status)
 * - Human input → Planner responds (always)
 * - @mention overrides default routing
 */
const DEFAULT_RESPONDER: Record<string, NestTTYRole> = {
  planner: 'observer',
  observer: 'planner',
  assurance: 'planner',
  qa: 'planner',
  human: 'planner',
};

export class Orchestrator extends EventEmitter {
  private agents: Map<string, SessionAgent> = new Map();
  private transcript: NestTTYMessage[] = [];
  private currentTurn: ConversationTurn | null = null;
  private turnTimer: ReturnType<typeof setTimeout> | null = null;
  private ptyManager: PTYManagerInterface | null = null;
  private config: Required<OrchestratorConfig> & { serverUrl: string; observerIntervalMs: number };
  private running = false;
  private observerTimer: ReturnType<typeof setInterval> | null = null;
  private consecutiveAutoTurns = 0;
  private static readonly MAX_AUTO_TURNS = 4; // max exchanges before waiting for HITL
  private assemblyState: AssemblyState;
  private pendingValidationTaskId: string | null = null; // task currently being validated by Assurance

  constructor(config: OrchestratorConfig) {
    super();
    this.config = {
      projectName: config.projectName,
      turnTimeoutMs: config.turnTimeoutMs ?? DEFAULT_TURN_TIMEOUT,
      maxTurnReplies: config.maxTurnReplies ?? DEFAULT_MAX_REPLIES,
      roles: config.roles ?? ['planner', 'observer', 'assurance', 'qa'],
      serverUrl: config.serverUrl ?? 'http://localhost:8080',
      observerIntervalMs: config.observerIntervalMs ?? DEFAULT_OBSERVER_INTERVAL,
    };
    this.assemblyState = {
      projectName: config.projectName,
      queue: [],
      assembled: [],
      deliverable: null,
    };
  }

  /** Wire up the PTY manager (called after both are constructed) */
  setPTYManager(mgr: PTYManagerInterface): void {
    this.ptyManager = mgr;
  }

  /** Register an agent role in the session */
  addAgent(role: NestTTYRole): void {
    this.agents.set(role, {
      role,
      status: 'spawning',
      messageHistory: [],
    });
  }

  /** Handle incoming message from an agent's PTY (called by PTY manager) */
  handleMessage(msg: NestTTYMessage): void {
    switch (msg.type) {
      case 'ready':
        this.handleReady(msg);
        break;
      case 'message':
        this.handleAgentMessage(msg);
        break;
      case 'error':
        this.handleError(msg);
        break;
      case 'exit':
        this.handleExit(msg);
        break;
    }
  }

  /** Inject human input from the $> prompt */
  handleHumanInput(text: string): void {
    const msg: NestTTYMessage = {
      role: 'human',
      type: 'message',
      content: text,
      time: new Date().toISOString(),
    };

    this.transcript.push(msg);
    this.emitEvent({ type: 'hitl', content: text, timestamp: msg.time });

    // Human input resets auto-turn counter
    this.consecutiveAutoTurns = 0;
    this.closeTurn();

    // Check for @mention override — route to specific agent
    const mentionMatch = text.match(/@(planner|observer|assurance|qa)\b/i);
    if (mentionMatch) {
      const target = mentionMatch[1].toLowerCase() as NestTTYRole;
      if (this.agents.get(target)?.status !== 'dead') {
        this.openTurn('human', [target]);
        this.injectContext(target, `[Human]: ${text}`);
        return;
      }
    }

    // Default: route to Planner with project context
    const wrappedInput = buildHumanRequestPrompt(text);
    this.openTurn('human', ['planner']);
    this.injectContext('planner', wrappedInput);
  }

  /** Start the orchestrator — called after all agents are spawned */
  start(): void {
    this.running = true;
    this.startObserverLoop();
  }

  /** Graceful shutdown */
  shutdown(): void {
    this.running = false;
    this.closeTurn();
    if (this.observerTimer) {
      clearInterval(this.observerTimer);
      this.observerTimer = null;
    }
    this.ptyManager?.killAll();
    this.emitEvent({ type: 'shutdown', timestamp: new Date().toISOString() });
  }

  /** Get full transcript */
  getTranscript(): NestTTYMessage[] {
    return [...this.transcript];
  }

  /** Check if all configured agents are ready */
  allReady(): boolean {
    for (const role of this.config.roles) {
      const agent = this.agents.get(role);
      if (!agent || agent.status !== 'ready') return false;
    }
    return true;
  }

  /**
   * Trigger an immediate Observer check (e.g., on-demand from Planner or Human).
   */
  async triggerObserverCheck(): Promise<void> {
    await this.runObserverCheck();
  }

  /**
   * Route a completed task to Assurance for validation.
   * Called when a task is submitted for validation (server emits event).
   * Builds a structured validation prompt from SNLP success criteria.
   */
  routeToAssurance(task: {
    id: string;
    title?: string;
    description: string;
    successCriteria: string[];
    result: string;
    selfVerification?: string;
  }): void {
    const assurance = this.agents.get('assurance');
    if (!assurance || assurance.status === 'dead') return;

    this.pendingValidationTaskId = task.id;
    const prompt = buildValidationPrompt(task);
    this.closeTurn();
    this.openTurn('system', ['assurance']);
    this.injectContext('assurance', prompt);
  }

  /**
   * Route a failed validation back to the original agent.
   * Builds a gap analysis prompt from the verdict.
   * The agent's role is passed so we can target the right PTY.
   */
  routeGapAnalysis(agentRole: NestTTYRole, verdict: ValidationVerdict): void {
    const agent = this.agents.get(agentRole);
    if (!agent || agent.status === 'dead') return;

    const prompt = buildGapAnalysisPrompt(verdict);
    this.closeTurn();
    this.openTurn('assurance', [agentRole]);
    this.injectContext(agentRole, prompt);
  }

  /**
   * Route a validated task output to QA/Synthesizer for assembly.
   * Called when a task passes Assurance validation.
   */
  routeToSynthesizer(output: ValidatedOutput): void {
    const qa = this.agents.get('qa');
    if (!qa || qa.status === 'dead') return;

    // Add to assembly queue
    this.assemblyState.queue.push(output);

    const prompt = buildSynthesisPrompt(this.assemblyState);
    this.closeTurn();
    this.openTurn('system', ['qa']);
    this.injectContext('qa', prompt);
  }

  /**
   * Mark an output as assembled (called after QA reports SYNTHESIS_COMPLETE for it).
   */
  markAssembled(taskId: string): void {
    const idx = this.assemblyState.queue.findIndex(o => o.taskId === taskId);
    if (idx >= 0) {
      const [output] = this.assemblyState.queue.splice(idx, 1);
      this.assemblyState.assembled.push(output);
    }
  }

  /** Get current assembly state (for status display) */
  getAssemblyState(): AssemblyState {
    return { ...this.assemblyState };
  }

  // ─── Server Communication ────────────────────────────────────────────────

  /** Detect CREATE_TASK directives in Planner's response and POST to server */
  private async detectAndCreateTasks(content: string): Promise<void> {
    const tasks = parseCreateTaskDirectives(content);
    for (const taskDef of tasks) {
      await this.createTaskOnServer(taskDef);
    }
  }

  /** POST a single task to the ACT server */
  private async createTaskOnServer(taskDef: any): Promise<void> {
    try {
      const res = await fetch(`${this.config.serverUrl}/api/tasks`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: taskDef.title || taskDef.name,
          description: taskDef.description || '',
          requiredCapabilities: taskDef.requiredCapabilities || taskDef.capabilities || [],
          priority: taskDef.priority || 'medium',
          dependencies: taskDef.dependencies || [],
          metadata: { projectName: this.config.projectName, createdBy: 'planner' },
        }),
      });
      const data = await res.json() as any;
      if (data.task?.id) {
        this.emitEvent({
          type: 'turn_message', role: 'system',
          content: `Task created: ${taskDef.title || taskDef.name} (${data.task.id.substring(0, 8)})`,
          timestamp: new Date().toISOString(),
        });
      }
    } catch { /* server unreachable */ }
  }

  /** Send gap analysis to the swarm agent that built the task, via ACT messaging */
  private async sendGapToSwarmAgent(taskId: string, verdict: ValidationVerdict): Promise<void> {
    try {
      // Fetch task to find which agent was assigned
      const res = await fetch(`${this.config.serverUrl}/api/tasks/${taskId}`);
      const data = await res.json() as any;
      const agentId = data.task?.assignedAgent;
      if (!agentId) return;

      const gapPrompt = buildGapAnalysisPrompt(verdict);
      await fetch(`${this.config.serverUrl}/api/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          sender: 'assurance',
          message: `@${agentId} ${gapPrompt}`,
        }),
      });
    } catch {
      // Best effort — gap is also in the verdict metadata on the task
    }
  }

  /** POST validation verdict to ACT server to update task status */
  private async submitVerdict(taskId: string, verdict: ValidationVerdict): Promise<void> {
    try {
      await fetch(`${this.config.serverUrl}/api/tasks/${taskId}/validation-verdict`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          agentId: 'assurance',
          passed: verdict.passed,
          score: verdict.overallScore,
          criteriaResults: verdict.criteriaResults,
          gaps: verdict.gaps,
          feedback: verdict.feedback,
        }),
      });
    } catch {
      // Server unreachable — verdict logged in transcript, can be retried
    }
  }

  // ─── Observer Monitoring ──────────────────────────────────────────────────

  /** Start the periodic Observer monitoring loop */
  private startObserverLoop(): void {
    // Only run if Observer role is configured
    if (!this.config.roles.includes('observer')) return;

    this.observerTimer = setInterval(() => {
      this.runObserverCheck().catch(() => {});
    }, this.config.observerIntervalMs);
  }

  /** Run a single Observer monitoring check */
  private async runObserverCheck(): Promise<void> {
    if (!this.running) return;

    const observer = this.agents.get('observer');
    if (!observer || observer.status === 'dead') return;

    // Don't interrupt an active turn
    if (this.currentTurn && this.currentTurn.allowedResponders.includes('observer')) return;

    const snapshot = await buildStatusSnapshot(this.config.serverUrl);
    if (!snapshot) return;

    const anomalies = detectAnomalies(snapshot);

    // Only inject if there are issues to report (don't spam the Observer when idle)
    if (anomalies.length === 0) return;

    const prompt = buildObserverPrompt(snapshot, anomalies);
    this.closeTurn();
    this.openTurn('system', ['observer']);
    this.injectContext('observer', prompt);
  }

  // ─── Internal ────────────────────────────────────────────────────────────

  private handleReady(msg: NestTTYMessage): void {
    const agent = this.agents.get(msg.role);
    if (agent) {
      agent.status = 'ready';
      this.emitEvent({ type: 'agent_ready', role: msg.role, timestamp: msg.time });

      // Once all agents are ready, kick off the session
      if (this.allReady() && this.running && !this.currentTurn) {
        this.kickoff();
      }
    }
  }

  private handleAgentMessage(msg: NestTTYMessage): void {
    const agent = this.agents.get(msg.role);
    if (!agent) return;

    // Record in transcript and agent history
    this.transcript.push(msg);
    agent.messageHistory.push(msg);
    agent.status = 'ready';

    this.emitEvent({
      type: 'turn_message',
      role: msg.role,
      content: msg.content,
      timestamp: msg.time,
    });

    // Planner task creation — detect CREATE_TASK directives and POST to server
    if (msg.role === 'planner') {
      this.detectAndCreateTasks(msg.content).catch(() => {});
    }

    // Assurance response handling — detect validation verdicts and POST to server
    if (msg.role === 'assurance' && this.pendingValidationTaskId) {
      const taskId = this.pendingValidationTaskId;
      const verdict = parseValidationResponse(taskId, msg.content);
      if (verdict) {
        this.pendingValidationTaskId = null;
        // POST verdict to ACT server to update task status
        this.submitVerdict(taskId, verdict).catch(() => {});

        if (verdict.passed) {
          this.emitEvent({ type: 'turn_message', role: 'system',
            content: `Validation PASSED (${verdict.overallScore}/100) — task ${taskId.substring(0, 8)}`,
            timestamp: new Date().toISOString() });
          // QA will pick up validated tasks via polling
        } else {
          // Send gap analysis to the swarm agent's inbox so it knows what to fix
          this.sendGapToSwarmAgent(taskId, verdict).catch(() => {});
          this.closeTurn();
          this.openTurn('assurance', ['planner']);
          this.injectContext('planner',
            `[Assurance]: Validation FAILED for task ${taskId.substring(0, 8)} (${verdict.overallScore}/100). ` +
            `Gaps: ${verdict.gaps || 'see criteria results'}. ` +
            `Gap analysis sent to the swarm agent's inbox. Task returned to 'assigned' for rework.`);
          return;
        }
      }
    }

    // QA/Synthesizer response handling — detect synthesis outcomes
    if (msg.role === 'qa') {
      const result = parseSynthesisResponse(msg.content);
      if (result.type === 'complete') {
        // Mark all queued items as assembled
        for (const o of this.assemblyState.queue) {
          this.assemblyState.assembled.push(o);
        }
        this.assemblyState.queue = [];
        this.assemblyState.deliverable = result.summary || null;
        // Notify Planner that synthesis is complete
        this.closeTurn();
        this.openTurn('qa', ['planner']);
        this.injectContext('planner', `[QA/Synthesizer]: Synthesis complete. ${result.summary || ''}`);
        return;
      }
      if (result.type === 'need_clarification' && result.targetAgent) {
        // Route clarification question — QA needs to ask a swarm agent
        // This goes through Planner who can spawn the swarm agent via runner
        this.closeTurn();
        this.openTurn('qa', ['planner']);
        this.injectContext('planner',
          `[QA/Synthesizer]: Needs clarification from ${result.targetAgent}: ${result.question || ''}\n` +
          `Please coordinate with the swarm to get this answered.`);
        return;
      }
    }

    // Standard turn routing
    if (this.currentTurn) {
      if (this.currentTurn.allowedResponders.includes(msg.role)) {
        this.currentTurn.repliesSoFar++;
      }

      // Close turn if max replies reached
      if (this.currentTurn.repliesSoFar >= this.currentTurn.maxReplies) {
        this.closeTurn();
        this.decideNextTurn(msg);
      }
    } else {
      // No active turn — this message opens one
      this.decideNextTurn(msg);
    }
  }

  private handleError(msg: NestTTYMessage): void {
    this.emitEvent({
      type: 'error',
      role: msg.role,
      content: msg.content,
      timestamp: msg.time,
    });
  }

  private handleExit(msg: NestTTYMessage): void {
    const agent = this.agents.get(msg.role);
    if (agent) {
      agent.status = 'dead';
    }
  }

  /** Kick off the session — Planner speaks first */
  private kickoff(): void {
    const kickoffPrompt = buildKickoffPrompt(this.config.projectName, this.config.roles);
    this.openTurn('system', ['planner']);
    this.injectContext('planner', kickoffPrompt);
  }

  /** Decide who speaks next based on who just spoke */
  private decideNextTurn(lastMsg: NestTTYMessage): void {
    // Guard: prevent infinite ping-pong between agents
    this.consecutiveAutoTurns++;
    if (this.consecutiveAutoTurns > Orchestrator.MAX_AUTO_TURNS) {
      // Conversation has gone back and forth enough — wait for human input or external event
      this.emitEvent({
        type: 'turn_close',
        content: 'Paused: waiting for human input ($> prompt)',
        timestamp: new Date().toISOString(),
      });
      return;
    }

    // Check for @mention override
    const mentionMatch = lastMsg.content.match(/@(planner|observer|assurance|qa)\b/i);
    if (mentionMatch) {
      const target = mentionMatch[1].toLowerCase() as NestTTYRole;
      if (this.agents.get(target)?.status === 'ready') {
        this.openTurn(lastMsg.role, [target]);
        this.injectContext(target, `[${capitalize(lastMsg.role)}]: ${lastMsg.content}`);
        return;
      }
    }

    // Default routing
    const defaultResponder = DEFAULT_RESPONDER[lastMsg.role];
    if (defaultResponder && this.agents.get(defaultResponder)?.status === 'ready') {
      this.openTurn(lastMsg.role, [defaultResponder]);
      this.injectContext(defaultResponder, `[${capitalize(lastMsg.role)}]: ${lastMsg.content}`);
    }
    // If no responder available, turn stays open — orchestrator will retry on next event
  }

  /** Open a new conversation turn */
  private openTurn(speaker: string, allowedResponders: string[]): void {
    this.closeTurn(); // safety: close any stale turn

    this.currentTurn = {
      speaker,
      allowedResponders,
      maxReplies: this.config.maxTurnReplies,
      repliesSoFar: 0,
      startedAt: Date.now(),
      timeoutMs: this.config.turnTimeoutMs,
    };

    // Mark responders as busy
    for (const role of allowedResponders) {
      const agent = this.agents.get(role);
      if (agent) agent.status = 'busy';
    }

    this.emitEvent({
      type: 'turn_start',
      role: speaker,
      content: `Turn: ${allowedResponders.join(', ')} may respond`,
      timestamp: new Date().toISOString(),
    });

    // Set timeout
    this.turnTimer = setTimeout(() => {
      this.closeTurn();
      // After timeout, let orchestrator decide next action
      this.emitEvent({
        type: 'turn_close',
        content: 'Turn timed out',
        timestamp: new Date().toISOString(),
      });
    }, this.config.turnTimeoutMs);
  }

  /** Close the current turn */
  private closeTurn(): void {
    if (this.turnTimer) {
      clearTimeout(this.turnTimer);
      this.turnTimer = null;
    }

    if (this.currentTurn) {
      // Reset busy agents back to ready
      for (const role of this.currentTurn.allowedResponders) {
        const agent = this.agents.get(role);
        if (agent && agent.status === 'busy') {
          agent.status = 'ready';
        }
      }

      this.currentTurn = null;
      this.emitEvent({ type: 'turn_close', timestamp: new Date().toISOString() });
    }
  }

  /** Write context/turn input to an agent's PTY stdin */
  private injectContext(role: string, text: string): void {
    if (!this.ptyManager) return;
    // Collapse to single line for the stdin protocol
    const singleLine = text.replace(/\n/g, '\\n');
    this.ptyManager.writeTurn(role, singleLine);
  }

  /** Emit a typed event for the display layer */
  private emitEvent(event: OrchestratorEvent): void {
    this.emit('event', event);
  }
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}
