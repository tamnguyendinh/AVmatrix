# avmatrix-GO TypeScript/Node To Go Conversion Plan

Date: 2026-05-08

Status: Draft for execution in `F:\AVmatrix-GO`.

Companion files:

- Evidence: [2026-05-08-avmatrix-go-typescript-node-to-go-conversion-evidence.md](2026-05-08-avmatrix-go-typescript-node-to-go-conversion-evidence.md)
- Benchmarks: [2026-05-08-avmatrix-go-typescript-node-to-go-conversion-benchmark.md](2026-05-08-avmatrix-go-typescript-node-to-go-conversion-benchmark.md)

# Rules

1. Use AVmatrix for codebase analysis and impact checks while working on this plan.
2. As each task is completed, update the corresponding checklist item immediately.
3. Run a full build before testing; the test suite must include an e2e test.
4. Record benchmark results as each benchmarkable task is completed.
5. Record evidence as each evidenced task is completed.
6. After each completed implementation slice, commit the work, then continue until the full plan is complete.
7. commit doc only do not use avmatrix.

## Goal

The repository named `avmatrix-GO` currently contains the AVmatrix implementation written in
TypeScript/Node.

The goal of this plan is to convert that same repository to Go while preserving the current product
contract. The only code that is intentionally allowed to remain TypeScript/React after cutover is
the Web UI display layer.

```text
same target repositories analyzed by the current product
same analyze semantics
same graph schema
same HTTP/MCP contracts
same or better graph accuracy
faster graph generation and graph serving
```

The current TypeScript/Node implementation in this repository is the conversion baseline and
contract authority during the rewrite. Contract snapshots must be generated from this repository
before the equivalent Go implementation replaces each runtime surface.

Benchmark rule: every benchmark in this plan and the benchmark companion file is conversion
evidence. During conversion, benchmark regressions are recorded as optimization backlog signals
unless they prove a correctness, contract, architecture, or runtime-shape blocker. Keep conversion
phases focused on preserving architecture, runtime flow, contracts, and graph accuracy; do not start
optimization work inside conversion phases just to improve a number.

2026-05-14 priority correction:
the primary goal is a Go tool that runs correctly and accurately enough to put into real use as an
independent AVmatrix implementation. The target is not merely parity: `AVmatrix-GO` must run more
correctly and accurately than the currently used `AVmatrix-main` wherever the Go conversion has
identified and fixed legacy weaknesses. Correctness, graph accuracy, contract parity, and runtime
completion outrank heavy optimization work. Benchmarks remain mandatory evidence and regression
checks for each slice, and the Go implementation must not become slower than the current accepted
baseline; because current accepted evidence already shows Go faster on the large current-repo run,
new benchmark work during conversion is limited to measurement and light optimization unless a
speed regression blocks correctness, parity, or usability. Heavy optimization belongs to a later
optimization plan after the tool is correct, accurate, and usable.

The target conversion scope is:

- Convert CLI, backend server, analyzer pipeline, parser orchestration, ScopeIR extraction,
  resolution, graph emission, LadybugDB load/read, MCP server, benchmark tooling, and runtime
  repo management to Go.
- Convert all non-Web UI runtime code to Go, including process control, launcher/runtime
  integration, setup/config helpers, direct CLI tool commands, group/contract tooling, wiki
  gating/status commands, benchmark comparison, local session bridge, embeddings/search runtime,
  and MCP stdio/HTTP transports.
- Keep only the existing Web UI display layer in TypeScript/React. Small Web UI adapter changes
  are allowed only when needed to call the Go backend without changing user-facing behavior.
- Generated TypeScript contract types are allowed only as browser-side Web UI build artifacts, not
  as an independent shared runtime or backend authority.
- Do not leave a TypeScript/Node runtime shim on the normal cutover path. Temporary shims are
  allowed only during migration and must be removed before cutover.
- Preserve the same local-first runtime model. No cloud service, required daemon, or hosted
  workspace is introduced.

2026-05-14 correction:
the conversion target is stricter than "the Go binary works on the normal path." All non-Web UI
TypeScript/JavaScript implementation in this repository is in scope. Moving the CLI/MCP/analyzer
entrypoint to Go is not enough if related TypeScript packages, scripts, shared contracts, tests, or
distribution glue remain as active implementation authority. The only intentionally retained
TypeScript/React code after final cutover is the Web UI display/build surface plus Go-generated
browser contract glue. Any other TypeScript/JavaScript must be ported to Go, removed, or explicitly
classified as non-runtime fixture/baseline data with a cutover-package exclusion and a recorded
decision.

## Non-Goals

- Do not rewrite the Web UI to Go.
- Do not keep TypeScript/Node as the normal CLI, backend, MCP, analyzer, launcher runtime, or
  repository-management implementation after cutover.
- Do not change user-facing CLI semantics while porting.
- Do not drop graph accuracy work to claim speed.
- Do not replace the current graph schema unless a compatibility bridge proves exact behavior.
- Do not replace the HTTP API with a new API before the current Web UI contract passes.
- Do not do a file-by-file mechanical translation if a clean Go module boundary is safer.
- Do not optimize by skipping scope resolution, type propagation, MRO, audit metadata, or DB
  persistence.
- Do not change the launcher display UX as part of the core Go port; only its non-Web UI
  runtime/process-control implementation should move to Go.

## Safety Model

Work is scoped to the `avmatrix-GO` repository:

```text
F:\AVmatrix-GO
  role: TypeScript/Node codebase being converted to Go
  default action: active implementation
```

Rules:

- Use the current TypeScript/Node code in `F:\AVmatrix-GO` to generate baseline artifacts and parity
  outputs before replacing each runtime surface with Go.
- Keep every conversion phase independently runnable.
- Commit rollback points after each green milestone.
- If a phase cannot meet parity, stop and fix parity before moving to the next phase.

## Architecture And Runtime Flow Reference

Use `F:/avmatrix-main` as the architecture and runtime-flow reference for this conversion. The
purpose is to prevent the Go port from drifting into a different architecture, different module
responsibility split, or different execution flow.

Reference rules:

- `F:/AVmatrix-GO` remains the active implementation repository and the place where all changes are
  made.
- `F:/avmatrix-main` is used only to understand and cross-check architecture, runtime orchestration,
  module boundaries, CLI/server/MCP flow, analyzer flow, launcher flow, and packaging flow.
- Behavioral parity snapshots are generated from `F:/AVmatrix-GO` because that is the repository
  being converted.
- Architecture and runtime-flow shape are checked against `F:/avmatrix-main` before each major Go
  surface is designed.
- Do not switch implementation work to `F:/avmatrix-main`.
- Do not mechanically copy files from `F:/avmatrix-main`; use it to validate design and behavior.
- Before porting a major runtime surface to Go, compare the planned Go architecture and execution
  flow against `F:/avmatrix-main`.
- If `F:/AVmatrix-GO` and `F:/avmatrix-main` disagree on architecture or runtime flow, record the
  mismatch in Phase 1 evidence and classify it before encoding that surface in Go:
  intentional product delta, local drift to correct, or architecture decision needed.
- Do not silently implement from whichever repository is easier to read. Every mismatch must have a
  recorded decision before the Go implementation encodes that surface.

## Version Freshness Policy

The Go rewrite must not inherit stale dependency choices from the current TypeScript/Node runtime.
Before implementing or pinning any new Go runtime dependency, verify the latest stable upstream
release at that time and record the selected version in the contract-freeze evidence.

Required freshness checks:

- Go toolchain: use the latest stable Go release supported on Windows.
- Tree-sitter core and language grammars: use the latest stable upstream core plus latest stable
  grammar revisions that can pass the parser feasibility proof and provider parity fixtures. If a
  Go binding lags behind upstream core, that binding is feasibility-only until the plan records a
  direct native binding path or an explicit blocker.
- LadybugDB client/runtime: use the latest stable upstream runtime through a native integration
  that can pass the Windows load/read feasibility proof. A lagging Go wrapper is not a cutover-path
  dependency.
- MCP SDK/protocol support: use the latest stable Go-compatible MCP implementation or document a
  minimal in-repo implementation when no stable SDK exists.
- HTTP/router/SSE/WebSocket/session dependencies: use maintained latest stable releases.
- Embeddings/vector/search dependencies: use maintained latest stable releases and verify Windows
  runtime compatibility before accepting the dependency.
- Launcher/packaging dependencies: use maintained latest stable releases and avoid deprecated
  process-control or installer tooling.

Rules:

- Do not pin an old version only because the current TypeScript implementation uses an old package.
- Do not treat a wrapper package as "latest" when the authoritative upstream runtime has a newer
  stable release. The upstream runtime version is the authority for conversion acceptance.
- If the latest stable version cannot be used, write the blocker, the tested version range, and the
  exact reason before selecting a fallback.
- No unmaintained, archived, deprecated, or prerelease dependency is allowed on the cutover path
  unless the plan records an explicit blocker and mitigation.
- Dependency freshness is part of acceptance. A phase is not complete if it works only with an
  avoidably stale runtime dependency.

## Target Architecture

After conversion, the repository should be organized around stable Go contracts, not around the
current TypeScript file layout.

Proposed package layout:

```text
cmd/avmatrix/
  CLI entrypoint: setup, analyze, benchmark-compare, index, serve, mcp, list,
  status, clean, wiki, wiki-mode, augment, query, context, impact, cypher,
  detect-changes, group subcommands, version/help

cmd/avmatrix-server/
  Optional backend-only entrypoint if launcher packaging needs it

cmd/avmatrix-launcher/
  Optional launcher/runtime-control entrypoint if packaging needs a separate binary

internal/config/
  config loading, defaults, env handling, runtime paths

internal/contracts/
  Go-owned contract structs, schema constants, JSON payloads, and generated TypeScript adapter
  types needed by the Web UI display layer

internal/repo/
  registry, meta.json, repo identity, storage paths, local path policy

internal/ignore/
  .gitignore, built-in excludes, skip rules, traversal filters

internal/scanner/
  repo walk, language detection, file hashing, batching, scan metrics

internal/parser/
  tree-sitter setup, grammar registry, parser pool, parse workers

internal/scopeir/
  serializable facts: definitions, imports, call sites, accesses, inheritance,
  lexical scopes, type hints, return types, framework facts

internal/providers/
  language providers and AST-to-ScopeIR extractors

internal/resolution/
  global symbol index, import binding, scope lookup, reference resolution,
  method dispatch, MRO, edge emission, audit metadata

internal/graph/
  node/edge model, graph IDs, duplicate handling, graph summaries

internal/lbug/
  LadybugDB schema, primary CSV/COPY load, read pools, query helpers, legacy read compatibility

internal/httpapi/
  local HTTP API used by the Web UI, local runtime, MCP HTTP transport, analyze/embed jobs,
  graph streaming, file/query/search/process/cluster/session endpoints

internal/mcp/
  MCP stdio/HTTP tools, prompts, resources, and compatibility payload schemas

internal/search/
  BM25/FTS, semantic/vector search, query ranking, search snapshots

internal/group/
  repository groups, contract registry, cross-repo contract sync/query/status

internal/session/
  local session/chat bridge runtime used by the Web UI

internal/launcher/
  launcher process control, reset/stop/start/protocol integration, packaging hooks

internal/benchmark/
  phase timing, memory, graph parity, JSON artifact generation

internal/fixtures/
  parity fixtures and generated expected artifacts

internal/testutil/
  golden graph comparison, temp repo generation, API snapshot helpers
```

Shared contract rule:

- The Web UI may remain TypeScript/React, but shared contracts must not remain an independent
  TypeScript runtime authority after cutover.
- Go owns graph schema, HTTP payloads, MCP payloads, session stream events, pipeline progress,
  registry/meta shapes, and benchmark artifact shapes.
- The Web UI receives those contracts through generated TypeScript types or a thin Web-only
  adapter package whose only purpose is browser compilation.
- No CLI/backend/analyzer/MCP/launcher behavior may depend on `avmatrix-shared` TypeScript code
  after cutover.

## Compatibility Contracts

### CLI Contract

The Go CLI must preserve the existing behavior before it replaces the TypeScript CLI:

- `avmatrix setup`
- `avmatrix analyze`
- `avmatrix analyze <path>`
- `avmatrix analyze --force`
- `avmatrix analyze --embeddings`
- `avmatrix analyze --skills`
- `avmatrix analyze --skip-agents-md`
- `avmatrix analyze --no-stats`
- `avmatrix analyze --skip-git`
- `avmatrix analyze --skip-compatibility-cross-file`
- `avmatrix analyze --benchmark-json <file>`
- `avmatrix analyze --benchmark-label <label>`
- `avmatrix analyze --name <alias>`
- `avmatrix analyze --allow-duplicate-name`
- `avmatrix analyze --verbose`
- `avmatrix benchmark-compare <before> <after>`
- `avmatrix benchmark-compare --json`
- `avmatrix index [path...]`
- `avmatrix index --force`
- `avmatrix index --allow-non-git`
- `avmatrix serve`
- `avmatrix serve --port <port>`
- `avmatrix serve --host <host>`
- `avmatrix mcp`
- `avmatrix list`
- `avmatrix status`
- `avmatrix clean`
- `avmatrix clean --force`
- `avmatrix clean --all`
- `avmatrix wiki [path]`
- `avmatrix wiki-mode [mode]`
- `avmatrix augment <pattern>`
- `avmatrix query <search_query>`
- `avmatrix context [name]`
- `avmatrix impact [target]`
- `avmatrix cypher <query>`
- `avmatrix detect-changes`
- `avmatrix group create <name>`
- `avmatrix group create --force`
- `avmatrix group add <group> <groupPath> <registryName>`
- `avmatrix group remove <group> <path>`
- `avmatrix group list [name]`
- `avmatrix group status <name>`
- `avmatrix group sync <name>`
- `avmatrix group sync --skip-embeddings`
- `avmatrix group sync --exact-only`
- `avmatrix group sync --allow-stale`
- `avmatrix group sync --verbose`
- `avmatrix group sync --json`
- `avmatrix group query <name> <query>`
- `avmatrix group query --subgroup <path>`
- `avmatrix group query --limit <n>`
- `avmatrix group query --json`
- `avmatrix group contracts <name>`
- `avmatrix group contracts --type <type>`
- `avmatrix group contracts --repo <repo>`
- `avmatrix group contracts --unmatched`
- `avmatrix group contracts --json`
- direct tool flags for `query`, `context`, `impact`, `cypher`, and `detect-changes`, including
  repo selection, UID/file disambiguation, content inclusion, depth, include-tests, base ref, task
  context, goal, and limit flags from the local CLI baseline
- `avmatrix version`
- `avmatrix help`

Required CLI invariants:

- Running `analyze` inside a repo analyzes only that repo.
- Passing an explicit repo path analyzes that repo, not a repo list entry with the same name.
- Analyze writes repo-local data under `<repo>/.avmatrix/`.
- Registry entries preserve path, storage path, indexed time, stats, and commit metadata.
- CLI output remains scriptable enough for current workflows.
- All current flags, defaults, environment variables, exit codes, prompts, and error text that
  scripts depend on are frozen from the local TypeScript baseline before implementation.
- Every current CLI command and flag is required for cutover unless a separate product decision
  changes the contract before this plan reaches implementation. An unported command blocks cutover.

### HTTP API Contract

The Go HTTP backend must remain compatible with the current Web UI:

- `GET /api/heartbeat`
- `GET /api/info`
- `GET /api/repos`
- `GET /api/repo`
- `DELETE /api/repo`
- `GET /api/graph`
- `POST /api/query`
- `POST /api/search`
- `GET /api/file`
- `GET /api/grep`
- `GET /api/processes`
- `GET /api/process`
- `GET /api/clusters`
- `GET /api/cluster`
- `POST /api/local/folder-picker`
- `POST /api/analyze`
- `GET /api/analyze/:jobId`
- `GET /api/analyze/:jobId/progress`
- `DELETE /api/analyze/:jobId`
- `POST /api/embed`
- `GET /api/embed/:jobId`
- `GET /api/embed/:jobId/progress`
- `DELETE /api/embed/:jobId`
- `GET /api/session/status`
- `POST /api/session/chat`
- `DELETE /api/session/:sessionId`
- `ALL /api/mcp`
- graph streaming NDJSON format
- file/read/search/query endpoints used by the UI
- repo-scoped query/search/follow-up calls
- session/chat bridge endpoints that exist in the current runtime
- MCP StreamableHTTP endpoint behavior

Required HTTP invariants:

- Analyze requests from the Web UI run full analyze.
- Analyze completion includes `repoName` and `repoPath`.
- Post-analyze graph load uses the same selected `repoPath`.
- Repo switching remains repo-scoped and does not depend on mutable process-global active repo.
- Errors include selected repo path when path routing fails.
- Every endpoint must preserve method, path, query/body schema, response shape, status code,
  SSE event format, timeout behavior, CORS behavior, and path-first repo routing.
- `embed`, `session`, and `mcp` endpoints are part of the non-Web UI runtime and must be ported to
  Go before cutover.

### MCP Contract

The Go MCP server must preserve tool behavior:

- `list_repos`
- `query`
- `cypher`
- `context`
- `detect_changes`
- `rename`
- `impact`
- `route_map`
- `tool_map`
- `shape_check`
- `api_impact`
- `group_list`
- `group_sync`
- `group_contracts`
- `group_query`
- `group_status`
- static resources for indexed repos and setup content
- resources for repo context, clusters, processes, graph schema, cluster detail, and process traces
- resource templates and prompts exposed by the local TypeScript baseline

Required MCP invariants:

- Multi-repo ambiguity is handled by explicit repo parameter.
- Query/context/impact read the same graph schema as HTTP.
- Audit metadata is visible where it is visible today.
- Existing MCP clients can continue to call the same tool names and payload shapes.
- Tool names, input schemas, optional/default fields, result payloads, markdown formatting,
  resource URIs, resource templates, prompt names, stale-index warnings, and error behavior are
  frozen from the local TypeScript baseline before the Go server is accepted.
- Tool discovery must expose the same tool set in stdio MCP and HTTP MCP where the current
  runtime exposes both.

### Graph Contract

The graph contract is the central conversion contract:

- Node labels and IDs remain stable.
- Relationship types remain stable.
- Known node label authorities currently include:
  `File`, `Folder`, `Function`, `Class`, `Interface`, `Method`, `CodeElement`, `Community`,
  `Process`, `Section`, `Struct`, `Enum`, `Macro`, `Typedef`, `Union`, `Namespace`, `Trait`,
  `Impl`, `TypeAlias`, `Const`, `Static`, `Variable`, `Property`, `Record`, `Delegate`,
  `Annotation`, `Constructor`, `Template`, `Module`, `Route`, `Tool`, and type-level candidates
  such as `Project`, `Package`, `Decorator`, `Import`, and `Type`.
- Known relationship authorities currently include:
  `CONTAINS`, `DEFINES`, `IMPORTS`, `CALLS`, `USES`, `INHERITS`, `EXTENDS`,
  `IMPLEMENTS`, `HAS_METHOD`, `HAS_PROPERTY`, `ACCESSES`, `METHOD_OVERRIDES`,
  `OVERRIDES`, `METHOD_IMPLEMENTS`, `MEMBER_OF`, `STEP_IN_PROCESS`, `HANDLES_ROUTE`,
  `FETCHES`, `HANDLES_TOOL`, `ENTRY_POINT_OF`, `WRAPS`, and `QUERIES`. `DECORATES`
  remains a type-level candidate until a runtime emitter or fixture proves it.
- Legacy compatibility aliases, especially `OVERRIDES`, remain readable until an explicit
  migration says otherwise.
- If the local TypeScript baseline emits graph facts outside the shared schema constants, the conflict must be
  resolved in the contract-freeze phase before Go code is written.
- Verification alone is not enough for runtime-emitted graph facts. If a runtime-emitted node label
  or relationship type is classified as a required schema member, the TypeScript baseline must be
  wired first across shared schema constants, exposed schema resources/docs, security allowlists,
  fixtures, and contract tests. The Go port must then copy the reconciled contract, not defer or
  reinterpret the mismatch during conversion.
- Current Phase 1 decision: `USES` and `INHERITS` are required runtime-emitted relationship types,
  not legacy-only aliases. If either is absent from any schema authority or exposed schema surface,
  fix that baseline drift before writing Go persistence/analyzer code.
- Audit metadata remains persisted and readable:
  - `resolutionSource`
  - `confidence`
  - `evidence`
  - `fileHash`
- Duplicate-edge merge behavior remains deterministic.
- Missing endpoints fail closed instead of emitting low-confidence fake edges as exact facts.

### Language Provider Contract

The Go port must preserve the current language matrix. Removing a currently supported language is a
product-contract change outside this conversion plan and blocks cutover until resolved:

- JavaScript
- TypeScript
- Python
- Java
- C
- C++
- C#
- Go
- Ruby
- Rust
- PHP
- Kotlin
- Swift
- Dart
- Vue
- COBOL

Required language-provider invariants:

- Each provider keeps file detection, tree-sitter grammar behavior or regex fallback behavior,
  definition extraction, import/include/use extraction, call extraction, member access extraction,
  inheritance/implements/mixin extraction, type/reference extraction, route/tool/framework facts,
  and unresolved-reference metrics compatible with the local TypeScript baseline.
- Provider parity is measured per language. Full conversion cannot be claimed while an existing
  provider is silently missing.

## Accuracy Policy

The Go port is not accepted if it is faster only because it does less work.

Every phase must distinguish:

```text
real optimization:
  same or better graph facts with less duplicate IO, less duplicate parse, better batching,
  better concurrency, or faster DB load

not acceptable:
  fewer references resolved, skipped scope resolution, skipped access/type/inheritance facts,
  missing audit metadata, missing language providers, or hidden fallback to stale graph data
```

Accuracy gates:

- Fixture parity is exact unless the expected Go behavior is explicitly better and documented.
- Large-repo parity compares node counts, edge counts by type, unresolved references, and sampled
  precision checks.
- Any intentional graph difference must have a written reason and an expected output update.

## Baseline Artifacts

Before each major Go port phase, capture local TypeScript baseline artifacts:

```text
baseline/
  repo-name/
    ts-analyze-metrics.json
    ts-graph-summary.json
    ts-edge-counts.json
    ts-unresolved-references.json
    ts-api-snapshots/
    ts-mcp-snapshots/
```

Go artifacts should mirror the same structure:

```text
baseline/
  repo-name/
    go-analyze-metrics.json
    go-graph-summary.json
    go-edge-counts.json
    go-unresolved-references.json
    go-api-snapshots/
    go-mcp-snapshots/
    parity-report.json
```

Minimum repositories for baselines:

- A tiny fixture repo for deterministic exact parity.
- This repository itself.
- One medium TypeScript/JavaScript repo.
- One mixed-language repo if available.
- The user's larger known repo such as `Restaurant_manager` when ready for performance validation.

## Phase 1 - Contract Freeze

- [x] Record the local TypeScript baseline commit hash used for comparison.
- [x] Record architecture/runtime-flow reference notes from `F:/avmatrix-main` for the surface being
      frozen, including any mismatch against `F:/AVmatrix-GO`, mismatch classification, and the
      chosen decision for that surface.
- [x] Record the Go toolchain version selected for this repository.
- [x] Verify and record latest stable upstream versions for Go, tree-sitter core, required
      tree-sitter grammars, LadybugDB integration, MCP protocol support, HTTP/router/SSE/session
      runtime, embeddings/search runtime, and launcher/packaging tooling.
- [x] For every selected dependency that is not the latest stable release, record the tested latest
      version, failure reason, fallback version, and mitigation plan.
- [x] Export current CLI command list, flags, defaults, env vars, exit codes, prompts, and
      expected outputs from this repository.
- [x] Freeze runtime environment variables used by the current code, including repo home/config,
      gitignore bypass, verbose/debug/shadow modes, max process limit, scope-resolution worker
      controls, embeddings provider/model/dimensions/API key, local session/Codex executable and
      execution mode, wiki/LLM config, HuggingFace cache, and ONNX logging/device behavior.
