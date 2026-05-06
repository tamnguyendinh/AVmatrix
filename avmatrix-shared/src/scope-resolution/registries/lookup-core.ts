/**
 * `lookupCore` — the shared 7-step canonical resolution algorithm
 * (RFC §4.2; Ring 2 SHARED #917).
 *
 * Pure function. Given a name, a starting scope, and per-kind parameters,
 * walks lexical scopes + optional type-binding MRO + optional owner
 * contributor + global qualified-name fallback, and returns a ranked
 * `Resolution[]` with per-candidate evidence.
 *
 * All three public registries (`ClassRegistry` / `MethodRegistry` /
 * `FieldRegistry`) dispatch into this function, differing only in the
 * parameters they pass. The CHOICE of which steps fire is expressed
 * through `LookupParams`, not through different algorithms per kind.
 *
 * ## Algorithm (RFC §4.2, verbatim names)
 *
 * **Step 1 — Lexical scope-chain walk.** From `startScope`, walk
 *   parent-ward. At each scope, consult `scope.bindings.get(name)`:
 *     - Filter candidates whose `def.type ∈ acceptedKinds`.
 *     - For each surviving candidate, record a raw signal with the
 *       binding's origin + the current scope-chain depth.
 *     - **Hard shadow.** If `bindings.get(name)` is non-empty (including
 *       non-kind-matching candidates), stop walking. The name is
 *       lexically bound here; outer scopes are not consulted.
 *
 * **Step 2 — Type-binding resolution.** When `useReceiverTypeBinding`
 *   is true, resolve the receiver's type at `startScope` (from
 *   `scope.typeBindings`), then walk the MRO via
 *   `MethodDispatchIndex.mroFor(ownerDefId)`. Membership per owner comes
 *   through `RegistryContext.methodDispatch` + owner lookups into
 *   `scope.ownedDefs`; each hit records a raw signal with the owner's
 *   MRO depth.
 *
 * **Step 3 — Owner-scoped contributor.** When
 *   `params.ownerScopedContributor` is present, merge its `byName(name)`
 *   hits with `origin: 'local'` (they are declared directly on the
 *   receiver). Distinct from Step 2 — Step 2 walks the MRO; Step 3 only
 *   looks at the directly-declared owner members.
 *
 * **Step 4 — Kind filter (emit `kind-match` evidence).** Already
 *   applied during Steps 1-3; this step just adds a `kind-match` signal
 *   at weight 0 to every candidate for debuggability (so the evidence
 *   array is self-describing).
 *
 * **Step 5 — Arity filter.** Call `providers.arityCompatibility(callsite,
 *   def)` per surviving candidate. Verdicts: `compatible` / `unknown` /
 *   `incompatible`. If at least one candidate is `compatible`, drop
 *   `incompatible` ones. Otherwise keep all (the penalty weight alone
 *   will rank them lower but they remain in the result).
 *
 * **Step 6 — Global fallback.** When Steps 1-3 produced **no**
 *   candidates and the name contains a `.`, consult the
 *   `QualifiedNameIndex` via `lookupQualified` — see §4.5. The `scope`
 *   argument is NOT passed here because global lookup is scope-agnostic.
 *
 * **Step 7 — Rank + tie-break.** Compose evidence, compute confidence
 *   (sum capped at 1.0), sort by the RFC Appendix B cascade.
 *
 * ## What this module does NOT do
 *
 *   - No AST reads (pure data in, pure data out).
 *   - No `avmatrix/` imports.
 *   - No language switches. Language-specific behavior flows exclusively
 *     through `providers.*` and the `params` object.
 *   - No caching. Callers that want memoization can wrap this function.
 */

import type { NodeLabel } from '../../graph/types.js';
import type { SymbolDefinition } from '../symbol-definition.js';
import type {
  BindingRef,
  Callsite,
  DefId,
  LookupParams,
  Resolution,
  Scope,
  ScopeId,
  TypeRef,
} from '../types.js';
import type { OriginForTieBreak } from '../origin-priority.js';
import { composeEvidence, confidenceFromEvidence, type RawSignals } from './evidence.js';
import { compareByConfidenceWithTiebreaks, type TieBreakKey } from './tie-breaks.js';
import { lookupQualified } from './lookup-qualified.js';
import type { ArityVerdict, OwnerScopedContributor, RegistryContext } from './context.js';
import { resolveTypeRef } from '../resolve-type-ref.js';

