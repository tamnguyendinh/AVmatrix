import { act, render, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { ReactNode } from 'react';
import { setBackendUrl } from '../../src/services/backend-client';
import {
  AppStateProvider,
  useAppState,
} from '../../src/hooks/useAppState.local-runtime';

const readyStatus = {
  provider: 'codex' as const,
  availability: 'ready' as const,
  available: true,
  authenticated: true,
  executablePath: 'bin/codex',
  version: 'test-version',
  runtimeEnvironment: 'wsl2' as const,
  executionMode: 'bypass' as const,
  supportsSse: true,
  supportsCancel: true,
  supportsMcp: true,
  repo: {
    repoName: 'GitNexus',
    state: 'indexed' as const,
    resolvedRepoName: 'GitNexus',
    resolvedRepoPath: 'repos/GitNexus',
  },
};

let appState: any = null;

function Harness() {
  appState = useAppState();
  return null;
}

function renderHarness(children?: ReactNode) {
  return render(
    <AppStateProvider>
      <Harness />
      {children}
    </AppStateProvider>,
  );
}

beforeEach(() => {
  appState = null;
  setBackendUrl('http://localhost:4747');
  vi.stubGlobal('requestAnimationFrame', ((callback: FrameRequestCallback) => {
    callback(0);
    return 1;
  }) as typeof requestAnimationFrame);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe('useAppState.local-runtime', () => {
  it('initializes against the local session runtime', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify(readyStatus), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    );

    renderHarness();
    await waitFor(() => expect(appState).not.toBeNull());

    await act(async () => {
      await appState.initializeAgent('GitNexus');
    });

    expect(appState.isAgentReady).toBe(true);
    expect(appState.agentError).toBeNull();
  });

  it('streams assistant content through the local session bridge', async () => {
    const encoder = new TextEncoder();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);

      if (url.includes('/api/session/status')) {
        return new Response(JSON.stringify(readyStatus), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      if (url.includes('/api/session/chat')) {
        const stream = new ReadableStream<Uint8Array>({
          start(controller) {
            controller.enqueue(
              encoder.encode(
                [
                  `event: session_started\ndata: ${JSON.stringify({
                    type: 'session_started',
                    sessionId: 'session-1',
                    provider: 'codex',
                    repoName: 'GitNexus',
                    repoPath: 'repos/GitNexus',
                    timestamp: Date.now(),
                    runtimeEnvironment: 'wsl2',
                    executionMode: 'bypass',
                  })}\n\n`,
                  `event: content\ndata: ${JSON.stringify({
                    type: 'content',
                    sessionId: 'session-1',
                    provider: 'codex',
                    repoName: 'GitNexus',
                    repoPath: 'repos/GitNexus',
                    timestamp: Date.now(),
                    content: 'Local runtime answer',
                  })}\n\n`,
                  `event: done\ndata: ${JSON.stringify({
                    type: 'done',
                    sessionId: 'session-1',
                    provider: 'codex',
                    repoName: 'GitNexus',
                    repoPath: 'repos/GitNexus',
                    timestamp: Date.now(),
                  })}\n\n`,
                ].join(''),
              ),
            );
            controller.close();
          },
        });

        return new Response(stream, {
          status: 200,
          headers: { 'Content-Type': 'text/event-stream' },
        });
      }

      throw new Error(`Unexpected fetch: ${url}`);
    });
    vi.stubGlobal('fetch', fetchMock);

    renderHarness();
    await waitFor(() => expect(appState).not.toBeNull());

    await act(async () => {
      await appState.initializeAgent('GitNexus');
    });

    await act(async () => {
      await appState.sendChatMessage('Explain the repo');
    });

    expect(appState.chatMessages).toHaveLength(2);
    expect(appState.chatMessages[0].role).toBe('user');
    expect(appState.chatMessages[1].role).toBe('assistant');
    expect(appState.chatMessages[1].content).toContain('Local runtime answer');
    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:4747/api/session/chat',
      expect.objectContaining({ method: 'POST' }),
    );
  });

  it('dedupes a final answer repeated as reasoning then content', async () => {
    const encoder = new TextEncoder();
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);

        if (url.includes('/api/session/status')) {
          return new Response(JSON.stringify(readyStatus), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        }

        if (url.includes('/api/session/chat')) {
          const stream = new ReadableStream<Uint8Array>({
            start(controller) {
              controller.enqueue(
                encoder.encode(
                  [
                    `event: session_started\ndata: ${JSON.stringify({
                      type: 'session_started',
                      sessionId: 'session-dedupe',
                      provider: 'codex',
                      repoName: 'GitNexus',
                      repoPath: 'repos/GitNexus',
                      timestamp: Date.now(),
                      runtimeEnvironment: 'wsl2',
                      executionMode: 'bypass',
                    })}\n\n`,
                    `event: reasoning\ndata: ${JSON.stringify({
                      type: 'reasoning',
                      sessionId: 'session-dedupe',
                      provider: 'codex',
                      repoName: 'GitNexus',
                      repoPath: 'repos/GitNexus',
                      timestamp: Date.now(),
                      reasoning: 'Chào bạn.',
                    })}\n\n`,
                    `event: content\ndata: ${JSON.stringify({
                      type: 'content',
                      sessionId: 'session-dedupe',
                      provider: 'codex',
                      repoName: 'GitNexus',
                      repoPath: 'repos/GitNexus',
                      timestamp: Date.now(),
                      content: 'Chào bạn.',
                    })}\n\n`,
                    `event: done\ndata: ${JSON.stringify({
                      type: 'done',
                      sessionId: 'session-dedupe',
                      provider: 'codex',
                      repoName: 'GitNexus',
                      repoPath: 'repos/GitNexus',
                      timestamp: Date.now(),
                    })}\n\n`,
                  ].join(''),
                ),
              );
              controller.close();
            },
          });

          return new Response(stream, {
            status: 200,
            headers: { 'Content-Type': 'text/event-stream' },
          });
        }

        throw new Error(`Unexpected fetch: ${url}`);
      }),
    );

    renderHarness();
    await waitFor(() => expect(appState).not.toBeNull());

    await act(async () => {
      await appState.initializeAgent('GitNexus');
    });

    await act(async () => {
      await appState.sendChatMessage('chào');
    });

    expect(appState.chatMessages).toHaveLength(2);
    const assistant = appState.chatMessages[1];
    expect(assistant.content).toBe('Chào bạn.');
    expect(assistant.steps).toHaveLength(1);
    expect(assistant.steps[0]).toMatchObject({
      type: 'content',
      content: 'Chào bạn.',
    });
  });

  it('cancels an active local session when stopChatResponse is called', async () => {
    const encoder = new TextEncoder();
    let chatController: ReadableStreamDefaultController<Uint8Array> | null = null;

    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);

      if (url.includes('/api/session/status')) {
        return new Response(JSON.stringify(readyStatus), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      if (url.includes('/api/session/chat')) {
        const stream = new ReadableStream<Uint8Array>({
          start(controller) {
            chatController = controller;
            controller.enqueue(
              encoder.encode(
                `event: session_started\ndata: ${JSON.stringify({
                  type: 'session_started',
                  sessionId: 'session-2',
                  provider: 'codex',
                  repoName: 'GitNexus',
                  repoPath: 'repos/GitNexus',
                  timestamp: Date.now(),
                  runtimeEnvironment: 'wsl2',
                  executionMode: 'bypass',
                })}\n\n`,
              ),
            );
          },
          cancel() {
            chatController = null;
          },
        });

        return new Response(stream, {
          status: 200,
          headers: { 'Content-Type': 'text/event-stream' },
        });
      }

      if (url.includes('/api/session/session-2')) {
        return new Response(JSON.stringify({ sessionId: 'session-2', status: 'cancelled' }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      throw new Error(`Unexpected fetch: ${url}`);
    });
    vi.stubGlobal('fetch', fetchMock);

    renderHarness();
    await waitFor(() => expect(appState).not.toBeNull());

    await act(async () => {
      await appState.initializeAgent('GitNexus');
    });

    const sendPromise = act(async () => {
      void appState.sendChatMessage('Start long running stream');
      await Promise.resolve();
    });
    await sendPromise;

    await waitFor(() => expect(appState.isChatLoading).toBe(true));

    await act(async () => {
      appState.stopChatResponse();
      chatController?.error(new DOMException('Aborted', 'AbortError'));
      await Promise.resolve();
    });

    await waitFor(() => expect(appState.isChatLoading).toBe(false));
    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:4747/api/session/session-2',
      expect.objectContaining({ method: 'DELETE' }),
    );
  });
});