- [x] Export current HTTP route list, methods, query/body schemas, status codes, payload
      snapshots, SSE events, and CORS behavior from this repository.
- [x] Export current MCP tool list, input schemas, output snapshots, resources, resource
      templates, prompts, and stale-index warning behavior from this repository.
- [x] Export current LadybugDB schema, node tables, relationship columns, extension usage,
      FTS/vector setup, and legacy fallback behavior from this repository.
- [x] Export current graph node labels and relationship types from shared constants and from
      generated fixture output.
- [x] Reconcile graph schema authorities before writing Go persistence code:
      `avmatrix-shared/src/graph/types.ts`, `avmatrix-shared/src/lbug/schema-constants.ts`,
      runtime edge emitters, LadybugDB DDL, MCP schema resources, Web UI graph types, and fixture
      output must agree or the mismatch must be explicitly resolved in the contract snapshot.
- [x] Specifically verify whether relationships emitted by runtime code but missing from schema
      constants, including `USES` and `INHERITS`, are required schema members, legacy-only aliases,
      or bugs to remove before Go code is written.
- [x] Wire required runtime-emitted relationship types back into the TypeScript baseline before Go
      code is written: `USES` and `INHERITS` must be present in shared schema constants, exposed
      schema resources/docs, security allowlists, fixtures, and contract tests, or the runtime
      emitters must be changed if later proven to be bugs.
- [x] Fix the current TypeScript baseline MCP impact/context path for `USES` and `INHERITS`:
      impact relation parser/default traversal allowlist, context incoming/outgoing relationship
      queries, exposed MCP schema/help text, confidence floors, and impact/security contract tests.
- [x] Reconcile the schema-constant authority decision for `USES` and `INHERITS` before Go schema
      generation: both are now in `avmatrix-shared` `REL_TYPES`, downstream schema-surface tests,
      MCP impact/context traversal, exposed MCP schema/help text, and security/impact contract
      tests.
- [x] Specifically verify whether node labels present in graph types but missing from schema
      constants, including `Project`, `Package`, `Decorator`, `Import`, and `Type`, are required
      schema members, browser-only types, legacy-only labels, or bugs to remove before Go code is
      written.
- [x] Export Web UI-facing shared contracts from this repository: graph node/relationship payloads,
      pipeline progress, analyze/embed job status, session status/events, repo list entries, and
      error payloads.
- [ ] [P1-CONTRACT-AUTHORITY-REOPENED-2026-05-14] Reopen the Go-owned contract authority decision
      for the full conversion scope. The earlier tick only covered generated Web UI TypeScript
      adapter types; it did not close the broader question of `avmatrix-shared`, legacy
      `avmatrix/src` contract consumers, package scripts, and Node/Vitest harnesses. Phase-jump
      reason: work returns from Phase 17/15 correction back to Phase 1 because contract authority is
      a prerequisite for deciding which non-Web TypeScript/JavaScript can remain as fixture data,
      which must be generated from Go, and which must be removed from the final cutover package.
- [x] Export current supported language matrix and per-language fixture coverage.
- [x] Export current metrics JSON shape.
- [x] Export current repo registry/meta shape.
- [x] Export the exact analyzer phase list from runtime code, including scan, structure,
      markdown, COBOL, parse, routes, tools, ORM, cross-file binding, resolution, MRO,
      communities, and processes.
- [x] Prove a minimal Go LadybugDB load/read path on Windows before broad analyzer work.
- [x] Prove a minimal Go tree-sitter parse path on Windows before broad provider work.
- [x] Add a contract test folder in `avmatrix-GO`.

Exit gate:

- Contract snapshots exist before Go code starts replacing behavior.
- Early LadybugDB and tree-sitter feasibility proofs exist, or the plan records a blocker before
  implementation continues.
- Dependency freshness evidence exists before Go dependencies are pinned beyond feasibility
  prototypes.

## Phase 2 - Go Skeleton

- [x] Create `go.mod`.
- [x] Add `cmd/avmatrix`.
- [x] Add `internal/version`.
- [x] Add structured logging.
- [x] Add CLI argument parsing.
- [x] Implement `avmatrix version`.
- [x] Implement stub `avmatrix help`.
- [x] Add unit test harness.
- [x] Add Windows-compatible path test helpers.
- [x] Add CI/local test command documentation.

Exit gate:

- During TypeScript/Node coexistence, `go test ./cmd/... ./internal/...` passes. Root
  `go test ./...` becomes mandatory after legacy analyzer fixtures and dependency trees are removed
  or isolated from the Go module path during cutover.
- `go run ./cmd/avmatrix version` works.

## Phase 3 - Repo Registry And Local Path Policy

- [x] Port local path validation.
- [x] Preserve rejection of remote URLs for local analyze.
- [x] Preserve rejection of UNC/network-share paths if the current contract requires it.
- [x] Implement repo-local `.avmatrix` path resolution.
- [x] Implement registry read/write.
- [x] Implement meta.json read/write.
- [x] Implement storage path compatibility.
- [x] Implement repo identity helpers: path-first matching, runtime ID generation, duplicate-name
      labels.
- [x] Add duplicate basename/name tests.
- [x] Add same-path tests for Windows case-insensitive comparison.

Exit gate:

- Go registry/meta output matches TypeScript output for a sample repo.
- Path-first identity prevents name-collision routing.

## Phase 4 - HTTP API Shell

- [x] Implement local HTTP server.
- [x] Implement CORS policy compatible with the current backend.
- [x] Implement `/api/info`.
- [x] Preserve `/api/info` response compatibility for the existing Web UI, including legacy
      fields such as `launchContext` and `nodeVersion` if the current UI still reads them.
- [x] Implement `/api/repos`.
- [x] Implement `/api/repo`.
- [x] Implement path-first repo resolution.
- [x] Implement placeholder `/api/graph` response for empty graph.
- [x] Implement API error shape compatibility.
- [x] Add API snapshot tests.

Exit gate:

- Existing Web UI can connect to the Go server enough to show repo list and repo info.

## Phase 4A - Go-Aware Launcher Build Gate

- [x] Rewrite `avmatrix-launcher/build.ps1` so the full launcher build compiles the Go backend
      binary from `cmd/avmatrix` instead of rebuilding the TypeScript CLI as the launcher backend.
- [x] Keep Web UI build in TypeScript/React because the Web UI display layer remains TypeScript.
- [x] Build the launcher backend wrapper as a Go process that starts the packaged Go backend with
      `serve --host 127.0.0.1 --port 4747`.
- [x] Remove bundled `node.exe` from the normal launcher backend package.
- [x] Update launcher reset/stop process sweep so it targets packaged Go backend processes instead
      of bundled Node.
- [x] Update active testing/runtime docs so the full launcher build gate reflects the Go runtime
      package.

Exit gate:

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` produces
  `AVmatrixLauncher.exe`, `server-bundle\avmatrix-server.exe`,
  `server-bundle\avmatrix.exe`, and `web-dist\`, without `server-bundle\node.exe`.

## Phase 5 - Scanner And Ignore Semantics

- [x] Port filesystem walker.
- [x] Port language detection.
- [x] Port skip rules for vendor/build/cache folders.
- [x] Port `.gitignore` handling.
- [x] Preserve `.avmatrixignore` handling and `AVMATRIX_NO_GITIGNORE` behavior.
- [x] Port explicit include/exclude behavior.
- [x] Preserve hardcoded ignored directories/files/extensions and the current 512 KB file-size
      scan cutoff unless Phase 1 records an intentional contract change.
- [x] Compute file hashes during scan.
- [x] Emit scan metrics.
- [x] Add fixture parity tests for included/skipped files.
- [x] Benchmark scan phase against the local TypeScript baseline.

Exit gate:

- Go and TypeScript select the same files for deterministic fixtures.
- Large-repo scan is not slower without a documented reason.

## Phase 6 - Parser Runtime

- [x] Choose Go tree-sitter binding.
- [x] Prove Windows build/install path.
- [x] Add grammar loading strategy.
- [x] Add parser pool.
- [x] Preserve canonical worker/pool behavior and failure semantics; do not add a hidden sequential
      fallback unless a contract snapshot explicitly changes that behavior.
- [x] Add per-file parse timeout/error handling.
- [x] Add parse result model.
- [x] Add parser metrics.
- [x] Prove parse worker does not reread files after scan/parse handoff unless required.
- [x] Add fixture tests for parse success/failure.
- [x] Reconcile tree-sitter upstream/core freshness before final cutover: upstream tree-sitter
      `v0.26.8` must be represented by the Go parser runtime path, or the plan must record an
      explicit blocker proving why the latest Go-compatible binding cannot yet use that runtime.
      Checked on 2026-05-13: upstream `tree-sitter/tree-sitter` latest release is `v0.26.8`, but
      `go list -m -json github.com/tree-sitter/go-tree-sitter@latest` resolves the official Go
      binding to `v0.25.0`; `go list -m -versions` lists only `v0.23.0`, `v0.23.1`, `v0.24.0`,
      and `v0.25.0`; the repository `master` pseudo-version is older than `v0.25.0`; no official
      Go-compatible `v0.26.x` binding is available for this parser runtime path today. Keep
      `github.com/tree-sitter/go-tree-sitter v0.25.0` as the latest Go-compatible binding and
      re-check this item at final cutover.

Exit gate:

- Parser runtime can parse at least TypeScript and JavaScript fixtures.
- Parse errors are reported but do not crash full analyze.

## Phase 7 - ScopeIR Model

- [x] Define Go structs for `ScopeIR`.
- [x] Define `DefinitionFact`.
- [x] Define `ImportFact`.
- [x] Define `CallSiteFact`.
- [x] Define `AccessFact`.
- [x] Define `HeritageFact`.
- [x] Define `ScopeFact`.
- [x] Define type annotation and return type facts.
- [x] Define framework/domain facts if currently emitted.
- [x] Add JSON serialization tests.
- [x] Add golden fixture output tests.

Exit gate:

- ScopeIR can be serialized deterministically and compared against golden output.

## Phase 8 - TypeScript/JavaScript Provider

Port TypeScript/JavaScript first because it exercises most of the accurate single-pass work.

- [x] Extract files/modules.
- [x] Extract function/class/method/interface definitions.
- [x] Preserve owner-qualified member names.
- [x] Extract imports and export aliases.
- [x] Extract constructor calls.
- [x] Extract function/method calls.
- [x] Extract member reads/writes.
- [x] Extract inheritance/implements facts.
- [x] Extract type annotations.
- [x] Extract return types.
- [x] Extract local variable type bindings from same-file return annotations.
- [x] Extract interface property signatures.
- [x] Extract type-alias RHS references.
- [x] Preserve file hashes on facts.
- [x] Add exact ScopeIR parity fixture.
- [x] Add exact graph parity fixture for current relationship types used by TypeScript/JavaScript,
      including `CALLS`, `IMPORTS`, `ACCESSES`, `EXTENDS`, `IMPLEMENTS`, `HAS_METHOD`,
      `HAS_PROPERTY`, `METHOD_OVERRIDES`, `METHOD_IMPLEMENTS`, and any route/tool/process edges
      emitted by the local baseline.

Exit gate:

- TypeScript/JavaScript fixture parity passes.
- Go does not reread or reparse source for scope extraction after parse.

## Phase 9 - Resolution Phase

- [x] Build global symbol table.
- [x] Build import graph.
- [x] Build scope tree/index.
- [x] Build reference index.
- [x] Build owner-member index.
- [x] Build method dispatch index.
- [x] Resolve imports.
- [x] Preserve finalized file-level `IMPORTS` and per-symbol import-use edge emission.
- [x] Resolve constructor references.
- [x] Resolve call targets.
- [x] Resolve member access targets.
- [x] Resolve type annotation references.
- [x] Resolve inheritance references.
- [x] Resolve imported type aliases.
- [x] Resolve receiver method dispatch.
- [x] Preserve cross-file binding accumulator lifecycle and the
      `--skip-compatibility-cross-file` diagnostic behavior.
- [x] Emit graph edges once through the unified resolution path.
- [x] Attach audit metadata to scope-resolved edges.
- [x] Merge audit metadata into semantic duplicate edges.
- [x] Add duplicate-edge tests.
- [x] Add unresolved reference metrics.

Exit gate:

- TypeScript/JavaScript accurate single-pass graph parity passes.
- No second source read or second AST parse is introduced for resolution.

## Phase 10 - LadybugDB Persistence

- [x] Verify Go can load/read LadybugDB with required extensions.
- [x] If direct Go support is blocked, document the blocker before selecting any bridge.
- [x] Port schema creation.
- [x] Port bulk CSV/COPY load.
- [x] Port node load.
- [x] Port relationship load.
- [x] Preserve relationship CSV splitting by source/target label pair for LadybugDB COPY.
- [x] Keep fallback insert behavior out of the required runtime contract. It may exist only as a
      recovery/legacy diagnostic path for explicitly unsupported schema-pair gaps or injected COPY
      failures; normal supported relationships must stay on the COPY path with
      `FallbackInsertCount == 0`.
- [x] Reassess fallback correctness after Phase 15 work exposed the question of silent bad data.
      Phase-jump note: this intentionally jumped from the open Phase 15 `context` optimization
      slice back to Phase 10 persistence correctness because fallback can make a DB load look
      successful while relationships are dropped, duplicated, or semantically mutated. Normal
      `LoadCSVExport` now fails closed on relationship COPY failure, unsupported schema-pair COPY,
      skipped relationships, and any fallback insert failure. Fallback and `IGNORE_ERRORS=true`
      retry are available only through explicit diagnostic `LoadCSVExportWithOptions` flags.
      Validation followed the plan rule: full launcher build first (`34,207.9ms`), lbugload tests
      and benchmarks, packaged current-repo analyze (`33,573` nodes / `66,603` relationships,
      DB load fallback/skipped `0`), full Go tests (`27,772.9ms`), browser E2E through the packaged
      Go backend (`32` passed / `1` skipped, `512,685.5ms`), and `cd avmatrix && npm test`
      (`438,204.3ms`).
- [x] [P10-TYPEALIAS-METHOD-SCHEMA-GAP] Fix fail-closed LadybugDB schema gap found through Web UI
      repo selection on `F:\Restaurant_manager`: analyze failed with
      `db_load phase: copy relationships TypeAlias->Method: schema pair unsupported`.
      Phase-jump note: this intentionally jumped from the active Phase 15 performance work back to
      Phase 10 persistence/schema correctness because the failure blocks normal runtime DB load for
      a real repo; it is not an optimization issue. The correct fix is not fallback or skipped
      relationships: the schema must support the graph shape directly and keep
      `fallbackInsertFailures=0` plus `skippedRelationships=0`. Impact: AVmatrix reported
      `RelationPairs` LOW, while `relationPairSupported` was CRITICAL through
      `ExportGraphCSVs -> loadGraph -> Run -> newAnalyzeCommand` and Web analyze paths.
      Implementation: add the `TypeAlias -> Method` relation pair to the LadybugDB schema and add
      regression coverage proving `TypeAlias -> Method` uses normal relationship COPY with no
      fallback insert. Benchmark/evidence: before patch, packaged analyze on `F:\Restaurant_manager`
      failed closed in `25,374.4ms`; after the patch and full launcher build, the same repo analyzed
      successfully in `30,931.7ms` wall / `28,889.6ms` benchmark total, with `6,198` scanned files,
      `1,228` parsed files, `4,970` unsupported files, `77,901` nodes, `129,560` relationships,
      and DB fallback/skipped `0`. Validation followed the plan rule: full launcher build first
      (`56,762.9ms` after stopping the stale launcher/backend process that held the bundle lock),
      focused schema/load tests (`4,284.6ms`), after-build loader benchmarks
      (`ExportGraphCSVs` `16.05-22.24ms/op`; `BenchmarkLoadCSVExportCopyPathNoop`
      `2.87-3.28us/op`, `1,360-1,376 B/op`, `17 allocs/op`), full Go tests (`61,720.4ms`),
      browser E2E through packaged Go backend (`32` passed / `1` skipped; isolated analyze
      `27,291.0ms`, Playwright `475,727.0ms`), and `cd avmatrix && npm test`
      (`440,065.0ms`).
- [x] Persist audit metadata columns.
- [x] Document legacy schema compatibility separately from the primary write path. Legacy read
      compatibility may be implemented only if current repository indexes require it; fallback
      insert throughput is not an acceptance gate.
- [x] Implement read pool/concurrency rules.
- [x] Preserve read-only query guard and stdout-silencing/stdio safety for MCP read paths.
- [x] Preserve FTS and VECTOR extension lifecycle, embedding table/index shape, and stale
      embedding cache behavior.
- [x] Implement lock/error handling.
- [x] Add DB persistence/readback tests.
- [x] Add graph stream read tests.

Exit gate:

- Go can write an index that the Go HTTP API can read.
- If compatibility with existing TypeScript-generated indexes is required, Go can read them.

## Phase 11 - Analyze Pipeline

- [x] Implement phase orchestration.
- [x] Implement the final runtime phase chain: scan -> structure -> markdown/documents -> COBOL -> parse ->
      routes -> tools -> ORM -> cross-file binding -> resolution -> MRO -> communities ->
      processes -> DB load.
  - [x] Reorder the implemented Go runtime chain to `scan -> structure -> documents -> COBOL ->
        parse -> routes -> tools -> ORM -> cross_file_binding -> resolution -> MRO ->
        communities -> processes -> DB load`, with base `File` node/path ownership
        bootstrapped by structure before later enrichments attach to it.
  - [x] Split cross-file binding into an explicit phase before resolution. The Go runtime now
        emits `cross_file_binding` phase timing and benchmark metrics, builds/finalizes the
        resolution binding workspace there, then feeds it into `resolution`.
  - [x] Add Go structure enrichment phase that emits `Folder` nodes and `CONTAINS`
        relationships from scanned repository paths.
  - [x] Add Go document enrichment phase that handles Markdown sections/links plus metadata for
        `.doc`, `.docx`, `.pdf`, and Excel-family files (`.xls`, `.xlsx`, `.xlsm`, `.xlsb`,
        templates, `.ods`, `.csv`, `.tsv`) without treating binary formats as text.
  - [x] Add Go COBOL/JCL enrichment phase that scans `.cob`, `.cbl`, `.cobol`, `.cpy`,
        `.copybook`, `.jcl`, `.job`, and `.proc` files, emits COBOL module/section/paragraph
        structure, COPY imports, PERFORM/CALL edges, and JCL EXEC program links.
  - [x] Add Go route enrichment phase that emits Next.js filesystem routes, framework route
        registrations, `HANDLES_ROUTE` edges, and local `fetch('/path')` `FETCHES` edges.
  - [x] Add Go tool enrichment phase that emits MCP/RPC tool definitions from object definitions,
        `.tool(...)` registrations, decorator tools, and `HANDLES_TOOL` edges.
  - [x] Add Go ORM enrichment phase that detects Prisma and Supabase query calls, emits
        `QUERIES` edges from file nodes to existing model nodes when uniquely matched, and creates
        fallback `CodeElement` model/table nodes only when no unambiguous model node exists.
  - [x] Add Go MRO enrichment phase that walks `EXTENDS`/`IMPLEMENTS` plus `HAS_METHOD`,
        materializes graph-level `METHOD_OVERRIDES` for ancestor method collisions, and emits
        `METHOD_IMPLEMENTS` edges for concrete interface/trait matches.
  - [x] Add Go community enrichment phase that emits `Community` nodes and `MEMBER_OF`
        relationships before DB load.
  - [x] Add Go process enrichment phase that emits `Process` nodes plus `ENTRY_POINT_OF` and
        `STEP_IN_PROCESS` relationships before DB load.
  - [x] Replace deterministic community heuristic with the final parity algorithm required for the
        current cutover slice: Go no longer uses connected-components as the community partition.
        It now builds the same symbol/clustering graph shape as the TypeScript community phase and
        runs a deterministic modularity local-move partition, preserving singleton membership edges
        while only materializing non-singleton `Community` nodes.
  - [x] [P11-COMMUNITY-PARITY] Close the Phase 11 TypeScript baseline comparison for the current
        analyze/community/process graph slice reached from Phase 14: Go and TypeScript both produce
        `Community=3`, `MEMBER_OF=9`, `Process=1`, and `STEP_IN_PROCESS=3` on
        `.tmp\resolution-baseline-fixture`. Total node/edge counts still differ because Go emits
        additional resolver facts, but the Phase 11 community/process surface is aligned for this
        return path. After this item, return to `[P14-PYTHON-TO-P11-AFTER-BATCH]` and tick it.
- [x] Preserve AI context, skill generation, wiki gating/status behavior, and benchmark artifact
      generation if those CLI surfaces remain in the product contract.
  - [x] Wire Go CLI `status`, `wiki`, and `wiki-mode` behavior: git/non-git detection,
        indexed/stale status output, stale Kuzu notice, local-only wiki gate, persisted
        `runtime.json` wiki mode, invalid-mode stderr guidance, and silent non-zero wiki exit.
  - [x] Port AI context and skill generation command surfaces: `analyze --skills` forces a fresh
        analysis, writes `.avmatrix/meta.json`, registers the repo in the global registry, creates
        generated community skills, installs AVmatrix base skills, and upserts AVmatrix sections in
        `AGENTS.md`/`CLAUDE.md` with `--skip-agents-md` and `--no-stats` behavior.
  - [x] [P11-AI-CONTEXT-SKILL-GEN-PHASE17-CONVERSION-GUARD-2026-05-14] Record the Phase 17
        conversion guard for agent-facing generated content. Phase-jump note: Phase 17 is allowed
        to process the remaining TypeScript/JavaScript files that used to implement AI context,
        skill generation, analyze side effects, setup, and hook/config installation, but this does
        not reopen Phase 11 by itself because the Go analyze pipeline already owns the accepted
        runtime behavior above. If a Phase 17 conversion/removal slice changes generated
        `AGENTS.md`, `CLAUDE.md`, generated `SKILL.md` content, editor MCP/skills/hooks config,
        section markers, idempotent upsert behavior, skip flags, or path layout, jump back to Phase
        11 with a concrete reopened task, output snapshot evidence, E2E coverage, benchmark entry,
        and commit before returning to Phase 17.
- [x] Add phase timing metrics.
- [x] Add memory metrics where feasible.
- [x] Add benchmark JSON output.
- [x] Add cancellation handling.
- [x] Add progress events.
- [x] Add force/full analyze behavior.
- [x] Port current embeddings behavior for CLI and server paths, including local model mode, HTTP
      embedding mode, dimensions validation, incremental stale-cache handling, and progress events.
  - [x] Add Go HTTP embedding config/client boundary matching the current product contract:
        default model/dimensions/text limits, `AVMATRIX_EMBEDDING_URL`,
        `AVMATRIX_EMBEDDING_MODEL`, `AVMATRIX_EMBEDDING_API_KEY`,
        `AVMATRIX_EMBEDDING_DIMS`, OpenAI-compatible `/embeddings` requests, batch splitting,
        retryable status handling, vector-count validation, dimension mismatch errors, and safe
        URL error messages.
  - [x] Add Go embedding pipeline core for graph-backed embedding runs: embeddable label
        selection, metadata text generation, deterministic content hashes, character chunk
        fallback, fresh-cache skip, stale-row delete intent, `CodeEmbedding` insert intent,
        vector extension/index lifecycle, dimension guard, and progress/error events.
  - [x] Wire `avmatrix analyze --embeddings` into the Go analyze runtime for HTTP embedding mode:
        native DB runner required, HTTP env dimensions honored, stale hash readback supported when
        the runner exposes row reads, embedding phase metrics recorded, HTTP mode preferred when
        configured, and local model mode resolved through the Go Hugot runtime.
  - [x] Add Go semantic vector search runtime: query embedding, vector-index query construction,
        chunk deduplication by best distance, metadata hydration, limit handling, and query-vector
        dimension guard.
  - [x] Wire server embed/search paths into the Go embedding runtime.
    - [x] Wire `/api/search` POST into the Go semantic vector search runtime with Web UI-compatible
          `{results}` shape, body/query repo resolution, limit clamping, semantic/hybrid mode
          acceptance, and explicit unavailable errors when the native DB read runner or local model
          mode is not available.
    - [x] Wire `/api/embed`, `/api/embed/{jobId}`, `/api/embed/{jobId}/progress`, and
          `DELETE /api/embed/{jobId}` into the Go embedding runtime with repo lock reuse,
          graph-snapshot loading, native DB write runner execution, incremental hash reuse,
          SSE terminal events, cancellation, timeout, and Web UI-compatible job/progress shapes.
  - [x] Wire local model mode and server progress lifecycle into the Go CLI/server runtime.
    - [x] Wire server embedding job progress lifecycle for start/status/SSE/cancel/complete/fail.
    - [x] Wire local embedding model mode into Go CLI/server runtime.
- [x] Add repo lock to prevent concurrent writers on the same repo.
- [x] Add failure cleanup behavior.

Exit gate:

- `avmatrix analyze <fixture>` produces a readable graph in Go.
- Phase metrics are emitted before any speed claim is made.

## Phase 12 - HTTP Analyze And Graph Serving

- [x] Implement `/api/analyze`.
- [x] Implement analyze job manager.
- [x] Implement SSE progress stream.
- [x] Implement analyze completion payload with `repoName` and `repoPath`.
- [x] Implement `/api/graph` JSON and NDJSON streaming.
- [x] Implement `/api/repo` hold-queue behavior if analyze is active.
- [x] Preserve loopback-only CORS/private-network behavior.
- [x] Preserve repo-scoped analyze/embed locks so two writers cannot mutate the same repo index.
- [x] Preserve analyze lifecycle semantics that still exist in the Go runtime: same-repo active-job
      dedupe, cancel, timeout, cleanup, and lock release behavior. Record that child-worker crash
      retry has no in-process Go analyze surface unless a future out-of-process worker is added.
- [x] Implement graph read endpoints used by Web UI panels.
- [x] Add Web API contract tests.
- [x] Run existing Web UI against Go backend.

Exit gate:

- Existing Web UI can analyze a repo through Go backend and render the resulting graph.
- Web analyze path remains selected path -> analyze path -> graph path.

## Phase 13 - MCP Server

- [x] Implement MCP stdio server.
- [x] Implement MCP HTTP endpoint compatibility for `/api/mcp`.
- [x] Implement repo context resource.
- [x] Implement clusters/processes resources.
- [x] Implement resource templates and prompts exposed by the local baseline.
- [x] Implement `list_repos`.
- [x] Implement `query`.
- [x] Implement `cypher`.
- [x] Implement `context`.
- [x] Implement `impact`.
- [x] Implement `detect_changes`.
- [x] Implement `rename`.
- [x] Implement `route_map`.
- [x] Implement `tool_map`.
- [x] Implement `shape_check`.
- [x] Implement `api_impact`.
- [x] Implement `group_list`.
- [x] Implement `group_sync`.
- [x] Implement `group_contracts`.
- [x] Implement `group_query`.
- [x] Implement `group_status`.
- [x] Preserve MCP stdio stdout safety so database/native logs cannot corrupt JSON-RPC output.
- [x] Preserve MCP HTTP StreamableHTTP session behavior and session TTL.
- [x] Preserve next-step hint behavior appended to tool responses.
- [x] Add MCP snapshot tests against the local TypeScript baseline.
- [x] Add stale-index warning behavior.
- [x] [P13-MCP-BENCHMARK-CLASSIFY] Classify the Phase 13 MCP benchmark regressions after a provider
      batch changes graph shape. When reached from `[P14-GO-TO-P13-BEFORE-PYTHON]`, decide here
      whether `route_map`, `context`, `impact`, and HTTP `group_sync` are immediate Phase 13
      blockers or Phase 15 optimization work. If they are immediate blockers, fix them in Phase 13,
      then return to `[P14-GO-TO-P13-BEFORE-PYTHON]` and tick it. If they are Phase 15 optimization
      work, jump to `[P15-MCP-OPTIMIZATION-REGRESSIONS]` to record that target, then return to
      `[P14-GO-TO-P13-BEFORE-PYTHON]` and tick it.
      Use this benchmark the same way all benchmarks in this plan are used: to find where the
      current Go path is below expected Go runtime potential and to create concrete optimization
      work. The observed strong paths are `initialize`, `query`, smaller `tools/list` payload, and
      clean protocol noise. The observed optimization candidates are `context`, `impact`,
      `route_map`, and HTTP `group_sync`. Record optimization backlog for those candidates without
      dropping correctness, provider coverage, contract parity, or required runtime work.
      Classified after the Python provider batch: these are not immediate Phase 13 correctness
      blockers because the MCP runtime reject fixes are closed and the affected tools return valid
      contract-shaped results. They are mandatory Phase 15 optimization work with fixed priority:
      `P0 route_map`, `P1 context`, `P1 impact`, `P2 HTTP group_sync`, `P3 preserve initialize/query
      wins, smaller `tools/list`, and zero protocol-noise bytes. Return to
      `[P14-PYTHON-TO-P13-AFTER-BATCH]` and tick it.

