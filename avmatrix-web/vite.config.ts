import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
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
const avmatrixCliPkg = _require(path.join(CLI_ROOT, 'package.json'));

export default defineConfig({
  plugins: [react(), tailwindcss()],
  define: {
    __REQUIRED_NODE_VERSION__: JSON.stringify(avmatrixCliPkg.engines.node.replace(/[>=^~\s]/g, '')),
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      'avmatrix-shared': path.join(SHARED_ROOT, 'src', 'index.ts'),
      // Fix for Rollup failing to resolve this deep import from @langchain/anthropic
      '@anthropic-ai/sdk/lib/transform-json-schema': path.resolve(
        __dirname,
        'node_modules/@anthropic-ai/sdk/lib/transform-json-schema.mjs',
      ),
      // Fix for mermaid d3-color prototype crash on Vercel (known issue with mermaid 10.9.0+ and Vite)
      mermaid: path.resolve(__dirname, 'node_modules/mermaid/dist/mermaid.esm.min.mjs'),
    },
  },
  server: {
    // Allow serving files from node_modules
    fs: {
      allow: ['..'],
    },
  },
});
