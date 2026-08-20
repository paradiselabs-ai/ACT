/**
 * Runner session-brief tests — ticket agent-brief-session-save-never-fires-2026-08-13.
 *
 * The Runner is the only writer of the per-{project, agent} brief. The write
 * must be deterministic (no LLM), capped, and completely non-fatal: the task
 * is already marked complete by the time it runs.
 *
 * Run: node --test act-agent/runner/act-runner-brief.test.mjs
 */

import test from 'node:test';
import assert from 'node:assert/strict';

// Minimal env so importing the module has no main() side-effects (act-runner.mjs
// guards main() behind an isDirectInvocation check).
process.env.ACT_SERVER_URL = 'http://127.0.0.1:0';
process.argv = [process.argv[0], '/tmp/__not__act_runner__.mjs', '--agent-id', 'backend-1', '--name', 'test'];

const { saveAgentBrief, buildRecentWorkBrief } = await import('./act-runner.mjs');

const HEADER = '## Recent Work (most recent first)';

test('buildRecentWorkBrief: newest first, capped at 5 entries', () => {
  let brief = '';
  for (let i = 1; i <= 7; i++) {
    brief = buildRecentWorkBrief(brief, `- [t${i}] task ${i} — did thing ${i}`);
  }
  const entries = brief.split('\n').filter(l => l.startsWith('- '));
  assert.equal(entries.length, 5);
  assert.ok(entries[0].includes('task 7'), `newest first, got: ${entries[0]}`);
  assert.ok(entries[4].includes('task 3'), `oldest kept is task 3, got: ${entries[4]}`);
  assert.ok(!brief.includes('task 2'), 'oldest entries trimmed');
});

test('buildRecentWorkBrief: preserves any non-section preamble', () => {
  const preamble = '# Project Brief\n\nYou are the backend agent.';
  const brief = buildRecentWorkBrief(preamble, '- [t1] first task — ok');
  assert.ok(brief.startsWith(preamble), 'preamble preserved verbatim');
  assert.ok(brief.includes(HEADER));
  assert.equal(brief.split(HEADER).length - 1, 1, 'header not duplicated');

  const brief2 = buildRecentWorkBrief(brief, '- [t2] second task — ok');
  assert.ok(brief2.startsWith(preamble));
  assert.equal(brief2.split(HEADER).length - 1, 1, 'header still not duplicated');
  assert.equal(brief2.split('\n').filter(l => l.startsWith('- ')).length, 2);
});

test('buildRecentWorkBrief: stays within the 2000-char cap, trimming oldest', () => {
  const big = 'x'.repeat(600);
  let brief = '';
  for (let i = 1; i <= 6; i++) {
    brief = buildRecentWorkBrief(brief, `- [t${i}] task ${i} — ${big}`);
    assert.ok(brief.length <= 2000, `brief exceeded cap at i=${i}: ${brief.length}`);
  }
  assert.ok(brief.includes('task 6'), 'newest entry survives trimming');
});

test('saveAgentBrief: first completion (no brief yet) POSTs a one-entry brief', async () => {
  const posts = [];
  const ok = await saveAgentBrief(
    'authsvc-beta',
    { id: 'task-1', title: 'Build /login endpoint' },
    'Added POST /login with bcrypt hashing.\nAll tests pass.',
    {
      get: async () => { throw new Error('HTTP 404 GET /briefs: No brief for this agent'); },
      post: async (path, body) => { posts.push({ path, body }); return {}; },
      log: () => {},
      now: () => '2026-08-19T00:00:00.000Z',
    }
  );

  assert.equal(ok, true);
  assert.equal(posts.length, 1);
  assert.equal(posts[0].path, '/api/projects/authsvc-beta/briefs');
  assert.equal(posts[0].body.agentId, 'backend-1');
  assert.ok(posts[0].body.content.includes(HEADER));
  assert.ok(posts[0].body.content.includes('Build /login endpoint'));
  // Result collapsed to one line
  assert.ok(posts[0].body.content.includes('Added POST /login with bcrypt hashing. All tests pass.'));
  assert.equal(posts[0].body.content.split('\n').filter(l => l.startsWith('- ')).length, 1);
});

test('saveAgentBrief: read-modify-write appends to an existing brief', async () => {
  const existing = `${HEADER}\n\n- [2026-08-18T00:00:00.000Z] Earlier task — done`;
  const posts = [];
  await saveAgentBrief(
    'authsvc-beta',
    { id: 'task-2', title: 'Add JWT refresh' },
    'refresh endpoint live',
    {
      get: async () => ({ success: true, content: existing }),
      post: async (path, body) => { posts.push({ path, body }); return {}; },
      log: () => {},
      now: () => '2026-08-19T00:00:00.000Z',
    }
  );
  const lines = posts[0].body.content.split('\n').filter(l => l.startsWith('- '));
  assert.equal(lines.length, 2);
  assert.ok(lines[0].includes('Add JWT refresh'));
  assert.ok(lines[1].includes('Earlier task'));
});

test('saveAgentBrief: a 500 on the brief write is swallowed (task completion unaffected)', async () => {
  const logged = [];
  const ok = await saveAgentBrief(
    'authsvc-beta',
    { id: 'task-3', title: 'anything' },
    'result',
    {
      get: async () => ({ content: '' }),
      post: async () => { throw new Error('HTTP 500 POST /api/projects/authsvc-beta/briefs: boom'); },
      log: (m) => { logged.push(m); },
      now: () => '2026-08-19T00:00:00.000Z',
    }
  );
  assert.equal(ok, false, 'reports failure via return value, never by throwing');
  assert.ok(logged.some(m => m.includes('non-fatal')), `expected a non-fatal log line, got: ${logged.join(' | ')}`);
});

test('saveAgentBrief: no project name is a no-op, not a throw', async () => {
  const ok = await saveAgentBrief(null, { id: 't', title: 'x' }, 'r', {
    get: async () => { throw new Error('must not be called'); },
    post: async () => { throw new Error('must not be called'); },
    log: () => {},
  });
  assert.equal(ok, false);
});

test('saveAgentBrief: no LLM/subprocess in the write path (deterministic)', async () => {
  const posts = [];
  const args = [
    'proj',
    { id: 'task-4', title: 'Deterministic check' },
    'same result',
    {
      get: async () => ({ content: '' }),
      post: async (path, body) => { posts.push(body.content); return {}; },
      log: () => {},
      now: () => '2026-08-19T00:00:00.000Z',
    },
  ];
  await saveAgentBrief(...args);
  await saveAgentBrief(...args);
  assert.equal(posts[0], posts[1], 'same inputs produce byte-identical briefs');
});