Exit gate:

- MCP stdio and HTTP discovery expose the same required tool/resource/prompt surface as the local
  baseline.
- MCP tools return equivalent results on fixture repos.
- MCP correctness may be considered runtime-complete only after correctness rejects are closed. MCP
  performance is not complete while `route_map`, `context`, `impact`, or HTTP `group_sync`
  regressions remain open in Phase 15.

## Phase 14 - Additional Language Providers

Port providers one by one. Do not claim full conversion until the language matrix is clear.

Suggested order:

- [x] Go.
- [x] Python.
- [x] Java.
- [x] Kotlin.
- [x] C.
- [x] C#.
- [x] C++.
- [x] Rust.
- [x] PHP.
- [x] Dart.
- [x] Vue.
- [x] Swift.
- [x] Ruby.
- [x] COBOL.
- [x] Framework-specific facts currently supported by the TypeScript implementation.

For each provider:

- [x] Extract definitions across every Phase 14 provider batch listed below.
- [x] Extract imports/includes/usings/packages across every Phase 14 provider batch listed below.
- [x] Extract calls across every Phase 14 provider batch listed below.
- [x] Extract member accesses across every Phase 14 provider batch listed below.
- [x] Extract inheritance/implements/mixins across every Phase 14 provider batch listed below, or
      explicitly record when the language has no equivalent surface.
- [x] Extract type references where available across every Phase 14 provider batch listed below, or
      explicitly record when the language has no static type-reference surface.
- [x] Add exact fixture ScopeIR parity for parser-backed provider batches, with COBOL/JCL recorded
      as direct graph enrichment rather than ScopeIR parser output.
- [x] Add graph parity for edge counts by type across every Phase 14 provider batch listed below.
- [x] Add unresolved reference metrics across every parser-backed Phase 14 provider batch listed
      below, with COBOL/JCL recorded as not applicable because it does not emit ScopeIR references.

Go provider batch:

- [x] Extract definitions: package, struct, interface, function, method, property, variable, and
      const facts.
- [x] Extract imports/packages: standard imports, named package imports, alias imports, dot imports,
      and blank imports are represented as ScopeIR imports.
- [x] Extract calls: free calls, member calls, and composite-literal constructor calls.
- [x] Extract member accesses: selector read/write facts, including chained receiver text such as
      `s.repo`.
- [x] Extract inheritance/implements/mixins: Go embedded struct/interface declarations emit
      heritage facts.
- [x] Extract type references where available: parameters, receivers, fields, return types, aliases,
      embedded types, and inferred local assignment/composite-literal types.
- [x] Add exact fixture ScopeIR parity:
      `internal/providers/golang/testdata/go_scopeir_signature.golden.json`.
- [x] Add graph parity for edge counts by type:
      `ACCESSES=2`, `CALLS=2`, `DEFINES=14`, `EXTENDS=2`, `HAS_METHOD=3`,
      `HAS_PROPERTY=4`, `INHERITS=2`, `USES=8`.
- [x] Add unresolved reference metrics: Go fixture records `UnresolvedReferences=4` for external
      package/type references intentionally outside the local fixture graph.
- [x] Record why the Go provider batch was allowed before closing Phase 11/13 residuals: language
      providers are upstream graph fact producers, while Phase 11 community/process parity and Phase
      13 MCP graph-tool performance consume the produced graph.

Python provider batch:

- [x] Extract definitions: class, function, method, property, and local variable facts.
- [x] Extract imports/packages: named imports and alias imports are represented as ScopeIR imports.
- [x] Extract calls: constructor calls and member calls.
- [x] Extract member accesses: attribute read/write facts with receiver text such as `self.repo`.
- [x] Extract inheritance/implements/mixins: class base lists emit heritage facts.
- [x] Extract type references where available: parameter annotations, class property annotations,
      return annotations, and constructor-inferred local assignment types.
- [x] Add exact fixture ScopeIR parity:
      `internal/providers/python/testdata/python_scopeir_signature.golden.json`.
- [x] Add graph parity for edge counts by type:
      `ACCESSES=2`, `CALLS=2`, `DEFINES=7`, `EXTENDS=1`, `HAS_METHOD=2`,
      `HAS_PROPERTY=1`, `INHERITS=1`, `USES=1`.
- [x] Add unresolved reference metrics: Python fixture records external import/type/member
      references as unresolved when they are intentionally outside the local fixture graph.
- [x] Keep unsupported-language parser coverage by moving that stale expectation from Python to
      Ruby after Python grammar wiring.
- [x] Benchmark Python provider parse plus extraction and record the baseline in the benchmark
      companion file.
- [x] Run full runtime E2E through the launcher-built Go backend and Web UI on an isolated Python
      fixture: CLI analyze produced a readable graph, and Playwright completed with `32 passed`,
      `1 skipped`.
- [x] [P14-PYTHON-TO-P11-AFTER-BATCH] After the Python provider batch is complete, jump to
      `[P11-COMMUNITY-PARITY]`, close the full analyze/community/process baseline comparison for the
      current graph slice, then return to this item and tick it. If Phase 11 records a blocker, keep
      the next provider batch blocked here.
      Done for the current graph slice: Phase 11 replaced the connected-component community
      heuristic with modularity partitioning, re-ran Go/TypeScript baseline comparison on
      `.tmp\resolution-baseline-fixture`, and matched community/process counts.
- [x] [P14-PYTHON-TO-P13-AFTER-BATCH] After the Python provider batch is complete, jump to
      `[P13-MCP-BENCHMARK-CLASSIFY]`, decide whether `route_map`, `context`, `impact`, and HTTP
      `group_sync` are immediate Phase 13 blockers or optimization backlog, then return to this item
      and tick it. If Phase 13 records a blocker, keep the next provider batch blocked here.
- [x] Do not open the provider batch after Python until the two jump-back items above are checked in
      this Phase 14 checklist.
      Done: Phase 11 community/process parity and Phase 13 MCP benchmark classification are both
      checked, so the next provider batch may open.

Java provider batch:

- [x] Extract definitions: package, interface, class, constructor, method, property, and local
      variable facts.
- [x] Extract imports/packages: Java package declarations, normal imports, and static imports are
      represented as ScopeIR package/import facts.
- [x] Extract calls: method invocations with receiver text and arity, including same-file local
      calls and chained receiver calls.
- [x] Extract member accesses: field access write/read facts with receiver text such as
      `this.repo`.
- [x] Extract inheritance/implements/mixins: `extends` and `implements` clauses emit heritage facts.
- [x] Extract type references where available: fields, constructor parameters, method parameters,
      return types, local variable declarations, and inferred local assignments.
- [x] Add exact fixture ScopeIR parity:
      `internal/providers/java/testdata/java_scopeir_signature.golden.json`.
- [x] Add graph parity for edge counts by type:
      `ACCESSES=1`, `CALLS=3`, `DEFINES=14`, `EXTENDS=1`, `HAS_METHOD=6`,
      `HAS_PROPERTY=1`, `IMPLEMENTS=1`, `INHERITS=2`, `METHOD_IMPLEMENTS=1`, `USES=5`.
- [x] Add unresolved reference metrics: Java fixture records unresolved external import/static call
      references while local class, method, inheritance, implementation, and type-reference edges
      resolve.
- [x] Wire `.java` files into the parser registry and analyze provider dispatch using
      `github.com/tree-sitter/tree-sitter-java v0.23.5`.
- [x] Benchmark Java provider parse plus extraction and record the baseline in the benchmark
      companion file.
- [x] Run full runtime E2E through the launcher-built Go executable on an isolated Java fixture:
      CLI analyze produced a readable graph with Java class/interface/method/package nodes and
      `CALLS`, `USES`, `INHERITS`, `METHOD_IMPLEMENTS`, community, and process edges.

Kotlin provider batch:

- [x] Extract definitions: package, interface, class, primary constructor, method, property, and
      local variable facts.
- [x] Extract imports/packages: Kotlin package headers and imports are represented as ScopeIR
      package/import facts.
- [x] Extract calls: direct calls, receiver calls, nested receiver calls, and constructor-shaped
      uppercase calls.
- [x] Extract member accesses: navigation-expression access facts with receiver text such as
      `this.repo`.
- [x] Extract inheritance/implements/mixins: delegation specifiers emit `extends` for constructor
      delegation and `implements` for interface delegation.
- [x] Extract type references where available: class constructor parameters, method parameters,
      return types, local properties, and inferred local assignments.
- [x] Add exact fixture ScopeIR parity:
      `internal/providers/kotlin/testdata/kotlin_scopeir_signature.golden.json`.
- [x] Add graph parity for edge counts by type:
      `ACCESSES=1`, `CALLS=3`, `DEFINES=14`, `EXTENDS=1`, `HAS_METHOD=6`,
      `HAS_PROPERTY=1`, `IMPLEMENTS=1`, `INHERITS=2`, `METHOD_IMPLEMENTS=1`, `USES=4`.
- [x] Add unresolved reference metrics: Kotlin fixture records unresolved external import/member
      references while local class, method, inheritance, implementation, and type-reference edges
      resolve.
- [x] Wire `.kt`/`.kts` files into the parser registry and analyze provider dispatch using
      `github.com/tree-sitter-grammars/tree-sitter-kotlin v1.1.0`.
- [x] Benchmark Kotlin provider parse plus extraction and record the baseline in the benchmark
      companion file.
- [x] Run full runtime E2E through the launcher-built Go executable on an isolated Kotlin fixture:
      CLI analyze produced a readable graph with Kotlin class/interface/method/package nodes and
      `CALLS`, `USES`, `INHERITS`, `METHOD_IMPLEMENTS`, community, and process edges.

C provider batch:

- [x] Extract definitions: struct, function, struct field/property, and local variable facts.
- [x] Extract imports/includes/usings/packages: C `#include` directives are represented as ScopeIR
      import facts for quoted and system includes.
- [x] Extract calls: free function calls and receiver-like struct field/function-pointer calls.
- [x] Extract member accesses: `field_expression` reads with receiver text such as `service`.
- [x] Extract inheritance/implements/mixins: no C inheritance surface is emitted; struct composition
      remains type/reference evidence instead of a fake inheritance edge.
- [x] Extract type references where available: struct fields, function parameters, function returns,
      and local declarations.
- [x] Add exact fixture ScopeIR parity:
      `internal/providers/c/testdata/c_scopeir_signature.golden.json`.
- [x] Add graph parity for edge counts by type:
      `ACCESSES=1`, `CALLS=1`, `DEFINES=8`, `HAS_PROPERTY=3`, `USES=2`.
- [x] Add unresolved reference metrics: C fixture records unresolved external include/stdlib call
      references while local function, field access, and struct type-reference edges resolve.
- [x] Wire `.c` files into the parser registry and analyze provider dispatch using
      `github.com/tree-sitter/tree-sitter-c v0.24.2`.
- [x] Benchmark C provider parse plus extraction and record the baseline in the benchmark
      companion file.
- [x] Run full runtime E2E through the launcher-built Go executable on an isolated C fixture:
      CLI analyze produced a readable graph with C struct/function/property/variable nodes and
      `CALLS`, `ACCESSES`, `USES`, `HAS_PROPERTY`, and community edges.

C# provider batch:

- [x] Extract definitions: namespace/package, interface, class, constructor, method, field/property,
      and local variable facts.
- [x] Extract imports/includes/usings/packages: C# `using` directives and file-scoped namespace
      declarations are represented as ScopeIR import/package facts.
- [x] Extract calls: free/local calls, member calls, and nested receiver calls.
- [x] Extract member accesses: member access read/write facts with receiver text such as
      `this.repo`.
- [x] Extract inheritance/implements/mixins: C# base lists emit `extends` and `implements`
      heritage facts.
- [x] Extract type references where available: constructor parameters, method parameters, fields,
      return types, and local declarations.
- [x] Add exact fixture ScopeIR parity:
      `internal/providers/csharp/testdata/csharp_scopeir_signature.golden.json`.
- [x] Add graph parity for edge counts by type:
      `ACCESSES=2`, `CALLS=3`, `DEFINES=14`, `EXTENDS=1`, `HAS_METHOD=6`,
      `HAS_PROPERTY=1`, `IMPLEMENTS=1`, `INHERITS=2`, `METHOD_IMPLEMENTS=1`, `USES=5`.
- [x] Add unresolved reference metrics: C# fixture records unresolved external namespace/type/member
      references while local class, method, inheritance, implementation, and type-reference edges
      resolve.
- [x] Wire `.cs` files into the parser registry and analyze provider dispatch using
      `github.com/tree-sitter/tree-sitter-c-sharp v0.23.5`.
- [x] Benchmark C# provider parse plus extraction and record the baseline in the benchmark
      companion file.
- [x] Run full runtime E2E through the launcher-built Go executable on an isolated C# fixture:
      CLI analyze produced a readable graph with C# class/interface/method/package nodes and
      `CALLS`, `USES`, `INHERITS`, `IMPLEMENTS`, `METHOD_IMPLEMENTS`, community, and process
      edges.

C++ provider batch:

- [x] Extract definitions: namespace/package, class, constructor, method, property, and local
      variable facts.
- [x] Extract imports/includes/usings/packages: C++ `#include` directives and namespaces are
      represented as ScopeIR import/package facts.
- [x] Extract calls: local/free calls and member calls with explicit receiver text.
- [x] Extract member accesses: field expression read facts with receiver text such as `this->repo`.
- [x] Extract inheritance/implements/mixins: C++ base class clauses emit heritage facts; multiple
      base classes are represented as `extends` edges instead of a fake interface split.
- [x] Extract type references where available: constructor parameters, method parameters, fields,
      return types, and local declarations.
- [x] Add exact fixture ScopeIR parity:
      `internal/providers/cpp/testdata/cpp_scopeir_signature.golden.json`.
- [x] Add graph parity for edge counts by type:
      `ACCESSES=1`, `CALLS=1`, `DEFINES=13`, `EXTENDS=2`, `HAS_METHOD=5`,
      `HAS_PROPERTY=2`, `INHERITS=2`, `METHOD_OVERRIDES=1`, `USES=2`.
- [x] Add unresolved reference metrics: C++ fixture records unresolved external include/std/member
      references while local method, access, inheritance, override, and type-reference edges
      resolve.
- [x] Wire `.cpp`/`.cc`/`.cxx`/`.h`/`.hpp`/`.hxx`/`.hh` files into the parser registry and analyze
      provider dispatch using `github.com/tree-sitter/tree-sitter-cpp v0.23.4`.
- [x] Benchmark C++ provider parse plus extraction and record the baseline in the benchmark
      companion file.
- [x] Run full runtime E2E through the launcher-built Go executable on an isolated C++ fixture:
      CLI analyze produced a readable graph with C++ class/method/package/property nodes and
      `CALLS`, `ACCESSES`, `USES`, `INHERITS`, `METHOD_OVERRIDES`, community, and process edges.

Rust provider batch:

- [x] Extract definitions: module/package, trait, struct, method, property, and local variable
      facts.
- [x] Extract imports/includes/usings/packages: Rust `use` declarations and modules are represented
      as ScopeIR import/package facts.
- [x] Extract calls: constructor-shaped struct expressions, free calls, and member calls with
      explicit receiver text.
- [x] Extract member accesses: field expression read facts with receiver text such as `self`.
- [x] Extract inheritance/implements/mixins: Rust `impl Trait for Type` emits `trait-impl`
      heritage facts instead of class inheritance.
- [x] Extract type references where available: method parameters, struct fields, returns, and local
      declarations/inferred call returns.
- [x] Add exact fixture ScopeIR parity:
      `internal/providers/rust/testdata/rust_scopeir_signature.golden.json`.
- [x] Add graph parity for edge counts by type:
      `ACCESSES=1`, `CALLS=1`, `DEFINES=12`, `HAS_METHOD=4`, `HAS_PROPERTY=3`,
      `IMPLEMENTS=1`, `INHERITS=1`, `METHOD_IMPLEMENTS=1`, `USES=2`.
- [x] Add unresolved reference metrics: Rust fixture records unresolved external crate/member
      references while local method, access, trait implementation, method implementation, and
      type-reference edges resolve.
- [x] Wire `.rs` files into the parser registry and analyze provider dispatch using
      `github.com/tree-sitter/tree-sitter-rust v0.24.2`.
- [x] Benchmark Rust provider parse plus extraction and record the baseline in the benchmark
      companion file.
- [x] Run full runtime E2E through the launcher-built Go executable on an isolated Rust fixture:
      CLI analyze produced a readable graph with Rust trait/struct/method/package/property nodes and
      `CALLS`, `ACCESSES`, `USES`, `IMPLEMENTS`, `METHOD_IMPLEMENTS`, community, and process edges.

PHP provider batch:

- [x] Extract definitions: namespace/package, interface, class, constructor, method, property, and
      local variable facts.
- [x] Extract imports/includes/usings/packages: PHP `use` clauses and `require/include`
      expressions are represented as ScopeIR import/include facts.
- [x] Extract calls: free function calls and member calls with normalized receiver text such as
      `this.repo`.
- [x] Extract member accesses: `->` member access read/write facts with receiver text such as
      `this` and `user`.
- [x] Extract inheritance/implements/mixins: PHP `extends` and `implements` clauses emit heritage
      facts.
- [x] Extract type references where available: method parameters, property declarations, return
      types, and inferred local assignment/call-return types.
- [x] Add exact fixture ScopeIR parity:
      `internal/providers/php/testdata/php_scopeir_signature.golden.json`.
- [x] Add graph parity for edge counts by type:
      `ACCESSES=3`, `CALLS=2`, `DEFINES=15`, `EXTENDS=1`, `HAS_METHOD=5`,
      `HAS_PROPERTY=3`, `IMPLEMENTS=1`, `INHERITS=2`, `METHOD_IMPLEMENTS=1`, `USES=5`.
- [x] Add unresolved reference metrics: PHP fixture records unresolved external include/free-call
      references while local method, access, inheritance, implementation, and type-reference edges
      resolve.
- [x] Wire `.php`/`.phtml`/`.php3`/`.php4`/`.php5`/`.php8` files into the parser registry and
      analyze provider dispatch using `github.com/tree-sitter/tree-sitter-php v0.24.2`.
- [x] Benchmark PHP provider parse plus extraction and record the baseline in the benchmark
      companion file.
- [x] Run full runtime E2E through the launcher-built Go executable on an isolated PHP fixture:
      CLI analyze produced a readable graph with PHP class/interface/method/package/property nodes
      and `CALLS`, `ACCESSES`, `USES`, `INHERITS`, `IMPLEMENTS`, `METHOD_IMPLEMENTS`, community,
      and process edges.

Dart provider batch:

- [x] Extract definitions: abstract interface-shaped class, class, constructor, method, property,
      and local variable facts.
- [x] Extract imports/includes/usings/packages: Dart `import` URIs and aliases are represented as
      ScopeIR import facts.
- [x] Extract calls: local/same-class calls and member calls with receiver text such as `repo` and
      `user.id`.
- [x] Extract member accesses: selector-chain read facts with receiver text such as `user`; bare
      class-property reads resolve against the current owner scope.
