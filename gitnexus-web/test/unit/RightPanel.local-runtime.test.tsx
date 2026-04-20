import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

const requestRepoAnalyzeDialog = vi.fn();

vi.mock('../../src/hooks/useAppState.local-runtime', () => ({
  useAppState: () => ({
    isRightPanelOpen: true,
    setRightPanelOpen: vi.fn(),
    graph: null,
    addCodeReference: vi.fn(),
    chatMessages: [],
    isChatLoading: false,
    currentToolCalls: [],
    agentError: 'Repository is not indexed yet. Run analyze first.',
    isAgentReady: false,
    isAgentInitializing: false,
    sendChatMessage: vi.fn(),
    stopChatResponse: vi.fn(),
    clearChat: vi.fn(),
    requestRepoAnalyzeDialog,
  }),
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
  MarkdownRenderer: ({ content }: { content: string }) => <div>{content}</div>,
}));

vi.mock('../../src/components/ProcessesPanel', () => ({
  ProcessesPanel: () => <div>Processes</div>,
}));

import { RightPanel } from '../../src/components/RightPanel';

describe('RightPanel.local-runtime', () => {
  it('shows an Analyze now CTA when the repo is not indexed', () => {
    render(<RightPanel />);

    const button = screen.getByRole('button', { name: 'Analyze now' });
    expect(button).toBeInTheDocument();

    fireEvent.click(button);
    expect(requestRepoAnalyzeDialog).toHaveBeenCalledTimes(1);
  });
});
