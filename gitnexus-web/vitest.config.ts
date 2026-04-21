import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import fs from 'fs';
import path from 'path';
import { createRequire } from 'module';

const _require = createRequire(import.meta.url);

function findSiblingPackageDir(packageName: string): string {
  const monorepoRoot = path.resolve(__dirname, '..');

  for (const entry of fs.readdirSync(monorepoRoot, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;

    const packageJsonPath = path.join(monorepoRoot, entry.name, 'package.json');
    if (!fs.existsSync(packageJsonPath)) continue;

    try {
      const pkg = _require(packageJsonPath);
      if (pkg.name === packageName) {
        return path.join(monorepoRoot, entry.name);
      }
    } catch {
      // Ignore malformed or unrelated package manifests.
    }
  }

  throw new Error(`Could not find sibling package '${packageName}' from ${__dirname}`);
}

const CLI_ROOT = findSiblingPackageDir('avmatrix');
const SHARED_ROOT = findSiblingPackageDir('avmatrix-shared');
const avmatrixPkg = _require(path.join(CLI_ROOT, 'package.json'));

export default defineConfig({
  plugins: [react()],
  define: {
    __REQUIRED_NODE_VERSION__: JSON.stringify(avmatrixPkg.engines.node.replace(/[>=^~\s]/g, '')),
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      'avmatrix-shared': path.join(SHARED_ROOT, 'src', 'index.ts'),
      '@anthropic-ai/sdk/lib/transform-json-schema': path.resolve(
        __dirname,
        'node_modules/@anthropic-ai/sdk/lib/transform-json-schema.mjs',
      ),
      mermaid: path.resolve(__dirname, 'node_modules/mermaid/dist/mermaid.esm.min.mjs'),
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./test/setup.ts'],
    include: ['test/**/*.test.{ts,tsx}'],
    testTimeout: 15000,
    coverage: {
      provider: 'v8',
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/workers/**', // Web workers (require worker env)
        'src/core/lbug/**', // WASM (requires SharedArrayBuffer)
        'src/core/tree-sitter/**', // WASM (requires tree-sitter binaries)
        'src/core/embeddings/**', // WASM (requires ML model)
        'src/main.tsx', // Entry point
        'src/vite-env.d.ts', // Type declarations
      ],
      thresholds: {
        statements: 10,
        branches: 10,
        functions: 10,
        lines: 10,
      },
    },
  },
});
