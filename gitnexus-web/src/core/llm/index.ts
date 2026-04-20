/**
 * LLM Module Exports
 *
 * The active product path is the local session runtime bridge.
 * Legacy agent/context/tool exports remain available here as compatibility
 * surfaces while provider-based modules are retired behind the scenes.
 */

// Types
export * from './types';

// Active local-runtime settings + session bridge
export {
  loadSettings,
  saveSettings,
  updateProviderSettings,
  setActiveProvider,
  getActiveProviderConfig,
  isProviderConfigured,
  clearSettings,
  getProviderDisplayName,
  getAvailableModels,
} from './settings-service';
export {
  SessionClientError,
  fetchSessionStatus,
  cancelSession,
  streamSessionChat,
  toAgentStreamChunk,
} from './session-client';

// Legacy compatibility exports
export { createGraphRAGTools } from './tools';
export {
  buildCodebaseContext,
  formatContextForPrompt,
  buildDynamicSystemPrompt,
  type CodebaseContext,
  type CodebaseStats,
  type Hotspot,
} from './context-builder';
export {
  createChatModel,
  createGraphRAGAgent,
  streamAgentResponse,
  invokeAgent,
  BASE_SYSTEM_PROMPT,
  type AgentMessage,
} from './agent';
