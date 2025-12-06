/**
 * ACT Coordination File Service
 * 
 * BULLETPROOF TEXT-BASED PARSING
 * 
 * This service treats act-coordination.json as a TEXT file, not JSON.
 * It extracts messages using pattern matching, so syntax errors never break reading.
 * Writes are still valid JSON to maintain structure for future parsing.
 * 
 * Features:
 * - Text-based message extraction (regex + bracket matching)
 * - Individual message parsing (one bad message doesn't break everything)
 * - Graceful degradation (returns what it can parse)
 * - Append-only writes (inserts at correct position without full parse)
 * - File locking for concurrent access
 */

import { promises as fs } from 'node:fs';
import path from 'node:path';
import lockfile from 'proper-lockfile';
import JSON5 from 'json5';
import type {
  CoordinationFile,
  CoordinationMessage,
  AgentCapabilities,
  AgentPreference,
  Phase,
  CurrentStatus,
  DocumentationEntry,
  ProjectStructureEntry
} from '../types.js';
import { CoordinationError } from '../types.js';

// ============================================================================
// Configuration
// ============================================================================

const DEFAULT_COORDINATION_FILE = process.env.ACT_COORDINATION_FILE 
  || '/Users/user/Documents/Developer/dev/AI/act/act-coordination.json';

const LOCK_OPTIONS = {
  retries: {
    retries: 5,
    factor: 2,
    minTimeout: 100,
    maxTimeout: 2000,
    randomize: true
  },
  stale: 10000
};

// ============================================================================
// Text-Based Message Extraction
// ============================================================================

/**
 * Extract messages from file content using text parsing (NOT JSON parsing)
 * This is bulletproof - it never fails due to JSON syntax errors
 */
function extractMessagesFromText(content: string): CoordinationMessage[] {
  const messages: CoordinationMessage[] = [];
  
  // Find the communication_log section
  const logMatch = content.match(/"communication_log"\s*:\s*\[/);
  if (!logMatch || logMatch.index === undefined) {
    return messages;
  }
  
  const startIndex = logMatch.index + logMatch[0].length;
  
  // Find message objects using bracket matching
  let depth = 0;
  let inString = false;
  let escaped = false;
  let messageStart = -1;
  
  for (let i = startIndex; i < content.length; i++) {
    const char = content[i];
    
    // Handle escape sequences in strings
    if (escaped) {
      escaped = false;
      continue;
    }
    
    if (char === '\\' && inString) {
      escaped = true;
      continue;
    }
    
    // Track string boundaries
    if (char === '"' && !escaped) {
      inString = !inString;
      continue;
    }
    
    // Skip if inside a string
    if (inString) continue;
    
    // Track object boundaries
    if (char === '{') {
      if (depth === 0) {
        messageStart = i;
      }
      depth++;
    } else if (char === '}') {
      depth--;
      if (depth === 0 && messageStart !== -1) {
        // Extract this message object
        const messageText = content.slice(messageStart, i + 1);
        const parsed = parseMessageSafe(messageText);
        if (parsed) {
          messages.push(parsed);
        }
        messageStart = -1;
      }
    } else if (char === ']' && depth === 0) {
      // End of communication_log array
      break;
    }
  }
  
  return messages;
}

/**
 * Safely parse a single message object, returning null if it fails
 */
function parseMessageSafe(messageText: string): CoordinationMessage | null {
  try {
    const obj = JSON5.parse(messageText);
    
    // Validate required fields exist
    if (typeof obj.timestamp === 'string' && 
        typeof obj.agent === 'string' && 
        typeof obj.message === 'string') {
      return {
        timestamp: obj.timestamp,
        agent: obj.agent,
        message: obj.message,
        type: obj.type || 'coordination'
      };
    }
    return null;
  } catch {
    // Try to extract fields with regex as last resort
    return extractMessageWithRegex(messageText);
  }
}

/**
 * Last-resort extraction using regex when JSON parsing completely fails
 */
function extractMessageWithRegex(text: string): CoordinationMessage | null {
  try {
    const timestampMatch = text.match(/"timestamp"\s*:\s*"([^"]+)"/);
    const agentMatch = text.match(/"agent"\s*:\s*"([^"]+)"/);
    const typeMatch = text.match(/"type"\s*:\s*"([^"]+)"/);
    
    // Message field is complex - might contain escaped quotes
    const messageMatch = text.match(/"message"\s*:\s*"((?:[^"\\]|\\.)*)"/s);
    
    if (timestampMatch && agentMatch && messageMatch) {
      return {
        timestamp: timestampMatch[1],
        agent: agentMatch[1],
        message: messageMatch[1].replace(/\\n/g, '\n').replace(/\\"/g, '"'),
        type: typeMatch ? typeMatch[1] : 'coordination'
      };
    }
    return null;
  } catch {
    return null;
  }
}

