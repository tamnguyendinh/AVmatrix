# Web UI Full Analyze Contract Re-review

Timestamp: 2026-05-08 18:10:08 UTC+7
Lane: supervisor
Scope: `docs/plans/2026-05-08-web-ui-full-analyze-contract-plan.md`
Basis: plan + current worktree source only. No SPEC, `ARCHITECTURE.md`, or other docs used as authority per user instruction.
Verdict: REJECT
Severity: HIGH
Owner: coder

## Summary

The implementation closes several requirements from the plan: Web/API analyze now forces full analyze, repo-card analyze captures the selected path, SSE completion can carry `repoPath`, and active frontend repo state is mostly path-based.

Approval is still blocked because the plan's selected-path chain is not fully closed: selected physical repo path must remain the routing identity through analyze, repo hydration, graph load, and failure handling. Current source still has paths where name/basename routing can re-enter after a path miss, and header re-analyze failures are effectively silent.

## Blocking Issues

### [HIGH] Absolute-path resolver retry drops the selected path and can fall back to a name-collided repo

File: `avmatrix/src/server/api.ts:296`

Issue: `resolveRepo` computes `requestedPath` when the request uses an absolute repo path, but the emergency retry path calls `resolveRepo(normalizedName, true, req, opts)` at `avmatrix/src/server/api.ts:366`. `normalizedName` comes from `normalizeRepoParam`, which is `path.basename(repoParam)` at `avmatrix/src/runtime/repo-resolver.ts:43`.

That means a first-pass miss for selected repo `F:\two\demo` can retry as only `demo`. On retry, `findRepoCandidate` can match by name at `avmatrix/src/runtime/repo-resolver.ts:122`, so a different registered repo such as `F:\one\demo` can be returned after the fallback. This violates the plan's hard contract that the selected `repoPath` must not be redirected by `repoName` or basename collision.

Fix: preserve the original absolute `repoName` or `requestedPath` through the retry, or resolve from the refreshed registry by `requestedPath` before any basename/name lookup. Add a regression test where the first registry read misses selected repo B, the refresh includes repo A and repo B with the same basename/name, and `/api/repo` or `/api/graph` resolves repo B by absolute path.

### [MEDIUM] `connectToServer` validates repo info but downloads graph using the original argument instead of the resolved repo path

File: `avmatrix-web/src/services/backend-client.ts:811`

Issue: `connectToServer` fetches repo info first, but then calls `fetchGraph(repoName, ...)` at `avmatrix-web/src/services/backend-client.ts:814`. The plan contract is `complete repoPath -> load graph repoPath`; once `/api/repo` returns `repoInfo.repoPath`, the graph request should use that canonical path, not the original argument, because the original argument may be a display name, old URL `project` value, runtime id, or otherwise ambiguous input.

Fix: after `fetchRepoInfo`, compute `const graphRepo = repoInfo.repoPath ?? repoInfo.path ?? repoName` and pass `graphRepo` to `fetchGraph`. Add a test that `fetchRepoInfo('demo')` returns `repoPath: 'F:\\two\\demo'` and the graph request uses `repo=F%3A%5Ctwo%5Cdemo`, not `repo=demo`.

### [MEDIUM] Header re-analyze failure path clears progress without a path-specific visible failure

File: `avmatrix-web/src/components/Header.tsx:305`

Issue: Header re-analyze selects `repoPath` at `avmatrix-web/src/components/Header.tsx:305`, but SSE failure only logs `console.error('Re-analyze failed:', errMsg)` at `avmatrix-web/src/components/Header.tsx:319`, and start failure only logs `console.error('Failed to start re-analysis:', err)` at `avmatrix-web/src/components/Header.tsx:326`. Both paths then clear `reanalyzing` / `reanalyzeProgress`, leaving the user on the previous graph with no visible error and no selected path in the diagnostic. This violates the plan requirement that analyze/load failures include the selected canonical `repoPath` and never silently fall back to stale or unrelated graph state.

