import { beforeAll, describe, expect, it } from 'vitest';
import type Parser from 'tree-sitter';
import { createHash } from 'node:crypto';
import { SupportedLanguages } from 'avmatrix-shared';
import { createParserForLanguage } from '../../../src/core/tree-sitter/parser-loader.js';
import { typescriptProvider } from '../../../src/core/ingestion/languages/typescript.js';
import { finalizeScopeModel } from '../../../src/core/ingestion/finalize-orchestrator.js';
import {
  resolveScopeReferenceSites,
  resolveScopeReferenceSitesInWorkers,
} from '../../../src/core/ingestion/scope-reference-resolver.js';
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
      (def) => def.type === 'Method' && def.qualifiedName === 'User.save',
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

  it('resolves JSDoc parameter receiver bindings without source rereads', () => {
    const source = `
class User {
  save() {}
}

/**
 * @param {User} user
 */
function run(user) {
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
      (def) => def.type === 'Method' && def.qualifiedName === 'User.save',
    );
    const user = parsed!.localDefs.find(
      (def) => def.type === 'Class' && def.qualifiedName === 'User',
    );
    expect(save).toBeDefined();
    expect(user).toBeDefined();

    expect(result.referenceIndex.byTargetDef.get(save!.nodeId)?.map((ref) => ref.kind)).toEqual([
      'call',
    ]);
    expect(result.referenceIndex.byTargetDef.get(user!.nodeId)?.map((ref) => ref.kind)).toEqual([
      'type-reference',
    ]);
    expect(result.stats).toMatchObject({
      totalReferenceSites: 2,
      resolvedReferences: 2,
      unresolvedReferences: 0,
      resolvedCalls: 1,
      resolvedTypeReferences: 1,
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
      (def) => def.type === 'Method' && def.qualifiedName === 'Base.greet',
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

  it('resolves reference-site chunks in workers without changing ReferenceIndex output', async () => {
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
    const serial = resolveScopeReferenceSites(indexes, { chunkSize: 1 });
    const worker = await resolveScopeReferenceSitesInWorkers(indexes, {
      chunkSize: 1,
      useWorkers: true,
      minWorkerReferenceSites: 0,
      workerCount: 2,
    });

    expect(worker.stats).toMatchObject({
      totalReferenceSites: serial.stats.totalReferenceSites,
      chunkSize: 1,
      chunksResolved: serial.stats.chunksResolved,
      maxChunkReferenceSites: serial.stats.maxChunkReferenceSites,
      resolvedReferences: serial.stats.resolvedReferences,
      unresolvedReferences: serial.stats.unresolvedReferences,
      referenceIndexSourceScopes: serial.stats.referenceIndexSourceScopes,
      referenceIndexTargetDefs: serial.stats.referenceIndexTargetDefs,
    });
    expect([...worker.referenceIndex.byTargetDef.keys()].sort()).toEqual(
      [...serial.referenceIndex.byTargetDef.keys()].sort(),
    );
    expect(worker.timings.readonlyIndexInitMs).toBeGreaterThanOrEqual(0);
    expect(worker.timings.referenceWorkerResolveMs).toBeGreaterThanOrEqual(0);
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
      (def) => def.type === 'Method' && def.qualifiedName === 'User.save',
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
      (def) => def.type === 'Property' && def.qualifiedName === 'User.name',
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

  it('resolves interface property reads and type-alias RHS type references', () => {
    const source = `
class User {}

interface Runnable {
  current: User;
}

type MaybeUser = User | null;

function run(runnable: Runnable, maybe: MaybeUser) {
  runnable.current;
}
`;
    const parsed = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/interface-facts.ts',
      SupportedLanguages.TypeScript,
      parser.parse(source).rootNode,
    ).parsedFile;
    expect(parsed).toBeDefined();

    const indexes = finalizeScopeModel([parsed!]);
    const result = resolveScopeReferenceSites(indexes);

    const user = parsed!.localDefs.find(
      (def) => def.type === 'Class' && def.qualifiedName === 'User',
    );
    const runnable = parsed!.localDefs.find(
      (def) => def.type === 'Interface' && def.qualifiedName === 'Runnable',
    );
    const maybeUser = parsed!.localDefs.find(
      (def) => def.type === 'TypeAlias' && def.qualifiedName === 'MaybeUser',
    );
    const currentProperty = parsed!.localDefs.find(
      (def) => def.type === 'Property' && def.qualifiedName === 'Runnable.current',
    );
    expect(user).toBeDefined();
    expect(runnable).toBeDefined();
    expect(maybeUser).toBeDefined();
    expect(currentProperty).toBeDefined();

    const refsToCurrent = result.referenceIndex.byTargetDef.get(currentProperty!.nodeId) ?? [];
    const refsToUser = result.referenceIndex.byTargetDef.get(user!.nodeId) ?? [];
    const refsToRunnable = result.referenceIndex.byTargetDef.get(runnable!.nodeId) ?? [];
    const refsToMaybeUser = result.referenceIndex.byTargetDef.get(maybeUser!.nodeId) ?? [];

    expect(refsToCurrent.map((ref) => ref.kind)).toEqual(['read']);
    expect(refsToUser.map((ref) => ref.kind)).toEqual(['type-reference', 'type-reference']);
    expect(refsToRunnable.map((ref) => ref.kind)).toEqual(['type-reference']);
    expect(refsToMaybeUser.map((ref) => ref.kind)).toEqual(['type-reference']);
    expect(result.stats).toMatchObject({
      totalReferenceSites: 5,
      resolvedReferences: 5,
      unresolvedReferences: 0,
      resolvedAccesses: 1,
      resolvedTypeReferences: 4,
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
      (def) => def.type === 'Method' && def.qualifiedName === 'User.save',
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

  it('resolves member calls through imported function return annotation bindings without source rereads', () => {
    const modelsSource = `
export class User {
  save() {}
}

export function makeUser(): User {
  return new User();
}
`;
    const appSource = `
import { makeUser } from './models';

function run() {
  const user = makeUser();
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

    const save = modelsParsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'User.save',
    );
    const makeUser = modelsParsed!.localDefs.find(
      (def) => def.type === 'Function' && def.qualifiedName === 'makeUser',
    );
    expect(save).toBeDefined();
    expect(makeUser).toBeDefined();

    const refsToSave = result.referenceIndex.byTargetDef.get(save!.nodeId) ?? [];
    const refsToMakeUser = result.referenceIndex.byTargetDef.get(makeUser!.nodeId) ?? [];

    expect(refsToSave.map((ref) => ref.kind)).toEqual(['call']);
    expect(refsToMakeUser.map((ref) => ref.kind)).toEqual(['call']);
    expect(result.stats).toMatchObject({
      totalReferenceSites: 4,
      resolvedReferences: 4,
      unresolvedReferences: 0,
      resolvedCalls: 3,
      resolvedTypeReferences: 1,
    });
  });

  it('resolves awaited imported function return annotation bindings without source rereads', () => {
    const modelsSource = `
export class User {
  save() {}
}

export function makeUser(): User {
  return new User();
}
`;
    const appSource = `
import { makeUser } from './models';

async function run() {
  const user = await makeUser();
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

    const save = modelsParsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'User.save',
    );
    const makeUser = modelsParsed!.localDefs.find(
      (def) => def.type === 'Function' && def.qualifiedName === 'makeUser',
    );
    expect(save).toBeDefined();
    expect(makeUser).toBeDefined();

    const refsToSave = result.referenceIndex.byTargetDef.get(save!.nodeId) ?? [];
    const refsToMakeUser = result.referenceIndex.byTargetDef.get(makeUser!.nodeId) ?? [];

    expect(refsToSave.map((ref) => ref.kind)).toEqual(['call']);
    expect(refsToMakeUser.map((ref) => ref.kind)).toEqual(['call']);
    expect(result.stats).toMatchObject({
      totalReferenceSites: 4,
      resolvedReferences: 4,
      unresolvedReferences: 0,
      resolvedCalls: 3,
      resolvedTypeReferences: 1,
    });
  });

  it('resolves receiver-propagated aliases from imported function returns without source rereads', () => {
    const modelsSource = `
export class User {
  save() {}
}

export function makeUser(): User {
  return new User();
}
`;
    const appSource = `
import { makeUser } from './models';

function run() {
  const user = makeUser();
  const current = user;
  current.save();
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

    const save = modelsParsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'User.save',
    );
    const makeUser = modelsParsed!.localDefs.find(
      (def) => def.type === 'Function' && def.qualifiedName === 'makeUser',
    );
    expect(save).toBeDefined();
    expect(makeUser).toBeDefined();

    const refsToSave = result.referenceIndex.byTargetDef.get(save!.nodeId) ?? [];
    const refsToMakeUser = result.referenceIndex.byTargetDef.get(makeUser!.nodeId) ?? [];

    expect(refsToSave.map((ref) => ref.kind)).toEqual(['call']);
    expect(refsToMakeUser.map((ref) => ref.kind)).toEqual(['call']);
    expect(result.stats).toMatchObject({
      totalReferenceSites: 4,
      resolvedReferences: 4,
      unresolvedReferences: 0,
      resolvedCalls: 3,
      resolvedTypeReferences: 1,
    });
  });

  it('resolves field-access and method-return bindings from imported receiver types without source rereads', () => {
    const modelsSource = `
export class Profile {
  save() {}
}

export class User {
  profile: Profile;
  getProfile(): Profile {
    return this.profile;
  }
}
`;
    const appSource = `
import { User } from './models';

function run(user: User) {
  const fromField = user.profile;
  fromField.save();
  const fromMethod = user.getProfile();
  fromMethod.save();
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

    const save = modelsParsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'Profile.save',
    );
    const profile = modelsParsed!.localDefs.find(
      (def) => def.type === 'Property' && def.qualifiedName === 'User.profile',
    );
    const getProfile = modelsParsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'User.getProfile',
    );
    expect(save).toBeDefined();
    expect(profile).toBeDefined();
    expect(getProfile).toBeDefined();

    const refsToSave = result.referenceIndex.byTargetDef.get(save!.nodeId) ?? [];
    const refsToProfile = result.referenceIndex.byTargetDef.get(profile!.nodeId) ?? [];
    const refsToGetProfile = result.referenceIndex.byTargetDef.get(getProfile!.nodeId) ?? [];

    expect(refsToSave.map((ref) => ref.kind)).toEqual(['call', 'call']);
    expect(refsToProfile.map((ref) => ref.kind)).toEqual(['read', 'read']);
    expect(refsToGetProfile.map((ref) => ref.kind)).toEqual(['call']);
    expect(result.stats).toMatchObject({
      totalReferenceSites: 8,
      resolvedReferences: 8,
      unresolvedReferences: 0,
      resolvedCalls: 3,
      resolvedAccesses: 2,
      resolvedTypeReferences: 3,
    });
  });

  it('resolves object-pattern field-access bindings from imported receiver types without source rereads', () => {
    const modelsSource = `
export class Profile {
  save() {}
}

export class User {
  profile: Profile;
}
`;
    const appSource = `
import { User } from './models';

function run(user: User) {
  const { profile } = user;
  profile.save();
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

    const save = modelsParsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'Profile.save',
    );
    expect(save).toBeDefined();

    const refsToSave = result.referenceIndex.byTargetDef.get(save!.nodeId) ?? [];
    expect(refsToSave.map((ref) => ref.kind)).toEqual(['call']);
    expect(result.stats).toMatchObject({
      totalReferenceSites: 3,
      resolvedReferences: 3,
      unresolvedReferences: 0,
      resolvedCalls: 1,
      resolvedTypeReferences: 2,
    });
  });

  it('resolves object-pattern field-access bindings from call-result receivers without source rereads', () => {
    const modelsSource = `
export class Profile {
  save() {}
}

export class User {
  profile: Profile;
}

export class Provider {
  getUser(): User {
    return new User();
  }
}

export function makeUser(): User {
  return new User();
}
`;
    const appSource = `
import { makeUser, Provider } from './models';

async function run(provider: Provider) {
  const { profile: fromCall } = await makeUser();
  fromCall.save();
  const { profile: fromMethod } = provider.getUser();
  fromMethod.save();
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

    const save = modelsParsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'Profile.save',
    );
    const makeUser = modelsParsed!.localDefs.find(
      (def) => def.type === 'Function' && def.qualifiedName === 'makeUser',
    );
    const getUser = modelsParsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'Provider.getUser',
    );
    expect(save).toBeDefined();
    expect(makeUser).toBeDefined();
    expect(getUser).toBeDefined();

    const refsToSave = result.referenceIndex.byTargetDef.get(save!.nodeId) ?? [];
    const refsToMakeUser = result.referenceIndex.byTargetDef.get(makeUser!.nodeId) ?? [];
    const refsToGetUser = result.referenceIndex.byTargetDef.get(getUser!.nodeId) ?? [];

    expect(refsToSave.map((ref) => ref.kind)).toEqual(['call', 'call']);
    expect(refsToMakeUser.map((ref) => ref.kind)).toEqual(['call']);
    expect(refsToGetUser.map((ref) => ref.kind)).toEqual(['call']);
    expect(result.stats).toMatchObject({
      totalReferenceSites: 10,
      resolvedReferences: 10,
      unresolvedReferences: 0,
      resolvedCalls: 6,
      resolvedTypeReferences: 4,
    });
  });

  it('resolves imported iterable function return element bindings without source rereads', () => {
    const modelsSource = `
export class User {
  save() {}
}

export function listUsers(): User[] {
  return [];
}
`;
    const appSource = `
import { listUsers } from './models';

function run() {
  for (const user of listUsers()) {
    user.save();
  }
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

    const save = modelsParsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'User.save',
    );
    const listUsers = modelsParsed!.localDefs.find(
      (def) => def.type === 'Function' && def.qualifiedName === 'listUsers',
    );
    expect(save).toBeDefined();
    expect(listUsers).toBeDefined();

    const refsToSave = result.referenceIndex.byTargetDef.get(save!.nodeId) ?? [];
    const refsToListUsers = result.referenceIndex.byTargetDef.get(listUsers!.nodeId) ?? [];

    expect(refsToSave.map((ref) => ref.kind)).toEqual(['call']);
    expect(refsToListUsers.map((ref) => ref.kind)).toEqual(['call']);
    expect(result.stats).toMatchObject({
      totalReferenceSites: 3,
      resolvedReferences: 3,
      unresolvedReferences: 0,
      resolvedCalls: 2,
      resolvedTypeReferences: 1,
    });
  });

  it('resolves iterable variable return element bindings without source rereads', () => {
    const modelsSource = `
export class User {
  save() {}
}

export function listUsers(): User[] {
  return [];
}
`;
    const appSource = `
import { listUsers } from './models';

function run() {
  const users = listUsers();
  for (const user of users) {
    user.save();
  }
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

    const save = modelsParsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'User.save',
    );
    const listUsers = modelsParsed!.localDefs.find(
      (def) => def.type === 'Function' && def.qualifiedName === 'listUsers',
    );
    expect(save).toBeDefined();
    expect(listUsers).toBeDefined();

    const refsToSave = result.referenceIndex.byTargetDef.get(save!.nodeId) ?? [];
    const refsToListUsers = result.referenceIndex.byTargetDef.get(listUsers!.nodeId) ?? [];

    expect(refsToSave.map((ref) => ref.kind)).toEqual(['call']);
    expect(refsToListUsers.map((ref) => ref.kind)).toEqual(['call']);
    expect(result.stats).toMatchObject({
      totalReferenceSites: 3,
      resolvedReferences: 3,
      unresolvedReferences: 0,
      resolvedCalls: 2,
      resolvedTypeReferences: 1,
    });
  });
});

function sourceHash(source: string): string {
  return `sha256:${createHash('sha256').update(source).digest('hex')}`;
}
