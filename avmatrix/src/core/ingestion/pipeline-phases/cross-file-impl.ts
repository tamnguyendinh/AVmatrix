/**
 * Cross-file binding propagation — extracted from pipeline.ts.
 *
 * Seeds downstream files with resolved type bindings from upstream exports.
 * Files are processed in topological import order so upstream bindings
 * are available when downstream files are re-resolved.
 *
 * @module
 */

import { performance } from 'node:perf_hooks';
import {
  processCalls,
  buildImportedReturnTypes,
  buildImportedRawReturnTypes,
  type ExportedTypeMap,
  type ProcessCallsQueryCache,
  type ProcessCallsTimingSink,
} from '../call-processor.js';
import type { createResolutionContext } from '../model/resolution-context.js';
import { createASTCache } from '../ast-cache.js';
import { type PipelineProgress, getLanguageFromFilename } from 'avmatrix-shared';
import { readFileContents } from '../filesystem-walker.js';
import { isLanguageAvailable } from '../../tree-sitter/parser-loader.js';
import { topologicalLevelSort } from '../utils/graph-sort.js';
import type { KnowledgeGraph } from '../../graph/types.js';
import { isDev } from '../utils/env.js';
import type { CrossFileMetrics } from '../../analyze/analyze-metrics.js';

/** Max AST trees to keep in LRU cache for cross-file binding propagation. */
const AST_CACHE_CAP = 50;

/** Minimum percentage of files that must benefit from cross-file seeding. */
const CROSS_FILE_SKIP_THRESHOLD = 0.03;
/** Hard cap on files re-processed during cross-file propagation. */
const MAX_CROSS_FILE_REPROCESS = 2000;

const hasIterableReturnType = (returnType: unknown): returnType is string =>
  typeof returnType === 'string' &&
  (returnType.includes('[]') ||
    /\b(?:Array|Collection|Iterable|List|Set|Map|Sequence)\s*</.test(returnType));

const collectGlobalReturnCandidateFiles = (graph: KnowledgeGraph): Set<string> => {
  const candidates = new Set<string>();
  graph.forEachRelationship((rel) => {
    if (rel.type !== 'CALLS') return;
    const source = graph.getNode(rel.sourceId);
    const target = graph.getNode(rel.targetId);
    const sourceFile = source?.properties?.filePath;
    if (!sourceFile) return;
    if (hasIterableReturnType(target?.properties?.returnType)) {
      candidates.add(sourceFile);
    }
  });
  return candidates;
};

export interface CrossFilePropagationResult {
  filesReprocessed: number;
  metrics: CrossFileMetrics;
}

const roundMs = (value: number): number => Math.round(value * 10) / 10;

const addTiming = (
  metrics: CrossFileMetrics,
  key: keyof CrossFileMetrics['timings'],
  durationMs: number,
): void => {
  metrics.timings[key] = roundMs((metrics.timings[key] ?? 0) + durationMs);
};

const timeSync = <T>(
  metrics: CrossFileMetrics,
  key: keyof CrossFileMetrics['timings'],
  fn: () => T,
): T => {
  const start = performance.now();
  try {
    return fn();
  } finally {
    addTiming(metrics, key, performance.now() - start);
  }
};

const timeAsync = async <T>(
  metrics: CrossFileMetrics,
  key: keyof CrossFileMetrics['timings'],
  fn: () => Promise<T>,
): Promise<T> => {
  const start = performance.now();
  try {
    return await fn();
  } finally {
    addTiming(metrics, key, performance.now() - start);
  }
};

const makeProcessCallsTimingSink = (metrics: CrossFileMetrics): ProcessCallsTimingSink => ({
  mark(key, durationMs) {
    addTiming(metrics, key, durationMs);
  },
});

/**
 * Cross-file binding propagation.
 * Returns the number of files re-processed plus instrumentation.
 */
