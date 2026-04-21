import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(testDir, '../..');

describe('package bin aliases', () => {
  it('exposes both gitnexus and avmatrix during the transition', async () => {
    const raw = await fs.readFile(path.join(repoRoot, 'package.json'), 'utf8');
    const pkg = JSON.parse(raw);

    expect(pkg.bin.gitnexus).toBe('dist/cli/index.js');
    expect(pkg.bin.avmatrix).toBe('dist/cli/index.js');
  });
});
