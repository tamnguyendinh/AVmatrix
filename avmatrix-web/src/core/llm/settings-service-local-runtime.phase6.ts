import type { LLMProvider, LLMSettings } from './types.local-runtime';
import { DEFAULT_LLM_SETTINGS } from './types.local-runtime';

const STORAGE_KEY = 'avmatrix-llm-settings';

export type LocalRuntimeProvider = 'codex';

export interface LocalRuntimeProviderConfig {
  provider: 'codex';
  model: string;
  temperature?: number;
  maxTokens?: number;
}

export type LocalRuntimeSettings = LLMSettings;

const LOCAL_RUNTIME_DEFAULTS: LocalRuntimeSettings = {
  ...DEFAULT_LLM_SETTINGS,
};

type RawSettingsShape = Partial<LocalRuntimeSettings> & {
  activeProvider?: string;
  codex?: Partial<NonNullable<LocalRuntimeSettings['codex']>>;
};

const normalizeSettings = (parsed?: RawSettingsShape | null): LocalRuntimeSettings => ({
  ...LOCAL_RUNTIME_DEFAULTS,
  activeProvider: 'codex',
  codex: {
    ...LOCAL_RUNTIME_DEFAULTS.codex,
    ...parsed?.codex,
  },
  intelligentClustering:
    typeof parsed?.intelligentClustering === 'boolean'
      ? parsed.intelligentClustering
      : DEFAULT_LLM_SETTINGS.intelligentClustering,
  hasSeenClusteringPrompt:
    typeof parsed?.hasSeenClusteringPrompt === 'boolean'
      ? parsed.hasSeenClusteringPrompt
      : DEFAULT_LLM_SETTINGS.hasSeenClusteringPrompt,
  useSameModelForClustering:
    typeof parsed?.useSameModelForClustering === 'boolean'
      ? parsed.useSameModelForClustering
      : LOCAL_RUNTIME_DEFAULTS.useSameModelForClustering,
});

const readSettings = (storage: Storage): RawSettingsShape | null => {
  const raw = storage.getItem(STORAGE_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as RawSettingsShape;
  } catch (error) {
    console.warn('Failed to parse local runtime settings:', error);
    return null;
  }
};

const writeSettings = (storage: Storage, settings: LocalRuntimeSettings): void => {
  storage.setItem(STORAGE_KEY, JSON.stringify(settings));
};

export const loadLocalRuntimeSettings = (): LocalRuntimeSettings => {
  try {
    const sessionData = typeof sessionStorage !== 'undefined' ? readSettings(sessionStorage) : null;
    if (sessionData) {
      return normalizeSettings(sessionData);
    }

    const legacyData = typeof localStorage !== 'undefined' ? readSettings(localStorage) : null;
    if (legacyData) {
      const migrated = normalizeSettings(legacyData);
      try {
        if (typeof sessionStorage !== 'undefined') {
          writeSettings(sessionStorage, migrated);
        }
        if (typeof localStorage !== 'undefined') {
          localStorage.removeItem(STORAGE_KEY);
        }
      } catch (error) {
        console.warn('Failed to migrate legacy settings to sessionStorage:', error);
      }
      return migrated;
    }

    return LOCAL_RUNTIME_DEFAULTS;
  } catch (error) {
    console.warn('Failed to load local runtime settings:', error);
    return LOCAL_RUNTIME_DEFAULTS;
  }
};

export const saveLocalRuntimeSettings = (settings: LocalRuntimeSettings): void => {
  try {
    if (typeof sessionStorage !== 'undefined') {
      writeSettings(sessionStorage, normalizeSettings(settings));
    }
  } catch (error) {
    console.error('Failed to save local runtime settings:', error);
  }
};

export const updateLocalRuntimeProviderSettings = (
  _provider: LocalRuntimeProvider,
  updates: Record<string, unknown>,
): LocalRuntimeSettings => {
  const current = loadLocalRuntimeSettings();
  const updated: LocalRuntimeSettings = {
    ...current,
    activeProvider: 'codex',
    codex: {
      ...(current.codex ?? {}),
      ...updates,
    },
  };
  saveLocalRuntimeSettings(updated);
  return updated;
};

export const setLocalRuntimeProvider = (
  _provider: LocalRuntimeProvider | LLMProvider,
): LocalRuntimeSettings => {
  const current = loadLocalRuntimeSettings();
  const updated: LocalRuntimeSettings = {
    ...current,
    activeProvider: 'codex',
  };
  saveLocalRuntimeSettings(updated);
  return updated;
};

export const getLocalRuntimeProviderConfig = (): LocalRuntimeProviderConfig => {
  const settings = loadLocalRuntimeSettings();
  return {
    provider: 'codex',
    model: settings.codex?.model || LOCAL_RUNTIME_DEFAULTS.codex?.model || 'codex-account',
    temperature: settings.codex?.temperature,
    maxTokens: settings.codex?.maxTokens,
  };
};

export const isLocalRuntimeConfigured = (): boolean => true;

export const clearLocalRuntimeSettings = (): void => {
  try {
    if (typeof sessionStorage !== 'undefined') {
      sessionStorage.removeItem(STORAGE_KEY);
    }
    if (typeof localStorage !== 'undefined') {
      localStorage.removeItem(STORAGE_KEY);
    }
  } catch (error) {
    console.warn('Failed to clear local runtime settings:', error);
  }
};

export const getLocalRuntimeProviderDisplayName = (
  provider: LocalRuntimeProvider | LLMProvider,
): string => {
  return provider === 'codex' ? 'Codex Account' : 'Retired provider';
};

export const getLocalRuntimeAvailableModels = (
  provider: LocalRuntimeProvider | LLMProvider,
): string[] => {
  return provider === 'codex' ? ['codex-account'] : [];
};