export async function runCrossFileBindingPropagation(
  graph: KnowledgeGraph,
  ctx: ReturnType<typeof createResolutionContext>,
  parseExportedTypeMap: ReadonlyMap<string, ReadonlyMap<string, string>>,
  allPathSet: ReadonlySet<string>,
  totalFiles: number,
  repoPath: string,
  pipelineStart: number,
  onProgress: (progress: PipelineProgress) => void,
): Promise<CrossFilePropagationResult> {
  const metrics: CrossFileMetrics = { timings: {}, counters: {} };
  const processCallsTimingSink = makeProcessCallsTimingSink(metrics);
  const processCallsQueryCache: ProcessCallsQueryCache = new Map();
  const totalStart = performance.now();
  const finish = (filesReprocessed: number, skipReason?: string): CrossFilePropagationResult => {
    metrics.timings.totalMs = roundMs(performance.now() - totalStart);
    metrics.counters.filesReprocessed = filesReprocessed;
    if (skipReason) {
      metrics.counters.skipped = true;
      metrics.counters.skipReason = skipReason;
    }
    return { filesReprocessed, metrics };
  };

  if (parseExportedTypeMap.size === 0) return finish(0, 'no-exported-types');
  const globalReturnCandidates = collectGlobalReturnCandidateFiles(graph);
  if (ctx.namedImportMap.size === 0 && globalReturnCandidates.size === 0) {
    return finish(0, 'no-named-imports');
  }

  // Build a local mutable working copy. Per-file re-resolution below mutates
  // this map (each `processCalls` writes that file's exports back into it so
  // later iterations in the same level/loop can resolve transitive bindings).
  // Owning a local copy here keeps `ParseOutput.exportedTypeMap` truly
  // read-only at the phase boundary — no cast, no shared-mutable handoff.
  const exportedTypeMap: ExportedTypeMap = new Map();
  for (const [fp, exports] of parseExportedTypeMap) {
    exportedTypeMap.set(fp, new Map(exports));
  }

  const { levels, cycleCount } = timeSync(metrics, 'topologicalSortMs', () =>
    topologicalLevelSort(ctx.importMap),
  );
  metrics.counters.importLevels = levels.length;
  metrics.counters.importCycleFiles = cycleCount;

  if (isDev && cycleCount > 0) {
    console.log(`🔄 ${cycleCount} files in import cycles (processed last in undefined order)`);
  }

  let filesWithGaps = globalReturnCandidates.size;
  const gapThreshold = Math.max(1, Math.ceil(totalFiles * CROSS_FILE_SKIP_THRESHOLD));
  timeSync(metrics, 'candidateSelectionMs', () => {
    outer: for (const level of levels) {
      for (const filePath of level) {
        const imports = ctx.namedImportMap.get(filePath);
        if (!imports) continue;
        for (const [, binding] of imports) {
          const upstream = exportedTypeMap.get(binding.sourcePath);
          if (upstream?.has(binding.exportedName)) {
            filesWithGaps++;
            break;
          }
          const def = ctx.model.symbols.lookupExactFull(binding.sourcePath, binding.exportedName);
          if (def?.returnType) {
            filesWithGaps++;
            break;
          }
        }
        if (filesWithGaps >= gapThreshold) break outer;
      }
    }
  });
  metrics.counters.filesWithGaps = filesWithGaps;

  const gapRatio = totalFiles > 0 ? filesWithGaps / totalFiles : 0;
  if (gapRatio < CROSS_FILE_SKIP_THRESHOLD && filesWithGaps < gapThreshold) {
    if (isDev) {
      console.log(
        `⏭️ Cross-file re-resolution skipped (${filesWithGaps}/${totalFiles} files, ${(gapRatio * 100).toFixed(1)}% < ${CROSS_FILE_SKIP_THRESHOLD * 100}% threshold)`,
      );
    }
    return finish(0, 'below-gap-threshold');
  }

  // Intentionally reports `phase: 'parsing'` rather than a separate
  // 'crossFile' phase: cross-file re-resolution is logically a continuation of
  // the parsing/resolution work and is bucketed under "parsing" in any
  // telemetry that groups events by phase name. Kept consistent with the
  // upstream `parse` phase's progress events so the UI shows one continuous
  // progress segment instead of a phase flicker. If a future change splits
  // this out into its own phase, also rename `parse-impl.ts` per-chunk
  // progress events accordingly.
  onProgress({
    phase: 'parsing',
    percent: 82,
    message: `Cross-file type propagation (${filesWithGaps}+ files)...`,
    stats: { filesProcessed: totalFiles, totalFiles, nodesCreated: graph.nodeCount },
  });

  let crossFileResolved = 0;
  const crossFileStart = Date.now();
  const astCache = createASTCache(AST_CACHE_CAP);

  const levelsToProcess =
    globalReturnCandidates.size > 0 ? [...levels, [...globalReturnCandidates]] : levels;

  for (const level of levelsToProcess) {
    const levelCandidates: {
      filePath: string;
      seeded: Map<string, string>;
      importedReturns: ReadonlyMap<string, string>;
      importedRawReturns: ReadonlyMap<string, string>;
    }[] = [];
    for (const filePath of level) {
      const selectionStart = performance.now();
      if (crossFileResolved + levelCandidates.length >= MAX_CROSS_FILE_REPROCESS) break;
      const imports = ctx.namedImportMap.get(filePath);
      const isGlobalReturnCandidate = globalReturnCandidates.has(filePath);
      if (!imports && !isGlobalReturnCandidate) {
        addTiming(metrics, 'candidateSelectionMs', performance.now() - selectionStart);
        continue;
      }

      const seeded = new Map<string, string>();
      if (imports) {
        for (const [localName, binding] of imports) {
          const upstream = exportedTypeMap.get(binding.sourcePath);
          if (upstream) {
            const type = upstream.get(binding.exportedName);
            if (type) seeded.set(localName, type);
          }
        }
      }
      addTiming(metrics, 'candidateSelectionMs', performance.now() - selectionStart);

      const { importedReturns, importedRawReturns } = timeSync(
        metrics,
        'importedReturnMapsMs',
        () => ({
          importedReturns: buildImportedReturnTypes(
            filePath,
            ctx.namedImportMap,
            ctx.model.symbols,
          ),
          importedRawReturns: buildImportedRawReturnTypes(
            filePath,
            ctx.namedImportMap,
            ctx.model.symbols,
          ),
        }),
      );

      const postReturnSelectionStart = performance.now();
      if (seeded.size === 0 && importedReturns.size === 0 && !isGlobalReturnCandidate) {
        addTiming(metrics, 'candidateSelectionMs', performance.now() - postReturnSelectionStart);
        continue;
      }
      if (!allPathSet.has(filePath)) {
        addTiming(metrics, 'candidateSelectionMs', performance.now() - postReturnSelectionStart);
        continue;
      }

      const lang = getLanguageFromFilename(filePath);
      if (!lang || !isLanguageAvailable(lang)) {
        addTiming(metrics, 'candidateSelectionMs', performance.now() - postReturnSelectionStart);
        continue;
      }

      levelCandidates.push({ filePath, seeded, importedReturns, importedRawReturns });
      addTiming(metrics, 'candidateSelectionMs', performance.now() - postReturnSelectionStart);
    }

    if (levelCandidates.length === 0) continue;
    metrics.counters.candidateFiles =
      (metrics.counters.candidateFiles ?? 0) + levelCandidates.length;

    const levelPaths = levelCandidates.map((c) => c.filePath);
    const contentMap = await timeAsync(metrics, 'readContentsMs', () =>
      readFileContents(repoPath, levelPaths),
    );

    for (const { filePath, seeded, importedReturns, importedRawReturns } of levelCandidates) {
      const content = contentMap.get(filePath);
      if (!content) continue;

      const reFile = [{ path: filePath, content }];
      const bindings = new Map<string, ReadonlyMap<string, string>>();
      if (seeded.size > 0) bindings.set(filePath, seeded);

      const importedReturnTypesMap = new Map<string, ReadonlyMap<string, string>>();
      if (importedReturns.size > 0) {
        importedReturnTypesMap.set(filePath, importedReturns);
      }

      const importedRawReturnTypesMap = new Map<string, ReadonlyMap<string, string>>();
      if (importedRawReturns.size > 0) {
        importedRawReturnTypesMap.set(filePath, importedRawReturns);
      }

      await timeAsync(metrics, 'processCallsMs', () =>
        processCalls(
          graph,
          reFile,
          astCache,
          ctx,
          undefined,
          exportedTypeMap,
          bindings.size > 0 ? bindings : undefined,
          importedReturnTypesMap.size > 0 ? importedReturnTypesMap : undefined,
          importedRawReturnTypesMap.size > 0 ? importedRawReturnTypesMap : undefined,
          undefined,
          undefined,
          processCallsTimingSink,
          processCallsQueryCache,
        ),
      );
      crossFileResolved++;
    }

    if (crossFileResolved >= MAX_CROSS_FILE_REPROCESS) {
      if (isDev)
        console.log(`⚠️ Cross-file re-resolution capped at ${MAX_CROSS_FILE_REPROCESS} files`);
      break;
    }
  }

  astCache.clear();

  if (isDev) {
    const elapsed = Date.now() - crossFileStart;
    const totalElapsed = Date.now() - pipelineStart;
    const reResolutionPct = totalElapsed > 0 ? ((elapsed / totalElapsed) * 100).toFixed(1) : '0';
    console.log(
      `🔗 Cross-file re-resolution: ${crossFileResolved} candidates re-processed` +
        ` in ${elapsed}ms (${reResolutionPct}% of total ingestion time so far)`,
    );
  }

  return finish(crossFileResolved);
}
