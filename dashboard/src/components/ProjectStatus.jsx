import React from 'react';

const ProjectStatus = ({ status, tasks = [], conflicts = [] }) => {
  const getStatusColor = (status) => {
    switch (status) {
      case 'active': return 'bg-green-500';
      case 'paused': return 'bg-yellow-500';
      case 'completed': return 'bg-blue-500';
      case 'initializing': return 'bg-purple-500';
      case 'error': return 'bg-red-500';
      default: return 'bg-gray-500';
    }
  };

  const getStatusText = (status) => {
    switch (status) {
      case 'active': return 'Active';
      case 'paused': return 'Paused';
      case 'completed': return 'Completed';
      case 'initializing': return 'Initializing';
      case 'error': return 'Error';
      default: return status;
    }
  };

  return (
    <div className="project-status bg-gray-800 rounded-lg p-6">
      <h2 className="text-xl font-semibold mb-4">Project Status</h2>

      <div className="space-y-4">
        {/* Overall Status */}
        <div className="bg-gray-700 rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-medium">Overall Status</h3>
            <div className={`w-3 h-3 rounded-full ${getStatusColor(status.status)}`}></div>
          </div>
          <div className="flex items-center gap-2">
            <span className={`px-3 py-1 text-sm rounded-full ${
              status.status === 'active' ? 'bg-green-600' :
              status.status === 'paused' ? 'bg-yellow-600' :
              status.status === 'completed' ? 'bg-blue-600' :
              status.status === 'initializing' ? 'bg-purple-600' : 'bg-red-600'
            }`}>
              {getStatusText(status.status)}
            </span>
          </div>
        </div>

        {/* Progress Bar */}
        <div className="bg-gray-700 rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <h3 className="font-medium">Progress</h3>
            <span className="text-sm text-gray-400">{status.progress}%</span>
          </div>
          <div className="w-full bg-gray-600 rounded-full h-3">
            <div
              className="bg-gradient-to-r from-blue-500 to-green-500 h-3 rounded-full transition-all duration-500"
              style={{ width: `${status.progress}%` }}
            ></div>
          </div>
        </div>

        {/* Active Agents */}
        <div className="bg-gray-700 rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-medium">Active Agents</h3>
            <span className="text-2xl font-bold text-blue-400">{status.activeAgents || 0}</span>
          </div>
          <p className="text-sm text-gray-400">
            AI agents currently coordinating autonomously
          </p>
        </div>

        {/* Total Tasks */}
        <div className="bg-gray-700 rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-medium">Total Tasks</h3>
            <span className="text-2xl font-bold text-purple-400">{status.totalTasks || 0}</span>
          </div>
          <p className="text-sm text-gray-400">
            Tasks created and assigned by ACT
          </p>
        </div>

        {/* Completed Tasks */}
        <div className="bg-gray-700 rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-medium">Completed Tasks</h3>
            <span className="text-2xl font-bold text-green-400">{status.completedTasks || 0}</span>
          </div>
          <p className="text-sm text-gray-400">
            Tasks successfully completed by agents
          </p>
        </div>
      </div>
    </div>
  );
};

export default ProjectStatus;
