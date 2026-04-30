import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import type {
  ChatMessage,
  MessageStep,
  ToolCallInfo,
  AgentStreamChunk,
} from '../../core/llm/types';
import {
  loadLocalRuntimeSettings,
  saveLocalRuntimeSettings,
  type LocalRuntimeSettings,
} from '../../core/llm/settings-service-local-runtime';
import {
  SessionClientError,
  cancelSession,
  fetchSessionStatus,
  streamSessionChat,
  toAgentStreamChunk,
} from '../../core/llm/session-client';
import type { ChatRuntimeBridge, ChatRuntimeState } from './types';

const ChatRuntimeContext = createContext<ChatRuntimeState | null>(null);

interface ChatRuntimeProviderProps {
  bridge: ChatRuntimeBridge;
  children: ReactNode;
}

export const ChatRuntimeProvider = ({ bridge, children }: ChatRuntimeProviderProps) => {
  const [llmSettings, setLLMSettings] = useState<LocalRuntimeSettings>(loadLocalRuntimeSettings);
  const [isAgentReady, setIsAgentReady] = useState(false);
  const [isAgentInitializing, setIsAgentInitializing] = useState(false);
  const [agentError, setAgentError] = useState<string | null>(null);
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([]);
  const [isChatLoading, setIsChatLoading] = useState(false);
  const [currentToolCalls, setCurrentToolCalls] = useState<ToolCallInfo[]>([]);

  const activeSessionAbortRef = useRef<AbortController | null>(null);
  const activeSessionIdRef = useRef<string | null>(null);
  const currentRepoName = bridge.getRepoName();
  const previousRepoNameRef = useRef<string | undefined>(currentRepoName);

  useEffect(() => {
    if (previousRepoNameRef.current === currentRepoName) return;
    previousRepoNameRef.current = currentRepoName;

    const abortController = activeSessionAbortRef.current;
    const sessionId = activeSessionIdRef.current;
    activeSessionAbortRef.current = null;
    activeSessionIdRef.current = null;

    if (abortController) {
      abortController.abort('Repository changed');
    }

    if (sessionId) {
      void cancelSession(sessionId).catch(() => {
        // Runtime may already have cancelled or disconnected.
      });
    }

    setChatMessages([]);
    setCurrentToolCalls([]);
    setIsChatLoading(false);
    setIsAgentReady(false);
    setAgentError(null);
  }, [currentRepoName]);

  const updateLLMSettings = useCallback((updates: Partial<LocalRuntimeSettings>) => {
    setLLMSettings((prev) => {
      const next: LocalRuntimeSettings = {
        ...prev,
        ...updates,
        activeProvider: 'codex',
        codex: {
          ...(prev.codex ?? {}),
          ...(updates.codex ?? {}),
        },
      };
      saveLocalRuntimeSettings(next);
      return next;
    });
  }, []);

  const refreshLLMSettings = useCallback(() => {
    setLLMSettings(loadLocalRuntimeSettings());
  }, []);

  const cancelActiveSession = useCallback(async (reason = 'Cancelled by user'): Promise<void> => {
    const abortController = activeSessionAbortRef.current;
    const sessionId = activeSessionIdRef.current;

    activeSessionAbortRef.current = null;
    activeSessionIdRef.current = null;

    if (abortController) {
      abortController.abort(reason);
    }

    if (sessionId) {
      try {
        await cancelSession(sessionId);
      } catch {
        // Runtime may already have cancelled on disconnect.
      }
    }
  }, []);

  const initializeAgent = useCallback(
    async (overrideProjectName?: string): Promise<boolean> => {
      setIsAgentInitializing(true);
      setAgentError(null);

      try {
        const repoName = overrideProjectName || bridge.getRepoName();
        if (!repoName) {
          setIsAgentReady(false);
          setAgentError('Connect to a local repository before starting a session');
          return false;
        }

        const status = await fetchSessionStatus({ repoName });
        const repoState = status.repo?.state;

        if (repoState === 'not_found' || repoState === 'invalid') {
          setIsAgentReady(false);
          setAgentError(status.repo?.message || 'The selected repository is no longer available');
          return false;
        }

        if (repoState === 'index_required') {
          setIsAgentReady(false);
          setAgentError(
            status.repo?.message || 'Repository is not indexed yet. Run analyze first.',
          );
          return false;
        }

        if (status.availability !== 'ready') {
          setIsAgentReady(false);
          setAgentError(status.message || 'Local Codex session is not ready');
          return false;
        }

        setIsAgentReady(true);
        setAgentError(null);
        return true;
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        setAgentError(message);
        setIsAgentReady(false);
        return false;
      } finally {
        setIsAgentInitializing(false);
      }
    },
    [bridge],
  );

  const sendChatMessage = useCallback(
    async (message: string): Promise<void> => {
      bridge.clearAICodeReferences();
      bridge.clearAIToolHighlights();

      const repoName = bridge.getRepoName();
      if (!repoName) {
        setAgentError('Connect to a local repository before starting a session');
        return;
      }

      if (!isAgentReady) {
        const ready = await initializeAgent(repoName);
        if (!ready) return;
      }

      const userMessage: ChatMessage = {
        id: `user-${Date.now()}`,
        role: 'user',
        content: message,
        timestamp: Date.now(),
      };
      setChatMessages((prev) => [...prev, userMessage]);

      if (bridge.getEmbeddingStatus() === 'indexing') {
        const assistantMessage: ChatMessage = {
          id: `assistant-${Date.now()}`,
          role: 'assistant',
          content: 'Wait a moment, vector index is being created.',
          timestamp: Date.now(),
        };
        setChatMessages((prev) => [...prev, assistantMessage]);
        setAgentError(null);
        setIsChatLoading(false);
        setCurrentToolCalls([]);
        return;
      }

      setIsChatLoading(true);
      setCurrentToolCalls([]);

      const assistantMessageId = `assistant-${Date.now()}`;
      const stepsForMessage: MessageStep[] = [];
      const toolCallsForMessage: ToolCallInfo[] = [];
      let stepCounter = 0;

      const updateMessage = () => {
        const contentParts = stepsForMessage
          .filter((s) => s.type === 'reasoning' || s.type === 'content')
          .map((s) => s.content)
          .filter(Boolean);
        const content = contentParts.join('\n\n');

        setChatMessages((prev) => {
          const existing = prev.find((m) => m.id === assistantMessageId);
          const newMessage: ChatMessage = {
            id: assistantMessageId,
            role: 'assistant',
            content,
            steps: [...stepsForMessage],
            toolCalls: [...toolCallsForMessage],
            timestamp: existing?.timestamp ?? Date.now(),
          };
          if (existing) {
            return prev.map((m) => (m.id === assistantMessageId ? newMessage : m));
          }
          return [...prev, newMessage];
        });
      };

      let pendingUpdate = false;
      const scheduleMessageUpdate = () => {
        if (pendingUpdate) return;
        pendingUpdate = true;
        requestAnimationFrame(() => {
          pendingUpdate = false;
          updateMessage();
        });
      };

      try {
        const onChunk = (chunk: AgentStreamChunk) => {
          switch (chunk.type) {
            case 'reasoning':
              if (chunk.reasoning) {
                const lastStep = stepsForMessage[stepsForMessage.length - 1];
                if (lastStep && lastStep.type === 'reasoning') {
                  stepsForMessage[stepsForMessage.length - 1] = {
                    ...lastStep,
                    content: (lastStep.content || '') + chunk.reasoning,
                  };
                } else {
                  stepsForMessage.push({
                    id: `step-${stepCounter++}`,
                    type: 'reasoning',
                    content: chunk.reasoning,
                  });
                }
                scheduleMessageUpdate();
              }
              break;

            case 'content':
              if (chunk.content) {
                const normalizedIncoming = chunk.content.trim();
                const lastStep = stepsForMessage[stepsForMessage.length - 1];
                if (lastStep && lastStep.type === 'content') {
                  stepsForMessage[stepsForMessage.length - 1] = {
                    ...lastStep,
                    content: (lastStep.content || '') + chunk.content,
                  };
                } else if (
                  lastStep &&
                  lastStep.type === 'reasoning' &&
                  (lastStep.content || '').trim() === normalizedIncoming
                ) {
                  stepsForMessage[stepsForMessage.length - 1] = {
                    ...lastStep,
                    type: 'content',
                    content: chunk.content,
                  };
                } else {
                  stepsForMessage.push({
                    id: `step-${stepCounter++}`,
                    type: 'content',
                    content: chunk.content,
                  });
                }
                scheduleMessageUpdate();

                const currentContentStep = stepsForMessage[stepsForMessage.length - 1];
                const fullText =
                  currentContentStep && currentContentStep.type === 'content'
                    ? currentContentStep.content || ''
                    : '';
                bridge.handleContentGrounding(fullText);
              }
              break;

            case 'tool_call':
              if (chunk.toolCall) {
                const tc = chunk.toolCall;
                toolCallsForMessage.push(tc);
                stepsForMessage.push({
                  id: `step-${stepCounter++}`,
                  type: 'tool_call',
                  toolCall: tc,
                });
                setCurrentToolCalls((prev) => [...prev, tc]);
                scheduleMessageUpdate();
              }
              break;

            case 'tool_result':
              if (chunk.toolCall) {
                const tc = chunk.toolCall;
                let idx = toolCallsForMessage.findIndex((t) => t.id === tc.id);
                if (idx < 0) {
                  idx = toolCallsForMessage.findIndex(
                    (t) => t.name === tc.name && t.status === 'running',
                  );
                }
                if (idx < 0) {
                  idx = toolCallsForMessage.findIndex((t) => t.name === tc.name && !t.result);
                }
                if (idx >= 0) {
                  toolCallsForMessage[idx] = {
                    ...toolCallsForMessage[idx],
                    result: tc.result,
                    status: 'completed',
                  };
                }

                const stepIdx = stepsForMessage.findIndex(
                  (s) =>
                    s.type === 'tool_call' &&
                    s.toolCall &&
                    (s.toolCall.id === tc.id ||
                      (s.toolCall.name === tc.name && s.toolCall.status === 'running')),
                );
                if (stepIdx >= 0 && stepsForMessage[stepIdx].toolCall) {
                  stepsForMessage[stepIdx] = {
                    ...stepsForMessage[stepIdx],
                    toolCall: {
                      ...stepsForMessage[stepIdx].toolCall!,
                      result: tc.result,
                      status: 'completed',
                    },
                  };
                }

                setCurrentToolCalls((prev) => {
                  let targetIdx = prev.findIndex((t) => t.id === tc.id);
                  if (targetIdx < 0) {
                    targetIdx = prev.findIndex((t) => t.name === tc.name && t.status === 'running');
                  }
                  if (targetIdx < 0) {
                    targetIdx = prev.findIndex((t) => t.name === tc.name && !t.result);
                  }
                  if (targetIdx >= 0) {
                    return prev.map((t, i) =>
                      i === targetIdx ? { ...t, result: tc.result, status: 'completed' } : t,
                    );
                  }
                  return prev;
                });

                scheduleMessageUpdate();

                if (tc.result) {
                  bridge.handleToolResult(tc.result);
                }
              }
              break;

            case 'error':
              setAgentError(chunk.error ?? 'Unknown error');
              break;

            case 'done':
              scheduleMessageUpdate();
              break;
          }
        };

        const abortController = new AbortController();
        activeSessionAbortRef.current = abortController;
        activeSessionIdRef.current = null;

        for await (const event of streamSessionChat(
          {
            repoName,
            message,
          },
          abortController.signal,
        )) {
          if (event.type === 'session_started') {
            activeSessionIdRef.current = event.sessionId;
            continue;
          }

          const chunk = toAgentStreamChunk(event);
          if (chunk) {
            onChunk(chunk);
          }
        }
      } catch (error) {
        if (error instanceof DOMException && error.name === 'AbortError') {
          return;
        }

        if (error instanceof SessionClientError) {
          setAgentError(error.message);
          if (
            error.code === 'SESSION_RUNTIME_UNAVAILABLE' ||
            error.code === 'SESSION_NOT_SIGNED_IN' ||
            error.code === 'REPO_NOT_FOUND' ||
            error.code === 'INDEX_REQUIRED'
          ) {
            setIsAgentReady(false);
          }
          return;
        }

        const fallbackMessage = error instanceof Error ? error.message : String(error);
        setAgentError(fallbackMessage);
      } finally {
        activeSessionAbortRef.current = null;
        activeSessionIdRef.current = null;
        setIsChatLoading(false);
        setCurrentToolCalls([]);
      }
    },
    [bridge, initializeAgent, isAgentReady],
  );

  const stopChatResponse = useCallback(() => {
    if (isChatLoading) {
      void cancelActiveSession('Cancelled by user');
      setIsChatLoading(false);
      setCurrentToolCalls([]);
    }
  }, [cancelActiveSession, isChatLoading]);

  const clearChat = useCallback(() => {
    void cancelActiveSession('Chat cleared by user');
    setChatMessages([]);
    setCurrentToolCalls([]);
    setAgentError(null);
  }, [cancelActiveSession]);

  const value = useMemo<ChatRuntimeState>(
    () => ({
      llmSettings,
      updateLLMSettings,
      refreshLLMSettings,
      isAgentReady,
      isAgentInitializing,
      agentError,
      chatMessages,
      isChatLoading,
      currentToolCalls,
      initializeAgent,
      sendChatMessage,
      stopChatResponse,
      clearChat,
      handleTranscriptLinkClick: bridge.handleTranscriptLinkClick,
    }),
    [
      agentError,
      bridge,
      chatMessages,
      clearChat,
      currentToolCalls,
      initializeAgent,
      isAgentInitializing,
      isAgentReady,
      isChatLoading,
      llmSettings,
      refreshLLMSettings,
      sendChatMessage,
      stopChatResponse,
      updateLLMSettings,
    ],
  );

  return <ChatRuntimeContext.Provider value={value}>{children}</ChatRuntimeContext.Provider>;
};

export const useChatRuntime = (): ChatRuntimeState => {
  const context = useContext(ChatRuntimeContext);
  if (!context) {
    throw new Error('useChatRuntime must be used within ChatRuntimeProvider');
  }
  return context;
};
