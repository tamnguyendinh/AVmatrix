# Full Analyze Performance Optimization Plan

Date: 2026-04-24  
Scope: `avmatrix/` analyze pipeline, persistence, FTS, embeddings, metadata; `avmatrix-web/` only if analyze progress/status contract changes  
Status: Proposed

## Goal

Make full `analyze` faster while preserving the existing full-repo semantics and output.

The final outcome is not just better logs or profiling. The final outcome is a measurably faster full `analyze` run on representative repositories, with the same graph, persistence, search, metadata, and caller-visible behavior.

This plan is only about full analyze. It is not about `re-analyze`, delta indexing, affected-file analysis, fast mode, stale derived state, or deferred graph output.

This plan ends when the analyzed index is ready faster: in-memory `KnowledgeGraph` has been built, LadybugDB has been loaded, FTS/optional embeddings have run, and metadata/AI context finalization has completed. It does not optimize how the web app later fetches, streams, parses, lays out, or renders the graph.

Success criteria:

- full `analyze --force` wall time is reduced on the benchmark repos
- each optimized phase shows a real timing improvement in the instrumentation report
- no correctness guard regression is accepted
- no optimization is considered complete unless it moves wall-clock time or removes a measured bottleneck

Optimization target model:

Full analyze wall time is treated as:

`total wall time = measured phase/bucket time + orchestration/progress overhead`

Primary optimization targets are the measured buckets:

- `scan`
- `structure`
- `markdown`
- `cobol`
- `parse`
- `routes`
- `tools`
- `orm`
- `crossFile`
- `mro`
- `communities`
- `processes`
- `lbugLoad`
- `fts`
- `embeddings`
- `metadata`
- `aiContext`

The plan is to reduce total wall time by reducing the slowest measured buckets first. Do not optimize all buckets equally. A change only counts as a speedup when it reduces the targeted bucket time and improves or preserves total wall time with identical output.

Related separate plan:

- [Web graph load and render performance plan](/F:/AVmatrix-main/docs/plans/2026-04-24-web-graph-load-render-performance-plan.md)

## Non-Negotiable Principles

1. `analyze` must represent the entire repository.
2. `analyze` must keep the same graph semantics and relationship semantics.
3. No phase may be skipped, deferred, or marked stale in the default full analyze path.
4. No new linker / resolver / graph-link architecture may be introduced.
5. Optimizations must preserve output: nodes, relationships, route/tool/process/community results, metadata, and query behavior.
6. Any optional incomplete mode must be a separate feature and must not be called full `analyze`.
7. The default full analyze path must not start using `skipGraphPhases`; that option is not a performance shortcut for production analyze.
8. Phase-level parallelism is out of scope unless shared `KnowledgeGraph` mutation safety and deterministic output are proven first.
9. Web `/api/graph` serialization, NDJSON client parsing, Sigma graph construction, and visual layout/rendering are separate concerns.
10. Preserve existing production skip/cap behavior unless a separate correctness review explicitly changes it.

## Current Full Analyze Shape

The current full analyze flow is:

1. early up-to-date check
2. full ingestion pipeline
3. LadybugDB rebuild / load
4. FTS index creation
5. optional embeddings
6. metadata / registry / AI context finalization

Pipeline phase order:

`scan -> structure -> markdown/cobol -> parse -> routes/tools/orm -> crossFile -> mro -> communities -> processes`

Current codebase constraints:

- `runFullAnalysis()` calls `runPipelineFromRepo(repoPath, onProgress)` without pipeline options; full analyze therefore includes graph phases by default.
- `PipelineOptions.skipGraphPhases` exists, but it is not a valid optimization for full analyze semantics.
- The pipeline runner executes phases in dependency order and each phase mutates or reads the shared graph; optimize inside phases before considering phase parallelism.
- The server analyze worker is a short-lived forked child process and exits after completion, so cross-job worker reuse is not a safe V1 assumption.
- Full analyze currently rebuilds LadybugDB by removing the existing DB/WAL/lock files, then loading the complete graph.
- FTS and AI context generation are best-effort today; performance changes must preserve their failure behavior unless deliberately changed.
- Embeddings are optional and are auto-skipped above `EMBEDDING_NODE_LIMIT`; current code threshold is `100_000` nodes. This is a tuning knob, not a full-analyze completeness rule.
- Scan currently skips files larger than `512KB`; keep that behavior unless a separate parser/file-size plan changes it.
- Parse currently skips languages whose tree-sitter parser is unavailable and warns; do not hide or reinterpret those warnings.
- Cross-file propagation currently has an internal benefit threshold and a hard cap; do not remove, tighten, loosen, or bypass them as part of generic performance work.

