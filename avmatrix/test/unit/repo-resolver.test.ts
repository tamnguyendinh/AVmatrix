import fs from 'fs/promises';
import os from 'os';
import path from 'path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { SessionRepoBinding } from 'avmatrix-shared';
import {
  assignRepoRuntimeIds,
  buildRepoLabels,
  findRepoCandidate,
  RepoResolverError,
  resolveSessionRepoBinding,
  type RepoLookupCandidate,
} from '../../src/runtime/repo-resolver.js';

const repoManagerMocks = vi.hoisted(() => ({
  listRegisteredRepos: vi.fn(),
  hasIndex: vi.fn(),
  getStoragePath: vi.fn((repoPath: string) => `${repoPath}/.avmatrix`),
}));

vi.mock('../../src/storage/repo-manager.js', () => repoManagerMocks);

describe('repo-resolver', () => {
  const tempDirs: string[] = [];

  const createTempRepo = async (label: string): Promise<string> => {
    const dir = await fs.mkdtemp(path.join(os.tmpdir(), `avmatrix-${label}-`));
    tempDirs.push(dir);
    return dir;
  };

  beforeEach(() => {
    vi.clearAllMocks();
    repoManagerMocks.listRegisteredRepos.mockResolvedValue([]);
    repoManagerMocks.hasIndex.mockResolvedValue(false);
  });

  afterEach(async () => {
    await Promise.allSettled(
      tempDirs.splice(0).map((dir) => fs.rm(dir, { recursive: true, force: true })),
    );
  });

  it('builds disambiguated labels only for colliding names', () => {
    const repos: RepoLookupCandidate[] = [
      { name: 'demo', repoPath: 'F:/one/demo', id: 'demo' },
      { name: 'demo', repoPath: 'F:/two/demo', id: 'demo-2' },
      { name: 'other', repoPath: 'F:/other/other', id: 'other' },
    ];

    expect(buildRepoLabels(repos)).toEqual(['demo (F:/one/demo)', 'demo (F:/two/demo)', 'other']);
  });

  it('finds candidates by id, name, path, and partial name', () => {
    const repos: RepoLookupCandidate[] = [
      { id: 'alpha', name: 'Alpha', repoPath: path.resolve('F:/repos/alpha') },
      { id: 'beta', name: 'BetaTool', repoPath: path.resolve('F:/repos/beta') },
    ];

    expect(findRepoCandidate(repos, 'alpha', { matchId: true })?.name).toBe('Alpha');
    expect(findRepoCandidate(repos, 'betatool')?.name).toBe('BetaTool');
    expect(findRepoCandidate(repos, 'F:/repos/beta')?.name).toBe('BetaTool');
    expect(findRepoCandidate(repos, 'beta', { allowPartialName: true })?.name).toBe('BetaTool');
  });

  it('assigns stable runtime ids and hashes colliding names', () => {
    const repos = assignRepoRuntimeIds([
      { name: 'Demo', repoPath: 'F:/one/demo' },
      { name: 'Demo', repoPath: 'F:/two/demo' },
      { name: 'Other', repoPath: 'F:/other/other' },
    ]);

    expect(repos[0].id).toBe('demo');
    expect(repos[1].id).toMatch(/^demo-/);
    expect(repos[1].id).not.toBe('demo');
    expect(repos[2].id).toBe('other');
  });

  it('resolves session binding by repoName through the shared resolver core', async () => {
    const repoPath = await createTempRepo('resolver-name');
    repoManagerMocks.listRegisteredRepos.mockResolvedValue([
      {
        name: 'demo',
        path: repoPath,
        storagePath: `${repoPath}/.avmatrix`,
        indexedAt: '2026-04-29T00:00:00.000Z',
        lastCommit: 'abc123',
      },
    ]);
    repoManagerMocks.hasIndex.mockResolvedValue(true);

    const resolved = await resolveSessionRepoBinding({ repoName: 'demo' });

    expect(resolved).toMatchObject({
      repoName: 'demo',
      repoPath: await fs.realpath(repoPath),
      indexed: true,
      storagePath: `${repoPath}/.avmatrix`,
    });
  });

  it('rejects mismatched repoName/repoPath via shared binding resolver', async () => {
    const repoPath = await createTempRepo('resolver-name-path');
    const differentRepoPath = await createTempRepo('resolver-different-path');
    repoManagerMocks.listRegisteredRepos.mockResolvedValue([
      {
        name: 'demo',
        path: repoPath,
        storagePath: `${repoPath}/.avmatrix`,
        indexedAt: '2026-04-29T00:00:00.000Z',
        lastCommit: 'abc123',
      },
    ]);
    repoManagerMocks.hasIndex.mockResolvedValue(true);

    const binding: SessionRepoBinding = {
      repoName: 'demo',
      repoPath: differentRepoPath,
    };

    await expect(resolveSessionRepoBinding(binding)).rejects.toMatchObject<
      Partial<RepoResolverError>
    >({
      name: 'RepoResolverError',
      code: 'INVALID_REPO_BINDING',
    });
  });
});
