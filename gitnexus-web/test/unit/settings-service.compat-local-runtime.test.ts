import { describe, expect, it } from 'vitest';
import {
  clearSettings,
  getActiveProviderConfig,
  getAvailableModels,
  getProviderDisplayName,
  isProviderConfigured,
  loadSettings,
  saveSettings,
  setActiveProvider,
} from '../../src/core/llm/settings-service.compat-local-runtime';

describe('settings-service.compat-local-runtime', () => {
  it('returns codex defaults from the compatibility surface', () => {
    const settings = loadSettings();
    expect(settings.activeProvider).toBe('codex');
    expect(settings.codex).toBeDefined();
  });

  it('persists codex settings and ignores legacy provider selection', () => {
    const settings = loadSettings();
    saveSettings({
      ...settings,
      activeProvider: 'codex',
      codex: { ...settings.codex, maxTokens: 4096 },
    });

    expect(loadSettings().codex?.maxTokens).toBe(4096);

    setActiveProvider('gemini');
    expect(loadSettings().activeProvider).toBe('codex');
  });

  it('returns codex-only compatibility metadata', () => {
    expect(getActiveProviderConfig()).toEqual({
      provider: 'codex',
      model: 'codex-account',
      temperature: 0,
      maxTokens: undefined,
    });
    expect(isProviderConfigured()).toBe(true);
    expect(getProviderDisplayName('codex')).toBe('Codex Account');
    expect(getProviderDisplayName('openai')).toBe('Retired provider');
    expect(getAvailableModels('codex')).toEqual(['codex-account']);
    expect(getAvailableModels('anthropic')).toEqual([]);

    clearSettings();
  });
});
