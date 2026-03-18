export enum AgentRole {
  PLANNER = 'planner',
  OBSERVER = 'observer',
  ASSURANCE = 'assurance',
  QA_SYNTHESIZER = 'qa_synthesizer',
  FRONTEND_DEV = 'frontend_dev',
  BACKEND_DEV = 'backend_dev',
  QA_ENGINEER = 'qa_engineer',
  RESEARCHER = 'researcher',
  DEVELOPER = 'developer',
}

export type AgentTier = 1 | 2;

const TIER_1_ROLES: AgentRole[] = [
  AgentRole.PLANNER,
  AgentRole.OBSERVER,
  AgentRole.ASSURANCE,
  AgentRole.QA_SYNTHESIZER,
];

export function getTier(role: AgentRole): AgentTier {
  return TIER_1_ROLES.includes(role) ? 1 : 2;
}

export interface RoleConstraints {
  canCreateTasks: boolean;
  canValidateTasks: boolean;
  canAssembleOutput: boolean;
  canMonitorLogs: boolean;
  maxConcurrentTasks: number;
  persistent: boolean;
}

const ROLE_CONSTRAINTS_MAP: Record<AgentRole, RoleConstraints> = {
  [AgentRole.PLANNER]: {
    canCreateTasks: true,
    canValidateTasks: false,
    canAssembleOutput: false,
    canMonitorLogs: true,
    maxConcurrentTasks: 1,
    persistent: true,
  },
  [AgentRole.OBSERVER]: {
    canCreateTasks: false,
    canValidateTasks: false,
    canAssembleOutput: false,
    canMonitorLogs: true,
    maxConcurrentTasks: 1,
    persistent: true,
  },
  [AgentRole.ASSURANCE]: {
    canCreateTasks: false,
    canValidateTasks: true,
    canAssembleOutput: false,
    canMonitorLogs: false,
    maxConcurrentTasks: 1,
    persistent: true,
  },
  [AgentRole.QA_SYNTHESIZER]: {
    canCreateTasks: false,
    canValidateTasks: false,
    canAssembleOutput: true,
    canMonitorLogs: false,
    maxConcurrentTasks: 1,
    persistent: true,
  },
  [AgentRole.FRONTEND_DEV]: {
    canCreateTasks: false,
    canValidateTasks: false,
    canAssembleOutput: false,
    canMonitorLogs: false,
    maxConcurrentTasks: 1,
    persistent: false,
  },
  [AgentRole.BACKEND_DEV]: {
    canCreateTasks: false,
    canValidateTasks: false,
    canAssembleOutput: false,
    canMonitorLogs: false,
    maxConcurrentTasks: 1,
    persistent: false,
  },
  [AgentRole.QA_ENGINEER]: {
    canCreateTasks: false,
    canValidateTasks: false,
    canAssembleOutput: false,
    canMonitorLogs: false,
    maxConcurrentTasks: 1,
    persistent: false,
  },
  [AgentRole.RESEARCHER]: {
    canCreateTasks: false,
    canValidateTasks: false,
    canAssembleOutput: false,
    canMonitorLogs: false,
    maxConcurrentTasks: 1,
    persistent: false,
  },
  [AgentRole.DEVELOPER]: {
    canCreateTasks: false,
    canValidateTasks: false,
    canAssembleOutput: false,
    canMonitorLogs: false,
    maxConcurrentTasks: 1,
    persistent: false,
  },
};

export function getRoleConstraints(role: AgentRole): RoleConstraints {
  return ROLE_CONSTRAINTS_MAP[role];
}