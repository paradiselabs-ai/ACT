#!/usr/bin/env node
// Lazy-fetch SPIL A/B test runner.
// Four arms: A (full-SPIL), B1 (lazy minimal manifest), B2 (lazy guided manifest), C (full-plain).
// Usage:
//   node run-lazy.mjs --model <id> --arm <A|B1|B2|C> --trials N
//   node run-lazy.mjs --models id1,id2,... --arms A,B1,B2,C --trials N
//   node run-lazy.mjs --fixture lazy-01-api-bootstrap --models ... --arms ... --trials N

import { readFileSync, writeFileSync, readdirSync, mkdirSync, existsSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  TOOL_SCHEMAS,
  LAZY_TOOL_SCHEMAS,
  makeToolExecutor,
  resetSandbox,
  readSandboxFile,
} from './tools.mjs';
import { splitSpilSections, formatManifest, assembleFullSpil } from './spil-manifest.mjs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const config = JSON.parse(readFileSync(join(__dirname, 'models.json'), 'utf8'));

// ---------- arg parsing ----------
const args = process.argv.slice(2);
function getArg(name, fallback = null) {
  const i = args.indexOf(`--${name}`);
  return i >= 0 ? args[i + 1] : fallback;
}
const fixtureName = getArg('fixture', 'lazy-01-api-bootstrap');
const singleModel = getArg('model');
const modelList = getArg('models');
const singleArm = getArg('arm');
const armList = getArg('arms');
const trials = parseInt(getArg('trials', '2'), 10);
const seed = parseInt(getArg('seed', String(Date.now())), 10);

const modelIds = singleModel
  ? [singleModel]
  : (modelList || 'openai/gpt-oss-20b,qwen/qwen2.5-coder-14b,google/gemma-3-12b')
      .split(',')
      .map(s => s.trim());
const models = config.models.filter(m => modelIds.includes(m.id));
if (models.length !== modelIds.length) {
  const found = models.map(m => m.id);
  const missing = modelIds.filter(id => !found.includes(id));
  console.error(`WARN: models not in models.json: ${missing.join(', ')}`);
}

const arms = (singleArm ? [singleArm] : (armList || 'A,B1,B2,C').split(','))
  .map(s => s.trim())
  .filter(Boolean);

// ---------- fixture loading ----------
const fixtureDir = join(__dirname, 'fixtures', fixtureName);
const fixture = JSON.parse(readFileSync(join(fixtureDir, 'fixture.json'), 'utf8'));
const specSpil = readFileSync(join(fixtureDir, fixture.specSpilPath), 'utf8');
const specPlain = readFileSync(join(fixtureDir, fixture.specPlainPath), 'utf8');
const manifestB1 = JSON.parse(readFileSync(join(fixtureDir, fixture.manifestB1Path), 'utf8'));
const manifestB2 = JSON.parse(readFileSync(join(fixtureDir, fixture.manifestB2Path), 'utf8'));

const { order: sectionOrder, sections: sectionMap } = splitSpilSections(specSpil);

// ---------- output paths ----------
const runId = new Date().toISOString().replace(/[:.]/g, '-');
const runDir = join(__dirname, 'results-lazy', runId);
mkdirSync(runDir, { recursive: true });
const sandboxBase = join(__dirname, 'sandbox', runId);
mkdirSync(sandboxBase, { recursive: true });
function slug(s) { return String(s).replace(/[^a-z0-9.-]/gi, '_'); }

// ---------- prompts ----------
const COMMON_RULES = `Rules:
- Work only inside the sandbox. All paths are relative; the sandbox is your root.
- When modifying or extending an existing file, READ IT FIRST so you preserve content.
- Re-read at least one of your written files with read_file before calling mark_complete to verify what was actually written.
- Do not call mark_complete until every requirement is verifiable from the sandbox state.
- Brief reasoning is fine; do not narrate every step.
- DO NOT invoke npm, vitest, or any external process. This bootstrap is graded by file contents and structure only.`;

const TOOLS_BLOCK_BASE = `You have these tools:
- read_file(path) — read a sandbox file
- write_file(path, content) — write a sandbox file (overwrites)
- list_dir(path) — list a sandbox directory
- mark_complete(summary) — call EXACTLY ONCE when ALL requirements are satisfied. After this, grading proceeds.`;

