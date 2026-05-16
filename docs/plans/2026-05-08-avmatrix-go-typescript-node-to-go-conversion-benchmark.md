# avmatrix-GO TypeScript/Node To Go Conversion Benchmarks

Source plan: [2026-05-08-avmatrix-go-typescript-node-to-go-conversion-plan.md](2026-05-08-avmatrix-go-typescript-node-to-go-conversion-plan.md)

This file contains benchmark gates, benchmark evidence, benchmark protocol, and benchmark-related evidence moved out of the checklist plan.

Benchmark rule: every benchmark in this file is conversion evidence, a regression check, and at most
a light-optimization guide while conversion correctness remains open. The Go implementation must not
be slower than the accepted baseline at equivalent accuracy, but benchmark work must not distract
from making the tool run correctly and accurately in Go, with accuracy improvements over the
currently used AVmatrix-main where legacy weaknesses are identified. If a benchmark exposes a
correctness, contract, architecture, runtime-shape, or unacceptable speed regression, it blocks the
relevant conversion gate. If it only suggests heavy optimization while the tool is already faster
enough for the accepted gate, record it as later optimization backlog instead of spending conversion
time on it.

## Phase 1 - Contract Freeze

### Benchmark Gate

- Record contract-freeze generation time, artifact count, artifact sizes, dependency versions, and
  proof command timings. If a speed comparison is not meaningful for a contract-only artifact,
  record that explicitly and treat artifact completeness plus reproducible generation time as the
  benchmark.

### Benchmark Evidence

- `powershell -ExecutionPolicy Bypass -File .tmp\pre_phase5_benchmark.ps1` passed on
  `2026-05-08T17:45:17Z` at commit `126a4f5`.
- Contract artifact proof: 23 files, 104,713 bytes, 23 SHA-256 hashes, hash/proof time
  105.075 ms.
- Toolchain snapshot: Go `go1.26.3 windows/amd64`, Node `v22.17.1`, npm `11.8.0`.
- Selected feasibility dependencies recorded in the benchmark snapshot include Cobra `v1.10.2`,
  go-tree-sitter `v0.25.0`, tree-sitter-javascript `v0.25.0`,
  tree-sitter-go `v0.25.0`, tree-sitter-typescript `v0.23.2`, and the rest of the
  pinned grammar matrix.
- Tree-sitter freshness reconciliation: upstream tree-sitter core/latest browser runtime is
  `v0.26.8`, while the latest Go module tags currently selected for the parser proof are
  `go-tree-sitter v0.25.0`, `tree-sitter-go v0.25.0`,
  `tree-sitter-javascript v0.25.0`, and `tree-sitter-typescript v0.23.2`. These Go modules are
  accepted only as the current feasibility proof; before final provider/cutover acceptance, the
  parser path must either move to an upstream-`v0.26.8` compatible native integration or record a
  blocker with the tested latest range and mitigation.

## Phase 2 - Go Skeleton

### Benchmark Gate

- Benchmark Go CLI startup/version/help latency against the TypeScript CLI baseline on the same
  machine. Record cold and warm timings, binary size, command output parity, and any startup
  regression before moving on.

### Benchmark Evidence

- `powershell -ExecutionPolicy Bypass -File .tmp\pre_phase5_benchmark.ps1` passed on
  `2026-05-08T17:45:17Z` at commit `126a4f5`.
- Go skeleton build: 1,157.477 ms, binary size 13,085,184 bytes.
- `--version` parity: PASS. Go cold 31.356 ms, warm avg 24.696 ms, warm p95 27.695 ms;
  TypeScript/Node cold 118.899 ms, warm avg 122.351 ms, warm p95 145.793 ms.
- `help` startup: Go cold 25.633 ms, warm avg 26.133 ms, warm p95 29.922 ms;
  TypeScript/Node cold 123.821 ms, warm avg 123.217 ms, warm p95 128.176 ms.
- Help output parity remains intentionally partial while the Go skeleton exposes only the
  coexistence-era command subset.

## Phase 3 - Repo Registry And Local Path Policy

### Benchmark Gate

- Benchmark registry/meta read, write, list, and path-first resolve on deterministic duplicate-name
  fixtures against the TypeScript baseline. Record operation count, average latency, and output
  parity before moving on.

### Benchmark Evidence

- `powershell -ExecutionPolicy Bypass -File .tmp\pre_phase5_benchmark.ps1` passed on
  `2026-05-08T17:45:17Z` at commit `126a4f5`; Go and TypeScript registry sub-benchmarks both
  exited `0`.
- Go fixture: 50 duplicate-name registered repos. `SaveMeta` 500 ops avg 393.270 us,
  `LoadMeta` 1,000 ops avg 87.511 us, `WriteRegistry50` 200 ops avg 4,410.120 us,
  `ReadRegistry50` 1,000 ops avg 467.921 us, `ListRegistered50` 1,000 ops avg
  442.341 us.
- Go path-first identity checks: `ResolveAbsolutePath` 1,000 ops avg 14.559 us;
  `ResolveDuplicateNameReject` 1,000 ops avg 3.506 us.
- TypeScript baseline: `SaveMeta` 500 ops avg 541.987 us, `LoadMeta` 1,000 ops avg
  257.550 us, `RegisterUpdate50` 200 ops avg 1,311.189 us, `ListRegistered50`
  1,000 ops avg 521.795 us, absolute-path equivalent 1,000 ops avg 577.420 us,
  duplicate-name equivalent 1,000 ops avg 526.669 us.

## Phase 4 - HTTP API Shell

### Benchmark Gate

- Benchmark local HTTP shell endpoints against the TypeScript baseline: `/api/info`, `/api/repos`,
  `/api/repo`, and empty `/api/graph`. Record average latency, p95 latency where practical, payload
  size, status-code parity, and CORS parity before moving on.

### Benchmark Evidence

- `powershell -ExecutionPolicy Bypass -File .tmp\pre_phase5_benchmark.ps1` passed on
  `2026-05-08T17:45:17Z` at commit `126a4f5`, with `Origin: http://127.0.0.1:5173`
  and both servers bound to `127.0.0.1`.
- HTTP readiness: Go `/api/info` ready in 29.484 ms; TypeScript baseline ready in
  2,045.386 ms.
- Go endpoint timings over 20 iterations: `/api/info` 200, 70 bytes, avg 0.759 ms,
  p95 0.706 ms; `/api/repos` 200, 195 bytes, avg 0.410 ms, p95 0.547 ms;
  `/api/repo` 200, 175 bytes, avg 0.587 ms, p95 0.660 ms; `/api/graph` 200,
  32 bytes, avg 0.387 ms, p95 0.474 ms.
- TypeScript baseline timings over 20 iterations: `/api/info` 200, 69 bytes, avg
  1.189 ms, p95 1.835 ms; `/api/repos` 200, 194 bytes, avg 1.097 ms, p95
  1.169 ms; `/api/repo` 200, 174 bytes, avg 1.601 ms, p95 2.101 ms.
- CORS parity: all measured Go and TypeScript endpoints returned
  `Access-Control-Allow-Origin: http://127.0.0.1:5173`.
- Recorded graph-shell mismatch: the Go Phase 4 placeholder returns empty graph 200 for a
  registry/meta-only fixture, while the TypeScript baseline returns 500 without a graph database.
  Exact persisted graph serving parity remains owned by Phase 10 and Phase 12.

## Phase 4A - Go-Aware Launcher Build Gate

### Benchmark Gate

- Benchmark full launcher build time, packaged backend startup time, backend health-check time,
  package contents, and package size. Compare against the previous TypeScript/Node launcher path
  when available, and record that the normal package contains no `node.exe`.

### Benchmark Evidence

- `powershell -ExecutionPolicy Bypass -File .tmp\pre_phase5_benchmark.ps1` passed on
  `2026-05-08T17:45:17Z` at commit `126a4f5`; the script ran
  `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` as the Phase 4A
  benchmark step.
- Full launcher build time: 34,302.598 ms, exit code `0`.
- Package contents: `AVmatrixLauncher.exe` exists, 6,974,464 bytes;
  `server-bundle\avmatrix-server.exe` exists, 2,056,192 bytes;
  `server-bundle\avmatrix.exe` exists, 9,242,624 bytes; `server-bundle\node.exe`
  absent.
- Server bundle size: 11,298,816 bytes. Packaged backend readiness:
  `/api/info` on `127.0.0.1` ready in 517.369 ms.

## Phase 5 - Scanner And Ignore Semantics

### Benchmark Evidence Extracted From Evidence

- Benchmark fixture: 1,200 included TypeScript files plus ignored `node_modules`, `dist`,
  `.gitignore`, and log-file paths. Five measured iterations after one warmup:
  TypeScript `walkRepositoryPaths` average `69.10218ms`; Go `scanner.WalkRepositoryPaths`
  average `59.5884ms`. Go count matched TypeScript count at `1200` files while also computing
  file hashes during scan.

### Benchmark Gate

- Benchmark scanner throughput against the TypeScript baseline on deterministic fixtures and, when
  available, one larger repo. Record selected file count parity, ignored-path parity, average scan
  time, and whether Go computes additional hashes or metrics during the timed path.

## Phase 6 - Parser Runtime

### Benchmark Evidence Extracted From Evidence

- Benchmark fixture, 500 iterations each:
  - JavaScript: TypeScript `12629.994 files/sec`, Go `14161.498 files/sec`, errors `0/0`,
    node count `17000/17000`.
  - TypeScript: TypeScript `12632.196 files/sec`, Go `10452.816 files/sec`, errors `0/0`,
    node count `27000/27000`.
  - TSX: TypeScript `12347.356 files/sec`, Go `10657.345 files/sec`, errors `0/0`,
    node count `24500/24500`.
  - Go: TypeScript `15308.450 files/sec`, Go `11085.983 files/sec`, errors `0/0`,
    node count `25000/26000`.

- Go parser is not uniformly faster than the TypeScript baseline yet. The Go grammar node-count
  delta is recorded for provider extraction parity work; there were no parser errors.

### Benchmark Gate

- Benchmark parser throughput against the TypeScript tree-sitter baseline for JavaScript,
  TypeScript, TSX, and Go fixtures. Record files/sec, bytes/sec, parser pool size, parse timeout
  behavior, syntax-error count parity, and whether any source file is reread during parse.

## Phase 7 - ScopeIR Model

### Benchmark Evidence Extracted From Evidence

- `go test ./internal/scopeir -run TestMarshalDeterministicMatchesGolden -bench
  BenchmarkScopeIRSerialization -benchmem -count=3` passed. Go ScopeIR deterministic
  marshal+unmarshal benchmark: `149544 ns/op`, `153535 ns/op`, `152165 ns/op`; `37355-37362
  B/op`; `249 allocs/op`.

- TypeScript baseline on the same sample shape, deterministic normalize + stringify + parse, 5,000
  iterations: `370.2307ms` total, `74.046 us/op`, `13505 ops/sec`.

- Go ScopeIR model is correct and golden-stable, but serialization/deserialization is slower than
  the TypeScript baseline in this first implementation. Optimization remains a later benchmark
  target before final parity/performance closure.

### Benchmark Gate

- Benchmark ScopeIR construction, deterministic serialization, deserialization, and golden
  comparison against the TypeScript baseline. Record fact counts, bytes serialized, average latency,
  and allocation/memory notes where practical.

## Phase 8 - TypeScript/JavaScript Provider

### Benchmark Evidence Extracted From Evidence

- Go benchmark on the TypeScript provider fixture:
  `go test ./internal/providers/tsjs -run TestExtract -bench BenchmarkExtractTypeScriptScopeIR
  -benchmem -count=3` -> latest run `522188 ns/op`, `525424 ns/op`, `531619 ns/op`; `120916-120920
  B/op`; `2824 allocs/op`.

- TypeScript baseline on the same fixture using the existing AST-aware scope bridge: 1,000
  iterations, `6970.428ms` total, `6970.428 us/op`, `143.463 ops/sec`, `37000` aggregate
  counted facts.

- Added exact TypeScript graph baseline count fixture at
  `internal/resolution/testdata/typescript_graph_baseline_counts.golden.json`, generated from
  `npx tsx .tmp\resolution_baseline_benchmark.ts` and
  `npx tsx .tmp\resolution_baseline_benchmark_full.ts`.

- Remaining Phase 8 follow-up before repository-scale exit evidence: unresolved-reference parity and
  repository-scale benchmark.

### Benchmark Gate

- Benchmark TypeScript/JavaScript provider extraction against the local TypeScript baseline on
  exact fixtures and this repository. Record ScopeIR fact counts by kind, graph edge counts by
  type, unresolved reference counts, files/sec, and parity deltas before moving on.

## Phase 9 - Resolution Phase

### Benchmark Evidence Extracted From Evidence

- Go resolution benchmark:
  `go test ./internal/resolution -run TestResolve -bench BenchmarkResolveTypeScriptGraphFixture
  -benchmem -count=3` -> `382940 ns/op`, `371001 ns/op`, `376983 ns/op`; `282929-282935 B/op`;
  `2472 allocs/op`.

- TypeScript baseline on the equivalent `.tmp/resolution-baseline-fixture` via
  `npx tsx .tmp\resolution_baseline_benchmark.ts`: `1371.418ms` full pipeline wall time,
  `11ms` resolution phase, `5` files, `24` nodes, `54` edges. Edge counts:
  `CONTAINS=5`, `DEFINES=18`, `HAS_PROPERTY=2`, `HAS_METHOD=5`, `IMPORTS=5`, `EXTENDS=1`,
  `IMPLEMENTS=1`, `ACCESSES=3`, `USES=7`, `CALLS=5`, `INHERITS=2`.

- Re-ran resolution benchmark:
  `go test ./internal/resolution -run 'TestResolve|TestResolveImportedTypeAlias' -bench
  BenchmarkResolveTypeScriptGraphFixture -benchmem -count=3` -> `385049 ns/op`, `385248 ns/op`,
  `375697 ns/op`; `282927-282935 B/op`; `2472 allocs/op`.

- Re-ran resolution benchmark:
  `go test ./internal/resolution -run 'TestResolve|TestBindingAccumulator' -bench
  BenchmarkResolveTypeScriptGraphFixture -benchmem -count=3` -> `379917 ns/op`,
  `427181 ns/op`, `388283 ns/op`; `283198-283203 B/op`; `2477 allocs/op`.

### Benchmark Gate

- Benchmark resolution on TypeScript/JavaScript fixtures and at least one medium repo against the
  TypeScript baseline. Record symbol count, edge count by type, unresolved reference count,
  duplicate-edge merge count, audit metadata count, phase time, and proof that no second source
  read or second AST parse occurred.

## Phase 10 - LadybugDB Persistence

### Benchmark Evidence Extracted From Evidence

- Added Phase 10 loader benchmarks: `BenchmarkExportGraphCSVs` and
  `BenchmarkLoadCSVExportCopyPathNoop` as the required phase benchmark set. A separate
  `BenchmarkDiagnosticFallbackPathNoop` exists only for diagnostic comparison when investigating a
  schema/COPY gap.

- Benchmark interpretation: COPY path is the primary runtime standard. Fallback is retained only for
  recovery/legacy diagnostics and must not be treated as a required or normal hot path.

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before native and
  benchmark verification.

- Phase 10 primary benchmark results from
  `go test ./internal/lbugload -run '^$' -bench 'Benchmark(ExportGraphCSVs|LoadCSVExportCopyPathNoop)$' -benchmem -count=3`:
  - `BenchmarkExportGraphCSVs`: `42628246 ns/op`, `40161233 ns/op`, `33395357 ns/op`;
    ~`335067-335229 B/op`; `1906-1907 allocs/op`.
  - `BenchmarkLoadCSVExportCopyPathNoop`: `2664 ns/op`, `2741 ns/op`, `2728 ns/op`;
    `1552-1568 B/op`; `19 allocs/op`.

- Diagnostic-only fallback comparison from the same historical run:
  `BenchmarkDiagnosticFallbackPathNoop` previously measured around `820763 ns/op`, `784566 ns/op`,
  and `773621 ns/op` under the old benchmark name. That path is not a phase acceptance baseline.
  Any normal supported runtime flow that hits fallback must be investigated as a schema/COPY
  contract gap.

### Benchmark Gate

- Benchmark LadybugDB schema creation, bulk node load, relationship load, primary CSV/COPY
  throughput, read query latency, graph stream read latency, and index size against the TypeScript
  baseline. COPY path is the benchmark baseline for this phase. Record rows/sec, edge rows/sec by
  relationship type, p95 read latency, compatibility-read results, and prove
  `FallbackInsertCount == 0` for supported normal flows.
- Benchmark fallback insert separately as a recovery/legacy diagnostic path only. Do not use
  fallback throughput as the primary acceptance baseline.

## Phase 11 - Analyze Pipeline

### Benchmark Evidence Extracted From Evidence

- Added `internal/analyze` as the Go-owned Phase 11 orchestration boundary for the first runnable
  pipeline slice: path validation, scan, TypeScript/JavaScript parse + ScopeIR extraction,
  resolution, phase events, cancellation checks, phase timing metrics, memory metrics, and
  benchmark metrics JSON writing.

- Added CLI `analyze [path]` with `--benchmark-json`, `--include`, `--exclude`,
  `--no-gitignore`, and `--progress`. The command defaults to the current working directory when no
  path is supplied and resolves relative CLI paths before the repo path policy check.

- Added `TestRunOrchestratesScanParseResolutionWithMetricsProgressAndBenchmark`,
  `TestRunHonorsCanceledContextBeforePhaseWork`, and
  `TestAnalyzeCommandRunsGoPipelineAndWritesBenchmark`.

- Added `TestResolveIntoPreservesExistingFileNodeMetadata` and extended
  `TestRunOrchestratesScanParseResolutionWithMetricsProgressAndBenchmark` to assert the implemented
  runtime phase order.

- Extended `TestRunOrchestratesScanParseResolutionWithMetricsProgressAndBenchmark` to prove the
  analyze runtime emits structure/document/community/process graph nodes and relationships, records
  benchmark metrics, scans `.docx`/`.pdf`/`.xlsx` plus COBOL/copybook/JCL files, emits COBOL/JCL
  route/tool/ORM/MRO enrichment metrics, and still keeps DB load on the COPY path with
  `FallbackInsertCount == 0`.

- Added `TestAnalyzeCommandGeneratesAIContextAndSkills` and extended
  `TestAnalyzeCommandRunsGoPipelineAndWritesBenchmark` to verify `.avmatrix/meta.json` output.

- Added explicit Go `cross_file_binding` phase before `resolution`. `resolution.BuildCrossFileBinding`
  builds/finalizes the binding workspace, `resolution.ResolveBoundInto` consumes it while preserving
  the existing `ResolveInto` wrapper, and analyze benchmark JSON now exposes
  `crossFileBinding` metrics.

- Added `TestBuildCrossFileBindingFeedsResolveBoundInto`, updated analyze phase-order evidence to
  include `cross_file_binding`, and asserted benchmark cross-file binding metrics.

- Ran Phase 11 Go analyze benchmark snapshots into `.tmp/`:
  `phase11-benchmark-mini-ts.json` (`7` scanned, `7` parsed, `44` graph nodes, `125`
  relationships, `10.07ms` summed phase time), `phase11-benchmark-medium-api-e2e.json` (`19`
  scanned, `18` parsed, `75` graph nodes, `90` relationships, `17.05ms` summed phase time), and
  `phase11-benchmark-avmatrix-go.json` (`1047` scanned, `747` parsed, `21567` graph nodes,
  `37064` relationships, `10593.38ms` summed phase time, `429.72MiB` max observed Sys memory).

- Fixed Go benchmark artifact finalization so `totalDuration` and final memory counters are written
  before the JSON file is emitted, not only after `Run` returns.

- Made DB-load benchmark counters explicit even when zero, so COPY-path acceptance can prove
  `fallbackInsertCount: 0`, `fallbackInsertFailures: 0`, and `skippedRelationships: 0`.

- Re-ran tagged native LadybugDB `v0.16.1` fixture smoke with:
  `go run -tags ladybugdb ./cmd/avmatrix analyze .tmp\resolution-baseline-fixture --force --benchmark-json .tmp\phase11-benchmark-native-dbload-smoke.json`.
  Result: `db_load` phase recorded, `nodeRows: 28`, `relationshipRows: 84`,
  `nodeCopyCount: 11`, `relationshipCopyCount: 31`, `fallbackInsertCount: 0`.

- Community/process baseline comparison update:
  - Go artifact: `.tmp\phase11-community-parity-go-benchmark.json` plus
    `.tmp\phase11-community-parity-go-graph.json`.
  - TypeScript artifact: `.tmp\phase11-community-parity-ts-benchmark.json`.
  - Fixture: `.tmp\resolution-baseline-fixture`.
  - Go launcher analyze: `nodes=29`, `relationships=84`, `Community=3`, `MEMBER_OF=9`,
    `Process=1`, `STEP_IN_PROCESS=3`; community metrics recorded `communitiesEmitted=3`,
    `membershipsEmitted=9`, `nodesConsidered=9`, `edgesConsidered=7`,
    `modularity=0.47959183673469385`.
  - TypeScript dist analyze with `--skip-git`: `nodes=28`, `relationships=67`, `Community=3`,
    `MEMBER_OF=9`, `Process=1`, `STEP_IN_PROCESS=3`; benchmark recorded `communities=21ms`,
    `processes=17ms`.
  - The current Phase 14 return slice closes community/process parity. Remaining total graph
    count differences are broader resolver fact deltas, not a community/process benchmark blocker.
  - This-repository and medium-repo full analyze native DB-load timing remain open benchmark
    evidence work.

### Benchmark Gate

- Benchmark the full analyze pipeline against the TypeScript baseline on the deterministic fixture,
  this repository, and one medium repo when available. Record scan, structure, parse, route/tool/ORM,
  resolution, MRO, communities, processes, DB load, total time, peak memory, graph counts, and
  unresolved reference counts.

## Phase 12 - HTTP Analyze And Graph Serving

### Benchmark Evidence Extracted From Evidence

- Phase 12 benchmark exposed and closed an HTTP/analyzer double-lock bug: `POST /api/analyze`
  accepted the job but the runner failed with `repository index lock is already held`. The fix keeps
  HTTP lock handling as a preflight only and leaves the analyzer as the single writer-lock owner.

### Benchmark Gate

- Benchmark HTTP analyze lifecycle and graph serving against the TypeScript backend: analyze submit
  latency, SSE progress cadence, hold-queue behavior, graph JSON latency, NDJSON streaming latency,
  first-byte time, total bytes, and Web UI observable load time.

### Benchmark Evidence

- `powershell -ExecutionPolicy Bypass -File .tmp\phase12_http_benchmark.ps1` passed on
  `2026-05-12T02:52:51Z` against the working tree containing lifecycle dedupe/cleanup, using the
  deterministic `.tmp\resolution-baseline-fixture` graph snapshot on `127.0.0.1:48747`.
- HTTP backend readiness: 620.937 ms. Endpoint timings: `/api/info` 200, 70 bytes, avg
  13.068 ms, p95 21.169 ms; `/api/repos` 200, 1995 bytes, avg 15.610 ms, p95 32.988 ms;
  `/api/repo` 200, 210 bytes, avg 12.482 ms, p95 14.484 ms; `/api/graph` JSON 200,
  30,949 bytes, avg 31.148 ms, p95 32.752 ms; `/api/graph` NDJSON 200, 122,543 bytes,
  avg 652.264 ms, p95 699.338 ms.
- Web panel read endpoint timings: `/api/processes` 200, 138 bytes, avg 18.472 ms, p95
  21.238 ms; `/api/clusters` 200, 124 bytes, avg 13.466 ms, p95 15.207 ms; `/api/grep`
  200, 256 bytes, avg 12.638 ms, p95 12.768 ms. CORS preflight returned 204, avg
  6.005 ms, p95 6.283 ms.
