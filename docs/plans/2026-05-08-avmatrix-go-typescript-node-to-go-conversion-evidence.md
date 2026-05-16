# avmatrix-GO TypeScript/Node To Go Conversion Evidence

Source plan: [2026-05-08-avmatrix-go-typescript-node-to-go-conversion-plan.md](2026-05-08-avmatrix-go-typescript-node-to-go-conversion-plan.md)

This file contains execution evidence moved out of the checklist plan. Benchmark gates and benchmark result numbers live in [2026-05-08-avmatrix-go-typescript-node-to-go-conversion-benchmark.md](2026-05-08-avmatrix-go-typescript-node-to-go-conversion-benchmark.md).

## Phase 4 - HTTP API Shell

### Evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Runtime smoke passed: `go run ./cmd/avmatrix serve --host 127.0.0.1 --port 48747`
  served `/api/info` and `/api/repos`.

## Phase 4A - Go-Aware Launcher Build Gate

### Evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests using
  Go `1.26.3`.

- Package contents verified: `server-bundle\avmatrix-server.exe` exists,
  `server-bundle\avmatrix.exe` exists, and `server-bundle\node.exe` is absent.

- Runtime smoke passed: `server-bundle\avmatrix-server.exe` started the packaged Go backend and
  `http://127.0.0.1:4747/api/info` returned version `1.2.1`, launch context `global`, and
  runtime `go1.26.3`.

- Local runtime defaults were standardized on `127.0.0.1` for Go serve, TypeScript serve, Web UI
  backend defaults, launcher health/open URLs, Playwright/E2E defaults, Docker health checks, and
  active local-run documentation. `localhost` remains accepted only as a loopback compatibility
  alias or as fixture data.

## Phase 5 - Scanner And Ignore Semantics

### Evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./cmd/... ./internal/... -count=1` passed, including scanner/ignore tests for
  built-in ignores, `.gitignore`, `.avmatrixignore`, `AVMATRIX_NO_GITIGNORE`, include/exclude
  filters, large-file cutoff, hashing, missing-file reads, and language detection.

## Phase 6 - Parser Runtime

### Evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./cmd/... ./internal/... -count=1` passed, including parser fixtures for JavaScript,
  TypeScript, TSX, Go, syntax-error reporting without failure, unsupported-language failure, canceled
  context failure, and expired-deadline timeout metrics.

- Parser feasibility proof uses the latest checked Go module tags on this machine:
  `go-tree-sitter v0.25.0`, `tree-sitter-go v0.25.0`,
  `tree-sitter-javascript v0.25.0`, and `tree-sitter-typescript v0.23.2`.

- Freshness reconciliation on 2026-05-13:
  - Upstream `tree-sitter/tree-sitter` latest release is `v0.26.8`.
  - `go list -m -json github.com/tree-sitter/go-tree-sitter@latest` resolves the official Go
    binding to `v0.25.0`.
  - `go list -m -versions github.com/tree-sitter/go-tree-sitter` lists only `v0.23.0`,
    `v0.23.1`, `v0.24.0`, and `v0.25.0`; repository `master` resolves to an older
    pseudo-version than `v0.25.0`.
  - Decision: keep `github.com/tree-sitter/go-tree-sitter v0.25.0` as the latest
    Go-compatible runtime binding for this conversion path, and re-check at final cutover instead
    of introducing an unofficial native/runtime fork.

- Scan/parse handoff contract: parser `Request` consumes caller-provided `Source []byte`; it does
  not reread source files during parse.

## Phase 7 - ScopeIR Model

### Evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./cmd/... ./internal/... -count=1` passed, including `internal/scopeir`.

- `internal/scopeir` defines the ScopeIR root plus definition, import, call-site, access,
  heritage, scope, type annotation, return type, framework, and domain fact shapes. JSON
  serialization uses deterministic ordering and keeps type text such as `Promise<void>` unescaped.

## Phase 8 - TypeScript/JavaScript Provider

### Evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./cmd/... ./internal/... -count=1` passed, including `internal/providers/tsjs`.

- Exact ScopeIR parity signature fixture is checked in under
  `internal/providers/tsjs/testdata/typescript_scopeir_signature.golden.json`; it was generated
  from the existing TypeScript AST-aware scope bridge for the same fixture and is compared by
  `TestExtractTypeScriptScopeIRParityFixture`.

- Provider API consumes caller-provided source bytes plus the already parsed tree-sitter root node;
  it does not reread source files or reparse for scope extraction.

- Direct Go provider tests cover TypeScript module/class/interface/method/property/function/
  variable/type-alias definitions, owner-qualified member names, imports, alias imports, reexports,
  constructor calls, free calls, member calls, member reads/writes, extends/implements heritage,
  parameter and field type annotations, return types, local variable binding from same-file return
  annotations, interface property signatures, type-alias RHS references, and fileHash propagation
  on emitted facts.

- JavaScript provider test covers module/function/variable definitions, named imports, free calls,
  and member calls.

- Added `TestResolveTypeScriptGraphBaselineCountsAreReconciled`, which proves Go node count parity
  against the TypeScript resolution-only fixture; exact count parity for `ACCESSES`, `CALLS`,
  `EXTENDS`, `HAS_METHOD`, `IMPLEMENTS`, `IMPORTS`, `INHERITS`, and full-graph
  `METHOD_IMPLEMENTS`; and explicitly classified deltas for `CONTAINS`, `DEFINES`,
  `HAS_PROPERTY`, `MEMBER_OF`, `METHOD_OVERRIDES`, `STEP_IN_PROCESS`, and `USES`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before tests.

- Re-ran `go test ./internal/resolution -run 'TestResolveTypeScriptGraphBaselineCountsAreReconciled|TestResolveTypeScriptGraphSignatureFixture' -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

## Phase 9 - Resolution Phase

### Evidence

- Added `internal/graph` with Go graph node/relationship contracts, relationship constants, stable
  `Label:name` ID generation, deterministic relationship sorting, and relationship type counts.

- Added `internal/resolution` with ScopeIR-backed workspace indexes for definitions, imports,
  scopes, reference sites, owner members, type bindings, and basic method dispatch.

- The Go resolver now emits definition edges, finalized file-level `IMPORTS`, per-symbol import-use
  `USES`, scope-resolved `CALLS`, `ACCESSES`, type-reference `USES`, compatibility
  `EXTENDS`/`IMPLEMENTS`, scope `INHERITS`, `METHOD_OVERRIDES`, and `METHOD_IMPLEMENTS`.

- Direct tests:
  - `TestResolveTypeScriptGraphFixture` covers import resolution, constructor calls, free calls,
    member calls through receiver type binding, member read/write access, type references,
    extends/implements, method override, method implements, and populated reference index.
  - `TestResolveTypeScriptGraphSignatureFixture` checks the current Go resolution graph signature
    fixture at `internal/resolution/testdata/typescript_graph_signature.golden.json`.
  - `TestResolveMergesDuplicateSemanticEdgesAndCountsUnresolved` covers semantic duplicate-edge
    merge and unresolved reference metrics.

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Current Go resolution fixture emits `24` nodes and `66` relationships. Graph parity is not closed:
  current deltas include no structure-phase `CONTAINS` in the Go resolution-only surface, extra
  import-use/type-reference `USES` edges, and one additional interface-property edge. Keep the
  Phase 8 exact graph parity item open until these deltas are reconciled against the TypeScript
  baseline contract.

- Added `TestResolveImportedTypeAlias` to prove an imported TypeScript `type` alias resolves through
  the import binding and emits both file-level import-use `USES` and function-level type-reference
  `USES` edges.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before tests.

- Re-ran `go test ./cmd/... ./internal/... -count=1` after the imported type-alias test.

- Added Go resolution accumulator lifecycle support mirroring the TypeScript compatibility
  contract: finalize rejects late appends, dispose clears storage and makes the accumulator terminal,
  `Resolve` always finalizes/disposes around the single-pass ScopeIR resolution surface, and
  `Options.SkipCompatibilityCrossFile` records diagnostic skip reason
  `disabled-by-pipeline-option` without changing graph output.

- Added direct tests:
  - `TestBindingAccumulatorLifecycle` covers append, file-scope lookup, finalize, append-after-
    finalize rejection, dispose clearing, idempotent dispose, and append-after-dispose rejection.
  - `TestResolveSkipCompatibilityCrossFileReportsDiagnosticWithoutChangingGraph` proves the flag
    reports `disabled-by-pipeline-option`, default Go resolution reports
    `covered-by-scopeir-single-pass-resolution`, files reprocessed stays `0`, the accumulator is
    finalized/disposed, and the graph signature is unchanged.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before tests.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

## Phase 10 - LadybugDB Persistence

### Evidence

- Phase 1 base proof is historical feasibility only:
  `baseline\phase-1-contract-freeze\go-ladybugdb-windows-proof.md` confirmed the native
  LadybugDB `v0.16.1` runtime can open an in-memory DB, create a node table, insert a node, and
  read it back on Windows. Phase 10 cutover evidence must use the direct native `v0.16.1` C API;
  lagging wrapper modules are not accepted as runtime authority.

- Phase 10 extension proof passed in `.tmp\phase10-ladybug-extension-proof` using
  `CGO_ENABLED=1`, `CGO_LDFLAGS=-L.tmp\phase1-go-proofs\ladybugdb\native\liblbug-windows-x86_64 -llbug_shared`,
  and the native runtime directory prepended to `PATH`.

- Proof operations: `INSTALL FTS`, `LOAD EXTENSION FTS`, `INSTALL VECTOR`,
  `LOAD EXTENSION VECTOR`, create `File`, create FTS index, create `CodeEmbedding` with
  `FLOAT[4]`, create vector index with cosine metric, insert/read `File`.

- Observed output: `ladybugdb extension proof ok: FTS and VECTOR loaded, indexed, and read`.

- Direct Go support is not blocked for the current Windows development path; no bridge is selected
  for Phase 10 at this point.

- Added `internal/lbugschema` with Go-owned LadybugDB schema creation contract: node table list,
  relationship type list, `CodeRelation` relationship table with 203 TypeScript-baseline
  source/target label pairs, audit metadata columns (`type`, `confidence`, `reason`, `step`,
  `resolutionSource`, `evidence`, `fileHash`), `CodeEmbedding`, vector index query, and FTS index
  query shapes.

- Added `TestSchemaConstantsMatchFrozenContract`, `TestSchemaQueriesPreserveDDLShape`, and
  `TestEmbeddingAndIndexQueries`; tests compare Go constants to
  `baseline\phase-1-contract-freeze\ladybugdb-graph-contract.json`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before tests.

- Re-ran `go test ./internal/lbugschema -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Added `internal/lbugload` with Go-owned CSV/COPY load contract:
  - node CSV export for every LadybugDB node table shape;
  - relationship CSV export with audit metadata columns (`type`, `confidence`, `reason`, `step`,
    `resolutionSource`, `evidence`, `fileHash`);
  - per-source/target-label relationship CSV splitting for LadybugDB `COPY`;
  - node and relationship `COPY` query generation with normalized `/` paths;
  - relationship fallback insert path retained only as a diagnostic/recovery path for unsupported
    schema pairs or failed relationship `COPY`;
  - `QueryRunner` execution boundary so the default launcher build does not require native
    LadybugDB linker flags before the packaging/runtime adapter is wired.

- Added `TestExportGraphCSVsWritesNodeRelationshipAndSplitContracts`,
  `TestLoadCSVExportUsesCopyForSupportedNodeAndRelationshipPairs`,
  `TestLoadCSVExportReportsDiagnosticFallbackOnlyForSchemaOrCopyGaps`, and
  `TestCopyQueriesMatchLadybugCSVContract`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before loader
  tests.

- Re-ran `go test ./internal/lbugload -count=1`.

- Re-ran `go test ./internal/lbugschema -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Attempted `go test ./... -count=1`; current full-module glob still enters baseline sample
  fixture directories under `avmatrix\test\fixtures` and fails on intentionally non-buildable
  sample packages/C files, so the clean Go runtime gate for this batch remains
  `go test ./cmd/... ./internal/... -count=1`.

- Added `internal/lbugruntime` with Go-owned runtime contracts:
  - read-only Cypher guard that blocks write keywords while allowing read-only `CALL`
    FTS/VECTOR queries and label names such as `:CREATE`;
  - pre-warmed bounded read pool with checkout/release semantics and blocked-write protection
    before a query reaches a connection;
  - legacy `CodeEmbedding` content-hash fallback, treating old rows without chunk-aware metadata
    or `contentHash` as stale via `STALE_HASH_SENTINEL`;
  - FTS/VECTOR extension lifecycle query sequencing with idempotent already-loaded handling;
  - missing-schema, busy/lock, WAL-corruption, and already-loaded error classifiers plus busy
    retry policy.

- Added `TestIsWriteQueryMatchesPoolAdapterContract`, `TestValidateReadQueryRejectsWrites`,
  `TestReadPoolLimitsConcurrentCheckout`, `TestReadPoolExecuteBlocksWritesBeforeQuery`,
  `TestFetchExistingEmbeddingHashesReadsCurrentSchema`,
  `TestFetchExistingEmbeddingHashesFallsBackForLegacyRows`,
  `TestFetchExistingEmbeddingHashesMissingTableMeansNoCache`,
  `TestExtensionStateLoadsFTSAndVectorIdempotently`, and
  `TestBusyRetryPolicyRetriesOnlyBusyErrors`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before runtime
  tests.

- Re-ran `go test ./internal/lbugruntime -count=1`.

- Re-ran `go test ./internal/lbugload -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Added `StdioSilencer` to guard future native read calls with serialized scoped
  `os.Stdout`/`os.Stderr` redirection to `os.DevNull`, restoring both streams after success or
  error.

- Added `TestStdioSilencerSuppressesScopedOutputAndRestores` and
  `TestStdioSilencerRestoresAfterError`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before stdio
  guard tests.

- Re-ran `go test ./internal/lbugruntime -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Added optional native LadybugDB integration coverage under build tag `ladybugdb` in
  `internal/lbugnative`: direct C API calls to native LadybugDB `v0.16.1`, schema creation,
  CSV/COPY load into a real LadybugDB file, read-only reopen, row readback, and streamed
  relationship readback through the Go runtime query guard and stdio silencer.

- LadybugDB version authority: `LadybugDB/ladybug v0.16.1` is the selected runtime. The Go runtime
  path must not depend on lagging wrapper modules for cutover acceptance.

- Re-ran `go test ./internal/lbugload -count=1`.

- Re-ran `go test ./internal/lbugnative -count=1` -> no default tests, confirming the native test
  is gated out of normal launcher builds.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Re-ran native tagged verification with
  `CGO_ENABLED=1`,
  `CGO_CFLAGS=-I.tmp\phase1-go-proofs\ladybugdb\native\liblbug-windows-x86_64`,
  `CGO_LDFLAGS=-L.tmp\phase1-go-proofs\ladybugdb\native\liblbug-windows-x86_64 -llbug_shared`,
  and native LadybugDB `v0.16.1` runtime directory prepended to `PATH`:
  `go test -tags ladybugdb ./internal/lbugnative -run TestNativeLadybugPersistenceReadbackAndStream -count=1`.

## Phase 11 - Analyze Pipeline

### Evidence

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before tests.

- Re-ran `go test ./internal/analyze -count=1`.

- Re-ran `go test ./internal/cli -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Added repo storage lock paths (`analyze.lock`) and temp workspace path (`analyze.tmp`) to
  `internal/repo` storage contracts.

- Added atomic repo writer lock acquisition/release with `ErrLockHeld`, wired into
  `analyze.Run`, and covered by `TestAcquireStorageLockExcludesConcurrentWritersAndReleases` and
  `TestRunRejectsConcurrentWriterLock`.

- Added `Force` analyze option plus CLI `--force`; force removes stale `lbug` output before the
  pipeline starts and always removes the analyze temp workspace on exit. Covered by
  `TestRunForceRemovesPreviousLbugOutput`.

- Added `.avmatrix\graph.json` as the current readable Go graph snapshot artifact for CLI analyze;
  writes are atomic via `graph.json.tmp` rename and `--force` removes stale graph snapshots before
  a new run. This is the readable graph bridge until native LadybugDB DB load is wired.

- CLI `analyze` now writes the graph snapshot by default and prints its path in the graph summary.

- Added `db_load` orchestration hook using `lbugload.QueryRunner`: the phase exports graph CSVs
  under the analyze temp workspace, runs the Phase 10 CSV/COPY loader, records DB load metrics, and
  keeps fallback count observable.

- Added native DB runner selection for the Go analyze path: default builds report a skipped DB load
  with an explicit native-unavailable reason, while `ladybugdb` builds open the native LadybugDB
  database, initialize the Go schema contract, and feed the same CSV/COPY loader used by the test
  runner path.

- Added `internal/communities` as a Go-owned community enrichment boundary. The first runnable
  implementation uses deterministic connected components over `CALLS`/`EXTENDS`/`IMPLEMENTS`
  symbol edges, emits `Community` nodes, and writes `MEMBER_OF` edges for supported schema pairs.

- Added `internal/processes` as a Go-owned process enrichment boundary. It scores entry-point
  candidates, traces deterministic `CALLS` paths, emits `Process` nodes, and writes
  `ENTRY_POINT_OF`/`STEP_IN_PROCESS` relationships with step ordering.

- Added `internal/structure` as a Go-owned project structure enrichment boundary. It creates
  `Folder` nodes and `CONTAINS` edges from scanned paths while preserving richer `File` node
  metadata already emitted by resolution.

- Added `internal/documents` as a Go-owned document enrichment boundary. Markdown/MDX files create
  `Section` hierarchy nodes and local-link `IMPORTS` edges; `.doc`, `.docx`, `.pdf`, and
  Excel-family files are scanned and marked with document metadata without binary text parsing.

- Updated scanner/ignore contracts so `.doc`, `.docx`, `.pdf`, `.xls`, `.xlsx`, `.xlsm`,
  `.xlsb`, Excel templates, `.ods`, `.csv`, and `.tsv` can enter the Go graph instead of being
  dropped by hardcoded binary/data ignore rules.

- Added `internal/cobol` as a Go-owned COBOL/JCL enrichment boundary. It marks mainframe file
  metadata, emits COBOL `Module`/`Namespace`/`Function` graph structure, links COPY copybooks with
  `IMPORTS`, links PERFORM/CALL with `CALLS`, and links JCL EXEC steps to COBOL modules.

- Updated scanner contract so `.copybook`, `.jcl`, `.job`, and `.proc` enter the Go graph as the
  COBOL/mainframe language slice.

- Added the required LadybugDB relation pairs for COBOL/JCL COPY-path loading:
  `CodeElement -> CodeElement`, `CodeElement -> Module`, `Module -> Namespace`, and
  `Namespace -> Function`.

- Added `internal/routes` as a Go-owned route enrichment boundary. It emits Next.js `app`/`pages`
  filesystem route nodes, Express-style framework route registrations, file-owned
  `HANDLES_ROUTE` edges, and local fetch-consumer `FETCHES` edges.

- Added `internal/tools` as a Go-owned tool enrichment boundary. It detects object-style tool
  definitions with `inputSchema`, `.tool(...)` registrations, decorator tools, emits `Tool` nodes,
  and links owning files with `HANDLES_TOOL`.

- Added `internal/orm` as a Go-owned ORM dataflow boundary. It detects Prisma client calls and
  Supabase `.from(...).select/insert/update/delete/upsert` chains, emits `QUERIES` edges, reuses a
  unique existing `Class`/`Interface`/`CodeElement` model node by name, and otherwise creates a
  fallback `CodeElement` model/table node.

- Added `internal/mro` as a Go-owned graph-level MRO boundary. It builds adjacency from
  `EXTENDS`/`IMPLEMENTS` and `HAS_METHOD`, emits `METHOD_OVERRIDES` for ancestor method-name
  collisions using first-definition order, and emits `METHOD_IMPLEMENTS` for concrete methods that
  match interface/trait contracts by name and parameter signature.

- Added `internal/embeddings` as a Go-owned embedding runtime boundary for the first embedding
  parity slice. It preserves default embedding config values, env-driven HTTP mode, OpenAI-style
  batch requests, retry handling, vector-count checks, dimension validation, and sanitized endpoint
  errors before the full analyze/server pipeline is wired.

- Extended `internal/embeddings` with graph-backed embedding pipeline behavior: embeddable label
  selection, metadata text generation, SHA-1 content hashing, character chunk fallback, incremental
  fresh/stale split, stale embedding delete queries, chunk-aware `CodeEmbedding` inserts, vector
  extension/index queries, progress events, and fail-fast dimension mismatch checks.

- Wired the Go analyze path to the embedding runtime behind `--embeddings`. The CLI now exposes the
  flag, resolves HTTP-mode dimensions from `AVMATRIX_EMBEDDING_DIMS`, runs an `embeddings` phase
  after DB load, records embedding metrics, rejects embedding runs when no DB runner exists, prefers
  HTTP embedding mode when configured, and falls back to the Go local Hugot runtime.

- Extended native LadybugDB write runners with row-read support so the analyze embedding phase can
  reuse the existing stale-hash cache reader before deciding which nodes require re-embedding.

- Added Go semantic search runtime in `internal/embeddings`: query text is embedded once, vector
  search uses the `CodeEmbedding` vector index with the configured dimensions, chunk rows are
  deduplicated by nearest distance per node, and metadata is hydrated from node tables before
  returning sorted search results.

- Added `/api/search` to the Go HTTP server. The handler preserves the Web UI response shape,
  resolves repos from body or query params, normalizes query/mode/limit/enrich inputs, maps
  unavailable embedding/search dependencies to HTTP 501, and delegates search execution through an
  injectable Go service.

- Added a read-only native LadybugDB runner for server search. It opens the repo `.avmatrix/lbug`
  database in read-only mode, validates read queries before execution, and keeps default builds on
  the same explicit native-unavailable error path.

- Wired the server search service to the Go semantic runtime: it resolves HTTP embedding mode when
  configured, falls back to local Hugot model mode, runs `embeddings.SemanticSearch`, and maps
  semantic distance into Web UI-compatible score/rank/source fields.

- Added Go server embedding job lifecycle endpoints: `POST /api/embed`,
  `GET /api/embed/{jobId}`, `GET /api/embed/{jobId}/progress`, and
  `DELETE /api/embed/{jobId}`. The server reuses the repo analyze lock, returns 409 for concurrent
  analyze/embed writers, emits Web UI-compatible progress/status payloads, streams SSE terminal
  events, supports cancellation through job contexts, and preserves the 30-minute timeout contract.

- Wired the Go server embedding service to graph snapshot + native DB runtime. It loads
  `.avmatrix/graph.json`, opens `.avmatrix/lbug` through the native write runner, reuses existing
  embedding hashes for incremental runs, resolves HTTP embedding mode when configured, falls back to
  local Hugot model mode, and calls `embeddings.Run` with runtime progress mapping.

- Reordered the implemented Go Phase 11 runtime chain to `scan -> structure -> documents -> cobol
  -> parse -> routes -> tools -> orm -> cross_file_binding -> resolution -> mro -> communities ->
  processes -> db_load`. Structure now bootstraps base `File` nodes before
  document/COBOL/route/tool/ORM enrichment, and `resolution.ResolveInto` writes symbol/reference
  output into that existing graph while preserving earlier file metadata and relationships.

- Added `TestApplyEmitsFolderNodesAndContainsEdges` and
  `TestApplyPreservesExistingFileNodeProperties`.

- Added `TestApplyEmitsMarkdownSectionsHierarchyAndLocalLinks`,
  `TestApplyMarksWordPDFAndSpreadsheetFilesWithoutTextParsing`, and
  `TestWalkRepositoryPathsIncludesDocumentAndSpreadsheetFiles`.

- Added `TestApplyEmitsCobolProgramsCopyPerformCallAndJCLLinks`,
  `TestApplyReturnsZeroForNonMainframeFiles`, and
  `TestWalkRepositoryPathsIncludesMainframeFiles`.

- Added `TestApplyEmitsFilesystemFrameworkAndFetchRouteEdges` and
  `TestApplySkipsDuplicateRoutesAndExternalFetches`.

- Added `TestApplyEmitsObjectRegistrationAndDecoratorTools` and
  `TestApplyDeduplicatesToolsAndSkipsNonCodeFiles`.

- Added `TestApplyEmitsPrismaAndSupabaseQueryEdges` and
  `TestApplyReusesUniqueExistingModelNodeAndDeduplicates`.

- Added `TestApplyEmitsMethodOverrideForAncestorCollision` and
  `TestApplyEmitsMethodImplementsForConcreteInterfaceMatch`.

- Added `TestNodesFromGraphSelectsEmbeddableLabelsAndContext`,
  `TestGenerateTextBuildsMetadataAndCleansContent`,
  `TestContentHashIgnoresStructuralNameEnrichment`,
  `TestExtractDeclarationOnlySkipsMethodBodies`, `TestChunkNodeKeepsShortLabelsSingleChunk`,
  `TestCharacterChunksUsesOverlapAndLineRanges`,
  `TestRunEmbedsNewAndStaleNodesAndCreatesVectorIndex`,
  `TestRunCreatesVectorIndexWhenAllNodesAreFresh`,
  `TestRunRejectsEmbeddingDimensionMismatch`, and
  `TestCreateEmbeddingQueryEscapesStringsAndFormatsVector`.

- Added `TestRunExecutesEmbeddingsWhenEnabledWithInjectedEmbedder`,
  `TestRunRejectsEmbeddingsWhenDBRunnerIsUnavailable`, and
  `TestAnalyzeHelpShowsEmbeddingsFlag`.

- Added `TestSemanticSearchQueriesVectorIndexDedupsChunksAndHydratesMetadata`,
  `TestSemanticSearchRejectsQueryDimensionMismatch`, and
  `TestDedupBestChunksKeepsNearestChunkPerNode`.

- Added `TestSearchEndpointCallsSearchServiceAndReturnsResults`,
  `TestSearchEndpointRejectsMissingQuery`, `TestSearchEndpointReturnsUnavailableStatus`,
  `TestSearchServiceUsesSemanticRuntime`, and `TestSearchServiceMapsNativeUnavailable`.

- Added `TestEmbedEndpointStartsJobAndCompletes`, `TestEmbedEndpointRejectsHeldRepoLock`,
  `TestEmbedEndpointCancelMarksJobFailed`, `TestEmbedProgressEndpointSendsCompleteEvent`,
  `TestEmbedServiceRunsEmbeddingPipelineFromGraphSnapshot`, and
  `TestEmbedServiceMapsNativeUnavailable`.

- Added `TestOpenReadRunnerUnavailableWithoutNativeBuild`.

- Added `TestApplyEmitsCommunityNodesAndMembershipEdges`, `TestApplySkipsSingletonCommunities`,
  `TestApplyEmitsProcessesAndStepRelationships`, and `TestApplySkipsShortAndTestFileTraces`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before lock/force
  and enrichment tests.

- Re-ran `go test ./internal/ignore -count=1`.

- Re-ran `go test ./internal/scanner -count=1`.

- Re-ran `go test ./internal/structure -count=1`.

- Re-ran `go test ./internal/documents -count=1`.

- Re-ran `go test ./internal/cobol -count=1`.

- Re-ran `go test ./internal/routes -count=1`.

- Re-ran `go test ./internal/tools -count=1`.

- Re-ran `go test ./internal/orm -count=1`.

- Re-ran `go test ./internal/mro -count=1`.

- Re-ran `go test ./internal/embeddings -count=1`.

- Re-ran `go test ./internal/analyze -count=1`.

- Re-ran `go test ./internal/cli -count=1`.

- Re-ran `go test ./internal/lbugnative -count=1`.

- Re-ran `go test ./internal/lbugschema -count=1`.

- Re-ran `go test ./internal/communities -count=1`.

- Re-ran `go test ./internal/processes -count=1`.

- Re-ran `go test ./internal/repo -count=1`.

- Re-ran `go test ./internal/analyze -count=1`.

- Re-ran `go test ./internal/cli -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before HTTP search
  tests.

- Re-ran `go test ./internal/httpapi -count=1`.

- Re-ran `go test ./internal/lbugnative -count=1`.

- Re-ran `go test ./internal/embeddings -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before HTTP embed
  job tests.

- Re-ran `go test ./internal/httpapi -count=1`.

- Re-ran `go test ./internal/embeddings -count=1`.

- Re-ran `go test ./internal/lbugnative -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before runtime
  chain tests.

- Re-ran `go test ./internal/resolution -count=1`.

- Re-ran `go test ./internal/analyze -count=1`.

- Re-ran `go test ./internal/cli -count=1`.

- Re-ran `go test ./internal/structure ./internal/documents ./internal/cobol ./internal/routes
  ./internal/tools ./internal/orm -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Added Go CLI `status`, `wiki`, and `wiki-mode` runtime surfaces. `status` preserves non-git
  repo output, indexed commit/current commit output, stale Kuzu migration guidance, and stale index
  status. `wiki`/`wiki-mode` preserve local-only wiki gating through global `runtime.json`, with
  invalid-mode stderr guidance and a silent exit-code-only failure path for disabled wiki
  generation.

- Added `TestStatusReportsNotGitRepository`, `TestStatusReportsIndexedRepositoryState`,
  `TestStatusReportsStaleKuzuIndex`, `TestWikiCommandReportsDisabledAndFailsSilently`,
  `TestWikiModeWritesRuntimeConfig`, and `TestWikiModeRejectsInvalidMode`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before CLI
  status/wiki tests.

- Re-ran `go test ./internal/repo ./internal/cli -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Added Go `analyze --skills`, `--skip-agents-md`, and `--no-stats` runtime behavior. Successful
  CLI analyze now writes local repo metadata plus global registry entries before optional AI
  context generation, so `status` has a real Go-produced index state to read.

- Added `internal/aicontext` for Go-owned AI context generation: repo-specific generated skills
  from graph `Community`/`MEMBER_OF`/`Process` data, base AVmatrix skill installation, and
  managed AVmatrix block upsert for `AGENTS.md` and `CLAUDE.md`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before AI
  context/skill generation tests.

- Re-ran `go test ./internal/aicontext ./internal/cli -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before
  cross-file boundary tests.

