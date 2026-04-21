import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { GraphNode } from 'avmatrix-shared';
import { createKnowledgeGraph } from '../../src/core/graph/graph';
import { AppStateProvider, useAppState } from '../../src/hooks/useAppState.local-runtime';
import type { ChatRuntimeState } from '../../src/hooks/chat-runtime/types';

let transcriptLinkHandler: (href: string) => void = vi.fn();

const mockChatRuntime: ChatRuntimeState = {
  llmSettings: {
    activeProvider: 'codex',
    intelligentClustering: false,
    hasSeenClusteringPrompt: false,
    useSameModelForClustering: true,
    codex: { model: 'codex-account', temperature: 0 },
  },
  updateLLMSettings: vi.fn(),
  refreshLLMSettings: vi.fn(),
  isAgentReady: true,
  isAgentInitializing: false,
  agentError: null as string | null,
  chatMessages: [] as Array<{
    id: string;
    role: 'assistant' | 'user';
    content: string;
    timestamp: number;
  }>,
  isChatLoading: false,
  currentToolCalls: [],
  initializeAgent: vi.fn(),
  sendChatMessage: vi.fn(),
  stopChatResponse: vi.fn(),
  clearChat: vi.fn(),
  handleTranscriptLinkClick: (href: string) => transcriptLinkHandler(href),
};

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

import { ChatPanel } from '../../src/components/ChatPanel';

let appState: ReturnType<typeof useAppState> | null = null;

function createFileNode(id: string, filePath: string): GraphNode {
  return {
    id,
    label: 'File',
    properties: {
      name: filePath.split('/').pop() ?? filePath,
      filePath,
    },
  };
}

function createFunctionNode(
  id: string,
  name: string,
  filePath: string,
  startLine: number,
  endLine: number,
): GraphNode {
  return {
    id,
    label: 'Function',
    properties: {
      name,
      filePath,
      startLine,
      endLine,
    },
  };
}

function ChatPanelHarness() {
  const state = useAppState();
  appState = state;
  transcriptLinkHandler = state.chatRuntimeBridge.handleTranscriptLinkClick;
  return <ChatPanel onRequestAnalyze={vi.fn()} />;
}

describe('ChatPanel grounding links', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    transcriptLinkHandler = vi.fn();
    appState = null;
    mockChatRuntime.isAgentReady = true;
    mockChatRuntime.isAgentInitializing = false;
    mockChatRuntime.agentError = null;
    mockChatRuntime.chatMessages = [];
    mockChatRuntime.isChatLoading = false;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('routes file grounding clicks into app-state code references', async () => {
    mockChatRuntime.chatMessages = [
      {
        id: 'assistant-1',
        role: 'assistant',
        content: 'Inspect [[src/foo.ts:4-6]]',
        timestamp: Date.now(),
      },
    ];

    render(
      <AppStateProvider>
        <ChatPanelHarness />
      </AppStateProvider>,
    );

    await waitFor(() => expect(appState).not.toBeNull());

    const graph = createKnowledgeGraph();
    graph.addNode(createFileNode('File:src/foo.ts', 'src/foo.ts'));

    await act(async () => {
      appState!.setGraph(graph);
    });

    fireEvent.click(screen.getByRole('link', { name: 'src/foo.ts:4-6' }));

    await waitFor(() => expect(appState!.codeReferences).toHaveLength(1));
    expect(appState!.isCodePanelOpen).toBe(true);
    expect(appState!.codeReferences[0]).toEqual(
      expect.objectContaining({
        filePath: 'src/foo.ts',
        startLine: 3,
        endLine: 5,
        label: 'File',
        source: 'ai',
        nodeId: 'File:src/foo.ts',
      }),
    );
  });

  it('routes node grounding clicks into app-state code references', async () => {
    mockChatRuntime.chatMessages = [
      {
        id: 'assistant-2',
        role: 'assistant',
        content: 'Inspect [[Function:loadFoo]]',
        timestamp: Date.now(),
      },
    ];

    render(
      <AppStateProvider>
        <ChatPanelHarness />
      </AppStateProvider>,
    );

    await waitFor(() => expect(appState).not.toBeNull());

    const graph = createKnowledgeGraph();
    graph.addNode(createFileNode('File:src/foo.ts', 'src/foo.ts'));
    graph.addNode(
      createFunctionNode('Function:src/foo.ts:loadFoo', 'loadFoo', 'src/foo.ts', 10, 20),
    );

    await act(async () => {
      appState!.setGraph(graph);
    });

    fireEvent.click(screen.getByRole('link', { name: 'Function:loadFoo' }));

    await waitFor(() => expect(appState!.codeReferences).toHaveLength(1));
    expect(appState!.isCodePanelOpen).toBe(true);
    expect(appState!.codeReferences[0]).toEqual(
      expect.objectContaining({
        filePath: 'src/foo.ts',
        startLine: 9,
        endLine: 19,
        label: 'Function',
        name: 'loadFoo',
        source: 'ai',
        nodeId: 'Function:src/foo.ts:loadFoo',
      }),
    );
  });
});
