import { Worker } from 'node:worker_threads';
import os from 'node:os';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export interface WorkerPool {
  /**
   * Dispatch items across workers. Items are split into stable work units,
   * idle workers pull the next unit, and results are returned in input order
   * regardless of worker completion order.
   */
  dispatch<TInput, TResult>(
    items: TInput[],
    onProgress?: (filesProcessed: number) => void,
    options?: WorkerDispatchOptions<TInput>,
  ): Promise<TResult[]>;

  /** Terminate all workers. Must be called when done. */
  terminate(): Promise<void>;

  /** Number of workers in the pool */
  readonly size: number;
}

/** Message shapes sent back by worker threads. */
type WorkerOutgoingMessage =
  | { type: 'progress'; filesProcessed: number }
  | { type: 'heartbeat'; filePath?: string; filesProcessed?: number }
  | { type: 'warning'; message: string }
  | { type: 'sub-batch-done' }
  | { type: 'error'; error: string }
  | { type: 'result'; data: unknown };

/**
 * Default byte budget for one dynamic work unit.
 * Keeps structured-clone and per-unit wall time bounded without changing output.
 */
const DEFAULT_WORK_UNIT_BYTES = 2 * 1024 * 1024;

/** Max files per work unit when file contents are small. */
const DEFAULT_MAX_FILES_PER_UNIT = 50;

/**
 * Inactivity timeout. A unit may take longer than this in total as long as the
 * worker keeps emitting heartbeat/progress messages.
 */
const DEFAULT_INACTIVITY_TIMEOUT_MS = 30_000;

/** How many times to split/retry a failed multi-file unit before surfacing. */
const DEFAULT_MAX_RETRIES = 4;

export interface WorkerDispatchOptions<TInput> {
  /** Target content bytes per work unit. */
  targetUnitBytes?: number;
  /** Hard cap for files per work unit. */
  maxFilesPerUnit?: number;
  /** Worker inactivity timeout in milliseconds. */
  inactivityTimeoutMs?: number;
  /** Max split/retry depth for failed units. */
  maxRetries?: number;
  /** Optional item-size override. */
  getItemSize?: (item: TInput) => number;
  /** Optional item-path override for diagnostics. */
  getItemPath?: (item: TInput) => string | undefined;
  /** Emit per-unit scheduler diagnostics. */
  verbose?: boolean;
}

interface WorkUnit<TInput> {
  unitId: number;
  startIndex: number;
  items: TInput[];
  totalBytes: number;
  languageBreakdown: Record<string, number>;
  retryDepth: number;
}

interface CompletedResult<TResult> {
  startIndex: number;
  unitId: number;
  result: TResult;
}

interface WorkerState<TInput> {
  index: number;
  worker: Worker;
  current?: WorkUnit<TInput>;
  startedAtMs?: number;
  timer?: ReturnType<typeof setTimeout>;
  lastActiveFile?: string;
  cleanup?: () => void;
}

/**
 * Create a pool of worker threads.
 */