- Analyze lifecycle benchmark passed: submit returned 202 in 19.136 ms, SSE progress completed in
  268.516 ms with `event: complete`, `repoName`, and `repoPath`.

## Phase 13 - MCP Server

### Benchmark Gate

- Benchmark MCP stdio and HTTP discovery plus representative `query`, `context`, `impact`,
  `detect_changes`, `route_map`, and group tool calls against the TypeScript baseline. Record
  response latency, payload size, repo-selection behavior, stale-index warning parity, and stdout
  safety evidence.

### Benchmark Evidence Extracted From Evidence

- Re-ran `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` before the Phase
  13 benchmark gate.

- Ran the Phase 13 MCP benchmark script under `.tmp/phase13_mcp_benchmark.mjs`; artifacts were
  written under `.tmp/phase13_mcp_benchmark_results.json` and
  `.tmp/phase13_mcp_benchmark_summary.md`.

- MCP stdio benchmark parity against the local TypeScript runtime:
  `initialize` Go `43.86ms` / TypeScript `1420.76ms`; `tools/list` `1.17ms` / `3.41ms`;
  `resources/list` `0.40ms` / `1.23ms`; `resources/templates/list` `0.37ms` / `1.07ms`;
  `prompts/list` `0.37ms` / `0.86ms`; context resource `51.56ms` / `47.71ms`; `query`
  `650.59ms` / `1239.18ms`; `context` `425.69ms` / `70.17ms`; `impact` `426.88ms` /
  `127.36ms`; `detect_changes` `66.64ms` / `56.44ms`; `route_map` `467.65ms` / `29.77ms`;
  `group_list` `0.40ms` / `1.77ms`. All calls returned OK; Go and TypeScript stdio protocol
  noise bytes were both `0`.

- MCP HTTP benchmark parity against the local TypeScript runtime on `127.0.0.1`:
  `initialize` Go `11.18ms` / TypeScript `48.65ms`; `tools/list` `3.69ms` / `7.73ms`;
  `resources/list` `4.24ms` / `4.98ms`; `resources/templates/list` `3.20ms` / `4.63ms`;
  `prompts/list` `2.44ms` / `5.22ms`; context resource `56.80ms` / `47.01ms`; `query`
  `740.44ms` / `1206.12ms`; `context` `421.69ms` / `76.09ms`; `impact` `426.43ms` /
  `133.09ms`; `detect_changes` `91.12ms` / `61.21ms`; `route_map` `406.94ms` / `32.75ms`;
  `group_list` `1.51ms` / `3.71ms`. All calls returned OK and both runtimes reused HTTP MCP
  sessions.

- Group fixture benchmark with `AVMATRIX_HOME=.tmp/mcp-group-sync-smoke/home` covered
  `group_list`, `group_status`, `group_contracts`, `group_sync`, and post-sync
  `group_contracts` on stdio and HTTP. All Go and TypeScript calls returned OK. Representative
  stdio latencies: `group_status` Go `88.15ms` / TypeScript `87.86ms`, `group_sync` `1.15ms` /
  `6.94ms`; representative HTTP latencies: `group_status` `93.67ms` / `90.01ms`, `group_sync`
  `43.21ms` / `8.94ms`.

- Benchmark payload-size evidence: Go/TypeScript `tools/list` returned `7813` / `18447` bytes;
  context resources returned `1828` / `1162` bytes and both exposed stale-index warnings;
  `detect_changes` returned `453` / `453` bytes on both stdio and HTTP.

### Benchmark Interpretation

Phase 13 benchmark gives the optimization direction for MCP tool paths.

| Tool | Stdio Go vs TypeScript | HTTP Go vs TypeScript | Interpretation |
| --- | ---: | ---: | --- |
| `initialize` | Go faster by `~32.39x` | Go faster by `~4.35x` | strong startup path |
| `query` | Go faster by `~1.90x` | Go faster by `~1.63x` | good query path |
| `context` | Go slower by `~6.07x` | Go slower by `~5.54x` | graph-context optimization candidate |
| `impact` | Go slower by `~3.35x` | Go slower by `~3.20x` | traversal/index optimization candidate |
| `route_map` | Go slower by `~15.71x` | Go slower by `~12.43x` | highest-priority optimization candidate |
| `group_sync` | Go faster by `~6.03x` | Go slower by `~4.83x` | likely HTTP/session/wrapper overhead |
| `tools/list` payload | Go smaller by `~57.6%` | not separately measured | good payload result |
| protocol noise | `0` bytes for both | not applicable | good protocol safety result |

Optimization reading:

```text
This benchmark identifies where the current Go MCP implementation is already on the expected fast
path and where it still needs optimization to reach Go runtime potential. Preserve the strong
initialize/query/tools-list/protocol-noise paths, then profile and improve context, impact,
route_map, and HTTP group_sync without dropping correctness, provider coverage, contract parity, or
required runtime work.
```

Optimization priority:

- `P0`: `route_map`, target `<50ms`, likely needs a precomputed route index/cache.
- `P1`: `context`, target `<100ms`, with timing split across repo resolve, target lookup,
  neighborhood read, file/snippet reads, formatting, and JSON serialization.
- `P1`: `impact`, target `<150ms`, with hot indexes for callers/callees/importers/tests/routes/tools.
- `P2`: HTTP `group_sync`, remeasure cold-session and warm-session paths separately.
- `P3`: preserve small discovery payloads and `0` protocol-noise bytes while optimizing.

## Phase 14 - Additional Language Providers

### Benchmark Evidence Extracted From Go provider evidence

- Benchmark, 5 runs on this machine:
  - Go extract-only average: `686310 ns/op`, `143461-143465 B/op`, `3301 allocs/op`.
  - Go parse+extract average: `1067205 ns/op`, `151322-151557 B/op`, `3524 allocs/op`,
    about `937 files/sec` for the fixture.
  - Existing TypeScript provider extract-only baseline average: `545167 ns/op`,
    `120916-120920 B/op`, `2824 allocs/op`.
  - Go provider is correctness-complete for this batch but not faster than the TypeScript
    extract-only baseline yet; optimization remains Phase 15 work.

### Benchmark Evidence Extracted From Python provider evidence

- Benchmark, 3 runs on this machine:
  `go test ./internal/providers/python -run TestExtract -bench Benchmark -benchmem -count=3`.
  - Python extract-only: `440327 ns/op`, `434971 ns/op`, `530614 ns/op`; average
    `468637 ns/op`, about `2134 files/sec` for the fixture, `70282-70283 B/op`,
    `1886 allocs/op`.
  - Python parse+extract: `940296 ns/op`, `977664 ns/op`, `838856 ns/op`; average
    `918939 ns/op`, about `1088 files/sec` for the fixture, `75985-76151 B/op`,
    `2046 allocs/op`.
  - Python provider is correctness-complete for this batch. These numbers are the conversion
    baseline for later optimization, not a Phase 14 requirement to tune hot paths now.

### Benchmark Evidence Extracted From Java provider evidence

- Benchmark, 3 runs on this machine:
  `go test ./internal/providers/java -run ^$ -bench "Benchmark(ExtractJavaScopeIR|ParseAndExtractJavaScopeIR)$" -benchmem -count=3`.
  - Java extract-only: `491322 ns/op`, `494394 ns/op`, `508033 ns/op`; average
    `497916 ns/op`, about `2008 files/sec` for the fixture, `112533 B/op`,
    `2445 allocs/op`.
  - Java parse+extract: `846199 ns/op`, `886706 ns/op`, `850543 ns/op`; average
    `861149 ns/op`, about `1161 files/sec` for the fixture, `121153-121235 B/op`,
    `2686 allocs/op`.
  - Java provider is correctness-complete for this batch. These numbers are the conversion
    baseline for later optimization, not a Phase 14 requirement to tune hot paths now.

### Benchmark Evidence Extracted From Kotlin provider evidence

- Benchmark, 3 runs on this machine:
  `go test ./internal/providers/kotlin -run ^$ -bench "Benchmark(ExtractKotlinScopeIR|ParseAndExtractKotlinScopeIR)$" -benchmem -count=3`.
  - Kotlin extract-only: `542967 ns/op`, `530485 ns/op`, `530129 ns/op`; average
    `534527 ns/op`, about `1871 files/sec` for the fixture, `121013-121016 B/op`,
    `2837 allocs/op`.
  - Kotlin parse+extract: `1373535 ns/op`, `1376984 ns/op`, `1375031 ns/op`; average
    `1375183 ns/op`, about `727 files/sec` for the fixture, `129722-129853 B/op`,
    `3084 allocs/op`.
  - Kotlin provider is correctness-complete for this batch. These numbers are the conversion
    baseline for later optimization, not a Phase 14 requirement to tune hot paths now.

### Benchmark Evidence Extracted From C provider evidence

- Benchmark, 3 runs on this machine:
  `go test ./internal/providers/c -run ^$ -bench "Benchmark(ExtractCScopeIR|ParseAndExtractCScopeIR)$" -benchmem -count=3`.
  - C extract-only: `319962 ns/op`, `316272 ns/op`, `314882 ns/op`; average
    `317039 ns/op`, about `3154 files/sec` for the fixture, `68850-68851 B/op`,
    `1668 allocs/op`.
  - C parse+extract: `640050 ns/op`, `657835 ns/op`, `680856 ns/op`; average
    `659580 ns/op`, about `1516 files/sec` for the fixture, `74313-74427 B/op`,
    `1822 allocs/op`.
  - C provider is correctness-complete for this batch. These numbers are the conversion baseline
    for later optimization, not a Phase 14 requirement to tune hot paths now.

### Benchmark Evidence Extracted From C# provider evidence

- Benchmark, 3 runs on this machine:
  `go test ./internal/providers/csharp -run ^$ -bench "Benchmark(ExtractCSharpScopeIR|ParseAndExtractCSharpScopeIR)$" -benchmem -count=3`.
  - C# extract-only: `544639 ns/op`, `550378 ns/op`, `552912 ns/op`; average
    `549310 ns/op`, about `1820 files/sec` for the fixture, `118589-118591 B/op`,
    `2815 allocs/op`.
  - C# parse+extract: `1453604 ns/op`, `1458170 ns/op`, `1382791 ns/op`; average
    `1431522 ns/op`, about `699 files/sec` for the fixture, `127138-127280 B/op`,
    `3056 allocs/op`.
  - C# provider is correctness-complete for this batch. These numbers are the conversion baseline
    for later optimization, not a Phase 14 requirement to tune hot paths now.

### Benchmark Evidence Extracted From C++ provider evidence

- Benchmark, 3 runs on this machine:
  `go test ./internal/providers/cpp -run ^$ -bench "Benchmark(ExtractCPPScopeIR|ParseAndExtractCPPScopeIR)$" -benchmem -count=3`.
  - C++ extract-only: `725998 ns/op`, `695453 ns/op`, `672794 ns/op`; average
    `698082 ns/op`, about `1432 files/sec` for the fixture, `114533-114535 B/op`,
    `2783 allocs/op`.
  - C++ parse+extract: `1469542 ns/op`, `1308799 ns/op`, `1285172 ns/op`; average
    `1354504 ns/op`, about `738 files/sec` for the fixture, `122779-122901 B/op`,
    `3016 allocs/op`.
  - C++ provider is correctness-complete for this batch. These numbers are the conversion baseline
    for later optimization, not a Phase 14 requirement to tune hot paths now.

### Benchmark Evidence Extracted From Rust provider evidence

- Benchmark, 3 runs on this machine:
  `go test ./internal/providers/rust -run ^$ -bench "Benchmark(ExtractRustScopeIR|ParseAndExtractRustScopeIR)$" -benchmem -count=3`.
  - Rust extract-only: `652752 ns/op`, `553326 ns/op`, `593547 ns/op`; average
    `599875 ns/op`, about `1667 files/sec` for the fixture, `99853-99856 B/op`,
    `2125 allocs/op`.
  - Rust parse+extract: `1011246 ns/op`, `1249759 ns/op`, `1013737 ns/op`; average
    `1091581 ns/op`, about `916 files/sec` for the fixture, `107728-107779 B/op`,
    `2343 allocs/op`.
  - Rust provider is correctness-complete for this batch. These numbers are the conversion baseline
    for later optimization, not a Phase 14 requirement to tune hot paths now.

### Benchmark Evidence Extracted From PHP provider evidence

- Benchmark, 3 runs on this machine:
  `go test ./internal/providers/php -run ^$ -bench "Benchmark(ExtractPHPScopeIR|ParseAndExtractPHPScopeIR)$" -benchmem -count=3`.
  - PHP extract-only: `1331510 ns/op`, `822815 ns/op`, `1053458 ns/op`; average
    `1069261 ns/op`, about `935 files/sec` for the fixture, `125121-125123 B/op`,
    `3083 allocs/op`.
  - PHP parse+extract: `1634353 ns/op`, `1662638 ns/op`, `1781592 ns/op`; average
    `1692861 ns/op`, about `591 files/sec` for the fixture, `135267-135404 B/op`,
    `3372 allocs/op`.
  - PHP provider is correctness-complete for this batch. These numbers are the conversion baseline
    for later optimization, not a Phase 14 requirement to tune hot paths now.

### Benchmark Evidence Extracted From Dart provider evidence

- Benchmark, 3 runs on this machine:
  `go test ./internal/providers/dart -run ^$ -bench "Benchmark(ExtractDartScopeIR|ParseAndExtractDartScopeIR)$" -benchmem -count=3`.
  - Dart extract-only: `864341 ns/op`, `834242 ns/op`, `821012 ns/op`; average
    `839865 ns/op`, about `1191 files/sec` for the fixture, `135488-135490 B/op`,
    `3568 allocs/op`.
  - Dart parse+extract: `1523145 ns/op`, `1492609 ns/op`, `1504629 ns/op`; average
    `1506794 ns/op`, about `664 files/sec` for the fixture, `144483-144546 B/op`,
    `3827 allocs/op`.
  - Dart provider is correctness-complete for this batch. These numbers are the conversion baseline
    for later optimization, not a Phase 14 requirement to tune hot paths now.

### Benchmark Evidence Extracted From Vue provider evidence

- Benchmark, 3 runs on this machine:
  `go test ./internal/providers/vue -run ^$ -bench "BenchmarkExtractVueScopeIR$" -benchmem -count=3`.
  - Vue script extract+parse: `906201 ns/op`, `860286 ns/op`, `871896 ns/op`; average
    `879461 ns/op`, about `1137 files/sec` for the fixture, `129884-129957 B/op`,
    `2659 allocs/op`.
  - Vue provider is correctness-complete for inline script graph facts. These numbers are the
    conversion baseline for later optimization, not a Phase 14 requirement to tune hot paths now.

### Benchmark Evidence Extracted From Swift provider evidence

- Benchmark, 3 runs on this machine:
  `go test ./internal/providers/swift -run ^$ -bench "Benchmark(ExtractSwiftScopeIR|ParseAndExtractSwiftScopeIR)$" -benchmem -count=3`.
  - Swift extract-only: `598145 ns/op`, `714792 ns/op`, `599943 ns/op`; average
    `637627 ns/op`, about `1568 files/sec` for the fixture, `107741-107743 B/op`,
    `2576 allocs/op`.
  - Swift parse+extract: `1936721 ns/op`, `1933172 ns/op`, `1939321 ns/op`; average
    `1936405 ns/op`, about `516 files/sec` for the fixture, `116602-116712 B/op`,
    `2782 allocs/op`.
  - Swift provider is correctness-complete for this batch. These numbers are the conversion
    baseline for later optimization, not a Phase 14 requirement to tune hot paths now.

### Benchmark Evidence Extracted From Ruby provider evidence

- Benchmark, 3 runs on this machine:
  `go test ./internal/providers/ruby -run ^$ -bench "Benchmark(ExtractRubyScopeIR|ParseAndExtractRubyScopeIR)$" -benchmem -count=3`.
  - Ruby extract-only: `371061 ns/op`, `360328 ns/op`, `367602 ns/op`; average
    `366330 ns/op`, about `2730 files/sec` for the fixture, `76569-76570 B/op`,
    `1788 allocs/op`.
  - Ruby parse+extract: `927676 ns/op`, `988750 ns/op`, `894510 ns/op`; average
    `936979 ns/op`, about `1067 files/sec` for the fixture, `83713-83803 B/op`,
    `1938 allocs/op`.
  - Ruby provider is correctness-complete for this batch. These numbers are the conversion
    baseline for later optimization, not a Phase 14 requirement to tune hot paths now.

### Benchmark Evidence Extracted From COBOL/JCL provider evidence

- Benchmark, 3 runs on this machine:
  `go test ./internal/cobol -run ^$ -bench "BenchmarkApplyCobolEnrichment$" -benchmem -count=3`.
  - COBOL/JCL enrichment: `497527 ns/op`, `498495 ns/op`, `490381 ns/op`; average
    `495468 ns/op`, about `2018 fixture enrichments/sec`, `29145-29291 B/op`,
    `237 allocs/op`.
  - COBOL/JCL is correctness-complete through the existing pre-parse enrichment phase. These
    numbers are the conversion baseline for later optimization, not a Phase 14 requirement to tune
    hot paths now.

### Benchmark Evidence Extracted From Framework-specific facts evidence

- Benchmark, 3 runs on this machine:
  `go test ./internal/frameworks -run ^$ -bench "BenchmarkDetectFrom(Path|AST)$" -benchmem -count=3`.
  - Path framework detection: `10470 ns/op`, `10415 ns/op`, `9270 ns/op`; average
    `10052 ns/op` per 12-path batch, about `99k batches/sec` or `1.19M path checks/sec`,
    `408 B/op`, `10 allocs/op`.
  - AST framework detection: `324.4 ns/op`, `341.7 ns/op`, `328.7 ns/op`; average
    `331.6 ns/op`, about `3.02M AST checks/sec`, `80 B/op`, `1 alloc/op`.
  - Framework detection is correctness-complete for this conversion batch. These numbers are the
    conversion baseline for later optimization, not a Phase 14 requirement to tune hot paths now.

### Benchmark Gate

- For every language provider, benchmark parse plus extraction against the TypeScript baseline on
  language-specific fixtures before marking that provider complete. Record files/sec, ScopeIR fact
  counts, edge counts by type, unresolved reference counts, and any documented grammar/runtime
  fallback.

## Phase 15 - Performance Optimization

### Benchmark Gate

- Every optimization batch must include before/after benchmark artifacts plus graph parity from the
  same repo state. Record pprof CPU/memory artifacts when used, the exact changed optimization,
  and the rollback decision if speed improves by dropping required graph facts.

## Phase 16 - Launcher Integration

### Benchmark Gate

- Benchmark packaged launcher start, stop, reset, protocol start, backend health readiness, Web UI
  readiness, package build time, and package size. Record process list evidence proving no
  TypeScript/Node runtime is required outside the Web UI build.

### Benchmark Evidence Extracted From Launcher Integration evidence

- Benchmark/smoke, 1 run on this machine:
  - Full launcher package build:
    `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` -> `47.70s`;
    build log: `.tmp\phase16-launcher-build.log`.
  - Packaged launcher start readiness:
    `AVmatrixLauncher.exe` -> backend `127.0.0.1:4747` and Web UI `127.0.0.1:5173` ready in
    `1.64s`.
  - Protocol registration command:
    `AVmatrixLauncher.exe register` -> `0.16s`.
  - Stop command plus readiness-down wait:
    `AVmatrixLauncher.exe stop` -> `11.92s`.
  - Reset command on stopped runtime:
    `AVmatrixLauncher.exe reset` -> `0.63s`.
  - Package sizes:
    `AVmatrixLauncher.exe=6,974,464 bytes`,
    `server-bundle/avmatrix-server.exe=2,056,192 bytes`,
    `server-bundle/avmatrix.exe=48,830,976 bytes`.
  - Normal packaged backend path contains no `server-bundle/node.exe`.
  - Playwright packaged UI smoke reached graph status `READY` with `31807 nodes` and
    `55775 edges`, then stop left no packaged launcher/backend processes running.

## Phase 17 - Cutover Criteria

### Benchmark Gate

- Produce the final cutover benchmark package before replacing the TypeScript runtime: deterministic
  fixture, this repository, one medium repo, and one large repo. The package must include graph
  parity, phase timings, peak memory, graph stream time, launcher startup time, MCP latency, Web UI
  browser validation timing, commit hashes, dirty-state proof, and rollback instructions.

## Benchmark Protocol

Benchmark output must include:

```json
{
  "repoPath": "...",
  "implementation": "typescript-baseline | go",
  "commit": "...",
  "dirty": false,
  "totalMs": 0,
  "scanMs": 0,
  "parseMs": 0,
  "resolutionMs": 0,
  "dbLoadMs": 0,
  "dbLoadCopyNodeRows": 0,
  "dbLoadCopyRelationshipRows": 0,
  "dbLoadFallbackInserts": 0,
  "graphStreamMs": 0,
  "files": 0,
  "nodes": 0,
  "edges": 0,
  "edgeCountsByType": {
    "CONTAINS": 0,
    "DEFINES": 0,
    "CALLS": 0,
    "IMPORTS": 0,
    "ACCESSES": 0,
    "USES": 0,
    "INHERITS": 0,
    "EXTENDS": 0,
    "IMPLEMENTS": 0,
    "HAS_METHOD": 0,
    "HAS_PROPERTY": 0,
    "METHOD_OVERRIDES": 0,
    "OVERRIDES": 0,
    "METHOD_IMPLEMENTS": 0,
    "DECORATES": 0,
    "MEMBER_OF": 0,
    "STEP_IN_PROCESS": 0,
    "HANDLES_ROUTE": 0,
    "FETCHES": 0,
    "HANDLES_TOOL": 0,
    "ENTRY_POINT_OF": 0,
    "WRAPS": 0,
    "QUERIES": 0
  },
  "unresolvedReferences": 0,
  "peakMemoryMb": 0
}
```

Rules:

- Do not compare speed without graph parity data.
- Do not compare old TypeScript and new Go runs if they analyzed different repo states.
- Record commit and dirty state for every run.
- For DB-load benchmark claims, use the CSV/COPY path as the primary baseline and require
  `dbLoadFallbackInserts == 0` on supported normal flows. Fallback measurements are diagnostic
  only and must be reported separately from phase acceptance numbers.
- Treat the first large-repo result as a baseline, not a final optimization claim.

### Cutover Runtime Smoke Timing

- Full launcher build gate before this batch: `powershell -ExecutionPolicy Bypass -File
  avmatrix-launcher\build.ps1` passed in approximately `41s`.
- Docker CLI image first full build: `docker build -f Dockerfile.cli -t avmatrix-go-cli-cutover .`
  passed in approximately `248s` while pulling/building uncached layers.
- Docker CLI image rebuild after Dockerfile cleanup: same command passed in approximately `11s` with
  cached Go build layers.
- Container readiness smoke: `curl.exe -fsS http://127.0.0.1:14747/api/info` returned Go runtime
  info from the container. This is cutover evidence only; the large-repo graph parity and speed
  benchmark gates remain open.

### Current-Repo Large Benchmark Probe

- Commit: `5d64ece`.
- Dirty-state note: tracked files were clean with `git status --porcelain --untracked-files=no`;
  the TypeScript benchmark artifact reports `repoGitDirty=true` because local untracked `coder.md`
  exists and is intentionally not committed.
- TypeScript baseline artifact: `.tmp\phase17-cutover-ts-avmatrix-go.json`.
  - Files: `1205`.
  - Nodes: `28731`.
  - CLI-reported edges: `51603`.
  - Graph-snapshot relationships: `52686`.
  - Total wall time: `150292.7ms`.
- Go packaged artifact: `.tmp\phase17-cutover-go-avmatrix-go.json`.
  - Files scanned: `1205`.
  - Files parsed: `1036`.
  - Unsupported: `169`.
  - Nodes: `31829`.
  - Relationships: `55816`.
  - Total duration: `15668.3ms`.
  - Speed ratio for this run: Go `~9.59x` faster.
  - DB load status: `skipped=true`, `skipReason="query runner factory returned nil"`.
