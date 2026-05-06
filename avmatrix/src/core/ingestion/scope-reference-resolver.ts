import {
  buildClassRegistry,
  buildDefIndex,
  buildFieldRegistry,
  buildMethodRegistry,
  buildModuleScopeIndex,
  buildQualifiedNameIndex,
  buildScopeTree,
  type BindingRef,
  type DefId,
  type ImportEdge,
  type Reference,
  type ReferenceIndex,
  type ReferenceSite,
  type RegistryContext,
  type Resolution,
  type Scope,
  type ScopeId,
  type SymbolDefinition,
  type TypeRef,
} from 'avmatrix-shared';
import { Buffer } from 'node:buffer';
import fs from 'node:fs';
import path from 'node:path';
import { performance } from 'node:perf_hooks';
import { fileURLToPath, pathToFileURL } from 'node:url';
import type { ScopeResolutionIndexes } from './model/scope-resolution-indexes.js';
import { createWorkerPool } from './workers/worker-pool.js';

const DEFAULT_REFERENCE_RESOLUTION_CHUNK_SIZE = 1000;
const DEFAULT_PARALLEL_REFERENCE_SITE_THRESHOLD = DEFAULT_REFERENCE_RESOLUTION_CHUNK_SIZE * 2;

export interface ScopeReferenceResolutionOptions {
  readonly chunkSize?: number;
  readonly readonlyIndexBytes?: number;
  readonly readonlyIndexInitMs?: number;
  readonly useWorkers?: boolean;
  readonly workerCount?: number;
  readonly minWorkerReferenceSites?: number;
}

export interface ScopeReferenceResolutionStats {
  readonly totalReferenceSites: number;
  readonly chunkSize: number;
  readonly chunksResolved: number;
  readonly maxChunkReferenceSites: number;
  readonly resolvedReferences: number;
  readonly unresolvedReferences: number;
  readonly resolvedCalls: number;
  readonly resolvedAccesses: number;
  readonly resolvedTypeReferences: number;
  readonly resolvedInheritance: number;
  readonly resolvedImportUses: number;
  readonly referenceIndexSourceScopes: number;
  readonly referenceIndexTargetDefs: number;
  readonly readonlyIndexBytes: number;
}

export interface ScopeReferenceResolutionTimings {
  readonly readonlyIndexInitMs: number;
  readonly referenceWorkerResolveMs: number;
  readonly referenceMergeMs: number;
  readonly referenceIndexBuildMs: number;
}

export interface ScopeReferenceResolutionResult {
  readonly referenceIndex: ReferenceIndex;
  readonly stats: ScopeReferenceResolutionStats;
  readonly timings: ScopeReferenceResolutionTimings;
}

export interface ReferenceResolutionChunk {
  readonly chunkId: number;
  readonly referenceSites: readonly ReferenceSite[];
}

export interface ReferenceResolutionChunkStats {
  readonly totalReferenceSites: number;
  readonly resolvedReferences: number;
  readonly unresolvedReferences: number;
  readonly resolvedCalls: number;
  readonly resolvedAccesses: number;
  readonly resolvedTypeReferences: number;
  readonly resolvedInheritance: number;
  readonly resolvedImportUses: number;
}

export interface ReferenceResolutionChunkResult {
  readonly chunkId: number;
  readonly refs: readonly Reference[];
  readonly stats: ReferenceResolutionChunkStats;
}

export interface ReferenceResolutionContext {
  readonly scopes: ScopeResolutionIndexes;
  readonly classRegistry: ReturnType<typeof buildClassRegistry>;
  readonly methodRegistry: ReturnType<typeof buildMethodRegistry>;
  readonly fieldRegistry: ReturnType<typeof buildFieldRegistry>;
}

export interface SerializedScope {
  readonly id: Scope['id'];
  readonly parent: Scope['parent'];
  readonly kind: Scope['kind'];
  readonly range: Scope['range'];
  readonly filePath: Scope['filePath'];
  readonly bindings: readonly [string, readonly BindingRef[]][];
  readonly ownedDefs: readonly SymbolDefinition[];
  readonly imports: readonly ImportEdge[];
  readonly typeBindings: readonly [string, TypeRef][];
}

