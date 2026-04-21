/**
 * Companion compatibility shim for legacy imports that still reference
 * `settings-service`.
 *
 * The product now runs through the local session runtime. This module preserves
 * the old function names so existing code can be swapped over safely after the
 * companion passes parity tests.
 */

import type {
  LLMProvider,
  LLMSettings,
  ProviderConfig,
} from './types.local-runtime';
import {
  clearLocalRuntimeSettings,
  getLocalRuntimeAvailableModels,
  getLocalRuntimeProviderConfig,
  getLocalRuntimeProviderDisplayName,
  isLocalRuntimeConfigured,
  loadLocalRuntimeSettings,
  saveLocalRuntimeSettings,
  setLocalRuntimeProvider,
  updateLocalRuntimeProviderSettings,
  type LocalRuntimeProviderConfig,
  type LocalRuntimeSettings,
} from './settings-service-local-runtime.phase6';

const toLocalRuntimeSettings = (settings?: Partial<LLMSettings> | null): LocalRuntimeSettings => {
  const current = loadLocalRuntimeSettings();
  return {
    ...current,
    ...settings,
    activeProvider: 'codex',
    codex: {
      ...current.codex,
      ...settings?.codex,
    },
  };
};

export const loadSettings = (): LLMSettings => {
  return loadLocalRuntimeSettings();
};

export const saveSettings = (settings: LLMSettings): void => {
  saveLocalRuntimeSettings(toLocalRuntimeSettings(settings));
};

export const updateProviderSettings = (
  provider: LLMProvider,
  updates: Record<string, unknown>,
): LLMSettings => {
  if (provider === 'codex') {
    return updateLocalRuntimeProviderSettings('codex', updates);
  }

  const current = loadLocalRuntimeSettings();
  const next: LocalRuntimeSettings = {
    ...current,
    activeProvider: 'codex',
  };
  saveLocalRuntimeSettings(next);
  return next;
};

export const setActiveProvider = (_provider: LLMProvider): LLMSettings => {
  return setLocalRuntimeProvider('codex');
};

export const getActiveProviderConfig = (): ProviderConfig | null => {
  return getLocalRuntimeProviderConfig() as LocalRuntimeProviderConfig as ProviderConfig;
};

export const isProviderConfigured = (): boolean => {
  return isLocalRuntimeConfigured();
};

export const clearSettings = (): void => {
  clearLocalRuntimeSettings();
};

export const getProviderDisplayName = (provider: LLMProvider): string => {
  return getLocalRuntimeProviderDisplayName(provider);
};

export const getAvailableModels = (provider: LLMProvider): string[] => {
  return getLocalRuntimeAvailableModels(provider);
};

export const fetchOpenRouterModels = async (): Promise<Array<{ id: string; name: string }>> => {
  return [];
};
