import { ImprovementRequest, ImprovementResult, BackgroundTask } from './types.js';
import { v4 as uuidv4 } from 'uuid';
import { io, Socket } from 'socket.io-client';

export class SelfImprovementEngine {
  private backgroundTaskQueue: BackgroundTask[] = [];
  private isProcessing: boolean = false;
  private lastActivityTimestamp: number = Date.now();
  private socket!: Socket;  // Definite assignment assertion
  private isConnected: boolean = false;
  private eventHistory: any[] = [];
  private agentProfiles: Map<string, any> = new Map();
  
  constructor() {
    // Connect to ACT server
    this.connectToActServer();
    
    // Start background task processor
    this.startBackgroundProcessor();
    
    // Monitor activity for idle detection
    this.monitorActivity();
  }
  
  private connectToActServer(): void {
    const ACT_SERVER_URL = process.env.ACT_SERVER_URL || 'http://localhost:8080';
    
    console.log(`[SelfImprovementEngine] Connecting to ACT server at ${ACT_SERVER_URL}`);
    
    this.socket = io(ACT_SERVER_URL, {
      transports: ['websocket'],
      reconnection: true,
      reconnectionAttempts: 5,
      reconnectionDelay: 1000,
      timeout: 20000,
      path: '/socket.io/'
    });
    
    this.setupSocketHandlers();
  }
  
  private setupSocketHandlers(): void {
    this.socket.on('connect', () => {
      console.log('[SelfImprovementEngine] Connected to ACT server');
      this.isConnected = true;
      
      // Note: SelfImprovementEngine does NOT register as an agent —
      // it's internal infrastructure, not a coordinating agent.
    });
    
    this.socket.on('disconnect', () => {
      console.log('[SelfImprovementEngine] Disconnected from ACT server');
      this.isConnected = false;
    });
    
    this.socket.on('connect_error', (error) => {
      console.error('[SelfImprovementEngine] Connection error:', error);
      this.isConnected = false;
    });
    
    // Listen for coordination events to build history
    
    // Add listener for agent communication events
    this.socket.on('agent_message', (data) => {
      this.eventHistory.push({
        type: 'agent_message',
        timestamp: new Date(),
        data
      });
      this.trimEventHistory();
    });
    this.socket.on('task_created', (data) => {
      this.eventHistory.push({
        type: 'task_created',
        timestamp: new Date(),
        data
      });
      this.trimEventHistory();
    });
    
    this.socket.on('task_assigned', (data) => {
      this.eventHistory.push({
        type: 'task_assigned',
        timestamp: new Date(),
        data
      });
      this.trimEventHistory();
    });
    
    this.socket.on('task_progress_updated', (data) => {
      this.eventHistory.push({
        type: 'task_progress',
        timestamp: new Date(),
        data
      });
      this.trimEventHistory();
    });
    
    this.socket.on('agent_status_update', (data) => {
      this.eventHistory.push({
        type: 'agent_status',
        timestamp: new Date(),
        data
      });
      this.trimEventHistory();
    });
    
    this.socket.on('conflicts_detected', (data) => {
      this.eventHistory.push({
        type: 'conflict',
        timestamp: new Date(),
        data
      });
      this.trimEventHistory();
    });
    
    this.socket.on('agent_performance_updated', (data) => {
      // Update agent profiles with performance data
      if (data.agentId) {
        this.agentProfiles.set(data.agentId, {
          ...this.agentProfiles.get(data.agentId),
          ...data
        });
      }
    });
  }
  
  private trimEventHistory(): void {
    // Keep only last 1000 events
    if (this.eventHistory.length > 1000) {
      this.eventHistory = this.eventHistory.slice(-1000);
    }
  }
  
