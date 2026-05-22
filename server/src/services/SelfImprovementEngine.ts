import { EventEmitter } from 'events';
import { v4 as uuidv4 } from 'uuid';
import { AgentRegistry } from './AgentRegistry';
import { TaskCoordinator } from './TaskCoordinator';
import { EventHub, CoordinationEvent } from './EventHub';
import { logger } from '../utils/logger';

export interface ImprovementRequest {
  scope: 'communication' | 'tools' | 'assignments' | 'conflicts' | 'collaboration' | 'performance' | 'knowledge';
  agents?: string[];
  session?: string;
  filter?: 'good' | 'bad' | 'all';
  output: 'summary' | 'detailed-report' | 'action-items';
}

export interface ImprovementResult {
  requestId: string;
  scope: string;
  analysis: string;
  recommendations: string[];
  metrics: Record<string, number>;
  timestamp: string;
  executionTime: number;
  outputFormat: string;
}

export interface BackgroundTask {
  taskId: string;
  type: 'improvement';
  priority: 'low' | 'medium' | 'high';
  status: 'pending' | 'running' | 'completed' | 'failed';
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  requestData: any;
  result?: {
    success: boolean;
    analysis?: string;
    recommendations?: string[];
    metrics?: Record<string, number>;
    error?: string;
  };
}

export class SelfImprovementEngine extends EventEmitter {
  private backgroundTaskQueue: BackgroundTask[] = [];
  private isProcessing: boolean = false;
  private lastActivityTimestamp: number = Date.now();
  private eventHistory: CoordinationEvent[] = [];
  private agentProfiles: Map<string, any> = new Map();
  private agentRegistry: AgentRegistry;
  private taskCoordinator: TaskCoordinator;
  private eventHub: EventHub;
  
  constructor(agentRegistry: AgentRegistry, taskCoordinator: TaskCoordinator, eventHub: EventHub) {
    super();
    this.agentRegistry = agentRegistry;
    this.taskCoordinator = taskCoordinator;
    this.eventHub = eventHub;
    
    // Start background task processor
    this.startBackgroundProcessor();
    
    // Monitor activity for idle detection
    this.monitorActivity();
    
    // Listen to coordination events from EventHub
    this.setupEventListeners();
  }
  
