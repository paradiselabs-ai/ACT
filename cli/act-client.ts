/**
 * ACT Client - Communication layer for ACT REPL
 *
 * Handles HTTP requests to ACT server endpoints for coordination operations.
 */

export interface Agent {
  id: string;
  name?: string;
  capabilities: string[];
  status: string;
  model?: string;
  provider?: string;
  currentTask?: string;
}

export interface Project {
  id: string;
  name: string;
  workspace: string;
  status: string;
  progress: number;
  tasks: Task[];
}

export interface Task {
  id: string;
  description: string;
  status: string;
  assignedAgent?: string;
  progress: number;
}

export interface PVMStatus {
  isRunning: boolean;
  isIndexing: boolean;
  lastIndexedTimestamp: string | null;
  indexedEventCount: number;
}

export interface ServerStatus {
  status: string;
  uptime?: string;
  agents: number;
  tasks?: any[]; // Can be number or array depending on endpoint
}

export class ACTClient {
  private serverUrl: string;

  constructor(serverUrl: string = 'http://localhost:8080') {
    this.serverUrl = serverUrl.replace(/\/$/, ''); // Remove trailing slash
  }

  getServerUrl(): string {
    return this.serverUrl;
  }

  async testConnection(): Promise<void> {
    try {
      const response = await fetch(`${this.serverUrl}/health`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      const data = await response.json();
      if (data.status !== 'healthy') {
        throw new Error('Server not healthy');
      }
    } catch (error: any) {
      throw new Error(`Cannot connect to ACT server: ${error.message}`);
    }
  }

  async getAgents(): Promise<Agent[]> {
    try {
      const response = await fetch(`${this.serverUrl}/api/agents`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      return await response.json();
    } catch (error: any) {
      console.warn('Failed to fetch agents:', error.message);
      return [];
    }
  }

  async getServerStatus(): Promise<ServerStatus> {
    try {
      const response = await fetch(`${this.serverUrl}/health`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      const data = await response.json();
      return {
        status: data.status,
        uptime: 'Unknown', // Would need server to track this
        agents: data.agents || 0,
        tasks: data.tasks || 0
      };
    } catch (error: any) {
      throw new Error(`Failed to get server status: ${error.message}`);
    }
  }

  async getPVMStatus(): Promise<PVMStatus> {
    try {
      const response = await fetch(`${this.serverUrl}/api/pvm/status`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      return await response.json();
    } catch (error: any) {
      // Return default status if PVM not available
      return {
        isRunning: false,
        isIndexing: false,
        lastIndexedTimestamp: null,
        indexedEventCount: 0
      };
    }
  }

  async searchPVM(query: string, limit: number = 10): Promise<any[]> {
    try {
      const response = await fetch(`${this.serverUrl}/api/pvm/search?query=${encodeURIComponent(query)}&limit=${limit}`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      const data = await response.json();
      return data.results || [];
    } catch (error: any) {
      console.warn('PVM search failed:', error.message);
      return [];
    }
  }

  async triggerImprovement(request: any): Promise<any> {
    try {
      const response = await fetch(`${this.serverUrl}/api/improve`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(request)
      });
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      return await response.json();
    } catch (error: any) {
      throw new Error(`Improvement request failed: ${error.message}`);
    }
  }

  async createTaskREST(task: any): Promise<any> {
    const response = await fetch(`${this.serverUrl}/api/tasks`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(task)
    });
    const data = await response.json();
    if (!response.ok || !data.success) {
      throw new Error(data.error || `HTTP ${response.status}`);
    }
    return data.task;
  }

  async getTask(taskId: string): Promise<any> {
    try {
      const response = await fetch(`${this.serverUrl}/api/tasks/${taskId}`);
      if (!response.ok) return null;
      const data = await response.json();
      return data.task || null;
    } catch {
      return null;
    }
  }

  async updateTaskStatus(taskId: string, agentId: string, status: string): Promise<void> {
    await fetch(`${this.serverUrl}/api/tasks/${taskId}/progress`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ agentId, status, progress: 0 })
    });
  }

  async createProject(data: {
    name: string;
    workspace: string;
    description: string;
    techStack: string;
    constraints?: string;
    successCriteria: string;
    agents: string[];
  }): Promise<any> {
    const response = await fetch(`${this.serverUrl}/api/projects`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    });
    const result = await response.json();
    if (!response.ok || !result.success) {
      throw new Error(result.error || `HTTP ${response.status}`);
    }
    return result.project;
  }

  async getProjects(): Promise<any[]> {
    try {
      const response = await fetch(`${this.serverUrl}/api/projects`);
      if (!response.ok) return [];
      return await response.json();
    } catch {
      return [];
    }
  }

  async getProject(name: string): Promise<any> {
    try {
      const response = await fetch(`${this.serverUrl}/api/projects/${encodeURIComponent(name)}`);
      if (!response.ok) return null;
      const data = await response.json();
      return data.project || null;
    } catch {
      return null;
    }
  }

  async storeBrief(projectName: string, agentId: string, content: string): Promise<void> {
    const response = await fetch(`${this.serverUrl}/api/projects/${encodeURIComponent(projectName)}/briefs`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ agentId, content })
    });
    const data = await response.json();
    if (!response.ok || !data.success) {
      throw new Error(data.error || `HTTP ${response.status}`);
    }
  }

  async getBrief(projectName: string, agentId: string): Promise<string | null> {
    try {
      const response = await fetch(`${this.serverUrl}/api/projects/${encodeURIComponent(projectName)}/briefs/${encodeURIComponent(agentId)}`);
      if (!response.ok) return null;
      const data = await response.json();
      return data.content || null;
    } catch {
      return null;
    }
  }
}
