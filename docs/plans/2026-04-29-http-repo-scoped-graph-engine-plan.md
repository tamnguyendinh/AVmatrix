# HTTP Repo-Scoped Graph Engine Replacement Plan

Date: 2026-04-29  
Status: Draft  
Scope: `avmatrix/src/server/`, `avmatrix/src/core/lbug/`, `avmatrix-web/src/`, related tests/docs

## Goal

Replace the **HTTP repo-switch / graph-load sub-engine** that currently sits under the Web UI, so switching repos from the dropdown becomes stable and smooth without depending on the old process-global DB retargeting path.

This plan does **not** replace AVmatrix as a local MCP.

This plan does **not** replace the whole tool.

This plan replaces one broken sub-engine:

```text
HTTP repo attach / graph load / repo-scoped read-write execution path
```

## What is being replaced

The part being replaced is the current HTTP path that behaves like:

```text
resolve repo
-> compute lbugPath
-> switch module-global LadybugDB state to that dbPath
-> run graph/query/search/embed against the newly active DB
```

In current code, that path is centered around:

- [api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts:729)
- [api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts:798)
- [api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts:824)
- [api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts:1411)
- [lbug-adapter.ts](F:/AVmatrix-main/avmatrix/src/core/lbug/lbug-adapter.ts:148)
- [withLbugDb()](F:/AVmatrix-main/avmatrix/src/core/lbug/lbug-adapter.ts:234)
- [doInitLbug()](F:/AVmatrix-main/avmatrix/src/core/lbug/lbug-adapter.ts:283)

The current engine has:

```ts
let db: lbug.Database | null = null;
let conn: lbug.Connection | null = null;
let currentDbPath: string | null = null;
```

That is the broken sub-engine.

## What is NOT being replaced

These are explicitly out of scope:

- AVmatrix's role as a **local MCP**
- `avmatrix mcp` as the canonical local MCP entrypoint
- Codex / Claude Code talking directly to AVmatrix local MCP
- the full codebase
- the graph schema itself
- the whole Web UI
- the whole CLI

This plan is **not**:

- a Web UI-only patch
- a dropdown-only patch
- a whole-tool rewrite
- a replacement of MCP with HTTP

## Why this specific sub-engine must be replaced

The current HTTP path is built on a design that is structurally fragile under multi-repo switching:

### Current HTTP read path

Example from `/api/graph`:

- resolve repo in [api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts:731)
- derive `lbugPath`
- call `withLbugDb(lbugPath, ...)` in [api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts:763)

### Current HTTP write path

Example from `/api/embed`:

- resolve repo in [api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts:1414)
- derive `lbugPath`
- call `withLbugDb(lbugPath, ...)` in [api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts:1452)

### Why this is wrong

That means repo B requests are served by **retargeting** a process-global DB context that may previously have been serving repo A.

The failure mode is not surprising:

```text
repo A read/write lifecycle is active
-> repo B request retargets the same global native state
-> graph for repo B can fail, hang, or crash transport
```

The broken design is not the dropdown itself.  
The broken design is the HTTP sub-engine under it.

## Best-practice basis for the replacement

This plan is based on an already-proven pattern that exists **inside AVmatrix today**:

### Pattern 1: explicit repo binding

MCP local backend resolves repo explicitly:

- [LocalBackend.resolveRepo()](F:/AVmatrix-main/avmatrix/src/mcp/local/local-backend.ts:342)

It does not depend on a hidden, mutable "active repo".

### Pattern 2: repo-scoped DB pool

MCP local backend uses:

- [pool-adapter.ts](F:/AVmatrix-main/avmatrix/src/core/lbug/pool-adapter.ts:6)

That adapter:

- keys DB state by `repoId`
- manages one pool per repo
- supports concurrent query execution via multiple connections
- does **not** retarget one global `currentDbPath`

### Pattern 3: transport-adapter separation

MCP local backend is not itself the database engine.  
It is a transport-facing layer that calls repo-scoped execution primitives.

That is the design HTTP should move toward.

## Architectural target

### One sentence summary

Replace:

```text
HTTP route -> global DB retarget -> query
```

with:

```text
HTTP route -> explicit repo binding -> repo-scoped executor -> graph/query/embed service
```

