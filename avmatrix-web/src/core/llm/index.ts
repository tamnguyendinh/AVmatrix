/**
 * LLM Module Exports
 *
 * The active product path is the local session runtime bridge.
 */

// Types
export * from './types.local-runtime';

// Active local-runtime settings + session bridge
export {
  loadLocalRuntimeSettings,
  saveLocalRuntimeSettings,
  updateLocalRuntimeProviderSettings,
  setLocalRuntimeProvider,
  getLocalRuntimeProviderConfig,
  isLocalRuntimeConfigured,
  clearLocalRuntimeSettings,
  getLocalRuntimeProviderDisplayName,
  getLocalRuntimeAvailableModels,
} from './settings-service-local-runtime';
export {
  SessionClientError,
  fetchSessionStatus,
  cancelSession,
  streamSessionChat,
  toAgentStreamChunk,
} from './session-client';
