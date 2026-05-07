import { describe, expect, it } from 'vitest';
import fs from 'fs/promises';
import path from 'path';
import {
  buildDefIndex,
  buildMethodDispatchIndex,
  buildModuleScopeIndex,
  buildQualifiedNameIndex,
  buildScopeTree,
  type DefId,
  type Range,
  type Reference,
  type ReferenceIndex,
  type Scope,
  type ScopeId,
  type SymbolDefinition,
} from 'avmatrix-shared';
import { withTestLbugDB } from '../helpers/test-indexed-db.js';
import { createKnowledgeGraph } from '../../src/core/graph/graph.js';
import { emitReferencesToGraph } from '../../src/core/ingestion/emit-references.js';
import type { ScopeResolutionIndexes } from '../../src/core/ingestion/model/scope-resolution-indexes.js';

const range = (sl = 1, sc = 0, el = 100, ec = 0): Range => ({
  startLine: sl,
  startCol: sc,
  endLine: el,
  endCol: ec,
});

const def = (
  nodeId: string,
  type: SymbolDefinition['type'],
  qualifiedName: string,
): SymbolDefinition => ({
  nodeId,
  filePath: 'src/app.ts',
  type,
  qualifiedName,
});

const scope = (
  id: ScopeId,
  parent: ScopeId | null,
  kind: Scope['kind'],
  ownedDefs: readonly SymbolDefinition[],
  r: Range = range(),
): Scope => ({
  id,
  parent,
  kind,
  range: r,
  filePath: 'src/app.ts',
  bindings: new Map(),
  ownedDefs,
  imports: [],
  typeBindings: new Map(),
});

function makeIndexes(
  scopes: readonly Scope[],
  defs: readonly SymbolDefinition[],
): ScopeResolutionIndexes {
  return {
    scopeTree: buildScopeTree(scopes),
    defs: buildDefIndex(defs),
    qualifiedNames: buildQualifiedNameIndex(defs),
    moduleScopes: buildModuleScopeIndex([{ filePath: 'src/app.ts', moduleScopeId: 'scope:m' }]),
    methodDispatch: buildMethodDispatchIndex({
      owners: [],
      computeMro: () => [],
      implementsOf: () => [],
    }),
    imports: new Map(),
    bindings: new Map(),
    fileHashes: new Map([['src/app.ts', 'sha256:scope']]),
    referenceSites: [],
    sccs: [],
    stats: {
      totalFiles: 1,
      totalEdges: 0,
      linkedEdges: 0,
      unresolvedEdges: 0,
      sccCount: 0,
      largestSccSize: 0,
    },
  };
}

function buildRefIndex(sourceScope: ScopeId, refs: readonly Reference[]): ReferenceIndex {
  const byTarget = new Map<DefId, Reference[]>();
  for (const ref of refs) {
    const bucket = byTarget.get(ref.toDef) ?? [];
    bucket.push(ref);
    byTarget.set(ref.toDef, bucket);
  }
  return {
    bySourceScope: new Map([[sourceScope, refs]]),
    byTargetDef: new Map(
      Array.from(byTarget.entries()).map(([key, value]) => [key, Object.freeze([...value])]),
    ),
  };
}

function buildOverlapGraph() {
  const run = def('def:run', 'Function', 'run');
  const save = def('def:A.save', 'Method', 'A.save');
  const module = scope('scope:m', null, 'Module', [run, save]);
  const indexes = makeIndexes([module], [run, save]);
  const graph = createKnowledgeGraph();

  graph.addNode({
    id: 'Function:src/app.ts:run',
    label: 'Function',
    properties: { name: 'run', filePath: 'src/app.ts', startLine: 1, endLine: 3 },
  });
  graph.addNode({
    id: 'Method:src/app.ts:A.save#0',
    label: 'Method',
    properties: { name: 'save', filePath: 'src/app.ts', startLine: 5, endLine: 5 },
  });
  graph.addRelationship({
    id: 'compatibility-call',
    sourceId: 'Function:src/app.ts:run',
    targetId: 'Method:src/app.ts:A.save#0',
    type: 'CALLS',
    confidence: 0.5,
    reason: 'compatibility call',
  });

  emitReferencesToGraph({
    graph,
    scopes: indexes,
    referenceIndex: buildRefIndex('scope:m', [
      {
        fromScope: 'scope:m',
        toDef: 'def:A.save',
        fileHash: 'sha256:scope',
        atRange: range(2, 2, 2, 8),
        kind: 'call',
        confidence: 0.95,
        evidence: [{ kind: 'type-binding', weight: 0.35, note: 'receiver A' }],
      },
    ]),
  });

  return graph;
}

withTestLbugDB(
  'scope-audit-overlap',
  () => {
    describe('scope audit persistence', () => {
      it('persists merged scope audit metadata for overlapped CALLS edges', async () => {
        const { executeQuery } = await import('../../src/core/lbug/lbug-adapter.js');
        const rows = await executeQuery(`
          MATCH (a:Function)-[r:CodeRelation]->(b:Method)
          WHERE r.type = 'CALLS'
          RETURN r.type AS type, r.confidence AS confidence, r.reason AS reason,
                 r.resolutionSource AS resolutionSource, r.evidence AS evidence,
                 r.fileHash AS fileHash
        `);

        expect(rows).toHaveLength(1);
        expect(rows[0]).toMatchObject({
          type: 'CALLS',
          confidence: 0.95,
          reason: 'scope-resolution: call | confidence 0.950',
          resolutionSource: 'scope-resolution',
          fileHash: 'sha256:scope',
        });
        expect(rows[0].evidence).toContain('type-binding');
      });
    });
  },
  {
    afterSetup: async (handle) => {
      const { loadGraphToLbug } = await import('../../src/core/lbug/lbug-adapter.js');
      const repoDir = path.join(handle.tmpHandle.dbPath, 'scope-audit-repo');
      await fs.mkdir(path.join(repoDir, 'src'), { recursive: true });
      await fs.writeFile(
        path.join(repoDir, 'src', 'app.ts'),
        'function run() { a.save(); }\nclass A { save() {} }\n',
      );
      const storagePath = path.join(handle.tmpHandle.dbPath, 'scope-audit-storage');
      await fs.mkdir(storagePath, { recursive: true });
      await loadGraphToLbug(buildOverlapGraph(), repoDir, storagePath);
    },
    timeout: 120_000,
  },
);
