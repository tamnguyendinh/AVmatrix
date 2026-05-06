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

Every implementation slice must be measurable. A change is not considered an optimization unless an AVmatrix analyze benchmark artifact records the relevant before/after timings, counters, edge counts, unresolved counts, and correctness snapshot. If the slice changes resolution behavior, benchmark output must include resolution timings and edge parity counters.

Benchmarking in this plan means measuring AVmatrix before and after each implementation slice. Do not spend plan time running GitNexus locally unless the user explicitly asks for that run. GitNexus numbers can be supplied externally and are used as the accuracy/performance baseline, not as the benchmark command for each AVmatrix slice.

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
- available audit metadata (`resolutionSource`, `evidence`, `fileHash`) is now persisted through LadybugDB and surfaced in graph/context/impact read paths; remaining work is coverage, not storage plumbing.
- `finalizeScopeModel` now builds an initial `methodDispatch` index from finalized inheritance reference sites; full per-language Heritage/MRO inputs remain to be integrated before parity claims.
- providers can emit scope captures through an AST-aware hook; source-text capture remains a compatibility path and any provider relying on it is not on the optimized default path.
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
- [x] Add analyze benchmark JSON output that combines graph correctness snapshot, performance timings, and key optimization counters.
- [x] Expose edge-count, scope-finalize, chunk/index, and resolution-kind counters directly in benchmark JSON key metrics.
- [x] Record AVmatrix version, Node runtime, platform, and target repo git commit/dirty state in benchmark JSON.
- [x] Add benchmark snapshot comparison helper for before/after timing, edge-count, unresolved-count, and graph-diff deltas.
- [x] Expose benchmark before/after comparison through `avmatrix benchmark-compare <before.json> <after.json>` so optimization claims have a usable CLI check.
- [x] Emit finalized file-level `IMPORTS` edges through default resolution emission with duplicate guard, without enabling full scope graph.
- [x] Emit finalized per-symbol import-use `USES` edges through the default graph emission path with audit evidence.
- [x] Make finalized import bindings visible to scope lookup so imported constructor/type references resolve without source rereads.
- [x] Add TypeScript/JavaScript AST-reused member read/write access facts and resolve them into `ACCESSES` edges.
- [x] Add TypeScript/JavaScript AST-reused type-reference facts from annotations and emit them as `USES` edges.
- [x] Capture AVmatrix baseline metrics on the selected representative repositories.
- [x] Define fixture-level parity expectations for `CALLS`, `IMPORTS`, `ACCESSES`, `USES`, and `INHERITS`.
- [x] Require a benchmark JSON artifact before and after each optimization slice that claims speedup.
- [ ] Migrate the first provider, preferably TypeScript, to emit complete AST-reused scope captures.
- [x] Prove the migrated provider does not re-read or reparse source for scope extraction.
- [x] Build an initial `MethodDispatchIndex` from finalized `inherits` reference sites before scope-aware call resolution depends on it.
- [ ] Expand `MethodDispatchIndex` construction to full per-language Heritage/MRO strategy inputs before claiming parity for inherited dispatch.
- [x] Implement `resolutionPhase`.
- [x] Populate `ReferenceIndex` from finalized scope indexes and reference sites.
- [x] Resolve imported constructor references through finalized import bindings in `ScopeTree`.
- [x] Resolve TypeScript member read/write access facts to property definitions and emit graph `ACCESSES` edges.
- [x] Resolve TypeScript type annotation facts to class/interface definitions and emit graph `USES` edges.
- [x] Add TypeScript/JavaScript AST-reused return type reference facts and resolve them into `USES` edges.
- [x] Infer TypeScript/JavaScript local variable type bindings from same-file function return annotations without source rereads.
- [x] Preserve TypeScript/JavaScript declaration `returnType` and `declaredType` metadata in `ParsedFile` facts from the reused AST.
- [x] Resolve receiver method dispatch through imported type aliases, for example `current: U` followed by `current.save()`, without reparsing.
- [x] Add a TypeScript accurate single-pass parity fixture covering AST reuse, finalized imports, resolved reference counts, emitted edge counts, unresolved count, and audit metadata.
- [ ] Complete scope-resolved `CALLS`, `ACCESSES`, `USES`, `INHERITS`, and import-use edge coverage across migrated providers through one graph emission path.
- [x] Add deterministic reference-site chunk scheduling plus chunk/index cardinality metrics as the scaffold for workerized resolution.
- [x] Add readonly resolution index size plus init/worker/merge timing metrics as workerization scaffolding.
- [x] Refactor scope reference resolution into deterministic chunk functions with serializable readonly-index payloads.
- [x] Add worker-pool `workerData` support so future resolution workers can receive readonly indexes once at startup.
- [x] Add an opt-in reference-resolution worker prototype plus parity coverage for chunk output.
- [ ] Parallelize reference resolution by file/chunk against readonly indexes.
- [x] Persist available audit metadata (`resolutionSource`, `confidence`, `evidence`, `fileHash` column) through LadybugDB CSV/load/read-back.
- [x] Populate real `fileHash` values on scope-resolved edges from parse-time `ParsedFile.fileHash` without rereading source.
- [x] Expose scope-resolution audit metadata in MCP context/impact readers, not only graph read-back.
- [x] Attach audit metadata to finalized scope `IMPORTS` edges so import parity is inspectable after graph emission.
- [x] Add legacy relationship-schema fallback so existing indexes without `resolutionSource`, `evidence`, or `fileHash` remain queryable.
- [x] Move the first imported function return-type receiver propagation slice into AST-reused `call-return` scope facts and scope lookup without source rereads.
- [x] Extend TypeScript/JavaScript AST-reused `call-return` facts to awaited imported calls, for example `const user = await makeUser(); user.save()`, without source rereads.
- [ ] Move useful `crossFilePhase` type propagation into `resolutionPhase`.
- [ ] Retire or narrow `crossFilePhase` only after parity is proven.
- [x] Add duplicate-edge checks so legacy and scope-aware paths do not emit overlapping edges.
- [x] Map scope-resolved references to real graph node ids and fail closed when either endpoint is missing.
- [x] Preserve owner-qualified TypeScript/JavaScript member declaration names so same-file same-name members resolve distinctly.
- [x] Merge scope audit metadata into existing semantic duplicate edges instead of discarding the scope-resolved evidence.
- [x] Add regression coverage for same-file same-name member mapping, duplicate audit metadata merge, and DB persistence/readback.
- [x] Keep native DB and CLI analyze E2E tests in a sequential Vitest project to reduce Windows full-suite instability.
- [x] Validate targeted parity fixtures and audit persistence tests cleanly.
- [x] Make aggregate full `cd avmatrix && npm test` pass without Vitest worker-fork unhandled errors on Windows.
- [x] Define full UI validation build as `avmatrix-launcher\build.ps1`, not CLI-only build.
- [x] Add TypeScript/JavaScript AST-reused interface property signatures and type-alias RHS type-reference facts.
- [x] Add a precomputed owner-member index for receiver method/field dispatch so expanded scope facts do not force O(total defs) member scans per lookup.
- [ ] Benchmark equivalent-accuracy runs and confirm the speed target is met.

