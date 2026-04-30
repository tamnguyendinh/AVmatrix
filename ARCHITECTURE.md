# Architecture — AVmatrix

Monorepo: **CLI/MCP/HTTP backend** (`avmatrix/`) + **browser UI** (`avmatrix-web/`) + **shared contracts** (`avmatrix-shared/`) + **local launcher** (`avmatrix-launcher/`).

## Repository layout

| Path | Role |
|------|------|
| `avmatrix/` | npm package `avmatrix`: CLI, MCP server (stdio), HTTP API, ingestion pipeline, LadybugDB graph, embeddings. |
| `avmatrix-web/` | Vite + React thin client: graph explorer, repo picker/analyze UI, and Codex/Claude Code session chat. Runtime calls go through the local HTTP API from `avmatrix serve`. |
| `avmatrix-shared/` | Shared TypeScript types and constants (consumed by CLI and Web). |
| `avmatrix-launcher/` | Windows local launcher: protocol handler, packaged web static server, backend wrapper, runtime reset, and release build script. |
| `.claude/`, `avmatrix-claude-plugin/`, `avmatrix-cursor-integration/` | Agent skills and plugin metadata. |
| `eval/` | Evaluation harness and offline quality checks. |
| `.github/` | CI workflows + composite actions (`setup-avmatrix/`, `setup-avmatrix-web/`). |

## End-to-end flow: index → graph → tools

1. **Ingestion** — `analyze.ts` → `runFullAnalysis` (`run-analyze.ts`) → `runPipelineFromRepo` (`pipeline.ts`). DAG of 12 phases builds a `KnowledgeGraph` in memory, then loads into LadybugDB under `.avmatrix/`. Repo registered in `~/.avmatrix/registry.json` for MCP discovery.

2. **Persistence** — `repo-manager.ts` (paths, registry, KuzuDB cleanup). `lbug-adapter.ts` (graph load, queries, embedding batches).

3. **Runtime interfaces** — four local interfaces over the same indexed repos:
   - **MCP (stdio):** `mcp.ts` → `LocalBackend` → tools (`tools.ts`) + resources (`resources.ts`).
   - **HTTP local API:** `serve.ts` → Express (`server/api.ts`, `server/mcp-http.ts`, `server/session-bridge.ts`) for web UI and local adapters.
   - **CLI direct:** `avmatrix query|context|impact|cypher` in `tool.ts`.
   - **Launcher:** `avmatrix-launcher` starts the packaged web UI and backend wrapper on localhost.

4. **Staleness** — `staleness.ts` compares indexed `lastCommit` to `HEAD`, surfaces hints.

## Local HTTP runtime

`avmatrix serve` is a local-only backend. It binds localhost by default, allows localhost/private-network browser access, and does not introduce an AVmatrix-hosted cloud path.

| Endpoint family | Purpose | Main implementation |
|-----------------|---------|---------------------|
| `/api/info`, `/api/heartbeat` | Backend liveness and runtime metadata | `src/server/api.ts` |
| `/api/repos`, `/api/repo` | List, select, and remove indexed local repos | `src/server/api.ts`, `src/core/repo-manager.ts` |
| `/api/graph` | Load or stream graph data for an explicit repo | `src/runtime/repo-runtime/`, `src/server/graph-stream-http.ts` |
| `/api/query`, `/api/search`, `/api/file`, `/api/grep`, `/api/process*`, `/api/cluster*` | Code search, file access, and graph-derived views | `src/server/api.ts` |
| `/api/local/folder-picker` | Open an OS folder picker from the local backend and return an absolute path | `src/server/local-folder-picker.ts` |
| `/api/analyze`, `/api/embed` | Local background jobs for indexing and embeddings | `src/server/jobs/`, `src/core/run-analyze.ts`, `src/core/embeddings/` |
| `/api/mcp` | MCP-over-HTTP bridge for local clients | `src/server/mcp-http.ts`, `src/mcp/local/local-backend.ts` |
| `/api/session/*` | Local session bridge for Codex/Claude Code style chat | `src/server/session-bridge.ts`, `src/runtime/runtime-controller.ts` |