- Re-ran `go test ./internal/resolution ./internal/analyze ./internal/cli -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Added Go local embedding model mode with Hugot `v0.7.2` and `onnxruntime_go v1.30.0`. HTTP
  embedding env config remains the priority override; when unset, CLI/server embedding and semantic
  search resolve the local model runtime, download the required ONNX/tokenizer/config files into the
  AVmatrix-Go HuggingFace cache without Hub symlink/lock paths, normalize embeddings, and validate
  384-dimensional vectors.

- Added `TestResolveRuntimeEmbedderPrefersHTTPMode`,
  `TestResolveRuntimeEmbedderUsesLocalModeWhenHTTPUnset`,
  `TestEnsureLocalModelDownloadsMissingCacheWithoutHubSymlinks`, and local cache/config guard tests.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before local
  embedding tests.

- Re-ran `go test ./internal/embeddings ./internal/analyze ./internal/httpapi -count=1`.

- Re-ran local model smoke with `go run .tmp\local_embedding_smoke.go`: `vectors=1 dimensions=384
  configured=384`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Re-ran tagged native read/write integration:
  `go test -tags ladybugdb ./internal/lbugnative -run TestNativeLadybugPersistenceReadbackAndStream
  -count=1`.

- Phase 11 community/process parity follow-up:
  - Replaced the Go connected-component community partition with a deterministic modularity
    local-move partition over the same symbol/clustering graph shape used by the TypeScript
    community phase. The Go pass now preserves singleton membership edges and only materializes
    non-singleton `Community` nodes, matching the TypeScript membership/node split.
  - Added regression coverage for bridge-linked dense clusters so a single bridge edge no longer
    collapses two dense communities into one component.
  - Verification order followed the repo rule: full launcher build first, then tests.
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed,
    `go test ./internal/communities ./internal/processes ./internal/analyze -count=1` passed, and
    `go test ./cmd/... ./internal/... -count=1` passed.
  - Baseline comparison on `.tmp\resolution-baseline-fixture`:
    Go launcher analyze produced `nodes=29`, `relationships=84`, `Community=3`, `MEMBER_OF=9`,
    `Process=1`, `STEP_IN_PROCESS=3`.
    TypeScript dist analyze with `--skip-git` produced benchmark stats `nodes=28`,
    `relationships=67`, `Community=3`, `MEMBER_OF=9`, `Process=1`, `STEP_IN_PROCESS=3`.
    The remaining node/edge delta is from broader Go resolver fact emission, not from the Phase 11
    community/process surface.
  - Residual Phase 11 benchmark scope: this-repository and medium-repo full analyze native DB-load
    timing remain benchmark evidence work; they are not a blocker for returning to the Phase 14
    provider checklist after the current community/process parity slice.

## Phase 12 - HTTP Analyze And Graph Serving

### Evidence

- Added Go HTTP analyze endpoints: `POST /api/analyze`, `GET /api/analyze/{jobId}`,
  `DELETE /api/analyze/{jobId}`, and `GET /api/analyze/{jobId}/progress`.

- Analyze jobs now run the Go analyzer in-process with 30-minute context timeout, cancel support,
  progress mapping from analyzer phases, registry/meta recording, graph snapshot output, and SSE
  completion payload containing `repoName` and `repoPath`.

- `/api/graph` now reads the repo-scoped Go `graph.json`, strips node `content` by default,
  honors `includeContent=true`, and streams Web UI-compatible NDJSON records:
  `{"type":"node","data":...}` and `{"type":"relationship","data":...}`.

- `/api/repo?awaitAnalysis=true` now waits for the active analyze job for the requested repo path
  before returning metadata, including the path-before-registry case used immediately after local
  analyze starts.

- Added `TestAnalyzeRejectsInvalidRequests`, `TestAnalyzeStartsJobAndStreamsCompletionPayload`,
  `TestAnalyzeRejectsConcurrentJobAndCancelsRunningJob`, `TestGraphReturnsJSONForRegisteredRepo`,
  `TestGraphStreamingReturnsNDJSON`, and `TestRepoInfoAwaitAnalysisHoldsUntilAnalyzeCompletes`.

- Preserved loopback-only CORS and private-network behavior through the existing
  `TestCORSAllowsLoopbackAndPrivateNetworkPreflight` and
  `TestCORSLeavesDisallowedOriginUnreflected` coverage under
  `go test ./internal/httpapi -count=1`.

- Added partial Web panel graph-read coverage: `GET /api/file` now reads repo-relative source files
  with 0-indexed line ranges and repository-escape rejection, and `POST /api/query` now supports
  the `ProcessesPanel` `STEP_IN_PROCESS` and bounded `CALLS` query shapes against the repo
  `graph.json` snapshot.

- Added `TestFileEndpointReadsRegisteredRepoFileRange`,
  `TestFileEndpointRejectsRepositoryEscape`, and
  `TestQueryEndpointReturnsProcessStepsAndCallEdges`.

- Added the remaining Web graph-read endpoints: `GET /api/grep`, `GET /api/processes`,
  `GET /api/process`, `GET /api/clusters`, and `GET /api/cluster`.

- Extended `POST /api/query` to cover the existing QueryFAB example shapes for node label lists
  and bounded relationship scans in addition to the `ProcessesPanel` process/edge queries.

- Added `TestQueryEndpointSupportsQueryFABExamples`, `TestGrepEndpointSearchesIndexedRepoFiles`,
  `TestProcessesAndProcessDetailEndpointsUseGraphSnapshot`, and
  `TestClustersAndClusterDetailEndpointsUseGraphSnapshot`.

- HTTP analyze now preflights the same repo-local `analyze.lock` as embedding before accepting a
  job, while the analyzer runtime remains the lock owner during execution. This preserves same-repo
  analyze/embed writer exclusion without double-acquiring the production analyze lock.

- Added `TestAnalyzeRejectsHeldRepoLock` and
  `TestAnalyzeLockBlocksEmbedAndReleasesAfterCancel` to prove same-repo analyze/embed writer
  exclusion and lock release after cancel through the runner-owned production lock path.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before HTTP tests.

- Re-ran `go test ./internal/httpapi -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Go HTTP analyze now preserves same-repo active-job dedupe by returning the existing running job
  with 202 instead of starting a second runner or returning 409. Added
  `TestAnalyzeDeduplicatesActiveSameRepoJob`.

- Go `JobManager` now cleans expired terminal jobs after the one-hour lifecycle TTL and keeps
  same-repo dedupe/different-repo single-slot behavior covered by direct `jobs_test.go` tests.

- Child-worker crash retry is not ported because the Go Phase 12 analyze path runs in-process and
  has no child worker exit/crash surface. If a future Go implementation introduces an
  out-of-process analyze worker, that worker boundary must add retry/backoff before cutover.

- Web UI runtime validation passed against the packaged Go backend. The test used isolated
  `AVMATRIX_HOME` registries under `.tmp/`, a deterministic fixture with `package.json` plus the
  resolution/process fixture files, `avmatrix-launcher\server-bundle\avmatrix.exe serve --host
  127.0.0.1 --port 4747`, and `pnpm exec vite --host 127.0.0.1 --port 5173`.

- `pnpm exec playwright test e2e/server-connect.spec.ts --project=chromium --workers=1
  --reporter=list` passed: 5/5 tests in 25.8 s, covering repo graph load, My AI shell,
  Processes panel, process highlight, node selection, and highlight clearing against the Go HTTP
  API.

- Additional Web UI analyze smoke passed from an empty isolated registry: the UI opened the
  Analyze Repository form, submitted the fixture path through `POST /api/analyze`, waited for Go
  SSE completion, reached status `Ready`, and rendered `package.json` from the resulting graph.

- Residual Phase 12 scope: none for the currently implemented Go slice.

## Phase 13 - MCP Server

### Evidence

- Added Go-owned `internal/mcp` stdio server with MCP Content-Length framing, `initialize`,
  `ping`, `tools/list`, `tools/call`, empty `resources/list`, empty `resources/templates/list`,
  and empty `prompts/list` handlers. The first tool slice exposes and executes `list_repos`.

- Added CLI `avmatrix mcp`, routed through the Go stdio server. The command writes protocol frames
  to stdout and keeps normal CLI logging on stderr through the existing CLI logger boundary.

- `list_repos` reads the Go global registry via `repo.Store`, returns the current repo metadata
  array shape, and appends the same next-step hint pattern pointing callers to
  `avmatrix://repo/{name}/context`.

- Added `TestServeHandlesInitializeAndToolsList`, `TestServeCallToolListRepos`, and
  `TestMCPHelpShowsStdioServer`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before MCP tests.

- Re-ran `go test ./internal/mcp ./internal/cli -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe mcp` with a real stdio
  `tools/list` frame; response contained `list_repos`.

- Added Go `/api/mcp` HTTP endpoint backed by the shared MCP JSON-RPC handler. The endpoint
  supports initial POST session creation, `Mcp-Session-Id` reuse, unknown-session JSON-RPC errors,
  notification-only `202 Accepted`, DELETE session close, and a 30-minute idle session TTL.

- Exposed `Mcp-Session-Id` through loopback CORS so browser-based MCP clients can read and reuse
  the session header.

- Added `TestMCPHTTPInitializesAndListsTools`, `TestMCPHTTPUnknownSessionReturnsJSONRPCError`,
  `TestMCPHTTPNotificationReturnsAccepted`, and `TestMCPHTTPSessionsExpireIdleSessions`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before HTTP MCP
  tests.

- Re-ran `go test ./internal/mcp ./internal/httpapi -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe serve --host 127.0.0.1 --port 48713`
  with POST `/api/mcp` initialize followed by session-scoped `tools/list`; response contained
  `list_repos`.

- Added Go MCP context resource support through `resources/templates/list` and `resources/read` for
  `avmatrix://repo/{name}/context`. The resource reads the Go registry/meta snapshot, emits
  text/yaml project stats, available MCP tools, and related resource URIs.

- Added `TestServeReadsRepoContextResource`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before context
  resource tests.

- Re-ran `go test ./internal/mcp ./internal/httpapi -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe serve --host 127.0.0.1 --port 48714`
  with HTTP MCP `resources/read` for `avmatrix://repo/AVmatrix-GO/context`; response contained
  `tools_available` and `project: AVmatrix-GO`.

- Added Go MCP `resources/read` support for `avmatrix://repo/{name}/clusters` and
  `avmatrix://repo/{name}/processes`, backed by the Go `graph.json` snapshot and YAML summaries.

- Expanded `resources/templates/list` to expose context, clusters, and processes templates.

- Added `TestServeReadsRepoClustersAndProcessesResources`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before
  clusters/processes resource tests.

- Re-ran `go test ./internal/mcp ./internal/httpapi -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe serve --host 127.0.0.1 --port 48715`
  with HTTP MCP `resources/read` for `avmatrix://repo/AVmatrix-GO/clusters` and
  `avmatrix://repo/AVmatrix-GO/processes`; responses contained `modules:` and `processes:`.

- Added Go MCP static resources `avmatrix://repos` and `avmatrix://setup`.

- Added schema/detail resource support for `avmatrix://repo/{name}/schema`,
  `avmatrix://repo/{name}/cluster/{clusterName}`, and
  `avmatrix://repo/{name}/process/{processName}`.

- Expanded `resources/templates/list` to expose the six baseline templates.

- Added `prompts/list` and `prompts/get` for `detect_impact` and `generate_map`.

- Added `TestServeReadsSchemaDetailResourcesAndPrompts`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before
  schema/detail/prompts tests.

- Re-ran `go test ./internal/mcp ./internal/httpapi -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe serve --host 127.0.0.1 --port 48716`
  with HTTP MCP `resources/list`, `resources/templates/list`, `resources/read` for schema,
  `prompts/list`, and `prompts/get`; responses contained `avmatrix://repos`,
  `avmatrix://repo/{name}/schema`, `INHERITS`, `detect_impact`, and the generated map prompt.

- Added Go MCP `query` and `cypher` tools to stdio/HTTP discovery. `query` ranks process matches
  from the Go graph snapshot and returns related process symbols plus matching definitions;
  `cypher` runs the shared read-only query guard before executing the bounded Go graph adapter.

- Added `TestServeCallToolsQueryAndCypher` for `query` happy path, `cypher` happy path, and
  write-query rejection, and updated HTTP MCP discovery tests for the expanded tool surface.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before MCP tests.

- Re-ran `go test ./internal/mcp ./internal/httpapi -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Re-ran `avmatrix analyze --force` before MCP smoke and staged-scope detection.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe serve --host 127.0.0.1 --port 48718`
  with HTTP MCP `tools/list`, `tools/call query`, and `tools/call cypher`; responses contained
  `list_repos`, `query`, `cypher`, `process_symbols`, `row_count`, and `markdown`.

- Added Go MCP `context` tool to stdio/HTTP discovery. The tool resolves by `uid` or symbol name,
  supports `file_path`, `kind`, and `include_content`, returns disambiguation candidates when a
  symbol cannot be selected safely, and emits the baseline-shaped `status`, `symbol`, categorized
  `incoming`/`outgoing` refs, and process participation payload.

- Added direct `query -> context` workflow support by including symbol `id` in `query.process_symbols`.

- Added next-step hints for `query`, `context`, and `cypher` tool responses so the Go MCP workflow
  continues toward deeper context, impact review, or schema lookup.

- Expanded `TestServeCallToolsQueryAndCypher` to cover `context` happy path, categorized outgoing
  `calls`, process participation, and context next-step hint behavior.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before MCP context
  tests.

- Re-ran `go test ./internal/mcp ./internal/httpapi -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Re-ran `avmatrix analyze --force` before MCP context smoke and staged-scope detection.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe serve --host 127.0.0.1 --port 48719`
  with HTTP MCP `tools/list`, `tools/call query`, and `tools/call context` by passing a UID from
  `query.process_symbols`; response contained `status: found`, `symbol`, `incoming`, `outgoing`,
  `processes`, and the context impact next-step hint.

- Added Go MCP `impact` tool to stdio/HTTP discovery. The tool resolves by `target` or
  `target_uid`, honors `direction`, `file_path`, `kind`, `maxDepth`, `relationTypes`,
  `includeTests`, and `minConfidence`, traverses the Go graph snapshot with bounded BFS, and
  returns the baseline-shaped `target`, `direction`, `impactedCount`, `risk`, `summary`,
  `affected_processes`, `affected_modules`, and `byDepth` payload.

- Added direct `impact` fixture coverage for upstream blast radius, depth grouping, affected
  process/module enrichment, and impact next-step hint behavior.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before MCP impact
  tests.

- Re-ran `go test ./internal/mcp ./internal/httpapi -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Re-ran `avmatrix analyze --force` before MCP impact smoke and staged-scope detection.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe serve --host 127.0.0.1 --port 48720`
  with HTTP MCP `tools/list`, `tools/call query`, and `tools/call impact` by passing a UID from
  `query.process_symbols`; response contained `target`, `direction`, `impactedCount`, `risk`,
  `summary`, `affected_processes`, `affected_modules`, `byDepth`, and the impact next-step hint.

- Added Go MCP `detect_changes` tool to stdio/HTTP discovery. The tool runs `git diff -U0` for
  `unstaged`, `staged`, `all`, and `compare` scopes, parses hunk ranges, maps changed lines to Go
  graph symbols by `filePath/startLine/endLine`, and returns the baseline-shaped `summary`,
  `changed_symbols`, and `affected_processes` payload.

- Added temp-git fixture coverage for `detect_changes(scope="unstaged")`, proving
  `git diff -> changed symbol -> affected process` through a real repository diff.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before MCP
  `detect_changes` tests.

- Re-ran `go test ./internal/mcp ./internal/httpapi -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Re-ran `avmatrix analyze --force` before MCP `detect_changes` smoke and staged-scope detection.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe serve --host 127.0.0.1 --port 48721`
  with HTTP MCP `tools/list` and `tools/call detect_changes` against the current unstaged working
  tree; response contained `summary`, `changed_symbols`, `affected_processes`, and the
  `detect_changes` next-step hint.

- Added Go MCP `rename` tool to stdio/HTTP discovery. The tool resolves by `symbol_name` or
  `symbol_uid`, supports `file_path`, `new_name`, and `dry_run`, collects graph-confidence edits
  for the definition and incoming graph refs, blocks path traversal outside the selected repo, and
  returns the baseline-shaped `status`, `old_name`, `new_name`, `files_affected`, `total_edits`,
  `graph_edits`, `text_search_edits`, `changes`, and `applied` payload.

- Added direct dry-run fixture coverage for `rename`, proving definition/ref edit preview and the
  rename next-step hint without modifying files.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before MCP rename
  tests.

- Re-ran `go test ./internal/mcp ./internal/httpapi -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Re-ran `avmatrix analyze --force` before MCP rename smoke and staged-scope detection.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe serve --host 127.0.0.1 --port 48722`
  with HTTP MCP `tools/list`, `tools/call query`, and `tools/call rename` in `dry_run` mode using a
  UID from `query.process_symbols`; response contained `status`, `old_name`, `new_name`,
  `files_affected`, `total_edits`, `changes`, `applied: false`, and the rename next-step hint.

- Added Go MCP `route_map` and `tool_map` tools to stdio/HTTP discovery. `route_map` reads Route
  nodes from the Go graph snapshot, attaches handler file paths, middleware arrays, FETCHES
  consumers, accessed-key/fetch-count metadata from relationship reasons, and linked
  `ENTRY_POINT_OF` process names. `tool_map` reads Tool nodes, handler files, descriptions, and
  linked flow names from the same graph snapshot.

- Updated repo context/setup resources so the available tool list includes `route_map` and
  `tool_map`.

- Added direct fixture coverage for `route_map` and `tool_map`, including filtered lookup,
  consumers, middleware, linked flows, and next-step hints.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before MCP
  route/tool map tests.

- Re-ran `go test ./internal/mcp ./internal/httpapi -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe serve --host 127.0.0.1 --port 48724`
  with HTTP MCP `tools/call route_map` and `tools/call tool_map`; responses contained `routes`,
  `tools`, `route_map`, and the route/tool next-step hints.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe mcp` over stdio with
  `tools/call tool_map`; response was framed with `Content-Length` and contained `tools` plus
  `route_map`.

- Added Go MCP `shape_check` and `api_impact` tools to stdio/HTTP discovery. Both tools read Route
  nodes from the Go graph snapshot, including `responseKeys`, `errorKeys`, `middleware`, FETCHES
  consumers, relationship reason metadata, and `ENTRY_POINT_OF` flow links.

- `shape_check` now reports route response-shape data, consumer accessed keys, mismatch status,
  mismatch confidence, multi-fetch attribution notes, and summary mismatch counts.

- `api_impact` now accepts `route` or `file`, returns a single route payload for one match or
  `{ routes, total }` for multiple matches, includes response shape, middleware, consumers,
  mismatches, execution flows, and direct-consumer risk summary.

- Normalized route/tool/API payload slices so `middleware`, `consumers`, `flows`,
  `executionFlows`, and response shape arrays return JSON arrays instead of `null`.

- Updated repo context/setup resources so the available tool list includes `shape_check` and
  `api_impact`.

- Expanded direct fixture coverage for route analysis to prove
  `route_map -> shape_check -> api_impact -> tool_map` against one Route node with response keys,
  error keys, middleware, FETCHES consumer keys, a mismatch, and linked flow names.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before MCP
  shape/API impact tests.

- Re-ran `go test ./internal/mcp ./internal/httpapi -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe serve --host 127.0.0.1 --port 48726`
  with HTTP MCP `tools/call route_map`, `tool_map`, `shape_check`, and `api_impact`; responses
  contained route/tool/API payloads and the next-step hints.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe mcp` over stdio with
  `tools/call api_impact`; response was framed with `Content-Length` and contained the API impact
  next-step hint.

- Added Go `internal/group` read-only support for listing configured groups, loading the
  `group.yaml` contract, parsing `version`, `name`, `description`, `repos`, and manifest `links`,
  reading optional `contracts.json` snapshots, and computing per-member index/contract staleness.

- Added Go MCP `group_list` and `group_status` tools to stdio/HTTP discovery. `group_list` returns
  `{ groups }` when no name is provided or group details when `name` is provided. `group_status`
  returns `group`, `lastSync`, `missingRepos`, and per-repo `indexStale`, `contractsStale`,
  `missing`, and `commitsBehind` fields.

- Updated repo context/setup resources so the available tool list includes `group_list` and
  `group_status`.

- Added direct group config/parser coverage and MCP fixture coverage for list, detail, registered
  member status, contract-stale status, and missing member status.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before group MCP
  tests.

- Re-ran `go test ./internal/group ./internal/mcp ./internal/httpapi -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe serve --host 127.0.0.1 --port 48727`
  with temporary `AVMATRIX_HOME` and HTTP MCP `tools/call group_list` plus
  `tools/call group_status`; responses contained group list/detail/status payloads and next-step
  hints.

- Smoke-tested `avmatrix-launcher\server-bundle\avmatrix.exe mcp` over stdio with temporary
  `AVMATRIX_HOME` and `tools/call group_status`; response was framed with `Content-Length` and
  contained the group status next-step hint.

- Added Go group Contract Registry read support for `contracts.json` payloads, including
  `StoredContract`, `CrossLink`, repo snapshot, missing repo, type filter, repo filter, and
  unmatched-only filtering.

- Added Go group query support over registered group members by resolving `group.yaml` registry
  names, reading each member's Go `graph.json`, ranking matching `Process` nodes, merging
  `_rrf_score` results, and reporting per-repo hit counts plus missing member counts.

- Added Go MCP `group_contracts` and `group_query` tools to stdio/HTTP discovery. Both tools return
  the baseline payload shapes and append next-step hints.

- Updated repo context/setup resources so the available tool list includes `group_contracts` and
  `group_query`.

- Added direct group contract/query coverage and expanded MCP fixture coverage for registry
  filters, unmatched-only output, merged group query results, missing member counts, and next-step
  hints.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before group
  contract/query tests.

- Re-ran `go test ./internal/group ./internal/mcp ./internal/httpapi -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Smoke-tested `go run ./cmd/avmatrix serve --host 127.0.0.1 --port <ephemeral>` with temporary
  `AVMATRIX_HOME` and HTTP MCP `tools/call group_contracts` plus `tools/call group_query`;
  responses contained contract registry payloads and merged query results.

- Smoke-tested `go run ./cmd/avmatrix mcp` over stdio with temporary `AVMATRIX_HOME` and
  `tools/call group_contracts`; response was framed with `Content-Length` and contained the
  contract registry payload.

- Added Go group sync support for the currently implemented graph slice: registered group members
  are resolved from `group.yaml`, member `meta.json` snapshots are recorded, HTTP provider
  contracts are extracted from Route nodes, HTTP consumer contracts are extracted from FETCHES
  edges, manifest links emit deterministic synthetic contracts/cross-links, exact matching writes
  `contracts.json`, and the result reports contract, cross-link, unmatched, and missing repo
  counts.

- Added Go MCP `group_sync` to stdio/HTTP discovery and dispatch. The tool accepts the baseline
  `allowStale`, `verbose`, `exactOnly`, and `skipEmbeddings` flags and returns the baseline count
  payload shape.

- Updated repo context/setup resources so the available tool list includes `group_sync`.

- Added direct group sync coverage and MCP fixture coverage for
  `group_sync -> contracts.json -> group_contracts`, including HTTP provider/consumer exact
  matching and next-step hints.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before group sync
  tests.

- Re-ran `go test ./internal/group ./internal/mcp ./internal/httpapi -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Smoke-tested `go run ./cmd/avmatrix serve --host 127.0.0.1 --port <ephemeral>` with temporary
  `AVMATRIX_HOME` and HTTP MCP `tools/call group_sync` followed by `group_contracts`; responses
  contained `contracts: 2`, `crossLinks: 1`, `unmatched: 0`, and the written registry payload.

- Smoke-tested `go run ./cmd/avmatrix mcp` over stdio with temporary `AVMATRIX_HOME` and
  `tools/call group_sync`; response was framed with `Content-Length` and contained the sync count
  payload.

- Added Go HTTP MCP StreamableHTTP SSE response support for clients that advertise
  `Accept: application/json, text/event-stream`: POST request responses are returned as
  `event: message` frames, valid session GET opens one standalone SSE stream, duplicate GET returns
  conflict, DELETE closes a session with the StreamableHTTP success status, and idle session TTL
  cleanup still closes tracked streams.

- Added HTTP MCP tests for SSE initialize/tools-list responses, single active GET SSE stream
  behavior, duplicate GET conflict, DELETE session close, and existing TTL expiry.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before HTTP MCP
  tests.

- Re-ran `go test ./internal/httpapi ./internal/mcp -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Smoke-tested `go run ./cmd/avmatrix serve --host 127.0.0.1 --port <ephemeral>` with temporary
  `AVMATRIX_HOME` and HTTP MCP StreamableHTTP initialize/tools-list calls; responses used
  `text/event-stream`, preserved `Mcp-Session-Id`, and exposed `group_sync`.

- Added Go MCP repo context stale-index warning parity: context resources now check
  `git rev-list <indexedCommit>..HEAD` for the indexed repo and emit the baseline
  `staleness` hint when the graph snapshot is behind HEAD.

- Added MCP resource coverage with a real temporary git repo where `meta.json` points at the
  initial commit and HEAD is one commit ahead.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before stale-index
  tests.

- Re-ran `go test ./internal/mcp -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Added scoped stdout/stderr guarding around Go MCP stdio message handling. JSON-RPC frames still
  write through the captured MCP output writer, while global `os.Stdout`/`os.Stderr` writes during
  tool/resource handling are redirected away from protocol stdout.

- Added an MCP stdio stress test with a writer that deliberately writes `leaked stdout` through
  global `os.Stdout` while serving an MCP frame; the leak is suppressed and the Content-Length frame
  remains valid.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before stdout-safety
  tests.

- Re-ran `go test ./internal/mcp ./internal/lbugruntime -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Added `internal/mcp/testdata/typescript_baseline_surface.json` plus
  `TestMCPSurfaceMatchesTypeScriptBaselineSnapshot` to lock Go MCP discovery against the local
  TypeScript runtime surface for tool names/order, required baseline input schema properties,
  resources, resource templates, and prompts.

- Aligned Go MCP discovery order with the local TypeScript runtime order and exposed the baseline
  `query` discovery fields `task_context`, `goal`, `max_symbols`, and `include_content`; Go-only
  additive `group_sync` fields remain allowed but baseline fields cannot be dropped.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before MCP snapshot
  tests.

- Re-ran `go test ./internal/mcp -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Rebuilt the local TypeScript runtime with `cd avmatrix && npm run build` before comparing MCP
  behavior.

- Supervisor reject fix batch:
  - Wired MCP `cypher` to the Go LadybugDB read-runner path after read-only validation, with graph
    snapshot adapter retained only when the native runner is unavailable in the current build.
  - Fixed `rename` to use the already resolved repository storage path instead of re-resolving by
    registry name, and made applied edits line-targeted so duplicate text earlier in a file cannot be
    renamed accidentally.
  - Fixed `detect_changes` deletion parsing by using the `--- a/...` path plus old-side hunk ranges
    when `+++ /dev/null` appears, so deleted symbols are reported with `change_type: deleted`.
  - Expanded `context` incoming refs for class/interface targets through constructor incoming refs
    and defining-file incoming refs.
  - Tightened HTTP MCP session creation so POSTs without `Mcp-Session-Id` must be `initialize`, and
    added `Mcp-Protocol-Version` to loopback CORS allowed headers.
  - Added group config shape/default parsing for `detect`, `matching`, and `packages`; validates
    manifest link `type`, `role`, and `contract`; and gates HTTP contract extraction on
    `detect.http`.

- Reject regression tests added:
  `TestServeCallToolCypherUsesReadRunner`,
  `TestServeCallToolRenameUsesResolvedRepoStoragePath`,
  `TestApplyRenameChangesUsesTargetLine`,
  `TestServeCallToolDetectChangesReportsDeletedSymbols`,
  `TestServeCallToolContextExpandsClassIncomingRefs`,
  `TestMCPHTTPRequiresInitializeBeforeSession`,
  `TestMCPHTTPNotificationReturnsAcceptedAfterInitialize`,
  `TestParseConfigRejectsInvalidLinks`, and
  `TestSyncSkipsHTTPContractsWhenDetectHTTPDisabled`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before reject
  regression tests.

- Re-ran `go test ./internal/mcp ./internal/httpapi ./internal/group -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Second supervisor reject fix batch for the remaining Phase 13 MCP runtime blockers:
  - Removed broad fallback `CodeRelation {type: ...}` relationship rows from MCP `cypher`, so the
    default no-native-runner build now fails closed for unsupported relationship predicates instead
    of returning unrelated edges. Supported graph-adapter fallback remains limited to the explicit
    internal shapes already covered by tests.
  - Fixed same-file MCP `rename` by reading precise reference lines from relationship IDs,
    deduplicating edits by file/line/source line, and downgrading no-location fallback matches to
    `text_search` instead of graph confidence.
  - Added `TestServeCallToolCypherFallbackRejectsUnsupportedRelationshipPredicates` and
    `TestServeCallToolRenameUsesReferenceLineForSameFileCalls`.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before the second
  reject regression pass.

- Re-ran `go test ./internal/mcp ./internal/httpapi ./internal/group -count=1`.

- Re-ran `go test ./cmd/... ./internal/... -count=1`.

- Ran stdio + HTTP MCP smoke with `.tmp/phase13_reject_mcp_smoke.mjs` against the launcher-built
  Go executable and an isolated fixture repo/home:
  `PASS phase13 reject MCP smoke: stdio cypher fail-closed, stdio rename precise dry-run, HTTP
  cypher fail-closed, HTTP rename applied precise edits`.

- Phase 13 MCP benchmark classification after the Python provider batch:
  - `route_map`, `context`, `impact`, and HTTP `group_sync` are not immediate Phase 13 correctness
    blockers. The Phase 13 runtime reject fixes are closed, the smoke path proves valid
    contract-shaped MCP behavior, and the benchmark gaps are hot-path performance gaps rather than
    missing runtime work.
  - Optimization remains mandatory and is carried by Phase 15 with fixed priority:
    `P0 route_map`, `P1 context`, `P1 impact`, `P2 HTTP group_sync`, `P3 preserve initialize/query
    wins, smaller `tools/list`, and zero protocol-noise bytes`.
  - Phase 14 provider work may continue after both jump-back items are checked.

### Residual Phase 13 scope

- None for correctness/runtime blockers. MCP graph-context performance remains open under Phase 15
  optimization tracking.

## Phase 14 - Additional Language Providers

### Go provider evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Direct tests:
  `TestExtractGoScopeIR`, `TestExtractGoScopeIRParityFixture`,
  `TestResolveGoGraphParityCounts`, and `TestParseFilesRoutesGoFilesToGoProvider`.

- Fixture ScopeIR fact counts: `scopes=9`, `definitions=14`, `imports=2`, `calls=2`,
  `accesses=2`, `heritage=2`, `typeReferences=9`, `typeBindings=11`.

### Python provider evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./internal/providers/python -count=1` passed.

- `go test ./internal/parser ./internal/analyze ./internal/providers/python -count=1` passed.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Direct tests:
  `TestExtractPythonScopeIR`, `TestExtractPythonScopeIRParityFixture`,
  `TestResolvePythonGraphParityCounts`, `TestParseFilesRoutesPythonFilesToPythonProvider`, and
  `TestPoolRejectsUnsupportedLanguage`.

- Fixture ScopeIR fact counts: `scopes=6`, `definitions=7`, `imports=2`, `calls=3`,
  `accesses=3`, `heritage=1`, `typeReferences=6`, `typeBindings=7`.

