/**
 * TaskCoordinator Unit Tests
 *
 * Focus: terminal-state guard (KI-10). Forbidden transitions (e.g. completed
 * -> failed, validated -> anything, failed -> completed) are rejected with
 * TerminalStateTransitionError so the server surfaces a 409. The normal
 * completed -> submitted_for_validation -> validated happy path still works.
 */

import { AgentRegistry } from '../services/AgentRegistry.js';
import { TaskCoordinator, TerminalStateTransitionError } from '../services/TaskCoordinator.js';

describe('TaskCoordinator terminal-state guard', () => {
  let agentRegistry: AgentRegistry;
  let coordinator: TaskCoordinator;

  beforeEach(() => {
    agentRegistry = new AgentRegistry();
    coordinator = new TaskCoordinator(agentRegistry);
  });

  it('rejects completed -> failed with TerminalStateTransitionError', async () => {
    const task = await coordinator.createTask({ description: 'race candidate' });

    await coordinator.updateTaskProgress(task.id, { status: 'completed', progress: 100 });
    expect(coordinator.getTask(task.id)?.status).toBe('completed');

    await expect(
      coordinator.updateTaskProgress(task.id, { status: 'failed', message: 'late broadcast exit 1' })
    ).rejects.toBeInstanceOf(TerminalStateTransitionError);

    const after = coordinator.getTask(task.id);
    expect(after?.status).toBe('completed');
    expect(after?.progress).toBe(100);
  });

  it('carries the original + target status on the thrown error', async () => {
    const task = await coordinator.createTask({ description: 'error shape' });
    await coordinator.updateTaskProgress(task.id, { status: 'completed', progress: 100 });

    try {
      await coordinator.updateTaskProgress(task.id, { status: 'failed' });
      throw new Error('expected TerminalStateTransitionError');
    } catch (err) {
      expect(err).toBeInstanceOf(TerminalStateTransitionError);
      const tse = err as TerminalStateTransitionError;
      expect(tse.fromStatus).toBe('completed');
      expect(tse.toStatus).toBe('failed');
      expect(tse.code).toBe('TERMINAL_STATE_TRANSITION');
      expect(tse.taskId).toBe(task.id);
    }
  });

  it('rejects validated -> failed and preserves validated state', async () => {
    const task = await coordinator.createTask({ description: 'validated race' });

    await coordinator.updateTaskProgress(task.id, { status: 'completed', progress: 100 });
    await coordinator.updateTaskProgress(task.id, { status: 'submitted_for_validation' });
    await coordinator.updateTaskProgress(task.id, { status: 'validated' });
    expect(coordinator.getTask(task.id)?.status).toBe('validated');

    await expect(
      coordinator.updateTaskProgress(task.id, { status: 'failed' })
    ).rejects.toBeInstanceOf(TerminalStateTransitionError);

    expect(coordinator.getTask(task.id)?.status).toBe('validated');
  });

  it('rejects completed -> in_progress (no regression out of completed)', async () => {
    const task = await coordinator.createTask({ description: 'no-regression' });

    await coordinator.updateTaskProgress(task.id, { status: 'completed', progress: 100 });

    await expect(
      coordinator.updateTaskProgress(task.id, { status: 'in_progress' })
    ).rejects.toBeInstanceOf(TerminalStateTransitionError);

    expect(coordinator.getTask(task.id)?.status).toBe('completed');
  });

  it('rejects failed -> completed (retry must go through reset, not progress update)', async () => {
    const task = await coordinator.createTask({ description: 'failed freeze' });
    await coordinator.updateTaskProgress(task.id, { status: 'failed' });

    await expect(
      coordinator.updateTaskProgress(task.id, { status: 'completed' })
    ).rejects.toBeInstanceOf(TerminalStateTransitionError);

    expect(coordinator.getTask(task.id)?.status).toBe('failed');
  });

  it('allows the happy path completed -> submitted_for_validation -> validated', async () => {
    const task = await coordinator.createTask({ description: 'happy path' });

    await coordinator.updateTaskProgress(task.id, { status: 'completed', progress: 100 });
    await expect(
      coordinator.updateTaskProgress(task.id, { status: 'submitted_for_validation' })
    ).resolves.toBeUndefined();
    expect(coordinator.getTask(task.id)?.status).toBe('submitted_for_validation');

    await expect(
      coordinator.updateTaskProgress(task.id, { status: 'validated' })
    ).resolves.toBeUndefined();
    expect(coordinator.getTask(task.id)?.status).toBe('validated');
  });

  it('allows idempotent completed -> completed writes (no-op)', async () => {
    const task = await coordinator.createTask({ description: 'idempotent complete' });

    await coordinator.updateTaskProgress(task.id, { status: 'completed', progress: 100 });
    await expect(
      coordinator.updateTaskProgress(task.id, { status: 'completed', message: 'redundant' })
    ).resolves.toBeUndefined();

    expect(coordinator.getTask(task.id)?.status).toBe('completed');
  });

  it('allows progress-only updates on a completed task (no status change)', async () => {
    const task = await coordinator.createTask({ description: 'metadata update' });
    await coordinator.updateTaskProgress(task.id, { status: 'completed', progress: 100 });

    await expect(
      coordinator.updateTaskProgress(task.id, { message: 'post-completion note' })
    ).resolves.toBeUndefined();

    expect(coordinator.getTask(task.id)?.status).toBe('completed');
  });

  it('pending -> failed still works (non-terminal source)', async () => {
    const task = await coordinator.createTask({ description: 'ordinary failure' });

    await expect(
      coordinator.updateTaskProgress(task.id, { status: 'failed', message: 'legit failure' })
    ).resolves.toBeUndefined();

    expect(coordinator.getTask(task.id)?.status).toBe('failed');
  });
});

describe('TaskCoordinator.abandonTask', () => {
  let agentRegistry: AgentRegistry;
  let coordinator: TaskCoordinator;

  beforeEach(() => {
    agentRegistry = new AgentRegistry();
    coordinator = new TaskCoordinator(agentRegistry);
  });

  it('marks a pending task as failed with abandoned metadata', async () => {
    const task = await coordinator.createTask({ description: 'will be abandoned' });

    const abandoned = await coordinator.abandonTask(task.id, 'spec changed, no longer needed');

    expect(abandoned.status).toBe('failed');
    expect(abandoned.metadata?.abandoned).toBe(true);
    expect(abandoned.metadata?.abandonReason).toBe('spec changed, no longer needed');
    expect(typeof abandoned.metadata?.abandonedAt).toBe('string');
    expect(abandoned.retryCount).toBeGreaterThanOrEqual(3); // permanent-failed semantics
  });

  it('refuses to abandon a completed task', async () => {
    const task = await coordinator.createTask({ description: 'already done' });
    await coordinator.updateTaskProgress(task.id, { status: 'completed', progress: 100 });

    await expect(coordinator.abandonTask(task.id, 'too late')).rejects.toThrow(/cannot abandon task/);
    expect(coordinator.getTask(task.id)?.status).toBe('completed');
  });

  it('throws on unknown task id', async () => {
    await expect(coordinator.abandonTask('nope-id', 'why')).rejects.toThrow(/not found/);
  });
});
