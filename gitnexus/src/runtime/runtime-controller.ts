import fs from 'fs/promises';
import path from 'path';
import {
  hasIndex,
  listRegisteredRepos,
  getStoragePath,
} from '../storage/repo-manager.js';
import type {
  ResolvedSessionRepo,
  SessionChatRequest,
  SessionRepoBinding,
  SessionRepoResolution,
  SessionStatusResponse,
} from 'gitnexus-shared';
import type { SessionAdapter } from './session-adapter.js';
import { SessionJob, SessionRuntimeError } from './session-adapter.js';

const isLikelyRemoteUrl = (value: string): boolean =>
  /^[a-z][a-z0-9+.-]*:\/\//i.test(value) || /^git@/i.test(value);

const isUncPath = (value: string): boolean => value.startsWith('\\\\') || value.startsWith('//');

const samePath = (a: string, b: string): boolean =>
  process.platform === 'win32' ? a.toLowerCase() === b.toLowerCase() : a === b;

interface StartChatResult {
  job: SessionJob;
  repo: ResolvedSessionRepo;
}

export class RuntimeController {
  private readonly jobs = new Map<string, SessionJob>();
  private readonly activeRepoSessions = new Map<string, string>();

  constructor(private readonly adapter: SessionAdapter) {}

  async getStatus(binding?: SessionRepoBinding): Promise<SessionStatusResponse> {
    const status = await this.adapter.getStatus();
    if (!binding?.repoName && !binding?.repoPath) {
      return status;
    }

    try {
      const repo = await this.resolveRepo(binding);
      return {
        ...status,
        repo: {
          ...binding,
          state: repo.indexed ? 'indexed' : 'index_required',
          resolvedRepoName: repo.repoName,
          resolvedRepoPath: repo.repoPath,
        },
      };
    } catch (error) {
      if (error instanceof SessionRuntimeError) {
        return {
          ...status,
          repo: {
            ...binding,
            state: error.code === 'REPO_NOT_FOUND' ? 'not_found' : error.code === 'INDEX_REQUIRED' ? 'index_required' : 'invalid',
            message: error.message,
          },
        };
      }
      throw error;
    }
  }

  async startChat(request: SessionChatRequest): Promise<StartChatResult> {
    const repo = await this.resolveRepo(request);
    const adapterStatus = await this.adapter.getStatus();
    if (!repo.indexed) {
      throw new SessionRuntimeError(
        'INDEX_REQUIRED',
        `Repository "${repo.repoPath}" is not indexed yet. Run analyze first.`,
        409,
        {
          repoName: repo.repoName,
          repoPath: repo.repoPath,
        },
      );
    }

    const existingSessionId = this.activeRepoSessions.get(repo.repoPath);
    if (existingSessionId) {
      this.cancelSession(existingSessionId, 'Superseded by a newer chat on the same repository');
    }

    const job = new SessionJob(this.adapter.provider, repo.repoName, repo.repoPath, new AbortController());
    this.jobs.set(job.id, job);
    this.activeRepoSessions.set(repo.repoPath, job.id);

    job.emit({
      sessionId: job.id,
      provider: job.provider,
      repoName: job.repoName,
      repoPath: job.repoPath,
      timestamp: Date.now(),
      type: 'session_started',
      runtimeEnvironment: adapterStatus.runtimeEnvironment,
      executionMode: this.adapter.executionMode,
    });

    void this.adapter
      .runChat(job, request, { repo }, job.signal)
      .catch((error) => {
        if (job.status !== 'running') return;
        const message = error instanceof Error ? error.message : String(error);
        const runtimeError =
          error instanceof SessionRuntimeError
            ? error
            : new SessionRuntimeError('SESSION_START_FAILED', message, 500);
        job.emit({
          sessionId: job.id,
          provider: job.provider,
          repoName: job.repoName,
          repoPath: job.repoPath,
          timestamp: Date.now(),
          type: 'error',
          code: runtimeError.code,
          error: runtimeError.message,
        });
      })
      .finally(() => {
        if (this.activeRepoSessions.get(repo.repoPath) === job.id) {
          this.activeRepoSessions.delete(repo.repoPath);
        }
      });

    return { job, repo };
  }

  cancelSession(sessionId: string, reason = 'Cancelled by user'): boolean {
    const job = this.jobs.get(sessionId);
    if (!job || job.status !== 'running') return false;
    job.cancel(reason);
    return true;
  }

  getSession(sessionId: string): SessionJob | undefined {
    return this.jobs.get(sessionId);
  }

  dispose(): void {
    for (const job of this.jobs.values()) {
      if (job.status === 'running') {
        job.cancel('Runtime shutting down');
      }
    }
    this.jobs.clear();
    this.activeRepoSessions.clear();
  }