- Graph relationship counts: `ACCESSES=2`, `CALLS=2`, `DEFINES=7`, `EXTENDS=1`,
  `HAS_METHOD=2`, `HAS_PROPERTY=1`, `INHERITS=1`, `USES=1`.

- Parser wiring: `.py` files now route through the Python provider, and unsupported-language
  coverage now uses Ruby so the stale Python-is-unsupported expectation is removed.

- Full runtime E2E:
  - Fixture: `.tmp\phase14-python-full-e2e`, indexed with launcher-built
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force` under isolated
    `AVMATRIX_HOME=.tmp\phase14-python-full-e2e-home`.
  - CLI observable graph result: `files=3`, `nodes=19`, `relationships=39`, `communities=3`,
    `processes=1`, with Python `Class`/`Function`, `CALLS`, `INHERITS`, `MEMBER_OF`,
    `ENTRY_POINT_OF`, and `STEP_IN_PROCESS` present in `.avmatrix\graph.json`.
  - Full Web E2E command:
    `npx playwright test --workers=1 --reporter='list,json' --output=..\.tmp\phase14-full-playwright-test-results`
    with Go `serve` on `127.0.0.1:4747`, Vite on `127.0.0.1:5173`, and isolated registry.
  - Result: `32 passed`, `1 skipped`.
  - E2E-driven fix included in the follow-up batch: Web URL `project` now preserves the registry
    name instead of absolute repo path, graph fetch prioritizes the original repo query over
    mocked/fallback `repoPath`, and heartbeat recovery probes the SSE endpoint after disconnect.

### Java provider evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./internal/parser ./internal/providers/java -count=1` passed.

- `go test ./internal/parser ./internal/providers/java ./internal/analyze -count=1` passed.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Direct tests:
  `TestPoolParsesJavaFixture`, `TestExtractJavaScopeIR`,
  `TestExtractJavaScopeIRParityFixture`, `TestResolveJavaGraphParityCounts`, and
  `TestParseFilesRoutesJavaFilesToJavaProvider`.

- Fixture ScopeIR fact counts: `scopes=12`, `definitions=14`, `imports=2`, `calls=4`,
  `accesses=1`, `heritage=2`, `typeReferences=7`, `typeBindings=12`.

- Graph relationship counts: `ACCESSES=1`, `CALLS=3`, `DEFINES=14`, `EXTENDS=1`,
  `HAS_METHOD=6`, `HAS_PROPERTY=1`, `IMPLEMENTS=1`, `INHERITS=2`,
  `METHOD_IMPLEMENTS=1`, `USES=5`.

- Parser/analyze wiring: `.java` files now route through the Java tree-sitter grammar and Java
  provider dispatch.

- Full runtime E2E:
  - Fixture: `.tmp\phase14-java-full-e2e`, indexed with launcher-built
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --benchmark-json` under isolated
    `AVMATRIX_HOME=.tmp\phase14-java-full-e2e-home`.
  - CLI observable graph result: `files=2`, `parsed=1`, `unsupported=1`, `nodes=23`,
    `relationships=49`, `communities=3`, `processes=1`, `dbLoad.fallbackInsertCount=0`.
  - Graph labels present: `Class=4`, `Interface=1`, `Constructor=1`, `Method=5`, `Package=1`,
    `Property=1`, `Variable=1`.
  - Relationship labels present: `CALLS=3`, `USES=5`, `INHERITS=2`, `EXTENDS=1`,
    `IMPLEMENTS=1`, `METHOD_IMPLEMENTS=1`, `MEMBER_OF=7`, `STEP_IN_PROCESS=3`.

### Kotlin provider evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./internal/parser ./internal/providers/kotlin ./internal/analyze -count=1` passed.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Direct tests:
  `TestPoolParsesKotlinFixture`, `TestExtractKotlinScopeIR`,
  `TestExtractKotlinScopeIRParityFixture`, `TestResolveKotlinGraphParityCounts`, and
  `TestParseFilesRoutesKotlinFilesToKotlinProvider`.

- Fixture ScopeIR fact counts: `scopes=11`, `definitions=14`, `imports=2`, `calls=6`,
  `accesses=1`, `heritage=2`, `typeReferences=6`, `typeBindings=12`.

- Graph relationship counts: `ACCESSES=1`, `CALLS=3`, `DEFINES=14`, `EXTENDS=1`,
  `HAS_METHOD=6`, `HAS_PROPERTY=1`, `IMPLEMENTS=1`, `INHERITS=2`,
  `METHOD_IMPLEMENTS=1`, `USES=4`.

- Parser/analyze wiring: `.kt`/`.kts` files now route through the Kotlin tree-sitter grammar and
  Kotlin provider dispatch.

- Full runtime E2E:
  - Fixture: `.tmp\phase14-kotlin-full-e2e`, indexed with launcher-built
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --benchmark-json` under isolated
    `AVMATRIX_HOME=.tmp\phase14-kotlin-full-e2e-home`.
  - CLI observable graph result: `files=2`, `parsed=1`, `unsupported=1`, `nodes=23`,
    `relationships=48`, `communities=3`, `processes=1`, `dbLoad.fallbackInsertCount=0`.
  - Graph labels present: `Class=4`, `Interface=1`, `Constructor=1`, `Method=5`, `Package=1`,
    `Property=1`, `Variable=1`.
  - Relationship labels present: `CALLS=3`, `USES=4`, `INHERITS=2`, `EXTENDS=1`,
    `IMPLEMENTS=1`, `METHOD_IMPLEMENTS=1`, `MEMBER_OF=7`, `STEP_IN_PROCESS=3`.

### C provider evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./internal/parser ./internal/providers/c ./internal/analyze -count=1` passed.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Direct tests:
  `TestPoolParsesCFixture`, `TestExtractCScopeIR`, `TestExtractCScopeIRParityFixture`,
  `TestResolveCGraphParityCounts`, and `TestParseFilesRoutesCFilesToCProvider`.

- Fixture ScopeIR fact counts: `scopes=5`, `definitions=8`, `imports=2`, `calls=3`,
  `accesses=1`, `typeReferences=8`, `typeBindings=8`.

- Graph relationship counts: `ACCESSES=1`, `CALLS=1`, `DEFINES=8`, `HAS_PROPERTY=3`,
  `USES=2`.

- Parser/analyze wiring: `.c` files now route through the C tree-sitter grammar and C provider
  dispatch.

- Full runtime E2E:
  - Fixture: `.tmp\phase14-c-full-e2e`, indexed with launcher-built
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --benchmark-json` under isolated
    `AVMATRIX_HOME=.tmp\phase14-c-full-e2e-home`.
  - CLI observable graph result: `files=2`, `parsed=1`, `unsupported=1`, `nodes=12`,
    `relationships=18`, `communities=1`, `dbLoad.fallbackInsertCount=0`.
  - Graph labels present: `Struct=2`, `Function=2`, `Property=3`, `Variable=1`.
  - Relationship labels present: `CALLS=1`, `ACCESSES=1`, `USES=2`, `HAS_PROPERTY=3`,
    `MEMBER_OF=2`.

### C# provider evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./internal/parser ./internal/providers/csharp ./internal/analyze -count=1` passed.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Direct tests:
  `TestPoolParsesCSharpFixture`, `TestExtractCSharpScopeIR`,
  `TestExtractCSharpScopeIRParityFixture`, `TestResolveCSharpGraphParityCounts`, and
  `TestParseFilesRoutesCSharpFilesToCSharpProvider`.

- Fixture ScopeIR fact counts: `scopes=12`, `definitions=14`, `imports=2`, `calls=6`,
  `accesses=2`, `heritage=2`, `typeReferences=7`, `typeBindings=13`.

- Graph relationship counts: `ACCESSES=2`, `CALLS=3`, `DEFINES=14`, `EXTENDS=1`,
  `HAS_METHOD=6`, `HAS_PROPERTY=1`, `IMPLEMENTS=1`, `INHERITS=2`,
  `METHOD_IMPLEMENTS=1`, `USES=5`.

- Parser/analyze wiring: `.cs` files now route through the C# tree-sitter grammar and C# provider
  dispatch.

- Full runtime E2E:
  - Fixture: `.tmp\phase14-csharp-full-e2e`, indexed with launcher-built
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --benchmark-json` under isolated
    `AVMATRIX_HOME=.tmp\phase14-csharp-full-e2e-home`.
  - CLI observable graph result: `files=2`, `parsed=1`, `unsupported=1`, `nodes=21`,
    `relationships=48`, `communities=3`, `processes=1`, `dbLoad.fallbackInsertCount=0`.
  - Graph labels present: `Class=4`, `Interface=1`, `Constructor=1`, `Method=5`,
    `Package=1`, `Property=1`, `Variable=1`.
  - Relationship labels present: `CALLS=3`, `USES=5`, `INHERITS=2`, `EXTENDS=1`,
    `IMPLEMENTS=1`, `METHOD_IMPLEMENTS=1`, `MEMBER_OF=7`, `STEP_IN_PROCESS=3`.

### C++ provider evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./internal/parser ./internal/providers/cpp ./internal/analyze -count=1` passed.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Direct tests:
  `TestPoolParsesCPPFixture`, `TestExtractCPPScopeIR`,
  `TestExtractCPPScopeIRParityFixture`, `TestResolveCPPGraphParityCounts`, and
  `TestParseFilesRoutesCPPFilesToCPPProvider`.

- Fixture ScopeIR fact counts: `scopes=11`, `definitions=13`, `imports=2`, `calls=2`,
  `accesses=2`, `heritage=2`, `typeReferences=8`, `typeBindings=13`.

- Graph relationship counts: `ACCESSES=1`, `CALLS=1`, `DEFINES=13`, `EXTENDS=2`,
  `HAS_METHOD=5`, `HAS_PROPERTY=2`, `INHERITS=2`, `METHOD_OVERRIDES=1`, `USES=2`.

- Parser/analyze wiring: `.cpp`/`.cc`/`.cxx`/`.h`/`.hpp`/`.hxx`/`.hh` files now route through
  the C++ tree-sitter grammar and C++ provider dispatch.

- Full runtime E2E:
  - Fixture: `.tmp\phase14-cpp-full-e2e`, indexed with launcher-built
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --benchmark-json` under isolated
    `AVMATRIX_HOME=.tmp\phase14-cpp-full-e2e-home`.
  - CLI observable graph result: `files=2`, `parsed=1`, `unsupported=1`, `nodes=18`,
    `relationships=36`, `communities=2`, `processes=1`, `dbLoad.fallbackInsertCount=0`.
  - Graph labels present: `Class=4`, `Constructor=1`, `Method=4`, `Package=1`, `Property=2`,
    `Variable=1`.
  - Relationship labels present: `CALLS=1`, `ACCESSES=1`, `USES=2`, `INHERITS=2`, `EXTENDS=2`,
    `METHOD_OVERRIDES=1`, `HAS_METHOD=5`, `HAS_PROPERTY=2`, `MEMBER_OF=5`.

### Rust provider evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./internal/parser ./internal/providers/rust ./internal/analyze -count=1` passed.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Direct tests:
  `TestPoolParsesRustFixture`, `TestExtractRustScopeIR`,
  `TestExtractRustScopeIRParityFixture`, `TestResolveRustGraphParityCounts`, and
  `TestParseFilesRoutesRustFilesToRustProvider`.

- Fixture ScopeIR fact counts: `scopes=9`, `definitions=12`, `imports=2`, `calls=3`,
  `accesses=1`, `heritage=1`, `typeReferences=9`, `typeBindings=13`.

- Graph relationship counts: `ACCESSES=1`, `CALLS=1`, `DEFINES=12`, `HAS_METHOD=4`,
  `HAS_PROPERTY=3`, `IMPLEMENTS=1`, `INHERITS=1`, `METHOD_IMPLEMENTS=1`, `USES=2`.

- Parser/analyze wiring: `.rs` files now route through the Rust tree-sitter grammar and Rust
  provider dispatch.

- Full runtime E2E:
  - Fixture: `.tmp\phase14-rust-full-e2e`, indexed with launcher-built
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --benchmark-json` under isolated
    `AVMATRIX_HOME=.tmp\phase14-rust-full-e2e-home`.
  - CLI observable graph result: `files=2`, `parsed=1`, `unsupported=1`, `nodes=16`,
    `relationships=30`, `communities=1`, `processes=1`, `dbLoad.fallbackInsertCount=0`.
  - Graph labels present: `Trait=1`, `Struct=2`, `Method=4`, `Package=1`, `Property=3`,
    `Variable=1`.
  - Relationship labels present: `CALLS=1`, `ACCESSES=1`, `USES=2`, `IMPLEMENTS=1`,
    `INHERITS=1`, `METHOD_IMPLEMENTS=1`, `HAS_METHOD=4`, `HAS_PROPERTY=3`, `MEMBER_OF=2`.

### PHP provider evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./internal/parser ./internal/providers/php ./internal/analyze -count=1` passed.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Direct tests:
  `TestPoolParsesPHPFixture`, `TestExtractPHPScopeIR`,
  `TestExtractPHPScopeIRParityFixture`, `TestResolvePHPGraphParityCounts`, and
  `TestParseFilesRoutesPHPFilesToPHPProvider`.

- Fixture ScopeIR fact counts: `scopes=12`, `definitions=15`, `imports=4`, `calls=3`,
  `accesses=3`, `heritage=2`, `typeReferences=9`, `typeBindings=14`.

- Graph relationship counts: `ACCESSES=3`, `CALLS=2`, `DEFINES=15`, `EXTENDS=1`,
  `HAS_METHOD=5`, `HAS_PROPERTY=3`, `IMPLEMENTS=1`, `INHERITS=2`,
  `METHOD_IMPLEMENTS=1`, `USES=5`.

- Parser/analyze wiring: `.php`, `.phtml`, `.php3`, `.php4`, `.php5`, and `.php8` files now route
  through the PHP tree-sitter grammar and PHP provider dispatch.

- Full runtime E2E:
  - Fixture: `.tmp\phase14-php-full-e2e`, indexed with launcher-built
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --benchmark-json` under isolated
    `AVMATRIX_HOME=.tmp\phase14-php-full-e2e-home`.
  - CLI observable graph result: `files=2`, `parsed=1`, `unsupported=1`, `nodes=20`,
    `relationships=46`, `communities=2`, `dbLoad.skipped=true`,
    `dbLoad.fallbackInsertCount=0`.
  - Graph labels present: `Class=4`, `Constructor=1`, `Interface=1`, `Method=4`, `Package=1`,
    `Property=3`, `Variable=1`.
  - Relationship labels present: `CALLS=2`, `ACCESSES=3`, `USES=5`, `INHERITS=2`,
    `IMPLEMENTS=1`, `METHOD_IMPLEMENTS=1`, `HAS_METHOD=5`, `HAS_PROPERTY=3`, `MEMBER_OF=6`.

### Dart provider evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./internal/parser ./internal/providers/dart ./internal/analyze -count=1` passed.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Direct tests:
  `TestPoolParsesDartFixture`, `TestExtractDartScopeIR`,
  `TestExtractDartScopeIRParityFixture`, `TestResolveDartGraphParityCounts`, and
  `TestParseFilesRoutesDartFilesToDartProvider`.

- Fixture ScopeIR fact counts: `scopes=13`, `definitions=16`, `imports=2`, `calls=3`,
  `accesses=2`, `heritage=2`, `typeReferences=8`, `typeBindings=15`.

- Graph relationship counts: `ACCESSES=2`, `CALLS=2`, `DEFINES=16`, `EXTENDS=1`,
  `HAS_METHOD=7`, `HAS_PROPERTY=3`, `IMPLEMENTS=1`, `INHERITS=2`,
  `METHOD_IMPLEMENTS=1`, `USES=4`.

- Parser/analyze wiring: `.dart` files now route through the Dart tree-sitter grammar and Dart
  provider dispatch using the latest checked upstream HEAD with Go bindings.

- Full runtime E2E:
  - Fixture: `.tmp\phase14-dart-full-e2e`, indexed with launcher-built
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --benchmark-json` under isolated
    `AVMATRIX_HOME=.tmp\phase14-dart-full-e2e-home`.
  - CLI observable graph result: `files=2`, `parsed=1`, `unsupported=1`, `nodes=21`,
    `relationships=47`, `communities=2`, `dbLoad.skipped=true`,
    `dbLoad.fallbackInsertCount=0`.
  - Graph labels present: `Class=4`, `Constructor=3`, `Interface=1`, `Method=4`,
    `Property=3`, `Variable=1`.
  - Relationship labels present: `CALLS=2`, `ACCESSES=2`, `USES=4`, `INHERITS=2`,
    `IMPLEMENTS=1`, `METHOD_IMPLEMENTS=1`, `HAS_METHOD=7`, `HAS_PROPERTY=3`, `MEMBER_OF=6`.

### Vue provider evidence

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before full tests.

- `go test ./internal/providers/vue ./internal/analyze -count=1` passed.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Direct tests:
  `TestExtractVueScopeIR`, `TestExtractVueRejectsNonVueLanguage`,
  `TestExtractVueWithoutInlineScriptReturnsEmptyIR`, `TestExtractVueScopeIRParityFixture`,
  `TestResolveVueGraphParityCounts`, and `TestParseFilesRoutesVueFilesToVueProvider`.

- Fixture ScopeIR fact counts: `scopes=7`, `definitions=10`, `imports=1`, `calls=2`,
  `accesses=3`, `heritage=1`, `typeReferences=3`, `typeBindings=10`.

- Graph relationship counts: `ACCESSES=3`, `CALLS=1`, `DEFINES=10`, `HAS_METHOD=3`,
  `HAS_PROPERTY=3`, `IMPLEMENTS=1`, `INHERITS=1`, `USES=6`.

- Parser/analyze wiring: `.vue` files now route to a Vue single-file-component provider. The
  provider extracts the first inline `<script>` block, detects TypeScript via `lang="ts"`, and
  delegates script graph facts to the existing JS/TS provider while preserving Vue file identity.

- Full runtime E2E:
  - Fixture: `.tmp\phase14-vue-full-e2e`, indexed with launcher-built
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --benchmark-json` under isolated
    `AVMATRIX_HOME=.tmp\phase14-vue-full-e2e\.avmatrix-home`.
  - CLI observable graph result: `files=3`, `parsed=2`, `unsupported=1`, `nodes=17`,
    `relationships=40`, `communities=2`, `dbLoad.skipped=true`,
    `dbLoad.fallbackInsertCount=0`.
  - Graph labels present: `Class=2`, `Constructor=1`, `File=3`, `Folder=1`, `Function=1`,
    `Interface=1`, `Method=2`, `Property=3`, `Variable=1`.
  - Relationship labels present: `CALLS=2`, `ACCESSES=3`, `USES=7`, `INHERITS=1`,
    `IMPLEMENTS=1`, `HAS_METHOD=3`, `HAS_PROPERTY=3`, `IMPORTS=1`, `MEMBER_OF=5`.

### Swift provider evidence

- Dependency/version decision:
  `github.com/alex-pinkus/tree-sitter-swift v0.0.0-20260510231341-3d38a39612ba` was checked as the
  newest available Go module, but its Go binding fails to build because `src/parser.c` is absent.
  The batch uses the newest checked buildable Go binding:
  `github.com/flamingoosesoftwareinc/tree-sitter-swift v0.0.0-20260212012612-56ffc4e2dcc9`.

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./internal/parser ./internal/providers/swift ./internal/analyze -count=1` passed.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Direct tests:
  `TestPoolParsesSwiftFixture`, `TestExtractSwiftScopeIR`, `TestExtractSwiftRejectsNonSwiftLanguage`,
  `TestExtractSwiftScopeIRParityFixture`, `TestResolveSwiftGraphParityCounts`, and
  `TestParseFilesRoutesSwiftFilesToSwiftProvider`.

- Fixture ScopeIR fact counts: `scopes=9`, `definitions=12`, `imports=1`, `calls=2`,
  `accesses=2`, `heritage=1`, `typeAnnotations=6`, `typeBindings=11`.

- Graph relationship counts: `ACCESSES=2`, `CALLS=2`, `DEFINES=12`, `HAS_METHOD=5`,
  `HAS_PROPERTY=3`, `IMPLEMENTS=1`, `INHERITS=1`, `METHOD_IMPLEMENTS=1`, `USES=3`.

- Parser/analyze wiring: `.swift` files now route through the Swift tree-sitter grammar and Swift
  provider dispatch.

- Full runtime E2E:
  - Fixture: `.tmp\phase14-swift-full-e2e`, indexed with launcher-built
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --benchmark-json` under isolated
    `AVMATRIX_HOME=.tmp\phase14-swift-full-e2e\.avmatrix-home`.
  - CLI observable graph result: `files=2`, `parsed=1`, `unsupported=1`, `nodes=18`,
    `relationships=38`, `communities=2`, `dbLoad.skipped=true`,
    `dbLoad.fallbackInsertCount=0`.
  - Graph labels present: `Class=2`, `Constructor=1`, `File=2`, `Folder=2`,
    `Interface=1`, `Method=4`, `Property=3`, `Variable=1`.
  - Relationship labels present: `CALLS=2`, `ACCESSES=2`, `USES=3`, `INHERITS=1`,
    `IMPLEMENTS=1`, `METHOD_IMPLEMENTS=1`, `HAS_METHOD=5`, `HAS_PROPERTY=3`, `MEMBER_OF=5`.

### Ruby provider evidence

- Dependency/version decision:
  `github.com/tree-sitter/tree-sitter-ruby v0.23.1` is the latest checked tagged Go module and has a
  buildable Go binding with generated parser sources.

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./internal/parser ./internal/providers/ruby ./internal/analyze -count=1` passed.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Direct tests:
  `TestPoolParsesRubyFixture`, `TestExtractRubyScopeIR`, `TestExtractRubyRejectsNonRubyLanguage`,
  `TestExtractRubyScopeIRParityFixture`, `TestResolveRubyGraphParityCounts`, and
  `TestParseFilesRoutesRubyFilesToRubyProvider`.

- Fixture ScopeIR fact counts: `scopes=10`, `definitions=12`, `imports=1`, `calls=5`,
  `accesses=2`, `heritage=2`.

- Graph relationship counts: `ACCESSES=2`, `CALLS=1`, `DEFINES=12`, `EXTENDS=1`,
  `HAS_METHOD=5`, `HAS_PROPERTY=2`, `INHERITS=1`.

- Parser/analyze wiring: `.rb` files now route through the Ruby tree-sitter grammar and Ruby
  provider dispatch.

- Full runtime E2E:
  - Fixture: `.tmp\phase14-ruby-full-e2e`, indexed with launcher-built
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --benchmark-json` under isolated
    `AVMATRIX_HOME=.tmp\phase14-ruby-full-e2e\.avmatrix-home`.
  - CLI observable graph result: `files=2`, `parsed=1`, `unsupported=1`, `nodes=18`,
    `relationships=30`, `communities=2`, `dbLoad.skipped=true`,
    `dbLoad.fallbackInsertCount=0`.
  - Relationship labels present: `CALLS=1`, `ACCESSES=2`, `INHERITS=1`, `HAS_METHOD=5`,
    `HAS_PROPERTY=2`, plus file/folder/document containment edges from the mixed fixture.

### COBOL/JCL provider evidence

- Runtime/path decision:
  COBOL/JCL remains implemented by the pre-parse `internal/cobol` enrichment phase instead of the
  tree-sitter ScopeIR parser path. The current checked tree-sitter COBOL options do not fit this
  runtime cleanly: `github.com/BloopAI/tree-sitter-cobol` latest has no Go binding, and
  `github.com/madeindigio/go-tree-sitter v0.1.0` provides COBOL through a different tree-sitter Go
  runtime than the project parser pool.

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./internal/cobol ./internal/analyze -count=1` passed.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Direct tests:
  `TestApplyEmitsCobolProgramsCopyPerformCallAndJCLLinks`,
  `TestApplyReturnsZeroForNonMainframeFiles`, and `BenchmarkApplyCobolEnrichment`.

- Fixture graph counts on the full runtime fixture: `Module=2`, `Namespace=1`, `Function=3`,
  `CodeElement=2`, `File=4`, `Folder=3`, `Community=1`.

- Relationship counts on the full runtime fixture: `CALLS=3`, `CONTAINS=12`, `IMPORTS=1`,
  `MEMBER_OF=2`.

- Full runtime E2E:
  - Fixture: `.tmp\phase14-cobol-full-e2e`, indexed with launcher-built
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --benchmark-json` under isolated
    `AVMATRIX_HOME=.tmp\phase14-cobol-full-e2e\.avmatrix-home`.
  - CLI observable graph result: `files=4`, `parsed=0`, `unsupported=4`, `failed=0`, `nodes=16`,
    `relationships=18`, `communities=1`, `dbLoad.skipped=true`,
    `dbLoad.fallbackInsertCount=0`.
  - COBOL metrics present: `programs=2`, `copybooks=1`, `jclJobs=1`, `jclSteps=1`,
    `jclProgramLinks=1`, `performs=1`, `calls=1`, `copies=1`.
  - Parse reports COBOL/JCL files as unsupported by design because the COBOL/JCL graph surface is
    enriched before the parse phase.

### Framework-specific facts evidence

- Runtime/contract decision:
  framework detection is a graph metadata and process-scoring concern, not a separate language
  provider. Go now owns a dedicated `internal/frameworks` boundary for path conventions,
  AST/decorator/annotation patterns, and ScopeIR framework fact annotation.

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests.

- `go test ./internal/frameworks ./internal/analyze ./internal/resolution ./internal/processes -count=1`
  passed.

- `go test ./cmd/... ./internal/... -count=1` passed.

- Direct tests:
  `TestDetectFromPathMatchesFrameworkConventions`,
  `TestDetectFromPathReturnsFalseForUnknownPath`,
  `TestDetectFromASTMatchesFrameworkConventions`,
  `TestAnnotateScopeIRAddsFrameworkFactsFromDefinitionWindow`,
  `TestResolveAnnotatesFrameworkHintProperties`,
  `TestResolveAppliesScopeIRFrameworkFacts`, and
  `TestApplyUsesFrameworkMultiplierWhenOrderingEntryPoints`.

- Runtime wiring:
  - `parseFiles` calls `frameworks.AnnotateScopeIR` after provider extraction so AST/decorator facts
    are available before cross-file binding and resolution.
  - Resolution applies path-based framework hints to file and symbol nodes, then applies ScopeIR
    framework facts to definition nodes.
  - Process detection reads `astFrameworkMultiplier` to rank framework entry points without dropping
    any required graph facts.

- Full runtime E2E:
  - Fixture: `.tmp\phase14-framework-facts-full-e2e-clean`, indexed with launcher-built
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --benchmark-json` under isolated
    `AVMATRIX_HOME=.tmp\phase14-framework-facts-avmatrix-home`.
  - CLI observable graph result: `files=3`, `parsed=3`, `unsupported=0`, `failed=0`,
    `nodes=26`, `relationships=47`.
  - Path framework facts: `PathFrameworkNodes=4`, `frameworkReason=nextjs-api-route`.
  - AST framework facts: `ASTFrameworkNodes=2`, `framework=nestjs`,
    `astFrameworkMultiplier=3.2`.
  - Process evidence: `EntryPointEdges=3`, with process order beginning
    `proc_0_get:Function:app/api/users/route.ts:GET`.

## Phase 16 - Launcher Integration

### Evidence

- Runtime decision:
  the packaged launcher runtime is Go-owned. `AVmatrixLauncher.exe` is built from
  `avmatrix-launcher/src`, `avmatrix-server.exe` is built from `avmatrix-launcher/server-wrapper`,
  and the backend CLI/server binary is `avmatrix-launcher/server-bundle/avmatrix.exe`.

- Server-wrapper runtime:
  `avmatrix-launcher/server-wrapper/main.go` starts
  `avmatrix-launcher/server-bundle/avmatrix.exe serve --host 127.0.0.1 --port 4747`; it no longer
  starts a bundled `node.exe` plus `avmatrix/dist/cli/index.js serve`.

- Direct launcher tests:
  - `cd avmatrix-launcher/src && go test ./...` passed, covering action parsing, Web UI static
    serving/fallback, web-dist validation, and launcher state read/write.
  - `cd avmatrix-launcher/server-wrapper && go test ./...` passed, covering hidden process
    attributes for the packaged backend wrapper.

- Full launcher build:
  `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before launcher
  tests and smoke checks.

- Package contents:
  `avmatrix-launcher/server-bundle` contains `avmatrix.exe` and `avmatrix-server.exe`;
  `avmatrix-launcher/server-bundle/node.exe` is absent.

- Packaged runtime smoke:
  - Started `AVmatrixLauncher.exe`.
  - Backend readiness passed at `http://127.0.0.1:4747/api/info`.
  - Web UI readiness passed at `http://127.0.0.1:5173`.
  - Playwright smoke loaded the packaged UI, clicked the `AVmatrix-GO` repo card, and reached
    status `READY` with `31807 nodes` and `55775 edges`.
  - `AVmatrixLauncher.exe stop` left no packaged `AVmatrixLauncher`, `avmatrix-server`, or
    `avmatrix` processes running.

## Phase 17 - Cutover Criteria

### Evidence

- Full launcher build first:
  `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before Phase 17
  test batches after each code/test edit.

- Runtime Go package gate:
  `go test ./cmd/... ./internal/... -count=1` passed after the Phase 17 heartbeat and CLI-list fixes.
  This covers CLI contracts, HTTP API, MCP, group tooling, search/embed runtime, LadybugDB
  persistence/readback packages, TypeScript/JavaScript provider parity, and the existing language
  provider parity packages.

- Launcher module gates:
  - `cd avmatrix-launcher/src && go test ./... -count=1` passed.
  - `cd avmatrix-launcher/server-wrapper && go test ./... -count=1` passed.

- Web unit gate:
  `cd avmatrix-web && npm test` passed with `39` test files and `296` tests after fixing
  `connectToServer` to download graphs through the canonical `repoPath` returned by `/api/repo`.

- Browser E2E gate:
  - Packaged launcher started backend `http://127.0.0.1:4747` and Web UI
    `http://127.0.0.1:5173`.
  - `cd avmatrix-web && npx playwright test --workers=1` passed with `33/33` tests against the
    packaged launcher.

- Phase 17 runtime fixes discovered by cutover validation:
  - `connectToServer` now fetches `/api/graph` through `repoInfo.repoPath ?? repoInfo.path` before
    falling back to the input repo name, preserving path-first duplicate-name safety.
  - Go `/api/heartbeat` now keeps the SSE stream open, flushes the initial heartbeat, and sends
    periodic heartbeats until client cancellation. This fixed browser reconnect recovery.
  - Go CLI now includes `avmatrix list`, backed by the repo registry store. Packaged command
    evidence: `avmatrix.exe --help`, `avmatrix.exe status`, and `avmatrix.exe list` all ran from
    `avmatrix-launcher\server-bundle\avmatrix.exe`.

