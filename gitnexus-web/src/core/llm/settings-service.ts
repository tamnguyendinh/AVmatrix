/**
 * Compatibility wrapper for the legacy settings-service import path.
 *
 * Existing imports keep working, but the implementation is now backed by the
 * local session runtime compatibility shim.
 */

export * from './settings-service.compat-local-runtime';