  private async resolveRepo(binding: SessionRepoBinding): Promise<ResolvedSessionRepo> {
    if (!binding.repoName && !binding.repoPath) {
      throw new SessionRuntimeError(
        'INVALID_REPO_BINDING',
        'Provide either "repoName" or "repoPath" for session binding',
        400,
      );
    }

    let resolvedFromName: ResolvedSessionRepo | null = null;
    if (binding.repoName) {
      const repos = await listRegisteredRepos();
      const entry =
        repos.find((repo) => repo.name === binding.repoName) ||
        repos.find((repo) => repo.name.toLowerCase() === binding.repoName!.toLowerCase());
      if (!entry) {
        throw new SessionRuntimeError(
          'REPO_NOT_FOUND',
          `Indexed repository "${binding.repoName}" was not found`,
          404,
        );
      }

      const entryPath = path.resolve(entry.path);
      let realRepoPath: string;
      try {
        realRepoPath = await fs.realpath(entryPath);
      } catch (error) {
        const code = typeof error === 'object' && error && 'code' in error ? (error as NodeJS.ErrnoException).code : undefined;
        if (code === 'ENOENT') {
          throw new SessionRuntimeError(
            'REPO_NOT_FOUND',
            `Indexed repository "${entry.name}" no longer exists at "${entry.path}"`,
            404,
            {
              repoName: entry.name,
              repoPath: entry.path,
            },
          );
        }
        throw new SessionRuntimeError(
          'INVALID_REPO_PATH',
          `Failed to resolve repository path "${entry.path}" for "${entry.name}"`,
          400,
          {
            repoName: entry.name,
            repoPath: entry.path,
          },
        );
      }

      let stat: Awaited<ReturnType<typeof fs.stat>>;
      try {
        stat = await fs.stat(realRepoPath);
      } catch (error) {
        const code = typeof error === 'object' && error && 'code' in error ? (error as NodeJS.ErrnoException).code : undefined;
        if (code === 'ENOENT') {
          throw new SessionRuntimeError(
            'REPO_NOT_FOUND',
            `Indexed repository "${entry.name}" no longer exists at "${entry.path}"`,
            404,
            {
              repoName: entry.name,
              repoPath: entry.path,
            },
          );
        }
        throw new SessionRuntimeError(
          'INVALID_REPO_PATH',
          `Failed to inspect repository path "${entry.path}" for "${entry.name}"`,
          400,
          {
            repoName: entry.name,
            repoPath: entry.path,
          },
        );
      }
      if (!stat.isDirectory()) {
        throw new SessionRuntimeError(
          'INVALID_REPO_PATH',
          `Indexed repository "${entry.name}" does not point to a directory`,
          400,
          {
            repoName: entry.name,
            repoPath: entry.path,
          },
        );
      }

      const indexed = await hasIndex(realRepoPath);
      resolvedFromName = {
        repoName: entry.name,
        repoPath: realRepoPath,
        indexed,
        storagePath: indexed ? (entry.storagePath || getStoragePath(realRepoPath)) : undefined,
      };
    }

    let resolvedFromPath: ResolvedSessionRepo | null = null;
    if (binding.repoPath) {
      if (isLikelyRemoteUrl(binding.repoPath)) {
        throw new SessionRuntimeError(
          'INVALID_REPO_PATH',
          'Remote URLs are not allowed for local session runtime',
          400,
        );
      }
      if (isUncPath(binding.repoPath)) {
        throw new SessionRuntimeError(
          'INVALID_REPO_PATH',
          'UNC and network-share paths are not allowed',
          400,
        );
      }
      if (!path.isAbsolute(binding.repoPath)) {
        throw new SessionRuntimeError(
          'INVALID_REPO_PATH',
          '"repoPath" must be an absolute local path',
          400,
        );
      }

      let realRepoPath: string;
      try {
        realRepoPath = await fs.realpath(binding.repoPath);
      } catch (error) {
        const code = typeof error === 'object' && error && 'code' in error ? (error as NodeJS.ErrnoException).code : undefined;
        if (code === 'ENOENT') {
          throw new SessionRuntimeError(
            'REPO_NOT_FOUND',
            `Repository path "${binding.repoPath}" does not exist`,
            404,
          );
        }
        throw new SessionRuntimeError(
          'INVALID_REPO_PATH',
          `Failed to resolve repository path "${binding.repoPath}"`,
          400,
        );
      }

      let stat: Awaited<ReturnType<typeof fs.stat>>;
      try {
        stat = await fs.stat(realRepoPath);
      } catch (error) {
        const code = typeof error === 'object' && error && 'code' in error ? (error as NodeJS.ErrnoException).code : undefined;
        if (code === 'ENOENT') {
          throw new SessionRuntimeError(
            'REPO_NOT_FOUND',
            `Repository path "${binding.repoPath}" does not exist`,
            404,
          );
        }
        throw new SessionRuntimeError(
          'INVALID_REPO_PATH',
          `Failed to inspect repository path "${binding.repoPath}"`,
          400,
        );
      }
      if (!stat.isDirectory()) {
        throw new SessionRuntimeError(
          'INVALID_REPO_PATH',
          `"${binding.repoPath}" is not a directory`,
          400,
        );
      }

      const indexed = await hasIndex(realRepoPath);
      const repos = await listRegisteredRepos();
      const entry = repos.find((repo) => samePath(path.resolve(repo.path), realRepoPath));

      resolvedFromPath = {
        repoName: entry?.name || path.basename(realRepoPath),
        repoPath: realRepoPath,
        indexed,
        storagePath: indexed ? (entry?.storagePath || getStoragePath(realRepoPath)) : undefined,
      };
    }

    if (resolvedFromName && resolvedFromPath) {
      if (!samePath(resolvedFromName.repoPath, resolvedFromPath.repoPath)) {
        throw new SessionRuntimeError(
          'INVALID_REPO_BINDING',
          `"repoName" and "repoPath" refer to different repositories`,
          400,
          {
            repoName: resolvedFromName.repoName,
            repoPath: resolvedFromPath.repoPath,
          },
        );
      }
      return resolvedFromName;
    }

    return resolvedFromName || resolvedFromPath!;
  }
}