- Test harness alignment:
  - `repo-switching` Windows-path E2E now mocks `/api/graph` for the canonical Windows `repoPath`
    so the test verifies the intended request contract rather than falling through to a real
    non-existent repo.
  - Heartbeat reconnect E2E now uses a deterministic route flag before falling back to the real
    heartbeat endpoint.
  - Onboarding terminal prompt E2E uses `.first()` because the UI intentionally renders two command
    prompts.

- Cutover test target note:
  root `go test ./...` is not the Phase 17 gate because it traverses analyzer fixture corpora under
  `avmatrix/test/fixtures` that intentionally include non-buildable Go/C examples. The cutover Go
  runtime gate is `go test ./cmd/... ./internal/... -count=1`.

- Container and CI runtime cutover:
  - `Dockerfile.cli` now builds `cmd/avmatrix` with Go `1.26.3` on Debian slim, auto-resolves the
    latest stable LadybugDB native release, builds with `-tags ladybugdb`, copies the Go `avmatrix`
    binary plus native LadybugDB SO into the runtime image, and contains no Node runtime.
  - `docker build -f Dockerfile.cli -t avmatrix-go-cli-cutover .` passed.
  - Container smoke passed: `docker run --rm avmatrix-go-cli-cutover avmatrix version` returned
    `1.2.1`; `docker exec ... sh -c "command -v node || true"` produced no node path; container
    `/api/info` returned `{"version":"1.2.1","launchContext":"global","nodeVersion":"go1.26.3"}` on
    `http://127.0.0.1:14747/api/info`.
  - `docker-compose.yaml` now uses `/api/info` for the server healthcheck because `/api/heartbeat`
    is the long-lived SSE heartbeat stream.
  - `.github/workflows/ci-e2e.yml` has been ported to the Go backend path: setup Go from `go.mod`,
    prepare latest-stable LadybugDB native runtime, build `.tmp/avmatrix` with `-tags ladybugdb`,
    run Go `analyze --force`, start Go `serve` on `127.0.0.1:4747`, wait on `/api/info`, and run
    Playwright with `--workers=1`. Node `24` is used only for Web UI dependency install, Vite, and
    Playwright.
  - `README.md`, `TESTING.md`, `docs/local-usage.md`, and `.env.example` local runtime
    instructions now run the Go backend directly from `cmd/avmatrix` on `127.0.0.1`; Web UI remains
    the TypeScript/React display layer.

- Phase 17 current-repo large benchmark/parity probe at commit `5d64ece`:
  - Full launcher build was run first and passed.
  - TypeScript baseline command:
    `node avmatrix\dist\cli\index.js analyze . --force --skip-agents-md --no-stats --benchmark-json
    .tmp\phase17-cutover-ts-avmatrix-go.json --benchmark-label ts-phase17-current-repo` passed with
    `28,731` nodes, `51,603` CLI-reported edges, `52,686` graph-snapshot relationships, and
    `150,292.7ms` wall time.
  - Go packaged command:
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze . --force --skip-agents-md --no-stats
    --benchmark-json .tmp\phase17-cutover-go-avmatrix-go.json` passed with `31,829` nodes,
    `55,816` relationships, and `15,668.3ms` total duration.
  - Summary artifact:
    `.tmp\phase17-cutover-parity-summary.json`.
  - Result: `NOT READY`. Go is faster on this run, but graph parity fails on multiple relationship
    and node-label counts, and the packaged Go benchmark skipped DB load because the native
    LadybugDB runner was unavailable.
  - Native DB build probe:
    `go test -tags ladybugdb ./internal/lbugnative -count=1` and
    `go build -tags ladybugdb -trimpath -o .tmp\avmatrix-ladybugdb-smoke.exe .\cmd\avmatrix` both
    failed with `fatal error: lbug.h: No such file or directory`.
  - Dependency check: LadybugDB core release is `0.16.1`, but the Go binding module currently
    exposes versions only through `v0.13.1` via `go list -m -versions github.com/LadybugDB/go-ladybug`.

- Phase 17 native LadybugDB packaging fix:
  - Added `scripts/ensure-ladybug-native.ps1` and `scripts/ensure-ladybug-native.sh`. Default
    version mode is `auto`: the scripts resolve GitHub latest stable release once per UTC day,
    cache the resolved tag under `.tmp/ladybug-native/latest-release.json`, download platform
    assets, and still allow explicit pinning through `AVMATRIX_LADYBUGDB_VERSION` or Docker build
    ARG for rollback/debug.
  - Full launcher build command:
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed after rollback
    testing. Before the build, the local latest cache was lowered to `v0.15.0` with stale
    `checkedDateUtc=2026-05-12` and the current Windows native runtime directory was moved aside;
    the build refreshed the cache to latest `v0.16.1`, downloaded/restored
    `.tmp\ladybug-native\v0.16.1\windows-x86_64`, built `server-bundle\avmatrix.exe` with
    `-tags ladybugdb`, and copied `lbug_shared.dll` beside the binary.
  - Native test command:
    `go test -tags ladybugdb ./internal/lbugnative -count=1` passed with the script-provided
    `CGO_CFLAGS`, `CGO_LDFLAGS`, and `PATH`.
  - Tagged runtime subset:
    `go test -tags ladybugdb ./internal/lbugnative ./internal/analyze ./internal/cli -count=1`
    passed.
  - Default runtime test:
    `go test ./cmd/... ./internal/... -count=1` passed, preserving the non-native fallback build
    surface for developer environments that are not packaging a cutover runtime.
  - Packaged current-repo analyze command:
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze . --force --skip-agents-md --no-stats
    --benchmark-json .tmp\phase17-cutover-go-native-avmatrix-go.json` passed with active DB load:
    `nodeRows=31,549`, `relationshipRows=55,824`, `fallbackInsertCount=0`, and no
    `dbLoad.skipped=true`.
  - Docker command:
    `docker build -f Dockerfile.cli -t avmatrix-go-cli-cutover .` passed after switching the image
    to Debian slim and copying native LadybugDB SO files into `/usr/local/lib`.
  - Container smoke command:
    `docker run --rm avmatrix-go-cli-cutover sh -c 'avmatrix version; ... analyze ...'` passed:
    no `node` command was found, analyze completed, benchmark JSON included `db_load` and `dbLoad`
    metrics with `nodeRows=3`, `relationshipRows=2`, `fallbackInsertCount=0`, and no
    `dbLoad.skipped=true`.

- Phase 17 process parity fix:
  - Impact precheck:
    `avmatrix impact --uid Function:internal/processes/processes.go:Apply#2 --repo AVmatrix-GO
    --direction upstream --depth 2` returned `LOW` risk.
  - Runtime change:
    Go process detection now uses the TypeScript-style dynamic default process budget
    `min(700, max(20, round(symbolCount / 10)))` when no explicit `MaxProcesses` is configured,
    instead of the fixed `75` cap. It also links `Route` and `Tool` nodes from the process entry
    file back to the generated `Process` with `ENTRY_POINT_OF`.
  - Regression tests:
    `go test ./internal/processes -count=1` passed and covers both the dynamic budget and
    Route/Tool process-entry links.
  - Integration checks after full launcher build:
    `go test ./internal/analyze -count=1` passed, and
    `go test ./cmd/... ./internal/... -count=1` passed.
  - Packaged current-repo rerun:
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze . --force --skip-agents-md --no-stats
    --benchmark-json .tmp\phase17-cutover-go-native-processfix.json` passed. Compared with the
    current TypeScript baseline `.tmp\phase17-cutover-ts-current.json`, `Process` improved from
    `75` before this fix to `556` (`TS=659`), `STEP_IN_PROCESS` improved from `293` to `1,954`
    (`TS=2,640`), and `ENTRY_POINT_OF` includes restored Route/Tool links. Remaining process-family
    gap is coupled to the still-open `CALLS` deficit.

- Phase 17 import parity fix:
  - Impact prechecks:
    `avmatrix impact "resolveImports" --repo AVmatrix-GO --direction upstream --depth 2` and
    `avmatrix impact "emitImportEdges" --repo AVmatrix-GO --direction upstream --depth 2` both
    returned `LOW` risk.
  - Runtime change:
    Go resolution now expands local Go package imports to all non-test Go files in the package
    directory before emitting finalized file-level `IMPORTS` edges. This preserves the existing
    single-target resolver for non-Go and relative import paths, and deliberately excludes package
    `_test.go` targets from production import edges.
  - Regression tests:
    `go test ./internal/resolution -count=1` passed and includes
    `TestResolveExpandsGoPackageImportsToPackageFiles`, proving one package import emits `IMPORTS`
    to both package source files and not to the package test file.
  - Integration checks after full launcher build:
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed,
    `go test ./internal/analyze -count=1` passed, and
    `go test ./cmd/... ./internal/... -count=1` passed.
  - Auto-update proof was reconfirmed during this batch: the local LadybugDB latest cache was
    lowered to `v0.15.0` with a stale date, then full launcher build refreshed it to `v0.16.1`
    and rebuilt `server-bundle\avmatrix.exe` with `lbug_shared.dll` beside the binary.
  - Packaged current-repo rerun:
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze . --force --skip-agents-md --no-stats
    --benchmark-json .tmp\phase17-cutover-go-importfix-notests.json` passed. Compared with
    `.tmp\phase17-cutover-ts-current.json`, `IMPORTS` improved from `201` before this fix to
    `2,140` (`TS=2,367`, remaining delta `-227`). `CALLS` remains open at `7,004`
    (`TS=10,555`, delta `-3,551`) and remains the next graph-parity target before the current-repo
    parity gate can be closed.

- Phase 17 CALLS compatibility and DB schema reconciliation:
  - Impact prechecks:
    `avmatrix impact "semanticEdgeKey" --repo AVmatrix-GO --direction upstream --depth 2`
    returned `LOW`; `avmatrix impact "emitRelationship" --repo AVmatrix-GO --direction upstream
    --depth 2` returned `MEDIUM`; `avmatrix impact "resolveCall" --repo AVmatrix-GO --direction
    upstream --depth 2` returned `LOW`; `avmatrix impact "RelationPairs" --repo AVmatrix-GO
    --direction upstream --depth 2` returned `LOW`; `avmatrix impact "NodeTables" --repo
    AVmatrix-GO --direction upstream --depth 2` returned `LOW`.
  - Runtime change:
    Go resolution now keeps distinct semantic `CALLS` edges by call name, falls back from missing
    scope callers to file callers for top-level calls, resolves same-file free/constructor calls,
    resolves imported package/member calls, and uses an arity-compatible global fallback only where
    it does not widen explicit receiver/member calls. The LadybugDB schema now includes the
    runtime-emitted `Package` node table and the relation pairs required by the expanded
    `CALLS`/`USES`/`ACCESSES` edges.
  - Regression tests:
    `go test ./internal/lbugschema ./internal/lbugload ./internal/resolution ./internal/analyze
    -count=1` passed and covers semantic call-edge de-duplication, same-file fallback, imported
    package/member call resolution, arity-safe global call fallback, schema DDL shape, and DB load.
  - Full build and test:
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed, then
    `go test ./cmd/... ./internal/... -count=1` passed.
  - Auto-update proof:
    the local LadybugDB latest cache was manually lowered to `v0.15.0` with stale date
    `2000-01-01`, the cached `v0.16.1\windows-x86_64` runtime directory was removed, and a full
    launcher build refreshed `latest-release.json` to `v0.16.1` on `2026-05-13` and restored both
    `.tmp\ladybug-native\v0.16.1\windows-x86_64\lbug_shared.dll` and
    `avmatrix-launcher\server-bundle\lbug_shared.dll`.
  - Packaged current-repo rerun:
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze . --force --skip-agents-md --no-stats
    --benchmark-json .tmp\phase17-cutover-go-call-compatfix-schema.json` passed with `32,735`
    nodes, `63,933` relationships, `CALLS=8,746`, `IMPORTS=2,140`, `STEP_IN_PROCESS=2,671`,
    `dbLoad.fallbackInsertFailures=0`, and `dbLoad.skippedRelationships=0`. Compared with the
    latest TypeScript diagnostic (`CALLS=10,589`, `IMPORTS=2,370`, `STEP_IN_PROCESS=2,646`),
    `CALLS` and `IMPORTS` remain open graph-parity work while process edges are now close enough to
    require classification instead of another process-specific fix.

- Phase 17 call-return type binding enrichment:
  - Impact prechecks:
    `avmatrix impact "buildWorkspace" --repo AVmatrix-GO --direction upstream --depth 2`
    returned `HIGH`; `avmatrix impact "resolveMember" --repo AVmatrix-GO --direction upstream
    --depth 2` returned `LOW`; `avmatrix impact "lookupTypeBinding" --repo AVmatrix-GO
    --direction upstream --depth 2` returned `LOW`; `avmatrix impact "baseTypeName" --repo
    AVmatrix-GO --direction upstream --depth 2` returned `LOW`.
  - Runtime change:
    resolution now enriches missing variable/const type bindings from a single call inside the
    declaration range when that call resolves through scope/same-file/imported-member paths and has
    an explicit return type. Pointer/slice/variadic type names are normalized before owner lookup,
    so `*Graph` resolves to `Graph`.
  - Regression tests:
    `go test ./internal/resolution ./internal/analyze -count=1` passed and includes
    `TestResolveMemberCallThroughImportedCallReturnBinding`, proving `graph.New()` return type
    enables a later `g.AddNode()` member call to resolve.
  - Full build and test:
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before tests,
    then `go test ./cmd/... ./internal/... -count=1` passed.
  - Packaged current-repo rerun:
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze . --force --skip-agents-md --no-stats
    --benchmark-json .tmp\phase17-cutover-go-call-returntype.json` passed with `32,767` nodes,
    `64,567` relationships, `CALLS=8,906`, `IMPORTS=2,140`, `STEP_IN_PROCESS=2,682`,
    `dbLoad.fallbackInsertFailures=0`, and `dbLoad.skippedRelationships=0`. Compared with the
    previous Go run (`CALLS=8,746`), this closes `160` additional call edges without introducing
    DB fallback load.

- Phase 17 current-repo graph parity classification:
  - Summary artifact:
    `.tmp\phase17-cutover-parity-summary-current.json` compares
    `.tmp\phase17-ts-pipeline-graph-calls-diagnostic.json` with
    `.tmp\phase17-go-graph-call-returntype.json`.
  - Accepted speed benchmark:
    TypeScript diagnostic `133,541ms`; packaged Go with real DB load `61,562.9ms`, approximately
    `2.17x` faster.
  - Accepted DB load:
    `dbLoad.fallbackInsertFailures=0`, `dbLoad.skippedRelationships=0`, `nodeRows=32,767`,
    `relationshipRows=64,567`.
  - CALLS classification:
    remaining deficit is dominated by TypeScript-baseline global/member over-resolution and
    source-label differences. High-signal examples are `testingFataler.Fatalf` (`TS=541`, Go `1`),
    `testingFataler.Helper` (`TS=217`, Go `1`), and `FileContentCache.set` (`TS=307`, Go `1`).
    Go keeps explicit receiver member calls bounded to type/import evidence instead of widening them
    through global fallback.
  - IMPORTS classification:
    remaining deficit is dominated by TypeScript-baseline cross-language/path false positives from
    Go files to TypeScript/shared helper files. Go keeps language-aware package expansion and does
    not create those cross-language import edges.
  - Expanded coverage classification:
    Go intentionally emits broader `USES`, `ACCESSES`, `DEFINES`, `HAS_PROPERTY`, `Variable`,
    `Package`, `TypeAlias`, `Section`, and `Community` coverage from ScopeIR/provider facts. These
    are documented as better coverage, not speed shortcuts.

- Phase 17 TypeScript/Node runtime-authority audit (2026-05-13):
  - Audit trigger:
    remaining Phase 17 cutover gates for normal local runtime and TypeScript contract authority.
    This is correctness/runtime-shape work and must stay in Phase 17; it is not Phase 15
    optimization work.
  - Graph-tool freshness:
    `.avmatrix/meta.json` showed `embeddings=0`, so the repo graph was refreshed with
    `avmatrix analyze --force` before AVmatrix graph queries. The refresh completed with `28,828`
    nodes and `51,858` edges.
  - Graph exploration:
    `query` for TypeScript shared contract/runtime terms surfaced the legacy TypeScript MCP/backend
    class `avmatrix/src/mcp/local/local-backend.ts:LocalBackend`, the Phase 1 contract snapshot
    script, and Go runtime symbols under `internal/cli`/`internal/mcp`. A second query for Node CLI
    runtime terms surfaced Go `NewRootCommand` plus legacy TypeScript CLI tests and scripts, which
    matched the filesystem audit split between Go runtime and legacy TypeScript baseline.
  - Filesystem/package audit:
    `Dockerfile.cli` builds `/usr/local/bin/avmatrix` from `./cmd/avmatrix` and contains no Node
    runtime in the final image. `.github/workflows/ci-e2e.yml` builds the Go backend, runs Go
    `analyze --force`, starts Go `serve`, and uses Node only for Web UI/Playwright. The launcher
    build compiles `cmd/avmatrix` into `avmatrix-launcher/server-bundle/avmatrix.exe`, compiles Go
    launcher/server-wrapper binaries, copies `lbug_shared.dll`, and uses Node only to build
    `avmatrix-web`.
  - Blocker found:
    `Get-Command avmatrix` on the current machine resolves to
    `C:\Users\TAM PC\AppData\Roaming\npm\avmatrix.ps1`, whose script invokes `node` with
    `node_modules/avmatrix/dist/cli/index.js`. `avmatrix/package.json` still declares
    `"bin": { "avmatrix": "dist/cli/index.js" }`, Node/tsx build scripts, TypeScript CLI/server/MCP
    dependencies, and publish/release workflows still publish the TypeScript npm package.
  - Contract-authority blocker found:
    `avmatrix-shared` remains a TypeScript package with source contracts and generated dist output,
    and the legacy `avmatrix/` TypeScript runtime still imports it. This has not yet been reduced to
    browser-only generated glue owned by Go contract definitions.
  - Cutover conclusion:
    the large-repo graph parity and speed gates are accepted, but the two TypeScript/Node cutover
    gates remain open. The next Phase 17 work is `[P17-GO-CLI-DISTRIBUTION-CUTOVER]` and
    `[P17-GO-CONTRACT-AUTHORITY-CUTOVER]`; after those are fixed, return to the same Phase 17 gates
    and re-run the audit before ticking them.

- Phase 17 local npm/PATH CLI distribution cutover (2026-05-13):
  - Phase-jump note:
    this work continued inside Phase 17 because it resolves the normal runtime authority blocker
    discovered by the TypeScript/Node audit. It was not moved to Phase 15 because it is correctness
    and cutover shape, not performance optimization.
  - Graph freshness and impact:
    the graph was refreshed before graph-based work with
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force`; the Go analyzer completed for
    `F:\AVmatrix-GO` with `scanned=1208`, `parsed=1036`, `unsupported=172`, `failed=0`,
    `nodes=32769`, and `relationships=64569`. Distribution config/script targets were not graph
    symbols, so they were validated by build/package/test/E2E. Hook symbol impact was LOW:
    `resolveCliPath` and `runAVmatrixCli` each had `handlePreToolUse` as the direct caller.
  - Implementation:
    `avmatrix/package.json` now maps `bin.avmatrix` to `bin/avmatrix.exe` and includes `bin` in the
    package files. `avmatrix/scripts/build.js` still builds the legacy TypeScript baseline for
    comparison/tests, then runs `scripts/build-go-runtime.cjs`. The new Go runtime build script
    builds `cmd/avmatrix` with `-tags ladybugdb`, resolves/copies the LadybugDB native runtime next
    to the package binary, and writes `bin/avmatrix-runtime.json` with the platform metadata.
    `postinstall` also runs the Go runtime builder so local package installs produce the native bin.
    The Claude hook now resolves the package Go binary first and spawns it directly instead of
    running `node dist/cli/index.js`.
  - Native runtime packaging:
    `scripts/ensure-ladybug-native.sh` now supports Linux x86_64, Linux aarch64, macOS x86_64, and
    macOS arm64 LadybugDB native assets instead of only Linux x86_64. The Windows path continues to
    use the PowerShell resolver.
  - Validation order and results:
    full launcher build was run first with
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` and passed. Then
    `cd avmatrix && npm run build` passed, `go test ./cmd/... ./internal/... -count=1` passed, and
    `cd avmatrix && npx vitest run test/unit/package-bin.test.ts test/unit/hooks.test.ts` passed
    `74` tests across `2` files. Browser E2E was run against
    `avmatrix\bin\avmatrix.exe serve --host 127.0.0.1 --port 4747`; `/api/info` returned
    `nodeVersion="go1.26.3"` and `cd avmatrix-web && npx playwright test --workers=1` passed
    `33/33`.
  - Package/runtime verification:
    `npm link` in `avmatrix/` rewrote the global PATH shim so
    `C:\Users\TAM PC\AppData\Roaming\npm\avmatrix.ps1` invokes
    `node_modules/avmatrix/bin/avmatrix.exe` instead of
    `node_modules/avmatrix/dist/cli/index.js`. `avmatrix --help` from PATH prints the Go CLI help.
    `npm publish --dry-run` passed and the dry-run tarball included `bin/avmatrix.exe`,
    `bin/lbug_shared.dll`, and `bin/avmatrix-runtime.json`.
  - Benchmark:
    old PATH npm shim `avmatrix --help` median was `133.7ms` and average was `150.8ms`; direct
    packaged Go `avmatrix.exe --help` median was `25.7ms`. After cutover and `npm link`, PATH
    `avmatrix --help` median was `46.5ms` and average was `57.5ms`; direct package
    `avmatrix/bin/avmatrix.exe --help` median was `29.6ms`. The PATH command is now about `2.9x`
    faster than the old Node shim median.
  - Cutover conclusion:
    local developer/PATH distribution no longer requires the TypeScript CLI runtime and the local
    npm shim no longer treats `dist/cli/index.js` as runtime authority. The broader
    `[P17-GO-CLI-DISTRIBUTION-CUTOVER]` gate remains open because the Windows dry-run tarball proves
    the current local package build but does not yet prove portable npm install behavior for
    Windows, Linux, and macOS from a single published artifact or platform-specific package/download
    strategy.

- Phase 17 portable npm source-build distribution cutover (2026-05-13):
  - Phase-jump note:
    this work stayed in Phase 17 because it closes the npm install/publish side of the normal CLI
    runtime cutover. It is not Phase 15 optimization work. The test was designed around the real
    goal: an installed npm package must select/build the correct Go runtime for the consumer machine
    instead of reusing the publisher machine's TypeScript CLI or native binary.
  - Implementation:
    `avmatrix/scripts/prepare-go-source-package.cjs` copies a minimal Go source package into
    `avmatrix/go-src` during `prepack`: `go.mod`, `go.sum`, non-test `.go` files under `cmd/` and
    `internal/`, plus the LadybugDB native resolver scripts. `clean-go-source-package.cjs` removes
    `go-src` during `postpack` so the working tree does not retain generated
    source. `avmatrix/package.json` now includes `go-src` in package files and runs
    `prepack -> build + prepare-go-source-package`, then `postpack -> clean-go-source-package`.
    `build-go-runtime.cjs` now resolves source roots in order: repo root Go source, packaged
    `go-src`, then same-platform packaged runtime fallback. The package install path therefore
    builds `cmd/avmatrix` from `node_modules/avmatrix/go-src` when installed outside the repo.
  - Native resolver fix:
    the tarball install test exposed a real Windows bug: `scripts/ensure-ladybug-native.ps1` wrote
    `latest-release.json` before creating the output root when invoked from packaged `go-src`.
    The resolver now creates the output root before reading/writing the latest-release cache.
  - Validation order and results:
    after the resolver fix, full launcher build was rerun first with
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` and passed in about
    `39.9s`. Then `cd avmatrix && npm run build` passed, `go test ./cmd/... ./internal/... -count=1`
    passed, and `cd avmatrix && npx vitest run test/unit/package-bin.test.ts test/unit/hooks.test.ts`
    passed `74` tests across `2` files.
  - Portable tarball install test:
    a real package tarball was generated with `npm pack`; it included `212` `go-src` files plus the
    source manifest and `3` package `bin` files. The tarball was installed into
    `.tmp\npm-portable-install` as a consumer project outside the repo source path. During package
    `postinstall`, `build-go-runtime` logged `Go source root:
    F:\AVmatrix-GO\.tmp\npm-portable-install\node_modules\avmatrix\go-src`, resolved LadybugDB
    native under that packaged source tree, wrote the installed package binary, and the installed
    `bin/avmatrix-runtime.json` reported `source="go-src"`, `platform="win32"`, `arch="x64"`.
    `node_modules\.bin\avmatrix.cmd --help` returned
    `AVmatrix local CLI and MCP server`. Optional `tree-sitter-swift` native build still emitted the
    known optional node-gyp failure on Windows, but npm install exited successfully and reported
    `found 0 vulnerabilities`.
  - Browser E2E:
    Playwright E2E was run again after the full build and package validations with the Go backend
    started from `avmatrix\bin\avmatrix.exe serve --host 127.0.0.1 --port 4747` and Vite on
    `127.0.0.1:5173`. The run exited `0`; `avmatrix-web/test-results/.last-run.json` reported
    `status="passed"`, and `npx playwright test --list` listed `33` tests in `6` files.
  - Publish dry-run:
    `cd avmatrix && npm publish --dry-run --loglevel=error` passed. The lifecycle ran `prepack`,
    copied `211` files to `go-src`, ran `prepare`, then `postpack` removed
    `F:\AVmatrix-GO\avmatrix\go-src`.
  - Cutover conclusion:
    portable npm source-build selection is closed for Phase 17: an installed tarball can build the
    Go runtime from packaged Go source without falling back to TypeScript runtime authority. The
    parent `[P17-GO-CLI-DISTRIBUTION-CUTOVER]` remains open because a follow-up command-surface
    audit found TypeScript-only CLI commands that still need Go ports or explicit baseline/test-only
    quarantine before normal local CLI cutover is complete.
  - Command-surface blocker found:
    current Go CLI help exposes `analyze`, `list`, `mcp`, `serve`, `status`, `version`, `wiki`, and
    `wiki-mode`. The legacy TypeScript CLI directory still contains command files for `benchmark`,
    `clean`, `group`, `index-repo`/index flows, `augment`, `setup`, `skill-gen`, `tool`, and
    AI-context support. This is the next Phase 17 blocker because distribution now reaches the Go
    binary, but normal local CLI parity is not fully true until required commands are ported or
    deliberately retired/quarantined.

- Phase 17 direct graph-tool CLI command surface batch (2026-05-13):
  - Phase-jump note:
    this work stayed in Phase 17 because direct CLI graph-tool commands are normal local runtime
    authority for the cutover. It is not Phase 15 optimization work.
  - Graph freshness and impact:
    the graph was refreshed before graph-based work with `avmatrix analyze --force`; the run
    completed with `scanned=1211`, `parsed=1039`, `unsupported=172`, `failed=0`, `nodes=32820`, and
    `relationships=64693`. `NewRootCommand` impact was LOW with direct caller
    `cmd/avmatrix/main.go:main`. The graph reported HIGH risk for existing `newAnalyzeCommand` and
    `newMCPCommand` through root CLI execution flows; the batch was limited to adding sibling
    command wrappers and did not change analyze/MCP behavior.
  - Implementation:
    the Go root command now exposes `augment`, `query`, `context`, `impact`, `cypher`, and
    `detect-changes`. The wrappers call the in-process Go MCP `tools/call` handler instead of
    shelling through Node or reimplementing tool logic. Compatibility flags were added for repo
    selection, task context, goal, limit, UID/file disambiguation, content inclusion, direction,
    depth, include-tests, diff scope, and base ref. `augment` now uses the current working directory
    as the repo hint, matching the legacy TypeScript command's `process.cwd()` behavior, and still
    swallows graph lookup errors so hook usage is non-fatal.
  - Regression tests:
    `internal/cli/command_test.go` now verifies root help includes the new direct tool commands,
    direct tool help exposes the compatibility flags, and `augment` remains a no-op for short
    patterns.
  - Validation order and results:
    after the final `augment` fix, full launcher build was rerun first with
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` and passed in
    `38,947.9ms`. Then `cd avmatrix && npm run build` passed in `32,498.7ms`, and
    `go test ./cmd/... ./internal/... -count=1` passed in `30,142.1ms`.
  - Direct CLI smoke/benchmark:
    using `avmatrix\bin\avmatrix.exe`, all new direct graph-tool commands exited `0`:
    `query "CLI command surface" --repo F:\AVmatrix-GO --limit 1`,
    `context NewRootCommand --repo F:\AVmatrix-GO --file internal/cli/command.go`,
    `impact NewRootCommand --repo F:\AVmatrix-GO --direction upstream --depth 1`,
    `cypher "MATCH (n:Function) RETURN n.id AS id, n.name AS name, n.filePath AS path LIMIT 1"
    --repo F:\AVmatrix-GO`, `detect-changes --scope unstaged --repo F:\AVmatrix-GO`, and
    `augment NewRootCommand`. `augment` returned `5151` characters of graph context after the cwd
    repo hint fix.
  - Browser E2E:
    after the full build and command validation, the Go backend was started from
    `avmatrix\bin\avmatrix.exe serve --host 127.0.0.1 --port 4747` and the Web UI was served on
    `127.0.0.1:5173`. `cd avmatrix-web && E2E=1 npx playwright test --workers=1` exited `0` in
    `219,001.4ms`; `avmatrix-web/test-results/.last-run.json` reported `status="passed"`, and
    `npx playwright test --list` listed `33` tests in `6` files.
  - Cutover conclusion:
    the direct graph-tool CLI command batch is closed for Phase 17, but the parent
    `[P17-GO-CLI-COMMAND-SURFACE-CUTOVER]` remained open after this batch. At this point the
    remaining command-surface work still included `group`, `clean`, `index`, `benchmark-compare`,
    `setup`, and analyze flag parity or explicit quarantine/retirement decisions for non-cutover
    commands such as legacy skill/tool generation; the admin/analyze batch below closes the
    `clean`/`index`/`benchmark-compare`/analyze-flag portion.

