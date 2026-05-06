# Accurate Single-Pass Graph Plan

## Position

AVmatrix should not choose between speed and graph accuracy.

If AVmatrix is faster because it removed or reduced GitNexus-style scope resolution work, that is workload reduction, not a complete optimization. The target is an accurate graph in one analyze run, without a separate deep mode and without a second source/AST pass.

## Goal

One `analyze` run should perform:

```text
scan
parse
extract scope facts from the same AST
finalize imports/scopes
build heritage, MRO, and method-dispatch indexes
resolve references/calls/accesses/uses/inheritance
communities
processes
load DB
```

The graph should keep or exceed the accuracy of GitNexus scope resolution while avoiding duplicate reads, duplicate parses, and duplicate resolution work.

GitNexus is only the accuracy baseline, not the performance target. AVmatrix should use GitNexus deep graph behavior as the minimum correctness floor, then beat its wall time by changing the architecture.

Success means:

- accuracy is equal or better than GitNexus deep/scope graph on the measured edge categories;
- wall time is materially lower than GitNexus for the same repository and equivalent graph accuracy;
- target speedup is at least 2x on large repositories, with a minimum acceptable first milestone of 40% lower wall time if parity work exposes unavoidable correctness cost;
- query/context behavior after load remains fast because audit metadata and indexes are available in the default graph.

## Non-Goals

- Do not re-add GitNexus `scopeResolutionPhase` as a second full pass.
- Do not introduce `fast` first and `deep` later as the primary path.
- Do not pass native AST `Tree` objects between workers and the main thread.
- Do not rely on find-and-replace refactors for symbol movement.

## Problem

GitNexus has valuable scope resolution logic after cross-file analysis. That work improves graph accuracy by resolving imports, calls, accesses, inheritance, and uses with language-aware scope rules.

The cost comes from the execution model:

- files are filtered and processed by language;
- file contents can be read again;
- AST cache may only exist in the main process;
- worker-mode parsing can leave no reusable AST cache for scope resolution;
- scope resolution may parse again;
- cross-file and scope-resolution responsibilities overlap.

AVmatrix currently avoids much of that cost by not running the same scope-resolution phase in the main phase list. That makes the pipeline faster, but it can also reduce deep graph accuracy. The correct fix is to bring the accuracy back through a different architecture.

## Current AVmatrix Constraints

AVmatrix already contains the beginning of the correct architecture:

- parse workers can emit `ParsedFile[]`;
- shared scope-resolution types already define `ParsedFile`, `ReferenceSite`, `Reference`, `ReferenceIndex`, and `ResolutionEvidence`;
- `finalizeScopeModel(parsedFiles)` already materializes scope indexes;
- `emitReferencesToGraph(...)` already drains a `ReferenceIndex` into graph edges with confidence and evidence;
- `GraphRelationship.evidence` exists in memory, but LadybugDB CSV/schema currently persist only `type`, `confidence`, `reason`, and `step`.
- `finalizeScopeModel` currently builds an empty `methodDispatch` index until heritage/MRO integration is wired.
- `emitScopeCaptures` currently accepts only `sourceText` and `filePath`. If a provider reparses inside this hook, it violates the single-AST goal.
- the existing graph-level `mroPhase` currently depends on `crossFile`; it cannot serve as the pre-resolution method-dispatch dependency without being split or reordered.

Therefore this plan must not introduce a parallel `ScopeIR` schema unless the existing shared model is proven insufficient. The default approach is to reuse and extend the existing shared scope-resolution contracts.

## Target Architecture

### 1. Parse Once, Emit Existing Scope Facts

Parse workers should return serialized scope artifacts in addition to the existing legacy extracted facts.

```ts
type AccurateParseArtifact = ParsedFile;
```

`ParsedFile` already carries:

- lexical scopes;
- local definitions;
- parsed imports;
- pre-resolution `ReferenceSite[]`.

If extra fields are needed, add them to the shared scope-resolution model deliberately. Do not create local duplicate fact types that drift from `avmatrix-shared`.

The worker already has source text and AST context. It should extract all resolution facts before returning. The main process should not read the same file again for scope resolution.

Scope extraction must reuse the AST already produced in the parse worker. A provider path that reparses source text to emit scope captures is not acceptable for the optimized default path.

### 2. Pass Facts, Not AST

