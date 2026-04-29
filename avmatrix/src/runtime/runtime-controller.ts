import type { SessionAdapter } from './session-adapter.js';
import type {
  ResolvedSessionRepo,
  SessionChatRequest,
  SessionRepoBinding,
  SessionRepoResolution,
  SessionStatusResponse,
} from 'avmatrix-shared';
import { SessionJob, SessionRuntimeError } from './session-adapter.js';
import { RepoResolverError, resolveSessionRepoBinding } from './repo-resolver.js';

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
    try {
      return await resolveSessionRepoBinding(binding);
    } catch (error) {
      if (error instanceof RepoResolverError) {
        const status =
          error.code === 'REPO_NOT_FOUND' ? 404 : error.code === 'INVALID_REPO_BINDING' ? 400 : 400;
        throw new SessionRuntimeError(error.code, error.message, status, error.details);
      }
      throw error;
    }
  }
}