- Phase 17 local admin/analyze CLI command surface batch (2026-05-13):
  - Phase-jump note:
    this work stayed in Phase 17 because `clean`, `index`, `benchmark-compare`, and analyze flag
    parity are normal local CLI cutover behavior. It is not Phase 15 optimization work.
  - Graph freshness and impact:
    after commit `54f3254`, the graph was refreshed with
    `avmatrix\bin\avmatrix.exe analyze --force --skip-agents-md --no-stats` in `54,730.6ms`
    before graph-based impact checks. Impact was CRITICAL for the edited root/analyze/benchmark
    symbols because they sit on the normal CLI entry path: `NewRootCommand` affected `10`
    processes, `newAnalyzeCommand` affected `29`, `recordAnalyzeResult` affected `34`, and
    `WriteBenchmark` affected `27`. The batch was intentionally kept to command registration,
    admin commands, analyze flag parity, and benchmark compare output.
  - Implementation:
    Go now exposes `clean`, `index`, and `benchmark-compare`. `clean` supports `--force` and
    `--all`, unregisters repos, deletes `.avmatrix`, and preserves `.avmatrix/settings.json`.
    `index` registers an existing `.avmatrix`/LadybugDB index, supports `--force` and
    `--allow-non-git`, writes a minimal meta file when forced, and appends `.avmatrix/` to
    `.gitignore` for Git repos. `benchmark-compare` compares Go metrics JSON, TypeScript
    schema-version benchmark snapshots, and graph snapshot count shapes, with text and `--json`
    output. `analyze` now accepts `--skip-git`, `--skip-compatibility-cross-file`,
    `--benchmark-label`, `--name`, `--allow-duplicate-name`, and `--verbose`; non-Git folders now
    require `--skip-git` at the CLI layer, matching the legacy TypeScript command contract.
  - Regression tests:
    Go CLI tests now cover root help for the new commands, analyze help for the remaining flags,
    benchmark label emission, non-Git analyze rejection without `--skip-git`, clean confirmation
    and settings preservation, index registration/gitignore behavior, and benchmark compare text
    output.
  - Validation order and results:
    full launcher build was run first with
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` and passed in
    `37,419.6ms`. Then `cd avmatrix && npm run build` passed in `31,215ms`, and
    `go test ./cmd/... ./internal/... -count=1` passed in `29,775.5ms`.
  - Admin/analyze CLI smoke/benchmark:
    using `avmatrix\bin\avmatrix.exe` with a temp `AVMATRIX_HOME` and temp repos, all commands
    exited `0`: root help `58.4ms`; `analyze --help` `34.2ms`; non-Git
    `analyze --skip-git --skip-compatibility-cross-file --benchmark-label --name` `2,690.2ms`;
    `index <temp git repo with .avmatrix/lbug>` `164ms`; `clean --force` on a temp indexed repo
    `33.2ms`; `benchmark-compare` `44.8ms`; and `benchmark-compare --json` `35.9ms`.
  - Browser E2E:
    after the full build and command validation, the Go backend was started from
    `avmatrix\bin\avmatrix.exe serve --host 127.0.0.1 --port 4747`. `cd avmatrix-web && E2E=1
    npx playwright test --workers=1` exited `0` in `198,151.4ms`;
    `avmatrix-web/test-results/.last-run.json` reported `status="passed"`, and
    `npx playwright test --list` listed `33` tests in `6` files.
  - Cutover conclusion:
    the local admin/analyze command batch is closed for Phase 17. The parent
    `[P17-GO-CLI-COMMAND-SURFACE-CUTOVER]` remained open after this batch for `group`, `setup`, and
    explicit quarantine/retirement decisions for legacy `skill-gen`, `tool`, and AI-context CLI
    files; the group batch below closes the `group` portion.

- Phase 17 group CLI command surface batch (2026-05-13):
  - Phase-jump note:
    this work stayed in Phase 17 because repository group commands are normal cross-repo CLI
    runtime/cutover behavior. It is not Phase 15 optimization work.
  - Graph freshness and impact:
    after commit `c7c278a`, the graph was refreshed with
    `avmatrix\bin\avmatrix.exe analyze --force --skip-agents-md --no-stats` in `58,846.9ms`.
    `NewRootCommand` impact remained CRITICAL because the root CLI fans out to normal runtime
    commands; the impact report showed `11` affected processes. The batch was limited to adding the
    `group` sibling command and tests, with no analyze/admin behavior changes.
  - Implementation:
    Go now exposes `group create`, `group add`, `group remove`, `group list`, `group status`,
    `group sync`, `group query`, and `group contracts`. The commands use the existing Go
    `internal/group` storage/status/sync/query/contracts services, write `group.yaml`, support the
    CLI contract flags (`--force`, `--skip-embeddings`, `--exact-only`, `--allow-stale`,
    `--verbose`, `--json`, `--subgroup`, `--limit`, `--type`, `--repo`, `--unmatched`), and format
    text or JSON output directly from Go.
  - Regression tests:
    Go CLI tests now cover group help, create/add/list/remove config management, and a full
    temp-fixture group sync/query/contracts flow using registered temp repos and graph snapshots.
  - Validation order and results:
    full launcher build was run first with
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` and passed in
    `56,230ms`. Then `cd avmatrix && npm run build` passed in `34,282.8ms`, and
    `go test ./cmd/... ./internal/... -count=1` passed in `32,754.3ms`.
  - Group CLI smoke/benchmark:
    using `avmatrix\bin\avmatrix.exe` with a temp `AVMATRIX_HOME` and temp indexed repos, all group
    commands exited `0`: `group --help` `28.8ms`; `group create fixture` `30.9ms`;
    `group add` backend `29.1ms`; `group add` frontend `29.1ms`; `group list` `29.7ms`;
    `group list fixture` `28.1ms`; `group status fixture` `118.5ms`; `group sync fixture --json`
    `47.3ms`; `group query fixture UserFlow --limit 2` `30.3ms`;
    `group contracts fixture --json` `29.4ms`; and `group remove fixture app/frontend` `29ms`.
  - Browser E2E:
    after the full build and command validation, the Go backend was started from
    `avmatrix\bin\avmatrix.exe serve --host 127.0.0.1 --port 4747`. `cd avmatrix-web && E2E=1
    npx playwright test --workers=1` exited `0` in `201,799.8ms`;
    `avmatrix-web/test-results/.last-run.json` reported `status="passed"`, and
    `npx playwright test --list` listed `33` tests in `6` files.
  - Cutover conclusion:
    the group command batch is closed for Phase 17. The parent
    `[P17-GO-CLI-COMMAND-SURFACE-CUTOVER]` remains open only for `setup` and explicit
    quarantine/retirement decisions for legacy `skill-gen`, `tool`, and AI-context CLI files.

- Phase 17 setup/quarantine CLI command surface batch (2026-05-13):
  - Phase-jump note:
    this work stayed in Phase 17 because `avmatrix setup`, editor MCP config, global skill
    installation, Claude hook installation, direct tool command authority, and AI-context/skill
    generation ownership are normal local runtime/cutover behavior. It is not Phase 15
    optimization work.
  - Graph freshness and impact:
    after commit `e0e4283`, the graph was refreshed with
    `avmatrix\bin\avmatrix.exe analyze --force --skip-agents-md --no-stats` in `83,500.2ms`.
    `NewRootCommand` impact remained CRITICAL because the root CLI fans out to normal runtime
    commands; the impact report showed `11` affected processes. A follow-up impact check on the
    path-normalization test assertion (`TestIndexRegistersExistingIndex`) was LOW with no affected
    processes.
  - Implementation:
    Go now exposes `avmatrix setup`. The command creates the global AVmatrix directory, writes MCP
    config for Cursor (`~/.cursor/mcp.json`), Claude Code (`~/.claude.json`), OpenCode
    (`~/.config/opencode/opencode.json`), and Codex (`codex mcp add` with fallback
    `~/.codex/config.toml`), installs packaged skills into supported editor skill directories, and
    installs the bundled Claude hook plus `settings.json` hook entries when `~/.claude` exists.
    Legacy `tool.ts` is quarantined as baseline/test-only because Go direct tool commands already
    own `query`, `context`, `impact`, `cypher`, `detect-changes`, and `augment`. Legacy
    `skill-gen.ts` and `ai-context.ts` are quarantined as baseline/test-only because Go
    `analyze --skills` and `internal/aicontext` own normal runtime AI-context/skill generation.
  - Regression tests:
    Go CLI tests now cover root help showing `setup`, setup writing editor MCP configs, preserving
    existing Claude config keys, installing Cursor/Claude/OpenCode/Codex skills from packaged
    content, installing Claude hooks/settings, creating the configured global AVmatrix directory,
    Codex TOML fallback, and repeated setup idempotency. The Windows short-path/long-path index
    assertion was normalized to compare equivalent paths without changing runtime behavior.
  - Validation order and results:
    after the test assertion fix, full launcher build was rerun first with
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` and passed in
    `33,954.6ms`. Then `cd avmatrix && npm run build` passed in `14,970.8ms`, and
    `go test ./cmd/... ./internal/... -count=1` passed in `23,659.4ms`.
  - Setup CLI smoke/benchmark:
    using `avmatrix\bin\avmatrix.exe` with temp `HOME`/`USERPROFILE`/`AVMATRIX_HOME` and an empty
    `PATH`, all setup commands exited `0`: `setup --help` `1,450.2ms`; first `setup` run
    `206.2ms`; repeated `setup` run `78.2ms`. Required temp artifacts were present for Cursor,
    Claude Code, OpenCode, Codex fallback TOML, editor skills, Claude hook, and Claude settings.
    The Codex fallback contained exactly one `[mcp_servers.avmatrix]` section after two runs.
  - Browser E2E:
    after the full build and command validation, the Go backend was started from
    `avmatrix\bin\avmatrix.exe serve --host 127.0.0.1 --port 4747`. `cd avmatrix-web && E2E=1
    npx playwright test --workers=1` exited `0` in `201,707.1ms`;
    `avmatrix-web/test-results/.last-run.json` reported `status="passed"`, and
    `npx playwright test --list` listed `33` tests in `6` files.
  - Cutover conclusion:
    `[P17-GO-CLI-COMMAND-SURFACE-CUTOVER]` is closed. This does not close
    `[P17-GO-CLI-DISTRIBUTION-CUTOVER]` or the normal-runtime no-Node gate yet; the next Phase 17
    step is a fresh TypeScript/Node audit after this batch.

- Phase 17 post-command-surface TypeScript/Node runtime audit (2026-05-13):
  - Phase-jump note:
    this work stayed in Phase 17 because it verifies runtime authority and cutover shape after the
    command-surface gate closed. It is not Phase 15 optimization work.
  - Graph freshness and impact:
    after commit `3f8e5dc`, the graph was refreshed with
    `avmatrix\bin\avmatrix.exe analyze --force --skip-agents-md --no-stats` in `55,369.8ms`.
    `OnboardingGuide` impact was LOW: one direct file-level use from `DropZone.tsx` and one
    downstream import from `App.tsx`, with no affected processes.
  - Implementation:
    `avmatrix/package.json` now routes `npm run serve` to `bin/avmatrix.exe serve` and `npm run
    dev` to `go run ../cmd/avmatrix`. The old TypeScript watch entrypoint is retained only as
    `dev:ts-baseline`, making it explicit baseline/dev material rather than normal runtime
    authority. The Web onboarding command now shows `avmatrix serve` and no longer shows
    `cd avmatrix && npm run serve` or global npm install fallbacks.
  - Runtime audit:
    the local PowerShell PATH shim invokes `node_modules/avmatrix/bin/avmatrix.exe`. Runtime-file
    search found no `tsx src/cli/index.ts serve`, no `"serve": "tsx`, no default `"dev": "tsx
    watch src/cli/index.ts`, and no Web onboarding runtime command `cd avmatrix && npm run serve`;
    the only remaining literal old onboarding command is a negative unit-test assertion.
  - Validation order and results:
    full launcher build was run first with
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` and passed in
    `35,880.6ms`. Then `cd avmatrix && npm run build` passed in `16,488.3ms`;
    `go test ./cmd/... ./internal/... -count=1` passed in `26,452.4ms`; and
    `cd avmatrix-web && npm test -- OnboardingGuide.local-only.test.tsx` passed in `4,364.1ms`.
  - Runtime benchmark:
    all audited entrypoints exited `0`: PATH `avmatrix --help` `75.7ms`; direct
    `avmatrix\bin\avmatrix.exe --help` `36.6ms`; `cd avmatrix && npm run --silent serve -- --help`
    `369.2ms`; and `cd avmatrix && npm run --silent dev -- --help` `411.9ms`.
  - Browser E2E:
    after the full build and command validation, the Go backend was started from
    `avmatrix\bin\avmatrix.exe serve --host 127.0.0.1 --port 4747`. `cd avmatrix-web && E2E=1
    npx playwright test --workers=1` exited `0` in `218,233.3ms`;
    `avmatrix-web/test-results/.last-run.json` reported `status="passed"`, and
    `npx playwright test --list` listed `33` tests in `6` files.
  - Cutover conclusion:
    `[P17-GO-CLI-DISTRIBUTION-CUTOVER]` and the normal-runtime no-Node gate are closed for normal
    local runtime. Remaining TypeScript under `avmatrix/src`, `avmatrix-shared`, and legacy tests is
    still a separate contract-authority/baseline cleanup concern, so
    `[P17-GO-CONTRACT-AUTHORITY-CUTOVER]` remains open.

- Phase 17 Go-owned Web contract authority cutover (2026-05-13):
  - Phase-jump note:
    this work stayed in Phase 17 because it closes a runtime/contract authority gate. It did not
    enter Phase 15; no MCP/provider performance optimization work was mixed into this slice.
  - Graph freshness and impact:
    after commit `2fac643`, the graph was refreshed with
    `avmatrix\bin\avmatrix.exe analyze --force --skip-agents-md --no-stats` in `66,617.2ms`,
    producing `33,239` nodes and `65,612` relationships. Impact checks before editing showed
    `getSyntaxLanguageFromFilename` LOW with one direct Web function and no affected processes,
    `NODE_TABLES` LOW, and the Vite/Vitest package-resolution helper LOW. `SessionStatusResponse`
    impact was HIGH because Web and legacy TypeScript session surfaces consume that contract; the
    slice did not change the JSON payload shape, only moved the Web type source from
    `avmatrix-shared` to Go-generated browser glue.
  - Implementation:
    Go now owns the Web contract manifest in `internal/contracts`. The generator emits
    `contracts/web-ui/avmatrix-web-contract.schema.json` and
    `avmatrix-web/src/generated/avmatrix-contracts.ts` from Go runtime constants for graph labels,
    relationship types, LadybugDB tables, language detection/syntax mapping, pipeline progress, and
    session contracts. `cmd/generate-web-contracts` provides both generation and `--check`
    freshness validation. Web source/tests now import contract types/constants from the generated
    adapter, and the Web package no longer depends on or aliases `avmatrix-shared`.
  - Web/runtime audit:
    `rg -n "avmatrix-shared" avmatrix-web` returned zero hits in `55.8ms`. The Web package
    lockfile, Vite config, Vitest config, Vercel install command, source imports, and test imports
    no longer point at `avmatrix-shared`.
  - Validation order and results:
    full launcher build was run first with
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` and passed in
    `38,426.7ms`. Then `go test ./cmd/... ./internal/... -count=1` passed in `29,795.3ms`;
    targeted Web contract unit tests passed in `13,013.4ms` with `6` files and `130` tests;
    `go run ./cmd/generate-web-contracts --check` passed in `559.9ms`; `cd avmatrix-web && npm ci`
    passed in `50,004.7ms`; a post-install full launcher build passed in `33,223.8ms`; and
    `cd avmatrix && npm test` passed in `338,908.6ms`.
  - Browser E2E:
    after the post-install full build, the packaged Go backend was started from
    `avmatrix-launcher\server-bundle\avmatrix.exe serve --host 127.0.0.1 --port 4747`. `cd
    avmatrix-web && E2E=1 npx playwright test --workers=1` exited `0` in `205,536.5ms` with
    `33/33` tests passed.
  - Cutover conclusion:
    `[P17-GO-CONTRACT-AUTHORITY-CUTOVER]` is closed for the normal cutover path. The Web UI remains
    TypeScript/React, but its contract source is Go-generated browser glue. Legacy
    `avmatrix-shared` and `avmatrix/src` TypeScript remain quarantined as baseline/dev/test
    material and are not backend/CLI/MCP/analyzer/Web runtime authority.

- Phase 15 MCP route_map hot-path optimization (2026-05-13):
  - Phase-jump note:
    this is the first post-cutover Phase 15 optimization slice after Phase 17 cutover gates closed.
    It targets the fixed-priority P0 `route_map` regression from the Phase 13 MCP benchmark.
  - Graph freshness and impact:
    after commit `60e7ed8`, the packaged Go runtime refreshed the graph with
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --skip-agents-md --no-stats` in
    `58,520ms`, producing `33,355` nodes and `65,816` relationships. Impact checks before editing:
    `graphForResource` LOW with no affected processes; `mcpRouteMapItems` LOW; `mcpRouteAnalysisRecords`
    LOW; `routeMapTool` LOW when addressed by UID; `loadResourceGraphSnapshot` HIGH because it is
    shared by graph resources, cypher fallback, detect_changes, and rename; `NewServer` CRITICAL
    because all MCP stdio/HTTP sessions and direct CLI tool wrappers construct a server. The change
    stayed narrow: no JSON-RPC payload shape changed, and graph snapshot decoding behavior is now
    wrapped by stat-based caching.
  - Implementation:
    `Server` now owns a `resourceGraphCache` that caches decoded `graph.json` by path, file mtime,
    and size. The same cache builds and retains an `mcpRouteIndex` for route records. `route_map`,
    `shape_check`, and `api_impact` read route data from that index instead of rebuilding route
    consumers and flow links by scanning the full graph for each call. Cache invalidation is covered
    by a unit test that rewrites `graph.json` and verifies stale routes disappear.
  - Benchmark result:
    before this slice, current Go `route_map` measured `759.56ms` over stdio and `730.87ms` over
    HTTP. After the cache/index change, packaged Go measured `7.70ms` stdio and `8.80ms` HTTP, below
    the Phase 15 `<50ms` target. Warm-session `context` improved from `782.99ms` to `24.01ms`
    stdio and from `769.33ms` to `32.17ms` HTTP; warm-session `impact` improved from `780.34ms` to
    `24.96ms` stdio and from `794.49ms` to `37.18ms` HTTP. `group_sync fixture` stayed healthy at
    `1.32ms` stdio and `2.37ms` HTTP. `query` remains an open Phase 15 candidate and was not part
    of this optimization.
  - Graph parity/analyze smoke:
    packaged Go analyze with benchmark output passed in `60,975ms` and wrote
    `.tmp\phase15-route-cache-graph-parity.json`: `33,399` nodes, `66,035` relationships,
    `fallbackInsertCount=0`, `fallbackInsertFailures=0`, `skippedRelationships=0`,
    `nodeRows=33,399`, and `relationshipRows=66,035`.
  - Validation order and results:
    full launcher build was run first with
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` and passed in
    `35,387ms`. Then `go test ./cmd/... ./internal/... -count=1` passed in `29,410.2ms`. Full
    browser E2E passed `33/33` in `227,307.3ms` with the packaged Go backend and isolated
    `AVMATRIX_HOME` containing only `AVmatrix-GO`. `cd avmatrix && npm test` passed in
    `373,303.6ms`.
  - Optimization conclusion:
    the Phase 15 P0 `route_map` hot-path item is closed. The shared graph cache also removes the
    largest warm-session context/impact graph-decode cost, but their detailed timing-split checklist
    items remain open until a dedicated profiling slice records per-stage timings or explicitly
    closes them with owner/target evidence.

- Phase 14 frontend/mobile app coverage addendum (2026-05-13):
  - Phase-jump note:
    after the Phase 15 P0 `route_map` slice was committed, work intentionally jumped back to Phase
    14 because the user's requested React/Electron/TypeScript/Next.js/Vue/Nuxt/Svelte/Astro/React
    Native/Flutter/SwiftUI/Jetpack Compose coverage is provider/framework capability work. It is
    not Phase 15 optimization, and the Phase 15 `context`, `impact`, and `group_sync` optimization
    items remain open.
  - Graph freshness and impact:
    after commit `8e3a8dd`, the packaged Go runtime refreshed the graph with
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --skip-agents-md --no-stats`,
    producing `33,400` nodes and `66,036` relationships. Impact checks before editing showed
    `DetectLanguage` HIGH (`3` impacted symbols, `4` affected processes), `parseFiles` CRITICAL
    (`3` impacted symbols, `61` affected processes), `DetectFromPath` CRITICAL (`3` impacted
    symbols, `8` affected processes), and `DetectFromAST` HIGH (`3` impacted symbols, `4` affected
    processes). The E2E spec files were not present as graph targets, so their selector hardening
    was verified by Playwright rather than graph impact.
  - Implementation:
    Go now detects `.svelte` and `.astro` files and exposes them in the Go-owned Web contract
    schema/generated browser adapter. Svelte and Astro providers reuse a shared single-file
    component helper: Svelte extracts inline `<script>` blocks, Astro extracts frontmatter, both
    parse script content through the existing TypeScript/JavaScript ScopeIR extractor, and both
    preserve the container language in the emitted IR. Analyze dispatch routes Vue/Svelte/Astro
    through those container providers before normal tree-sitter parser dispatch.
  - Framework coverage:
    framework path/AST facts now cover the requested surfaces: React, Electron, Next.js, Vue, Nuxt,
    Svelte/SvelteKit, Astro, React Native, Flutter, SwiftUI, and Jetpack Compose. TypeScript remains
    the existing base language provider; Vue, Flutter/Dart, SwiftUI/Swift, and Android/Kotlin keep
    their existing provider foundations with expanded app-framework facts.
  - Benchmark result:
    after the full launcher build, provider microbenchmarks recorded Svelte median `896,080ns/op`
    and Astro median `830,770ns/op`. The isolated frontend/mobile fixture analyze passed with
    `11/11` files parsed, `0` unsupported, `0` failed, `54` nodes, `61` relationships, and DB load
    `fallbackInsertFailures=0`, `skippedRelationships=0`. Its graph properties included `react`,
    `electron`, `nextjs`, `nextjs-api`, `nuxt`, `svelte`, `sveltekit`, `astro`, `react-native`,
    `flutter`, `swiftui`, `ios`, `android-kotlin`, and `jetpack-compose`.
  - Current-repo graph parity/analyze smoke:
    packaged Go analyze passed in `61,853.6ms` wall time with benchmark total `61,521.2ms`:
    `1,226` files scanned, `1,053` parsed, `173` unsupported, `0` failed, `33,489` nodes, `66,356`
    relationships, DB load `fallbackInsertCount=0`, `fallbackInsertFailures=0`,
    `skippedRelationships=0`, `nodeRows=33,489`, and `relationshipRows=66,356`.
  - Validation order and results:
    full launcher build was run first with
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` and passed in
    `39,240.9ms`. Focused Go tests for providers/scanner/frameworks/analyze/contracts passed after
    the build. `go test ./cmd/... ./internal/... -count=1` passed. Browser E2E used an isolated
    `AVMATRIX_HOME` and a verified packaged Go backend whose `/api/repos` returned only
    `AVmatrix-GO`; the final accepted full run passed with `32` passed / `1` skipped in
    `494,207.2ms`. `cd avmatrix && npm test` passed in `335,723.4ms`.
  - Test hardening note:
    larger current-repo graphs expose lower-case `processes` file/tool buttons near the My AI panel.
    Three E2E interactions were made more specific so they click the capitalized Processes
    navigation tab and assert panel closure through the chat composer. This was test selector
    hardening only; app behavior and graph output were unchanged.
  - Provider conclusion:
    the requested Phase 14 frontend/mobile app coverage addendum is closed with benchmark and E2E
    evidence. The next planned work returns to the open Phase 15 `context` timing-split/profile
    optimization slice.

- Phase 10 LadybugDB fallback fail-closed correctness reassessment (2026-05-13):
  - Phase-jump note:
    this work intentionally interrupted the open Phase 15 `context` optimization slice because the
    user's fallback question is a persistence correctness issue. The Phase 15 `context` item remains
    open until its own timing-split/profile evidence is recorded.
  - Correctness assessment:
    yes, the old fallback mechanism could produce wrong data if it became part of the normal DB load
    success path. The risk cases were: `IGNORE_ERRORS=true` retry could silently drop rows while
    returning success; a COPY implementation that partially inserted before fallback could create
    duplicates; fallback parsing defaulted malformed `confidence` and `step` to `1` and `0`, which
    mutates audit semantics; `FallbackInsertFailures > 0` was counted but did not make analyze fail;
    and unsupported schema-pair fallback could hide missing schema coverage.
  - AVmatrix/impact evidence:
    `context LoadCSVExport --repo AVmatrix-GO --file internal/lbugload/load.go` showed the normal DB
    load path is called by `loadGraph`, then analyze/server runtime flows. Upstream impact for
    `LoadCSVExport` was HIGH across analyze/httpapi/lbugload, and upstream impact for
    `fallbackRelationshipFile` was CRITICAL because it is directly under the DB load path and tests.
    This justified changing the default loader behavior only with explicit diagnostic opt-in for
    fallback.
  - Implementation:
    `LoadCSVExport` now delegates to `LoadCSVExportWithOptions` with zero fallback permissions.
    The default path fails closed before node COPY when export metrics report skipped
    relationships, fails on relationship COPY errors, fails on unsupported schema-pair COPY, and
    fails when any fallback insert row fails. Diagnostic fallback and COPY retry with
    `IGNORE_ERRORS=true` are exposed only through explicit `LoadOptions` flags for tests or manual
    investigation.
  - Tests:
    new loader tests prove the normal path fails closed on relationship COPY failure and unsupported
    schema pairs, while diagnostic fallback returns an error when any fallback insert fails. The
    existing fallback compatibility test now opts into diagnostic mode explicitly.
  - Benchmark result:
    normal COPY loader microbenchmark samples were `2,537`, `2,541`, `2,557`, `2,571`, and
    `2,596ns/op` with `19 allocs/op`. Diagnostic fallback samples were `736,094`, `740,211`,
    `751,108`, `760,664`, and `764,844ns/op` with `5,061 allocs/op`.
  - Graph parity/analyze smoke:
    packaged Go analyze passed in `59,937.2ms` and wrote
    `.tmp\fallback-failclosed-graph-parity.json`: `33,573` nodes, `66,603` relationships,
    `fallbackInsertCount=0`, `fallbackInsertFailures=0`, `skippedRelationships=0`,
    `nodeRows=33,573`, and `relationshipRows=66,603`.
  - Validation order and results:
    full launcher build was run first with
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` and passed in
    `34,207.9ms`. Then `go test ./internal/lbugload -count=1` passed; lbugload benchmarks passed;
    packaged analyze graph parity passed; `go test ./cmd/... ./internal/... -count=1` passed in
    `27,772.9ms`; full browser E2E through the packaged Go backend and isolated `AVMATRIX_HOME`
    passed with `32` passed / `1` skipped in `512,685.5ms`; and `cd avmatrix && npm test` passed in
    `438,204.3ms`.
  - Conclusion:
    fallback is no longer a silent normal-runtime success mechanism. If the current repo ever needs
    fallback or skipped relationship acceptance, analyze fails and exposes the schema/COPY gap
    instead of publishing a potentially wrong LadybugDB graph.

- Phase 15 MCP context one-pass neighborhood optimization (2026-05-13):
  - Phase-jump note:
    this slice resumed Phase 15 after the Phase 10 fallback correctness commit `7fe392b`. It closes
    the dedicated `context` timing-split/profile item only; the `impact` and HTTP `group_sync`
    optimization items remain open.
  - Graph freshness and impact:
    before editing, the packaged Go runtime refreshed the graph with
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --skip-agents-md --no-stats`,
    producing `33,491` nodes and `66,358` relationships. Impact checks showed `contextTool` LOW by
    exact UID, `contextCandidates` CRITICAL because it is also used by impact/rename flows,
    `contextCategorizedRefs` HIGH, `contextClassLikeIncomingRefs` HIGH,
    `contextProcessParticipation` HIGH, and the shared graph-resource cache CRITICAL. The final
    implementation avoided changing the shared graph-resource cache contract and kept the payload
    shape unchanged.
  - Implementation:
    `contextTool` now delegates to `contextToolInternal`; benchmarks can call
    `contextToolProfiled` to collect repo resolve, target lookup, neighborhood read, symbol payload,
    and formatting timings. The normal runtime path does not collect those timings. The new
    `contextNeighborhood` helper scans relationships once to assemble incoming refs, outgoing refs,
    process participation, and class-like constructor/file incoming refs, then reuses shared sorting
    helpers for deterministic payload order.
  - Benchmark result:
    current before-run `context` measured `23.20ms` stdio and `25.87ms` HTTP. Final packaged Go
    measured `15.36ms` stdio and `18.67ms` HTTP, below the `<100ms` target and faster than the same
    final TypeScript baseline row (`101.02ms` stdio / `107.27ms` HTTP). Route-map stayed healthy at
    `6.83ms` stdio / `8.85ms` HTTP, and `group_sync fixture` stayed healthy at `1.27ms` stdio /
    `2.62ms` HTTP.
  - Timing split:
    `BenchmarkContextToolWarmNeighborhood` used `2,500` incoming refs, `2,500` outgoing refs, and
    `750` process memberships. Samples were `26.96-27.67ms/op`; the neighborhood read accounted
    for roughly `24.58-26.57ms/op`, target lookup for `1.05-1.89ms/op`, and repo resolve for less
    than `0.61ms/op`. This confirms the remaining cost is the single relationship-neighborhood
    pass and payload allocation, not repeated full-graph scans.
  - Graph parity/analyze smoke:
    packaged Go analyze passed in `59,979.6ms` and wrote
    `.tmp\phase15-context-final-graph-parity.json`: `33,574` nodes, `66,604` relationships,
    `fallbackInsertCount=0`, `fallbackInsertFailures=0`, `skippedRelationships=0`,
    `nodeRows=33,574`, and `relationshipRows=66,604`.
  - Validation order and results:
    full launcher build was run first with
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` and passed in
    `34,207.9ms`. Then the final MCP benchmark passed in `18,917.6ms`; focused MCP tests passed;
    the context warm-neighborhood benchmark passed; packaged analyze graph parity passed;
    `go test ./cmd/... ./internal/... -count=1` passed in `27,772.9ms`; browser E2E through the
    packaged Go backend and isolated `AVMATRIX_HOME` passed with `32` passed / `1` skipped in
    `512,685.5ms`; and `cd avmatrix && npm test` passed in `438,204.3ms`.
  - Optimization conclusion:
    the Phase 15 `context` timing-split/profile target is closed. The next Phase 15 item is the
    dedicated `impact` timing-split/profile slice.

