import { afterEach, describe, expect, it, vi } from 'vitest';
import type { SessionStatusResponse, SessionStreamEvent } from 'gitnexus-shared';
import { setBackendUrl } from '../../src/services/backend-client';
import {
  SessionClientError,
  cancelSession,
  fetchSessionStatus,
  streamSessionChat,
  toAgentStreamChunk,
} from '../../src/core/llm/session-client';

const readyStatus: SessionStatusResponse = {
  provider: 'codex',
  availability: 'ready',
  available: true,
  authenticated: true,
  executablePath: 'bin/codex',
  version: 'test-version',
  runtimeEnvironment: 'wsl2',
  executionMode: 'bypass',
  supportsSse: true,
  supportsCancel: true,
  supportsMcp: true,
  repo: {
    repoName: 'GitNexus',
    state: 'indexed',
    resolvedRepoName: 'GitNexus',
    resolvedRepoPath: 'repos/GitNexus',
  },
};

describe('session-client', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('fetches session status with repo binding query params', async () => {
    setBackendUrl('http://localhost:4747');

    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(readyStatus), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
    vi.stubGlobal('fetch', fetchMock);

    const result = await fetchSessionStatus({
      repoName: 'GitNexus',
      repoPath: 'repos/GitNexus',
    });

    expect(result).toEqual(readyStatus);
    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:4747/api/session/status?repoName=GitNexus&repoPath=repos%2FGitNexus',
    );
  });

  it('throws parsed SessionClientError for non-2xx status responses', async () => {
    setBackendUrl('http://localhost:4747');

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: 'INDEX_REQUIRED',
            error: 'Repository must be analyzed first',
            details: { repoName: 'GitNexus' },
          }),
          {
            status: 409,
            headers: { 'Content-Type': 'application/json' },
          },
        ),
      ),
    );

    await expect(fetchSessionStatus({ repoName: 'GitNexus' })).rejects.toMatchObject({
      name: 'SessionClientError',
      status: 409,
      code: 'INDEX_REQUIRED',
      details: { repoName: 'GitNexus' },
    });
  });

  it('sends cancel requests to the session endpoint', async () => {
    setBackendUrl('http://localhost:4747');

    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ sessionId: 'session-1', status: 'cancelled' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
    vi.stubGlobal('fetch', fetchMock);

    await cancelSession('session-1');

    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:4747/api/session/session-1',
      expect.objectContaining({ method: 'DELETE' }),
    );
  });

  it('parses SSE chat streams into ordered session events', async () => {
    setBackendUrl('http://localhost:4747');

    const encoder = new TextEncoder();
    const events: SessionStreamEvent[] = [
      {
        type: 'session_started',
        sessionId: 'session-1',
        provider: 'codex',
        repoName: 'GitNexus',
        repoPath: 'repos/GitNexus',
        timestamp: Date.now(),
        runtimeEnvironment: 'wsl2',
        executionMode: 'bypass',
      },
      {
        type: 'content',
        sessionId: 'session-1',
        provider: 'codex',
        repoName: 'GitNexus',
        repoPath: 'repos/GitNexus',
        timestamp: Date.now(),
        content: 'Hello from local runtime',
      },
      {
        type: 'done',
        sessionId: 'session-1',
        provider: 'codex',
        repoName: 'GitNexus',
        repoPath: 'repos/GitNexus',
        timestamp: Date.now(),
      },
    ];

    const stream = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(
          encoder.encode(
            events
              .map((event) => `event: ${event.type}\ndata: ${JSON.stringify(event)}\n\n`)
              .join(''),
          ),
        );
        controller.close();
      },
    });

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(stream, {
          status: 200,
          headers: { 'Content-Type': 'text/event-stream' },
        }),
      ),
    );

    const received: SessionStreamEvent[] = [];
    for await (const event of streamSessionChat({
      repoName: 'GitNexus',
      message: 'Explain this repo',
    })) {
      received.push(event);
    }

    expect(received).toEqual(events);
  });

  it('maps session stream events into legacy AgentStreamChunk shape', () => {
    expect(
      toAgentStreamChunk({
        type: 'reasoning',
        sessionId: 'session-1',
        provider: 'codex',
        repoName: 'GitNexus',
        repoPath: 'repos/GitNexus',
        timestamp: Date.now(),
        reasoning: 'Need to inspect callers',
      }),
    ).toEqual({
      type: 'reasoning',
      reasoning: 'Need to inspect callers',
    });

    expect(
      toAgentStreamChunk({
        type: 'tool_result',
        sessionId: 'session-1',
        provider: 'codex',
        repoName: 'GitNexus',
        repoPath: 'repos/GitNexus',
        timestamp: Date.now(),
        toolCall: {
          id: 'tool-1',
          name: 'search',
          args: { query: 'auth' },
          result: 'matched auth.ts',
          status: 'completed',
        },
      }),
    ).toEqual({
      type: 'tool_result',
      toolCall: {
        id: 'tool-1',
        name: 'search',
        args: { query: 'auth' },
        result: 'matched auth.ts',
        status: 'completed',
      },
    });

    expect(
      toAgentStreamChunk({
        type: 'session_started',
        sessionId: 'session-1',
        provider: 'codex',
        repoName: 'GitNexus',
        repoPath: 'repos/GitNexus',
        timestamp: Date.now(),
        runtimeEnvironment: 'wsl2',
        executionMode: 'bypass',
      }),
    ).toBeNull();
  });

  it('throws when the stream body is missing', async () => {
    setBackendUrl('http://localhost:4747');

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        body: null,
      }),
    );

    await expect(
      (async () => {
        const iterator = streamSessionChat({ repoName: 'GitNexus', message: 'hi' });
        await iterator.next();
      })(),
    ).rejects.toBeInstanceOf(SessionClientError);
  });
});
