#!/usr/bin/env tsx
/**
 * One-shot migration: read the existing global coordination-log.jsonl and
 * replay every event into per-project files. Non-destructive — the source
 * file is copied to <source>.pre-split.bak BEFORE any writes.
 *
 * Idempotent: running twice on the same source produces the same per-project
 * files (we append, but events have unique timestamps so no duplication is
 * possible across re-runs IF the project files don't already exist; if they
 * do exist, we skip and tell the user to wipe ./projects/ first).
 */
import * as fs from 'fs/promises';
import * as path from 'path';
import { fileURLToPath } from 'url';
import { extractProjectName } from '../src/services/PVMIndexer';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const SOURCE = path.resolve(__dirname, '../data/coordination-log.jsonl');
const BACKUP = SOURCE + '.pre-split.bak';
const PROJECTS_DIR = path.join(path.dirname(SOURCE), 'projects');

function safeProjectName(name: string): string {
  return name.replace(/[^A-Za-z0-9_-]/g, '_');
}

async function main() {
  // Refuse to run if projects dir already has content — indicates partial run
  try {
    const existing = await fs.readdir(PROJECTS_DIR);
    if (existing.length > 0) {
      console.error(`[migrate] ${PROJECTS_DIR} already has content. Wipe it (rm -rf) and re-run.`);
      process.exit(1);
    }
  } catch (err: any) {
    if (err.code !== 'ENOENT') throw err;
  }

  // Snapshot source for rollback
  console.log(`[migrate] copying ${SOURCE} -> ${BACKUP}`);
  await fs.copyFile(SOURCE, BACKUP);

  // Stream-read the JSONL (could be large — read line by line)
  console.log(`[migrate] reading events from ${SOURCE}`);
  const raw = await fs.readFile(SOURCE, 'utf-8');
  const lines = raw.split('\n').filter(l => l.trim().length > 0);
  console.log(`[migrate] ${lines.length} events to migrate`);

  // Group by project
  const groups = new Map<string, string[]>();
  let badLines = 0;
  for (const line of lines) {
    try {
      const event = JSON.parse(line);
      const project = safeProjectName(extractProjectName(event));
      if (!groups.has(project)) groups.set(project, []);
      groups.get(project)!.push(line);
    } catch {
      badLines++;
    }
  }

  // Write per-project files
  for (const [project, projLines] of groups.entries()) {
    const dir = path.join(PROJECTS_DIR, project);
    await fs.mkdir(dir, { recursive: true });
    const filePath = path.join(dir, 'coordination-log.jsonl');
    await fs.writeFile(filePath, projLines.join('\n') + '\n', 'utf-8');
    console.log(`[migrate] ${project}: ${projLines.length} events -> ${filePath}`);
  }

  console.log(`[migrate] done. ${groups.size} project(s), ${lines.length - badLines} events. ${badLines} unparsable lines.`);
}

main().catch(err => {
  console.error('[migrate] FATAL:', err);
  process.exit(2);
});