Do not attempt to move native tree-sitter AST objects across worker boundaries. Workers should return deterministic JSON facts, primarily `ParsedFile[]` and existing extracted arrays.

Benefits:

- no native object transfer issues;
- no worker/main AST cache mismatch;
- easier snapshot testing;
- stable incremental hashing;
- smaller and more explicit resolution inputs.

### 3. Wire The Existing Scope Model Into The Pipeline

The current worker path can collect `parsedFiles`, but the main parse phase must preserve them through `ParseOutput` and downstream phases.

The intended flow is:

```text
parse workers
  -> ParsedFile[]
  -> finalizeScopeModel(parsedFiles)
  -> SemanticModel.attachScopeIndexes(...)
  -> build pre-resolution heritage/MRO/method-dispatch indexes
  -> resolutionPhase builds ReferenceIndex
  -> emitReferencesToGraph(...)
```

This keeps the accurate graph path in the default analyze run and avoids source rereads.

The pre-resolution method-dispatch index is a lookup structure for resolving calls. It is separate from graph-level MRO edge emission. The current `mroPhase` may be split or replaced, but call resolution must not wait for a post-`crossFile` phase.

### 4. Replace CrossFile + ScopeResolution With ResolutionPhase

Introduce one `resolutionPhase` that owns cross-file symbol resolution and scope-aware edge emission.

Input:

- parsed symbols;
- `ParsedFile[]`;
- finalized scope indexes;
- route/tool/ORM facts if available;
- heritage/MRO facts needed for owner-scoped dispatch;
- language provider resolution hooks.

Work:

- build global symbol table;
- consume finalized import/export graph;
- build per-file lexical indexes;
- build inheritance and method-dispatch indexes before resolving calls;
- resolve references;
- resolve calls;
- resolve member accesses;
- resolve inheritance;
- emit import-use and finalized import graph edges where needed;
- emit graph edges once.

This phase should replace overlapping responsibilities currently split between `crossFilePhase` and a GitNexus-style `scopeResolutionPhase`.

Important: `crossFilePhase` currently performs useful type propagation by re-reading selected files and re-running call processing. Do not delete that behavior early. Move the useful propagation logic into `resolutionPhase`, then retire or narrow `crossFilePhase` only after parity is proven.

### 5. Parallelize Resolution

Resolution should not run sequentially by language.

Proposed model:

```text
Phase A: parse workers produce ParsedFile[]
Phase B: main builds immutable global indexes
Phase C: resolution workers resolve chunks of ParsedFile/reference sites against readonly indexes
Phase D: main merges edges and diagnostics
```

Chunking should follow the AVmatrix dynamic worker model: small work units, byte/file limits, retry support, and ordered merge after completion.

Readonly indexes should be initialized once per resolution worker, not serialized with every chunk. Track serialized index size and worker init time as explicit metrics.

### 6. Keep Accuracy Auditable

Edges should include source, confidence, and evidence metadata.

```json
{
  "type": "CALLS",
  "resolutionSource": "scope-resolution",
  "confidence": 0.95,
  "evidence": [
    { "kind": "type-binding", "weight": 0.35, "note": "receiver User" },
    { "kind": "import", "weight": 0.25, "note": "imported from models/user.ts" }
  ],
  "fileHash": "..."
}
```

This preserves auditability without making accurate graph generation optional.

Because the in-memory graph and LadybugDB persistence are not currently equivalent for evidence metadata, this work must persist audit metadata through DB load.

In-memory-only evidence is not sufficient for the optimized accurate graph because query/context tools operate after graph load.

## Implementation Plan

### Progress Checklist

Last updated: 2026-05-06.

Use this checklist to update implementation progress. Do not mark the target architecture complete until the default analyze path emits accurate scope-aware edges without duplicate parse/read/resolution work.