- [x] Extract inheritance/implements/mixins: Dart `extends` and `implements` clauses emit heritage
      facts.
- [x] Extract type references where available: method parameters, property declarations, return
      types, and inferred local assignment/call-return types.
- [x] Add exact fixture ScopeIR parity:
      `internal/providers/dart/testdata/dart_scopeir_signature.golden.json`.
- [x] Add graph parity for edge counts by type:
      `ACCESSES=2`, `CALLS=2`, `DEFINES=16`, `EXTENDS=1`, `HAS_METHOD=7`,
      `HAS_PROPERTY=3`, `IMPLEMENTS=1`, `INHERITS=2`, `METHOD_IMPLEMENTS=1`, `USES=4`.
- [x] Add unresolved reference metrics: Dart fixture records unresolved external import/builtin
      member references while local method, access, inheritance, implementation, and type-reference
      edges resolve.
- [x] Wire `.dart` files into the parser registry and analyze provider dispatch using
      `github.com/UserNobody14/tree-sitter-dart v0.0.0-20260508020638-507c5546dc73`
      (HEAD `507c5546dc73667c03d36803ee9bd4df0bbe4b0b`, no upstream tags).
- [x] Benchmark Dart provider parse plus extraction and record the baseline in the benchmark
      companion file.
- [x] Run full runtime E2E through the launcher-built Go executable on an isolated Dart fixture:
      CLI analyze produced a readable graph with Dart class/interface/constructor/method/property
      nodes and `CALLS`, `ACCESSES`, `USES`, `INHERITS`, `IMPLEMENTS`,
      `METHOD_IMPLEMENTS`, community, and process edges.

Vue provider batch:

- [x] Extract definitions: Vue single-file components route the first inline `<script>` or
      `<script lang="ts">` block through the existing JS/TS provider, preserving class, interface,
      constructor, method, property, and local variable facts.
- [x] Extract imports/includes/usings/packages: inline Vue script imports are represented as ScopeIR
      import facts through the JS/TS provider path.
- [x] Extract calls: inline Vue script free calls and member calls keep normalized receiver text such
      as `this.repo`.
- [x] Extract member accesses: inline Vue script read/write access facts keep receiver text such as
      `this` and `user`.
- [x] Extract inheritance/implements/mixins: inline Vue TypeScript `implements` clauses emit heritage
      facts through the JS/TS provider path.
- [x] Extract type references where available: inline Vue TypeScript parameters, properties, return
      types, and inferred local assignment/call-return types are preserved.
- [x] Add exact fixture ScopeIR parity:
      `internal/providers/vue/testdata/vue_scopeir_signature.golden.json`.
- [x] Add graph parity for edge counts by type:
      `ACCESSES=3`, `CALLS=1`, `DEFINES=10`, `HAS_METHOD=3`, `HAS_PROPERTY=3`,
      `IMPLEMENTS=1`, `INHERITS=1`, `USES=6`.
- [x] Add unresolved reference metrics: Vue fixture records unresolved external import/member
      references while local method, access, inheritance, and type-reference edges resolve.
- [x] Wire `.vue` files into analyze provider dispatch as a single-file-component container:
      script content is parsed by the current JS/TS provider; template/framework-specific facts remain
      under the separate Phase 14 framework-specific facts checklist item.
- [x] Benchmark Vue provider script parse plus extraction and record the baseline in the benchmark
      companion file.
- [x] Run full runtime E2E through the launcher-built Go executable on an isolated Vue fixture:
      CLI analyze produced a readable graph with Vue-script class/interface/constructor/method/property
      nodes and `CALLS`, `ACCESSES`, `USES`, `INHERITS`, `IMPLEMENTS`, community, and process edges.

Swift provider batch:

- [x] Extract definitions: protocol, class, initializer, method, property, and local variable facts.
- [x] Extract imports/includes/usings/packages: Swift `import` declarations are represented as
      ScopeIR import facts.
- [x] Extract calls: navigation call expressions emit member call facts with receiver text such as
      `user` and `repo`.
- [x] Extract member accesses: assignment targets such as `self.repo` and `self.id` emit write
      access facts.
- [x] Extract inheritance/implements/mixins: Swift inheritance clauses emit `implements` when the
      target is a local protocol/interface, otherwise `extends`.
- [x] Extract type references where available: parameters, properties, return types, local
      assignment-inferred types, and `self` bindings are preserved.
- [x] Add exact fixture ScopeIR parity:
      `internal/providers/swift/testdata/swift_scopeir_signature.golden.json`.
- [x] Add graph parity for edge counts by type:
      `ACCESSES=2`, `CALLS=2`, `DEFINES=12`, `HAS_METHOD=5`, `HAS_PROPERTY=3`,
      `IMPLEMENTS=1`, `INHERITS=1`, `METHOD_IMPLEMENTS=1`, `USES=3`.
- [x] Add unresolved reference metrics: Swift fixture records unresolved external import/builtin
      references while local method, access, inheritance, implementation, and type-reference edges
      resolve.
- [x] Wire `.swift` files into the parser registry and analyze provider dispatch using
      `github.com/flamingoosesoftwareinc/tree-sitter-swift v0.0.0-20260212012612-56ffc4e2dcc9`.
      Note: the newer `github.com/alex-pinkus/tree-sitter-swift`
      `v0.0.0-20260510231341-3d38a39612ba` was checked first but its Go binding is not buildable
      because the module lacks `src/parser.c`.
- [x] Benchmark Swift provider parse plus extraction and record the baseline in the benchmark
      companion file.
- [x] Run full runtime E2E through the launcher-built Go executable on an isolated Swift fixture:
      CLI analyze produced a readable graph with Swift class/protocol/initializer/method/property
      nodes and `CALLS`, `ACCESSES`, `USES`, `INHERITS`, `IMPLEMENTS`, `METHOD_IMPLEMENTS`,
      community, and process edges.

Ruby provider batch:

- [x] Extract definitions: module, class, method, property, and local variable facts.
- [x] Extract imports/includes/usings/packages: Ruby `require` calls are represented as ScopeIR
      named import facts.
- [x] Extract calls: free/member-shaped Ruby calls keep name, receiver text where available, and
      arity.
- [x] Extract member accesses: instance variable assignments such as `@repo = repo` emit write
      access facts with `self` receiver.
- [x] Extract inheritance/implements/mixins: class superclasses emit `extends`; `include`,
      `extend`, and `prepend` calls emit mixin heritage facts.
- [x] Extract type references where available: no Ruby static type-reference surface is emitted in
      this batch; dynamic receiver/member calls remain call/access facts unless the local graph can
      resolve them.
- [x] Add exact fixture ScopeIR parity:
      `internal/providers/ruby/testdata/ruby_scopeir_signature.golden.json`.
- [x] Add graph parity for edge counts by type:
      `ACCESSES=2`, `CALLS=1`, `DEFINES=12`, `EXTENDS=1`, `HAS_METHOD=5`,
      `HAS_PROPERTY=2`, `INHERITS=1`.
- [x] Add unresolved reference metrics: Ruby fixture records unresolved external/dynamic
      references while local helper call, self property writes, and inheritance resolve.
- [x] Wire `.rb` files into the parser registry and analyze provider dispatch using
      `github.com/tree-sitter/tree-sitter-ruby v0.23.1`.
- [x] Benchmark Ruby provider parse plus extraction and record the baseline in the benchmark
      companion file.
- [x] Run full runtime E2E through the launcher-built Go executable on an isolated Ruby fixture:
      CLI analyze produced a readable graph with Ruby module/class/method/property/variable nodes
      and `CALLS`, `ACCESSES`, `INHERITS`, `HAS_METHOD`, `HAS_PROPERTY`, community, and process
      edges.

COBOL/JCL provider batch:

- [x] Extract definitions: COBOL `PROGRAM-ID`, sections, and paragraphs emit module, namespace, and
      function graph nodes through the pre-parse COBOL/JCL enrichment phase.
- [x] Extract imports/includes/usings/packages: COBOL `COPY` statements emit import edges to
      scanned copybooks.
- [x] Extract calls: COBOL `PERFORM`, COBOL `CALL`, and JCL `EXEC PGM=` statements emit call edges.
- [x] Extract member accesses: not applicable for this COBOL/JCL graph surface; no fake member
      access facts are emitted.
- [x] Extract inheritance/implements/mixins: not applicable for this COBOL/JCL graph surface; no
      fake inheritance facts are emitted.
- [x] Extract type references where available: COBOL copybook linkage is represented as imports;
      no static type-reference surface is emitted in this batch.
- [x] Add exact fixture ScopeIR parity: not applicable because COBOL/JCL is deliberately handled by
      the `cobol` enrichment phase before parse, not by the tree-sitter ScopeIR parser path.
- [x] Add graph parity for edge counts by type:
      `CALLS=3`, `CONTAINS=12`, `IMPORTS=1`, `MEMBER_OF=2` on the full runtime fixture.
- [x] Add unresolved reference metrics: not applicable because this direct graph enrichment does not
      emit ScopeIR references; unresolved external program names simply do not create call edges.
- [x] Keep COBOL/JCL out of the parser registry for this batch: current tree-sitter COBOL options
      were checked, but the latest direct grammar has no Go binding and the available tagged Go
      binding uses a different tree-sitter runtime than the project parser pool.
- [x] Benchmark COBOL/JCL enrichment and record the baseline in the benchmark companion file.
- [x] Run full runtime E2E through the launcher-built Go executable on an isolated COBOL/JCL
      fixture: CLI analyze produced a readable graph with COBOL program/section/paragraph,
      copybook import, PERFORM/CALL, JCL job/step, community, and process edges. Parse reports COBOL
      files as unsupported by design because this surface is enriched before the parse phase.

Framework-specific facts batch:

- [x] Port path-based framework detection for the current language matrix: Next.js, Expo Router,
      Prisma, Supabase, Express/MVC/handlers, React, Django/FastAPI/Python API, Spring/Ktor/Android,
      ASP.NET/SignalR/Blazor, Go HTTP, Rust web/bin, C/C++ entry files, Laravel/PHP, Ruby CLI/Rake,
      iOS/SwiftUI/UIKit, Flutter/Dart, and generic API index conventions.
- [x] Port AST/decorator/annotation framework detection for the currently supported TypeScript
      baseline patterns: NestJS, Expo Router, FastAPI, Flask, Spring/JAX-RS, Ktor/Android,
      ASP.NET/SignalR/Blazor/EF Core, Laravel route attributes, Go HTTP/gin/echo/fiber/gRPC,
      Actix/Axum/Rocket/Tokio, Qt, UIKit/SwiftUI/Vapor, Rails/Sinatra, Flutter/Riverpod.
- [x] Annotate parsed ScopeIR with framework facts during the Go parse phase before cross-file
      binding/resolution, using the definition window plus nearby decorators/annotations.
- [x] Emit framework properties onto graph nodes: path facts are applied to file and symbol nodes;
      AST facts are applied to their definition nodes and expose `astFrameworkMultiplier` plus
      `astFrameworkReason` for process scoring.
- [x] Use framework multipliers in process entry-point ordering without changing required graph
      facts or language provider extraction.
- [x] Add direct tests for path detection, AST detection, ScopeIR annotation, resolution graph
      properties, and process entry-point ordering.
- [x] Benchmark framework detection and record the baseline in the benchmark companion file.
- [x] Run full runtime E2E through the launcher-built Go executable on an isolated framework
      fixture: CLI analyze produced Next.js path framework facts, NestJS decorator framework facts,
      process entry edges, and benchmark JSON from the packaged executable.

Frontend/mobile app coverage addendum batch:

- [x] Phase-jump note: after closing the Phase 15 P0 `route_map` slice, work intentionally jumped
      back to Phase 14 because the requested React/Electron/TypeScript/Next.js/Vue/Nuxt/Svelte/
      Astro/React Native/Flutter/SwiftUI/Jetpack Compose coverage is provider/framework capability
      work, not Phase 15 performance optimization. Phase 15 `context`, `impact`, and `group_sync`
      optimization items remain open below.
- [x] Classify the requested surfaces by real runtime responsibility: TypeScript is already a base
      language provider; Vue, Flutter/Dart, SwiftUI/Swift, and Android/Kotlin already had provider
      foundations; React, Electron, Next.js, Nuxt, React Native, Flutter, SwiftUI, and Jetpack
      Compose needed stronger framework facts; Svelte and Astro needed Go-owned single-file
      container providers.
- [x] Add Go-owned `.svelte` and `.astro` language detection, Web contract entries, analyze dispatch,
      and provider tests. Svelte extracts inline `<script>` blocks; Astro extracts frontmatter; both
      delegate script content through the existing TypeScript/JavaScript ScopeIR extractor and
      preserve the container language in the final IR.
- [x] Add framework/path/AST facts for the requested app surfaces: React, Electron, Next.js, Vue,
      Nuxt, Svelte/SvelteKit, Astro, React Native, Flutter, SwiftUI, and Jetpack Compose.
- [x] Regenerate the Go-owned Web UI contract schema and generated browser TypeScript glue so the
      Web UI recognizes Svelte/Astro as display languages without making TypeScript a runtime
      authority.
- [x] Harden E2E selectors that became ambiguous on the larger Go graph: the Processes tab is now
      selected by the capitalized navigation role instead of matching lower-case `processes` file/
      tool entries, and the shell interaction close assertion checks the chat composer instead of a
      graph file named `processes`.
- [x] Benchmark the addendum batch and record evidence: Svelte extractor median `896,080ns/op`;
      Astro extractor median `830,770ns/op`; isolated packaged analyze fixture scanned/parsed
      `11/11` files with `0` unsupported/failed, emitted `54` nodes and `61` relationships, and
      reported DB load `fallbackInsertFailures=0`, `skippedRelationships=0`.
- [x] Validate through the required order after the full launcher build: full launcher build
      `39,240.9ms`; focused Go tests passed; provider benchmarks passed; isolated packaged analyze
      benchmark passed; current-repo packaged analyze passed with `33,489` nodes / `66,356`
      relationships and DB load fallback/skipped `0`; `go test ./cmd/... ./internal/... -count=1`
      passed; browser E2E passed with `32` passed / `1` skipped in `494,207.2ms`; `cd avmatrix &&
      npm test` passed in `335,723.4ms`.

Exit gate:

- Provider parity is documented before marking the provider complete.

## Phase 15 - Performance Optimization

Optimization starts only after correctness gates pass. The 2026-05-14 conversion-scope correction
reopens this phase gate: completed Phase 15 benchmark/optimization slices remain valid evidence,
but Phase 15 is not the active work while non-Web TypeScript/JavaScript conversion and independent
Go tool readiness remain open.

- [ ] [P15-DEFER-UNTIL-CONVERSION-CORRECTNESS-2026-05-14] Keep Phase 15 deferred until
      `[P17-NON-WEB-TSJS-CONVERSION-BLOCKER]`, `[P17-NON-WEB-TSJS-CLASSIFICATION-MATRIX]`, and
      `[P17-INDEPENDENT-GO-MCP-READINESS-PROOF]` are closed. Phase-jump reason: earlier Phase 15
      work assumed the runtime cutover was effectively complete. The corrected plan says the next
      work is conversion completeness and tool correctness, so Phase 15 may only record
      measurement/light optimization or fix a benchmark-proven correctness/usability blocker.

- [x] Benchmark the local TypeScript baseline and Go on the same machine.
- [x] Benchmark scan time.
- [x] Benchmark parse time.
- [x] Benchmark resolution time.
- [x] Benchmark DB load time.
- [x] Benchmark graph stream time.
      Completed in the Phase 15 large-repo graph stream/profile slice. Final current-repo endpoint
      benchmark after stream batching: Go `/api/graph` JSON `1,384.3ms`, Go NDJSON stream
      `1,384.8ms`, TypeScript JSON `5,515.6ms`, and TypeScript NDJSON stream `5,852.4ms` on the
      same machine.
- [x] Benchmark memory peak.
      Final packaged Go analyze/profile run recorded `maxObservedSys=941,588,728` bytes
      (`~898MiB`), `endAllocBytes=343,455,784`, `33,702` nodes, `66,931` relationships, and DB load
      fallback/skipped `0`.
- [x] Add pprof CPU profile for large repos.
      `avmatrix analyze` now supports opt-in `--cpuprofile`; the final current-repo profile artifact
      `.tmp\phase15-large-profile-final-cpu.pprof` was `67,778` bytes. CPU pprof top showed
      `runtime.cgocall` at `42.12s` flat (`67.83%`) and
      `resolution.(*workspace).resolveImportedMember` at `4.45s` flat (`7.17%`).
- [x] Add pprof memory profile for large repos.
      `avmatrix analyze` now supports opt-in `--memprofile`; the final current-repo heap profile
      artifact `.tmp\phase15-large-profile-final-mem.pprof` was `57,861` bytes. Heap alloc-space top
      was `bytes.genSplit` (`1,022.06MB`) plus tree-sitter node/text extraction and ScopeIR
      normalization/range-key allocation.
- [x] Optimize file IO batching.
      Rejected as a current Phase 15 target by `[P15-FILE-IO-BATCHING-REJECTED]`: after the native
      DB-load transaction slice, heap pprof showed file reads at `24.81-31.48MB` alloc-space and no
      CPU hotspot. Parse/resolution/native DB work dominated the real benchmark, so no file IO
      batching change should be made without a new pprof showing file reads as material.
- [x] Optimize parser pool sizing.
      Rejected as a current Phase 15 target by `[P15-PARSER-POOL-SIZING-REJECTED]`: the accepted
      current-repo profile still creates only `4` parsers and shows no pool wait/size contention.
      The remaining parse cost is tree-sitter/provider work (`Pool.Parse`, `tsjs.Extract`, and
      cgo), not parser pool capacity.
- [x] Optimize resolution indexes.
      Completed for the profile-backed resolution workspace/name-lookup allocation target in
      `[P15-RESOLUTION-WORKSPACE-ALLOC-OPT]`, then extended by
      `[P15-IMPORTED-MEMBER-INDEX-OPT]` for the post-DB-load CPU hotspot in
      `resolveImportedMember`. Any later resolution hotspot such as `callerForScope`,
      `emitDefinitionNodes`, or graph relationship allocation still needs its own
      benchmark/evidence slice.
- [x] Optimize DB load batching.
      Completed by `[P15-DBLOAD-CGO-BATCH-OPT]` with native LadybugDB load transactions around
      the existing COPY sequence. This closes the DB-load batching item only; generic file IO
      batching and parser pool sizing were later closed as rejected by current-profile evidence.
- [x] Optimize graph streaming.
      Completed in the same large-repo graph stream/profile slice. Impact note:
      `newAnalyzeCommand` was CRITICAL because it is CLI root/analyze wiring, so pprof flags were
      kept opt-in and default analyze behavior was unchanged; `streamGraphNDJSON` impact was LOW.
      Implementation: NDJSON graph streaming now flushes every `512` records plus a final flush,
      instead of flushing every node/relationship. Benchmark: Go NDJSON stream improved from
      `2,915.0ms` to `1,384.8ms` on the current graph while preserving NDJSON shape and content
      type. Validation followed the plan rule: full launcher build first (`34,126.1ms`), focused CLI
      and HTTP tests, graph stream benchmark, final analyze with CPU/memory pprof and graph parity
      (`33,702` nodes / `66,931` relationships, fallback/skipped `0`), full Go tests
      (`27,043.8ms`), browser E2E (`32` passed / `1` skipped, `408,728.9ms` after isolated analyze
      `64,342.3ms`), and `cd avmatrix && npm test` (`419,432.8ms`).
- [x] [P15-DBLOAD-CGO-BATCH-OPT] Profile-backed DB load optimization target: reduce native
      LadybugDB/cgo overhead shown by CPU pprof (`runtime.cgocall` `42.12s` flat, `67.83%`) while
      preserving fail-closed load semantics, `fallbackInsertFailures=0`, `skippedRelationships=0`,
      and node/relationship row parity.
      Phase-stay note: this stayed in Phase 15 because it is performance work on the already
      cut-over Go runtime, after the rejected `PARALLEL=true` candidate and after the separate
      schema-lookup allocation slice. Impact: AVmatrix reported `loadGraph` CRITICAL because it
      feeds `analyze`, `newAnalyzeCommand`, `main`, and HTTP embed/analyze paths; `LoadCSVExport`
      context showed the direct analyze/test/native callers. Implementation: `lbugload` now opens a
      load transaction only when the runner implements the optional transaction interface; the
      native write runner wraps the existing node/relationship COPY sequence with
      `BEGIN TRANSACTION` / `COMMIT` and rolls back on any fail-closed COPY/schema/fallback error.
      Non-native/noop runners keep the old per-query behavior. Benchmark: packaged current-repo
      analyze moved from `.tmp\phase15-continue-start-graph.json` wall `62,660.7ms`, benchmark
      total `62,327.7ms`, DB load `32,246.0ms` to
      `.tmp\phase15-dbload-tx-attempt-analyze.json` wall `35,469.1ms`, benchmark total
      `34,167.8ms`, DB load `5,773.7ms`, `nodeCopyCount=19`, `relationshipCopyCount=91`, graph/DB
      rows `33,799` / `67,272`, and fallback/skipped `0`. CPU pprof moved `runtime.cgocall` from
      the previous `42.12s` flat to `19.64s` flat / `21.83s` cumulative in
      `.tmp\phase15-dbload-tx-attempt-cpu.pprof`; cgo remains a runtime cost because parser/native
      work still crosses C. Validation followed the plan rule: full launcher build first
      (`52,836.0ms`), focused `lbugload` tests (`2,154.4ms`), native LadybugDB integration with
      CGO runtime path (`13,648.5ms`), after-build loader benchmarks (`23,453.1ms`), packaged
      analyze plus CPU pprof (`35,469.1ms`), full Go tests (`30,764.5ms`), browser E2E through the
      packaged Go backend and isolated `AVMATRIX_HOME` (`32` passed / `1` skipped; isolated analyze
      `37,658.0ms`, Playwright `447,486.9ms`), and backend smoke confirmed `/api/info` plus
      `/api/repos` served the isolated indexed repo. `cd avmatrix && npm test` had one initial
      flaky Rust skills E2E failure, then the failed suite rerun passed (`25/25`, `110,639.9ms`)
      and the full command rerun passed (`393,820.1ms`).
- [x] [P15-FILE-IO-BATCHING-REJECTED] Reassess the generic file IO batching target after the
      DB-load transaction slice changed the macro bottleneck. Phase-stay note: this stayed in Phase
      15 because it is performance triage on the already cut-over Go runtime. Benchmark/evidence:
      `.tmp\phase15-next-after-dbtx-start-graph.json` recorded total `35,015.3ms`, parse
      `19,150.8ms`, resolution `5,326.8ms`, DB load `5,803.5ms`, and DB fallback/skipped `0`.
      Heap pprof showed `os.readFileContents` at only `31.48MB` alloc-space before the imported
      member index and `24.81MB` after it, while CPU pprof did not show file read work as a top
      frame. Conclusion: do not spend Phase 15 time on file IO batching until a future profile makes
      file reads a real bottleneck.
- [x] [P15-PARSER-POOL-SIZING-REJECTED] Reassess the generic parser pool sizing target after the
      DB-load transaction slice. Phase-stay note: this stayed in Phase 15 because it is performance
      triage, not provider coverage or cutover authority. Benchmark/evidence: the current-repo
      parser metrics showed `createdParsers=4`, `total=1058`, `failed=0`, and no evidence of pool
      contention. CPU pprof attributed parse time to tree-sitter/provider work (`Pool.Parse`
      `8.34s` cumulative, `tsjs.Extract` `6.62s` cumulative) rather than parser pool capacity.
      Conclusion: parser pool sizing is closed as rejected for the current bottleneck; future parse
      optimization must target tree-sitter/provider/cgo cost with its own benchmark/evidence.
