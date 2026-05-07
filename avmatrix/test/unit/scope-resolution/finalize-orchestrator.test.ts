/**
 * Unit tests for `finalize-orchestrator` for the accurate single-pass graph pipeline.
 *
 * Covers empty-input, single-file, multi-file-with-imports, and the
 * `MutableSemanticModel.attachScopeIndexes` one-shot contract.
 *
 * Builds synthetic `ParsedFile` inputs directly — the orchestrator is
 * below the extraction layer and independent of tree-sitter, so the
 * tests don't need a real parser.
 */

import { describe, it, expect } from 'vitest';
import type {
  BindingRef,
  ParsedFile,
  ParsedImport,
  Scope,
  ScopeId,
  SymbolDefinition,
} from 'avmatrix-shared';
import { finalizeScopeModel } from '../../../src/core/ingestion/finalize-orchestrator.js';
import { createSemanticModel } from '../../../src/core/ingestion/model/semantic-model.js';
import type { ScopeResolutionIndexes } from '../../../src/core/ingestion/model/scope-resolution-indexes.js';

// ─── Fixture helpers ────────────────────────────────────────────────────────

const mkScope = (
  id: ScopeId,
  parent: ScopeId | null,
  filePath: string,
  bindings: Record<string, readonly BindingRef[]> = {},
): Scope => ({
  id,
  parent,
  kind: parent === null ? 'Module' : 'Class',
  range: { startLine: 1, startCol: 0, endLine: 100, endCol: 0 },
  filePath,
  bindings: new Map(Object.entries(bindings)),
  ownedDefs: [],
  imports: [],
  typeBindings: new Map(),
});

const mkFile = (filePath: string, overrides: Partial<ParsedFile> = {}): ParsedFile => ({
  filePath,
  ...(overrides.fileHash !== undefined ? { fileHash: overrides.fileHash } : {}),
  moduleScope: `scope:${filePath}#module`,
  scopes: overrides.scopes ?? [mkScope(`scope:${filePath}#module`, null, filePath)],
  parsedImports: overrides.parsedImports ?? [],
  localDefs: overrides.localDefs ?? [],
  referenceSites: overrides.referenceSites ?? [],
});

const mkDef = (
  nodeId: string,
  filePath: string,
  qname: string,
  type: SymbolDefinition['type'] = 'Class',
): SymbolDefinition => ({
  nodeId,
  filePath,
  type,
  qualifiedName: qname,
});

// ─── Empty input ───────────────────────────────────────────────────────────

describe('finalizeScopeModel: empty input', () => {
  it('produces a valid but empty bundle for zero parsedFiles', () => {
    const out = finalizeScopeModel([]);
    expect(out.scopeTree.size).toBe(0);
    expect(out.defs.size).toBe(0);
    expect(out.qualifiedNames.size).toBe(0);
    expect(out.moduleScopes.size).toBe(0);
    expect(out.methodDispatch.mroByOwnerDefId.size).toBe(0);
    expect(out.imports.size).toBe(0);
    expect(out.fileHashes.size).toBe(0);
    expect(out.bindings.size).toBe(0);
    expect(out.referenceSites).toEqual([]);
    expect(out.sccs).toEqual([]);
    expect(out.stats.totalFiles).toBe(0);
    expect(out.stats.totalEdges).toBe(0);
  });
});

describe('finalizeScopeModel: file hashes', () => {
  it('preserves ParsedFile source hashes for resolution audit metadata', () => {
    const out = finalizeScopeModel([mkFile('src/app.ts', { fileHash: 'sha256:abc' })]);
    expect(out.fileHashes.get('src/app.ts')).toBe('sha256:abc');
  });
});

// ─── Single file ───────────────────────────────────────────────────────────