  /**
   * Explicit trigger: User-controlled surgical precision improvement
   * @param request Improvement request with surgical precision parameters
   * @returns Improvement result
   */
  async triggerExplicitImprovement(request: ImprovementRequest): Promise<ImprovementResult> {
    console.log(`[SelfImprovementEngine] Explicit improvement triggered for scope: ${request.scope}`);
    
    const startTime = Date.now();
    
    // Analyze coordination patterns based on scope
    const analysis = await this.analyzeCoordinationPatterns(request);
    
    // Generate targeted recommendations
    const recommendations = this.generateRecommendationsFromAnalysis(request, analysis);
    
    // Calculate metrics
    const metrics = this.calculateImprovementMetrics(analysis);
    
    const result: ImprovementResult = {
      requestId: uuidv4(),
      scope: request.scope,
      analysis: analysis.summary,
      recommendations,
      metrics,
      timestamp: new Date().toISOString(),
      executionTime: Date.now() - startTime,
      outputFormat: request.output
    };
    
    console.log(`[SelfImprovementEngine] Explicit improvement completed in ${result.executionTime}ms`);
    
    // Store improvement result in ACT server
    if (this.isConnected) {
      this.socket.emit('broadcast_coordination_event', {
        type: 'improvement_analysis',
        data: {
          requestId: result.requestId,
          scope: request.scope,
          analysis: result.analysis,
          recommendations: result.recommendations,
          timestamp: result.timestamp
        }
      });
    }
    
    return result;
  }
  
  /**
   * Implicit trigger: Background improvement during idle periods
   * @param request Improvement request for background processing
   */
  async scheduleBackgroundImprovement(request: ImprovementRequest): Promise<void> {
    console.log(`[SelfImprovementEngine] Scheduling background improvement for scope: ${request.scope}`);
    
    const task: BackgroundTask = {
      taskId: uuidv4(),
      type: 'improvement',
      priority: this.determinePriority(request),
      status: 'pending',
      createdAt: new Date().toISOString(),
      requestData: request as any
    };
    
    this.backgroundTaskQueue.push(task);
    console.log(`[SelfImprovementEngine] Background task scheduled. Queue size: ${this.backgroundTaskQueue.length}`);
  }
  
  /**
   * Process background tasks when system is idle
   */
  private async processBackgroundTasks(): Promise<void> {
    if (this.isProcessing || this.backgroundTaskQueue.length === 0 || !this.isConnected) {
      return;
    }
    
    this.isProcessing = true;
    
    try {
      // Sort tasks by priority
      this.backgroundTaskQueue.sort((a, b) => {
        const priorityOrder = { high: 3, medium: 2, low: 1 };
        return priorityOrder[b.priority] - priorityOrder[a.priority];
      });
      
      // Process highest priority task
      const task = this.backgroundTaskQueue[0];
      if (task && task.status === 'pending') {
        console.log(`[SelfImprovementEngine] Processing background task: ${task.taskId}`);
        
        task.status = 'running';
        task.startedAt = new Date().toISOString();
        
        try {
          // Execute the actual improvement logic
          const request = task.requestData as ImprovementRequest;
          const analysis = await this.analyzeCoordinationPatterns(request);
          
          task.result = {
            success: true,
            analysis: analysis.summary,
            recommendations: this.generateRecommendationsFromAnalysis(request, analysis),
            metrics: this.calculateImprovementMetrics(analysis)
          };
          
          task.status = 'completed';
          task.completedAt = new Date().toISOString();
          
          console.log(`[SelfImprovementEngine] Background task completed: ${task.taskId}`);
          
          // Store result in ACT server
          if (this.isConnected) {
            this.socket.emit('broadcast_coordination_event', {
              type: 'background_improvement',
              data: {
                taskId: task.taskId,
                scope: request.scope,
                analysis: analysis.summary,
                recommendations: task.result.recommendations,
                timestamp: task.completedAt
              }
            });
          }
        } catch (error: any) {
          task.status = 'failed';
          task.completedAt = new Date().toISOString();
          task.result = {
            success: false,
            error: error.message
          };
          
          console.error(`[SelfImprovementEngine] Background task failed: ${task.taskId}`, error);
        }
        
        // Remove completed task
        this.backgroundTaskQueue.shift();
      }
    } finally {
      this.isProcessing = false;
    }
  }
  