- [x] [P15-IMPORTED-MEMBER-INDEX-OPT] Profile-backed resolution CPU target: reduce
      `workspace.resolveImportedMember` after the post-DB-load CPU profile showed it at `4.91s`
      flat / `5.11s` cumulative and resolution still cost `5,326.8ms` on the current repo.
      Phase-jump note: this intentionally stayed inside Phase 15 and selected a pprof-backed
      hotspot after `[P15-FILE-IO-BATCHING-REJECTED]` and `[P15-PARSER-POOL-SIZING-REJECTED]`
      showed the generic checklist items were not the real current bottleneck. Impact: AVmatrix
      reported `resolveImportedMember` CRITICAL because it feeds `resolveCall`,
      `ResolveBoundInto`, `Run`, and analyze. Implementation: the resolution workspace now indexes
      resolved imports by `(sourceFile, localName)` and `resolveImportedMember` scans only relevant
      imports for that receiver instead of every workspace import. Benchmark:
      `BenchmarkResolveImportedMemberManyImports` improved from `17.2-17.6us/op` to
      `492-512ns/op` after build, with allocations unchanged at `48 B/op` and `3 allocs/op`.
      Packaged current-repo analyze moved from `.tmp\phase15-next-after-dbtx-start-graph.json`
      total `35,015.3ms`, resolution `5,326.8ms`, cross-file binding `1,585.8ms` to
      `.tmp\phase15-import-index-final-analyze.json` total `28,727.0ms`, resolution `955.3ms`,
      cross-file binding `579.8ms`, graph/DB rows `33,803` / `67,310`, and DB fallback/skipped
      `0`; benchmark compare reported total `-18%`, resolution `-82.1%`, and cross-file binding
      `-63.4%`. Final CPU pprof no longer showed `resolveImportedMember` in the top table.
      Validation followed the plan rule: full launcher build first (`41,236.0ms`), focused
      resolution tests (`2,312.7ms`), after-build micro benchmark (`11,508.5ms`), packaged analyze
      plus CPU/memory pprof (`30,412.5ms` wall), full Go tests (`31,327.0ms`), and browser E2E
      through packaged Go backend with isolated `AVMATRIX_HOME` (`32` passed / `1` skipped;
      isolated analyze `29,045.1ms`, Playwright `428,776.4ms` using `--workers=1 --retries=1`,
      with no retry needed in the accepted output), and `cd avmatrix && npm test`
      (`378,034.1ms`).
- [x] [P15-PARSER-NODECOUNT-OPT] Profile-backed parser diagnostic target: remove unconditional
      parser node counting from the default analyze path after
      `.tmp\phase15-next-after-import-index-start-cpu.pprof` showed
      `internal/parser.countNodes` at `1.58s` cumulative under `Pool.Parse`.
      Phase-stay note: this intentionally stayed in Phase 15 because it is parser performance work
      on the already cut-over Go runtime, not Phase 14 provider coverage or Phase 17 cutover
      authority. Impact: AVmatrix reported `countNodes` LOW risk with direct caller `Pool.Parse`;
      `PoolOptions` was CRITICAL because it is shared by analyze, providers, resolution tests, and
      HTTP paths. Implementation: `PoolOptions.CountNodes` now makes node-count metrics opt-in;
      the default analyze/runtime path leaves `Result.NodeCount=0`, while parser tests that assert
      node metrics explicitly enable the option. Benchmark: `BenchmarkPoolParseNodeCount` after
      build measured disabled `221-297us/op`, `1032-1339 B/op`, `17 allocs/op` vs enabled
      `276-387us/op`, `5352-5416 B/op`, `152 allocs/op`. Packaged current-repo analyze moved from
      `.tmp\phase15-next-after-import-index-start-graph.json` parser `totalDuration=8,646.6ms` and
      CPU `Pool.Parse=8.28s` / `countNodes=1.58s` to
      `.tmp\phase15-nodecount-final-analyze.json` parser `totalDuration=7,586.2ms` and CPU
      `Pool.Parse=7.58s`, with `countNodes` absent from the top CPU table. Overall wall was neutral
      in that single run (`benchmark-compare` total `+2.2%`) because parse/provider and DB phases
      varied and the commit adds benchmark/test code; the accepted improvement is the removed
      diagnostic node-walk cost, not a claimed macro speedup. Graph/DB rows were `33,843` /
      `67,328`, with DB fallback/skipped `0`. Validation followed the plan rule: full launcher
      build first (`45,641.2ms`), focused parser tests after build (`2,508.1ms`), after-build
      parser benchmark (`14,867ms` package time), packaged analyze plus CPU/memory pprof
      (`32,432.8ms` wall), full Go tests (`32,805.6ms`), browser E2E through packaged Go backend
      with isolated `AVMATRIX_HOME` (`33/33` passed; isolated analyze `29,087.9ms`, Playwright
      `206,434.9ms`), and `cd avmatrix && npm test` (`400,152.3ms`). An earlier isolated-analyze
      attempt is rejected from evidence because the PowerShell variable `$HOME` was accidentally
      used instead of an AVmatrix-specific variable.
- [x] [P15-TSJS-TRAVERSAL-KIND-OPT] Profile-backed TypeScript/JavaScript provider traversal target:
      reduce repeated full-tree walks and repeated `node.Kind()` cgo calls after the post-node-count
      profile showed `tsjs.Extract` at `6.35s` cumulative, `tsjs.walk` at `6.28s`, `Node.Kind` at
      `2.27s`, and `NamedChild` at `2.28s`.
      Phase-stay note: this stayed in Phase 15 because it is provider performance work on the
      already cut-over Go runtime, not new provider coverage or cutover authority. Impact: AVmatrix
      reported `Extract` CRITICAL because it feeds analyze/runtime/test flows; `emitDefinition`,
      `emitReference`, `emitTypeBinding`, `buildContext`, and `collectScopes` were LOW. The change
      stayed internal to the TS/JS collector and preserved existing wrapper methods. Implementation:
      `walkKind` now computes a node kind once per visited node, `collectScopesAndContext` combines
      the previous scope and context passes, and the main extraction pass calls kind-aware emitter
      helpers instead of letting every emitter call `node.Kind()` independently. Benchmark:
      `BenchmarkExtractTypeScriptScopeIR` improved from `446-466us/op`, about `87.3KB/op`, and
      `1966 allocs/op` to `295-337us/op`, `68.3KB/op`, and `996 allocs/op` after build. Packaged
      current-repo analyze moved from `.tmp\phase15-next-after-nodecount-start-graph.json` total
      `27,545.4ms`, parse `17,141.5ms` to `.tmp\phase15-tsjs-traversal-final-analyze.json` total
      `25,728.2ms`, parse `14,549.4ms`, graph/DB rows `33,828` / `67,364`, and DB fallback/skipped
      `0`; benchmark compare reported total `-6.6%` and parse `-15.1%`. Final CPU pprof moved
      `tsjs.Extract` from `6.35s` to `4.08s`, reduced `Node.Kind` from `2.27s` to `0.92s`, and
      reduced `NamedChild` from `2.28s` to `1.59s`. Validation followed the plan rule: full
      launcher build first (`40,658.1ms`), focused provider/resolution tests after build
      (`9,854.5ms`), after-build TS/JS benchmark (`12,129.3ms`), packaged analyze plus CPU/memory
      pprof (`27,697.2ms` wall), full Go tests (`29,375.2ms`), browser E2E through packaged Go
      backend with isolated `AVMATRIX_HOME` (`33/33` passed; isolated analyze `25,467.0ms`,
      Playwright `194,306.0ms`), and `cd avmatrix && npm test` (`343,081.8ms`).
- [x] [P15-GO-PROVIDER-TRAVERSAL-KIND-OPT] Profile-backed Go provider traversal target: reduce
      repeated Go-provider full-tree passes and repeated `node.Kind()` cgo calls after the
      post-TS/JS profile showed `golang.Extract` at `1.95s` cumulative, `golang.walk` at `1.95s`,
      `golang.Extract.func1` at `1.10s`, `Node.Kind` at `1.14s`, and Go provider heap at
      `24.40MB`.
      Phase-stay note: this stayed in Phase 15 because it is provider performance work on the
      already cut-over Go runtime, not Phase 14 language coverage or Phase 17 runtime authority.
      Native tree-sitter parse cgo and native LadybugDB query/commit remain separate targets and
      were not claimed by this slice. Impact: AVmatrix reported `Extract` LOW with one direct
      caller (`extractScopeIR`) and two affected flows; the targeted emitter/context helpers were
      LOW. AVmatrix reported `walk` HIGH because it is the Go provider traversal helper under
      `Extract`, so the implementation kept `walk` compatible, added `walkKind`, and moved only the
      main Go extraction path to the kind-aware traversal. Implementation: `collectScopesAndContext`
      combines the previous scope/context passes, kind-aware emitter helpers avoid repeated
      `node.Kind()` calls in the main pass, and `emitTypeReferences` uses `walkKind` for nested type
      walks. Benchmark: `BenchmarkExtractGoScopeIR` improved from `549-572us/op`, about
      `106.1KB/op`, and `2379 allocs/op` to `380-500us/op`, `85.2KB/op`, and `1310 allocs/op`
      after build. Packaged current-repo analyze moved from
      `.tmp\phase15-next-after-tsjs-traversal-start-graph.json` total `24,607.2ms`, parse
      `14,595.9ms` to `.tmp\phase15-go-provider-traversal-analyze.json` total `24,211.0ms`,
      parse `13,943.3ms`, graph/DB rows `33,843` / `67,378`, and DB fallback/skipped `0`;
      `benchmark-compare` reported total `-1.6%` and parse `-4.5%`. Final CPU pprof moved
      `golang.Extract` from `1.95s` to `1.14s`, `golang.Extract.func1` from `1.10s` to `0.56s`,
      and `Node.Kind` from `1.14s` to `0.66s`. Final heap pprof moved Go provider extract from
      `24.40MB` to `19.91MB`. Validation followed the plan rule: full launcher build first
      (`39,231.7ms`), focused provider/analyze/resolution tests after build (`6,337.4ms`),
      after-build Go provider benchmark (`10,258.5ms`), packaged analyze plus CPU/memory pprof
      (`26,449.5ms` wall), full Go tests (`34,494.9ms`), browser E2E through packaged Go backend
      with isolated `AVMATRIX_HOME` (`32` passed / `1` skipped; isolated analyze `34,598.5ms`,
      Playwright `436,101.7ms`), and `cd avmatrix && npm test` (`404,491.2ms` exit code `0`).
- [x] [P15-GRAPH-SNAPSHOT-STREAM-OPT] Profile-backed graph snapshot memory target: remove the
      full-graph `json.MarshalIndent` buffer from `writeGraphSnapshot` after the post-Go-provider
      heap profile showed `bytes.growSlice` at `64MB` under graph snapshot JSON marshaling and
      `maxObservedSys=822,550,776`.
      Phase-stay note: this stayed in Phase 15 because it is pprof-backed memory optimization on
      the already cut-over Go runtime, not provider coverage or Phase 17 cutover authority. Native
      tree-sitter parse cgo and native LadybugDB query/commit/COPY remain separate targets and were
      not claimed by this slice. Impact: AVmatrix reported `writeGraphSnapshot` CRITICAL because it
      is called from analyze `Run` and participates in `newAnalyzeCommand` / `main` execution flows.
      The implementation therefore preserved the public `graph.json` object shape (`nodes`,
      `relationships`) and the existing temp-file-plus-rename behavior while streaming top-level
      arrays through a buffered writer. Benchmark: baseline
      `.tmp\phase15-next-after-go-provider-start-graph.json` recorded total `25,256.0ms`, DB load
      `5,632.8ms`, rows `33,844` / `67,379`, fallback/skipped `0`, and
      `maxObservedSys=822,550,776`; final
      `.tmp\phase15-graph-snapshot-stream-final-analyze.json` recorded total `27,335.2ms`, DB load
      `6,858.4ms`, rows `33,857` / `67,412`, fallback/skipped `0`, and
      `maxObservedSys=632,242,424`. Heap pprof removed the `64MB`
      `bytes.growSlice` / full-snapshot `json.MarshalIndent` frame from the top table and moved
      inuse total from `178.39MB` to `135.09MB`. CPU pprof stayed roughly neutral
      (`writeGraphSnapshot` `0.86s` before, `0.82s` after). `benchmark-compare` showed total
      `+8.2%` because native DB load moved `+21.8%`; this slice is accepted as a memory/pprof win,
      not a macro wall-time speedup. Validation followed the plan rule: full launcher build first
      (`40,160.2ms` final build), focused analyze/CLI/HTTP/MCP tests (`21,748.4ms`), after-build
      `BenchmarkWriteGraphSnapshot` (`28-45ms/op`, about `3.54MB/op`, `32,527-32,531 allocs/op`),
      packaged analyze plus CPU/memory pprof (`29,806.5ms` wall), full Go tests (`31,950.0ms`),
      browser E2E through packaged Go backend with isolated `AVMATRIX_HOME` (`31` passed /
      `1` skipped / `1` flaky recovered on retry; isolated analyze `26,815.2ms`, Playwright
      `528,159.0ms`), and `cd avmatrix && npm test` (`414,123.7ms` exit code `0`).
- [x] [P15-SCOPEIR-OWNED-NORMALIZE-ALLOC-OPT] Profile-backed ScopeIR normalization allocation
      target: reduce provider extract allocation after the post-lifecycle Phase 15 heap profile
      showed `ScopeIR.Normalized` at `54.01MB` flat and TS/JS + Go provider extracts as the current
      retained IR source.
      Phase-stay note: this resumed Phase 15 after the Phase 10 schema correctness and Phase 16
      launcher lifecycle jumps. It is provider/runtime performance work, not provider coverage and
      not a cutover authority gate. Rejected path: a pure in-place normalization variant reduced
      micro allocations but retained oversized append backing arrays in full-repo analyze
      (`endAllocBytes=204,682,944`, `maxObservedSys=839,073,016`), so it was not accepted.
      Accepted implementation: `scopeir.NormalizeOwned` compacts top-level IR slices before sorting
      owned provider facts; TS/JS and Go providers use it while the existing `Normalized()` API
      remains non-mutating. Benchmark: after the full launcher build, `BenchmarkExtractTypeScriptScopeIR`
      moved from about `68.3KB/op`, `996 allocs/op` to `66.4KB/op`, `980 allocs/op`; `BenchmarkExtractGoScopeIR`
      moved from about `85.2KB/op`, `1310 allocs/op` to `82.8KB/op`, `1281 allocs/op`.
      Packaged current-repo analyze moved from `.tmp\phase15-select-next-target.json` total
      `30,058.4ms`, parse `14,450.4ms`, DB load `8,278.5ms`, rows `33,909` / `67,556`,
      fallback/skipped `0`, and `maxObservedSys=745,119,992` to
      `.tmp\phase15-scopeir-owned-normalize-final.json` total `24,982.3ms`, parse `14,202.9ms`,
      DB load `6,046.2ms`, rows `33,918` / `67,609`, fallback/skipped `0`, and
      `maxObservedSys=743,596,280`. `benchmark-compare` reported total `-16.9%`, but the accepted
      claim is the small provider allocation reduction plus neutral full-repo memory, because DB
      load variance dominated the wall-time delta. Validation followed the required build-first
      order: full launcher build `38,220.6ms`, focused scopeir/TSJS/Go provider tests passed,
      focused benchmarks passed, packaged analyze with CPU/heap pprof passed in `26,867.1ms` wall,
      full Go tests passed in `35,542.3ms`, browser E2E through packaged Go backend and isolated
      `AVMATRIX_HOME` passed (`33` listed tests, `.last-run.json` status `passed`; command wall
      `590.4s`; isolated analyze `28,503.9ms`, rows `33,919` / `67,610`, fallback/skipped `0`),
      `cd avmatrix && npm test` passed in `461,316.4ms`, and AVmatrix `detect_changes(scope=all)`
      reported MEDIUM risk limited to the normalize/provider/doc slice.
- [x] [P15-SCOPEIR-RELEASE-AFTER-RESOLUTION-OPT] Profile-backed retained-heap target:
      release parsed ScopeIRs after resolution for CLI/Web analyze runs. Phase-stay note: this
      stayed in Phase 15 because it is memory/performance work on the already cut-over Go runtime;
      it is not provider coverage, Phase 10 persistence correctness, or Phase 17 cutover authority.
      Impact: AVmatrix reported `analyze.Run` CRITICAL and `newAnalyzeCommand` HIGH because they
      are core analyze/CLI orchestration paths; the implementation is therefore limited to an
      opt-in `ReleaseScopeIRsAfterResolution` option that CLI and HTTP analyze enable after
      resolution has consumed the IRs. Direct `analyze.Run` callers keep the default behavior.
      Benchmark: the post-owned-normalize baseline `.tmp\phase15-after-owned-normalize-start.json`
      recorded total `29,665.3ms`, end alloc `170,499,696`, and heap inuse `131.94MB` with
      `ScopeIR.NormalizeOwned=55.67MB` flat. Final
      `.tmp\phase15-release-scopeir-after-resolution.json` recorded total `24,476.3ms`, end alloc
      `80,059,072`, graph rows `33,928` / `67,617`, fallback/skipped `0`, and heap inuse
      `58.67MB`; `ScopeIR.NormalizeOwned` no longer appears in the heap top table. Do not claim a
      peak-RSS win: `maxObservedSys` stayed neutral/slightly higher (`709,378,296` ->
      `713,580,792`), and DB-load timing variance dominates the wall-time delta. Validation
      followed the plan rule: full launcher build first (`44.3s`), packaged analyze with CPU/heap
      pprof, full Go tests (`31.7s`), browser E2E through packaged Go backend and isolated
      `AVMATRIX_HOME` after hardening the async process-highlight waits (`32` passed / `1` skipped;
      Playwright `441,444.3ms`; isolated analyze total `29,141.3ms`, rows `33,928` / `67,617`,
      fallback/skipped `0`), Prettier check for the touched E2E specs, and `cd avmatrix &&
      npm test` (`571.4s`).
- [x] [P15-GRAPH-COMPACT-AFTER-PROCESSES-OPT] Profile-backed graph retained-heap target:
      trim graph slice capacity and drop lazy graph indexes after Phase Processes, before DB load
      and graph snapshot. Phase-stay note: this stayed in Phase 15 because it is memory/performance
      work on the already cut-over Go runtime; it is not a provider coverage or cutover authority
      gate. Impact: AVmatrix reported `Graph` CRITICAL (`357` impacted symbols, `376` processes),
      `Graph.AddRelationship` CRITICAL (`130` symbols, `300` processes), and `analyze.Run`
      CRITICAL (`4` symbols, `51` processes), so the patch does not change add/lookup semantics.
      Implementation: `Graph.Compact()` copies `Nodes` and `Relationships` to `cap=len` and clears
      `nodeIndex`/`relIndex`; `GetNode`, `GetRelationship`, and later `Add*` calls rebuild indexes
      lazily. Benchmark: baseline heap pprof from
      `.tmp\phase15-release-scopeir-after-resolution-mem.pprof` showed `58.67MB` in use with
      `Graph.AddRelationship=15.77MB` flat. Final
      `.tmp\phase15-graph-compact-after-processes-mem.pprof` showed `49.64MB` in use; the old
      `Graph.AddRelationship` retained frame disappeared and `Graph.Compact` retained `11.48MB` of
      compact graph slices. Do not claim a wall-time win: final total was `28,249.7ms` versus the
      profiled baseline `24,476.3ms` because native parse cgo varied upward. Validation followed
      the plan rule: full launcher build first (`40.7s`), focused graph/analyze tests and
      `BenchmarkWriteGraphSnapshot`, packaged analyze with CPU/heap pprof, full Go tests (`31.0s`),
      browser E2E through packaged Go backend and isolated `AVMATRIX_HOME` (`32` passed /
      `1` skipped; Playwright `434,795.2ms`; isolated analyze total `25,136.7ms`, rows `33,982` /
      `67,657`, fallback/skipped `0`), and `cd avmatrix && npm test` (`477.1s`).
- [x] [P15-NATIVE-CGO-BOUNDARY-CLASSIFICATION] Profile-backed native-boundary classification:
      reject treating native tree-sitter parse cgo or native LadybugDB query/COPY/commit as a
      same-slice Go-level optimization target. Phase-stay note: this stayed in Phase 15 because it
      is performance target triage, not a correctness/cutover phase jump. Evidence from
      `.tmp\phase15-graph-compact-after-processes-cpu.pprof` shows the largest remaining CPU costs
      cross C/native APIs: `runtime.cgocall=19.25s` flat / `20.49s` cumulative,
      `Parser.ParseWithOptions=8.07s` cumulative, `lbugnative.Query=5.66s` cumulative,
      `lbugload.runCopy=1.94s` cumulative, and `CommitLoadTransaction=3.23s` cumulative. Prior
      Phase 15 slices already reduced Go-level work around parser traversal, graph snapshot, DB
      transaction/load setup, and retained heap; changing these native costs further requires
      upstream/native parser or LadybugDB API/design changes, not another local Go micro-patch.
      This closes the native cgo target as classified/rejected for the current plan, but it does
      not close Phase 15 because residual Go-level `tsjs` traversal and resolution property
      allocation remain visible in the same profiles.