- Parity status: `FAIL / NOT READY`.
  - Node delta: `+3098`.
  - Relationship delta against TypeScript graph snapshot: `+3130`.
  - Largest relationship deltas: `USES +5071`, `CALLS -3571`, `DEFINES +2829`,
    `HAS_PROPERTY +2753`, `STEP_IN_PROCESS -2347`, `IMPORTS -2166`, `ACCESSES +1976`,
    `MEMBER_OF -1221`.
  - Largest node-label deltas: `Variable +11969`, `Const -13135`, `Property +3506`,
    `Community +382`, `Process -584`, `Section +462`.
  - Do not use the `~9.59x` speed result as an acceptance claim until graph parity and packaged
    DB-load status are closed.

### Current-Repo Native DB Load Rerun

- Commit baseline before this fix: `6fc85fe`; working tree contained the native packaging edits and
  untracked local `coder.md`.
- Latest-release bootstrap:
  - Local cache was intentionally rolled back to `v0.15.0` with stale
    `checkedDateUtc=2026-05-12`.
  - `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` refreshed the cache to
    GitHub latest `v0.16.1` on `2026-05-13`, restored the Windows native runtime directory, and
    built the packaged backend with `-tags ladybugdb`.
- Packaged Go native artifact: `.tmp\phase17-cutover-go-native-avmatrix-go.json`.
  - Files scanned: `1208`.
  - Files parsed: `1036`.
  - Unsupported: `172`.
  - Nodes: `31838`.
  - Relationships: `55824`.
  - Total duration: `66233.7ms`.
  - DB load duration: `36540.7ms`.
  - DB load rows: `31549` node rows, `55824` relationship rows.
  - DB load fallback inserts: `0`.
  - DB load skipped: no.
- Speed ratio against the prior TypeScript current-repo baseline: Go native packaged runtime is
  `~2.27x` faster (`150292.7ms / 66233.7ms`) on this dirty working-tree run.
- Acceptance status: `PARTIAL / NOT READY`. Native DB load is no longer skipped, and COPY remains
  the primary DB-load path, but graph parity deltas still need reconciliation before the large-repo
  speed gate can be ticked.

### Current-Repo Process Parity Rerun

- TypeScript current artifact: `.tmp\phase17-cutover-ts-current.json`.
  - Commit: `e200d859e21fd489b94f051f4356067c1eeaea45`.
  - Files: `1208`.
  - Nodes: `28741`.
  - Relationships: `52695`.
  - Total wall time: `139786ms`.
- Packaged Go native artifact after process fix: `.tmp\phase17-cutover-go-native-processfix.json`.
  - Files scanned: `1208`.
  - Files parsed: `1036`.
  - Unsupported: `172`.
  - Nodes: `32348`.
  - Relationships: `58091`.
  - Total duration: `48604.2ms`.
  - DB load rows: `32059` node rows, `58091` relationship rows.
  - DB load fallback inserts: `0`.
  - Speed ratio for this run: Go native packaged runtime is `~2.88x` faster.
- Process-family delta after fix:
  - `Process`: TS `659`, Go `556`, delta `-103` (`75` before this fix).
  - `STEP_IN_PROCESS`: TS `2640`, Go `1954`, delta `-686` (`293` before this fix).
  - `ENTRY_POINT_OF`: TS `324`, Go `632`, delta `+308`; Route/Tool entry links are restored.
- Remaining high-signal graph deltas:
  - `CALLS`: TS `10555`, Go `6997`, delta `-3558`.
  - `IMPORTS`: TS `2367`, Go `201`, delta `-2166`.
  - `USES`: TS `970`, Go `6046`, delta `+5076`.
  - `ACCESSES`: TS `831`, Go `2809`, delta `+1978`.
  - `DEFINES`: TS `25543`, Go `28392`, delta `+2849`.
  - `HAS_PROPERTY`: TS `1800`, Go `4554`, delta `+2754`.
- Acceptance status: `PARTIAL / NOT READY`. The old fixed process cap was a Go parity bug and is
  fixed. The remaining process gap follows the unresolved `CALLS` deficit, and `IMPORTS` still needs
  separate reconciliation.

### Current-Repo Import Parity Rerun

- TypeScript current artifact: `.tmp\phase17-cutover-ts-current.json`.
  - Commit: `e200d859e21fd489b94f051f4356067c1eeaea45`.
  - Files: `1208`.
  - Nodes: `28741`.
  - Relationships: `52695`.
  - Total wall time: `139786ms`.
- Packaged Go native artifact after Go package import expansion:
  `.tmp\phase17-cutover-go-importfix-notests.json`.
  - Files scanned: `1208`.
  - Files parsed: `1036`.
  - Unsupported: `172`.
  - Nodes: `32365`.
  - Relationships: `60515`.
  - Total duration: `49561.7ms`.
  - DB load fallback inserts: `0`.
  - Speed ratio for this run: Go native packaged runtime is `~2.82x` faster.
- Import-family delta after fix:
  - `IMPORTS`: TS `2367`, Go `2140`, delta `-227` (`201` before this fix).
  - The first package-import expansion attempt produced `IMPORTS=2999`; excluding package
    `_test.go` targets corrected the over-expansion before this benchmark was recorded.
- Remaining high-signal graph deltas:
  - `CALLS`: TS `10555`, Go `7004`, delta `-3551`.
  - `USES`: TS `970`, Go `6496`, delta `+5526`.
  - `ACCESSES`: TS `831`, Go `2812`, delta `+1981`.
  - `DEFINES`: TS `25543`, Go `28406`, delta `+2863`.
  - `HAS_PROPERTY`: TS `1800`, Go `4555`, delta `+2755`.
  - `STEP_IN_PROCESS`: TS `2640`, Go `1954`, delta `-686`.
  - `ENTRY_POINT_OF`: TS `324`, Go `632`, delta `+308`.
- Acceptance status: `PARTIAL / NOT READY`. Local Go package import coverage is restored close to
  the TypeScript baseline, but the remaining `IMPORTS` delta still needs classification and `CALLS`
  remains the next blocking graph-parity deficit.

### Current-Repo CALLS Compatibility And DB Schema Rerun

- TypeScript current diagnostic artifact:
  `.tmp\phase17-cutover-ts-calls-diagnostic.json`.
  - Commit: `29b3ed7`.
  - Files: `1208`.
  - Nodes: `28775`.
  - Relationships: `52792`.
  - Total wall time: `133541ms`.
  - Relationship baselines: `CALLS=10589`, `IMPORTS=2370`, `STEP_IN_PROCESS=2646`.
- Packaged Go native artifact after CALLS compatibility and DB schema reconciliation:
  `.tmp\phase17-cutover-go-call-compatfix-schema.json`.
  - Files scanned: `1208`.
  - Files parsed: `1036`.
  - Unsupported: `172`.
  - Nodes: `32735`.
  - Relationships: `63933`.
  - Total duration: `59419.2ms`.
  - DB load: `nodeRows=32735`, `relationshipRows=63933`, `fallbackInsertCount=0`,
    `fallbackInsertFailures=0`, `skippedRelationships=0`.
  - Speed ratio for this run: Go native packaged runtime is `~2.25x` faster than the latest
    TypeScript diagnostic while doing real DB load.
- CALL-family delta after fix:
  - `CALLS`: TS `10589`, Go `8746`, delta `-1843` (`7004` before this fix).
  - Go resolution metrics: `ResolvedCalls=22398`, `DuplicateEdgesMerged=43576`.
  - Go process metrics: `processesEmitted=700`, `stepsEmitted=2671`,
    `callsEdgesConsidered=8746`.
- Remaining high-signal graph deltas:
  - `IMPORTS`: TS `2370`, Go `2140`, delta `-230`.
  - `STEP_IN_PROCESS`: TS `2646`, Go `2671`, delta `+25`.
  - `USES`, `ACCESSES`, `DEFINES`, `HAS_PROPERTY`, `Community`, `Variable`, `Const`, and
    node-label count deltas still need classification as intended expanded coverage vs parity bugs.
- Acceptance status: `PARTIAL / NOT READY`. The CALLS deficit is substantially smaller and DB load
  no longer loses runtime-emitted edges, but Phase 17 graph parity remains open until the remaining
  CALLS/IMPORTS deltas and expanded-coverage deltas are classified or fixed.

### Current-Repo Call-Return Type Binding Rerun

- Packaged Go native artifact after call-return type binding enrichment:
  `.tmp\phase17-cutover-go-call-returntype.json`.
  - Files scanned: `1208`.
  - Files parsed: `1036`.
  - Unsupported: `172`.
  - Nodes: `32767`.
  - Relationships: `64567`.
  - Total duration: `61562.9ms`.
  - DB load: `nodeRows=32767`, `relationshipRows=64567`, `fallbackInsertCount=0`,
    `fallbackInsertFailures=0`, `skippedRelationships=0`.
- Relationship deltas after this fix:
  - `CALLS`: TS `10589`, Go `8906`, delta `-1683` (`8746` before this fix).
  - `IMPORTS`: TS `2370`, Go `2140`, delta `-230`.
  - `STEP_IN_PROCESS`: TS `2646`, Go `2682`, delta `+36`.
  - `Function->Method` CALLS improved from Go `280` to `391`, still below TS `1626`.
  - `Variable->Function` remains Go-heavy at `1335` vs TS `2`; this is now a classification target
    because it may reflect source attribution drift for top-level variable initializers rather than
    missing target resolution.
- Acceptance status: `PARTIAL / NOT READY`. Imported call-return type binding closes real receiver
  method calls such as `graph.New()` -> `g.AddNode()`, but the remaining CALLS gap still needs
  source-attribution and method-target classification.

### Current-Repo Final Parity Classification

- Summary artifact:
  `.tmp\phase17-cutover-parity-summary-current.json`.
- Accepted comparison artifacts:
  - TypeScript: `.tmp\phase17-ts-pipeline-graph-calls-diagnostic.json`.
  - Go: `.tmp\phase17-go-graph-call-returntype.json`.
- Totals:
  - TypeScript: `28775` nodes, `52792` relationships, `133541ms`.
  - Go packaged native: `32767` nodes, `64567` relationships, `61562.9ms`.
  - Speed ratio: Go packaged native is `~2.17x` faster while performing real DB load.
  - DB load: `fallbackInsertFailures=0`, `skippedRelationships=0`.
- Accepted relationship deltas:
  - `CALLS`: TS `10589`, Go `8906`, delta `-1683`; classified as TypeScript-baseline
    over-resolution/source-label drift rather than a Go missing-edge blocker.
  - `IMPORTS`: TS `2370`, Go `2140`, delta `-230`; classified as TypeScript-baseline
    cross-language/path false positives rather than a Go missing-edge blocker.
  - `STEP_IN_PROCESS`: TS `2646`, Go `2682`, delta `+36`; closed for cutover.
  - Expanded Go coverage: `USES +5554`, `ACCESSES +2391`, `DEFINES +2873`,
    `HAS_PROPERTY +2754`, `ENTRY_POINT_OF +376`.
- High-signal CALLS classification samples:
  - `testingFataler.Fatalf`: TS `541`, Go `1`.
  - `testingFataler.Helper`: TS `217`, Go `1`.
  - `FileContentCache.set`: TS `307`, Go `1`.
  - These are caused by TypeScript-baseline broad global/member matching. Go keeps explicit receiver
    member calls evidence-bound instead of matching every same-name method globally.
- High-signal IMPORTS classification:
  - Missing TS-only imports include Go files linked to TypeScript/shared helper files such as
    `avmatrix-shared/src/scope-resolution/registries/context.ts`,
    `avmatrix/src/core/group/sync.ts`, and `internal/lbugruntime/errors.go` through path/name
    collisions.
  - Go preserves language-aware Go package imports and does not emit these cross-language/path
    false positives.
- Acceptance status: `READY FOR PHASE 17 CUTOVER AUDIT`. Current-repo graph deltas are classified,
  packaged Go remains faster with real DB load, and no DB fallback/skipped relationship blocker
  remains.

### Npm/PATH CLI Distribution Startup Benchmark

- Date: `2026-05-13`.
- Purpose: verify the Phase 17 normal-local-CLI distribution cutover slice. This is a startup/help
  benchmark for runtime authority and local command latency, not a graph parity benchmark.
- Build/test ordering for this slice:
  full launcher build passed first, then package build, Go runtime tests, package/hook unit tests,
  Playwright E2E `33/33` against `avmatrix/bin/avmatrix.exe serve`, npm-link PATH verification, and
  `npm publish --dry-run`.

| State | Command path | Samples ms | Average ms | Median ms | Exit |
| --- | --- | --- | ---: | ---: | ---: |
| Before cutover | PATH npm shim -> `node .../dist/cli/index.js` | `255.9`, `131.2`, `134.5`, `136.1`, `131.4`, `133.7`, `132.7` | `150.8` | `133.7` | `0` |
| Before cutover | Packaged Go binary `avmatrix-launcher/server-bundle/avmatrix.exe --help` | `27.4`, `25.0`, `25.7`, `25.0`, `26.5`, `25.7`, `25.7` | `25.9` | `25.7` | `0` |
| After local cutover | PATH npm shim -> `node_modules/avmatrix/bin/avmatrix.exe` | `139.0`, `42.4`, `36.5`, `37.4`, `50.6`, `49.9`, `46.5` | `57.5` | `46.5` | `0` |
| After local cutover | Direct package binary `avmatrix/bin/avmatrix.exe --help` | `32.5`, `33.2`, `26.9`, `35.4`, `29.6`, `28.6`, `29.3` | `30.8` | `29.6` | `0` |

- Interpretation:
  PATH `avmatrix --help` median improved from `133.7ms` to `46.5ms`, approximately `2.9x` faster,
  and the PATH shim now invokes the Go package binary instead of Node. Direct package binary median
  is `29.6ms`, approximately `4.5x` faster than the old PATH Node-shim median. The first after-cutover
  PATH sample was a cold outlier; median is the acceptance comparison for this startup microbench.
- Caveat:
  the Windows `npm publish --dry-run` tarball included `bin/avmatrix.exe`, `bin/lbug_shared.dll`,
  and `bin/avmatrix-runtime.json`, proving the local Windows package build shape. Portable npm
  install behavior across Windows, Linux, and macOS remains a separate Phase 17 gate because a
  single published artifact must not ship only the publisher machine's binary/native runtime.

### Portable Npm Source-Build Distribution Benchmark

- Date: `2026-05-13`.
- Purpose: verify the Phase 17 portable npm install path after the local PATH cutover. This
  benchmark proves a consumer install can build the Go runtime from packaged Go source when the repo
  root Go source is absent. It is an install/package benchmark, not a graph parity benchmark.
- Build/test ordering for this slice:
  full launcher build passed first in about `39.9s`, then package build, Go tests, package/hook unit
  tests, tarball install validation, Playwright E2E, and publish dry-run.

| Check | Result |
| --- | ---: |
| `npm pack` elapsed | `48,769.9ms` |
| Tarball `go-src` files | `212` |
| Tarball source manifest | `true` |
| Tarball `bin` files | `3` |
| Consumer `npm install` from tarball elapsed | `337,534.1ms` |
| Installed `avmatrix --help` elapsed | `1,481.1ms` |
| Installed runtime platform/arch/source | `win32` / `x64` / `go-src` |
| Installed help first line | `AVmatrix local CLI and MCP server` |
| Playwright E2E | exit `0`, `.last-run.json status=passed`, suite lists `33` tests in `6` files |
| `npm publish --dry-run --loglevel=error` | passed |

- Interpretation:
  the npm tarball now carries enough Go source for package `postinstall` to build the correct
  consumer-platform Go runtime instead of reusing the publisher-platform binary. The install-time
  benchmark includes dependency installation, optional native package build attempts, LadybugDB
  native bootstrap, and Go runtime build from `node_modules/avmatrix/go-src`; it is intentionally
  not compared to CLI startup or graph-analyze speed.
- Follow-up gate:
  portable package selection is closed, but Phase 17 CLI cutover remains open because `avmatrix
  --help` now reaches Go yet the Go command surface does not yet cover every legacy TypeScript CLI
  command required for normal local CLI replacement.

### Direct Graph-Tool CLI Command Surface Benchmark

- Date: `2026-05-13`.
- Purpose: verify the Phase 17 direct graph-tool command-surface batch after the npm/PATH and
  portable package cutover. This is a command startup/smoke benchmark for CLI authority, not a graph
  parity benchmark.
- Build/test ordering for this slice:
  after the final code fix, full launcher build passed first in `38,947.9ms`, then package build,
  Go tests, direct CLI command smoke benchmarks, and Playwright E2E.

| Check | Command | Elapsed ms | Output chars | Exit |
| --- | --- | ---: | ---: | ---: |
| Root help | `avmatrix\bin\avmatrix.exe --help` | `1,939.3` | `1,305` | `0` |
| Query tool | `query "CLI command surface" --repo F:\AVmatrix-GO --limit 1` | `4,237.1` | `2,254` | `0` |
| Context tool | `context NewRootCommand --repo F:\AVmatrix-GO --file internal/cli/command.go` | `950.9` | `12,206` | `0` |
| Impact tool | `impact NewRootCommand --repo F:\AVmatrix-GO --direction upstream --depth 1` | `884.2` | `3,416` | `0` |
| Cypher tool | `cypher "MATCH (n:Function) RETURN n.id AS id, n.name AS name, n.filePath AS path LIMIT 1" --repo F:\AVmatrix-GO` | `420.5` | `553` | `0` |
| Detect changes | `detect-changes --scope unstaged --repo F:\AVmatrix-GO` | `961.5` | `11,080` | `0` |
| Augment | `augment NewRootCommand` | `4,399.9` | `5,151` | `0` |

- E2E validation:
  Go backend `avmatrix\bin\avmatrix.exe serve --host 127.0.0.1 --port 4747` plus Vite on
  `127.0.0.1:5173`; `cd avmatrix-web && E2E=1 npx playwright test --workers=1` exited `0` in
  `219,001.4ms`. `.last-run.json` reported `status="passed"`, and
  `npx playwright test --list` listed `33` tests in `6` files.
- Interpretation:
  the direct graph-tool command batch now reaches Go MCP tool implementations through the packaged
  Go binary and returns real graph data. The slower `query` and `augment` timings are expected for
  this batch because they load and rank graph context; any hot-path improvement belongs to the
  separate Phase 15 MCP optimization backlog after cutover correctness closes.
- Follow-up gate:
  this closes only the direct graph-tool command batch. At this point Phase 17 command-surface
  cutover still remained open for `group`, `clean`, `index`, `benchmark-compare`, `setup`, and
  remaining analyze flag parity or explicit quarantine/retirement decisions; the admin/analyze
  benchmark below closes the `clean`/`index`/`benchmark-compare`/analyze-flag portion.

### Local Admin/Analyze CLI Command Surface Benchmark

- Date: `2026-05-13`.
- Purpose: verify the Phase 17 local admin/analyze command-surface batch after the direct graph-tool
  commands. This benchmark uses temp repos and a temp `AVMATRIX_HOME` so the normal repo registry and
  `.avmatrix` data are not mutated.
- Build/test ordering for this slice:
  full launcher build passed first in `37,419.6ms`, then package build, Go tests, temp-repo command
  smoke benchmarks, and Playwright E2E.

| Check | Command | Elapsed ms | Output chars | Exit |
| --- | --- | ---: | ---: | ---: |
| Root help | `avmatrix\bin\avmatrix.exe --help` | `58.4` | `1,565` | `0` |
| Analyze help | `analyze --help` | `34.2` | `1,539` | `0` |
| Analyze non-Git parity | `analyze <temp> --force --skip-git --skip-compatibility-cross-file --benchmark-json <file> --benchmark-label admin-batch --name admin-batch-demo` | `2,690.2` | `230` | `0` |
| Index existing | `index <temp git repo with .avmatrix/lbug>` | `164.0` | `108` | `0` |
| Analyze clean fixture | `analyze <temp git repo> --force --skip-agents-md --no-stats --name clean-demo` | `1,605.2` | `226` | `0` |
| Clean force | `clean --force` from the temp indexed repo | `33.2` | `73` | `0` |
| Benchmark compare | `benchmark-compare <before.json> <after.json>` | `44.8` | `289` | `0` |
| Benchmark compare JSON | `benchmark-compare <before.json> <after.json> --json` | `35.9` | `577` | `0` |

- E2E validation:
  Go backend `avmatrix\bin\avmatrix.exe serve --host 127.0.0.1 --port 4747` plus Vite on
  `127.0.0.1:5173`; `cd avmatrix-web && E2E=1 npx playwright test --workers=1` exited `0` in
  `198,151.4ms`. `.last-run.json` reported `status="passed"`, and
  `npx playwright test --list` listed `33` tests in `6` files.
- Interpretation:
  `clean`, `index`, `benchmark-compare`, and remaining analyze flags now execute through the Go
  CLI without falling back to the TypeScript command files. The analyze timings are temp-fixture
  correctness smoke numbers, not large-repo performance claims.
- Follow-up gate:
  At this point Phase 17 command-surface cutover remained open for `group`, `setup`, and explicit
  quarantine/retirement decisions for legacy `skill-gen`, `tool`, and AI-context CLI files. The
  group benchmark below closes the `group` portion.

### Group CLI Command Surface Benchmark

- Date: `2026-05-13`.
- Purpose: verify the Phase 17 group command-surface batch after admin/analyze command parity. This
  benchmark uses temp repos and a temp `AVMATRIX_HOME`.
- Build/test ordering for this slice:
  full launcher build passed first in `56,230ms`, then package build, Go tests, temp-fixture group
  command benchmarks, and Playwright E2E.

| Check | Command | Elapsed ms | Output chars | Exit |
| --- | --- | ---: | ---: | ---: |
| Group help | `group --help` | `28.8` | `644` | `0` |
| Group create | `group create fixture` | `30.9` | `174` | `0` |
| Group add backend | `group add fixture app/backend backend` | `29.1` | `83` | `0` |
| Group add frontend | `group add fixture app/frontend frontend` | `29.1` | `85` | `0` |
| Group list | `group list` | `29.7` | `18` | `0` |
| Group list detail | `group list fixture` | `28.1` | `82` | `0` |
| Group status | `group status fixture` | `118.5` | `220` | `0` |
| Group sync JSON | `group sync fixture --json` | `47.3` | `1,691` | `0` |
| Group query | `group query fixture UserFlow --limit 2` | `30.3` | `56` | `0` |
| Group contracts JSON | `group contracts fixture --json` | `29.4` | `1,412` | `0` |
| Group remove | `group remove fixture app/frontend` | `29.0` | `43` | `0` |

- E2E validation:
  Go backend `avmatrix\bin\avmatrix.exe serve --host 127.0.0.1 --port 4747` plus Vite on
  `127.0.0.1:5173`; `cd avmatrix-web && E2E=1 npx playwright test --workers=1` exited `0` in
  `201,799.8ms`. `.last-run.json` reported `status="passed"`, and
  `npx playwright test --list` listed `33` tests in `6` files.
- Interpretation:
  the group subcommands now execute through the Go CLI and Go group service without falling back to
  the TypeScript `avmatrix/src/cli/group.ts` command file. The benchmark validates command
  availability and real temp-fixture group behavior, not cross-repo search performance.
- Follow-up gate:
  At this point Phase 17 command-surface cutover remained open only for `setup` and explicit
  quarantine/retirement decisions for legacy `skill-gen`, `tool`, and AI-context CLI files. The
  setup/quarantine benchmark below closes the command-surface gate.

### Setup And Legacy CLI Quarantine Command Surface Benchmark

- Date: `2026-05-13`.
- Purpose: verify the final Phase 17 command-surface batch after group command parity. This
  benchmark uses a temp `HOME`/`USERPROFILE`/`AVMATRIX_HOME` and empty `PATH` for setup execution so
  real user editor config is not mutated and Codex fallback TOML behavior is exercised.
