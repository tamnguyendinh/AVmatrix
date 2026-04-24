import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';

const workerState = {
  mode: 'ok' as 'ok' | 'create-fail' | 'dispatch-fail',
  terminateCalls: 0,
  dispatchCalls: 0,
  reset() {
    this.mode = 'ok';
    this.terminateCalls = 0;
    this.dispatchCalls = 0;
  },
};

vi.mock('../../src/core/ingestion/workers/worker-pool.js', () => ({
  createWorkerPool: vi.fn(() => {
    if (workerState.mode === 'create-fail') {
      throw new Error('mock worker script missing');
    }
    return {
      size: 1,
      dispatch: vi.fn(async () => {
        workerState.dispatchCalls += 1;
        if (workerState.mode === 'dispatch-fail') {
          throw new Error('mock worker unit failed');
        }
        return [
          {
            fileCount: 1,
            nodes: [],
            relationships: [],
            symbols: [],
            imports: [],
            calls: [],
            assignments: [],
            heritage: [],
            routes: [],
            fetchCalls: [],
            decoratorRoutes: [],
            toolDefs: [],
            ormQueries: [],
            constructorBindings: [],
            fileScopeBindings: [],
            parsedFiles: [],
            skippedLanguages: {},
          },
        ];
      }),
      terminate: vi.fn(async () => {
        workerState.terminateCalls += 1;
      }),
    };
  }),
}));

const { runChunkedParseAndResolve } =
  await import('../../src/core/ingestion/pipeline-phases/parse-impl.js');
const { createKnowledgeGraph } = await import('../../src/core/graph/graph.js');

function makeTempRepo(files: Record<string, string>): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'parse-impl-worker-canonical-'));
  for (const [rel, content] of Object.entries(files)) {
    const abs = path.join(dir, rel);
    fs.mkdirSync(path.dirname(abs), { recursive: true });
    fs.writeFileSync(abs, content);
  }
  return dir;
}

function scanned(repo: string, files: string[]) {
  return files.map((rel) => ({
    path: rel,
    size: fs.statSync(path.join(repo, rel)).size,
  }));
}

describe('parse-impl worker-canonical behavior', () => {
  let repoPath = '';

  beforeEach(() => {
    workerState.reset();
    repoPath = makeTempRepo({
      'a.ts': `export function foo() { return 1; }\n`,
    });
  });

  afterEach(() => {
    if (repoPath && fs.existsSync(repoPath)) {
      fs.rmSync(repoPath, { recursive: true, force: true });
    }
  });

  it('fails clearly when the worker pool cannot be created', async () => {
    workerState.mode = 'create-fail';

    await expect(
      runChunkedParseAndResolve(
        createKnowledgeGraph(),
        scanned(repoPath, ['a.ts']),
        ['a.ts'],
        1,
        repoPath,
        Date.now(),
        () => {},
      ),
    ).rejects.toThrow(/Worker pool creation failed.*mock worker script missing/);

    expect(workerState.dispatchCalls).toBe(0);
    expect(workerState.terminateCalls).toBe(0);
  });

  it('propagates worker unit failure and still terminates the pool', async () => {
    workerState.mode = 'dispatch-fail';

    await expect(
      runChunkedParseAndResolve(
        createKnowledgeGraph(),
        scanned(repoPath, ['a.ts']),
        ['a.ts'],
        1,
        repoPath,
        Date.now(),
        () => {},
      ),
    ).rejects.toThrow(/mock worker unit failed/);

    expect(workerState.dispatchCalls).toBe(1);
    expect(workerState.terminateCalls).toBe(1);
  });
});
