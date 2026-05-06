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
        'Constructor:constructor',
        'Function:makeService',
        'Function:run',
        'Method:save',
        'Property:current',
      ]),
    );

    const service = parsed!.localDefs.find(
      (def) => def.type === 'Class' && def.qualifiedName === 'Service',
    );
    const save = parsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'save',
    );
    const current = parsed!.localDefs.find(
      (def) => def.type === 'Property' && def.qualifiedName === 'current',
    );
    expect(service).toBeDefined();
    expect(save?.ownerId).toBe(service!.nodeId);
    expect(current?.ownerId).toBe(service!.nodeId);

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
});