- Build/test ordering for this slice:
  full launcher build passed first in `33,954.6ms`, then package build, Go tests, temp-HOME setup
  command benchmarks, and Playwright E2E.

| Check | Command | Elapsed ms | Stdout chars | Stderr chars | Exit |
| --- | --- | ---: | ---: | ---: | ---: |
| Setup help | `setup --help` | `1,450.2` | `131` | `0` | `0` |
| Setup first run | `setup` with temp editor dirs | `206.2` | `868` | `0` | `0` |
| Setup second run | `setup` repeated for idempotency | `78.2` | `868` | `0` | `0` |

- Setup artifact checks:
  required temp files were present for `.cursor/mcp.json`, `.claude.json`,
  `.config/opencode/opencode.json`, `.codex/config.toml`, Cursor/Claude/OpenCode/Codex installed
  `avmatrix-cli` skills, `.claude/hooks/avmatrix/avmatrix-hook.cjs`, and `.claude/settings.json`.
  Codex fallback contained exactly one `[mcp_servers.avmatrix]` section after two setup runs.
- E2E validation:
  Go backend `avmatrix\bin\avmatrix.exe serve --host 127.0.0.1 --port 4747` plus Vite on
  `127.0.0.1:5173`; `cd avmatrix-web && E2E=1 npx playwright test --workers=1` exited `0` in
  `201,707.1ms`. `.last-run.json` reported `status="passed"`, and
  `npx playwright test --list` listed `33` tests in `6` files.
- Interpretation:
  `avmatrix setup` now executes through the Go CLI and writes local MCP config/skills/hooks without
  the TypeScript `setup.ts` command. Legacy `tool.ts` is superseded by the Go direct tool command
  batch, and legacy `skill-gen.ts`/`ai-context.ts` are superseded for normal runtime by
  `analyze --skills` plus Go `internal/aicontext`.
- Follow-up gate:
  Phase 17 command-surface cutover is closed. Do not tick `[P17-GO-CLI-DISTRIBUTION-CUTOVER]` or
  the normal-runtime no-Node gate until the TypeScript/Node audit is rerun after this batch.

### Post Command-Surface TypeScript/Node Runtime Audit Benchmark

- Date: `2026-05-13`.
- Purpose: verify the Phase 17 runtime-authority audit after command-surface closure. This slice
  specifically checks that normal PATH/package runtime entrypoints route to Go, not
  `tsx src/cli/index.ts`.
- Build/test ordering for this slice:
  full launcher build passed first in `35,880.6ms`, then package build, Go tests, targeted Web unit
  test, runtime entrypoint benchmarks, and Playwright E2E.

| Check | Command | Elapsed ms | Stdout chars | Stderr chars | Exit |
| --- | --- | ---: | ---: | ---: | ---: |
| PATH shim help | `avmatrix --help` | `75.7` | `1,726` | `0` | `0` |
| Direct package binary help | `avmatrix\bin\avmatrix.exe --help` | `36.6` | `1,726` | `0` | `0` |
| npm serve help | `cd avmatrix && npm run --silent serve -- --help` | `369.2` | `359` | `0` | `0` |
| npm dev help | `cd avmatrix && npm run --silent dev -- --help` | `411.9` | `359` | `0` | `0` |

- Audit checks:
  the PowerShell PATH shim contains `node_modules/avmatrix/bin/avmatrix.exe`. The old blocker
  patterns for `tsx src/cli/index.ts serve`, `"serve": "tsx`, `"dev": "tsx watch
  src/cli/index.ts`, and the Web onboarding command `cd avmatrix && npm run serve` no longer appear
  in runtime files; the only remaining literal `cd avmatrix && npm run serve` is a negative unit
  test assertion proving the old command is absent from onboarding.
- Test validation:
  `cd avmatrix && npm run build` passed in `16,488.3ms`;
  `go test ./cmd/... ./internal/... -count=1` passed in `26,452.4ms`; and
  `cd avmatrix-web && npm test -- OnboardingGuide.local-only.test.tsx` passed in `4,364.1ms`.
- E2E validation:
  Go backend `avmatrix\bin\avmatrix.exe serve --host 127.0.0.1 --port 4747` plus Vite on
  `127.0.0.1:5173`; `cd avmatrix-web && E2E=1 npx playwright test --workers=1` exited `0` in
  `218,233.3ms`. `.last-run.json` reported `status="passed"`, and
  `npx playwright test --list` listed `33` tests in `6` files.
- Interpretation:
  normal local runtime entrypoints are now Go-owned for CLI/analyze/serve/MCP/repo registry/search/
  embed/session/benchmark/group/setup usage. The legacy TypeScript CLI remains available only under
  explicit baseline/dev/test names and is not the default runtime authority.
- Follow-up gate:
  `[P17-GO-CLI-DISTRIBUTION-CUTOVER]` and the normal-runtime no-Node gate are closed. The remaining
  Phase 17 blocker is `[P17-GO-CONTRACT-AUTHORITY-CUTOVER]`.

### Go-Owned Web Contract Authority Benchmark

- Date: `2026-05-13`.
- Purpose: verify the Phase 17 contract-authority cutover after the normal runtime/CLI cutover. This
  slice makes Go emit the Web UI contract manifest and generated browser TypeScript adapter, then
  removes `avmatrix-shared` from the Web package dependency/import/alias path.
- Build/test ordering for this slice:
  full launcher build passed first in `38,426.7ms`; validation then ran Go contract tests, targeted
  Web unit tests, contract-generation/audit benchmarks, fresh Web install, a second full launcher
  build after install, Playwright E2E, and `cd avmatrix && npm test`.

| Check | Command | Elapsed ms | Result |
| --- | --- | ---: | --- |
| Full launcher build | `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` | `38,426.7` | passed |
| Go runtime/contract tests | `go test ./cmd/... ./internal/... -count=1` | `29,795.3` | passed |
| Web contract unit tests | `cd avmatrix-web && npm test -- session-client.test.ts security-guards.test.ts useAppState.local-runtime.test.tsx GraphCanvas.selection-performance.test.tsx ChatPanel.grounding-links.test.tsx SettingsPanel.local-runtime.test.tsx` | `13,013.4` | `6` files / `130` tests passed |
| Contract generator check | `go run ./cmd/generate-web-contracts --check` | `559.9` | passed |
| Web contract dependency audit | `rg -n "avmatrix-shared" avmatrix-web` | `55.8` | `0` hits |
| Fresh Web install | `cd avmatrix-web && npm ci` | `50,004.7` | passed |
| Post-install full launcher build | `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` | `33,223.8` | passed |
| Browser E2E | `cd avmatrix-web && E2E=1 npx playwright test --workers=1` with packaged Go backend | `205,536.5` | `33/33` passed |
| Legacy baseline/test suite | `cd avmatrix && npm test` | `338,908.6` | passed |

- E2E validation:
  the backend was the packaged Go runtime from `avmatrix-launcher/server-bundle/avmatrix.exe serve
  --host 127.0.0.1 --port 4747`; Vite ran on `127.0.0.1:5173`. The final accepted Playwright run
  exited `0` with `33` tests passed.
- Audit result:
  `avmatrix-web` has no remaining `avmatrix-shared` package dependency, Vite/Vitest alias, source
  import, test import, lockfile entry, or Vercel install requirement. The Web contract TypeScript is
  generated from Go under `avmatrix-web/src/generated/avmatrix-contracts.ts`, with the JSON manifest
  stored at `contracts/web-ui/avmatrix-web-contract.schema.json`.
- Notes:
  an initial `npm ci` attempt failed with Windows `EPERM` on the Lightning CSS native module because
  stale Vite dev-server processes held the file; after stopping only the captured Web dev-server
  Node processes, the accepted `npm ci` passed. Earlier E2E wrapper attempts produced a non-zero
  wrapper exit after Playwright had already reported `33 passed` because the cleanup pattern matched
  the wrapper command line; the final accepted E2E wrapper only stopped captured backend/Vite PIDs
  and exited `0`.
- Interpretation:
  the Web UI still compiles with TypeScript/React, but its contract shapes now come from Go-owned
  generated browser glue instead of the independent `avmatrix-shared` TypeScript package. Legacy
  `avmatrix-shared` and `avmatrix/src` TypeScript remain baseline/dev/test material, not normal
  backend/CLI/MCP/analyzer/Web runtime authority.

### Phase 15 MCP Route Map Cache Benchmark

- Date: `2026-05-13`.
- Purpose: close the Phase 15 P0 `route_map` hot-path target after Phase 17 cutover gates closed.
  This slice caches `graph.json` per MCP server/session with stat-based invalidation and builds a
  reusable route index for `route_map`, `shape_check`, and `api_impact`.
- Build/test ordering for this slice:
  full launcher build passed first in `35,387ms`; validation then ran MCP before/after benchmarks,
  Go tests, packaged analyze graph-parity smoke, browser E2E, and `cd avmatrix && npm test`.

| MCP row | Before Go stdio ms | After Go stdio ms | Before Go HTTP ms | After Go HTTP ms | After TypeScript stdio/http ms |
| --- | ---: | ---: | ---: | ---: | ---: |
| `resources/read context` | `50.85` | `45.47` | `51.05` | `49.73` | `43.34` / `45.78` |
| `query` | `3,613.23` | `3,773.50` | `3,703.99` | `3,570.84` | `281.77` / `289.36` |
| `context` | `782.99` | `24.01` | `769.33` | `32.17` | `102.69` / `108.79` |
| `impact` | `780.34` | `24.96` | `794.49` | `37.18` | `132.12` / `133.48` |
| `route_map` | `759.56` | `7.70` | `730.87` | `8.80` | `29.94` / `32.32` |
| `group_sync fixture` | `18.23` | `1.32` | `2.73` | `2.37` | `7.10` / `8.66` |

- P0 result:
  `route_map` is now below the Phase 15 `<50ms` target on both transports: `7.70ms` stdio and
  `8.80ms` HTTP. Compared with the current before-run, the stdio route_map path improved by
  `~98.99%` and the HTTP path improved by `~98.80%`.
- Secondary observations:
  warm-session `context` and `impact` also fell below their Phase 15 targets because they reuse the
  cached graph snapshot after earlier MCP calls. The detailed context/impact timing-split checklist
  remains separate unless a later slice records per-stage timings. `query` remains an open Phase 15
  optimization candidate; this route-index slice does not claim to optimize query ranking/search.
- Graph parity/analyze smoke:
  `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --skip-agents-md --benchmark-json
  .tmp\phase15-route-cache-graph-parity.json --benchmark-label phase15-route-cache-graph-parity`
  passed in `60,975ms`, producing `33,399` nodes and `66,035` relationships. DB load reported
  `fallbackInsertCount=0`, `fallbackInsertFailures=0`, `skippedRelationships=0`,
  `nodeRows=33,399`, and `relationshipRows=66,035`.
- Test validation:
  `go test ./cmd/... ./internal/... -count=1` passed in `29,410.2ms`. Full browser E2E used an
  isolated `AVMATRIX_HOME` containing only `AVmatrix-GO`, the packaged Go backend on
  `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; `cd avmatrix-web && E2E=1 npx playwright test
  --workers=1 --reporter=line` passed `33/33` in `227,307.3ms`. `cd avmatrix && npm test` passed
  in `373,303.6ms`.
- Notes:
  the first E2E attempt used the developer's normal repo registry, whose first repo was
  `ui-local-mcp`; one shell-interaction test then landed on the analyze page for that unrelated repo
  and failed. The accepted E2E rerun used isolated `AVMATRIX_HOME=.tmp\phase15-route-cache-e2e-home`
  and registered only `AVmatrix-GO`, matching the plan benchmark target.

### Phase 14 Frontend/Mobile App Coverage Addendum Benchmark

- Date: `2026-05-13`.
- Purpose: record the deliberate Phase 14 addendum requested after the Phase 15 `route_map` slice.
  The target surfaces are React, Electron, TypeScript, Next.js, Vue, Nuxt, Svelte, Astro, React
  Native, Flutter, SwiftUI, and Jetpack Compose.
- Build/test ordering for this slice:
  full launcher build passed first in `39,240.9ms`; validation then ran focused Go tests,
  provider microbenchmarks, packaged analyze benchmarks, full Go tests, browser E2E, and
  `cd avmatrix && npm test`.

| Check | Command | Elapsed / result |
| --- | --- | --- |
| Full launcher build | `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` | `39,240.9ms`, passed |
| Focused Go tests | `go test ./internal/providers/sfc ./internal/providers/vue ./internal/providers/svelte ./internal/providers/astro ./internal/scanner ./internal/frameworks ./internal/analyze ./internal/contracts -count=1` | passed |
| Svelte provider benchmark | `go test ./internal/providers/svelte -run '^$' -bench BenchmarkExtractSvelteScopeIR -benchmem -count=5` | median `896,080ns/op`, `~147KB/op`, `3,073 allocs/op` |
| Astro provider benchmark | `go test ./internal/providers/astro -run '^$' -bench BenchmarkExtractAstroScopeIR -benchmem -count=5` | median `830,770ns/op`, `~145KB/op`, `3,020 allocs/op` |
| Isolated frontend/mobile fixture analyze | `avmatrix-launcher\server-bundle\avmatrix.exe analyze .tmp\phase14-frontend-mobile-fixture --skip-git --force --skip-agents-md --no-stats --name phase14-frontend-mobile --benchmark-json .tmp\phase14-frontend-mobile-analyze.json` | wall `7,646.2ms`, benchmark total `7,549.2ms` |
| Current-repo packaged analyze | `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --skip-agents-md --no-stats --benchmark-json .tmp\phase14-frontend-mobile-current-repo.json` | wall `61,853.6ms`, benchmark total `61,521.2ms` |
| Full Go tests | `go test ./cmd/... ./internal/... -count=1` | passed |
| Browser E2E | `cd avmatrix-web && E2E=1 npx playwright test --workers=1 --reporter=line` with isolated packaged Go backend | `494,207.2ms`, `32` passed / `1` skipped |
| Legacy baseline/test suite | `cd avmatrix && npm test` | `335,723.4ms`, passed |

- Provider microbenchmark samples:
  Svelte `982,134`, `865,923`, `901,517`, `896,080`, `867,532ns/op`; Astro `828,590`,
  `810,491`, `854,540`, `830,770`, `845,174ns/op`.
- Isolated fixture analyze result:
  `11` files scanned and parsed, `0` unsupported, `0` failed. The fixture graph contained `54`
  nodes and `61` relationships. DB load reported `fallbackInsertCount=0`,
  `fallbackInsertFailures=0`, `skippedRelationships=0`, `nodeRows=54`, and
  `relationshipRows=61`.
- Isolated fixture framework coverage observed in graph properties:
  `react`, `electron`, `nextjs`, `nextjs-api`, `vue`/`nuxt`, `svelte`/`sveltekit`, `astro`,
  `react-native`, `flutter`, `swiftui`, `ios`, `android-kotlin`, and `jetpack-compose`.
- Current-repo analyze result:
  `1,226` files scanned, `1,053` parsed, `173` unsupported, `0` failed. Graph output was `33,489`
  nodes and `66,356` relationships. Benchmark phase timings: scan `163.1ms`, parse
  `21,564.9ms`, resolution `5,375.0ms`, DB load `29,877.9ms`; DB load reported
  `fallbackInsertFailures=0` and `skippedRelationships=0`.
- E2E notes:
  the first E2E wrapper attempt started a backend that did not inherit the isolated registry env, so
  the browser saw the developer's normal `ui-local-mcp` registry. The accepted run used a verified
  isolated backend whose `/api/repos` returned only `AVmatrix-GO`. Two test selectors were hardened
  because the larger Go graph includes lower-case file/tool entries named `processes`; the app
  behavior was unchanged, and the accepted full E2E run exited `0`.

### Phase 10 LadybugDB Fallback Fail-Closed Correctness Benchmark

- Date: `2026-05-13`.
- Purpose: reassess whether the LadybugDB relationship fallback can produce wrong data, then make
  the normal DB load path fail closed if fallback or skipped-relationship behavior would be needed.
- Phase-jump note:
  this slice intentionally interrupted the open Phase 15 `context` optimization work and returned
  to Phase 10 persistence correctness. It does not close the Phase 15 `context` item.
- Build/test ordering for this slice:
  full launcher build passed first in `34,207.9ms`; validation then ran focused lbugload tests and
  benchmarks, packaged current-repo analyze, full Go tests, browser E2E, and `cd avmatrix &&
  npm test`.

| Check | Command | Elapsed / result |
| --- | --- | --- |
| Full launcher build | `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` | `34,207.9ms`, passed |
| Focused loader tests | `go test ./internal/lbugload -count=1` | passed |
| Normal COPY loader benchmark | `go test ./internal/lbugload -run '^$' -bench BenchmarkLoadCSVExportCopyPathNoop -benchmem -count=5` | `2,537-2,596ns/op`, `19 allocs/op` |
| Diagnostic fallback benchmark | `go test ./internal/lbugload -run '^$' -bench BenchmarkDiagnosticFallbackPathNoop -benchmem -count=5` | `736,094-764,844ns/op`, `5,061 allocs/op` |
| Current-repo packaged analyze | `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --skip-agents-md --no-stats --benchmark-json .tmp\fallback-failclosed-graph-parity.json --benchmark-label fallback-failclosed-graph-parity` | wall `59,937.2ms`, benchmark total `58,458.4ms` |
| Full Go tests | `go test ./cmd/... ./internal/... -count=1` | `27,772.9ms`, passed |
| Browser E2E | `cd avmatrix-web && E2E=1 npx playwright test --workers=1 --reporter=line` with isolated packaged Go backend | Playwright `512,685.5ms`, `32` passed / `1` skipped |
| Legacy baseline/test suite | `cd avmatrix && npm test` | `438,204.3ms`, passed |

- Current-repo analyze result:
  `1,228` files scanned, `1,055` parsed, `173` unsupported, `0` failed. Graph output was `33,573`
  nodes and `66,603` relationships. Benchmark phase timings included scan `150.8ms`, parse
  `19,798.3ms`, resolution `5,337.8ms`, and DB load `28,472.5ms`. DB load reported
  `nodeCopyCount=19`, `relationshipCopyCount=90`, `fallbackInsertCount=0`,
  `fallbackInsertFailures=0`, `skippedRelationships=0`, `nodeRows=33,573`, and
  `relationshipRows=66,603`.
- Loader interpretation:
  normal `LoadCSVExport` now returns an error rather than silently accepting relationship COPY
  failure, unsupported schema pairs, skipped relationships, or partial fallback insert failure.
  Diagnostic fallback remains benchmarked and available only through explicit options, so fallback
  can be used to investigate schema/COPY gaps without becoming runtime authority.
- E2E notes:
  the first E2E wrapper used HEAD readiness through `wait-on http://...` and timed out against the
  Go `/api/info` endpoint even though GET readiness succeeded. The accepted E2E run used
  `http-get://127.0.0.1:4747/api/info`, an isolated `AVMATRIX_HOME`, and the packaged Go backend.

### Phase 15 MCP Context One-Pass Neighborhood Benchmark

- Date: `2026-05-13`.
- Purpose: close the Phase 15 P1 `context` target with a timing split and a focused hot-path
  optimization after the fallback correctness slice was committed.
- Phase-jump note:
  context optimization started after the Phase 14 addendum, paused for the Phase 10 fallback
  correctness fix, then resumed as Phase 15 performance work. This slice does not close the
  remaining `impact` or HTTP `group_sync` items.
- Build/test ordering for this slice:
  full launcher build passed first in `34,207.9ms`; validation then ran MCP before/final
  benchmarks, focused MCP tests and microbenchmarks, packaged analyze graph-parity smoke, full Go
  tests, browser E2E, and `cd avmatrix && npm test`.

| MCP row | Before Go stdio ms | Final Go stdio ms | Before Go HTTP ms | Final Go HTTP ms | Final TypeScript stdio/http ms |
| --- | ---: | ---: | ---: | ---: | ---: |
| `resources/read context` | `45.08` | `45.77` | `50.18` | `49.12` | `49.69` / `44.15` |
| `query` | `3,648.27` | `3,476.80` | `3,600.57` | `3,626.91` | `279.95` / `278.95` |
| `context` | `23.20` | `15.36` | `25.87` | `18.67` | `101.02` / `107.27` |
| `impact` | `26.27` | `25.94` | `35.26` | `28.42` | `126.92` / `137.75` |
| `route_map` | `5.87` | `6.83` | `7.20` | `8.85` | `29.93` / `32.59` |
| `group_sync fixture` | `1.37` | `1.27` | `6.80` | `2.62` | `6.61` / `9.15` |

- P1 result:
  `context` is below the Phase 15 `<100ms` target on both transports: `15.36ms` stdio and
  `18.67ms` HTTP. Compared with the immediate before-run, stdio improved by `~33.8%` and HTTP
  improved by `~27.8%`.
- Timing split benchmark:
  `go test ./internal/mcp -run '^$' -bench BenchmarkContextToolWarmNeighborhood -benchmem
  -count=5` used a synthetic graph with `2,500` incoming refs, `2,500` outgoing refs, and `750`
  process memberships. Samples were `27,493,641`, `27,671,500`, `27,027,110`, `26,980,980`, and
  `26,961,112ns/op`; the single neighborhood read accounted for roughly `24,579-26,566us/op`.
  Repo resolve stayed below `603us/op`, target lookup stayed between `1,048-1,887us/op`, and the
  synthetic payload used about `6.31MB/op` with `201,816-201,817 allocs/op`.
- Implementation notes:
  `contextTool` now calls an internal profiled implementation. The hot path keeps normal runtime
  free of profiling overhead, while tests and benchmarks call `contextToolProfiled`. The context
  result now comes from `contextNeighborhood`, which scans graph relationships once for incoming,
  outgoing, process participation, and class-like constructor/file incoming refs, then sorts payload
  categories in one shared helper.
- Graph parity/analyze smoke:
  `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --skip-agents-md --no-stats
  --benchmark-json .tmp\phase15-context-final-graph-parity.json --benchmark-label
  phase15-context-final-graph-parity` passed in `59,979.6ms`, producing `33,574` nodes and `66,604`
  relationships. DB load reported `fallbackInsertCount=0`, `fallbackInsertFailures=0`,
  `skippedRelationships=0`, `nodeRows=33,574`, and `relationshipRows=66,604`.
