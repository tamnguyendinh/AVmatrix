# Architecture - AVmatrix

AVmatrix is a local-first code intelligence system. The backend, CLI, MCP server,
analyzer, storage, contracts, and session bridge are implemented in Go under
`cmd/` and `internal/`. The Web UI remains a thin React/Vite client under
`avmatrix-web/`. The npm package under `avmatrix/` ships the Go runtime binary,
package metadata, skills, and generated Go source fallback for install-time
rebuilds.

## Repository Layout

| Path | Role |
|------|------|
| `cmd/avmatrix/` | CLI entry point. |
| `cmd/generate-web-contracts/` | Regenerates browser-side Web contract glue from Go contract definitions. |
| `internal/analyze/` | End-to-end analyze orchestration, benchmark output, graph loading flow. |
| `internal/scanner/`, `internal/parser/`, `internal/providers/` | File scanning, parser readiness/pool, language-specific ScopeIR extraction. |
| `internal/scopeir/`, `internal/resolution/` | Serialized scope facts, indexes, import/call/reference resolution, audited graph edge emission. |
| `internal/graph/`, `internal/lbugschema/`, `internal/lbugload/`, `internal/lbugruntime/` | Graph model, LadybugDB schema, CSV/load path, query/runtime helpers. |
| `internal/httpapi/` | Local HTTP API used by the Web UI and launcher. |
| `internal/mcp/` | MCP stdio server, tools, resources, prompts, impact, context, detect-changes, rename support. |
| `internal/cli/` | Cobra CLI commands and hidden package lifecycle helpers. |
| `internal/contracts/` | Go-owned contracts and generated Web UI TypeScript contract source. |
| `internal/session/` | Local session bridge and Codex CLI adapter. |
| `internal/group/`, `internal/tools/`, `internal/routes/`, `internal/communities/`, `internal/processes/` | Higher-level graph enrichments and runtime views. |
| `avmatrix/` | npm package metadata, packaged skills, built `bin/` runtime artifacts, and generated `go-src` only during package fallback workflows. |
| `avmatrix-web/` | React/Vite Web UI. Runtime calls go through `avmatrix serve`. |
| `contracts/web-ui/` | Go-generated contract manifest for browser glue. |
| `avmatrix-launcher/` | Windows launcher, server wrapper, packaged Web assets, and backend bundle. |

## End-to-End Flow

1. **CLI or HTTP starts analyze**
   - CLI path: `cmd/avmatrix` -> `internal/cli` -> `internal/analyze`.
   - HTTP path: `avmatrix serve` -> `internal/httpapi` -> `internal/analyze`.

2. **Scan and parse**
   - `internal/scanner` selects files and language IDs.
   - `internal/parser` manages parser readiness, parser pool, and parse metrics.
   - `internal/providers/*` emit ScopeIR facts from language-specific tree-sitter nodes or fallback parsing.

3. **Build graph facts**
   - `internal/scopeir` provides the serialized fact boundary.
   - `internal/resolution` builds definition/import/reference indexes, resolves calls/accesses/uses/inheritance, and emits audited relationships.
   - `internal/routes`, `internal/tools`, `internal/orm`, `internal/mro`, `internal/communities`, and `internal/processes` enrich the graph.

4. **Persist**
   - `internal/lbugschema` defines table and relationship constants.
   - `internal/lbugload` writes graph data to CSV/load paths.
   - `internal/lbugruntime` opens/query LadybugDB and handles compatibility/runtime helpers.
   - Repo metadata and registry state are managed by `internal/repo`.

5. **Serve local interfaces**
   - CLI: `avmatrix query|context|impact|cypher|detect-changes|rename|...`.
   - MCP stdio: `avmatrix mcp`.
   - HTTP/Web: `avmatrix serve`, then `avmatrix-web` talks to loopback HTTP.
   - Launcher: `avmatrix-launcher` wraps the same backend and Web UI.

## Local HTTP Runtime

`avmatrix serve` binds to loopback by default and exposes the local backend. It
does not introduce an AVmatrix-hosted cloud service.