### Current architecture

```text
Web UI
-> /api/repo
-> /api/graph
-> api.ts resolves repo
-> withLbugDb(lbugPath)
-> lbug-adapter switches global db/conn/currentDbPath
-> graph/query/embed run on whichever DB is currently active
```

### Target architecture

```text
Web UI
-> /api/repo
-> /api/graph
-> HTTP repo runtime adapter resolves explicit RepoHandle
-> HTTP repo runtime gets repo-scoped executor for repo.id
-> graph/query/embed service runs only inside that repo-scoped executor
-> repo A and repo B do not retarget one shared active DB slot
```

## Exact replacement mapping

### A. Repo resolution

#### Current

- HTTP has a route-local `resolveRepo()` inside [api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts:514)
- MCP has its own repo resolution logic in [local-backend.ts](F:/AVmatrix-main/avmatrix/src/mcp/local/local-backend.ts:342)

#### Replace with

Create a shared repo-handle resolver for transport adapters:

- `avmatrix/src/runtime/repo-handle.ts`
- `avmatrix/src/runtime/repo-handle-resolver.ts`

Suggested types:

```ts
export interface RepoRuntimeHandle {
  id: string;
  name: string;
  repoPath: string;
  storagePath: string;
  lbugPath: string;
  indexedAt: string;
  lastCommit: string;
  stats?: unknown;
}
```

```ts
export interface RepoBinding {
  repo?: string;
  repoPath?: string;
}
```

#### Reuse

Reuse the **resolution semantics** from:

- [LocalBackend.resolveRepo()](F:/AVmatrix-main/avmatrix/src/mcp/local/local-backend.ts:342)
- [LocalBackend.resolveRepoFromCache()](F:/AVmatrix-main/avmatrix/src/mcp/local/local-backend.ts:380)

Do **not** reuse the entire `LocalBackend` class.

#### Why not reuse the whole class

`LocalBackend` is MCP-tool-oriented:

- tool dispatch
- tool result formatting
- MCP caches/context assumptions

HTTP routes should not depend on MCP tool orchestration just to resolve a repo.

## B. HTTP read execution substrate

### Current

HTTP read endpoints use:

- `withLbugDb(lbugPath, ...)`
- global `executeQuery()`
- global `streamQuery()`

Examples:

- `/api/graph` in [api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts:729)
- `/api/query` in [api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts:798)
- `/api/search` in [api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts:824)

### Replace with

Introduce a repo-scoped read executor abstraction:

- `avmatrix/src/server/repo-runtime/repo-read-executor.ts`
- `avmatrix/src/server/repo-runtime/pool-repo-read-executor.ts`

Suggested contract:

```ts
export interface RepoReadExecutor {
  ensureReady(repo: RepoRuntimeHandle): Promise<void>;
  query(repo: RepoRuntimeHandle, cypher: string): Promise<any[]>;
  queryParams(repo: RepoRuntimeHandle, cypher: string, params: Record<string, unknown>): Promise<any[]>;
  stream(
    repo: RepoRuntimeHandle,
    cypher: string,
    onRow: (row: any) => Promise<void>,
    signal?: AbortSignal,
  ): Promise<void>;
  close?(repo: RepoRuntimeHandle): Promise<void>;
}
```

### Reuse

Reuse directly from `pool-adapter.ts`:

- [initLbug(repoId, dbPath)](F:/AVmatrix-main/avmatrix/src/core/lbug/pool-adapter.ts:227)
- [executeQuery(repoId, cypher)](F:/AVmatrix-main/avmatrix/src/core/lbug/pool-adapter.ts:512)
- [executeParameterized(repoId, cypher, params)](F:/AVmatrix-main/avmatrix/src/core/lbug/pool-adapter.ts:543)
- [closeLbug(repoId?)](F:/AVmatrix-main/avmatrix/src/core/lbug/pool-adapter.ts:580)
- [isLbugReady(repoId)](F:/AVmatrix-main/avmatrix/src/core/lbug/pool-adapter.ts:599)
- [touchRepo(repoId)](F:/AVmatrix-main/avmatrix/src/core/lbug/pool-adapter.ts:91)

### Must be written new

`pool-adapter` currently does **not** provide:

