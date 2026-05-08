import fs from 'fs/promises';
import os from 'os';
import path from 'path';
import { afterEach, describe, expect, it } from 'vitest';
import { resolveAnalyzeRepoPath } from '../../src/server/local-path-policy.js';
import {
  LOCAL_ANALYZE_PREPARING_PROGRESS,
  buildAnalyzeCompleteEventPayload,
  buildAnalyzeWorkerOptions,
  isActiveAnalyzeJobStatus,
} from '../../src/server/api.js';
import type { AnalyzeJob } from '../../src/server/analyze-job.js';

const tempDirs: string[] = [];

async function createTempRepo(name: string): Promise<string> {
  const dir = await fs.mkdtemp(path.join(os.tmpdir(), `${name}-`));
  tempDirs.push(dir);
  return dir;
}

afterEach(async () => {
  await Promise.all(tempDirs.splice(0).map((dir) => fs.rm(dir, { recursive: true, force: true })));
});

type AnalyzeStatus = AnalyzeJob['status'];
type AllowsCloneEraStatus = 'cloning' extends AnalyzeStatus ? true : false;
const allowsCloneEraStatus: AllowsCloneEraStatus = false;

describe('analyze API local path policy', () => {
  it('removes clone-era analyze job states from the active local-only contract', () => {
    expect(allowsCloneEraStatus).toBe(false);
    expect(isActiveAnalyzeJobStatus('queued')).toBe(true);
    expect(isActiveAnalyzeJobStatus('analyzing')).toBe(true);
    expect(isActiveAnalyzeJobStatus('loading')).toBe(false);
    expect(LOCAL_ANALYZE_PREPARING_PROGRESS).toEqual({
      phase: 'analyzing',
      percent: 0,
      message: 'Preparing local analysis...',
    });
  });

  it('builds full-analyze worker options for Web/API analyze requests', () => {
    expect(buildAnalyzeWorkerOptions()).toEqual({ force: true, embeddings: false });
    expect(buildAnalyzeWorkerOptions({ embeddings: true })).toEqual({
      force: true,
      embeddings: true,
    });
    expect(buildAnalyzeWorkerOptions({ force: false })).toEqual({
      force: true,
      embeddings: false,
    });
  });

  it('includes repoPath in analyze completion payloads', () => {
    expect(
      buildAnalyzeCompleteEventPayload({
        repoName: 'DisplayedName',
        repoPath: 'F:\\repos\\selected',
      }),
    ).toEqual({
      repoName: 'DisplayedName',
      repoPath: 'F:\\repos\\selected',
      error: undefined,
    });
  });

  it('resolves an existing absolute local repo path to its canonical path', async () => {
    const repoPath = await createTempRepo('avmatrix-analyze');

    await expect(resolveAnalyzeRepoPath(repoPath)).resolves.toBe(await fs.realpath(repoPath));
  });

  it('rejects remote URLs', async () => {
    await expect(resolveAnalyzeRepoPath('https://github.com/user/repo')).rejects.toThrow(
      /local filesystem path/,
    );
  });

  it('rejects relative paths', async () => {
    await expect(resolveAnalyzeRepoPath('repos/AVmatrix')).rejects.toThrow(/absolute path/);
  });

  it('rejects UNC paths', async () => {
    await expect(resolveAnalyzeRepoPath('\\\\server\\share\\repo')).rejects.toThrow(
      /UNC and network-share paths/,
    );
  });

  it('rejects missing folders', async () => {
    const missingPath = path.join(os.tmpdir(), 'avmatrix-missing-repo');
    await expect(resolveAnalyzeRepoPath(missingPath)).rejects.toThrow(/does not exist/);
  });
});
