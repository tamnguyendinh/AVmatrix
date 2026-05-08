# Web UI / CLI Analyze Contract Drift

Timestamp: 2026-05-08 13:54:57 UTC+7
Lane: supervisor
Scope: Web UI analyze semantics vs CLI analyze semantics
Verdict: BLOCKER
Severity: HIGH
Owner: coder

## Summary

The Web UI currently does not consistently preserve the product contract that an `analyze` action must mean a full analyze, equivalent to the CLI full analyze path. The measured `~6-10s` Web UI path is a graph-load path, while the measured `~91-99s` CLI benchmark path is a full forced rebuild path.

If the intended product rule is "running analyze must always perform full analyze, whether triggered from CLI or Web UI", current behavior is incorrect.

## Expected Contract

- CLI is the execution core.
- Web UI is only a visual/control layer over the same runtime behavior.
- Any user-facing action named or understood as `Analyze` / `Re-analyze` must run the same full analyze semantics as CLI full analyze.
- Opening an existing graph must be visibly distinct from analyzing a repository.

## Observed Evidence

### CLI full rebuild on GitNexus-main

Command:

```powershell
node avmatrix\dist\cli\index.js analyze E:\Lap_trinh\GitNexus-main --force --skip-git --skip-agents-md --no-stats --verbose
```

Observed result:

- CLI reported: `Repository indexed successfully (97.6s)`
- External stopwatch: `99.13s`
- Breakdown: `parse 48.6s`, `lbugLoad 26.1s`, `fts 16.3s`, `resolution 2.4s`

This matches the earlier benchmark class (`~91s`) as a full rebuild measurement.

### Web/backend graph load on GitNexus-main

Endpoint:

```text
GET http://127.0.0.1:4755/api/graph?repo=GitNexus-main&stream=true
```

Observed result:

- Before the fresh rebuild: `10.6s`
- After the fresh rebuild: `6.0s`
- Response: `application/x-ndjson`, about `16.97 MB`

This endpoint only loads/streams an existing graph from LadybugDB. It does not run analyze.

### Web/API analyze non-force on an up-to-date git repo

Endpoint:

```text
POST http://127.0.0.1:4755/api/analyze
body: { "path": "F:\\Website" }
```

Observed result:

- Completed in `1.48s`
- Repo was already up to date (`meta.lastCommit` matched git `HEAD`)
- Follow-up graph load took `6.11s`

This proves Web/API analyze can complete quickly by taking the up-to-date shortcut when `force` is not sent.

## Source Evidence

### Full analyze path includes pipeline, DB load, FTS, and total timing

- `avmatrix/src/core/run-analyze.ts:217` calls `runPipelineFromRepo(...)`.
- `avmatrix/src/core/run-analyze.ts:250` loads the graph into LadybugDB via `loadGraphToLbug(...)`.
- `avmatrix/src/core/run-analyze.ts:263` begins FTS index creation.
- `avmatrix/src/core/run-analyze.ts:464` records `totalWallMs: metrics.elapsedMs()`.

### Up-to-date shortcut bypasses full analyze

- `avmatrix/src/core/run-analyze.ts:164` returns early when `existingMeta && !options.force && existingMeta.lastCommit === currentCommit`.
- `avmatrix/src/cli/analyze.ts:255` prints `Already up to date`.
- `avmatrix/src/cli/analyze.ts:257` explicitly says benchmark JSON is not written unless rerun with `--force`.

### Web Analyze form does not force full analyze

- `avmatrix-web/src/components/RepoAnalyzer.tsx:157` sends `const request = { path: localPath.trim() };`.
- `avmatrix-web/src/components/RepoAnalyzer.tsx:158` calls `startAnalyze(request)`.
- No `force: true` is sent from this form.

### Header Re-analyze does force

- `avmatrix-web/src/components/Header.tsx:304` calls `startAnalyze({ ... })`.
- `avmatrix-web/src/components/Header.tsx:306` sends `force: true`.

This means Web UI has inconsistent analyze semantics depending on which UI path is used.

### Repo click / connect path only loads graph

- `avmatrix-web/src/components/DropZone.tsx:178` defines `connectToRepo(...)`.
- `avmatrix-web/src/components/DropZone.tsx:188` calls `connectToServer(...)`.
- `avmatrix-web/src/services/backend-client.ts:806` calls `fetchGraph(...)`.
- `avmatrix-web/src/services/backend-client.ts:441` builds `/api/graph?...stream=true`.
- `avmatrix/src/server/api.ts:490` serves `GET /api/graph`.
- `avmatrix/src/runtime/repo-runtime/graph-read-service.ts:217` reads graph via `executeRepoReadQuery(...)`.

This path is graph load, not analyze.

### Backend only forces when client sends force

- `avmatrix/src/server/api.ts:1157` sends worker options as `{ force: !!force, embeddings: !!embeddings }`.

Therefore the backend allows non-force analyze from Web clients.

## Root Cause

The Web UI currently mixes three separate operations under user flows that are easy to interpret as "analyze":

1. Open/load an already indexed graph.
2. Analyze without `force`, allowing an up-to-date shortcut.
3. Re-analyze with `force: true`.

The CLI benchmark and CLI full analyze path measure operation 3. The Web UI timing observed around `~6-10s` measures operation 1, and Web form analyze may measure operation 2.

This is a runtime contract drift between the visual interface and the CLI semantics expected by the product.

## Impact

- Users can believe the Web UI re-analyzed source code when it only loaded an existing graph.
- Performance numbers can be misreported by comparing graph-load time to full-analyze time.
- Stale graph risk increases if UI language implies a fresh analyze but no full rebuild happened.
- The Web UI stops being a faithful visual layer over CLI analyze behavior.

## Required Fix Direction

Coder should align the product/runtime contract before this can be considered closed:

1. Define `Open graph` / `Load graph` separately from `Analyze`.
2. Any UI action named or understood as `Analyze` must send full-analyze semantics, likely `force: true`.
3. Backend `/api/analyze` should make the force mode explicit in its contract. If product policy is always-full analyze, default backend analyze should be full analyze or reject ambiguous requests.
4. UI progress text should distinguish:
   - checking existing index,
   - full analyzing,
   - loading existing graph,
   - streaming graph to browser.
5. Benchmark or diagnostic output should never treat `/api/graph` load time as analyze time.

## Required Closure Evidence

Before approval, provide same-head evidence for the corrected contract:

- CLI full analyze and Web-triggered analyze run on the same repo and both perform a full rebuild.
- Web action named `Analyze` cannot early-return as already up to date unless the UI explicitly labels it as a cache/check operation and the product owner approves that behavior.
- Repo-card click is labeled/implemented as opening an existing graph, not as analyze.
- Automated or manual evidence captures the request payload and backend worker options for Web analyze.
- Timing evidence separates full analyze time from graph load/render time.

## Notes

No source code was changed as part of this investigation. Runtime measurement did regenerate the `.avmatrix` index for `E:\Lap_trinh\GitNexus-main` during the CLI full-rebuild measurement.
