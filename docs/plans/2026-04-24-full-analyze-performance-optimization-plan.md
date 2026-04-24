# Full Analyze Performance Optimization Plan

Date: 2026-04-24  
Scope: `avmatrix/` analyze pipeline, persistence, FTS, embeddings, metadata; `avmatrix-web/` only if analyze progress/status contract changes  
Status: Proposed

## Goal

Make full `analyze` faster while preserving the existing full-repo semantics and output.

This plan is only about full analyze. It is not about `re-analyze`, delta indexing, affected-file analysis, fast mode, stale derived state, or deferred graph output.

This plan ends when the analyzed index is ready faster: in-memory `KnowledgeGraph` has been built, LadybugDB has been loaded, FTS/optional embeddings have run, and metadata/AI context finalization has completed. It does not optimize how the web app later fetches, streams, parses, lays out, or renders the graph.

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

Outcome:

- a baseline timing report for representative repos
- no performance change yet
- no output change

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
- same downstream node/edge output

## Workstream C. Parse Worker Pool

Possible optimizations that preserve output:

- benchmark parse chunk sizes: `10MB`, `20MB`, `40MB`, `80MB`
- benchmark worker count relative to CPU cores
- reduce serialization payload to workers if profiling shows worker transfer overhead
- reuse worker pool only within a single analyze run if lifecycle is safe and output remains identical
- tune worker threshold to avoid slow sequential path on medium repos

Validation:

- same parsed symbols
- same imports, heritage, calls, routes, tools, ORM outputs
- same worker and sequential fallback semantics

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

Possible optimizations that preserve output:

- profile `topologicalLevelSort`
- profile candidate selection
- profile `readFileContents`
- profile `processCalls`
- avoid rereading file content if a bounded, correct cache already has it
- skip internal reprocessing only when inputs prove no output change
- cache `buildImportedReturnTypes` and `buildImportedRawReturnTypes` inside the phase

Rules:

- `crossFile` remains the cross-file propagation mechanism
- no replacement closure engine
- no manual link creation outside existing processors

Validation:

- same CALLS edges after cross-file propagation
- same files reprocessed when output would otherwise differ

## Workstream F. MRO / Communities / Processes

Possible optimizations that preserve output:

- optimize MRO graph traversal and index lookup
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

Validation:

- same `METHOD_OVERRIDES` and `METHOD_IMPLEMENTS`
- same community count and memberships where deterministic
- same process count and process step relationships

## Workstream G. LadybugDB Load

This may be a major bottleneck. Optimize persistence without changing graph output.

Possible optimizations:

- profile `streamAllCSVsToDisk`
- reduce stringify / CSV escaping overhead
- increase write-stream buffer size
- split relationship CSV by label pair in one efficient pass
- reduce many tiny `COPY` calls if safe
- batch cleanup of generated CSV files
- measure disk CSV IO vs LadybugDB COPY time separately

Rules:

- full analyze continues to rebuild LadybugDB in this plan
- persisted DB must contain the same graph
- no delta writer in this plan

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

For risky changes, run the same repository before and after optimization and compare serialized graph output or LadybugDB query output.

## Suggested Implementation Order

1. Add timing breakdown.
2. Add progress throttling.
3. Profile `loadGraphToLbug`.
4. Tune scan/read concurrency.
5. Tune worker/chunk settings.
6. Optimize repeated graph scans in `crossFile`, `communities`, and `processes`.
7. Optimize LadybugDB CSV/COPY path.

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