const TOOLS_BLOCK_LAZY = `${TOOLS_BLOCK_BASE}
- spil_get(section) — fetch the body of one SPIL section by name (e.g. spil_get('endpoints')). You start with only the MANIFEST. You MUST call spil_get to read any section before writing files that depend on it. The manifest lists what's available — fetch what you need.`;

const SPIL_PRIMER = `You are a swarm agent in the ACT (Agent Coordination Toolkit) system. You execute tasks written in SPIL (Structured Progressive Instruction Language).

SPIL syntax:
- @keyword "value" or @keyword: — structured section markers
- - item — list item under the current section
- > "text" — natural language directive placed where most relevant

SPIL ordering (CTD: Conceptually Top-Down) — every section depends only on what's above it. Read top to bottom.`;

const SPIL_PRIMER_LAZY = `${SPIL_PRIMER}

This task is delivered LAZILY. You will see a manifest of section names + 1-line hints. The bodies are NOT in your context yet. Use the spil_get tool to fetch any section you need to do your work. Do not guess section contents — fetch them. Fetch a section ONCE; once fetched, its body is in the conversation history and you can refer back to it.`;

const PLAIN_PRIMER = `You are a coding assistant. You complete tasks by using the available tools to read and write files in a sandbox.`;

function buildMessages(arm) {
  if (arm === 'A') {
    const system = `${SPIL_PRIMER}\n\n${TOOLS_BLOCK_BASE}\n\n${COMMON_RULES}`;
    const user = `Below is your SPIL task spec.\n\n${assembleFullSpil(sectionOrder, sectionMap)}`;
    return { system, user };
  }
  if (arm === 'B1') {
    const system = `${SPIL_PRIMER_LAZY}\n\n${TOOLS_BLOCK_LAZY}\n\n${COMMON_RULES}`;
    const user = `Below is the MANIFEST of your SPIL task spec. Fetch sections you need with spil_get.\n\n${formatManifest(sectionOrder, manifestB1.hints)}`;
    return { system, user };
  }
  if (arm === 'B2') {
    const system = `${SPIL_PRIMER_LAZY}\n\n${TOOLS_BLOCK_LAZY}\n\n${COMMON_RULES}`;
    const user = `Below is the MANIFEST of your SPIL task spec. Each entry includes when in the task to fetch it. Use spil_get with the section name to load each.\n\n${formatManifest(sectionOrder, manifestB2.hints)}`;
    return { system, user };
  }
  if (arm === 'C') {
    const system = `${PLAIN_PRIMER}\n\n${TOOLS_BLOCK_BASE}\n\n${COMMON_RULES}`;
    const user = specPlain;
    return { system, user };
  }
  throw new Error(`unknown arm: ${arm}`);
}

// ---------- LLM call ----------
async function chatTurn(model, messages, opts, useTools) {
  const body = {
    model: model.id,
    messages,
    temperature: opts.temperature ?? 0.2,
    max_tokens: opts.maxTokens ?? 4000,
    tools: useTools.tools,
    tool_choice: 'auto',
  };
  const ctk = { ...(model.chatTemplateKwargs ?? {}) };
  if (Object.keys(ctk).length) body.chat_template_kwargs = ctk;

  const ctrl = new AbortController();
  const timeout = setTimeout(() => ctrl.abort(), opts.timeoutMs ?? 180000);
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

// ---------- assertion scoring (mirrors run-tools.mjs) ----------
function getInJson(obj, dottedPath) {
  return dottedPath.split('.').reduce((cur, k) => (cur == null ? cur : cur[k]), obj);
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
        results.push({ ...a, pass, got: content === null ? 'missing' : pass ? 'found' : 'not found' });
      } else if (a.type === 'json_has_keys') {
        const content = readSandboxFile(sandboxRoot, a.path);
        if (content === null) { results.push({ ...a, pass: false, got: 'file missing' }); continue; }
        let parsed;
        try { parsed = JSON.parse(content); }
        catch (e) { results.push({ ...a, pass: false, got: `invalid JSON: ${e.message}` }); continue; }
        const missing = (a.keys || []).filter(k => getInJson(parsed, k) === undefined);
        results.push({ ...a, pass: missing.length === 0, got: missing.length ? `missing: ${missing.join(',')}` : 'all keys present' });
      } else if (a.type === 'json_equals') {
        const content = readSandboxFile(sandboxRoot, a.path);
        if (content === null) { results.push({ ...a, pass: false, got: 'file missing' }); continue; }
        let parsed;
        try { parsed = JSON.parse(content); }
        catch (e) { results.push({ ...a, pass: false, got: `invalid JSON: ${e.message}` }); continue; }
        const got = getInJson(parsed, a.path_in_json);
        const pass = JSON.stringify(got) === JSON.stringify(a.value);
        results.push({ ...a, pass, got: JSON.stringify(got) });
      } else {
        results.push({ ...a, pass: false, got: `unknown assertion type: ${a.type}` });
      }
    } catch (e) {
      results.push({ ...a, pass: false, got: `error: ${e.message}` });
    }
  }
  return results;
}

