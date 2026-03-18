/**
 * NesTTY Process Manager — spawns agents as child processes.
 *
 * Drop-in replacement for PTYManager when node-pty is unavailable
 * (e.g., Node.js v25+ compatibility issues). Uses child_process.spawn
 * instead of node-pty. The agents still communicate via the same
 * JSON line protocol on stdin/stdout.
 *
 * Trade-offs vs PTYManager:
 * - No proper PTY emulation (no terminal signals, ANSI handling)
 * - Works on any Node.js version without native bindings
 * - Same PTYManagerInterface, same protocol
 */

import { spawn, type ChildProcess } from 'child_process';
import type { PTYManagerInterface } from './orchestrator.js';
import type { NestTTYMessage, NestTTYRole } from './types.js';

const MOCK_MODE = !!process.env.MOCK_AGENT;
const AGENT_CLI = process.env.AGENT_CLI || process.env.ACTOR_CLI || './act-agent/act-agent';

interface ManagedProcess {
  role: NestTTYRole;
  proc: ChildProcess;
  lineBuffer: string;
  alive: boolean;
}

export class ProcessManager implements PTYManagerInterface {
  private procs: Map<string, ManagedProcess> = new Map();
  private onMessage: (msg: NestTTYMessage) => void;

  constructor(onMessage: (msg: NestTTYMessage) => void) {
    this.onMessage = onMessage;
  }

  /**
   * Spawn an agent as a child process.
   */
  spawn(role: NestTTYRole, projectName: string, bootstrapPrompt?: string): void {
    if (this.procs.has(role)) {
      throw new Error(`Agent for role "${role}" already spawned`);
    }

    let cmd: string;
    let args: string[];

    if (MOCK_MODE) {
      const mockPath = new URL('./mock-agent.ts', import.meta.url).pathname;
      const nesttyDir = new URL('.', import.meta.url).pathname;
      // Find tsx register hook relative to nestty/ directory
      const tsxRegister = `${nesttyDir}node_modules/tsx/dist/esm/index.mjs`;
      cmd = process.execPath; // current node binary
      args = ['--import', tsxRegister, mockPath, '--role', role];
      if (bootstrapPrompt) args.push('--prompt', bootstrapPrompt);
    } else {
      cmd = AGENT_CLI;
      args = ['--nestty', role];
      if (bootstrapPrompt) args.push('--prompt', bootstrapPrompt);
    }

    const child = spawn(cmd, args, {
      cwd: process.cwd(),
      env: {
        ...process.env,
        ACT_NESTTY_ROLE: role,
        ACT_PROJECT: projectName,
      },
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    const managed: ManagedProcess = {
      role,
      proc: child,
      lineBuffer: '',
      alive: true,
    };

    this.procs.set(role, managed);

    // Parse stdout line-by-line
    child.stdout?.on('data', (data: Buffer) => {
      this.handleData(managed, data.toString());
    });

    // Log stderr (debug output from agent)
    child.stderr?.on('data', (data: Buffer) => {
      const text = data.toString().trim();
      if (text) {
        process.stderr.write(`[${role}:stderr] ${text}\n`);
      }
    });

    child.on('exit', (code, signal) => {
      managed.alive = false;
      // Flush remaining buffer
      if (managed.lineBuffer.trim()) {
        this.tryParseLine(managed, managed.lineBuffer);
        managed.lineBuffer = '';
      }
      this.onMessage({
        role,
        type: 'exit',
        content: `Process exited (code: ${code}, signal: ${signal ?? 'none'})`,
        time: new Date().toISOString(),
      });
    });

    child.on('error', (err) => {
      managed.alive = false;
      this.onMessage({
        role,
        type: 'error',
        content: `Spawn error: ${err.message}`,
        time: new Date().toISOString(),
      });
    });
  }

  /** Write a turn to an agent's stdin */
  writeTurn(role: string, input: string): void {
    const managed = this.procs.get(role);
    if (!managed || !managed.alive || !managed.proc.stdin?.writable) return;
    managed.proc.stdin.write(input + '\n');
  }

  /** Kill a specific agent */
  kill(role: string): void {
    const managed = this.procs.get(role);
    if (managed && managed.alive) {
      // Try graceful shutdown
      if (managed.proc.stdin?.writable) {
        managed.proc.stdin.write('__EXIT__\n');
      }
      // Force kill after 3 seconds
      setTimeout(() => {
        if (managed.alive) {
          managed.proc.kill('SIGKILL');
        }
      }, 3000);
    }
    this.procs.delete(role);
  }

  /** Kill all agents */
  killAll(): void {
    for (const role of [...this.procs.keys()]) {
      this.kill(role);
    }
  }

  // ─── Internal ────────────────────────────────────────────────────────────

  private handleData(managed: ManagedProcess, data: string): void {
    managed.lineBuffer += data;
    const lines = managed.lineBuffer.split('\n');
    managed.lineBuffer = lines.pop() ?? '';
    for (const line of lines) {
      const trimmed = line.trim();
      if (trimmed) this.tryParseLine(managed, trimmed);
    }
  }

  private tryParseLine(managed: ManagedProcess, line: string): void {
    try {
      const parsed = JSON.parse(line);
      if (parsed && typeof parsed.type === 'string' && typeof parsed.role === 'string') {
        this.onMessage({
          role: parsed.role,
          type: parsed.type,
          content: String(parsed.content ?? ''),
          time: parsed.time ?? new Date().toISOString(),
        });
      }
    } catch {
      if (line.startsWith('{')) {
        process.stderr.write(`[process-manager] ${managed.role}: malformed JSON: ${line.substring(0, 100)}\n`);
      }
    }
  }
}