- Phase 15 MCP impact timing-profile slice (2026-05-13):
  - Phase-jump note:
    this slice continued Phase 15 after the `context` optimization commit `b5630ce`. It did not
    jump to Phase 14, Phase 10, or Phase 17 because the work is MCP timing/profile work against an
    already cut-over Go runtime.
  - Graph freshness and impact:
    before editing, the packaged Go runtime refreshed the graph with
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --skip-agents-md --no-stats` in
    `57,478.7ms`, producing `33,575` nodes and `66,605` relationships. Impact checks showed
    `impactTool` LOW by exact UID, `runImpactBFS` LOW with `impactTool` as its direct caller, and
    helper paths such as `impactAffectedProcesses`, `impactAffectedModules`, and
    `impactClassLikeSeedIDs` HIGH because they sit under the shared impact traversal. The HIGH
    helper risk was accepted only with a narrow change that preserves the public payload shape and
    normal runtime behavior.
  - Implementation:
    `impactTool` now delegates to `impactToolInternal`; tests and benchmarks can call
    `impactToolProfiled` to collect repo resolve, target lookup, node-index setup, traversal,
    affected-summary, and formatting timings. The normal `impactTool` path calls the same internal
    implementation with profiling disabled, so normal JSON-RPC payloads are unchanged.
  - Rejected branch:
    a runtime hot-index/cache attempt was measured and rejected. It regressed `impact` to
    `64.90ms` stdio / `71.71ms` HTTP versus the previous accepted `25.94ms` / `28.42ms`, so it was
    rolled back instead of being recorded as completed work.
  - Benchmark result:
    final packaged Go `impact` measured `26.53ms` stdio and `26.47ms` HTTP, below the `<150ms`
    Phase 15 target and faster than TypeScript's `140.79ms` / `135.10ms` on the same row.
    Synthetic warm-traversal profiling measured `19.35-20.73ms/op`; affected process/module
    summaries dominated the remaining cost at `13.77-16.25ms/op`, with traversal at
    `2.83-4.38ms/op`.
  - TypeScript-vs-Go current-repo comparison:
    the baseline TypeScript analyze artifact `.tmp\phase15-impact-ts-analyze.json` recorded
    `130,582.2ms`, `29,670` nodes, and `54,858` relationships. The packaged Go artifact
    `.tmp\phase15-impact-go-analyze.json` recorded `58,693.8ms`, `33,612` nodes, and `66,727`
    relationships. This is about `2.22x` faster for Go while still doing real DB load with
    `fallbackInsertFailures=0` and `skippedRelationships=0`.
  - Graph parity/analyze smoke:
    packaged Go analyze passed in `79,991.1ms` and wrote
    `.tmp\phase15-impact-profile-graph-parity.json`: `33,612` nodes, `66,727` relationships,
    `fallbackInsertCount=0`, `fallbackInsertFailures=0`, `skippedRelationships=0`,
    `nodeRows=33,612`, and `relationshipRows=66,727`.
  - Validation order and results:
    full launcher build was run first with
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` and passed in
    `35,341.2ms`. Then focused MCP profile tests passed; `BenchmarkImpactToolWarmTraversalProfile`
    passed; the final MCP benchmark passed in `21,021.4ms`; packaged analyze graph parity passed;
    `go test ./cmd/... ./internal/... -count=1` passed in `27,295.0ms`; browser E2E through the
    packaged Go backend and isolated `AVMATRIX_HOME` passed with `32` passed / `1` skipped in
    `536,712.6ms`; and `cd avmatrix && npm test` passed in `624.8s`.
  - Optimization conclusion:
    the Phase 15 `impact` timing-split/profile target is closed. The next Phase 15 item is the HTTP
    `group_sync` cold/warm timing-split slice.