// ---------- trial execution ----------
async function runTrial(model, arm, trialIdx) {
  const sandboxRoot = join(sandboxBase, `${slug(model.id)}__${arm}__t${trialIdx}`);
  resetSandbox(sandboxRoot);

  const lazy = arm === 'B1' || arm === 'B2';
  const useTools = { tools: lazy ? LAZY_TOOL_SCHEMAS : TOOL_SCHEMAS };
  const executor = makeToolExecutor(sandboxRoot, {
    spilSections: lazy ? sectionMap : null,
  });

  const { system, user } = buildMessages(arm);
  const messages = [
    { role: 'system', content: system },
    { role: 'user', content: user },
  ];

  const t0 = Date.now();
  const maxIter = fixture.opts?.maxIter ?? 24;
  let iter = 0;
  let stop = null;
  let totalUsage = { prompt_tokens: 0, completion_tokens: 0 };
  let nudgesUsed = 0;

  while (iter < maxIter) {
    iter++;
    const r = await chatTurn(model, messages, fixture.opts || {}, useTools);
    if (!r.ok) { stop = `chat-error: ${r.error}`; break; }
    const msg = r.data.choices?.[0]?.message;
    const usage = r.data.usage;
    if (usage) {
      totalUsage.prompt_tokens += usage.prompt_tokens || 0;
      totalUsage.completion_tokens += usage.completion_tokens || 0;
    }
    if (!msg) { stop = 'no-message'; break; }

    const toolCalls = msg.tool_calls || [];
    messages.push({
      role: 'assistant',
      content: msg.content ?? null,
      tool_calls: toolCalls.length ? toolCalls : undefined,
    });

    if (!toolCalls.length) {
      // Model emitted text-only. If it hasn't called mark_complete, nudge it
      // ONCE per content-only turn (capped at 3 nudges total per trial) before giving up.
      if (executor.state().completeCalled) { stop = 'mark_complete'; break; }
      nudgesUsed = (nudgesUsed ?? 0) + 1;
      if (nudgesUsed > 3) { stop = msg.content ? 'too-many-nudges' : 'empty-response'; break; }
      messages.push({
        role: 'user',
        content: 'Continue. If any required file is still missing, write it now using write_file. If every requirement is met, verify by re-reading one file and then call mark_complete. Do not stop until mark_complete is called or you have determined the task cannot be completed.',
      });
      continue;
    }

    for (const tc of toolCalls) {
      const name = tc.function?.name;
      let argsObj = {};
      try { argsObj = JSON.parse(tc.function?.arguments ?? '{}'); } catch (_) { /* keep empty */ }
      const result = executor.execute(name, argsObj);
      messages.push({
        role: 'tool',
        tool_call_id: tc.id,
        content: typeof result === 'string' ? result : JSON.stringify(result),
      });
    }

    if (executor.state().completeCalled) {
      const finalR = await chatTurn(model, messages, fixture.opts || {}, useTools);
      if (finalR.ok && finalR.data.usage) {
        totalUsage.prompt_tokens += finalR.data.usage.prompt_tokens || 0;
        totalUsage.completion_tokens += finalR.data.usage.completion_tokens || 0;
      }
      if (finalR.ok && finalR.data.choices?.[0]?.message) {
        messages.push(finalR.data.choices[0].message);
      }
      stop = 'mark_complete';
      break;
    }
  }
  if (!stop) stop = `max-iter(${maxIter})`;

  // score
  const assertResults = evaluateAssertions(sandboxRoot, fixture.assertions);
  const passed = assertResults.filter(r => r.pass).length;
  const total = assertResults.length || 1;
  const score = Math.round((passed / total) * 100);

  const execState = executor.state();
  const fetchSet = new Set(execState.fetches);

  // dump
  const dumpName = `${slug(model.id)}__${arm}__t${trialIdx}`;
  const dumpPath = join(runDir, `${dumpName}.md`);
  const dump = [
    `# ${model.id} — arm ${arm} — trial ${trialIdx}`,
    ``,
    `**score:** ${score}/100 (${passed}/${total} assertions)`,
    `**stop:** ${stop}`,
    `**iterations:** ${iter}/${maxIter}`,
    `**elapsed:** ${Date.now() - t0}ms`,
    `**tokens:** prompt=${totalUsage.prompt_tokens} completion=${totalUsage.completion_tokens} total=${totalUsage.prompt_tokens + totalUsage.completion_tokens}`,
    `**spil_get calls:** ${execState.fetches.length} (unique: ${fetchSet.size})`,
    `**sections fetched:** ${[...fetchSet].join(', ') || '(none)'}`,
    ``,
    `## Assertions`,
    ...assertResults.map(r => `- ${r.pass ? '✓' : '✗'} \`${r.type}\` ${r.path || ''} ${r.path_in_json ? `[${r.path_in_json}]` : ''} — ${r.got}`),
    ``,
    `## Tool call log`,
    ...execState.log.map((e, i) => `${i + 1}. ${e.ok ? '✓' : '✗'} **${e.name}**  args=\`${JSON.stringify(e.args).slice(0, 200)}\`  → \`${String(e.result).slice(0, 160).replace(/\n/g, '\\n')}\``),
  ].join('\n');
  writeFileSync(dumpPath, dump);

  return {
    model: model.id,
    arm,
    trial: trialIdx,
    score,
    passed,
    total,
    iter,
    stop,
    elapsedMs: Date.now() - t0,
    promptTokens: totalUsage.prompt_tokens,
    completionTokens: totalUsage.completion_tokens,
    totalTokens: totalUsage.prompt_tokens + totalUsage.completion_tokens,
    fetchCount: execState.fetches.length,
    fetchUnique: fetchSet.size,
    sectionsFetched: [...fetchSet].join('|'),
    dumpFile: `${dumpName}.md`,
  };
}

