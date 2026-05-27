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

  async removeAgent(agentId: string): Promise<void> {
    const response = await fetch(`${this.serverUrl}/api/agents/${encodeURIComponent(agentId)}`, {
      method: 'DELETE'
    });
    const data = await response.json();
    if (!response.ok || !data.success) {
      throw new Error(data.error || `HTTP ${response.status}`);
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
      if (!data.project) return null;
      // Return merged object: project fields + live tasks + taskSummary
      return { ...data.project, tasks: data.tasks || [], taskSummary: data.taskSummary || {} };
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

  async sendMessage(sender: string, message: string, projectName?: string): Promise<void> {
    const project = projectName ?? process.env.ACT_PROJECT ?? '';
    await fetch(`${this.serverUrl}/api/messages`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ sender, projectName: project, message })
    });
  }

  async getMessages(agentId: string, limit = 10, projectName?: string): Promise<any[]> {
    try {
      const project = projectName ?? process.env.ACT_PROJECT ?? '';
      const projectParam = project ? `&project=${encodeURIComponent(project)}` : '';
      const response = await fetch(`${this.serverUrl}/api/agents/${encodeURIComponent(agentId)}/messages?limit=${limit}${projectParam}`);
      if (!response.ok) return [];
      const data = await response.json() as any;
      return data.messages || [];
    } catch {
      return [];
    }
  }

  async getTasks(project?: string): Promise<any[]> {
    try {
      const projectParam = project ? `?project=${encodeURIComponent(project)}` : '';
      const response = await fetch(`${this.serverUrl}/api/tasks${projectParam}`);
      if (!response.ok) return [];
      const data = await response.json();
      return data.tasks || data || [];
    } catch {
      return [];
    }
  }

  async getFileLocks(): Promise<any[]> {
    try {
      const response = await fetch(`${this.serverUrl}/api/files/locks`);
      if (!response.ok) return [];
      const data = await response.json();
      return data.locks || [];
    } catch {
      return [];
    }
  }

  async patchTaskDependencies(taskId: string, dependencies: string[]): Promise<void> {
    const response = await fetch(`${this.serverUrl}/api/tasks/${taskId}/dependencies`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ dependencies })
    });
    const data = await response.json();
    if (!response.ok || !data.success) {
      throw new Error(data.error || `HTTP ${response.status}`);
    }
  }

  async getRecentLog(limit: number = 500, project?: string): Promise<any[]> {
    try {
      const projectParam = project ? `&project=${encodeURIComponent(project)}` : '';
      const response = await fetch(`${this.serverUrl}/api/log?limit=${limit}${projectParam}`);
      if (!response.ok) return [];
      const data = await response.json();
      return data.events || [];
    } catch {
      return [];
    }
  }

  /** Returns tasks that are permanently failed (retryCount >= MAX_RETRIES). */
  async getPermanentlyFailedTasks(): Promise<any[]> {
    try {
      const response = await fetch(`${this.serverUrl}/api/tasks/failed-permanently`);
      if (!response.ok) return [];
      const data = await response.json();
      return data.tasks || [];
    } catch {
      return [];
    }
  }

  /** Retry a failed task — resets it to pending and increments retryCount.
   *  Returns the updated task, or null if permanently failed. */
  async retryTask(taskId: string): Promise<{ success: boolean; permanentlyFailed?: boolean; task?: any; error?: string }> {
    try {
      const response = await fetch(`${this.serverUrl}/api/tasks/${taskId}/retry`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
      });
      const data = await response.json();
      return data;
    } catch (error: any) {
      return { success: false, error: error.message };
    }
  }

  /** Abandon a task — marks it permanently failed with metadata.abandoned=true.
   *  Distinct from retry: the task is NOT re-dispatched. Reason is required for
   *  audit. Server returns 409 if the task is already validated/completed. */
  async abandonTask(taskId: string, reason: string): Promise<{ success: boolean; task?: any; error?: string }> {
    try {
      const response = await fetch(`${this.serverUrl}/api/tasks/${taskId}/abandon`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason })
      });
      const data = await response.json();
      if (!response.ok) {
        return { success: false, error: data.error || `HTTP ${response.status}` };
      }
      return data;
    } catch (error: any) {
      return { success: false, error: error.message };
    }
  }

  /** Get the currently assigned task for an agent */
  async getAssignedTask(agentId: string): Promise<any | null> {
    try {
      const response = await fetch(`${this.serverUrl}/api/tasks/assigned?agent_id=${encodeURIComponent(agentId)}`);
      if (!response.ok) return null;
      const data = await response.json();
      return data.task || null;
    } catch {
      return null;
    }
  }

  /** Mark a task as complete or failed */
  async completeTask(taskId: string, agentId: string, success: boolean, result?: string): Promise<{ success: boolean; error?: string }> {
    try {
      const response = await fetch(`${this.serverUrl}/api/tasks/${taskId}/complete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agentId, success, result })
      });
      const data = await response.json();
      if (!response.ok || !data.success) {
        return { success: false, error: data.error || `HTTP ${response.status}` };
      }
      return { success: true };
    } catch (error: any) {
      return { success: false, error: error.message };
    }
  }

  /** Update task progress */
  async updateTaskProgress(taskId: string, agentId: string, progress: number, status?: string): Promise<{ success: boolean; error?: string }> {
    try {
      const response = await fetch(`${this.serverUrl}/api/tasks/${taskId}/progress`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agentId, progress, status: status || 'in_progress' })
      });
      const data = await response.json();
      if (!response.ok || !data.success) {
        return { success: false, error: data.error || `HTTP ${response.status}` };
      }
      return { success: true };
    } catch (error: any) {
      return { success: false, error: error.message };
    }
  }

  /** Claim files for exclusive editing */
  async claimFiles(agentId: string, taskId: string, filePaths: string[], projectName?: string): Promise<{ success: boolean; claimed?: string[]; conflict?: boolean; conflicts?: any[]; error?: string }> {
    try {
      const project = projectName ?? process.env.ACT_PROJECT ?? '';
      const response = await fetch(`${this.serverUrl}/api/files/claim`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agent_id: agentId, task_id: taskId, project_name: project, file_paths: filePaths })
      });
      const data = await response.json();
      if (!response.ok) {
        return { success: false, error: data.error || `HTTP ${response.status}` };
      }
      return data;
    } catch (error: any) {
      return { success: false, error: error.message };
    }
  }

  /** Release file locks */
  async releaseFiles(agentId: string, filePaths: string[], taskId?: string, projectName?: string): Promise<{ success: boolean; released?: string[]; error?: string }> {
    try {
      const project = projectName ?? process.env.ACT_PROJECT ?? '';
      const response = await fetch(`${this.serverUrl}/api/files/release`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agent_id: agentId, task_id: taskId, project_name: project, file_paths: filePaths })
      });
      const data = await response.json();
      if (!response.ok || !data.success) {
        return { success: false, error: data.error || `HTTP ${response.status}` };
      }
      return data;
    } catch (error: any) {
      return { success: false, error: error.message };
    }
  }

  async submitForValidation(taskId: string, agentId: string, selfVerification?: string): Promise<{ success: boolean; task?: any; error?: string }> {
    try {
      const response = await fetch(`${this.serverUrl}/api/tasks/${taskId}/submit-for-validation`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agentId, selfVerification })
      });
      const data = await response.json() as any;
      return data;
    } catch (error: any) {
      return { success: false, error: error.message };
    }
  }

  async getValidationQueue(): Promise<any[]> {
    try {
      const response = await fetch(`${this.serverUrl}/api/tasks/pending-validation`);
      const data = await response.json() as any;
      return data.tasks || [];
    } catch {
      return [];
    }
  }

  async registerAgent(agentId: string, name?: string, capabilities?: string[]): Promise<{ success: boolean; error?: string; conflict?: boolean }> {
    try {
      const response = await fetch(`${this.serverUrl}/api/agents/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agentId, name: name || agentId, capabilities: capabilities || ['code', 'coordination'] })
      });
      const data = await response.json() as any;
      if (!response.ok) {
        return { success: false, error: data.error || `HTTP ${response.status}`, conflict: response.status === 409 };
      }
      return { success: true };
    } catch (error: any) {
      return { success: false, error: error.message };
    }
  }
}
