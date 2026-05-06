import { describe, expect, it } from 'vitest';
import { buildAnalyzePerformanceReport } from '../../src/core/analyze/analyze-metrics.js';

describe('analyze metrics', () => {
  it('sorts bottlenecks by measured bucket duration and computes overhead', () => {
    const report = buildAnalyzePerformanceReport({
      totalWallMs: 100,
      buckets: {
        lbugLoad: 30,
        fts: 10,
      },
      pipelinePhaseMs: {
        parse: 40,
        scan: 5,
      },
      counters: {
        totalFiles: 12,
      },
      resolution: {
        timings: {
          referenceResolveMs: 3,
          graphEmitMs: 0,
        },
        counters: {
          scopeResolutionReferenceSites: 4,
          scopeResolutionReadonlyIndexBytes: 2048,
          scopeResolutionResolvedReferences: 3,
          scopeResolutionUnresolvedReferences: 1,
          scopeResolutionResolvedCalls: 2,
          scopeResolutionResolvedAccesses: 0,
          scopeResolutionResolvedTypeReferences: 0,
          scopeResolutionResolvedInheritance: 1,
          scopeResolutionResolvedImportUses: 0,
          scopeResolutionEdgesEmitted: 0,
          scopeResolutionFinalizedImportsEmitted: 1,
          scopeResolutionDuplicateImportsSkipped: 0,
          scopeResolutionFinalizedImportUsesEmitted: 1,
          scopeResolutionDuplicateImportUsesSkipped: 0,
        },
      },
    });

    expect(report.bottlenecks.map((b) => b.bucket)).toEqual(['parse', 'lbugLoad', 'fts', 'scan']);
    expect(report.bottlenecks[0]).toMatchObject({
      bucket: 'parse',
      durationMs: 40,
      percentOfTotal: 40,
    });
    expect(report.overheadMs).toBe(15);
    expect(report.counters.totalFiles).toBe(12);
    expect(report.resolution?.counters.scopeResolutionResolvedReferences).toBe(3);
    expect(report.resolution?.counters.scopeResolutionReadonlyIndexBytes).toBe(2048);
    expect(report.resolution?.counters.scopeResolutionFinalizedImportsEmitted).toBe(1);
    expect(report.resolution?.counters.scopeResolutionFinalizedImportUsesEmitted).toBe(1);
  });
});
