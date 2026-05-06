import { beforeAll, describe, expect, it, vi } from 'vitest';
import type Parser from 'tree-sitter';
import { createHash } from 'node:crypto';
import { SupportedLanguages } from 'avmatrix-shared';
import { createKnowledgeGraph } from '../../../src/core/graph/graph.js';
import { typescriptProvider } from '../../../src/core/ingestion/languages/typescript.js';
import { createResolutionContext } from '../../../src/core/ingestion/model/resolution-context.js';
import { finalizeScopeModel } from '../../../src/core/ingestion/finalize-orchestrator.js';
import { extractParsedFileWithStats } from '../../../src/core/ingestion/scope-extractor-bridge.js';
import { resolutionPhase } from '../../../src/core/ingestion/pipeline-phases/resolution.js';
import { createParserForLanguage } from '../../../src/core/tree-sitter/parser-loader.js';
import type { ParseOutput } from '../../../src/core/ingestion/pipeline-phases/parse.js';
import type {
  PhaseResult,
  PipelineContext,
} from '../../../src/core/ingestion/pipeline-phases/types.js';

describe('resolutionPhase', () => {
  let parser: Parser;

  beforeAll(async () => {
    parser = await createParserForLanguage(SupportedLanguages.TypeScript, 'app.ts');
  });

  it('populates ReferenceIndex metrics and emits scope-resolved graph edges', async () => {
    const source = `
class User {
  save() {}
}

function run(user: User) {
  user.save();
}
`;
    const tree = parser.parse(source);
    const parsed = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/app.ts',
      SupportedLanguages.TypeScript,
      tree.rootNode,
    ).parsedFile;
    expect(parsed).toBeDefined();

    const resolutionContext = createResolutionContext();
    resolutionContext.model.attachScopeIndexes(finalizeScopeModel([parsed!]));

    const graph = createKnowledgeGraph();
    const output = await resolutionPhase.execute(
      {
        repoPath: '/tmp/repo',
        graph,
        onProgress: vi.fn(),
        pipelineStart: 0,
      } satisfies PipelineContext,
      new Map<string, PhaseResult<unknown>>([
        [
          'parse',
          {
            phaseName: 'parse',
            durationMs: 0,
            output: {
              totalFiles: 1,
              resolutionContext,
            } as ParseOutput,
          },
        ],
        [
          'crossFile',
          {
            phaseName: 'crossFile',
            durationMs: 0,
            output: { filesReprocessed: 0, metrics: { timings: {}, counters: {} } },
          },
        ],
      ]),
    );

    expect(output.metrics.counters.scopeResolutionReferenceSites).toBe(2);
    expect(output.metrics.counters.scopeResolutionChunks).toBe(1);
    expect(output.metrics.counters.scopeResolutionMaxChunkReferenceSites).toBe(2);
    expect(output.metrics.counters.scopeResolutionReferenceIndexSourceScopes).toBe(1);
    expect(output.metrics.counters.scopeResolutionReferenceIndexTargetDefs).toBe(2);
    expect(output.metrics.counters.scopeResolutionResolvedReferences).toBe(2);
    expect(output.metrics.counters.scopeResolutionResolvedCalls).toBe(1);
    expect(output.metrics.counters.scopeResolutionResolvedTypeReferences).toBe(1);
    expect(output.metrics.counters.scopeResolutionEdgesEmitted).toBe(2);
    expect(output.metrics.counters.scopeResolutionDuplicateEdgesSkipped).toBe(0);
    expect(output.referenceIndex.bySourceScope.size).toBe(1);
    expect(graph.relationshipCount).toBe(2);
    expect(graph.relationships.map((rel) => rel.type).sort()).toEqual(['CALLS', 'USES']);
    expect(graph.relationships.every((rel) => rel.fileHash === sourceHash(source))).toBe(true);
  });

  it('emits scope-resolved access edges from read/write reference facts', async () => {
    const source = `
class User {
  name: string;
}

function run(user: User) {
  user.name;
  user.name = 'Ada';
}
`;
    const tree = parser.parse(source);
    const parsed = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/app.ts',
      SupportedLanguages.TypeScript,
      tree.rootNode,
    ).parsedFile;
    expect(parsed).toBeDefined();

    const resolutionContext = createResolutionContext();
    resolutionContext.model.attachScopeIndexes(finalizeScopeModel([parsed!]));

    const graph = createKnowledgeGraph();
    const output = await resolutionPhase.execute(
      {
        repoPath: '/tmp/repo',
        graph,
        onProgress: vi.fn(),
        pipelineStart: 0,
      } satisfies PipelineContext,
      new Map<string, PhaseResult<unknown>>([
        [
          'parse',
          {
            phaseName: 'parse',
            durationMs: 0,
            output: {
              totalFiles: 1,
              resolutionContext,
            } as ParseOutput,
          },
        ],
        [
          'crossFile',
          {
            phaseName: 'crossFile',
            durationMs: 0,
            output: { filesReprocessed: 0, metrics: { timings: {}, counters: {} } },
          },
        ],
      ]),
    );

    expect(output.metrics.counters.scopeResolutionReferenceSites).toBe(3);
    expect(output.metrics.counters.scopeResolutionResolvedReferences).toBe(3);
    expect(output.metrics.counters.scopeResolutionResolvedAccesses).toBe(2);
    expect(output.metrics.counters.scopeResolutionResolvedTypeReferences).toBe(1);
    expect(output.metrics.counters.scopeResolutionEdgesEmitted).toBe(3);
    expect(graph.relationships.map((rel) => rel.type).sort()).toEqual([
      'ACCESSES',
      'ACCESSES',
      'USES',
    ]);
    expect(
      graph.relationships
        .filter((rel) => rel.type === 'ACCESSES')
        .map((rel) => rel.step)
        .sort(),
    ).toEqual([1, 2]);
  });

  it('emits finalized import edges even when there are no reference sites', async () => {
    const appSource = `import { User } from './models';\n`;
    const modelSource = `export class User {}\n`;
    const appTree = parser.parse(appSource);
    const modelTree = parser.parse(modelSource);
    const appParsed = extractParsedFileWithStats(
      typescriptProvider,
      appSource,
      'src/app.ts',
      SupportedLanguages.TypeScript,
      appTree.rootNode,
    ).parsedFile;
    const modelParsed = extractParsedFileWithStats(
      typescriptProvider,
      modelSource,
      'src/models.ts',
      SupportedLanguages.TypeScript,
      modelTree.rootNode,
    ).parsedFile;
    expect(appParsed).toBeDefined();
    expect(modelParsed).toBeDefined();

    const resolutionContext = createResolutionContext();
    resolutionContext.model.attachScopeIndexes(
      finalizeScopeModel([appParsed!, modelParsed!], {
        hooks: {
          resolveImportTarget: (targetRaw, fromFile) =>
            targetRaw === './models' && fromFile === 'src/app.ts' ? 'src/models.ts' : null,
        },
      }),
    );

    const graph = createKnowledgeGraph();
    const output = await resolutionPhase.execute(
      {
        repoPath: '/tmp/repo',
        graph,
        onProgress: vi.fn(),
        pipelineStart: 0,
      } satisfies PipelineContext,
      new Map<string, PhaseResult<unknown>>([
        [
          'parse',
          {
            phaseName: 'parse',
            durationMs: 0,
            output: {
              totalFiles: 2,
              resolutionContext,
            } as ParseOutput,
          },
        ],
        [
          'crossFile',
          {
            phaseName: 'crossFile',
            durationMs: 0,
            output: { filesReprocessed: 0, metrics: { timings: {}, counters: {} } },
          },
        ],
      ]),
    );

    expect(output.metrics.counters.scopeResolutionReferenceSites).toBe(0);
    expect(output.metrics.counters.scopeResolutionEdgesEmitted).toBe(0);
    expect(output.metrics.counters.scopeResolutionFinalizedImportsEmitted).toBe(1);
    expect(output.metrics.counters.scopeResolutionDuplicateImportsSkipped).toBe(0);
    expect(output.metrics.counters.scopeResolutionFinalizedImportUsesEmitted).toBe(1);
    expect(output.metrics.counters.scopeResolutionDuplicateImportUsesSkipped).toBe(0);
    expect(graph.relationships).toHaveLength(2);
    const importsEdge = graph.relationships.find((rel) => rel.type === 'IMPORTS');
    const usesEdge = graph.relationships.find((rel) => rel.type === 'USES');
    expect(importsEdge).toMatchObject({
      sourceId: 'File:src/app.ts',
      targetId: 'File:src/models.ts',
      type: 'IMPORTS',
      resolutionSource: 'scope-finalize',
      fileHash: sourceHash(appSource),
    });
    expect(usesEdge).toMatchObject({
      sourceId: 'File:src/app.ts',
      type: 'USES',
      resolutionSource: 'scope-finalize',
      fileHash: sourceHash(appSource),
    });
  });
});

function sourceHash(source: string): string {
  return `sha256:${createHash('sha256').update(source).digest('hex')}`;
}