export interface SerializedScopeResolutionIndexes {
  readonly scopes: readonly SerializedScope[];
  readonly defs: readonly SymbolDefinition[];
  readonly moduleScopes: readonly [string, ScopeId][];
  readonly methodDispatch: {
    readonly mroByOwnerDefId: readonly [DefId, readonly DefId[]][];
    readonly implsByInterfaceDefId: readonly [DefId, readonly DefId[]][];
  };
  readonly fileHashes: readonly [string, string][];
}

export function resolveScopeReferenceSites(
  scopes: ScopeResolutionIndexes,
  options: ScopeReferenceResolutionOptions = {},
): ScopeReferenceResolutionResult {
  const initStart = performance.now();
  const ctx = createReferenceResolutionContext(scopes);
  const readonlyIndexBytes = options.readonlyIndexBytes ?? estimateReadonlyIndexBytes(scopes);
  const readonlyIndexInitMs = options.readonlyIndexInitMs ?? performance.now() - initStart;

  const chunkSize = normalizeChunkSize(options.chunkSize);
  const chunks = createReferenceResolutionChunks(scopes.referenceSites, chunkSize);

  const workerStart = performance.now();
  const chunkResults = chunks.map((chunk) => resolveReferenceSiteChunk(ctx, chunk));
  const referenceWorkerResolveMs = performance.now() - workerStart;

  return mergeReferenceResolutionChunks(scopes.referenceSites.length, chunkSize, chunkResults, {
    readonlyIndexBytes,
    readonlyIndexInitMs,
    referenceWorkerResolveMs,
  });
}

export async function resolveScopeReferenceSitesInWorkers(
  scopes: ScopeResolutionIndexes,
  options: ScopeReferenceResolutionOptions = {},
): Promise<ScopeReferenceResolutionResult> {
  const chunkSize = normalizeChunkSize(options.chunkSize);
  if (!shouldUseReferenceWorkers(scopes.referenceSites.length, options)) {
    return resolveScopeReferenceSites(scopes, { ...options, chunkSize });
  }

  const initStart = performance.now();
  const serialized = serializeScopeResolutionIndexes(scopes);
  const readonlyIndexBytes = options.readonlyIndexBytes ?? estimateReadonlyIndexBytes(scopes);
  const workerUrl = resolveReferenceResolutionWorkerUrl();
  const pool = createWorkerPool(workerUrl, options.workerCount, {
    workerData: { scopeResolutionIndexes: serialized },
  });
  const readonlyIndexInitMs = options.readonlyIndexInitMs ?? performance.now() - initStart;

  try {
    const chunks = createReferenceResolutionChunks(scopes.referenceSites, chunkSize);
    const workerStart = performance.now();
    const chunkResults = await pool.dispatch<
      ReferenceResolutionChunk,
      ReferenceResolutionChunkResult
    >([...chunks], undefined, {
      maxFilesPerUnit: 1,
      getItemPath: (chunk) => `reference-chunk:${chunk.chunkId}`,
      getItemSize: (chunk) => estimateReferenceChunkBytes(chunk),
    });
    const referenceWorkerResolveMs = performance.now() - workerStart;

    return mergeReferenceResolutionChunks(scopes.referenceSites.length, chunkSize, chunkResults, {
      readonlyIndexBytes,
      readonlyIndexInitMs,
      referenceWorkerResolveMs,
    });
  } finally {
    await pool.terminate();
  }
}

export function createReferenceResolutionContext(
  scopes: ScopeResolutionIndexes,
): ReferenceResolutionContext {
  const registryContext: RegistryContext = {
    scopes: scopes.scopeTree,
    defs: scopes.defs,
    qualifiedNames: scopes.qualifiedNames,
    moduleScopes: scopes.moduleScopes,
    methodDispatch: scopes.methodDispatch,
    ownedMembersByOwner: buildOwnedMemberIndex(scopes.defs.byId.values()),
    typeBindingByDef: buildTypeBindingByDef(scopes.scopeTree.byId.values()),
    providers: {},
  };

  return {
    scopes,
    classRegistry: buildClassRegistry(registryContext),
    methodRegistry: buildMethodRegistry(registryContext),
    fieldRegistry: buildFieldRegistry(registryContext),
  };
}

