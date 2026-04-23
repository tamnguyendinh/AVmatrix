import fs from 'fs/promises';
import path from 'path';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import {
  DEFAULT_MAX_EXECUTION_FLOWS,
  getSettingsPath,
  loadSettings,
  saveSettings,
} from '../../src/storage/settings.js';
import { createTempDir } from '../helpers/test-db.js';

describe('settings', () => {
  let tempRepo: Awaited<ReturnType<typeof createTempDir>>;

  beforeEach(async () => {
    tempRepo = await createTempDir('avmatrix-settings-repo-');
  });

  afterEach(async () => {
    await tempRepo.cleanup();
  });

  it('creates repo-local settings.json with default maxExecutionFlows when missing', async () => {
    await expect(loadSettings(tempRepo.dbPath)).resolves.toEqual({
      maxExecutionFlows: DEFAULT_MAX_EXECUTION_FLOWS,
    });

    const raw = await fs.readFile(getSettingsPath(tempRepo.dbPath), 'utf8');
    expect(JSON.parse(raw)).toEqual({
      maxExecutionFlows: DEFAULT_MAX_EXECUTION_FLOWS,
    });
    expect(path.dirname(getSettingsPath(tempRepo.dbPath))).toBe(path.join(tempRepo.dbPath, '.avmatrix'));
  });

  it('persists maxExecutionFlows to repo-local settings.json', async () => {
    await saveSettings(tempRepo.dbPath, { maxExecutionFlows: 900 });

    await expect(loadSettings(tempRepo.dbPath)).resolves.toEqual({ maxExecutionFlows: 900 });
    const raw = await fs.readFile(getSettingsPath(tempRepo.dbPath), 'utf8');
    expect(JSON.parse(raw)).toEqual({ maxExecutionFlows: 900 });
  });
});
