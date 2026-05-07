import { describe, expect, it } from 'vitest';
import { getCompatibilityRepoCacheDir } from '../../src/server/compatibility-repo-cache.js';

describe('compatibility-repo-cache', () => {
  it('returns the compatibility cache path under ~/.avmatrix/repos', () => {
    const dir = getCompatibilityRepoCacheDir('my-repo');
    expect(dir).toContain('.avmatrix');
    expect(dir).toMatch(/repos/);
    expect(dir).toContain('my-repo');
  });
});
