/**
 * `finalizeScopeModel` — turn a workspace's `ParsedFile[]` into a
 * materialized `ScopeResolutionIndexes` (RFC §3.2 Phase 2; Ring 2 PKG #921).
 *
 * Thin integration glue, per issue #884's boundary: all algorithmic logic
 * lives in `avmatrix-shared` (finalize algorithm #915, the four per-file
 * indexes #913, the method-dispatch materialization #914, the scope tree
 * #912). This file does three things only:
 *
 *   1. Map `ParsedFile[]` → `FinalizeInput` and call shared `finalize()`.
 *   2. Build the four workspace-wide indexes from the union of per-file
 *      defs/scopes/modules/qualified-names.
 *   3. Bundle the results into `ScopeResolutionIndexes` for
 *      `MutableSemanticModel.attachScopeIndexes(...)`.
 *
 * ## What this module is NOT responsible for
 *
 *   - Invoking tree-sitter or running AST walks. That's the extractor (#919).
 *   - Per-language import-target resolution. Hooks are plumbed through
 *     but default to "unresolved" when no provider supplies them — the
 *     real adapters land with #922.
 *   - Populating `ReferenceIndex`. That's the resolution phase (#925).
 *   - Deciding which language uses registry-primary lookup. That's the
 *     flag reader (#924).
 *
 * ## Empty-input behavior
 *
 * When `parsedFiles` is empty (the common case today — no language has
 * migrated yet), the orchestrator produces a valid but empty bundle: all
 * indexes are zero-sized, the scope tree is empty, and
 * `finalize.stats.totalFiles === 0`. This lets downstream consumers
 * safely consult `model.scopes` without branching on presence.
 */

import type {
  BindingRef,
  DefId,
  FinalizeFile,
  FinalizeHooks,
  ParsedFile,
  ReferenceSite,
  Scope,
  ScopeId,
  ScopeTree,
  SymbolDefinition,
  WorkspaceIndex,
} from 'avmatrix-shared';
import {
  buildDefIndex,
  buildMethodDispatchIndex,
  buildModuleScopeIndex,
  buildQualifiedNameIndex,
  buildScopeTree,
  finalize,
} from 'avmatrix-shared';
import type { ScopeResolutionIndexes } from './model/scope-resolution-indexes.js';

// ─── Public entry point ─────────────────────────────────────────────────────

/**
 * Options forwarded to the orchestrator. All fields optional so callers
 * that don't yet have per-language hooks (today) get sensible defaults;
 * #922 will populate `hooks.resolveImportTarget` + friends per language.
 */
export interface FinalizeOrchestratorOptions {
  /**
   * Hooks forwarded to shared `finalize()`. Any omitted field gets a
   * no-op default: unresolved targets, empty wildcard expansion, append
   * merge for bindings.
   */
  readonly hooks?: Partial<FinalizeHooks>;
  /**
   * Opaque workspace context forwarded to hooks. `undefined` today; Ring
   * 2 PKG #922 populates this with a real cross-file index for the
   * per-language resolvers.
   */
  readonly workspaceIndex?: WorkspaceIndex;
}

/**
 * Produce a fully materialized `ScopeResolutionIndexes` from the
 * workspace's per-file artifacts.
 *
 * Pure function (given pure hooks). No I/O, no globals consulted. The
 * pipeline calls this once per ingestion run and hands the result to
 * `MutableSemanticModel.attachScopeIndexes`.
 */
export function finalizeScopeModel(
  parsedFiles: readonly ParsedFile[],
  options: FinalizeOrchestratorOptions = {},
): ScopeResolutionIndexes {
  const hooks = withDefaultHooks(options.hooks ?? {});
  const workspaceIndex: WorkspaceIndex = options.workspaceIndex ?? undefined;

  // ── Step 1: Shared finalize — runs SCC-aware cross-file link + binding
  // materialization. Returns linked imports + merged bindings per module
  // scope + SCC condensation + stats.
  const finalizeInput = {
    files: parsedFiles.map(toFinalizeFile),
    workspaceIndex,
  };
  const finalizeOut = finalize(finalizeInput, hooks);

  // ── Step 2: Workspace-wide indexes built from the per-file unions.
  // These are pure aggregations — no algorithm beyond what the builders
  // in avmatrix-shared already encapsulate (first-write-wins, qname
  // collision buckets, etc.).

  const allScopes: Scope[] = [];
  const allDefs: SymbolDefinition[] = [];
  const moduleEntries: { filePath: string; moduleScopeId: ScopeId }[] = [];
  const fileHashes = new Map<string, string>();
  const allReferenceSites = [] as ReturnType<typeof collectReferenceSites>;

  for (const file of parsedFiles) {
    if (file.fileHash !== undefined) fileHashes.set(file.filePath, file.fileHash);
    for (const s of file.scopes) {
      allScopes.push(withFinalizedImportBindings(s, file.moduleScope, finalizeOut.bindings));
    }
    for (const d of file.localDefs) allDefs.push(d);
    moduleEntries.push({ filePath: file.filePath, moduleScopeId: file.moduleScope });
  }
  // References kept out of the loop above to centralize list-init.
  appendAll(allReferenceSites, collectReferenceSites(parsedFiles));

  const scopeTree = buildScopeTree(allScopes);
  const defs = buildDefIndex(allDefs);
  const qualifiedNames = buildQualifiedNameIndex(allDefs);
  const moduleScopes = buildModuleScopeIndex(moduleEntries);

  // ── Step 3: MethodDispatchIndex. Use pre-resolution `inherits`
  // reference sites to materialize a deterministic owner → ancestor view
  // before the resolution phase asks method/field registries to walk MRO.
  const methodDispatch = buildMethodDispatchFromReferenceSites(allReferenceSites, scopeTree, defs);

  return {
    scopeTree,
    defs,
    qualifiedNames,
    moduleScopes,
    methodDispatch,
    imports: finalizeOut.imports,
    bindings: finalizeOut.bindings,
    fileHashes,
    referenceSites: Object.freeze([...allReferenceSites]),
    sccs: finalizeOut.sccs,
    stats: finalizeOut.stats,
  };
}