// ─── Public entry point ─────────────────────────────────────────────────────

/** Extended `LookupParams` narrowing `ownerScopedContributor` to the concrete shape. */
export interface CoreLookupParams extends Omit<LookupParams, 'ownerScopedContributor'> {
  readonly ownerScopedContributor: OwnerScopedContributor | null;
  /** Call-site description forwarded to `arityCompatibility`. Optional — for non-call lookups. */
  readonly callsite?: Callsite;
}

/**
 * Run the 7-step lookup. Returns a non-empty `Resolution[]` when any
 * candidate was found; an empty array otherwise. Callers consume `[0]`
 * for the best answer and optionally inspect the rest for alternates.
 */
export function lookupCore(
  name: string,
  startScope: ScopeId,
  params: CoreLookupParams,
  ctx: RegistryContext,
): readonly Resolution[] {
  const acceptedKinds = new Set<NodeLabel>(params.acceptedKinds);
  const perCandidate = new Map<DefId, CandidateState>();

  // ── Step 1: lexical scope-chain walk ──────────────────────────────────
  const lexicalShadowed = walkLexicalChain(name, startScope, acceptedKinds, ctx, perCandidate);

  // ── Step 2: type-binding / MRO walk (methods/fields) ──────────────────
  if (params.useReceiverTypeBinding && ctx.methodDispatch !== undefined) {
    walkReceiverTypeBinding(name, startScope, acceptedKinds, params, ctx, perCandidate);
  }

  // ── Step 3: owner-scoped contributor ──────────────────────────────────
  if (params.ownerScopedContributor !== null) {
    seedFromOwnerScopedContributor(
      name,
      params.ownerScopedContributor,
      acceptedKinds,
      perCandidate,
    );
  }

  // ── Step 4: kind-match evidence (emitted by composeEvidence directly) ──
  // Handled inside `composeEvidence`.

  // ── Step 5: arity filter ──────────────────────────────────────────────
  if (params.callsite !== undefined) {
    applyArityFilter(params.callsite, perCandidate, ctx);
  }

  // ── Step 6: global fallback (only when Steps 1-3 produced nothing) ──
  if (perCandidate.size === 0 && !lexicalShadowed && name.includes('.')) {
    const globals = lookupQualified(name, { acceptedKinds: params.acceptedKinds }, ctx);
    if (globals.length > 0) return globals;
  }

  if (perCandidate.size === 0) return EMPTY;

  // ── Step 7: compose evidence + rank ──────────────────────────────────
  return rankCandidates(perCandidate);
}

// ─── Internal state ────────────────────────────────────────────────────────

interface CandidateState {
  readonly def: SymbolDefinition;
  readonly signals: MutableRawSignals;
  readonly tieBreakKey: MutableTieBreakKey;
}

interface MutableRawSignals {
  origin?: BindingRef['origin'] | 'global-qualified' | 'global-name';
  scopeChainDepth?: number;
  viaUnlinkedImport?: boolean;
  typeBindingMroDepth?: number;
  ownerMatch?: boolean;
  kindMatch: true;
  arityVerdict?: ArityVerdict;
  dynamicUnresolved?: boolean;
}

interface MutableTieBreakKey {
  scopeDepth: number;
  mroDepth: number;
  origin: OriginForTieBreak;
}

function ensureCandidate(
  perCandidate: Map<DefId, CandidateState>,
  def: SymbolDefinition,
): CandidateState {
  const existing = perCandidate.get(def.nodeId);
  if (existing !== undefined) return existing;
  const fresh: CandidateState = {
    def,
    signals: { kindMatch: true },
    tieBreakKey: { scopeDepth: 0, mroDepth: 0, origin: 'local' },
  };
  perCandidate.set(def.nodeId, fresh);
  return fresh;
}

// ─── Step 1 implementation ─────────────────────────────────────────────────

/**
 * Walk the lexical scope chain from `startScope` upward. Returns `true`
 * iff a scope with any `bindings.get(name)` entries was found — the
 * caller uses this to decide whether to run the global fallback.
 */
