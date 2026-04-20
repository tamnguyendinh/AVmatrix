export type LocalSessionProvider = 'codex' | 'claude-code';

export type SessionAvailability =
  | 'ready'
  | 'not_installed'
  | 'not_signed_in'
  | 'error';

export type SessionExecutionMode = 'sandboxed' | 'bypass';

export type SessionErrorCode =
  | 'BAD_REQUEST'
  | 'INVALID_REPO_BINDING'
  | 'INVALID_REPO_PATH'
  | 'REPO_NOT_FOUND'
  | 'INDEX_REQUIRED'
  | 'SESSION_NOT_FOUND'
  | 'SESSION_RUNTIME_UNAVAILABLE'
  | 'SESSION_NOT_SIGNED_IN'
  | 'SESSION_START_FAILED'
  | 'SESSION_CANCELLED';

export interface SessionRepoBinding {
  repoName?: string;
  repoPath?: string;
}

export interface ResolvedSessionRepo {
  repoName: string;
  repoPath: string;
  indexed: boolean;
  storagePath?: string;
}

export interface SessionRepoResolution extends SessionRepoBinding {
  state: 'indexed' | 'index_required' | 'not_found' | 'invalid';
  resolvedRepoName?: string;
  resolvedRepoPath?: string;
  message?: string;
}

export interface SessionStatus {
  provider: LocalSessionProvider;
  availability: SessionAvailability;
  available: boolean;
  authenticated: boolean;
  executablePath?: string;
  version?: string;
  message?: string;
  recommendedEnvironment?: 'native' | 'wsl2';
  executionMode: SessionExecutionMode;
  supportsSse: boolean;
  supportsCancel: boolean;
  supportsMcp: boolean;
}

export interface SessionStatusResponse extends SessionStatus {
  repo?: SessionRepoResolution;
}

export interface SessionChatRequest extends SessionRepoBinding {
  message: string;
}

export interface SessionToolCall {
  id: string;
  name: string;
  args?: Record<string, unknown>;
  result?: string;
  status: 'pending' | 'running' | 'completed' | 'error';
}

interface SessionEventBase {
  sessionId: string;
  provider: LocalSessionProvider;
  repoName: string;
  repoPath: string;
  timestamp: number;
}

export interface SessionStartedEvent extends SessionEventBase {
  type: 'session_started';
  executionMode: SessionExecutionMode;
}

export interface SessionReasoningEvent extends SessionEventBase {
  type: 'reasoning';
  reasoning: string;
}

export interface SessionContentEvent extends SessionEventBase {
  type: 'content';
  content: string;
}

export interface SessionToolCallEvent extends SessionEventBase {
  type: 'tool_call';
  toolCall: SessionToolCall;
}

export interface SessionToolResultEvent extends SessionEventBase {
  type: 'tool_result';
  toolCall: SessionToolCall;
}

export interface SessionErrorEvent extends SessionEventBase {
  type: 'error';
  code: SessionErrorCode;
  error: string;
}

export interface SessionCancelledEvent extends SessionEventBase {
  type: 'cancelled';
  reason: string;
}

export interface SessionDoneEvent extends SessionEventBase {
  type: 'done';
  usage?: Record<string, number>;
}

export type SessionStreamEvent =
  | SessionStartedEvent
  | SessionReasoningEvent
  | SessionContentEvent
  | SessionToolCallEvent
  | SessionToolResultEvent
  | SessionErrorEvent
  | SessionCancelledEvent
  | SessionDoneEvent;