describe('finalizeScopeModel: single file', () => {
  it('builds all per-file indexes from a single ParsedFile', () => {
    const userClass = mkDef('def:User', 'models.ts', 'models.User');
    const file = mkFile('models.ts', {
      localDefs: [userClass],
    });
    const out = finalizeScopeModel([file]);

    expect(out.scopeTree.size).toBe(1);
    expect(out.defs.get('def:User')).toBe(userClass);
    expect(out.qualifiedNames.get('models.User')).toEqual(['def:User']);
    expect(out.moduleScopes.get('models.ts')).toBe(file.moduleScope);
    expect(out.stats.totalFiles).toBe(1);
  });

  it('forwards per-file referenceSites into the aggregated list', () => {
    const file = mkFile('a.ts', {
      referenceSites: [
        {
          name: 'save',
          atRange: { startLine: 5, startCol: 0, endLine: 5, endCol: 4 },
          inScope: 'scope:a.ts#module',
          kind: 'call',
        },
      ],
    });
    const out = finalizeScopeModel([file]);
    expect(out.referenceSites).toHaveLength(1);
    expect(out.referenceSites[0]!.name).toBe('save');
  });

  it('aggregates large referenceSite lists without relying on spread argument limits', () => {
    const referenceSites = Array.from({ length: 150_000 }, (_, index) => ({
      name: `ref${index}`,
      atRange: { startLine: index + 1, startCol: 0, endLine: index + 1, endCol: 4 },
      inScope: 'scope:a.ts#module',
      kind: 'call' as const,
    }));
    const file = mkFile('a.ts', { referenceSites });

    const out = finalizeScopeModel([file]);

    expect(out.referenceSites).toHaveLength(referenceSites.length);
    expect(out.referenceSites[0]!.name).toBe('ref0');
    expect(out.referenceSites.at(-1)!.name).toBe('ref149999');
  });

  it('builds MethodDispatchIndex from inherits reference sites', () => {
    const baseClass = mkDef('def:Base', 'models.ts', 'Base');
    const childClass = mkDef('def:Child', 'models.ts', 'Child');
    const moduleScope = mkScope('scope:module', null, 'models.ts', {
      Base: [{ def: baseClass, origin: 'local' }],
      Child: [{ def: childClass, origin: 'local' }],
    });
    const baseScope: Scope = {
      ...mkScope('scope:base', 'scope:module', 'models.ts'),
      range: { startLine: 2, startCol: 0, endLine: 10, endCol: 0 },
      ownedDefs: [baseClass],
    };
    const childScope: Scope = {
      ...mkScope('scope:child', 'scope:module', 'models.ts'),
      range: { startLine: 20, startCol: 0, endLine: 30, endCol: 0 },
      ownedDefs: [childClass],
    };
    const file = mkFile('models.ts', {
      moduleScope: 'scope:module',
      scopes: [moduleScope, baseScope, childScope],
      localDefs: [baseClass, childClass],
      referenceSites: [
        {
          name: 'Base',
          atRange: { startLine: 20, startCol: 20, endLine: 20, endCol: 24 },
          inScope: 'scope:child',
          kind: 'inherits',
        },
      ],
    });

    const out = finalizeScopeModel([file]);
    expect(out.methodDispatch.mroFor('def:Child')).toEqual(['def:Base']);
  });

  it('builds first-wins MRO breadth-first instead of repeating parse/cross-file DFS work', () => {
    const baseClass = mkDef('def:Base', 'models.ts', 'Base');
    const leftClass = mkDef('def:Left', 'models.ts', 'Left');
    const rightClass = mkDef('def:Right', 'models.ts', 'Right');
    const childClass = mkDef('def:Child', 'models.ts', 'Child');
    const bindings = Object.fromEntries(
      [baseClass, leftClass, rightClass, childClass].map((def) => [
        def.qualifiedName!,
        [{ def, origin: 'local' as const }],
      ]),
    );
    const moduleScope = mkScope('scope:module', null, 'models.ts', bindings);
    const scopes = [
      moduleScope,
      {
        ...mkScope('scope:base', 'scope:module', 'models.ts'),
        range: { startLine: 2, startCol: 0, endLine: 3, endCol: 0 },
        ownedDefs: [baseClass],
      },
      {
        ...mkScope('scope:left', 'scope:module', 'models.ts'),
        range: { startLine: 4, startCol: 0, endLine: 10, endCol: 0 },
        ownedDefs: [leftClass],
      },
      {
        ...mkScope('scope:right', 'scope:module', 'models.ts'),
        range: { startLine: 11, startCol: 0, endLine: 17, endCol: 0 },
        ownedDefs: [rightClass],
      },
      {
        ...mkScope('scope:child', 'scope:module', 'models.ts'),
        range: { startLine: 18, startCol: 0, endLine: 30, endCol: 0 },
        ownedDefs: [childClass],
      },
    ];
    const file = mkFile('models.ts', {
      moduleScope: 'scope:module',
      scopes,
      localDefs: [baseClass, leftClass, rightClass, childClass],
      referenceSites: [
        {
          name: 'Base',
          atRange: { startLine: 2, startCol: 0, endLine: 2, endCol: 4 },
          inScope: 'scope:left',
          kind: 'inherits',
          heritageKind: 'extends',
        },
        {
          name: 'Base',
          atRange: { startLine: 3, startCol: 0, endLine: 3, endCol: 4 },
          inScope: 'scope:right',
          kind: 'inherits',
          heritageKind: 'extends',
        },
        {
          name: 'Left',
          atRange: { startLine: 4, startCol: 0, endLine: 4, endCol: 4 },
          inScope: 'scope:child',
          kind: 'inherits',
          heritageKind: 'extends',
        },
        {
          name: 'Right',
          atRange: { startLine: 4, startCol: 6, endLine: 4, endCol: 11 },
          inScope: 'scope:child',
          kind: 'inherits',
          heritageKind: 'extends',
        },
      ],
    });

    const out = finalizeScopeModel([file], { mroStrategyForFile: () => 'first-wins' });
    expect(out.methodDispatch.mroFor('def:Child')).toEqual(['def:Left', 'def:Right', 'def:Base']);
  });

  it('honors qualified-syntax MRO by not adding implicit ancestor dispatch', () => {
    const baseClass = mkDef('def:Base', 'models.rs', 'Base');
    const childClass = mkDef('def:Child', 'models.rs', 'Child');
    const moduleScope = mkScope('scope:module', null, 'models.rs', {
      Base: [{ def: baseClass, origin: 'local' }],
      Child: [{ def: childClass, origin: 'local' }],
    });
    const childScope: Scope = {
      ...mkScope('scope:child', 'scope:module', 'models.rs'),
      range: { startLine: 2, startCol: 0, endLine: 10, endCol: 0 },
      ownedDefs: [childClass],
    };
    const file = mkFile('models.rs', {
      moduleScope: 'scope:module',
      scopes: [moduleScope, childScope],
      localDefs: [baseClass, childClass],
      referenceSites: [
        {
          name: 'Base',
          atRange: { startLine: 1, startCol: 0, endLine: 1, endCol: 4 },
          inScope: 'scope:child',
          kind: 'inherits',
          heritageKind: 'extends',
        },
      ],
    });

    const out = finalizeScopeModel([file], { mroStrategyForFile: () => 'qualified-syntax' });
    expect(out.methodDispatch.mroFor('def:Child')).toEqual([]);
  });

  it('materializes implementor buckets from inherited interface facts', () => {
    const iface = mkDef('def:IService', 'models.ts', 'IService', 'Interface');
    const impl = mkDef('def:Service', 'models.ts', 'Service');
    const moduleScope = mkScope('scope:module', null, 'models.ts', {
      IService: [{ def: iface, origin: 'local' }],
      Service: [{ def: impl, origin: 'local' }],
    });
    const serviceScope: Scope = {
      ...mkScope('scope:service', 'scope:module', 'models.ts'),
      range: { startLine: 2, startCol: 0, endLine: 10, endCol: 0 },
      ownedDefs: [impl],
    };
    const file = mkFile('models.ts', {
      moduleScope: 'scope:module',
      scopes: [moduleScope, serviceScope],
      localDefs: [iface, impl],
      referenceSites: [
        {
          name: 'IService',
          atRange: { startLine: 1, startCol: 0, endLine: 1, endCol: 8 },
          inScope: 'scope:service',
          kind: 'inherits',
          heritageKind: 'implements',
        },
      ],
    });

    const out = finalizeScopeModel([file]);
    expect(out.methodDispatch.mroFor('def:Service')).toEqual([]);
    expect(out.methodDispatch.implementorsOf('def:IService')).toEqual(['def:Service']);
  });
});

