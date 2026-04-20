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

// Legacy provider-based agent exports were retired from the active build path in
// Phase 6. The local session runtime bridge is the only supported web flow.