function walkLexicalChain(
  name: string,
  startScope: ScopeId,
  acceptedKinds: ReadonlySet<NodeLabel>,
  ctx: RegistryContext,
  perCandidate: Map<DefId, CandidateState>,
): boolean {
  let currentId: ScopeId | null = startScope;
  let depth = 0;
  const visited = new Set<ScopeId>();

  while (currentId !== null) {
    if (visited.has(currentId)) return false;
    visited.add(currentId);

    const scope: Scope | undefined = ctx.scopes.getScope(currentId);
    if (scope === undefined) return false;

    const bindings = scope.bindings.get(name);
    if (bindings !== undefined && bindings.length > 0) {
      for (const binding of bindings) {
        if (!acceptedKinds.has(binding.def.type)) continue;
        recordLexicalHit(perCandidate, binding, depth);
      }
      return true; // hard shadow regardless of kind-filter survivorship
    }

    currentId = scope.parent;
    depth++;
  }

  return false;
}

function recordLexicalHit(
  perCandidate: Map<DefId, CandidateState>,
  binding: BindingRef,
  scopeChainDepth: number,
): void {
  const state = ensureCandidate(perCandidate, binding.def);
  state.signals.origin = binding.origin;
  state.signals.scopeChainDepth = scopeChainDepth;
  if (binding.via?.linkStatus === 'unresolved') {
    state.signals.viaUnlinkedImport = true;
  }
  if (binding.via?.kind === 'dynamic-unresolved') {
    state.signals.dynamicUnresolved = true;
  }
  state.tieBreakKey.scopeDepth = scopeChainDepth;
  state.tieBreakKey.origin = binding.origin as OriginForTieBreak;
}

// ─── Step 2 implementation ─────────────────────────────────────────────────

function walkReceiverTypeBinding(
  name: string,
  startScope: ScopeId,
  acceptedKinds: ReadonlySet<NodeLabel>,
  params: CoreLookupParams,
  ctx: RegistryContext,
  perCandidate: Map<DefId, CandidateState>,
): void {
  const ownerDefId = resolveReceiverOwner(startScope, params, ctx);
  if (ownerDefId === undefined) return;

  if (ctx.methodDispatch === undefined) return;

  const ownerDef = ctx.defs.get(ownerDefId);
  if (ownerDef === undefined) return;

  // Walk the owner itself at depth 0, then its MRO chain.
  const walk: DefId[] = [ownerDefId, ...ctx.methodDispatch.mroFor(ownerDefId)];

  for (let mroDepth = 0; mroDepth < walk.length; mroDepth++) {
    const currentOwnerId = walk[mroDepth]!;
    const members = collectOwnedMembers(currentOwnerId, name, ctx);
    for (const def of members) {
      if (!acceptedKinds.has(def.type)) continue;
      recordTypeBindingHit(perCandidate, def, mroDepth, ownerDefId);
    }
  }
}

function resolveReceiverOwner(
  startScope: ScopeId,
  params: CoreLookupParams,
  ctx: RegistryContext,
): DefId | undefined {
  // Explicit receiver: consult the callsite scope's typeBindings for the
  // named receiver; the attached TypeRef identifies the owner. Without a
  // ready resolveTypeRef call (that module is separate), we do a direct
  // lookup and trust the caller to have populated the binding.
  if (params.explicitReceiver !== undefined) {
    return lookupReceiverType(startScope, params.explicitReceiver.name, ctx);
  }

  // Implicit `self` / `this` — the scope's typeBindings should carry it.
  for (const implicitName of IMPLICIT_RECEIVERS) {
    const owner = lookupReceiverType(startScope, implicitName, ctx);
    if (owner !== undefined) return owner;
  }
  return undefined;
}

const IMPLICIT_RECEIVERS: readonly string[] = Object.freeze(['self', 'this']);
const CALLABLE_RETURN_KINDS: ReadonlySet<NodeLabel> = new Set<NodeLabel>([
  'Function',
  'Method',
  'Constructor',
]);
const CALL_RETURN_STRICT_ORIGINS: ReadonlySet<BindingRef['origin']> = new Set<BindingRef['origin']>(
  ['local', 'import', 'namespace', 'reexport'],
);
const FIELD_DERIVED_KINDS: ReadonlySet<NodeLabel> = new Set<NodeLabel>([
  'Variable',
  'Property',
  'Const',
  'Static',
]);
const METHOD_DERIVED_KINDS: ReadonlySet<NodeLabel> = new Set<NodeLabel>([
  'Function',
  'Method',
  'Constructor',
]);