The HTTP runtime still contains some legacy endpoint paths that call `src/core/lbug/lbug-adapter.ts` directly. The graph loading path has been moved to the repo-scoped runtime described below, so changing repos in the Web UI does not require retargeting one process-wide active database handle.

## Repo-scoped graph reads

Web graph loading is explicit by repo. The browser sends the target repo name/path, and the server resolves that request into a repo runtime target before reading LadybugDB.

| Layer | Responsibility |
|-------|----------------|
| `src/runtime/repo-resolver.ts` | Resolve repo name/path, assign stable runtime IDs, reject unsupported local-session bindings. |
| `src/runtime/repo-runtime/repo-read-executor.ts` | Represent `RepoReadTarget { repoId, lbugPath }` and execute reads through the LadybugDB pool keyed by `repoId`. |
| `src/runtime/repo-runtime/graph-read-service.ts` | Build or stream graph nodes and `CodeRelation` relationships from repo-scoped queries. |
| `src/server/graph-stream-http.ts` | Stream NDJSON batches to the browser and handle disconnects/backpressure. |

This model is intentionally closer to MCP/local-backend semantics: repo context is explicit per operation. It avoids relying on a mutable "currently active repo" as the only source of truth for graph reads.

## Session runtime bridge

The Web chat does not run an AI model inside AVmatrix. The shared contract supports local session providers (`codex` and `claude-code`), while the current backend mounts the Codex CLI adapter. AVmatrix keeps repo binding, streaming, cancellation, and UI state local.

| Component | Role |
|-----------|------|
| `src/runtime/session-adapter.ts` | Provider-neutral session job and stream event contract. |
| `src/runtime/session-adapters/codex.ts` | Native Codex CLI adapter. No AVmatrix API key is stored in the browser, and AVmatrix does not host a chat proxy. |
| `src/runtime/runtime-controller.ts` | Resolves the repo binding, requires an indexed repo, starts/cancels one active chat job per repo. |
| `src/server/session-bridge.ts` | Exposes `/api/session/status`, `/api/session/chat` SSE, and session cancellation. |
| `avmatrix-web/src/hooks/chat-runtime/` | Browser-side status, message streaming, and transcript state. |

Session requests include `repoName` or `repoPath`. The runtime resolves that binding before execution, so chat/tool execution is attached to a concrete local indexed repo instead of ambient UI state.

## Packaged local launcher

`avmatrix-launcher/` packages the local runtime for Windows:

| File | Role |
|------|------|
| `avmatrix-launcher/build.ps1` | Builds CLI, Web UI, launcher executable, server wrapper executable, copies bundled assets, and registers the protocol. |
| `avmatrix-launcher/src/main.go` | Handles `avmatrix://start`, `avmatrix://reset`, and `avmatrix://stop`; serves packaged `web-dist` on `127.0.0.1:5173`; starts the backend wrapper; opens the browser. |
| `avmatrix-launcher/server-wrapper/main.go` | Starts bundled `node.exe avmatrix/dist/cli/index.js serve`. |
| `avmatrix-launcher/web-dist/` | Built Web UI used by the launcher. |
| `avmatrix-launcher/server-bundle/` | Bundled backend runtime used by the launcher. |

The launcher is a convenience layer around the same local backend. It must not become a required cloud/control-plane service, and `avmatrix serve` remains the direct backend entrypoint.

## MCP tools

| Tool | Purpose |
|------|---------|
| `list_repos` | Discover indexed repos |
| `query` | Hybrid BM25 + vector search over the graph |
| `cypher` | Ad hoc Cypher against the schema |
| `context` | Callers, callees, processes for one symbol |
| `impact` | Blast radius (upstream/downstream) with risk summary |
| `detect_changes` | Map git diffs to affected symbols and processes |
| `rename` | Graph-assisted multi-file rename with `dry_run` preview |
| `api_impact` | Pre-change impact report for an API route handler |
| `route_map` | API route → handler → consumer mappings |
| `tool_map` | MCP/RPC tool definitions and handlers |
| `shape_check` | Response shape vs consumer property access mismatches |
| `group_list` | List repo groups or details for one group |
| `group_query` | Cross-repo search in a group (reciprocal rank fusion) |
| `group_sync` | Rebuild group Contract Registry (`contracts.json`) |
| `group_contracts` | Inspect group contracts and cross-links |
| `group_status` | Index and Contract Registry staleness per repo in a group |