export function createReferenceResolutionChunks(
  referenceSites: readonly ReferenceSite[],
  chunkSizeValue?: number,
): readonly ReferenceResolutionChunk[] {
  const chunkSize = normalizeChunkSize(chunkSizeValue);
  const chunks: ReferenceResolutionChunk[] = [];
  for (let offset = 0; offset < referenceSites.length; offset += chunkSize) {
    chunks.push({
      chunkId: chunks.length,
      referenceSites: referenceSites.slice(
        offset,
        Math.min(offset + chunkSize, referenceSites.length),
      ),
    });
  }
  return Object.freeze(chunks);
}

export function resolveReferenceSiteChunk(
  ctx: ReferenceResolutionContext,
  chunk: ReferenceResolutionChunk,
): ReferenceResolutionChunkResult {
  let unresolvedReferences = 0;
  let resolvedCalls = 0;
  let resolvedAccesses = 0;
  let resolvedTypeReferences = 0;
  let resolvedInheritance = 0;
  let resolvedImportUses = 0;
  const refs: Reference[] = [];

  for (const site of chunk.referenceSites) {
    const resolution = bestResolutionForSite(ctx, site);
    if (resolution === undefined) {
      unresolvedReferences++;
      continue;
    }

    const fileHash = fileHashForSite(ctx.scopes, site);
    refs.push({
      fromScope: site.inScope,
      toDef: resolution.def.nodeId,
      ...(fileHash !== undefined ? { fileHash } : {}),
      atRange: site.atRange,
      kind: site.kind,
      confidence: resolution.confidence,
      evidence: resolution.evidence,
    });

    if (site.kind === 'call') resolvedCalls++;
    else if (site.kind === 'read' || site.kind === 'write') resolvedAccesses++;
    else if (site.kind === 'type-reference') resolvedTypeReferences++;
    else if (site.kind === 'inherits') resolvedInheritance++;
    else if (site.kind === 'import-use') resolvedImportUses++;
  }

  return {
    chunkId: chunk.chunkId,
    refs: Object.freeze(refs),
    stats: {
      totalReferenceSites: chunk.referenceSites.length,
      resolvedReferences: refs.length,
      unresolvedReferences,
      resolvedCalls,
      resolvedAccesses,
      resolvedTypeReferences,
      resolvedInheritance,
      resolvedImportUses,
    },
  };
}

export function mergeReferenceResolutionChunks(
  totalReferenceSites: number,
  chunkSize: number,
  chunkResults: readonly ReferenceResolutionChunkResult[],
  timings: {
    readonly readonlyIndexBytes: number;
    readonly readonlyIndexInitMs: number;
    readonly referenceWorkerResolveMs: number;
  },
): ScopeReferenceResolutionResult {
  const mergeStart = performance.now();
  const orderedChunks = [...chunkResults].sort((a, b) => a.chunkId - b.chunkId);
  const refs: Reference[] = [];
  const combinedStats = emptyChunkStats();
  let maxChunkReferenceSites = 0;

  for (const chunk of orderedChunks) {
    refs.push(...chunk.refs);
    maxChunkReferenceSites = Math.max(maxChunkReferenceSites, chunk.stats.totalReferenceSites);
    addChunkStats(combinedStats, chunk.stats);
  }
  const referenceMergeMs = performance.now() - mergeStart;

  const indexStart = performance.now();
  const referenceIndex = buildReferenceIndex(refs);
  const referenceIndexBuildMs = performance.now() - indexStart;

  return {
    referenceIndex,
    stats: {
      totalReferenceSites,
      chunkSize,
      chunksResolved: orderedChunks.length,
      maxChunkReferenceSites,
      resolvedReferences: refs.length,
      unresolvedReferences: combinedStats.unresolvedReferences,
      resolvedCalls: combinedStats.resolvedCalls,
      resolvedAccesses: combinedStats.resolvedAccesses,
      resolvedTypeReferences: combinedStats.resolvedTypeReferences,
      resolvedInheritance: combinedStats.resolvedInheritance,
      resolvedImportUses: combinedStats.resolvedImportUses,
      referenceIndexSourceScopes: referenceIndex.bySourceScope.size,
      referenceIndexTargetDefs: referenceIndex.byTargetDef.size,
      readonlyIndexBytes: timings.readonlyIndexBytes,
    },
    timings: {
      readonlyIndexInitMs: timings.readonlyIndexInitMs,
      referenceWorkerResolveMs: timings.referenceWorkerResolveMs,
      referenceMergeMs,
      referenceIndexBuildMs,
    },
  };
}

