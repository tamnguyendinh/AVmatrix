import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  ChatRuntimeProvider,
  useChatRuntime,
} from '../../src/hooks/chat-runtime/ChatRuntimeContext';
import type { ChatRuntimeBridge } from '../../src/hooks/chat-runtime/types';

const fetchSessionStatus = vi.fn();
const streamSessionChat = vi.fn();
const cancelSession = vi.fn();
const toAgentStreamChunk = vi.fn();

vi.mock('../../src/core/llm/session-client', () => ({
  SessionClientError: class SessionClientError extends Error {
    constructor(
      message: string,
      public readonly status: number,
      public readonly code: string,
      public readonly details?: Record<string, unknown>,
    ) {
      super(message);
      this.name = 'SessionClientError';
    }
  },
  fetchSessionStatus: (...args: unknown[]) => fetchSessionStatus(...args),
  streamSessionChat: (...args: unknown[]) => streamSessionChat(...args),
  cancelSession: (...args: unknown[]) => cancelSession(...args),
  toAgentStreamChunk: (...args: unknown[]) => toAgentStreamChunk(...args),
}));

const createBridge = (): ChatRuntimeBridge => ({
  getRepoName: vi.fn(() => 'website'),
  getEmbeddingStatus: vi.fn(() => 'idle'),
  clearAICodeReferences: vi.fn(),
  clearAIToolHighlights: vi.fn(),
  handleContentGrounding: vi.fn(),
  handleToolResult: vi.fn(),
  handleTranscriptLinkClick: vi.fn(),
});

describe('ChatRuntimeProvider', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    fetchSessionStatus.mockResolvedValue({
      provider: 'codex',
      availability: 'ready',
      available: true,
      authenticated: true,
      runtimeEnvironment: 'native',
      executionMode: 'bypass',
      supportsSse: true,
      supportsCancel: true,
      supportsMcp: true,
      repo: { state: 'indexed' },
    });
  });

  it('does not touch the session runtime on mount', () => {
    const bridge = createBridge();

    renderHook(() => useChatRuntime(), {
      wrapper: ({ children }) => <ChatRuntimeProvider bridge={bridge}>{children}</ChatRuntimeProvider>,
    });

    expect(fetchSessionStatus).not.toHaveBeenCalled();
    expect(streamSessionChat).not.toHaveBeenCalled();
  });

  it('starts the runtime only when send is requested', async () => {
    const bridge = createBridge();
    streamSessionChat.mockImplementation(async function* () {
      yield { type: 'session_started', sessionId: 'session-1' };
      yield { type: 'done', sessionId: 'session-1' };
    });
    toAgentStreamChunk.mockReturnValue({ type: 'done' });

    const { result } = renderHook(() => useChatRuntime(), {
      wrapper: ({ children }) => <ChatRuntimeProvider bridge={bridge}>{children}</ChatRuntimeProvider>,
    });

    await act(async () => {
      await result.current.sendChatMessage('hello');
    });

    await waitFor(() => {
      expect(fetchSessionStatus).toHaveBeenCalledTimes(1);
    });

    expect(streamSessionChat).toHaveBeenCalledTimes(1);
    expect(bridge.clearAICodeReferences).toHaveBeenCalledTimes(1);
    expect(bridge.clearAIToolHighlights).toHaveBeenCalledTimes(1);
  });

  it('clears chat state when the bound repo changes', async () => {
    let repoName = 'website';
    const bridge = createBridge();
    bridge.getRepoName = vi.fn(() => repoName);
    streamSessionChat.mockImplementation(async function* () {
      yield { type: 'session_started', sessionId: 'session-1' };
      yield { type: 'content', content: 'ready' };
      yield { type: 'done', sessionId: 'session-1' };
    });
    toAgentStreamChunk.mockImplementation((event: { type: string; content?: string }) => {
      if (event.type === 'content') {
        return { type: 'content', content: event.content };
      }
      if (event.type === 'done') {
        return { type: 'done' };
      }
      return null;
    });

    const { result, rerender } = renderHook(() => useChatRuntime(), {
      wrapper: ({ children }) => (
        <ChatRuntimeProvider bridge={bridge}>{children}</ChatRuntimeProvider>
      ),
    });

    await act(async () => {
      await result.current.sendChatMessage('hello');
    });

    await waitFor(() => {
      expect(result.current.chatMessages).toHaveLength(2);
    });

    repoName = 'restaurant-manager';
    rerender();

    await waitFor(() => {
      expect(result.current.chatMessages).toHaveLength(0);
      expect(result.current.isAgentReady).toBe(false);
    });
  });
});
