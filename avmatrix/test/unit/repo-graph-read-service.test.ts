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

  it('reads scope-resolution audit metadata from relationship rows', async () => {
    readExecutorMocks.executeRepoReadQuery.mockImplementation(
      async (_target: any, query: string) => {
        if (query.includes('MATCH (n:`File`)')) return [];
        if (query.includes('CodeRelation')) {
          expect(query).toContain('r.resolutionSource AS resolutionSource');
          expect(query).toContain('r.evidence AS evidence');
          expect(query).toContain('r.fileHash AS fileHash');
          return [
            {
              sourceId: 'Function:run',
              targetId: 'Method:save',
              type: 'CALLS',
              confidence: 0.95,
              reason: 'scope-resolution: call | confidence 0.950',
              step: 0,
              resolutionSource: 'scope-resolution',
              evidence: JSON.stringify([
                { kind: 'type-binding', weight: 0.35, note: 'receiver User' },
              ]),
              fileHash: 'sha256:abc',
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

    expect(graph.relationships).toEqual([
      {
        id: 'Function:run_CALLS_Method:save',
        type: 'CALLS',
        sourceId: 'Function:run',
        targetId: 'Method:save',
        confidence: 0.95,
        reason: 'scope-resolution: call | confidence 0.950',
        step: 0,
        resolutionSource: 'scope-resolution',
        fileHash: 'sha256:abc',
        evidence: [{ kind: 'type-binding', weight: 0.35, note: 'receiver User' }],
      },
    ]);
  });

  it('falls back to legacy relationship reads when audit metadata properties are absent', async () => {
    readExecutorMocks.executeRepoReadQuery.mockImplementation(
      async (_target: any, query: string) => {
        if (query.includes('MATCH (n:`File`)')) return [];
        if (query.includes('r.resolutionSource AS resolutionSource')) {
          throw new Error('Binder exception: Cannot find property resolutionSource for r.');
        }
        if (query.includes('CodeRelation')) {
          expect(query).not.toContain('r.resolutionSource AS resolutionSource');
          return [
            {
              sourceId: 'File:src/app.ts',
              targetId: 'Function:src/app.ts:main',
              type: 'CONTAINS',
              confidence: 1,
              reason: 'contains',
              step: 0,
            },
          ];
        }
        return [];
      },
    );

    const graph = await buildRepoGraph({
      repoId: 'legacy',
      lbugPath: 'F:/repos/legacy/.avmatrix/lbug',
    });

    expect(graph.relationships).toEqual([
      {
        id: 'File:src/app.ts_CONTAINS_Function:src/app.ts:main',
        type: 'CONTAINS',
        sourceId: 'File:src/app.ts',
        targetId: 'Function:src/app.ts:main',
        confidence: 1,
        reason: 'contains',
        step: 0,
      },
    ]);
  });
});
