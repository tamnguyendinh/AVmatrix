import fs from 'fs/promises';
import os from 'os';
import path from 'path';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { benchmarkCompareCommand, formatBenchmarkComparison } from '../../src/cli/benchmark.js';
import {
  compareAnalyzeBenchmarkSnapshots,
  createAnalyzeBenchmarkSnapshot,
} from '../../src/core/analyze/analyze-benchmark-snapshot.js';
import { buildTestGraph } from '../helpers/test-graph.js';

const tempDirs: string[] = [];

afterEach(async () => {
  vi.restoreAllMocks();
  await Promise.all(tempDirs.splice(0).map((dir) => fs.rm(dir, { recursive: true, force: true })));
});

describe('benchmarkCompareCommand', () => {
  it('prints a concise benchmark delta summary from two JSON artifacts', async () => {
    const dir = await fs.mkdtemp(path.join(os.tmpdir(), 'avmatrix-benchmark-'));
    tempDirs.push(dir);
    const beforePath = path.join(dir, 'before.json');
    const afterPath = path.join(dir, 'after.json');
    const before = makeSnapshot('before', 200, 0, 2);
    const after = makeSnapshot('after', 150, 1, 1);
    await fs.writeFile(beforePath, `${JSON.stringify(before, null, 2)}\n`, 'utf-8');
    await fs.writeFile(afterPath, `${JSON.stringify(after, null, 2)}\n`, 'utf-8');
    const log = vi.spyOn(console, 'log').mockImplementation(() => {});

    await benchmarkCompareCommand(beforePath, afterPath);

    expect(log).toHaveBeenCalledTimes(1);
    const output = String(log.mock.calls[0]![0]);
    expect(output).toContain('AVmatrix benchmark comparison');
    expect(output).toContain('labels: before -> after');
    expect(output).toContain('wall: 200 -> 150 (-50, -25%)');
    expect(output).toContain('scopeResolutionResolvedReferences: 0 -> 1 (+1)');
    expect(output).toContain('scopeResolutionUnresolvedReferences: 2 -> 1 (-1, -50%)');
    expect(output).toContain('scopeResolutionFinalizedImportsEmitted: 0 -> 1 (+1)');
    expect(output).toContain('scopeResolutionFinalizedImportUsesEmitted: 0 -> 1 (+1)');
    expect(output).toContain('languageCoverageByLanguage:');
    expect(output).toContain('  typescript:');
    expect(output).toContain('scopeResolutionResolvedReferences: 0 -> 1 (+1)');
  });

  it('can format the full comparison as JSON-compatible data', () => {
    const comparison = compareAnalyzeBenchmarkSnapshots(
      makeSnapshot('before', 200, 0, 2),
      makeSnapshot('after', 150, 1, 1),
    );

    expect(formatBenchmarkComparison(comparison)).toContain('phaseMs:');
    expect(JSON.parse(JSON.stringify(comparison))).toMatchObject({
      beforeLabel: 'before',
      afterLabel: 'after',
      totalWallMs: { before: 200, after: 150, delta: -50, percentChange: -25 },
    });
  });
});

function makeSnapshot(
  label: string,
  totalWallMs: number,
  resolvedReferences: number,
  unresolvedReferences: number,
) {
  const graph = buildTestGraph(
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
  return createAnalyzeBenchmarkSnapshot({
    repoName: 'demo',
    repoPath: 'F:/demo',
    createdAt: '2026-05-06T00:00:00.000Z',
    label,
    stats: { files: 1, nodes: 1, edges: 0 },
    pipelineResult: {
      graph,
      repoPath: 'F:/demo',
      totalFileCount: 1,
      usedWorkerPool: true,
    },
    performance: {
      totalWallMs,
      buckets: {},
      pipelinePhaseMs: { parse: 100, resolution: 50 },
      ftsIndexMs: {},
      counters: {
        scopeResolutionResolvedReferences: resolvedReferences,
        scopeResolutionUnresolvedReferences: unresolvedReferences,
        scopeResolutionFinalizedImportsEmitted: resolvedReferences,
        scopeResolutionFinalizedImportUsesEmitted: resolvedReferences,
        languageCoverageByLanguage: {
          typescript: {
            parseableFiles: 1,
            scopeExtractionAstReusedFiles: 1,
            scopeResolutionReferenceSites: 2,
            scopeResolutionResolvedReferences: resolvedReferences,
            scopeResolutionUnresolvedReferences: unresolvedReferences,
            astReusedScopeCoveragePercent: 100,
            compatibilityOrUnavailableScopePercent: 0,
          },
        },
      },
      bottlenecks: [],
      overheadMs: 50,
    },
  });
}