// ---------- shuffle (seeded for reproducibility) ----------
function mulberry32(s) { return () => { s |= 0; s = (s + 0x6D2B79F5) | 0; let t = Math.imul(s ^ (s >>> 15), 1 | s); t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t; return ((t ^ (t >>> 14)) >>> 0) / 4294967296; }; }
function shuffle(arr, rng) {
  const a = [...arr];
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(rng() * (i + 1));
    [a[i], a[j]] = [a[j], a[i]];
  }
  return a;
}

// ---------- main ----------
async function main() {
  const rng = mulberry32(seed);
  console.error(`\n=== Lazy-Fetch SPIL Run ${runId} ===`);
  console.error(`Fixture: ${fixtureName}`);
  console.error(`Models: ${models.map(m => m.id).join(', ')}`);
  console.error(`Arms: ${arms.join(', ')}`);
  console.error(`Trials per (model, arm): ${trials}`);
  console.error(`Seed: ${seed}`);
  console.error(`Sections in spec: ${sectionOrder.length} — ${sectionOrder.join(', ')}\n`);

  const results = [];
  for (const model of models) {
    // Build a shuffled (arm, trialIdx) schedule per model — defeats cold-load + warm-cache bias
    const schedule = [];
    for (const arm of arms) for (let t = 1; t <= trials; t++) schedule.push({ arm, trial: t });
    const order = shuffle(schedule, rng);
    console.error(`[${model.id}] schedule: ${order.map(o => `${o.arm}#${o.trial}`).join(' ')}`);
    for (const { arm, trial } of order) {
      const r = await runTrial(model, arm, trial);
      results.push(r);
      console.error(`  ${arm} t${trial}  score=${r.score}/100 (${r.passed}/${r.total})  iter=${r.iter}  total_tok=${r.totalTokens}  fetches=${r.fetchCount}  stop=${r.stop}`);
    }
  }

  // ---------- emit CSV ----------
  const csvPath = join(runDir, 'trials.csv');
  const csvHeader = 'model,arm,trial,score,passed,total,iter,stop,elapsed_ms,prompt_tokens,completion_tokens,total_tokens,fetch_count,fetch_unique,sections_fetched';
  const csvRows = results.map(r => [
    r.model, r.arm, r.trial, r.score, r.passed, r.total, r.iter, r.stop, r.elapsedMs,
    r.promptTokens, r.completionTokens, r.totalTokens, r.fetchCount, r.fetchUnique,
    JSON.stringify(r.sectionsFetched),
  ].join(','));
  writeFileSync(csvPath, [csvHeader, ...csvRows].join('\n'));

  // ---------- aggregate ----------
  const armKey = a => ({ A: 'A:full-SPIL', B1: 'B1:lazy-minimal', B2: 'B2:lazy-guided', C: 'C:full-plain' }[a] || a);
  const armOrder = ['A', 'B1', 'B2', 'C'].filter(a => arms.includes(a));
  const byArm = {};
  for (const r of results) (byArm[r.arm] ||= []).push(r);
  const avg = (rs, k) => rs.length ? (rs.reduce((s, r) => s + r[k], 0) / rs.length) : 0;
  const aggRows = armOrder.map(a => {
    const rs = byArm[a] || [];
    return {
      arm: a,
      label: armKey(a),
      n: rs.length,
      avgScore: avg(rs, 'score').toFixed(1),
      avgIter: avg(rs, 'iter').toFixed(1),
      avgPromptTok: Math.round(avg(rs, 'promptTokens')),
      avgComplTok: Math.round(avg(rs, 'completionTokens')),
      avgTotalTok: Math.round(avg(rs, 'totalTokens')),
      avgFetches: avg(rs, 'fetchCount').toFixed(1),
      avgFetchUnique: avg(rs, 'fetchUnique').toFixed(1),
    };
  });

  // baseline = arm A
  const baseA = aggRows.find(r => r.arm === 'A');
  function pct(now, base) {
    if (!base || base === 0) return '-';
    const d = ((now - base) / base) * 100;
    return `${d >= 0 ? '+' : ''}${d.toFixed(1)}%`;
  }

  // hypothesis verdicts
  const A = byArm['A'] || [], B1 = byArm['B1'] || [], B2 = byArm['B2'] || [], C = byArm['C'] || [];
  const avgPT = rs => avg(rs, 'promptTokens');
  const avgTT = rs => avg(rs, 'totalTokens');
  const avgScore = rs => avg(rs, 'score');
  const h1Delta = baseA && B2.length ? ((avgPT(B2) - avgPT(A)) / avgPT(A)) * 100 : null;
  const h1Pass = h1Delta != null && h1Delta <= -30;
  const h2Pass = [A, B1, B2, C].filter(rs => rs.length).every(rs => avgScore(rs) >= 90);
  const h3Delta = A.length && C.length ? ((avgTT(A) - avgTT(C)) / avgTT(C)) * 100 : null;
  const h3Pass = h3Delta != null && h3Delta <= 10;
  const h4ScorePass = B1.length && B2.length ? avgScore(B2) >= avgScore(B1) : false;
  const h4TokenPass = B1.length && B2.length ? (((avgTT(B2) - avgTT(B1)) / avgTT(B1)) * 100) <= 15 : false;
  const h4Pass = h4ScorePass && h4TokenPass;

  const mdLines = [
    `# Lazy-Fetch SPIL — Aggregate Report`,
    ``,
    `**Run:** \`${runId}\``,
    `**Fixture:** \`${fixtureName}\` (${sectionOrder.length} sections)`,
    `**Models:** ${models.map(m => m.id).join(', ')}`,
    `**Trials per (model, arm):** ${trials}`,
    `**Seed:** ${seed}`,
    ``,
    `## Per-arm averages (across all models × trials)`,
    ``,
    `| Arm | n | Avg score | Avg iter | Prompt tok | Compl tok | Total tok | vs A total | Fetches (unique/total) |`,
    `|---|---|---|---|---|---|---|---|---|`,
    ...aggRows.map(r => `| ${r.label} | ${r.n} | ${r.avgScore} | ${r.avgIter} | ${r.avgPromptTok} | ${r.avgComplTok} | ${r.avgTotalTok} | ${baseA ? pct(r.avgTotalTok, baseA.avgTotalTok) : '-'} | ${r.avgFetchUnique}/${r.avgFetches} |`),
    ``,
    `## Per-model breakdown`,
    ``,
  ];
  for (const m of models) {
    mdLines.push(`### ${m.id}`);
    mdLines.push('');
    mdLines.push('| Arm | Score | Iter | Prompt | Compl | Total | Fetches | Stop |');
    mdLines.push('|---|---|---|---|---|---|---|---|');
    const rs = results.filter(r => r.model === m.id);
    for (const r of rs) {
      mdLines.push(`| ${r.arm} t${r.trial} | ${r.score} | ${r.iter} | ${r.promptTokens} | ${r.completionTokens} | ${r.totalTokens} | ${r.fetchUnique}/${r.fetchCount} | ${r.stop} |`);
    }
    mdLines.push('');
  }

  mdLines.push(`## Hypothesis verdicts`);
  mdLines.push('');
  mdLines.push('```');
  mdLines.push(`H1: B2 prompt tokens ≥30% fewer than A     ${h1Pass ? 'PASS' : 'FAIL'}  [actual: ${h1Delta != null ? h1Delta.toFixed(1) + '%' : 'n/a'}]`);
  mdLines.push(`H2: all arms avg score ≥90                 ${h2Pass ? 'PASS' : 'FAIL'}  [A=${A.length ? avgScore(A).toFixed(1) : '-'} B1=${B1.length ? avgScore(B1).toFixed(1) : '-'} B2=${B2.length ? avgScore(B2).toFixed(1) : '-'} C=${C.length ? avgScore(C).toFixed(1) : '-'}]`);
  mdLines.push(`H3: A total tokens ≤ C total + 10%         ${h3Pass ? 'PASS' : 'FAIL'}  [delta: ${h3Delta != null ? h3Delta.toFixed(1) + '%' : 'n/a'}]`);
  mdLines.push(`H4: B2 score ≥ B1, B2 tokens ≤ B1 + 15%    ${h4Pass ? 'PASS' : 'FAIL'}  [score: B1=${B1.length ? avgScore(B1).toFixed(1) : '-'}, B2=${B2.length ? avgScore(B2).toFixed(1) : '-'}; token delta B2vsB1: ${B1.length && B2.length ? (((avgTT(B2) - avgTT(B1)) / avgTT(B1)) * 100).toFixed(1) + '%' : 'n/a'}]`);
  mdLines.push('```');
  mdLines.push('');
  mdLines.push(`## Files`);
  mdLines.push(`- \`trials.csv\` — raw per-trial data`);
  mdLines.push(`- \`<model>__<arm>__t<N>.md\` — per-trial dumps with tool log + assertion outcomes`);

  writeFileSync(join(runDir, 'aggregate.md'), mdLines.join('\n'));

  // console summary
  console.error('\n=== Aggregate ===');
  for (const r of aggRows) {
    console.error(`  ${r.label.padEnd(20)} n=${r.n}  score=${r.avgScore}  iter=${r.avgIter}  prompt=${r.avgPromptTok}  compl=${r.avgComplTok}  total=${r.avgTotalTok}  ${baseA ? pct(r.avgTotalTok, baseA.avgTotalTok) : ''}`);
  }
  console.error('\n=== Hypotheses ===');
  console.error(`  H1 B2≥30% fewer prompt tok vs A: ${h1Pass ? 'PASS' : 'FAIL'}  [${h1Delta != null ? h1Delta.toFixed(1) + '%' : 'n/a'}]`);
  console.error(`  H2 all arms ≥90 score:           ${h2Pass ? 'PASS' : 'FAIL'}`);
  console.error(`  H3 A ≤ C + 10% total tok:        ${h3Pass ? 'PASS' : 'FAIL'}  [${h3Delta != null ? h3Delta.toFixed(1) + '%' : 'n/a'}]`);
  console.error(`  H4 B2 score≥B1, B2 tok≤B1+15%:   ${h4Pass ? 'PASS' : 'FAIL'}`);
  console.error(`\nReport: ${runDir}/aggregate.md`);
  console.error(`CSV:    ${runDir}/trials.csv`);
}

main().catch(e => { console.error(e); process.exit(1); });