## Where to change what

| Concern | Start in |
|---------|----------|
| CLI commands/flags | `src/cli/` (`index.ts`, per-command modules) |
| HTTP server/endpoints | `src/server/api.ts`, `src/server/mcp-http.ts`, `src/server/session-bridge.ts` |
| Repo-scoped graph reads | `src/runtime/repo-runtime/`, `src/runtime/repo-resolver.ts` |
| Session runtime bridge | `src/runtime/runtime-controller.ts`, `src/runtime/session-adapter.ts`, `src/runtime/session-adapters/codex.ts` |
| Parsing/graph construction | `src/core/ingestion/pipeline-phases/` + `pipeline.ts` |
| Graph schema/DB | `src/core/lbug/` (`schema.ts`, `lbug-adapter.ts`) |
| MCP tools/resources | `src/mcp/server.ts`, `tools.ts`, `resources.ts` |
| Search ranking | `src/core/search/` (BM25, hybrid fusion) |
| Embeddings | `src/core/embeddings/` + `src/core/run-analyze.ts` |
| Wiki generation | `src/core/wiki/` |
| Language support | `src/core/ingestion/languages/` + `tree-sitter-queries.ts` + `avmatrix-shared/src/languages.ts` |
| Import resolution | `src/core/ingestion/import-processor.ts` + `import-resolvers/configs/` + `model/resolution-context.ts` |
| Call resolution/MRO | `src/core/ingestion/call-processor.ts` + `model/resolve.ts` |
| Type extraction | `src/core/ingestion/type-extractors/` |
| Worker pool | `src/core/ingestion/workers/` |
| Web UI local runtime | `avmatrix-web/src/hooks/useAppState.local-runtime.tsx`, `avmatrix-web/src/services/backend-client.ts` |
| Web UI chat runtime | `avmatrix-web/src/hooks/chat-runtime/`, `avmatrix-web/src/components/right-panel/` |
| Local launcher | `avmatrix-launcher/src/main.go`, `avmatrix-launcher/server-wrapper/main.go`, `avmatrix-launcher/build.ps1` |
| CI | `.github/workflows/*.yml`, `.github/actions/` |

> Paths above are relative to `avmatrix/` unless they start with `avmatrix-web/`, `avmatrix-launcher/`, `.github/`, or another repository-root path.

---

## Pipeline Phase DAG

12 phases defined in `avmatrix/src/core/ingestion/pipeline-phases/`, each with explicit `deps` and typed output.

```
scan → structure → [markdown, cobol] → parse → [routes, tools, orm]
  → crossFile → mro → communities → processes
```

| Phase | File | Deps | Output |
|-------|------|------|--------|
| `scan` | `scan.ts` | (root) | File paths + sizes |
| `structure` | `structure.ts` | `scan` | File/Folder nodes, CONTAINS edges, `allPathSet` |
| `markdown` | `markdown.ts` | `structure` | Section nodes, cross-link edges from .md/.mdx |
| `cobol` | `cobol.ts` | `structure` | COBOL program/paragraph/section nodes (regex, no tree-sitter) |
| `parse` | `parse.ts` + `parse-impl.ts` | `structure`, `markdown`, `cobol` | Symbol nodes, IMPORTS/CALLS/EXTENDS edges, extracted routes/tools/ORM queries |
| `routes` | `routes.ts` | `parse` | Route nodes + HANDLES_ROUTE edges (Next.js, Expo, PHP, decorators) |
| `tools` | `tools.ts` | `parse` | Tool nodes + HANDLES_TOOL edges |
| `orm` | `orm.ts` | `parse` | QUERIES edges (Prisma, Supabase) |
| `crossFile` | `cross-file.ts` + `cross-file-impl.ts` | `parse`, `routes`, `tools`, `orm` | Cross-file type propagation in topological import order |
| `mro` | `mro.ts` | `crossFile`, `structure` | METHOD_OVERRIDES + METHOD_IMPLEMENTS edges |
| `communities` | `communities.ts` | `mro`, `structure` | Community nodes + MEMBER_OF edges (Leiden algorithm) |
| `processes` | `processes.ts` | `communities`, `routes`, `tools`, `structure` | Process nodes + STEP_IN_PROCESS edges |