export function serializeScopeResolutionIndexes(
  scopes: ScopeResolutionIndexes,
): SerializedScopeResolutionIndexes {
  const serializedScopes: SerializedScope[] = [];
  for (const scope of scopes.scopeTree.byId.values()) {
    serializedScopes.push({
      id: scope.id,
      parent: scope.parent,
      kind: scope.kind,
      range: scope.range,
      filePath: scope.filePath,
      bindings: Array.from(scope.bindings.entries(), ([name, refs]) => [name, refs] as const),
      ownedDefs: scope.ownedDefs,
      imports: scope.imports,
      typeBindings: Array.from(
        scope.typeBindings.entries(),
        ([name, typeRef]) => [name, typeRef] as const,
      ),
    });
  }

  return {
    scopes: serializedScopes,
    defs: Array.from(scopes.defs.byId.values()),
    moduleScopes: Array.from(scopes.moduleScopes.byFilePath.entries()),
    methodDispatch: {
      mroByOwnerDefId: Array.from(scopes.methodDispatch.mroByOwnerDefId.entries()),
      implsByInterfaceDefId: Array.from(scopes.methodDispatch.implsByInterfaceDefId.entries()),
    },
    fileHashes: Array.from(scopes.fileHashes.entries()),
  };
}

export function serializedScopeResolutionIndexBytes(
  serialized: SerializedScopeResolutionIndexes,
): number {
  return Buffer.byteLength(JSON.stringify(serialized), 'utf8');
}

export function deserializeScopeResolutionIndexes(
  serialized: SerializedScopeResolutionIndexes,
): ScopeResolutionIndexes {
  const scopes: Scope[] = serialized.scopes.map((scope) => ({
    id: scope.id,
    parent: scope.parent,
    kind: scope.kind,
    range: scope.range,
    filePath: scope.filePath,
    bindings: new Map(scope.bindings),
    ownedDefs: scope.ownedDefs,
    imports: scope.imports,
    typeBindings: new Map(scope.typeBindings),
  }));

  const defs = buildDefIndex(serialized.defs);
  return {
    scopeTree: buildScopeTree(scopes),
    defs,
    qualifiedNames: buildQualifiedNameIndex(serialized.defs),
    moduleScopes: buildModuleScopeIndex(
      serialized.moduleScopes.map(([filePath, moduleScopeId]) => ({ filePath, moduleScopeId })),
    ),
    methodDispatch: deserializeMethodDispatch(serialized.methodDispatch),
    imports: new Map(),
    bindings: new Map(),
    fileHashes: new Map(serialized.fileHashes),
    referenceSites: Object.freeze([]),
    sccs: Object.freeze([]),
    stats: {
      totalFiles: 0,
      totalEdges: 0,
      linkedEdges: 0,
      unresolvedEdges: 0,
      sccCount: 0,
      largestSccSize: 0,
    },
  };
}