- [x] [P15-TSJS-FACT-KIND-DISPATCH-DEFERRED] Reclassify the residual TS/JS dispatch optimization
      attempt as deferred, not active plan work. Phase-stay note: this stayed in Phase 15 because
      it was performance triage on the already cut-over Go runtime, but the 2026-05-14 plan
      correction makes independent MCP/tool readiness the priority. Benchmark is only a measuring
      tool here, not a reason to keep optimizing indefinitely. Impact: AVmatrix reported
      `internal/providers/tsjs.Extract` CRITICAL (`8` impacted symbols, `10` affected processes`),
      so the attempted dispatcher patch was intentionally not committed. Evidence captured:
      full launcher build before tests (`37.9s`), focused TS/JS provider parity passed
      (`0.242s`), focused micro benchmark was inconclusive-to-slightly-better
      (`315,726ns/op` median before vs `313,876ns/op` after, allocations unchanged), and packaged
      analyze completed with rows `33,985` / `67,662`, fallback/skipped `0`. Decision: revert the
      working-tree optimization attempt and do not spend more Phase 15 time here until the final
      independent MCP/tool readiness proof is complete or a correctness gate shows this path
      produces wrong data.
- [x] [P15-DBLOAD-SCHEMA-LOOKUP-ALLOC-OPT] Profile-backed DB-load allocation target:
      remove repeated schema lookup allocation in `lbugload.ExportGraphCSVs` after heap pprof showed
      `lbugload.validNodeTables` at `39.57MB` flat and `ExportGraphCSVs` at `77.02MB` cumulative.
      Phase-stay note: this stayed in Phase 15 because it is DB-load/export performance work on the
      already cut-over Go runtime. It does not close `[P15-DBLOAD-CGO-BATCH-OPT]`, because native
      COPY/cgo overhead remains a separate target. Impact: AVmatrix reported `validNodeTables`,
      `nodeColumns`, and `relationPairSupported` CRITICAL because they feed `ExportGraphCSVs`,
      `loadGraph`, and analyze. Implementation: node column lists, valid node table lookup, and
      relation-pair lookup are now package-level schema tables instead of per-call allocations.
      Packaged analyze also exposed a real fail-closed schema gap for `Const->Function`; the slice
      added that relation pair and updated schema tests instead of using fallback. Benchmark:
      `BenchmarkExportGraphCSVs` allocation improved from `~335KB/op` and `1906 allocs/op` to
      `~285KB/op` and `1403 allocs/op`; `BenchmarkLoadCSVExportCopyPathNoop` moved from
      `~1.55KB/op`, `19 allocs/op` to `1376 B/op`, `17 allocs/op`. Packaged current-repo analyze
      passed with `33,783` nodes / `67,172` relationships, DB load `27,697.0ms`,
      `nodeCopyCount=19`, `relationshipCopyCount=91`, and fallback/skipped `0`. Final pprof removed
      `validNodeTables` from the top table and reduced `ExportGraphCSVs` to `31.95MB` cumulative.
      Validation followed the plan rule: full launcher build first (`35,785.3ms` final build),
      focused lbugload/lbugschema tests (`2,344.0ms`), after-build benchmarks (`26,244.7ms`),
      packaged analyze plus heap pprof (`58,132.1ms`), full Go tests (`31,352.4ms`), browser E2E
      through packaged Go backend (`32` passed / `1` skipped after E2E modal wait hardening,
      isolated analyze `57,103.4ms`, Playwright `412,198.7ms`), and `cd avmatrix && npm test`
      (`424,654.6ms`).
- [x] [P15-DBLOAD-COPY-PARALLEL-REJECTED] Evaluate LadybugDB COPY `PARALLEL=true` as the first
      DB-load/cgo reduction candidate. Phase-stay note: this remained Phase 15 because it tests a
      profile-backed performance hypothesis for the already cut-over Go runtime, not Phase 10
      fallback correctness or Phase 17 cutover authority. The candidate is rejected and must not be
      used as a hidden fallback: full launcher build passed before the candidate test (`36,663.0ms`),
      but packaged current-repo analyze failed closed in `33,143.2ms` because the LadybugDB parallel
      CSV reader does not support quoted newlines (`method.csv` line `315`). After rollback to
      `PARALLEL=false`, full launcher build passed (`33,780.9ms`) and packaged analyze refreshed the
      graph in `59,961.4ms`, with benchmark JSON total `58,086.4ms`, DB load `28,685.4ms`,
      `nodeRows=33,703`, `relationshipRows=66,932`, `nodeCopyCount=19`, `relationshipCopyCount=90`,
      and fallback/skipped counts all `0`. Do not tick `[P15-DBLOAD-CGO-BATCH-OPT]`: the real target
      remains reducing cgo/native load overhead without changing CSV content fidelity or fail-closed
      DB load semantics. Validation followed the plan rule: full launcher build first (`35,472.3ms`),
      full Go tests (`29,349.2ms`), browser E2E through the packaged Go backend (`33/33` passed,
      isolated analyze `63,905.4ms`, Playwright `180,285.1ms`), and `cd avmatrix && npm test`
      (`409,607.3ms`).
- [x] [P15-SCOPEIR-NORMALIZED-ALLOC-OPT] Profile-backed memory optimization target: reduce
      `ScopeIR.Normalized`, `callKey`, and `rangeKey` allocation shown by heap pprof while
      preserving resolution evidence, provider contract outputs, and graph edge parity.
      Completed in the Phase 15 ScopeIR comparator slice. Phase-stay note: this stayed in Phase 15
      because it is profile-backed memory optimization on the already cut-over Go runtime, not
      Phase 14 provider coverage, Phase 10 fallback correctness, or Phase 17 cutover work. Impact:
      AVmatrix reported `ScopeIR.Normalized` LOW, `callKey` LOW, and `rangeKey` MEDIUM with the
      affected execution process limited to `Normalized`. Implementation: `Normalized` now sorts
      ScopeIR facts with field comparators instead of allocating string sort keys; the old
      `callKey`/`rangeKey`/`padInt` path was removed from the hot sort path. Benchmark:
      `BenchmarkScopeIRNormalizedLargeSort` improved from `18.10-19.14ms/op`, `~12.85MB/op`, and
      `~242.6k allocs/op` to `3.38-3.86ms/op`, `~4.05MB/op`, and `~20.3k allocs/op`. Large-repo
      memprofile changed `ScopeIR.Normalized` from the previous `~932.17MB` cumulative allocation
      hotspot to `126.96MB` cumulative, with no `callKey`/`rangeKey` frame remaining. Packaged
      current-repo analyze passed with `33,695` nodes / `66,981` relationships and DB
      fallback/skipped `0`. Validation followed the plan rule: full launcher build first
      (`40,386.9ms`), full Go tests (`32,531.4ms`), browser E2E (`32` passed / `1` skipped,
      isolated analyze `57,957.6ms`, Playwright `405,663.7ms`), and `cd avmatrix && npm test`
      (`342,850.8ms`).
- [x] [P15-FRAMEWORK-DEFINITION-WINDOW-ALLOC-OPT] Profile-backed memory optimization target:
      reduce `frameworks.definitionWindow` / `bytes.genSplit` allocation now that ScopeIR sort-key
      allocation is reduced. The final ScopeIR pprof shows `bytes.genSplit` at `1,059.40MB`
      alloc-space through `frameworks.definitionWindow`; preserve framework detection behavior,
      definition-window text content, graph parity, and fallback/skipped DB load `0`.
      Completed in the Phase 15 framework definition-window slice. Phase-stay note: this stayed in
      Phase 15 because it is profile-backed memory optimization on the already cut-over Go runtime,
      not provider coverage, fallback correctness, or cutover authority work. Impact: AVmatrix
      reported `definitionWindow` LOW risk with one direct caller, while `AnnotateScopeIR` was
      CRITICAL because it is called by `parseFiles` in the analyze path. Implementation:
      `AnnotateScopeIR` now builds a line-start index once per source file and slices definition
      windows from the original source bytes; the public `definitionWindow` helper remains covered
      by line-range/cap contract tests. Benchmark:
      `BenchmarkAnnotateScopeIRDefinitionWindowManyDefinitions` improved from
      `334.7-357.0ms/op`, `~395.5MB/op`, and `10047-10049 allocs/op` to `6.72-7.06ms/op`,
      `~2.16MB/op`, and `6049 allocs/op`. Packaged current-repo analyze with heap profile passed in
      `57,942.5ms`; `.tmp\phase15-framework-window-final-analyze.json` recorded benchmark total
      `55,811.0ms`, parse `17,713.4ms`, resolution `5,324.2ms`, DB load `28,331.8ms`,
      `33,729` nodes / `67,032` relationships, `maxObservedSys=979,804,408`, and DB
      fallback/skipped `0`. Final alloc-space pprof no longer showed `bytes.genSplit`; the
      replacement `definitionWindowIndex.window` frame was `12MB`, and the next memory targets were
      resolution workspace/name-resolution allocation. Validation followed the plan rule: full
      launcher build first (`38,255.4ms`), full Go tests (`27,620.2ms`), browser E2E through the
      packaged Go backend (`32` passed / `1` skipped using `--workers=1`, isolated analyze
      `55,728.6ms`, Playwright `397,958.2ms`; a 4-worker run was rejected as a load artifact after
      graph stream smoke passed in `1,734.1ms`), and `cd avmatrix && npm test` (`537,613.2ms`).
- [x] [P15-RESOLUTION-WORKSPACE-ALLOC-OPT] Profile-backed memory optimization target:
      reduce `internal/resolution` workspace/name-resolution allocation shown by the final framework
      heap profile (`buildWorkspace` `298.10MB` cumulative, `uniqueDefs` `126.39MB` flat,
      `resolveGlobalName` `212.67MB` cumulative) while preserving resolution metrics, graph parity,
      process counts, and DB fallback/skipped `0`.
      Completed in the Phase 15 resolution workspace allocation slice. Phase-stay note: this stayed
      in Phase 15 because it is pprof-backed performance work on the already cut-over Go runtime,
      not Phase 14 provider coverage, Phase 10 fallback correctness, or Phase 17 runtime authority.
      Impact: AVmatrix reported `buildWorkspace` CRITICAL because it feeds `BuildCrossFileBinding`
      and analyze, `definitionLookupNames` HIGH, and the targeted name-resolution helpers LOW; the
      implementation was limited to pre-sizing workspace indexes and removing per-lookup temporary
      slices/maps while keeping ambiguous-name behavior unchanged. Benchmark:
      `BenchmarkResolveTypeScriptGraphFixture` improved from about `381.7us/op`,
      `259,341 B/op`, and `1733 allocs/op` to about `356.3us/op`, `247,954 B/op`, and
      `1681 allocs/op`. Packaged current-repo analyze after full build passed with
      `33,760` nodes / `67,153` relationships, resolution phase `5,014.8ms`, DB load
      `32,675.5ms`, `fallbackInsertFailures=0`, and `skippedRelationships=0`. Final alloc-space
      pprof no longer showed `uniqueDefs` or lookup closure allocation in the top table; the next
      profile-backed memory candidates are `ScopeIR.Normalized` residual allocation, `callerForScope`
      (`70.52MB` flat), `Graph.AddRelationship`, `emitDefinitionNodes`, and the still-open
      `[P15-DBLOAD-CGO-BATCH-OPT]`. Validation followed the plan rule: full launcher build first
      (`34,953.3ms`), focused resolution tests (`2,343.2ms`), after-build micro benchmark
      (`8,152.8ms`), packaged analyze plus heap pprof (`62,789.7ms`), full Go tests
      (`32,739.5ms`), browser E2E through packaged Go backend (`32` passed / `1` skipped,
      isolated analyze `56,796.4ms`, Playwright `411,382.6ms`), and `cd avmatrix && npm test`
      (`579,022.0ms`).
- [x] [P15-MCP-OPTIMIZATION-REGRESSIONS] Treat the Phase 13 MCP benchmark gaps as optimization
      candidates with an explicit plan, not as optional cleanup. Before Phase 15 can be marked
      complete, `route_map`, `context`, `impact`, and HTTP `group_sync` must have updated
      measurements plus either an implemented optimization or a queued optimization task with
      owner/target.
      When this item is reached from `[P13-MCP-BENCHMARK-CLASSIFY]`, record the Phase 15 target here,
      then return to `[P13-MCP-BENCHMARK-CLASSIFY]` so Phase 13 can return to the Phase 14 provider
      checklist.
      Priority order is fixed unless new benchmark evidence changes it: `P0 route_map`, `P1
      context`, `P1 impact`, `P2 HTTP group_sync overhead`, `P3 preserve startup/query wins,
      smaller tools/list payload, and 0 protocol-noise bytes`.
      Completed through the Phase 15 MCP optimization series. `route_map`, `context`, `impact`, and
      P3 `query` hot-path regressions received implemented optimizations with current benchmark
      evidence. HTTP `group_sync` received a cold/warm/core timing split and a queued owner/target
      task for `[P15-GROUPSYNC-REGISTRY-WRITE-OPT]` because the measured bottleneck is registry file
      persistence on a CRITICAL shared path. Phase-jump note: this stayed in Phase 15 because all
      work is performance/profile triage on the already cut-over Go runtime, not Phase 14 provider
      coverage or Phase 17 runtime authority.
- [x] Optimize MCP `route_map` hot path. Current Phase 13 benchmark is a mixed result and
      `route_map` is the worst regression; target `<50ms` on the benchmark fixture by using a
      precomputed route index/cache instead of rebuilding route maps from the whole graph per call.
      Completed in the Phase 15 route-cache slice after Phase 17 cutover gates closed. Phase-jump
      note: this work intentionally entered Phase 15 because it is performance optimization, not a
      cutover correctness gate. Implementation: MCP `Server` now owns a stat-invalidated
      `graph.json` cache plus reusable route index used by `route_map`, `shape_check`, and
      `api_impact`; graph snapshot decoding behavior and JSON-RPC payload shapes are unchanged.
      Benchmark: current before-run `route_map` was `759.56ms` stdio / `730.87ms` HTTP; after the
      cache/index change it is `7.70ms` stdio / `8.80ms` HTTP, below the `<50ms` target. Validation
      followed the plan rule: full launcher build first (`35,387ms`), MCP benchmark, Go tests
      (`29,410.2ms`), packaged analyze graph-parity smoke (`33,399` nodes / `66,035` relationships,
      DB load `fallbackInsertFailures=0`, `skippedRelationships=0`), browser E2E `33/33`
      (`227,307.3ms`), and `cd avmatrix && npm test` (`373,303.6ms`). Updated measurements also show
      warm-session `context` at `24.01ms` stdio / `32.17ms` HTTP, `impact` at `24.96ms` stdio /
      `37.18ms` HTTP, and `group_sync fixture` at `1.32ms` stdio / `2.37ms` HTTP; the detailed
      context/impact timing-split items remain open below.
- [x] Profile and optimize MCP `context`; target `<100ms` on the benchmark fixture, with timing
      split across repo resolve, target lookup, graph neighborhood read, snippet/file reads,
      formatting, and JSON-RPC serialization.
      Completed in the Phase 15 context one-pass neighborhood slice. Phase-jump note: this work was
      started after the Phase 14 addendum, paused to jump back to Phase 10 for the LadybugDB
      fallback correctness fix, then resumed as Phase 15 performance work. Implementation:
      `context` now gathers incoming references, outgoing references, process participation, and
      class-like constructor/file incoming references from one graph-neighborhood pass, with an
      internal profiling path for timing splits. Benchmark: current before-run `context` was
      `23.20ms` stdio / `25.87ms` HTTP; final packaged Go is `15.36ms` stdio / `18.67ms` HTTP,
      below the `<100ms` target and faster than TypeScript's `101.02ms` stdio / `107.27ms` HTTP in
      the same final run. Synthetic warm-neighborhood profiling on `2,500` incoming refs,
      `2,500` outgoing refs, and `750` process memberships measured `26.96-27.67ms/op`, with the
      single neighborhood read accounting for roughly `24.58-26.57ms/op`. Validation followed the
      plan rule: full launcher build first (`34,207.9ms`), MCP benchmark, focused MCP tests and
      microbenchmarks, packaged analyze graph-parity smoke (`33,574` nodes / `66,604`
      relationships, DB load fallback/skipped `0`), full Go tests (`27,772.9ms`), browser E2E
      (`32` passed / `1` skipped, `512,685.5ms`), and `cd avmatrix && npm test`
      (`438,204.3ms`).
- [x] Profile and optimize MCP `impact`; target `<150ms` on the benchmark fixture, with hot indexes
      for symbol callers/callees, file symbols/importers, related tests, routes, and tool handlers.
      Completed in the Phase 15 impact timing-profile slice. Phase-jump note: this stayed in Phase
      15 after the `context` slice because it is MCP performance/profile work, not Phase 14 provider
      coverage or Phase 17 cutover. The attempted hot-index runtime was rejected because it
      regressed the accepted MCP `impact` row to `64.90ms` stdio / `71.71ms` HTTP, so the committed
      change keeps payload/runtime behavior unchanged and adds a profiled internal path for timing
      splits. Final packaged Go `impact` is `26.53ms` stdio / `26.47ms` HTTP, below the `<150ms`
      target and faster than the same TypeScript row (`140.79ms` stdio / `135.10ms` HTTP).
      Synthetic warm-traversal profiling measured `19.35-20.73ms/op`; the main cost is affected
      summary construction (`13.77-16.25ms/op`) plus traversal (`2.83-4.38ms/op`), so any further
      optimization is a queued summaries-index task rather than a cutover blocker. Current-repo
      analyze comparison on the same machine recorded TypeScript `130,582.2ms` vs packaged Go
      `58,693.8ms`, about `2.22x` faster, with Go DB load fallback/skipped counts at `0`.
      Validation followed the plan rule: full launcher build first (`35,341.2ms`), focused MCP
      tests and microbenchmarks, MCP benchmark, packaged analyze graph-parity smoke (`33,612` nodes
      / `66,727` relationships, DB load fallback/skipped `0`), full Go tests (`27,295.0ms`),
      browser E2E (`32` passed / `1` skipped, `536,712.6ms`), and `cd avmatrix && npm test`
      (`624.8s`).
- [x] Re-benchmark MCP HTTP `group_sync` in cold-session and warm-session modes to separate
      session/transport overhead from core group-sync runtime cost.
      Completed in the Phase 15 group-sync cold/warm benchmark slice. Phase-jump note: this stayed
      in Phase 15 because it is MCP performance triage, not provider coverage, persistence
      correctness, or cutover. Implementation decision: no production runtime code was changed in
      this slice because `internal/group.Sync` impact is CRITICAL across CLI and MCP group runtime;
      the accepted work adds a focused core benchmark and records a queued owner/target task for
      `internal/group.WriteRegistry`/registry persistence instead. Benchmark: Go HTTP cold
      `group_sync` avg is `15.99ms`, warm avg `11.31ms`, warm p95 `11.92ms`; TypeScript is
      `10.10ms`, `7.20ms`, and `9.88ms` on the same fixture. Go cold total remains faster
      (`20.97ms` vs TypeScript `25.16ms`) because Go initialize is faster (`4.99ms` vs `15.06ms`).
      Core Go `Sync` benchmark is `10.39-11.57ms/op`, `~24.4KB/op`, `220 allocs/op`; pprof showed
      `WriteRegistry`/Windows file write dominates CPU time. Queued target:
      `[P15-GROUPSYNC-REGISTRY-WRITE-OPT]` owns `internal/group.WriteRegistry` and should reduce
      warm HTTP `group_sync` to `<=7ms` while preserving `contracts.json` schema, `GeneratedAt`,
      CLI/MCP payloads, and exact-match semantics. Validation followed the plan rule: full launcher
      build first (`33,425.1ms`), core benchmark, HTTP cold/warm benchmark, packaged analyze
      graph-parity smoke (`33,643` nodes / `66,799` relationships, DB load fallback/skipped `0`),
      full Go tests (`25,660.5ms`), browser E2E (`32` passed / `1` skipped, `503,825.4ms`), and
      `cd avmatrix && npm test` (`374,034.3ms`).
- [x] Preserve MCP startup/query strengths, small `tools/list` payload, and `0` protocol-noise
      bytes while optimizing graph-context tools.
      Completed in the Phase 15 P3 MCP query/process-step index slice. Phase-jump note: this stayed
      in Phase 15 after the HTTP `group_sync` slice because it is MCP startup/query/tools-list/noise
      preservation, not graph-provider or cutover work. Implementation: `queryTool` now builds one
      `STEP_IN_PROCESS` index per query and reuses it for ranking plus `process_symbols`, avoiding
      repeated full relationship scans per process. Benchmark: before this slice Go `query` was
      `3,501.00ms` stdio / `3,505.47ms` HTTP; final packaged Go is `763.95ms` stdio / `763.93ms`
      HTTP in the canonical benchmark, a `~78%` reduction. A same-session probe showed the remaining
      cold cost is first graph snapshot decode/cache: `query cold=768.41ms`, then `query warm
      1=7.84ms` and `query warm 2=7.22ms`, with `0` protocol-noise bytes. Tools-list stayed small
      (`7,795` bytes Go vs `18,447` bytes TypeScript), and no startup preload was added because that
      would only move cold graph decode into initialize/resources rather than reduce it. Validation
      followed the plan rule: full launcher build first (`33,418.9ms`), MCP benchmark
      (`14,401.6ms`), focused MCP tests and warm-query benchmark, packaged analyze graph-parity
      smoke (`33,660` nodes / `66,882` relationships, DB load fallback/skipped `0`), full Go tests
      (`28,306.0ms`), browser E2E (`32` passed / `1` skipped, `488,397.8ms` after isolated analyze
      `57,281.5ms`), and `cd avmatrix && npm test` (`385,336.0ms`).
- [ ] Re-run graph parity after every optimization.

Exit gate:

- Optimization candidates are measured, triaged, and either implemented or explicitly queued without
  reducing graph accuracy, contract parity, or required runtime coverage.
- Benchmark artifacts prove graph parity and show the before/after or current-state measurement used
  for each optimization decision.

## Phase 16 - Launcher Integration

The launcher display UX stays unchanged, but all non-Web UI launcher/runtime-control code must move
to Go before cutover.

- [x] Decide binary name and location for Go backend:
      `avmatrix-launcher/server-bundle/avmatrix.exe` is the packaged Go CLI/backend binary, and
      `avmatrix-launcher/server-bundle/avmatrix-server.exe` is the Go server-wrapper launcher
      entrypoint.
- [x] Replace the current server-wrapper path that starts bundled `node.exe` plus
      `avmatrix/dist/cli/index.js serve` with the real Go backend binary.
- [x] Port launcher process start/stop/reset behavior to Go or to a Go-owned packaging path.
- [x] Preserve direct `avmatrix serve` entrypoint.
- [x] Preserve reset/stop behavior.
- [x] Preserve protocol registration if currently supported.
- [x] Remove bundled Node from the normal launcher backend path before cutover.
- [x] Build full launcher package.
- [x] Browser-test launcher UI with Go backend.
- [ ] [P16-NON-WEB-NODE-AUDIT-REOPENED-2026-05-14] Reopen the no-Node launcher/package gate for
      the full conversion scope. The existing proof that the normal launcher backend path uses the
      Go binary remains valid, but it is not enough for final cutover while root/package scripts,
      Docker/web server scripts, hooks, and Node/Vitest harnesses may still act as non-Web support
      authority. Phase-jump reason: this returns from Phase 17 correction to Phase 16 because
      launcher/distribution proof must include the final package shape, not just the running backend
      process.
- [x] [P16-LAUNCHER-BUILD-FAIL-CLOSED-CGO-2026-05-14] Harden `avmatrix-launcher/build.ps1` so
      native command failures cannot be swallowed by PowerShell. The launcher build now checks
      `go version`, Web build, backend Go build, launcher build, server-wrapper build, and protocol
      registration exit codes, and forces `CGO_ENABLED=1` for the LadybugDB/tree-sitter backend
      build before restoring the caller environment.
- [x] [P16-LAUNCHER-UI-CLOSE-PROCESS-LIFECYCLE] Fix launcher lifecycle UX found while testing the
      Web UI repo picker: closing the UI/browser session can leave `AVmatrixLauncher.exe`,
      `avmatrix-server.exe`, or packaged `avmatrix.exe serve` running and holding locks under
      `avmatrix-launcher\server-bundle`.
      Phase-jump note: this must be handled as Phase 16/17 launcher/cutover behavior, not Phase 15
      performance and not as a user-error workaround. A launcher-started UI session needs an
      explicit owner/lifetime model; closing the UI or exiting the launcher should stop the backend
      it owns, without killing unrelated user-started `avmatrix serve` sessions. Do not mark this
      complete until it is validated by launcher build, a close-flow smoke, and a post-close
      process/lock check showing the bundle can be rebuilt or replaced. Implementation: the
      launcher-served `index.html` now receives a launcher-only lifecycle heartbeat script; the
      launcher exposes local heartbeat/closed endpoints on its static Web UI server, waits for a
      UI-done signal in `waitForExit`, and then shuts down only the backend process it started.
      `AVMATRIX_LAUNCHER_NO_BROWSER=1` is a test-only escape hatch so lifecycle smoke can run
      without opening the user's browser profile. Validation followed the plan rule: full launcher
      build first (`47,045.1ms` final build), launcher/server-wrapper tests after build
      (`5,110.9ms` / `2,052.4ms`), HTTP close-flow smoke (`15,647.2ms`) proving no launcher bundle
      processes remained and both packaged binaries could be opened with exclusive file locks, and
      Playwright close-flow E2E (`27,569ms`) proving Chromium page close triggers process cleanup.
      `cd avmatrix && npm test` passed in `415,943.7ms`.

Exit gate:

- Launcher starts Go backend and existing Web UI works. Phase 16 evidence: packaged launcher start
  reached backend `127.0.0.1:4747` and Web UI `127.0.0.1:5173`; Playwright clicked the
  `AVmatrix-GO` repo card in the packaged UI and reached `READY` with graph stats; stop left no
  packaged launcher processes running.

## Phase 17 - Cutover Criteria

The Go implementation can replace the TypeScript runtime only when all are true:

- [x] CLI contract tests pass.
- [x] Direct CLI tool command tests pass.
- [x] HTTP API contract tests pass.
- [x] MCP contract tests pass.
- [x] Group/contract tooling tests pass.
- [x] Search/embed/session runtime tests pass.
- [x] LadybugDB persistence/readback tests pass.
- [x] TypeScript/JavaScript provider parity passes.
- [x] All existing language provider parity gates pass.
- [x] Large-repo graph parity report passes.
- [x] Large-repo speed benchmark passes.
- [x] Web UI browser validation passes.
- [x] Launcher build and smoke validation pass.
- [x] Rollback path is documented in `docs/cutover-rollback.md`.
- [ ] No TypeScript/Node process or non-Web TypeScript/JavaScript implementation remains in the
      final cutover runtime, distribution, build, contract, test-harness, setup, packaging, or
      support surfaces. The Web UI display/build surface is the only TypeScript/React
      implementation allowed to remain. 2026-05-14 correction: the earlier "normal path uses Go"
      tick was premature because `avmatrix/src`, `avmatrix-shared`, root/server scripts, package
      scripts, and TypeScript test harnesses still exist and were not fully converted.
- [ ] Any remaining TypeScript contract code is browser-only Go-generated glue for the Web UI
      build, not runtime authority for backend/CLI/MCP/analyzer behavior. 2026-05-14 correction:
      this gate is reopened until `avmatrix-shared` and all non-Web contract consumers are removed,
      ported to Go, or classified as excluded baseline artifacts.

Cutover validation batch notes:

- [x] Full launcher build must remain the first command before test batches. Latest batch:
      `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed before CLI,
      Go runtime, Web unit, and Playwright E2E verification.
- [x] Go runtime test target is `go test ./cmd/... ./internal/... -count=1`; root
      `go test ./...` is not a cutover gate because it traverses analyzer fixture corpora under
      `avmatrix/test/fixtures` that intentionally contain non-buildable Go/C examples.
- [x] Direct packaged CLI command coverage includes `avmatrix.exe --help`, `avmatrix.exe status`,
      and `avmatrix.exe list`. The `list` command was added during Phase 17 because the packaged Go
      CLI initially lacked the registry listing command.
- [x] Browser E2E validation uses the packaged launcher on `127.0.0.1` and `npx playwright test
      --workers=1`; the latest full browser suite passed `33/33`.
- [x] Docker CLI runtime image now builds `cmd/avmatrix` as a Go binary with Go `1.26.3` on
      Debian slim, auto-resolves the latest stable LadybugDB native release once per UTC day,
      builds with `-tags ladybugdb`, contains no Node runtime, and starts `avmatrix serve --host
      0.0.0.0 --port 4747`.
- [x] Docker Compose server healthcheck uses finite `/api/info`; `/api/heartbeat` remains the
      long-lived SSE heartbeat stream and is not used as a healthcheck.
- [x] E2E CI backend path is ported from Node/TypeScript backend build/analyze/serve to the Go
      backend binary: `actions/setup-go@v6.4.0`, latest-stable LadybugDB native bootstrap,
      `go build -tags ladybugdb ./cmd/avmatrix`, Go `analyze --force`, Go `serve`,
      `127.0.0.1`, and Playwright `--workers=1`.
- [x] Phase 17 current-repo large benchmark/parity rerun is required before ticking large-repo
      graph parity or speed. At commit `5d64ece`, TypeScript baseline on this repo produced `28,731`
      nodes and `52,686` graph-snapshot relationships in `150,292.7ms`; packaged Go produced
      `31,829` nodes and `55,816` relationships in `15,668.3ms` (`~9.59x` faster), but graph
      parity failed on relationship and node-label counts. After native DB packaging was wired, the
      packaged Go rerun produced `31,838` nodes, `55,824` relationships, and `66,233.7ms` with active
      DB load (`nodeRows=31,549`, `relationshipRows=55,824`, `fallbackInsertCount=0`). Speed remains
      open until graph deltas are reconciled. After the process parity fix, packaged Go produced
      `32,348` nodes, `58,091` relationships, and `48,604.2ms`; `Process` improved from `75` to
      `556`, `STEP_IN_PROCESS` from `293` to `1,954`, and route/tool entry links are restored.
      After Go package import expansion was restored, `IMPORTS` improved from `201` to `2,140`
      (`TS=2,367`, remaining delta `-227`). After CALLS compatibility fallback and DB schema
      reconciliation, `CALLS` improved from `7,004` to `8,746`, `STEP_IN_PROCESS` improved to
      `2,671` (`TS=2,646` in the latest diagnostic), and packaged DB load reported
      `fallbackInsertFailures=0` plus `skippedRelationships=0`. After call-return type binding
      enrichment, imported factory calls such as `graph.New()` can type later receiver calls such
      as `g.AddNode()`, improving `CALLS` to `8,906` and `STEP_IN_PROCESS` to `2,682` while DB load
      remains zero-skip. Final parity classification artifact
      `.tmp\phase17-cutover-parity-summary-current.json` compares TypeScript diagnostic
      `28,775` nodes / `52,792` relationships / `133,541ms` with packaged Go `32,767` nodes /
      `64,567` relationships / `61,562.9ms` and DB load `fallbackInsertFailures=0`,
      `skippedRelationships=0`. Remaining graph deltas are classified in the next checklist item;
      Go is `~2.17x` faster on the accepted current-repo rerun while doing real DB load.
- [x] Resolve the packaged native LadybugDB blocker before cutover: launcher, Docker, and CI now
      auto-resolve the latest stable LadybugDB native release once per UTC day, build the backend
      with `-tags ladybugdb`, and place the required DLL/SO beside the packaged runtime. Rollback
      proof: after lowering the local cache to `v0.15.0` with a stale date and removing the current
      runtime directory, a full launcher build refreshed to latest `v0.16.1` and rebuilt
      successfully. Packaged and container benchmark JSON now include real `dbLoad` metrics instead
      of `dbLoad.skipped=true`.
- [x] Reconcile current graph deltas before Phase 17 completion, especially `CALLS`, `IMPORTS`,
      `STEP_IN_PROCESS`, `USES`, `ACCESSES`, `DEFINES`, `HAS_PROPERTY`, `Community`, `Process`,
      `Variable`, `Const`, and document whether each delta is intended improved coverage or a Go
      parity bug. Process-family delta is partly fixed: Go now uses the TypeScript-style dynamic
      process budget and links Route/Tool entry resources back to processes. Import-family delta is
      partly fixed: Go now expands local Go package imports to package files and excludes package
      `_test.go` targets, improving `IMPORTS` from `201` to `2,140`; the remaining `IMPORTS` delta
      must be classified with unresolved/non-Go import policy. CALL-family reconciliation is partly
      fixed: same-file top-level calls, imported package/member calls, global arity-compatible
      calls, and per-call semantic edge keys improved `CALLS` from `7,004` to `8,746`, and the
      required runtime-emitted relation pairs were wired into the LadybugDB schema so DB load no
      longer drops those edges. Call-return type binding enrichment then improved `CALLS` to
      `8,906` by resolving member calls through imported factory return types. The remaining
      `CALLS` deficit (`TS=10,589`, Go `8,906`, delta `-1,683`) is classified as
      TypeScript-baseline over-resolution plus source-label differences, dominated by global/member
      false positives such as `testingFataler.Fatalf` (`TS=541`, Go `1`),
      `testingFataler.Helper` (`TS=217`, Go `1`), and `FileContentCache.set` (`TS=307`, Go `1`).
      Go deliberately does not widen explicit receiver member calls through global fallback. The
      remaining `IMPORTS` deficit (`TS=2,370`, Go `2,140`, delta `-230`) is classified as
      TypeScript-baseline cross-language/path false positives such as Go files importing
      TypeScript/shared helper files; Go keeps language-aware Go package expansion and excludes
      `_test.go` package targets. `STEP_IN_PROCESS` is closed for cutover (`TS=2,646`, Go `2,682`,
      delta `+36`). Expanded `USES`, `ACCESSES`, `DEFINES`, `HAS_PROPERTY`, `Variable`, `Package`,
      `TypeAlias`, `Section`, and `Community` coverage is intentional Go ScopeIR/provider coverage
      and not a speed shortcut.
- [x] Accept the current-repo large graph parity report for the Phase 17 cutover scope after the
      delta classification above. This closes the graph-parity gate only; it does not close the
      overall cutover because the TypeScript/Node runtime-authority gates below remain open.
- [x] Accept the current-repo speed benchmark for the Phase 17 cutover scope: packaged Go is
      approximately `2.17x` faster than the TypeScript diagnostic baseline on the accepted rerun
      while doing real LadybugDB load with zero fallback insert failures and zero skipped
      relationships.
- [x] [P17-TYPESCRIPT-RUNTIME-AUDIT-2026-05-13] Audit remaining TypeScript/Node surfaces before
      closing cutover. This audit was reached from the Phase 17 open TypeScript contract/runtime
      gates, not from Phase 15, because it is correctness/runtime-shape work rather than
      performance optimization. Audit result: Phase 17 remains open. The packaged launcher path,
      Docker CLI image, and E2E backend path use Go binaries, but the current developer machine's
      `avmatrix` command still resolves to an npm PowerShell shim that runs
      `node .../node_modules/avmatrix/dist/cli/index.js`; `avmatrix/package.json` still exposes the
      npm `bin` as `dist/cli/index.js` and keeps TypeScript/Node build scripts and runtime
      dependencies; release/publish workflows still publish the TypeScript npm package; and
      `avmatrix-shared` remains a TypeScript contract source consumed by the legacy CLI and Web.
      Therefore the checklist item below was deliberately not ticked as "cutover complete".
- [x] [P17-GO-CLI-DISTRIBUTION-CUTOVER] Resolve the remaining normal-local-CLI distribution
      blocker found by `[P17-TYPESCRIPT-RUNTIME-AUDIT-2026-05-13]`: the committed cutover path must
      make `avmatrix` resolve to the Go runtime for CLI/analyze/serve/MCP/repo registry/search/embed
      session/benchmark/group tooling, or must quarantine the legacy npm TypeScript package so it is
      explicitly baseline/test-only and cannot be mistaken for the normal runtime authority. After
      this is fixed, return to the "No TypeScript/Node process remains required" Phase 17 gate and
      re-run the audit before ticking it.
- [x] [P17-GO-CLI-DISTRIBUTION-LOCAL-PATH-2026-05-13] Local npm/PATH distribution slice is
      cut over to the Go runtime. Phase-jump note: this stayed in Phase 17 because it is normal
      runtime authority/cutover work, not Phase 15 performance optimization. `avmatrix/package.json`
      now exposes `bin/avmatrix.exe`, package build/postinstall build `cmd/avmatrix` with
      `-tags ladybugdb` and copy the LadybugDB native runtime beside the binary, and the Claude hook
      resolves the native package binary before falling back to PATH/npx. Validation order followed
      the cutover rule: full launcher build first, then package build, Go tests, targeted package/hook
      unit tests, Playwright E2E `33/33` against `avmatrix/bin/avmatrix.exe serve`, npm link PATH
      verification, and `npm publish --dry-run`. Benchmark: old PATH npm shim median `133.7ms`
      (`node .../dist/cli/index.js`) vs new PATH npm shim median `46.5ms` (`bin/avmatrix.exe`);
      direct package Go binary median `29.6ms`. This proves the local developer/runtime command no
      longer uses the TypeScript CLI authority.
- [x] [P17-GO-CLI-DISTRIBUTION-PORTABLE-NPM] Finish the portable npm publish/install story before
      ticking `[P17-GO-CLI-DISTRIBUTION-CUTOVER]` or the normal-runtime no-Node gate. The Windows
      dry-run package contains the Windows Go binary and `lbug_shared.dll`, but the portable path is
      now source-build rather than publisher-platform binary reuse: `prepack` copies the minimal Go
      source set into `go-src`, `postpack` removes it from the working tree, and package
      `postinstall` builds `cmd/avmatrix` from `node_modules/avmatrix/go-src` when the repo root Go
      source is absent. Validation installed the generated tarball into `.tmp\npm-portable-install`
      outside the repo source path; the installed runtime metadata reported `source="go-src"` and
      `node_modules\.bin\avmatrix.cmd --help` returned the Go CLI help. Benchmark: `npm pack`
      `48,769.9ms`, tarball `go-src` files `212`, `npm install` source-build `337,534.1ms`, and
      installed help `1,481.1ms`. This closes portable npm source-build selection, but not the full
      CLI cutover gate below because the command surface audit found remaining TypeScript-only CLI
      commands.
- [x] [P17-GO-CLI-COMMAND-SURFACE-CUTOVER] Port or explicitly quarantine the remaining
      TypeScript-only CLI command surface before ticking `[P17-GO-CLI-DISTRIBUTION-CUTOVER]`.
      Current Go CLI help exposes `analyze`, `augment`, `context`, `cypher`, `detect-changes`,
      `impact`, `list`, `mcp`, `query`, `serve`, `setup`, `status`, `version`, and
      `wiki`/`wiki-mode`;
      follow-up Phase 17 batches added `benchmark-compare`, `clean`, and `index` plus remaining
      analyze flag parity, then `group` subcommands, and finally `setup`. The legacy TypeScript CLI
      still has command files for `skill-gen`, `tool`, and AI-context support, but those are now
      explicitly quarantined as baseline/test-only sources: Go owns direct tool commands via
      `internal/cli/tool_command.go`, and Go owns AI-context/skill generation through
      `analyze --skills` plus `internal/aicontext`. The next gate is a fresh TypeScript/Node audit
      before ticking `[P17-GO-CLI-DISTRIBUTION-CUTOVER]` or the normal-runtime no-Node gate.
- [x] [P17-GO-CLI-DIRECT-TOOLS-COMMAND-SURFACE-2026-05-13] Port the direct graph-tool CLI command
      batch inside `[P17-GO-CLI-COMMAND-SURFACE-CUTOVER]`. Phase-jump note: this stayed in Phase 17
      because direct tool commands are normal CLI runtime/cutover behavior, not Phase 15 MCP
      optimization. Go now owns `augment`, `query`, `context`, `impact`, `cypher`, and
      `detect-changes` CLI entrypoints by routing them to the in-process Go MCP tool handlers.
      Direct tool compatibility flags were added for repo selection, task context, goal, limit,
      UID/file disambiguation, content inclusion, direction/depth/include-tests, and diff scope/base
      ref. Validation order followed the cutover rule: full launcher build first, then package
      build, Go tests, direct CLI command benchmarks, and Playwright E2E with `33` tests listed.
      Benchmark smoke against `avmatrix\bin\avmatrix.exe` exited `0` for `query`, `context`,
      `impact`, `cypher`, `detect-changes`, and `augment`; `augment` now uses the current working
      directory as the repo hint, matching the old TypeScript command's `process.cwd()` behavior.
      The parent command-surface gate remained open after this batch; the next admin/analyze batch
      closed `clean`, `index`, `benchmark-compare`, and analyze flag parity.
- [x] [P17-GO-CLI-ADMIN-ANALYZE-COMMAND-SURFACE-2026-05-13] Port the local admin/analyze command
      surface batch inside `[P17-GO-CLI-COMMAND-SURFACE-CUTOVER]`. Phase-jump note: this stayed in
      Phase 17 because `clean`, `index`, `benchmark-compare`, and analyze flag parity are normal
      CLI runtime/cutover behavior, not Phase 15 optimization. Go now owns `clean --force/--all`,
      `index [path...] --force --allow-non-git`, `benchmark-compare <before> <after> --json`, and
      analyze flags `--skip-git`, `--skip-compatibility-cross-file`, `--benchmark-label`, `--name`,
      `--allow-duplicate-name`, and `--verbose`. Validation order followed the cutover rule: full
      launcher build first, then package build, Go tests, temp-repo command benchmarks, and
      Playwright E2E with `33` tests listed. The parent command-surface gate remained open after
      this batch; the group batch below closes the `group` portion.
- [x] [P17-GO-CLI-GROUP-COMMAND-SURFACE-2026-05-13] Port the group command surface inside
      `[P17-GO-CLI-COMMAND-SURFACE-CUTOVER]`. Phase-jump note: this stayed in Phase 17 because
      `group create/add/remove/list/status/sync/query/contracts` are normal cross-repo CLI runtime
      behavior, not Phase 15 optimization. Go now owns the group subcommands and flags from the CLI
      contract, backed by the existing Go `internal/group` service. Validation order followed the
      cutover rule: full launcher build first, then package build, Go tests, temp-fixture group
      command benchmarks, and Playwright E2E with `33` tests listed. The parent command-surface gate
      remains open only for `setup` plus explicit quarantine/retirement decisions for legacy
      `skill-gen`, `tool`, and AI-context CLI files.
- [x] [P17-GO-CLI-SETUP-QUARANTINE-COMMAND-SURFACE-2026-05-13] Port the remaining setup command
      and close the legacy command-surface quarantine inside `[P17-GO-CLI-COMMAND-SURFACE-CUTOVER]`.
      Phase-jump note: this stayed in Phase 17 because editor MCP setup, global skills, Claude
      hooks, direct tool commands, and AI-context/skill generation are normal local runtime/cutover
      behavior, not Phase 15 optimization. Go now owns `avmatrix setup` for Cursor, Claude Code,
      OpenCode, and Codex MCP config, packaged skill installation, Claude hook installation, and
      Codex fallback TOML idempotency. Legacy `tool.ts`, `skill-gen.ts`, and `ai-context.ts` are
      quarantined as TypeScript baseline/test-only files because the Go direct tool commands and
      `analyze --skills` path own those normal runtime surfaces. Validation order followed the
      cutover rule: full launcher build first, then package build, Go tests, temp-HOME setup command
      benchmarks, and Playwright E2E with `33` tests listed. This closes the parent command-surface
      gate only; the audit below closes the distribution/no-Node portion.
- [x] [P17-TYPESCRIPT-RUNTIME-AUDIT-POST-COMMAND-SURFACE-2026-05-13] Re-run the TypeScript/Node
      runtime audit after closing the command-surface gate. Phase-jump note: this stayed in Phase
      17 because it verifies runtime authority/cutover, not Phase 15 optimization. Audit result:
      the normal local runtime no longer requires TypeScript/Node for CLI, analyze, serve, MCP,
      launcher runtime control, repo registry, search/embed/session bridge, benchmark, group, or
      setup usage. The PATH shim points at `node_modules/avmatrix/bin/avmatrix.exe`; direct package
      binary help and PATH help both print the Go CLI surface; `npm run serve` now invokes
      `bin/avmatrix.exe serve`; `npm run dev` now invokes `go run ../cmd/avmatrix`; and the legacy
      TypeScript watch command is explicitly named `dev:ts-baseline`. Web onboarding now instructs
      `avmatrix serve`, not `cd avmatrix && npm run serve`. Remaining TypeScript under
      `avmatrix/src`, `avmatrix-shared`, and legacy tests is baseline/dev/test material, not normal
      runtime authority; the separate contract-authority gate below remains open.
- [x] [P17-GO-CONTRACT-AUTHORITY-CUTOVER] Resolve the remaining TypeScript contract-authority
      blocker found by `[P17-TYPESCRIPT-RUNTIME-AUDIT-2026-05-13]`: Go must own the backend/CLI/MCP/
      analyzer contract shapes, and any TypeScript left under `avmatrix-shared` or Web imports must
      be generated or browser-only glue for the Web UI build. After this is fixed, return to the
      "Any remaining TypeScript contract code..." Phase 17 gate and re-run the audit before ticking
      it.
      Phase-jump note: this stayed in Phase 17 because it is contract/runtime authority work, not
      Phase 15 MCP/provider optimization. Go now owns the Web contract manifest and generator under
      `internal/contracts` and `cmd/generate-web-contracts`; generated artifacts are
      `contracts/web-ui/avmatrix-web-contract.schema.json` and
      `avmatrix-web/src/generated/avmatrix-contracts.ts`. Web source, Web tests, Vite/Vitest config,
      Web package metadata, lockfile, and Vercel install config no longer import, alias, install, or
      build `avmatrix-shared`. The accepted Web audit returned zero `avmatrix-shared` hits under
      `avmatrix-web` in `55.8ms`. Validation followed the cutover rule: full launcher build first
      (`38,426.7ms`), Go tests (`29,795.3ms`), targeted Web contract unit tests (`13,013.4ms`,
      `130` tests), generator check (`559.9ms`), fresh `npm ci` (`50,004.7ms`), post-install full
      launcher build (`33,223.8ms`), Playwright E2E against the packaged Go backend (`205,536.5ms`,
      `33/33`), and `cd avmatrix && npm test` (`338,908.6ms`). `SessionStatusResponse` impact was
      HIGH because both Web and legacy TypeScript session surfaces consume it, but this slice did
      not change the JSON payload shape; it only moved the Web contract source to Go-generated
      browser glue. Legacy `avmatrix-shared` and `avmatrix/src` TypeScript remain baseline/dev/test
      material outside the normal cutover runtime path, not backend/CLI/MCP/analyzer/Web runtime
      authority.
- [x] [P17-NON-WEB-TSJS-AUDIT-CORRECTION-2026-05-14] Re-audit the plan goal after the user
      correction that this is a full repository conversion, not only a main-path runtime cutover.
      Phase-jump note: this stays in Phase 17 because the issue is conversion completeness and
      cutover authority, not Phase 15 optimization. Audit command:
      `rg --files -g '*.ts' -g '*.tsx' -g '*.js' -g '*.jsx' -g '*.mjs' -g '*.cjs'` with
      `node_modules`, `dist`, `build`, `vendor`, and launcher bundle exclusions. Result: `1051`
      TypeScript/JavaScript-family files remain in non-generated source areas: `avmatrix=895`,
      `avmatrix-web=119`, `avmatrix-shared=34`, root Docker/web-server scripts `2`, and root ESLint
      config `1`. Inside `avmatrix`, the split is `test=542`, `src=339`, `scripts=8`, `vendor=4`,
      `hooks=1`, and `vitest.config.ts=1`. The blocker is not the Web UI count; it is the remaining
      `avmatrix/src`, `avmatrix-shared`, root scripts, package scripts, and TypeScript test
      harness/support surface. Therefore the earlier Phase 17 ticks for "Web UI is the only
      TypeScript/React surface" and "remaining TypeScript contract code is browser-only generated
      glue" were premature and are reopened above.
- [ ] [P17-NON-WEB-TSJS-CONVERSION-BLOCKER] Convert, delete, or cutover-exclude every non-Web UI
      TypeScript/JavaScript surface before declaring `AVmatrix-GO` complete. Required work packages:
      `avmatrix/src` legacy CLI/server/MCP/analyzer/core implementation must be ported to Go or
      removed from final runtime/distribution; `avmatrix-shared` must stop being an independent
      TypeScript contract authority outside Go-generated Web glue; `avmatrix/test` and legacy
      Vitest integration harnesses must either move to Go/browser tests or be explicitly retained
      only as baseline fixtures outside the final package; package scripts such as build,
      postinstall, prepack/postpack, Docker web server scripts, hooks, and config files must be
      ported, replaced with Go/native tooling, or classified as build-only Web support with
      exclusion proof. This item blocks `[P17-INDEPENDENT-GO-MCP-READINESS-PROOF]`.
      File-level tracker: `docs/plans/2026-05-08-avmatrix-go-typescript-node-to-go-conversion-remaining-files.md`
      lists `640` tracked non-Web TS/JS/CJS/MJS files outside `avmatrix-web/` and
      `avmatrix/test/fixtures/`; every file there must be ticked through a port, removal/exclusion,
      or explicit non-runtime reclassification before this blocker closes.
      Translation-first cutover rule: process legacy implementation files by translating their
      behavior 1:1 into Go first, keeping contracts, flags, output, path layout, side effects,
      edge cases, and tests as close to the TypeScript/JavaScript source as practical. After the Go
      translation exists, switch the relevant entrypoints/package scripts/tests/runtime authority
      to the Go path, then run the required build, tests, E2E, benchmark, and evidence package.
      Only after the Go path is proven to run correctly may the legacy TypeScript/JavaScript files
      in that behavior cluster be deleted or removed from package/distribution/runtime authority.
      Reclassification is reserved for non-executed fixture/baseline data; obsolete implementation
      source must not remain as repo garbage after its Go translation is accepted.
      Validation cadence is behavior-cluster based, not per-file: translate the whole coherent
      cluster first, then run one full validation package for that completed cluster. Do not spend a
      full build/test/E2E cycle after each individual file inside the same cluster; tick individual
      tracker files only after the cluster-level Go path is proven and the legacy authority is
      retired.
- [x] [P17-WEB-DOCKER-NODE-SERVER-REMOVAL-2026-05-14] Process the root Docker/web-server matrix
      slice: replace the Web container runtime Node static server with nginx, delete
      `docker-server.mjs` and `docker-server.test.mjs`, add `docker/web-nginx.conf` with SPA
      fallback plus COOP/COEP/cache headers, and validate through full launcher build, Go tests,
      static nginx/Dockerfile checks, AVmatrix refresh, and Playwright E2E. Docker daemon was not
      running in this environment, so image build remains a later daemon-available smoke instead of
      a reason to keep the Node server.
- [x] [P17-CLAUDE-HOOK-GO-TRANSLATION-RETIREMENT-2026-05-14] Process the Claude hook runtime
      support cluster from `[P17-NON-WEB-TSJS-CONVERSION-BLOCKER]`. Phase-jump note: this stays in
      Phase 17/16 package-support cutover because the work is editor hook runtime authority, not
      Phase 15 optimization and not independent MCP readiness. The legacy
      `avmatrix/hooks/claude/avmatrix-hook.cjs` behavior was translated into the hidden Go command
      `avmatrix hook claude`: PreToolUse still augments Grep/Glob/Bash searches through
      `augment -- <pattern>`, PostToolUse still checks git mutation success against
      `.avmatrix/meta.json`, and invalid/missing input still fails silently. Setup now writes the
      Go hook command directly, removes old copied `avmatrix-hook.cjs` entries, and package
      metadata no longer ships the `hooks` directory. The legacy CJS hook plus shell wrappers were
      deleted after cluster-level validation. Required validation ran once for the cluster: full
      package build before tests, Go tests, TypeScript check, full `npm test` including e2e suites,
      Go hook PreToolUse/PostToolUse smoke, setup temp-HOME smoke, AVmatrix refresh, and
      final staged `detect_changes`. `detect_changes` reported HIGH because `NewRootCommand`
      changed and fans into CLI flows; this was expected for adding the hidden hook subcommand and
      was covered by CLI/setup/hook validation.
- [x] [P17-NON-WEB-TSJS-CLASSIFICATION-MATRIX] Start the next work inside Phase 17 by classifying
      the remaining non-Web TypeScript/JavaScript inventory before editing code. Phase-jump note:
      this intentionally stays in Phase 17 after the Phase 15 optimization loop and before the
      independent MCP smoke because the active work is conversion completeness and correctness, not
      performance optimization or superficial install proof. Required categories: Web UI allowed
      TS/React; analyzer fixtures/source-language samples; legacy runtime implementation to port or
      delete; Node/Vitest test harnesses that should move to Go where practical; package/build
      glue that may remain only if it is Web/npm ecosystem support and not runtime authority; and
      generated browser contract glue. Each category must record expected action, evidence, and
      benchmark/validation method before implementation resumes. Classification result:
      `avmatrix-web=119` is allowed Web UI/display/build/test surface, including
      `avmatrix-web/src/generated/avmatrix-contracts.ts` as Go-generated browser glue; `avmatrix/src=339`
      is legacy non-Web implementation and must be removed from runtime/package authority after Go
      equivalents are proven; `avmatrix-shared=34` is legacy TypeScript contract authority and must
      be replaced by Go-owned contracts/generated browser glue or removed; `avmatrix/test=542`
      splits into `fixtures=290` that may remain as analyzer input data and `Node/Vitest harness=252`
      that should be converted to Go/browser tests where practical or excluded from the final
      package; `avmatrix/scripts=8`, `avmatrix/hooks=1`, `avmatrix/vendor=4`, root Docker/web-server
      scripts `2`, and root ESLint config `1` are support/package/build surfaces and require
      per-file decisions before final cutover. The matrix closes classification only; it does not
      close `[P17-NON-WEB-TSJS-CONVERSION-BLOCKER]`.
- [ ] [P17-INDEPENDENT-GO-MCP-READINESS-PROOF] Prove the real plan goal before declaring victory:
      `AVmatrix-GO` must be an independent Go MCP/tool implementation, separate from the existing
      `AVmatrix-main` MCP. It may be installed under any MCP server name such as `avmatrix-go`,
      or run side-by-side with the existing `avmatrix` MCP from `AVmatrix-main`; this plan must not
      assume or require overwriting the user's existing machine-wide MCP config. Phase-jump note:
      this returns from Phase 15 optimization to Phase 17 because the open question is not "can
      another benchmark be improved?" but "does this repo produce a correct, complete Go MCP/tool
      that does not depend on `AVmatrix-main`?" Required package: final independent-runtime audit,
      standalone local CLI/analyze/serve/MCP/launcher proof, Web UI proof, DB load/readback proof
      with fail-closed behavior and fallback/skipped `0`, fresh-repo and representative repo
      validation beyond fixtures, provider parity classification, rollback proof, dirty-state
      proof, and multi-platform/package proof. Benchmarks must still be recorded for each
      validation batch, but they are evidence and regression detectors, not permission to keep doing
      Phase 15 micro-optimization before this proof is closed.

## Risk Register

| Risk | Severity | Mitigation |
| --- | --- | --- |
| Go port is faster because it misses graph facts | High | Exact fixture parity and edge-count parity before speed claims |
| LadybugDB Go support is incomplete | High | Prove DB load/read in Phase 1 before broad analyzer work |
| Tree-sitter grammar setup is unstable on Windows | High | Prove Windows parser runtime in Phase 1 before provider work |
| Go port pins stale runtime dependencies | High | Phase 1 latest-stable dependency check with documented fallback only when blocked |
| HTTP API drifts and Web UI breaks | High | API snapshot tests and browser validation |
| MCP payloads drift | Medium | MCP snapshot tests against the local TypeScript baseline |
| Repo identity regresses to name-based routing | High | Path-first duplicate-name tests from Phase 3 onward |
| Rewrite takes too long before usable output | High | Small runnable milestones and rollback commits |
| Provider parity becomes unbounded | High | Port language providers one by one with explicit acceptance gates |
| Hidden TypeScript runtime dependency remains after cutover | High | Cutover gate requires no TypeScript/Node runtime except the Web UI |

## Definition Of Done

The conversion is done only when:

- Go backend/CLI can analyze the same target repositories as TypeScript AVmatrix.
- Go graph output is better than the currently used AVmatrix-main behavior where the Go conversion
  has identified legacy inaccuracies, and at minimum passes the defined parity gates for behavior
  that must stay compatible.
- Go runtime is not slower at equivalent accuracy, and the accepted current evidence must continue
  to show Go faster on representative repositories.
- Benchmark evidence is mandatory for each meaningful validation slice, but during conversion it is
  a measurement/regression tool and a light-optimization guide. Heavy optimization must not block
  independent MCP/tool readiness once correctness, parity, and runtime authority are still open.
- Existing Web UI works against the Go backend without contract changes.
- Launcher can start/stop/reset the Go backend.
- MCP tools work against Go-generated graphs.
- Go toolchain and all cutover-path dependencies have recorded latest-stable checks, with any
  fallback justified by a concrete compatibility blocker.
- `AVmatrix-GO` does not require the separate `AVmatrix-main` repository or its TypeScript/Node MCP
  runtime for CLI, analyze, serve, MCP, launcher runtime control, repo registry, search/embed/
  session bridge, benchmark, or group usage.
- The Web UI display/build surface and Go-generated browser contract glue are the only
  TypeScript/React code allowed to remain in the final cutover package. Non-Web TypeScript/
  JavaScript under `avmatrix/src`, `avmatrix-shared`, root scripts, package scripts, hooks, and
  TypeScript test harnesses must be ported to Go, removed, or excluded from the final cutover
  package with evidence that they are baseline fixtures only and cannot become runtime authority.

## Immediate Next Steps

- [x] Finish Phase 17 large-repo graph parity report.
- [x] Finish Phase 17 large-repo speed benchmark.
- [x] Document the Phase 17 rollback path.
- [x] Run the first remaining TypeScript/Node audit for Phase 17 and record that it does not close
      cutover yet because CLI distribution and contract-authority blockers remain.
- [x] Resolve the local PATH/npm-link portion of `[P17-GO-CLI-DISTRIBUTION-CUTOVER]` and record the
      benchmark/evidence before commit.
- [x] Resolve `[P17-GO-CLI-DISTRIBUTION-PORTABLE-NPM]` with a source-build npm package path and
      record the tarball install benchmark/evidence before commit.
- [x] Port the Phase 17 direct graph-tool command batch (`augment`, `query`, `context`, `impact`,
      `cypher`, `detect-changes`) and record the benchmark/evidence before commit.
- [x] Port the Phase 17 local admin/analyze command batch (`clean`, `index`, `benchmark-compare`,
      remaining analyze flags) and record the benchmark/evidence before commit.
- [x] Port the Phase 17 group command batch and record the benchmark/evidence before commit.
- [x] Resolve `[P17-GO-CLI-COMMAND-SURFACE-CUTOVER]` with the setup/quarantine batch and record the
      benchmark/evidence before commit.
- [x] Re-run the Phase 17 TypeScript/Node audit before ticking
      `[P17-GO-CLI-DISTRIBUTION-CUTOVER]` or the normal-runtime no-Node gate.
- [x] Resolve `[P17-GO-CONTRACT-AUTHORITY-CUTOVER]`, then re-run the Phase 17 TypeScript contract
      audit before ticking the browser-only generated-glue gate.
- [x] Correct the Phase 17 plan after the 2026-05-14 non-Web TypeScript/JavaScript audit: the
      previous "normal path uses Go" interpretation is insufficient for this conversion. Reopen the
      final TypeScript/Node cutover gates and add `[P17-NON-WEB-TSJS-CONVERSION-BLOCKER]`.
- [x] Correct the conversion priority: the tool must run correctly and accurately in Go, with
      benchmark/evidence retained as validation and light optimization signals. Do not spend Phase
      15-style heavy optimization time while conversion completeness, correctness, and independent
      usability remain open.
- [x] Reopen the non-Phase-17 gates affected by the corrected conversion scope:
      `[P1-CONTRACT-AUTHORITY-REOPENED-2026-05-14]`,
      `[P16-NON-WEB-NODE-AUDIT-REOPENED-2026-05-14]`, and
      `[P15-DEFER-UNTIL-CONVERSION-CORRECTNESS-2026-05-14]`. Phase-jump reason: these phases were
      previously treated as complete because the normal Go path worked, but full conversion still
      requires contract authority, distribution/package shape, and optimization sequencing to be
      revisited.
- [x] Record the phase-jump after the priority correction: work returns/stays in Phase 17 because
      the next task is non-Web TypeScript/JavaScript conversion classification and correctness, not
      Phase 15 optimization and not an independent MCP smoke.
- [x] Build `[P17-NON-WEB-TSJS-CLASSIFICATION-MATRIX]` before implementation resumes. The matrix
      is recorded in Phase 17 and in evidence/benchmark files using AVmatrix-main refresh plus file
      inventory commands.
- [x] Add the file-level remaining TS/JS checklist and align phase reopen status against it. Result:
      no new phase beyond the existing reopened/deferred set is needed. Phase 1 remains open for
      contract authority (`avmatrix-shared` and legacy non-Web contract consumers), Phase 16 remains
      open for package/distribution/support surfaces (`avmatrix/scripts`, `avmatrix/hooks`,
      `avmatrix/vendor`, and config files), Phase 17 remains open for the full non-Web conversion
      blocker and independent Go readiness proof, and Phase 15 remains deferred until those
      correctness/cutover blockers close. Phases 8, 11, 13, and 14 stay closed unless a future
      conversion slice finds a real Go behavior/parity gap; the files in the checklist are tracked
      as legacy baseline, harness, packaging, or retirement surfaces rather than evidence that the
      already-validated Go provider/analyze/MCP/language phase gates are reopened.
- [x] Mark high-risk conversion surfaces in the file-level tracker before processing the matrix:
      files that generate `AGENTS.md`, `CLAUDE.md`, generated skills, editor MCP/skills/hooks
      config, or other agent-facing instructions must preserve generated markdown semantics,
      section markers, idempotent upserts, skip flags, and path layout before their checklist items
      can be ticked. Examples include `avmatrix/src/cli/ai-context.ts`,
      `avmatrix/src/cli/skill-gen.ts`, `avmatrix/src/core/run-analyze.ts`,
      `avmatrix/src/cli/analyze.ts`, and `avmatrix/src/cli/setup.ts`.
      Phase-jump note: this is a Phase 17 conversion-completeness guard over behavior that belongs
      to Phase 11. Phase 11 remains closed while Go output parity is preserved; any discovered
      behavior drift in generated agent instructions or setup/config semantics must create a
      reopened Phase 11 item with evidence before the Phase 17 checklist item is ticked.
- [ ] Process the matrix group by group with evidence, benchmark/validation, checklist updates, and
      commits. Start with a small implementation slice that removes or excludes one non-Web
      TypeScript/JavaScript authority surface without touching allowed Web UI or analyzer fixtures.
- [x] First matrix reduction slice completed: root Docker/web-server Node runtime was removed in
      favor of nginx Web static serving, and launcher build fail-closed behavior was hardened after
      the slice exposed a swallowed Go backend build failure.
- [x] Second matrix reduction slice completed: Claude hook runtime support was translated into the
      hidden Go `avmatrix hook claude` command, setup now installs that command directly, legacy
      copied CJS/shell hook files were deleted, evidence/benchmark were recorded, and the remaining
      file tracker ticked `avmatrix/hooks/claude/avmatrix-hook.cjs`.
- [ ] Stop independent MCP readiness proof until `[P17-NON-WEB-TSJS-CONVERSION-BLOCKER]` has a
      concrete conversion/removal/exclusion package. The next work must classify and reduce the
      non-Web TS/JS inventory, not run more Phase 15 optimization or a shallow MCP smoke.
- [ ] After Phase 17 cutover gates close, keep Phase 15 MCP/provider optimization as a separate
      follow-up scope; do not start optimization while cutover correctness gates remain open.
      Reopened 2026-05-14 because the Phase 17 cutover gates are no longer considered closed under
      the corrected full-conversion goal.
- [ ] Enter Phase 15 after Phase 17 cutover closure and close the P0 MCP `route_map` hot path with
      benchmark/evidence before commit. Reopened 2026-05-14: earlier Phase 15 entry was based on
      the old assumption that main-path Go runtime cutover was enough. Phase 15 stays deferred
      until the non-Web TS/JS conversion matrix and independent Go tool readiness are closed.
- [x] Jump back from Phase 15 to Phase 14 for the requested frontend/mobile app coverage addendum,
      then record the phase-jump note, benchmark, E2E, and commit for that slice before resuming
      Phase 15.
- [x] Jump back from the Phase 15 `context` optimization slice to Phase 10 LadybugDB fallback
      correctness after reassessing whether fallback can produce wrong data. Record benchmark,
      evidence, E2E, and commit for the fail-closed loader slice before resuming Phase 15.
- [x] Jump back from Phase 15 performance work to Phase 10 LadybugDB schema correctness for the
      real Web UI/new-repo failure `TypeAlias->Method: schema pair unsupported`; record impact,
      before-fail benchmark, after-success benchmark, evidence, E2E, and commit before resuming
      Phase 15.
- [x] Jump from the same Web UI investigation to Phase 16/17 launcher lifecycle cleanup for
      lingering launcher/backend processes after UI close. Do not treat the process lock as a
      manual-test artifact; record the owner/lifetime fix, close-flow evidence, process/lock proof,
      E2E, and commit before returning to Phase 15.
- [x] Resume Phase 15 after the Phase 10 schema correctness jump and Phase 16 launcher lifecycle
      cleanup commits. Selected `[P15-SCOPEIR-OWNED-NORMALIZE-ALLOC-OPT]` from the latest pprof:
      benchmark rejected pure in-place normalization, then accepted the top-level compact owned
      normalization variant as a small provider allocation reduction with neutral full-repo memory.
- [x] Finish the `[P15-SCOPEIR-OWNED-NORMALIZE-ALLOC-OPT]` validation package: full Go tests,
      browser E2E, `cd avmatrix && npm test`, and AVmatrix `detect_changes(scope=all)` all passed
      with benchmark/evidence recorded. Commit this slice before moving to the next target.
- [x] After the ScopeIR owned-normalize commit, continue Phase 15 with the next real pprof-backed
      bottleneck: this selected and closed `[P15-SCOPEIR-RELEASE-AFTER-RESOLUTION-OPT]` because the
      heap profile showed parsed ScopeIRs were retained after resolution even though CLI/Web analyze
      no longer needed them. Native tree-sitter parse cgo, native LadybugDB query/commit/COPY cost,
      residual TS/JS traversal, and residual graph/resolution allocations remain open.
- [x] Continue Phase 15 with the dedicated `context` timing-split/profile slice; keep benchmark,
      graph parity, E2E, and commit discipline for the next slice.
- [x] Continue Phase 15 with the dedicated `impact` timing-split/profile slice; keep benchmark,
      graph parity, E2E, and commit discipline for the next slice.
- [x] Continue Phase 15 with the HTTP `group_sync` cold/warm timing-split slice; keep benchmark,
      graph parity, E2E, and commit discipline for the next slice.
- [x] Continue Phase 15 with the P3 MCP startup/query/tools-list/protocol-noise preservation slice;
      include updated benchmark/evidence and commit before moving to larger pprof/IO/parser work.
- [x] Continue Phase 15 with large-repo graph stream and memory/pprof benchmarks; keep benchmark,
      graph parity, E2E, and commit discipline for the next slice.
- [x] Continue Phase 15 with the first DB-load/cgo optimization candidate, record the rejected
      LadybugDB COPY `PARALLEL=true` result, rollback benchmark, evidence, and commit discipline;
      keep `[P15-DBLOAD-CGO-BATCH-OPT]` open because the performance target is not solved.
- [x] Continue Phase 15 with the next profile-backed bottleneck target:
      `[P15-SCOPEIR-NORMALIZED-ALLOC-OPT]`, including impact, before/after benchmark, pprof, graph
      parity, E2E, evidence, and commit discipline.
- [x] Continue Phase 15 with the next profile-backed bottleneck target:
      `[P15-FRAMEWORK-DEFINITION-WINDOW-ALLOC-OPT]`, including impact, before/after benchmark,
      pprof, graph parity, E2E, evidence, and commit discipline.
- [x] Continue Phase 15 with the next profile-backed bottleneck target: either a safer DB-load/cgo
      batching design that preserves quoted CSV content and fail-closed semantics, or
      `[P15-RESOLUTION-WORKSPACE-ALLOC-OPT]` based on the final framework heap profile.
- [x] Continue Phase 15 with the remaining profile-backed bottleneck target: either a safe
      `[P15-DBLOAD-CGO-BATCH-OPT]` design that preserves quoted CSV content and fail-closed
      semantics, or a new scoped allocation target from the latest heap profile such as
      `callerForScope`, graph relationship allocation, or definition-node emission. Do not mark
      Phase 15 complete until the real bottleneck is reduced or rejected with benchmark/evidence.
- [x] Continue Phase 15 with the still-open native DB-load/cgo target or the next profile-backed
      non-cgo hotspot. This selected and closed `[P15-DBLOAD-CGO-BATCH-OPT]` after the earlier
      `[P15-DBLOAD-SCHEMA-LOOKUP-ALLOC-OPT]` only reduced DB-load export allocation. File IO
      batching, parser pool sizing, and any new pprof-backed graph/resolution allocation target
      remain open until benchmark/evidence closes or rejects them.
- [x] Continue Phase 15 with the remaining open file IO batching and parser pool sizing targets, or
      with the next pprof-backed hotspot if benchmark/evidence shows those two generic checklist
      items are not the real bottleneck. This slice rejected file IO batching and parser pool sizing
      as current bottlenecks with benchmark/pprof evidence, then selected and closed
      `[P15-IMPORTED-MEMBER-INDEX-OPT]` because `resolveImportedMember` was the real post-DB-load
      non-cgo hotspot.
- [x] Continue Phase 15 with the next real pprof-backed bottleneck after
      `[P15-IMPORTED-MEMBER-INDEX-OPT]`: this selected `[P15-PARSER-NODECOUNT-OPT]` after pprof
      showed `countNodes` under `Pool.Parse`, then closed it with benchmark/evidence while
      explicitly not claiming a macro wall-time win.
- [x] Continue Phase 15 with the next real pprof-backed bottleneck after
      `[P15-PARSER-NODECOUNT-OPT]`: this selected `[P15-TSJS-TRAVERSAL-KIND-OPT]` after pprof showed
      repeated TS/JS provider traversal and `node.Kind()` calls under `tsjs.Extract`.
- [x] Continue Phase 15 with the next real pprof-backed bottleneck after
      `[P15-TSJS-TRAVERSAL-KIND-OPT]`: current evidence points at native tree-sitter parse cgo
      (`runtime.cgocall`, `Parser.ParseWithOptions`), remaining TS/JS provider traversal,
      `golang.Extract` traversal, and native DB commit/COPY cost. This selected and closed
      `[P15-GO-PROVIDER-TRAVERSAL-KIND-OPT]` as the safe Go-level provider target; native parse cgo
      and native DB commit/COPY remain open.
- [x] Continue Phase 15 after `[P15-GO-PROVIDER-TRAVERSAL-KIND-OPT]` with the next real
      pprof-backed bottleneck. Current evidence points at native tree-sitter parse cgo
      (`runtime.cgocall`, `Parser.ParseWithOptions`), remaining TS/JS provider traversal, and native
      LadybugDB query/commit/COPY cost. Do not mark Phase 15 complete until the next slice either
      reduces the real remaining bottleneck or explicitly rejects it with benchmark/evidence.
      This selected and closed `[P15-GRAPH-SNAPSHOT-STREAM-OPT]` from the heap pprof because the
      full-graph snapshot buffer was a real current memory bottleneck. Native parse cgo and native
      LadybugDB query/commit/COPY remain open, and the graph snapshot slice is not claimed as a
      macro wall-time speedup.
- [x] Continue Phase 15 after `[P15-GRAPH-SNAPSHOT-STREAM-OPT]` with the next real pprof-backed
      bottleneck. This selected and closed `[P15-SCOPEIR-RELEASE-AFTER-RESOLUTION-OPT]` after the
      post-owned-normalize profile showed retained parsed ScopeIR memory. The accepted claim is a
      retained-heap reduction, not a peak-RSS or macro wall-time win.
- [x] Continue Phase 15 after `[P15-SCOPEIR-RELEASE-AFTER-RESOLUTION-OPT]` with the next real
      pprof-backed bottleneck. This selected and closed
      `[P15-GRAPH-COMPACT-AFTER-PROCESSES-OPT]` because retained graph slice/index memory was the
      next Go-level heap target after parsed ScopeIRs were released. Native parse cgo and native
      LadybugDB query/commit/COPY remain open.
- [x] Continue Phase 15 after `[P15-GRAPH-COMPACT-AFTER-PROCESSES-OPT]` with the next real
      pprof-backed bottleneck. This selected and closed
      `[P15-NATIVE-CGO-BOUNDARY-CLASSIFICATION]`: native tree-sitter parse cgo and native LadybugDB
      query/commit/COPY are the dominant CPU families, but they are rejected as same-slice Go-level
      optimization targets without upstream/native parser or LadybugDB API changes.
- [x] Stop the Phase 15 micro-optimization loop after
      `[P15-NATIVE-CGO-BOUNDARY-CLASSIFICATION]`. The attempted
      `[P15-TSJS-FACT-KIND-DISPATCH-DEFERRED]` patch was measured, found nonessential for the plan
      goal, and reverted from the working tree instead of being committed. Benchmark remains
      required as measurement/regression evidence, but the plan must now answer whether
      `AVmatrix-GO` is a complete independent Go MCP/tool implementation separate from
      `AVmatrix-main`.
- [ ] Return to Phase 17 and close `[P17-INDEPENDENT-GO-MCP-READINESS-PROOF]` before doing more
      Phase 15 optimization. Required focus: final independent-runtime audit, standalone operation,
      fresh/medium/large repo validation, Web UI, MCP, launcher lifecycle, DB correctness,
      provider parity classification, rollback, dirty-state, multi-platform/package proof, and
      evidence updates before commit. Do not overwrite the existing machine-wide `avmatrix` MCP
      during this proof; use an isolated config or a distinct server name when a real MCP install
      smoke is needed.