Key code areas:

- [run-analyze.ts](/F:/AVmatrix-main/avmatrix/src/core/run-analyze.ts)
- [pipeline.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline.ts)
- [runner.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline-phases/runner.ts)
- [filesystem-walker.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/filesystem-walker.ts)
- [parse-impl.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline-phases/parse-impl.ts)
- [cross-file-impl.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline-phases/cross-file-impl.ts)
- [lbug-adapter.ts](/F:/AVmatrix-main/avmatrix/src/core/lbug/lbug-adapter.ts)

## Workstream A. Instrumentation First

Before optimizing, add timing breakdown that is always available in analyze logs or debug output.

Required top-level timings:

- `scan`
- `structure`
- `markdown`
- `cobol`
- `parse`
- `routes`
- `tools`
- `orm`
- `crossFile`
- `mro`
- `communities`
- `processes`
- `lbugLoad`
- `fts`
- `embeddings`
- `metadata`
- `aiContext`
- orchestration/progress overhead where measurable
- total wall time

Required counters:

- total files
- parseable files
- total parseable MB
- node count
- edge count
- worker count
- parse chunk count
- CSV node rows
- CSV relationship rows
- LadybugDB COPY count
- FTS index timings by table

Required sub-timings:

- `loadGraphToLbug`: CSV generation, node COPY, relationship split, relationship COPY, cleanup
- `parse`: read contents, worker parse, import resolve, heritage resolve, call resolve, exported type map enrichment
- `markdown` / `cobol`: file reads, parse/extract time, graph write time
- `crossFile`: topological sort, candidate selection, file reread, `processCalls`
- `mro`: C3 linearization, ancestor traversal, transitive edge-type building, MRO name materialization, method collection, `METHOD_IMPLEMENTS` emission

Outcome:

- a baseline timing report for representative repos
- no performance change yet
- no output change

Phase 0 observed issue:

- `mro-processor.test.ts > handles very deep single-inheritance chain without stack overflow` can timeout in the full unit suite while passing alone.
- Isolated measurement showed the depth-2000 Python chain takes roughly `9-10s` inside `computeMRO`; the individual Vitest test is therefore close to its `15s` timeout and can fail under full-suite CPU contention.
- This is not a Phase 0 instrumentation regression: Phase 0 does not modify `mro-processor.ts` or `model/resolve.ts`.
- Treat this as a real `mro` performance bottleneck, not as a flaky-test-only problem.
- The measured hot spots are `c3Linearize`, `buildTransitiveEdgeTypes`, `gatherAncestors`, MRO name materialization, and `METHOD_IMPLEMENTS` ancestor scanning.
- Do not fix this by skipping MRO, reducing inheritance depth, weakening semantics, or merely raising the timeout without recording the underlying cost.

Initial benchmark baseline:

| Repo | Total | Top measured buckets | Decision signal |
|------|-------|----------------------|-----------------|
| `F:\Restaurant_manager` | `641.4s` | `parse 480.7s`, `lbugLoad 46.1s`, `crossFile 43.2s`, `fts 33.6s`, `markdown 29.5s`, `scan 5.1s` | Large-repo bottleneck is parse; worker pool timed out and fell back to sequential. |
| `F:\Website` | `60.4s` | `crossFile 21.0s`, `fts 13.2s`, `parse 12.3s`, `lbugLoad 11.6s`, `scan 0.3s` | Smaller repo has a different profile; crossFile/FTS matter more after parse is acceptable. |

Priority decision:

- Optimize for the large-repo profile first.
- `scan` and generic progress overhead are not priority targets from this baseline.
- The next implementation phase should investigate and fix parse worker timeout/fallback before tuning lower-impact areas.
- After parse improves, rerun both repos and then choose among `crossFile`, `fts`, and `lbugLoad` based on the new measured wall time.