function bestResolutionForSite(
  ctx: ReferenceResolutionContext,
  site: ReferenceSite,
): Resolution | undefined {
  if (site.kind === 'call') {
    if (site.callForm === 'constructor') {
      return (
        ctx.classRegistry.lookup(site.name, site.inScope)[0] ??
        ctx.methodRegistry.lookup(site.name, site.inScope, methodOptions(site))[0]
      );
    }
    return ctx.methodRegistry.lookup(site.name, site.inScope, methodOptions(site))[0];
  }

  if (site.kind === 'read' || site.kind === 'write') {
    return ctx.fieldRegistry.lookup(site.name, site.inScope, {
      ...(site.explicitReceiver !== undefined ? { explicitReceiver: site.explicitReceiver } : {}),
    })[0];
  }

  if (site.kind === 'type-reference' || site.kind === 'inherits') {
    return ctx.classRegistry.lookup(site.name, site.inScope)[0];
  }

  return (
    ctx.classRegistry.lookup(site.name, site.inScope)[0] ??
    ctx.methodRegistry.lookup(site.name, site.inScope, methodOptions(site))[0] ??
    ctx.fieldRegistry.lookup(site.name, site.inScope, {
      ...(site.explicitReceiver !== undefined ? { explicitReceiver: site.explicitReceiver } : {}),
    })[0]
  );
}

function fileHashForSite(scopes: ScopeResolutionIndexes, site: ReferenceSite): string | undefined {
  const scope = scopes.scopeTree.getScope(site.inScope);
  if (scope === undefined) return undefined;
  return scopes.fileHashes.get(scope.filePath);
}

function estimateReadonlyIndexBytes(scopes: ScopeResolutionIndexes): number {
  let bytes = 0;

  for (const scope of scopes.scopeTree.byId.values()) {
    bytes += stringBytes(scope.id);
    bytes += stringBytes(scope.parent);
    bytes += stringBytes(scope.kind);
    bytes += stringBytes(scope.filePath);
    for (const def of scope.ownedDefs) bytes += stringBytes(def.nodeId);
    for (const [name, bindings] of scope.bindings) {
      bytes += stringBytes(name);
      for (const binding of bindings) {
        bytes += stringBytes(binding.origin);
        bytes += stringBytes(binding.def.nodeId);
        bytes += stringBytes(binding.via?.targetFile);
        bytes += stringBytes(binding.via?.targetDefId);
      }
    }
    for (const [name, typeRef] of scope.typeBindings) {
      bytes += stringBytes(name);
      bytes += stringBytes(typeRef.rawName);
      bytes += stringBytes(typeRef.declaredAtScope);
      bytes += stringBytes(typeRef.source);
    }
  }

  for (const def of scopes.defs.byId.values()) {
    bytes += stringBytes(def.nodeId);
    bytes += stringBytes(def.type);
    bytes += stringBytes(def.filePath);
    bytes += stringBytes(def.qualifiedName);
    bytes += stringBytes(def.ownerId);
  }

  for (const [scopeId, edges] of scopes.imports) {
    bytes += stringBytes(scopeId);
    for (const edge of edges) {
      bytes += stringBytes(edge.localName);
      bytes += stringBytes(edge.targetFile);
      bytes += stringBytes(edge.targetExportedName);
      bytes += stringBytes(edge.targetModuleScope);
      bytes += stringBytes(edge.targetDefId);
      bytes += stringBytes(edge.kind);
      bytes += stringBytes(edge.linkStatus);
      for (const hop of edge.transitiveVia ?? []) bytes += stringBytes(hop);
    }
  }

  for (const [owner, mro] of scopes.methodDispatch.mroByOwnerDefId) {
    bytes += stringBytes(owner);
    for (const defId of mro) bytes += stringBytes(defId);
  }
  for (const [iface, impls] of scopes.methodDispatch.implsByInterfaceDefId) {
    bytes += stringBytes(iface);
    for (const defId of impls) bytes += stringBytes(defId);
  }
  for (const scc of scopes.sccs) {
    for (const file of scc.files) bytes += stringBytes(file);
    bytes += 1;
  }

  return bytes;
}

function stringBytes(value: string | null | undefined): number {
  return value === undefined || value === null ? 0 : Buffer.byteLength(value, 'utf8');
}

function normalizeChunkSize(value: number | undefined): number {
  if (value === undefined) return DEFAULT_REFERENCE_RESOLUTION_CHUNK_SIZE;
  const chunkSize = Math.floor(value);
  return Number.isFinite(chunkSize) && chunkSize > 0
    ? chunkSize
    : DEFAULT_REFERENCE_RESOLUTION_CHUNK_SIZE;
}