export const createWorkerPool = (workerUrl: URL, poolSize?: number): WorkerPool => {
  // Validate worker script exists before spawning to prevent uncaught
  // MODULE_NOT_FOUND crashes in worker threads (e.g. when running from src/ via vitest)
  const workerPath = fileURLToPath(workerUrl);
  if (!fs.existsSync(workerPath)) {
    throw new Error(`Worker script not found: ${workerPath}`);
  }

  const size = poolSize ?? Math.min(8, Math.max(1, os.cpus().length - 1));
  const workers: Worker[] = [];
  let terminated = false;

  for (let i = 0; i < size; i++) {
    workers.push(new Worker(workerUrl));
  }

  const dispatch = <TInput, TResult>(
    items: TInput[],
    onProgress?: (filesProcessed: number) => void,
    options: WorkerDispatchOptions<TInput> = {},
  ): Promise<TResult[]> => {
    if (items.length === 0) return Promise.resolve([]);
    if (terminated || workers.length === 0) {
      return Promise.reject(new Error('Worker pool has been terminated or has no workers.'));
    }

    return new Promise<TResult[]>((resolve, reject) => {
      const pending = createWorkUnits(items, options);
      const completed: Array<CompletedResult<TResult>> = [];
      const unitProgress = new Map<number, number>();
      const workerStates: Array<WorkerState<TInput>> = workers.map((worker, index) => ({
        index,
        worker,
      }));
      const inactivityTimeoutMs =
        options.inactivityTimeoutMs ?? DEFAULT_INACTIVITY_TIMEOUT_MS;
      const maxRetries = options.maxRetries ?? DEFAULT_MAX_RETRIES;
      let nextUnitId = pending.length;
      let remainingUnits = pending.length;
      let completedItems = 0;
      let settled = false;

      const cleanupAll = () => {
        for (const state of workerStates) {
          if (state.timer) clearTimeout(state.timer);
          state.timer = undefined;
          state.cleanup?.();
          state.cleanup = undefined;
        }
      };

      const reportProgress = () => {
        if (!onProgress) return;
        let inFlight = 0;
        for (const count of unitProgress.values()) inFlight += count;
        onProgress(Math.min(items.length, completedItems + inFlight));
      };

      const maybeResolve = () => {
        if (settled || remainingUnits !== 0) return;
        settled = true;
        cleanupAll();
        completed.sort((a, b) => a.startIndex - b.startIndex || a.unitId - b.unitId);
        resolve(completed.map((entry) => entry.result));
      };

      const failDispatch = (error: Error) => {
        if (settled) return;
        settled = true;
        cleanupAll();
        reject(error);
      };

      const attachHandlers = (state: WorkerState<TInput>) => {
        const handler = (msg: WorkerOutgoingMessage) => {
          if (settled) return;
          const unit = state.current;

          if (msg.type === 'heartbeat') {
            if (msg.filePath) state.lastActiveFile = msg.filePath;
            if (unit && typeof msg.filesProcessed === 'number') {
              unitProgress.set(unit.unitId, msg.filesProcessed);
              reportProgress();
            }
            resetInactivityTimer(state);
            return;
          }

          if (msg.type === 'progress') {
            if (unit) {
              unitProgress.set(unit.unitId, msg.filesProcessed);
              reportProgress();
            }
            resetInactivityTimer(state);
            return;
          }

          if (msg.type === 'warning') {
            console.warn(msg.message);
            resetInactivityTimer(state);
            return;
          }

          if (msg.type === 'sub-batch-done') {
            resetInactivityTimer(state);
            state.worker.postMessage({ type: 'flush' });
            return;
          }

          if (msg.type === 'error') {
            handleUnitFailure(state, new Error(`Worker ${state.index} error: ${msg.error}`));
            return;
          }

          if (msg.type === 'result') {
            if (!unit) {
              failDispatch(new Error(`Worker ${state.index} returned a result without a unit.`));
              return;
            }
            const elapsedMs = state.startedAtMs ? Date.now() - state.startedAtMs : 0;
            if (state.timer) clearTimeout(state.timer);
            state.timer = undefined;
            unitProgress.delete(unit.unitId);
            completedItems += unit.items.length;
            completed.push({
              startIndex: unit.startIndex,
              unitId: unit.unitId,
              result: msg.data as TResult,
            });
            if (options.verbose) {
              console.log(
                `[parse-worker] worker=${state.index} unit=${unit.unitId} files=${unit.items.length} bytes=${formatBytes(unit.totalBytes)} languages=${formatLanguageBreakdown(unit.languageBreakdown)} retry=${unit.retryDepth} elapsed=${elapsedMs}ms`,
              );
            }
            state.current = undefined;
            state.startedAtMs = undefined;
            state.lastActiveFile = undefined;
            remainingUnits--;
            reportProgress();
            assignNext(state);
            maybeResolve();
          }
        };

        const errorHandler = (err: Error) => {
          if (settled) return;
          if (state.current) {
            handleUnitFailure(state, err, true);
          } else {
            failDispatch(err);
          }
        };

        const exitHandler = (code: number) => {
          if (settled) return;
          const error = new Error(
            `Worker ${state.index} exited with code ${code}. Likely OOM or native addon failure.`,
          );
          if (state.current) {
            handleUnitFailure(state, error, true);
          } else {
            failDispatch(error);
          }
        };

        state.worker.on('message', handler);
        state.worker.once('error', errorHandler);
        state.worker.once('exit', exitHandler);
        state.cleanup = () => {
          state.worker.removeListener('message', handler);
          state.worker.removeListener('error', errorHandler);
          state.worker.removeListener('exit', exitHandler);
        };
      };

      const replaceWorker = (state: WorkerState<TInput>) => {
        state.cleanup?.();
        state.cleanup = undefined;
        void state.worker.terminate().catch(() => undefined);
        const replacement = new Worker(workerUrl);
        workers[state.index] = replacement;
        state.worker = replacement;
        attachHandlers(state);
      };

      const resetInactivityTimer = (state: WorkerState<TInput>) => {
        if (!state.current) return;
        if (state.timer) clearTimeout(state.timer);
        state.timer = setTimeout(() => {
          const unit = state.current;
          if (!unit || settled) return;
          const message =
            `Worker ${state.index} inactive for ${Math.round(inactivityTimeoutMs / 1000)}s ` +
            `while processing unit ${unit.unitId} (${unit.items.length} files, ` +
            `${formatBytes(unit.totalBytes)}, languages=${formatLanguageBreakdown(unit.languageBreakdown)}` +
            `${state.lastActiveFile ? `, lastFile=${state.lastActiveFile}` : ''}).`;
          handleUnitFailure(state, new Error(message), true);
        }, inactivityTimeoutMs);
      };

      const enqueueSplitRetry = (unit: WorkUnit<TInput>): boolean => {
        if (unit.retryDepth >= maxRetries || unit.items.length <= 1) return false;
        const midpoint = Math.ceil(unit.items.length / 2);
        const leftItems = unit.items.slice(0, midpoint);
        const rightItems = unit.items.slice(midpoint);
        const childDepth = unit.retryDepth + 1;
      const children = [
        createWorkUnit(leftItems, unit.startIndex, nextUnitId++, childDepth, options),
          createWorkUnit(
            rightItems,
            unit.startIndex + leftItems.length,
            nextUnitId++,
            childDepth,
            options,
          ),
      ].filter((child) => child.items.length > 0);
      pending.unshift(...children);
      remainingUnits += children.length - 1;
      if (options.verbose) {
        console.warn(
          `[parse-worker] retry unit=${unit.unitId} files=${unit.items.length} retry=${unit.retryDepth}->${childDepth} split=${children.map((child) => `${child.unitId}:${child.items.length}`).join(',')}`,
        );
      }
      return true;
    };

      const handleUnitFailure = (
        state: WorkerState<TInput>,
        error: Error,
        restartWorker = false,
      ) => {
        const unit = state.current;
        if (!unit || settled) return;
        if (state.timer) clearTimeout(state.timer);
        state.timer = undefined;
        unitProgress.delete(unit.unitId);
        state.current = undefined;
        state.startedAtMs = undefined;
        state.lastActiveFile = undefined;

        if (restartWorker) replaceWorker(state);

        if (!enqueueSplitRetry(unit)) {
          failDispatch(error);
          return;
        }

        assignNext(state);
      };

      const assignNext = (state: WorkerState<TInput>) => {
        if (settled || state.current) return;
        const unit = pending.shift();
        if (!unit) return;
        state.current = unit;
        state.startedAtMs = Date.now();
        state.lastActiveFile = firstItemPath(unit.items, options);
        unitProgress.set(unit.unitId, 0);
        resetInactivityTimer(state);
        state.worker.postMessage({ type: 'sub-batch', files: unit.items });
      };

      for (const state of workerStates) attachHandlers(state);
      for (const state of workerStates) assignNext(state);
    });
  };

  const terminate = async (): Promise<void> => {
    terminated = true;
    await Promise.all(workers.map((w) => w.terminate()));
    workers.length = 0;
  };

  return { dispatch, terminate, size };
};