  /**
   * Analyze coordination patterns based on the improvement request
   */
  private async analyzeCoordinationPatterns(request: ImprovementRequest): Promise<{summary: string, details: any}> {
    // Filter events based on request parameters
    let filteredEvents = [...this.eventHistory];
    
    // Filter by agents if specified
    if (request.agents && request.agents.length > 0) {
      filteredEvents = filteredEvents.filter(event => 
        request.agents?.includes(event.data.agentId) || 
        request.agents?.includes(event.data.task?.assignedAgent)
      );
    }
    
    // Filter by session if specified
    if (request.session) {
      // In a real implementation, this would filter by project/session
      // For now, we'll just note that session filtering is requested
    }
    
    // Analyze patterns based on scope
    let summary = '';
    let details = {};
    
    switch (request.scope) {
      case 'communication':
        summary = await this.analyzeCommunicationPatterns(filteredEvents);
        break;
      case 'tools':
        summary = await this.analyzeToolUsagePatterns(filteredEvents);
        break;
      case 'assignments':
        summary = await this.analyzeAssignmentPatterns(filteredEvents);
        break;
      case 'conflicts':
        summary = await this.analyzeConflictPatterns(filteredEvents);
        break;
      case 'collaboration':
        summary = await this.analyzeCollaborationPatterns(filteredEvents);
        break;
      case 'performance':
        summary = await this.analyzePerformancePatterns(filteredEvents);
        break;
      case 'knowledge':
        summary = await this.analyzeKnowledgePatterns(filteredEvents);
        break;
      default:
        summary = 'No specific analysis available for this scope.';
    }
    
    return {
      summary,
      details
    };
  }
  
  private async analyzeCommunicationPatterns(events: any[]): Promise<string> {
    const communicationEvents = events.filter(e => e.type === 'agent_message');
    const totalMessages = communicationEvents.length;
    
    if (totalMessages === 0) {
      return 'No communication events found in the analyzed period.';
    }
    
    // Count messages per agent
    const agentMessageCount: Record<string, number> = {};
    communicationEvents.forEach(event => {
      const agent = event.data.sender;
      agentMessageCount[agent] = (agentMessageCount[agent] || 0) + 1;
    });
    
    const agents = Object.keys(agentMessageCount);
    const avgMessages = totalMessages / agents.length;
    
    return `Analyzed ${totalMessages} communication events across ${agents.length} agents. ` +
           `Average messages per agent: ${avgMessages.toFixed(1)}. ` +
           `Most active agent: ${Object.entries(agentMessageCount).sort((a, b) => b[1] - a[1])[0][0]} ` +
           `with ${Math.max(...Object.values(agentMessageCount))} messages.`;
  }
  
  private async analyzeToolUsagePatterns(events: any[]): Promise<string> {
    // In a real implementation, this would analyze tool usage patterns
    // For now, we'll provide a placeholder
    return 'Tool usage analysis not yet implemented. Would analyze which tools are most/least effective.';
  }
  
  private async analyzeAssignmentPatterns(events: any[]): Promise<string> {
    const assignmentEvents = events.filter(e => e.type === 'task_assigned');
    const totalAssignments = assignmentEvents.length;
    
    if (totalAssignments === 0) {
      return 'No task assignment events found in the analyzed period.';
    }
    
    // Count assignments per agent
    const agentAssignmentCount: Record<string, number> = {};
    assignmentEvents.forEach(event => {
      const agent = event.data.agentId;
      agentAssignmentCount[agent] = (agentAssignmentCount[agent] || 0) + 1;
    });
    
    const agents = Object.keys(agentAssignmentCount);
    const avgAssignments = totalAssignments / agents.length;
    
    return `Analyzed ${totalAssignments} task assignments across ${agents.length} agents. ` +
           `Average assignments per agent: ${avgAssignments.toFixed(1)}. ` +
           `Most assigned agent: ${Object.entries(agentAssignmentCount).sort((a, b) => b[1] - a[1])[0][0]} ` +
           `with ${Math.max(...Object.values(agentAssignmentCount))} assignments.`;
  }
  
