#!/usr/bin/env node
// Run SPIL test fixtures against LM Studio models.
// Usage: node run.mjs [--model <id>] [--fixture <name>] [--tier <smoke|swarm|planner>]

import { readFileSync, writeFileSync, readdirSync, mkdirSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parseSPIL, stripThinking, extractJSON } from './spil-parser.mjs';

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

const models = config.models.filter(m => {
  if (onlyModel) return m.id === onlyModel;
  if (onlyTier) return m.tier === onlyTier;
  return true;
});

const fixturesDir = join(__dirname, 'fixtures');
const fixtureFiles = readdirSync(fixturesDir)
  .filter(f => f.endsWith('.spec.json'))
  .filter(f => !onlyFixture || f === `${onlyFixture}.spec.json`);

const runId = new Date().toISOString().replace(/[:.]/g, '-');
const runDir = join(__dirname, 'results', runId);
mkdirSync(runDir, { recursive: true });
const resultPath = join(runDir, 'summary.json');
const results = [];

function slug(s) { return String(s).replace(/[^a-z0-9.-]/gi, '_'); }

async function callModel(model, messages, opts = {}) {
  const t0 = Date.now();
  const body = {
    model: model.id,
    messages,
    temperature: opts.temperature ?? config.defaults.temperature,
    max_tokens: opts.maxTokens ?? config.defaults.maxTokens,
  };
  // Merge chat_template_kwargs from model defaults + fixture opts (fixture wins)
  const ctk = { ...(model.chatTemplateKwargs ?? {}), ...(opts.chatTemplateKwargs ?? {}) };
  if (Object.keys(ctk).length) body.chat_template_kwargs = ctk;
  const ctrl = new AbortController();
  const timeout = setTimeout(() => ctrl.abort(), opts.timeoutMs ?? config.defaults.timeoutMs);
  try {
    const res = await fetch(config.endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      signal: ctrl.signal,
    });
    clearTimeout(timeout);
    if (!res.ok) {
      const errText = await res.text();
      return { ok: false, error: `HTTP ${res.status}: ${errText}`, elapsedMs: Date.now() - t0 };
    }
    const json = await res.json();
    const content = json.choices?.[0]?.message?.content ?? '';
    const usage = json.usage ?? {};
    return {
      ok: true,
      content,
      stripped: stripThinking(content),
      usage,
      elapsedMs: Date.now() - t0,
      tokPerSec: usage.completion_tokens && (Date.now() - t0) > 0
        ? (usage.completion_tokens / ((Date.now() - t0) / 1000)).toFixed(2)
        : null,
    };
  } catch (e) {
    clearTimeout(timeout);
    return { ok: false, error: String(e.message || e), elapsedMs: Date.now() - t0 };
  }
}

function scoreParseFixture(spec, response) {
  if (!response.ok) return { score: 0, reasons: [`API error: ${response.error}`] };
  const parsed = extractJSON(response.stripped);
  if (!parsed) return { score: 0, reasons: ['No JSON object extractable from response'] };

  const expected = spec.expected;
  let score = 0;
  const reasons = [];

  // Section keyword set match (40 pts)
  const expectedKeywords = (expected.sections || []).map(s => s.keyword).sort();
  const gotKeywords = (parsed.sections || []).map(s => s.keyword).sort();
  const kwIntersection = expectedKeywords.filter(k => gotKeywords.includes(k));
  const kwScore = expectedKeywords.length ? 40 * (kwIntersection.length / expectedKeywords.length) : 40;
  score += kwScore;
  if (kwScore < 40) reasons.push(`section keywords: ${kwIntersection.length}/${expectedKeywords.length} (missing: ${expectedKeywords.filter(k => !gotKeywords.includes(k)).join(',')})`);

  // success_criteria items count + content (40 pts)
  const expectedSC = (expected.sections || []).find(s => s.keyword === 'success_criteria');
  const gotSC = (parsed.sections || []).find(s => s.keyword === 'success_criteria');
  if (expectedSC) {
    if (!gotSC) {
      reasons.push('missing @success_criteria section');
    } else {
      const expItems = expectedSC.items || [];
      const gotItems = gotSC.items || [];
      const countScore = 15 * Math.min(1, gotItems.length / Math.max(1, expItems.length));
      const overlapScore = 25 * (gotItems.filter(g => expItems.some(e => normalize(e) === normalize(g))).length / Math.max(1, expItems.length));
      score += countScore + overlapScore;
      if (countScore + overlapScore < 40) reasons.push(`success_criteria: matched ${gotItems.filter(g => expItems.some(e => normalize(e) === normalize(g))).length}/${expItems.length}`);
    }
  } else {
    score += 40;
  }

  // directives count (20 pts)
  const expectedDirCount = countAllDirectives(expected);
  const gotDirCount = countAllDirectives(parsed);
  const dirScore = expectedDirCount === 0
    ? (gotDirCount === 0 ? 20 : 10)
    : 20 * Math.min(1, gotDirCount / expectedDirCount);
  score += dirScore;
  if (dirScore < 20) reasons.push(`directives: got ${gotDirCount}, expected ${expectedDirCount}`);

  return { score: Math.round(score), reasons };
}