function shouldUseReferenceWorkers(
  referenceSiteCount: number,
  options: ScopeReferenceResolutionOptions,
): boolean {
  const enabled = options.useWorkers ?? process.env.AVMATRIX_SCOPE_RESOLUTION_WORKERS === '1';
  if (!enabled) return false;
  const threshold = options.minWorkerReferenceSites ?? DEFAULT_PARALLEL_REFERENCE_SITE_THRESHOLD;
  return referenceSiteCount >= threshold;
}

function estimateReferenceChunkBytes(chunk: ReferenceResolutionChunk): number {
  let bytes = 0;
  for (const site of chunk.referenceSites) {
    bytes += stringBytes(site.name);
    bytes += stringBytes(site.inScope);
    bytes += stringBytes(site.kind);
    bytes += stringBytes(site.callForm);
    bytes += stringBytes(site.explicitReceiver?.name);
    bytes += 32;
  }
  return bytes;
}

function resolveReferenceResolutionWorkerUrl(): URL {
  const sourceJsWorker = new URL('./workers/resolution-worker.js', import.meta.url);
  const sourceJsPath = fileURLToPath(sourceJsWorker);
  if (fs.existsSync(sourceJsPath)) return sourceJsWorker;

  const thisDir = fileURLToPath(new URL('.', import.meta.url));
  const distWorker = path.resolve(
    thisDir,
    '..',
    '..',
    '..',
    'dist',
    'core',
    'ingestion',
    'workers',
    'resolution-worker.js',
  );
  if (fs.existsSync(distWorker)) return pathToFileURL(distWorker);

  throw new Error(
    `Reference resolution worker script not found. Checked ${sourceJsPath} and ${distWorker}. ` +
      'Run `npm run build` in avmatrix or disable resolution workers.',
  );
}

function methodOptions(site: ReferenceSite) {
  return {
    ...(site.arity !== undefined ? { callsite: { arity: site.arity } } : {}),
    ...(site.explicitReceiver !== undefined ? { explicitReceiver: site.explicitReceiver } : {}),
  };
}

function buildOwnedMemberIndex(
  defs: Iterable<SymbolDefinition>,
): ReadonlyMap<DefId, ReadonlyMap<string, readonly SymbolDefinition[]>> {
  const byOwner = new Map<DefId, Map<string, SymbolDefinition[]>>();
  for (const def of defs) {
    if (def.ownerId === undefined) continue;
    const name = simpleNameOf(def);
    if (name === undefined) continue;

    let ownerBucket = byOwner.get(def.ownerId);
    if (ownerBucket === undefined) {
      ownerBucket = new Map<string, SymbolDefinition[]>();
      byOwner.set(def.ownerId, ownerBucket);
    }

    const memberBucket = ownerBucket.get(name) ?? [];
    memberBucket.push(def);
    ownerBucket.set(name, memberBucket);
  }

  const frozenByOwner = new Map<DefId, ReadonlyMap<string, readonly SymbolDefinition[]>>();
  for (const [owner, members] of byOwner) {
    const frozenMembers = new Map<string, readonly SymbolDefinition[]>();
    for (const [name, bucket] of members) {
      frozenMembers.set(name, Object.freeze(bucket.slice()));
    }
    frozenByOwner.set(owner, frozenMembers);
  }
  return frozenByOwner;
}

function buildTypeBindingByDef(scopes: Iterable<Scope>): ReadonlyMap<DefId, TypeRef> {
  const byDef = new Map<DefId, TypeRef>();
  const ambiguous = new Set<DefId>();

  for (const scope of scopes) {
    if (scope.typeBindings.size === 0 || scope.ownedDefs.length === 0) continue;

    const defsBySimpleName = new Map<string, SymbolDefinition[]>();
    for (const def of scope.ownedDefs) {
      const name = simpleNameOf(def);
      if (name === undefined) continue;
      const bucket = defsBySimpleName.get(name) ?? [];
      bucket.push(def);
      defsBySimpleName.set(name, bucket);
    }

    for (const [name, typeRef] of scope.typeBindings) {
      const defs = defsBySimpleName.get(name);
      if (defs === undefined || defs.length !== 1) continue;
      const defId = defs[0]!.nodeId;
      if (ambiguous.has(defId)) continue;

      const existing = byDef.get(defId);
      if (existing !== undefined && !sameTypeRef(existing, typeRef)) {
        byDef.delete(defId);
        ambiguous.add(defId);
        continue;
      }
      byDef.set(defId, typeRef);
    }
  }

  return byDef;
}

