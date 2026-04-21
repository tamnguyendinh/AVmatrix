import { describe, expect, it } from 'vitest';
import {
  loadSettings,
  saveSettings,
  setActiveProvider,
  getActiveProviderConfig,
  isProviderConfigured,
  clearSettings,
  getProviderDisplayName,
  getAvailableModels,
} from '../../src/core/llm/settings-service';

describe('loadSettings', () => {
  it('returns defaults when nothing is stored', () => {
    const settings = loadSettings();
    expect(settings.activeProvider).toBe('codex');
    expect(settings.codex).toBeDefined();
  });

  it('normalizes stored values back to the local runtime provider', () => {
    sessionStorage.setItem(
      'avmatrix-llm-settings',
      JSON.stringify({
        activeProvider: 'ollama',
        codex: { model: 'codex-account', maxTokens: 4096 },
      }),
    );

    const settings = loadSettings();
    expect(settings.activeProvider).toBe('codex');
    expect(settings.codex?.maxTokens).toBe(4096);
  });

  it('returns defaults on corrupted JSON', () => {
    sessionStorage.setItem('avmatrix-llm-settings', 'not-json{{{');
    const settings = loadSettings();
    expect(settings.activeProvider).toBeDefined();
  });

  it('migrates legacy localStorage to sessionStorage', () => {
    localStorage.setItem(
      'avmatrix-llm-settings',
      JSON.stringify({
        activeProvider: 'openai',
        codex: { model: 'codex-account', temperature: 0.2 },
      }),
    );

    const settings = loadSettings();
    expect(settings.activeProvider).toBe('codex');
    expect(settings.codex?.temperature).toBe(0.2);
    expect(sessionStorage.getItem('avmatrix-llm-settings')).not.toBeNull();
    expect(localStorage.getItem('avmatrix-llm-settings')).toBeNull();
  });
});

describe('saveSettings / clearSettings', () => {
  it('persists settings to sessionStorage', () => {
    const settings = loadSettings();
    settings.activeProvider = 'codex';
    settings.codex = { ...settings.codex, maxTokens: 8192 };
    saveSettings(settings);
    expect(loadSettings().activeProvider).toBe('codex');
    expect(loadSettings().codex?.maxTokens).toBe(8192);
  });

  it('clearSettings removes settings from both storages', () => {
    saveSettings({ ...loadSettings(), activeProvider: 'codex' });
    expect(sessionStorage.getItem('avmatrix-llm-settings')).not.toBeNull();
    clearSettings();
    expect(sessionStorage.getItem('avmatrix-llm-settings')).toBeNull();
    expect(localStorage.getItem('avmatrix-llm-settings')).toBeNull();
  });
});

describe('setActiveProvider', () => {
  it('pins the compatibility API to the local runtime provider', () => {
    setActiveProvider('gemini');
    expect(loadSettings().activeProvider).toBe('codex');
  });
});

describe('getActiveProviderConfig', () => {
  it('always returns the local runtime provider config', () => {
    setActiveProvider('openai');
    const config = getActiveProviderConfig();
    expect(config).not.toBeNull();
    expect(config!.provider).toBe('codex');
  });
});

describe('isProviderConfigured', () => {
  it('returns true for the local session runtime', () => {
    setActiveProvider('openai');
    expect(isProviderConfigured()).toBe(true);
  });
});

describe('getProviderDisplayName', () => {
  it('returns local-runtime labels and compatibility names', () => {
    expect(getProviderDisplayName('codex')).toBe('Codex Account');
    expect(getProviderDisplayName('openai')).toBe('Retired provider');
  });
});

describe('getAvailableModels', () => {
  it('returns models for the local runtime provider only', () => {
    expect(getAvailableModels('codex')).toEqual(['codex-account']);
  });

  it('returns empty arrays for retired remote provider paths', () => {
    expect(getAvailableModels('openai')).toEqual([]);
    expect(getAvailableModels('anthropic')).toEqual([]);
  });
});
