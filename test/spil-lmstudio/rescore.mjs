#!/usr/bin/env node
// Re-score an existing results directory without re-running models.
// Usage: node rescore.mjs <runDir>

import { readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parseSPIL, stripThinking, extractJSON } from './spil-parser.mjs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const runDir = process.argv[2];
if (!runDir) { console.error('usage: node rescore.mjs <runDir>'); process.exit(1); }

const fixturesDir = join(__dirname, 'fixtures');

function normalize(s) { return String(s).toLowerCase().replace(/\s+/g, ' ').trim(); }
function countAllDirectives(doc) {
  return (doc.directives?.length || 0) + (doc.sections || []).reduce((n, s) => n + (s.directives?.length || 0), 0);
}

function scoreParseFixture(spec, content) {
  const stripped = stripThinking(content);
  const parsed = extractJSON(stripped);
  if (!parsed) return { score: 0, reasons: ['No JSON object extractable'] };
  const expected = spec.expected;
  let score = 0;
  const reasons = [];
  const expectedKeywords = (expected.sections || []).map(s => s.keyword).sort();
  const gotKeywords = (parsed.sections || []).map(s => s.keyword).sort();
  const kwInter = expectedKeywords.filter(k => gotKeywords.includes(k));
  const kwScore = expectedKeywords.length ? 40 * (kwInter.length / expectedKeywords.length) : 40;
  score += kwScore;
  if (kwScore < 40) reasons.push(`section keywords: ${kwInter.length}/${expectedKeywords.length} (missing: ${expectedKeywords.filter(k => !gotKeywords.includes(k)).join(',')})`);
  const expSC = (expected.sections || []).find(s => s.keyword === 'success_criteria');
  const gotSC = (parsed.sections || []).find(s => s.keyword === 'success_criteria');
  if (expSC) {
    if (!gotSC) reasons.push('missing @success_criteria');
    else {
      const expItems = expSC.items || [];
      const gotItems = gotSC.items || [];
      const countScore = 15 * Math.min(1, gotItems.length / Math.max(1, expItems.length));
      const overlapScore = 25 * (gotItems.filter(g => expItems.some(e => normalize(e) === normalize(g))).length / Math.max(1, expItems.length));
      score += countScore + overlapScore;
      if (countScore + overlapScore < 40) reasons.push(`success_criteria: matched ${gotItems.filter(g => expItems.some(e => normalize(e) === normalize(g))).length}/${expItems.length}`);
    }
  } else score += 40;
  const expDir = countAllDirectives(expected);
  const gotDir = countAllDirectives(parsed);
  const dirScore = expDir === 0 ? (gotDir === 0 ? 20 : 10) : 20 * Math.min(1, gotDir / expDir);
  score += dirScore;
  if (dirScore < 20) reasons.push(`directives: got ${gotDir}, expected ${expDir}`);
  return { score: Math.round(score), reasons };
}

function scoreProduceFixture(spec, content) {
  const text = stripThinking(content);
  const doc = parseSPIL(text);
  let score = 0;
  const reasons = [];
  if (doc.sections.length === 0) { reasons.push('no @sections — not SPIL'); return { score: 0, reasons }; }
  const required = spec.expected.requiredKeywords || [];
  const found = required.filter(k => doc.sections.some(s => s.keyword === k));
  score += required.length ? 50 * (found.length / required.length) : 50;
  if (found.length < required.length) reasons.push(`missing required: ${required.filter(r => !found.includes(r)).join(',')}`);
  const minSC = spec.expected.minSuccessCriteria || 0;
  const sc = doc.sections.find(s => s.keyword === 'success_criteria');
  const scCount = sc?.items.length || 0;
  if (scCount >= minSC) score += 30;
  else { score += 30 * (scCount / Math.max(1, minSC)); reasons.push(`success_criteria: ${scCount} < ${minSC}`); }
  const forbidden = spec.expected.forbiddenStrings || [];
  const violations = forbidden.filter(s => {
    const esc = s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const isWordLike = /^\w/.test(s) && /\w$/.test(s);
    const re = new RegExp(isWordLike ? `\\b${esc}\\b` : esc, 'i');
    return re.test(text);
  });
  if (violations.length === 0) score += 20;
  else reasons.push(`forbidden strings found: ${violations.join(',')}`);
  return { score: Math.round(score), reasons };
}

const summary = JSON.parse(readFileSync(join(runDir, 'summary.json'), 'utf8'));
const updated = [];

for (const row of summary) {
  const dumpPath = join(runDir, row.dumpFile);
  if (!row.ok) { updated.push(row); continue; }
  const dump = readFileSync(dumpPath, 'utf8');
  // Extract Raw response section
  const rawMatch = dump.match(/## Raw response\s*\n```\s*\n([\s\S]*?)\n```\s*\n+## Stripped/);
  const content = rawMatch ? rawMatch[1] : '';
  const spec = JSON.parse(readFileSync(join(fixturesDir, `${row.fixture}.spec.json`), 'utf8'));
  const scoring = spec.type === 'parse'
    ? scoreParseFixture(spec, content)
    : spec.type === 'produce'
    ? scoreProduceFixture(spec, content)
    : { score: row.score, reasons: row.reasons };
  updated.push({ ...row, score: scoring.score, reasons: scoring.reasons, rescored: true });
  console.log(`${row.model.padEnd(40)} ${row.fixture.padEnd(12)} ${row.score} → ${scoring.score}  ${scoring.reasons.join('; ')}`);
}

writeFileSync(join(runDir, 'summary.rescored.json'), JSON.stringify(updated, null, 2));

const byModel = {};
for (const r of updated) (byModel[r.model] ||= []).push(r);
console.log('\n=== Matrix (rescored) ===');
for (const [m, rs] of Object.entries(byModel)) {
  const avg = (rs.reduce((s, r) => s + r.score, 0) / rs.length).toFixed(1);
  console.log(`  ${m.padEnd(40)} avg=${avg}`);
}
