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

    await queryCommand('auth flow', { repo: 'AVmatrix', content: true });
    await contextCommand('validateUser', { repo: 'AVmatrix', uid: 'sym-1' });

    expect(initMock).toHaveBeenCalledTimes(1);
    expect(callToolMock).toHaveBeenNthCalledWith(1, 'query', {
      query: 'auth flow',
      task_context: undefined,
      goal: undefined,
      limit: undefined,
      include_content: true,
      repo: 'AVmatrix',
    });
    expect(callToolMock).toHaveBeenNthCalledWith(2, 'context', {
      name: 'validateUser',
      uid: 'sym-1',
      file_path: undefined,
      include_content: false,
      repo: 'AVmatrix',
    });
    expect(writeSyncMock).toHaveBeenCalledTimes(2);
  });

  it('routes impact and cypher through the same backend contract', async () => {
    const { impactCommand, cypherCommand } = await import('../../src/cli/tool.js');

    await impactCommand('AuthService', {
      direction: 'upstream',
      repo: 'AVmatrix',
      depth: '2',
      includeTests: true,
    });
    await cypherCommand('MATCH (n) RETURN n LIMIT 1', { repo: 'AVmatrix' });

    expect(initMock).toHaveBeenCalledTimes(1);
    expect(callToolMock).toHaveBeenNthCalledWith(1, 'impact', {
      target: 'AuthService',
      target_uid: undefined,
      direction: 'upstream',
      maxDepth: 2,
      relationTypes: ['CALLS', 'IMPORTS', 'EXTENDS', 'IMPLEMENTS', 'METHOD_OVERRIDES', 'OVERRIDES', 'METHOD_IMPLEMENTS'],
      includeTests: true,
      minConfidence: 0,
      repo: 'AVmatrix',
    });
    expect(callToolMock).toHaveBeenNthCalledWith(2, 'cypher', {
      query: 'MATCH (n) RETURN n LIMIT 1',
      repo: 'AVmatrix',
    });
  });

  it('supports uid-only impact calls in the direct CLI surface', async () => {
    const { impactCommand } = await import('../../src/cli/tool.js');

    await impactCommand(undefined, {
      uid: 'uid:AuthService',
      direction: 'upstream',
      repo: 'AVmatrix',
    });

    expect(callToolMock).toHaveBeenCalledWith('impact', {
      target: undefined,
      target_uid: 'uid:AuthService',
      direction: 'upstream',
      maxDepth: 3,
      relationTypes: ['CALLS', 'IMPORTS', 'EXTENDS', 'IMPLEMENTS', 'METHOD_OVERRIDES', 'OVERRIDES', 'METHOD_IMPLEMENTS'],
      includeTests: false,
      minConfidence: 0,
      repo: 'AVmatrix',
    });
  });

  it('routes detect-changes through the same backend contract', async () => {
    const { detectChangesCommand } = await import('../../src/cli/tool.js');

    await detectChangesCommand({
      scope: 'compare',
      baseRef: 'main',
      repo: 'AVmatrix',
    });

    expect(initMock).toHaveBeenCalledTimes(1);
    expect(callToolMock).toHaveBeenCalledWith('detect_changes', {
      scope: 'compare',
      base_ref: 'main',
      repo: 'AVmatrix',
    });
    expect(writeSyncMock).toHaveBeenCalledTimes(1);
  });
});
