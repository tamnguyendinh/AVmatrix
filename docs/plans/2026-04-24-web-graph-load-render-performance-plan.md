# Web Graph Load and Render Performance Plan

Date: 2026-04-24  
Scope: `avmatrix/src/server/api.ts`, `avmatrix-web/src/services/backend-client.ts`, `avmatrix-web/src/hooks/`, `avmatrix-web/src/components/GraphCanvas.tsx`, `avmatrix-web/src/lib/graph-adapter.ts`  
Status: Proposed

## Goal

Make the web app open, transfer, parse, build, lay out, and render an already-analyzed graph faster.

This is a separate problem from full `analyze` performance. The input to this plan is an existing LadybugDB index under `.avmatrix/lbug`. The output is a faster web graph experience using the same indexed graph data.

Related separate plan:

- [Full analyze performance optimization plan](/F:/AVmatrix-main/docs/plans/2026-04-24-full-analyze-performance-optimization-plan.md)

## Non-Negotiable Principles

1. Do not change full analyze semantics or output.
2. Do not re-run analyze to improve web graph loading.
3. Do not rebuild or mutate LadybugDB in the graph-view path.
4. Keep full graph data available to users and tools.
5. Any partial, viewport, sampled, summarized, or lazy graph mode must be explicitly named and must not pretend to be the complete graph.
6. Preserve node IDs, relationship IDs, labels, and relationship semantics.
7. Optimize data transfer and rendering without inventing a new graph-link architecture.

## Current Web Graph Shape

Current flow:

1. Web calls `fetchGraph(...)`.
2. `fetchGraph` requests `/api/graph?...&stream=true`.
3. Server opens LadybugDB for the selected repo.
4. Server streams nodes table-by-table and then relationships as NDJSON.
5. Client parses NDJSON into `nodes[]` and `relationships[]`.
6. Client builds an in-memory `KnowledgeGraph`.
7. React stores the completed graph.
8. Graph adapter and Sigma build/render the visual graph.

Important current behavior:

- `/api/graph` has both JSON and NDJSON paths; the web client currently requests `stream=true`.
- NDJSON improves transfer behavior, but the frontend still accumulates all records before returning `{ nodes, relationships }`.
- `switchRepo` builds `KnowledgeGraph` after the full graph fetch has completed.
- Graph rendering then does additional node/edge mapping, layout, filtering, and Sigma updates.

Key code areas:

- [api.ts](/F:/AVmatrix-main/avmatrix/src/server/api.ts)
- [backend-client.ts](/F:/AVmatrix-main/avmatrix-web/src/services/backend-client.ts)
- [useAppState.local-runtime.tsx](/F:/AVmatrix-main/avmatrix-web/src/hooks/useAppState.local-runtime.tsx)
- [GraphCanvas.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/GraphCanvas.tsx)
- [graph-adapter.ts](/F:/AVmatrix-main/avmatrix-web/src/lib/graph-adapter.ts)
- [useSigma.ts](/F:/AVmatrix-main/avmatrix-web/src/hooks/useSigma.ts)

## Workstream A. Instrumentation First

Add timing and counters before changing behavior.

Required server timings:

- repo resolution
- LadybugDB open time
- per-node-table query/stream time
- relationship query/stream time
- rows streamed per table
- bytes written and chunk count
- time to first byte
- total `/api/graph` duration

Required client timings:

- request start to first byte
- NDJSON decode/parse time
- node count and relationship count parsed
- time to build `KnowledgeGraph`
- time to `setGraph`
- graph adapter conversion time
- Sigma graph creation/update time
- first visible render time
- layout stabilization time
- memory usage where available

Outcome:

- baseline report for small, medium, and large repos
- no behavior change yet

## Workstream B. API Graph Streaming

Possible optimizations:

- tune NDJSON chunk size and flush frequency
- avoid unnecessary per-row object churn when mapping DB rows to stream records
- measure whether node table order affects time to first useful render
- keep `includeContent=false` as the default for graph view
- verify relationship streaming does not block after all nodes when a progressive UI path is added
- ensure disconnect cancellation stops DB streaming promptly

Rules:

- `/api/graph` must remain read-only
- server must not write to LadybugDB
- response shape for existing callers must remain compatible

Validation:

- same node count and relationship count
- same node IDs and relationship fields
- same behavior for `includeContent=true`
- client disconnect does not leave long-running stream work

## Workstream C. Client NDJSON Parsing

Possible optimizations:

- parse incrementally without waiting for the entire response before updating load state
- batch parsed records before touching React state
- avoid building large intermediate strings when possible
- move heavy parse/build work to a Web Worker if profiling shows main-thread blocking
- keep progress accurate using records/bytes when total content length is unavailable

Rules:

- do not mutate graph semantics during parse
- avoid per-record React state updates
- keep abort behavior working when users switch repos

Validation:

- same final `KnowledgeGraph`
- no duplicate nodes or relationships
- repo switch cancellation remains correct

## Workstream D. Graph Construction and Derived Indexes

Possible optimizations:

- build `KnowledgeGraph` in batches or off-main-thread if needed
- precompute node maps, relationship maps, and type counts once
- avoid repeated full-array scans in panels that only need stable derived indexes
- keep file tree/search/process panels reading from shared derived structures where safe

Rules:

- preserve existing `KnowledgeGraph` public behavior
- do not change analyze-generated IDs or relationship directions

Validation:

- same node/edge counts in UI
- same file tree
- same search results
- same process/community panels

## Workstream E. Sigma Layout and Rendering

Possible optimizations:

- profile `graph-adapter` mapping and initial position calculation
- reduce repeated conversion from `KnowledgeGraph` to Sigma graph
- batch Sigma node/edge additions
- defer expensive visual styling until the base graph is visible
- tune ForceAtlas/layout start conditions without changing graph data
- measure edge visibility and reducer cost on large graphs

Rules:

- rendering optimization may change visual timing, not graph meaning
- any simplified initial view must be clearly a view mode over the full graph, not missing data

Validation:

- graph becomes visible faster
- interactions remain correct: select node, search, filter, hop range, process view
- no layout thrash or repeated full graph rebuilds on common UI actions

## Workstream F. Optional Progressive View Mode

This is optional and must be designed explicitly.

Possible approach:

- stream and render structural nodes first
- add symbol nodes in batches
- add expensive relationship categories later
- show a clear loading/progressive state until the complete graph is loaded

Rules:

- the final loaded graph must match the complete graph
- progressive mode must not affect CLI/MCP/analyze output
- do not persist progressive view state as canonical graph data

Validation:

- final graph equals baseline
- user can tell when loading is partial vs complete

## Correctness Guard

Every optimization must compare before/after behavior.

Minimum checks:

- node count
- relationship count
- node IDs, labels, names, paths
- relationship `type/source/target/confidence/reason/step`
- selected representative nodes and neighborhoods
- search results
- file tree output
- process/community panel output
- memory and time metrics

## Suggested Implementation Order

1. Add `/api/graph` and client graph-load timing.
2. Measure server DB query/stream time versus client parse/build/render time.
3. Tune API streaming chunks and cancellation.
4. Optimize client parse/build path.
5. Optimize graph adapter and Sigma batch construction.
6. Consider progressive view mode only after measuring the baseline.

## Validation Commands

- `cd avmatrix && npx tsc --noEmit`
- `cd avmatrix-web && npx tsc -b --noEmit`
- targeted web graph-load benchmark command to be added with instrumentation

## Non-Goals

- full analyze performance
- parse/resolve/crossFile/MRO/community/process pipeline optimization
- LadybugDB load optimization during analyze
- FTS or embedding generation optimization
- re-analyze / delta analyze
- changing relationship semantics
- replacing graph link logic
