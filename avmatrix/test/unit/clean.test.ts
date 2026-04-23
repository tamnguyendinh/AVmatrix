/// <reference types="node" />
/// <reference types="vitest/globals" />

import fs from 'fs/promises';
import path from 'path';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { cleanStoragePreservingSettings } from '../../src/cli/clean.js';
import { getSettingsPath } from '../../src/storage/settings.js';
import { createTempDir } from '../helpers/test-db.js';

describe('cleanStoragePreservingSettings', () => {
  let tempRepo: Awaited<ReturnType<typeof createTempDir>>;

  beforeEach(async () => {
    tempRepo = await createTempDir('avmatrix-clean-repo-');
    await fs.mkdir(path.join(tempRepo.dbPath, '.avmatrix'), { recursive: true });
  });

  afterEach(async () => {
    await tempRepo.cleanup();
  });

  it('preserves repo-local settings.json while removing index files', async () => {
    const storagePath = path.join(tempRepo.dbPath, '.avmatrix');
    const settingsPath = getSettingsPath(tempRepo.dbPath);
    const metaPath = path.join(storagePath, 'meta.json');

    await fs.writeFile(settingsPath, JSON.stringify({ maxExecutionFlows: 333 }, null, 2), 'utf-8');
    await fs.writeFile(metaPath, '{"stats":{"processes":333}}', 'utf-8');

    await cleanStoragePreservingSettings(tempRepo.dbPath, storagePath);

    await expect(fs.readFile(settingsPath, 'utf-8')).resolves.toContain('"maxExecutionFlows": 333');
    await expect(fs.access(metaPath)).rejects.toBeDefined();
  });
});