- Phase 15 MCP group_sync cold/warm timing split (2026-05-14):
  - Phase-jump note:
    this slice continued Phase 15 after the impact profiling commit `bb9749e`. It did not jump to
    Phase 14, Phase 10, or Phase 17 because the work is MCP transport/runtime performance triage on
    an already cut-over Go runtime.
  - Graph freshness and impact:
    after commit `bb9749e`, the packaged Go runtime refreshed the graph with
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --skip-agents-md --no-stats` in
    `58,272.3ms`, producing `33,613` nodes and `66,728` relationships. `groupSyncTool` impact by
    exact Go UID was LOW with no upstream impacted symbols. `internal/group.Sync` impact was
    CRITICAL because both CLI `group sync` and MCP `group_sync` depend on it, with CLI/MCP modules
    and eight affected group-command processes. That CRITICAL result is why this slice did not
    change production `Sync` behavior.
  - Implementation:
    no production runtime code changed. The committed code adds `BenchmarkSyncSmallFixture` for the
    Go core group-sync fixture, while HTTP cold/warm measurement stays in `.tmp` benchmark scripts.
    This records the real bottleneck without weakening the `group_sync` contract.
  - Benchmark result:
    final HTTP benchmark showed Go server ready `1,358.68ms` vs TypeScript `1,751.30ms`, Go cold
    initialize `4.99ms` vs TypeScript `15.06ms`, Go cold `group_sync` `15.99ms` vs TypeScript
    `10.10ms`, and Go cold total `20.97ms` vs TypeScript `25.16ms`. Warm-session Go
    `group_sync` averaged `11.31ms` with p95 `11.92ms`; TypeScript averaged `7.20ms` with p95
    `9.88ms`.
  - Core/profile result:
    `BenchmarkSyncSmallFixture` samples were `10.39-11.57ms/op`, about `24.4KB/op`, and
    `220 allocs/op`. CPU pprof showed the cost is dominated by Windows file-write syscalls under
    `group.WriteRegistry`/`os.WriteFile`, not MCP JSON-RPC overhead. The queued target is
    `[P15-GROUPSYNC-REGISTRY-WRITE-OPT]`: reduce `internal/group.WriteRegistry` persistence cost so
    warm HTTP `group_sync` reaches `<=7ms`, while preserving `contracts.json` schema,
    `GeneratedAt`, CLI/MCP payloads, and exact-match semantics.
  - Graph parity/analyze smoke:
    an earlier graph-parity rerun failed because the interrupted E2E analyze left stale
    `.avmatrix/analyze.lock` and partial `lbug.shadow`/`lbug.wal.checkpoint`; the recorded lock PID
    was not running. After removing only those repo-local stale artifacts, packaged Go analyze
    passed in `59,704.6ms` and wrote `.tmp\phase15-group-sync-final-graph-parity.json`:
    `33,643` nodes, `66,799` relationships, `fallbackInsertCount=0`,
    `fallbackInsertFailures=0`, `skippedRelationships=0`, `nodeRows=33,643`, and
    `relationshipRows=66,799`.
  - Validation order and results:
    full launcher build was run first after adding the benchmark file and passed in `33,425.1ms`.
    Then the core benchmark passed in `7,309.6ms`; the HTTP cold/warm benchmark passed in
    `4,263.3ms`; packaged analyze graph parity passed; `go test ./cmd/... ./internal/... -count=1`
    passed in `25,660.5ms`; browser E2E through the packaged Go backend and isolated
    `AVMATRIX_HOME` passed with `32` passed / `1` skipped in `503,825.4ms`; and `cd avmatrix &&
    npm test` passed in `374,034.3ms`.
  - Optimization conclusion:
    the Phase 15 HTTP `group_sync` timing-split target is closed with a queued, scoped registry
    persistence optimization rather than an unsafe runtime change to a CRITICAL path. The next Phase
    15 item is the P3 MCP startup/query/tools-list/protocol-noise preservation slice.

- Phase 15 MCP P3 startup/query/tools-list/noise preservation slice (2026-05-14):
  - Phase-jump note:
    this slice continued Phase 15 after the HTTP `group_sync` timing-split commit `656cfca`. It did
    not jump to Phase 14, Phase 10, or Phase 17 because the work is MCP performance/profile work on
    an already cut-over Go runtime.
  - Graph freshness and impact:
    before editing, the packaged Go runtime refreshed the graph with
    `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --skip-agents-md --no-stats` in
    `59,746.3ms`, producing `33,644` nodes and `66,800` relationships. AVmatrix impact checks were
    LOW for `queryTool`, `rankedProcessMatches`, and `resourceProcessSteps`/process-step resource
    helper usage in the edited path, with the direct query/resource callers identified before the
    change.
  - Implementation:
    `queryTool` now builds a per-process `STEP_IN_PROCESS` index once with
    `resourceProcessStepsByProcess(g)`, passes that index into `rankedProcessMatches`, and reuses it
    for `process_symbols`. The old path repeatedly called `resourceProcessSteps` for every process,
    rebuilding a node map and scanning all graph relationships per process. Payload shape and MCP
    protocol behavior are unchanged.
  - Benchmark result:
    the P3 before-run measured Go `query` at `3,501.00ms` stdio and `3,505.47ms` HTTP. The final
    packaged MCP benchmark measured `763.95ms` stdio and `763.93ms` HTTP, a roughly `78%` reduction.
    The remaining canonical query row is cold graph snapshot load: the same-session probe measured
    `query cold=768.41ms`, followed by `query warm 1=7.84ms` and `query warm 2=7.22ms`, with
    `noiseBytes=0`.
  - Startup/tools/noise result:
    final HTTP initialize stayed fast at `10.26ms`, and the group fixture HTTP initialize was
    `2.09ms`. The repo stdio initialize row was noisy at `1,248.27ms` but still below the same-run
    TypeScript row `1,311.27ms`; the isolated group fixture showed the expected fast Go stdio
    initialize at `55.83ms` vs TypeScript `1,300.03ms`. `tools/list` remained `7,795` bytes for Go
    versus `18,447` bytes for TypeScript, and stdio protocol noise remained `0` bytes.
  - Focused benchmark:
    `BenchmarkQueryToolWarmProcessIndex` on a synthetic `700` process / `4` step graph measured
    `2.21-2.69ms/op`, about `1.02MB/op`, and `5,012-5,013 allocs/op`. This proves the query
    ranking/step-index hot path is no longer doing repeated full relationship scans.
  - Graph parity/analyze smoke:
    packaged Go analyze passed in `58,540.0ms` and wrote `.tmp\phase15-p3-final-graph-parity.json`:
    `33,660` nodes, `66,882` relationships, `fallbackInsertCount=0`,
    `fallbackInsertFailures=0`, `skippedRelationships=0`, `nodeRows=33,660`, and
    `relationshipRows=66,882`.
  - Validation order and results:
    full launcher build was run first and passed in `33,418.9ms`. Then the final MCP benchmark
    passed in `14,401.6ms`; the cold/warm query probe passed; focused MCP query tests passed; the
    warm-query benchmark passed; packaged analyze graph parity passed; `go test ./cmd/...
    ./internal/... -count=1` passed in `28,306.0ms`; browser E2E through the packaged Go backend and
    isolated `AVMATRIX_HOME` passed with `32` passed / `1` skipped after isolated analyze
    `57,281.5ms` and Playwright `488,397.8ms`; and `cd avmatrix && npm test` passed in
    `385,336.0ms`.
  - Optimization conclusion:
    the Phase 15 P3 target is closed. The real improvement is not merely a checked box: current MCP
    query no longer performs process-count multiplied relationship scans, while startup, tool
    discovery payload size, and protocol cleanliness remain preserved. Remaining cold graph decode
    belongs to the upcoming large-repo graph stream/memory/pprof benchmark work, not to query
    ranking itself.

- Phase 15 large-repo graph stream and pprof slice (2026-05-14):
  - Phase-jump note:
    this slice continued Phase 15 after commit `99d1a9a`. It did not jump to provider coverage,
    fallback correctness, or cutover because the work is large-repo performance/profile validation
    for the already cut-over Go runtime.
  - Graph freshness and impact:
    the slice started by refreshing the graph with packaged Go analyze in `64,055.7ms`, producing
    `33,661` nodes and `66,883` relationships. AVmatrix impact for `newAnalyzeCommand` was
    CRITICAL because it is CLI root/analyze wiring; that risk was accepted only for narrow opt-in
    profiling flags that do not change default analyze behavior. AVmatrix impact for
    `streamGraphNDJSON` was LOW, with direct impact limited to `handleGraph`.
  - Implementation:
    `avmatrix analyze` now accepts `--cpuprofile` and `--memprofile`, writing Go pprof CPU and heap
    profiles when explicitly requested. `/api/graph?stream=true` now flushes NDJSON every `512`
    records plus a final flush instead of flushing every node/relationship. The NDJSON record shape,
    content type, and default non-streaming graph JSON path are unchanged.
  - Graph stream benchmark:
    before stream batching, Go `/api/graph?stream=true&repo=AVmatrix-GO` took `2,915.0ms`; final
    packaged Go stream took `1,384.8ms` on the final graph. Go JSON on the same final graph took
    `1,384.3ms`; TypeScript JSON and stream took `5,515.6ms` and `5,852.4ms`. Final Go stream
    emitted `40,341,034` bytes across `100,633` NDJSON records.
  - Analyze/profile benchmark:
    final packaged Go analyze with `--cpuprofile`, `--memprofile`, and `--benchmark-json` passed in
    `58,459.2ms`, writing `.tmp\phase15-large-profile-final-analyze.json`,
    `.tmp\phase15-large-profile-final-cpu.pprof`, and `.tmp\phase15-large-profile-final-mem.pprof`.
    The benchmark JSON recorded `33,702` nodes, `66,931` relationships,
    `fallbackInsertFailures=0`, `skippedRelationships=0`, `nodeRows=33,702`, and
    `relationshipRows=66,931`.
  - Memory peak:
    final `maxObservedSys` was `941,588,728` bytes (`~898MiB`), with
    `endAllocBytes=343,455,784`.
  - CPU pprof result:
    CPU profile size was `67,778` bytes. `go tool pprof -top` showed `runtime.cgocall` at `42.12s`
    flat (`67.83%`, `43.88s` cumulative) and
    `resolution.(*workspace).resolveImportedMember` at `4.45s` flat (`7.17%`, `4.65s` cumulative).
    This points the next CPU work at native DB load/cgo batching and imported-member resolution, not
    at the HTTP graph streamer after batching.
  - Memory pprof result:
    heap profile size was `57,861` bytes. Alloc-space top was `bytes.genSplit` (`1,022.06MB`),
    tree-sitter node allocation (`255.01MB`), tree-sitter `GoString` extraction (`247.00MB`),
    `scopeir.callKey` (`236.04MB`), and `ScopeIR.Normalized` cumulative allocation (`932.17MB`).
    In-use top was `bytes.growSlice` (`64.00MB`), `ScopeIR.Normalized` (`61.34MB`),
    `Graph.AddRelationship` (`14.34MB`), `emitDefinitionNodes` (`11.00MB`), and `graph.GenerateID`
    (`10.00MB`).
  - Queued targets:
    `[P15-DBLOAD-CGO-BATCH-OPT]` owns native DB load/cgo overhead and must preserve fail-closed DB
    load semantics plus node/relationship row parity. `[P15-SCOPEIR-NORMALIZED-ALLOC-OPT]` owns
    ScopeIR normalization/range-key/call-key allocation and must preserve resolution evidence,
    edge parity, and provider contract outputs.
  - Validation order and results:
    full launcher build was run before benchmark/test validation and passed in `34,126.1ms`.
    Focused CLI pprof tests and HTTP graph stream tests passed. `go test ./cmd/... ./internal/...
    -count=1` passed in `27,043.8ms`; browser E2E through the packaged Go backend and isolated
    `AVMATRIX_HOME` passed with isolated analyze `64,342.3ms`, Playwright `32` passed / `1` skipped
    in `408,728.9ms`; and `cd avmatrix && npm test` passed in `419,432.8ms`.
  - Optimization conclusion:
    graph stream time, memory peak, CPU pprof, memory pprof, and graph streaming optimization are
    now evidenced. The next Phase 15 work should target the profile-backed CPU/memory bottlenecks
    rather than continuing to tune `/api/graph` streaming blindly.

- Phase 15 DB-load parallel COPY rejection slice (2026-05-14):
  - Phase-stay note:
    this slice continued Phase 15 after commit `03fdc67` because it evaluates a profile-backed DB
    load/cgo performance hypothesis. It did not jump to Phase 10 fallback correctness because the
    existing fail-closed loader semantics are being preserved, and it did not jump to Phase 17
    because runtime authority is already cut over.
  - AVmatrix usage:
    packaged Go analyze refreshed the AVmatrix graph before graph-based work. AVmatrix context for
    `NodeCopyQuery` and `RelationshipCopyQuery` confirmed both COPY query builders feed
    `LoadCSVExportWithOptions`; the tested change therefore touched the real DB-load runtime path,
    not Web UI glue.
  - Impact/risk:
    prior impact checks for `NodeCopyQuery` and `RelationshipCopyQuery` were CRITICAL because they
    sit directly under native LadybugDB CSV loading and loader contract tests. The candidate was
    kept to a narrow experiment and rejected after runtime evidence; production code was restored to
    `PARALLEL=false`.
  - Candidate result:
    changing COPY options to `PARALLEL=true` built successfully in `36,663.0ms`, but current-repo
    packaged analyze failed closed in `33,143.2ms`. LadybugDB reported that quoted newlines are not
    supported by the parallel CSV reader and requested `PARALLEL=FALSE`; the failing file was
    `method.csv` at line `315`. Because AVmatrix stores source/content text in CSV fields, this is a
    real data-shape incompatibility, not just a synthetic benchmark failure.
  - Rollback result:
    after restoring `PARALLEL=false`, full launcher build passed in `33,780.9ms`. Packaged Go analyze
    then refreshed the graph in `59,961.4ms`; `.tmp\phase15-dbload-rollback-safe-analyze.json`
    recorded total `58,086.4ms`, parse `19,424.1ms`, resolution `5,311.4ms`, DB load `28,685.4ms`,
    `nodeRows=33,703`, `relationshipRows=66,932`, `nodeCopyCount=19`,
    `relationshipCopyCount=90`, `fallbackInsertCount=0`, `fallbackInsertFailures=0`, and
    `skippedRelationships=0`.
  - Validation:
    final validation kept the required order: full launcher build first (`35,472.3ms`), full Go tests
    (`29,349.2ms`), browser E2E through the packaged Go backend and isolated `AVMATRIX_HOME`
    (`33/33` passed; isolated analyze `63,905.4ms`; Playwright `180,285.1ms`), and
    `cd avmatrix && npm test` (`409,607.3ms`). The accepted E2E analyze artifact
    `.tmp\phase15-dbload-docs-e2e-analyze-final.json` recorded `33,704` nodes, `66,933`
    relationships, DB load `32,835.4ms`, `fallbackInsertFailures=0`, and
    `skippedRelationships=0`.
  - Conclusion:
    the plan did not advance by pretending the checkbox is done. `[P15-DBLOAD-CGO-BATCH-OPT]`
    remains open because global parallel COPY is unsafe for current AVmatrix CSV content. The useful
    evidence is that the unsafe candidate failed closed and the rollback path still performs real DB
    load with zero fallback/skipped rows. The next Phase 15 target must either find a safer DB-load
    batching design or move to the profile-backed ScopeIR allocation target.

- Phase 15 ScopeIR normalization allocation optimization (2026-05-14):
  - Phase-stay note:
    this slice continued Phase 15 after the DB-load parallel COPY rejection commit `69cca33`.
    It did not jump to Phase 14, Phase 10, or Phase 17 because the work is profile-backed memory
    optimization on the already cut-over Go runtime.
  - AVmatrix usage and impact:
    packaged Go analyze refreshed the AVmatrix graph before graph-based work. AVmatrix context on
    `ScopeIR.Normalized` showed it owns the ScopeIR fact sorting path. AVmatrix impact reported
    `ScopeIR.Normalized` LOW risk, `callKey` LOW risk, and `rangeKey` MEDIUM risk, with the affected
    execution process limited to `Normalized`.
  - Implementation:
    `ScopeIR.Normalized` now sorts scopes, definitions, imports, calls, accesses, heritage,
    type annotations, return types, framework facts, domain facts, bindings, and type bindings with
    field-by-field comparators instead of constructing concatenated string sort keys for each
    comparison. The removed hot path included `callKey`, `rangeKey`, and `padInt`; deterministic
    marshal/unmarshal behavior remains covered by the ScopeIR golden tests.
  - Focused benchmark:
    a new `BenchmarkScopeIRNormalizedLargeSort` fixture with `2,000` unordered facts measured the
    old string-key sort path at `18.10-19.14ms/op`, about `12.85MB/op`, and
    `242,580-242,623 allocs/op`. The final comparator path measured `3.38-3.86ms/op`,
    about `4.05MB/op`, and `20,287-20,329 allocs/op`.
  - Large-repo benchmark:
    the slice started from `.tmp\phase15-scopeir-start-graph.json`: wall `72,201.6ms`, benchmark
    total `71,802.7ms`, parse `19,879.0ms`, resolution `5,451.2ms`, DB load `41,450.6ms`,
    `33,704` nodes, `66,933` relationships, `maxObservedSys=944,668,920`, and DB fallback/skipped
    `0`. Final packaged analyze with heap profile passed in `60,934.4ms`; benchmark artifact
    `.tmp\phase15-scopeir-final-analyze.json` recorded total `58,448.7ms`, parse `18,730.4ms`,
    resolution `5,312.4ms`, DB load `29,923.8ms`, `33,695` nodes, `66,981` relationships,
    `maxObservedSys=928,522,488`, `fallbackInsertFailures=0`, and `skippedRelationships=0`.
  - Heap pprof result:
    `.tmp\phase15-scopeir-final-mem.pprof` was `43,081` bytes. Final alloc-space pprof showed
    `ScopeIR.Normalized` at `117.45MB` flat / `126.96MB` cumulative, down from the prior
    `~932.17MB` cumulative ScopeIR normalization allocation. `callKey` and `rangeKey` no longer
    appear in the final pprof top output. The next memory hotspot is now `bytes.genSplit`
    (`1,059.40MB`) via `frameworks.definitionWindow`, so the next Phase 15 memory target is
    `[P15-FRAMEWORK-DEFINITION-WINDOW-ALLOC-OPT]`.
  - Validation order and results:
    final validation followed the required order: full launcher build first (`40,386.9ms`), focused
    ScopeIR golden tests and benchmarks, full Go tests (`32,531.4ms`), browser E2E through the
    packaged Go backend and isolated `AVMATRIX_HOME` (`32` passed / `1` skipped; isolated analyze
    `57,957.6ms`; Playwright `405,663.7ms`), and `cd avmatrix && npm test` (`342,850.8ms`).
  - Optimization conclusion:
    `[P15-SCOPEIR-NORMALIZED-ALLOC-OPT]` is closed because the real allocation target moved, not
    because a checklist line was ticked. ScopeIR sort-key allocation has been reduced and graph/DB
    parity remains clean with fallback/skipped counts at `0`; remaining Phase 15 work should target
    the newly exposed `frameworks.definitionWindow` allocation and the still-open DB-load/cgo
    batching target.

- Phase 15 framework definition-window allocation optimization (2026-05-14):
  - Phase-stay note:
    this slice continued Phase 15 after the ScopeIR comparator commit `e1efe41`. It did not jump to
    Phase 14, Phase 10, or Phase 17 because the work is profile-backed memory optimization on the
    already cut-over Go runtime.
  - AVmatrix usage and impact:
    packaged Go analyze refreshed the AVmatrix graph before graph-based work. AVmatrix context on
    `definitionWindow` showed one direct caller, `AnnotateScopeIR`. AVmatrix impact reported
    `definitionWindow` LOW risk, while `AnnotateScopeIR` was CRITICAL because it feeds `parseFiles`
    and the CLI analyze path. The blast radius was handled by limiting the change to internal line
    indexing and by preserving the window text contract with focused tests.
  - Implementation:
    `AnnotateScopeIR` now creates a `definitionWindowIndex` once per source file, records line start
    offsets, and slices definition windows from original source bytes. This removes the old
    per-definition `bytes.Split` plus `bytes.Join` path while keeping the `definitionWindow` wrapper
    and its `600` byte cap semantics.
  - Focused benchmark:
    the new `BenchmarkAnnotateScopeIRDefinitionWindowManyDefinitions` fixture with `2,000`
    definitions measured the old split/join path at `334.7-357.0ms/op`, about `395.5MB/op`, and
    `10047-10049 allocs/op`. The final line-index path measured `6.72-7.06ms/op`, about
    `2.16MB/op`, and `6049 allocs/op`.
  - Large-repo benchmark:
    final packaged analyze with heap profile passed in `57,942.5ms`; benchmark artifact
    `.tmp\phase15-framework-window-final-analyze.json` recorded total `55,811.0ms`, parse
    `17,713.4ms`, resolution `5,324.2ms`, DB load `28,331.8ms`, `33,729` nodes,
    `67,032` relationships, `maxObservedSys=979,804,408`, `fallbackInsertFailures=0`, and
    `skippedRelationships=0`.
  - Heap pprof result:
    `.tmp\phase15-framework-window-final-mem.pprof` was `41,782` bytes. Final alloc-space pprof no
    longer showed `bytes.genSplit`; the replacement `frameworks.definitionWindowIndex.window` frame
    was `12MB` flat and `frameworks.AnnotateScopeIR` was `22.51MB` cumulative. The next memory
    target is now resolution workspace/name-resolution allocation (`buildWorkspace` `298.10MB`
    cumulative, `uniqueDefs` `126.39MB` flat, `resolveGlobalName` `212.67MB` cumulative), while
    `[P15-DBLOAD-CGO-BATCH-OPT]` remains open separately.
  - Validation order and results:
    final validation followed the required order: full launcher build first (`38,255.4ms`), focused
    framework tests and before/after benchmark, full Go tests (`27,620.2ms`), browser E2E through the
    packaged Go backend and isolated `AVMATRIX_HOME` (`32` passed / `1` skipped with
    `--workers=1`; isolated analyze `55,728.6ms`; graph stream smoke `1,734.1ms`; Playwright
    `397,958.2ms`), and `cd avmatrix && npm test` (`537,613.2ms`). A 4-worker Playwright run failed
    under concurrent graph-load pressure and was not accepted as the E2E gate.
  - Optimization conclusion:
    `[P15-FRAMEWORK-DEFINITION-WINDOW-ALLOC-OPT]` is closed because the actual `bytes.genSplit`
    hotspot has been removed from the heap profile and graph/DB parity remains clean with
    fallback/skipped counts at `0`. Remaining Phase 15 work should target either a safe DB-load/cgo
    batching design or `[P15-RESOLUTION-WORKSPACE-ALLOC-OPT]` from the new heap profile.

- Phase 15 resolution workspace allocation optimization (2026-05-14):
  - Phase-stay note:
    this slice continued Phase 15 after the framework definition-window commit `82a2961`. It did
    not jump to Phase 14, Phase 10, or Phase 17 because the work is profile-backed allocation
    optimization on the already cut-over Go runtime.
  - AVmatrix usage and impact:
    `avmatrix analyze --force --skip-agents-md` refreshed the graph before graph-based work.
    AVmatrix impact reported `buildWorkspace` CRITICAL because it feeds `BuildCrossFileBinding`,
    `definitionLookupNames` HIGH, and the targeted lookup helpers LOW. The direct caller and
    process blast radius were handled by limiting the change to pre-sizing and duplicate-candidate
    handling, then validating resolution fixture behavior and full analyze/E2E paths.
  - Implementation:
    `buildWorkspace` now pre-measures ScopeIR input sizes and pre-sizes workspace maps/slices.
    Definition lookup names use a fixed three-slot set for simple/qualified names, and hot lookup
    paths use `uniqueDefAccumulator` to preserve the "one unique definition only" contract without
    allocating candidate slices and maps. Focused tests cover trimmed/deduplicated lookup names and
    the two-item `uniqueDefs` fast path.
  - Focused benchmark:
    `BenchmarkResolveTypeScriptGraphFixture` moved from `368.3-403.8us/op`, about
    `259,341 B/op`, and `1733 allocs/op` to `345.1-378.1us/op`, about `247,954 B/op`, and
    `1681 allocs/op` after the final full build. This is the accepted benchmark improvement for
    the slice: roughly `11.4KB/op` and `52 allocs/op` removed from the fixture.
  - Large-repo benchmark:
    final packaged analyze with heap profile passed in `62,789.7ms`; benchmark artifact
    `.tmp\phase15-resolution-workspace-final2-analyze.json` recorded total `61,479.0ms`, parse
    `18,053.0ms`, resolution `5,014.8ms`, DB load `32,675.5ms`, `33,760` nodes,
    `67,153` relationships, `maxObservedSys=932,315,384`, `fallbackInsertFailures=0`, and
    `skippedRelationships=0`. The DB load phase varied upward, so this slice does not claim a macro
    wall-time speedup; it claims the focused allocation reduction and pprof movement.
  - Heap pprof result:
    `.tmp\phase15-resolution-workspace-final2-mem.pprof` alloc-space top showed `buildWorkspace`
    at `149.86MB` flat / `270.14MB` cumulative instead of the previous `298.10MB` cumulative target.
    `uniqueDefs` and the lookup closure frame no longer appeared in the top table. Remaining
    profile-backed candidates are `ScopeIR.Normalized` residual allocation (`122.29MB` cumulative),
    `workspace.callerForScope` (`70.52MB` flat), graph relationship allocation, definition-node
    emission, and `[P15-DBLOAD-CGO-BATCH-OPT]`.
  - Validation order and results:
    final validation followed the required order: full launcher build first (`34,953.3ms`), focused
    resolution tests (`2,343.2ms`), after-build micro benchmark (`8,152.8ms`), packaged analyze
    plus heap pprof (`62,789.7ms`), full Go tests (`32,739.5ms`), browser E2E through the packaged
    Go backend and isolated `AVMATRIX_HOME` (`32` passed / `1` skipped with `--workers=1`;
    isolated analyze `56,796.4ms`; Playwright `411,382.6ms`), and `cd avmatrix && npm test`
    (`579,022.0ms`).
  - Optimization conclusion:
    `[P15-RESOLUTION-WORKSPACE-ALLOC-OPT]` is closed because the targeted `uniqueDefs` and
    lookup-name allocation frames were removed from the heap profile, focused allocation counts
    improved, and graph/DB parity remains clean with fallback/skipped counts at `0`. Phase 15
    remains open for DB-load/cgo batching and any separately scoped hotspot from the latest pprof.

- Phase 15 DB-load schema lookup allocation optimization (2026-05-14):
  - Phase-stay note:
    this slice continued Phase 15 after the resolution workspace allocation commit `70e21b7`. It did
    not close the native COPY/cgo target; it reduced a separate DB-load export allocation hotspot
    shown by heap pprof.
  - AVmatrix usage and impact:
    `avmatrix analyze --force --skip-agents-md` refreshed the graph before graph-based work.
    AVmatrix impact reported `validNodeTables`, `nodeColumns`, and `relationPairSupported` CRITICAL
    because they feed `ExportGraphCSVs`, `loadGraph`, and the analyze path. `RelationPairs` was LOW
    risk. The blast radius was handled by keeping the CSV/COPY contract unchanged and validating
    with focused tests, packaged analyze, heap pprof, full E2E, and npm tests.
  - Implementation:
    `internal/lbugload` now uses package-level node column lists, a valid-node-table lookup, and a
    relation-pair lookup instead of rebuilding schema maps/slices during CSV export and COPY query
    generation. Packaged analyze initially failed closed on `Const->Function`; the slice added that
    LadybugDB relation pair and updated the DDL shape test instead of enabling fallback or skipping
    the relationship.
  - Focused benchmark:
    `BenchmarkExportGraphCSVs` moved from about `335KB/op` and `1906 allocs/op` to about
    `285KB/op` and `1403 allocs/op`. `BenchmarkLoadCSVExportCopyPathNoop` moved from
    `1536-1568 B/op` and `19 allocs/op` to `1376 B/op` and `17 allocs/op`. Time remained file-IO
    dominated and variable (`16.7-31.2ms/op` before, `14.8-30.9ms/op` after), so the accepted
    improvement is allocation reduction and pprof movement.
  - Large-repo benchmark:
    final packaged analyze with heap profile passed in `58,132.1ms`; benchmark artifact
    `.tmp\phase15-dbload-schema-lookup-final2-analyze.json` recorded total `56,490.3ms`, parse
    `18,873.0ms`, resolution `5,212.0ms`, DB load `27,697.0ms`, `33,783` nodes,
    `67,172` relationships, `nodeCopyCount=19`, `relationshipCopyCount=91`,
    `maxObservedSys=836,059,384`, `fallbackInsertFailures=0`, and `skippedRelationships=0`.
  - Heap pprof result:
    `.tmp\phase15-dbload-schema-lookup-final2-mem.pprof` alloc-space top no longer showed
    `lbugload.validNodeTables`; `lbugload.ExportGraphCSVs` was `31.95MB` cumulative instead of the
    previous `77.02MB` frame. Remaining DB-load work is native COPY/cgo overhead, plus any future
    pprof-backed export hotspot that has its own benchmark.
  - E2E validation note:
    the first two full Playwright runs were rejected because a `process-modal` visibility wait timed
    out at `5s`; the error snapshots showed the modal eventually opened. The E2E assertions were
    hardened to `15s` for process-modal visibility, then the full browser suite passed through the
    packaged Go backend. This changed validation tolerance only, not runtime behavior.
  - Validation order and results:
    final validation followed the required order: full launcher build first (`35,785.3ms` after the
    E2E wait hardening), focused `lbugload`/`lbugschema` tests (`2,344.0ms`), after-build
    benchmarks (`26,244.7ms`), packaged analyze plus heap pprof (`58,132.1ms`), full Go tests
    (`31,352.4ms`), browser E2E through the packaged Go backend and isolated `AVMATRIX_HOME`
    (`32` passed / `1` skipped with `--workers=1`; isolated analyze `57,103.4ms`; Playwright
    `412,198.7ms`), and `cd avmatrix && npm test` (`424,654.6ms`).
  - Optimization conclusion:
    `[P15-DBLOAD-SCHEMA-LOOKUP-ALLOC-OPT]` is closed because the targeted schema lookup allocation
    hotspot was removed from pprof and DB parity remains fail-closed-clean with fallback/skipped
    counts at `0`. `[P15-DBLOAD-CGO-BATCH-OPT]` remains open because this slice did not reduce the
    native COPY/cgo overhead shown by CPU pprof.

- Phase 15 native DB-load transaction optimization (2026-05-14):
  - Phase-stay note:
    this slice continued Phase 15 after the schema lookup allocation commit `1636cad`. It did not
    jump to Phase 10 fallback correctness or Phase 17 cutover authority because the work is
    profile-backed DB-load performance on the already cut-over Go runtime.
  - AVmatrix usage and impact:
    the packaged Go binary refreshed the AVmatrix graph before graph-based work. AVmatrix impact
    reported `loadGraph` CRITICAL because it feeds `analyze`, `newAnalyzeCommand`, `main`, and HTTP
    embed/analyze paths. AVmatrix context for `LoadCSVExport` showed analyze, native integration,
    benchmark, and fail-closed tests as the direct callers. The blast radius was handled by keeping
    the COPY SQL contract unchanged and adding optional transaction support only for runners that
    explicitly implement it.
  - Implementation:
    `lbugload.LoadCSVExportWithOptions` now wraps the node/relationship COPY sequence in a load
    transaction when the runner supports `BeginLoadTransaction`, `CommitLoadTransaction`, and
    `RollbackLoadTransaction`. The native LadybugDB write runner implements those hooks with
    `BEGIN TRANSACTION`, `COMMIT`, and `ROLLBACK`. Any normal fail-closed loader error still returns
    an error and rolls back; non-native/noop runners keep the old per-query behavior.
  - Benchmark:
    current-repo packaged analyze moved from wall `62,660.7ms`, benchmark total `62,327.7ms`, and
    DB load `32,246.0ms` in `.tmp\phase15-continue-start-graph.json` to wall `35,469.1ms`,
    benchmark total `34,167.8ms`, and DB load `5,773.7ms` in
    `.tmp\phase15-dbload-tx-attempt-analyze.json`. That is about `5.59x` faster in DB load and
    `1.82x` faster for the benchmark total, while `fallbackInsertFailures=0` and
    `skippedRelationships=0`.
  - CPU pprof:
    `.tmp\phase15-dbload-tx-attempt-cpu.pprof` showed `runtime.cgocall` at `19.64s` flat /
    `21.83s` cumulative, down from the earlier Phase 15 DB-load target profile at `42.12s` flat.
    Remaining cgo cost is expected because tree-sitter/parser and the native database still cross
    C.
  - E2E and smoke evidence:
    isolated E2E analyze under `.tmp\phase15-dbload-tx-e2e-home-*` took `37,658.0ms`; benchmark
    JSON recorded DB load `7,533.0ms`, graph rows `33,799` / `67,272`, and DB fallback/skipped `0`.
    Playwright through packaged Go backend on `127.0.0.1:4747` and Vite on `127.0.0.1:5173` passed
    with `32` passed / `1` skipped in `447,486.9ms`. A backend smoke check against the same
    isolated home confirmed `/api/info` and `/api/repos` served the indexed `AVmatrix-GO` repo.
  - Validation order and results:
    final validation followed the required order: full launcher build first (`52,836.0ms`), focused
    `lbugload` tests (`2,154.4ms`), native LadybugDB integration with launcher-equivalent CGO
    include/link/runtime path (`13,648.5ms`), after-build loader benchmarks (`23,453.1ms`),
    packaged analyze plus CPU pprof (`35,469.1ms`), full Go tests (`30,764.5ms`), and browser E2E
    through the packaged Go backend (`32` passed / `1` skipped). `cd avmatrix && npm test` had one
    initial flaky Rust skills E2E failure (`analyze --skills` exit `3221226505`), then the failed
    suite rerun passed (`25/25`, `110,639.9ms`) and the full command rerun passed
    (`393,820.1ms`).
  - Optimization conclusion:
    `[P15-DBLOAD-CGO-BATCH-OPT]` is closed because the native COPY sequence is now batched inside a
    transaction, the real current-repo DB load phase dropped from about `32.2s` to `5.8s`, and DB
    parity remains fail-closed-clean with fallback/skipped counts at `0`. Phase 15 remains open for
    file IO batching, parser pool sizing, or any next pprof-backed hotspot that benchmark/evidence
    proves is still material.

- Phase 15 imported-member resolution index optimization (2026-05-14):
  - Phase-stay / phase-jump note:
    this slice continued Phase 15 after the DB-load transaction commit `5de0fc6`. It first
    reassessed the open file IO batching and parser pool sizing checklist items, rejected both as
    current bottlenecks with pprof evidence, then stayed in Phase 15 and selected the real
    post-DB-load CPU hotspot: `workspace.resolveImportedMember`.
  - AVmatrix usage and impact:
    the packaged Go binary refreshed the AVmatrix graph before graph-based work. AVmatrix impact
    reported `parseFiles` CRITICAL when parser-pool work was considered, so that path was not
    changed. AVmatrix impact reported `resolveImportedMember` CRITICAL because it feeds
    `resolveCallTargetForTypeBinding`, `resolveCall`, `ResolveBoundInto`, and `Run`; the blast
    radius was handled by keeping the change internal to workspace lookup indexes and validating
    graph/DB parity through packaged analyze and E2E.
  - File IO/parser-pool reassessment:
    `.tmp\phase15-next-after-dbtx-start-graph.json` recorded total `35,015.3ms`, parse
    `19,150.8ms`, resolution `5,326.8ms`, DB load `5,803.5ms`, `33,800` nodes,
    `67,273` relationships, and DB fallback/skipped `0`. Heap pprof showed file read allocation at
    only `31.48MB` before the slice and `24.81MB` after it, with no file-read CPU hotspot. Parser
    metrics showed `createdParsers=4`, `total=1058`, and `failed=0`; parse CPU was in
    tree-sitter/provider work (`Pool.Parse`, `tsjs.Extract`) rather than pool capacity. These two
    generic checklist items are therefore closed as rejected for the current benchmark.
  - Implementation:
    `internal/resolution` now builds `importsByReceiver` keyed by `(sourceFile, localName)` while
    resolving imports. `resolveImportedMember` reads that index and scans only imports relevant to
    the call receiver, instead of scanning every workspace import for every imported member call.
    The existing `uniqueDefAccumulator` still preserves the single unique target requirement and
    ambiguous-target failure behavior.
  - Focused benchmark:
    `BenchmarkResolveImportedMemberManyImports` moved from `17.2-17.6us/op`, `48 B/op`, and
    `3 allocs/op` to `492-512ns/op`, `48 B/op`, and `3 allocs/op` after the full launcher build.
  - Large-repo benchmark:
    packaged current-repo analyze moved from `.tmp\phase15-next-after-dbtx-start-graph.json`
    benchmark total `35,015.3ms`, cross-file binding `1,585.8ms`, resolution `5,326.8ms`,
    DB load `5,803.5ms`, `33,800` nodes, and `67,273` relationships to
    `.tmp\phase15-import-index-final-analyze.json` benchmark total `28,727.0ms`, cross-file
    binding `579.8ms`, resolution `955.3ms`, DB load `5,596.5ms`, `33,803` nodes, and
    `67,310` relationships. The node/relationship count increase is from the new benchmark code in
    this commit; DB fallback/skipped stayed `0`. `benchmark-compare` reported total `-18%`,
    resolution `-82.1%`, and cross-file binding `-63.4%`.
  - CPU pprof:
    before profile `.tmp\phase15-next-after-dbtx-start-cpu.pprof` showed
    `resolveImportedMember` at `4.91s` flat / `5.11s` cumulative. Final profile
    `.tmp\phase15-import-index-final-cpu.pprof` no longer showed it in the top CPU table. The next
    real bottleneck is parse/native cgo: `runtime.cgocall` remained at `19.61s` flat /
    `21.62s` cumulative, with `Pool.Parse`, `tsjs.Extract`, `golang.Extract`, and native DB query
    work in the cumulative tree.
  - E2E validation note:
    the first full Playwright run was rejected because one UI timing check failed
    (`process-row` highlight class), although the same test passed when rerun directly in
    `36,780.5ms`. A second full run was also rejected due auto-connect readiness timing plus the
    same highlight check. The accepted full run used a fresh isolated `AVMATRIX_HOME`, packaged Go
    backend, `--workers=1 --retries=1`, and completed with `32` passed / `1` skipped in
    `428,776.4ms`; the output did not need an actual retry.
  - Validation order and results:
    final validation followed the required order: full launcher build first (`41,236.0ms`), focused
    resolution tests (`2,312.7ms`), after-build micro benchmark (`11,508.5ms`), packaged analyze
    plus CPU/memory pprof (`30,412.5ms` wall), full Go tests (`31,327.0ms`), and browser E2E
    through the packaged Go backend (`32` passed / `1` skipped; isolated analyze `29,045.1ms`;
    Playwright `428,776.4ms`). `cd avmatrix && npm test` passed in `378,034.1ms`.
  - Optimization conclusion:
    `[P15-FILE-IO-BATCHING-REJECTED]`, `[P15-PARSER-POOL-SIZING-REJECTED]`, and
    `[P15-IMPORTED-MEMBER-INDEX-OPT]` are closed. Phase 15 remains open for the next real
    pprof-backed bottleneck, currently tree-sitter/provider parse cgo and remaining native DB
    commit/COPY cost.

- Phase 15 parser node-count diagnostic optimization (2026-05-14):
  - Phase-stay note:
    this slice continued Phase 15 after the imported-member resolution index commit `ddfce15`. It
    did not jump to Phase 14 or Phase 17 because the work is profile-backed parser/runtime
    performance on the already cut-over Go runtime.
  - AVmatrix usage and impact:
    the packaged Go binary refreshed the AVmatrix graph before graph-based work. AVmatrix impact
    reported `countNodes` LOW risk with direct caller `Pool.Parse`. AVmatrix impact reported
    `PoolOptions` CRITICAL because it is shared by analyze, providers, resolution tests, and HTTP
    paths; the blast radius was handled by keeping the zero-value default production-safe and making
    node counting opt-in.
  - Implementation:
    `parser.PoolOptions.CountNodes` controls diagnostic node counting. `Pool.Parse` now sets
    `Result.NodeCount=0` by default and only walks the syntax tree when `CountNodes` is true. The
    parser test that asserts `NodeCount` explicitly enables the option, and a new test verifies the
    default skips node counting.
  - Focused benchmark:
    `BenchmarkPoolParseNodeCount` after the full build measured disabled `221-297us/op`,
    `1032-1339 B/op`, and `17 allocs/op` vs enabled `276-387us/op`, `5352-5416 B/op`, and
    `152 allocs/op`. This confirms the default analyze path avoids the diagnostic tree walk and its
    extra node allocations.
  - Large-repo benchmark:
    baseline `.tmp\phase15-next-after-import-index-start-graph.json` recorded total `29,088.0ms`,
    parse `18,428.5ms`, parser `totalDuration=8,646.6ms`, graph rows `33,804` / `67,311`, and DB
    fallback/skipped `0`. Final `.tmp\phase15-nodecount-final-analyze.json` recorded total
    `29,714.9ms`, parse `19,051.1ms`, parser `totalDuration=7,586.2ms`, graph rows `33,843` /
    `67,328`, and DB fallback/skipped `0`. `benchmark-compare` reported total `+2.2%`, so this
    slice does not claim an overall wall-time speedup.
  - CPU pprof:
    before profile `.tmp\phase15-next-after-import-index-start-cpu.pprof` showed
    `internal/parser.countNodes` at `1.58s` cumulative and `Pool.Parse` at `8.28s`. Final profile
    `.tmp\phase15-nodecount-final-cpu.pprof` no longer showed `countNodes` in the top table and
    showed `Pool.Parse` at `7.58s`. Remaining CPU cost is still dominated by cgo-backed
    tree-sitter/provider traversal and native DB work: `runtime.cgocall`, `Parser.ParseWithOptions`,
    `tsjs.Extract`, `golang.Extract`, `CommitLoadTransaction`, and `runCopy`.
  - Validation order and results:
    final validation followed the required order: full launcher build first (`45,641.2ms`), focused
    parser tests after build (`2,508.1ms`), after-build parser benchmark (`14,867ms` package time),
    packaged analyze plus CPU/memory pprof (`32,432.8ms` wall), full Go tests (`32,805.6ms`),
    browser E2E through packaged Go backend and isolated `AVMATRIX_HOME` (`33/33` passed; isolated
    analyze `29,087.9ms`; Playwright `206,434.9ms`), and `cd avmatrix && npm test`
    (`400,152.3ms`). An earlier E2E analyze run is explicitly rejected from evidence because it used
    the PowerShell reserved `$HOME` variable and therefore did not use the intended isolated
    AVmatrix home.
  - Optimization conclusion:
    `[P15-PARSER-NODECOUNT-OPT]` is closed because the targeted diagnostic node-count walk was
    removed from the default analyze path and CPU pprof. Phase 15 remains open; the next real
    targets are tree-sitter/provider parse cgo and remaining native DB commit/COPY cost.

- Phase 15 TS/JS provider traversal optimization (2026-05-14):
  - Phase-stay note:
    this slice continued Phase 15 after the parser node-count commit `bc39798`. It did not jump to
    Phase 14 or Phase 17 because the work is profile-backed provider performance on the already
    cut-over Go runtime.
  - AVmatrix usage and impact:
    the packaged Go binary refreshed the AVmatrix graph before graph-based work. AVmatrix context
    showed `tsjs.Extract` is called by `extractScopeIR`, SFC extraction, and resolution tests.
    AVmatrix impact reported `Extract` CRITICAL because it feeds analyze/runtime/test flows; the
    targeted collector methods `emitDefinition`, `emitReference`, `emitTypeBinding`,
    `buildContext`, and `collectScopes` were LOW. The blast radius was handled by keeping the
    change inside the TS/JS collector and preserving wrapper methods.
  - Implementation:
    `walkKind` computes a node kind once per visited node and passes it to the collector.
    `collectScopesAndContext` combines the previous scope and context pre-passes. The main
    extraction pass calls kind-aware emitter helpers so `emitDefinition`, `emitImport`,
    `emitTypeBinding`, and `emitReference` no longer each call `node.Kind()` for every node.
  - Focused benchmark:
    `BenchmarkExtractTypeScriptScopeIR` moved from `446-466us/op`, about `87.3KB/op`, and
    `1966 allocs/op` to `295-337us/op`, `68.3KB/op`, and `996 allocs/op` after the full build.
  - Large-repo benchmark:
    baseline `.tmp\phase15-next-after-nodecount-start-graph.json` recorded total `27,545.4ms`,
    parse `17,141.5ms`, parser `totalDuration=6,947.2ms`, graph rows `33,844` / `67,329`, and DB
    fallback/skipped `0`. Final `.tmp\phase15-tsjs-traversal-final-analyze.json` recorded total
    `25,728.2ms`, parse `14,549.4ms`, parser `totalDuration=6,777.7ms`, graph rows `33,828` /
    `67,364`, and DB fallback/skipped `0`. `benchmark-compare` reported total `-6.6%` and parse
    `-15.1%`; DB load moved up by `6.5%`, so remaining DB cost is still a Phase 15 target.
  - CPU pprof:
    before profile `.tmp\phase15-next-after-nodecount-start-cpu.pprof` showed `tsjs.Extract` at
    `6.35s`, `tsjs.walk` at `6.28s`, `Node.Kind` at `2.27s`, and `NamedChild` at `2.28s`. Final
    profile `.tmp\phase15-tsjs-traversal-final-cpu.pprof` showed `tsjs.Extract` at `4.08s`,
    `walkKind` at `3.99s`, `Node.Kind` at `0.92s`, and `NamedChild` at `1.59s`.
  - Validation order and results:
    final validation followed the required order: full launcher build first (`40,658.1ms`), focused
    TS/JS/SFC/Vue/Astro/Svelte/resolution tests after build (`9,854.5ms`), after-build TS/JS
    benchmark (`12,129.3ms`), packaged analyze plus CPU/memory pprof (`27,697.2ms` wall), full Go
    tests (`29,375.2ms`), browser E2E through packaged Go backend and isolated `AVMATRIX_HOME`
    (`33/33` passed; isolated analyze `25,467.0ms`; Playwright `194,306.0ms`), and
    `cd avmatrix && npm test` (`343,081.8ms`).
  - Optimization conclusion:
    `[P15-TSJS-TRAVERSAL-KIND-OPT]` is closed because it reduced the real TS/JS provider traversal
    hotspot and preserved graph/DB fail-closed cleanliness. Phase 15 remains open for native
    tree-sitter parse cgo, remaining provider traversal, `golang.Extract`, and native DB
    commit/COPY cost.

- Phase 15 Go provider traversal optimization (2026-05-14):
  - Phase-stay note:
    this slice continued Phase 15 after the TS/JS provider traversal commit `108010d`. It did not
    jump to Phase 14 or Phase 17 because the work is profile-backed provider performance on the
    already cut-over Go runtime.
  - AVmatrix usage and impact:
    the packaged Go binary refreshed the AVmatrix graph before graph-based work. AVmatrix impact
    reported `internal/providers/golang.Extract` LOW with one direct caller (`extractScopeIR`) and
    two affected flows (`extractScopeIR`, `newAnalyzeCommand`). The targeted collector methods
    `collectScopes`, `buildContext`, `emitDefinition`, `emitImport`, `emitTypeBinding`, and
    `emitReference` were LOW. `walk` reported HIGH because it is the Go provider traversal helper
    under `Extract`; the implementation handled that by preserving `walk` and adding `walkKind`
    rather than changing all traversal users.
  - Implementation:
    `walkKind` computes a node kind once per visited node and passes it to the collector.
    `collectScopesAndContext` combines the previous scope and context pre-passes. The main
    extraction pass calls kind-aware emitter helpers so `emitDefinition`, `emitImport`,
    `emitTypeBinding`, and `emitReference` no longer each call `node.Kind()` for every node.
    `emitTypeReferences` also uses `walkKind` for nested type-node walks.
  - Focused benchmark:
    `BenchmarkExtractGoScopeIR` moved from `549-572us/op`, about `106.1KB/op`, and
    `2379 allocs/op` to `380-500us/op`, `85.2KB/op`, and `1310 allocs/op` after the full build.
  - Large-repo benchmark:
    baseline `.tmp\phase15-next-after-tsjs-traversal-start-graph.json` recorded total
    `24,607.2ms`, parse `14,595.9ms`, parser `totalDuration=6,749.9ms`, graph rows `33,829` /
    `67,365`, and DB fallback/skipped `0`. Final `.tmp\phase15-go-provider-traversal-analyze.json`
    recorded total `24,211.0ms`, parse `13,943.3ms`, parser `totalDuration=6,938.5ms`, graph rows
    `33,843` / `67,378`, and DB fallback/skipped `0`. `benchmark-compare` reported total `-1.6%`
    and parse `-4.5%`; DB load moved up by `5.1%`, so native DB load remains a separate Phase 15
    target.
  - CPU/heap pprof:
    before profile `.tmp\phase15-next-after-tsjs-traversal-start-cpu.pprof` showed
    `golang.Extract` at `1.95s`, `golang.walk` at `1.95s`, `golang.Extract.func1` at `1.10s`, and
    `Node.Kind` at `1.14s`. Final profile `.tmp\phase15-go-provider-traversal-cpu.pprof` showed
    `golang.Extract` at `1.14s`, `golang.walkKind` at `1.12s`, `golang.Extract.func1` at `0.56s`,
    and `Node.Kind` at `0.66s`. Heap profile moved Go provider `Extract` from `24.40MB` to
    `19.91MB`.
  - Validation order and results:
    final validation followed the required order: full launcher build first (`39,231.7ms`), focused
    Go-provider/analyze/resolution tests after build (`6,337.4ms`), after-build Go provider
    benchmark (`10,258.5ms`), packaged analyze plus CPU/memory pprof (`26,449.5ms` wall), full Go
    tests (`34,494.9ms`), browser E2E through packaged Go backend and isolated `AVMATRIX_HOME`
    (`32` passed / `1` skipped; isolated analyze `34,598.5ms`; Playwright `436,101.7ms`), and
    `cd avmatrix && npm test` (`404,491.2ms`, exit code `0`).
  - Optimization conclusion:
    `[P15-GO-PROVIDER-TRAVERSAL-KIND-OPT]` is closed because it reduced the real Go provider
    traversal hotspot and preserved graph/DB fail-closed cleanliness. Phase 15 remains open for
    native tree-sitter parse cgo, residual TS/JS provider traversal, and native LadybugDB
    query/commit/COPY cost.

- Phase 15 graph snapshot stream memory optimization (2026-05-14):
  - Phase-stay note:
    this slice continued Phase 15 after the Go provider traversal commit `c7a746c`. It did not jump
    to provider coverage, fallback/schema correctness, or Phase 17 cutover because the target is a
    pprof-backed memory issue in the already cut-over Go analyze runtime.
  - AVmatrix usage and impact:
    the packaged Go binary refreshed the AVmatrix graph before graph-based work. AVmatrix impact
    reported `internal/analyze.writeGraphSnapshot` CRITICAL because the symbol is called from
    analyze `Run` and participates in the `newAnalyzeCommand` / `main` execution flows. The change
    was therefore limited to snapshot writing internals and preserved the `graph.json` object shape
    plus temp-file/rename behavior.
  - Implementation:
    `writeGraphSnapshot` now creates the temp file, streams `nodes` and `relationships` through a
    `bufio.Writer`, flushes, closes, and only then renames the temp file. Per-item JSON remains
    indented and the snapshot is still valid `graph.Graph` JSON. `TestRunOrchestrates...` now reads
    the snapshot back and checks node/relationship counts against the in-memory graph.
  - Focused benchmark:
    `BenchmarkWriteGraphSnapshot` was added for this slice and measured `28-45ms/op`, about
    `3.54MB/op`, and `32,527-32,531 allocs/op` after the full build. There was no old micro
    baseline for this new benchmark; the accepted before/after evidence is the macro heap profile.
  - Large-repo benchmark:
    baseline `.tmp\phase15-next-after-go-provider-start-graph.json` recorded total `25,256.0ms`,
    parse `14,643.6ms`, DB load `5,632.8ms`, graph rows `33,844` / `67,379`, DB fallback/skipped
    `0`, and `maxObservedSys=822,550,776`. Final
    `.tmp\phase15-graph-snapshot-stream-final-analyze.json` recorded total `27,335.2ms`, parse
    `15,004.9ms`, DB load `6,858.4ms`, graph rows `33,857` / `67,412`, DB fallback/skipped `0`,
    and `maxObservedSys=632,242,424`. `benchmark-compare` reported total `+8.2%` because native DB
    load moved `+21.8%`; this is not claimed as a macro wall-time speedup.
  - CPU/heap pprof:
    baseline heap profile `.tmp\phase15-next-after-go-provider-start-mem.pprof` showed
    `bytes.growSlice=64MB` flat/cumulative under `encoding/json.MarshalIndent` from
    `writeGraphSnapshot`, with heap inuse total `178.39MB`. Final heap profile
    `.tmp\phase15-graph-snapshot-stream-final-mem.pprof` no longer showed that full-snapshot
    marshal buffer in the top table and heap inuse total moved to `135.09MB`. CPU stayed roughly
    neutral: `writeGraphSnapshot` moved from `0.86s` to `0.82s`.
  - Validation order and results:
    final validation followed the required order: full launcher build first (`40,160.2ms`), focused
    analyze/CLI/HTTP/MCP tests after build (`21,748.4ms`), after-build graph snapshot benchmark
    (`21,757.3ms`), packaged analyze plus CPU/memory pprof (`29,806.5ms` wall), full Go tests
    (`31,950.0ms`), browser E2E through packaged Go backend and isolated `AVMATRIX_HOME`
    (`31` passed / `1` skipped / `1` flaky recovered on retry; isolated analyze `26,815.2ms`;
    Playwright `528,159.0ms`), and `cd avmatrix && npm test` (`414,123.7ms`, exit code `0`).
  - Optimization conclusion:
    `[P15-GRAPH-SNAPSHOT-STREAM-OPT]` is closed because it removed the full-graph snapshot buffer
    from the accepted heap profile while preserving graph/DB fail-closed cleanliness. Phase 15
    remains open for native tree-sitter parse cgo, native LadybugDB query/commit/COPY cost,
    residual TS/JS traversal, and residual graph/resolution allocations.

- Phase 10 TypeAlias->Method LadybugDB schema gap fix (2026-05-14):
  - Phase-jump note:
    this slice interrupted the active Phase 15 performance work because the user reproduced a real
    Web UI repo-selection failure on `F:\Restaurant_manager`. The error was
    `db_load phase: copy relationships TypeAlias->Method: schema pair unsupported`. This is
    persistence/schema correctness, not performance optimization and not a case for fallback.
  - Reproduction:
    before the patch, packaged analyze from `F:\Restaurant_manager` failed closed in `25,374.4ms`.
    This is the intended fail-closed behavior from the earlier fallback correctness work: the
    runtime refused to silently skip or fallback-insert an unsupported schema pair.
  - AVmatrix usage and impact:
    the packaged Go binary refreshed the AVmatrix graph before graph-based work. AVmatrix impact
    reported `internal/lbugschema.RelationPairs` LOW, but
    `internal/lbugload.relationPairSupported` CRITICAL because it flows through `ExportGraphCSVs`,
    `loadGraph`, analyze `Run`, `newAnalyzeCommand`, and Web analyze paths.
  - Implementation:
    `internal/lbugschema.RelationPairs` now includes `TypeAlias -> Method`. Schema DDL tests assert
    `FROM \`TypeAlias\` TO Method`. The loader supported-COPY regression test now includes a
    `TypeAlias` node, a `Method` node, and a `HAS_METHOD` relationship, then verifies the normal
    relationship COPY query uses `from="TypeAlias", to="Method"` and no fallback insert runs.
  - Benchmark:
    after the patch and full launcher build, packaged analyze on `F:\Restaurant_manager` succeeded
    in `30,931.7ms` wall / `28,889.6ms` benchmark total. It scanned `6,198` files, parsed `1,228`,
    marked `4,970` unsupported, failed `0`, and wrote `77,901` node rows plus `129,560`
    relationship rows with `fallbackInsertFailures=0` and `skippedRelationships=0`. Phase timings:
    parse `13,850.7ms`, DB load `8,018.7ms`, resolution `1,051.2ms`.
  - Focused benchmark:
    `BenchmarkExportGraphCSVs` recorded `16.05-22.24ms/op`, about `285KB/op`, and `1403 allocs/op`.
    `BenchmarkLoadCSVExportCopyPathNoop` recorded `2.87-3.28us/op`, `1,360-1,376 B/op`, and
    `17 allocs/op`.
  - Validation order and results:
    full launcher build was run first and passed in `56,762.9ms` after stopping the stale
    launcher/backend processes that held the bundle lock; focused schema/load tests passed in
    `4,284.6ms`; full Go tests passed in `61,720.4ms`. Browser E2E used isolated
    `AVMATRIX_HOME=F:\AVmatrix-GO\.tmp\typealias-method-e2e-home-20260514-unique`, packaged Go
    backend on `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated current-repo analyze took
    `27,291.0ms`, graph rows were `33,858` / `67,413`, DB fallback/skipped `0`, and Playwright
    passed with `32` passed / `1` skipped in `475,727.0ms`. `cd avmatrix && npm test` passed with
    exit code `0` in `440,065.0ms`.
  - Follow-up:
    the stale process lock uncovered a separate launcher lifecycle UX bug. That is tracked as
    `[P16-LAUNCHER-UI-CLOSE-PROCESS-LIFECYCLE]` and must be fixed as launcher/cutover behavior,
    not hidden inside this persistence schema fix.

- Phase 16 launcher UI-close process lifecycle fix (2026-05-14):
  - Phase-jump note:
    this slice intentionally stayed out of Phase 15 because the issue is launcher/cutover UX, not
    performance. While testing Web UI repo selection, a closed UI left launcher/backend processes
    running and holding locks under `avmatrix-launcher\server-bundle`. That breaks build/update
    behavior and must be treated as a real lifecycle bug, not a manual-test artifact.
  - AVmatrix usage and impact:
    AVmatrix impact reported LOW risk for `startRuntime`, `staticHandler`, `waitForExit`, and
    `openBrowser`. The changed surface is the Go launcher process that serves the packaged Web UI
    and starts the packaged backend; direct user-started `avmatrix serve` sessions are not killed.
  - Implementation:
    when the launcher owns the Web UI server, it injects a launcher-only heartbeat script into
    `index.html`. The browser page sends periodic heartbeats to
    `/__avmatrix_launcher/heartbeat` and sends `/__avmatrix_launcher/closed` on `pagehide`.
    `waitForExit` now listens for the UI-done channel in addition to backend exit and OS signals.
    When UI close is detected, the launcher returns through the existing defers, shutting down the
    Web server and stopping only the backend PID it started. `AVMATRIX_LAUNCHER_NO_BROWSER=1` is
    test-only and leaves normal launcher UX unchanged.
  - Unit/focused validation:
    after the final full launcher build, `go test ./... -count=1` in `avmatrix-launcher/src` passed
    in `5,110.9ms`, and `go test ./... -count=1` in `avmatrix-launcher/server-wrapper` passed in
    `2,052.4ms`. The tests cover lifecycle script injection, heartbeat handling, heartbeat timeout,
    close grace, and browser-open suppression for automated smoke.
  - Build and smoke validation:
    full launcher build passed in `47,045.1ms`. HTTP close-flow smoke started
    `AVmatrixLauncher.exe` hidden with `AVMATRIX_LAUNCHER_NO_BROWSER=1`, verified the backend and
    launcher-served Web UI became ready, verified the served index contained
    `data-avmatrix-launcher-lifecycle`, posted heartbeat and closed events, and completed in
    `15,647.2ms`. After close, no `AVmatrixLauncher.exe`, `avmatrix-server.exe`, or packaged
    `avmatrix.exe` process remained under `avmatrix-launcher`, and both packaged binaries opened
    with exclusive file locks.
  - Browser E2E validation:
    Playwright close-flow E2E started the packaged launcher hidden, opened Chromium to the
    launcher-served Web UI, verified the lifecycle script, closed Chromium, waited for the launcher
    process to exit, then verified no launcher bundle processes or binary locks remained. Runtime:
    `27,569ms`. `cd avmatrix && npm test` passed with exit code `0` in `415,943.7ms`.
  - Conclusion:
    `[P16-LAUNCHER-UI-CLOSE-PROCESS-LIFECYCLE]` is closed because a launcher-owned UI session now
    has an explicit heartbeat lifetime and closing the browser cleans up launcher-owned backend
    processes. Phase 15 can resume after this lifecycle commit.

- Phase 15 ScopeIR owned normalization allocation optimization (2026-05-14):
  - Phase-stay note:
    this slice resumed Phase 15 after the Phase 10 `TypeAlias->Method` schema correctness jump and
    the Phase 16 launcher lifecycle cleanup. It is pprof-backed provider/runtime allocation work,
    not provider coverage and not a cutover authority gate.
  - AVmatrix usage and impact:
    packaged Go AVmatrix refreshed the graph before graph-based work. AVmatrix impact reported
    `ScopeIR.Normalized` LOW, with direct callers `MarshalDeterministic`, `Unmarshal`, and the
    normalization benchmark. TS/JS and Go `collector.result` impact were LOW. The implementation
    keeps the existing defensive `Normalized()` behavior and uses the owned path only in providers
    that have just built fresh IR facts.
  - Rejected candidate:
    pure in-place normalization was benchmarked and rejected. It reduced micro allocation more
    aggressively, but full-repo packaged analyze retained oversized append backing arrays:
    `.tmp\phase15-scopeir-inplace-normalize-final.json` reported
    `endAllocBytes=204,682,944` and `maxObservedSys=839,073,016`, worse than the baseline. That
    candidate was not kept.
  - Implementation:
    `scopeir.NormalizeOwned` compacts the top-level IR fact slices before sorting owned provider
    data. TS/JS and Go provider `collector.result` now use `NormalizeOwned`. Tests assert
    `Normalized()` still does not mutate source data and that both `NormalizeInPlace` and
    `NormalizeOwned` match `Normalized()` output.
  - Focused benchmark:
    after the full launcher build, `BenchmarkExtractTypeScriptScopeIR` moved from about
    `68.3KB/op`, `996 allocs/op` to about `66.4KB/op`, `980 allocs/op`. `BenchmarkExtractGoScopeIR`
    moved from about `85.2KB/op`, `1310 allocs/op` to about `82.8KB/op`, `1281 allocs/op`.
    `BenchmarkScopeIRNormalizedLargeSort` remained around `4.4-5.3ms/op` and about `4.06MB/op`,
    confirming the old non-mutating path is still intact.
  - Large-repo benchmark:
    baseline `.tmp\phase15-select-next-target.json` recorded total `30,058.4ms`, parse
    `14,450.4ms`, DB load `8,278.5ms`, graph rows `33,909` / `67,556`, fallback/skipped `0`, and
    `maxObservedSys=745,119,992`. Final
    `.tmp\phase15-scopeir-owned-normalize-final.json` recorded total `24,982.3ms`, parse
    `14,202.9ms`, DB load `6,046.2ms`, graph rows `33,918` / `67,609`, fallback/skipped `0`, and
    `maxObservedSys=743,596,280`. `benchmark-compare` reported total `-16.9%`; this is recorded as
    noisy because native DB load moved `-27.0%`. The accepted claim is the provider allocation
    reduction with neutral full-repo memory, not a macro speedup.
  - CPU/heap pprof:
    baseline heap pprof showed `ScopeIR.Normalized=54.01MB` flat. Final heap pprof attributes the
    retained top-level compact arrays to `ScopeIR.NormalizeOwned=57.75MB` flat, while
    `maxObservedSys` stayed effectively neutral (`745,119,992` -> `743,596,280`). CPU remains
    dominated by native tree-sitter parse and LadybugDB query/commit/COPY costs.
  - Validation status:
    full launcher build was run first and passed in `38,220.6ms`; focused scopeir/TSJS/Go provider
    tests passed; TS/JS, Go provider, and ScopeIR benchmarks passed; packaged analyze with CPU/heap
    pprof passed in `26,867.1ms` wall; full Go tests passed in `35,542.3ms`. Browser E2E used
    isolated `AVMATRIX_HOME=F:\AVmatrix-GO\.tmp\phase15-owned-normalize-e2e-home-20260514`, the
    packaged Go backend on `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated analyze
    benchmark total was `28,503.9ms`, graph rows were `33,919` / `67,610`, DB fallback/skipped was
    `0`, Playwright `.last-run.json` recorded `status: passed`, `npx playwright test --list`
    listed `33` tests in `6` files, and the E2E command wall was `590.4s`. `cd avmatrix &&
    npm test` passed in `461,316.4ms`. AVmatrix `detect_changes(scope=all)` reported MEDIUM risk
    with the only affected process `NormalizeInPlace -> CompareInt`, matching the intended
    normalize/provider/doc slice.

