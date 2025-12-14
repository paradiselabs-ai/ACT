/**
 * ACT SSE Widget System
 * Lightweight HTML client for consuming ACT coordination events via Server-Sent Events
 */

class ACTWidgetSystem {
    constructor(serverUrl = 'http://localhost:8080') {
        this.serverUrl = serverUrl;
        this.socket = null;
        this.isConnected = false;

        // State
        this.agents = new Map();
        this.tasks = new Map();
        this.projects = new Map();
        this.conflicts = new Map();
        this.activities = [];
        this.messages = [];

        this.init();
    }

    init() {
        this.setupEventListeners();
        this.connectToServer();
    }

    setupEventListeners() {
        // Connection status updates
        document.addEventListener('act:connected', () => {
            this.updateConnectionStatus(true);
        });
        document.addEventListener('act:disconnected', () => {
            this.updateConnectionStatus(false);
        });
    }

    connectToServer() {
        try {
            console.log(`🔌 Connecting to ACT server via WebSocket at ${this.serverUrl}`);

            this.socket = io(this.serverUrl);

            this.socket.on('connect', () => {
                console.log('✅ Connected to ACT server');
                this.isConnected = true;
                document.dispatchEvent(new Event('act:connected'));
                this.clearErrorMessage();

                // Reset state on reconnect/reload
                this.agents.clear();
                this.tasks.clear();
                this.projects.clear();
                this.conflicts.clear();
                this.activities = [];
                this.messages = [];
                this.updateAgentRegistry();
                this.updateTaskCoordinator();
                this.updateProjectStatus();
                this.updateConflictResolution();
                this.updateActivityFeed();
                this.updateAgentMessages();

                // Request current state
                this.socket.emit('get_agent_registry');
                this.socket.emit('get_tasks');
                this.socket.emit('get_project_status');
            });

            this.socket.on('disconnect', () => {
                console.error('❌ WebSocket disconnected');
                this.isConnected = false;
                this.showConnectionError();
            });

            // Event handlers
            this.socket.on('agent_registered', (e) => {
                this.handleAgentRegistered(e.agent || e);
            });

            this.socket.on('agent_joined', (e) => {
                this.handleAgentRegistered(e.agent || e);
            });

            this.socket.on('agent_status_updated', (e) => {
                // Refresh registry on status updates
                this.socket.emit('get_agent_registry');
            });

            this.socket.on('task_created', (e) => {
                this.handleTaskCreated(e);
            });

            this.socket.on('task_assigned', (e) => {
                this.handleTaskAssigned(e);
            });

            this.socket.on('task_pending', (e) => {
                this.handleTaskPending(e);
            });

            this.socket.on('task_progress', (e) => {
                this.handleTaskProgress(e);
            });

            this.socket.on('task_completed', (e) => {
                this.handleTaskCompleted(e);
            });

            this.socket.on('agent_status_update', () => {
                // Refresh agent registry on status changes
                this.socket.emit('get_agent_registry');
            });

            this.socket.on('agent_message', (e) => {
                this.handleAgentMessage(e);
            });

            // Request project status updates
            this.socket.on('project_status_update', (e) => {
                this.handleProjectStatusUpdate(e);
            });

            this.socket.on('conflict_detected', (e) => {
                // e may be array from server
                const conflicts = Array.isArray(e) ? e : [e];
                conflicts.forEach(conflict => this.handleConflictDetected(conflict));
            });

        } catch (error) {
            console.error('Failed to connect to server:', error);
            this.showConnectionError();
        }
    }

