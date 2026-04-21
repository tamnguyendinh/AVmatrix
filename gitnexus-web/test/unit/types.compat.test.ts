import { describe, expect, it } from 'vitest';
import {
  DEFAULT_LLM_SETTINGS,
  type LLMSettings,
  type ProviderConfig,
  GRAPH_SCHEMA_DESCRIPTION,
} from '../../src/core/llm/types';

type HasLegacyProviderPayloads = 'openai' extends keyof LLMSettings ? true : false;
type ProviderConfigIsCodexOnly = ProviderConfig['provider'] extends 'codex' ? true : false;

const hasLegacyProviderPayloads: HasLegacyProviderPayloads = false;
const providerConfigIsCodexOnly: ProviderConfigIsCodexOnly = true;

describe('types compatibility wrapper', () => {
  it('preserves the legacy import path while exporting codex-only defaults', () => {
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

  it('keeps the compatibility type surface codex-only for active settings', () => {
    expect(hasLegacyProviderPayloads).toBe(false);
    expect(providerConfigIsCodexOnly).toBe(true);
  });

  it('exports the local-runtime graph schema wording through the old path', () => {
    expect(GRAPH_SCHEMA_DESCRIPTION).toMatch(/local session runtime bridge/i);
  });
});
