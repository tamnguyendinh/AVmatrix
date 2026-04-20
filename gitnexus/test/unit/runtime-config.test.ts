import fs from 'fs/promises';
import os from 'os';
import path from 'path';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import {
  getRuntimeConfigPath,
  loadRuntimeConfig,
  saveRuntimeConfig,
} from '../../src/storage/runtime-config.js';

describe('runtime-config', () => {
  let tempHome: string;
  const previousHome = process.env.GITNEXUS_HOME;

  beforeEach(async () => {
    tempHome = await fs.mkdtemp(path.join(os.tmpdir(), 'gitnexus-runtime-config-'));
    process.env.GITNEXUS_HOME = tempHome;
  });

  afterEach(async () => {
    if (previousHome === undefined) {
      delete process.env.GITNEXUS_HOME;
    } else {
      process.env.GITNEXUS_HOME = previousHome;
    }
    await fs.rm(tempHome, { recursive: true, force: true });
  });

  it('defaults wiki mode to off when no runtime config exists', async () => {
    await expect(loadRuntimeConfig()).resolves.toEqual({ wikiMode: 'off' });
  });

  it('persists runtime config to runtime.json', async () => {
    await saveRuntimeConfig({ wikiMode: 'local' });

    await expect(loadRuntimeConfig()).resolves.toEqual({ wikiMode: 'local' });
    const raw = await fs.readFile(getRuntimeConfigPath(), 'utf8');
    expect(JSON.parse(raw)).toEqual({ wikiMode: 'local' });
  });
});