function lookupReceiverType(
  startScope: ScopeId,
  receiverName: string,
  ctx: RegistryContext,
): DefId | undefined {
  return lookupReceiverTypeInner(startScope, receiverName, ctx, new Set());
}

function lookupReceiverTypeInner(
  startScope: ScopeId,
  receiverName: string,
  ctx: RegistryContext,
  visitedReceivers: Set<string>,
): DefId | undefined {
  const receiverVisitKey = `${startScope}\0${receiverName}`;
  if (visitedReceivers.has(receiverVisitKey)) return undefined;
  visitedReceivers.add(receiverVisitKey);

  let currentId: ScopeId | null = startScope;
  const visited = new Set<ScopeId>();
  while (currentId !== null) {
    if (visited.has(currentId)) return undefined;
    visited.add(currentId);

    const scope = ctx.scopes.getScope(currentId);
    if (scope === undefined) return undefined;

    const typeRef = scope.typeBindings.get(receiverName);
    if (typeRef !== undefined) {
      const owner = resolveReceiverTypeRef(typeRef, receiverName, ctx, visitedReceivers);
      if (owner !== undefined) return owner;
      if (
        typeRef.source === 'field-access' ||
        typeRef.source === 'method-return' ||
        typeRef.source === 'call-return' ||
        typeRef.source === 'call-return-element'
      ) {
        currentId = scope.parent;
        continue;
      }

      // Compatibility fallback for older/manual registry fixtures that
      // provide typeBindings without the corresponding lexical bindings.
      const candidateIds = ctx.qualifiedNames.get(typeRef.rawName);
      if (candidateIds.length === 1) return candidateIds[0];
      // Ambiguous (≥ 2) or missing (0): cannot claim the receiver type.
      return undefined;
    }

    const bindings = scope.bindings.get(receiverName);
    if (bindings !== undefined && bindings.length > 0) {
      return resolveReceiverOwnerFromBindings(bindings, ctx, visitedReceivers);
    }
    currentId = scope.parent;
  }
  return undefined;
}

function resolveReceiverTypeRef(
  typeRef: TypeRef,
  receiverName: string,
  ctx: RegistryContext,
  visitedReceivers: Set<string>,
): DefId | undefined {
  if (typeRef.source === 'receiver-propagated') {
    const key = `${typeRef.declaredAtScope}\0${receiverName}\0${typeRef.rawName}`;
    if (visitedReceivers.has(key)) return undefined;
    visitedReceivers.add(key);
    return lookupReceiverTypeInner(typeRef.declaredAtScope, typeRef.rawName, ctx, visitedReceivers);
  }

  if (typeRef.source === 'field-access' || typeRef.source === 'method-return') {
    return resolveMemberDerivedOwner(typeRef, ctx, visitedReceivers);
  }

  if (typeRef.source === 'call-return' || typeRef.source === 'call-return-element') {
    return resolveCallReturnOwner(typeRef, ctx, typeRef.source);
  }

  const resolved = resolveTypeRef(typeRef, {
    scopes: ctx.scopes,
    defIndex: ctx.defs,
    qualifiedNameIndex: ctx.qualifiedNames,
  });
  return resolved?.nodeId;
}

function resolveReceiverOwnerFromBindings(
  bindings: readonly BindingRef[],
  ctx: RegistryContext,
  visitedReceivers: Set<string>,
): DefId | undefined {
  let selected: DefId | undefined;
  for (const binding of bindings) {
    if (!CALL_RETURN_STRICT_ORIGINS.has(binding.origin)) continue;

    const owner = resolveReceiverOwnerFromBoundDef(binding.def, ctx, visitedReceivers);
    if (owner === undefined) continue;
    if (selected !== undefined && selected !== owner) return undefined;
    selected = owner;
  }
  return selected;
}

function resolveReceiverOwnerFromBoundDef(
  def: SymbolDefinition,
  ctx: RegistryContext,
  visitedReceivers: Set<string>,
): DefId | undefined {
  const inferredType = ctx.typeBindingByDef?.get(def.nodeId);
  if (inferredType !== undefined) {
    return resolveReceiverTypeRef(
      inferredType,
      simpleNameOf(def) ?? inferredType.rawName,
      ctx,
      visitedReceivers,
    );
  }

  const rawName = def.declaredType ?? def.returnType;
  if (rawName === undefined) return undefined;

  const declaredAtScope = ctx.moduleScopes.get(def.filePath);
  if (declaredAtScope === undefined) return undefined;
  return resolveReceiverTypeRef(
    {
      rawName,
      declaredAtScope,
      source: def.declaredType !== undefined ? 'annotation' : 'return-annotation',
    },
    simpleNameOf(def) ?? rawName,
    ctx,
    visitedReceivers,
  );
}

