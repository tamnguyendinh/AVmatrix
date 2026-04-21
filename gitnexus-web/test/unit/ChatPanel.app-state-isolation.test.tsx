import { act, render, waitFor } from '@testing-library/react';
import { useCallback } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AppStateProvider, useAppState } from '../../src/hooks/useAppState.local-runtime';
import { ChatRuntimeProvider } from '../../src/hooks/chat-runtime/ChatRuntimeContext';
import { ChatPanel } from '../../src/components/ChatPanel';

const transcriptRenderSpy = vi.fn();
const fetchSessionStatus = vi.fn();
const streamSessionChat = vi.fn();
const cancelSession = vi.fn();
const toAgentStreamChunk = vi.fn();

vi.mock('../../src/core/llm/session-client', () => ({
  SessionClientError: class SessionClientError extends Error {},
  fetchSessionStatus: (...args: unknown[]) => fetchSessionStatus(...args),
  streamSessionChat: (...args: unknown[]) => streamSessionChat(...args),
  cancelSession: (...args: unknown[]) => cancelSession(...args),
  toAgentStreamChunk: (...args: unknown[]) => toAgentStreamChunk(...args),
}));

vi.mock('../../src/hooks/useAutoScroll', () => ({
  useAutoScroll: () => ({
    scrollContainerRef: { current: null },
    messagesContainerRef: { current: null },
    isAtBottom: true,
    scrollToBottom: vi.fn(),
  }),
}));

vi.mock('../../src/components/right-panel/ChatTranscript', () => ({
  ChatTranscript: () => {
    transcriptRenderSpy();
    return <div>Transcript</div>;
  },
}));

vi.mock('../../src/components/right-panel/ChatComposer', () => ({
  ChatComposer: () => <div>Composer</div>,
}));

let appState: ReturnType<typeof useAppState> | null = null;

function ChatHarness() {
  const state = useAppState();
  const onRequestAnalyze = useCallback(() => {}, []);
  appState = state;

  return (
    <ChatRuntimeProvider bridge={state.chatRuntimeBridge}>
      <ChatPanel onRequestAnalyze={onRequestAnalyze} />
    </ChatRuntimeProvider>
  );
}

describe('ChatPanel app-state isolation', () => {
  beforeEach(() => {
    appState = null;
    transcriptRenderSpy.mockClear();
    fetchSessionStatus.mockReset();
    streamSessionChat.mockReset();
    cancelSession.mockReset();
    toAgentStreamChunk.mockReset();
    vi.stubGlobal('requestAnimationFrame', ((callback: FrameRequestCallback) => {
      callback(0);
      return 1;
    }) as typeof requestAnimationFrame);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('does not rerender the chat subtree or touch the runtime when graph state changes', async () => {
    render(
      <AppStateProvider>
        <ChatHarness />
      </AppStateProvider>,
    );

    await waitFor(() => expect(appState).not.toBeNull());

    const initialTranscriptRenders = transcriptRenderSpy.mock.calls.length;
    expect(fetchSessionStatus).not.toHaveBeenCalled();
    expect(streamSessionChat).not.toHaveBeenCalled();

    await act(async () => {
      appState!.setHighlightedNodeIds(new Set(['node-1']));
    });

    expect(transcriptRenderSpy.mock.calls).toHaveLength(initialTranscriptRenders);
    expect(fetchSessionStatus).not.toHaveBeenCalled();
    expect(streamSessionChat).not.toHaveBeenCalled();
  });
});