/**
 * Extract a section from the file as text
 */
function extractSection(content: string, sectionName: string): string | null {
  const pattern = new RegExp(`"${sectionName}"\\s*:\\s*`, 'g');
  const match = pattern.exec(content);
  if (!match || match.index === undefined) return null;
  
  let startIndex = match.index + match[0].length;
  const startChar = content[startIndex];
  
  if (startChar === '{' || startChar === '[') {
    const endChar = startChar === '{' ? '}' : ']';
    let depth = 1;
    let inString = false;
    let escaped = false;
    
    for (let i = startIndex + 1; i < content.length; i++) {
      const char = content[i];
      
      if (escaped) { escaped = false; continue; }
      if (char === '\\' && inString) { escaped = true; continue; }
      if (char === '"' && !escaped) { inString = !inString; continue; }
      if (inString) continue;
      
      if (char === startChar) depth++;
      else if (char === endChar) {
        depth--;
        if (depth === 0) {
          return content.slice(startIndex, i + 1);
        }
      }
    }
  }
  return null;
}

/**
 * Try to parse a section as JSON5, return null if it fails
 */
function parseSectionSafe<T>(content: string, sectionName: string): T | null {
  const section = extractSection(content, sectionName);
  if (!section) return null;
  try {
    return JSON5.parse(section) as T;
  } catch {
    return null;
  }
}

// ============================================================================
// File Service Class
// ============================================================================

export class CoordinationFileService {
  private filePath: string;
  private projectRoot: string;

  constructor(filePath?: string) {
    this.filePath = filePath || DEFAULT_COORDINATION_FILE;
    this.projectRoot = path.dirname(this.filePath);
  }

  // ==========================================================================
  // Core File Operations
  // ==========================================================================

  /**
   * Read file content as text (never fails on syntax errors)
   */
  private async readFileAsText(): Promise<string> {
    try {
      return await fs.readFile(this.filePath, 'utf-8');
    } catch (error) {
      throw new CoordinationError(
        `Failed to read file: ${error instanceof Error ? error.message : 'Unknown error'}`,
        'READ_ERROR',
        'Ensure the coordination file exists'
      );
    }
  }

  /**
   * Try to parse full file as JSON5 (best-effort, may return partial data)
   */
  async readFile(withLock: boolean = false): Promise<CoordinationFile> {
    const fn = async () => {
      const content = await this.readFileAsText();
      
      // Try full JSON5 parse first
      try {
        return JSON5.parse(content) as CoordinationFile;
      } catch {
        // Fall back to text-based extraction
        return this.reconstructFromText(content);
      }
    };
    
    if (withLock) {
      return await this.withLock(fn);
    }
    return await fn();
  }

  /**
   * Reconstruct coordination file structure from text when JSON parsing fails
   */
  private reconstructFromText(content: string): CoordinationFile {
    // Extract what we can from the text
    const messages = extractMessagesFromText(content);
    const project = parseSectionSafe<CoordinationFile['project']>(content, 'project');
    const agents = parseSectionSafe<CoordinationFile['agents']>(content, 'agents');
    const phases = parseSectionSafe<CoordinationFile['phases']>(content, 'phases');
    const currentStatus = parseSectionSafe<CurrentStatus>(content, 'current_status');
    const resources = parseSectionSafe<CoordinationFile['resources']>(content, 'resources');
    
    return {
      project: project || { name: 'ACT', description: 'Unknown', timeline: 'Unknown', goal: 'Unknown' },
      agents: agents || {},
      phases: phases || {},
      current_status: currentStatus || {
        active_phase: 'unknown',
        next_milestone: 'Unknown',
        total_progress: '0%',
        estimated_completion: new Date().toISOString(),
        critical_path: [],
        demo_ready: false,
        build_in_public_ready: false
      },
      communication_log: messages,
      resources: resources || { documentation: [], development_urls: [] }
    };
  }