function countAllDirectives(doc) {
  return (doc.directives?.length || 0) + (doc.sections || []).reduce((n, s) => n + (s.directives?.length || 0), 0);
}

function normalize(s) {
  return String(s).toLowerCase().replace(/\s+/g, ' ').trim();
}

function scoreProduceFixture(spec, response) {
  if (!response.ok) return { score: 0, reasons: [`API error: ${response.error}`] };
  const text = response.stripped;
  let score = 0;
  const reasons = [];

  // Round-trip parse — does output parse as valid SPIL?
  const doc = parseSPIL(text);
  if (doc.sections.length === 0) {
    reasons.push('output contains no @sections — not SPIL');
    return { score: 0, reasons };
  }

  // Required keywords present (50 pts)
  const required = spec.expected.requiredKeywords || [];
  const found = required.filter(k => doc.sections.some(s => s.keyword === k));
  score += required.length ? 50 * (found.length / required.length) : 50;
  if (found.length < required.length) reasons.push(`missing required: ${required.filter(r => !found.includes(r)).join(',')}`);

  // success_criteria items >= min (30 pts)
  const minSC = spec.expected.minSuccessCriteria || 0;
  const sc = doc.sections.find(s => s.keyword === 'success_criteria');
  const scCount = sc?.items.length || 0;
  if (scCount >= minSC) score += 30;
  else { score += 30 * (scCount / Math.max(1, minSC)); reasons.push(`success_criteria: ${scCount} < ${minSC}`); }

  // No forbidden content (20 pts) — word-boundary aware (avoids false positives like "Sure" in "Ensure")
  const forbidden = spec.expected.forbiddenStrings || [];
  const violations = forbidden.filter(s => {
    const esc = s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    // Words: enforce word boundaries. Non-word patterns (e.g. ```): substring match is fine.
    const isWordLike = /^\w/.test(s) && /\w$/.test(s);
    const re = new RegExp(isWordLike ? `\\b${esc}\\b` : esc, 'i');
    return re.test(text);
  });
  if (violations.length === 0) score += 20;
  else reasons.push(`forbidden strings found: ${violations.join(',')}`);

  return { score: Math.round(score), reasons };
}

function scoreCTDFixture(spec, response) {
  if (!response.ok) return { score: 0, reasons: [`API error: ${response.error}`] };
  const parsed = extractJSON(response.stripped);
  if (!parsed) return { score: 0, reasons: ['No JSON object extractable from response'] };

  const exp = spec.expected;
  let score = 0;
  const reasons = [];

  // Correct valid verdict (40 pts)
  if (typeof parsed.valid !== 'boolean') {
    reasons.push('valid field missing or non-boolean');
  } else if (parsed.valid === exp.valid) {
    score += 40;
  } else {
    reasons.push(`valid=${parsed.valid}, expected ${exp.valid}`);
  }

  const violations = Array.isArray(parsed.violations) ? parsed.violations : [];

  // Violation count in range (20 pts)
  const minV = exp.violationsMin ?? 0;
  const maxV = exp.violationsMax ?? 999;
  if (violations.length >= minV && violations.length <= maxV) score += 20;
  else reasons.push(`violations count=${violations.length}, expected [${minV},${maxV}]`);

  // For invalid docs: target section identified (20 pts) + reason mentions expected concept (20 pts)
  if (exp.valid === false) {
    const targetSections = (exp.expectedViolationSections || []).map(s => s.toLowerCase());
    const gotSections = violations.map(v => String(v.section || '').toLowerCase());
    const sectionMatch = targetSections.some(t => gotSections.some(g => g.includes(t)));
    if (sectionMatch) score += 20;
    else reasons.push(`expected violation in section(s) [${targetSections.join(',')}], got [${gotSections.join(',')}]`);

    const concepts = (exp.expectedConceptInReason || []).map(c => c.toLowerCase());
    const reasonsBlob = violations.map(v => String(v.reason || '')).join(' ').toLowerCase();
    const conceptHit = concepts.some(c => reasonsBlob.includes(c));
    if (conceptHit) score += 20;
    else reasons.push(`reason did not reference expected concept (${concepts.slice(0,3).join('|')}...)`);
  } else {
    // Valid doc: bonus 40 for not inventing false violations
    if (violations.length === 0) score += 40;
  }

  return { score: Math.round(score), reasons };
}

