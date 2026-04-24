import { performance } from 'node:perf_hooks';

export type AnalyzeTimingBucket =
  | 'scan'
  | 'structure'
  | 'markdown'
  | 'cobol'
  | 'parse'
  | 'routes'
  | 'tools'
  | 'orm'
  | 'crossFile'
  | 'mro'
  | 'communities'
  | 'processes'
  | 'lbugLoad'
  | 'fts'
  | 'embeddings'
  | 'metadata'
  | 'aiContext';

export type TimingMap = Record<string, number>;

export interface AnalyzeCounters {
  totalFiles?: number;
  parseableFiles?: number;
  totalParseableMB?: number;
  nodeCount?: number;
  edgeCount?: number;
  workerCount?: number;
  parseChunkCount?: number;
  csvNodeRows?: number;
  csvRelationshipRows?: number;
  ladybugCopyCount?: number;
  ftsIndexCount?: number;
  skippedLargeFiles?: number;
  parserUnavailableFiles?: number;
  usedWorkerPool?: boolean;
  crossFileReprocessedFiles?: number;
}

export interface LbugLoadTimingBreakdown {
  csvGenerationMs?: number;
  nodeCopyMs?: number;
  relationshipSplitMs?: number;
  relationshipCopyMs?: number;
  fallbackRelationshipInsertMs?: number;
  cleanupMs?: number;
}

export interface LbugLoadMetrics {
  timings: LbugLoadTimingBreakdown;
  counters: Pick<
    AnalyzeCounters,
    'csvNodeRows' | 'csvRelationshipRows' | 'ladybugCopyCount'
  > & {
    nodeCopyCount?: number;
    relationshipCopyCount?: number;
    insertedRelationships?: number;
    skippedRelationships?: number;
  };
}

export interface ParseTimingBreakdown {
  readContentsMs?: number;
  workerParseMs?: number;
  importResolveMs?: number;
  heritageResolveMs?: number;
  routeResolveMs?: number;
  callResolveMs?: number;
  assignmentResolveMs?: number;
  wildcardSynthesisMs?: number;
  exportedTypeMapEnrichMs?: number;
}

export interface ParseMetrics {
  timings: ParseTimingBreakdown;
  counters: Pick<
    AnalyzeCounters,
    'parseableFiles' | 'totalParseableMB' | 'workerCount' | 'parseChunkCount'
    | 'parserUnavailableFiles'
  >;
}

export interface AnalyzeBottleneck {
  bucket: string;
  durationMs: number;
  percentOfTotal: number;
}

export interface AnalyzePerformanceReport {
  totalWallMs: number;
  buckets: TimingMap;
  pipelinePhaseMs: TimingMap;
  ftsIndexMs: TimingMap;
  counters: AnalyzeCounters;
  bottlenecks: AnalyzeBottleneck[];
  overheadMs: number;
  lbugLoad?: LbugLoadMetrics;
  parse?: ParseMetrics;
}

export class AnalyzeMetricsCollector {
  private readonly startMs = performance.now();
  private readonly buckets = new Map<string, number>();
  private readonly counters: AnalyzeCounters = {};

  mark(bucket: string, durationMs: number): void {
    if (!Number.isFinite(durationMs) || durationMs < 0) return;
    this.buckets.set(bucket, roundMs((this.buckets.get(bucket) ?? 0) + durationMs));
  }

  async time<T>(bucket: string, fn: () => Promise<T>): Promise<T> {
    const start = performance.now();
    try {
      return await fn();
    } finally {
      this.mark(bucket, performance.now() - start);
    }
  }

  timeSync<T>(bucket: string, fn: () => T): T {
    const start = performance.now();
    try {
      return fn();
    } finally {
      this.mark(bucket, performance.now() - start);
    }
  }

  setCounter<K extends keyof AnalyzeCounters>(key: K, value: AnalyzeCounters[K]): void {
    this.counters[key] = value;
  }

  addCounters(counters: AnalyzeCounters): void {
    Object.assign(this.counters, counters);
  }

  snapshot(): { buckets: TimingMap; counters: AnalyzeCounters } {
    return {
      buckets: mapToRoundedRecord(this.buckets),
      counters: { ...this.counters },
    };
  }

  elapsedMs(): number {
    return roundMs(performance.now() - this.startMs);
  }
}

export const roundMs = (value: number): number => Math.round(value * 10) / 10;

export const mapToRoundedRecord = (map: ReadonlyMap<string, number>): TimingMap => {
  const out: TimingMap = {};
  for (const [key, value] of map) out[key] = roundMs(value);
  return out;
};

export function buildAnalyzePerformanceReport(params: {
  totalWallMs: number;
  buckets: TimingMap;
  pipelinePhaseMs?: TimingMap;
  ftsIndexMs?: TimingMap;
  counters?: AnalyzeCounters;
  lbugLoad?: LbugLoadMetrics;
  parse?: ParseMetrics;
}): AnalyzePerformanceReport {
  const pipelinePhaseMs = params.pipelinePhaseMs ?? {};
  const buckets = { ...pipelinePhaseMs, ...params.buckets };
  const totalWallMs = roundMs(params.totalWallMs);
  const measuredMs = Object.values(buckets).reduce((sum, value) => sum + value, 0);
  const overheadMs = roundMs(Math.max(0, totalWallMs - measuredMs));

  const bottlenecks = Object.entries(buckets)
    .filter(([, durationMs]) => durationMs > 0)
    .sort((a, b) => b[1] - a[1])
    .map(([bucket, durationMs]) => ({
      bucket,
      durationMs: roundMs(durationMs),
      percentOfTotal: totalWallMs > 0 ? roundMs((durationMs / totalWallMs) * 100) : 0,
    }));

  return {
    totalWallMs,
    buckets,
    pipelinePhaseMs,
    ftsIndexMs: params.ftsIndexMs ?? {},
    counters: params.counters ?? {},
    bottlenecks,
    overheadMs,
    lbugLoad: params.lbugLoad,
    parse: params.parse,
  };
}
