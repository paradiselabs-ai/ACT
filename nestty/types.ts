/**
 * NesTTY shared types — used by all nestty/ modules.
 */

/** JSON line protocol message from act-agent --nestty */
export interface NestTTYMessage {
  role: string;
  type: 'ready' | 'message' | 'error' | 'exit';
  content: string;
  time: string;
}

/** Tier 1 roles */
export type NestTTYRole = 'planner' | 'observer' | 'assurance' | 'qa';

export const ALL_ROLES: NestTTYRole[] = ['planner', 'observer', 'assurance', 'qa'];

/** Agent state within a session */
export interface SessionAgent {
  role: NestTTYRole;
  status: 'spawning' | 'ready' | 'busy' | 'dead';
  messageHistory: NestTTYMessage[];
}

/** Conversation turn — prevents infinite response loops */
export interface ConversationTurn {
  speaker: string;              // role of who just spoke
  allowedResponders: string[];  // who may reply (usually 1)
  maxReplies: number;           // default 1
  repliesSoFar: number;
  startedAt: number;            // Date.now()
  timeoutMs: number;            // auto-close after this
}

/** Orchestrator event for the display layer */
export interface OrchestratorEvent {
  type: 'agent_ready' | 'turn_start' | 'turn_message' | 'turn_close' | 'hitl' | 'error' | 'shutdown';
  role?: string;
  content?: string;
  timestamp: string;
}