| Endpoint family | Purpose | Main implementation |
|-----------------|---------|---------------------|
| `/api/info`, `/api/heartbeat` | Backend liveness and metadata. | `internal/httpapi/info.go`, `heartbeat.go` |
| `/api/repos`, `/api/repo` | List, resolve, select, and remove indexed repos. | `internal/httpapi/repos.go`, `internal/repo` |
| `/api/graph` | Load or stream graph data for an explicit repo. | `internal/httpapi/graph.go`, `internal/graph`, `internal/lbugruntime` |
| `/api/query`, `/api/search`, `/api/file`, `/api/grep`, `/api/process*`, `/api/cluster*` | Code search, file access, graph views. | `internal/httpapi/query.go`, `search.go`, `file.go`, `grep.go`, `panels.go` |
| `/api/local/folder-picker` | Open an OS folder picker from the local backend. | `internal/httpapi/local_folder_picker.go` |
| `/api/analyze`, `/api/embed` | Background indexing and embedding jobs. | `internal/httpapi/analyze.go`, `embed.go`, `jobs.go` |
| `/api/mcp` | MCP-over-HTTP bridge for local clients. | `internal/httpapi/mcp.go`, `internal/mcp` |
| `/api/session/*` | Local Codex-style session bridge. | `internal/httpapi/session.go`, `internal/session` |

Repo context is explicit by repo name/path/session. Backend and Web code must
not reintroduce a mutable process-global active repo as the source of truth.

## Web UI Contract

The browser-side contract source is generated from Go:

```text
internal/contracts -> cmd/generate-web-contracts
  -> contracts/web-ui/web-ui-contract.json
  -> avmatrix-web/src/generated/avmatrix-contracts.ts
```

`avmatrix-web/` may contain TypeScript/React UI code and generated browser glue.
Backend, CLI, MCP, analyzer, persistence, and session authority live in Go.

## Package Lifecycle

The npm package is a Go runtime distribution wrapper:

| Command | Behavior |
|---------|----------|
| `npm run build` in `avmatrix/` | Runs `go run ../cmd/avmatrix package build-runtime` and writes `avmatrix/bin/avmatrix.exe`. |
| `prepack` | Builds the runtime and runs `avmatrix package prepare-go-source` to create generated `go-src` fallback source. |
| `postpack` | Runs `avmatrix package clean-go-source` to remove generated package source from the working tree. |
| `postinstall` | Rebuilds from repo or packaged `go-src` when Go is available, otherwise verifies the packaged binary for the current platform. |

Package lifecycle helpers live in `internal/cli/package_command.go` and
`internal/cli/package_runtime.go`; there are no package JS/CJS helper files.
`go-src/` is a generated package artifact for prepack/postinstall fallback
paths, not a normal source directory that must exist in a working checkout.

## Analyze Pipeline

The Go analyze pipeline is implemented across focused packages instead of a
TypeScript phase directory.

```text
scan -> structure/documents/cobol -> parse/providers
  -> routes/tools/orm -> resolution -> mro
  -> communities -> processes -> LadybugDB load
```

| Stage | Main Go packages | Output |
|-------|------------------|--------|
| Scan | `internal/scanner` | Selected files, language classification, skip metadata. |
| Structure/documents | `internal/structure`, `internal/documents` | File/folder/section nodes and containment. |
| COBOL/JCL | `internal/cobol` | COBOL program, paragraph, section, copybook, and JCL facts. |
| Parse/providers | `internal/parser`, `internal/providers/*` | ScopeIR definitions, imports, references, routes/tools/ORM facts. |
| Resolution | `internal/scopeir`, `internal/resolution` | Audited CALLS, ACCESSES, USES, INHERITS, IMPORTS, and compatibility edges. |
| MRO | `internal/mro`, `internal/resolution` | METHOD_OVERRIDES and METHOD_IMPLEMENTS style graph edges. |
| Communities/processes | `internal/communities`, `internal/processes` | Community nodes and execution-flow process nodes. |
| Persist | `internal/lbugschema`, `internal/lbugload`, `internal/lbugruntime` | LadybugDB tables, graph load, search/query support. |

