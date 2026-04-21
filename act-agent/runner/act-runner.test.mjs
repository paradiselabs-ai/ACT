/**
 * Runner unit test — KI-10.
 *
 * broadcastCompletion runs AFTER the task has been marked complete on the
 * server. Its purpose is team notification. If its claude subprocess exits
 * non-zero, that is a broadcast failure, not a task failure — the runner
 * must NOT emit any task-state HTTP call (/complete, /progress) from this
 * code path.
 *
 * Run: node --test act-agent/runner/act-runner.test.mjs
 */

import test from 'node:test';
import assert from 'node:assert/strict';

// Set minimal env so the module can be imported without main() side-effects.
// act-runner.mjs guards main() behind an isDirectInvocation check, so
// importing it here is a no-op beyond defining exports.
process.env.ACT_SERVER_URL = 'http://127.0.0.1:0';
process.argv = [process.argv[0], '/tmp/__not__act_runner__.mjs', '--agent-id', 'test-agent', '--name', 'test'];

const { broadcastCompletion } = await import('./act-runner.mjs');

function collectingFetch() {
  const calls = [];
  const fn = async (url, opts) => {
    calls.push({ url, method: opts?.method, body: opts?.body });
    return { ok: true, status: 200, text: async () => '', json: async () => ({}) };
  };
  fn.calls = calls;
  return fn;
}

test('broadcastCompletion: claude exit 1 does not emit any task-state HTTP call', async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = collectingFetch();

  try {
    await broadcastCompletion(
      { id: 'task-abc', title: 'ki10 repro' },
      'task output that was already reported complete',
      {
        runAgent: async () => ({ success: false, code: 1, output: '', error: 'Warning: no stdin data received in 3s...' }),
        sendMessage: async (_msg) => { throw new Error('sendMessage must not be called on broadcast failure'); },
        log: () => {},
      }
    );

    const touches = globalThis.fetch.calls.filter(c =>
      typeof c.url === 'string' && /\/api\/tasks\/[^/]+\/(complete|progress)/.test(c.url)
    );
    assert.equal(touches.length, 0, 'broadcast must not POST /complete or /progress');
    assert.equal(globalThis.fetch.calls.length, 0, 'broadcast must not make any HTTP calls on failure');
  } finally {
    globalThis.fetch = originalFetch;
  }
});

test('broadcastCompletion: claude throw is caught and does not bubble to caller', async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = collectingFetch();

  try {
    await broadcastCompletion(
      { id: 'task-xyz', title: 'thrown error path' },
      'output',
      {
        runAgent: async () => { throw new Error('simulated claude crash'); },
        sendMessage: async (_msg) => { throw new Error('sendMessage must not be called when claude threw'); },
        log: () => {},
      }
    );

    assert.equal(globalThis.fetch.calls.length, 0);
  } finally {
    globalThis.fetch = originalFetch;
  }
});

test('broadcastCompletion: success path only sends a status: message, no task endpoints', async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = collectingFetch();
  const sentMessages = [];

  try {
    await broadcastCompletion(
      { id: 'task-ok', title: 'normal' },
      'output',
      {
        runAgent: async () => ({ success: true, code: 0, output: 'built the thing' }),
        sendMessage: async (msg) => { sentMessages.push(msg); },
        log: () => {},
      }
    );

    assert.equal(sentMessages.length, 1);
    assert.ok(sentMessages[0].startsWith('status:'), `expected status: prefix, got: ${sentMessages[0]}`);
    const touches = globalThis.fetch.calls.filter(c =>
      typeof c.url === 'string' && /\/api\/tasks\/[^/]+\/(complete|progress)/.test(c.url)
    );
    assert.equal(touches.length, 0);
  } finally {
    globalThis.fetch = originalFetch;
  }
});
