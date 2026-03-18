import { defineConfig } from '@nomik/core';

export default defineConfig({
  target: {
    root: './src',
    include: ['**/*.ts', '**/*.tsx', '**/*.js', '**/*.jsx', '**/*.md', '**/*.py', '**/*.rs'],
    exclude: ['**/node_modules/**', '**/dist/**', '**/*.test.*', '**/*.spec.*', '**/*.d.ts', '**/__pycache__/**', '**/target/**', '**/.venv/**'],
  },
  // Graph connection reads from .env (NOMIK_GRAPH_URI, NOMIK_GRAPH_USER, NOMIK_GRAPH_PASS)
  parser: {
    languages: ['typescript', 'python', 'rust'],
  },
});
