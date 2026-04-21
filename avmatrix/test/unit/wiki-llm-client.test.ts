import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const loadCLIConfigMock = vi.fn();

vi.mock('../../src/storage/repo-manager.js', () => ({
  loadCLIConfig: loadCLIConfigMock,
}));

describe('resolveLLMConfig', () => {
  beforeEach(() => {
    vi.resetModules();
    loadCLIConfigMock.mockReset();
    loadCLIConfigMock.mockResolvedValue({});
    delete process.env.AVMATRIX_API_KEY;
    delete process.env.AVMATRIX_LLM_BASE_URL;
    delete process.env.AVMATRIX_MODEL;
    delete process.env.AVMATRIX_AZURE_API_VERSION;
    delete process.env.OPENAI_API_KEY;
  });

  afterEach(() => {
    delete process.env.AVMATRIX_API_KEY;
    delete process.env.AVMATRIX_LLM_BASE_URL;
    delete process.env.AVMATRIX_MODEL;
    delete process.env.AVMATRIX_AZURE_API_VERSION;
    delete process.env.OPENAI_API_KEY;
  });

  it('reads AVmatrix env vars before saved config', async () => {
    loadCLIConfigMock.mockResolvedValue({
      apiKey: 'saved-key',
      baseUrl: 'https://saved.example/v1',
      model: 'saved-model',
      apiVersion: '2024-01-01',
      provider: 'custom',
    });

    process.env.AVMATRIX_API_KEY = 'env-key';
    process.env.AVMATRIX_LLM_BASE_URL = 'https://env.example/v1';
    process.env.AVMATRIX_MODEL = 'env-model';
    process.env.AVMATRIX_AZURE_API_VERSION = '2025-05-05';

    const { resolveLLMConfig } = await import('../../src/core/wiki/llm-client.js');
    const config = await resolveLLMConfig();

    expect(config.apiKey).toBe('env-key');
    expect(config.baseUrl).toBe('https://env.example/v1');
    expect(config.model).toBe('env-model');
    expect(config.apiVersion).toBe('2025-05-05');
    expect(config.provider).toBe('custom');
  });

  it('overrides AVmatrix env vars with explicit CLI options', async () => {
    process.env.AVMATRIX_API_KEY = 'env-key';
    process.env.AVMATRIX_LLM_BASE_URL = 'https://env.example/v1';
    process.env.AVMATRIX_MODEL = 'env-model';
    process.env.AVMATRIX_AZURE_API_VERSION = '2025-05-05';

    const { resolveLLMConfig } = await import('../../src/core/wiki/llm-client.js');
    const config = await resolveLLMConfig({
      apiKey: 'override-key',
      baseUrl: 'https://override.example/v1',
      model: 'override-model',
      apiVersion: '2026-06-06',
      provider: 'azure',
    });

    expect(config.apiKey).toBe('override-key');
    expect(config.baseUrl).toBe('https://override.example/v1');
    expect(config.model).toBe('override-model');
    expect(config.apiVersion).toBe('2026-06-06');
    expect(config.provider).toBe('azure');
  });

  it('falls back to OPENAI_API_KEY when AVMATRIX_API_KEY is unset', async () => {
    process.env.OPENAI_API_KEY = 'openai-key';

    const { resolveLLMConfig } = await import('../../src/core/wiki/llm-client.js');
    const config = await resolveLLMConfig();

    expect(config.apiKey).toBe('openai-key');
    expect(config.baseUrl).toBe('https://openrouter.ai/api/v1');
    expect(config.model).toBe('minimax/minimax-m2.5');
  });
});
