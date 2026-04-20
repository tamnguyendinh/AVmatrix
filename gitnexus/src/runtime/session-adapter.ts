import type {
  ResolvedSessionRepo,
  SessionChatRequest,
  SessionErrorCode,
  SessionExecutionMode,
  SessionRuntimeEnvironment,
  SessionStatus,
  SessionStreamEvent,
} from 'gitnexus-shared';
import { randomUUID } from 'crypto';
import { EventEmitter } from 'events';

export class SessionRuntimeError extends Error {
  constructor(
    public readonly code: SessionErrorCode,
    message: string,
    public readonly status: number = 400,
    public readonly details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = 'SessionRuntimeError';
  }
}

export interface SessionChatContext {
  repo: ResolvedSessionRepo;
}

export interface SessionAdapter {
  readonly provider: SessionStatus['provider'];
  readonly executionMode: SessionExecutionMode;
  readonly runtimeEnvironment: SessionRuntimeEnvironment;
  getStatus(): Promise<SessionStatus>;
  runChat(
    job: SessionJob,
    request: SessionChatRequest,
    context: SessionChatContext,
    signal: AbortSignal,
  ): Promise<void>;
}

export type SessionJobStatus = 'running' | 'completed' | 'failed' | 'cancelled';

type SessionJobListener = (event: SessionStreamEvent) => void;

export class SessionJob {
  readonly id = randomUUID();
  readonly startedAt = Date.now();
  status: SessionJobStatus = 'running';
  completedAt?: number;
  error?: string;

  private readonly emitter = new EventEmitter();
  private readonly history: SessionStreamEvent[] = [];

  constructor(
    public readonly provider: SessionStatus['provider'],
    public readonly repoName: string,
    public readonly repoPath: string,
    private readonly abortController: AbortController,
  ) {}

  emit(event: SessionStreamEvent): void {
    this.history.push(event);
    this.emitter.emit('event', event);

    if (event.type === 'done') {
      this.status = 'completed';
      this.completedAt = Date.now();
    } else if (event.type === 'error') {
      this.status = 'failed';
      this.error = event.error;
      this.completedAt = Date.now();
    } else if (event.type === 'cancelled') {
      this.status = 'cancelled';
      this.error = event.reason;
      this.completedAt = Date.now();
    }
  }

  onEvent(listener: SessionJobListener, replay = true): () => void {
    if (replay) {
      for (const event of this.history) {
        listener(event);
      }
    }

    this.emitter.on('event', listener);
    return () => this.emitter.off('event', listener);
  }

  cancel(reason: string): void {
    if (this.status !== 'running') return;
    this.abortController.abort(reason);
  }

  get signal(): AbortSignal {
    return this.abortController.signal;
  }
}
