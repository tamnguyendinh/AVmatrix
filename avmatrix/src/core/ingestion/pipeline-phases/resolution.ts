/**
 * Phase: resolution
 *
 * Scope-aware reference resolution over `ParsedFile` facts finalized in the
 * parse phase. This phase builds the `ReferenceIndex`, emits non-duplicate
 * scope-aware graph edges, and exposes metrics.
 *
 * @deps    parse, crossFile
 * @reads   SemanticModel.scopes
 * @writes  graph via emitReferencesToGraph
 */

import type { ReferenceIndex } from 'avmatrix-shared';
import { performance } from 'node:perf_hooks';
import type { ResolutionMetrics } from '../../analyze/analyze-metrics.js';
import { roundMs } from '../../analyze/analyze-metrics.js';
import { emitReferencesToGraph } from '../emit-references.js';
import { resolveScopeReferenceSites } from '../scope-reference-resolver.js';
import type { PipelineContext, PhaseResult, PipelinePhase } from './types.js';
import { getPhaseOutput } from './types.js';
import type { ParseOutput } from './parse.js';

export interface ResolutionOutput {
  readonly referenceIndex: ReferenceIndex;
  readonly metrics: ResolutionMetrics;
}

const EMPTY_REFERENCE_INDEX: ReferenceIndex = Object.freeze({
  bySourceScope: Object.freeze(new Map()),
  byTargetDef: Object.freeze(new Map()),
});

