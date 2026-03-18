/**
 * NesTTY PTY Manager — spawns act-agent processes in background PTYs.
 *
 * Each Tier 1 agent runs as a persistent interactive session inside a
 * background PTY. The orchestrator writes turns to stdin and reads
 * JSON line responses from stdout.
 *
 * Uses node-pty for proper PTY emulation (handles signals, job control,
 * terminal escape sequences that child processes may emit).
 */

import * as pty from 'node-pty';
import type { IPty } from 'node-pty';
import type { PTYManagerInterface } from './orchestrator.js';
import type { NestTTYMessage, NestTTYRole } from './types.js';

const MOCK_MODE = !!process.env.MOCK_AGENT;
const AGENT_CLI = process.env.AGENT_CLI || process.env.ACTOR_CLI || './act-agent/act-agent';

interface ManagedPTY {
  role: NestTTYRole;
  pty: IPty;
  lineBuffer: string;  // partial line accumulator
  alive: boolean;
}

export class PTYManager implements PTYManagerInterface {
  private ptys: Map<string, ManagedPTY> = new Map();
  private onMessage: (msg: NestTTYMessage) => void;

  constructor(onMessage: (msg: NestTTYMessage) => void) {
    this.onMessage = onMessage;
  }

  /**
   * Spawn an agent in a background PTY.
   * The agent runs: act-agent --nestty <role> [--prompt "<bootstrap>"]
   */
  spawn(role: NestTTYRole, projectName: string, bootstrapPrompt?: string): void {
    if (this.ptys.has(role)) {
      throw new Error(`Agent for role "${role}" already spawned`);
    }

    let spawnCmd = AGENT_CLI;
    let args: string[];
    if (MOCK_MODE) {
      // Mock mode: spawn the mock-agent TypeScript script via npx tsx
      const mockPath = new URL('./mock-agent.ts', import.meta.url).pathname;
      spawnCmd = '/opt/homebrew/bin/npx';
      args = ['tsx', mockPath, '--role', role];
      if (bootstrapPrompt) args.push('--prompt', bootstrapPrompt);
    } else {
      args = ['--nestty', role];
      if (bootstrapPrompt) args.push('--prompt', bootstrapPrompt);
    }

    const child = pty.spawn(spawnCmd, args, {
      name: 'xterm-256color',
      cols: 120,
      rows: 40,
      cwd: process.cwd(),
      env: {
        ...process.env as { [key: string]: string },
        ACT_NESTTY_ROLE: role,
        ACT_PROJECT: projectName,
      },
    });

    const managed: ManagedPTY = {
      role,
      pty: child,
      lineBuffer: '',
      alive: true,
    };

    this.ptys.set(role, managed);

    // Parse stdout line-by-line for JSON messages
    child.onData((data: string) => {
      this.handleData(managed, data);
    });

    // Handle PTY exit
    child.onExit(({ exitCode, signal }) => {
      managed.alive = false;
      // Flush any remaining buffer
      if (managed.lineBuffer.trim()) {
        this.tryParseLine(managed, managed.lineBuffer);
        managed.lineBuffer = '';
      }
      // Notify orchestrator
      this.onMessage({
        role,
        type: 'exit',
        content: `Process exited (code: ${exitCode}, signal: ${signal ?? 'none'})`,
        time: new Date().toISOString(),
      });
    });
  }

  /** Write a turn to an agent's PTY stdin */
  writeTurn(role: string, input: string): void {
    const managed = this.ptys.get(role);
    if (!managed || !managed.alive) {
      return; // silently skip dead agents
    }
    // Write input + newline — the agent reads one line at a time
    managed.pty.write(input + '\n');
  }

  /** Kill a specific agent's PTY */
  kill(role: string): void {
    const managed = this.ptys.get(role);
    if (managed && managed.alive) {
      // Try graceful shutdown first
      managed.pty.write('__EXIT__\n');
      // Force kill after 3 seconds if still alive
      setTimeout(() => {
        if (managed.alive) {
          managed.pty.kill();
        }
      }, 3000);
    }
    this.ptys.delete(role);
  }

  /** Kill all PTYs */
  killAll(): void {
    for (const role of [...this.ptys.keys()]) {
      this.kill(role);
    }
  }

  /** Check if a role has a live PTY */
  isAlive(role: string): boolean {
    return this.ptys.get(role)?.alive ?? false;
  }

  // ─── Internal ────────────────────────────────────────────────────────────

  /**
   * Handle raw data from a PTY. Buffer partial lines and parse complete ones.
   * node-pty may deliver data in chunks that split mid-JSON-line.
   */
  private handleData(managed: ManagedPTY, data: string): void {
    managed.lineBuffer += data;

    // Split on newlines — complete lines can be parsed
    const lines = managed.lineBuffer.split('\n');

    // Last element is either empty (if data ended with \n) or a partial line
    managed.lineBuffer = lines.pop() ?? '';

    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      this.tryParseLine(managed, trimmed);
    }
  }

  /**
   * Try to parse a line as a NestTTYMessage JSON.
   * If parsing fails, log a warning but don't crash.
   */
  private tryParseLine(managed: ManagedPTY, line: string): void {
    // Strip ANSI escape sequences that the PTY might inject
    const cleaned = stripAnsi(line);
    if (!cleaned) return;

    try {
      const parsed = JSON.parse(cleaned);

      // Validate expected shape
      if (parsed && typeof parsed.type === 'string' && typeof parsed.role === 'string') {
        const msg: NestTTYMessage = {
          role: parsed.role,
          type: parsed.type,
          content: String(parsed.content ?? ''),
          time: parsed.time ?? new Date().toISOString(),
        };
        this.onMessage(msg);
      }
    } catch {
      // Not valid JSON — could be debug output, ANSI noise, etc.
      // Only log if it looks like it was trying to be JSON
      if (cleaned.startsWith('{')) {
        process.stderr.write(`[pty-manager] ${managed.role}: malformed JSON: ${cleaned.substring(0, 100)}\n`);
      }
    }
  }
}

/**
 * Strip ANSI escape sequences from a string.
 * PTYs emit color codes, cursor movements, etc. that corrupt JSON.
 */
function stripAnsi(str: string): string {
  // eslint-disable-next-line no-control-regex
  return str.replace(/\x1b\[[0-9;]*[a-zA-Z]/g, '').replace(/\x1b\][^\x07]*\x07/g, '').trim();
}
