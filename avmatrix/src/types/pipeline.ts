import type { KnowledgeGraph } from '../core/graph/types.js';
import { CommunityDetectionResult } from '../core/ingestion/community-processor.js';
import { ProcessDetectionResult } from '../core/ingestion/process-processor.js';
import type { AnalyzeCounters, ParseMetrics, TimingMap } from '../core/analyze/analyze-metrics.js';

export interface PipelinePerformance {
  /** Phase wall-clock durations from the dependency-ordered pipeline runner. */
  phaseMs: TimingMap;
  /** Counters captured during ingestion. */
  counters: AnalyzeCounters;
  /** Parse-specific sub-step timings and counters. */
  parse?: ParseMetrics;
}

// CLI-specific: in-memory result with graph + detection results
export interface PipelineResult {
  graph: KnowledgeGraph;
  /** Absolute path to the repo root — used for lazy file reads during LadybugDB loading */
  repoPath: string;
  /** Total files scanned (for stats) */
  totalFileCount: number;
  communityResult?: CommunityDetectionResult;
  processResult?: ProcessDetectionResult;
  /**
   * True if the parse phase spawned a worker pool for this run. False only
   * means there were no parseable files.
   */
  usedWorkerPool: boolean;
  /** Phase 0 analyze-performance instrumentation. */
  performance?: PipelinePerformance;
}
