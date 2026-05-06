import fs from 'fs/promises';
import path from 'path';
import type { AnalyzePerformanceReport } from './analyze-metrics.js';
import {
  createGraphCorrectnessSnapshot,
  type GraphCorrectnessSnapshot,
} from './graph-correctness-snapshot.js';
import type { PipelineResult } from '../../types/pipeline.js';

export interface AnalyzeBenchmarkSnapshot {
  readonly schemaVersion: 1;
  readonly createdAt: string;
  readonly label?: string;
  readonly repoName: string;
  readonly repoPath: string;
  readonly environment?: AnalyzeBenchmarkEnvironment;
  readonly stats: {
    readonly files?: number;
    readonly nodes?: number;
    readonly edges?: number;
    readonly communities?: number;
    readonly processes?: number;
  };
  readonly graph?: GraphCorrectnessSnapshot;
  readonly performance?: AnalyzePerformanceReport;
  readonly keyMetrics: AnalyzeBenchmarkKeyMetrics;
}

export interface AnalyzeBenchmarkEnvironment {
  readonly avmatrixVersion?: string;
  readonly nodeVersion?: string;
  readonly platform?: string;
  readonly arch?: string;
  readonly repoGitCommit?: string;
  readonly repoGitDirty?: boolean;
}

export interface AnalyzeBenchmarkKeyMetrics {
  readonly totalWallMs?: number;
  readonly phaseMs: Record<string, number>;
  readonly nodeCount?: number;
  readonly relationshipCount?: number;
  readonly nodeCountsByLabel?: Record<string, number>;
  readonly relationshipCountsByType?: Record<string, number>;
  readonly parseMs?: number;
  readonly crossFileMs?: number;
  readonly resolutionMs?: number;
  readonly lbugLoadMs?: number;
  readonly parseableFiles?: number;
  readonly totalParseableMB?: number;
  readonly workerCount?: number;
  readonly parseChunkCount?: number;
  readonly scopeParsedFiles?: number;
  readonly scopeCount?: number;
  readonly scopeLocalDefs?: number;
  readonly scopeParsedImports?: number;
  readonly scopeReferenceSites?: number;
  readonly scopeExtractionAstReusedFiles?: number;
  readonly scopeExtractionCompatibilityFiles?: number;
  readonly scopeExtractionNoHookFiles?: number;
  readonly scopeExtractionFailedFiles?: number;
  readonly scopeFinalizedFiles?: number;
  readonly scopeFinalizeTotalImports?: number;
  readonly scopeFinalizeLinkedImports?: number;
  readonly scopeFinalizeUnresolvedImports?: number;
  readonly scopeResolutionReferenceSites?: number;
  readonly scopeResolutionChunkSize?: number;
  readonly scopeResolutionChunks?: number;
  readonly scopeResolutionMaxChunkReferenceSites?: number;
  readonly scopeResolutionReferenceIndexSourceScopes?: number;
  readonly scopeResolutionReferenceIndexTargetDefs?: number;
  readonly scopeResolutionResolvedReferences?: number;
  readonly scopeResolutionUnresolvedReferences?: number;
  readonly scopeResolutionResolvedCalls?: number;
  readonly scopeResolutionResolvedAccesses?: number;
  readonly scopeResolutionResolvedTypeReferences?: number;
  readonly scopeResolutionResolvedInheritance?: number;
  readonly scopeResolutionResolvedImportUses?: number;
  readonly scopeResolutionEdgesEmitted?: number;
  readonly scopeResolutionDuplicateEdgesSkipped?: number;
  readonly scopeResolutionEdgesSkippedNoCaller?: number;
  readonly scopeResolutionEdgesSkippedMissingTarget?: number;
  readonly crossFileReprocessedFiles?: number;
  readonly crossFileReadContentsMs?: number;
  readonly crossFileProcessCallsParserParseMs?: number;
  readonly csvRelationshipRows?: number;
  readonly ladybugCopyCount?: number;
}

export function createAnalyzeBenchmarkSnapshot(input: {
  readonly repoName: string;
  readonly repoPath: string;
  readonly stats: AnalyzeBenchmarkSnapshot['stats'];
  readonly pipelineResult?: PipelineResult;
  readonly performance?: AnalyzePerformanceReport;
  readonly environment?: AnalyzeBenchmarkEnvironment;
  readonly label?: string;
  readonly createdAt?: string;
}): AnalyzeBenchmarkSnapshot {
  const graph =
    input.pipelineResult !== undefined
      ? createGraphCorrectnessSnapshot(input.pipelineResult)
      : undefined;
  return {
    schemaVersion: 1,
    createdAt: input.createdAt ?? new Date().toISOString(),
    ...(input.label !== undefined ? { label: input.label } : {}),
    repoName: input.repoName,
    repoPath: input.repoPath,
    ...(input.environment !== undefined ? { environment: input.environment } : {}),
    stats: input.stats,
    ...(graph !== undefined ? { graph } : {}),
    ...(input.performance !== undefined ? { performance: input.performance } : {}),
    keyMetrics: createKeyMetrics(input.performance, graph),
  };
}