export const resolutionPhase: PipelinePhase<ResolutionOutput> = {
  name: 'resolution',
  deps: ['parse', 'crossFile'],

  async execute(
    ctx: PipelineContext,
    deps: ReadonlyMap<string, PhaseResult<unknown>>,
  ): Promise<ResolutionOutput> {
    const parseOutput = getPhaseOutput<ParseOutput>(deps, 'parse');
    const scopes = parseOutput.resolutionContext.model.scopes;
    const metrics: ResolutionMetrics = { timings: {}, counters: {} };

    ctx.onProgress({
      phase: 'enriching',
      percent: 82,
      message: 'Resolving scope reference facts...',
      stats: {
        filesProcessed: parseOutput.totalFiles,
        totalFiles: parseOutput.totalFiles,
        nodesCreated: ctx.graph.nodeCount,
      },
    });

    if (scopes === undefined || scopes.referenceSites.length === 0) {
      metrics.counters.scopeResolutionReferenceSites = scopes?.referenceSites.length ?? 0;
      metrics.counters.scopeResolutionChunkSize = 0;
      metrics.counters.scopeResolutionChunks = 0;
      metrics.counters.scopeResolutionMaxChunkReferenceSites = 0;
      metrics.counters.scopeResolutionReadonlyIndexBytes = 0;
      metrics.counters.scopeResolutionReferenceIndexSourceScopes = 0;
      metrics.counters.scopeResolutionReferenceIndexTargetDefs = 0;
      metrics.counters.scopeResolutionResolvedReferences = 0;
      metrics.counters.scopeResolutionUnresolvedReferences = 0;
      metrics.counters.scopeResolutionResolvedCalls = 0;
      metrics.counters.scopeResolutionResolvedAccesses = 0;
      metrics.counters.scopeResolutionResolvedTypeReferences = 0;
      metrics.counters.scopeResolutionResolvedInheritance = 0;
      metrics.counters.scopeResolutionResolvedImportUses = 0;
      metrics.counters.scopeResolutionEdgesEmitted = 0;
      metrics.counters.scopeResolutionDuplicateEdgesSkipped = 0;
      if (scopes !== undefined) {
        const emitStart = performance.now();
        const emitStats = emitReferencesToGraph({
          graph: ctx.graph,
          scopes,
          referenceIndex: EMPTY_REFERENCE_INDEX,
        });
        metrics.timings.graphEmitMs = roundMs(performance.now() - emitStart);
        metrics.counters.scopeResolutionFinalizedImportsEmitted =
          emitStats.finalizedImportEdgesEmitted;
        metrics.counters.scopeResolutionDuplicateImportsSkipped =
          emitStats.skippedDuplicateImportEdge;
        metrics.counters.scopeResolutionFinalizedImportUsesEmitted =
          emitStats.finalizedImportUseEdgesEmitted;
        metrics.counters.scopeResolutionDuplicateImportUsesSkipped =
          emitStats.skippedDuplicateImportUseEdge;
        metrics.counters.scopeResolutionEdgesSkippedNoCaller = emitStats.skippedNoCaller;
        metrics.counters.scopeResolutionEdgesSkippedMissingTarget = emitStats.skippedMissingTarget;
      } else {
        metrics.counters.scopeResolutionFinalizedImportsEmitted = 0;
        metrics.counters.scopeResolutionDuplicateImportsSkipped = 0;
        metrics.counters.scopeResolutionFinalizedImportUsesEmitted = 0;
        metrics.counters.scopeResolutionDuplicateImportUsesSkipped = 0;
        metrics.counters.scopeResolutionEdgesSkippedNoCaller = 0;
        metrics.counters.scopeResolutionEdgesSkippedMissingTarget = 0;
      }
      return { referenceIndex: EMPTY_REFERENCE_INDEX, metrics };
    }

    const start = performance.now();
    const result = resolveScopeReferenceSites(scopes);
    metrics.timings.referenceResolveMs = roundMs(performance.now() - start);
    metrics.timings.readonlyIndexInitMs = roundMs(result.timings.readonlyIndexInitMs);
    metrics.timings.referenceWorkerResolveMs = roundMs(result.timings.referenceWorkerResolveMs);
    metrics.timings.referenceMergeMs = roundMs(result.timings.referenceMergeMs);
    metrics.timings.referenceIndexBuildMs = roundMs(result.timings.referenceIndexBuildMs);

    const emitStart = performance.now();
    const emitStats = emitReferencesToGraph({
      graph: ctx.graph,
      scopes,
      referenceIndex: result.referenceIndex,
    });
    metrics.timings.graphEmitMs = roundMs(performance.now() - emitStart);

    metrics.counters.scopeResolutionReferenceSites = result.stats.totalReferenceSites;
    metrics.counters.scopeResolutionChunkSize = result.stats.chunkSize;
    metrics.counters.scopeResolutionChunks = result.stats.chunksResolved;
    metrics.counters.scopeResolutionMaxChunkReferenceSites = result.stats.maxChunkReferenceSites;
    metrics.counters.scopeResolutionReadonlyIndexBytes = result.stats.readonlyIndexBytes;
    metrics.counters.scopeResolutionReferenceIndexSourceScopes =
      result.stats.referenceIndexSourceScopes;
    metrics.counters.scopeResolutionReferenceIndexTargetDefs =
      result.stats.referenceIndexTargetDefs;
    metrics.counters.scopeResolutionResolvedReferences = result.stats.resolvedReferences;
    metrics.counters.scopeResolutionUnresolvedReferences = result.stats.unresolvedReferences;
    metrics.counters.scopeResolutionResolvedCalls = result.stats.resolvedCalls;
    metrics.counters.scopeResolutionResolvedAccesses = result.stats.resolvedAccesses;
    metrics.counters.scopeResolutionResolvedTypeReferences = result.stats.resolvedTypeReferences;
    metrics.counters.scopeResolutionResolvedInheritance = result.stats.resolvedInheritance;
    metrics.counters.scopeResolutionResolvedImportUses = result.stats.resolvedImportUses;
    metrics.counters.scopeResolutionEdgesEmitted = emitStats.edgesEmitted;
    metrics.counters.scopeResolutionDuplicateEdgesSkipped = emitStats.skippedDuplicateEdge;
    metrics.counters.scopeResolutionFinalizedImportsEmitted = emitStats.finalizedImportEdgesEmitted;
    metrics.counters.scopeResolutionDuplicateImportsSkipped = emitStats.skippedDuplicateImportEdge;
    metrics.counters.scopeResolutionFinalizedImportUsesEmitted =
      emitStats.finalizedImportUseEdgesEmitted;
    metrics.counters.scopeResolutionDuplicateImportUsesSkipped =
      emitStats.skippedDuplicateImportUseEdge;
    metrics.counters.scopeResolutionEdgesSkippedNoCaller = emitStats.skippedNoCaller;
    metrics.counters.scopeResolutionEdgesSkippedMissingTarget = emitStats.skippedMissingTarget;

    return { referenceIndex: result.referenceIndex, metrics };
  },
};
