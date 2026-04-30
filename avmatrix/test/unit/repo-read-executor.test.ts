import { beforeEach, describe, expect, it, vi } from 'vitest';

const poolMocks = vi.hoisted(() => ({
  initLbug: vi.fn(),
  touchRepo: vi.fn(),
  executeQuery: vi.fn(),
  executeParameterized: vi.fn(),
  streamQuery: vi.fn(),
}));

vi.mock('../../src/core/lbug/pool-adapter.js', async (importOriginal) => {
  const actual = await importOriginal();
  return { ...actual, ...poolMocks };
});

import {
  executeRepoParameterizedReadQuery,
  executeRepoReadQuery,
  streamRepoReadQuery,
} from '../../src/runtime/repo-runtime/repo-read-executor.js';

describe('repo-read-executor', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('initializes and executes a repo-scoped read query', async () => {
    poolMocks.executeQuery.mockResolvedValue([{ id: 'File:src/app.ts' }]);

    const rows = await executeRepoReadQuery(
      {
        repoId: 'demo',
        lbugPath: 'F:/repos/demo/.avmatrix/lbug',
      },
      'MATCH (n:`File`) RETURN n.id AS id',
    );

    expect(poolMocks.initLbug).toHaveBeenCalledWith('demo', 'F:/repos/demo/.avmatrix/lbug');
    expect(poolMocks.touchRepo).toHaveBeenCalledWith('demo');
    expect(poolMocks.executeQuery).toHaveBeenCalledWith(
      'demo',
      'MATCH (n:`File`) RETURN n.id AS id',
    );
    expect(rows).toEqual([{ id: 'File:src/app.ts' }]);
  });

  it('initializes and streams a repo-scoped read query', async () => {
    const seenRows: any[] = [];
    poolMocks.streamQuery.mockImplementation(
      async (_repoId: string, _query: string, onRow: (row: any) => Promise<void>) => {
        await onRow({ id: 'File:src/app.ts' });
        return 1;
      },
    );

    const count = await streamRepoReadQuery(
      {
        repoId: 'demo',
        lbugPath: 'F:/repos/demo/.avmatrix/lbug',
      },
      'MATCH (n:`File`) RETURN n.id AS id',
      async (row) => {
        seenRows.push(row);
      },
    );

    expect(poolMocks.initLbug).toHaveBeenCalledWith('demo', 'F:/repos/demo/.avmatrix/lbug');
    expect(poolMocks.touchRepo).toHaveBeenCalledWith('demo');
    expect(poolMocks.streamQuery).toHaveBeenCalledWith(
      'demo',
      'MATCH (n:`File`) RETURN n.id AS id',
      expect.any(Function),
    );
    expect(seenRows).toEqual([{ id: 'File:src/app.ts' }]);
    expect(count).toBe(1);
  });

  it('initializes and executes a repo-scoped parameterized read query', async () => {
    poolMocks.executeParameterized.mockResolvedValue([{ id: 'Function:src/app.ts:main' }]);

    const rows = await executeRepoParameterizedReadQuery(
      {
        repoId: 'demo',
        lbugPath: 'F:/repos/demo/.avmatrix/lbug',
      },
      'MATCH (n:Function {id: $id}) RETURN n.id AS id',
      { id: 'Function:src/app.ts:main' },
    );

    expect(poolMocks.initLbug).toHaveBeenCalledWith('demo', 'F:/repos/demo/.avmatrix/lbug');
    expect(poolMocks.touchRepo).toHaveBeenCalledWith('demo');
    expect(poolMocks.executeParameterized).toHaveBeenCalledWith(
      'demo',
      'MATCH (n:Function {id: $id}) RETURN n.id AS id',
      { id: 'Function:src/app.ts:main' },
    );
    expect(rows).toEqual([{ id: 'Function:src/app.ts:main' }]);
  });
});
