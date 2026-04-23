/**
 * Clean Command
 *
 * Removes the .avmatrix index from the current repository.
 * Also unregisters it from the global registry.
 */

import fs from 'fs/promises';
import { findRepo, unregisterRepo, listRegisteredRepos } from '../storage/repo-manager.js';
import { getSettingsPath } from '../storage/settings.js';

export const cleanStoragePreservingSettings = async (
  repoPath: string,
  storagePath: string,
): Promise<void> => {
  const settingsPath = getSettingsPath(repoPath);
  let preservedSettings: string | null = null;

  try {
    preservedSettings = await fs.readFile(settingsPath, 'utf-8');
  } catch {
    preservedSettings = null;
  }

  await fs.rm(storagePath, { recursive: true, force: true });

  if (preservedSettings !== null) {
    await fs.mkdir(storagePath, { recursive: true });
    await fs.writeFile(settingsPath, preservedSettings, 'utf-8');
  }
};

export const cleanCommand = async (options?: { force?: boolean; all?: boolean }) => {
  // --all flag: clean all indexed repos
  if (options?.all) {
    if (!options?.force) {
      const entries = await listRegisteredRepos();
      if (entries.length === 0) {
        console.log('No indexed repositories found.');
        return;
      }
      console.log(`This will delete AVmatrix indexes for ${entries.length} repo(s):`);
      for (const entry of entries) {
        console.log(`  - ${entry.name} (${entry.path})`);
      }
      console.log('\nRun with --force to confirm deletion.');
      return;
    }

    const entries = await listRegisteredRepos();
    for (const entry of entries) {
      try {
        await cleanStoragePreservingSettings(entry.path, entry.storagePath);
        await unregisterRepo(entry.path);
        console.log(`Deleted: ${entry.name} (${entry.storagePath})`);
      } catch (err) {
        console.error(`Failed to delete ${entry.name}:`, err);
      }
    }
    return;
  }

  // Default: clean current repo
  const cwd = process.cwd();
  const repo = await findRepo(cwd);

  if (!repo) {
    console.log('No indexed repository found in this directory.');
    return;
  }

  const repoName = repo.repoPath.split(/[/\\]/).pop() || repo.repoPath;

  if (!options?.force) {
    console.log(`This will delete the AVmatrix index for: ${repoName}`);
    console.log(`   Path: ${repo.storagePath}`);
    console.log('\nRun with --force to confirm deletion.');
    return;
  }

  try {
    await cleanStoragePreservingSettings(repo.repoPath, repo.storagePath);
    await unregisterRepo(repo.repoPath);
    console.log(`Deleted: ${repo.storagePath}`);
  } catch (err) {
    console.error('Failed to delete:', err);
  }
};