## Workstream B. Scan / File IO

Possible optimizations that preserve output:

- make `READ_CONCURRENCY` configurable and benchmark `32`, `64`, `128`
- throttle scan progress callbacks to every N files or every 100ms
- cache ignore matcher for repeated analyzes in the same process when safe
- remove `results.indexOf(result)` inside scan result handling
- verify `glob('**/*')` plus ignore rules do not enumerate unnecessary files

Validation:

- same scanned file set
- same skipped-large-file behavior
- same parser-unavailable warning behavior downstream
- same downstream node/edge output

## Workstream C. Parse Worker Pool

Possible optimizations that preserve output:

- replace static file-count worker partitioning with a deterministic dynamic parse scheduler
- split parse work into small byte-bounded work units with stable `unitId`s
- let idle workers pull the next available work unit while the main thread merges results by stable `unitId`, not completion order
- use worker heartbeat/inactivity timeout instead of treating a whole sub-batch wall time as failure
- retry failed work units at smaller granularity, then isolate single-file failures before using sequential fallback
- reduce serialization payload to workers if profiling shows worker transfer overhead
- tune worker count relative to CPU cores after the scheduler is deterministic

Validation:

- same parsed symbols
- same imports, heritage, calls, routes, tools, ORM outputs
- same deterministic output regardless of worker completion order
- same parser-unavailable skip/warn behavior
- fallback is granular: only failed units/files are retried or parsed sequentially

Out of scope for V1:

- reusing parser workers across separate analyze jobs in the server worker process; the worker currently exits after completion

## Workstream D. Parse Main-Thread Resolve

Possible optimizations that preserve output:

- profile `processCallsFromExtracted`
- optimize cache lookup, map access, and repeated `generateId` hot spots
- profile `processImportsFromExtracted`
- cache repeated suffix / normalized path work where safe
- avoid full `buildExportedTypeMapFromGraph` scan when parse already populated equivalent data
- measure `synthesizeWildcardImportBindings`; avoid redundant graph-global work when no new wildcard-relevant input exists

Rules:

- do not change call/import/heritage resolution semantics
- do not add a new resolver
- do not rewrite link graph logic

Validation:

- relationship `type/source/target/confidence/reason/step` matches baseline
- relevant node properties and search fields match baseline
- same node and edge counts

## Workstream E. CrossFile

Known correctness issue to resolve before optimizing this phase:

- In the worker path, raw parse-worker output for Go receiver methods can be correct, but `crossFile` can still add incorrect duplicate `CALLS` edges during reprocessing.
- Reproduction case observed in `H:\hotel_manager\apps\desktop\backend\internal\usecase\auth\service.go`:
  - actual code: calls inside `AuthService.Login/Logout/ForgotPassword/ResetPassword`
  - expected source attribution: `AuthService.*`
  - incorrect crossFile output: duplicate calls attributed to same-name interface methods such as `AccountClient.Login/Logout/ForgotPassword/ResetPassword`
- Root cause: `crossFile` calls `processCalls(...)`; `processCalls` uses `findEnclosingFunction(...)`; for Go `method_declaration` nodes, owner disambiguation can miss the receiver because the helper starts from `node.parent` while the caller already passes the `method_declaration` node. When an interface method and receiver method share the same name in the same file, resolution can fall back to the first same-file candidate, which is often the interface method.
- This is not a parse-worker extraction bug and must not be fixed by changing scheduler, partitioning, worker fallback, or graph-link architecture.
- Phase 3 must fix this within the existing crossFile / `processCalls` source-attribution path before treating crossFile timing optimization as complete.

Possible optimizations that preserve output:

- profile `topologicalLevelSort`
- profile candidate selection
- profile `readFileContents`
- profile `processCalls`
- avoid rereading file content if a bounded, correct cache already has it
- avoid internal reprocessing only when the existing threshold/cap semantics still produce identical output
- cache `buildImportedReturnTypes` and `buildImportedRawReturnTypes` inside the phase

Rules:

- `crossFile` remains the cross-file propagation mechanism
- preserve the existing `CROSS_FILE_SKIP_THRESHOLD` behavior unless a separate correctness review changes it
- preserve the existing `MAX_CROSS_FILE_REPROCESS` cap unless a separate correctness review changes it
- no replacement closure engine
- no manual link creation outside existing processors

Validation:

- same CALLS edges after cross-file propagation
- same cross-file reprocessed file count for baseline repos
- same files reprocessed under the existing threshold/cap behavior

## Workstream F. MRO / Communities / Processes

Possible optimizations that preserve output:

- optimize MRO graph traversal and index lookup
- optimize Python C3 linearization for deep single-inheritance chains without changing C3 output
- avoid eager transitive edge-type construction for languages or classes that do not need `implements-split` conflict resolution
- cache or reuse ancestor/MRO-derived data inside `computeMRO` where it preserves deterministic output
- avoid repeated `graph.getNode` / method-map scans in MRO hot loops
- keep the deep-chain unit benchmark as a tracked MRO performance case
- profile community input graph construction before Leiden execution
- reduce repeated graph scans in `processes`
- cache local adjacency / node maps within a phase
- avoid rebuilding equivalent temporary structures multiple times

Rules:

- keep default full analyze on the existing sequential phase runner unless deterministic graph mutation safety is proven
- do not skip MRO
- do not skip communities
- do not skip processes
- do not reduce process/community output for speed
- do not change language-specific MRO strategies or relationship semantics
- do not introduce a new inheritance/link graph architecture

Validation:

- same `METHOD_OVERRIDES` and `METHOD_IMPLEMENTS`
- same Python C3 MRO order for diamond, cyclic, and deep-chain cases
- same community count and memberships where deterministic
- same process count and process step relationships
- `mro-processor.test.ts` passes, including the depth-2000 single-inheritance chain

## Workstream G. LadybugDB Load

This may be a major bottleneck. Optimize persistence without changing graph output.

Possible optimizations:

- profile `streamAllCSVsToDisk`
- benchmark and tune the existing CSV escaping path
- benchmark and tune the existing `BufferedCSVWriter` / `FLUSH_EVERY` behavior
- benchmark and tune the existing file-content LRU cache
- split relationship CSV by label pair in one efficient pass
- reduce many tiny `COPY` calls if safe
- batch cleanup of generated CSV files
- measure disk CSV IO vs LadybugDB COPY time separately

Rules:

- full analyze continues to rebuild LadybugDB in this plan
- persisted DB must contain the same graph
- no delta writer in this plan
- keep LadybugDB writes sequential unless LadybugDB single-writer / transaction constraints are explicitly verified safe for concurrency
- preserve `COPY ... PARALLEL=false` semantics unless a DB-level benchmark and correctness check proves a safe alternative

Validation:

- same LadybugDB stats
- same query results for representative graph queries
- same FTS/search behavior after index creation

## Workstream H. FTS

Possible optimizations:

- time each index separately: `File`, `Function`, `Class`, `Method`, `Interface`
- optimize slow index creation paths
- test whether any FTS operations can run concurrently only if LadybugDB supports it safely

Rules:

- do not omit indexes from full analyze
- keep best-effort failure semantics unless deliberately changed and validated

Validation:

- same searchable tables and fields
- same search behavior on representative queries

## Workstream I. Progress / Logging

Possible optimizations:

- throttle progress callbacks at a common boundary
- emit progress at most every 100ms or when percent changes
- avoid per-file SSE / IPC churn for large repos
- keep terminal phase completion progress events
- never throttle or drop terminal/control events: `error`, `complete`, `done`, cancellation, retry, or `log`
- preserve progress `stats` and useful `detail` fields for CLI/server/web callers

Validation:

- UI still shows timely progress
- CLI/MCP callers still receive useful phase updates
- output graph unchanged

## Workstream J. Correctness Guard

Every optimization must compare before/after output.

Minimum correctness checks:

- node count
- edge count
- relevant node properties, labels, names, paths, content/search fields
- relationship `type/source/target/confidence/reason/step`
- route count and route links
- tool count and tool links
- MRO edge counts
- community count and memberships
- process count and process steps
- metadata stats
- LadybugDB query output for representative queries
- sample query results
- no missing files
- same skipped-large-file count
- same parser-unavailable skip/warn count
- same `usedWorkerPool` value for equivalent worker settings
- same cross-file reprocessed file count under existing threshold/cap settings

