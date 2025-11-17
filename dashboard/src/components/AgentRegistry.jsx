import React from 'react';

const AgentRegistry = ({ agents }) => {
  return (
    <div className="agent-registry bg-gray-800 rounded-lg p-6">
      <h2 className="text-xl font-semibold mb-4">Agent Registry</h2>

      {agents.length === 0 ? (
        <div className="text-center py-8 text-gray-400">
          <div className="text-4xl mb-2">🤖</div>
          <p>No agents registered yet</p>
          <p className="text-sm mt-1">Waiting for agents to connect...</p>
        </div>
      ) : (
        <div className="space-y-4">
          {agents.map((agent) => (
            <div
              key={agent.id}
              className="bg-gray-700 rounded-lg p-4 border border-gray-600"
            >
              <div className="flex items-center justify-between mb-3">
                <h3 className="font-medium text-lg">{agent.name}</h3>
                <div className={`w-3 h-3 rounded-full ${
                  agent.status === 'online' ? 'bg-green-500' :
                  agent.status === 'busy' ? 'bg-yellow-500' : 'bg-red-500'
                }`}></div>
              </div>

              <div className="mb-3">
                <p className="text-sm text-gray-400 mb-2">Capabilities:</p>
                <div className="flex flex-wrap gap-2">
                  {agent.capabilities?.map((capability, index) => (
                    <span
                      key={index}
                      className="px-2 py-1 bg-blue-600 text-xs rounded-full"
                    >
                      {capability}
                    </span>
                  )) || (
                    <span className="text-sm text-gray-500">No capabilities listed</span>
                  )}
                </div>
              </div>

              <div className="text-xs text-gray-400">
                {agent.currentTask ? (
                  <p>Working on: <span className="text-yellow-400">{agent.currentTask}</span></p>
                ) : (
                  <p>Status: Available</p>
                )}
                <p>Last seen: {new Date(agent.lastSeen).toLocaleTimeString()}</p>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Agent Statistics */}
      <div className="mt-6 pt-4 border-t border-gray-600">
        <div className="grid grid-cols-2 gap-4 text-center">
          <div>
            <div className="text-2xl font-bold text-green-400">{agents.length}</div>
            <div className="text-xs text-gray-400">Total Agents</div>
          </div>
          <div>
            <div className="text-2xl font-bold text-blue-400">
              {agents.filter(a => a.status === 'online').length}
            </div>
            <div className="text-xs text-gray-400">Online</div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AgentRegistry;