function resolveCallReturnOwner(
  typeRef: TypeRef,
  ctx: RegistryContext,
  source: 'call-return' | 'call-return-element',
): DefId | undefined {
  const directOwner = resolveCallableReturnOwner(
    typeRef.rawName,
    typeRef.declaredAtScope,
    ctx,
    source,
  );
  if (directOwner !== undefined) return directOwner;

  if (source === 'call-return-element') {
    return resolveIterableElementBindingOwner(typeRef.rawName, typeRef.declaredAtScope, ctx);
  }

  return undefined;
}

function resolveCallableReturnOwner(
  name: string,
  startScope: ScopeId,
  ctx: RegistryContext,
  source: 'call-return' | 'call-return-element',
): DefId | undefined {
  const callable = resolveCallableBinding(name, startScope, ctx);
  if (callable?.returnType === undefined) return undefined;

  const returnTypeName =
    source === 'call-return-element'
      ? extractIterableElementReturnTypeName(callable.returnType)
      : extractReturnTypeName(callable.returnType);
  if (returnTypeName === undefined) return undefined;

  const returnScope = ctx.moduleScopes.get(callable.filePath) ?? startScope;
  const resolved = resolveTypeRef(
    {
      rawName: returnTypeName,
      declaredAtScope: returnScope,
      source: 'return-annotation',
    },
    {
      scopes: ctx.scopes,
      defIndex: ctx.defs,
      qualifiedNameIndex: ctx.qualifiedNames,
    },
  );
  return resolved?.nodeId;
}

function resolveIterableElementBindingOwner(
  receiverName: string,
  startScope: ScopeId,
  ctx: RegistryContext,
): DefId | undefined {
  return resolveIterableElementBindingOwnerInner(receiverName, startScope, ctx, new Set());
}

function resolveIterableElementBindingOwnerInner(
  receiverName: string,
  startScope: ScopeId,
  ctx: RegistryContext,
  visitedReceivers: Set<string>,
): DefId | undefined {
  const receiverVisitKey = `${startScope}\0${receiverName}`;
  if (visitedReceivers.has(receiverVisitKey)) return undefined;
  visitedReceivers.add(receiverVisitKey);

  let currentId: ScopeId | null = startScope;
  const visitedScopes = new Set<ScopeId>();
  while (currentId !== null) {
    if (visitedScopes.has(currentId)) return undefined;
    visitedScopes.add(currentId);

    const scope = ctx.scopes.getScope(currentId);
    if (scope === undefined) return undefined;

    const receiverType = scope.typeBindings.get(receiverName);
    if (receiverType !== undefined) {
      if (receiverType.source === 'receiver-propagated') {
        return resolveIterableElementBindingOwnerInner(
          receiverType.rawName,
          receiverType.declaredAtScope,
          ctx,
          visitedReceivers,
        );
      }

      if (receiverType.source === 'call-return' || receiverType.source === 'call-return-element') {
        return resolveCallableReturnOwner(
          receiverType.rawName,
          receiverType.declaredAtScope,
          ctx,
          'call-return-element',
        );
      }

      const elementTypeName = extractIterableElementReturnTypeName(receiverType.rawName);
      if (elementTypeName === undefined) return undefined;
      const resolved = resolveTypeRef(
        {
          rawName: elementTypeName,
          declaredAtScope: receiverType.declaredAtScope,
          source: 'return-annotation',
        },
        {
          scopes: ctx.scopes,
          defIndex: ctx.defs,
          qualifiedNameIndex: ctx.qualifiedNames,
        },
      );
      return resolved?.nodeId;
    }

    currentId = scope.parent;
  }

  return undefined;
}