### Call-Resolution DAG

Call resolution is still organized as a typed multi-stage flow inside the parse
and resolution path:

1. Provider extraction emits call/reference facts in ScopeIR.
2. Definition/import indexes are built from all parsed files.
3. Language-specific implicit receiver and dispatch behavior remains behind
   provider hooks and resolution helpers.
4. References are resolved to graph node identities with confidence/evidence.
5. Duplicate/legacy-compatible edges are merged with audit metadata.
6. Graph relationships are emitted with stable type labels and file hashes.

Shared code in `internal/analyze`, `internal/scopeir`, and
`internal/resolution` must stay language-neutral; language behavior belongs in
`internal/providers/<language>/`.

## Storage

```text
<repo>/.avmatrix/
  graph.json        JSON graph snapshot for fallback/runtime readers
  lbug              LadybugDB database
  meta.json         repoPath, lastCommit, indexedAt, stats
  settings.json     optional repo-local settings such as maxExecutionFlows

~/.avmatrix/
  registry.json     indexed repo registry for CLI/MCP/Web discovery
```

LadybugDB runtime side files such as WAL or lock files may appear while the
database is open or recovering, but they are transient implementation details
rather than required index outputs.

Launcher runtime state and logs are managed under `avmatrix-launcher/` and the
user temp directory. The launcher is optional; `avmatrix serve` remains the
direct backend entry point.

## MCP Tools

| Tool | Purpose |
|------|---------|
| `list_repos` | Discover indexed repos. |
| `query` | Hybrid text/vector search over indexed graph content. |
| `context` | Callers, callees, files, and processes for one symbol. |
| `impact` | Upstream/downstream blast radius with risk summary. |
| `detect_changes` | Map git diffs to affected symbols and processes. |
| `rename` | Graph-assisted multi-file rename with dry-run preview. |
| `cypher` | Ad hoc graph query support. |
| `api_impact`, `route_map`, `tool_map`, `shape_check` | HTTP/API/MCP contract analysis. |
| `group_*` | Cross-repo group search, contracts, sync, and status. |

## Where To Change What

| Concern | Start in |
|---------|----------|
| CLI commands/flags | `internal/cli/` |
| Package lifecycle | `internal/cli/package_command.go`, `internal/cli/package_runtime.go`, `avmatrix/package.json` |
| Analyze orchestration | `internal/analyze/` |
| File scanning/language selection | `internal/scanner/` |
| Parser readiness/pool | `internal/parser/` |
| Language extraction | `internal/providers/<language>/` |
| Scope facts and indexes | `internal/scopeir/` |
| Import/call/reference resolution | `internal/resolution/` |
| Graph schema/load/runtime | `internal/lbugschema/`, `internal/lbugload/`, `internal/lbugruntime/` |
| HTTP backend | `internal/httpapi/` |
| MCP server/tools/resources | `internal/mcp/` |
| Search and embeddings | `internal/embeddings/`, `internal/httpapi/search.go` |
| Session bridge | `internal/session/`, `internal/httpapi/session.go` |
| Web UI | `avmatrix-web/src/` |
| Generated Web contracts | `internal/contracts/`, `cmd/generate-web-contracts/`, `avmatrix-web/src/generated/` |
| Launcher | `avmatrix-launcher/src/main.go`, `avmatrix-launcher/server-wrapper/main.go`, `avmatrix-launcher/build.ps1` |

## Known Constraints

- Only one analyze writer should touch a repo-local `.avmatrix/lbug` database at
  a time.
- Embeddings are opt-in. Use `avmatrix analyze --embeddings` when preserving or
  refreshing vector data matters.
- Web graph loading must stay repo-scoped by explicit repo path/name.
- `avmatrix-web/` must remain a thin client over the local backend.
- The launcher must remain a convenience layer over the same backend semantics.
