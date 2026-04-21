import { describe, expect, it } from 'vitest';

describe('retired legacy LLM companion modules', () => {
  it('fails fast when legacy agent helpers are imported directly', async () => {
    const { createChatModel, createGraphRAGAgent, invokeAgent } = await import('../../src/core/llm/agent');

    expect(() => createChatModel({})).toThrow(/local session runtime bridge/i);
    expect(() => createGraphRAGAgent({}, {} as never)).toThrow(/local session runtime bridge/i);
    await expect(invokeAgent({}, [])).rejects.toThrow(/local session runtime bridge/i);
  });

  it('fails fast when the retired graph rag tool factory is used directly', async () => {
    const { createGraphRAGTools } = await import('../../src/core/llm/tools');

    expect(() =>
      createGraphRAGTools({
        executeQuery: async () => [],
        search: async () => [],
        grep: async () => [],
        readFile: async () => '',
      }),
    ).toThrow(/local session runtime bridge/i);
  });

  it('fails fast when the retired prompt context builder is used directly', async () => {
    const { buildDynamicSystemPrompt } = await import('../../src/core/llm/context-builder');

    expect(() =>
      buildDynamicSystemPrompt('base prompt', {
        stats: {
          projectName: 'AVmatrix',
          fileCount: 1,
          functionCount: 1,
          classCount: 0,
          interfaceCount: 0,
          methodCount: 0,
        },
        hotspots: [],
        folderTree: 'src/',
      }),
    ).toThrow(/local session runtime bridge/i);
  });
});