- Phase 15 ScopeIR release-after-resolution retained-heap optimization (2026-05-14):
  - Phase-stay note:
    this slice stayed in Phase 15 because it is pprof-backed memory/performance work on the
    cut-over Go analyze runtime. It did not jump to Phase 10 persistence, Phase 14 provider
    coverage, or Phase 17 runtime authority.
  - AVmatrix usage and impact:
    AVmatrix impact reported `analyze.Run` CRITICAL and `newAnalyzeCommand` HIGH because these are
    core analyze/CLI orchestration paths. The patch is therefore limited to an opt-in
    `ReleaseScopeIRsAfterResolution` option, enabled by CLI and HTTP analyze after resolution has
    consumed ScopeIRs. Direct `analyze.Run` callers keep the default retained result behavior. The
    touched E2E spec files were LOW-risk validation hardening.
  - Implementation:
    after successful `resolution.ResolveBoundInto`, CLI/Web analyze can drop `result.ScopeIRs`,
    `parsedFiles.IRs`, and the cross-file binding result before MRO/community/process/DB phases.
    This keeps graph output and metrics intact while removing parsed ScopeIRs from the retained
    result heap for normal runtime analyze flows.
  - Large-repo benchmark:
    baseline `.tmp\phase15-after-owned-normalize-start.json` recorded total `29,665.3ms`, parse
    `14,751.4ms`, resolution `960.8ms`, DB load `8,835.6ms`, graph rows `33,919` / `67,610`,
    fallback/skipped `0`, `endAllocBytes=170,499,696`, and `maxObservedSys=709,378,296`. Final
    `.tmp\phase15-release-scopeir-after-resolution.json` recorded total `24,476.3ms`, parse
    `13,746.7ms`, resolution `904.5ms`, DB load `5,956.6ms`, graph rows `33,928` / `67,617`,
    fallback/skipped `0`, `endAllocBytes=80,059,072`, and `maxObservedSys=713,580,792`.
  - CPU/heap pprof:
    baseline heap pprof showed `131.94MB` in use with `ScopeIR.NormalizeOwned=55.67MB` flat. Final
    heap pprof showed `58.67MB` in use and no `ScopeIR.NormalizeOwned` entry in the top table. CPU
    pprof remains dominated by native tree-sitter cgo and LadybugDB query/COPY/commit work, so the
    accepted claim is retained heap reduction after resolution, not a peak-RSS or macro wall-time
    speedup.
  - Browser E2E hardening:
    full E2E before hardening exposed an async process-highlight assertion flake while the same
    tests passed when rerun directly. The specs now wait for the highlight button state before
    checking row styling. This changes validation tolerance only, not runtime behavior.
  - Validation status:
    final validation followed the required order: full launcher build first (`44.3s`), packaged
    analyze with CPU/heap pprof, full Go tests (`31.7s`), browser E2E through packaged Go backend
    and isolated `AVMATRIX_HOME` (`32` passed / `1` skipped; Playwright `441,444.3ms`; isolated
    analyze total `29,141.3ms`, graph rows `33,928` / `67,617`, fallback/skipped `0`), Prettier
    check for the touched E2E specs, and `cd avmatrix && npm test` (`571.4s`).

- Phase 15 graph compact-after-processes retained-heap optimization (2026-05-14):
  - Phase-stay note:
    this slice stayed in Phase 15 because it is pprof-backed memory/performance work on the
    cut-over Go analyze runtime. It did not jump to Phase 10 persistence, Phase 14 provider
    coverage, or Phase 17 runtime authority.
  - AVmatrix usage and impact:
    after the previous commit, packaged Go AVmatrix refreshed the graph. AVmatrix impact reported
    `Graph` CRITICAL (`357` impacted symbols, `376` processes), `Graph.AddRelationship` CRITICAL
    (`130` impacted symbols, `300` processes), and `analyze.Run` CRITICAL (`4` impacted symbols,
    `51` processes). The patch therefore avoids changing add/dedupe/lookup semantics.
  - Implementation:
    `Graph.Compact()` trims `Nodes` and `Relationships` to `cap=len`, then clears `nodeIndex` and
    `relIndex`. Existing `GetNode`, `GetRelationship`, and later `Add*` calls rebuild those indexes
    lazily. `analyze.Run` calls `Compact()` after Phase Processes, the last normal graph mutation
    phase, before DB load and graph snapshot.
  - Large-repo benchmark:
    baseline `.tmp\phase15-release-scopeir-after-resolution.json` recorded total `24,476.3ms`,
    parse `13,746.7ms`, resolution `904.5ms`, DB load `5,956.6ms`, graph rows `33,928` /
    `67,617`, fallback/skipped `0`, `endAllocBytes=80,059,072`, and
    `maxObservedSys=713,580,792`. Final `.tmp\phase15-graph-compact-after-processes.json`
    recorded total `28,249.7ms`, parse `17,855.3ms`, resolution `987.6ms`, DB load `5,794.7ms`,
    graph rows `33,982` / `67,657`, fallback/skipped `0`, `endAllocBytes=87,163,600`, and
    `maxObservedSys=623,452,408`.
  - CPU/heap pprof:
    baseline heap pprof showed `58.67MB` in use with `Graph.AddRelationship=15.77MB` flat. Final
    heap pprof showed `49.64MB` in use; `Graph.AddRelationship` disappeared from the top table and
    `Graph.Compact` retained `11.48MB` flat for compact graph backing slices. CPU remained dominated
    by native tree-sitter cgo and LadybugDB query/COPY/commit work, so this is a retained heap win,
    not a macro speedup claim.
  - Validation status:
    final validation followed the required order: full launcher build first (`40.7s`), focused
    graph/analyze tests, `BenchmarkWriteGraphSnapshot` (`22.6-26.5ms/op`, about `3.54MB/op`),
    packaged analyze with CPU/heap pprof, full Go tests (`31.0s`), browser E2E through packaged Go
    backend and isolated `AVMATRIX_HOME` (`32` passed / `1` skipped; Playwright `434,795.2ms`;
    isolated analyze total `25,136.7ms`, graph rows `33,982` / `67,657`, fallback/skipped `0`),
    and `cd avmatrix && npm test` (`477.1s`). The E2E wrapper had a cleanup exit-code quirk after
    Playwright printed exit `0`; `.last-run.json` was `passed` and a follow-up port check found no
    listeners on `4747` or `5173`.

- Phase 15 native cgo boundary classification (2026-05-14):
  - Phase-stay note:
    this is Phase 15 performance target triage, not a phase jump. It records why the next work
    should not pretend native cgo parser or LadybugDB costs are solved by another local Go-only
    micro-patch.
  - Evidence:
    `.tmp\phase15-graph-compact-after-processes-cpu.pprof` shows `runtime.cgocall=19.25s` flat /
    `20.49s` cumulative, `Parser.ParseWithOptions=8.07s` cumulative,
    `lbugnative.Query=5.66s` cumulative, `lbugload.runCopy=1.94s` cumulative, and
    `CommitLoadTransaction=3.23s` cumulative. These are the largest remaining CPU families after
    the graph compact slice.
  - Decision:
    native tree-sitter parse cgo and native LadybugDB query/COPY/commit are classified and rejected
    as same-slice Go-level optimization targets for this plan. Further progress here requires
    upstream/native parser or LadybugDB API/design changes.
  - Remaining Go-level work:
    the same CPU profile still shows `tsjs.Extract=4.98s` cumulative and
    `resolution.ResolveBoundInto=0.91s` cumulative, while heap pprof shows
    `emitDefinitionNodes=17,412.09kB` flat / `22,020.99kB` cumulative and
    `GenerateID=7,168.68kB` flat. Those remain the next Phase 15 Go-level candidates.
  - Validation status:
    no runtime code changed in this classification slice. The evidence relies on the immediately
    preceding validated runtime package: full launcher build first (`40.7s`), packaged analyze with
    CPU/heap pprof, full Go tests (`31.0s`), browser E2E (`32` passed / `1` skipped), and
    `cd avmatrix && npm test` (`477.1s`).

- Phase 15 TS/JS fact-kind dispatch deferred evidence (2026-05-14):
  - Deferral note:
    this slice is intentionally deferred and the attempted patch was reverted from the working tree.
    It is not active plan work and is not a reason to keep optimizing before independent MCP/tool
    readiness is proven.
  - Impact:
    AVmatrix reported `internal/providers/tsjs.Extract` as CRITICAL (`8` impacted symbols,
    `10` affected processes), directly touching `extractScopeIR` and SFC extraction.
  - Attempted implementation:
    the uncommitted patch kept the shared `walkKind` traversal, but routed through
    `collector.emitFactKind` so each AST kind called only the relevant definition/import/type-binding
    or reference emitter family. The patch was not retained.
  - Evidence captured:
    full launcher build ran before tests (`37.9s` after the patch), focused TS/JS provider parity
    tests passed (`go test ./internal/providers/tsjs -run TestExtract -count=1`, `0.242s`),
    `BenchmarkExtractTypeScriptScopeIR` moved from median `315,726ns/op` to `313,876ns/op` with
    allocations unchanged at `66,385B/op` and `980 allocs/op`, and packaged analyze completed with
    `.tmp\phase15-tsjs-kind-dispatch.json`: total `25,430.1ms`, rows `33,985` / `67,662`,
    fallback/skipped `0`.
  - Profile read:
    `.tmp\phase15-tsjs-kind-dispatch-cpu.pprof` still shows native cgo dominance
    (`runtime.cgocall=18.19s` flat / `19.19s` cumulative). The top visible Go-owned TS/JS frame is
    `collector.innermostScopeID=0.17s` flat / `0.20s` cumulative. Heap pprof shows a stronger next
    memory target in resolution/graph property allocation: `emitDefinitionNodes=15.50MB` flat /
    `22.00MB` cumulative and `graph.GenerateID=11.00MB` flat.
  - Decision:
    do not spend more Phase 15 time here now. Benchmark remains a measurement/regression aid, while
    the plan returns to Phase 17 to prove `AVmatrix-GO` is an independent Go MCP/tool implementation
    separate from the existing `AVmatrix-main` MCP.

### Phase 17 Non-Web TypeScript/JavaScript Audit Correction

- Date: `2026-05-14`.
- Reason:
  the plan goal was corrected from "the main Go runtime path works" to "the repository is converted
  to Go except for the Web UI display/build surface and Go-generated browser glue." A shallow MCP
  smoke would only prove the Go binary can speak MCP; it would not prove that non-Web
  TypeScript/JavaScript implementation has been converted.
- Phase-jump note:
  this returns from Phase 15 optimization/measurement back to Phase 17 conversion completeness.
  The issue is cutover authority and repository conversion scope, not benchmark optimization.
- Audit command:
  `rg --files -g '*.ts' -g '*.tsx' -g '*.js' -g '*.jsx' -g '*.mjs' -g '*.cjs'` with exclusions for
  `node_modules`, `dist`, `build`, `vendor`, `avmatrix-web/dist`, and
  `avmatrix-launcher/server-bundle`.
- Audit result:
  `1051` TypeScript/JavaScript-family files remain in source areas. Split by top-level path:
  `avmatrix=895`, `avmatrix-web=119`, `avmatrix-shared=34`, root Docker/web-server scripts `2`,
  and root ESLint config `1`. Inside `avmatrix`, the split is `test=542`, `src=339`, `scripts=8`,
  `vendor=4`, `hooks=1`, and `vitest.config.ts=1`.
- Interpretation:
  Web UI TypeScript/React is allowed, but the remaining non-Web inventory is not automatically
  acceptable. `avmatrix/src` still contains legacy CLI/server/MCP/analyzer/core TypeScript
  implementation; `avmatrix-shared` remains a TypeScript package; root and package scripts still
  contain Node build/support code; and the legacy Vitest harness still verifies TypeScript paths.
  These surfaces must be ported to Go, removed, or explicitly excluded as baseline fixtures with
  package/runtime proof.
- Plan correction:
  the Phase 17 gates for "No TypeScript/Node process remains required..." and "Any remaining
  TypeScript contract code is browser-only generated glue..." are reopened. A new blocker,
  `[P17-NON-WEB-TSJS-CONVERSION-BLOCKER]`, blocks independent MCP readiness until the non-Web
  TypeScript/JavaScript inventory has a concrete conversion/removal/exclusion package.

### Phase 17 Conversion Priority Correction

- Date: `2026-05-14`.
- Reason:
  the plan priority was clarified: `AVmatrix-GO` must become a correct, accurate, usable Go tool,
  not an endless benchmark optimization project and not a shallow "Go binary starts" cutover.
- Goal interpretation:
  Go is the implementation authority for the tool surfaces. The tool must analyze real repos
  correctly, preserve or improve graph accuracy, keep HTTP/MCP/CLI contracts usable, and be ready
  to run independently from `AVmatrix-main`. The target is for `AVmatrix-GO` to be correct and
  accurate enough to use for real work, and to be more accurate than the currently used
  `AVmatrix-main` where the conversion has identified legacy weaknesses; compatibility parity
  remains the floor for behavior that must stay the same.
- Benchmark interpretation:
  benchmark and evidence remain mandatory for validation slices. During conversion, benchmarks are
  used to prove the accepted speed gate, detect regressions, and guide light optimizations. Heavy
  optimization is deferred unless a benchmark proves a correctness, contract, runtime-shape, or
  unacceptable speed blocker. Current accepted large-repo evidence already shows Go faster, so the
  active work returns to correctness, completeness, and usability.
- Phase-jump decision:
  after this correction, work stays in Phase 17. The next step is not Phase 15 optimization and not
  a shallow MCP smoke. The next step is `[P17-NON-WEB-TSJS-CLASSIFICATION-MATRIX]`: classify the
  remaining non-Web TypeScript/JavaScript inventory into allowed Web UI, analyzer fixtures, legacy
  runtime implementation to port/delete, Node/Vitest harnesses to convert where practical,
  package/build glue, and generated browser contract glue before implementation resumes.

### Cross-Phase Reopen After Full-Conversion Correction

- Date: `2026-05-14`.
- Reason:
  the non-Web TypeScript/JavaScript audit showed that the issue is not limited to Phase 17 cutover
  packaging. Earlier phase gates also depended on the same incomplete assumption that the normal Go
  runtime path was sufficient.
- Reopened Phase 1:
  `[P1-CONTRACT-AUTHORITY-REOPENED-2026-05-14]` reopens contract authority because
  `avmatrix-shared`, legacy `avmatrix/src` contract consumers, package scripts, and Node/Vitest
  harnesses still need classification before the plan can say Go owns the full non-Web contract
  surface.
- Reopened Phase 16:
  `[P16-NON-WEB-NODE-AUDIT-REOPENED-2026-05-14]` reopens launcher/package no-Node proof beyond the
  running backend process. The existing launcher Go-backend proof remains valid, but final cutover
  still needs package/build/script/hook/test authority reviewed.
- Reopened Phase 15:
  `[P15-DEFER-UNTIL-CONVERSION-CORRECTNESS-2026-05-14]` reopens/defer-gates optimization sequencing.
  Completed Phase 15 slices remain benchmark evidence, but Phase 15 is not the active plan while
  conversion completeness and independent Go tool readiness remain open.
- Phase-jump rule:
  the next implementation work must start with `[P17-NON-WEB-TSJS-CLASSIFICATION-MATRIX]`, then
  jump back to Phase 1, Phase 16, Phase 15, or implementation phases only when the matrix assigns
  concrete work to that phase. Each jump must record why it moved, what evidence/benchmark is
  required, and what checklist item was opened or closed.

### Phase 17 Non-Web TypeScript/JavaScript Classification Matrix

- Date: `2026-05-14`.
- AVmatrix usage:
  confirmed Codex MCP `avmatrix` points at `F:\AVmatrix-main\avmatrix\dist\cli\index.js`, then used
  AVmatrix-main `status`, `list`, `analyze --force`, `query`, and `context` against `AVmatrix-GO`.
  The refresh artifact is `.tmp\phase17-tsjs-classification-avmatrix-main-refresh.json`.
- Phase-jump note:
  this stays in Phase 17 because it classifies conversion completeness before implementation. It
  does not jump to Phase 15 optimization and does not run an independent MCP smoke.
- Matrix:

| Category | Count | Classification | Required action |
| --- | ---: | --- | --- |
| `avmatrix-web` Web UI | `119` | Allowed TypeScript/React surface | Keep as Web UI display/build/test surface; ensure it calls Go backend contracts only. |
| Go-generated browser glue | `1` inside `avmatrix-web/src/generated` | Allowed generated TypeScript | Keep generated from Go; never make it backend/CLI/MCP/analyzer authority. |
| `avmatrix/src` legacy implementation | `339` | Non-Web TypeScript implementation | Remove from runtime/package authority after Go equivalents are proven; do not treat as completed conversion. |
| `avmatrix-shared` | `34` | Legacy TypeScript contract authority | Replace with Go-owned contracts/generated Web glue or exclude after all consumers are removed. |
| Analyzer fixture source files | `290` under `avmatrix/test/fixtures` | Source-language input data | May remain as analyzer fixtures; must be excluded from runtime/package authority. |
| Node/Vitest harness | `252` under `avmatrix/test` outside fixtures | Heavy legacy test runner surface | Convert runtime/analyzer coverage to Go tests where practical; keep only Web/browser harness where justified. |
| `avmatrix/scripts` | `8` | Package/build/test glue | Per-file decision: port to Go/native, remove with legacy TS package, or justify as minimal npm/Web ecosystem glue. |
| `avmatrix/hooks` | `1` | Non-Web support runtime | Port or prove it only delegates to Go and is required by editor ecosystem. |
| `avmatrix/vendor` JS/CJS | `4` | Legacy Node/vendor artifacts | Remove or exclude unless a remaining allowed Web/npm path proves it is required. |
| Root Docker/web server scripts | `2` | Runtime/support scripts | Prefer Go/native/static server replacement or classify as Web-only support with no tool runtime authority. |
| Root ESLint config | `1` | Dev-only lint config | May remain only for allowed Web/generated TS surfaces. |

- Result:
  `[P17-NON-WEB-TSJS-CLASSIFICATION-MATRIX]` is closed as a classification task. The conversion
  blocker remains open until the matrix groups are processed with implementation, evidence,
  benchmark/validation, and commits.

### Phase 17 Web Docker Node Server Removal

- Date: `2026-05-14`.
- Matrix slice:
  root Docker/web-server scripts from `[P17-NON-WEB-TSJS-CLASSIFICATION-MATRIX]`.
- AVmatrix usage:
  refreshed `AVmatrix-GO` with
  `.\\avmatrix\\bin\\avmatrix.exe analyze --force --skip-agents-md --no-stats --benchmark-json .tmp\\phase17-web-docker-nginx-preimpact-refresh.json`,
  then ran `detect_changes(scope=all)`, `query` for Docker/static-server concepts, and attempted
  `impact(target="avmatrix-launcher/build.ps1")`. The graph mapped no code execution flow for this
  support/config slice; `detect_changes` reported low risk and no affected processes. A final
  refresh was recorded at `.tmp\phase17-web-docker-node-removal-final-refresh.json`.
- Implementation:
  `Dockerfile.web` no longer copies or runs `docker-server.mjs`; the runtime stage now uses
  `nginx:alpine` and copies `docker/web-nginx.conf`. The nginx config preserves the Web UI static
  server contract: port `4173`, SPA fallback to `/index.html`, immutable cache headers for
  `/assets/`, and COOP/COEP headers for SharedArrayBuffer/WebGPU-sensitive browser behavior.
  `docker-server.mjs` and `docker-server.test.mjs` were deleted because the Node static server is no
  longer a cutover-path runtime.
- Build-gate hardening:
  the required full launcher build exposed that `avmatrix-launcher/build.ps1` could return success
  after native command failures. The script now checks `$LASTEXITCODE` after `go version`, Web build,
  backend Go build, launcher build, server-wrapper build, and protocol registration, and forces
  `CGO_ENABLED=1` only for the backend cgo build before restoring the caller environment.
- Environment prerequisites discovered:
  this machine had MSYS2 but no UCRT GCC package on PATH and Playwright Chromium was missing. Installed
  `mingw-w64-ucrt-x86_64-gcc` through MSYS2 pacman and Playwright Chromium through
  `npx playwright install chromium` to satisfy the documented native-build and E2E requirements.
- Validation:
  full launcher build passed after adding `C:\msys64\ucrt64\bin` to the build command PATH for the
  current process; `go test ./cmd/... ./internal/... -count=1` passed with `CGO_ENABLED=1`;
  static Dockerfile/nginx assertions passed; `npx playwright test e2e/server-connect.spec.ts
  --reporter=list` passed all `5` tests against packaged Go backend plus Vite.
- Docker daemon note:
  `docker version --format '{{.Server.Version}}'` failed because Docker Desktop's Linux engine pipe
  was unavailable. The image build smoke remains a daemon-available follow-up; this is recorded as
  environment-unavailable, not as a retained Node-runtime requirement.
- Remaining blocker:
  `[P17-NON-WEB-TSJS-CONVERSION-BLOCKER]` stays open for `avmatrix/src`, `avmatrix-shared`,
  Node/Vitest harnesses, package scripts, hooks, vendor JS/CJS, and final independent Go MCP/tool
  readiness.

### Phase 17 Claude Hook Go Translation And Retirement

- Date: `2026-05-14`.
- Matrix slice:
  Claude hook runtime support from `[P17-NON-WEB-TSJS-CONVERSION-BLOCKER]`.
- AVmatrix usage:
  refreshed `AVmatrix-GO` before graph-based work, then ran impact checks for
  `NewRootCommand`, `setupInstallClaudeHooks`, `setupMergeClaudeHookSettings`,
  `setupHookEntriesWithoutExisting`, `runClaudeHook`, and the legacy
  `installClaudeCodeHooks`. `NewRootCommand` returned HIGH blast radius because root CLI command
  registration fans into CLI flows and tests; the setup/hook helpers returned LOW. After
  validation, refreshed the graph again and ran final staged `detect_changes`, which reported
  `changed=90`, `affected=12`, `changed_files=14`, and `risk=HIGH` due to `NewRootCommand`.
- Translation/cutover:
  the legacy Claude hook behavior from `avmatrix/hooks/claude/avmatrix-hook.cjs` was translated
  into hidden Go command `avmatrix hook claude`. The Go command reads hook JSON from stdin,
  silently ignores invalid input, finds `.avmatrix` by walking up from `cwd`, handles PreToolUse
  for Grep/Glob/Bash by extracting the search pattern and running `augment -- <pattern>`, and
  handles PostToolUse for successful git mutations by comparing `git rev-parse HEAD` with
  `.avmatrix/meta.json` and emitting stale-index guidance with `--embeddings` when previous stats
  include embeddings.
- Setup/package retirement:
  Go setup and the legacy TypeScript setup shim now write `avmatrix hook claude` directly into
  Claude Code settings, replace old copied `avmatrix-hook.cjs` entries, and preserve user-owned
  settings. `avmatrix/package.json` no longer ships the `hooks` directory. The legacy
  `avmatrix/hooks/claude/avmatrix-hook.cjs`, `pre-tool-use.sh`, and `session-start.sh` files were
  deleted after the Go hook path was proven.
- Validation:
  full package build ran before tests and passed in `18,285.7ms`; Go tests
  `go test ./cmd/... ./internal/... -count=1` passed in `24,546.2ms`; `cd avmatrix && npx tsc
  --noEmit` passed in `8,745.7ms`; `cd avmatrix && npm test` passed in `375,380.3ms` and included
  the e2e/integration suites from the test runner. Direct Go hook smoke produced graph context for
  PreToolUse at `.tmp\phase17-claude-hook-pretool-20260514-221950.json` and stale-index guidance
  for PostToolUse at `.tmp\phase17-claude-hook-posttool-20260514-221950.json`. Temp-HOME setup
  smoke at `.tmp\phase17-claude-hook-setup-home-20260514-221950` wrote only
  `avmatrix hook claude` hook entries and no `avmatrix-hook.cjs` reference.
- Checklist/evidence result:
  `docs/plans/2026-05-08-avmatrix-go-typescript-node-to-go-conversion-remaining-files.md` now ticks
  `avmatrix/hooks/claude/avmatrix-hook.cjs`. The broader
  `[P17-NON-WEB-TSJS-CONVERSION-BLOCKER]` remains open for the remaining `avmatrix/src`,
  `avmatrix-shared`, Node/Vitest harnesses, package scripts, vendor JS/CJS, config, and final
  independent Go readiness work.