For risky changes, run the same repository before and after optimization and compare serialized graph output or LadybugDB query output.

## Sequential Implementation Phases

Implement this plan in order. Do not mix phases in one change unless the earlier phase has already passed its correctness gate. Each phase should leave the codebase in a releasable state.

### Phase 0. Baseline Instrumentation and Correctness Harness

Goal:

- make performance visible before changing behavior
- create before/after comparison tooling so later optimizations are measurable
- establish the baseline that later phases must beat
- rank the optimization targets by actual wall-clock cost

Scope:

- timing breakdown for pipeline phases and `runFullAnalysis`
- counters for files, parseable files, MB, nodes, edges, workers, chunks, CSV rows, COPY count
- sub-timings for parse, crossFile, `loadGraphToLbug`, FTS, embeddings, metadata, AI context
- baseline output comparator for representative repos
- bottleneck report that sorts measured buckets from slowest to fastest

Rules:

- no intended performance optimization in this phase
- no output change
- do not change phase ordering

Exit gate:

- baseline report exists for at least one medium repo and one large repo
- correctness comparator can detect node/edge/relationship/property differences
- target bottlenecks are ranked by wall time so Phase 1+ work is grounded in measured cost
- next phase selection is justified by measured cost, not guesswork
- `cd avmatrix && npx tsc --noEmit` passes

### Phase 1. Parse Worker Timeout and Throughput

Goal:

- eliminate large-repo parse worker timeout/fallback before optimizing smaller buckets
- replace the current static worker chunking with one coherent deterministic scheduler
- improve parse throughput without changing extracted symbols, relationships, or caller-visible behavior
- reduce full analyze wall time when parse worker scheduling/timeout/fallback is a measured bottleneck

Scope:

- reproduce and isolate the `Restaurant_manager` worker timeout: `Worker 1 sub-batch timed out after 30s (chunk: 176 items)`
- introduce a stable `ParseWorkUnit` model: `unitId`, file indexes/paths, total bytes, file count, and language breakdown
- build work units from the parseable file list with a byte/file budget; preserve a stable unit order independent of worker completion order
- replace one-static-chunk-per-worker dispatch with a dynamic scheduler: workers request or receive the next pending work unit when they become idle
- merge worker results only by sorted `unitId` after completion so output remains deterministic
- change timeout policy from whole-sub-batch wall time to worker inactivity: reset watchdog on heartbeat/progress, and report the last active unit/file on timeout
- emit verbose diagnostics for each worker/unit: elapsed time, file count, bytes, language breakdown, result sizes, retry count, and slow/stuck file hints
- implement granular retry/fallback: retry failed units with smaller units, isolate single-file failures, and only then sequentially parse the failed file/unit
- reduce worker payload serialization only where profiling proves overhead after scheduler correctness is established
- tune worker count relative to CPU cores after deterministic scheduling is in place
- keep worker reuse within a single analyze run only

Rules:

- do not skip files to avoid timeout
- do not hide parser failures or parser-unavailable warnings
- do not treat fallback-to-sequential as success when the goal is faster full analyze
- do not fallback the entire parse chunk/repo when only one unit or file failed
- do not reuse parser workers across server analyze jobs
- preserve sequential fallback behavior for real worker failure, but make it granular
- preserve parser-unavailable skip/warn behavior
- no new parser/linker/resolver architecture
- no temporary static-partition implementation that is expected to be replaced later by dynamic scheduling

Exit gate:

- `Restaurant_manager` no longer falls back to sequential for the whole parse workload because of worker timeout
- any remaining fallback is limited to failed work units/files and is reported with diagnostics
- same parsed symbols, imports, heritage, calls, routes, tools, ORM outputs
- same deterministic output across repeated runs even when work units complete in different orders
- same `usedWorkerPool` value for equivalent settings, unless the phase explicitly changes tested worker settings
- benchmark report shows parse time and full analyze wall time before/after on `Restaurant_manager`
- rerun `Website` after the change to catch regressions on the smaller profile
- full correctness guard passes

### Phase 2. Parse Main-Thread Resolve Hotspots