- [x] Define the target clearly: GitNexus is the accuracy baseline, not the performance target.
- [x] Confirm the plan reuses existing shared contracts (`ParsedFile`, `ReferenceSite`, `Reference`, `ReferenceIndex`, `ResolutionEvidence`) instead of adding a duplicate local `ScopeIR`.
- [x] Add an AST-aware provider contract path for scope capture extraction.
- [x] Keep source-text scope capture as a compatibility path, not the optimized default.
- [x] Preserve worker-produced `ParsedFile[]` through the parse worker and parsing processor.
- [x] Thread `ParsedFile[]` into `ParseOutput`.
- [x] Add parse/scope counters for parsed files, scopes, local defs, imports, reference sites, AST reuse, compatibility fallback, no-hook files, and failed files.
- [x] Call `finalizeScopeModel(parsedFiles, hooks)` in the default parse path.
- [x] Attach finalized scope indexes to the semantic model once per analyze run.
- [x] Add unit coverage for AST-aware scope extraction bridge behavior.
- [x] Add a first TypeScript/JavaScript AST-aware scope-capture provider slice for core imports, declarations, type bindings, inheritance reference sites, and call reference sites.
- [x] Add unit coverage proving TypeScript scope facts can be produced from an already-parsed tree-sitter AST.
- [x] Preserve owner metadata for provider-emitted method/property declarations so later method-dispatch resolution has an owner anchor.
- [x] Add a pure `ReferenceSite` to `ReferenceIndex` resolver over finalized scope indexes, with TypeScript unit coverage for member calls, constructor calls, and inheritance.
- [x] Wire a non-duplicating `resolutionPhase` into the default pipeline to populate `ReferenceIndex` metrics before graph edge emission is enabled.
- [x] Emit currently resolved scope references from `resolutionPhase` through `emitReferencesToGraph` with a semantic duplicate-edge guard.
- [x] Surface resolution timings and counters in the top-level analyze performance report and CLI summary.
- [x] Make finalized import bindings visible to scope lookup so imported constructor/type references resolve without source rereads.
- [x] Add TypeScript/JavaScript AST-reused member read/write access facts and resolve them into `ACCESSES` edges.
- [x] Add TypeScript/JavaScript AST-reused type-reference facts from annotations and emit them as `USES` edges.
- [ ] Capture AVmatrix baseline metrics on the selected representative repositories.
- [ ] Capture GitNexus deep/scope graph baseline metrics on the same repositories.
- [ ] Define fixture-level parity expectations for `CALLS`, `IMPORTS`, `ACCESSES`, `USES`, and `INHERITS`.
- [ ] Migrate the first provider, preferably TypeScript, to emit complete AST-reused scope captures.
- [ ] Prove the migrated provider does not re-read or reparse source for scope extraction.
- [x] Build an initial `MethodDispatchIndex` from finalized `inherits` reference sites before scope-aware call resolution depends on it.
- [ ] Expand `MethodDispatchIndex` construction to full per-language Heritage/MRO strategy inputs before claiming parity for inherited dispatch.
- [x] Implement `resolutionPhase`.
- [x] Populate `ReferenceIndex` from finalized scope indexes and reference sites.
- [x] Resolve imported constructor references through finalized import bindings in `ScopeTree`.
- [x] Resolve TypeScript member read/write access facts to property definitions and emit graph `ACCESSES` edges.
- [x] Resolve TypeScript type annotation facts to class/interface definitions and emit graph `USES` edges.
- [ ] Complete scope-resolved `CALLS`, `ACCESSES`, `USES`, `INHERITS`, and import-use edge coverage across migrated providers through one graph emission path.
- [ ] Parallelize reference resolution by file/chunk against readonly indexes.
- [ ] Persist audit metadata (`resolutionSource`, `confidence`, `evidence`, `fileHash`) through LadybugDB load.
- [ ] Move useful `crossFilePhase` type propagation into `resolutionPhase`.
- [ ] Retire or narrow `crossFilePhase` only after parity is proven.
- [x] Add duplicate-edge checks so legacy and scope-aware paths do not emit overlapping edges.
- [ ] Validate full test suite and targeted parity fixtures.
- [ ] Benchmark equivalent-accuracy runs and confirm the speed target is met.

### Milestone 1: Baseline And Parity Targets

- Run AVmatrix analyze metrics on representative repos.
- Run GitNexus deep/scope graph baseline on the same repos where possible. Treat it as the minimum accuracy baseline, not as an acceptable speed target.
- Record edge counts by type:
  - `CALLS`
  - `IMPORTS`
  - `ACCESSES`
  - `USES`
  - `INHERITS`
- Record unresolved reference counts.
- Sample-check precision for resolved calls/accesses/inheritance.
- Record precision and recall against fixture expectations where available.
- Treat edge counts as a secondary signal, not a success condition by themselves.

### Milestone 2: Align Existing Scope Contracts