  private async analyzeConflictPatterns(events: any[]): Promise<string> {
    const conflictEvents = events.filter(e => e.type === 'conflict');
    const totalConflicts = conflictEvents.length;
    
    if (totalConflicts === 0) {
      return 'No conflict events found in the analyzed period.';
    }
    
    // Count conflicts by type
    const conflictTypeCount: Record<string, number> = {};
    conflictEvents.forEach(event => {
      const type = event.data.conflicts?.[0]?.type || 'unknown';
      conflictTypeCount[type] = (conflictTypeCount[type] || 0) + 1;
    });
    
    const types = Object.keys(conflictTypeCount);
    
    return `Analyzed ${totalConflicts} conflicts across ${types.length} types. ` +
           `Conflict types: ${types.join(', ')}. ` +
           `Most common: ${Object.entries(conflictTypeCount).sort((a, b) => b[1] - a[1])[0][0]} ` +
           `with ${Math.max(...Object.values(conflictTypeCount))} occurrences.`;
  }
  
  private async analyzeCollaborationPatterns(events: any[]): Promise<string> {
    // In a real implementation, this would analyze collaboration patterns
    // For now, we'll provide a placeholder
    return 'Collaboration pattern analysis not yet implemented. Would analyze how agents work together.';
  }
  
  private async analyzePerformancePatterns(events: any[]): Promise<string> {
    const performanceEvents = events.filter(e => e.type === 'task_progress');
    const totalTasks = performanceEvents.length;
    
    if (totalTasks === 0) {
      return 'No performance events found in the analyzed period.';
    }
    
    // Calculate average completion time
    let totalCompletionTime = 0;
    let completedTasks = 0;
    
    // Group events by task ID
    const taskEvents: Record<string, any[]> = {};
    performanceEvents.forEach(event => {
      const taskId = event.data.taskId;
      if (!taskEvents[taskId]) {
        taskEvents[taskId] = [];
      }
      taskEvents[taskId].push(event);
    });
    
    // Calculate completion times
    Object.values(taskEvents).forEach(events => {
      const startEvent = events.find(e => e.data.progress === 0 || e.data.progress === 25);
      const endEvent = events.find(e => e.data.progress === 100);
      
      if (startEvent && endEvent) {
        const startTime = new Date(startEvent.timestamp).getTime();
        const endTime = new Date(endEvent.timestamp).getTime();
        totalCompletionTime += (endTime - startTime);
        completedTasks++;
      }
    });
    
    const avgCompletionTime = completedTasks > 0 ? totalCompletionTime / completedTasks : 0;
    
    return `Analyzed ${totalTasks} performance events across ${Object.keys(taskEvents).length} tasks. ` +
           `Average completion time: ${avgCompletionTime > 0 ? (avgCompletionTime / 1000 / 60).toFixed(1) + ' minutes' : 'not available'}. ` +
           `Completed tasks: ${completedTasks}.`;
  }
  
  private async analyzeKnowledgePatterns(events: any[]): Promise<string> {
    // In a real implementation, this would analyze knowledge patterns
    // For now, we'll provide a placeholder
    return 'Knowledge pattern analysis not yet implemented. Would analyze what the system has learned.';
  }
  
