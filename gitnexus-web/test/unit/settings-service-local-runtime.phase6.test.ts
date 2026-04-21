import { describe, expect, it } from 'vitest';
import {
  clearLocalRuntimeSettings,
  getLocalRuntimeAvailableModels,
  getLocalRuntimeProviderConfig,
  getLocalRuntimeProviderDisplayName,
  isLocalRuntimeConfigured,
  loadLocalRuntimeSettings,
  saveLocalRuntimeSettings,
  setLocalRuntimeProvider,
} from '../../src/core/llm/settings-service-local-runtime.phase6';

describe('settings-service-local-runtime.phase6', () => {
  it('defaults to codex when nothing is stored', () => {
    const settings = loadLocalRuntimeSettings();
    expect(settings.activeProvider).toBe('codex');
    expect(settings.codex?.model).toBe('codex-account');
  });

  it('migrates legacy provider payloads into codex-only local runtime settings', () => {
    localStorage.setItem(
      'gitnexus-llm-settings',
      JSON.stringify({
        activeProvider: 'gemini',
        gemini: { apiKey: 'legacy-key', model: 'gemini-2.0-flash' },
        codex: { model: 'codex-account', maxTokens: 8192 },
        intelligentClustering: true,
      }),
    );

    const settings = loadLocalRuntimeSettings();

    expect(settings.activeProvider).toBe('codex');
    expect(settings.codex?.maxTokens).toBe(8192);
    expect('gemini' in settings).toBe(false);
    expect(settings.intelligentClustering).toBe(true);
    expect(sessionStorage.getItem('gitnexus-llm-settings')).not.toBeNull();
    expect(localStorage.getItem('gitnexus-llm-settings')).toBeNull();
  });

  it('always resolves the runtime provider config to codex', () => {
    saveLocalRuntimeSettings({
      ...loadLocalRuntimeSettings(),
      activeProvider: 'codex',
      codex: {
        model: 'codex-account',
        temperature: 0,
      },
    });

    expect(getLocalRuntimeProviderConfig()).toEqual({
      provider: 'codex',
      model: 'codex-account',
      temperature: 0,
      maxTokens: undefined,
    });
  });

  it('forces active provider back to codex when a legacy provider is selected', () => {
    const settings = setLocalRuntimeProvider('openai');
    expect(settings.activeProvider).toBe('codex');
    expect(loadLocalRuntimeSettings().activeProvider).toBe('codex');
  });

  it('reports local runtime as configured and clears both storages', () => {
    expect(isLocalRuntimeConfigured()).toBe(true);

    saveLocalRuntimeSettings(loadLocalRuntimeSettings());
    expect(sessionStorage.getItem('gitnexus-llm-settings')).not.toBeNull();

    clearLocalRuntimeSettings();
    expect(sessionStorage.getItem('gitnexus-llm-settings')).toBeNull();
    expect(localStorage.getItem('gitnexus-llm-settings')).toBeNull();
  });

  it('returns codex-first labels and available models', () => {
    expect(getLocalRuntimeProviderDisplayName('codex')).toBe('Codex Account');
    expect(getLocalRuntimeAvailableModels('codex')).toEqual(['codex-account']);
    expect(getLocalRuntimeAvailableModels('gemini')).toEqual([]);
    expect(getLocalRuntimeProviderDisplayName('gemini')).toBe('Retired provider');
  });
});