function createWorkUnits<TInput>(
  items: TInput[],
  options: WorkerDispatchOptions<TInput>,
): Array<WorkUnit<TInput>> {
  const targetUnitBytes = Math.max(1, options.targetUnitBytes ?? DEFAULT_WORK_UNIT_BYTES);
  const maxFilesPerUnit = Math.max(1, options.maxFilesPerUnit ?? DEFAULT_MAX_FILES_PER_UNIT);
  const units: Array<WorkUnit<TInput>> = [];
  let current: TInput[] = [];
  let currentBytes = 0;
  let currentStart = 0;

  for (let index = 0; index < items.length; index++) {
    const item = items[index];
    const itemBytes = itemSize(item, options);
    const wouldExceedBytes = current.length > 0 && currentBytes + itemBytes > targetUnitBytes;
    const wouldExceedFiles = current.length >= maxFilesPerUnit;
    if (wouldExceedBytes || wouldExceedFiles) {
      units.push(createWorkUnit(current, currentStart, units.length, 0, options));
      current = [];
      currentBytes = 0;
      currentStart = index;
    }
    current.push(item);
    currentBytes += itemBytes;
  }

  if (current.length > 0) {
    units.push(createWorkUnit(current, currentStart, units.length, 0, options));
  }

  return units;
}

