import React from 'react';

const TaskCoordinator = ({ tasks, agents }) => {
  const getAgentName = (agentId) => {
    const agent = agents.find(a => a.id === agentId);
    return agent ? agent.name : 'Unknown Agent';
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'completed': return 'bg-green-500';
      case 'in_progress': return 'bg-blue-500';
      case 'assigned': return 'bg-yellow-500';
      case 'pending': return 'bg-gray-500';
      default: return 'bg-gray-500';
    }
  };

  const getStatusText = (status) => {
    switch (status) {
      case 'completed': return 'Completed';
      case 'in_progress': return 'In Progress';
      case 'assigned': return 'Assigned';
      case 'pending': return 'Pending';
      default: return status;
    }
  };

  return (
    <div className="task-coordinator bg-gray-800 rounded-lg p-6">
      <h2 className="text-xl font-semibold mb-4">Task Coordination</h2>

      {tasks.length === 0 ? (
        <div className="text-center py-8 text-gray-400">
          <div className="text-4xl mb-2">📋</div>
          <p>No tasks assigned yet</p>
          <p className="text-sm mt-1">Waiting for project to start...</p>
        </div>
      ) : (
        <div className="space-y-4">
          {tasks.map((task) => (
            <div
              key={task.id}
              className="bg-gray-700 rounded-lg p-4 border border-gray-600"
            >
              <div className="flex items-start justify-between mb-3">
                <h3 className="font-medium text-lg flex-1">{task.description}</h3>
                <div className={`w-3 h-3 rounded-full ${getStatusColor(task.status)} ml-3`}></div>
              </div>

              <div className="mb-3">
                <span className={`px-2 py-1 text-xs rounded-full ${
                  task.status === 'completed' ? 'bg-green-600' :
                  task.status === 'in_progress' ? 'bg-blue-600' :
                  task.status === 'assigned' ? 'bg-yellow-600' : 'bg-gray-600'
                }`}>
                  {getStatusText(task.status)}
                </span>
              </div>

              {task.assignedAgent && (
                <div className="text-sm text-gray-400">
                  <p>Assigned to: <span className="text-blue-400">{getAgentName(task.assignedAgent)}</span></p>
                </div>
              )}

              {task.requiredCapabilities && task.requiredCapabilities.length > 0 && (
                <div className="mt-2">
                  <p className="text-sm text-gray-400 mb-1">Required capabilities:</p>
                  <div className="flex flex-wrap gap-1">
                    {task.requiredCapabilities.map((cap, index) => (
                      <span
                        key={index}
                        className="px-2 py-1 bg-purple-600 text-xs rounded-full"
                      >
                        {cap}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Task Statistics */}
      <div className="mt-6 pt-4 border-t border-gray-600">
        <div className="grid grid-cols-4 gap-2 text-center">
          <div>
            <div className="text-lg font-bold text-gray-300">{tasks.length}</div>
            <div className="text-xs text-gray-400">Total</div>
          </div>
          <div>
            <div className="text-lg font-bold text-blue-400">
              {tasks.filter(t => t.status === 'in_progress').length}
            </div>
            <div className="text-xs text-gray-400">Active</div>
          </div>
          <div>
            <div className="text-lg font-bold text-green-400">
              {tasks.filter(t => t.status === 'completed').length}
            </div>
            <div className="text-xs text-gray-400">Done</div>
          </div>
          <div>
            <div className="text-lg font-bold text-yellow-400">
              {tasks.filter(t => t.status === 'pending' || t.status === 'assigned').length}
            </div>
            <div className="text-xs text-gray-400">Pending</div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default TaskCoordinator;
