import type { ChatMessage, ToolCallInfo } from '../../core/llm/types.local-runtime';
import type { LocalRuntimeSettings } from '../../core/llm/settings-service-local-runtime';
import type { EmbeddingStatus } from '../useAppState.local-runtime';

export interface ChatRuntimeBridge {
  getRepoName: () => string | undefined;
  getEmbeddingStatus: () => EmbeddingStatus;
  clearAICodeReferences: () => void;
  clearAIToolHighlights: () => void;
  handleContentGrounding: (content: string) => void;
  handleToolResult: (toolResult: string) => void;
  handleTranscriptLinkClick: (href: string) => void;
}

export interface ChatRuntimeState {
  llmSettings: LocalRuntimeSettings;
  updateLLMSettings: (updates: Partial<LocalRuntimeSettings>) => void;
  refreshLLMSettings: () => void;
  isAgentReady: boolean;
  isAgentInitializing: boolean;
  agentError: string | null;
  chatMessages: ChatMessage[];
  isChatLoading: boolean;
  currentToolCalls: ToolCallInfo[];
  initializeAgent: (overrideProjectName?: string) => Promise<boolean>;
  sendChatMessage: (message: string) => Promise<void>;
  stopChatResponse: () => void;
  clearChat: () => void;
  handleTranscriptLinkClick: (href: string) => void;
}
