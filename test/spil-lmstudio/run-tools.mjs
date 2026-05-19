#!/usr/bin/env node
// Tool-use SPIL runner.
// Usage: node run-tools.mjs [--model <id>] [--fixture <name>] [--tier <swarm|planner|smoke>] [--max-iter N]

import { readFileSync, writeFileSync, readdirSync, mkdirSync, existsSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { TOOL_SCHEMAS, makeToolExecutor, resetSandbox, seedSandbox, readSandboxFile } from './tools.mjs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const config = JSON.parse(readFileSync(join(__dirname, 'models.json'), 'utf8'));

const args = process.argv.slice(2);
const flag = (name) => {
  const i = args.indexOf(`--${name}`);
  return i >= 0 ? args[i + 1] : null;
};
const onlyModel = flag('model');
const onlyFixture = flag('fixture');
const onlyTier = flag('tier');
const maxIter = parseInt(flag('max-iter') ?? '12', 10);

const models = config.models.filter(m => {
  if (onlyModel) return m.id === onlyModel;
  if (onlyTier) return m.tier === onlyTier;
  return true;
});

const fixturesDir = join(__dirname, 'fixtures');
const fixtureFiles = readdirSync(fixturesDir)
  .filter(f => f.startsWith('tool-') && f.endsWith('.spec.json'))
  .filter(f => !onlyFixture || f === `${onlyFixture}.spec.json` || f.startsWith(`${onlyFixture}.`));

const runId = new Date().toISOString().replace(/[:.]/g, '-');
const runDir = join(__dirname, 'results-tools', runId);
mkdirSync(runDir, { recursive: true });
const sandboxBase = join(__dirname, 'sandbox', runId);
mkdirSync(sandboxBase, { recursive: true });
const results = [];

function slug(s) { return String(s).replace(/[^a-z0-9.-]/gi, '_'); }

async function chatTurn(model, messages, opts = {}) {
  const body = {
    model: model.id,
    messages,
    temperature: opts.temperature ?? 0.2,
    max_tokens: opts.maxTokens ?? 2000,
    tools: TOOL_SCHEMAS,
    tool_choice: 'auto',
  };
  const ctk = { ...(model.chatTemplateKwargs ?? {}), ...(opts.chatTemplateKwargs ?? {}) };
  if (Object.keys(ctk).length) body.chat_template_kwargs = ctk;

  const ctrl = new AbortController();
  const timeout = setTimeout(() => ctrl.abort(), opts.timeoutMs ?? 120000);
  try {
    const res = await fetch(config.endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      signal: ctrl.signal,
    });
    clearTimeout(timeout);
    if (!res.ok) return { ok: false, error: `HTTP ${res.status}: ${await res.text()}` };
    return { ok: true, data: await res.json() };
  } catch (e) {
    clearTimeout(timeout);
    return { ok: false, error: String(e.message || e) };
  }
}

function evaluateAssertions(sandboxRoot, assertions) {
  const results = [];
  for (const a of assertions) {
    try {
      if (a.type === 'file_exists') {
        const content = readSandboxFile(sandboxRoot, a.path);
        results.push({ ...a, pass: content !== null, got: content === null ? 'missing' : 'present' });
      } else if (a.type === 'file_contains') {
        const content = readSandboxFile(sandboxRoot, a.path);
        const pass = content !== null && content.includes(a.substring);
        results.push({ ...a, pass, got: content === null ? 'missing' : content.includes(a.substring) ? 'found' : 'not found' });
      } else if (a.type === 'json_has_keys') {
        const content = readSandboxFile(sandboxRoot, a.path);
        if (content === null) { results.push({ ...a, pass: false, got: 'file missing' }); continue; }
        let parsed;
        try { parsed = JSON.parse(content); } catch (e) { results.push({ ...a, pass: false, got: `invalid JSON: ${e.message}` }); continue; }
        const missing = (a.keys || []).filter(k => {
          const segs = k.split('.');
          let cur = parsed;
          for (const s of segs) { if (cur == null || !(s in cur)) return true; cur = cur[s]; }
          return false;
        });
        results.push({ ...a, pass: missing.length === 0, got: missing.length ? `missing keys: ${missing.join(',')}` : 'all keys present' });
      } else if (a.type === 'json_equals') {
        const content = readSandboxFile(sandboxRoot, a.path);
        if (content === null) { results.push({ ...a, pass: false, got: 'file missing' }); continue; }
        let parsed;
        try { parsed = JSON.parse(content); } catch (e) { results.push({ ...a, pass: false, got: `invalid JSON: ${e.message}` }); continue; }
        const segs = a.path_in_json.split('.');
        let cur = parsed;
        for (const s of segs) { if (cur == null) break; cur = cur[s]; }
        const pass = JSON.stringify(cur) === JSON.stringify(a.value);
        results.push({ ...a, pass, got: JSON.stringify(cur) });
      } else {
        results.push({ ...a, pass: false, got: `unknown assertion type: ${a.type}` });
      }
    } catch (e) {
      results.push({ ...a, pass: false, got: `error: ${e.message}` });
    }
  }
  return results;
}