const DISPATCH_OWNER_TYPES: ReadonlySet<SymbolDefinition['type']> = new Set([
  'Class',
  'Interface',
  'Struct',
  'Trait',
  'Record',
]);

const IMPLEMENTS_TARGET_TYPES: ReadonlySet<SymbolDefinition['type']> = new Set([
  'Interface',
  'Trait',
]);

const EMPTY_DEF_IDS: readonly DefId[] = Object.freeze([]);

function buildMethodDispatchFromReferenceSites(
  referenceSites: readonly ReferenceSite[],
  scopeTree: ScopeTree,
  defs: ReturnType<typeof buildDefIndex>,
) {
  const directParentsByOwner = new Map<DefId, DefId[]>();
  const directInterfacesByOwner = new Map<DefId, DefId[]>();
  const owners = new Set<DefId>();

  for (const def of defs.byId.values()) {
    if (isDispatchOwner(def)) owners.add(def.nodeId);
  }

  for (const site of referenceSites) {
    if (site.kind !== 'inherits') continue;

    const owner = findOwnerDefForScope(site.inScope, scopeTree);
    if (owner === undefined) continue;

    const target = resolveInheritanceTarget(site, scopeTree, defs);
    if (target === undefined || target.nodeId === owner.nodeId) continue;

    owners.add(owner.nodeId);
    if (IMPLEMENTS_TARGET_TYPES.has(target.type)) {
      appendUnique(directInterfacesByOwner, owner.nodeId, target.nodeId);
    } else if (isDispatchOwner(target)) {
      appendUnique(directParentsByOwner, owner.nodeId, target.nodeId);
    }
  }

  return buildMethodDispatchIndex({
    owners: Array.from(owners),
    computeMro: (ownerDefId) => computeMro(ownerDefId, directParentsByOwner),
    implementsOf: (ownerDefId) => directInterfacesByOwner.get(ownerDefId) ?? EMPTY_DEF_IDS,
  });
}

function findOwnerDefForScope(
  startScope: ScopeId,
  scopeTree: ScopeTree,
): SymbolDefinition | undefined {
  let current: ScopeId | null = startScope;
  const visited = new Set<ScopeId>();

  while (current !== null) {
    if (visited.has(current)) return undefined;
    visited.add(current);

    const scope = scopeTree.getScope(current);
    if (scope === undefined) return undefined;
    const owner = scope.ownedDefs.find(isDispatchOwner);
    if (owner !== undefined) return owner;
    current = scope.parent;
  }

  return undefined;
}

function resolveInheritanceTarget(
  site: ReferenceSite,
  scopeTree: ScopeTree,
  defs: ReturnType<typeof buildDefIndex>,
): SymbolDefinition | undefined {
  const bound = resolveLexicalDispatchOwner(site.name, site.inScope, scopeTree);
  if (bound !== undefined) return bound;
  return resolveUniqueDispatchOwnerByName(site.name, defs);
}

function resolveLexicalDispatchOwner(
  name: string,
  startScope: ScopeId,
  scopeTree: ScopeTree,
): SymbolDefinition | undefined {
  let current: ScopeId | null = startScope;
  const visited = new Set<ScopeId>();

  while (current !== null) {
    if (visited.has(current)) return undefined;
    visited.add(current);

    const scope = scopeTree.getScope(current);
    if (scope === undefined) return undefined;

    const bucket = scope.bindings.get(name);
    if (bucket !== undefined && bucket.length > 0) {
      const matches = bucket.map((binding) => binding.def).filter(isDispatchOwner);
      return matches.length === 1 ? matches[0] : undefined;
    }

    current = scope.parent;
  }

  return undefined;
}