Current benchmark artifact:

- `reports/benchmark/2026-05-06-avmatrix-current-gitnexus-main.json`
- `reports/benchmark/2026-05-06-avmatrix-parallel-resolution-gitnexus-main.json`
- `reports/benchmark/2026-05-06-avmatrix-safe-default-gitnexus-main.json`
- `reports/benchmark/2026-05-06-avmatrix-call-return-gitnexus-main.json`
- `reports/benchmark/2026-05-06-avmatrix-awaited-call-return-gitnexus-main.json`
- `reports/benchmark/2026-05-06-avmatrix-awaited-call-return-gitnexus-main-run2.json`
- `reports/benchmark/2026-05-06-avmatrix-awaited-call-return-gitnexus-main-run3.json`
- Target: `E:\Lap_trinh\GitNexus-main` using `--skip-git` because that local copy has no `.git` directory.
- Runtime command used built AVmatrix CLI with `--skip-agents-md --no-stats --benchmark-json` and `node --stack-size=4096`; the default stack failed in parse with `Maximum call stack size exceeded`, so the stack-size requirement is part of the recorded run context.
- Result: `110714.1ms` total wall time, `847` files, `19538` nodes, `31037` persisted relationships.
- Phase timings: parse `44866ms`, crossFile `19563ms`, resolution `4043ms`, lbugLoad `23891.3ms`, FTS `15676.9ms`.
- Edge counts: `CALLS=5550`, `IMPORTS=202`, `ACCESSES=184`, `USES=711`, `INHERITS=1`.
- Scope counters: `143226` reference sites, `10632` resolved, `132594` unresolved, `1348` emitted, `5180` duplicate edges merged/skipped, `2989` skipped no caller, `1115` skipped missing target.
- Parallel-resolution experiment: enabling worker resolution on this repo changed `resolution` from `4043ms` to `4838ms` and produced a small graph diff (`ACCESSES -1`). That is not an accepted optimization. The worker path is kept opt-in via `AVMATRIX_SCOPE_RESOLUTION_WORKERS=1` until index transfer/build overhead is reduced and graph parity is proven.
- Safe default after the experiment: `reports/benchmark/2026-05-06-avmatrix-safe-default-gitnexus-main.json` keeps worker resolution disabled by default and measured `resolution=3956ms`; graph counts still varied slightly from the first run, so benchmark claims must use repeated median runs and parity checks rather than a single artifact.
- `reports/benchmark/2026-05-06-avmatrix-ts-interface-typealias-gitnexus-main.json` records the AVmatrix slice that added TypeScript interface property and type-alias RHS scope facts. It increased resolved references but also increased wall time; it is a correctness expansion, not a speedup claim.
- `reports/benchmark/2026-05-06-avmatrix-owner-member-index-gitnexus-main.json` records the AVmatrix owner-member index optimization after that correctness expansion.
- AVmatrix benchmark comparison, `ts-interface-typealias-scope-coverage` -> `owner-member-index-scope-resolution`: wall `126700.1ms` -> `105688.7ms` (`-16.6%`), resolution `6905ms` -> `895ms` (`-87%`). This is an accepted optimization for the receiver-dispatch lookup path because it keeps the expanded scope facts and replaces repeated owner-member scans with a precomputed index.
- AVmatrix benchmark comparison, previous safe default -> `owner-member-index-scope-resolution`: wall `110029.8ms` -> `105688.7ms` (`-3.9%`), resolution `3956ms` -> `895ms` (`-77.4%`), `USES` `711` -> `816`, resolved references `10604` -> `14177`, unresolved references `131143` -> `129718`. Relationship counts still vary slightly (`ACCESSES +1` vs safe default), so do not claim final equivalent-accuracy success until repeated median runs and parity checks are done.
- `reports/benchmark/2026-05-06-avmatrix-call-return-gitnexus-main.json` records the first cross-file return-type migration slice: TypeScript/JavaScript emits AST-reused `call-return` type bindings for variables assigned from imported function calls, and scope lookup resolves the imported callable's `returnType` through finalized bindings. Unresolved `call-return` facts fail open to the previous lookup behavior so they do not suppress existing fallback resolution.
- AVmatrix benchmark comparison, `owner-member-index-scope-resolution` -> `call-return-type-binding-scope-resolution`: wall `105688.7ms` -> `107954.6ms` (`+2.1%`), resolution `895ms` -> `1041ms` (`+16.3%`), crossFile `19001ms` -> `18808ms` (`-1.0%`), `ACCESSES` `181` -> `185`, resolved references `14177` -> `14236`, unresolved references `129718` -> `129036`, scope reference sites `143895` -> `143272`, scope-emitted edges `1450` -> `1454`, and crossFile reprocessed files `188` -> `187`. This is a correctness/architecture migration slice, not a speedup claim; sample the new `ACCESSES` edges, explain the reference-site count movement, and use repeated median benchmark runs before treating it as equivalent-accuracy progress toward retiring `crossFilePhase`.
- `reports/benchmark/2026-05-06-avmatrix-awaited-call-return-gitnexus-main*.json` records the awaited-call-return migration slice: TypeScript/JavaScript now emits the same AST-reused `call-return` binding when a variable is assigned from `await makeUser()`. Three repeated AVmatrix runs on `E:\Lap_trinh\GitNexus-main` measured wall times `108632.8ms`, `108070.6ms`, and `108712.7ms`; median wall time is `108632.8ms`. Resolution timings were `999ms`, `895ms`, and `901ms`; median resolution is `901ms`. Relationship counts varied across identical-code runs (`ACCESSES` `184`, `181`, `184`), proving this benchmark target still has small graph-count nondeterminism. Do not use this slice to claim final equivalent-accuracy success or speedup; use it as correctness coverage plus evidence that the benchmark protocol must compare repeated medians and graph parity, not one artifact.