  /**
   * Execute a function with file locking
   */
  private async withLock<T>(fn: () => Promise<T>): Promise<T> {
    let release: (() => Promise<void>) | null = null;
    try {
      release = await lockfile.lock(this.filePath, LOCK_OPTIONS);
      return await fn();
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code === 'ELOCKED') {
        throw new CoordinationError(
          'Coordination file is currently locked by another agent',
          'LOCK_ERROR',
          'Wait a moment and retry - another agent is writing'
        );
      }
      throw error;
    } finally {
      if (release) {
        await release();
      }
    }
  }

  // ==========================================================================
  // Communication Log Operations
  // ==========================================================================

  /**
   * Read recent messages (text-based, bulletproof)
   */
  async readCommunicationLog(
    limit: number = 10,
    offset: number = 0
  ): Promise<{ messages: CoordinationMessage[]; total: number; has_more: boolean }> {
    const content = await this.readFileAsText();
    const allMessages = extractMessagesFromText(content);
    const total = allMessages.length;
    
    // Get messages from the end (most recent first)
    const startIndex = Math.max(0, total - offset - limit);
    const endIndex = Math.max(0, total - offset);
    const messages = allMessages.slice(startIndex, endIndex).reverse();
    
    return {
      messages,
      total,
      has_more: startIndex > 0
    };
  }

  /**
   * Append a message using text insertion (doesn't require full JSON parse)
   */
  async appendMessage(
    agent: string,
    message: string,
    type: string
  ): Promise<{ timestamp: string; index: number; message: CoordinationMessage }> {
    return await this.withLock(async () => {
      let content = await this.readFileAsText();
      const timestamp = new Date().toISOString();
      
      // Create the new message as valid JSON
      const newMessage: CoordinationMessage = {
        timestamp,
        agent,
        message,
        type
      };
      
      const messageJson = JSON.stringify(newMessage, null, 2)
        .split('\n')
        .map((line, i) => i === 0 ? line : '    ' + line)
        .join('\n');
      
      // Find where to insert: before the closing ] of communication_log
      // Strategy: Find "communication_log" then find the last } before ] or "resources"
      
      const logMatch = content.match(/"communication_log"\s*:\s*\[/);
      if (!logMatch || logMatch.index === undefined) {
        throw new CoordinationError(
          'Could not find communication_log in file',
          'STRUCTURE_ERROR',
          'The coordination file may be missing the communication_log section'
        );
      }
      
      // Find the end of communication_log array
      // Look for the pattern: }  followed by ] (end of array)
      // or } followed by ], (end of array with comma for next section)
      
      const startSearch = logMatch.index + logMatch[0].length;
      let depth = 1; // We're inside the [ already
      let inString = false;
      let escaped = false;
      let insertPosition = -1;
      let lastObjectEnd = -1;
      let objectDepth = 0;
      
      for (let i = startSearch; i < content.length; i++) {
        const char = content[i];
        
        if (escaped) { escaped = false; continue; }
        if (char === '\\' && inString) { escaped = true; continue; }
        if (char === '"' && !escaped) { inString = !inString; continue; }
        if (inString) continue;
        
        if (char === '{') objectDepth++;
        else if (char === '}') {
          objectDepth--;
          if (objectDepth === 0) {
            lastObjectEnd = i + 1; // Position after the }
          }
        }
        else if (char === '[') depth++;
        else if (char === ']') {
          depth--;
          if (depth === 0) {
            // Found the end of communication_log array
            insertPosition = lastObjectEnd > 0 ? lastObjectEnd : i;
            break;
          }
        }
      }
      
      if (insertPosition === -1) {
        throw new CoordinationError(
          'Could not find end of communication_log array',
          'STRUCTURE_ERROR',
          'The coordination file structure may be corrupted'
        );
      }
      
      // Count existing messages to get index
      const existingMessages = extractMessagesFromText(content);
      const index = existingMessages.length;
      
      // Insert the new message
      const needsComma = lastObjectEnd > 0; // There are existing messages
      const insertion = needsComma 
        ? `,\n    ${messageJson}`
        : `\n    ${messageJson}`;
      
      content = content.slice(0, insertPosition) + insertion + content.slice(insertPosition);
      
      // Write back
      await fs.writeFile(this.filePath, content, 'utf-8');
      
      return { timestamp, index, message: newMessage };
    });
  }

  /**
   * Search the communication log (text-based)
   */
  async searchCommunicationLog(
    query: string,
    options: {
      agent_filter?: string;
      type_filter?: string;
      timeframe?: 'last_day' | 'last_week' | 'last_month' | 'all';
    } = {}
  ): Promise<{
    results: Array<{
      message: CoordinationMessage;
      index: number;
      context_before?: CoordinationMessage;
      context_after?: CoordinationMessage;
    }>;
    total: number;
  }> {
    const content = await this.readFileAsText();
    const log = extractMessagesFromText(content);
    
    // Calculate time boundary
    const now = new Date();
    let timeBoundary: Date | null = null;
    
    switch (options.timeframe) {
      case 'last_day':
        timeBoundary = new Date(now.getTime() - 24 * 60 * 60 * 1000);
        break;
      case 'last_week':
        timeBoundary = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
        break;
      case 'last_month':
        timeBoundary = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
        break;
    }
    
    const queryLower = query.toLowerCase();
    const results: Array<{
      message: CoordinationMessage;
      index: number;
      context_before?: CoordinationMessage;
      context_after?: CoordinationMessage;
    }> = [];
    
    for (let i = 0; i < log.length; i++) {
      const msg = log[i];
      
      // Apply filters
      if (options.agent_filter && msg.agent !== options.agent_filter) continue;
      if (options.type_filter && msg.type !== options.type_filter) continue;
      if (timeBoundary) {
        try {
          const msgTime = new Date(msg.timestamp);
          if (msgTime < timeBoundary) continue;
        } catch {
          // Skip messages with invalid timestamps
          continue;
        }
      }
      
      // Search in message content
      if (msg.message.toLowerCase().includes(queryLower) ||
          msg.agent.toLowerCase().includes(queryLower) ||
          (msg.type && msg.type.toLowerCase().includes(queryLower))) {
        results.push({
          message: msg,
          index: i,
          context_before: i > 0 ? log[i - 1] : undefined,
          context_after: i < log.length - 1 ? log[i + 1] : undefined
        });
      }
    }
    
    return { results, total: results.length };
  }

  /**
   * Check for updates since a given timestamp
   */
  async checkForUpdates(
    lastReadTimestamp: string
  ): Promise<{
    has_updates: boolean;
    new_count: number;
    messages: CoordinationMessage[];
  }> {
    const content = await this.readFileAsText();
    const log = extractMessagesFromText(content);
    
    let lastRead: Date;
    try {
      lastRead = new Date(lastReadTimestamp);
    } catch {
      return { has_updates: false, new_count: 0, messages: [] };
    }
    
    const newMessages = log.filter(msg => {
      try {
        return new Date(msg.timestamp) > lastRead;
      } catch {
        return false;
      }
    });
    
    return {
      has_updates: newMessages.length > 0,
      new_count: newMessages.length,
      messages: newMessages
    };
  }

  // ==========================================================================
  // Agent Status Operations
  // ==========================================================================

  async getAgentStatus(agentName: string): Promise<{
    found: boolean;
    capabilities?: AgentCapabilities;
    preferences?: AgentPreference;
    recent_messages: CoordinationMessage[];
  }> {
    const content = await this.readFileAsText();
    
    // Extract agents section
    const agents = parseSectionSafe<Record<string, AgentCapabilities>>(content, 'agents');
    const capabilities = agents?.[agentName];
    
    // Get recent messages from this agent
    const allMessages = extractMessagesFromText(content);
    const recentMessages = allMessages
      .filter(msg => msg.agent === agentName || msg.agent.includes(agentName))
      .slice(-5);
    
    return {
      found: !!capabilities || recentMessages.length > 0,
      capabilities,
      recent_messages: recentMessages
    };
  }

  // ==========================================================================
  // Phase Status Operations
  // ==========================================================================

  async getPhaseStatus(): Promise<{
    active_phase: string;
    phase_details?: Phase;
    current_status: CurrentStatus;
    critical_decisions?: string[];
  }> {
    const content = await this.readFileAsText();
    
    const currentStatus = parseSectionSafe<CurrentStatus>(content, 'current_status');
    const phases = parseSectionSafe<Record<string, Phase>>(content, 'phases');
    
    const activePhase = currentStatus?.active_phase || 'unknown';
    const phaseDetails = phases?.[activePhase];
    
    // Try to get critical decisions
    const phase5Status = parseSectionSafe<{ key_decisions_documented?: Record<string, unknown> }>(content, 'phase_5_status');
    const criticalDecisions = phase5Status?.key_decisions_documented 
      ? Object.keys(phase5Status.key_decisions_documented)
      : undefined;
    
    return {
      active_phase: activePhase,
      phase_details: phaseDetails,
      current_status: currentStatus || {
        active_phase: 'unknown',
        next_milestone: 'Unknown',
        total_progress: '0%',
        estimated_completion: new Date().toISOString(),
        critical_path: [],
        demo_ready: false,
        build_in_public_ready: false
      },
      critical_decisions: criticalDecisions
    };
  }

  // ==========================================================================
  // Documentation Index Operations
  // ==========================================================================

  async getDocumentationIndex(includeSizes: boolean = false): Promise<DocumentationEntry[]> {
    const docsDir = path.join(this.projectRoot, 'docs');
    const entries: DocumentationEntry[] = [];
    
    try {
      const files = await fs.readdir(docsDir);
      
      for (const file of files) {
        if (!file.endsWith('.md')) continue;
        
        const filePath = path.join(docsDir, file);
        const stats = await fs.stat(filePath);
        
        let title = file.replace('.md', '').replace(/_/g, ' ');
        let purpose = 'Documentation file';
        
        try {
          const content = await fs.readFile(filePath, 'utf-8');
          const lines = content.split('\n');
          
          const titleMatch = lines.find(l => l.startsWith('# '));
          if (titleMatch) {
            title = titleMatch.replace('# ', '').trim();
          }
          
          const purposeLines = lines.slice(1).filter(l => l.trim().length > 0);
          if (purposeLines.length > 0 && !purposeLines[0].startsWith('#')) {
            purpose = purposeLines[0].slice(0, 200);
          }
        } catch {
          // Ignore read errors
        }
        
        const entry: DocumentationEntry = {
          path: `docs/${file}`,
          title,
          purpose,
          last_updated: stats.mtime.toISOString()
        };
        
        if (includeSizes) {
          entry.size_bytes = stats.size;
        }
        
        entries.push(entry);
      }
    } catch (error) {
      throw new CoordinationError(
        `Failed to read documentation directory: ${error instanceof Error ? error.message : 'Unknown error'}`,
        'DOCS_READ_ERROR',
        'Ensure the docs/ directory exists'
      );
    }
    
    return entries;
  }

  // ==========================================================================
  // Project Structure Operations
  // ==========================================================================

  async getProjectStructure(
    maxDepth: number = 3,
    includeHidden: boolean = false,
    excludePatterns: string[] = ['node_modules', '.git', 'build', 'dist', '__pycache__', '.next']
  ): Promise<ProjectStructureEntry[]> {
    return await this.traverseDirectory(
      this.projectRoot,
      0,
      maxDepth,
      includeHidden,
      excludePatterns
    );
  }

  private async traverseDirectory(
    dirPath: string,
    currentDepth: number,
    maxDepth: number,
    includeHidden: boolean,
    excludePatterns: string[]
  ): Promise<ProjectStructureEntry[]> {
    if (currentDepth >= maxDepth) return [];
    
    const entries: ProjectStructureEntry[] = [];
    
    try {
      const items = await fs.readdir(dirPath, { withFileTypes: true });
      
      for (const item of items) {
        if (!includeHidden && item.name.startsWith('.')) continue;
        if (excludePatterns.some(p => item.name === p || item.name.includes(p))) continue;
        
        const relativePath = path.relative(this.projectRoot, path.join(dirPath, item.name));
        
        if (item.isDirectory()) {
          const children = await this.traverseDirectory(
            path.join(dirPath, item.name),
            currentDepth + 1,
            maxDepth,
            includeHidden,
            excludePatterns
          );
          
          entries.push({
            path: relativePath,
            type: 'directory',
            children: children.length > 0 ? children : undefined
          });
        } else {
          entries.push({
            path: relativePath,
            type: 'file'
          });
        }
      }
    } catch {
      // Silently skip directories we can't read
    }
    
    entries.sort((a, b) => {
      if (a.type === b.type) return a.path.localeCompare(b.path);
      return a.type === 'directory' ? -1 : 1;
    });
    
    return entries;
  }

  // ==========================================================================
  // Utility Methods
  // ==========================================================================

  getFilePath(): string {
    return this.filePath;
  }

  getProjectRoot(): string {
    return this.projectRoot;
  }

  async fileExists(): Promise<boolean> {
    try {
      await fs.access(this.filePath);
      return true;
    } catch {
      return false;
    }
  }
}

export const coordinationFileService = new CoordinationFileService();