function resolveUniqueDispatchOwnerByName(
  name: string,
  defs: ReturnType<typeof buildDefIndex>,
): SymbolDefinition | undefined {
  const matches: SymbolDefinition[] = [];
  for (const def of defs.byId.values()) {
    if (!isDispatchOwner(def)) continue;
    if (def.qualifiedName === name || simpleNameOf(def) === name) matches.push(def);
  }
  return matches.length === 1 ? matches[0] : undefined;
}

function computeMro(ownerDefId: DefId, directParentsByOwner: ReadonlyMap<DefId, readonly DefId[]>) {
  const out: DefId[] = [];
  const visited = new Set<DefId>([ownerDefId]);

  const visit = (id: DefId): void => {
    const parents = directParentsByOwner.get(id) ?? EMPTY_DEF_IDS;
    for (const parent of parents) {
      if (visited.has(parent)) continue;
      visited.add(parent);
      out.push(parent);
      visit(parent);
    }
  };

  visit(ownerDefId);
  return out;
}

function appendUnique(map: Map<DefId, DefId[]>, owner: DefId, target: DefId): void {
  const bucket = map.get(owner) ?? [];
  if (!bucket.includes(target)) bucket.push(target);
  map.set(owner, bucket);
}

function appendAll<T>(target: T[], items: readonly T[]): void {
  for (const item of items) target.push(item);
}

function isDispatchOwner(def: SymbolDefinition): boolean {
  return DISPATCH_OWNER_TYPES.has(def.type);
}

function simpleNameOf(def: SymbolDefinition): string | undefined {
  const qualifiedName = def.qualifiedName;
  if (qualifiedName === undefined || qualifiedName.length === 0) return undefined;
  const dot = qualifiedName.lastIndexOf('.');
  return dot === -1 ? qualifiedName : qualifiedName.slice(dot + 1);
}

function withFinalizedImportBindings(
  scope: Scope,
  moduleScope: ScopeId,
  finalizedBindings: ReadonlyMap<ScopeId, ReadonlyMap<string, readonly BindingRef[]>>,
): Scope {
  if (scope.id !== moduleScope) return scope;

  const moduleBindings = finalizedBindings.get(moduleScope);
  if (moduleBindings === undefined || moduleBindings.size === 0) return scope;

  const bindings = new Map(scope.bindings);
  let changed = false;
  for (const [name, refs] of moduleBindings) {
    if (bindings.has(name)) continue;

    const importedRefs = refs.filter((ref) => ref.origin !== 'local');
    if (importedRefs.length === 0) continue;

    bindings.set(name, Object.freeze(importedRefs.slice()));
    changed = true;
  }

  return changed ? { ...scope, bindings } : scope;
}

// ─── Internal ───────────────────────────────────────────────────────────────

/** Shape-reduce a `ParsedFile` to the narrower `FinalizeFile` the shared
 *  algorithm reads. The subset is stable — `FinalizeFile` is a proper
 *  subset of `ParsedFile`. */
function toFinalizeFile(file: ParsedFile): FinalizeFile {
  return {
    filePath: file.filePath,
    moduleScope: file.moduleScope,
    parsedImports: file.parsedImports,
    localDefs: file.localDefs,
  };
}

/** Flatten every file's reference sites into one list. Order reflects
 *  input-file order, then capture order inside each file. Deterministic. */
function collectReferenceSites(parsedFiles: readonly ParsedFile[]) {
  const out: ParsedFile['referenceSites'][number][] = [];
  for (const file of parsedFiles) {
    for (const site of file.referenceSites) out.push(site);
  }
  return out;
}

/**
 * Fill in no-op defaults for any omitted hook. Keeps `finalize()`
 * behavior well-defined for the zero-provider case today:
 *
 *   - `resolveImportTarget: () => null` — every import edge ends up
 *     `linkStatus: 'unresolved'` (or dynamic-unresolved pass-through).
 *   - `expandsWildcardTo: () => []` — wildcards don't materialize.
 *   - `mergeBindings: (existing, incoming) => [...existing, ...incoming]`
 *     — append without precedence; providers override to implement local-
 *     shadows-import and similar rules.
 */
function withDefaultHooks(partial: Partial<FinalizeHooks>): FinalizeHooks {
  return {
    resolveImportTarget: partial.resolveImportTarget ?? (() => null),
    expandsWildcardTo: partial.expandsWildcardTo ?? (() => []),
    mergeBindings:
      partial.mergeBindings ??
      ((
        existing: readonly BindingRef[],
        incoming: readonly BindingRef[],
      ): readonly BindingRef[] => [...existing, ...incoming]),
  };
}