Full build for UI/manual validation through `Start-AVmatrix.html`:

```powershell
powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1
```

This script builds `avmatrix`, builds `avmatrix-web`, builds `avmatrix-launcher\AVmatrixLauncher.exe`, builds `avmatrix-launcher\server-bundle\avmatrix-server.exe`, copies `node.exe`, copies the web build to `avmatrix-launcher\web-dist\`, and registers the `avmatrix://` protocol. A CLI-only `cd avmatrix && npm run build` is not enough before asking the user to test through the root launcher HTML.

Latest validation after the awaited call-return and test-runner stability slice:

- Full launcher build passed with `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1`.
- Targeted unit scope-resolution tests passed cleanly: `72/72` across `typescript-scope-captures`, `scope-reference-resolver`, `registries`, and `resolve-type-ref`.
- `cd avmatrix && npm test` now routes the default Vitest project through `vmForks` and runs native LadybugDB integration files one file at a time with per-file pool settings. The command exited `0` and the captured log contained no `Unhandled Errors` or `Unhandled Error` entries.

### Milestone 1: Baseline And Parity Targets

- Add a machine-readable AVmatrix benchmark artifact, for example `avmatrix analyze --force --benchmark-json <file> --benchmark-label <label>`.
- The artifact must include graph correctness snapshot, edge counts by type, unresolved counts, phase timings, parse/crossFile/resolution/lbug timings, duplicate-read/parse proxy counters, and resolution chunk/index counters.
- Do not accept a speedup claim from console wall time alone.
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
- Keep chunk boundaries deterministic and expose chunk/index cardinality metrics before moving execution out to workers.
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
- Keep `fileHash` as a nullable persisted field for legacy edges; scope-resolved edges should populate it from parse-time `ParsedFile.fileHash`.
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

- run AVmatrix before/after on the same machine, same repository checkout, same exclude rules, and same parser availability;
- use externally supplied GitNexus deep/scope numbers as the baseline where the plan needs GitNexus comparison; do not re-measure GitNexus locally unless explicitly requested;
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
