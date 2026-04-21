/**
 * Compatibility wrapper for the legacy local-runtime settings import path.
 *
 * The active implementation now lives in the Phase 6 local-runtime settings
 * module, which normalizes storage to the Codex-first session model.
 */

export * from './settings-service-local-runtime.phase6';
