import { beforeAll, describe, expect, it } from 'vitest';
import type Parser from 'tree-sitter';
import { createHash } from 'node:crypto';
import { SupportedLanguages } from 'avmatrix-shared';
import { createParserForLanguage } from '../../../src/core/tree-sitter/parser-loader.js';
import { typescriptProvider } from '../../../src/core/ingestion/languages/typescript.js';
import { finalizeScopeModel } from '../../../src/core/ingestion/finalize-orchestrator.js';
import { resolveScopeReferenceSites } from '../../../src/core/ingestion/scope-reference-resolver.js';
import { extractParsedFileWithStats } from '../../../src/core/ingestion/scope-extractor-bridge.js';

describe('resolveScopeReferenceSites', () => {
  let parser: Parser;

  beforeAll(async () => {
    parser = await createParserForLanguage(SupportedLanguages.TypeScript, 'app.ts');
  });

  it('populates ReferenceIndex from finalized TypeScript scope facts', () => {
    const source = `
class User {
  save() {}
}

class Admin extends User {}

function run(user: User) {
  user.save();
  const admin = new Admin();
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

    const indexes = finalizeScopeModel([parsed!]);
    const result = resolveScopeReferenceSites(indexes);

    const user = parsed!.localDefs.find(
      (def) => def.type === 'Class' && def.qualifiedName === 'User',
    );
    const admin = parsed!.localDefs.find(
      (def) => def.type === 'Class' && def.qualifiedName === 'Admin',
    );
    const save = parsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'save',
    );
    expect(user).toBeDefined();
    expect(admin).toBeDefined();
    expect(save).toBeDefined();

    const refsToSave = result.referenceIndex.byTargetDef.get(save!.nodeId) ?? [];
    const refsToAdmin = result.referenceIndex.byTargetDef.get(admin!.nodeId) ?? [];
    const refsToUser = result.referenceIndex.byTargetDef.get(user!.nodeId) ?? [];
    const fileHash = sourceHash(source);

    expect(refsToSave.map((ref) => ref.kind)).toEqual(['call']);
    expect(refsToSave[0]?.fileHash).toBe(fileHash);
    expect(refsToAdmin.map((ref) => ref.kind)).toEqual(['call']);
    expect(refsToUser.map((ref) => ref.kind).sort()).toEqual(['inherits', 'type-reference']);
    expect(result.stats).toMatchObject({
      totalReferenceSites: 4,
      chunksResolved: 1,
      maxChunkReferenceSites: 4,
      resolvedReferences: 4,
      unresolvedReferences: 0,
      resolvedCalls: 2,
      resolvedTypeReferences: 1,
      resolvedInheritance: 1,
      referenceIndexSourceScopes: 2,
      referenceIndexTargetDefs: 3,
    });
  });

  it('resolves member calls through the pre-resolution method dispatch index', () => {
    const source = `
class Base {
  greet() {}
}

class Child extends Base {}

function run(child: Child) {
  child.greet();
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

    const indexes = finalizeScopeModel([parsed!]);
    const result = resolveScopeReferenceSites(indexes);

    const base = parsed!.localDefs.find(
      (def) => def.type === 'Class' && def.qualifiedName === 'Base',
    );
    const greet = parsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'greet',
    );

    expect(base).toBeDefined();
    expect(greet).toBeDefined();

    const child = parsed!.localDefs.find(
      (def) => def.type === 'Class' && def.qualifiedName === 'Child',
    );
    expect(child).toBeDefined();
    expect(indexes.methodDispatch.mroFor(child!.nodeId)).toEqual([base!.nodeId]);

    const refsToGreet = result.referenceIndex.byTargetDef.get(greet!.nodeId) ?? [];
    expect(refsToGreet.map((ref) => ref.kind)).toEqual(['call']);
    expect(result.stats).toMatchObject({
      totalReferenceSites: 3,
      resolvedReferences: 3,
      unresolvedReferences: 0,
      resolvedCalls: 1,
      resolvedTypeReferences: 1,
      resolvedInheritance: 1,
    });
  });

  it('resolves deterministic reference-site chunks without changing ReferenceIndex output', () => {
    const source = `
class User {
  name: string;
  save() {}
}

function run(user: User) {
  user.name;
  user.save();
}
`;
    const parsed = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/app.ts',
      SupportedLanguages.TypeScript,
      parser.parse(source).rootNode,
    ).parsedFile;
    expect(parsed).toBeDefined();

    const indexes = finalizeScopeModel([parsed!]);
    const unchunked = resolveScopeReferenceSites(indexes);
    const chunked = resolveScopeReferenceSites(indexes, { chunkSize: 1 });

    expect(chunked.stats).toMatchObject({
      totalReferenceSites: unchunked.stats.totalReferenceSites,
      chunkSize: 1,
      chunksResolved: unchunked.stats.totalReferenceSites,
      maxChunkReferenceSites: 1,
      readonlyIndexBytes: unchunked.stats.readonlyIndexBytes,
      resolvedReferences: unchunked.stats.resolvedReferences,
      unresolvedReferences: unchunked.stats.unresolvedReferences,
      referenceIndexSourceScopes: unchunked.stats.referenceIndexSourceScopes,
      referenceIndexTargetDefs: unchunked.stats.referenceIndexTargetDefs,
    });
    expect([...chunked.referenceIndex.byTargetDef.keys()].sort()).toEqual(
      [...unchunked.referenceIndex.byTargetDef.keys()].sort(),
    );
    expect(chunked.stats.readonlyIndexBytes).toBeGreaterThan(0);
    expect(chunked.timings.readonlyIndexInitMs).toBeGreaterThanOrEqual(0);
    expect(chunked.timings.referenceWorkerResolveMs).toBeGreaterThanOrEqual(0);
    expect(chunked.timings.referenceMergeMs).toBeGreaterThanOrEqual(0);
  });

  it('resolves constructor calls through finalized import bindings without source rereads', () => {
    const modelsSource = `
export class User {
  save() {}
}
`;
    const appSource = `
import { User } from './models';

function run() {
  const user = new User();
  user.save();
}
`;
    const modelsParsed = extractParsedFileWithStats(
      typescriptProvider,
      modelsSource,
      'src/models.ts',
      SupportedLanguages.TypeScript,
      parser.parse(modelsSource).rootNode,
    ).parsedFile;
    const appParsed = extractParsedFileWithStats(
      typescriptProvider,
      appSource,
      'src/app.ts',
      SupportedLanguages.TypeScript,
      parser.parse(appSource).rootNode,
    ).parsedFile;
    expect(modelsParsed).toBeDefined();
    expect(appParsed).toBeDefined();

    const indexes = finalizeScopeModel([appParsed!, modelsParsed!], {
      hooks: {
        resolveImportTarget: (targetRaw) => (targetRaw === './models' ? 'src/models.ts' : null),
      },
    });
    const result = resolveScopeReferenceSites(indexes);

    const user = modelsParsed!.localDefs.find(
      (def) => def.type === 'Class' && def.qualifiedName === 'User',
    );
    const save = modelsParsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'save',
    );
    expect(user).toBeDefined();
    expect(save).toBeDefined();

    const refsToUser = result.referenceIndex.byTargetDef.get(user!.nodeId) ?? [];
    const refsToSave = result.referenceIndex.byTargetDef.get(save!.nodeId) ?? [];

    expect(refsToUser.map((ref) => ref.kind)).toEqual(['call']);
    expect(refsToSave.map((ref) => ref.kind)).toEqual(['call']);
    expect(result.stats).toMatchObject({
      totalReferenceSites: 2,
      resolvedReferences: 2,
      unresolvedReferences: 0,
      resolvedCalls: 2,
    });
  });

  it('resolves member read and write access facts to property definitions', () => {
    const source = `
class User {
  name: string;
}

function run(user: User) {
  user.name;
  user.name = 'Ada';
}
`;
    const parsed = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/app.ts',
      SupportedLanguages.TypeScript,
      parser.parse(source).rootNode,
    ).parsedFile;
    expect(parsed).toBeDefined();

    const indexes = finalizeScopeModel([parsed!]);
    const result = resolveScopeReferenceSites(indexes);

    const nameProperty = parsed!.localDefs.find(
      (def) => def.type === 'Property' && def.qualifiedName === 'name',
    );
    expect(nameProperty).toBeDefined();

    const refsToName = result.referenceIndex.byTargetDef.get(nameProperty!.nodeId) ?? [];
    expect(refsToName.map((ref) => ref.kind).sort()).toEqual(['read', 'write']);
    expect(result.stats).toMatchObject({
      totalReferenceSites: 3,
      resolvedReferences: 3,
      unresolvedReferences: 0,
      resolvedTypeReferences: 1,
      resolvedAccesses: 2,
    });
  });

  it('resolves return type annotation facts to class definitions', () => {
    const source = `
class User {}

function create(): User {
  return new User();
}
`;
    const parsed = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/app.ts',
      SupportedLanguages.TypeScript,
      parser.parse(source).rootNode,
    ).parsedFile;
    expect(parsed).toBeDefined();

    const indexes = finalizeScopeModel([parsed!]);
    const result = resolveScopeReferenceSites(indexes);

    const user = parsed!.localDefs.find(
      (def) => def.type === 'Class' && def.qualifiedName === 'User',
    );
    expect(user).toBeDefined();

    const refsToUser = result.referenceIndex.byTargetDef.get(user!.nodeId) ?? [];
    expect(refsToUser.map((ref) => ref.kind).sort()).toEqual(['call', 'type-reference']);
    expect(result.stats).toMatchObject({
      totalReferenceSites: 2,
      resolvedReferences: 2,
      unresolvedReferences: 0,
      resolvedCalls: 1,
      resolvedTypeReferences: 1,
    });
  });

  it('resolves member calls through local function return annotation bindings', () => {
    const source = `
class User {
  save() {}
}

function makeUser(): User {
  return new User();
}

function run() {
  const user = makeUser();
  user.save();
}
`;
    const parsed = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/app.ts',
      SupportedLanguages.TypeScript,
      parser.parse(source).rootNode,
    ).parsedFile;
    expect(parsed).toBeDefined();

    const indexes = finalizeScopeModel([parsed!]);
    const result = resolveScopeReferenceSites(indexes);

    const save = parsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'save',
    );
    expect(save).toBeDefined();

    const refsToSave = result.referenceIndex.byTargetDef.get(save!.nodeId) ?? [];
    expect(refsToSave.map((ref) => ref.kind)).toEqual(['call']);
    expect(result.stats).toMatchObject({
      resolvedReferences: 4,
      unresolvedReferences: 0,
      resolvedCalls: 3,
      resolvedTypeReferences: 1,
    });
  });
});

function sourceHash(source: string): string {
  return `sha256:${createHash('sha256').update(source).digest('hex')}`;
}
