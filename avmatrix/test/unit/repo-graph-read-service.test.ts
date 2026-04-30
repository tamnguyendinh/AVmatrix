import { beforeEach, describe, expect, it, vi } from 'vitest';

const readExecutorMocks = vi.hoisted(() => ({
  executeRepoReadQuery: vi.fn(),
  streamRepoReadQuery: vi.fn(),
}));

vi.mock('../../src/runtime/repo-runtime/repo-read-executor.js', async (importOriginal) => {
  const actual = await importOriginal();
  return { ...actual, ...readExecutorMocks };
});

import {
  buildRepoGraph,
  streamRepoGraph,
} from '../../src/runtime/repo-runtime/graph-read-service.js';

describe('repo-graph-read-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('builds graph through the repo-scoped pool executor', async () => {
    readExecutorMocks.executeRepoReadQuery.mockImplementation(
      async (target: any, query: string) => {
        expect(target).toEqual({
          repoId: 'demo',
          lbugPath: 'F:/repos/demo/.avmatrix/lbug',
        });
        if (query.includes('MATCH (n:`File`)')) {
          return [{ id: 'File:src/app.ts', name: 'app.ts', filePath: 'src/app.ts' }];
        }
        if (query.includes('CodeRelation')) {
          return [
            {
              sourceId: 'File:src/app.ts',
              targetId: 'Function:src/app.ts:main',
              type: 'CONTAINS',
            },
          ];
        }
        return [];
      },
    );

    const graph = await buildRepoGraph({
      repoId: 'demo',
      lbugPath: 'F:/repos/demo/.avmatrix/lbug',
    });

    expect(graph).toEqual({
      nodes: [
        {
          id: 'File:src/app.ts',
          label: 'File',
          properties: {
            name: 'app.ts',
            filePath: 'src/app.ts',
            startLine: undefined,
            endLine: undefined,
            content: undefined,
            responseKeys: undefined,
            errorKeys: undefined,
            middleware: undefined,
            heuristicLabel: undefined,
            cohesion: undefined,
            symbolCount: undefined,
            description: undefined,
            processType: undefined,
            stepCount: undefined,
            communities: undefined,
            entryPointId: undefined,
            terminalId: undefined,
          },
        },
      ],
      relationships: [
        {
          id: 'File:src/app.ts_CONTAINS_Function:src/app.ts:main',
          type: 'CONTAINS',
          sourceId: 'File:src/app.ts',
          targetId: 'Function:src/app.ts:main',
          confidence: undefined,
          reason: undefined,
          step: undefined,
        },
      ],
    });
  });

  it('streams graph records through the repo-scoped pool executor', async () => {
    readExecutorMocks.streamRepoReadQuery.mockImplementation(
      async (target: any, query: string, onRow: (row: any) => Promise<void>) => {
        expect(target).toEqual({
          repoId: 'demo',
          lbugPath: 'F:/repos/demo/.avmatrix/lbug',
        });
        if (query.includes('MATCH (n:`File`)')) {
          await onRow({ id: 'File:src/app.ts', name: 'app.ts', filePath: 'src/app.ts' });
          return 1;
        }
        if (query.includes('CodeRelation')) {
          await onRow({
            sourceId: 'File:src/app.ts',
            targetId: 'Function:src/app.ts:main',
            type: 'CONTAINS',
          });
          return 1;
        }
        return 0;
      },
    );

    const writes: any[] = [];
    let flushCount = 0;
    await streamRepoGraph(
      {
        repoId: 'demo',
        lbugPath: 'F:/repos/demo/.avmatrix/lbug',
      },
      {
        write: async (record) => {
          writes.push(record);
        },
        flush: async () => {
          flushCount += 1;
        },
      },
    );

    expect(writes).toEqual([
      {
        type: 'node',
        data: {
          id: 'File:src/app.ts',
          label: 'File',
          properties: {
            name: 'app.ts',
            filePath: 'src/app.ts',
            startLine: undefined,
            endLine: undefined,
            content: undefined,
            responseKeys: undefined,
            errorKeys: undefined,
            middleware: undefined,
            heuristicLabel: undefined,
            cohesion: undefined,
            symbolCount: undefined,
            description: undefined,
            processType: undefined,
            stepCount: undefined,
            communities: undefined,
            entryPointId: undefined,
            terminalId: undefined,
          },
        },
      },
      {
        type: 'relationship',
        data: {
          id: 'File:src/app.ts_CONTAINS_Function:src/app.ts:main',
          type: 'CONTAINS',
          sourceId: 'File:src/app.ts',
          targetId: 'Function:src/app.ts:main',
          confidence: undefined,
          reason: undefined,
          step: undefined,
        },
      },
    ]);
    expect(flushCount).toBeGreaterThan(0);
  });
});
