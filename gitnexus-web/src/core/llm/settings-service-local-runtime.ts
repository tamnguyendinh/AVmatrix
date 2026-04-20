import type { LLMProvider, LLMSettings } from './types';
import { DEFAULT_LLM_SETTINGS } from './types';

const STORAGE_KEY = 'gitnexus-llm-settings';

export type LocalRuntimeProvider = 'codex';

export interface LocalRuntimeProviderConfig {
  provider: 'codex';
  model: string;
  temperature?: number;
  maxTokens?: number;
}

export type LocalRuntimeSettings = Omit<LLMSettings, 'activeProvider'> & {
  activeProvider: 'codex';
  codex?: {
    model?: string;
    temperature?: number;
    maxTokens?: number;
  };
};

const LOCAL_RUNTIME_DEFAULTS: LocalRuntimeSettings = {
  ...DEFAULT_LLM_SETTINGS,
  activeProvider: 'codex',
  codex: {
    model: 'codex-account',
    temperature: 0,
  },
};

const normalizeSettings = (parsed?: Partial<LocalRuntimeSettings> | null): LocalRuntimeSettings => ({
  ...LOCAL_RUNTIME_DEFAULTS,
  ...parsed,
  activeProvider: 'codex',
  codex: {
    ...LOCAL_RUNTIME_DEFAULTS.codex,
    ...parsed?.codex,
  },
  openai: {
    ...DEFAULT_LLM_SETTINGS.openai,
    ...parsed?.openai,
  },
  azureOpenAI: {
    ...DEFAULT_LLM_SETTINGS.azureOpenAI,
    ...parsed?.azureOpenAI,
  },
  gemini: {
    ...DEFAULT_LLM_SETTINGS.gemini,
    ...parsed?.gemini,
  },
  anthropic: {
    ...DEFAULT_LLM_SETTINGS.anthropic,
    ...parsed?.anthropic,
  },
  ollama: {
    ...DEFAULT_LLM_SETTINGS.ollama,
    ...parsed?.ollama,
  },
  openrouter: {
    ...DEFAULT_LLM_SETTINGS.openrouter,
    ...parsed?.openrouter,
  },
  minimax: {
    ...DEFAULT_LLM_SETTINGS.minimax,
    ...parsed?.minimax,
  },
  glm: {
    ...DEFAULT_LLM_SETTINGS.glm,
    ...parsed?.glm,
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

const readSettings = (storage: Storage): Partial<LocalRuntimeSettings> | null => {
  const raw = storage.getItem(STORAGE_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as Partial<LocalRuntimeSettings>;
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
  provider: LocalRuntimeProvider,
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

export const setLocalRuntimeProvider = (_provider: LocalRuntimeProvider): LocalRuntimeSettings => {
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
  switch (provider) {
    case 'codex':
      return 'Codex Account';
    case 'azure-openai':
      return 'Azure OpenAI';
    case 'gemini':
      return 'Google Gemini';
    case 'anthropic':
      return 'Anthropic';
    case 'ollama':
      return 'Ollama (Local)';
    case 'openrouter':
      return 'OpenRouter';
    case 'minimax':
      return 'MiniMax';
    case 'glm':
      return 'GLM (Z.AI)';
    case 'openai':
    default:
      return provider === 'openai' ? 'OpenAI' : provider;
  }
};

export const getLocalRuntimeAvailableModels = (
  provider: LocalRuntimeProvider | LLMProvider,
): string[] => {
  return provider === 'codex' ? ['codex-account'] : [];
};
