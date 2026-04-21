import fs from 'fs/promises';
import os from 'os';
import path from 'path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { wikiCommand } from '../../src/cli/wiki.js';

describe('wiki compatibility wrapper', () => {
  let tempHome: string;
  const previousHome = process.env.AVMATRIX_HOME;
  const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});

  beforeEach(async () => {
    tempHome = await fs.mkdtemp(path.join(os.tmpdir(), 'gitnexus-wiki-compat-'));
    process.env.AVMATRIX_HOME = tempHome;
    process.exitCode = undefined;
    logSpy.mockClear();
  });

  afterEach(async () => {
    if (previousHome === undefined) {
      delete process.env.AVMATRIX_HOME;
    } else {
      process.env.AVMATRIX_HOME = previousHome;
    }
    process.exitCode = undefined;
    await fs.rm(tempHome, { recursive: true, force: true });
  });

  it('routes the legacy wiki import path to the local-only capability gate', async () => {
    await wikiCommand();

    expect(process.exitCode).toBe(1);
    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('Wiki capability mode: off'));
    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('disabled in local-only mode'));
  });
});