- repo-scoped streaming row iteration for HTTP graph stream
- a transport-neutral HTTP-facing executor wrapper

So these must be added:

1. `streamQuery(repoId, cypher, onRow, signal?)` in `pool-adapter.ts`, or
2. a new wrapper in `pool-repo-read-executor.ts` that uses pool connections and streams rows safely

This is one of the most important new abstractions in the plan.

## C. Graph load engine

### Current

Graph materialization is embedded inside `api.ts`:

- [buildGraph()](F:/AVmatrix-main/avmatrix/src/server/api.ts:227)
- [streamGraphNdjson()](F:/AVmatrix-main/avmatrix/src/server/api.ts:321)

Both functions currently depend on global query primitives:

- [executeQuery() call](F:/AVmatrix-main/avmatrix/src/server/api.ts:245)
- [streamQuery() call](F:/AVmatrix-main/avmatrix/src/server/api.ts:330)

### Replace with

Extract graph logic into repo-scoped services:

- `avmatrix/src/server/repo-runtime/graph-read-service.ts`
- `avmatrix/src/server/repo-runtime/graph-stream-service.ts`

Suggested contracts:

```ts
export interface RepoGraphReadService {
  buildGraph(repo: RepoRuntimeHandle, options: { includeContent: boolean }): Promise<{
    nodes: GraphNode[];
    relationships: GraphRelationship[];
  }>;
}
```

```ts
export interface RepoGraphStreamService {
  streamGraphNdjson(
    repo: RepoRuntimeHandle,
    res: express.Response,
    options: { includeContent: boolean; signal?: AbortSignal },
  ): Promise<void>;
}
```

### Reuse

Reuse almost all **query-shape and mapping logic** already in `api.ts`:

- [GRAPH_RELATIONSHIP_QUERY](F:/AVmatrix-main/avmatrix/src/server/api.ts:253)
- [getNodeQuery()](F:/AVmatrix-main/avmatrix/src/server/api.ts:259)
- [mapGraphNodeRow()](F:/AVmatrix-main/avmatrix/src/server/api.ts:287)
- [mapGraphRelationshipRow()](F:/AVmatrix-main/avmatrix/src/server/api.ts:311)

These are not the broken part.  
The broken part is the execution substrate under them.

### Must be written new

The signatures must change so graph services depend on `RepoReadExecutor`, not global `executeQuery/streamQuery`.

## D. HTTP route adapter

### Current

`api.ts` owns too much:

- repo resolution
- DB lifecycle coupling
- graph build orchestration
- search execution
- embed execution
- route formatting

### Replace with

Make `api.ts` only a route adapter over a small HTTP repo runtime facade:

- `avmatrix/src/server/repo-runtime/http-repo-runtime.ts`

Suggested facade:

```ts
export interface HttpRepoRuntime {
  resolve(binding: RepoBinding): Promise<RepoRuntimeHandle>;
  buildGraph(repo: RepoRuntimeHandle, options: { includeContent: boolean }): Promise<GraphPayload>;
  streamGraph(repo: RepoRuntimeHandle, res: express.Response, options: StreamOptions): Promise<void>;
  executeCypher(repo: RepoRuntimeHandle, cypher: string): Promise<any[]>;
  search(repo: RepoRuntimeHandle, request: SearchRequest): Promise<any[]>;
  startEmbed(repo: RepoRuntimeHandle): Promise<JobHandle>;
}
```

### Files in HTTP path that must be changed

These current files are in the replacement blast radius:

- [avmatrix/src/server/api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts)
- [avmatrix/src/core/lbug/lbug-adapter.ts](F:/AVmatrix-main/avmatrix/src/core/lbug/lbug-adapter.ts)
- [avmatrix/src/core/lbug/pool-adapter.ts](F:/AVmatrix-main/avmatrix/src/core/lbug/pool-adapter.ts)

These client-side files may need small API-alignment changes depending on whether route contracts stay identical:

- [avmatrix-web/src/services/backend-client.ts](F:/AVmatrix-main/avmatrix-web/src/services/backend-client.ts)
- [avmatrix-web/src/hooks/useAppState.local-runtime.tsx](F:/AVmatrix-main/avmatrix-web/src/hooks/useAppState.local-runtime.tsx)