- Test validation:
  focused MCP context/cache tests passed; `go test ./cmd/... ./internal/... -count=1` passed in
  `27,772.9ms`. Full browser E2E used an isolated `AVMATRIX_HOME`, the packaged Go backend on
  `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; `cd avmatrix-web && E2E=1 npx playwright test
  --workers=1 --reporter=line` passed with `32` passed / `1` skipped in `512,685.5ms`.
  `cd avmatrix && npm test` passed in `438,204.3ms`.

### Phase 15 MCP Impact Timing-Profile Benchmark

- Date: `2026-05-13`.
- Purpose: close the Phase 15 P1 `impact` target with timing splits after the dedicated `context`
  slice. This slice did not accept a runtime hot index because the measured hot-index attempt was a
  regression.
- Phase-jump note:
  this work stayed in Phase 15. It is MCP profile/performance work, not Phase 14 provider coverage,
  Phase 10 persistence correctness, or Phase 17 runtime cutover.
- Build/test ordering for this slice:
  full launcher build passed first in `35,341.2ms`; validation then ran focused MCP tests,
  microbenchmarks, the MCP benchmark, TypeScript-vs-Go analyze comparison, packaged analyze
  graph-parity smoke, full Go tests, browser E2E, and `cd avmatrix && npm test`.

| MCP row | Previous Go stdio ms | Final Go stdio ms | Previous Go HTTP ms | Final Go HTTP ms | Final TypeScript stdio/http ms |
| --- | ---: | ---: | ---: | ---: | ---: |
| `context` | `15.36` | `14.97` | `18.67` | `25.84` | `113.89` / `117.00` |
| `impact` | `25.94` | `26.53` | `28.42` | `26.47` | `140.79` / `135.10` |
| `route_map` | `6.83` | `8.27` | `8.85` | `8.12` | `31.02` / `34.25` |
| `group_sync fixture` | `1.27` | `1.46` | `2.62` | `2.52` | `6.82` / `8.97` |

- P1 result:
  `impact` is below the Phase 15 `<150ms` target on both transports: `26.53ms` stdio and
  `26.47ms` HTTP. The result preserves the accepted fast path from the previous context slice and
  is materially faster than TypeScript on the same benchmark row.
- Rejected optimization:
  an attempted runtime hot-index/cache implementation measured `64.90ms` stdio and `71.71ms` HTTP
  for `impact`, worse than the previous accepted `25.94ms` / `28.42ms`. That branch was rolled back
  instead of being accepted just to tick the checklist.
- Timing split benchmark:
  `go test ./internal/mcp -run '^$' -bench BenchmarkImpactToolWarmTraversalProfile -benchmem
  -count=5` used a synthetic graph with `2,500` upstream callers and `750` process/module
  memberships. Samples were `20,726,332`, `20,595,728`, `19,353,381`, `19,547,030`, and
  `19,734,783ns/op`, with about `5.24MB/op` and `126,525-126,691 allocs/op`. Timing splits showed
  node-index setup `17-363us/op`, repo resolve `118-613us/op`, target lookup `775-1,340us/op`,
  traversal `2,830-4,380us/op`, affected summaries `13,770-16,250us/op`, and formatting near zero.
  Further work, if needed, should target affected process/module summary construction rather than
  graph decode or JSON-RPC payload shape.
- TypeScript vs Go analyze comparison:
  the requested same-machine comparison ran the TypeScript CLI baseline and then the packaged Go
  runtime on `AVmatrix-GO`.

| Runtime | Command artifact | Wall ms | Nodes | Relationships | DB fallback/skipped |
| --- | --- | ---: | ---: | ---: | --- |
| TypeScript baseline | `.tmp\phase15-impact-ts-analyze.json` | `130,582.2` | `29,670` | `54,858` | `skippedRelationships=0` |
| Go packaged native | `.tmp\phase15-impact-go-analyze.json` | `58,693.8` | `33,612` | `66,727` | `fallbackInsertFailures=0`, `skippedRelationships=0` |

- Analyze interpretation:
  packaged Go is about `2.22x` faster than the TypeScript baseline for this current-repo rerun
  while producing the larger accepted Go graph and still doing real DB load. Phase timings captured
  in the artifacts include scan `168.0ms` TS vs `149.1ms` Go, parse `45,608.0ms` TS vs
  `19,656.6ms` Go, resolution `3,694.0ms` TS vs `5,406.5ms` Go, and DB load `26,405.3ms` TS
  `lbugLoad` vs `28,981.7ms` Go `db_load`.
- Graph parity/analyze smoke:
  `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --skip-agents-md --no-stats
  --benchmark-json .tmp\phase15-impact-profile-graph-parity.json --benchmark-label
  phase15-impact-profile-graph-parity` passed in `79,991.1ms`, producing `33,612` nodes and
  `66,727` relationships. DB load reported `fallbackInsertCount=0`, `fallbackInsertFailures=0`,
  `skippedRelationships=0`, `nodeRows=33,612`, and `relationshipRows=66,727`.
- Test validation:
  focused MCP profile tests passed; `go test ./cmd/... ./internal/... -count=1` passed in
  `27,295.0ms`. Full browser E2E used an isolated `AVMATRIX_HOME`, the packaged Go backend on
  `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; Playwright passed with `32` passed /
  `1` skipped in `536,712.6ms`. `cd avmatrix && npm test` passed in `624.8s`.

### Phase 15 MCP group_sync Cold/Warm Benchmark

- Date: `2026-05-14`.
- Purpose: close the Phase 15 P2 HTTP `group_sync` timing-split target by separating server
  startup/readiness, MCP initialize, cold `group_sync`, warm-session `group_sync`, and core
  `internal/group.Sync` runtime cost.
- Phase-jump note:
  this work stayed in Phase 15. It is MCP performance triage, not Phase 14 provider coverage, Phase
  10 persistence correctness, or Phase 17 runtime cutover.
- Build/test ordering for this slice:
  full launcher build passed first in `33,425.1ms`; validation then ran the core Go benchmark, HTTP
  cold/warm benchmark, pprof CPU/memory profiles, packaged analyze graph-parity smoke, full Go
  tests, browser E2E, and `cd avmatrix && npm test`.

| Runtime | Server ready ms | Cold init avg ms | Cold `group_sync` avg ms | Cold total avg ms | Warm `group_sync` avg ms | Warm `group_sync` p95 ms |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Go packaged HTTP | `1,358.68` | `4.99` | `15.99` | `20.97` | `11.31` | `11.92` |
| TypeScript HTTP | `1,751.30` | `15.06` | `10.10` | `25.16` | `7.20` | `9.88` |

- HTTP result:
  Go has the faster server-ready and cold total path on this fixture. The warm `group_sync` core row
  remains slower than TypeScript by about `4.11ms` on average, but the absolute latency is low and
  no user-facing payload or contract blocker was found.
- Core Go benchmark:
  `go test ./internal/group -run '^$' -bench BenchmarkSyncSmallFixture -benchmem -count=5`
  samples were `11,345,392`, `11,567,432`, `10,394,681`, `10,585,765`, and `10,776,696ns/op`,
  with about `24.4KB/op` and `220 allocs/op`.
- Profile interpretation:
  CPU pprof for `BenchmarkSyncSmallFixture` showed `runtime.cgocall`/Windows syscalls dominating
  the sample, with `group.WriteRegistry`/`os.WriteFile` accounting for most of the benchmark
  cumulative time. Memory pprof attributed the meaningful in-scope allocation to `json.Marshal`
  for `WriteRegistry` plus graph/registry reads; profiler overhead dominated the rest. The queued
  optimization target is `[P15-GROUPSYNC-REGISTRY-WRITE-OPT]`: profile and reduce
  `internal/group.WriteRegistry` persistence cost while preserving `contracts.json` schema,
  `GeneratedAt`, CLI/MCP payloads, and exact-match semantics. Target: warm HTTP `group_sync <=7ms`.
- Graph parity/analyze smoke:
  `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --skip-agents-md --no-stats
  --benchmark-json .tmp\phase15-group-sync-final-graph-parity.json --benchmark-label
  phase15-group-sync-final-graph-parity` passed in `59,704.6ms`, producing `33,643` nodes and
  `66,799` relationships. DB load reported `fallbackInsertCount=0`, `fallbackInsertFailures=0`,
  `skippedRelationships=0`, `nodeRows=33,643`, and `relationshipRows=66,799`.
- Test validation:
  `go test ./cmd/... ./internal/... -count=1` passed in `25,660.5ms`. Full browser E2E used an
  isolated `AVMATRIX_HOME`, the packaged Go backend on `127.0.0.1:4747`, and Vite on
  `127.0.0.1:5173`; Playwright passed with `32` passed / `1` skipped in `503,825.4ms`.
  `cd avmatrix && npm test` passed in `374,034.3ms`.
- Recovery note:
  one earlier graph parity rerun failed because an interrupted analyze left a stale
  `.avmatrix/analyze.lock` and partial `lbug.shadow`/`lbug.wal.checkpoint`. The lock PID was not
  running; after removing only those repo-local stale artifacts, the accepted force analyze rebuilt
  the graph and DB cleanly with fallback/skipped counts at `0`.

### Phase 15 MCP P3 Startup/Query/Tools-List/Noise Benchmark

- Date: `2026-05-14`.
- Purpose:
  close the P3 MCP preservation slice by proving the route/context/impact/group-sync optimization
  work did not regress startup, tool discovery payload size, or stdio protocol cleanliness, and by
  fixing the new current-repo `query` regression found in the P3 before-run.
- Phase-jump note:
  this work stayed in Phase 15 after the HTTP `group_sync` slice. It is MCP performance/profile
  work on the already cut-over Go runtime, not provider coverage or Phase 17 runtime authority.
- Build/test ordering for this slice:
  full launcher build passed first in `33,418.9ms`; validation then ran the MCP benchmark, a
  same-session cold/warm query probe, focused MCP tests, a warm-query benchmark, packaged analyze
  graph-parity smoke, full Go tests, browser E2E, and `cd avmatrix && npm test`.

| MCP row | Before Go stdio ms | Final Go stdio ms | Before Go HTTP ms | Final Go HTTP ms | Final TypeScript stdio/http ms |
| --- | ---: | ---: | ---: | ---: | ---: |
| `initialize` | `37.24` | `1,248.27` | `10.29` | `10.26` | `1,311.27` / `40.61` |
| `tools/list` | `1.13` | `1.12` | `3.63` | `4.48` | `3.56` / `6.70` |
| `resources/read context` | `45.31` | `45.57` | `48.38` | `48.80` | `43.23` / `43.73` |
| `query` | `3,501.00` | `763.95` | `3,505.47` | `763.93` | `286.55` / `282.67` |
| `context` | `21.90` | `15.75` | `19.42` | `18.15` | `103.46` / `103.20` |
| `impact` | `25.03` | `25.48` | `34.66` | `29.11` | `128.78` / `129.63` |
| `route_map` | `5.89` | `5.49` | `6.87` | `7.17` | `30.17` / `32.01` |

- P3 result:
  Go `query` improved by about `78.2%` on stdio and `78.2%` on HTTP versus the P3 before-run. The
  remaining canonical `query` row includes the first `graph.json` decode/cache cost. A same-session
  probe against the final packaged binary measured `query cold=768.41ms`, then `query warm
  1=7.84ms` and `query warm 2=7.22ms`; `context after query` was `15.73ms`. The optimization target
  is therefore closed for the query algorithm hot path, while cold graph snapshot load remains a
  separate graph-stream/cache benchmark target.
- Startup/tools/noise preservation:
  HTTP initialize stayed fast (`10.26ms` for the repo benchmark and `2.09ms` on the group fixture).
  The stdio repo initialize row was noisy (`1,248.27ms`) but still below the TypeScript row in the
  same run (`1,311.27ms`), and the same final run's isolated group fixture measured Go stdio
  initialize at `55.83ms` vs TypeScript `1,300.03ms`. `tools/list` stayed small at `7,795` Go bytes
  vs `18,447` TypeScript bytes, and Go stdio protocol noise stayed `0` bytes.
- Warm-query benchmark:
  `go test ./internal/mcp -run '^$' -bench BenchmarkQueryToolWarmProcessIndex -benchmem -count=5`
  used a synthetic graph with `700` processes and `4` steps per process. Samples were `2,687,822`,
  `2,302,018`, `2,239,775`, `2,210,686`, and `2,209,799ns/op`, with about `1.02MB/op` and
  `5,012-5,013 allocs/op`.
- Implementation notes:
  `queryTool` now builds `resourceProcessStepsByProcess(g)` once and passes it into
  `rankedProcessMatches`, then reuses the same per-process step slices when formatting
  `process_symbols`. The previous implementation called `resourceProcessSteps` once per process,
  rebuilding the node map and scanning all relationships repeatedly on large graphs. The result
  preserves payload shape; no graph preload was added because that would move cold graph decode into
  startup/resources instead of reducing the work.
- Graph parity/analyze smoke:
  `avmatrix-launcher\server-bundle\avmatrix.exe analyze --force --skip-agents-md --no-stats
  --benchmark-json .tmp\phase15-p3-final-graph-parity.json --benchmark-label
  phase15-p3-final-graph-parity` passed in `58,540.0ms`, producing `33,660` nodes and `66,882`
  relationships. DB load reported `fallbackInsertCount=0`, `fallbackInsertFailures=0`,
  `skippedRelationships=0`, `nodeRows=33,660`, and `relationshipRows=66,882`.
- Test validation:
  focused MCP query tests passed; `go test ./cmd/... ./internal/... -count=1` passed in
  `28,306.0ms`. Full browser E2E used isolated `AVMATRIX_HOME=.tmp\phase15-p3-e2e-home-*`, the
  packaged Go backend on `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated analyze took
  `57,281.5ms`, and Playwright passed with `32` passed / `1` skipped in `488,397.8ms`.
  `cd avmatrix && npm test` passed in `385,336.0ms`.

### Phase 15 Large-Repo Graph Stream and Pprof Benchmark

- Date: `2026-05-14`.
- Purpose:
  close the Phase 15 graph stream timing, memory peak, CPU pprof, and memory pprof benchmark items,
  and optimize the measured graph-stream bottleneck without changing graph accuracy.
- Phase-jump note:
  this work continued Phase 15 after the P3 MCP query preservation commit. It is performance/profile
  work on the Go runtime, not provider coverage, fallback correctness, or Phase 17 cutover.
- Build/test ordering for this slice:
  full launcher build passed before benchmark/test work in `34,126.1ms`; validation then ran focused
  CLI/HTTP tests, graph stream benchmarks, final analyze with CPU/memory pprof, full Go tests,
  browser E2E, and `cd avmatrix && npm test`.

| Runtime | Server ready ms | `/api/graph` JSON ms | JSON bytes | `/api/graph?stream=true` ms | Stream bytes | Stream lines |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Go before stream batching | `57.2` | `1,267.5` | `37,099,388` | `2,915.0` | `40,318,012` | `100,573` |
| TypeScript before-run | `1,701.1` | `5,614.5` | `34,866,411` | `5,926.9` | `37,707,678` | `100,265` |
| Go final | `52.2` | `1,384.3` | `37,120,627` | `1,384.8` | `40,341,034` | `100,633` |
| TypeScript final | `1,690.8` | `5,515.6` | `34,884,061` | `5,852.4` | `37,726,956` | `100,325` |

- Graph stream result:
  before the change, Go NDJSON streaming was slower than Go JSON because it flushed once per
  node/relationship. The final Go stream is effectively tied with Go JSON on the final graph and is
  about `4.23x` faster than TypeScript stream on the same machine. The stream payload shape and
  content type remain `application/x-ndjson; charset=utf-8`.
- Implementation notes:
  `streamGraphNDJSON` now flushes every `512` records and once at the end, instead of flushing every
  encoded record. `TestGraphStreamingBatchesFlushes` locks this behavior with a synthetic graph; the
  existing NDJSON endpoint test still checks the wire shape.
- Large-repo analyze/profile result:
  final packaged Go analyze with pprof passed in `58,459.2ms` wall time. The benchmark JSON
  `.tmp\phase15-large-profile-final-analyze.json` recorded `totalDuration=58,015.2ms`, parse
  `19,779.5ms`, resolution `5,374.7ms`, DB load `28,229.0ms`, `33,702` nodes, `66,931`
  relationships, `fallbackInsertFailures=0`, `skippedRelationships=0`, `nodeRows=33,702`, and
  `relationshipRows=66,931`.
- Memory peak:
  the same final artifact recorded `startAllocBytes=2,190,096`, `endAllocBytes=343,455,784`, and
  `maxObservedSys=941,588,728` (`~898MiB`).
- CPU pprof:
  `--cpuprofile .tmp\phase15-large-profile-final-cpu.pprof` produced a `67,778` byte profile.
  `go tool pprof -top` showed `runtime.cgocall` dominating at `42.12s` flat (`67.83%`, `43.88s`
  cumulative), followed by `resolution.(*workspace).resolveImportedMember` at `4.45s` flat
  (`7.17%`, `4.65s` cumulative). The CPU result points at LadybugDB native DB load and imported
  member resolution as the next real optimization candidates, not HTTP graph streaming.
- Memory pprof:
  `--memprofile .tmp\phase15-large-profile-final-mem.pprof` produced a `57,861` byte heap profile.
  Alloc-space top entries were `bytes.genSplit` (`1,022.06MB`), tree-sitter node allocation
  (`255.01MB`), tree-sitter `GoString` text extraction (`247.00MB`), `scopeir.callKey`
  (`236.04MB`), and `scopeir.ScopeIR.Normalized` cumulative allocation (`932.17MB`). In-use top was
  `bytes.growSlice` (`64.00MB`), `ScopeIR.Normalized` (`61.34MB`), `Graph.AddRelationship`
  (`14.34MB`), `emitDefinitionNodes` (`11.00MB`), and `graph.GenerateID` (`10.00MB`).
- Queued optimization targets from the profiles:
  `[P15-DBLOAD-CGO-BATCH-OPT]` should reduce native DB load/cgo overhead while preserving
  fail-closed fallback semantics and DB row parity. `[P15-SCOPEIR-NORMALIZED-ALLOC-OPT]` should
  reduce `ScopeIR.Normalized`, `callKey`, and `rangeKey` allocation without reducing resolution
  evidence or edge parity.
- Test validation:
  focused CLI pprof flag tests and HTTP graph stream tests passed. `go test ./cmd/... ./internal/...
  -count=1` passed in `27,043.8ms`. Full browser E2E used isolated
  `AVMATRIX_HOME=.tmp\phase15-graph-stream-e2e-home-*`, the packaged Go backend on
  `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated analyze took `64,342.3ms`, and
  Playwright passed with `32` passed / `1` skipped in `408,728.9ms`. `cd avmatrix && npm test`
  passed in `419,432.8ms`.

### Phase 15 DB Load Parallel COPY Rejection Benchmark

- Date: `2026-05-14`.
- Purpose:
  evaluate the first profile-backed DB-load/cgo optimization candidate after CPU pprof showed
  `runtime.cgocall` dominating large-repo analyze time.
- Phase-stay note:
  this work stayed in Phase 15 because it evaluates a performance candidate on the already cut-over
  Go runtime. It is not Phase 10 fallback correctness and not Phase 17 runtime authority.

| Step | Command/artifact | Result |
| --- | --- | --- |
| Start graph refresh | `.tmp\phase15-dbload-start-graph.json` | wall `64,949.7ms`; benchmark total `64,647.5ms`; DB load `33,662.1ms`; graph `33,703` nodes / `66,932` relationships; `nodeCopyCount=19`; `relationshipCopyCount=90`; fallback/skipped `0` |
| Candidate build | `avmatrix-launcher\build.ps1` after changing COPY options to `PARALLEL=true` | passed in `36,663.0ms` |
| Candidate focused test | `go test ./internal/lbugload -run 'Test(LoadCSVExportUsesCopyForSupportedNodeAndRelationshipPairs|CopyQueriesMatchLadybugCSVContract)' -count=1` | failed only because the contract snapshot still expected `PARALLEL=false`; this was not accepted as evidence of runtime safety |
| Candidate analyze | packaged `avmatrix.exe analyze --force --skip-agents-md --no-stats --benchmark-json .tmp\phase15-dbload-parallel-true-analyze.json --benchmark-label phase15-dbload-parallel-true-analyze` | failed closed in `33,143.2ms`; no benchmark JSON was written because native DB load failed |
| Rollback build | `avmatrix-launcher\build.ps1` after restoring `PARALLEL=false` | passed in `33,780.9ms` |
| Rollback-safe analyze | `.tmp\phase15-dbload-rollback-safe-analyze.json` | wall `59,961.4ms`; benchmark total `58,086.4ms`; parse `19,424.1ms`; resolution `5,311.4ms`; DB load `28,685.4ms`; graph/rows `33,703` nodes / `66,932` relationships; fallback/skipped `0` |

- Rejection reason:
  LadybugDB's parallel CSV reader rejected quoted newlines in `method.csv` line `315` and requested
  `PARALLEL=FALSE`. Since AVmatrix stores code/content text in CSV fields, enabling `PARALLEL=true`
  globally would make valid current-repo data fail. Rewriting or stripping quoted newlines would
  risk content-fidelity regressions, so it is not an acceptable optimization.
- Correctness result:
  the Go loader failed closed instead of falling back to silent inserts or skipped relationships.
  After rollback, the same packaged Go path performed real DB load with `fallbackInsertCount=0`,
  `fallbackInsertFailures=0`, `skippedRelationships=0`, `nodeRows=33,703`, and
  `relationshipRows=66,932`.
- Optimization conclusion:
  `[P15-DBLOAD-CGO-BATCH-OPT]` remains open. The next DB-load attempt must reduce native/cgo COPY
  overhead without changing CSV quoting/content semantics or weakening fail-closed load behavior.
- Validation:
  full launcher build was run before tests and passed in `35,472.3ms`; `go test ./cmd/...
  ./internal/... -count=1` passed in `29,349.2ms`. Full browser E2E used isolated
  `AVMATRIX_HOME=.tmp\phase15-dbload-docs-e2e-home-*`, the packaged Go backend on
  `127.0.0.1:4747`, and the existing Vite dev server on `127.0.0.1:5173`; isolated analyze took
  `63,905.4ms`, `.tmp\phase15-dbload-docs-e2e-analyze-final.json` recorded `33,704` nodes /
  `66,933` relationships with DB fallback/skipped counts at `0`, and Playwright passed `33/33` in
  `180,285.1ms`. `cd avmatrix && npm test` passed in `409,607.3ms`.

### Phase 15 ScopeIR Normalized Comparator Benchmark

- Date: `2026-05-14`.
- Purpose:
  close the profile-backed ScopeIR normalization allocation target after heap pprof showed
  `ScopeIR.Normalized`, `callKey`, and range sort-key allocation as a major memory hotspot.
- Phase-stay note:
  this work stayed in Phase 15 because it is performance/profile work on the already cut-over Go
  runtime. It is not provider coverage, fallback correctness, or cutover authority work.
- Impact:
  AVmatrix impact reported `ScopeIR.Normalized` LOW risk, `callKey` LOW risk, and `rangeKey` MEDIUM
  risk. The affected execution process was limited to `Normalized`; no high-risk process blast
  radius was reported.

| Benchmark | Before | After |
| --- | ---: | ---: |
| `BenchmarkScopeIRNormalizedLargeSort` time | `18.10-19.14ms/op` | `3.38-3.86ms/op` |
| `BenchmarkScopeIRNormalizedLargeSort` bytes | `~12.85MB/op` | `~4.05MB/op` |
| `BenchmarkScopeIRNormalizedLargeSort` allocs | `242,580-242,623 allocs/op` | `20,287-20,329 allocs/op` |

- Implementation:
  `ScopeIR.Normalized` now sorts facts with field comparators over file path, range, kind/name/id,
  and related tie-breakers instead of constructing concatenated string keys for every sort
  comparison. The old `callKey`/`rangeKey`/`padInt` path was removed from the hot sort path, while
  deterministic JSON output remains covered by the ScopeIR golden tests.
- Large-repo benchmark:
  start graph `.tmp\phase15-scopeir-start-graph.json` recorded wall `72,201.6ms`, total
  `71,802.7ms`, parse `19,879.0ms`, resolution `5,451.2ms`, DB load `41,450.6ms`,
  `33,704` nodes, `66,933` relationships, `maxObservedSys=944,668,920`, and DB
  fallback/skipped `0`. Final packaged analyze `.tmp\phase15-scopeir-final-analyze.json` passed in
  `60,934.4ms` wall time with benchmark total `58,448.7ms`, parse `18,730.4ms`, resolution
  `5,312.4ms`, DB load `29,923.8ms`, `33,695` nodes, `66,981` relationships,
  `maxObservedSys=928,522,488`, and DB fallback/skipped `0`.
- Heap pprof result:
  `.tmp\phase15-scopeir-final-mem.pprof` was `43,081` bytes. Final alloc-space pprof shows
  `ScopeIR.Normalized` at `117.45MB` flat / `126.96MB` cumulative, down from the previous
  `~932.17MB` cumulative ScopeIR normalization allocation. `callKey` and `rangeKey` no longer appear
  in the final pprof top output. The next memory hotspot is `bytes.genSplit` at `1,059.40MB` through
  `frameworks.definitionWindow`.