async function runFixture(model, fixtureFile) {
  const spec = JSON.parse(readFileSync(join(fixturesDir, fixtureFile), 'utf8'));
  const messages = spec.messages || [
    { role: 'system', content: spec.system || '' },
    { role: 'user', content: spec.user || spec.prompt || '' },
  ];
  // Apply per-model user-message suffix (e.g. /no_think for Qwen3 family)
  if (model.userSuffix && messages.length) {
    const last = messages[messages.length - 1];
    if (last.role === 'user') last.content = `${last.content}\n${model.userSuffix}`;
  }
  console.error(`  → ${fixtureFile} (${spec.type})`);
  const response = await callModel(model, messages, spec.opts || {});

  let scoring;
  if (spec.type === 'parse') scoring = scoreParseFixture(spec, response);
  else if (spec.type === 'produce') scoring = scoreProduceFixture(spec, response);
  else if (spec.type === 'ctd') scoring = scoreCTDFixture(spec, response);
  else scoring = { score: response.ok ? 100 : 0, reasons: response.ok ? ['smoke pass'] : [response.error] };

  // Dump full raw response to disk
  const dumpName = `${slug(model.id)}__${slug(spec.id)}`;
  const dumpPath = join(runDir, `${dumpName}.md`);
  const dump = [
    `# ${model.id} — ${spec.id} (${spec.type})`,
    ``,
    `**score:** ${scoring.score}/100`,
    `**reasons:** ${scoring.reasons.join('; ') || '(none)'}`,
    `**elapsedMs:** ${response.elapsedMs}  **tok/s:** ${response.tokPerSec ?? '-'}`,
    `**usage:** ${JSON.stringify(response.usage)}`,
    ``,
    `## Prompt`,
    '```',
    JSON.stringify(messages, null, 2),
    '```',
    ``,
    `## Raw response`,
    '```',
    response.content ?? `(API error: ${response.error})`,
    '```',
    ``,
    `## Stripped (thinking removed)`,
    '```',
    response.stripped ?? '',
    '```',
  ].join('\n');
  writeFileSync(dumpPath, dump);

  return {
    model: model.id,
    fixture: spec.id,
    type: spec.type,
    score: scoring.score,
    reasons: scoring.reasons,
    elapsedMs: response.elapsedMs,
    tokPerSec: response.tokPerSec,
    usage: response.usage,
    dumpFile: `${dumpName}.md`,
    ok: response.ok,
  };
}

async function main() {
  console.error(`\n=== SPIL LM Studio Test Run ${runId} ===`);
  console.error(`Endpoint: ${config.endpoint}`);
  console.error(`Models: ${models.map(m => m.id).join(', ')}`);
  console.error(`Fixtures: ${fixtureFiles.join(', ')}\n`);

  for (const model of models) {
    console.error(`\n[${model.id}] (${model.class}, ~${model.approxRamGb}GB)`);
    for (const fixtureFile of fixtureFiles) {
      const r = await runFixture(model, fixtureFile);
      results.push(r);
      console.error(`    score=${r.score}/100  ${r.elapsedMs}ms  ${r.tokPerSec ?? '-'}t/s  ${r.reasons[0] ?? ''}`);
    }
  }

  writeFileSync(resultPath, JSON.stringify(results, null, 2));
  console.error(`\nResults written: ${resultPath}`);

  // Quick matrix
  console.error('\n=== Matrix ===');
  const byModel = {};
  for (const r of results) {
    (byModel[r.model] ||= []).push(r);
  }
  for (const [m, rs] of Object.entries(byModel)) {
    const avg = (rs.reduce((s, r) => s + r.score, 0) / rs.length).toFixed(1);
    const tps = rs.filter(r => r.tokPerSec).map(r => +r.tokPerSec);
    const avgTps = tps.length ? (tps.reduce((a,b)=>a+b,0)/tps.length).toFixed(1) : '-';
    console.error(`  ${m.padEnd(40)} avg=${avg}  tok/s=${avgTps}`);
  }
}

main().catch(e => { console.error(e); process.exit(1); });
