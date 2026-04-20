/**
 * LLM Module Exports
 *
 * The active product path is the local session runtime bridge.
 * Compatibility shims stay only for settings/app-state import paths that the
 * active product still needs while provider-based modules are retired.
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

// Provider-based agent/context/tool modules are retired from the active build
// path in Phase 6. The local session runtime bridge is the only supported web
// flow.