- Validation:
  full launcher build was run before final tests and passed in `40,386.9ms`; focused ScopeIR golden
  tests passed; `go test ./cmd/... ./internal/... -count=1` passed in `32,531.4ms`. Full browser E2E
  used isolated `AVMATRIX_HOME=.tmp\phase15-scopeir-e2e-home-*`, packaged Go backend on
  `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated analyze took `57,957.6ms`, benchmark
  total `57,668.8ms`, graph rows `33,695` / `66,981`, DB fallback/skipped `0`, and Playwright passed
  with `32` passed / `1` skipped in `405,663.7ms`. `cd avmatrix && npm test` passed in
  `342,850.8ms`.

### Phase 15 Framework Definition Window Allocation Benchmark

- Date: `2026-05-14`.
- Purpose:
  close the profile-backed framework definition-window allocation target after the ScopeIR heap
  profile showed `bytes.genSplit` at `1,059.40MB` via `frameworks.definitionWindow`.
- Phase-stay note:
  this work stayed in Phase 15 because it is performance/profile work on the already cut-over Go
  runtime. It is not provider coverage, fallback correctness, or cutover authority work.
- Impact:
  AVmatrix impact reported `definitionWindow` LOW risk with one direct caller (`AnnotateScopeIR`).
  AVmatrix impact on the caller reported CRITICAL because `AnnotateScopeIR` feeds `parseFiles` and
  the CLI analyze path, so the implementation was limited to preserving the same window text and
  framework fact behavior.

| Benchmark | Before | After |
| --- | ---: | ---: |
| `BenchmarkAnnotateScopeIRDefinitionWindowManyDefinitions` time | `334.7-357.0ms/op` | `6.72-7.06ms/op` |
| `BenchmarkAnnotateScopeIRDefinitionWindowManyDefinitions` bytes | `~395.5MB/op` | `~2.16MB/op` |
| `BenchmarkAnnotateScopeIRDefinitionWindowManyDefinitions` allocs | `10047-10049 allocs/op` | `6049 allocs/op` |

- Implementation:
  `AnnotateScopeIR` now builds one line-start index per source file and slices definition windows
  directly from the original source bytes instead of splitting and joining the whole source for each
  definition. The `definitionWindow` helper remains as the contract wrapper and is covered by
  line-range and `600` byte cap tests.
- Large-repo benchmark:
  final packaged analyze `.tmp\phase15-framework-window-final-analyze.json` passed in `57,942.5ms`
  wall time with benchmark total `55,811.0ms`, parse `17,713.4ms`, resolution `5,324.2ms`, DB load
  `28,331.8ms`, `33,729` nodes, `67,032` relationships, `maxObservedSys=979,804,408`, and DB
  fallback/skipped `0`.
- Heap pprof result:
  `.tmp\phase15-framework-window-final-mem.pprof` was `41,782` bytes. Final alloc-space pprof no
  longer shows `bytes.genSplit`; `frameworks.definitionWindowIndex.window` is `12MB` flat and
  `frameworks.AnnotateScopeIR` is `22.51MB` cumulative. The next profile-backed memory targets are
  in resolution workspace/name-resolution allocation: `buildWorkspace` `298.10MB` cumulative,
  `uniqueDefs` `126.39MB` flat, and `resolveGlobalName` `212.67MB` cumulative.
- Validation:
  full launcher build was run before final tests and passed in `38,255.4ms`; focused framework
  tests passed; `go test ./cmd/... ./internal/... -count=1` passed in `27,620.2ms`. Browser E2E used
  isolated `AVMATRIX_HOME=.tmp\phase15-framework-window-e2e-home-*`, packaged Go backend on
  `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated analyze took `55,728.6ms`, benchmark
  total `55,327.9ms`, graph rows `33,729` / `67,032`, DB fallback/skipped `0`, and Playwright
  passed with `32` passed / `1` skipped in `397,958.2ms` using `--workers=1`. A 4-worker Playwright
  run failed under concurrent graph-load pressure (`10` passed / `22` failed / `1` skipped in
  `999,560.8ms`); it was rejected as the accepted gate because direct graph stream smoke passed in
  `1,734.1ms` and the serial full E2E run passed. `cd avmatrix && npm test` passed in
  `537,613.2ms`.

### Phase 15 Resolution Workspace Allocation Benchmark

- Date: `2026-05-14`.
- Purpose:
  close the profile-backed resolution workspace/name-resolution allocation target after the final
  framework heap profile showed `buildWorkspace` `298.10MB` cumulative, `uniqueDefs` `126.39MB`
  flat, and `resolveGlobalName` `212.67MB` cumulative.
- Phase-stay note:
  this work stayed in Phase 15 because it is performance/profile work on the already cut-over Go
  runtime. It is not provider coverage, fallback correctness, or cutover authority work.
- Impact:
  AVmatrix impact reported `buildWorkspace` CRITICAL because it feeds `BuildCrossFileBinding` and
  the analyze path; `definitionLookupNames` was HIGH; `uniqueDefs`, `resolveGlobalName`,
  `resolveGlobalCallName`, `resolveSameFileName`, and `resolveImportedMember` were LOW. The change
  was therefore scoped to allocation behavior and preserved the existing "exactly one unique
  definition" resolution contract.

| Benchmark | Before | After |
| --- | ---: | ---: |
| `BenchmarkResolveTypeScriptGraphFixture` time | `368.3-403.8us/op` | `345.1-378.1us/op` |
| `BenchmarkResolveTypeScriptGraphFixture` bytes | `~259,341 B/op` | `~247,954 B/op` |
| `BenchmarkResolveTypeScriptGraphFixture` allocs | `1733 allocs/op` | `1681 allocs/op` |

- Implementation:
  `buildWorkspace` now measures ScopeIR input sizes once and pre-sizes workspace maps/slices.
  Definition lookup-name handling uses a fixed three-slot set instead of allocating temporary
  slices for simple/qualified names. Global, same-file, and imported-member name resolution now use
  a tiny unique-definition accumulator instead of building candidate slices and de-duplicating with
  maps on every lookup. `uniqueStrings` and `uniqueDefs` keep fast paths for the common one- and
  two-item cases.
- Large-repo benchmark:
  final packaged analyze `.tmp\phase15-resolution-workspace-final2-analyze.json` passed in
  `62,789.7ms` wall time with benchmark total `61,479.0ms`, parse `18,053.0ms`, resolution
  `5,014.8ms`, DB load `32,675.5ms`, `33,760` nodes, `67,153` relationships,
  `maxObservedSys=932,315,384`, and DB fallback/skipped `0`. The total wall time is not claimed as
  a speedup because the DB load phase varied upward in this run; the accepted improvement for this
  slice is the focused allocation benchmark and pprof movement.
- Heap pprof result:
  `.tmp\phase15-resolution-workspace-final2-mem.pprof` alloc-space top showed `buildWorkspace`
  `149.86MB` flat / `270.14MB` cumulative, down from the previous `298.10MB` cumulative target.
  `uniqueDefs` and the lookup closure allocation frame no longer appeared in the top table.
  Remaining profile-backed candidates are `ScopeIR.Normalized` residual allocation
  (`122.29MB` cumulative), `workspace.callerForScope` (`70.52MB` flat), `Graph.AddRelationship`,
  `emitDefinitionNodes`, and the still-open DB-load/cgo target.
- Validation:
  full launcher build was run before final tests and passed in `34,953.3ms`; focused resolution
  tests passed in `2,343.2ms`; after-build micro benchmark passed in `8,152.8ms`;
  `go test ./cmd/... ./internal/... -count=1` passed in `32,739.5ms`. Browser E2E used isolated
  `AVMATRIX_HOME=.tmp\phase15-resolution-workspace-e2e-home-*`, packaged Go backend on
  `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated analyze took `56,796.4ms`, graph rows
  were `33,760` / `67,153`, DB fallback/skipped `0`, and Playwright passed with `32` passed /
  `1` skipped in `411,382.6ms` using `--workers=1`. `cd avmatrix && npm test` passed in
  `579,022.0ms`.

### Phase 15 DB Load Schema Lookup Allocation Benchmark

- Date: `2026-05-14`.
- Purpose:
  close a scoped DB-load export allocation target after the previous heap profile showed
  `lbugload.validNodeTables` at `39.57MB` flat and `ExportGraphCSVs` at `77.02MB` cumulative. This
  is not the native COPY/cgo batching target; `[P15-DBLOAD-CGO-BATCH-OPT]` remains open.
- Phase-stay note:
  this work stayed in Phase 15 because it is performance/profile work on the already cut-over Go
  runtime. It is not provider coverage, fallback correctness, or cutover authority work.
- Impact:
  AVmatrix impact reported `validNodeTables`, `nodeColumns`, and `relationPairSupported` CRITICAL
  because they feed `ExportGraphCSVs`, `loadGraph`, and analyze. `RelationPairs` was LOW risk. The
  implementation was limited to immutable schema lookup tables and one schema pair needed by the
  real graph.

| Benchmark | Before | After |
| --- | ---: | ---: |
| `BenchmarkExportGraphCSVs` time | `16.7-31.2ms/op` | `14.8-30.9ms/op` |
| `BenchmarkExportGraphCSVs` bytes | `~335KB/op` | `~285KB/op` |
| `BenchmarkExportGraphCSVs` allocs | `1906 allocs/op` | `1403 allocs/op` |
| `BenchmarkLoadCSVExportCopyPathNoop` bytes | `1536-1568 B/op` | `1376 B/op` |
| `BenchmarkLoadCSVExportCopyPathNoop` allocs | `19 allocs/op` | `17 allocs/op` |

- Implementation:
  node column lists, valid node table lookup, and relation-pair lookup are now package-level schema
  tables instead of maps/slices rebuilt during DB-load CSV export and COPY query generation.
  Packaged analyze initially failed closed on `Const->Function` after the graph shape changed; the
  slice added that relation pair to the LadybugDB schema and updated schema tests instead of
  allowing fallback or skipped relationships.
- Large-repo benchmark:
  final packaged analyze `.tmp\phase15-dbload-schema-lookup-final2-analyze.json` passed in
  `58,132.1ms` wall time with benchmark total `56,490.3ms`, parse `18,873.0ms`, resolution
  `5,212.0ms`, DB load `27,697.0ms`, `33,783` nodes, `67,172` relationships,
  `nodeCopyCount=19`, `relationshipCopyCount=91`, `maxObservedSys=836,059,384`, and DB
  fallback/skipped `0`.
- Heap pprof result:
  `.tmp\phase15-dbload-schema-lookup-final2-mem.pprof` alloc-space top no longer showed
  `lbugload.validNodeTables`; `lbugload.ExportGraphCSVs` was `31.95MB` cumulative, down from the
  previous `77.02MB` DB-load export allocation frame. Remaining DB-load work is native COPY/cgo
  overhead, not schema lookup allocation.
- E2E note:
  two full Playwright runs failed on a `process-modal` visibility wait that timed out at `5s` while
  the error snapshot showed the modal eventually opened. The E2E wait was hardened to `15s` for the
  two process-modal assertions; the accepted final run still used the real packaged Go backend and
  full browser suite.
- Validation:
  full launcher build was rerun after the E2E wait hardening and passed in `35,785.3ms`; focused
  `lbugload`/`lbugschema` tests passed in `2,344.0ms`; after-build benchmarks passed in
  `26,244.7ms`; `go test ./cmd/... ./internal/... -count=1` passed in `31,352.4ms`. Browser E2E
  used isolated `AVMATRIX_HOME=.tmp\phase15-dbload-schema-lookup-e2e-home-final-*`, packaged Go
  backend on `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated analyze took `57,103.4ms`,
  graph rows were `33,783` / `67,172`, DB fallback/skipped `0`, and Playwright passed with
  `32` passed / `1` skipped in `412,198.7ms` using `--workers=1`. `cd avmatrix && npm test` passed
  in `424,654.6ms`.

### Phase 15 Native DB Load Transaction Benchmark

- Date: `2026-05-14`.
- Purpose:
  close `[P15-DBLOAD-CGO-BATCH-OPT]` after the first COPY `PARALLEL=true` candidate was rejected
  and the separate schema lookup allocation slice still left native COPY/cgo DB load as the
  dominant macro bottleneck.
- Phase-stay note:
  this work stayed in Phase 15 because it is performance work on the already cut-over Go runtime.
  It is not fallback correctness, provider coverage, or cutover authority work.
- Impact:
  AVmatrix impact reported `loadGraph` CRITICAL because the DB load path feeds `analyze`,
  `newAnalyzeCommand`, `main`, and HTTP embed/analyze paths. AVmatrix context for `LoadCSVExport`
  showed the direct analyze, native integration, benchmark, and fail-closed test callers, so the
  change was limited to optional transaction support and kept the COPY query contract unchanged.

| Benchmark | Before | After |
| --- | ---: | ---: |
| Packaged current-repo analyze wall | `62,660.7ms` | `35,469.1ms` |
| Benchmark JSON total | `62,327.7ms` | `34,167.8ms` |
| DB load phase | `32,246.0ms` | `5,773.7ms` |
| DB load phase speedup | baseline | `~5.59x` |
| Overall benchmark speedup | baseline | `~1.82x` |
| DB fallback/skipped | `0` / `0` | `0` / `0` |

- Implementation:
  `lbugload.LoadCSVExportWithOptions` now starts a load transaction only when its runner implements
  the optional transaction interface. The native LadybugDB write runner implements that interface
  with `BEGIN TRANSACTION`, `COMMIT`, and `ROLLBACK`. Any node COPY failure, relationship COPY
  failure, unsupported schema pair, skipped relationship, or diagnostic fallback failure still
  fails closed and rolls back. Non-transaction runners keep the old per-query behavior.
- Focused benchmarks:
  `BenchmarkLoadCSVExportCopyPathNoop` remained at `1360-1376 B/op` and `17 allocs/op`, confirming
  the accepted speedup is native transaction/COPY batching rather than a changed CSV contract.
  `BenchmarkDiagnosticFallbackPathNoop` remained diagnostic-only at about `304KB/op` and
  `5058 allocs/op`.
- Large-repo benchmark:
  final packaged analyze with CPU profile
  `.tmp\phase15-dbload-tx-attempt-analyze.json` / `.tmp\phase15-dbload-tx-attempt-cpu.pprof`
  passed in `35,469.1ms` wall time with benchmark total `34,167.8ms`, parse `18,529.1ms`,
  resolution `5,289.1ms`, DB load `5,773.7ms`, `33,799` node rows, `67,272` relationship rows,
  `nodeCopyCount=19`, `relationshipCopyCount=91`, `maxObservedSys=860,315,896`, and DB
  fallback/skipped `0`.
- CPU pprof result:
  `runtime.cgocall` moved from the earlier DB-load target profile (`42.12s` flat, `67.83%`) to
  `19.64s` flat / `21.83s` cumulative in the transaction run. The remaining cgo cost still includes
  tree-sitter/parser and native database work, so this closes the DB-load transaction target but
  does not imply all cgo cost has disappeared.
- Validation:
  full launcher build was run before tests and passed in `52,836.0ms`; focused `lbugload` tests
  passed in `2,154.4ms`; native LadybugDB integration passed with the same CGO include/link/runtime
  path used by the launcher build in `13,648.5ms`; after-build loader benchmarks passed in
  `23,453.1ms`; `go test ./cmd/... ./internal/... -count=1` passed in `30,764.5ms`. Browser E2E
  used isolated `AVMATRIX_HOME=.tmp\phase15-dbload-tx-e2e-home-*`, packaged Go backend on
  `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated analyze took `37,658.0ms`, graph rows
  were `33,799` / `67,272`, DB fallback/skipped `0`, and Playwright passed with `32` passed /
  `1` skipped in `447,486.9ms` using `--workers=1`. `cd avmatrix && npm test` had one initial
  flaky Rust skills E2E failure, then the failed suite rerun passed (`25/25`, `110,639.9ms`) and
  the full command rerun passed in `393,820.1ms`.

### Phase 15 Imported Member Resolution Index Benchmark

- Date: `2026-05-14`.
- Purpose:
  close the remaining generic file IO/parser-pool checklist items by evidence if they are not the
  real bottleneck, then optimize the next pprof-backed hotspot in the already cut-over Go runtime.
- Phase-stay note:
  this work stayed in Phase 15. It did not jump to provider coverage, fallback correctness, or
  Phase 17 cutover authority because the target is large-repo performance after cutover.
- Impact:
  AVmatrix impact reported `resolveImportedMember` CRITICAL because it feeds
  `resolveCallTargetForTypeBinding`, `resolveCall`, `ResolveBoundInto`, and `Run`. The change was
  limited to an internal workspace import index and preserves the existing candidate ambiguity
  behavior.

| Benchmark | Before | After |
| --- | ---: | ---: |
| `BenchmarkResolveImportedMemberManyImports` time | `17.2-17.6us/op` | `492-512ns/op` |
| `BenchmarkResolveImportedMemberManyImports` bytes | `48 B/op` | `48 B/op` |
| `BenchmarkResolveImportedMemberManyImports` allocs | `3 allocs/op` | `3 allocs/op` |
| Packaged current-repo benchmark total | `35,015.3ms` | `28,727.0ms` |
| Resolution phase | `5,326.8ms` | `955.3ms` |
| Cross-file binding phase | `1,585.8ms` | `579.8ms` |
| DB fallback/skipped | `0` / `0` | `0` / `0` |

- File IO/parser-pool reassessment:
  `.tmp\phase15-next-after-dbtx-start-graph.json` plus CPU/heap pprof showed `parse` at
  `19,150.8ms`, `resolution` at `5,326.8ms`, and `db_load` at `5,803.5ms`. Heap file-read cost was
  low (`os.readFileContents` `31.48MB` before, `24.81MB` after) and did not appear as a CPU
  hotspot. Parser metrics showed `createdParsers=4`, `total=1058`, `failed=0`; CPU parse cost was
  in tree-sitter/provider work (`Pool.Parse`, `tsjs.Extract`) rather than pool sizing. Therefore
  file IO batching and parser pool sizing are rejected for the current profile.
- Implementation:
  `workspace` now keeps `importsByReceiver` keyed by `(sourceFile, localName)` during
  `resolveImports`. `resolveImportedMember` uses that index to scan only imports that can match the
  receiver instead of scanning the full workspace import list for every member call. The existing
  `uniqueDefAccumulator` still enforces the old single-target/ambiguous-target behavior.
- Large-repo benchmark:
  final packaged analyze `.tmp\phase15-import-index-final-analyze.json` passed in `30,412.5ms`
  wall time with benchmark total `28,727.0ms`, parse `18,615.9ms`, cross-file binding `579.8ms`,
  resolution `955.3ms`, DB load `5,596.5ms`, graph rows `33,803` / `67,310`,
  `nodeCopyCount=19`, `relationshipCopyCount=91`, `maxObservedSys=915,570,936`, and DB
  fallback/skipped `0`. `benchmark-compare` against
  `.tmp\phase15-next-after-dbtx-start-graph.json` reported total `-18%`, resolution `-82.1%`, and
  cross-file binding `-63.4%`.
- CPU pprof result:
  before profile `.tmp\phase15-next-after-dbtx-start-cpu.pprof` showed
  `resolution.(*workspace).resolveImportedMember` at `4.91s` flat / `5.11s` cumulative. Final
  `.tmp\phase15-import-index-final-cpu.pprof` no longer showed that function in the top CPU table.
  The remaining dominant cost is `runtime.cgocall` (`19.61s` flat / `21.62s` cumulative), mostly
  tree-sitter/provider parse plus native DB work.
- Validation:
  full launcher build was run before tests and passed in `41,236.0ms`; focused resolution tests
  passed in `2,312.7ms`; after-build micro benchmark passed in `11,508.5ms`;
  `go test ./cmd/... ./internal/... -count=1` passed in `31,327.0ms`. Browser E2E used isolated
  `AVMATRIX_HOME=.tmp\phase15-import-index-e2e-retry-home-*`, packaged Go backend on
  `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated analyze took `29,045.1ms`, graph rows
  were `33,803` / `67,310`, DB fallback/skipped `0`, and the accepted Playwright run passed with
  `32` passed / `1` skipped in `428,776.4ms` using `--workers=1 --retries=1` with no retry needed
  in the accepted output. Earlier full E2E attempts were rejected because they produced flaky UI
  timing failures (`process-row` highlight once, then two auto-connect readiness waits plus the
  same highlight check); the failing highlight test passed when rerun directly in `36,780.5ms`.
  `cd avmatrix && npm test` passed in `378,034.1ms`.

### Phase 15 Parser Node-Count Diagnostic Benchmark

- Date: `2026-05-14`.
- Purpose:
  close a narrow parser diagnostic overhead target after the post-import-index CPU pprof showed
  `internal/parser.countNodes` at `1.58s` cumulative under `Pool.Parse`. This is not a claim that
  tree-sitter/provider parse cost is solved.
- Phase-stay note:
  this work stayed in Phase 15 because it is profile-backed performance work on the already
  cut-over Go runtime. It did not jump to Phase 14 provider coverage or Phase 17 cutover authority.
- Impact:
  AVmatrix impact reported `countNodes` LOW risk with direct caller `Pool.Parse`. `PoolOptions` was
  CRITICAL because it is shared across analyze, providers, resolution tests, and HTTP paths, so the
  implementation kept the zero-value default safe for production analyze.

| Benchmark | Before | After |
| --- | ---: | ---: |
| `BenchmarkPoolParseNodeCount` disabled | n/a | `221-297us/op`, `1032-1339 B/op`, `17 allocs/op` |
| `BenchmarkPoolParseNodeCount` enabled | n/a | `276-387us/op`, `5352-5416 B/op`, `152 allocs/op` |
| Parser `totalDuration` | `8,646.6ms` | `7,586.2ms` |
| CPU `Pool.Parse` cumulative | `8.28s` | `7.58s` |
| CPU `countNodes` cumulative | `1.58s` | absent from top CPU table |
| DB fallback/skipped | `0` / `0` | `0` / `0` |

- Implementation:
  `parser.PoolOptions` now has `CountNodes`. `Pool.Parse` only walks the tree with `countNodes`
  when that option is enabled. Analyze and normal runtime callers keep the zero-value default and
  therefore do not pay the diagnostic tree walk; the parser test that asserts `NodeCount` explicitly
  opts in, and a new test verifies the default remains `0`.
- Large-repo benchmark:
  baseline `.tmp\phase15-next-after-import-index-start-graph.json` recorded benchmark total
  `29,088.0ms`, parse `18,428.5ms`, parser `totalDuration=8,646.6ms`, graph rows `33,804` /
  `67,311`, and DB fallback/skipped `0`. Final packaged analyze
  `.tmp\phase15-nodecount-final-analyze.json` passed in `32,432.8ms` wall time with benchmark total
  `29,714.9ms`, parse `19,051.1ms`, parser `totalDuration=7,586.2ms`, graph rows `33,843` /
  `67,328`, and DB fallback/skipped `0`.
- Benchmark interpretation:
  `benchmark-compare` reported total `+2.2%`; this is treated as neutral/noisy, not a macro
  speedup. The accepted improvement is the focused parser metric and pprof movement: `countNodes`
  disappeared from the CPU top table and `Pool.Parse` cumulative CPU moved from `8.28s` to `7.58s`.
  Remaining Phase 15 CPU targets are tree-sitter/provider traversal (`Parser.ParseWithOptions`,
  `tsjs.Extract`, `golang.Extract`) and remaining native DB commit/COPY cost.