function resolveMemberDerivedOwner(
  typeRef: TypeRef,
  ctx: RegistryContext,
  visitedReceivers: Set<string>,
): DefId | undefined {
  const member = splitMemberDerivedRawName(typeRef.rawName);
  if (member === undefined) return undefined;

  const receiverOwner = lookupReceiverTypeInner(
    typeRef.declaredAtScope,
    member.receiverName,
    ctx,
    visitedReceivers,
  );
  if (receiverOwner === undefined) return undefined;

  const acceptedKinds =
    typeRef.source === 'field-access' ? FIELD_DERIVED_KINDS : METHOD_DERIVED_KINDS;
  const valueType = uniqueMemberValueType(receiverOwner, member.memberName, acceptedKinds, ctx);
  if (valueType === undefined) return undefined;

  const memberScope = ctx.moduleScopes.get(valueType.filePath) ?? typeRef.declaredAtScope;
  const resolved = resolveTypeRef(
    {
      rawName: valueType.typeName,
      declaredAtScope: memberScope,
      source: typeRef.source === 'field-access' ? 'annotation' : 'return-annotation',
    },
    {
      scopes: ctx.scopes,
      defIndex: ctx.defs,
      qualifiedNameIndex: ctx.qualifiedNames,
    },
  );
  return resolved?.nodeId;
}

function splitMemberDerivedRawName(
  rawName: string,
): { receiverName: string; memberName: string } | undefined {
  const dot = rawName.lastIndexOf('.');
  if (dot <= 0 || dot >= rawName.length - 1) return undefined;
  return { receiverName: rawName.slice(0, dot), memberName: rawName.slice(dot + 1) };
}

function uniqueMemberValueType(
  ownerDefId: DefId,
  memberName: string,
  acceptedKinds: ReadonlySet<NodeLabel>,
  ctx: RegistryContext,
): { typeName: string; filePath: string } | undefined {
  for (const owner of memberOwnerWalk(ownerDefId, ctx)) {
    let selected: { typeName: string; filePath: string } | undefined;
    for (const def of collectOwnedMembers(owner, memberName, ctx)) {
      if (!acceptedKinds.has(def.type)) continue;
      const raw = def.returnType ?? def.declaredType;
      if (raw === undefined) continue;
      const typeName = extractReturnTypeName(raw);
      if (typeName === undefined) continue;
      if (selected !== undefined && selected.typeName !== typeName) return undefined;
      selected = { typeName, filePath: def.filePath };
    }
    if (selected !== undefined) return selected;
  }
  return undefined;
}

function memberOwnerWalk(ownerDefId: DefId, ctx: RegistryContext): readonly DefId[] {
  return ctx.methodDispatch === undefined
    ? [ownerDefId]
    : [ownerDefId, ...ctx.methodDispatch.mroFor(ownerDefId)];
}

function resolveCallableBinding(
  name: string,
  startScope: ScopeId,
  ctx: RegistryContext,
): SymbolDefinition | undefined {
  let currentId: ScopeId | null = startScope;
  const visited = new Set<ScopeId>();
  while (currentId !== null) {
    if (visited.has(currentId)) return undefined;
    visited.add(currentId);

    const scope = ctx.scopes.getScope(currentId);
    if (scope === undefined) return undefined;

    const bindings = scope.bindings.get(name);
    if (bindings !== undefined && bindings.length > 0) {
      for (const binding of bindings) {
        if (!CALL_RETURN_STRICT_ORIGINS.has(binding.origin)) continue;
        if (CALLABLE_RETURN_KINDS.has(binding.def.type)) return binding.def;
      }
      return undefined;
    }

    currentId = scope.parent;
  }
  return undefined;
}

function collectOwnedMembers(
  ownerDefId: DefId,
  memberName: string,
  ctx: RegistryContext,
): readonly SymbolDefinition[] {
  const indexedOwner = ctx.ownedMembersByOwner?.get(ownerDefId);
  if (indexedOwner !== undefined) return indexedOwner.get(memberName) ?? EMPTY_DEFS;

  // An owner's members are defs whose `ownerId === ownerDefId` and whose
  // simple name matches `memberName`. We iterate `defs.byId` — O(D) per
  // call for compatibility when callers have not supplied an owner-member
  // index.
  const out: SymbolDefinition[] = [];
  for (const def of ctx.defs.byId.values()) {
    if (def.ownerId !== ownerDefId) continue;
    if (simpleNameOf(def) !== memberName) continue;
    out.push(def);
  }
  return out;
}

