import fs from 'fs/promises';
import path from 'path';
import {
  compareAnalyzeBenchmarkSnapshots,
  type AnalyzeBenchmarkComparison,
  type AnalyzeBenchmarkSnapshot,
  type NumericDelta,
} from '../core/analyze/analyze-benchmark-snapshot.js';

export interface BenchmarkCompareOptions {
  readonly json?: boolean;
}

export async function benchmarkCompareCommand(
  beforePath?: string,
  afterPath?: string,
  options: BenchmarkCompareOptions = {},
): Promise<void> {
  if (beforePath === undefined || afterPath === undefined) {
    console.error('Usage: avmatrix benchmark-compare <before.json> <after.json> [--json]');
    process.exitCode = 1;
    return;
  }

  const [before, after] = await Promise.all([
    readBenchmarkSnapshot(beforePath),
    readBenchmarkSnapshot(afterPath),
  ]);
  const comparison = compareAnalyzeBenchmarkSnapshots(before, after);
  const output = options.json
    ? JSON.stringify(comparison, null, 2)
    : formatBenchmarkComparison(comparison);
  console.log(output);
}

export function formatBenchmarkComparison(comparison: AnalyzeBenchmarkComparison): string {
  const lines: string[] = [];
  lines.push('AVmatrix benchmark comparison');
  lines.push(
    `labels: ${comparison.beforeLabel ?? 'before'} -> ${comparison.afterLabel ?? 'after'}`,
  );
  lines.push(`wall: ${formatDelta(comparison.totalWallMs)}`);

  lines.push('');
  lines.push('phaseMs:');
  for (const key of Object.keys(comparison.phaseMs).sort()) {
    lines.push(`  ${key}: ${formatDelta(comparison.phaseMs[key])}`);
  }

  lines.push('');
  lines.push('relationshipCountsByType:');
  for (const key of Object.keys(comparison.relationshipCountsByType).sort()) {
    lines.push(`  ${key}: ${formatDelta(comparison.relationshipCountsByType[key])}`);
  }

  lines.push('');
  lines.push('resolutionKeyMetrics:');
  for (const key of RESOLUTION_KEY_METRICS) {
    const delta = comparison.keyMetrics[key];
    if (delta === undefined) continue;
    lines.push(`  ${key}: ${formatDelta(delta)}`);
  }

  lines.push('');
  lines.push(`graphDiffs: ${comparison.graphDiffs.length}`);
  for (const diff of comparison.graphDiffs.slice(0, 10)) {
    lines.push(`  ${diff.field}`);
  }
  if (comparison.graphDiffs.length > 10) {
    lines.push(`  ... ${comparison.graphDiffs.length - 10} more`);
  }

  return `${lines.join('\n')}\n`;
}

const RESOLUTION_KEY_METRICS = Object.freeze([
  'scopeResolutionReferenceSites',
  'scopeResolutionReadonlyIndexBytes',
  'scopeResolutionResolvedReferences',
  'scopeResolutionUnresolvedReferences',
  'scopeResolutionResolvedCalls',
  'scopeResolutionResolvedAccesses',
  'scopeResolutionResolvedTypeReferences',
  'scopeResolutionResolvedInheritance',
  'scopeResolutionResolvedImportUses',
  'scopeResolutionEdgesEmitted',
  'scopeResolutionDuplicateEdgesSkipped',
  'scopeResolutionFinalizedImportsEmitted',
  'scopeResolutionDuplicateImportsSkipped',
  'scopeResolutionFinalizedImportUsesEmitted',
  'scopeResolutionDuplicateImportUsesSkipped',
]);

async function readBenchmarkSnapshot(filePath: string): Promise<AnalyzeBenchmarkSnapshot> {
  const absolute = path.resolve(filePath);
  const raw = await fs.readFile(absolute, 'utf-8');
  const parsed = JSON.parse(raw) as Partial<AnalyzeBenchmarkSnapshot>;
  if (parsed.schemaVersion !== 1 || parsed.keyMetrics === undefined) {
    throw new Error(`Invalid analyze benchmark snapshot: ${absolute}`);
  }
  return parsed as AnalyzeBenchmarkSnapshot;
}

function formatDelta(delta: NumericDelta | undefined): string {
  if (delta === undefined) return 'n/a';
  const before = formatNumber(delta.before);
  const after = formatNumber(delta.after);
  const change = delta.delta === undefined ? 'n/a' : signedNumber(delta.delta);
  const pct = delta.percentChange === undefined ? '' : `, ${signedNumber(delta.percentChange)}%`;
  return `${before} -> ${after} (${change}${pct})`;
}

function formatNumber(value: number | undefined): string {
  if (value === undefined) return 'n/a';
  if (Number.isInteger(value)) return value.toLocaleString();
  return value.toFixed(1);
}

function signedNumber(value: number): string {
  const formatted = Number.isInteger(value) ? value.toLocaleString() : value.toFixed(1);
  return value > 0 ? `+${formatted}` : formatted;
}
