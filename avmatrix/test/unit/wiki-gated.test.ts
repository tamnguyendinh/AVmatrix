import fs from 'fs/promises';
import os from 'os';
import path from 'path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  formatWikiModeStatus,
  wikiGatedCommand,
  wikiModeCommand,
} from '../../src/cli/wiki-gated.js';
import { loadRuntimeConfig } from '../../src/storage/runtime-config.js';

describe('wiki capability gate', () => {
  let tempHome: string;
  const previousHome = process.env.AVMATRIX_HOME;
  const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
  const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

  beforeEach(async () => {
    tempHome = await fs.mkdtemp(path.join(os.tmpdir(), 'avmatrix-wiki-gate-'));
    process.env.AVMATRIX_HOME = tempHome;
    process.exitCode = undefined;
    logSpy.mockClear();
    errorSpy.mockClear();
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

  it('blocks wiki execution when wiki mode is off', async () => {
    await wikiGatedCommand();

    expect(process.exitCode).toBe(1);
    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('Wiki capability mode: off'));
    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('disabled in local-only mode'));
  });

  it('stores wiki mode as local through the CLI toggle command', async () => {
    await wikiModeCommand('local');

    await expect(loadRuntimeConfig()).resolves.toEqual({ wikiMode: 'local' });
    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('Wiki capability mode: local'));
    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('not available yet in this build'));
  });

  it('rejects invalid wiki mode values', async () => {
    await wikiModeCommand('remote');

    expect(process.exitCode).toBe(1);
    expect(errorSpy).toHaveBeenCalledWith(expect.stringContaining('Invalid wiki mode'));
  });

  it('formats local mode as fail-safe without remote fallback', () => {
    expect(formatWikiModeStatus('local')).toContain('will not fall back to any remote wiki service');
  });
});
