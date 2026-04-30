import fs from 'fs/promises';
import path from 'path';
import {
  getStoragePath,
  hasIndex,
  listRegisteredRepos,
  type RegistryEntry,
} from '../storage/repo-manager.js';
import type { ResolvedSessionRepo, SessionRepoBinding } from 'avmatrix-shared';

export type RepoResolverErrorCode = 'INVALID_REPO_BINDING' | 'INVALID_REPO_PATH' | 'REPO_NOT_FOUND';

export class RepoResolverError extends Error {
  constructor(
    readonly code: RepoResolverErrorCode,
    message: string,
    readonly details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = 'RepoResolverError';
  }
}

export interface RepoLookupCandidate {
  name: string;
  repoPath: string;
  id?: string;
}

export interface RepoRuntimeCandidate extends RepoLookupCandidate {
  id: string;
}

export const isLikelyRemoteUrl = (value: string): boolean =>
  /^[a-z][a-z0-9+.-]*:\/\//i.test(value) || /^git@/i.test(value);

export const isUncPath = (value: string): boolean =>
  value.startsWith('\\\\') || value.startsWith('//');

export const samePath = (a: string, b: string): boolean =>
  process.platform === 'win32' ? a.toLowerCase() === b.toLowerCase() : a === b;

export const normalizeRepoParam = (repoParam?: string): string | undefined =>
  repoParam ? path.basename(repoParam) : undefined;

export const buildRepoLabels = <T extends RepoLookupCandidate>(repos: Iterable<T>): string[] => {
  const all = [...repos];
  const nameCounts = new Map<string, number>();
  for (const repo of all) {
    const key = repo.name.toLowerCase();
    nameCounts.set(key, (nameCounts.get(key) ?? 0) + 1);
  }

  return all.map((repo) =>
    (nameCounts.get(repo.name.toLowerCase()) ?? 0) > 1
      ? `${repo.name} (${repo.repoPath})`
      : repo.name,
  );
};

export const buildRepoRuntimeId = (
  name: string,
  repoPath: string,
  existing: Iterable<RepoLookupCandidate> = [],
): string => {
  const normalizedPath = path.resolve(repoPath);
  const base = name.toLowerCase();
  for (const repo of existing) {
    if (repo.id === base && !samePath(path.resolve(repo.repoPath), normalizedPath)) {
      const hash = Buffer.from(normalizedPath).toString('base64url').slice(0, 6);
      return `${base}-${hash}`;
    }
  }
  return base;
};

export const assignRepoRuntimeIds = <T extends RepoLookupCandidate>(
  repos: Iterable<T>,
): Array<T & RepoRuntimeCandidate> => {
  const assigned: Array<T & RepoRuntimeCandidate> = [];
  for (const repo of repos) {
    assigned.push({
      ...repo,
      id: buildRepoRuntimeId(repo.name, repo.repoPath, assigned),
    });
  }
  return assigned;
};

interface FindRepoOptions {
  allowPartialName?: boolean;
  allowSingleDefault?: boolean;
  matchId?: boolean;
}

export const findRepoCandidate = <T extends RepoLookupCandidate>(
  repos: Iterable<T>,
  repoParam?: string,
  options: FindRepoOptions = {},
): T | null => {
  const { allowPartialName = false, allowSingleDefault = false, matchId = false } = options;
  const all = [...repos];
  if (all.length === 0) return null;

  if (!repoParam) {
    return allowSingleDefault && all.length === 1 ? all[0] : null;
  }

  const normalizedParam = normalizeRepoParam(repoParam) ?? repoParam;
  const paramLower = normalizedParam.toLowerCase();

  if (matchId) {
    const byId = all.find((repo) => repo.id?.toLowerCase() === paramLower);
    if (byId) return byId;
  }

  const byName = all.find((repo) => repo.name.toLowerCase() === paramLower);
  if (byName) return byName;

  const resolvedPath = path.resolve(repoParam);
  const byPath = all.find((repo) => samePath(path.resolve(repo.repoPath), resolvedPath));
  if (byPath) return byPath;

  if (allowPartialName) {
    const byPartialName = all.find((repo) => repo.name.toLowerCase().includes(paramLower));
    if (byPartialName) return byPartialName;
  }

  return null;
};

