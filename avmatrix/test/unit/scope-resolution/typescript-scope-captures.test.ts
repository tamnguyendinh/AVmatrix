import { beforeAll, describe, expect, it } from 'vitest';
import type Parser from 'tree-sitter';
import { SupportedLanguages } from 'avmatrix-shared';
import { createParserForLanguage } from '../../../src/core/tree-sitter/parser-loader.js';
import { typescriptProvider } from '../../../src/core/ingestion/languages/typescript.js';
import { extractParsedFileWithStats } from '../../../src/core/ingestion/scope-extractor-bridge.js';

describe('TypeScript AST-aware scope captures', () => {
  let parser: Parser;

  beforeAll(async () => {
    parser = await createParserForLanguage(SupportedLanguages.TypeScript, 'sample.ts');
  });

  it('uses the AST-aware optimized hook without a source-text compatibility hook', () => {
    expect(typescriptProvider.emitScopeCapturesFromTree).toBeDefined();
    expect(typescriptProvider.emitScopeCaptures).toBeUndefined();
  });

  it('emits ParsedFile facts from the already-parsed AST', () => {
    const source = `
import DefaultUser, { Repo, User as U } from './models';
import * as utils from './utils';
export { Audit as AuditLog } from './audit';

class Service extends Base implements Runnable {
  current: U;
  constructor(current: U) {
    this.current = current;
  }
  save(repo: Repo) {
    repo.find(this.current.id);
  }
}

const makeService = (repo: Repo): Service => new Service(repo);

export function run(service: Service) {
  service.save(new Repo());
}
`;
    const tree = parser.parse(source);
    const result = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/service.ts',
      SupportedLanguages.TypeScript,
      tree.rootNode,
    );

    expect(result.mode).toBe('ast-reused');
    const parsed = result.parsedFile;
    expect(parsed).toBeDefined();
    expect(parsed!.filePath).toBe('src/service.ts');

    expect(parsed!.scopes.map((scope) => scope.kind)).toEqual(
      expect.arrayContaining(['Module', 'Class', 'Function']),
    );

    const defs = parsed!.localDefs.map((def) => `${def.type}:${def.qualifiedName}`).sort();
    expect(defs).toEqual(
      expect.arrayContaining([
        'Class:Service',
        'Constructor:Service.constructor',
        'Function:makeService',
        'Function:run',
        'Method:Service.save',
        'Property:Service.current',
      ]),
    );

    const service = parsed!.localDefs.find(
      (def) => def.type === 'Class' && def.qualifiedName === 'Service',
    );
    const save = parsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'Service.save',
    );
    const current = parsed!.localDefs.find(
      (def) => def.type === 'Property' && def.qualifiedName === 'Service.current',
    );
    const makeService = parsed!.localDefs.find(
      (def) => def.type === 'Function' && def.qualifiedName === 'makeService',
    );
    expect(service).toBeDefined();
    expect(save?.ownerId).toBe(service!.nodeId);
    expect(current?.ownerId).toBe(service!.nodeId);
    expect(current?.declaredType).toBe('U');
    expect(makeService?.returnType).toBe('Service');

    expect(parsed!.parsedImports).toEqual(
      expect.arrayContaining([
        { kind: 'named', localName: 'DefaultUser', importedName: 'default', targetRaw: './models' },
        { kind: 'named', localName: 'Repo', importedName: 'Repo', targetRaw: './models' },
        {
          kind: 'alias',
          localName: 'U',
          importedName: 'User',
          alias: 'U',
          targetRaw: './models',
        },
        { kind: 'namespace', localName: 'utils', importedName: 'utils', targetRaw: './utils' },
        {
          kind: 'reexport',
          localName: 'AuditLog',
          importedName: 'Audit',
          alias: 'AuditLog',
          targetRaw: './audit',
        },
      ]),
    );

    const typeBindings = new Map<string, string>();
    for (const scope of parsed!.scopes) {
      for (const [name, typeRef] of scope.typeBindings) {
        typeBindings.set(name, typeRef.rawName);
      }
    }
    expect(typeBindings.get('current')).toBe('U');
    expect(typeBindings.get('repo')).toBe('Repo');
    expect(typeBindings.get('service')).toBe('Service');
    expect(typeBindings.get('this')).toBe('Service');

    const references = parsed!.referenceSites.map((site) => ({
      name: site.name,
      kind: site.kind,
      callForm: site.callForm,
      receiver: site.explicitReceiver?.name,
      arity: site.arity,
    }));
    expect(references).toEqual(
      expect.arrayContaining([
        {
          name: 'Base',
          kind: 'inherits',
          callForm: undefined,
          receiver: undefined,
          arity: undefined,
        },
        {
          name: 'Runnable',
          kind: 'inherits',
          callForm: undefined,
          receiver: undefined,
          arity: undefined,
        },
        {
          name: 'U',
          kind: 'type-reference',
          callForm: undefined,
          receiver: undefined,
          arity: undefined,
        },
        {
          name: 'Repo',
          kind: 'type-reference',
          callForm: undefined,
          receiver: undefined,
          arity: undefined,
        },
        {
          name: 'Service',
          kind: 'type-reference',
          callForm: undefined,
          receiver: undefined,
          arity: undefined,
        },
        { name: 'find', kind: 'call', callForm: 'member', receiver: 'repo', arity: 1 },
        { name: 'current', kind: 'write', callForm: undefined, receiver: 'this', arity: undefined },
        { name: 'current', kind: 'read', callForm: undefined, receiver: 'this', arity: undefined },
        {
          name: 'Service',
          kind: 'call',
          callForm: 'constructor',
          receiver: undefined,
          arity: 1,
        },
        { name: 'save', kind: 'call', callForm: 'member', receiver: 'service', arity: 1 },
        { name: 'Repo', kind: 'call', callForm: 'constructor', receiver: undefined, arity: 0 },
      ]),
    );
  });

  it('emits return type references from the already-parsed AST', () => {
    const source = `
class User {}

export function makeUser(): User {
  return new User();
}

const makeOther = (): User => new User();

class Service {
  current(): User {
    return new User();
  }
}
`;
    const tree = parser.parse(source);
    const result = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/returns.ts',
      SupportedLanguages.TypeScript,
      tree.rootNode,
    );

    expect(result.mode).toBe('ast-reused');
    const parsed = result.parsedFile;
    expect(parsed).toBeDefined();

    const returnTypeRefs = parsed!.referenceSites.filter(
      (site) => site.kind === 'type-reference' && site.name === 'User',
    );
    expect(returnTypeRefs).toHaveLength(3);
  });

  it('does not emit duplicate read facts for member calls and classifies writes by stable node range', () => {
    const source = `
class User {
  name = '';
  save() {}
}

function run(user: User) {
  user.save();
  user.name = 'Ada';
  user.name++;
}
`;
    const tree = parser.parse(source);
    const result = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/member-access.ts',
      SupportedLanguages.TypeScript,
      tree.rootNode,
    );

    expect(result.mode).toBe('ast-reused');
    const parsed = result.parsedFile;
    expect(parsed).toBeDefined();

    expect(
      parsed!.referenceSites.filter(
        (site) =>
          site.kind === 'call' &&
          site.name === 'save' &&
          site.callForm === 'member' &&
          site.explicitReceiver?.name === 'user',
      ),
    ).toHaveLength(1);
    expect(
      parsed!.referenceSites.filter(
        (site) =>
          site.kind === 'read' && site.name === 'save' && site.explicitReceiver?.name === 'user',
      ),
    ).toHaveLength(0);
    expect(
      parsed!.referenceSites.filter(
        (site) =>
          site.kind === 'write' && site.name === 'name' && site.explicitReceiver?.name === 'user',
      ),
    ).toHaveLength(2);
  });

  it('emits for-of call-return element bindings from the already-parsed AST', () => {
    const source = `
import { listUsers } from './models';

class User {
  save() {}
}

function localUsers(): User[] {
  return [];
}

function run() {
  for (const localUser of localUsers()) {
    localUser.save();
  }
  for (const importedUser of listUsers()) {
    importedUser.save();
  }
}
`;
    const tree = parser.parse(source);
    const result = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/for-of.ts',
      SupportedLanguages.TypeScript,
      tree.rootNode,
    );

    expect(result.mode).toBe('ast-reused');
    const parsed = result.parsedFile;
    expect(parsed).toBeDefined();

    const typeBindings = new Map<string, { rawName: string; source: string }>();
    for (const scope of parsed!.scopes) {
      for (const [name, typeRef] of scope.typeBindings) {
        typeBindings.set(name, { rawName: typeRef.rawName, source: typeRef.source });
      }
    }

    expect(typeBindings.get('localUser')).toEqual({
      rawName: 'localUsers',
      source: 'call-return-element',
    });
    expect(typeBindings.get('importedUser')).toEqual({
      rawName: 'listUsers',
      source: 'call-return-element',
    });
  });

  it('emits interface property and type-alias RHS facts from the already-parsed AST', () => {
    const source = `
class User {}
class Task {}
class Result {}

interface Runnable {
  current: User;
  run(input: Task): Result;
}

type MaybeUser = User | null;
`;
    const tree = parser.parse(source);
    const result = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/interface-facts.ts',
      SupportedLanguages.TypeScript,
      tree.rootNode,
    );

    expect(result.mode).toBe('ast-reused');
    const parsed = result.parsedFile;
    expect(parsed).toBeDefined();

    const runnable = parsed!.localDefs.find(
      (def) => def.type === 'Interface' && def.qualifiedName === 'Runnable',
    );
    const current = parsed!.localDefs.find(
      (def) => def.type === 'Property' && def.qualifiedName === 'Runnable.current',
    );
    const maybeUser = parsed!.localDefs.find(
      (def) => def.type === 'TypeAlias' && def.qualifiedName === 'MaybeUser',
    );

    expect(runnable).toBeDefined();
    expect(current?.ownerId).toBe(runnable!.nodeId);
    expect(current?.declaredType).toBe('User');
    expect(maybeUser).toBeDefined();

    const typeBindings = new Map<string, string>();
    for (const scope of parsed!.scopes) {
      for (const [name, typeRef] of scope.typeBindings) {
        typeBindings.set(name, typeRef.rawName);
      }
    }
    expect(typeBindings.get('current')).toBe('User');
    expect(typeBindings.get('input')).toBe('Task');

    const typeRefs = parsed!.referenceSites
      .filter((site) => site.kind === 'type-reference')
      .map((site) => site.name)
      .sort();
    expect(typeRefs).toEqual(['Result', 'Task', 'User', 'User']);
  });

  it('infers local variable type bindings from local function return annotations', () => {
    const source = `
class User {}

function makeUser(): User {
  return new User();
}

const makeOther = (): User => new User();

function run() {
  const user = makeUser();
  const other = makeOther();
}
`;
    const tree = parser.parse(source);
    const result = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/inferred-return.ts',
      SupportedLanguages.TypeScript,
      tree.rootNode,
    );

    expect(result.mode).toBe('ast-reused');
    const parsed = result.parsedFile;
    expect(parsed).toBeDefined();

    const typeBindings = new Map<string, string>();
    for (const scope of parsed!.scopes) {
      for (const [name, typeRef] of scope.typeBindings) {
        typeBindings.set(name, typeRef.rawName);
      }
    }
    expect(typeBindings.get('user')).toBe('User');
    expect(typeBindings.get('other')).toBe('User');
    const makeUser = parsed!.localDefs.find(
      (def) => def.type === 'Function' && def.qualifiedName === 'makeUser',
    );
    const makeOther = parsed!.localDefs.find(
      (def) => def.type === 'Function' && def.qualifiedName === 'makeOther',
    );
    expect(makeUser?.returnType).toBe('User');
    expect(makeOther?.returnType).toBe('User');
  });

  it('emits call-return type binding facts when a variable is assigned from an unresolved call', () => {
    const source = `
import { makeUser } from './models';

function run() {
  const user = makeUser();
  user.save();
}
`;
    const tree = parser.parse(source);
    const result = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/app.ts',
      SupportedLanguages.TypeScript,
      tree.rootNode,
    );

    expect(result.mode).toBe('ast-reused');
    const parsed = result.parsedFile;
    expect(parsed).toBeDefined();

    const userBinding = parsed!.scopes
      .map((scope) => scope.typeBindings.get('user'))
      .find((binding) => binding !== undefined);

    expect(userBinding).toMatchObject({
      rawName: 'makeUser',
      source: 'call-return',
    });
  });

  it('emits call-return type binding facts for awaited imported calls', () => {
    const source = `
import { makeUser } from './models';

async function run() {
  const user = await makeUser();
  user.save();
}
`;
    const tree = parser.parse(source);
    const result = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/app.ts',
      SupportedLanguages.TypeScript,
      tree.rootNode,
    );

    expect(result.mode).toBe('ast-reused');
    const parsed = result.parsedFile;
    expect(parsed).toBeDefined();

    const userBinding = parsed!.scopes
      .map((scope) => scope.typeBindings.get('user'))
      .find((binding) => binding !== undefined);

    expect(userBinding).toMatchObject({
      rawName: 'makeUser',
      source: 'call-return',
    });
  });

  it('emits receiver-propagated type binding facts for identifier aliases', () => {
    const source = `
class User {
  save() {}
}

function run(user: User) {
  const current = user;
  current.save();
}
`;
    const tree = parser.parse(source);
    const result = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/alias.ts',
      SupportedLanguages.TypeScript,
      tree.rootNode,
    );

    expect(result.mode).toBe('ast-reused');
    const parsed = result.parsedFile;
    expect(parsed).toBeDefined();

    const currentBinding = parsed!.scopes
      .map((scope) => scope.typeBindings.get('current'))
      .find((binding) => binding !== undefined);

    expect(currentBinding).toMatchObject({
      rawName: 'user',
      source: 'receiver-propagated',
    });
  });

  it('emits field-access and method-return binding facts from the already-parsed AST', () => {
    const source = `
class Profile {
  save() {}
}

class User {
  profile: Profile;
  getProfile(): Profile {
    return this.profile;
  }
}

function run(user: User) {
  const fromField = user.profile;
  const fromMethod = user.getProfile();
}
`;
    const tree = parser.parse(source);
    const result = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/member-derived.ts',
      SupportedLanguages.TypeScript,
      tree.rootNode,
    );

    expect(result.mode).toBe('ast-reused');
    const parsed = result.parsedFile;
    expect(parsed).toBeDefined();

    const typeBindings = new Map<string, { rawName: string; source: string }>();
    for (const scope of parsed!.scopes) {
      for (const [name, typeRef] of scope.typeBindings) {
        typeBindings.set(name, { rawName: typeRef.rawName, source: typeRef.source });
      }
    }

    expect(typeBindings.get('fromField')).toEqual({
      rawName: 'user.profile',
      source: 'field-access',
    });
    expect(typeBindings.get('fromMethod')).toEqual({
      rawName: 'user.getProfile',
      source: 'method-return',
    });
  });

  it('emits object-pattern field-access bindings from the already-parsed AST', () => {
    const source = `
class Profile {
  save() {}
}

class User {
  profile: Profile;
  displayName: string;
}

function run(user: User) {
  const { profile, displayName: name } = user;
}
`;
    const tree = parser.parse(source);
    const result = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/destructure.ts',
      SupportedLanguages.TypeScript,
      tree.rootNode,
    );

    expect(result.mode).toBe('ast-reused');
    const parsed = result.parsedFile;
    expect(parsed).toBeDefined();

    const typeBindings = new Map<string, { rawName: string; source: string }>();
    for (const scope of parsed!.scopes) {
      for (const [name, typeRef] of scope.typeBindings) {
        typeBindings.set(name, { rawName: typeRef.rawName, source: typeRef.source });
      }
    }

    expect(typeBindings.get('profile')).toEqual({
      rawName: 'user.profile',
      source: 'field-access',
    });
    expect(typeBindings.get('name')).toEqual({
      rawName: 'user.displayName',
      source: 'field-access',
    });
  });

  it('emits object-pattern field-access bindings from call-result receivers', () => {
    const source = `
import { makeUser } from './models';

class Profile {
  save() {}
}

class User {
  profile: Profile;
}

class Provider {
  getUser(): User {
    return new User();
  }
}

async function run(provider: Provider) {
  const { profile } = await makeUser();
  const { profile: fromMethod } = provider.getUser();
}
`;
    const tree = parser.parse(source);
    const result = extractParsedFileWithStats(
      typescriptProvider,
      source,
      'src/destructure-call.ts',
      SupportedLanguages.TypeScript,
      tree.rootNode,
    );

    expect(result.mode).toBe('ast-reused');
    const parsed = result.parsedFile;
    expect(parsed).toBeDefined();

    const typeBindings = new Map<string, { rawName: string; source: string }>();
    for (const scope of parsed!.scopes) {
      for (const [name, typeRef] of scope.typeBindings) {
        typeBindings.set(name, { rawName: typeRef.rawName, source: typeRef.source });
      }
    }

    const profileBinding = typeBindings.get('profile');
    expect(profileBinding).toBeDefined();
    expect(profileBinding!.source).toBe('field-access');
    expect(profileBinding!.rawName).toMatch(/^__destr_makeUser_\d+\.profile$/);
    const callReceiver = profileBinding!.rawName.replace(/\.profile$/, '');
    expect(typeBindings.get(callReceiver)).toEqual({
      rawName: 'makeUser',
      source: 'call-return',
    });

    const methodBinding = typeBindings.get('fromMethod');
    expect(methodBinding).toBeDefined();
    expect(methodBinding!.source).toBe('field-access');
    expect(methodBinding!.rawName).toMatch(/^__destr_getUser_\d+\.profile$/);
    const methodReceiver = methodBinding!.rawName.replace(/\.profile$/, '');
    expect(typeBindings.get(methodReceiver)).toEqual({
      rawName: 'provider.getUser',
      source: 'method-return',
    });
  });
});
