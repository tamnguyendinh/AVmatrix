import { describe, expect, it } from 'vitest';
import {
  DEFAULT_LLM_SETTINGS,
  type LLMSettings,
  type ProviderConfig,
  GRAPH_SCHEMA_DESCRIPTION,
} from '../../src/core/llm/types.local-runtime';

type HasLegacyProviderPayloads = 'openai' extends keyof LLMSettings ? true : false;
type ProviderConfigIsCodexOnly = ProviderConfig['provider'] extends 'codex' ? true : false;

const hasLegacyProviderPayloads: HasLegacyProviderPayloads = false;
const providerConfigIsCodexOnly: ProviderConfigIsCodexOnly = true;

describe('types.local-runtime', () => {
  it('pins default settings to codex-only storage', () => {
    expect(DEFAULT_LLM_SETTINGS).toEqual({
      activeProvider: 'codex',
      intelligentClustering: false,
      hasSeenClusteringPrompt: false,
      useSameModelForClustering: true,
      codex: {
        model: 'codex-account',
        temperature: 0,
      },
    });
  });

  it('removes remote provider payloads from the stored settings shape', () => {
    expect(hasLegacyProviderPayloads).toBe(false);
    expect(providerConfigIsCodexOnly).toBe(true);
  });

  it('keeps a compatibility graph schema export with local-runtime wording', () => {
    expect(GRAPH_SCHEMA_DESCRIPTION).toMatch(/local session runtime bridge/i);
  });
});
