/**
 * Compatibility wrapper for the retired provider-based app state module.
 *
 * The active product path lives in `useAppState.local-runtime.tsx`. This file
 * preserves the original import path while forwarding all exports to the shared
 * local-runtime implementation.
 */

export * from './useAppState.local-runtime';
