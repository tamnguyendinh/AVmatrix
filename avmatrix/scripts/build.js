#!/usr/bin/env node
/**
 * Build script that compiles the AVmatrix CLI and inlines avmatrix-shared into dist.
 *
 * Steps:
 *  1. Build avmatrix-shared (tsc)
 *  2. Build AVmatrix CLI (tsc)
 *  3. Copy avmatrix-shared/dist → dist/_shared
 *  4. Rewrite bare 'avmatrix-shared' specifiers → relative paths
 */
import { execSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '..');
const DIST = path.join(ROOT, 'dist');
const SHARED_DEST = path.join(DIST, '_shared');

function findSiblingPackageDir(packageName) {
  const monorepoRoot = path.resolve(ROOT, '..');

  for (const entry of fs.readdirSync(monorepoRoot, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;

    const packageJsonPath = path.join(monorepoRoot, entry.name, 'package.json');
    if (!fs.existsSync(packageJsonPath)) continue;

    try {
      const pkg = JSON.parse(fs.readFileSync(packageJsonPath, 'utf-8'));
      if (pkg.name === packageName) {
        return path.join(monorepoRoot, entry.name);
      }
    } catch {
      // Ignore non-package directories.
    }
  }

  throw new Error(`Could not find sibling package '${packageName}' from ${ROOT}`);
}

const SHARED_ROOT = findSiblingPackageDir('avmatrix-shared');

// ── 1. Build avmatrix-shared ───────────────────────────────────────
console.log('[build] compiling avmatrix-shared…');
execSync('npx tsc', { cwd: SHARED_ROOT, stdio: 'inherit' });

// ── 2. Build AVmatrix CLI ──────────────────────────────────────────
console.log('[build] compiling AVmatrix CLI…');
execSync('npx tsc', { cwd: ROOT, stdio: 'inherit' });

// ── 3. Copy shared dist ────────────────────────────────────────────
console.log('[build] copying shared module into dist/_shared…');
fs.cpSync(path.join(SHARED_ROOT, 'dist'), SHARED_DEST, { recursive: true });

// ── 4. Rewrite imports ─────────────────────────────────────────────
console.log('[build] rewriting avmatrix-shared imports…');
let rewritten = 0;

function rewriteFile(filePath) {
  const content = fs.readFileSync(filePath, 'utf-8');
  if (!content.includes('avmatrix-shared')) return;

  const relDir = path.relative(path.dirname(filePath), SHARED_DEST);
  // Always use posix separators and point to the package index
  const relImport = relDir.split(path.sep).join('/') + '/index.js';

  const updated = content
    .replace(/from\s+['"]avmatrix-shared['"]/g, `from '${relImport}'`)
    .replace(/import\(\s*['"]avmatrix-shared['"]\s*\)/g, `import('${relImport}')`);

  if (updated !== content) {
    fs.writeFileSync(filePath, updated);
    rewritten++;
  }
}

function walk(dir, extensions, cb) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walk(full, extensions, cb);
    } else if (extensions.some((ext) => entry.name.endsWith(ext))) {
      cb(full);
    }
  }
}

walk(DIST, ['.js', '.d.ts'], rewriteFile);

// ── 5. Make CLI entry executable ────────────────────────────────────
const cliEntry = path.join(DIST, 'cli', 'index.js');
if (fs.existsSync(cliEntry)) fs.chmodSync(cliEntry, 0o755);

console.log(`[build] done — rewrote ${rewritten} files.`);
