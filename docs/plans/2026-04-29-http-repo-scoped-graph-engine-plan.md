# HTTP Repo-Scoped Graph Engine Replacement Plan

Date: 2026-04-29  
Status: Draft  
Scope: `avmatrix/src/server/`, `avmatrix/src/runtime/`, `avmatrix/src/core/lbug/`, `avmatrix-web/src/`, `avmatrix-shared/`, related tests/docs

## Goal

Replace the **HTTP repo-switch / graph-load sub-engine** that currently sits under the Web UI, so switching repos from the dropdown becomes stable and smooth without depending on the old process-global DB retargeting path.

This plan does **not** replace AVmatrix as a local MCP.

This plan does **not** replace the whole tool.

This plan replaces one broken sub-engine:

```text
HTTP repo attach / graph load / repo-scoped read execution path
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

## Architecture constraints

The replacement must obey these constraints:

1. The new engine belongs to shared runtime/core, not to HTTP.
2. `api.ts` stays an adapter, in line with the existing local-runtime direction in:
   - [2026-04-20-convert-all-to-local.md](F:/AVmatrix-main/docs/plans/2026-04-20-convert-all-to-local.md:50)
   - [2026-04-20-convert-all-to-local.md](F:/AVmatrix-main/docs/plans/2026-04-20-convert-all-to-local.md:54)
3. The plan must not create a second repo-binding/type family parallel to the runtime/session types that already exist in:
   - [session.ts](F:/AVmatrix-main/avmatrix-shared/src/session.ts:24)
   - [runtime-controller.ts](F:/AVmatrix-main/avmatrix/src/runtime/runtime-controller.ts:154)
4. Runtime/core contracts must stay transport-neutral. `express.Response`, route objects, and SSE formatting stay in HTTP adapter code.
5. The plan must define one canonical repo identity / pool key for repo-scoped execution. Do not let HTTP, runtime, and pool layers invent different keys.
6. First rollout deliverable is the HTTP read-path replacement that serves dropdown repo switching and graph load.
7. Write-side isolation for embed/analyze is allowed only as a gated follow-up phase, not as an excuse to turn this plan into a half-tool rewrite.
8. First rollout must preserve current Web route contracts, including `repo` query-param compatibility, and map them inside the adapter to shared repo-binding types.

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
HTTP route -> explicit repo binding -> repo-scoped executor -> graph/query/search/file read service
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
-> HTTP repo runtime adapter maps `repo` query param to explicit SessionRepoBinding
-> HTTP repo runtime gets repo-scoped executor for repo.id
-> graph/query/search/file read service runs only inside that repo-scoped executor
-> repo A and repo B do not retarget one shared active DB slot
```

## Exact replacement mapping

### A. Repo resolution

#### Current

- HTTP has a route-local `resolveRepo()` inside [api.ts](F:/AVmatrix-main/avmatrix/src/server/api.ts:514)
- MCP has its own repo resolution logic in [local-backend.ts](F:/AVmatrix-main/avmatrix/src/mcp/local/local-backend.ts:342)

#### Replace with

Create a shared repo resolver core under `src/runtime/`, extracted from the current runtime/session path instead of inventing a second binding model:

- `avmatrix/src/runtime/repo-resolver.ts`
- optionally `avmatrix/src/runtime/repo-descriptor.ts`

Canonical binding types should continue to come from:

- [SessionRepoBinding](F:/AVmatrix-main/avmatrix-shared/src/session.ts:24)
- [ResolvedSessionRepo](F:/AVmatrix-main/avmatrix-shared/src/session.ts:29)

If the HTTP graph engine needs more metadata than `ResolvedSessionRepo` currently exposes, prefer one companion descriptor that extends the existing resolved-repo model instead of creating a parallel family:

```ts
export interface IndexedRepoDescriptor extends ResolvedSessionRepo {
  repoId: string;
  storagePath: string;
  lbugPath: string;
  indexedAt: string;
  lastCommit: string;
  stats?: unknown;
}
```

`repoId` is the canonical repo identity for repo-scoped execution and pool lookup. The first rollout must not leave this implicit.

#### Reuse

Reuse the **resolution semantics and existing runtime path** from:

- [LocalBackend.resolveRepo()](F:/AVmatrix-main/avmatrix/src/mcp/local/local-backend.ts:342)
- [LocalBackend.resolveRepoFromCache()](F:/AVmatrix-main/avmatrix/src/mcp/local/local-backend.ts:380)
- [RuntimeController.resolveRepo()](F:/AVmatrix-main/avmatrix/src/runtime/runtime-controller.ts:154)

Do **not** duplicate repo resolution one more time in HTTP. Extract a shared resolver that both HTTP and runtime/session code can call.

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

Introduce a repo-scoped read executor abstraction in shared runtime/core:

- `avmatrix/src/runtime/repo-runtime/repo-read-executor.ts`
- `avmatrix/src/runtime/repo-runtime/pool-repo-read-executor.ts`

Suggested contract:

```ts
export interface RepoReadExecutor {
  ensureReady(repo: IndexedRepoDescriptor): Promise<void>;
  query(repo: IndexedRepoDescriptor, cypher: string): Promise<any[]>;
  queryParams(
    repo: IndexedRepoDescriptor,
    cypher: string,
    params: Record<string, unknown>,
  ): Promise<any[]>;
  stream(
    repo: IndexedRepoDescriptor,
    cypher: string,
    onRow: (row: any) => Promise<void>,
    signal?: AbortSignal,
  ): Promise<void>;
  close?(repo: IndexedRepoDescriptor): Promise<void>;
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

Extract graph logic into repo-scoped services in shared runtime/core:

- `avmatrix/src/runtime/repo-runtime/graph-read-service.ts`
- `avmatrix/src/runtime/repo-runtime/graph-stream-service.ts`

Transport-neutral streaming support should live in runtime/core, for example:

```ts
export interface GraphStreamSink {
  write(chunk: string): Promise<void>;
  end(): Promise<void>;
}
```

Suggested contracts:

```ts
export interface RepoGraphReadService {
  buildGraph(repo: IndexedRepoDescriptor, options: { includeContent: boolean }): Promise<{
    nodes: GraphNode[];
    relationships: GraphRelationship[];
  }>;
}
```

```ts
export interface RepoGraphStreamService {
  streamGraphNdjson(
    repo: IndexedRepoDescriptor,
    sink: GraphStreamSink,
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

Make `api.ts` only a route adapter over a runtime-core facade plus a thin server adapter:

- runtime/core facade:
  - `avmatrix/src/runtime/repo-runtime/repo-runtime-core.ts`
- optional thin HTTP adapter helper:
  - `avmatrix/src/server/http-repo-runtime-adapter.ts`

Suggested facade:

```ts
export interface RepoRuntimeCore {
  resolve(binding: SessionRepoBinding): Promise<IndexedRepoDescriptor>;
  buildGraph(repo: IndexedRepoDescriptor, options: { includeContent: boolean }): Promise<GraphPayload>;
  streamGraph(
    repo: IndexedRepoDescriptor,
    sink: GraphStreamSink,
    options: StreamOptions,
  ): Promise<void>;
  executeCypher(repo: IndexedRepoDescriptor, cypher: string): Promise<any[]>;
  search(repo: IndexedRepoDescriptor, request: SearchRequest): Promise<any[]>;
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

Move them to repo-scoped services in runtime/core:

- `avmatrix/src/runtime/repo-runtime/repo-query-service.ts`
- `avmatrix/src/runtime/repo-runtime/repo-search-service.ts`
- `avmatrix/src/runtime/repo-runtime/repo-file-service.ts`

### Reuse

Reuse:

- query text and response shaping logic from current `api.ts`
- search ranking logic already in current imports
- repo-scoped query execution pattern from MCP local-backend:
  - [executeQuery(repo.id, ...)](F:/AVmatrix-main/avmatrix/src/mcp/local/local-backend.ts:999)

### Must be written new

The HTTP-facing services and their repo-scoped contracts.

## F. Embed / analyze isolation

This section is a **gated follow-up**, not the primary deliverable of the first rollout.

The first rollout must land the HTTP read-path replacement for dropdown repo switching and graph load.

Write isolation should only be pulled into the same implementation wave if the read-path replacement still leaves the same bug class alive.

### Current

`/api/embed` uses:

- [withLbugDb(lbugPath, ...)](F:/AVmatrix-main/avmatrix/src/server/api.ts:1452)
- [fetchExistingEmbeddingHashes(executeQuery)](F:/AVmatrix-main/avmatrix/src/server/api.ts:1457)
- [runEmbeddingPipeline(executeQuery, executeWithReusedStatement, ...)](F:/AVmatrix-main/avmatrix/src/server/api.ts:1464)

This is still built on the old global mutable adapter.

### Replace with

Do **not** move write work onto the current read-only `pool-adapter`.

Instead create a dedicated write-side abstraction in runtime/core:

- `avmatrix/src/runtime/repo-runtime/repo-write-runtime.ts`
- `avmatrix/src/runtime/repo-runtime/repo-embed-service.ts`
- optionally `avmatrix/src/runtime/repo-runtime/repo-write-worker.ts`

Suggested direction:

- one repo-scoped write session at a time per repo
- isolate write-side native lifecycle from HTTP read-side repo pools
- publish progress through a runtime-neutral progress/event interface; HTTP may adapt that into `JobManager` / SSE, but runtime/core must not depend on `api.ts`

### Reuse

Reuse:

- progress/SSE mechanics already in `api.ts`
- embedding pipeline logic itself:
  - [runEmbeddingPipeline](F:/AVmatrix-main/avmatrix/src/server/api.ts:1453)

If `JobManager` remains useful, only reuse it at the HTTP adapter layer, not as a runtime-core dependency.

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

1. repo descriptor fields from [local-backend.ts](F:/AVmatrix-main/avmatrix/src/mcp/local/local-backend.ts:187), but attached to the existing shared repo-binding model  
2. repo resolution cache/refresh logic from `LocalBackend` and `RuntimeController`  
3. graph query/mapping helpers from `api.ts`  

### Must be written new

These do not exist yet and are required:

1. `IndexedRepoDescriptor` companion type, only if extending `ResolvedSessionRepo` is not enough  
2. `RepoReadExecutor` abstraction  
3. `PoolRepoReadExecutor` runtime-core implementation  
4. repo-scoped streaming query primitive for HTTP graph streaming  
5. transport-neutral graph stream sink / output contract  
6. `RepoGraphReadService`  
7. `RepoGraphStreamService`  
8. `RepoRuntimeCore` facade  
9. write-side isolated runtime/service for embed  

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

Compatibility rule for rollout 1:

- keep `repo` as the external HTTP query parameter on current Web routes
- map `repo` inside the HTTP adapter to `SessionRepoBinding`
- do not force Web UI to adopt a new binding payload just to land the runtime-core replacement

### Optional later phase

If needed, add explicit repo-session endpoints later by extending the existing session/runtime surface rather than inventing a second unrelated session API.

But they are not required to replace the broken sub-engine in the first pass.

## Implementation phases

### Phase 0: Architecture lock

Deliverables:

- confirm this plan only replaces the HTTP repo/graph sub-engine
- confirm MCP local remains canonical and unchanged in role
- confirm first pass keeps current HTTP route surface stable
- confirm canonical repo identity / pool key (`repoId`) and where it enters the runtime path
- confirm runtime/core stream contracts do not depend on Express objects

### Phase 1: Shared repo resolver core

Create:

- `avmatrix/src/runtime/repo-resolver.ts`
- optionally `avmatrix/src/runtime/repo-descriptor.ts`

Refactor:

- `api.ts` repo resolution to use shared resolver core
- `RuntimeController` to reuse the same resolver core
- `LocalBackend` to reuse the same resolver core where practical

### Phase 2: Repo-scoped read executor

Create:

- `avmatrix/src/runtime/repo-runtime/repo-read-executor.ts`
- `avmatrix/src/runtime/repo-runtime/pool-repo-read-executor.ts`

Extend:

- `pool-adapter.ts` with repo-scoped streaming support

### Phase 3: Graph engine extraction

Create:

- `avmatrix/src/runtime/repo-runtime/graph-read-service.ts`
- `avmatrix/src/runtime/repo-runtime/graph-stream-service.ts`

Refactor:

- move `buildGraph()` logic out of `api.ts`
- move `streamGraphNdjson()` logic out of `api.ts`
- change both to depend on `RepoReadExecutor`

### Phase 4: HTTP route migration

Create:

- runtime/core facade:
  - `avmatrix/src/runtime/repo-runtime/repo-runtime-core.ts`
- optional server adapter helper:
  - `avmatrix/src/server/http-repo-runtime-adapter.ts`

Refactor `api.ts` routes to call:

- resolver
- graph services
- query/search services

At this point, `api.ts` should stop calling `withLbugDb()` for HTTP read paths.

### Phase 5: Embed write isolation (gated follow-up)

Create:

- `avmatrix/src/runtime/repo-runtime/repo-write-runtime.ts`
- `avmatrix/src/runtime/repo-runtime/repo-embed-service.ts`

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
- new tests for shared repo resolver
- new tests for repo-scoped read executor
- new tests for graph stream service
- new tests for HTTP adapter mapping `repo` -> `SessionRepoBinding`
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
4. MCP local remains intact and still talks directly to Codex / Claude Code
5. the replacement does not create a second repo-binding/type family beside `SessionRepoBinding` / `ResolvedSessionRepo`
6. runtime/core graph streaming no longer depends on `express.Response` or route-owned transport objects
7. repo-scoped execution uses one canonical repo identity / pool key (`repoId`)
8. AVmatrix does not require a whole-tool rewrite to achieve the fix

## Recommendation

Do not spend more time hardening the old HTTP `withLbugDb(currentDbPath)` engine for repo switching.

Replace that sub-engine with:

```text
 explicit repo binding / repo descriptor
-> repo-scoped read executor
-> extracted graph/query/search services
-> transport-neutral graph stream output
-> isolated write runtime for embed/analyze only if gated follow-up is still needed
```

That is the smallest replacement that changes the broken class of design without replacing AVmatrix as a whole.
