/**
 * Parse implementation — chunked parse + resolve loop.
 *
 * This is the core parsing engine of the ingestion pipeline. It reads
 * source files in byte-budget chunks (~20MB each), parses via worker
 * pool, resolves imports/calls/heritage per
 * chunk, and synthesizes wildcard import bindings.
 *
 * Consumed by the parse phase (`parse.ts`) — the phase file handles
 * dependency wiring while the heavy implementation lives here.
 *
 * @module
 */

import {
  BindingAccumulator,
  enrichExportedTypeMap,
  type BindingEntry,
} from '../binding-accumulator.js';
import { processParsing } from '../parsing-processor.js';
import { processImportsFromExtracted, buildImportResolutionContext } from '../import-processor.js';
import { EMPTY_INDEX } from '../import-resolvers/utils.js';
import {
  processCallsFromExtracted,
  processAssignmentsFromExtracted,
  processRoutesFromExtracted,
  seedCrossFileReceiverTypes,
  buildExportedTypeMapFromGraph,
  type ExportedTypeMap,
} from '../call-processor.js';
import { buildHeritageMap } from '../model/heritage-map.js';
import {
  processHeritageFromExtracted,
  getHeritageStrategyForLanguage,
} from '../heritage-processor.js';
import { createResolutionContext } from '../model/resolution-context.js';
import { type PipelineProgress, getLanguageFromFilename } from 'avmatrix-shared';
import { readFileContents } from '../filesystem-walker.js';
import { isLanguageAvailable } from '../../tree-sitter/parser-loader.js';
import { createWorkerPool } from '../workers/worker-pool.js';
import type { WorkerPool } from '../workers/worker-pool.js';
import type {
  ExtractedAssignment,
  ExtractedCall,
  ExtractedDecoratorRoute,
  ExtractedFetchCall,
  ExtractedORMQuery,
  ExtractedRoute,
  ExtractedToolDef,
  FileConstructorBindings,
} from '../workers/parse-worker.js';
import type { ExtractedHeritage } from '../model/heritage-map.js';
import type { KnowledgeGraph } from '../../graph/types.js';
import fs from 'node:fs';
import path from 'node:path';
import { performance } from 'node:perf_hooks';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { isDev } from '../utils/env.js';
import { synthesizeWildcardImportBindings, needsSynthesis } from './wildcard-synthesis.js';
import type { ParseMetrics, ParseTimingBreakdown } from '../../analyze/analyze-metrics.js';
import { roundMs } from '../../analyze/analyze-metrics.js';

// ── Constants ──────────────────────────────────────────────────────────────

/** Max bytes of source content to load per parse chunk. */
const CHUNK_BYTE_BUDGET = 20 * 1024 * 1024; // 20MB

// ── Main parse + resolve function ──────────────────────────────────────────

type ScannedFile = { path: string; size: number };
type ProgressFn = (progress: PipelineProgress) => void;

/**
 * Chunked parse + resolve loop.
 *
 * Reads source in byte-budget chunks (~20MB each). For each chunk:
 * 1. Parse via worker pool
 * 2. Resolve imports from extracted data
 * 3. Synthesize wildcard import bindings (Go/Ruby/C++/Swift/Python)
 * 4. Resolve heritage + routes per chunk; defer worker CALLS until all chunks
 *    have contributed heritage so interface-dispatch implementor map is complete
 * 5. Collect TypeEnv bindings for cross-file propagation
 */
