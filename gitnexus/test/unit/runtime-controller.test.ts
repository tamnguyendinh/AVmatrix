import fs from 'fs/promises';
import os from 'os';
import path from 'path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { SessionStatus } from 'gitnexus-shared';
import { RuntimeController } from '../../src/runtime/runtime-controller.js';
import type { SessionAdapter, SessionChatContext } from '../../src/runtime/session-adapter.js';
import { SessionRuntimeError } from '../../src/runtime/session-adapter.js';
import type { SessionJob } from '../../src/runtime/session-adapter.js';

const repoManagerMocks = vi.hoisted(() => ({
  listRegisteredRepos: vi.fn(),
  hasIndex: vi.fn(),
  getStoragePath: vi.fn((repoPath: string) => `${repoPath}/.gitnexus`),
}));

vi.mock('../../src/storage/repo-manager.js', () => repoManagerMocks);

class FakeSessionAdapter implements SessionAdapter {
  readonly provider = 'codex' as const;
  readonly executionMode = 'bypass' as const;
  readonly status: SessionStatus = {
    provider: 'codex',
    availability: 'ready',
    available: true,
    authenticated: true,
    executablePath: 'codex.cmd',
    version: 'codex-cli test',
    recommendedEnvironment: 'wsl2',
    executionMode: 'bypass',
    supportsSse: true,
    supportsCancel: true,
    supportsMcp: true,
  };

  readonly runs: Array<{
    job: SessionJob;
    message: string;
    repoName: string;
    repoPath: string;
  }> = [];

  runImpl: (
    job: SessionJob,
    context: SessionChatContext,
    signal: AbortSignal,
  ) => Promise<void> = async (job, _context, signal) => {
    await new Promise<void>((resolve) => {
      signal.addEventListener(
        'abort',
        () => {
          job.emit({
            sessionId: job.id,
            provider: job.provider,
            repoName: job.repoName,
            repoPath: job.repoPath,
            timestamp: Date.now(),
            type: 'cancelled',
            reason: typeof signal.reason === 'string' ? signal.reason : 'Cancelled',
          });
          resolve();
        },
        { once: true },
      );
    });
  };

  async getStatus(): Promise<SessionStatus> {
    return this.status;
  }

  async runChat(
    job: SessionJob,
    request: { message: string },
    context: SessionChatContext,
    signal: AbortSignal,
  ): Promise<void> {
    this.runs.push({
      job,
      message: request.message,
      repoName: context.repo.repoName,
      repoPath: context.repo.repoPath,
    });
    await this.runImpl(job, context, signal);
  }
}

describe('RuntimeController', () => {
  const tempDirs: string[] = [];
  const runtimes: RuntimeController[] = [];

  const createTempRepo = async (label: string): Promise<string> => {
    const dir = await fs.mkdtemp(path.join(os.tmpdir(), `gitnexus-${label}-`));
    tempDirs.push(dir);
    return dir;
  };

  const createRuntime = (adapter = new FakeSessionAdapter()) => {
    const runtime = new RuntimeController(adapter);
    runtimes.push(runtime);
    return { runtime, adapter };
  };

  beforeEach(() => {
    vi.clearAllMocks();
    repoManagerMocks.listRegisteredRepos.mockResolvedValue([]);
    repoManagerMocks.hasIndex.mockResolvedValue(false);
  });

  afterEach(async () => {
    for (const runtime of runtimes.splice(0)) {
      runtime.dispose();
    }
    await Promise.allSettled(tempDirs.splice(0).map((dir) => fs.rm(dir, { recursive: true, force: true })));
  });

  it('reports indexed repo binding in session status', async () => {
    const repoPath = await createTempRepo('indexed-status');
    repoManagerMocks.listRegisteredRepos.mockResolvedValue([
      {
        name: 'demo',
        path: repoPath,
        storagePath: `${repoPath}/.gitnexus`,
        indexedAt: '2026-04-20T00:00:00.000Z',
        lastCommit: 'abc123',
      },
    ]);

    const { runtime } = createRuntime();
    const status = await runtime.getStatus({ repoName: 'demo' });

    expect(status.available).toBe(true);
    expect(status.repo).toMatchObject({
      state: 'indexed',
      resolvedRepoName: 'demo',
      resolvedRepoPath: repoPath,
    });
  });

  it('reports index_required for an existing local repo path without an index', async () => {
    const repoPath = await createTempRepo('index-required-status');
    const canonicalRepoPath = await fs.realpath(repoPath);
    repoManagerMocks.hasIndex.mockResolvedValue(false);

    const { runtime } = createRuntime();
    const status = await runtime.getStatus({ repoPath });

    expect(status.repo).toMatchObject({
      state: 'index_required',
      resolvedRepoName: path.basename(canonicalRepoPath),
      resolvedRepoPath: canonicalRepoPath,
    });
  });

  it('rejects chat starts with INDEX_REQUIRED for unindexed local repos', async () => {
    const repoPath = await createTempRepo('index-required-start');
    repoManagerMocks.hasIndex.mockResolvedValue(false);

    const { runtime } = createRuntime();

    await expect(runtime.startChat({ repoPath, message: 'hello' })).rejects.toMatchObject({
      name: 'SessionRuntimeError',
      code: 'INDEX_REQUIRED',
      status: 409,
    });
  });

  it('rejects mismatched repoName and repoPath bindings', async () => {
    const indexedRepoPath = await createTempRepo('indexed-name');
    const differentRepoPath = await createTempRepo('different-path');
    repoManagerMocks.listRegisteredRepos.mockResolvedValue([
      {
        name: 'demo',
        path: indexedRepoPath,
        storagePath: `${indexedRepoPath}/.gitnexus`,
        indexedAt: '2026-04-20T00:00:00.000Z',
        lastCommit: 'abc123',
      },
    ]);
    repoManagerMocks.hasIndex.mockResolvedValue(true);

    const { runtime } = createRuntime();

    await expect(
      runtime.startChat({
        repoName: 'demo',
        repoPath: differentRepoPath,
        message: 'hello',
      }),
    ).rejects.toMatchObject({
      name: 'SessionRuntimeError',
      code: 'INVALID_REPO_BINDING',
      status: 400,
    });
  });

  it('cancels the previous repo session when a new chat starts on the same repo', async () => {
    const repoPath = await createTempRepo('steal-session');
    repoManagerMocks.hasIndex.mockResolvedValue(true);

    const { runtime, adapter } = createRuntime();

    const first = await runtime.startChat({ repoPath, message: 'first' });
    const second = await runtime.startChat({ repoPath, message: 'second' });

    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(first.job.signal.aborted).toBe(true);
    expect(first.job.status).toBe('cancelled');
    expect(second.job.status).toBe('running');
    expect(adapter.runs).toHaveLength(2);
    expect(adapter.runs.map((run) => run.message)).toEqual(['first', 'second']);
  });

  it('rejects remote URLs and UNC paths for repoPath bindings', async () => {
    const { runtime } = createRuntime();

    await expect(
      runtime.startChat({ repoPath: 'https://github.com/example/repo.git', message: 'hello' }),
    ).rejects.toMatchObject({
      code: 'INVALID_REPO_PATH',
      status: 400,
    });

    await expect(
      runtime.startChat({ repoPath: '\\\\server\\share\\repo', message: 'hello' }),
    ).rejects.toMatchObject({
      code: 'INVALID_REPO_PATH',
      status: 400,
    });
  });
});