- Reuse `ParsedFile`, `ReferenceSite`, `Reference`, `ReferenceIndex`, and `ResolutionEvidence`.
- Avoid introducing `ScopeIR` unless an explicit gap is documented.
- If the shared model needs more fields, add them to `avmatrix-shared` as the single source of truth.
- Keep the schema language-neutral.
- Put language-specific extraction behind provider hooks such as `emitScopeCaptures`, `interpretImport`, `classifyCallForm`, and `arityCompatibility`.
- Extend or replace `emitScopeCaptures` with an AST-aware worker path, for example `emitScopeCapturesFromTree(...)`, so providers do not reparse source text.
- Treat source-text-only capture emission as a compatibility path, not the optimized default.
- Make the AST-aware hook a hard implementation gate before TypeScript or any other provider is migrated to the optimized path.
- Add JSON snapshot tests for small fixtures.

### Milestone 3: Thread ParsedFile Through The Pipeline

- Make `runChunkedParseAndResolve` preserve `chunkWorkerData.parsedFiles`.
- Add `parsedFiles` to `ParseOutput`.
- Ensure `parsing-processor` output is not dropped at the phase boundary.
- Add metrics for parsed file count, scope count, local def count, import fact count, and reference site count.
- Add a metric that proves scope extraction reused the worker AST and did not trigger a second parse.

### Milestone 4: Complete Scope Extraction Coverage

- Migrate providers to implement `emitScopeCaptures` and related hooks.
- Start with one end-to-end language before broad migration. TypeScript is the preferred first target because it exercises imports, classes, methods, fields, calls, and arity without optional native parser risk.
- Add TypeScript fixture parity before enabling another language.
- Follow with Python, then JVM, Go, Rust, C#, PHP, Ruby, Swift/Kotlin where parser availability allows.
- Extract facts while source text and AST are already available.
- Avoid any main-thread source reread for these facts.
- Add per-language coverage counters:
  - parseable files;
  - files with `ParsedFile`;
  - scopes emitted;
  - definitions emitted;
  - reference sites emitted;
  - unresolved reference rate.
- Define language coverage gates. AVmatrix can claim optimized accurate graph only for languages whose scope extraction and resolution coverage meet the parity threshold; mixed-language repo claims must report covered vs legacy language shares.

### Milestone 5: Finalize Scope Indexes

- Build the import-target workspace from existing language import resolvers.
- Call `finalizeScopeModel(parsedFiles, hooks)`.
- Attach scope indexes to the semantic model once per analyze run.
- Validate linked vs unresolved import counts.
- Preserve SCC information for resolution scheduling.
- Replace the temporary empty `methodDispatch` construction with an index built from heritage/MRO inputs before call resolution depends on it.
- Split pre-resolution method-dispatch index construction from the current graph-level `mroPhase`, or replace `mroPhase` with phases whose dependencies match this plan.
- Treat finalized imports as the import-resolution source of truth. Downstream resolution may emit import-use or graph `IMPORTS` edges, but must not resolve the same import targets again.

### Milestone 6: Implement Parallel ResolutionPhase

- Create `resolutionPhase`.
- Build a `ReferenceIndex` from finalized scope indexes and reference sites.
- Implement `resolveReferenceSites(...)`:
  - map `ReferenceSite.kind` to class, method, field, or generic definition registries;
  - use `callForm`, `explicitReceiver`, receiver/type bindings, and arity when resolving calls;
  - use method dispatch/MRO indexes for owner-scoped method calls;
  - select the top candidate by confidence and tie-break rules;
  - emit unresolved diagnostics when no candidate is safe enough.
- Resolve file/chunk reference-site units in workers.
- Initialize readonly resolution indexes once per worker.
- Emit import, call, access, use, and inheritance edges in one place.
- Merge results deterministically.
- Add timing metrics:
  - index build ms;
  - worker index init ms;
  - resolution worker ms;
  - merge ms;
  - serialized index bytes;
  - emitted edge counts;
  - unresolved counts.

### Milestone 7: Persist Audit Metadata

- Decide the durable relationship metadata shape for evidence/resolutionSource/fileHash.
- Extend `GraphRelationship` if `resolutionSource` and `fileHash` become first-class relationship properties instead of encoded `reason` text.
- Extend LadybugDB relationship schema.
- Extend relationship CSV generation columns.
- Extend relationship CSV split/load paths.
- Extend fallback relationship insert parsing.
- Extend query/context readers so audit metadata is visible through tools after DB load.
- Preserve backward compatibility for existing relationship queries.
- Add tests that prove evidence metadata survives graph load.