    showConnectionError() {
        this.isConnected = false;
        document.dispatchEvent(new Event('act:disconnected'));

        const container = document.getElementById('agentRegistry');
        const errorHTML = `
            <div style="background: #7f1d1d; border: 1px solid #991b1b; border-radius: 4px; padding: 1.5rem; color: #fca5a5;">
                <div style="font-weight: 600; margin-bottom: 0.5rem; color: #fecaca; font-size: 1rem;">⚠️ Connection Error</div>
                <div style="margin-bottom: 1rem; font-size: 0.95rem;">Unable to connect to ACT server at <code style="background: #5f1717; padding: 0.25rem 0.5rem; border-radius: 2px; font-family: monospace;">${this.serverUrl}</code></div>

                <div style="margin-bottom: 1rem;">
                    <strong style="color: #fed7aa;">Troubleshooting Steps:</strong>
                    <ol style="margin-top: 0.5rem; padding-left: 1.5rem; font-size: 0.9rem;">
                        <li>Verify ACT server is running on <code style="background: #5f1717; padding: 0.25rem 0.5rem; border-radius: 2px; font-family: monospace;">localhost:8080</code></li>
                        <li>Check firewall settings allow local connections</li>
                        <li>Ensure no port conflicts on port 8080</li>
                        <li>Restart the ACT server and refresh this page</li>
                    </ol>
                </div>

                <div style="border-top: 1px solid #991b1b; padding-top: 1rem;">
                    <a href="https://docs.agentmix.dev/act/troubleshooting" target="_blank" style="color: #fca5a5; text-decoration: underline;">View detailed troubleshooting guide →</a>
                </div>
            </div>
        `;

        container.innerHTML = errorHTML;

        // Show similar error in all widgets
        const widgets = ['taskCoordinator', 'projectStatus', 'conflictResolution'];
        widgets.forEach(widgetId => {
            const widget = document.getElementById(widgetId);
            if (widget) {
                widget.innerHTML = '<p style="color: #94a3b8;">Waiting for server connection...</p>';
            }
        });

        // Update activity feed
        const feedContainer = document.getElementById('activityFeed');
        feedContainer.innerHTML = `
            <div style="text-align: center; color: #94a3b8; padding: 2rem;">
                <p>Connection lost. Please check troubleshooting steps above.</p>
            </div>
        `;
    }

    clearErrorMessage() {
        // Once connected, error states will be cleared by incoming events
    }

    // Event Handlers
    handleAgentRegistered(event) {
        const src = event.agent || event || {};
        const id = src.id || src.agentId || src.agent_id;
        const name = src.name || src.agentId || id || 'agent';
        const capabilities = src.capabilities || [];
        const model = src.model || src.modelName || '';
        const provider = src.provider || src.providerName || '';
        const status = src.status || 'active';

        if (!id) return;

        const agent = {
            id,
            name,
            capabilities,
            model,
            provider,
            status,
            timestamp: new Date().toISOString()
        };

        this.agents.set(agent.id, agent);
        this.updateAgentRegistry();
        this.addActivity('agent_registered', `Agent registered: ${agent.name}`, agent);
    }

    handleTaskCreated(event) {
        const src = event.task || event || {};
        const id = event.taskId || event.task_id || src.id;
        const description = src.description || event.description || 'Task';
        const title = src.title || description || 'Untitled Task';

        if (!id) return;

        const task = {
            id,
            title,
            description,
            status: src.status || 'created',
            progress: src.progress || 0,
            timestamp: new Date().toISOString()
        };

        this.tasks.set(task.id, task);
        this.updateTaskCoordinator();
        this.addActivity('task_created', `Task created: ${task.title}`, task);
    }

    handleTaskAssigned(event) {
        const taskId = event.taskId || event.task_id;
        const agentId = event.agentId || event.agent_id;
        const taskPayload = event.task || {};
        const description = taskPayload.description || taskPayload.title || 'Task';
        const title = taskPayload.title || description || 'Untitled Task';

        if (!taskId || !agentId) return;

        const existing = this.tasks.get(taskId) || {
            id: taskId,
            title,
            description,
            status: 'created',
            progress: taskPayload.progress || 0
        };

        const statusVal = (!agentId || agentId === 'unassigned') ? 'pending' : 'assigned';
        existing.status = statusVal;
        existing.assignedAgent = agentId;
        existing.reasoning = event.reason || event.reasoning || '';
        this.tasks.set(taskId, existing);
        this.updateTaskCoordinator();

        const agent = this.agents.get(agentId);
        const agentName = agent ? agent.name : (agentId && agentId !== 'unassigned' ? agentId : 'Unassigned');
        const taskName = existing.title || taskId;

        this.addActivity('task_assigned',
            `Task "${taskName}" assigned to ${agentName}`,
            { taskId, agentId, reasoning: existing.reasoning }
        );
    }