Goal:

- reduce CPU cost in import/call/heritage resolution without changing graph-link semantics
- reduce full analyze wall time when main-thread resolution is a measured bottleneck

Scope:

- optimize `processCallsFromExtracted` hot paths
- optimize `processImportsFromExtracted` suffix/normalization/cache paths
- avoid redundant `generateId` and map lookup work
- avoid redundant `buildExportedTypeMapFromGraph` scans only when equivalent data already exists
- avoid redundant `synthesizeWildcardImportBindings` graph-global work only when inputs prove no new work

Rules:

- no new resolver
- no replacement link graph logic
- no manual relationship creation outside existing processors

Exit gate:

- relationship `type/source/target/confidence/reason/step` matches baseline
- relevant node properties and search fields match baseline
- benchmark report shows resolve sub-step time and full analyze wall time before/after
- if main-thread resolution is not a top bottleneck after Phase 1, record that and move to the next measured bottleneck
- full correctness guard passes

### Phase 3. CrossFile Optimization

Goal:

- first fix known crossFile correctness regressions caused by reprocessing before optimizing runtime
- reduce cross-file propagation cost while preserving existing propagation semantics
- reduce full analyze wall time when cross-file propagation is a measured bottleneck

Scope:

- reproduce the Go interface/concrete receiver same-name method case where parse-worker output is correct but crossFile reprocessing adds duplicate wrong-source `CALLS`
- fix source attribution in the existing `crossFile` / `processCalls` path so calls inside Go receiver methods remain attributed to the receiver owner, not to same-name interface methods in the same file
- add a regression check that `AuthService.* -> AccountClient.*` remains the edge shape and no `AccountClient.* -> AccountClient.*` self-edge is created by crossFile reprocessing
- profile `topologicalLevelSort`
- cache `buildImportedReturnTypes` and `buildImportedRawReturnTypes` inside the phase
- avoid repeated read/parse work only where the same files are still selected by existing logic
- reduce temporary structure rebuilds

Rules:

- preserve `CROSS_FILE_SKIP_THRESHOLD`
- preserve `MAX_CROSS_FILE_REPROCESS`
- preserve selected/reprocessed file count for baseline repos
- `crossFile` remains the propagation mechanism
- no new graph-link architecture
- do not change parse worker scheduling, partitioning, or fallback to address this crossFile bug

Exit gate:

- known Go receiver/interface same-name regression is fixed in the existing crossFile reprocess path
- no duplicate wrong-source `CALLS` are emitted for the observed `AuthService.*` / `AccountClient.*` case
- same cross-file reprocessed file count under equivalent settings
- same CALLS edges after cross-file propagation
- benchmark report shows crossFile time and full analyze wall time before/after
- if crossFile is not a top bottleneck after parse work, record that and move to the next measured bottleneck
- full correctness guard passes

### Phase 4. LadybugDB Load and FTS

Goal:

- reduce persistence and index creation time without changing stored graph/query behavior
- reduce full analyze wall time when DB load or FTS is a measured bottleneck

Scope:

- profile `streamAllCSVsToDisk`
- benchmark/tune existing CSV escaping, `BufferedCSVWriter`, `FLUSH_EVERY`, and file-content LRU cache
- optimize relationship CSV split by label pair if profiling shows cost
- reduce many tiny COPY calls only if safe
- batch CSV cleanup where safe
- time each FTS index separately

Rules:

- full analyze continues to rebuild LadybugDB
- no delta writer
- keep LadybugDB writes sequential unless DB-level safety is proven
- preserve `COPY ... PARALLEL=false` semantics unless a DB-level benchmark and correctness check proves a safe alternative
- keep FTS best-effort failure behavior

Exit gate:

- same LadybugDB stats
- same representative query/search results
- same searchable tables and fields
- benchmark report shows LadybugDB/FTS timings and full analyze wall time before/after
- if DB load/FTS is not a top bottleneck after parse/crossFile work, record that and move to the next measured bottleneck
- full correctness guard passes

### Phase 5. Derived Graph Phases: MRO, Communities, Processes

Goal:

- reduce repeated graph traversal and temporary structure rebuilds after parse/crossFile
- reduce full analyze wall time when derived graph phases are measured bottlenecks

