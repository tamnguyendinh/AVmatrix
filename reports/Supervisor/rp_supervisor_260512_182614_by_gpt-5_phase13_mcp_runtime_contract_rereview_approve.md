# Supervisor Report: Phase 13 MCP Runtime Contract Re-review

## Verdict

PASS.

Scope reviewed: same-head re-review of the Phase 13 MCP runtime-contract blockers raised in `reports/problem/pb_supervisor_260512_160919_by_gpt-5_phase13_mcp_tool_runtime_contract.md`, within the cumulative Phase 1-13 Go conversion scope. Phase 14+ plan items were not used as rejection criteria.

Reviewed head: `6e3eeed Fix MCP cypher and rename reject blockers`

Previous blocker closure status: closed.

Residual same-family unverified surfaces: none.

## Zero-trust Review Basis

I did not accept coder claims, plan wording, report wording, or test names as proof. I inspected the source diff from `18ebce2..6e3eeed`, read the touched runtime files before running verification, refreshed the AVmatrix graph, used AVmatrix context for the affected MCP symbols, reproduced the prior runtime failures through stdio and HTTP MCP paths, then ran targeted and broad Go verification.

Untracked file left untouched: `coder.md`.

## Critical Issues

None.

## High Issues

None.

## Medium Issues

None.

## Source-level Closure Evidence

### MCP cypher runtime contract

Clearance: the previous fail-open false-positive path is closed.

- `internal/mcp/tools.go:311` keeps native read-runner execution first when available.
- `internal/mcp/tools.go:327` falls back to the Go graph snapshot only when the native runner is unavailable.
- `internal/mcp/tools.go:410` now routes the fallback through a narrow query adapter.
- `internal/mcp/tools.go:414` only serves `CALLS` relationship rows when both `from.id IN` and `to.id IN` constraints are present.
- `internal/mcp/tools.go:417` rejects node-label queries that include a `WHERE` clause.
- `internal/mcp/tools.go:423` returns `errUnsupportedMCPGraphQuery` for unsupported fallback queries instead of broad relationship rows.
- `internal/mcp/server.go:185` converts `tools/call` errors into JSON-RPC `-32602`, so unsupported fallback queries fail closed at the MCP boundary.

Runtime repro on an isolated fixture:

```text
cypher query:
MATCH (a)-[:CodeRelation {type: 'CALLS'}]->(b:Function {name: "doesNotExist"}) RETURN a.name, a.filePath

stdio MCP result:
error code -32602, message "unsupported graph query in Go MCP graph adapter"

HTTP MCP result:
error code -32602, message "unsupported graph query in Go MCP graph adapter"
```

This directly closes the prior bug where the same unsupported predicate returned unrelated rows.

### MCP rename runtime contract

Clearance: the previous same-file call-site rename mismatch is closed.

- `internal/resolution/emit.go:57` emits reference line and column into semantic relationship IDs.
- `internal/mcp/rename.go:95` adds the target definition edit from graph definition location.
- `internal/mcp/rename.go:116` reads the relationship reference line when present.
- `internal/mcp/rename.go:117` adds the call-site edit as graph confidence on the exact reference line.
- `internal/mcp/rename.go:120` falls back to text search only when the relationship has no reference line.
- `internal/mcp/rename.go:130` bounds line-targeted edits to a valid file line.
- `internal/mcp/rename.go:136` requires the old symbol to appear as a word on that line before editing.
- `internal/mcp/rename.go:142` scans all matching fallback lines instead of only the first match.
- `internal/mcp/rename.go:154` deduplicates edit entries by line and old-line content.
- `internal/mcp/rename.go:177` parses the relationship ID to recover the encoded reference line.
- `internal/mcp/rename.go:239` still keeps the apply-time mismatch guard, so stale or wrong edits fail rather than silently corrupting files.

Runtime repro on an isolated fixture:

```text
Before:
function helper() { return 1; }
function main() {
  const helperLabel = "helper";
  return helper();
}

rename helper -> renamedHelper:
dry run: graph=2, text=0, total=2, lines=[1,7]
apply: applied=true, total=2

After:
function renamedHelper()
return renamedHelper();
const helperLabel = "helper";
```

This directly closes the prior bug where the definition and same-file call reference collapsed onto the same line and apply failed with `rename edit mismatch`.

### Regression coverage

Clearance: targeted regression coverage exists for both prior blockers.

- `internal/mcp/server_test.go:451` covers same-file rename reference lines and verifies graph edit counts.
- `internal/mcp/server_test.go:487` checks the dry-run payload is exactly two graph edits and zero text-search edits.
- `internal/mcp/server_test.go:502` checks the apply path succeeds.
- `internal/mcp/server_test.go:736` covers unsupported cypher relationship predicates.
- `internal/mcp/server_test.go:775` asserts the fallback returns JSON-RPC `-32602` with an unsupported-query message.

Tests were treated as regression evidence only after source and runtime inspection.

## Verification Commands

Commands run after source review:

```text
avmatrix analyze --force --skip-agents-md --no-stats
go test ./internal/mcp ./internal/httpapi ./internal/group -count=1
go test ./cmd/... ./internal/... -count=1
powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1
```

Results:

- AVmatrix graph refresh completed successfully: 25,488 nodes, 44,615 edges, 614 clusters, 658 flows.
- Targeted Go tests passed for `internal/mcp`, `internal/httpapi`, and `internal/group`.
- Broad Go command/internal tests passed.
- Launcher build passed, including `avmatrix-web` Vite build. Vite emitted existing chunk-size and mixed static/dynamic import warnings only.
- Stdio MCP smoke passed for both cypher fail-close and rename apply.
- HTTP MCP smoke passed for both cypher fail-close and rename apply.

## Overall Coder Evaluation

- Coding style: small, direct runtime changes in the affected MCP files.
- Code cleanliness: the broad fallback relationship adapter was removed instead of patched around, reducing fail-open behavior.
- Rule compliance: the fix stayed inside the approved Phase 13 MCP runtime-contract scope.
- Logic quality: source changes preserve native read-runner behavior, make fallback support explicit, and keep apply-time rename mismatch protection.
- Best practices: regression tests cover the exact two prior runtime failures without turning Phase 14+ incomplete work into review criteria.
- Current style: conservative Go implementation with explicit guards, narrow fallback behavior, and fixture-based regression tests.

## Approval Conclusion

The two previously verified Phase 13 MCP runtime blockers are closed on current head. The same-family sibling surfaces for stdio MCP, HTTP MCP, cypher fallback, and rename apply were checked on the same head. No CRITICAL, HIGH, or MEDIUM issue remains in this reviewed scope.