    handleTaskPending(event) {
        const taskPayload = event.task || {};
        const taskId = event.taskId || event.task_id || taskPayload.id;
        if (!taskId) return;

        const description = taskPayload.description || taskPayload.title || 'Task';
        const title = taskPayload.title || description || 'Untitled Task';
        const reason = event.reason || 'Pending (no suitable agent yet)';

        const existing = this.tasks.get(taskId) || {
            id: taskId,
            title,
            description,
            status: 'pending',
            progress: taskPayload.progress || 0
        };

        existing.status = 'pending';
        existing.assignedAgent = 'unassigned';
        existing.reasoning = reason;
        this.tasks.set(taskId, existing);
        this.updateTaskCoordinator();

        this.addActivity('task_pending', `Task "${title}" pending: ${reason}`, { taskId, reason });
    }

    handleTaskProgress(event) {
        const taskId = event.taskId || event.task_id;

        if (this.tasks.has(taskId)) {
            const task = this.tasks.get(taskId);
            task.status = 'in_progress';
            task.progress = event.progress || 0;
            task.message = event.message || '';
            this.updateTaskCoordinator();
        }

        this.addActivity('task_progress',
            `Progress: ${event.progress || 0}%`,
            event
        );
    }

    handleTaskCompleted(event) {
        const taskId = event.taskId || event.task_id;

        if (this.tasks.has(taskId)) {
            const task = this.tasks.get(taskId);
            task.status = 'completed';
            task.progress = 100;
            task.result = event.result || '';
            this.updateTaskCoordinator();
        }

        const taskName = this.tasks.has(taskId) ? this.tasks.get(taskId).title : taskId;
        this.addActivity('task_completed',
            `Task completed: ${taskName}`,
            event
        );
    }

    // Agent-to-agent messages
    handleAgentMessage(event) {
        const sender = event.sender || 'Unknown';
        const message = event.message || '';
        if (!message) return;
        this.addActivity('agent_message', `${sender}: ${message}`, event);

        const entry = {
            sender,
            message,
            timestamp: event.timestamp || new Date().toISOString()
        };
        this.messages.unshift(entry);
        if (this.messages.length > 20) this.messages.pop();
        this.updateAgentMessages();
    }

    handleConflictDetected(event) {
        const conflict = {
            id: event.conflictId || `conflict_${Date.now()}`,
            type: event.type || 'unknown',
            agents: event.agents || [],
            description: event.description || '',
            status: 'detected',
            timestamp: new Date().toISOString()
        };

        this.conflicts.set(conflict.id, conflict);
        this.updateConflictResolution();
        this.addActivity('conflict_detected',
            `Conflict detected: ${conflict.type}`,
            conflict
        );
    }

    handleConflictResolved(event) {
        const conflictId = event.conflictId || event.conflict_id;

        if (this.conflicts.has(conflictId)) {
            const conflict = this.conflicts.get(conflictId);
            conflict.status = 'resolved';
            conflict.resolution = event.resolution || '';
            // Remove from display after a delay
            setTimeout(() => this.conflicts.delete(conflictId), 5000);
            this.updateConflictResolution();
        }

        this.addActivity('conflict_resolved',
            `Conflict resolved`,
            event
        );
    }

    handleProjectMilestone(event) {
        const project = {
            id: event.projectId || event.project_id,
            name: event.name || 'Project',
            milestone: event.milestone || '',
            progress: event.progress || 0
        };

        this.projects.set(project.id, project);
        this.updateProjectStatus();
        this.addActivity('project_milestone',
            `Milestone: ${project.milestone}`,
            project
        );
    }

    handleProjectStatusUpdate(event) {
        const status = event.status || {};
        const project = {
            id: 'project_status',
            name: 'Project',
            milestone: status.status || 'active',
            progress: status.progress || 0
        };
        this.projects.set(project.id, project);
        this.updateProjectStatus();
        this.addActivity('project_status',
            `Project status: ${project.milestone} (${project.progress}%)`,
            project
        );
    }

    handleProgressUpdate(event) {
        // Generic progress update for any metric
        this.addActivity('progress',
            event.message || 'Progress update',
            event
        );
    }

    // UI Update Methods
    updateConnectionStatus(connected) {
        const indicator = document.getElementById('connectionStatus');
        const text = document.getElementById('connectionText');

        if (connected) {
            indicator.classList.add('connected');
            text.textContent = 'Connected';
        } else {
            indicator.classList.remove('connected');
            text.textContent = 'Disconnected';
        }
    }