Scope:

- optimize MRO graph traversal and lookup indexes
- specifically address measured MRO deep-chain cost in `computeMRO` / `c3Linearize`
- avoid unnecessary transitive edge-type work when a language strategy does not use it
- cache/reuse ancestor and MRO-derived structures only within the phase and only when output remains identical
- profile community input graph construction before Leiden execution
- cache local adjacency / node maps within a phase
- reduce repeated scans in process extraction

Rules:

- do not skip MRO, communities, or processes
- do not reduce process/community output
- keep default full analyze on the existing sequential phase runner
- keep existing language-specific MRO behavior: Python C3, C++ leftmost, C#/Java/Kotlin implements split, Rust qualified syntax
- do not change `METHOD_OVERRIDES` or `METHOD_IMPLEMENTS` relationship direction, confidence, or reason semantics

Exit gate:

- same MRO edge counts and override/implementation semantics
- deep-chain MRO benchmark improves materially from the Phase 0 baseline without increasing failures elsewhere
- full `mro-processor.test.ts` passes, not only the deep-chain test in isolation
- same community count and memberships where deterministic
- same process count and process step relationships
- benchmark report shows derived phase timings and full analyze wall time before/after
- if derived graph phases are not top bottlenecks, record that and move to the next measured bottleneck
- full correctness guard passes

### Phase 6. Scan / File IO / Progress Throttling

Goal:

- reduce low-risk overhead from progress churn and file stat/read scheduling only if later benchmarks show it matters
- keep this phase behind parse/crossFile/DB/FTS because initial baseline showed `scan` is not a primary bottleneck

Scope:

- throttle high-frequency progress callbacks
- preserve terminal/control events and useful `stats` / `detail`
- make scan/read concurrency configurable
- benchmark `READ_CONCURRENCY` values such as `32`, `64`, `128`
- remove `results.indexOf(result)` from scan result handling
- cache ignore matcher only when same-process reuse is safe

Rules:

- same scanned file set
- same skipped-large-file count
- same parser-unavailable behavior downstream
- no change to `MAX_FILE_SIZE`
- never throttle or drop terminal/control events: `error`, `complete`, `done`, cancellation, retry, or `log`

Exit gate:

- scan output matches baseline
- progress remains usable in CLI/server/web callers
- benchmark report shows whether this phase reduced full analyze wall time
- if scan/progress remains negligible, stop rather than over-optimizing it
- full correctness guard passes

### Phase 7. Optional Embeddings, Metadata, and Finalization

Goal:

- reduce finalization overhead without changing analyze result semantics
- reduce full analyze wall time when embeddings/finalization are measured bottlenecks

Scope:

- measure embedding cache restore, embedding generation, vector index creation, and embedding count query
- measure metadata save, registry update, `.gitignore` update, and AI context generation
- optimize only after Phase 4 confirms DB load/FTS costs are understood

Rules:

- embeddings remain optional
- preserve `EMBEDDING_NODE_LIMIT`
- preserve cached embedding restore semantics
- preserve AI context best-effort failure behavior
- preserve registry collision semantics

Exit gate:

- same metadata stats
- same embedding count behavior
- same registry result
- benchmark report shows finalization timings and full analyze wall time before/after
- if embeddings/finalization are not top bottlenecks, record that and defer further work
- full correctness guard passes

## Final Definition of Done

The plan is complete only when full `analyze` is faster end-to-end.

Required final evidence:

- before/after full `analyze --force` wall time on representative repos
- phase timing report showing where the speedup came from
- correctness guard report showing no accepted graph/output regression
- summary of any tunables changed and their selected values
- clear statement of remaining bottlenecks deferred to future work

## Validation Commands

- `cd avmatrix && npx tsc --noEmit`
- `cd avmatrix && npm test`
- targeted performance benchmark command to be added with instrumentation

## Non-Goals

- re-analyze / delta analyze
- fast mode
- skipped phases
- deferred derived graph
- stale metadata
- partial repo analysis
- changing relationship semantics
- replacing existing graph link logic
- cross-job analyze worker lifecycle changes
- web `/api/graph` response optimization
- frontend graph parsing, layout, and rendering optimization