// ─── Multi-file with cross-file imports ────────────────────────────────────

describe('finalizeScopeModel: cross-file imports', () => {
  it('links a named import when the caller provides resolveImportTarget', () => {
    const userClass = mkDef('def:User', 'models.ts', 'models.User');
    const modelsFile = mkFile('models.ts', { localDefs: [userClass] });

    const importOfUser: ParsedImport = {
      kind: 'named',
      localName: 'User',
      importedName: 'User',
      targetRaw: 'models.ts',
    };
    const appFile = mkFile('app.ts', { parsedImports: [importOfUser] });

    const out = finalizeScopeModel([appFile, modelsFile], {
      hooks: {
        resolveImportTarget: (targetRaw) => (targetRaw === 'models.ts' ? 'models.ts' : null),
      },
    });

    const appImports = out.imports.get(appFile.moduleScope) ?? [];
    expect(appImports).toHaveLength(1);
    expect(appImports[0]!.linkStatus).toBeUndefined();
    expect(appImports[0]!.targetFile).toBe('models.ts');
    expect(appImports[0]!.targetDefId).toBe('def:User');
  });

  it('makes finalized import bindings visible from the module scope tree', () => {
    const userClass = mkDef('def:User', 'models.ts', 'models.User');
    const modelsFile = mkFile('models.ts', { localDefs: [userClass] });
    const appFile = mkFile('app.ts', {
      parsedImports: [
        {
          kind: 'named',
          localName: 'User',
          importedName: 'User',
          targetRaw: 'models.ts',
        },
      ],
    });

    const out = finalizeScopeModel([appFile, modelsFile], {
      hooks: {
        resolveImportTarget: (targetRaw) => (targetRaw === 'models.ts' ? 'models.ts' : null),
      },
    });

    const appScope = out.scopeTree.getScope(appFile.moduleScope);
    const userBindings = appScope?.bindings.get('User') ?? [];
    expect(userBindings).toHaveLength(1);
    expect(userBindings[0]).toMatchObject({
      def: userClass,
      origin: 'import',
    });
  });

  it('leaves imports unresolved when no resolveImportTarget is supplied (default hook)', () => {
    // Default `resolveImportTarget: () => null` — every import ends up
    // with `linkStatus: 'unresolved'`. This is the zero-provider case
    // today; behavior is well-defined, not a crash.
    const importOfUser: ParsedImport = {
      kind: 'named',
      localName: 'User',
      importedName: 'User',
      targetRaw: 'models.ts',
    };
    const appFile = mkFile('app.ts', { parsedImports: [importOfUser] });

    const out = finalizeScopeModel([appFile]);
    const appImports = out.imports.get(appFile.moduleScope) ?? [];
    expect(appImports).toHaveLength(1);
    expect(appImports[0]!.linkStatus).toBe('unresolved');
  });

  it('surfaces FinalizeStats for observability', () => {
    const userClass = mkDef('def:User', 'models.ts', 'models.User');
    const modelsFile = mkFile('models.ts', { localDefs: [userClass] });
    const appFile = mkFile('app.ts', {
      parsedImports: [
        {
          kind: 'named',
          localName: 'User',
          importedName: 'User',
          targetRaw: 'models.ts',
        },
      ],
    });
    const out = finalizeScopeModel([appFile, modelsFile], {
      hooks: { resolveImportTarget: () => 'models.ts' },
    });
    expect(out.stats.totalFiles).toBe(2);
    expect(out.stats.totalEdges).toBe(1);
    expect(out.stats.linkedEdges).toBe(1);
    expect(out.stats.unresolvedEdges).toBe(0);
  });
});

