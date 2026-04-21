import { describe, expect, it } from 'vitest';
import { getLegacyRepoCacheDir } from '../../src/server/legacy-repo-cache.js';

describe('legacy-repo-cache', () => {
  it('returns the legacy cache path under ~/.avmatrix/repos', () => {
    const dir = getLegacyRepoCacheDir('my-repo');
    expect(dir).toContain('.avmatrix');
    expect(dir).toMatch(/repos/);
    expect(dir).toContain('my-repo');
  });
});
