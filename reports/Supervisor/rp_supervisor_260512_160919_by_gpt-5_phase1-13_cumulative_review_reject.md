# Supervisor Report: Go Conversion Cumulative Review Through Phase 13

Timestamp: 2026-05-12 16:09:19 UTC+7
Lane: supervisor
Scope: `docs/plans/2026-05-08-avmatrix-go-typescript-node-to-go-conversion-plan.md`, cumulative implementation through Phase 13 only.
Reviewed head: `41e6753` (`Fix MCP phase reject findings`)
Mode: Mode 2 current worktree / post-coder re-review.
Verdict: REJECT
Severity: HIGH
Owner: coder

## Scope Boundary

This is not a final-plan review. Coder has implemented through Phase 13, so Phase 14+ items are read only to understand future direction and are not rejection criteria.

`Docs/SPEC/*` and `Docs/execution/*` do not exist in this repo. I used the assigned plan only to identify Phase 1-13 scope, then verified current source and runtime behavior directly. Existing reports were treated as pointers, not approval evidence.

Open historical report check:
- `reports/problem/pb_supervisor_260508_135457_by_gpt-5_web_cli_analyze_contract_drift.md` is closed by `reports/Supervisor/rp_supervisor_260508_183325_by_gpt-5_web_ui_full_analyze_contract_rereview_approve.md` for that same web analyze scope.
- No same-scope Phase 13 MCP approval report exists after the current `41e6753` fix batch.

## Critical Issues

None found.

## High Issues

### [HIGH] MCP `cypher` returns false-positive rows in the default Go/launcher build

File: `internal/mcp/tools.go:312`

Issue: `cypherTool` validates read-only syntax and calls `runCypherRead`. In default builds, `lbugnative.OpenReadRunner` returns `ErrUnavailable` (`internal/lbugnative/runner_default.go:9`), and the packaged launcher build compiles without the `ladybugdb` tag (`avmatrix-launcher/build.ps1:72`). That makes default MCP `cypher` fall back to `runMCPGraphQuery` (`internal/mcp/tools.go:328`), which pattern-matches only broad query fragments (`internal/mcp/tools.go:411`) and ignores actual Cypher predicates/projections for relationship queries (`internal/mcp/tools.go:421`, `internal/mcp/tools.go:498`).

Independent runtime repro on a temporary indexed repo:

```text
Query:
MATCH (a)-[:CodeRelation {type: 'CALLS'}]->(b:Function {name: "doesNotExist"})
RETURN a.name, a.filePath

Expected:
row_count = 0

Observed from `go run ./cmd/avmatrix mcp`:
row_count = 1
row: main -> helper
```

Root cause: the fallback adapter treats any query containing `CodeRelation {type: 'CALLS'}` as "return CALLS edges", not as a filtered Cypher query. This violates the Phase 13 MCP `cypher` runtime contract because a read-only query can succeed with wrong data instead of using the real read runner or failing closed.

Fix:
- Default MCP `cypher` must not return approximate graph-adapter rows for arbitrary Cypher.
- Either wire a real read runner in the default launcher/runtime path, or fail closed with a clear `native LadybugDB read runner unavailable` error for unsupported builds.
- If keeping a fallback, restrict it to explicitly supported internal query shapes and reject any query with unsupported `WHERE`, node-property predicates, `RETURN` projection, `ORDER BY`, or multi-pattern semantics.
- Add a regression where a `CALLS` query filtered to a nonexistent target returns zero rows, not every `CALLS` relationship.

### [HIGH] MCP `rename` cannot safely rename same-file references and can generate duplicate edits on the definition line

File: `internal/mcp/rename.go:90`

Issue: `collectRenameChanges` adds the target definition by exact `startLine` (`internal/mcp/rename.go:92`) but incoming graph references only provide the caller file, then call `addRenameFirstMatchingLineEdit` (`internal/mcp/rename.go:115`). That helper scans from the top of the file and edits the first line containing `oldName` (`internal/mcp/rename.go:137`), not the actual reference range/line. For a same-file caller, the first matching line is commonly the definition itself, so dry-run reports duplicate definition edits and misses the call-site. Applying the result fails with a line mismatch (`internal/mcp/rename.go:216`).

Independent runtime repro on a temporary indexed repo:

