import { beforeEach, describe, expect, it, vi } from 'vitest';

const initMock = vi.fn();
const callToolMock = vi.fn();
const writeSyncMock = vi.fn();

vi.mock('node:fs', async () => {
  const actual = await vi.importActual<typeof import('node:fs')>('node:fs');
  return {
    ...actual,
    writeSync: (...args: unknown[]) => writeSyncMock(...args),
  };
});

vi.mock('../../src/mcp/local/local-backend.js', () => ({
  LocalBackend: class LocalBackend {
    init = initMock;
    callTool = callToolMock;
  },
}));

describe('direct tool commands reuse the shared backend/runtime core', () => {
  beforeEach(() => {
    initMock.mockReset().mockResolvedValue(true);
    callToolMock.mockReset().mockResolvedValue({ ok: true });
    writeSyncMock.mockReset();
    vi.resetModules();
  });

  it('reuses one LocalBackend instance across multiple direct tool commands', async () => {
    const { queryCommand, contextCommand } = await import('../../src/cli/tool.js');

    await queryCommand('auth flow', { repo: 'GitNexus', content: true });
    await contextCommand('validateUser', { repo: 'GitNexus', uid: 'sym-1' });

    expect(initMock).toHaveBeenCalledTimes(1);
    expect(callToolMock).toHaveBeenNthCalledWith(1, 'query', {
      query: 'auth flow',
      task_context: undefined,
      goal: undefined,
      limit: undefined,
      include_content: true,
      repo: 'GitNexus',
    });
    expect(callToolMock).toHaveBeenNthCalledWith(2, 'context', {
      name: 'validateUser',
      uid: 'sym-1',
      file_path: undefined,
      include_content: false,
      repo: 'GitNexus',
    });
    expect(writeSyncMock).toHaveBeenCalledTimes(2);
  });

  it('routes impact and cypher through the same backend contract', async () => {
    const { impactCommand, cypherCommand } = await import('../../src/cli/tool.js');

    await impactCommand('AuthService', {
      direction: 'upstream',
      repo: 'GitNexus',
      depth: '2',
      includeTests: true,
    });
    await cypherCommand('MATCH (n) RETURN n LIMIT 1', { repo: 'GitNexus' });

    expect(initMock).toHaveBeenCalledTimes(1);
    expect(callToolMock).toHaveBeenNthCalledWith(1, 'impact', {
      target: 'AuthService',
      direction: 'upstream',
      maxDepth: 2,
      includeTests: true,
      repo: 'GitNexus',
    });
    expect(callToolMock).toHaveBeenNthCalledWith(2, 'cypher', {
      query: 'MATCH (n) RETURN n LIMIT 1',
      repo: 'GitNexus',
    });
  });
});