function sameTypeRef(left: TypeRef, right: TypeRef): boolean {
  return (
    left.rawName === right.rawName &&
    left.declaredAtScope === right.declaredAtScope &&
    left.source === right.source
  );
}

function simpleNameOf(def: SymbolDefinition): string | undefined {
  const qualifiedName = def.qualifiedName;
  if (qualifiedName === undefined || qualifiedName.length === 0) return undefined;
  const dot = qualifiedName.lastIndexOf('.');
  return dot === -1 ? qualifiedName : qualifiedName.slice(dot + 1);
}

function buildReferenceIndex(refs: readonly Reference[]): ReferenceIndex {
  const bySourceScope = new Map<string, Reference[]>();
  const byTargetDef = new Map<DefId, Reference[]>();

  for (const ref of refs) {
    const sourceBucket = bySourceScope.get(ref.fromScope) ?? [];
    sourceBucket.push(ref);
    bySourceScope.set(ref.fromScope, sourceBucket);

    const targetBucket = byTargetDef.get(ref.toDef) ?? [];
    targetBucket.push(ref);
    byTargetDef.set(ref.toDef, targetBucket);
  }

  return {
    bySourceScope: freezeBuckets(bySourceScope),
    byTargetDef: freezeBuckets(byTargetDef),
  };
}

function freezeBuckets<K>(
  input: ReadonlyMap<K, readonly Reference[]>,
): ReadonlyMap<K, readonly Reference[]> {
  const out = new Map<K, readonly Reference[]>();
  for (const [key, refs] of input) out.set(key, Object.freeze([...refs]));
  return out;
}

function emptyChunkStats(): MutableReferenceResolutionChunkStats {
  return {
    totalReferenceSites: 0,
    resolvedReferences: 0,
    unresolvedReferences: 0,
    resolvedCalls: 0,
    resolvedAccesses: 0,
    resolvedTypeReferences: 0,
    resolvedInheritance: 0,
    resolvedImportUses: 0,
  };
}

interface MutableReferenceResolutionChunkStats {
  totalReferenceSites: number;
  resolvedReferences: number;
  unresolvedReferences: number;
  resolvedCalls: number;
  resolvedAccesses: number;
  resolvedTypeReferences: number;
  resolvedInheritance: number;
  resolvedImportUses: number;
}

function addChunkStats(
  target: MutableReferenceResolutionChunkStats,
  stats: ReferenceResolutionChunkStats,
): void {
  target.totalReferenceSites += stats.totalReferenceSites;
  target.resolvedReferences += stats.resolvedReferences;
  target.unresolvedReferences += stats.unresolvedReferences;
  target.resolvedCalls += stats.resolvedCalls;
  target.resolvedAccesses += stats.resolvedAccesses;
  target.resolvedTypeReferences += stats.resolvedTypeReferences;
  target.resolvedInheritance += stats.resolvedInheritance;
  target.resolvedImportUses += stats.resolvedImportUses;
}

const EMPTY_DEF_IDS: readonly DefId[] = Object.freeze([]);

function deserializeMethodDispatch(serialized: SerializedScopeResolutionIndexes['methodDispatch']) {
  const mroByOwnerDefId = new Map<DefId, readonly DefId[]>(serialized.mroByOwnerDefId);
  const implsByInterfaceDefId = new Map<DefId, readonly DefId[]>(serialized.implsByInterfaceDefId);
  return {
    mroByOwnerDefId,
    implsByInterfaceDefId,
    mroFor(ownerDefId: DefId): readonly DefId[] {
      return mroByOwnerDefId.get(ownerDefId) ?? EMPTY_DEF_IDS;
    },
    implementorsOf(interfaceDefId: DefId): readonly DefId[] {
      return implsByInterfaceDefId.get(interfaceDefId) ?? EMPTY_DEF_IDS;
    },
  };
}
