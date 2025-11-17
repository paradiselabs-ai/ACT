import React from 'react';
import { useACTConnection } from '../hooks/useACTConnection';
import AgentRegistry from './AgentRegistry';
import TaskCoordinator from './TaskCoordinator';
import ConflictResolution from './ConflictResolution';
import ProjectStatus from './ProjectStatus';

const ACTDashboard = () => {
  console.log('ACTDashboard component rendering!');

  const {
    isConnected,
    connectionError,
    agents,
    tasks,
    conflicts,
    projectStatus,
    requestProjectStatus,
    requestAgentRegistry,
    requestTasks,
    clearError
  } = useACTConnection();

  console.log('Dashboard data:', { isConnected, agents: agents.length, tasks: tasks.length });

  return (
    <div className="act-dashboard min-h-screen bg-gray-900 text-white p-6">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold mb-2">ACT Coordination Dashboard</h1>
        <div className="flex items-center gap-4">
          <div className={`w-3 h-3 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`}></div>
          <span className="text-sm">
            {isConnected ? 'Connected to ACT Server' : 'Disconnected from ACT Server'}
          </span>
          {connectionError && (
            <div className="flex items-center gap-2">
              <span className="text-sm text-red-400">Error: {connectionError}</span>
              <button
                onClick={clearError}
                className="text-xs px-2 py-1 bg-red-600 hover:bg-red-700 rounded"
              >
                Clear
              </button>
            </div>
          )}
          <span className="text-sm text-gray-400">
            {agents.length} agents registered
          </span>
        </div>
      </div>

      {/* Main Dashboard Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left Column - Agent Registry */}
        <div className="lg:col-span-1">
          <AgentRegistry agents={agents} />
        </div>

        {/* Middle Column - Task Coordination */}
        <div className="lg:col-span-1">
          <TaskCoordinator tasks={tasks} agents={agents} />
        </div>

        {/* Right Column - Status & Conflicts */}
        <div className="lg:col-span-1 space-y-6">
          <ProjectStatus status={projectStatus} tasks={tasks} conflicts={conflicts} />
          <ConflictResolution conflicts={conflicts} />
        </div>
      </div>

      {/* Real-time Activity Feed */}
      <div className="mt-8">
        <h2 className="text-xl font-semibold mb-4">Real-Time Coordination Activity</h2>
        <div className="bg-gray-800 rounded-lg p-4 max-h-64 overflow-y-auto">
          <div className="space-y-2 text-sm">
            {tasks.slice(-5).reverse().map((task, index) => (
              <div key={task.id || index} className="flex items-center gap-2">
                <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                <span>
                  Task "{task.description}" assigned to {task.assignedAgent || 'unassigned'}
                  <span className="text-gray-400 ml-2">
                    {task.timestamp ? new Date(task.timestamp).toLocaleTimeString() : 'Just now'}
                  </span>
                </span>
              </div>
            ))}
            {agents.slice(-3).reverse().map((agent, index) => (
              <div key={agent.id || index} className="flex items-center gap-2">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span>
                  Agent "{agent.name}" registered with capabilities: {agent.capabilities?.join(', ') || 'none'}
                  <span className="text-gray-400 ml-2">
                    {agent.timestamp ? new Date(agent.timestamp).toLocaleTimeString() : 'Just now'}
                  </span>
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Connection Controls */}
      <div className="mt-6 flex gap-4">
        <button
          onClick={requestProjectStatus}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded text-sm"
          disabled={!isConnected}
        >
          Request Status
        </button>
        <button
          onClick={requestAgentRegistry}
          className="px-4 py-2 bg-green-600 hover:bg-green-700 rounded text-sm"
          disabled={!isConnected}
        >
          Refresh Agents
        </button>
        <button
          onClick={requestTasks}
          className="px-4 py-2 bg-purple-600 hover:bg-purple-700 rounded text-sm"
          disabled={!isConnected}
        >
          Refresh Tasks
        </button>
      </div>
    </div>
  );
};

export default ACTDashboard;