export async function writeAnalyzeBenchmarkSnapshot(
  filePath: string,
  snapshot: AnalyzeBenchmarkSnapshot,
): Promise<void> {
  await fs.mkdir(path.dirname(filePath), { recursive: true });
  await fs.writeFile(filePath, `${JSON.stringify(snapshot, null, 2)}\n`, 'utf-8');
}

function createKeyMetrics(
  performance: AnalyzePerformanceReport | undefined,
  graph: GraphCorrectnessSnapshot | undefined,
): AnalyzeBenchmarkKeyMetrics {
  const counters = performance?.counters ?? {};
  const phaseMs = performance?.pipelinePhaseMs ?? {};
  return {
    totalWallMs: performance?.totalWallMs,
    phaseMs,
    nodeCount: graph?.nodeCount ?? counters.nodeCount,
    relationshipCount: graph?.relationshipCount ?? counters.edgeCount,
    nodeCountsByLabel: graph?.byNodeLabel,
    relationshipCountsByType: graph?.byRelationshipType,
    parseMs: phaseMs.parse ?? performance?.buckets.parse,
    crossFileMs: phaseMs.crossFile ?? performance?.buckets.crossFile,
    resolutionMs: phaseMs.resolution ?? performance?.buckets.resolution,
    lbugLoadMs: performance?.buckets.lbugLoad,
    parseableFiles: counters.parseableFiles,
    totalParseableMB: counters.totalParseableMB,
    workerCount: counters.workerCount,
    parseChunkCount: counters.parseChunkCount,
    scopeParsedFiles: counters.scopeParsedFiles,
    scopeCount: counters.scopeCount,
    scopeLocalDefs: counters.scopeLocalDefs,
    scopeParsedImports: counters.scopeParsedImports,
    scopeReferenceSites: counters.scopeReferenceSites,
    scopeExtractionAstReusedFiles: counters.scopeExtractionAstReusedFiles,
    scopeExtractionCompatibilityFiles: counters.scopeExtractionCompatibilityFiles,
    scopeExtractionNoHookFiles: counters.scopeExtractionNoHookFiles,
    scopeExtractionFailedFiles: counters.scopeExtractionFailedFiles,
    scopeFinalizedFiles: counters.scopeFinalizedFiles,
    scopeFinalizeTotalImports: counters.scopeFinalizeTotalImports,
    scopeFinalizeLinkedImports: counters.scopeFinalizeLinkedImports,
    scopeFinalizeUnresolvedImports: counters.scopeFinalizeUnresolvedImports,
    scopeResolutionReferenceSites: counters.scopeResolutionReferenceSites,
    scopeResolutionChunkSize: counters.scopeResolutionChunkSize,
    scopeResolutionChunks: counters.scopeResolutionChunks,
    scopeResolutionMaxChunkReferenceSites: counters.scopeResolutionMaxChunkReferenceSites,
    scopeResolutionReferenceIndexSourceScopes: counters.scopeResolutionReferenceIndexSourceScopes,
    scopeResolutionReferenceIndexTargetDefs: counters.scopeResolutionReferenceIndexTargetDefs,
    scopeResolutionResolvedReferences: counters.scopeResolutionResolvedReferences,
    scopeResolutionUnresolvedReferences: counters.scopeResolutionUnresolvedReferences,
    scopeResolutionResolvedCalls: counters.scopeResolutionResolvedCalls,
    scopeResolutionResolvedAccesses: counters.scopeResolutionResolvedAccesses,
    scopeResolutionResolvedTypeReferences: counters.scopeResolutionResolvedTypeReferences,
    scopeResolutionResolvedInheritance: counters.scopeResolutionResolvedInheritance,
    scopeResolutionResolvedImportUses: counters.scopeResolutionResolvedImportUses,
    scopeResolutionEdgesEmitted: counters.scopeResolutionEdgesEmitted,
    scopeResolutionDuplicateEdgesSkipped: counters.scopeResolutionDuplicateEdgesSkipped,
    scopeResolutionEdgesSkippedNoCaller: counters.scopeResolutionEdgesSkippedNoCaller,
    scopeResolutionEdgesSkippedMissingTarget: counters.scopeResolutionEdgesSkippedMissingTarget,
    crossFileReprocessedFiles: counters.crossFileReprocessedFiles,
    crossFileReadContentsMs: performance?.crossFile?.timings.readContentsMs,
    crossFileProcessCallsParserParseMs: performance?.crossFile?.timings.processCallsParserParseMs,
    csvRelationshipRows: counters.csvRelationshipRows,
    ladybugCopyCount: counters.ladybugCopyCount,
  };
}
