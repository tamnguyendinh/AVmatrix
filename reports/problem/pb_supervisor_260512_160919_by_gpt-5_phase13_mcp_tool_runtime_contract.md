# Phase 13 MCP Tool Runtime Contract Blocker

Timestamp: 2026-05-12 16:09:19 UTC+7
Lane: supervisor
Scope: Phase 13 MCP tools in `docs/plans/2026-05-08-avmatrix-go-typescript-node-to-go-conversion-plan.md`
Reviewed head: `41e6753`
Verdict: BLOCKER
Severity: HIGH
Owner: coder

## Incident Scope

The current Go MCP implementation through Phase 13 exposes `cypher` and `rename` as completed tools, but direct runtime checks on current head show both can return incorrect or unusable results in default stdio MCP execution.

Phase 14+ future-provider work is not part of this blocker.

## Verified Evidence

### `cypher` false positive

Default build path:
- `internal/lbugnative/runner_default.go:9` returns `ErrUnavailable`.
- `internal/mcp/tools.go:312` falls back from read runner to graph snapshot adapter at `internal/mcp/tools.go:328`.
- `internal/mcp/tools.go:411` pattern-matches query fragments and `internal/mcp/tools.go:498` returns every matching relationship type without honoring node predicates.

Runtime repro:

```text
MATCH (a)-[:CodeRelation {type: 'CALLS'}]->(b:Function {name: "doesNotExist"})
RETURN a.name, a.filePath
```

Observed through `go run ./cmd/avmatrix mcp`: `row_count = 1`, returning `main -> helper`.

Expected: zero rows or a closed/unavailable error.

### `rename` duplicate definition edit and apply failure

Source path:
- `internal/mcp/rename.go:90` collects edits.
- `internal/mcp/rename.go:92` adds the definition line.
- `internal/mcp/rename.go:115` maps every incoming reference to the first matching line in the caller file.
- `internal/mcp/rename.go:137` scans from top-of-file, not from graph evidence/range.
- `internal/mcp/rename.go:216` rejects the second duplicate edit during apply.

Runtime repro:

```ts
function helper() {
  return 1;
}

function main() {
  const helperLabel = "helper";
  return helper();
}
```

Observed through `go run ./cmd/avmatrix mcp`:
- dry-run returns two `graph` edits for `src/app.ts:1`.
- dry-run omits the actual `return helper();` call-site.
- `dry_run=false` returns `rename edit mismatch at src/app.ts:1`.

## Root Cause

The shared invariant family is Phase 13 MCP tool runtime correctness:
- `cypher` must not claim successful read-query execution while ignoring predicates/projections.
- `rename` must not claim graph-confidence edits without precise reference locations.

The current tests cover surface shape and narrow happy paths, but not default-build `cypher` semantic correctness or same-file `rename` reference targeting.

## Required Next Direction

Return to coder for implementation fix inside Phase 13 MCP scope.

Required closure evidence on the same head:
- `cypher` uses a real read runner in default callable runtime or fails closed for unsupported builds/query shapes.
- `rename` uses precise reference locations, deduplicates edits, and handles same-file references.
- Regression tests cover the two repros above.
- Re-run `go test ./internal/mcp ./internal/httpapi ./internal/group -count=1`.
- Re-run `go test ./cmd/... ./internal/... -count=1`.
- Provide stdio and HTTP MCP smoke evidence for the fixed `cypher` and `rename` paths.
