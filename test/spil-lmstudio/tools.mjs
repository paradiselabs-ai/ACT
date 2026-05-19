// Sandboxed tool implementations for SPIL tool-use fixtures.
// All paths are resolved relative to sandboxRoot. Attempts to escape return an error.

import { resolve, relative, dirname, isAbsolute, join } from 'node:path';
import { readFileSync, writeFileSync, readdirSync, mkdirSync, existsSync, rmSync, statSync } from 'node:fs';

export const TOOL_SCHEMAS = [
  {
    type: 'function',
    function: {
      name: 'read_file',
      description: 'Read the contents of a file in the sandbox. Path is relative to the sandbox root.',
      parameters: {
        type: 'object',
        properties: { path: { type: 'string', description: 'Relative path inside sandbox' } },
        required: ['path'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'write_file',
      description: 'Write content to a file in the sandbox. Creates parent dirs. Overwrites existing files.',
      parameters: {
        type: 'object',
        properties: {
          path: { type: 'string', description: 'Relative path inside sandbox' },
          content: { type: 'string', description: 'Full file content' },
        },
        required: ['path', 'content'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'list_dir',
      description: 'List directory contents in the sandbox.',
      parameters: {
        type: 'object',
        properties: { path: { type: 'string', description: 'Relative path inside sandbox (use "." for root)' } },
        required: ['path'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'mark_complete',
      description: 'Signal that the task is complete. Provide a short summary of what was done. The grading happens after this is called.',
      parameters: {
        type: 'object',
        properties: { summary: { type: 'string', description: 'Brief summary of work done' } },
        required: ['summary'],
      },
    },
  },
];

// Schemas for lazy-fetch arms: base tools + spil_get for on-demand section retrieval.
export const LAZY_TOOL_SCHEMAS = [
  ...TOOL_SCHEMAS,
  {
    type: 'function',
    function: {
      name: 'spil_get',
      description: 'Fetch the body of one named SPIL section from the task spec. The section name is what appears after @ in the manifest (e.g. "endpoints", "data_models"). Returns the section content as text. Call this whenever you need a section you have not yet loaded — do not guess or improvise around missing sections.',
      parameters: {
        type: 'object',
        properties: { section: { type: 'string', description: 'SPIL section name without the @ prefix' } },
        required: ['section'],
      },
    },
  },
];

function safePath(sandboxRoot, p) {
  if (typeof p !== 'string') throw new Error('path must be a string');
  if (isAbsolute(p)) throw new Error(`absolute path forbidden: ${p}`);
  const abs = resolve(sandboxRoot, p);
  const rel = relative(sandboxRoot, abs);
  if (rel.startsWith('..') || isAbsolute(rel)) throw new Error(`path escapes sandbox: ${p}`);
  return abs;
}

export function makeToolExecutor(sandboxRoot, opts = {}) {
  const maxFileBytes = opts.maxFileBytes ?? 65536;
  const spilSections = opts.spilSections ?? null; // Map<name, body> or null when not in lazy mode
  let completeCalled = false;
  let completeSummary = null;
  const log = [];
  const fetches = []; // ordered list of section names fetched (lazy arms only)

  function execute(name, args) {
    const t0 = Date.now();
    try {
      const result = run(name, args);
      log.push({ name, args, result: typeof result === 'string' ? result.slice(0, 1000) : result, ok: true, ms: Date.now() - t0 });
      return result;
    } catch (e) {
      const err = `Error: ${e.message}`;
      log.push({ name, args, result: err, ok: false, ms: Date.now() - t0 });
      return err;
    }
  }

  function run(name, args) {
    switch (name) {
      case 'read_file': {
        const p = safePath(sandboxRoot, args.path);
        if (!existsSync(p)) return `Error: file not found: ${args.path}`;
        const buf = readFileSync(p);
        if (buf.length > maxFileBytes) return `Error: file too large (${buf.length} > ${maxFileBytes})`;
        return buf.toString('utf8');
      }
      case 'write_file': {
        const p = safePath(sandboxRoot, args.path);
        const content = String(args.content ?? '');
        if (content.length > maxFileBytes) throw new Error(`content too large (${content.length} > ${maxFileBytes})`);
        mkdirSync(dirname(p), { recursive: true });
        writeFileSync(p, content);
        return `OK: wrote ${content.length} bytes to ${args.path}`;
      }
      case 'list_dir': {
        const p = safePath(sandboxRoot, args.path);
        if (!existsSync(p)) return `Error: not found: ${args.path}`;
        const entries = readdirSync(p).map(e => {
          const st = statSync(join(p, e));
          return st.isDirectory() ? `${e}/` : e;
        });
        return entries.length ? entries.join('\n') : '(empty)';
      }
      case 'mark_complete': {
        completeCalled = true;
        completeSummary = String(args.summary ?? '');
        return 'OK: marked complete. Grading will proceed.';
      }
      case 'spil_get': {
        if (!spilSections) return 'Error: spil_get not available in this run (no SPIL source configured)';
        const raw = String(args.section ?? '').trim();
        const key = raw.startsWith('@') ? raw.slice(1) : raw;
        const body = spilSections.get(key);
        if (body == null) {
          const available = [...spilSections.keys()].join(', ');
          return `Error: unknown section "${raw}". Available sections: ${available}`;
        }
        fetches.push(key);
        return body;
      }
      default:
        return `Error: unknown tool ${name}`;
    }
  }

  return {
    execute,
    state: () => ({ completeCalled, completeSummary, log: [...log], fetches: [...fetches] }),
  };
}

export function resetSandbox(sandboxRoot) {
  if (existsSync(sandboxRoot)) rmSync(sandboxRoot, { recursive: true, force: true });
  mkdirSync(sandboxRoot, { recursive: true });
}

export function seedSandbox(sandboxRoot, files = {}) {
  for (const [relPath, content] of Object.entries(files)) {
    const abs = safePath(sandboxRoot, relPath);
    mkdirSync(dirname(abs), { recursive: true });
    writeFileSync(abs, content);
  }
}

export function readSandboxFile(sandboxRoot, relPath) {
  const abs = safePath(sandboxRoot, relPath);
  if (!existsSync(abs)) return null;
  return readFileSync(abs, 'utf8');
}
