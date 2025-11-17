/**
 * ACT SSE Widget System
 * Lightweight HTML client for consuming ACT coordination events via Server-Sent Events
 */

class ACTWidgetSystem {
    constructor(serverUrl = 'http://localhost:8080') {
        this.serverUrl = serverUrl;
        this.eventSource = null;
        this.isConnected = false;

        // State
        this.agents = new Map();
        this.tasks = new Map();
        this.projects = new Map();
        this.conflicts = new Map();
        this.activities = [];

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
            console.log(`🔌 Connecting to ACT server at ${this.serverUrl}/coordinate`);

            this.eventSource = new EventSource(`${this.serverUrl}/coordinate`);

            this.eventSource.onopen = () => {
                console.log('✅ Connected to ACT server');
                this.isConnected = true;
                document.dispatchEvent(new Event('act:connected'));
                this.clearErrorMessage();
            };

            // Handle different event types
            this.eventSource.addEventListener('agent_registered', (e) => {
                this.handleAgentRegistered(JSON.parse(e.data));
            });

            this.eventSource.addEventListener('task_created', (e) => {
                this.handleTaskCreated(JSON.parse(e.data));
            });

            this.eventSource.addEventListener('task_assigned', (e) => {
                this.handleTaskAssigned(JSON.parse(e.data));
            });

            this.eventSource.addEventListener('task_progress', (e) => {
                this.handleTaskProgress(JSON.parse(e.data));
            });

            this.eventSource.addEventListener('task_completed', (e) => {
                this.handleTaskCompleted(JSON.parse(e.data));
            });

            this.eventSource.addEventListener('conflict_detected', (e) => {
                this.handleConflictDetected(JSON.parse(e.data));
            });

            this.eventSource.addEventListener('conflict_resolved', (e) => {
                this.handleConflictResolved(JSON.parse(e.data));
            });

            this.eventSource.addEventListener('project_milestone', (e) => {
                this.handleProjectMilestone(JSON.parse(e.data));
            });

            this.eventSource.addEventListener('progress_update', (e) => {
                this.handleProgressUpdate(JSON.parse(e.data));
            });

            this.eventSource.onerror = (error) => {
                console.error('❌ SSE connection error:', error);
                this.eventSource.close();
                this.showConnectionError();
            };

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
        const agent = {
            id: event.agentId || event.agent_id,
            name: event.name || event.agentId,
            capabilities: event.capabilities || [],
            status: 'active',
            timestamp: new Date().toISOString()
        };

        this.agents.set(agent.id, agent);
        this.updateAgentRegistry();
        this.addActivity('agent_registered', `Agent registered: ${agent.name}`, agent);
    }

    handleTaskCreated(event) {
        const task = {
            id: event.taskId || event.task_id,
            title: event.title || 'Untitled Task',
            description: event.description || '',
            status: 'created',
            progress: 0,
            timestamp: new Date().toISOString()
        };

        this.tasks.set(task.id, task);
        this.updateTaskCoordinator();
        this.addActivity('task_created', `Task created: ${task.title}`, task);
    }

    handleTaskAssigned(event) {
        const taskId = event.taskId || event.task_id;
        const agentId = event.agentId || event.agent_id;

        if (this.tasks.has(taskId)) {
            const task = this.tasks.get(taskId);
            task.status = 'assigned';
            task.assignedAgent = agentId;
            task.reasoning = event.reasoning || '';
            this.updateTaskCoordinator();
        }

        const agent = this.agents.get(agentId);
        const agentName = agent ? agent.name : agentId;
        const taskName = this.tasks.has(taskId) ? this.tasks.get(taskId).title : taskId;

        this.addActivity('task_assigned',
            `Task "${taskName}" assigned to ${agentName}`,
            { taskId, agentId, reasoning: event.reasoning }
        );
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
            html += `
                <div class="agent-item status-${agent.status}">
                    <div class="agent-name">${agent.name}</div>
                    <div class="agent-capabilities">📌 ${capabilitiesStr}</div>
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