**Non-phase files in the same directory:** `parse-impl.ts`, `cross-file-impl.ts` (implementation), `wildcard-synthesis.ts` (whole-module import expansion), `orm-extraction.ts` (sequential ORM fallback), `types.ts`, `runner.ts`, `index.ts`.

### DAG runner

`runner.ts` — static phase graph, no plugins, compile-time type safety.

1. **Validation** — Kahn's topological sort. Rejects on: duplicate names, missing deps, cycles (DFS traces the concrete cycle path, e.g., `A -> B -> C -> A`, plus count of transitively blocked dependents).

2. **Execution** — sequential in topological order. Each phase receives:
   - `ctx: PipelineContext` — shared mutable `KnowledgeGraph`, `repoPath`, progress callback, options
   - `deps: ReadonlyMap<string, PhaseResult>` — **declared deps only** (runner filters the results map to prevent hidden coupling)

3. **Error handling** — wraps phase errors with the phase name, emits terminal `error` progress event, swallows progress handler errors to preserve the original cause.

4. **Timing** — per-phase `durationMs` in `PhaseResult`, dev-mode console logging.

**Design patterns:**
- **Single graph accumulator** — all phases mutate the same `KnowledgeGraph` in `ctx`; the graph is the primary output.
- **Typed phase access** — `getPhaseOutput<T>(deps, 'name')` for type-safe upstream results.
- **Binding accumulator lifecycle** — created in `parse`, disposed by `crossFile` (in `finally`). No other phase should take ownership.
- **Skippable phases** — `skipGraphPhases` omits MRO/communities/processes (faster tests). `skipWorkers` forces sequential parsing.

### How to add a new phase

1. Create `pipeline-phases/my-phase.ts` with a `PipelinePhase<MyOutput>` (name, deps, execute)
2. Export from `pipeline-phases/index.ts`
3. Add to `buildPhaseList()` in `pipeline.ts`

```typescript
import type { PipelinePhase, PhaseResult } from './types.js';
import { getPhaseOutput } from './types.js';
import type { ParseOutput } from './parse.js';

export interface MyPhaseOutput { /* ... */ }

export const myPhase: PipelinePhase<MyPhaseOutput> = {
  name: 'myPhase',
  deps: ['parse'],
  async execute(ctx, deps) {
    const { allPaths } = getPhaseOutput<ParseOutput>(deps, 'parse');
    // ... write to ctx.graph ...
    return { /* typed output */ };
  },
};
```

---

## Call-Resolution DAG

Typed 6-stage pipeline in `call-processor.ts` (inside the `parse` phase) that resolves method/function calls and emits CALLS edges. Language behavior plugs in at two `LanguageProvider` hook points (stages 3–4); shared code names no languages. Scope: call resolution only — import resolution, type extraction, heritage, and symbol-table population live in other phases.

### Stages

```
extract-call ──▶ classify-form ──▶ infer-receiver ──▶ select-dispatch ──▶ resolve-target ──▶ emit-edge
     (1)              (2)            (3)  [hook]       (4)  [hook]         (5)                 (6)
```

| Stage | Produces | Location |
|-------|----------|----------|
| **extract-call** | `ExtractedCallSite` (name, form, receiver, argCount) | `call-extractors/` (per-language); runs in worker |
| **classify-form** | callForm (`free`/`member`/`constructor`) + arity | `call-analysis.ts` → `inferCallForm`; shared, runs in worker |
| **infer-receiver** | `ReceiverEnriched` (receiver type finalized) | `call-processor.ts`; shared default chain, then `inferImplicitReceiver` hook |
| **select-dispatch** | `DispatchDecision` (primary, fallback, ancestryView) | `selectDispatch` hook, falls back to shared default |
| **resolve-target** | `TieredCandidates` | `model/resolve.ts` → `lookupMethodByOwnerWithMRO` (MRO walk) |
| **emit-edge** | CALLS edge in graph | `call-processor.ts`; writes edge with confidence tier |