## E. Search/query/file endpoints

### Current

These HTTP routes also use the old global path:

- `/api/query` in [api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts:798)
- `/api/search` in [api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts:824)
- `/api/file` path later in `api.ts`

### Replace with

Move them to repo-scoped services:

- `avmatrix/src/server/repo-runtime/repo-query-service.ts`
- `avmatrix/src/server/repo-runtime/repo-search-service.ts`
- `avmatrix/src/server/repo-runtime/repo-file-service.ts`

### Reuse

Reuse:

- query text and response shaping logic from current `api.ts`
- search ranking logic already in current imports
- repo-scoped query execution pattern from MCP local-backend:
  - [executeQuery(repo.id, ...)](F:/AVmatrix-main/avmatrix/src/mcp/local/local-backend.ts:999)

### Must be written new

The HTTP-facing services and their repo-scoped contracts.

## F. Embed / analyze isolation

### Current

`/api/embed` uses:

- [withLbugDb(lbugPath, ...)](F:/AVmatrix-main/avmatrix/src/server/api.ts:1452)
- [fetchExistingEmbeddingHashes(executeQuery)](F:/AVmatrix-main/avmatrix/src/server/api.ts:1457)
- [runEmbeddingPipeline(executeQuery, executeWithReusedStatement, ...)](F:/AVmatrix-main/avmatrix/src/server/api.ts:1464)

This is still built on the old global mutable adapter.

### Replace with

Do **not** move write work onto the current read-only `pool-adapter`.

Instead create a dedicated write-side abstraction:

- `avmatrix/src/server/repo-runtime/repo-write-runtime.ts`
- `avmatrix/src/server/repo-runtime/repo-embed-service.ts`
- optionally `avmatrix/src/server/repo-runtime/repo-write-worker.ts`

Suggested direction:

- one repo-scoped write session at a time per repo
- isolate write-side native lifecycle from HTTP read-side repo pools
- communicate job progress back to HTTP routes via current `JobManager`

### Reuse

Reuse:

- [JobManager](F:/AVmatrix-main/avmatrix/src/server/api.ts:491)
- progress/SSE mechanics already in `api.ts`
- embedding pipeline logic itself:
  - [runEmbeddingPipeline](F:/AVmatrix-main/avmatrix/src/server/api.ts:1453)

### Must be written new

New write-side runtime boundary.

This is critical because:

- `pool-adapter` is intentionally read-only
- `lbug-adapter` is currently global and mutable
- the write path is where cross-repo poisoning risk is highest

## What can be reused directly vs what must be extracted

### Reuse directly

These can be reused largely as-is:

1. `pool-adapter.ts` read-side pool and connection management  
2. `LocalBackend.resolveRepo()` semantics  
3. graph query strings and row-mapping helpers in `api.ts`  
4. existing HTTP route URLs if we keep compatibility  
5. `JobManager` and SSE progress plumbing  

### Extract then reuse

These should be extracted from their current locations, then shared:

1. repo handle shape from [local-backend.ts](F:/AVmatrix-main/avmatrix/src/mcp/local/local-backend.ts:187)  
2. repo resolution cache/refresh logic from `LocalBackend`  
3. graph query/mapping helpers from `api.ts`  

### Must be written new

These do not exist yet and are required:

1. `RepoRuntimeHandle` canonical type  
2. `RepoReadExecutor` abstraction  
3. `PoolRepoReadExecutor` HTTP implementation  
4. repo-scoped streaming query primitive for HTTP graph streaming  
5. `RepoGraphReadService`  
6. `RepoGraphStreamService`  
7. `HttpRepoRuntime` facade  
8. write-side isolated runtime/service for embed  

## What should NOT be reused

Do **not** reuse `LocalBackend` itself as the HTTP runtime.

Reasons:

- it is MCP tool shaped
- it contains MCP-specific dispatch and formatting behavior
- it assumes read-oriented tool execution
- it is not the right place to host HTTP graph streaming semantics

The right move is:

```text
reuse MCP's repo-scoped patterns
not MCP's whole transport class
```

## External API strategy

### Phase 1 target

Keep the current Web-facing endpoints stable if possible:

- `/api/repos`
- `/api/repo`
- `/api/graph`
- `/api/query`
- `/api/search`
- `/api/embed`

