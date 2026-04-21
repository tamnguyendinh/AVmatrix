import { fireEvent, render, screen } from '@testing-library/react';
import type { ChatMessage } from '../../src/core/llm/types.local-runtime';
import { describe, expect, it, vi } from 'vitest';

const requestRepoAnalyzeDialog = vi.fn();
const markdownRenderSpy = vi.fn();
const sendChatMessage = vi.fn();

const mockAppState = {
  isRightPanelOpen: true,
  setRightPanelOpen: vi.fn(),
  graph: null,
  addCodeReference: vi.fn(),
  chatMessages: [] as ChatMessage[],
  isChatLoading: false,
  currentToolCalls: [],
  agentError: 'Repository is not indexed yet. Run analyze first.',
  isAgentReady: false,
  isAgentInitializing: false,
  sendChatMessage,
  stopChatResponse: vi.fn(),
  clearChat: vi.fn(),
  requestRepoAnalyzeDialog,
};

vi.mock('../../src/hooks/useAppState.local-runtime', () => ({
  useAppState: () => mockAppState,
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

vi.mock('../../src/components/ProcessesPanel', () => ({
  ProcessesPanel: () => <div>Processes</div>,
}));

import { RightPanel } from '../../src/components/RightPanel';

describe('RightPanel.local-runtime', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAppState.chatMessages = [];
    mockAppState.agentError = 'Repository is not indexed yet. Run analyze first.';
    mockAppState.isAgentReady = false;
    mockAppState.isAgentInitializing = false;
    mockAppState.isChatLoading = false;
  });

  it('shows an Analyze now CTA when the repo is not indexed', () => {
    render(<RightPanel />);

    const button = screen.getByRole('button', { name: 'Analyze now' });
    expect(button).toBeInTheDocument();

    fireEvent.click(button);
    expect(requestRepoAnalyzeDialog).toHaveBeenCalledTimes(1);
  });

  it('does not rerender the transcript markdown when typing in the composer', () => {
    mockAppState.chatMessages = [
      {
        id: 'assistant-1',
        role: 'assistant',
        content: 'Architecture overview',
        timestamp: Date.now(),
      },
    ];
    mockAppState.agentError = null;
    mockAppState.isAgentReady = true;

    render(<RightPanel />);

    expect(markdownRenderSpy).toHaveBeenCalledTimes(1);

    fireEvent.change(screen.getByPlaceholderText('Ask about the codebase...'), {
      target: { value: 'hello world' },
    });

    expect(markdownRenderSpy).toHaveBeenCalledTimes(1);
  });
});