export async function runChunkedParseAndResolve(
  graph: KnowledgeGraph,
  scannedFiles: ScannedFile[],
  allPaths: string[],
  totalFiles: number,
  repoPath: string,
  pipelineStart: number,
  onProgress: ProgressFn,
): Promise<{
  exportedTypeMap: ExportedTypeMap;
  allFetchCalls: ExtractedFetchCall[];
  allExtractedRoutes: ExtractedRoute[];
  allDecoratorRoutes: ExtractedDecoratorRoute[];
  allToolDefs: ExtractedToolDef[];
  allORMQueries: ExtractedORMQuery[];
  bindingAccumulator: BindingAccumulator;
  resolutionContext: ReturnType<typeof createResolutionContext>;
  usedWorkerPool: boolean;
  metrics: ParseMetrics;
}> {
  const ctx = createResolutionContext();
  const symbolTable = ctx.model.symbols;
  const metrics: ParseMetrics = { timings: {}, counters: {} };

  const parseableScanned = scannedFiles.filter((f) => {
    const lang = getLanguageFromFilename(f.path);
    return lang && isLanguageAvailable(lang);
  });

  // Warn about files skipped due to unavailable parsers
  const skippedByLang = new Map<string, number>();
  let parserUnavailableFiles = 0;
  for (const f of scannedFiles) {
    const lang = getLanguageFromFilename(f.path);
    if (lang && !isLanguageAvailable(lang)) {
      parserUnavailableFiles++;
      skippedByLang.set(lang, (skippedByLang.get(lang) || 0) + 1);
    }
  }
  for (const [lang, count] of skippedByLang) {
    console.warn(
      `Skipping ${count} ${lang} file(s) — ${lang} parser not available (native binding may not have built). Try: npm rebuild tree-sitter-${lang}`,
    );
  }

  const totalParseable = parseableScanned.length;
  const totalParseableBytes = parseableScanned.reduce((s, f) => s + f.size, 0);

  if (totalParseable === 0) {
    onProgress({
      phase: 'parsing',
      percent: 82,
      message: 'No parseable files found — skipping parsing phase',
      stats: { filesProcessed: 0, totalFiles: 0, nodesCreated: graph.nodeCount },
    });
  }

  // Build byte-budget chunks
  const chunks: string[][] = [];
  let currentChunk: string[] = [];
  let currentBytes = 0;
  for (const file of parseableScanned) {
    if (currentChunk.length > 0 && currentBytes + file.size > CHUNK_BYTE_BUDGET) {
      chunks.push(currentChunk);
      currentChunk = [];
      currentBytes = 0;
    }
    currentChunk.push(file.path);
    currentBytes += file.size;
  }
  if (currentChunk.length > 0) chunks.push(currentChunk);

  const numChunks = chunks.length;

  if (isDev) {
    const totalMB = totalParseableBytes / (1024 * 1024);
    console.log(
      `📂 Scan: ${totalFiles} paths, ${totalParseable} parseable (${totalMB.toFixed(0)}MB), ${numChunks} chunks @ ${CHUNK_BYTE_BUDGET / (1024 * 1024)}MB budget`,
    );
  }

  onProgress({
    phase: 'parsing',
    percent: 20,
    message: `Parsing ${totalParseable} files in ${numChunks} chunk${numChunks !== 1 ? 's' : ''}...`,
    stats: { filesProcessed: 0, totalFiles: totalParseable, nodesCreated: graph.nodeCount },
  });

  // Create worker pool once, reuse across chunks. Worker parsing is the
  // canonical full-analyze path; startup failures are surfaced instead of
  // silently switching to the removed sequential parser.
  const workerPool = totalParseable > 0 ? createCanonicalParseWorkerPool(metrics) : undefined;

  let filesParsedSoFar = 0;

  // Build import resolution context once — suffix index, file lists, resolve cache.
  const importCtx = buildImportResolutionContext(allPaths);
  const allPathObjects = allPaths.map((p) => ({ path: p }));

  const chunkNeedsSynthesis = chunks.map((paths) =>
    paths.some((p) => {
      const lang = getLanguageFromFilename(p);
      return lang != null && needsSynthesis(lang);
    }),
  );
  const exportedTypeMap: ExportedTypeMap = new Map();
  const bindingAccumulator = new BindingAccumulator();
  // Tracks whether per-chunk wildcard-binding synthesis already
  // ran, so the unconditional final call below can be skipped when redundant.
  // synthesizeWildcardImportBindings is graph-global; once any chunk runs it
  // after parsing wildcard files, later non-wildcard chunks add no work for
  // it, and later wildcard chunks re-run it themselves.
  let hasSynthesized = false;
  const allFetchCalls: ExtractedFetchCall[] = [];
  const allExtractedRoutes: ExtractedRoute[] = [];
  const allDecoratorRoutes: ExtractedDecoratorRoute[] = [];
  const allToolDefs: ExtractedToolDef[] = [];
  const allORMQueries: ExtractedORMQuery[] = [];
  const deferredWorkerCalls: ExtractedCall[] = [];
  const deferredWorkerHeritage: ExtractedHeritage[] = [];
  const deferredConstructorBindings: FileConstructorBindings[] = [];
  const deferredAssignments: ExtractedAssignment[] = [];

  try {
    for (let chunkIdx = 0; chunkIdx < numChunks; chunkIdx++) {
      const chunkPaths = chunks[chunkIdx];

      const chunkContents = await timeParseStep(metrics, 'readContentsMs', () =>
        readFileContents(repoPath, chunkPaths),
      );
      const chunkFiles = chunkPaths
        .filter((p) => chunkContents.has(p))
        .map((p) => ({ path: p, content: chunkContents.get(p)! }));

      const chunkWorkerData = await timeParseStep(metrics, 'workerParseMs', () =>
        processParsing(graph, chunkFiles, symbolTable, workerPool!, (current, _total, filePath) => {
          const globalCurrent = filesParsedSoFar + current;
          const parsingProgress = 20 + (globalCurrent / totalParseable) * 62;
          onProgress({
            phase: 'parsing',
            percent: Math.round(parsingProgress),
            message: `Parsing chunk ${chunkIdx + 1}/${numChunks}...`,
            detail: filePath,
            stats: {
              filesProcessed: globalCurrent,
              totalFiles: totalParseable,
              nodesCreated: graph.nodeCount,
            },
          });
        }),
      );

      const chunkBasePercent = 20 + (filesParsedSoFar / totalParseable) * 62;

      {
        await timeParseStep(metrics, 'importResolveMs', () =>
          processImportsFromExtracted(
            graph,
            allPathObjects,
            chunkWorkerData.imports,
            ctx,
            (current, total) => {
              onProgress({
                phase: 'parsing',
                percent: Math.round(chunkBasePercent),
                message: `Resolving imports (chunk ${chunkIdx + 1}/${numChunks})...`,
                detail: `${current}/${total} files`,
                stats: {
                  filesProcessed: filesParsedSoFar,
                  totalFiles: totalParseable,
                  nodesCreated: graph.nodeCount,
                },
              });
            },
            repoPath,
            importCtx,
          ),
        );
        if (chunkNeedsSynthesis[chunkIdx]) {
          timeParseStepSync(metrics, 'wildcardSynthesisMs', () =>
            synthesizeWildcardImportBindings(graph, ctx),
          );
          hasSynthesized = true;
        }
        if (exportedTypeMap.size > 0 && ctx.namedImportMap.size > 0) {
          const { enrichedCount } = seedCrossFileReceiverTypes(
            chunkWorkerData.calls,
            ctx.namedImportMap,
            exportedTypeMap,
          );
          if (isDev && enrichedCount > 0) {
            console.log(
              `🔗 E1: Seeded ${enrichedCount} cross-file receiver types (chunk ${chunkIdx + 1})`,
            );
          }
        }
        for (const item of chunkWorkerData.calls) deferredWorkerCalls.push(item);
        for (const item of chunkWorkerData.heritage) deferredWorkerHeritage.push(item);
        for (const item of chunkWorkerData.constructorBindings)
          deferredConstructorBindings.push(item);
        if (chunkWorkerData.assignments?.length) {
          for (const item of chunkWorkerData.assignments) deferredAssignments.push(item);
        }

        await Promise.all([
          timeParseStep(metrics, 'heritageResolveMs', () =>
            processHeritageFromExtracted(graph, chunkWorkerData.heritage, ctx, (current, total) => {
              onProgress({
                phase: 'parsing',
                percent: Math.round(chunkBasePercent),
                message: `Resolving heritage (chunk ${chunkIdx + 1}/${numChunks})...`,
                detail: `${current}/${total} records`,
                stats: {
                  filesProcessed: filesParsedSoFar,
                  totalFiles: totalParseable,
                  nodesCreated: graph.nodeCount,
                },
              });
            }),
          ),
          timeParseStep(metrics, 'routeResolveMs', () =>
            processRoutesFromExtracted(
              graph,
              chunkWorkerData.routes ?? [],
              ctx,
              (current, total) => {
                onProgress({
                  phase: 'parsing',
                  percent: Math.round(chunkBasePercent),
                  message: `Resolving routes (chunk ${chunkIdx + 1}/${numChunks})...`,
                  detail: `${current}/${total} routes`,
                  stats: {
                    filesProcessed: filesParsedSoFar,
                    totalFiles: totalParseable,
                    nodesCreated: graph.nodeCount,
                  },
                });
              },
            ),
          ),
        ]);

        if (chunkWorkerData.fileScopeBindings?.length) {
          for (const { filePath, bindings } of chunkWorkerData.fileScopeBindings) {
            if (typeof filePath !== 'string' || filePath.length === 0) continue;
            if (!Array.isArray(bindings)) continue;
            const entries: BindingEntry[] = [];
            for (const tuple of bindings) {
              if (!Array.isArray(tuple) || tuple.length !== 2) continue;
              const [varName, typeName] = tuple;
              if (typeof varName !== 'string' || typeof typeName !== 'string') continue;
              entries.push({ scope: '', varName, typeName });
            }
            if (entries.length > 0) {
              bindingAccumulator.appendFile(filePath, entries);
            }
          }
        }
        if (chunkWorkerData.fetchCalls?.length) {
          for (const item of chunkWorkerData.fetchCalls) allFetchCalls.push(item);
        }
        if (chunkWorkerData.routes?.length) {
          for (const item of chunkWorkerData.routes) allExtractedRoutes.push(item);
        }
        if (chunkWorkerData.decoratorRoutes?.length) {
          for (const item of chunkWorkerData.decoratorRoutes) allDecoratorRoutes.push(item);
        }
        if (chunkWorkerData.toolDefs?.length) {
          for (const item of chunkWorkerData.toolDefs) allToolDefs.push(item);
        }
        if (chunkWorkerData.ormQueries?.length) {
          for (const item of chunkWorkerData.ormQueries) allORMQueries.push(item);
        }
      }

      filesParsedSoFar += chunkFiles.length;
    }

    const fullWorkerHeritageMap =
      deferredWorkerHeritage.length > 0
        ? buildHeritageMap(deferredWorkerHeritage, ctx, getHeritageStrategyForLanguage)
        : undefined;

    if (deferredWorkerCalls.length > 0) {
      await timeParseStep(metrics, 'callResolveMs', () =>
        processCallsFromExtracted(
          graph,
          deferredWorkerCalls,
          ctx,
          (current, total) => {
            onProgress({
              phase: 'parsing',
              percent: 82,
              message: 'Resolving calls (all chunks)...',
              detail: `${current}/${total} files`,
              stats: {
                filesProcessed: filesParsedSoFar,
                totalFiles: totalParseable,
                nodesCreated: graph.nodeCount,
              },
            });
          },
          deferredConstructorBindings.length > 0 ? deferredConstructorBindings : undefined,
          fullWorkerHeritageMap,
          bindingAccumulator,
        ),
      );
    }

    if (deferredAssignments.length > 0) {
      timeParseStepSync(metrics, 'assignmentResolveMs', () =>
        processAssignmentsFromExtracted(
          graph,
          deferredAssignments,
          ctx,
          deferredConstructorBindings.length > 0 ? deferredConstructorBindings : undefined,
          bindingAccumulator,
        ),
      );
    }
  } finally {
    await workerPool?.terminate();
  }

  const enriched = timeParseStepSync(metrics, 'exportedTypeMapEnrichMs', () => {
    bindingAccumulator.finalize();
    return enrichExportedTypeMap(bindingAccumulator, graph, exportedTypeMap);
  });
  if (isDev && enriched > 0) {
    console.log(
      `🔗 Worker TypeEnv enrichment: ${enriched} fixpoint-inferred exports added to ExportedTypeMap`,
    );
  }

  if (!hasSynthesized) {
    const synthesized = timeParseStepSync(metrics, 'wildcardSynthesisMs', () =>
      synthesizeWildcardImportBindings(graph, ctx),
    );
    if (isDev && synthesized > 0) {
      console.log(
        `🔗 Synthesized ${synthesized} additional wildcard import bindings (Go/Ruby/C++/Swift/Python)`,
      );
    }
  }

  // Worker-path enrichment: if exportedTypeMap is empty (e.g. the worker pool
  // built TypeEnv inside workers without access to SymbolTable), reconstruct
  // the map from graph nodes + SymbolTable here in the main thread before
  // handing the (now read-only) map to downstream phases. Doing it here means
  // crossFile receives a fully-populated map and never needs to mutate it for
  // initial-graph enrichment.
  if (exportedTypeMap.size === 0 && graph.nodeCount > 0) {
    const graphExports = timeParseStepSync(metrics, 'exportedTypeMapEnrichMs', () =>
      buildExportedTypeMapFromGraph(graph, ctx.model.symbols),
    );
    for (const [fp, exports] of graphExports) exportedTypeMap.set(fp, exports);
  }

  allPathObjects.length = 0;
  // Safe to reset importCtx caches here: `importCtx` (ImportResolutionContext)
  // is a scratch workspace used only during import path resolution. The
  // `resolutionContext` (`ctx`) returned below is a distinct object — it owns
  // the fully-populated, post-parse `importMap` / `namedImportMap` /
  // `packageMap` / `moduleAliasMap` / `model`, and never references
  // `importCtx`. Cross-file re-resolution in cross-file-impl.ts consumes only
  // `ctx` (via `processCalls`), so clearing the suffix index / resolveCache /
  // normalizedFileList here cannot lose import matches downstream.
  importCtx.resolveCache.clear();
  importCtx.index = EMPTY_INDEX;
  importCtx.normalizedFileList = [];

  metrics.counters.parseableFiles = totalParseable;
  metrics.counters.totalParseableMB = roundMs(totalParseableBytes / (1024 * 1024));
  metrics.counters.parseChunkCount = numChunks;
  metrics.counters.workerCount = metrics.counters.workerCount ?? 0;
  metrics.counters.parserUnavailableFiles = parserUnavailableFiles;

  return {
    exportedTypeMap,
    allFetchCalls,
    allExtractedRoutes,
    allDecoratorRoutes,
    allToolDefs,
    allORMQueries,
    bindingAccumulator,
    resolutionContext: ctx,
    // Whether a worker pool was live for this run. False only means there
    // were no parseable files.
    usedWorkerPool: workerPool !== undefined,
    metrics,
  };
}

