import { beforeAll, describe, expect, it } from 'vitest';
import type Parser from 'tree-sitter';
import { SupportedLanguages, type ParsedFile } from 'avmatrix-shared';
import { createKnowledgeGraph } from '../../../src/core/graph/graph.js';
import { finalizeScopeModel } from '../../../src/core/ingestion/finalize-orchestrator.js';
import {
  emitReferencesToGraph,
  emitScopeGraph,
} from '../../../src/core/ingestion/emit-references.js';
import { typescriptProvider } from '../../../src/core/ingestion/languages/typescript.js';
import { extractParsedFileWithStats } from '../../../src/core/ingestion/scope-extractor-bridge.js';
import { resolveScopeReferenceSites } from '../../../src/core/ingestion/scope-reference-resolver.js';
import { createParserForLanguage } from '../../../src/core/tree-sitter/parser-loader.js';

describe('TypeScript accurate single-pass parity fixture', () => {
  let parser: Parser;

  beforeAll(async () => {
    parser = await createParserForLanguage(SupportedLanguages.TypeScript, 'app.ts');
  });

  it('pins CALLS/IMPORTS/ACCESSES/USES/INHERITS counts from AST-reused facts', () => {
    const modelsSource = `
export class User {
  name: string;
  save() {}
}

export class Repo {
  find(user: User) {
    return user.name;
  }
}

export interface Runnable {
  run(): void;
}
`;
    const appSource = `
import { Repo, User, Runnable } from './models';

class Admin extends User implements Runnable {
  run() {}
}

export function run(repo: Repo, user: User) {
  repo.find(user);
  user.name = 'Ada';
  user.save();
  const created = new User();
  created.save();
}
`;
    const parsedFiles = [
      parseFromExistingAst(parser, modelsSource, 'src/models.ts'),
      parseFromExistingAst(parser, appSource, 'src/app.ts'),
    ];

    const indexes = finalizeScopeModel(parsedFiles, {
      hooks: {
        resolveImportTarget: (targetRaw) => (targetRaw === './models' ? 'src/models.ts' : null),
      },
    });
    const resolution = resolveScopeReferenceSites(indexes);

    const graph = createKnowledgeGraph();
    emitReferencesToGraph({
      graph,
      scopes: indexes,
      referenceIndex: resolution.referenceIndex,
    });
    emitScopeGraph({ graph, scopes: indexes });

    expect(countFinalizedImports(indexes.imports)).toBe(3);
    expect(resolution.stats.unresolvedReferences).toBe(0);
    expect(resolution.stats.resolvedReferences).toBe(11);
    expect(resolution.stats.resolvedCalls).toBe(4);
    expect(edgeCounts(graph.relationships)).toMatchObject({
      CALLS: 3,
      IMPORTS: 3,
      ACCESSES: 2,
      USES: 3,
      INHERITS: 2,
    });
    expect(
      graph.relationships
        .filter((rel) => ['CALLS', 'ACCESSES', 'USES', 'INHERITS'].includes(rel.type))
        .every((rel) => rel.resolutionSource === 'scope-resolution' && rel.fileHash !== undefined),
    ).toBe(true);
    expect(
      graph.relationships
        .filter((rel) => rel.type === 'IMPORTS')
        .every(
          (rel) =>
            rel.resolutionSource === 'scope-finalize' &&
            rel.fileHash !== undefined &&
            rel.evidence?.some((entry) => entry.kind === 'import') === true,
        ),
    ).toBe(true);
  });

  it('resolves member calls through imported type aliases without reparsing', () => {
    const modelsSource = `
export class User {
  save() {}
}
`;
    const appSource = `
import { User as U } from './models';

export function run(current: U) {
  current.save();
}
`;
    const parsedFiles = [
      parseFromExistingAst(parser, modelsSource, 'src/models.ts'),
      parseFromExistingAst(parser, appSource, 'src/app.ts'),
    ];

    const indexes = finalizeScopeModel(parsedFiles, {
      hooks: {
        resolveImportTarget: (targetRaw) => (targetRaw === './models' ? 'src/models.ts' : null),
      },
    });
    const resolution = resolveScopeReferenceSites(indexes);
    const graph = createKnowledgeGraph();
    emitReferencesToGraph({
      graph,
      scopes: indexes,
      referenceIndex: resolution.referenceIndex,
    });

    expect(resolution.stats.unresolvedReferences).toBe(0);
    expect(resolution.stats.resolvedReferences).toBe(2);
    expect(resolution.stats.resolvedCalls).toBe(1);
    expect(resolution.stats.resolvedTypeReferences).toBe(1);
    expect(edgeCounts(graph.relationships)).toMatchObject({
      CALLS: 1,
      USES: 1,
    });
  });
});

function parseFromExistingAst(parser: Parser, source: string, filePath: string): ParsedFile {
  const result = extractParsedFileWithStats(
    typescriptProvider,
    source,
    filePath,
    SupportedLanguages.TypeScript,
    parser.parse(source).rootNode,
  );
  expect(result.mode).toBe('ast-reused');
  expect(result.parsedFile).toBeDefined();
  return result.parsedFile!;
}

function countFinalizedImports(imports: ReadonlyMap<string, readonly unknown[]>): number {
  let total = 0;
  for (const edges of imports.values()) total += edges.length;
  return total;
}

function edgeCounts(relationships: readonly { type: string }[]): Record<string, number> {
  const counts: Record<string, number> = {};
  for (const relationship of relationships) {
    counts[relationship.type] = (counts[relationship.type] ?? 0) + 1;
  }
  return counts;
}
