import { describe, it, expect, vi, beforeEach } from 'vitest';
import { BindingAccumulator } from '../../src/core/ingestion/binding-accumulator.js';

// Mock the compatibility propagation module so we can control whether the
// narrowed fallback step throws or returns cleanly. The `crossFilePhase` only
// depends on this one external symbol — nothing else has to be stubbed.
vi.mock('../../src/core/ingestion/pipeline-phases/cross-file-impl.js', () => ({
  runCrossFileBindingPropagation: vi.fn(),
}));

import { runCrossFileBindingPropagation } from '../../src/core/ingestion/pipeline-phases/cross-file-impl.js';
import { crossFilePhase } from '../../src/core/ingestion/pipeline-phases/cross-file.js';
import type {
  PipelineContext,
  PhaseResult,
} from '../../src/core/ingestion/pipeline-phases/types.js';
import type { ParseOutput } from '../../src/core/ingestion/pipeline-phases/parse.js';

const runCrossFileMock = vi.mocked(runCrossFileBindingPropagation);

function makeCtx(options?: PipelineContext['options']): PipelineContext {
  return {
    repoPath: '/tmp/repo',
    // Cast — the body never touches graph methods on the happy/error paths
    // this test exercises (the propagation call is stubbed).
    graph: {} as PipelineContext['graph'],
    onProgress: () => {},
    pipelineStart: 0,
    ...(options !== undefined ? { options } : {}),
  };
}

function makeParseOutput(acc: BindingAccumulator, metrics?: ParseOutput['metrics']): ParseOutput {
  return {
    exportedTypeMap: new Map(),
    allFetchCalls: [],
    allExtractedRoutes: [],
    allDecoratorRoutes: [],
    allToolDefs: [],
    allORMQueries: [],
    bindingAccumulator: acc,
    // Cast — the body forwards this to the (mocked) propagation fn but
    // never inspects it.
    resolutionContext: {} as ParseOutput['resolutionContext'],
    allPaths: [],
    allPathSet: new Set(),
    totalFiles: 0,
    parsedFiles: [],
    usedWorkerPool: false,
    ...(metrics !== undefined ? { metrics } : {}),
  };
}

function makeDeps(
  acc: BindingAccumulator,
  metrics?: ParseOutput['metrics'],
): ReadonlyMap<string, PhaseResult<unknown>> {
  return new Map<string, PhaseResult<unknown>>([
    [
      'parse',
      {
        phaseName: 'parse',
        output: makeParseOutput(acc, metrics),
        durationMs: 0,
      },
    ],
  ]);
}

describe('crossFilePhase compatibility fallback', () => {
  beforeEach(() => {
    runCrossFileMock.mockReset();
  });

  it('disposes the binding accumulator on the happy path', async () => {
    runCrossFileMock.mockResolvedValueOnce({
      filesReprocessed: 7,
      metrics: { timings: { processCallsMs: 12.3 }, counters: { filesReprocessed: 7 } },
    });

    const acc = new BindingAccumulator();
    acc.appendFile('src/a.ts', [{ scope: '', varName: 'x', typeName: 'X' }]);
    expect(acc.disposed).toBe(false);

    const result = await crossFilePhase.execute(makeCtx(), makeDeps(acc));

    expect(result.filesReprocessed).toBe(7);
    expect(result.metrics.timings.processCallsMs).toBe(12.3);
    expect(acc.disposed).toBe(true);
    // Post-dispose contract holds.
    expect(acc.fileCount).toBe(0);
    expect(acc.totalBindings).toBe(0);
  });

  it('disposes the binding accumulator even when propagation throws', async () => {
    // Error-injection: the leak-on-throw gap — without the finally block,
    // the accumulator would stay live (and reachable via the closed-over
    // ParseOutput) until GC. With the finally block, dispose runs on the
    // unwind and the heap is released regardless.
    const boom = new Error('cross-file propagation exploded');
    runCrossFileMock.mockRejectedValueOnce(boom);

    const acc = new BindingAccumulator();
    acc.appendFile('src/a.ts', [{ scope: '', varName: 'x', typeName: 'X' }]);

    await expect(crossFilePhase.execute(makeCtx(), makeDeps(acc))).rejects.toBe(boom);

    expect(acc.disposed).toBe(true);
    expect(acc.fileCount).toBe(0);
    expect(acc.totalBindings).toBe(0);
  });

  it('skips compatibility propagation by option while still disposing the accumulator', async () => {
    const acc = new BindingAccumulator();
    acc.appendFile('src/a.ts', [{ scope: '', varName: 'x', typeName: 'X' }]);

    const result = await crossFilePhase.execute(
      makeCtx({ skipCompatibilityCrossFile: true }),
      makeDeps(acc),
    );

    expect(runCrossFileMock).not.toHaveBeenCalled();
    expect(result.filesReprocessed).toBe(0);
    expect(result.metrics.counters.skipReason).toBe('disabled-by-pipeline-option');
    expect(acc.disposed).toBe(true);
  });

  it('auto-skips compatibility propagation when every parseable file has AST-reused scope facts', async () => {
    const acc = new BindingAccumulator();
    acc.appendFile('src/a.ts', [{ scope: '', varName: 'x', typeName: 'X' }]);

    const result = await crossFilePhase.execute(
      makeCtx(),
      makeDeps(acc, {
        timings: {},
        counters: {
          parseableFiles: 3,
          scopeParsedFiles: 3,
          scopeExtractionAstReusedFiles: 3,
          scopeExtractionCompatibilityFiles: 0,
          scopeExtractionNoHookFiles: 0,
          scopeExtractionFailedFiles: 0,
        },
      }),
    );

    expect(runCrossFileMock).not.toHaveBeenCalled();
    expect(result.filesReprocessed).toBe(0);
    expect(result.metrics.counters.skipReason).toBe('covered-by-ast-reused-scope-resolution');
    expect(acc.disposed).toBe(true);
  });

  it('keeps compatibility propagation when any parseable file lacks AST-reused scope facts', async () => {
    runCrossFileMock.mockResolvedValueOnce({
      filesReprocessed: 2,
      metrics: { timings: { processCallsMs: 4.5 }, counters: { filesReprocessed: 2 } },
    });
    const acc = new BindingAccumulator();
    acc.appendFile('src/a.ts', [{ scope: '', varName: 'x', typeName: 'X' }]);

    const result = await crossFilePhase.execute(
      makeCtx(),
      makeDeps(acc, {
        timings: {},
        counters: {
          parseableFiles: 3,
          scopeParsedFiles: 2,
          scopeExtractionAstReusedFiles: 2,
          scopeExtractionCompatibilityFiles: 0,
          scopeExtractionNoHookFiles: 1,
          scopeExtractionFailedFiles: 0,
        },
      }),
    );

    expect(runCrossFileMock).toHaveBeenCalledTimes(1);
    expect(result.filesReprocessed).toBe(2);
    expect(acc.disposed).toBe(true);
  });
});
