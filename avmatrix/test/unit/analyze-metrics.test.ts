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
    });

    expect(report.bottlenecks.map((b) => b.bucket)).toEqual(['parse', 'lbugLoad', 'fts', 'scan']);
    expect(report.bottlenecks[0]).toMatchObject({
      bucket: 'parse',
      durationMs: 40,
      percentOfTotal: 40,
    });
    expect(report.overheadMs).toBe(15);
    expect(report.counters.totalFiles).toBe(12);
  });
});