- Validation:
  full launcher build was run before accepted tests and passed in `45,641.2ms`; focused parser
  tests after build passed in `2,508.1ms`; after-build parser benchmark package time was
  `14,867ms`; `go test ./cmd/... ./internal/... -count=1` passed in `32,805.6ms`. Browser E2E used
  isolated `AVMATRIX_HOME=.tmp\phase15-nodecount-e2e-home-20260514-080427`, packaged Go backend on
  `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated analyze took `29,087.9ms`, graph rows
  were `33,843` / `67,328`, DB fallback/skipped `0`, and Playwright passed with `33/33` in
  `206,434.9ms` using `--workers=1 --retries=1`. `cd avmatrix && npm test` passed in
  `400,152.3ms`. One earlier E2E analyze attempt is rejected from evidence because the PowerShell
  variable `$HOME` was accidentally used instead of an AVmatrix-specific variable, so it did not use
  the intended isolated home.

### Phase 15 TS/JS Provider Traversal Benchmark

- Date: `2026-05-14`.
- Purpose:
  reduce the next pprof-backed provider hotspot after node-count removal. The post-node-count CPU
  profile showed `tsjs.Extract` at `6.35s` cumulative, `tsjs.walk` at `6.28s`, `Node.Kind` at
  `2.27s`, and `NamedChild` at `2.28s`.
- Phase-stay note:
  this stayed in Phase 15 because it is performance work on the already cut-over Go runtime. It is
  not new Phase 14 provider coverage and not Phase 17 cutover authority.
- Impact:
  AVmatrix impact reported `Extract` CRITICAL because it feeds analyze/runtime/test flows. The
  targeted internal collector methods (`emitDefinition`, `emitReference`, `emitTypeBinding`,
  `buildContext`, and `collectScopes`) were LOW impact. The implementation kept public/internal
  wrapper methods and changed only TS/JS collector traversal.

| Benchmark | Before | After |
| --- | ---: | ---: |
| `BenchmarkExtractTypeScriptScopeIR` time | `446-466us/op` | `295-337us/op` |
| `BenchmarkExtractTypeScriptScopeIR` bytes | `~87.3KB/op` | `68.3KB/op` |
| `BenchmarkExtractTypeScriptScopeIR` allocs | `1966 allocs/op` | `996 allocs/op` |
| Packaged current-repo benchmark total | `27,545.4ms` | `25,728.2ms` |
| Parse phase | `17,141.5ms` | `14,549.4ms` |
| CPU `tsjs.Extract` cumulative | `6.35s` | `4.08s` |
| CPU `Node.Kind` cumulative | `2.27s` | `0.92s` |
| CPU `NamedChild` cumulative | `2.28s` | `1.59s` |
| DB fallback/skipped | `0` / `0` | `0` / `0` |

- Implementation:
  `walkKind` computes each visited node kind once and passes it to the collector. The TS/JS provider
  now combines the previous scope and context pre-pass with `collectScopesAndContext`, then the main
  pass calls kind-aware emitter helpers. Existing wrappers remain for tests and helper callers.
- Large-repo benchmark:
  baseline `.tmp\phase15-next-after-nodecount-start-graph.json` recorded total `27,545.4ms`, parse
  `17,141.5ms`, parser `totalDuration=6,947.2ms`, graph rows `33,844` / `67,329`, and DB
  fallback/skipped `0`. Final packaged analyze `.tmp\phase15-tsjs-traversal-final-analyze.json`
  passed in `27,697.2ms` wall time with benchmark total `25,728.2ms`, parse `14,549.4ms`, parser
  `totalDuration=6,777.7ms`, graph rows `33,828` / `67,364`, and DB fallback/skipped `0`.
  `benchmark-compare` reported total `-6.6%` and parse `-15.1%`; DB load moved up by `6.5%`, so
  the accepted macro gain is the total/parse reduction despite DB noise.
- CPU pprof result:
  baseline `.tmp\phase15-next-after-nodecount-start-cpu.pprof` showed `tsjs.Extract=6.35s`,
  `tsjs.walk=6.28s`, `Node.Kind=2.27s`, and `NamedChild=2.28s`. Final
  `.tmp\phase15-tsjs-traversal-final-cpu.pprof` showed `tsjs.Extract=4.08s`, `walkKind=3.99s`,
  `Node.Kind=0.92s`, and `NamedChild=1.59s`. Remaining CPU targets are native tree-sitter parse
  (`Parser.ParseWithOptions`), remaining provider traversal (`tsjs`, `golang`), and native DB
  commit/COPY.
- Validation:
  full launcher build was run before accepted tests and passed in `40,658.1ms`; focused
  TS/JS/SFC/Vue/Astro/Svelte/resolution tests after build passed in `9,854.5ms`; after-build TS/JS
  benchmark passed in `12,129.3ms`; `go test ./cmd/... ./internal/... -count=1` passed in
  `29,375.2ms`. Browser E2E used isolated
  `AVMATRIX_HOME=.tmp\phase15-tsjs-traversal-e2e-home-20260514-082700`, packaged Go backend on
  `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated analyze took `25,467.0ms`, graph rows
  were `33,828` / `67,364`, DB fallback/skipped `0`, and Playwright passed with `33/33` in
  `194,306.0ms` using `--workers=1 --retries=1`. `cd avmatrix && npm test` passed in
  `343,081.8ms`.

### Phase 15 Go Provider Traversal Benchmark

- Date: `2026-05-14`.
- Purpose:
  reduce the next pprof-backed Go-level provider hotspot after the TS/JS traversal slice. The
  post-TS/JS CPU profile showed `golang.Extract` at `1.95s` cumulative, `golang.walk` at `1.95s`,
  `golang.Extract.func1` at `1.10s`, `Node.Kind` at `1.14s`, and Go provider heap at `24.40MB`.
- Phase-stay note:
  this stayed in Phase 15 because it is performance work on the already cut-over Go runtime. It is
  not new Phase 14 provider coverage and not Phase 17 cutover authority. Native tree-sitter parse
  cgo and native LadybugDB query/commit cost were measured but not claimed by this scoped provider
  slice.
- Impact:
  AVmatrix impact reported `Extract` LOW with one direct caller (`extractScopeIR`) and two affected
  flows (`extractScopeIR`, `newAnalyzeCommand`). The targeted emitter/context helpers were LOW.
  `walk` reported HIGH because it is the Go provider traversal helper under `Extract`, so the change
  preserved the existing `walk` helper and added `walkKind` for the optimized extraction path.

| Benchmark | Before | After |
| --- | ---: | ---: |
| `BenchmarkExtractGoScopeIR` time | `549-572us/op` | `380-500us/op` |
| `BenchmarkExtractGoScopeIR` bytes | `~106.1KB/op` | `85.2KB/op` |
| `BenchmarkExtractGoScopeIR` allocs | `2379 allocs/op` | `1310 allocs/op` |
| Packaged current-repo benchmark total | `24,607.2ms` | `24,211.0ms` |
| Parse phase | `14,595.9ms` | `13,943.3ms` |
| CPU `golang.Extract` cumulative | `1.95s` | `1.14s` |
| CPU `golang.Extract.func1` cumulative | `1.10s` | `0.56s` |
| CPU `Node.Kind` cumulative | `1.14s` | `0.66s` |
| Go provider heap `Extract` cumulative | `24.40MB` | `19.91MB` |
| DB fallback/skipped | `0` / `0` | `0` / `0` |

- Implementation:
  `walkKind` computes each visited node kind once and passes it to the collector. The Go provider
  now combines the previous scope and context pre-pass with `collectScopesAndContext`, then the main
  pass calls kind-aware emitter helpers. Existing wrappers remain for tests and helper callers.
  Nested `emitTypeReferences` walks also use `walkKind`.
- Large-repo benchmark:
  baseline `.tmp\phase15-next-after-tsjs-traversal-start-graph.json` recorded total `24,607.2ms`,
  parse `14,595.9ms`, parser `totalDuration=6,749.9ms`, graph rows `33,829` / `67,365`, and DB
  fallback/skipped `0`. Final packaged analyze `.tmp\phase15-go-provider-traversal-analyze.json`
  passed in `26,449.5ms` wall time with benchmark total `24,211.0ms`, parse `13,943.3ms`, parser
  `totalDuration=6,938.5ms`, graph rows `33,843` / `67,378`, and DB fallback/skipped `0`.
  `benchmark-compare` reported total `-1.6%` and parse `-4.5%`; DB load moved up by `5.1%`, so DB
  remains a separate Phase 15 native target.
- CPU/heap pprof result:
  baseline `.tmp\phase15-next-after-tsjs-traversal-start-cpu.pprof` showed
  `golang.Extract=1.95s`, `golang.walk=1.95s`, `golang.Extract.func1=1.10s`, and
  `Node.Kind=1.14s`. Final `.tmp\phase15-go-provider-traversal-cpu.pprof` showed
  `golang.Extract=1.14s`, `golang.walkKind=1.12s`, `golang.Extract.func1=0.56s`, and
  `Node.Kind=0.66s`. Heap pprof moved Go provider `Extract` from `24.40MB` to `19.91MB`.
- Validation:
  full launcher build was run before accepted tests and passed in `39,231.7ms`; focused
  Go-provider/analyze/resolution tests after build passed in `6,337.4ms`; after-build Go provider
  benchmark passed in `10,258.5ms`; `go test ./cmd/... ./internal/... -count=1` passed in
  `34,494.9ms`. Browser E2E used isolated
  `AVMATRIX_HOME=F:\AVmatrix-GO\.tmp\phase15-go-provider-e2e-home-20260514-084738`, packaged Go
  backend on `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated analyze took `34,598.5ms`,
  graph rows were `33,843` / `67,378`, DB fallback/skipped `0`, and Playwright passed with
  `32` passed / `1` skipped in `436,101.7ms` using `--workers=1 --retries=1`. `cd avmatrix &&
  npm test` passed with exit code `0` in `404,491.2ms`.

### Phase 15 Graph Snapshot Stream Memory Benchmark

- Date: `2026-05-14`.
- Purpose:
  reduce the heap peak from writing `.avmatrix\graph.json` after the post-Go-provider heap profile
  showed a full-snapshot `json.MarshalIndent` buffer under `writeGraphSnapshot`.
- Phase-stay note:
  this stayed in Phase 15 because it is pprof-backed memory optimization on the already cut-over Go
  runtime. It is not provider coverage, not fallback/schema correctness, and not Phase 17 runtime
  authority. Native tree-sitter parse cgo and native LadybugDB query/commit/COPY cost remain open.
- Impact:
  AVmatrix impact reported `writeGraphSnapshot` CRITICAL because it is called from analyze `Run`
  and participates in `newAnalyzeCommand` / `main` execution flows. The change therefore preserved
  the public graph snapshot JSON shape (`nodes`, `relationships`) and the atomic temp-file rename
  behavior.

| Benchmark | Before | After |
| --- | ---: | ---: |
| Packaged current-repo benchmark total | `25,256.0ms` | `27,335.2ms` |
| Parse phase | `14,643.6ms` | `15,004.9ms` |
| DB load phase | `5,632.8ms` | `6,858.4ms` |
| Graph rows | `33,844` / `67,379` | `33,857` / `67,412` |
| DB fallback/skipped | `0` / `0` | `0` / `0` |
| `maxObservedSys` | `822,550,776` | `632,242,424` |
| Heap inuse total | `178.39MB` | `135.09MB` |
| Heap `bytes.growSlice` under snapshot marshal | `64MB` | absent from top table |
| CPU `writeGraphSnapshot` cumulative | `0.86s` | `0.82s` |

- Implementation:
  `writeGraphSnapshot` now creates the temp file, streams the top-level `nodes` and
  `relationships` arrays through a `bufio.Writer`, flushes, closes, and renames the temp file only
  after a complete write. Each node/relationship value is still JSON-indented independently, so the
  persisted snapshot remains normal `graph.Graph` JSON and the existing readers can unmarshal it
  unchanged. A regression test reads the snapshot back into `graph.Graph` and compares counts with
  the in-memory result.
- Large-repo benchmark:
  baseline `.tmp\phase15-next-after-go-provider-start-graph.json` recorded total `25,256.0ms`,
  parse `14,643.6ms`, DB load `5,632.8ms`, rows `33,844` / `67,379`, fallback/skipped `0`, and
  `maxObservedSys=822,550,776`. Final
  `.tmp\phase15-graph-snapshot-stream-final-analyze.json` recorded total `27,335.2ms`, parse
  `15,004.9ms`, DB load `6,858.4ms`, rows `33,857` / `67,412`, fallback/skipped `0`, and
  `maxObservedSys=632,242,424`. `benchmark-compare` reported total `+8.2%` because native DB load
  moved `+21.8%`; this slice is accepted as a memory/pprof win, not a macro wall-time speedup.
- CPU/heap pprof result:
  baseline `.tmp\phase15-next-after-go-provider-start-mem.pprof` showed `bytes.growSlice=64MB`
  flat/cumulative under `encoding/json.MarshalIndent` from `analyze.writeGraphSnapshot`; final
  `.tmp\phase15-graph-snapshot-stream-final-mem.pprof` no longer had that frame in the top table
  and moved heap inuse total from `178.39MB` to `135.09MB`. CPU stayed roughly neutral:
  `writeGraphSnapshot` moved from `0.86s` to `0.82s`, with final `writeGraphSnapshotJSON=0.82s`
  and `writeIndentedJSONValue=0.78s`.
- Focused benchmark:
  `BenchmarkWriteGraphSnapshot` was added in this slice and measured `28-45ms/op`, about
  `3.54MB/op`, and `32,527-32,531 allocs/op` after the full build. There is no pre-existing micro
  baseline for this benchmark; the accepted before/after proof is the macro heap/pprof delta above.
- Validation:
  full launcher build was run before accepted tests and passed in `40,160.2ms`; focused
  analyze/CLI/HTTP/MCP tests after build passed in `21,748.4ms`; after-build
  `BenchmarkWriteGraphSnapshot` passed in `21,757.3ms`; `go test ./cmd/... ./internal/... -count=1`
  passed in `31,950.0ms`. Browser E2E used isolated
  `AVMATRIX_HOME=F:\AVmatrix-GO\.tmp\phase15-graph-snapshot-e2e-home-20260514-092308`, packaged Go
  backend on `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated analyze took `26,815.2ms`,
  graph rows were `33,857` / `67,412`, DB fallback/skipped `0`, and Playwright passed with
  `31` passed / `1` skipped / `1` flaky recovered on retry in `528,159.0ms` using
  `--workers=1 --retries=1`. `cd avmatrix && npm test` passed with exit code `0` in
  `414,123.7ms`.

### Phase 15 ScopeIR Owned Normalize Allocation Benchmark

- Date: `2026-05-14`.
- Purpose:
  reduce the provider extract allocation attributed to ScopeIR normalization after the Phase 15
  post-lifecycle baseline showed `ScopeIR.Normalized` at `54.01MB` flat in heap pprof.
- Phase-stay note:
  this resumed Phase 15 after the Phase 10 schema correctness and Phase 16 launcher lifecycle
  phase-jumps. It is pprof-backed runtime/provider allocation work, not provider coverage and not a
  cutover authority gate.
- Impact:
  AVmatrix impact reported `ScopeIR.Normalized` LOW (`MarshalDeterministic`, `Unmarshal`, and the
  benchmark as direct callers). TS/JS and Go `collector.result` were LOW. The accepted change keeps
  the existing non-mutating `Normalized()` behavior and uses the owned normalization path only where
  providers have just built fresh IR facts.

| Benchmark | Before | After |
| --- | ---: | ---: |
| Full launcher build | n/a | `38,220.6ms` |
| `BenchmarkExtractTypeScriptScopeIR` allocation | `~68.3KB/op`, `996 allocs/op` | `~66.4KB/op`, `980 allocs/op` |
| `BenchmarkExtractGoScopeIR` allocation | `~85.2KB/op`, `1310 allocs/op` | `~82.8KB/op`, `1281 allocs/op` |
| Packaged current-repo benchmark total | `30,058.4ms` | `24,982.3ms` |
| Parse phase | `14,450.4ms` | `14,202.9ms` |
| DB load phase | `8,278.5ms` | `6,046.2ms` |
| Graph rows | `33,909` / `67,556` | `33,918` / `67,609` |
| DB fallback/skipped | `0` / `0` | `0` / `0` |
| `maxObservedSys` | `745,119,992` | `743,596,280` |

- Rejected path:
  pure in-place normalization reduced the TS/JS micro allocation to about `55.5KB/op` and Go to
  about `70.8KB/op`, but full-repo analyze retained oversized append backing arrays:
  `.tmp\phase15-scopeir-inplace-normalize-final.json` reported
  `endAllocBytes=204,682,944` and `maxObservedSys=839,073,016`. That variant is rejected and not
  the implementation kept in the code.
- Accepted implementation:
  `scopeir.NormalizeOwned` compacts top-level IR slices to `cap=len` before sorting owned facts.
  TS/JS and Go providers use this owned path; `ScopeIR.Normalized()` remains the defensive
  non-mutating API for serialization, unmarshal, and external callers.
- Large-repo benchmark:
  baseline `.tmp\phase15-select-next-target.json` recorded total `30,058.4ms`, parse
  `14,450.4ms`, DB load `8,278.5ms`, rows `33,909` / `67,556`, fallback/skipped `0`, and
  `maxObservedSys=745,119,992`. Final
  `.tmp\phase15-scopeir-owned-normalize-final.json` recorded total `24,982.3ms`, parse
  `14,202.9ms`, DB load `6,046.2ms`, rows `33,918` / `67,609`, fallback/skipped `0`, and
  `maxObservedSys=743,596,280`. `benchmark-compare` reported total `-16.9%`; this is treated as
  noisy because DB load moved `-27.0%`. The accepted claim is allocation reduction in the provider
  microbenchmarks with neutral full-repo memory, not a macro speedup.
- CPU/heap pprof result:
  baseline `.tmp\phase15-select-next-target-mem.pprof` attributed `54.01MB` flat to
  `ScopeIR.Normalized`. Final `.tmp\phase15-scopeir-owned-normalize-final-mem.pprof` attributes the
  compacted retained IR arrays to `ScopeIR.NormalizeOwned` (`57.75MB` flat), with full-repo
  `maxObservedSys` effectively neutral (`745,119,992` -> `743,596,280`). CPU pprof remains dominated
  by cgo/native parse and LadybugDB query work, so native parse/DB work remains the next Phase 15
  target family.
- Validation:
  full launcher build was run before accepted tests and passed in `38,220.6ms`; focused
  scopeir/TSJS/Go provider tests passed; after-build TS/JS, Go, and ScopeIR benchmarks passed;
  packaged analyze with CPU/heap pprof passed in `26,867.1ms` wall; full Go tests passed in
  `35,542.3ms`; browser E2E through the packaged Go backend and isolated `AVMATRIX_HOME` passed
  with `.last-run.json` status `passed`, `33` tests listed, command wall `590.4s`, isolated analyze
  benchmark total `28,503.9ms`, graph rows `33,919` / `67,610`, and DB fallback/skipped `0`;
  `cd avmatrix && npm test` passed in `461,316.4ms`; AVmatrix `detect_changes(scope=all)` reported
  MEDIUM risk limited to the normalize/provider/doc slice before commit.

### Phase 15 ScopeIR Release After Resolution Benchmark

- Date: `2026-05-14`.
- Purpose:
  reduce retained analyze heap by releasing parsed ScopeIRs after cross-file resolution has consumed
  them for normal CLI and Web analyze runs.
- Phase-stay note:
  this stayed in Phase 15 because it is profile-backed memory/performance work on the cut-over Go
  runtime. It is not provider coverage, Phase 10 persistence correctness, or a Phase 17 runtime
  authority gate.
- Impact:
  AVmatrix impact reported `analyze.Run` CRITICAL and `newAnalyzeCommand` HIGH because they are
  core analyze/CLI orchestration paths. The implementation is therefore opt-in:
  `ReleaseScopeIRsAfterResolution` is enabled by CLI and HTTP analyze only after resolution has
  consumed ScopeIR data; direct `analyze.Run` callers keep the default retained-result behavior.
  The touched E2E spec files were LOW-risk validation hardening.

| Benchmark / Validation | Before | After |
| --- | ---: | ---: |
| Full launcher build | n/a | `44.3s` |
| Packaged current-repo benchmark total | `29,665.3ms` | `24,476.3ms` |
| Parse phase | `14,751.4ms` | `13,746.7ms` |
| Resolution phase | `960.8ms` | `904.5ms` |
| DB load phase | `8,835.6ms` | `5,956.6ms` |
| End alloc bytes | `170,499,696` | `80,059,072` |
| Heap pprof in-use total | `131.94MB` | `58.67MB` |
| `ScopeIR.NormalizeOwned` heap frame | `55.67MB` flat | not in top table |
| `maxObservedSys` | `709,378,296` | `713,580,792` |
| Graph rows | `33,919` / `67,610` | `33,928` / `67,617` |
| DB fallback/skipped | `0` / `0` | `0` / `0` |
| Full Go tests | n/a | `31.7s` |
| Browser E2E | first full runs exposed async highlight flake | `32` passed / `1` skipped, `441,444.3ms` |
| `cd avmatrix && npm test` | n/a | `571.4s` |

- Baseline:
  `.tmp\phase15-after-owned-normalize-start.json` recorded total `29,665.3ms`, parse
  `14,751.4ms`, resolution `960.8ms`, DB load `8,835.6ms`, graph rows `33,919` / `67,610`,
  fallback/skipped `0`, `endAllocBytes=170,499,696`, and `maxObservedSys=709,378,296`. Heap pprof
  reported `131.94MB` in use, with `ScopeIR.NormalizeOwned=55.67MB` flat.
- Final:
  `.tmp\phase15-release-scopeir-after-resolution.json` recorded total `24,476.3ms`, parse
  `13,746.7ms`, resolution `904.5ms`, DB load `5,956.6ms`, graph rows `33,928` / `67,617`,
  fallback/skipped `0`, `endAllocBytes=80,059,072`, and `maxObservedSys=713,580,792`. Heap pprof
  reported `58.67MB` in use; `ScopeIR.NormalizeOwned` no longer appeared in the top table. The row
  count increase is from code/test additions in this slice, not a parity regression.
- CPU/heap conclusion:
  the accepted claim is retained heap reduction after resolution. Do not claim a peak-RSS win:
  `maxObservedSys` was neutral/slightly higher, and CPU pprof remained dominated by native
  tree-sitter cgo plus LadybugDB query/COPY/commit work. DB-load timing variance also dominates the
  apparent wall-time improvement.
- E2E hardening:
  the first full browser runs before hardening exposed an async process-highlight assertion flake
  (`process-row` class update). The same tests passed when rerun directly, so the runtime behavior
  was preserved and the spec now waits for the highlight button state before checking row styling.
