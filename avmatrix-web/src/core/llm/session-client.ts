import type {
  SessionChatRequest,
  SessionRepoBinding,
  SessionStatusResponse,
  SessionStreamEvent,
} from 'avmatrix-shared';
import { getBackendUrl } from '../../services/backend-client';
import type { AgentStreamChunk } from './types';

export class SessionClientError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly code: string,
    public readonly details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = 'SessionClientError';
  }
}

const buildQuery = (binding?: SessionRepoBinding): string => {
  const params = new URLSearchParams();
  if (binding?.repoName) params.set('repoName', binding.repoName);
  if (binding?.repoPath) params.set('repoPath', binding.repoPath);
  const query = params.toString();
  return query ? `?${query}` : '';
};

const parseJsonError = async (response: Response): Promise<SessionClientError> => {
  try {
    const body = (await response.json()) as {
      error?: string;
      code?: string;
      details?: Record<string, unknown>;
    };

    return new SessionClientError(
      body.error || response.statusText || 'Session request failed',
      response.status,
      body.code || 'SESSION_START_FAILED',
      body.details,
    );
  } catch {
    return new SessionClientError(
      response.statusText || 'Session request failed',
      response.status,
      'SESSION_START_FAILED',
    );
  }
};

export const fetchSessionStatus = async (
  binding?: SessionRepoBinding,
): Promise<SessionStatusResponse> => {
  const response = await fetch(`${getBackendUrl()}/api/session/status${buildQuery(binding)}`);
  if (!response.ok) {
    throw await parseJsonError(response);
  }
  return response.json() as Promise<SessionStatusResponse>;
};

export const cancelSession = async (sessionId: string): Promise<void> => {
  const response = await fetch(`${getBackendUrl()}/api/session/${encodeURIComponent(sessionId)}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    throw await parseJsonError(response);
  }
};

export const toAgentStreamChunk = (event: SessionStreamEvent): AgentStreamChunk | null => {
  switch (event.type) {
    case 'reasoning':
      return { type: 'reasoning', reasoning: event.reasoning };
    case 'content':
      return { type: 'content', content: event.content };
    case 'tool_call':
      return {
        type: 'tool_call',
        toolCall: {
          id: event.toolCall.id,
          name: event.toolCall.name,
          args: event.toolCall.args ?? {},
          result: event.toolCall.result,
          status: event.toolCall.status,
        },
      };
    case 'tool_result':
      return {
        type: 'tool_result',
        toolCall: {
          id: event.toolCall.id,
          name: event.toolCall.name,
          args: event.toolCall.args ?? {},
          result: event.toolCall.result,
          status: event.toolCall.status,
        },
      };
    case 'error':
      return { type: 'error', error: event.error };
    case 'cancelled':
    case 'done':
      return { type: 'done' };
    case 'session_started':
    default:
      return null;
  }
};

export async function* streamSessionChat(
  request: SessionChatRequest,
  signal?: AbortSignal,
): AsyncGenerator<SessionStreamEvent> {
  const response = await fetch(`${getBackendUrl()}/api/session/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
    signal,
  });

  if (!response.ok) {
    throw await parseJsonError(response);
  }

  const reader = response.body?.getReader();
  if (!reader) {
    throw new SessionClientError(
      'Session stream did not return a response body',
      500,
      'BAD_REQUEST',
    );
  }

  const decoder = new TextDecoder();
  let buffer = '';
  let currentData: string[] = [];

  const flush = (): SessionStreamEvent | null => {
    if (currentData.length === 0) return null;
    const payload = currentData.join('\n');
    currentData = [];
    try {
      return JSON.parse(payload) as SessionStreamEvent;
    } catch {
      return null;
    }
  };

  while (true) {
    const { done, value } = await reader.read();
    if (done) {
      const finalEvent = flush();
      if (finalEvent) {
        yield finalEvent;
      }
      return;
    }

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split(/\r?\n/);
    buffer = lines.pop() || '';

    for (const line of lines) {
      if (!line) {
        const event = flush();
        if (event) {
          yield event;
        }
        continue;
      }

      if (line.startsWith(':') || line.startsWith('event: ')) {
        continue;
      }

      if (line.startsWith('data: ')) {
        currentData.push(line.slice(6));
      }
    }
  }
}
