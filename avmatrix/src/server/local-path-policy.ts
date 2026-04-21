import fs from 'fs/promises';
import path from 'path';

const isLikelyRemoteUrl = (value: string): boolean =>
  /^[a-z][a-z0-9+.-]*:\/\//i.test(value) || /^git@/i.test(value);

const isUncPath = (value: string): boolean => value.startsWith('\\\\') || value.startsWith('//');

export async function resolveAnalyzeRepoPath(repoPath: string): Promise<string> {
  if (isLikelyRemoteUrl(repoPath)) {
    throw new Error('"path" must be a local filesystem path');
  }

  if (isUncPath(repoPath)) {
    throw new Error('UNC and network-share paths are not allowed');
  }

  if (!path.isAbsolute(repoPath)) {
    throw new Error('"path" must be an absolute path');
  }

  let realRepoPath: string;
  try {
    realRepoPath = await fs.realpath(repoPath);
  } catch (error) {
    const code =
      typeof error === 'object' && error && 'code' in error
        ? (error as NodeJS.ErrnoException).code
        : undefined;
    if (code === 'ENOENT') {
      throw new Error(`Repository path "${repoPath}" does not exist`);
    }
    throw new Error(`Failed to resolve repository path "${repoPath}"`);
  }

  let stat: Awaited<ReturnType<typeof fs.stat>>;
  try {
    stat = await fs.stat(realRepoPath);
  } catch (error) {
    const code =
      typeof error === 'object' && error && 'code' in error
        ? (error as NodeJS.ErrnoException).code
        : undefined;
    if (code === 'ENOENT') {
      throw new Error(`Repository path "${repoPath}" does not exist`);
    }
    throw new Error(`Failed to inspect repository path "${repoPath}"`);
  }

  if (!stat.isDirectory()) {
    throw new Error(`"${repoPath}" is not a directory`);
  }

  return realRepoPath;
}