```ts
function helper() {
  return 1;
}

function main() {
  const helperLabel = "helper";
  return helper();
}
```

Observed from `go run ./cmd/avmatrix mcp` with `rename(helper -> renamedHelper, dry_run=true)`:

```json
{
  "changes": [{
    "file_path": "src/app.ts",
    "edits": [
      {"line": 1, "old_text": "function helper() {", "confidence": "graph"},
      {"line": 1, "old_text": "function helper() {", "confidence": "graph"}
    ]
  }],
  "total_edits": 2,
  "text_search_edits": 0
}
```

Observed with `dry_run=false`: `rename edit mismatch at src/app.ts:1`.

Root cause: the graph edge points from caller symbol to target symbol, but `rename` does not use reference evidence/ranges and does not perform fallback text search. It therefore has no reliable call-site location and silently maps caller files to the first textual occurrence.

Fix:
- Use the resolved reference range/line from graph evidence or persist enough reference-location metadata for `rename`.
- Deduplicate edits by `file_path + line + oldLine`.
- For references without precise graph location, either add `text_search` edits for all word-boundary matches or report them as ambiguous review-required edits, not `graph` confidence.
- Add regression coverage for same-file definition + caller where the actual call line is not the first occurrence of the old name.

## Medium Issues

None found beyond the two HIGH runtime blockers above.

## Suggestions

- Align the Go MCP `rename` discovery text with actual behavior after the fix. The TypeScript baseline describes graph plus text search (`avmatrix/src/mcp/tools.ts:208`), while the Go tool currently reports `text_search_edits: 0` unconditionally (`internal/mcp/rename.go:84`).
- Keep `go test ./...` out of the Phase 1-13 gate until the plan's later cutover isolation/removal of legacy fixture packages is done. The valid current Go gate is `go test ./cmd/... ./internal/...`.

## Source-Level Clearance Notes By Production File Group

- CLI entry and command surface: cleared for Phase 1-13. The root command exposes `serve`, `analyze`, `mcp`, `status`, `wiki`, and `wiki-mode` at `internal/cli/command.go:59`; stdio MCP is wired at `internal/cli/command.go:72`; analyze resolves local paths and writes graph snapshots at `internal/cli/command.go:136`.
- Repo path and registry policy: cleared. Remote URLs, UNC paths, relative paths, and non-directories are rejected at `internal/repo/path_policy.go:14`; registry storage paths are normalized at `internal/repo/registry.go:57`; absolute repo lookup is path-first at `internal/repo/resolve.go:18`.
- Scanner and ignore/selection: cleared. Repository walk applies `.gitignore`, include/exclude selection, file-size guard, language detection, hashing, and deterministic sorting at `internal/scanner/scan.go:42`, `internal/scanner/scan.go:85`, `internal/scanner/selection.go:10`, and `internal/scanner/language.go:105`.
- Parser and ScopeIR: cleared for Phase 1-13. Tree-sitter registry covers JS/TS/TSX and Go grammar availability at `internal/parser/registry.go:28`; ScopeIR has deterministic normalization/marshal behavior at `internal/scopeir/ir.go:31`; TS/JS extraction emits definitions/imports/type bindings/references at `internal/providers/tsjs/extract.go:20`.
- Resolution and graph emission: cleared for the reviewed Phase 8-9 family. Cross-file binding finalizes before resolution at `internal/resolution/resolve.go:22`; calls/accesses/type annotations emit relationship evidence and file hashes at `internal/resolution/resolve.go:96`, `internal/resolution/resolve.go:132`, and `internal/resolution/resolve.go:164`.
- Analyze pipeline: cleared for Phase 11 orchestration. The Go pipeline runs scan, structure/docs/cobol, parse, route/tool/ORM, cross-file, resolution, MRO, communities, processes, DB load, embeddings, and graph snapshot at `internal/analyze/analyze.go:198`, `internal/analyze/analyze.go:244`, `internal/analyze/analyze.go:289`, `internal/analyze/analyze.go:300`, `internal/analyze/analyze.go:346`, and `internal/analyze/analyze.go:388`.
- LadybugDB schema/load/runtime: cleared except as it contributes to the `cypher` blocker above. COPY load is primary at `internal/lbugload/load.go:27` and `internal/lbugload/load.go:39`; fallback is diagnostic after COPY failure/schema gap at `internal/lbugload/load.go:45`; read query guard blocks writes at `internal/lbugruntime/query_guard.go:49`; default no-tag runner is explicitly unavailable at `internal/lbugnative/runner_default.go:9`.
- HTTP API: cleared for Phase 12/13 routes. Handler registers analyze, graph, search/embed, and MCP endpoints at `internal/httpapi/server.go:47`; analyze rejects remote URLs and resolves local path at `internal/httpapi/analyze.go:63`; graph loads/streams `graph.json` at `internal/httpapi/graph.go:32`; HTTP MCP session creation requires initialize before no-session use at `internal/httpapi/mcp.go:153`.
- MCP resources/prompts/context/impact/route tools: cleared except for `cypher` and `rename`. Tool dispatch includes all Phase 13 tools at `internal/mcp/server.go:244`; resources/templates/prompts are exposed at `internal/mcp/resources.go:74`, `internal/mcp/resources.go:91`, and `internal/mcp/prompts.go:30`; context expands class/interface incoming refs at `internal/mcp/context.go:61`; impact validates direction/relation types and runs bounded BFS at `internal/mcp/impact.go:101`; route/API shape tools read Route nodes and consumer metadata at `internal/mcp/route_shape_impact.go:68`.
- Group tools and contract registry: cleared for the implemented Phase 13 graph slice. MCP group tools dispatch to group core at `internal/mcp/group_tools.go:47`; config parses repos/links/detect/matching/packages and validates manifest link type/role/contract at `internal/group/config.go:9`; sync resolves registered members, extracts HTTP contracts only when enabled, exact-matches, and writes `contracts.json` at `internal/group/sync.go:10`.
- Launcher build gate: cleared for Go-aware packaging, with one relevant caveat for `cypher`. The launcher build compiles the Go backend CLI into `server-bundle/avmatrix.exe` at `avmatrix-launcher/build.ps1:72`, but that build line does not set `-tags ladybugdb`, so default MCP `cypher` reaches the fallback bug described above.