    updateAgentRegistry() {
        const container = document.getElementById('agentRegistry');

        if (this.agents.size === 0) {
            container.innerHTML = '<p style="color: #94a3b8;">No agents registered yet...</p>';
            return;
        }

        let html = '';
        for (const [id, agent] of this.agents) {
            const capabilitiesStr = agent.capabilities.join(', ') || 'No capabilities';
            const modelStr = agent.model ? `🧠 ${agent.model}${agent.provider ? ` (${agent.provider})` : ''}` : '🧠 model: n/a';
            html += `
                <div class="agent-item status-${agent.status}">
                    <div class="agent-name">${agent.name}</div>
                    <div class="agent-capabilities">📌 ${capabilitiesStr}</div>
                    <div class="agent-capabilities">${modelStr}</div>
                </div>
            `;
        }

        container.innerHTML = html;
        document.getElementById('agentCount').textContent = this.agents.size;
    }

    updateTaskCoordinator() {
        const container = document.getElementById('taskCoordinator');

        if (this.tasks.size === 0) {
            container.innerHTML = '<p style="color: #94a3b8;">No tasks assigned yet...</p>';
            return;
        }

        let html = '';
        for (const [id, task] of this.tasks) {
            const statusClass = task.status;
            const agentName = task.assignedAgent ?
                (this.agents.has(task.assignedAgent) ?
                    this.agents.get(task.assignedAgent).name :
                    task.assignedAgent) :
                'Unassigned';

            html += `
                <div class="task-item">
                    <div class="task-header">
                        <div class="task-title">${task.title}</div>
                        <span class="task-status ${statusClass}">${task.status.toUpperCase()}</span>
                    </div>
                    <div class="task-details">
                        <span>👤 ${agentName}</span>
                        <span>${task.progress}%</span>
                    </div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: ${task.progress}%"></div>
                    </div>
                </div>
            `;
        }

        container.innerHTML = html;
        document.getElementById('taskCount').textContent = this.tasks.size;
    }

    updateProjectStatus() {
        const container = document.getElementById('projectStatus');

        if (this.projects.size === 0) {
            container.innerHTML = '<p style="color: #94a3b8;">No active projects...</p>';
            return;
        }

        let html = '';
        for (const [id, project] of this.projects) {
            html += `
                <div style="margin-bottom: 1rem;">
                    <div style="display: flex; justify-content: space-between; margin-bottom: 0.5rem;">
                        <strong>${project.name}</strong>
                        <span>${project.progress}%</span>
                    </div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: ${project.progress}%"></div>
                    </div>
                    <div style="font-size: 0.85rem; color: #94a3b8; margin-top: 0.25rem;">
                        ${project.milestone}
                    </div>
                </div>
            `;
        }

        container.innerHTML = html;
    }

    updateConflictResolution() {
        const container = document.getElementById('conflictResolution');

        if (this.conflicts.size === 0) {
            container.innerHTML = '<p style="color: #94a3b8;">No active conflicts...</p>';
            return;
        }

        let html = '';
        for (const [id, conflict] of this.conflicts) {
            const agentsStr = conflict.agents.join(', ') || 'Unknown agents';
            html += `
                <div class="conflict-item">
                    <div class="conflict-type">⚠️ ${conflict.type}</div>
                    <div class="conflict-agents">Agents: ${agentsStr}</div>
                    <div class="conflict-resolution">${conflict.description}</div>
                </div>
            `;
        }

        container.innerHTML = html;
    }

    addActivity(type, message, data = {}) {
        const activity = {
            type,
            message,
            timestamp: new Date(),
            data
        };

        this.activities.unshift(activity);

        // Keep only last 50 activities
        if (this.activities.length > 50) {
            this.activities.pop();
        }

        this.updateActivityFeed();
    }

    updateActivityFeed() {
        const container = document.getElementById('activityFeed');

        if (this.activities.length === 0) {
            container.innerHTML = '<p style="color: #94a3b8; text-align: center;">Waiting for coordination events...</p>';
            return;
        }

        let html = '';
        for (const activity of this.activities) {
            const time = activity.timestamp.toLocaleTimeString();
            html += `
                <div class="activity-item">
                    <div>
                        <span class="activity-type">${activity.type}</span>
                        <span class="activity-time">${time}</span>
                    </div>
                    <div class="activity-message">${activity.message}</div>
                </div>
            `;
        }

        container.innerHTML = html;
    }

}

// Initialize the widget system when page loads
document.addEventListener('DOMContentLoaded', () => {
    window.actDashboard = new ACTWidgetSystem();
});