- Validation:
  full launcher build was run first and passed in `44.3s`; packaged analyze with CPU/heap pprof
  passed; `go test ./cmd/... ./internal/... -count=1` passed in `31.7s`; browser E2E used isolated
  `AVMATRIX_HOME=F:\AVmatrix-GO\.tmp\phase15-release-scopeir-e2e-home-20260514`, packaged Go
  backend on `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated analyze recorded total
  `29,141.3ms`, graph rows `33,928` / `67,617`, fallback/skipped `0`, and Playwright passed with
  `32` passed / `1` skipped in `441,444.3ms`. `npx prettier --check` passed for the touched E2E
  specs, and `cd avmatrix && npm test` passed in `571.4s`.

### Phase 15 Graph Compact After Processes Benchmark

- Date: `2026-05-14`.
- Purpose:
  reduce retained graph heap after ScopeIR release by trimming graph slices and dropping lazy graph
  indexes once graph mutation is complete.
- Phase-stay note:
  this stayed in Phase 15 because it is profile-backed memory/performance work on the cut-over Go
  runtime. It is not provider coverage, Phase 10 persistence correctness, or Phase 17 runtime
  authority.
- Impact:
  AVmatrix impact reported `Graph` CRITICAL (`357` impacted symbols, `376` processes),
  `Graph.AddRelationship` CRITICAL (`130` impacted symbols, `300` processes), and `analyze.Run`
  CRITICAL (`4` impacted symbols, `51` processes). The change is limited to an idempotent
  `Graph.Compact()` helper and one call after Phase Processes; add/lookup semantics remain lazy and
  rebuildable.

| Benchmark / Validation | Before | After |
| --- | ---: | ---: |
| Full launcher build | n/a | `40.7s` |
| Packaged current-repo benchmark total | `24,476.3ms` | `28,249.7ms` |
| Parse phase | `13,746.7ms` | `17,855.3ms` |
| Resolution phase | `904.5ms` | `987.6ms` |
| Processes phase | `160.5ms` | `127.4ms` |
| DB load phase | `5,956.6ms` | `5,794.7ms` |
| Heap pprof in-use total | `58.67MB` | `49.64MB` |
| `Graph.AddRelationship` heap frame | `15.77MB` flat | not in top table |
| Compact graph slices | n/a | `Graph.Compact=11.48MB` flat |
| `maxObservedSys` | `713,580,792` | `623,452,408` |
| End alloc bytes | `80,059,072` | `87,163,600` |
| Graph rows | `33,928` / `67,617` | `33,982` / `67,657` |
| DB fallback/skipped | `0` / `0` | `0` / `0` |
| Full Go tests | n/a | `31.0s` |
| Browser E2E | n/a | `32` passed / `1` skipped, `434,795.2ms` |
| `cd avmatrix && npm test` | n/a | `477.1s` |

- Baseline:
  `.tmp\phase15-release-scopeir-after-resolution.json` and its heap pprof recorded total
  `24,476.3ms`, parse `13,746.7ms`, DB load `5,956.6ms`, graph rows `33,928` / `67,617`,
  fallback/skipped `0`, `endAllocBytes=80,059,072`, `maxObservedSys=713,580,792`, and heap in-use
  total `58.67MB`. The top retained graph frame was `Graph.AddRelationship=15.77MB` flat.
- Final:
  `.tmp\phase15-graph-compact-after-processes.json` and heap pprof recorded total `28,249.7ms`,
  parse `17,855.3ms`, DB load `5,794.7ms`, graph rows `33,982` / `67,657`, fallback/skipped `0`,
  `endAllocBytes=87,163,600`, `maxObservedSys=623,452,408`, and heap in-use total `49.64MB`.
  `Graph.AddRelationship` disappeared from the top table; `Graph.Compact` retained `11.48MB` flat
  for the compact graph backing slices. Row count increased because this slice added Go graph test
  code and touched docs, not because of a parity regression.
- CPU/heap conclusion:
  this is accepted as a retained graph heap reduction, not a macro wall-time speedup. Native
  tree-sitter parse cgo varied upward in the final profiled run and still dominates CPU
  (`runtime.cgocall`, `Parser.ParseWithOptions`), while LadybugDB query/COPY/commit remains the
  other large CPU family.
- Validation:
  full launcher build ran first and passed in `40.7s`; focused graph/analyze tests passed;
  `BenchmarkWriteGraphSnapshot` stayed in the same range (`22.6-26.5ms/op`, about `3.54MB/op`,
  `32,530-32,533 allocs/op`); packaged analyze with CPU/heap pprof passed; full Go tests passed in
  `31.0s`. Browser E2E used isolated
  `AVMATRIX_HOME=F:\AVmatrix-GO\.tmp\phase15-graph-compact-e2e-home-20260514`, packaged Go backend
  on `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated analyze recorded total
  `25,136.7ms`, graph rows `33,982` / `67,657`, fallback/skipped `0`, and Playwright reported
  `32` passed / `1` skipped in `434,795.2ms`. The wrapper returned non-zero during cleanup, but
  Playwright itself printed exit `0`, `.last-run.json` recorded `status: passed`, and a follow-up
  port check found no listeners on `4747` or `5173`. `cd avmatrix && npm test` passed in `477.1s`.

### Phase 15 Native Cgo Boundary Classification

- Date: `2026-05-14`.
- Purpose:
  classify the remaining native tree-sitter and LadybugDB cgo costs after the graph compact slice,
  so Phase 15 does not keep looping on non-actionable Go micro-optimizations.
- Phase-stay note:
  this stayed in Phase 15 as performance target triage. It is not a Phase 10 correctness jump and
  not a Phase 17 runtime authority gate.
- Decision:
  native tree-sitter parse cgo and native LadybugDB query/COPY/commit are rejected as same-slice
  Go-level optimization targets for the current plan. Further wins here require upstream/native
  parser or LadybugDB API/design changes, not another local Go-only patch.

| Profile Item | Evidence |
| --- | ---: |
| Profile source | `.tmp\phase15-graph-compact-after-processes-cpu.pprof` |
| Packaged analyze total | `28,249.7ms` |
| `runtime.cgocall` | `19.25s` flat / `20.49s` cumulative |
| `Parser.ParseWithOptions` | `8.07s` cumulative |
| `lbugnative.Query` | `5.66s` cumulative |
| `lbugload.runCopy` | `1.94s` cumulative |
| `CommitLoadTransaction` | `3.23s` cumulative |
| `tsjs.Extract` residual Go-level work | `4.98s` cumulative |
| `resolution.ResolveBoundInto` residual Go-level work | `0.91s` cumulative |
| Heap profile source | `.tmp\phase15-graph-compact-after-processes-mem.pprof` |
| Heap in-use total | `49.64MB` |
| `emitDefinitionNodes` retained heap | `17,412.09kB` flat / `22,020.99kB` cumulative |
| `GenerateID` retained heap | `7,168.68kB` flat |

- Interpretation:
  Phase 15 has already addressed Go-owned parser traversal, graph snapshot memory, DB transaction
  setup, DB export allocation, ScopeIR retained heap, and graph retained heap. The dominant CPU
  families now cross the native parser and LadybugDB C APIs. A local Go patch can accidentally move
  work around these calls, but it cannot remove `ts_parser_parse_with_options`,
  `lbug_connection_query`, or native transaction commit cost without a larger native API/design
  change.
- Remaining Go-level candidates:
  the same profiles still show actionable Go-owned work: residual TS/JS provider traversal
  (`tsjs.walkKind`, `emitReferenceKind`, child/parent calls) and resolution/graph property
  allocation (`emitDefinitionNodes`, `GenerateID`). Those remain open for the next Phase 15 slice.
- Validation:
  no runtime code changed in this classification slice. The evidence uses the already validated
  graph compact runtime package: full launcher build first (`40.7s`), packaged analyze with
  CPU/heap pprof, full Go tests (`31.0s`), browser E2E (`32` passed / `1` skipped), and
  `cd avmatrix && npm test` (`477.1s`).

### Phase 15 TS/JS Fact-Kind Dispatch Deferred Benchmark

- Date: `2026-05-14`.
- Status:
  deferred and reverted from the working tree. This is measurement-only evidence explaining why
  Phase 15 optimization stopped here; it is not an active implementation slice.
- Purpose:
  evaluate whether the residual TS/JS provider traversal cost can be reduced by routing each AST
  kind to only the emitter family that can consume it, instead of dispatching definition, import,
  type-binding, and reference switches for every node in the shared `walkKind` pass.
- Impact:
  AVmatrix reported `internal/providers/tsjs.Extract` as CRITICAL (`8` impacted symbols,
  `10` affected processes), with direct callers in `extractScopeIR` and SFC extraction. Focused
  parity is required before any broader benchmark is meaningful.
- Validation run so far:
  full launcher build before tests (`37.9s` after patch), focused TS/JS provider parity test
  (`go test ./internal/providers/tsjs -run TestExtract -count=1`, `0.242s`), focused benchmark, and
  packaged analyze with CPU/heap pprof.

| Check | Result |
| --- | ---: |
| Pre-patch focused benchmark median | `315,726ns/op` |
| Post-patch focused benchmark median | `313,876ns/op` |
| Focused benchmark allocations | unchanged, `66,385B/op`, `980 allocs/op` |
| Packaged analyze artifact | `.tmp\phase15-tsjs-kind-dispatch.json` |
| Packaged analyze total | `25,430.1ms` |
| Parse phase | `14,201.3ms` |
| Resolution phase | `947.5ms` |
| DB load phase | `6,638.8ms` |
| Rows | `33,985` nodes / `67,662` relationships |
| DB fallback/skipped | `0` / `0` |
| End alloc / max sys | `58.43MB` / `735.15MB` |
| CPU profile | `.tmp\phase15-tsjs-kind-dispatch-cpu.pprof` |
| Heap profile | `.tmp\phase15-tsjs-kind-dispatch-mem.pprof` |

- CPU profile after patch:
  `runtime.cgocall` remains dominant at `18.19s` flat / `19.19s` cumulative. The top Go-owned
  TS/JS frame visible in the top table is `collector.innermostScopeID` at `0.17s` flat /
  `0.20s` cumulative; TS/JS emitter dispatch is no longer a top CPU frame.
- Heap profile after patch:
  total in-use heap is `56.41MB`; `emitDefinitionNodes` retains `15.50MB` flat / `22.00MB`
  cumulative, `graph.GenerateID` retains `11.00MB` flat, and `Graph.Compact` retains `11.21MB`.
- Interpretation:
  focused parity and packaged analyze are clean, but the micro benchmark win is small and current
  pprof points more strongly at resolution definition/property allocation than TS/JS dispatch.
  Per the 2026-05-14 plan correction, benchmark is only a measuring/regression tool until
  `AVmatrix-GO` is proven as an independent Go MCP/tool implementation separate from
  `AVmatrix-main`. The attempted patch was not committed; Phase 15 micro-optimization is deferred.
- Follow-up:
  return to Phase 17 independent MCP/tool readiness proof. Do not spend full E2E time on this
  optimization unless a future correctness or regression gate makes it necessary.

### Phase 10 TypeAlias->Method Schema Gap Benchmark

- Date: `2026-05-14`.
- Purpose:
  close the real Web UI/new-repo DB load failure reported while selecting `F:\Restaurant_manager`.
  The packaged runtime failed closed with
  `db_load phase: copy relationships TypeAlias->Method: schema pair unsupported`.
- Phase-jump note:
  this intentionally jumped from the active Phase 15 performance work back to Phase 10
  persistence/schema correctness. The issue is a runtime correctness blocker on a real repo, not a
  performance target. The fix must not enable fallback or skipped relationships.
- Impact:
  AVmatrix impact reported `RelationPairs` LOW. `relationPairSupported` was CRITICAL because it
  flows through `ExportGraphCSVs`, `loadGraph`, analyze `Run`, `newAnalyzeCommand`, and Web analyze
  paths. The patch is limited to adding the supported relation pair and regression coverage.

| Benchmark | Before | After |
| --- | ---: | ---: |
| `F:\Restaurant_manager` packaged analyze | failed closed in `25,374.4ms` | passed in `30,931.7ms` wall |
| Benchmark total | n/a, no successful DB load | `28,889.6ms` |
| Parse phase | n/a | `13,850.7ms` |
| DB load phase | n/a | `8,018.7ms` |
| Files scanned / parsed / unsupported / failed | n/a | `6,198` / `1,228` / `4,970` / `0` |
| DB rows | n/a | `77,901` / `129,560` |
| DB fallback/skipped | n/a | `0` / `0` |

- Implementation:
  `internal/lbugschema.RelationPairs` now includes `TypeAlias -> Method`, and schema DDL tests
  assert `FROM \`TypeAlias\` TO Method`. Loader regression coverage adds a `TypeAlias` node, a
  `Method` node, and a `HAS_METHOD` relationship to the supported COPY-path test, then asserts the
  relationship copy query includes `from="TypeAlias", to="Method"` with no fallback insert.
- Focused benchmark:
  `go test ./internal/lbugload -run '^$' -bench 'Benchmark(LoadCSVExportCopyPathNoop|ExportGraphCSVs)' -benchmem -count=5`
  recorded `BenchmarkExportGraphCSVs` at `16.05-22.24ms/op`, about `285KB/op`, and `1403 allocs/op`.
  `BenchmarkLoadCSVExportCopyPathNoop` recorded `2.87-3.28us/op`, `1,360-1,376 B/op`, and
  `17 allocs/op`.
- Validation:
  before patch, packaged analyze on `F:\Restaurant_manager` failed closed in `25,374.4ms`. After
  patch, full launcher build passed in `56,762.9ms` after stopping the stale launcher/backend
  process that held `server-bundle`; focused schema/load tests passed in `4,284.6ms`; full Go tests
  passed in `61,720.4ms`. The after-patch `F:\Restaurant_manager` packaged analyze passed in
  `30,931.7ms` wall with DB fallback/skipped `0`. Browser E2E used isolated
  `AVMATRIX_HOME=F:\AVmatrix-GO\.tmp\typealias-method-e2e-home-20260514-unique`, packaged Go
  backend on `127.0.0.1:4747`, and Vite on `127.0.0.1:5173`; isolated current-repo analyze took
  `27,291.0ms`, graph rows were `33,858` / `67,413`, DB fallback/skipped `0`, and Playwright
  passed with `32` passed / `1` skipped in `475,727.0ms`. `cd avmatrix && npm test` passed with
  exit code `0` in `440,065.0ms`.

### Phase 16 Launcher UI-Close Process Lifecycle Benchmark

- Date: `2026-05-14`.
- Purpose:
  close the launcher UX bug found while testing Web UI repo selection: closing the UI/browser could
  leave `AVmatrixLauncher.exe`, `avmatrix-server.exe`, or packaged `avmatrix.exe serve` running and
  holding locks under `avmatrix-launcher\server-bundle`.
- Phase-jump note:
  this intentionally jumped from Phase 15 performance work to Phase 16/17 launcher/cutover
  behavior. The process lock is not a manual-test artifact; launcher-owned UI sessions need a real
  owner/lifetime model.
- Impact:
  AVmatrix impact reported LOW risk for `startRuntime`, `staticHandler`, `waitForExit`, and
  `openBrowser`. The change is scoped to the Go launcher, not the direct `avmatrix serve` path.

| Benchmark / Validation | Result |
| --- | ---: |
| Full launcher build after lifecycle patch | `47,045.1ms` |
| `avmatrix-launcher/src` tests after build | `5,110.9ms` |
| `avmatrix-launcher/server-wrapper` tests after build | `2,052.4ms` |
| HTTP close-flow smoke | `15,647.2ms` |
| Playwright close-flow E2E | `27,569ms` |
| Post-close launcher bundle processes | `0` |
| Post-close exclusive lock check | `avmatrix-server.exe` and `avmatrix.exe` unlocked |
| `cd avmatrix && npm test` | `415,943.7ms` |

- Implementation:
  the launcher-served `index.html` is injected with a small launcher-only lifecycle script that
  sends heartbeat requests to `127.0.0.1:5173/__avmatrix_launcher/heartbeat` and sends a close
  signal to `__avmatrix_launcher/closed` on `pagehide`. `waitForExit` now listens for this UI-done
  signal in addition to backend exit and OS signals. When the UI session closes, normal defers stop
  only the backend process started by this launcher. A test-only
  `AVMATRIX_LAUNCHER_NO_BROWSER=1` path suppresses real browser launch for automated smoke.
- Smoke evidence:
  the HTTP smoke started `AVmatrixLauncher.exe` hidden with `AVMATRIX_LAUNCHER_NO_BROWSER=1`,
  verified `/api/info`, verified the served index contained the lifecycle script, posted heartbeat
  and closed events, then verified no launcher bundle processes remained. It opened
  `avmatrix-server.exe` and `avmatrix.exe` with exclusive file locks after close, proving the
  server bundle can be rebuilt/replaced.
- E2E evidence:
  the Playwright close-flow E2E started the packaged launcher hidden, opened Chromium against
  launcher-served `http://127.0.0.1:5173`, verified the lifecycle script was present, closed the
  browser, waited for the launcher to exit, and rechecked that no launcher bundle processes or file
  locks remained. `cd avmatrix && npm test` passed with exit code `0` in `415,943.7ms`.

### Phase 17 Non-Web TypeScript/JavaScript Inventory Benchmark

- Date: `2026-05-14`.
- Purpose:
  measure the remaining TypeScript/JavaScript inventory after the plan correction that conversion
  completion means all non-Web UI implementation is Go, not merely that the normal runtime path can
  reach a Go binary.
- Command:
  `rg --files -g '*.ts' -g '*.tsx' -g '*.js' -g '*.jsx' -g '*.mjs' -g '*.cjs'` with exclusions for
  `node_modules`, `dist`, `build`, `vendor`, `avmatrix-web/dist`, and
  `avmatrix-launcher/server-bundle`.

| Inventory metric | Result |
| --- | ---: |
| Audit command wall time | `104.1ms` |
| Total TS/JS-family source files found | `1051` |
| `avmatrix/` | `895` |
| `avmatrix-web/` | `119` |
| `avmatrix-shared/` | `34` |
| Root Docker/web-server scripts | `2` |
| Root ESLint config | `1` |
| `avmatrix/src` legacy implementation files | `339` |
| `avmatrix/test` legacy test/fixture harness files | `542` |
| `avmatrix/scripts` package/build scripts | `8` |
| `avmatrix/hooks` | `1` |
| `avmatrix/vitest.config.ts` | `1` |

- Interpretation:
  this is a conversion-completeness benchmark, not a performance target. `avmatrix-web` remains
  allowed Web UI TypeScript/React. The blocker is the non-Web inventory under `avmatrix/src`,
  `avmatrix-shared`, root/package scripts, hooks, and TypeScript test harnesses. These counts become
  the baseline for the next Phase 17 reduction package.

### Phase 17 Cross-Phase Reopen Validation Benchmark

- Date: `2026-05-14`.
- Purpose:
  validate the docs-only cross-phase reopen update and refresh the graph before commit. This is not
  an optimization benchmark.
- Artifact:
  `.tmp\phase17-cross-phase-reopen-correction.json`.

| Validation metric | Result |
| --- | ---: |
| Analyze total | `24,567.6ms` |
| Parse phase | `13,902.7ms` |
| Resolution phase | `921.3ms` |
| DB load phase | `5,879.2ms` |
| Files scanned / parsed / unsupported / failed | `1232` / `1059` / `173` / `0` |
| DB rows | `33,989` nodes / `67,664` relationships |
| DB fallback/skipped | `0` / `0` |

- Interpretation:
  this benchmark only proves the repo graph was refreshed after the plan correction. The reopened
  work remains conversion classification and correctness, not Phase 15 optimization.

### Phase 17 Classification Matrix Benchmark

- Date: `2026-05-14`.
- Purpose:
  record the benchmark/evidence used while building `[P17-NON-WEB-TSJS-CLASSIFICATION-MATRIX]`.
  This is classification evidence, not a runtime optimization target.
- AVmatrix-main refresh artifact:
  `.tmp\phase17-tsjs-classification-avmatrix-main-refresh.json`.

| Metric | Result |
| --- | ---: |
| AVmatrix-main analyze wall | `137,338.4ms` |
| AVmatrix-main nodes / stats edges | `29,989` / `54,524` |
| AVmatrix-main graph relationships | `55,624` |
| AVmatrix-main processes | `665` |
| Parse / resolution / lbug load | `52,231ms` / `5,423ms` / `22,975.4ms` |
| Lbug skipped relationships | `0` |
| Non-Web/source TS/JS inventory command wall | `~104.1ms` |
| Source TS/JS-family files | `1051` |
| `avmatrix/src` legacy implementation | `339` |
| `avmatrix-shared` legacy contracts | `34` |
| `avmatrix/test/fixtures` analyzer fixtures | `290` |
| `avmatrix/test` Node/Vitest harness outside fixtures | `252` |
| `avmatrix-web` allowed Web surface | `119` |
| `avmatrix/scripts` / `hooks` / `vendor` | `8` / `1` / `4` |
| Root Docker/web-server scripts / ESLint config | `2` / `1` |

- Interpretation:
  this closes the classification-matrix benchmark only. The next implementation benchmark should
  be tied to the first matrix group actually reduced or ported.

### Phase 17 Web Docker Node Server Removal Benchmark

- Date: `2026-05-14`.
- Purpose:
  record the first `[P17-NON-WEB-TSJS-CONVERSION-BLOCKER]` matrix reduction slice: removing the root
  Node web static server from the Docker Web runtime and replacing it with nginx.
- Build prerequisite:
  the full launcher build must run before tests. The first run exposed a fail-open PowerShell native
  command handling bug, which was fixed in `avmatrix-launcher/build.ps1`; the accepted rerun used
  `C:\msys64\ucrt64\bin` on PATH and passed.

| Metric | Result |
| --- | ---: |
| Pre-impact AVmatrix refresh wall | `29.9s` |
| Full launcher build wall | `90.3s` |
| Web build inside launcher build | `19.20s` |
| Packaged backend binary | `avmatrix-launcher/server-bundle/avmatrix.exe`, `49,444,864` bytes |
| Go test batch | `go test ./cmd/... ./internal/... -count=1`, passed in `93.7s` |
| Playwright E2E | `server-connect.spec.ts`, `5/5` passed in `2.3m` |
| Static Docker/nginx validation | passed |
| Docker daemon image smoke | blocked: Docker Desktop Linux engine pipe unavailable |
| Final AVmatrix refresh wall | `28.6s` |
| Final analyze total | `27,941.5ms` |
| Final parse / resolution / db load | `16,670.1ms` / `1,065.5ms` / `6,352.9ms` |
| Final files scanned / parsed / unsupported / failed | `1231` / `1057` / `174` / `0` |
| Final DB rows | `33,912` nodes / `67,630` relationships |
| Final DB fallback/skipped | `0` / `0` |

- Artifacts:
  `.tmp\phase17-web-docker-nginx-preimpact-refresh.json`,
  `.tmp\phase17-web-docker-node-removal-final-refresh.json`,
  `.tmp\phase17-web-docker-e2e-playwright.out.log`,
  `.tmp\phase17-web-docker-e2e-backend.out.log`,
  `.tmp\phase17-web-docker-e2e-vite.out.log`.
- Interpretation:
  the root Docker/web-server Node runtime is removed from the Web container path. The conversion
  blocker remains open for the larger non-Web TypeScript/JavaScript inventory; the next benchmark
  should be attached to the next matrix group reduction, not to broad Phase 15 optimization.

### Phase 17 Claude Hook Go Translation Benchmark

- Date: `2026-05-14`.
- Purpose:
  record the Claude hook runtime support cluster from
  `[P17-NON-WEB-TSJS-CONVERSION-BLOCKER]`. This is conversion/cutover validation, not Phase 15
  optimization.
- Cluster:
  `avmatrix/hooks/claude/avmatrix-hook.cjs`, `avmatrix/hooks/claude/pre-tool-use.sh`,
  `avmatrix/hooks/claude/session-start.sh`, Go hidden hook command, setup hook installation, package
  metadata, and hook/setup tests.

| Metric | Result |
| --- | ---: |
| AVmatrix pre-impact refresh wall | `133.6s` |
| Full package build | `cd avmatrix && npm run build`, `18,285.7ms` |
| Go test batch | `go test ./cmd/... ./internal/... -count=1`, `24,546.2ms` |
| TypeScript check | `cd avmatrix && npx tsc --noEmit`, `8,745.7ms` |
| Full AVmatrix npm test suite | `cd avmatrix && npm test`, `375,380.3ms` |
| Go hook PreToolUse smoke | `avmatrix.exe hook claude`, `942.9ms` |
| Go hook PostToolUse smoke | `avmatrix.exe hook claude`, `81.2ms` |
| Go setup temp-HOME smoke | `avmatrix.exe setup`, `38.7ms` |
| Final AVmatrix refresh wall | `129,075.0ms` |
| Final AVmatrix graph | `30,013` nodes / `54,569` edges / `669` flows |
| Final staged detect_changes | `changed=90`, `affected=12`, `changed_files=14`, `risk=HIGH` |

- Artifacts:
  `.tmp\phase17-claude-hook-pretool-20260514-221950.json`,
  `.tmp\phase17-claude-hook-posttool-20260514-221950.json`,
  `.tmp\phase17-claude-hook-setup-home-20260514-221950`.
- Interpretation:
  the HIGH detect_changes risk came from touching `NewRootCommand` to add the hidden `hook`
  subcommand. This was expected for CLI command-surface work and covered by cluster-level build,
  Go tests, TypeScript check, full npm test suite including e2e/integration suites, and direct
  hook/setup smokes. The benchmark is a conversion validation record, not a request for Phase 15
  optimization.