### Provider hooks

Both hooks are optional on `LanguageProvider`. Ruby is the only current implementer.

**`inferImplicitReceiver`** — called after shared infer-receiver defaults. Returns `ImplicitReceiverOverride | null`.

| | |
|---|---|
| Inputs | `calledName`, `callForm`, `receiverName`, `receiverTypeName`, `callNode` (AST), `filePath` |
| Non-null fields | `callForm`, `receiverName`, `receiverTypeName` (required); `receiverSource: 'implicit-self'` (fixed); `hint?` (opaque, passed to `selectDispatch`) |
| Null | Keep existing `ReceiverEnriched` state |

**`selectDispatch`** — called after infer-receiver (including hook). Returns `DispatchDecision | null`; null uses shared default (constructor → `primary:'constructor'`; typed receiver → `primary:'owner-scoped'`; else → `primary:'free'`).

| | |
|---|---|
| Inputs | `calledName`, `callForm`, `receiverName`, `receiverTypeName`, `receiverSource`, `hint` |
| Non-null fields | `primary: 'owner-scoped' \| 'free' \| 'constructor'`; `fallback?: 'free-arity-narrowed'`; `ancestryView?: 'instance' \| 'singleton'`; `hint?` |

**`DispatchDecision` field semantics:**
- `primary: 'owner-scoped'` — MRO walk from receiver's type; used when receiver type is known.
- `fallback: 'free-arity-narrowed'` — after owner-scoped miss, search free-call candidates by arity only (Ruby uses this for implicit-self calls that miss their owner's MRO).
- `ancestryView: 'singleton'` — walk singleton/class ancestry instead of instance ancestry (Ruby `def self.foo` bodies, so `extend`-ed methods are found).

### Adding language behavior

1. **Implicit receivers** — implement `inferImplicitReceiver`: return null if call already has a receiver; otherwise use `findEnclosingClassInfo` (`ast-helpers.ts`) to find the enclosing context, return `ImplicitReceiverOverride` with `receiverSource: 'implicit-self'`, and optionally set `hint` for `selectDispatch`.
2. **Custom dispatch** — implement `selectDispatch`: inspect `receiverSource` and `hint`, return `DispatchDecision` with `primary`, optional `fallback`, optional `ancestryView`; return null to keep shared defaults.
3. **MRO strategy** — confirm `mroStrategy` is `'first-wins'`, `'c3'`, `'ruby-mixin'`, or `'none'`; consumed by `lookupMethodByOwnerWithMRO`.

**Ruby example** (`languages/ruby.ts` + `utils/ruby-self-call.ts`): `inferImplicitReceiver` rewrites bare-identifier calls to `self.method` and sets `hint` to `'instance'`/`'singleton'`; `selectDispatch` uses hint for `ancestryView` and adds `fallback: 'free-arity-narrowed'` for implicit-self calls.

### Code references

| Module | Purpose |
|--------|---------|
| `core/ingestion/call-types.ts` | DAG types: `ReceiverEnriched`, `DispatchDecision`, `ImplicitReceiverOverride` |
| `core/ingestion/language-provider.ts` | Hook signatures: `inferImplicitReceiver`, `selectDispatch` |
| `core/ingestion/call-processor.ts` | `processCalls`: stages 3–6 |
| `core/ingestion/model/resolve.ts` | `lookupMethodByOwnerWithMRO`: stage 5 MRO walk |
| `core/ingestion/languages/ruby.ts` | Both hooks + `mroStrategy: 'ruby-mixin'` |
| `core/ingestion/utils/ruby-self-call.ts` | Bare-call rewrite for `inferImplicitReceiver` |

---

## Language-agnostic graph feeding

16 languages → single unified graph. Four abstraction layers:

```
 Unified Graph Schema (shared NODE_TABLES + REL_TYPES constants)
           ↑
 Unified Resolution (3-tier name lookup + MRO walk)
           ↑
 Language Providers (import semantics, type config, export checker, MRO strategy)
           ↑
 Tree-Sitter Queries (per-language S-expressions, unified capture tags)
```

### Language providers

Each language implements `LanguageProvider` (`language-provider.ts`). Key fields:

| Field | Purpose |
|-------|---------|
| `id`, `extensions` | Language identity and file matching |
| `treeSitterQueries` | S-expression queries for AST extraction |
| `importSemantics` | `named` / `wildcard-leaf` / `wildcard-transitive` / `namespace` |
| `importResolver` | Language-specific path → file resolution |
| `exportChecker` | Public/exported symbol detection |
| `typeConfig` | Type annotation extraction rules |
| `mroStrategy` | `first-wins` / `c3` / `none` |

16 providers in `languages/index.ts` via `satisfies Record<SupportedLanguages, LanguageProvider>` — missing a language is a compile error.

### Unified capture tags

Per-language tree-sitter queries use different AST node names but produce the **same semantic capture tags**: `@definition.class`, `@definition.function`, `@call.name`, `@import.source`, `@heritage.extends`. Downstream extraction needs no language branching. Defined in `tree-sitter-queries.ts`.

### Import resolution

Per-language import resolution uses the **configs + factory** pattern (like call/method/class extractors). Each language declares an `ImportResolutionConfig` in `import-resolvers/configs/`, listing an ordered chain of `ImportResolverStrategy` functions. `createImportResolver()` (in `resolver-factory.ts`) composes them: first non-null result wins. Low-level helpers shared across strategies live alongside the configs in `import-resolvers/` (e.g. `go.ts`, `rust.ts`, `python.ts`).

Unified 3-tier algorithm (`model/resolution-context.ts`), per-language `importSemantics` controls which tier activates:

| Tier | Confidence | Mechanism |
|------|-----------|-----------|
| 1 — same-file | 0.95 | Symbol table for caller's file |
| 2 — import-scoped | 0.9 | `NamedImportMap` chains (named) or all files in `importMap` (wildcard) |
| 3 — global | 0.5 | O(1) index lookups: class, impl, callable. Fallback only |

| Import strategy | Languages | Behavior |
|----------------|-----------|----------|
| `named` | TS, JS, Java, C#, Rust, PHP, Kotlin | Only explicitly imported names visible |
| `wildcard-leaf` | Go, Ruby, Swift, Dart | Whole-package import, no transitive re-exports |
| `wildcard-transitive` | C, C++ | `#include` closure chains through re-exports |
| `namespace` | Python | Module aliases resolved at call site |

### Chunked parse-and-resolve

`parse` processes files in ~20 MB byte-budget chunks to bound memory. Per chunk:
1. Worker pool dispatches files (or sequential fallback via `skipWorkers`)
2. Each worker: detect language → load grammar → run queries → return unified `ParseWorkerResult`
3. Synthesize wildcard bindings (`wildcard-synthesis.ts`)
4. Resolve imports and heritage
5. Collect `BindingAccumulator` entries for cross-file propagation

Workers: `workers/worker-pool.ts`, `workers/parse-worker.ts`.

### Heritage and MRO

All languages emit unified `ExtractedHeritage` (child, parent, `EXTENDS`/`IMPLEMENTS`). MRO phase walks the heritage graph using per-language strategy:
- **`first-wins`** — Java, C#, C++, TS, Ruby, Go
- **`c3`** — Python (C3 linearization)
- **`none`** — single-inheritance languages

Unified walk: `lookupMethodByOwnerWithMRO()` in `model/resolve.ts`.

---

## Full analysis flow

`runFullAnalysis` in `run-analyze.ts` orchestrates everything around the pipeline:

```
CLI (analyze.ts) → runFullAnalysis(repoPath, options, callbacks)
  1. Early exit if lastCommit == HEAD (unless --force)     [0%]
  2. Cache existing embeddings from prior index             [0%]
  3. runPipelineFromRepo() → KnowledgeGraph                [0-60%]
  4. Clean up legacy KuzuDB files                          [60%]
  5. initLbug() → loadGraphToLbug() via CSV streaming      [60-85%]
  6. Create FTS indexes (File, Function, Class, Method...) [85-90%]
  7. Restore cached embeddings (batch insert)              [88%]
  8. Generate new embeddings if --embeddings               [90-98%]
  9. Save metadata + register repo + update .gitignore     [98-100%]
 10. Generate AI context files (AGENTS.md, CLAUDE.md)      [100%]
```

**Options:** `--force` (rebuild regardless), `--embeddings` (opt-in, skipped if >50k nodes), `--skipGit`, `--noStats`.

## Storage

```
<repo>/.avmatrix/
  ├── lbug           # LadybugDB database
  ├── lbug.wal       # Write-ahead log
  ├── lbug.lock      # Single-writer lock
  └── meta.json      # lastCommit, indexedAt, stats

~/.avmatrix/
  └── registry.json  # Global repo registry (MCP discovery)

avmatrix-launcher/
  ├── web-dist/      # Packaged Web UI built from avmatrix-web/
  ├── server-bundle/ # Packaged CLI/backend runtime plus bundled node.exe
  └── logs/          # launcher.log, backend.log, server-wrapper.log

%TEMP%/
  └── avmatrix-launcher-<hash>.json # Launcher/backend PID state for start/reset/stop
```

Repo index storage is managed by `repo-manager.ts`. Launcher state and logs are managed by `avmatrix-launcher/src/main.go` and `avmatrix-launcher/server-wrapper/main.go`.

## LadybugDB schema

Defined in `lbug/schema.ts`; the table-name constants come from `avmatrix-shared/src/lbug/schema-constants.ts`. Separate node tables per type, one `CodeRelation` table for edges, and one `CodeEmbedding` table for vector chunks.

**Node tables:** File, Folder, Function, Class, Interface, Method, CodeElement, Community, Process, Section, Struct, Enum, Macro, Typedef, Union, Namespace, Trait, Impl, TypeAlias, Const, Static, Variable, Property, Record, Delegate, Annotation, Constructor, Template, Module, Route, Tool.

**Relation types** (`CodeRelation.type`): CONTAINS, DEFINES, IMPORTS, CALLS, EXTENDS, IMPLEMENTS, HAS_METHOD, HAS_PROPERTY, ACCESSES, METHOD_OVERRIDES, OVERRIDES (legacy compat), METHOD_IMPLEMENTS, MEMBER_OF, STEP_IN_PROCESS, HANDLES_ROUTE, FETCHES, HANDLES_TOOL, ENTRY_POINT_OF, WRAPS, QUERIES.

## Embeddings and search

**Embeddings** (`src/core/embeddings/`): Snowflake arctic-embed-xs (384D). Embeddable: File, Function, Class, Method, Interface. Incremental via SHA1 content hash. Stored in the separate `CodeEmbedding` table.

**Search** (`src/core/search/`): Hybrid BM25 + semantic vector, merged via Reciprocal Rank Fusion (K=60).

## Known limitations

### Overloaded method resolution

Node IDs use arity suffix (`#<paramCount>`): `Method:file:Class.method#1` vs `#2`.

**Same-arity disambiguation:** type-hash suffix `~type1,type2` when collision detected and type annotations present. Languages without types (Python, Ruby, JS) use arity-only. TS/JS overload signatures excluded (collapse to implementation body). See #651.

**C++ const-qualified:** `$const` suffix after type-hash when non-const collision exists: `Method:file:Container.begin#0$const`.

**Generic/template types:** type-hash uses `rawType` (full AST text including generics): `~vector<int>` vs `~vector<std::string>`.

**ID stability:** collision-only tags mean IDs change when overloads are added. `save#1` becomes `save#1~int` when `save(String)` is added.

**Variadic matching:** confidence 0.7 when one side is variadic and the other has fixed count.

**METHOD_IMPLEMENTS confidence tiering:**

| Match quality | Confidence |
|---|---|
| Exact parameter types match | 1.0 |
| Arity match, types unavailable | 1.0 |
| Variadic vs fixed | 0.7 |
| Insufficient info | 0.7 |

## Related docs

- [MIGRATION.md](MIGRATION.md) — breaking changes and migration guidance
- [RUNBOOK.md](RUNBOOK.md) — operational commands and recovery
- [GUARDRAILS.md](GUARDRAILS.md) — safety boundaries for humans and agents
- [TESTING.md](TESTING.md) — how to run tests
- `AGENTS.md` / `CLAUDE.md` — agent workflows and tool usage
