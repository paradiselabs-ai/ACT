import { defineConfig } from '@nomik/core';

export default defineConfig({
  target: {
    root: '.',
    include: [
      'act-agent/**/*.go',
      'server/src/**/*.ts',
      'act-agent/cli/**/*.ts',
      'act-agent/runner/**/*.mjs',
    ],
    exclude: [
      '**/node_modules/**',
      '**/dist/**',
      '**/*_test.go',
      '**/*.test.*',
      '**/*.spec.*',
      '**/*.d.ts',
      'act-agent/act-agent',
    ],
  },
  // Graph connection reads from .env (NOMIK_GRAPH_URI, NOMIK_GRAPH_USER, NOMIK_GRAPH_PASS)
  parser: {
    languages: ['go', 'typescript', 'javascript'],
  },
});
