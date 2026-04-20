import { describe, expect, it } from 'vitest';
import pkg from '../../package.json' with { type: 'json' };

describe('gitnexus-web package contract', () => {
  it('does not pull provider-based LangChain or hosted runtime packages into the active app', () => {
    const deps = { ...pkg.dependencies, ...pkg.devDependencies };

    expect(deps['@langchain/anthropic']).toBeUndefined();
    expect(deps['@langchain/core']).toBeUndefined();
    expect(deps['@langchain/google-genai']).toBeUndefined();
    expect(deps['@langchain/langgraph']).toBeUndefined();
    expect(deps['@langchain/ollama']).toBeUndefined();
    expect(deps['@langchain/openai']).toBeUndefined();
    expect(deps.langchain).toBeUndefined();
    expect(deps['@vercel/node']).toBeUndefined();
  });
});