### Milestone 8: Retire Overlap

- Move useful type propagation from `crossFilePhase` into `resolutionPhase`.
- Remove or narrow `crossFilePhase` responsibilities only after `resolutionPhase` reaches parity.
- Do not keep duplicate edge emission paths.
- Add duplicate-edge checks with source/confidence metadata.
- Remove hard caps that trade accuracy for speed, or make them explicit safety limits with diagnostics.
- Do not claim the target architecture is achieved while the default accurate pipeline still runs both `crossFilePhase` re-resolution and `resolutionPhase` for the same call/access/inheritance responsibilities.

### Milestone 9: Validate Performance And Accuracy

Validation commands:

```bash
cd avmatrix && npm test
cd avmatrix && npx tsc --noEmit
```

Benchmark protocol:

- run AVmatrix and GitNexus on the same machine, same repository checkout, same exclude rules, and same parser availability;
- record tool versions and commit hashes;
- record cold-cache and warm-cache runs separately;
- run at least three iterations and compare median wall time;
- include a small, medium, and large repository set, with "large" defined before measurement by file count and parseable MB;
- compare only equivalent accuracy configurations;
- publish language coverage for every benchmarked repo.

Benchmark dimensions:

- total wall time;
- parse ms;
- scope fact extraction ms;
- resolution ms;
- lbug load ms;
- graph stream ms;
- memory peak;
- worker utilization;
- unresolved references;
- edge precision/recall against fixtures;
- accuracy delta against GitNexus baseline;
- speedup over GitNexus at equivalent or better accuracy;
- second-parse count for scope extraction;
- evidence persistence coverage.

## Acceptance Criteria

- One default analyze run produces accurate scope-aware graph edges.
- No second source read for scope resolution.
- No second AST parse for scope resolution.
- `ParsedFile[]` is preserved from worker output through parse output.
- `finalizeScopeModel` is called in the default accurate pipeline.
- `ReferenceIndex` is populated and emitted to graph edges.
- `crossFilePhase` and scope resolution responsibilities are unified or clearly non-overlapping.
- Resolution is parallelized for large repos.
- The default accurate pipeline does not run duplicate cross-file re-resolution and new scope resolution for the same edge responsibilities.
- Edge precision/recall meets fixture expectations.
- Accuracy is equal or better than GitNexus baseline without relying only on higher edge counts.
- Wall time is at least 2x faster than GitNexus on large repositories at equivalent or better graph accuracy, or at least 40% lower for the first parity milestone with follow-up work explicitly identified.
- Analyze metrics expose parse, fact extraction, resolution, and DB load costs separately.
- Scope extraction reuses the worker AST and does not parse source a second time.
- Audit metadata is queryable after DB load.
- Benchmark results follow the documented protocol and report language coverage.

## Risks

- Scope contracts may grow too broad if language-specific details leak into shared code.
- Serialized facts may become large on generated or minified files.
- Parallel resolution needs deterministic merge behavior to avoid unstable graph diffs.
- Confidence metadata must not become a substitute for fixing low-quality resolution.
- Persisting evidence metadata may increase relationship storage size.
- Partial provider migration can create mixed legacy/new graph behavior unless metrics make coverage obvious.
- Retiring `crossFilePhase` too early can lose useful type propagation.
- Reusing the current post-`crossFile` `mroPhase` as a pre-resolution dependency would create an invalid phase dependency cycle or leave method dispatch incomplete.
- Benchmarks without equivalent accuracy settings can produce misleading speedup claims.

## First Code Areas To Inspect

- `avmatrix/src/core/ingestion/pipeline.ts`
- `avmatrix/src/core/ingestion/pipeline-phases/parse-impl.ts`
- `avmatrix/src/core/ingestion/pipeline-phases/cross-file-impl.ts`
- `avmatrix/src/core/ingestion/workers/`
- `avmatrix/src/core/ingestion/finalize-orchestrator.ts`
- `avmatrix/src/core/ingestion/emit-references.ts`
- `avmatrix/src/core/ingestion/model/scope-resolution-indexes.ts`
- `avmatrix/src/core/ingestion/call-types.ts`
- `avmatrix-shared/src/scope-resolution/`
- `avmatrix/src/core/lbug/`
- GitNexus scope resolver implementation for logic porting only, not phase structure.