async function runFixture(model, fixtureFile) {
  const spec = JSON.parse(readFileSync(join(fixturesDir, fixtureFile), 'utf8'));
  const sandboxRoot = join(sandboxBase, `${slug(model.id)}__${spec.id}`);
  resetSandbox(sandboxRoot);
  if (spec.seed) seedSandbox(sandboxRoot, spec.seed);

  const executor = makeToolExecutor(sandboxRoot);
  const baseSystem = (spec.system || '') + (model.userSuffix ? '' : '');
  let messages = [
    { role: 'system', content: baseSystem },
    { role: 'user', content: (spec.user || '') + (model.userSuffix ? `\n${model.userSuffix}` : '') },
  ];

  const t0 = Date.now();
  let iter = 0;
  let stop = null;
  let totalUsage = { prompt_tokens: 0, completion_tokens: 0 };

  while (iter < maxIter) {
    iter++;
    const r = await chatTurn(model, messages, spec.opts || {});
    if (!r.ok) { stop = `chat error: ${r.error}`; break; }
    const choice = r.data.choices?.[0];
    const msg = choice?.message;
    if (r.data.usage) {
      totalUsage.prompt_tokens += r.data.usage.prompt_tokens || 0;
      totalUsage.completion_tokens += r.data.usage.completion_tokens || 0;
    }
    if (!msg) { stop = 'no message in response'; break; }

    const toolCalls = msg.tool_calls || [];
    // Push assistant message verbatim
    messages.push({
      role: 'assistant',
      content: msg.content ?? null,
      tool_calls: toolCalls.length ? toolCalls : undefined,
    });

    if (!toolCalls.length) {
      // No more tool calls — model wrote a final message. End loop.
      stop = msg.content ? 'final-message' : 'empty-response';
      break;
    }

    // Execute each tool call, push tool results
    for (const tc of toolCalls) {
      const name = tc.function?.name;
      let argsObj;
      try { argsObj = JSON.parse(tc.function?.arguments ?? '{}'); }
      catch (e) { argsObj = {}; }
      const result = executor.execute(name, argsObj);
      messages.push({
        role: 'tool',
        tool_call_id: tc.id,
        content: typeof result === 'string' ? result : JSON.stringify(result),
      });
    }

    // If model called mark_complete this turn, exit after the next assistant turn (let it acknowledge)
    if (executor.state().completeCalled) {
      // One more turn to let model wrap up
      const finalR = await chatTurn(model, messages, spec.opts || {});
      if (finalR.ok && finalR.data.choices?.[0]?.message) {
        messages.push(finalR.data.choices[0].message);
      }
      stop = 'mark_complete';
      break;
    }
  }
  if (!stop) stop = `max-iter (${maxIter})`;

  // Score
  const assertResults = evaluateAssertions(sandboxRoot, spec.assertions || []);
  const passed = assertResults.filter(r => r.pass).length;
  const total = assertResults.length || 1;
  const score = Math.round((passed / total) * 100);

  // Dump
  const dumpName = `${slug(model.id)}__${spec.id}`;
  const dumpPath = join(runDir, `${dumpName}.md`);
  const dump = [
    `# ${model.id} — ${spec.id} (tool-use)`,
    ``,
    `**score:** ${score}/100 (${passed}/${total} assertions)`,
    `**stop reason:** ${stop}`,
    `**iterations:** ${iter}/${maxIter}`,
    `**elapsed:** ${Date.now() - t0}ms`,
    `**usage:** ${JSON.stringify(totalUsage)}`,
    ``,
    `## Assertions`,
    ...assertResults.map(r => `- ${r.pass ? '✓' : '✗'} \`${r.type}\` ${r.path || r.path_in_json || ''} — ${r.got}`),
    ``,
    `## Tool call log`,
    ...executor.state().log.map((e, i) => `${i + 1}. ${e.ok ? '✓' : '✗'} **${e.name}** (${e.ms}ms)  \n   args: \`${JSON.stringify(e.args).slice(0, 200)}\`  \n   result: \`${String(e.result).slice(0, 200).replace(/\n/g, '\\n')}\``),
    ``,
    `## Conversation (assistant + tool messages only, content trimmed)`,
    ...messages.filter(m => m.role !== 'system' && m.role !== 'user').map(m => {
      const lines = [`**${m.role}**${m.tool_call_id ? ` (tool_call_id=${m.tool_call_id})` : ''}:`];
      if (m.content) lines.push('```', String(m.content).slice(0, 600), '```');
      if (m.tool_calls) {
        for (const tc of m.tool_calls) lines.push(`  → calls \`${tc.function?.name}\`(${String(tc.function?.arguments).slice(0, 200)})`);
      }
      return lines.join('\n');
    }),
  ].join('\n');
  writeFileSync(dumpPath, dump);

  return {
    model: model.id,
    fixture: spec.id,
    score,
    passed,
    total,
    stop,
    iter,
    elapsedMs: Date.now() - t0,
    usage: totalUsage,
    dumpFile: `${dumpName}.md`,
  };
}

async function main() {
  console.error(`\n=== SPIL LM Studio Tool-Use Run ${runId} ===`);
  console.error(`Models: ${models.map(m => m.id).join(', ')}`);
  console.error(`Fixtures: ${fixtureFiles.join(', ')}\n`);

  for (const model of models) {
    console.error(`\n[${model.id}]`);
    for (const fixtureFile of fixtureFiles) {
      console.error(`  → ${fixtureFile}`);
      const r = await runFixture(model, fixtureFile);
      results.push(r);
      console.error(`    score=${r.score}/100 (${r.passed}/${r.total})  iter=${r.iter}  ${r.elapsedMs}ms  stop=${r.stop}`);
    }
  }

  writeFileSync(join(runDir, 'summary.json'), JSON.stringify(results, null, 2));
  console.error(`\nResults: ${runDir}`);
  console.error('\n=== Matrix ===');
  const byModel = {};
  for (const r of results) (byModel[r.model] ||= []).push(r);
  for (const [m, rs] of Object.entries(byModel)) {
    const avg = (rs.reduce((s, r) => s + r.score, 0) / rs.length).toFixed(1);
    console.error(`  ${m.padEnd(40)} avg=${avg}`);
  }
}

main().catch(e => { console.error(e); process.exit(1); });
