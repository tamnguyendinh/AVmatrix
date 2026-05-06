import { describe, expect, it } from 'vitest';
import {
  compareAnalyzeBenchmarkSnapshots,
  createAnalyzeBenchmarkSnapshot,
} from '../../src/core/analyze/analyze-benchmark-snapshot.js';
import { buildTestGraph } from '../helpers/test-graph.js';

describe('analyze benchmark snapshot', () => {
  it('captures graph correctness and key performance metrics for benchmark comparisons', () => {
    const graph = buildTestGraph(
      [
        {
          id: 'Function:src/app.ts:run',
          label: 'Function',
          name: 'run',
          filePath: 'src/app.ts',
        },
        {
          id: 'Method:src/model.ts:save',
          label: 'Method',
          name: 'save',
          filePath: 'src/model.ts',
        },
      ],
      [
        {
          sourceId: 'Function:src/app.ts:run',
          targetId: 'Method:src/model.ts:save',
          type: 'CALLS',
          confidence: 0.95,
          reason: 'scope-resolution: call | confidence 0.950',
        },
      ],
    );

    const snapshot = createAnalyzeBenchmarkSnapshot({
      repoName: 'demo',
      repoPath: 'F:/demo',
      createdAt: '2026-05-06T00:00:00.000Z',
      label: 'baseline',
      environment: {
        avmatrixVersion: '1.2.1-test',
        nodeVersion: 'v20.0.0',
        platform: 'win32',
        arch: 'x64',
        repoGitCommit: 'abcdef123',
        repoGitDirty: false,
      },
      stats: { files: 2, nodes: 2, edges: 1 },
      pipelineResult: {
        graph,
        repoPath: 'F:/demo',
        totalFileCount: 2,
        usedWorkerPool: true,
        performance: {
          phaseMs: { parse: 10, crossFile: 20, resolution: 5 },
          counters: { crossFileReprocessedFiles: 3 },
        },
      },
      performance: {
        totalWallMs: 100,
        buckets: { lbugLoad: 30 },
        pipelinePhaseMs: { parse: 10, crossFile: 20, resolution: 5 },
        ftsIndexMs: {},
        counters: {
          parseableFiles: 2,
          scopeCount: 4,
          scopeLocalDefs: 2,
          scopeParsedImports: 1,
          scopeReferenceSites: 4,
          scopeExtractionNoHookFiles: 0,
          scopeExtractionFailedFiles: 0,
          scopeFinalizedFiles: 2,
          scopeFinalizeTotalImports: 1,
          scopeFinalizeLinkedImports: 1,
          scopeFinalizeUnresolvedImports: 0,
          scopeResolutionReferenceSites: 4,
          scopeResolutionChunkSize: 128,
          scopeResolutionChunks: 1,
          scopeResolutionMaxChunkReferenceSites: 4,
          scopeResolutionReadonlyIndexBytes: 2048,
          scopeResolutionReferenceIndexSourceScopes: 2,
          scopeResolutionReferenceIndexTargetDefs: 2,
          scopeResolutionResolvedReferences: 3,
          scopeResolutionUnresolvedReferences: 1,
          scopeResolutionResolvedCalls: 2,
          scopeResolutionResolvedAccesses: 1,
          scopeResolutionResolvedTypeReferences: 0,
          scopeResolutionResolvedInheritance: 0,
          scopeResolutionResolvedImportUses: 0,
          scopeResolutionEdgesEmitted: 3,
          scopeResolutionDuplicateEdgesSkipped: 1,
          scopeResolutionFinalizedImportsEmitted: 1,
          scopeResolutionDuplicateImportsSkipped: 0,
          scopeResolutionFinalizedImportUsesEmitted: 1,
          scopeResolutionDuplicateImportUsesSkipped: 0,
          scopeResolutionEdgesSkippedNoCaller: 0,
          scopeResolutionEdgesSkippedMissingTarget: 0,
          crossFileReprocessedFiles: 3,
          csvRelationshipRows: 12,
          ladybugCopyCount: 3,
        },
        bottlenecks: [],
        overheadMs: 35,
        crossFile: {
          timings: {
            readContentsMs: 7,
            processCallsParserParseMs: 9,
          },
          counters: {},
        },
      },
    });

    expect(snapshot.graph?.byRelationshipType).toEqual({ CALLS: 1 });
    expect(snapshot.environment).toEqual({
      avmatrixVersion: '1.2.1-test',
      nodeVersion: 'v20.0.0',
      platform: 'win32',
      arch: 'x64',
      repoGitCommit: 'abcdef123',
      repoGitDirty: false,
    });
    expect(snapshot.keyMetrics).toMatchObject({
      totalWallMs: 100,
      nodeCount: 2,
      relationshipCount: 1,
      nodeCountsByLabel: { Function: 1, Method: 1 },
      relationshipCountsByType: { CALLS: 1 },
      parseMs: 10,
      crossFileMs: 20,
      resolutionMs: 5,
      lbugLoadMs: 30,
      parseableFiles: 2,
      scopeCount: 4,
      scopeLocalDefs: 2,
      scopeParsedImports: 1,
      scopeReferenceSites: 4,
      scopeExtractionNoHookFiles: 0,
      scopeExtractionFailedFiles: 0,
      scopeFinalizedFiles: 2,
      scopeFinalizeTotalImports: 1,
      scopeFinalizeLinkedImports: 1,
      scopeFinalizeUnresolvedImports: 0,
      scopeResolutionChunkSize: 128,
      scopeResolutionChunks: 1,
      scopeResolutionMaxChunkReferenceSites: 4,
      scopeResolutionReadonlyIndexBytes: 2048,
      scopeResolutionReferenceIndexSourceScopes: 2,
      scopeResolutionReferenceIndexTargetDefs: 2,
      scopeResolutionResolvedReferences: 3,
      scopeResolutionUnresolvedReferences: 1,
      scopeResolutionResolvedCalls: 2,
      scopeResolutionResolvedAccesses: 1,
      scopeResolutionResolvedTypeReferences: 0,
      scopeResolutionResolvedInheritance: 0,
      scopeResolutionResolvedImportUses: 0,
      scopeResolutionEdgesEmitted: 3,
      scopeResolutionDuplicateEdgesSkipped: 1,
      scopeResolutionFinalizedImportsEmitted: 1,
      scopeResolutionDuplicateImportsSkipped: 0,
      scopeResolutionFinalizedImportUsesEmitted: 1,
      scopeResolutionDuplicateImportUsesSkipped: 0,
      scopeResolutionEdgesSkippedNoCaller: 0,
      scopeResolutionEdgesSkippedMissingTarget: 0,
      crossFileReprocessedFiles: 3,
      crossFileReadContentsMs: 7,
      crossFileProcessCallsParserParseMs: 9,
      csvRelationshipRows: 12,
      ladybugCopyCount: 3,
    });
  });

  it('compares before/after benchmark artifacts for timing and parity deltas', () => {
    const beforeGraph = buildTestGraph(
      [
        {
          id: 'Function:src/app.ts:run',
          label: 'Function',
          name: 'run',
          filePath: 'src/app.ts',
        },
      ],
      [],
    );
    const afterGraph = buildTestGraph(
      [
        {
          id: 'Function:src/app.ts:run',
          label: 'Function',
          name: 'run',
          filePath: 'src/app.ts',
        },
        {
          id: 'Method:src/model.ts:save',
          label: 'Method',
          name: 'save',
          filePath: 'src/model.ts',
        },
      ],
      [
        {
          sourceId: 'Function:src/app.ts:run',
          targetId: 'Method:src/model.ts:save',
          type: 'CALLS',
          confidence: 0.95,
          reason: 'scope-resolution: call | confidence 0.950',
        },
      ],
    );

    const before = createAnalyzeBenchmarkSnapshot({
      repoName: 'demo',
      repoPath: 'F:/demo',
      createdAt: '2026-05-06T00:00:00.000Z',
      label: 'before',
      stats: { files: 1, nodes: 1, edges: 0 },
      pipelineResult: {
        graph: beforeGraph,
        repoPath: 'F:/demo',
        totalFileCount: 1,
        usedWorkerPool: true,
      },
      performance: {
        totalWallMs: 200,
        buckets: {},
        pipelinePhaseMs: { parse: 100, resolution: 50 },
        ftsIndexMs: {},
        counters: {
          scopeResolutionResolvedReferences: 0,
          scopeResolutionUnresolvedReferences: 2,
          scopeResolutionFinalizedImportsEmitted: 0,
          scopeResolutionFinalizedImportUsesEmitted: 0,
        },
        bottlenecks: [],
        overheadMs: 50,
      },
    });
    const after = createAnalyzeBenchmarkSnapshot({
      repoName: 'demo',
      repoPath: 'F:/demo',
      createdAt: '2026-05-06T00:01:00.000Z',
      label: 'after',
      stats: { files: 1, nodes: 2, edges: 1 },
      pipelineResult: {
        graph: afterGraph,
        repoPath: 'F:/demo',
        totalFileCount: 1,
        usedWorkerPool: true,
      },
      performance: {
        totalWallMs: 150,
        buckets: {},
        pipelinePhaseMs: { parse: 90, resolution: 35 },
        ftsIndexMs: {},
        counters: {
          scopeResolutionResolvedReferences: 1,
          scopeResolutionUnresolvedReferences: 1,
          scopeResolutionFinalizedImportsEmitted: 1,
          scopeResolutionFinalizedImportUsesEmitted: 1,
        },
        bottlenecks: [],
        overheadMs: 25,
      },
    });

    const comparison = compareAnalyzeBenchmarkSnapshots(before, after);

    expect(comparison.totalWallMs).toEqual({
      before: 200,
      after: 150,
      delta: -50,
      percentChange: -25,
    });
    expect(comparison.phaseMs.resolution).toEqual({
      before: 50,
      after: 35,
      delta: -15,
      percentChange: -30,
    });
    expect(comparison.relationshipCountsByType.CALLS).toEqual({ after: 1 });
    expect(comparison.keyMetrics.scopeResolutionResolvedReferences).toEqual({
      before: 0,
      after: 1,
      delta: 1,
    });
    expect(comparison.keyMetrics.scopeResolutionUnresolvedReferences).toEqual({
      before: 2,
      after: 1,
      delta: -1,
      percentChange: -50,
    });
    expect(comparison.keyMetrics.scopeResolutionFinalizedImportsEmitted).toEqual({
      before: 0,
      after: 1,
      delta: 1,
    });
    expect(comparison.keyMetrics.scopeResolutionFinalizedImportUsesEmitted).toEqual({
      before: 0,
      after: 1,
      delta: 1,
    });
    expect(comparison.graphDiffs.map((diff) => diff.field)).toEqual(
      expect.arrayContaining(['nodeCount', 'relationshipCount', 'byRelationshipType']),
    );
  });
});