function createCanonicalParseWorkerPool(metrics: ParseMetrics): WorkerPool {
  try {
    const workerUrl = resolveParseWorkerUrl();
    const workerPool = createWorkerPool(workerUrl);
    metrics.counters.workerCount = workerPool.size;
    return workerPool;
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    throw new Error(
      'Worker pool creation failed. Full analyze requires parse workers and no longer ' +
        `falls back to sequential parsing. ${message}`,
    );
  }
}

function resolveParseWorkerUrl(): URL {
  const sourceJsWorker = new URL('../workers/parse-worker.js', import.meta.url);
  const sourceJsPath = fileURLToPath(sourceJsWorker);
  if (fs.existsSync(sourceJsPath)) return sourceJsWorker;

  const thisDir = fileURLToPath(new URL('.', import.meta.url));
  const distWorker = path.resolve(
    thisDir,
    '..',
    '..',
    '..',
    '..',
    'dist',
    'core',
    'ingestion',
    'workers',
    'parse-worker.js',
  );
  if (fs.existsSync(distWorker)) return pathToFileURL(distWorker);

  throw new Error(
    `Parse worker script not found. Checked ${sourceJsPath} and ${distWorker}. ` +
      'Run `npm run build` in avmatrix or fix the package install before analyzing.',
  );
}

async function timeParseStep<T>(
  metrics: ParseMetrics,
  key: keyof ParseTimingBreakdown,
  fn: () => Promise<T>,
): Promise<T> {
  const start = performance.now();
  try {
    return await fn();
  } finally {
    addParseTiming(metrics, key, performance.now() - start);
  }
}

function timeParseStepSync<T>(
  metrics: ParseMetrics,
  key: keyof ParseTimingBreakdown,
  fn: () => T,
): T {
  const start = performance.now();
  try {
    return fn();
  } finally {
    addParseTiming(metrics, key, performance.now() - start);
  }
}

function addParseTiming(
  metrics: ParseMetrics,
  key: keyof ParseTimingBreakdown,
  durationMs: number,
): void {
  metrics.timings[key] = roundMs((metrics.timings[key] ?? 0) + durationMs);
}