## Verification Run

Passed:

```powershell
go test ./internal/mcp ./internal/httpapi ./internal/group -count=1
```

Result: all 3 packages passed.

Passed:

```powershell
go test ./cmd/... ./internal/... -count=1
```

Result: all command/internal packages passed.

Passed:

```powershell
cd avmatrix-web
npm run build
```

Result: TypeScript/Vite build passed. Vite emitted existing chunk-size/dynamic-import warnings only.

Independent runtime repros:

```powershell
go run ./cmd/avmatrix analyze .tmp\rename-line-repro --force --skip-agents-md --no-stats
go run ./cmd/avmatrix mcp
```

Observed the `cypher` false-positive row and `rename` duplicate-line dry-run/apply failure described above.

## Required Fix List For Resubmission

1. Close the Phase 13 MCP `cypher` invariant: default stdio/HTTP MCP must either execute real read-only Cypher or fail closed. It must not return broad graph-adapter approximations for unsupported Cypher predicates/projections.
2. Close the Phase 13 MCP `rename` invariant: graph-confidence edits must use precise reference locations, avoid duplicate edits, and handle same-file caller/definition cases without apply-time mismatch.
3. Add same-head regression tests for:
   - `cypher` query with nonexistent target predicate returns zero rows or a closed/unavailable error, never unrelated rows.
   - same-file `rename` where definition, unrelated text, and call-site all contain the old name.
4. Re-run the same-head evidence bundle:
   - `go test ./internal/mcp ./internal/httpapi ./internal/group -count=1`
   - `go test ./cmd/... ./internal/... -count=1`
   - Phase 13 MCP stdio and HTTP smoke for `cypher`, `rename`, `detect_changes`, `context`, and group tools.

Residual same-family unverified surfaces: `cypher` and `rename` remain open; do not approve Phase 13 until both are closed on the same head.

## Overall Coder Evaluation

Coding style is generally consistent with the Go repo style: small packages, clear structs, explicit error returns, and tests around most new MCP/HTTP/group surfaces. The current failure is not broad architectural misunderstanding of the whole plan; it is a Phase 13 MCP runtime-contract gap where fixture/snapshot coverage checks shape and happy paths but misses incorrect default runtime behavior. Rule compliance is improved from earlier rounds, but zero-trust runtime checks still find blocking issues in code paths that users can call today.
