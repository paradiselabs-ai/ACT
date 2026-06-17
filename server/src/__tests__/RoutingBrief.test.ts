/**
 * RoutingBrief aggregation tests
 *
 * Verifies getProjectOutcomes() derives per-project composition + pass/kickback
 * counts from the event log, and getRoutingBrief() emits confidence-labeled
 * per-role and role-pair evidence. embed() is stubbed so the test doesn't load
 * the real model — the aggregation reads message fields, not vectors.
 */

import { LocalEmbeddingVectorStore } from '../services/LocalEmbeddingVectorStore.js';
import { CoordinationMessage } from '../types/coordination.js';

// Each event needs a unique (timestamp, agent) — store() keys points on
// `${timestamp}_${agent}`, so a shared stamp would collapse them into one point.
let _seq = 0;
function ev(partial: Partial<CoordinationMessage> & { type: string }): CoordinationMessage {
  _seq++;
  return {
    timestamp: new Date(Date.UTC(2026, 0, 1, 0, 0, _seq)).toISOString(),
    agent: 'system',
    message: '',
    ...partial,
  } as CoordinationMessage;
}

describe('RoutingBrief aggregation', () => {
  let store: LocalEmbeddingVectorStore;

  beforeEach(async () => {
    store = new LocalEmbeddingVectorStore();
    // Stub embedding so store() is fast + offline; vectors are irrelevant here.
    (store as any).embed = async () => new Array(384).fill(0);

    const events: CoordinationMessage[] = [
      // Project "alpha": composition 1×developer + 1×backend_dev, 1 pass / 1 kickback
      ev({ type: 'agent_registered', projectName: 'alpha', data: { agentId: 'dev-1', capabilities: ['node', 'javascript'] } } as any),
      ev({ type: 'agent_registered', projectName: 'alpha', data: { agentId: 'backend-1', capabilities: ['api', 'db'] } } as any),
      ev({ type: 'task_created', projectName: 'alpha', data: { taskId: 'A1', requiredCapabilities: ['node'] } } as any),
      ev({ type: 'task_assigned', projectName: 'alpha', data: { taskId: 'A1', agentId: 'dev-1' } } as any),
      ev({ type: 'task_validated', projectName: 'alpha', data: { taskId: 'A1' } } as any),
      ev({ type: 'task_created', projectName: 'alpha', data: { taskId: 'A2', requiredCapabilities: ['api'] } } as any),
      ev({ type: 'task_assigned', projectName: 'alpha', data: { taskId: 'A2', agentId: 'backend-1' } } as any),
      ev({ type: 'task_validation_failed', projectName: 'alpha', data: { taskId: 'A2', gaps: 'missing auth check' } } as any),

      // Project "beta": composition 3×developer, 3 pass / 0 kickback
      ev({ type: 'agent_registered', projectName: 'beta', data: { agentId: 'dev-1', capabilities: ['node'] } } as any),
      ev({ type: 'agent_registered', projectName: 'beta', data: { agentId: 'dev-2', capabilities: ['node'] } } as any),
      ev({ type: 'agent_registered', projectName: 'beta', data: { agentId: 'dev-3', capabilities: ['node'] } } as any),
      ev({ type: 'task_assigned', projectName: 'beta', data: { taskId: 'B1', agentId: 'dev-1' } } as any),
      ev({ type: 'task_validated', projectName: 'beta', data: { taskId: 'B1' } } as any),
      ev({ type: 'task_assigned', projectName: 'beta', data: { taskId: 'B2', agentId: 'dev-2' } } as any),
      ev({ type: 'task_validated', projectName: 'beta', data: { taskId: 'B2' } } as any),
      ev({ type: 'task_assigned', projectName: 'beta', data: { taskId: 'B3', agentId: 'dev-3' } } as any),
      ev({ type: 'task_validated', projectName: 'beta', data: { taskId: 'B3' } } as any),
    ];
    for (const e of events) await store.store(e);
  });

  afterEach(async () => { await store.close(); });

  it('derives per-project composition + pass/kickback counts', () => {
    const outcomes = store.getProjectOutcomes();
    const alpha = outcomes.find(o => o.project === 'alpha')!;
    const beta = outcomes.find(o => o.project === 'beta')!;

    expect(alpha.composition).toEqual({ developer: 1, backend_dev: 1 });
    expect(alpha.taskCount).toBe(2);
    expect(alpha.passed).toBe(1);
    expect(alpha.kickbacks).toBe(1);
    expect(alpha.perRole.developer).toEqual({ tasks: 1, passed: 1 });
    expect(alpha.perRole.backend_dev).toEqual({ tasks: 1, passed: 0 });
    expect(alpha.topGaps[0]).toContain('missing auth');

    expect(beta.composition).toEqual({ developer: 3 });
    expect(beta.taskCount).toBe(3);
    expect(beta.passed).toBe(3);
    expect(beta.kickbacks).toBe(0);
  });

  it('emits confidence-labeled per-role + role-pair evidence', async () => {
    const brief = await store.getRoutingBrief('node cli tool', ['node']);

    // per-role: developer ran 4 tasks (1 alpha + 3 beta), all passed
    const dev = brief.perRole.find((r: any) => r.role === 'developer');
    expect(dev.tasks).toBe(4);
    expect(dev.passRate).toBe(1);
    expect(dev.signal).toBe('low'); // <5 samples

    // role-pair from alpha: developer + backend_dev ran together
    const pair = brief.rolePairs.find((p: any) => p.pair === 'backend_dev + developer');
    expect(pair).toBeDefined();
    expect(pair.kickbacks).toBe(1);

    // text block carries signal labels + the failure note
    expect(brief.text).toContain('signal');
    expect(brief.text).toContain('Per-role track record');
    expect(brief.text).toContain('missing auth');
  });
});