function simpleNameOf(def: SymbolDefinition): string | undefined {
  if (def.qualifiedName === undefined || def.qualifiedName.length === 0) return undefined;
  const dot = def.qualifiedName.lastIndexOf('.');
  return dot === -1 ? def.qualifiedName : def.qualifiedName.slice(dot + 1);
}

function extractReturnTypeName(raw: string, depth = 0): string | undefined {
  if (depth > 10 || raw.length > 2048) return undefined;
  let text = raw.trim();
  if (text.length === 0) return undefined;

  text = text.replace(/^[&*]+\s*(mut\s+)?/, '').replace(/\?$/, '');

  if (text.includes('|')) {
    const parts = text
      .split('|')
      .map((part) => part.trim())
      .filter((part) => part.length > 0 && !NULLABLE_RETURN_KEYWORDS.has(part));
    if (parts.length !== 1) return undefined;
    text = parts[0]!;
  }

  const generic = text.match(/^(\w+)\s*<(.+)>$/);
  if (generic !== null) {
    const [, base, args] = generic;
    if (WRAPPER_RETURN_TYPES.has(base)) {
      return extractReturnTypeName(firstGenericTypeArg(args), depth + 1);
    }
    return PRIMITIVE_RETURN_TYPES.has(base.toLowerCase()) ? undefined : base;
  }

  if (WRAPPER_RETURN_TYPES.has(text)) return undefined;

  if (text.includes('::') || text.includes('.') || text.includes('\\')) {
    text = text.split(/::|[.\\]/).pop() ?? '';
  }

  if (PRIMITIVE_RETURN_TYPES.has(text) || PRIMITIVE_RETURN_TYPES.has(text.toLowerCase())) {
    return undefined;
  }
  if (!/^[A-Z_]\w*$/.test(text) || text.length > 512) return undefined;
  return text;
}

function extractIterableElementReturnTypeName(raw: string, depth = 0): string | undefined {
  if (depth > 10 || raw.length > 2048) return undefined;
  let text = raw.trim();
  if (text.length === 0) return undefined;

  text = text.replace(/^[&*]+\s*(mut\s+)?/, '').replace(/\?$/, '');

  if (text.includes('|')) {
    const parts = text
      .split('|')
      .map((part) => part.trim())
      .filter((part) => part.length > 0 && !NULLABLE_RETURN_KEYWORDS.has(part));
    if (parts.length !== 1) return undefined;
    text = parts[0]!;
  }

  if (text.endsWith('[]')) {
    return extractReturnTypeName(text.slice(0, -2), depth + 1);
  }

  const generic = text.match(/^(\w+)\s*<(.+)>$/);
  if (generic !== null) {
    const [, base, args] = generic;
    if (ITERABLE_RETURN_TYPES.has(base)) {
      return extractReturnTypeName(firstGenericTypeArg(args), depth + 1);
    }
  }

  return undefined;
}

function firstGenericTypeArg(args: string): string {
  let depth = 0;
  for (let index = 0; index < args.length; index++) {
    const char = args[index];
    if (char === '<') depth++;
    else if (char === '>') depth--;
    else if (char === ',' && depth === 0) return args.slice(0, index).trim();
  }
  return args.trim();
}

function recordTypeBindingHit(
  perCandidate: Map<DefId, CandidateState>,
  def: SymbolDefinition,
  mroDepth: number,
  receiverOwner: DefId,
): void {
  const state = ensureCandidate(perCandidate, def);
  const existingMroDepth = state.signals.typeBindingMroDepth;
  const firstHit = existingMroDepth === undefined;
  // Only replace if this hit is shallower (smaller MRO depth). The local
  // const lets TS narrow to `number` in the `else` branch so no `!`
  // assertion is needed.
  if (firstHit || mroDepth < existingMroDepth) {
    state.signals.typeBindingMroDepth = mroDepth;
    state.tieBreakKey.mroDepth = mroDepth;
  }
  if (def.ownerId === receiverOwner) {
    state.signals.ownerMatch = true;
  }
  // Pure type-binding candidates (no lexical hit) would otherwise keep the
  // `ensureCandidate` default `tieBreakKey.origin === 'local'`, making the
  // Appendix B cascade lump them with local-origin candidates. Demote them
  // to `'import'` — the strongest non-local origin — only when no earlier
  // phase set an origin for this candidate. Lexical hits from Step 1 set
  // `signals.origin` before Step 2 runs, so the guard skips them; Step 3
  // (`seedFromOwnerScopedContributor`) runs AFTER Step 2 and unconditionally
  // overrides `tieBreakKey.origin` back to `'local'` for direct-owner
  // members, so any same-def overlap still ends up ranked correctly.
  if (firstHit && state.signals.origin === undefined) {
    state.tieBreakKey.origin = 'import';
  }
}

