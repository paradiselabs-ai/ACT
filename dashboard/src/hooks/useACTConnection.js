import { useState, useEffect, useCallback, useRef } from 'react';
import io from 'socket.io-client';

export const useACTConnection = () => {
  console.log('🔌 useACTConnection hook initializing...');

  const [socket, setSocket] = useState(null);
  const [isConnected, setIsConnected] = useState(false);
  const [connectionError, setConnectionError] = useState(null);
  const [agents, setAgents] = useState([]);
  const [tasks, setTasks] = useState([]);
  const [conflicts, setConflicts] = useState([]);
  const [projectStatus, setProjectStatus] = useState({
    status: 'initializing',
    progress: 0,
    activeAgents: 0
  });

  console.log('📊 useACTConnection state initialized');

  // Stable reference to current socket
  const socketRef = useRef(socket);
  socketRef.current = socket;

  // Connect to ACT server - simplified to avoid dependency issues
  const connect = useCallback(() => {
    console.log('🔗 Attempting to connect to ACT server...');

    if (socketRef.current) {
      console.log('Already have socket, skipping connection');
      return;
    }

    try {
      console.log('📡 Creating new socket connection...');
      const newSocket = io('ws://localhost:8080', {
        transports: ['websocket', 'polling'],
        timeout: 20000,
        forceNew: true
      });

      console.log('🔌 Socket created, setting up event handlers...');
      socketRef.current = newSocket;
      setSocket(newSocket);
      setConnectionError(null);

      // Connection events
      newSocket.on('connect', () => {
        console.log('Connected to ACT server');
        setIsConnected(true);
        setConnectionError(null);
      });

      newSocket.on('disconnect', (reason) => {
        console.log('Disconnected from ACT server:', reason);
        setIsConnected(false);
        socketRef.current = null;
      });

      newSocket.on('connect_error', (error) => {
        console.error('Connection error:', error);
        setConnectionError(error.message);
        setIsConnected(false);
        socketRef.current = null;
      });

      // Agent coordination events
      newSocket.on('agent_registered', (data) => {
        console.log('Agent registered:', data);
        setAgents(prev => {
          const filtered = prev.filter(a => a.id !== data.agent.id);
          return [...filtered, data.agent];
        });
        setProjectStatus(prev => ({
          ...prev,
          activeAgents: prev.activeAgents + 1
        }));
      });

      newSocket.on('agent_disconnected', (data) => {
        console.log('Agent disconnected:', data);
        setAgents(prev => prev.filter(a => a.id !== data.agentId));
        setProjectStatus(prev => ({
          ...prev,
          activeAgents: Math.max(0, prev.activeAgents - 1)
        }));
      });

      newSocket.on('task_assigned', (data) => {
        console.log('Task assigned:', data);
        setTasks(prev => {
          const filtered = prev.filter(t => t.id !== data.task.id);
          return [...filtered, data.task];
        });
      });

      newSocket.on('task_completed', (data) => {
        console.log('Task completed:', data);
        setTasks(prev => prev.map(t =>
          t.id === data.taskId ? { ...t, status: 'completed' } : t
        ));
      });

      newSocket.on('task_progress', (data) => {
        console.log('Task progress:', data);
        setTasks(prev => prev.map(t =>
          t.id === data.taskId ? { ...t, status: data.status, progress: data.progress } : t
        ));
      });

      newSocket.on('conflict_detected', (data) => {
        console.log('Conflict detected:', data);
        setConflicts(prev => [...prev, data.conflict]);
      });

      newSocket.on('conflict_resolved', (data) => {
        console.log('Conflict resolved:', data);
        setConflicts(prev => prev.filter(c => c.id !== data.conflictId));
      });

      newSocket.on('project_status_update', (data) => {
        console.log('Project status update:', data);
        setProjectStatus(prev => ({ ...prev, ...data.status }));
      });

      newSocket.on('coordination_error', (data) => {
        console.error('Coordination error:', data);
        setConnectionError(data.message);
      });

      // Agent-to-agent messaging for coordination display
      newSocket.on('agent_message', (data) => {
        console.log('Agent message:', data);
        // Could add this to activity feed or separate messaging display
      });

    } catch (error) {
      console.error('Failed to create socket connection:', error);
      setConnectionError(error.message);
    }
  }, []); // No dependencies - stable function

  // Disconnect from server
  const disconnect = useCallback(() => {
    if (socket) {
      socket.disconnect();
      setSocket(null);
      setIsConnected(false);
    }
  }, [socket]);

  // Send message to server
  const sendMessage = useCallback((event, data) => {
    if (socket && isConnected) {
      socket.emit(event, data);
    } else {
      console.warn('Cannot send message: not connected to ACT server');
    }
  }, [socket, isConnected]);

  // Request project status
  const requestProjectStatus = useCallback(() => {
    sendMessage('get_project_status', {});
  }, [sendMessage]);

  // Request agent registry
  const requestAgentRegistry = useCallback(() => {
    sendMessage('get_agent_registry', {});
  }, [sendMessage]);

  // Request task list
  const requestTasks = useCallback(() => {
    sendMessage('get_tasks', {});
  }, [sendMessage]);

  // Connect on mount, cleanup on unmount
  useEffect(() => {
    // Only connect if not already connected
    if (!socket && !isConnected) {
      console.log('Initial connection to ACT server...');
      connect();
    }

    return () => {
      disconnect();
    };
  }, []); // Empty dependency array - only run once on mount

  // Stable reference to connect function
  const connectRef = useRef(connect);
  connectRef.current = connect;

  // Auto-reconnect logic - stable implementation
  useEffect(() => {
    let reconnectTimer = null;

    if (!isConnected && !connectionError && !socketRef.current) {
      console.log('Scheduling reconnect in 5 seconds...');
      reconnectTimer = setTimeout(() => {
        console.log('Executing scheduled reconnect...');
        connectRef.current();
      }, 5000);
    }

    return () => {
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
      }
    };
  }, [isConnected, connectionError]); // No connect dependency

  return {
    // Connection state
    isConnected,
    connectionError,
    socket,

    // Data
    agents,
    tasks,
    conflicts,
    projectStatus,

    // Actions
    connect,
    disconnect,
    sendMessage,
    requestProjectStatus,
    requestAgentRegistry,
    requestTasks,

    // Utilities
    clearError: () => setConnectionError(null)
  };
};
