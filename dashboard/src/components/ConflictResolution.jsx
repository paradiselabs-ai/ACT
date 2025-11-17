import React from 'react';

const ConflictResolution = ({ conflicts }) => {
  return (
    <div className="conflict-resolution bg-gray-800 rounded-lg p-6">
      <h2 className="text-xl font-semibold mb-4">Conflict Resolution</h2>

      {conflicts.length === 0 ? (
        <div className="text-center py-8 text-gray-400">
          <div className="text-4xl mb-2">✌️</div>
          <p>No conflicts detected</p>
          <p className="text-sm mt-1">Autonomous coordination working smoothly</p>
        </div>
      ) : (
        <div className="space-y-4">
          {conflicts.map((conflict) => (
            <div
              key={conflict.id}
              className="bg-red-900/20 border border-red-500/30 rounded-lg p-4"
            >
              <div className="flex items-start justify-between mb-3">
                <h3 className="font-medium text-lg text-red-300">
                  {conflict.type === 'resource_contention' ? 'Resource Contention' :
                   conflict.type === 'dependency_deadlock' ? 'Dependency Deadlock' :
                   conflict.type === 'capability_overlap' ? 'Capability Overlap' :
                   conflict.type === 'priority_mismatch' ? 'Priority Mismatch' :
                   'Unknown Conflict'}
                </h3>
                <div className="w-3 h-3 rounded-full bg-red-500 animate-pulse"></div>
              </div>

              <div className="mb-3">
                <p className="text-sm text-gray-300 mb-2">{conflict.description}</p>
                <span className="px-2 py-1 text-xs bg-red-600 rounded-full">
                  Resolving...
                </span>
              </div>

              {conflict.involvedAgents && conflict.involvedAgents.length > 0 && (
                <div className="text-sm text-gray-400">
                  <p>Involved agents: {conflict.involvedAgents.join(', ')}</p>
                </div>
              )}

              {conflict.tasks && conflict.tasks.length > 0 && (
                <div className="mt-2">
                  <p className="text-sm text-gray-400 mb-1">Affected tasks:</p>
                  <ul className="text-sm text-gray-300">
                    {conflict.tasks.map((taskId, index) => (
                      <li key={index} className="flex items-center gap-2">
                        <span className="w-1 h-1 bg-gray-400 rounded-full"></span>
                        Task {taskId}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Conflict Statistics */}
      <div className="mt-6 pt-4 border-t border-gray-600">
        <div className="grid grid-cols-2 gap-4 text-center">
          <div>
            <div className="text-2xl font-bold text-red-400">{conflicts.length}</div>
            <div className="text-xs text-gray-400">Active Conflicts</div>
          </div>
          <div>
            <div className="text-2xl font-bold text-green-400">
              {conflicts.filter(c => c.status === 'resolved').length}
            </div>
            <div className="text-xs text-gray-400">Resolved Today</div>
          </div>
        </div>
      </div>

      {/* Autonomous Resolution Message */}
      {conflicts.length > 0 && (
        <div className="mt-4 p-3 bg-blue-900/20 border border-blue-500/30 rounded-lg">
          <p className="text-sm text-blue-300">
            🤖 ACT is autonomously resolving conflicts through negotiation protocols.
            No human intervention required.
          </p>
        </div>
      )}
    </div>
  );
};

export default ConflictResolution;