// ─── Step 3 implementation ─────────────────────────────────────────────────

function seedFromOwnerScopedContributor(
  name: string,
  contributor: OwnerScopedContributor,
  acceptedKinds: ReadonlySet<NodeLabel>,
  perCandidate: Map<DefId, CandidateState>,
): void {
  for (const def of contributor.byName(name)) {
    if (!acceptedKinds.has(def.type)) continue;
    const state = ensureCandidate(perCandidate, def);
    // Treat the contributor's direct membership as `origin: 'local'` —
    // strongest visibility, no scope-chain penalty.
    state.signals.origin = 'local';
    state.signals.scopeChainDepth = 0;
    state.signals.ownerMatch = def.ownerId === contributor.ownerDefId;
    state.tieBreakKey.origin = 'local';
  }
}

// ─── Step 5 implementation ─────────────────────────────────────────────────

function applyArityFilter(
  callsite: Callsite,
  perCandidate: Map<DefId, CandidateState>,
  ctx: RegistryContext,
): void {
  const arityFn = ctx.providers.arityCompatibility;
  if (arityFn === undefined) {
    // No provider → record 'unknown' for every candidate; keeps signal
    // shape uniform for composeEvidence.
    for (const state of perCandidate.values()) {
      state.signals.arityVerdict = 'unknown';
    }
    return;
  }

  let anyCompatible = false;
  for (const state of perCandidate.values()) {
    const verdict = arityFn(callsite, state.def);
    state.signals.arityVerdict = verdict;
    if (verdict === 'compatible') anyCompatible = true;
  }

  if (!anyCompatible) return;

  // Filter: when at least one compatible candidate exists, drop incompatibles.
  for (const [defId, state] of perCandidate) {
    if (state.signals.arityVerdict === 'incompatible') {
      perCandidate.delete(defId);
    }
  }
}

// ─── Step 7 implementation ─────────────────────────────────────────────────

function rankCandidates(perCandidate: Map<DefId, CandidateState>): readonly Resolution[] {
  const resolutions: Resolution[] = [];
  const tieKeys = new Map<string, TieBreakKey>();

  for (const state of perCandidate.values()) {
    const evidence = composeEvidence(state.signals as RawSignals);
    const confidence = confidenceFromEvidence(evidence);
    resolutions.push({ def: state.def, confidence, evidence });
    tieKeys.set(state.def.nodeId, { ...state.tieBreakKey });
  }

  resolutions.sort((a, b) => compareByConfidenceWithTiebreaks(a, b, tieKeys));
  return Object.freeze(resolutions);
}

// ─── Constants ──────────────────────────────────────────────────────────────

const EMPTY: readonly Resolution[] = Object.freeze([]);
const EMPTY_DEFS: readonly SymbolDefinition[] = Object.freeze([]);

const NULLABLE_RETURN_KEYWORDS: ReadonlySet<string> = new Set([
  'null',
  'undefined',
  'void',
  'None',
  'nil',
]);

const PRIMITIVE_RETURN_TYPES: ReadonlySet<string> = new Set([
  'string',
  'number',
  'boolean',
  'void',
  'int',
  'float',
  'double',
  'long',
  'short',
  'byte',
  'char',
  'bool',
  'str',
  'i8',
  'i16',
  'i32',
  'i64',
  'u8',
  'u16',
  'u32',
  'u64',
  'f32',
  'f64',
  'usize',
  'isize',
  'undefined',
  'null',
  'none',
  'nil',
]);

const WRAPPER_RETURN_TYPES: ReadonlySet<string> = new Set([
  'Promise',
  'Observable',
  'Future',
  'CompletableFuture',
  'Task',
  'ValueTask',
  'Option',
  'Some',
  'Optional',
  'Maybe',
  'Result',
  'Either',
]);

const ITERABLE_RETURN_TYPES: ReadonlySet<string> = new Set([
  'Array',
  'ReadonlyArray',
  'Iterable',
  'AsyncIterable',
  'Collection',
  'List',
  'Set',
  'Sequence',
]);