// ─── Integration with MutableSemanticModel ─────────────────────────────────

describe('MutableSemanticModel.attachScopeIndexes', () => {
  it('starts as undefined and accepts a one-shot attach', () => {
    const model = createSemanticModel();
    expect(model.scopes).toBeUndefined();

    const indexes = finalizeScopeModel([]);
    model.attachScopeIndexes(indexes);

    expect(model.scopes).toBe(indexes);
    expect(model.scopes!.stats.totalFiles).toBe(0);
  });

  it('freezes the attached bundle (callers cannot mutate after attach)', () => {
    const model = createSemanticModel();
    const indexes: ScopeResolutionIndexes = finalizeScopeModel([]);
    model.attachScopeIndexes(indexes);

    expect(Object.isFrozen(model.scopes)).toBe(true);
  });

  it('throws on a second attach without clear()', () => {
    const model = createSemanticModel();
    model.attachScopeIndexes(finalizeScopeModel([]));
    expect(() => model.attachScopeIndexes(finalizeScopeModel([]))).toThrowError(/already attached/);
  });

  it('clear() resets the bundle, enabling re-attach', () => {
    const model = createSemanticModel();
    model.attachScopeIndexes(finalizeScopeModel([]));
    model.clear();
    expect(model.scopes).toBeUndefined();
    // Second attach now succeeds.
    model.attachScopeIndexes(finalizeScopeModel([]));
    expect(model.scopes).toBeDefined();
  });
});
