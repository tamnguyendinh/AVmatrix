import { beforeAll, describe, expect, it } from 'vitest';
import type Parser from 'tree-sitter';
import { SupportedLanguages } from 'avmatrix-shared';
import { createParserForLanguage } from '../../../src/core/tree-sitter/parser-loader.js';
import { pythonProvider } from '../../../src/core/ingestion/languages/python.js';
import { extractParsedFileWithStats } from '../../../src/core/ingestion/scope-extractor-bridge.js';
import { finalizeScopeModel } from '../../../src/core/ingestion/finalize-orchestrator.js';
import { resolveScopeReferenceSites } from '../../../src/core/ingestion/scope-reference-resolver.js';

describe('Python AST-aware scope captures', () => {
  let parser: Parser;

  beforeAll(async () => {
    parser = await createParserForLanguage(SupportedLanguages.Python, 'agent.py');
  });

  it('uses the AST-aware optimized hook without a source-text compatibility hook', () => {
    expect(pythonProvider.emitScopeCapturesFromTree).toBeDefined();
    expect(pythonProvider.emitScopeCaptures).toBeUndefined();
  });

  it('emits Python scope facts from the already-parsed AST', () => {
    const source = `
import numpy as np
from .models import User, Repo as R
from pkg import *

class AvmatrixAgent:
    def __init__(self):
        self.avmatrix_metrics = AvmatrixMetrics()

    def serialize(self, *extra_dicts) -> dict:
        avmatrix_data = {
            "metrics": self.avmatrix_metrics.to_dict(),
        }
        return avmatrix_data

class AvmatrixMetrics:
    def to_dict(self) -> dict:
        return {}
`;
    const tree = parser.parse(source);
    const result = extractParsedFileWithStats(
      pythonProvider,
      source,
      'eval/agents/avmatrix_agent.py',
      SupportedLanguages.Python,
      tree.rootNode,
    );

    expect(result.mode).toBe('ast-reused');
    const parsed = result.parsedFile;
    expect(parsed).toBeDefined();

    expect(parsed!.scopes.map((scope) => scope.kind)).toEqual(
      expect.arrayContaining(['Module', 'Class', 'Function']),
    );

    const defs = parsed!.localDefs.map((def) => `${def.type}:${def.qualifiedName}`).sort();
    expect(defs).toEqual(
      expect.arrayContaining([
        'Class:AvmatrixAgent',
        'Class:AvmatrixMetrics',
        'Method:AvmatrixAgent.__init__',
        'Method:AvmatrixAgent.serialize',
        'Method:AvmatrixMetrics.to_dict',
        'Property:AvmatrixAgent.avmatrix_metrics',
      ]),
    );

    const agent = parsed!.localDefs.find(
      (def) => def.type === 'Class' && def.qualifiedName === 'AvmatrixAgent',
    );
    const serialize = parsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'AvmatrixAgent.serialize',
    );
    const metrics = parsed!.localDefs.find(
      (def) => def.type === 'Property' && def.qualifiedName === 'AvmatrixAgent.avmatrix_metrics',
    );
    expect(agent).toBeDefined();
    expect(serialize?.ownerId).toBe(agent!.nodeId);
    expect(metrics?.ownerId).toBe(agent!.nodeId);
    expect(metrics?.declaredType).toBe('AvmatrixMetrics');

    expect(parsed!.parsedImports).toEqual(
      expect.arrayContaining([
        { kind: 'namespace', localName: 'np', importedName: 'numpy', targetRaw: 'numpy' },
        { kind: 'named', localName: 'User', importedName: 'User', targetRaw: '.models' },
        {
          kind: 'alias',
          localName: 'R',
          importedName: 'Repo',
          alias: 'R',
          targetRaw: '.models',
        },
        { kind: 'wildcard', targetRaw: 'pkg' },
      ]),
    );

    const selfBindings = parsed!.scopes
      .map((scope) => scope.typeBindings.get('self'))
      .filter((binding) => binding !== undefined);
    expect(selfBindings).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ rawName: 'AvmatrixAgent', source: 'self' }),
        expect.objectContaining({ rawName: 'AvmatrixMetrics', source: 'self' }),
      ]),
    );

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
          name: 'to_dict',
          kind: 'call',
          callForm: 'member',
          receiver: 'self.avmatrix_metrics',
          arity: 0,
        },
      ]),
    );
  });

  it('resolves Python dotted self member calls without rereading or reparsing source', () => {
    const source = `
class AvmatrixAgent:
    def __init__(self):
        self.avmatrix_metrics = AvmatrixMetrics()

    def serialize(self, *extra_dicts) -> dict:
        return self.avmatrix_metrics.to_dict()

class AvmatrixMetrics:
    def to_dict(self) -> dict:
        return {}
`;
    const parsed = extractParsedFileWithStats(
      pythonProvider,
      source,
      'eval/agents/avmatrix_agent.py',
      SupportedLanguages.Python,
      parser.parse(source).rootNode,
    ).parsedFile;
    expect(parsed).toBeDefined();

    const indexes = finalizeScopeModel([parsed!]);
    const result = resolveScopeReferenceSites(indexes);

    const toDict = parsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'AvmatrixMetrics.to_dict',
    );
    expect(toDict).toBeDefined();

    const refsToToDict = result.referenceIndex.byTargetDef.get(toDict!.nodeId) ?? [];
    expect(refsToToDict.map((ref) => ref.kind)).toEqual(['call']);
    expect(result.stats.resolvedCalls).toBeGreaterThanOrEqual(1);
  });

  it('resolves Python annotated self-field accesses, calls, and type uses', () => {
    const source = `
class User:
    def save(self) -> None:
        pass

class Admin(User):
    pass

class Service:
    def __init__(self, user: User):
        self.user = user

    def run(self, current: User) -> User:
        self.user = current
        self.user.save()
        return current
`;
    const parsed = extractParsedFileWithStats(
      pythonProvider,
      source,
      'src/service.py',
      SupportedLanguages.Python,
      parser.parse(source).rootNode,
    ).parsedFile;
    expect(parsed).toBeDefined();

    const service = parsed!.localDefs.find(
      (def) => def.type === 'Class' && def.qualifiedName === 'Service',
    );
    const userClass = parsed!.localDefs.find(
      (def) => def.type === 'Class' && def.qualifiedName === 'User',
    );
    const userProperty = parsed!.localDefs.find(
      (def) => def.type === 'Property' && def.qualifiedName === 'Service.user',
    );
    const save = parsed!.localDefs.find(
      (def) => def.type === 'Method' && def.qualifiedName === 'User.save',
    );
    expect(service).toBeDefined();
    expect(userClass).toBeDefined();
    expect(userProperty?.ownerId).toBe(service!.nodeId);
    expect(userProperty?.declaredType).toBe('User');
    expect(save).toBeDefined();

    const indexes = finalizeScopeModel([parsed!]);
    const result = resolveScopeReferenceSites(indexes);
    expect(
      parsed!.referenceSites.find((site) => site.kind === 'inherits' && site.name === 'User')
        ?.heritageKind,
    ).toBe('extends');

    const refsToSave = result.referenceIndex.byTargetDef.get(save!.nodeId) ?? [];
    expect(refsToSave.map((ref) => ref.kind)).toEqual(['call']);

    const refsToProperty = result.referenceIndex.byTargetDef.get(userProperty!.nodeId) ?? [];
    expect(refsToProperty.map((ref) => ref.kind).sort()).toEqual(['read', 'write', 'write']);

    const refsToUser = result.referenceIndex.byTargetDef.get(userClass!.nodeId) ?? [];
    expect(refsToUser.filter((ref) => ref.kind === 'type-reference')).toHaveLength(3);
    expect(refsToUser.filter((ref) => ref.kind === 'inherits')).toHaveLength(1);
    expect(result.stats.resolvedCalls).toBeGreaterThanOrEqual(1);
    expect(result.stats.resolvedAccesses).toBeGreaterThanOrEqual(3);
    expect(result.stats.resolvedTypeReferences).toBeGreaterThanOrEqual(3);
    expect(result.stats.resolvedInheritance).toBeGreaterThanOrEqual(1);
  });
});
