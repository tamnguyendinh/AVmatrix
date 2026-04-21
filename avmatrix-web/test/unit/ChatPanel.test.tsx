import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockChatRuntime = {
  llmSettings: {
    activeProvider: 'codex',
    intelligentClustering: false,
    hasSeenClusteringPrompt: false,
    useSameModelForClustering: true,
    codex: { model: 'codex-account', temperature: 0 },
  },
  updateLLMSettings: vi.fn(),
  refreshLLMSettings: vi.fn(),
  isAgentReady: false,
  isAgentInitializing: false,
  agentError: 'Repository is not indexed yet. Run analyze first.',
  chatMessages: [],
  isChatLoading: false,
  currentToolCalls: [],
  initializeAgent: vi.fn(),
  sendChatMessage: vi.fn(),
  stopChatResponse: vi.fn(),
  clearChat: vi.fn(),
  handleTranscriptLinkClick: vi.fn(),
};

const markdownRenderSpy = vi.fn();

vi.mock('../../src/hooks/chat-runtime/ChatRuntimeContext', () => ({
  useChatRuntime: () => mockChatRuntime,
}));

vi.mock('../../src/hooks/useAutoScroll', () => ({
  useAutoScroll: () => ({
    scrollContainerRef: { current: null },
    messagesContainerRef: { current: null },
    isAtBottom: true,
    scrollToBottom: vi.fn(),
  }),
}));

vi.mock('../../src/core/llm/settings-service-local-runtime', () => ({
  isLocalRuntimeConfigured: () => true,
}));

vi.mock('../../src/components/ToolCallCard', () => ({
  ToolCallCard: () => null,
}));

vi.mock('../../src/components/MarkdownRenderer', () => ({
  MarkdownRenderer: ({ content }: { content: string }) => {
    markdownRenderSpy(content);
    return <div>{content}</div>;
  },
}));

import { ChatPanel } from '../../src/components/ChatPanel';

describe('ChatPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockChatRuntime.isAgentReady = false;
    mockChatRuntime.isAgentInitializing = false;
    mockChatRuntime.agentError = 'Repository is not indexed yet. Run analyze first.';
    mockChatRuntime.chatMessages = [];
    mockChatRuntime.isChatLoading = false;
  });

  it('renders Analyze now CTA for index-required repos', () => {
    const onRequestAnalyze = vi.fn();
    render(<ChatPanel onRequestAnalyze={onRequestAnalyze} />);

    const button = screen.getByRole('button', { name: 'Analyze now' });
    fireEvent.click(button);
    expect(onRequestAnalyze).toHaveBeenCalledTimes(1);
  });

  it('does not rerender transcript markdown when typing in the composer', () => {
    mockChatRuntime.chatMessages = [
      {
        id: 'assistant-1',
        role: 'assistant',
        content: 'Architecture overview',
        timestamp: Date.now(),
      },
    ];
    mockChatRuntime.agentError = null;
    mockChatRuntime.isAgentReady = true;

    render(<ChatPanel onRequestAnalyze={vi.fn()} />);

    expect(markdownRenderSpy).toHaveBeenCalledTimes(1);

    fireEvent.change(screen.getByPlaceholderText('Ask about the codebase...'), {
      target: { value: 'hello world' },
    });

    expect(markdownRenderSpy).toHaveBeenCalledTimes(1);
  });
});