  private setupEventListeners(): void {
    // Listen for coordination events to build history
    this.eventHub.on('agent_registered', (data) => {
      logger.debug('[SelfImprovementEngine] Collected agent_registered event:', data?.agent?.name);
      this.eventHistory.push({
        type: 'agent_registered',
        agentId: data.agent.id,
        taskId: undefined,
        data,
        timestamp: new Date()
      });
      this.trimEventHistory();
    });
    
    this.eventHub.on('agent_status_updated', (data) => {
      logger.debug('[SelfImprovementEngine] Collected agent_status_updated event:', data?.agentId, data?.status);
      this.eventHistory.push({
        type: 'agent_status',
        agentId: data.agentId,
        taskId: undefined,
        data,
        timestamp: new Date()
      });
      this.trimEventHistory();
    });
    
    this.eventHub.on('agent_performance_updated', (data) => {
      logger.debug('[SelfImprovementEngine] Collected agent_performance_updated event:', data?.agentId);
      this.eventHistory.push({
        type: 'agent_performance',
        agentId: data.agentId,
        taskId: undefined,
        data,
        timestamp: new Date()
      });
      this.trimEventHistory();
      
      // Update agent profiles with performance data
      if (data.agentId) {
        this.agentProfiles.set(data.agentId, {
          ...this.agentProfiles.get(data.agentId),
          ...data
        });
      }
    });
    
    this.eventHub.on('task_created', (data) => {
      logger.debug('[SelfImprovementEngine] Collected task_created event:', data?.task?.description?.substring(0, 50) || 'no description');
      this.eventHistory.push({
        type: 'task_created',
        agentId: undefined,
        taskId: data.task.id,
        data,
        timestamp: new Date()
      });
      this.trimEventHistory();
    });
    
    this.eventHub.on('task_assigned', (data) => {
      logger.debug('[SelfImprovementEngine] Collected task_assigned event:', data?.task?.description?.substring(0, 50) || 'no description');
      this.eventHistory.push({
        type: 'task_assigned',
        agentId: data.agentId,
        taskId: data.task.id,
        data,
        timestamp: new Date()
      });
      this.trimEventHistory();
    });
    
    this.eventHub.on('task_progress_updated', (data) => {
      logger.debug('[SelfImprovementEngine] Collected task_progress_updated event:', data?.taskId, data?.progress + '%');
      this.eventHistory.push({
        type: 'task_progress',
        agentId: undefined,
        taskId: data.taskId,
        data,
        timestamp: new Date()
      });
      this.trimEventHistory();
    });
    
    this.eventHub.on('conflicts_detected', (data) => {
      logger.debug('[SelfImprovementEngine] Collected conflicts_detected event:', data?.conflicts?.length || 0, 'conflicts');
      this.eventHistory.push({
        type: 'conflict',
        agentId: undefined,
        taskId: undefined,
        data,
        timestamp: new Date()
      });
      this.trimEventHistory();
    });
    
    // Add listener for agent communication events
    this.eventHub.on('agent_message', (data) => {
      logger.debug('[SelfImprovementEngine] Collected agent_message event from', data?.sender?.substring(0, 20) || 'unknown');
      this.eventHistory.push({
        type: 'agent_message',
        agentId: data.sender,
        taskId: undefined,
        data,
        timestamp: new Date()
      });
      this.trimEventHistory();
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
    logger.info(`[SelfImprovementEngine] Explicit improvement triggered for scope: ${request.scope}`);
    
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
    
    logger.info(`[SelfImprovementEngine] Explicit improvement completed in ${result.executionTime}ms`);
    
    // Store improvement result in EventHub
    await this.eventHub.broadcastCoordinationEvent('improvement_analysis', {
      requestId: result.requestId,
      scope: request.scope,
      analysis: result.analysis,
      recommendations: result.recommendations,
      timestamp: result.timestamp
    });
    
    return result;
  }
  
  /**
   * Implicit trigger: Background improvement during idle periods
   * @param request Improvement request for background processing
   */
  async scheduleBackgroundImprovement(request: ImprovementRequest): Promise<void> {
    logger.info(`[SelfImprovementEngine] Scheduling background improvement for scope: ${request.scope}`);
    
    const task: BackgroundTask = {
      taskId: uuidv4(),
      type: 'improvement',
      priority: this.determinePriority(request),
      status: 'pending',
      createdAt: new Date().toISOString(),
      requestData: request as any
    };
    
    this.backgroundTaskQueue.push(task);
    logger.info(`[SelfImprovementEngine] Background task scheduled. Queue size: ${this.backgroundTaskQueue.length}`);
  }
  
  /**
   * Process background tasks when system is idle
   */
  private async processBackgroundTasks(): Promise<void> {
    if (this.isProcessing || this.backgroundTaskQueue.length === 0) {
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
        logger.info(`[SelfImprovementEngine] Processing background task: ${task.taskId}`);
        
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
          
          logger.info(`[SelfImprovementEngine] Background task completed: ${task.taskId}`);
          
          // Store result in EventHub
          await this.eventHub.broadcastCoordinationEvent('background_improvement', {
            taskId: task.taskId,
            scope: request.scope,
            analysis: analysis.summary,
            recommendations: task.result.recommendations,
            timestamp: task.completedAt
          });
        } catch (error: any) {
          task.status = 'failed';
          task.completedAt = new Date().toISOString();
          task.result = {
            success: false,
            error: error.message
          };
          
          logger.error(`[SelfImprovementEngine] Background task failed: ${task.taskId}`, error);
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
    logger.debug('[SelfImprovementEngine] Analyzing coordination patterns, total events:', this.eventHistory.length);
    
    // Filter events based on request parameters
    let filteredEvents = [...this.eventHistory];
    
    // Filter by agents if specified
    if (request.agents && request.agents.length > 0) {
      filteredEvents = filteredEvents.filter(event => 
        request.agents?.includes(event.agentId || '') || 
        request.agents?.includes(event.data?.task?.assignedAgent || '')
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
  
  private async analyzeCommunicationPatterns(events: CoordinationEvent[]): Promise<string> {
    const communicationEvents = events.filter(e => e.type === 'agent_message');
    const totalMessages = communicationEvents.length;
    
    logger.debug('[SelfImprovementEngine] Found communication events:', totalMessages);
    
    if (totalMessages === 0) {
      return 'No communication events found in the analyzed period.';
    }
    
    // Count messages per agent
    const agentMessageCount: Record<string, number> = {};
    communicationEvents.forEach(event => {
      const agent = event.data.sender || 'unknown';
      agentMessageCount[agent] = (agentMessageCount[agent] || 0) + 1;
    });
    
    const agents = Object.keys(agentMessageCount);
    const avgMessages = totalMessages / agents.length;
    
    return `Analyzed ${totalMessages} communication events across ${agents.length} agents. ` +
           `Average messages per agent: ${avgMessages.toFixed(1)}. ` +
           `Most active agent: ${Object.entries(agentMessageCount).sort((a, b) => b[1] - a[1])[0][0]} ` +
           `with ${Math.max(...Object.values(agentMessageCount))} messages.`;
  }
  
  private async analyzeToolUsagePatterns(events: CoordinationEvent[]): Promise<string> {
    // Tool events surface either as discrete tool_call/tool_result types or as
    // task_progress events containing tool annotations in event.data.tool.
    const toolEvents = events.filter(e =>
      e.type === 'tool_call' ||
      e.type === 'tool_result' ||
      (e as any).data?.tool
    );
    if (toolEvents.length === 0) {
      return 'No tool usage events found in the analyzed period.';
    }

    const toolCount: Record<string, number> = {};
    const toolErrors: Record<string, number> = {};
    for (const e of toolEvents) {
      const tool = (e as any).data?.tool || (e as any).tool || 'unknown';
      toolCount[tool] = (toolCount[tool] || 0) + 1;
      const isError = (e as any).data?.isError || /error|failed/i.test((e as any).message || '');
      if (isError) toolErrors[tool] = (toolErrors[tool] || 0) + 1;
    }

    const sorted = Object.entries(toolCount).sort((a, b) => b[1] - a[1]);
    const top = sorted.slice(0, 3).map(([t, c]) => {
      const errs = toolErrors[t] || 0;
      const errRate = ((errs / c) * 100).toFixed(0);
      return `${t} (${c} calls, ${errRate}% error rate)`;
    }).join(', ');

    return `Analyzed ${toolEvents.length} tool events across ${sorted.length} tools. ` +
           `Most used: ${top}.`;
  }
  
  private async analyzeAssignmentPatterns(events: CoordinationEvent[]): Promise<string> {
    const assignmentEvents = events.filter(e => e.type === 'task_assigned');
    const totalAssignments = assignmentEvents.length;
    
    if (totalAssignments === 0) {
      return 'No task assignment events found in the analyzed period.';
    }
    
    // Count assignments per agent
    const agentAssignmentCount: Record<string, number> = {};
    assignmentEvents.forEach(event => {
      const agent = event.agentId || 'unknown';
      agentAssignmentCount[agent] = (agentAssignmentCount[agent] || 0) + 1;
    });
    
    const agents = Object.keys(agentAssignmentCount);
    const avgAssignments = totalAssignments / agents.length;
    
    return `Analyzed ${totalAssignments} task assignments across ${agents.length} agents. ` +
           `Average assignments per agent: ${avgAssignments.toFixed(1)}. ` +
           `Most assigned agent: ${Object.entries(agentAssignmentCount).sort((a, b) => b[1] - a[1])[0][0]} ` +
           `with ${Math.max(...Object.values(agentAssignmentCount))} assignments.`;
  }
  
  private async analyzeConflictPatterns(events: CoordinationEvent[]): Promise<string> {
    const conflictEvents = events.filter(e => e.type === 'conflict');
    const totalConflicts = conflictEvents.length;
    
    if (totalConflicts === 0) {
      return 'No conflict events found in the analyzed period.';
    }
    
    // Count conflicts by type
    const conflictTypeCount: Record<string, number> = {};
    conflictEvents.forEach(event => {
      const type = (event.data.conflicts && event.data.conflicts[0] && event.data.conflicts[0].type) || 'unknown';
      conflictTypeCount[type] = (conflictTypeCount[type] || 0) + 1;
    });
    
    const types = Object.keys(conflictTypeCount);
    
    return `Analyzed ${totalConflicts} conflicts across ${types.length} types. ` +
           `Conflict types: ${types.join(', ')}. ` +
           `Most common: ${Object.entries(conflictTypeCount).sort((a, b) => b[1] - a[1])[0][0]} ` +
           `with ${Math.max(...Object.values(conflictTypeCount))} occurrences.`;
  }
  
  private async analyzeCollaborationPatterns(events: CoordinationEvent[]): Promise<string> {
    // Two agents collaborate when they both appear on events sharing a taskId.
    // Build the agent set per task, then count distinct unordered pairs.
    const agentsByTask: Map<string, Set<string>> = new Map();
    for (const e of events) {
      const taskId = (e as any).taskId || (e as any).data?.taskId || (e as any).data?.task?.id;
      if (!taskId) continue;
      const agent = e.agentId || (e as any).data?.agentId || (e as any).data?.assignedAgent;
      if (!agent) continue;
      if (!agentsByTask.has(taskId)) agentsByTask.set(taskId, new Set());
      agentsByTask.get(taskId)!.add(agent);
    }

    const pairCount: Map<string, number> = new Map();
    for (const agentSet of agentsByTask.values()) {
      if (agentSet.size < 2) continue;
      const agents = Array.from(agentSet).sort();
      for (let i = 0; i < agents.length; i++) {
        for (let j = i + 1; j < agents.length; j++) {
          const key = `${agents[i]}|${agents[j]}`;
          pairCount.set(key, (pairCount.get(key) || 0) + 1);
        }
      }
    }

    if (pairCount.size === 0) {
      return 'No multi-agent task collaborations found in the analyzed period.';
    }

    const sorted = Array.from(pairCount.entries()).sort((a, b) => b[1] - a[1]);
    const top = sorted.slice(0, 3)
      .map(([pair, count]) => `${pair.replace('|', ' + ')} (${count} shared tasks)`)
      .join(', ');

    return `Detected ${pairCount.size} distinct agent pairs collaborating across ${agentsByTask.size} multi-agent tasks. ` +
           `Top pairs: ${top}.`;
  }
  
  private async analyzePerformancePatterns(events: CoordinationEvent[]): Promise<string> {
    const performanceEvents = events.filter(e => e.type === 'task_progress');
    const totalTasks = performanceEvents.length;
    
    if (totalTasks === 0) {
      return 'No performance events found in the analyzed period.';
    }
    
    // Calculate average completion time
    let totalCompletionTime = 0;
    let completedTasks = 0;
    
    // Group events by task ID
    const taskEvents: Record<string, CoordinationEvent[]> = {};
    performanceEvents.forEach(event => {
      const taskId = event.taskId;
      if (taskId !== undefined && taskId !== null && !taskEvents[taskId]) {
        taskEvents[taskId] = [];
      }
      if (taskId !== undefined && taskId !== null) {
        taskEvents[taskId].push(event);
      }
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
  
  private async analyzeKnowledgePatterns(events: CoordinationEvent[]): Promise<string> {
    // peer_response events are the primary knowledge-transfer surface. Count
    // outbound responses per agent and identify recurring topics via simple
    // keyword frequency on the response text.
    const peerEvents = events.filter(e => e.type === 'peer_response');
    if (peerEvents.length === 0) {
      return 'No peer-response events found in the analyzed period.';
    }

    // Count responses per agent (who is answering)
    const responseCount: Record<string, number> = {};
    const wordFreq: Record<string, number> = {};
    const STOP = new Set(['the','a','an','is','was','are','were','be','been','being','have','has','had','do','does','did','will','would','shall','should','may','might','must','can','could','of','in','to','for','with','on','at','by','from','as','into','through','during','before','after','above','below','between','and','but','or','nor','not','so','yet','both','either','neither','each','every','all','any','few','more','most','other','some','such','no','only','own','same','than','too','very','just','that','this','it','its','i','we','you','he','she','they','me','him','her','us','them','my','your','his','our','their','if','then','else','when','where','why','how','which','who','whom','what']);

    for (const e of peerEvents) {
      const agent = e.agentId || 'unknown';
      responseCount[agent] = (responseCount[agent] || 0) + 1;
      const words = ((e as any).message || '').toLowerCase().split(/\W+/).filter((w: string) => w.length > 3 && !STOP.has(w));
      for (const w of words) {
        wordFreq[w] = (wordFreq[w] || 0) + 1;
      }
    }

    const topResponders = Object.entries(responseCount).sort((a, b) => b[1] - a[1]).slice(0, 3)
      .map(([a, c]) => `${a} (${c} responses)`).join(', ');
    const topTopics = Object.entries(wordFreq).sort((a, b) => b[1] - a[1]).slice(0, 5)
      .map(([w]) => w).join(', ');

    return `Analyzed ${peerEvents.length} peer-response events. ` +
           `Top responders: ${topResponders}. ` +
           `Recurring topics: ${topTopics || 'none significant'}.`;
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
  private calculateImprovementMetrics(analysis: {summary: string, details: any}, recommendationsCount?: number): Record<string, number> {
    // improvement_score: ratio of successful outcomes (validation passes) to
    // total completed tasks in the analysis window. Derived from eventHistory
    // since the analysis object doesn't carry raw counts.
    const completed = this.eventHistory.filter(e => e.type === 'task_completed').length;
    const validated = this.eventHistory.filter(e => e.type === 'task_validated').length;
    const improvementScore = completed > 0 ? validated / completed : 0;

    // confidence_level: log-scale function of sample size, saturates ~0.95 for >100 events
    const n = this.eventHistory.length;
    const confidenceLevel = Math.min(0.95, Math.log10(Math.max(1, n)) / 2);

    // recommendations_generated: from optional param, or count from details if available
    let recsGenerated = recommendationsCount ?? 0;
    if (!recsGenerated && analysis.details?.recommendations) {
      recsGenerated = Array.isArray(analysis.details.recommendations)
        ? analysis.details.recommendations.length
        : 0;
    }

    return {
      'improvement_score': Math.round(improvementScore * 100) / 100,
      'confidence_level': Math.round(confidenceLevel * 100) / 100,
      'data_points_analyzed': n,
      'recommendations_generated': recsGenerated
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
    eventHistorySize: number;
  } {
    return {
      queueSize: this.backgroundTaskQueue.length,
      isProcessing: this.isProcessing,
      lastActivity: this.lastActivityTimestamp,
      idleTime: Date.now() - this.lastActivityTimestamp,
      eventHistorySize: this.eventHistory.length
    };
  }
  
  /**
   * Get agent profiles
   */
  getAgentProfiles(): Map<string, any> {
    return new Map(this.agentProfiles);
  }
}

export default SelfImprovementEngine;