const resolveExistingDirectory = async (
  targetPath: string,
  notFoundMessage: string,
  invalidMessage: string,
  details?: Record<string, unknown>,
): Promise<string> => {
  let realRepoPath: string;
  try {
    realRepoPath = await fs.realpath(targetPath);
  } catch (error) {
    const code =
      typeof error === 'object' && error && 'code' in error
        ? (error as NodeJS.ErrnoException).code
        : undefined;
    if (code === 'ENOENT') {
      throw new RepoResolverError('REPO_NOT_FOUND', notFoundMessage, details);
    }
    throw new RepoResolverError('INVALID_REPO_PATH', invalidMessage, details);
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
      throw new RepoResolverError('REPO_NOT_FOUND', notFoundMessage, details);
    }
    throw new RepoResolverError('INVALID_REPO_PATH', invalidMessage, details);
  }

  if (!stat.isDirectory()) {
    throw new RepoResolverError('INVALID_REPO_PATH', invalidMessage, details);
  }

  return realRepoPath;
};

const resolveRepoFromName = async (repoName: string): Promise<ResolvedSessionRepo> => {
  const repos = await listRegisteredRepos();
  const entry =
    findRepoCandidate(
      repos.map((repo) => ({
        ...repo,
        repoPath: path.resolve(repo.path),
      })),
      repoName,
      { allowSingleDefault: false },
    ) ?? null;

  if (!entry) {
    throw new RepoResolverError(
      'REPO_NOT_FOUND',
      `Indexed repository "${repoName}" was not found`,
      {
        repoName,
      },
    );
  }

  const realRepoPath = await resolveExistingDirectory(
    entry.repoPath,
    `Indexed repository "${entry.name}" no longer exists at "${entry.path}"`,
    `Failed to inspect repository path "${entry.path}" for "${entry.name}"`,
    {
      repoName: entry.name,
      repoPath: entry.path,
    },
  );

  const indexed = await hasIndex(realRepoPath);
  return {
    repoName: entry.name,
    repoPath: realRepoPath,
    indexed,
    storagePath: indexed ? entry.storagePath || getStoragePath(realRepoPath) : undefined,
  };
};

const resolveRepoFromPath = async (repoPath: string): Promise<ResolvedSessionRepo> => {
  if (isLikelyRemoteUrl(repoPath)) {
    throw new RepoResolverError(
      'INVALID_REPO_PATH',
      'Remote URLs are not allowed for local session runtime',
    );
  }
  if (isUncPath(repoPath)) {
    throw new RepoResolverError('INVALID_REPO_PATH', 'UNC and network-share paths are not allowed');
  }
  if (!path.isAbsolute(repoPath)) {
    throw new RepoResolverError('INVALID_REPO_PATH', '"repoPath" must be an absolute local path');
  }

  const realRepoPath = await resolveExistingDirectory(
    repoPath,
    `Repository path "${repoPath}" does not exist`,
    `Failed to inspect repository path "${repoPath}"`,
    { repoPath },
  );

  const indexed = await hasIndex(realRepoPath);
  const repos = await listRegisteredRepos();
  const entry = repos.find((repo) => samePath(path.resolve(repo.path), realRepoPath));

  return {
    repoName: entry?.name || path.basename(realRepoPath),
    repoPath: realRepoPath,
    indexed,
    storagePath: indexed ? entry?.storagePath || getStoragePath(realRepoPath) : undefined,
  };
};

export const resolveSessionRepoBinding = async (
  binding: SessionRepoBinding,
): Promise<ResolvedSessionRepo> => {
  if (!binding.repoName && !binding.repoPath) {
    throw new RepoResolverError(
      'INVALID_REPO_BINDING',
      'Provide either "repoName" or "repoPath" for session binding',
    );
  }

  const resolvedFromName = binding.repoName ? await resolveRepoFromName(binding.repoName) : null;
  const resolvedFromPath = binding.repoPath ? await resolveRepoFromPath(binding.repoPath) : null;

  if (resolvedFromName && resolvedFromPath) {
    if (!samePath(resolvedFromName.repoPath, resolvedFromPath.repoPath)) {
      throw new RepoResolverError(
        'INVALID_REPO_BINDING',
        '"repoName" and "repoPath" refer to different repositories',
        {
          repoName: resolvedFromName.repoName,
          repoPath: resolvedFromPath.repoPath,
        },
      );
    }
    return resolvedFromName;
  }

  return resolvedFromName || resolvedFromPath!;
};

export const toRepoLookupCandidate = (entry: RegistryEntry): RepoLookupCandidate => ({
  name: entry.name,
  repoPath: path.resolve(entry.path),
});