function createWorkUnit<TInput>(
  items: TInput[],
  startIndex: number,
  unitId: number,
  retryDepth: number,
  options: WorkerDispatchOptions<TInput>,
): WorkUnit<TInput> {
  const languageBreakdown: Record<string, number> = {};
  let totalBytes = 0;
  for (const item of items) {
    totalBytes += itemSize(item, options);
    const ext = itemExtension(item, options);
    languageBreakdown[ext] = (languageBreakdown[ext] ?? 0) + 1;
  }
  return { unitId, startIndex, items, totalBytes, languageBreakdown, retryDepth };
}

function itemSize<TInput>(item: TInput, options: WorkerDispatchOptions<TInput>): number {
  const explicit = options.getItemSize?.(item);
  if (typeof explicit === 'number' && Number.isFinite(explicit) && explicit >= 0) return explicit;
  const content = (item as { content?: unknown }).content;
  return typeof content === 'string' ? Buffer.byteLength(content) : 0;
}

function itemPath<TInput>(item: TInput, options: WorkerDispatchOptions<TInput>): string | undefined {
  const explicit = options.getItemPath?.(item);
  if (explicit) return explicit;
  const value = (item as { path?: unknown; filePath?: unknown }).path;
  if (typeof value === 'string') return value;
  const filePath = (item as { filePath?: unknown }).filePath;
  return typeof filePath === 'string' ? filePath : undefined;
}

function firstItemPath<TInput>(
  items: TInput[],
  options: WorkerDispatchOptions<TInput>,
): string | undefined {
  for (const item of items) {
    const path = itemPath(item, options);
    if (path) return path;
  }
  return undefined;
}

function itemExtension<TInput>(item: TInput, options: WorkerDispatchOptions<TInput>): string {
  const pathValue = itemPath(item, options);
  if (!pathValue) return 'unknown';
  const ext = path.extname(pathValue).toLowerCase();
  return ext || 'no-ext';
}

function formatLanguageBreakdown(breakdown: Record<string, number>): string {
  const entries = Object.entries(breakdown);
  if (entries.length === 0) return 'unknown';
  return entries
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([ext, count]) => `${ext}:${count}`)
    .join(',');
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)}KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
}