  /**
   * Generate recommendations based on analysis
   */
  private generateRecommendationsFromAnalysis(request: ImprovementRequest, analysis: {summary: string, details: any}): string[] {
    const recommendations: string[] = [];
    
    switch (request.scope) {
      case 'communication':
        recommendations.push(
          'Enhance communication patterns between agents',
          'Implement structured message formats for better clarity',
          'Add automatic summarization for long conversations'
        );
        break;
      case 'tools':
        recommendations.push(
          'Optimize tool selection algorithms',
          'Add caching for frequently used tools',
          'Implement tool performance monitoring'
        );
        break;
      case 'assignments':
        recommendations.push(
          'Improve task assignment algorithms',
          'Add load balancing for agent workloads',
          'Implement skill-based matching'
        );
        break;
      case 'conflicts':
        recommendations.push(
          'Enhance conflict resolution strategies',
          'Add proactive conflict detection',
          'Implement mediation workflows'
        );
        break;
      case 'collaboration':
        recommendations.push(
          'Improve collaboration patterns between agents',
          'Add shared workspace features',
          'Implement collaborative decision-making workflows'
        );
        break;
      case 'performance':
        recommendations.push(
          'Optimize system performance and resource usage',
          'Add performance monitoring and alerting',
          'Implement caching strategies'
        );
        break;
      case 'knowledge':
        recommendations.push(
          'Enhance knowledge sharing between agents',
          'Implement knowledge base indexing',
          'Add semantic search capabilities'
        );
        break;
    }
    
    return recommendations;
  }
  
  /**
   * Calculate improvement metrics
   */
  private calculateImprovementMetrics(analysis: {summary: string, details: any}): Record<string, number> {
    // In a real implementation, this would calculate actual metrics
    // For now, we'll provide placeholder metrics
    return {
      'improvement_score': 0.85,
      'confidence_level': 0.92,
      'data_points_analyzed': this.eventHistory.length,
      'recommendations_generated': 3
    };
  }
  
  /**
   * Start background processor that runs during idle periods
   */
  private startBackgroundProcessor(): void {
    // Check for background tasks every 30 seconds
    setInterval(() => {
      const idleTime = Date.now() - this.lastActivityTimestamp;
      
      // Process background tasks if idle for more than 5 minutes
      if (idleTime > 5 * 60 * 1000) {
        this.processBackgroundTasks();
      }
    }, 30 * 1000);
  }
  
  /**
   * Monitor activity to detect idle periods
   */
  private monitorActivity(): void {
    // Update last activity timestamp on any activity
    const updateActivity = () => {
      this.lastActivityTimestamp = Date.now();
    };
    
    // Monitor various activity sources
    if (typeof process !== 'undefined') {
      process.on('message', updateActivity);
    }
    
    // Update activity on console output
    const originalLog = console.log;
    console.log = (...args: any[]) => {
      updateActivity();
      originalLog(...args);
    };
  }
  
  /**
   * Determine priority for background task
   */
  private determinePriority(request: ImprovementRequest): 'low' | 'medium' | 'high' {
    // Higher priority for critical scopes
    if (request.scope === 'performance' || request.scope === 'conflicts') {
      return 'high';
    }
    
    // Medium priority for important scopes
    if (request.scope === 'assignments' || request.scope === 'tools') {
      return 'medium';
    }
    
    // Low priority for other scopes
    return 'low';
  }
  
  /**
   * Get current status of the improvement engine
   */
  getStatus(): {
    queueSize: number;
    isProcessing: boolean;
    lastActivity: number;
    idleTime: number;
    isConnected: boolean;
    eventHistorySize: number;
  } {
    return {
      queueSize: this.backgroundTaskQueue.length,
      isProcessing: this.isProcessing,
      lastActivity: this.lastActivityTimestamp,
      idleTime: Date.now() - this.lastActivityTimestamp,
      isConnected: this.isConnected,
      eventHistorySize: this.eventHistory.length
    };
  }
  
  /**
   * Get agent profiles
   */
  getAgentProfiles(): Map<string, any> {
    return new Map(this.agentProfiles);
  }
  
  /**
   * Close connection
   */
  async close(): Promise<void> {
    if (this.socket) {
      this.socket.close();
    }
  }
}