Fix: surface a visible header/dropdown error or app progress error that includes `repoPath`, and include `repoPath` in console diagnostics. Add tests for SSE failure and startAnalyze rejection proving no graph reload occurs and the selected path is shown/logged.

## Source-Level Clearance Notes

- Backend analyze API: `buildAnalyzeWorkerOptions` now forces `{ force: true }` at `avmatrix/src/server/api.ts:67`, worker start sends that at `avmatrix/src/server/api.ts:1185`, and job creation uses the resolved local path at `avmatrix/src/server/api.ts:1032` and `avmatrix/src/server/api.ts:1034`. This part matches the full-analyze requirement.
- Backend SSE payload: `buildAnalyzeCompleteEventPayload` includes `repoPath` at `avmatrix/src/server/api.ts:83`, and SSE terminal events use it at `avmatrix/src/server/api.ts:157` and `avmatrix/src/server/api.ts:182`. This part is directionally correct, subject to the resolver issue above.
- Backend graph route: `/api/graph` still resolves through `resolveRepo(requestedRepo(req))` at `avmatrix/src/server/api.ts:520`; the path-first resolver helps, but the retry bug above keeps this group unapproved.
- Web repo-card/path analyze components: `RepoLanding` passes the clicked repo object at `avmatrix-web/src/components/RepoLanding.tsx:169`, `DropZone` starts analyze with selected `repoPath` at `avmatrix-web/src/components/DropZone.tsx:231` and `avmatrix-web/src/components/DropZone.tsx:245`, then loads `repoPath` at `avmatrix-web/src/components/DropZone.tsx:254`. `RepoAnalyzer` completes with `repoPath` at `avmatrix-web/src/components/RepoAnalyzer.tsx:175`. These are acceptable for the primary success path.
- Web app state/follow-up calls: `App` stores the loaded path as current repo at `avmatrix-web/src/App.tsx:71` and `avmatrix-web/src/App.tsx:78`; `useAppState` keeps `repoRef.current` on the loaded path at `avmatrix-web/src/hooks/useAppState.local-runtime.tsx:802`, and follow-up query/search calls use `repoRef.current` at `avmatrix-web/src/hooks/useAppState.local-runtime.tsx:640` and `avmatrix-web/src/hooks/useAppState.local-runtime.tsx:712`. This is acceptable if the graph loaded was the correct path.
- Web repo-list merge: `includeRepoInList` now prefers path identity at `avmatrix-web/src/services/repo-list.ts:8` and `avmatrix-web/src/services/repo-list.ts:10`, so same-name repos are no longer replaced by name when path exists.
- Header UI group: success completion prefers `data.repoPath ?? repoPath` at `avmatrix-web/src/components/Header.tsx:316`, but the failure handling at `avmatrix-web/src/components/Header.tsx:319` and `avmatrix-web/src/components/Header.tsx:326` remains blocking.

## Verification Run

Passed:

```powershell
cd avmatrix
npx vitest run test/unit/analyze-api.test.ts test/unit/repo-resolver.test.ts
```

Result: 2 files passed, 14 tests passed.

Passed:

```powershell
cd avmatrix-web
npm test -- test/unit/DropZone.full-analyze-flow.test.tsx test/unit/Header.reanalyze-flow.test.tsx test/unit/RepoAnalyzer.local-only.test.tsx test/unit/repo-list.test.ts test/unit/server-connection.test.ts
```

Result: 5 files passed, 30 tests passed.

Passed:

```powershell
cd avmatrix-web
npx tsc --noEmit
```

Passed:

```powershell
cd avmatrix
npx tsc --noEmit
```

## Required Re-approval Evidence

- Same-head test or integration evidence that absolute path retry never resolves by basename/name when repo paths collide.
- `connectToServer` graph download uses the resolved canonical `repoPath` from `/api/repo`.
- Header re-analyze start/SSE failure shows or records a path-specific failure and does not silently leave the user believing the selected repo was refreshed.
- Existing targeted backend/web tests and TypeScript checks still pass.