That lets us replace the broken sub-engine under the routes without forcing a large Web rewrite in the first pass.

### Optional later phase

If needed, add explicit repo-session endpoints later.

But they are not required to replace the broken sub-engine in the first pass.

## Implementation phases

### Phase 0: Architecture lock

Deliverables:

- confirm this plan only replaces the HTTP repo/graph sub-engine
- confirm MCP local remains canonical and unchanged in role
- confirm first pass keeps current HTTP route surface stable

### Phase 1: Shared repo-handle resolver

Create:

- `avmatrix/src/runtime/repo-handle.ts`
- `avmatrix/src/runtime/repo-handle-resolver.ts`

Refactor:

- `api.ts` repo resolution to use shared resolver
- `LocalBackend` to reuse the same resolver or the same resolver core

### Phase 2: Repo-scoped read executor

Create:

- `avmatrix/src/server/repo-runtime/repo-read-executor.ts`
- `avmatrix/src/server/repo-runtime/pool-repo-read-executor.ts`

Extend:

- `pool-adapter.ts` with repo-scoped streaming support

### Phase 3: Graph engine extraction

Create:

- `avmatrix/src/server/repo-runtime/graph-read-service.ts`
- `avmatrix/src/server/repo-runtime/graph-stream-service.ts`

Refactor:

- move `buildGraph()` logic out of `api.ts`
- move `streamGraphNdjson()` logic out of `api.ts`
- change both to depend on `RepoReadExecutor`

### Phase 4: HTTP route migration

Create:

- `avmatrix/src/server/repo-runtime/http-repo-runtime.ts`

Refactor `api.ts` routes to call:

- resolver
- graph services
- query/search services
- embed service

At this point, `api.ts` should stop calling `withLbugDb()` for HTTP read paths.

### Phase 5: Embed write isolation

Create:

- `avmatrix/src/server/repo-runtime/repo-write-runtime.ts`
- `avmatrix/src/server/repo-runtime/repo-embed-service.ts`

Refactor `/api/embed` to stop using the old global `withLbugDb()` path directly in the transport process.

### Phase 6: Remove old HTTP dependency on global DB retargeting

After parity:

- remove HTTP read reliance on `withLbugDb()`
- keep `lbug-adapter.ts` only where still justified
- clearly separate legacy/global adapter from repo-scoped HTTP path

## Validation and test plan

### Unit / integration

Add or update:

- `avmatrix/test/unit/api-graph-streaming.test.ts`
- new tests for repo-handle resolver
- new tests for repo-scoped read executor
- new tests for graph stream service
- new tests for embed isolation service

### Web E2E

Add or expand:

- [avmatrix-web/e2e/repo-switching.spec.ts](F:/AVmatrix-main/avmatrix-web/e2e/repo-switching.spec.ts)
- [avmatrix-web/e2e/multi-repo-scoping.spec.ts](F:/AVmatrix-main/avmatrix-web/e2e/multi-repo-scoping.spec.ts)

Required stress cases:

1. A -> B -> C -> A repeated dropdown switching  
2. repo A embed active while repo B graph loads  
3. repo B graph load must not corrupt repo A  
4. graph must render for the target repo without backend restart  

### MCP regression

Even though MCP is not being replaced, run regression coverage to ensure:

- `avmatrix mcp` still resolves repo explicitly
- MCP local-backend behavior does not regress while shared resolver code is extracted

## Acceptance criteria

The replacement is only complete when:

1. HTTP graph load no longer depends on process-global `currentDbPath`
2. HTTP query/search/file routes no longer depend on process-global DB retargeting
3. repo switch in Web UI can repeatedly load target graph without transport restart
4. repo A embed does not poison repo B graph load
5. MCP local remains intact and still talks directly to Codex / Claude Code
6. AVmatrix does not require a whole-tool rewrite to achieve the fix

## Recommendation

Do not spend more time hardening the old HTTP `withLbugDb(currentDbPath)` engine for repo switching.

Replace that sub-engine with:

```text
explicit repo handle
-> repo-scoped read executor
-> extracted graph/query/search services
-> isolated write runtime for embed/analyze
```

That is the smallest replacement that changes the broken class of design without replacing AVmatrix as a whole.
