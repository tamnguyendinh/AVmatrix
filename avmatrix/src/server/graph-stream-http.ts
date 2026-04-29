import type express from 'express';
import type { GraphStreamRecord } from '../runtime/repo-runtime/graph-read-service.js';

export class ClientDisconnectedError extends Error {
  constructor() {
    super('Client disconnected during graph stream');
    this.name = 'ClientDisconnectedError';
  }
}

const GRAPH_STREAM_TARGET_CHUNK_BYTES = 64 * 1024;
const GRAPH_STREAM_TARGET_CHUNK_RECORDS = 128;

const ensureStreamIsWritable = (res: express.Response, signal?: AbortSignal): void => {
  if (signal?.aborted || res.destroyed || res.writableEnded) {
    throw new ClientDisconnectedError();
  }
};

const waitForDrain = async (res: express.Response, signal?: AbortSignal): Promise<void> => {
  ensureStreamIsWritable(res, signal);

  await new Promise<void>((resolve, reject) => {
    const cleanup = () => {
      res.off('drain', onDrain);
      res.off('close', onClose);
      signal?.removeEventListener('abort', onAbort);
    };

    const onDrain = () => {
      cleanup();
      resolve();
    };
    const onClose = () => {
      cleanup();
      reject(new ClientDisconnectedError());
    };
    const onAbort = () => {
      cleanup();
      reject(new ClientDisconnectedError());
    };

    res.once('drain', onDrain);
    res.once('close', onClose);
    signal?.addEventListener('abort', onAbort, { once: true });

    if (signal?.aborted || res.destroyed || res.writableEnded) {
      onAbort();
    }
  });

  ensureStreamIsWritable(res, signal);
};

const isClientDisconnectWriteError = (err: unknown): boolean => {
  if (!(err instanceof Error)) return false;
  return (
    (err as NodeJS.ErrnoException).code === 'ERR_STREAM_DESTROYED' ||
    (err as NodeJS.ErrnoException).code === 'EPIPE' ||
    (err as NodeJS.ErrnoException).code === 'ECONNRESET' ||
    err.message.includes('write after end')
  );
};

export const writeNdjsonRecord = async (
  res: express.Response,
  record: GraphStreamRecord,
  signal?: AbortSignal,
): Promise<void> => {
  await writeNdjsonChunk(res, JSON.stringify(record) + '\n', signal);
};

export const writeNdjsonChunk = async (
  res: express.Response,
  chunk: string,
  signal?: AbortSignal,
): Promise<void> => {
  ensureStreamIsWritable(res, signal);

  try {
    const canContinue = res.write(chunk);
    if (!canContinue) {
      await waitForDrain(res, signal);
    }
  } catch (err) {
    if (isClientDisconnectWriteError(err)) {
      throw new ClientDisconnectedError();
    }
    throw err;
  }
};

export const createGraphStreamBatchWriter = (res: express.Response, signal?: AbortSignal) => {
  let bufferedChunk = '';
  let bufferedRecords = 0;

  const flush = async (): Promise<void> => {
    if (bufferedChunk.length === 0) return;
    const chunk = bufferedChunk;
    bufferedChunk = '';
    bufferedRecords = 0;
    await writeNdjsonChunk(res, chunk, signal);
  };

  const write = async (record: GraphStreamRecord): Promise<void> => {
    bufferedChunk += JSON.stringify(record) + '\n';
    bufferedRecords += 1;

    if (
      bufferedRecords >= GRAPH_STREAM_TARGET_CHUNK_RECORDS ||
      bufferedChunk.length >= GRAPH_STREAM_TARGET_CHUNK_BYTES
    ) {
      await flush();
    }
  };

  return { write, flush };
};
