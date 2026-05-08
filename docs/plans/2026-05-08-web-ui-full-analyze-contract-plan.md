# Web UI Full Analyze Contract Plan

Date: 2026-05-08

Status: Complete after path-contract fixes, tests, full build, and browser validation.

## Goal

Make the Web UI behave as a visual control surface for the same full analyze behavior users get from the CLI full analyze path. The Web UI should not introduce a separate analyze meaning, shortcut, mode, or cached-graph interpretation for actions that users understand as analyzing a repository.

The user-facing contract is:

- Clicking a repository card runs a full analyze for that repository, then loads the newly generated graph.
- Analyzing by local path runs a full analyze, then loads the newly generated graph.
- Re-analyze runs a full analyze, then loads the newly generated graph.

Header repository dropdown selection is a separate "switch/open existing graph" action if it remains fast. It is not required to run full analyze and must not be labeled or treated as analyze. It still must open the selected repository, not a default/current/name-collided repository.

The stronger technical contract is:

```text
selected repoPath
  -> analyze repoPath
  -> complete repoPath
  -> load graph repoPath
  -> render graph for the same repoPath
```

`repoName` is display metadata. It must not decide post-analyze graph loading, because name-only routing can drift from the selected repository path or become ambiguous when multiple repos share a basename/name.

The implementation goal is intentionally narrow:

```text
Web button for repo B
  = avmatrix analyze "<repo B path>" from the CLI contract
  -> then open graph from "<repo B path>"
```

Do not turn this into a broad repository-identity refactor. The repo list is only a UI picker. Once a user selects a repo, the selected path is the only source of truth for that analyze/load action.

## Non-Goals

- Do not change CLI analyze semantics.
- Do not change the core analyze pipeline.
- Do not add new analyze modes such as `full` versus `ifChanged`.
- Do not add benchmark work for this fix.
- Do not compare graph-load time with analyze time.
- Do not make cached graph loading the default repo-card behavior.
- Do not refactor global repo identity/routing beyond what is required to preserve the selected path through the Web UI action.
- Do not change CLI analyze semantics or make CLI depend on repo list behavior.
- Do not use a repo list lookup after analyze completion to decide which repo to load.

## Problem

The current Web UI has multiple paths that can be interpreted as analyze:

- Repo-card click can connect to an already indexed repository and load the stored graph.
- Analyze by path can call `/api/analyze` without forcing a rebuild, allowing the up-to-date shortcut.
- Header re-analyze currently forces a rebuild.

That creates contract drift. The CLI full analyze path rebuilds the graph, while some Web UI paths can return quickly because they only load an existing graph or take the non-force shortcut. This makes the UI easy to misread and can show stale graph data after source code changes.

Re-review finding:

- The first implementation fixed only the narrow ordering rule: repo-card click starts analyze before graph loading.
- It did not preserve the selected path through the whole flow.
- Landing repo-card analyze starts from `repo.path`, but the completion path can fall back to `repoName`.
- The backend SSE complete payload currently carries `repoName`, not the analyzed `repoPath`.
- Some graph-load calls use `awaitAnalysis: true`; the landing post-analyze graph load does not.
- The existing tests mostly prove call order with mocks. They do not prove that the repo being analyzed and the repo being loaded are the same physical path.
- Therefore the plan was marked complete too early.
- Browser/static-asset cache and already-running local servers can hide whether the newly built UI is the code being tested.
- Even if graph rendering uses the right repo, follow-up repo-scoped calls can still drift if active repo state is updated by name instead of the loaded path.

Contract review finding:

- CLI analyze at `0a87f9ed6c025d382c571a99ad90767dd2f788f3` was correct: analyze runs the current repo or the explicit path passed by the user.
- The regression was introduced in the Web UI wiring after that commit, not in the CLI analyze contract.
- Commit `965939e` connected repo-card clicks to analyze but only preserved the selected path for the first half of the flow.
- After analyze completed, the Web UI loaded by `data.repoName ?? repo.name`, which broke the path contract.
- Therefore the fix should be path-through wiring, not a repo-list or global resolver redesign.

## Correct Contract

The Web UI is only the button/progress/graph layer over the analyze runtime.

It should do this:

1. User clicks a Web UI analyze entry point.
2. Web UI starts analyze for the selected repository/path.
3. Backend runs full analyze using the existing analyze implementation.
4. Web UI streams analyze progress.
5. After analyze completes, Web UI loads the graph produced by that analyze.
6. Web UI renders that graph.

It should not do this for any default repo/open/analyze action:

1. User clicks a repository card.
2. Web UI loads the existing graph directly.
3. User sees stale graph data while believing analyze just ran.

It also must not do this:

1. User selects repo B.
2. Web UI starts analyze for repo B by path.
3. Analyze completes.
4. Web UI loads graph by name only, or by a stale/default/current repo binding.
5. User sees repo A, stale repo B, or a LadybugDB missing error for the wrong path/timing.

It also must not silently recover from analyze/load failure by showing a cached or previously active graph. Failures must mention the selected path:

- If analyze fails, show an analyze failure for that selected repo and do not load an old graph.
- If post-analyze graph load fails, show a load failure for that selected repo path and do not fall back to another repo.
- If repo resolution fails, include the selected `repoPath` in diagnostics/logs so the mismatch is traceable.

All analyze entry points must preserve the selected canonical path through the whole chain.

Analyze SSE completion must use an explicit payload:

```ts
{
  repoName: string;
  repoPath: string;
}
```

Frontend completion handlers must prefer `repoPath` and must not let `repoName` redirect post-analyze loading away from the selected path.

The key invariant is simpler than a new architecture:

```text
const selectedPath = repo.path || repo.repoPath;
startAnalyze({ path: selectedPath });
...
connectToServer(..., selectedPath, { awaitAnalysis: true });
```

The completion event may report the analyzed path for verification, but the frontend must not replace the selected path with a repo name.

## Flow Family To Keep In Sync

These flows are one family and must be reviewed together whenever this contract changes:

- Landing repo card selection.
- Landing "Analyze Another Repository" by path.
- Header repository dropdown selection, as open-existing-graph, not analyze.
- Header re-analyze button.
- Header "Analyze a new repository..." sheet.
- Auto-connect from `?project=...`.
- Backend `/api/analyze` job creation, worker execution, SSE completion.
- Backend `/api/repo` and `/api/graph` repo resolution.
- Frontend repo list refresh after analyze.
- Active frontend repo state after graph load.
- Registry/meta/storage state after analyze, for validation only: `meta.json.repoPath`, registry `path`, registry `storagePath`, and the LadybugDB `lbugPath`.
- At least one repo-scoped follow-up call after graph load.
- Browser reload/back-to-landing behavior after failures.

The family review is for consistency only. The implementation should still prefer the smallest path-through change that fixes the broken button flows.

## Implementation Plan

- [x] Keep CLI behavior unchanged.
- [x] Keep `runFullAnalysis` and the pipeline behavior unchanged unless a direct Web API wrapper bug requires a minimal fix.
- [x] Make backend `/api/analyze` always start a full analyze for Web/API analyze requests by passing `force: true` to the analyze worker.
- [x] Remove Web UI reliance on client-provided `force` for analyze semantics.
- [x] Keep any `force` request field only as legacy-tolerated input if needed, but do not let it make Web analyze non-full.
- [x] Update local-path analyze flow to call `/api/analyze`, wait for completion, then load the newly generated graph.
- [x] Update header re-analyze flow to call the same analyze path without adding special semantics beyond full analyze.
- [x] Update repo-card click flow so it starts full analyze for `repo.path`, streams progress, and only connects/loads graph after completion.
- [x] Ensure repo-card click no longer calls graph load directly as the primary action.
- [x] Keep graph loading as a post-analyze rendering step, not the semantic action.
- [x] Adjust user-visible progress wording so analyze and graph loading are distinct steps.
- [x] Add or update backend tests proving `/api/analyze` sends worker options with `force: true` even when the request omits `force` or sends `force: false`.
- [x] Add or update Web tests proving repo-card click starts analyze before graph loading.
- [x] Add or update Web tests proving path analyze starts analyze and then loads the completed repo graph.
- [x] Add or update Web tests proving re-analyze uses the same full-analyze flow.
- [x] Run full launcher build before tests.
- [x] Run targeted backend and Web tests for the analyze flows.
- [x] Run broader relevant test suites if targeted tests pass.

## Reopened Implementation Plan

- [x] Treat `repoPath` as the primary identity for post-analyze loading.
- [x] Do not introduce a broad `RepoIdentity` refactor for this fix; keep the change focused on preserving the selected path through analyze and load.
- [x] Change landing repo-card selection so the selected path is captured before analyze and reused for graph loading after analyze completes.
- [x] Make analyze completion carry the analyzed `repoPath` from backend job to SSE complete payload.
- [x] Define the SSE completion payload as `{ repoName, repoPath }` and update client types accordingly.
- [x] Update frontend analyze completion handlers to load graph by the selected path, optionally verifying it against completed `repoPath`, and never by `data.repoName ?? repo.name`.
- [x] Avoid changing global backend repo resolution unless direct testing proves absolute-path graph loading cannot work without a minimal path-first fix.
- [x] If absolute-path graph loading requires backend routing work, implement only the minimal path-first fix needed for selected-path graph loading and cover that exact case with tests.
- [x] Make landing post-analyze `connectToServer` use the same hold-queue behavior as auto-connect/header flows when loading a just-analyzed graph.
- [x] Verify `connectToServer`, `fetchRepoInfo`, and `fetchGraph` can route by full path without path basename normalization breaking the lookup.
- [x] Verify one representative follow-up repo-scoped call after graph load still targets the loaded path.
- [x] Ensure header "Analyze a new repository..." refreshes the repo list and adds/selects the newly analyzed repo by path.
- [x] Ensure header re-analyze and landing repo-card both use the same simple pattern: selected path -> analyze selected path -> load selected path.
- [x] Keep header repo dropdown switch behavior intentionally separate if it remains a fast graph switch, label/implement it as "switch/open existing graph", and still route it by canonical `repoPath`.
- [x] Ensure no analyze/load failure path silently falls back to a cached graph, previous active graph, default repo, or name-matched repo.
- [x] Ensure analyze/load failure messages and logs include the selected canonical `repoPath`.
- [x] During validation, verify post-analyze storage state: `meta.json.repoPath`, registry `path`, registry `storagePath`, and graph-load `lbugPath` all point to the selected repo path.
- [x] Add regression tests where current/default repo A exists and selected repo B is analyzed; assert all API calls after completion target repo B's path.
- [x] Add regression tests for duplicate repo names/basenames so name-only routing cannot pass.
- [x] Add tests for landing repo-card, local-path analyze, header re-analyze, and header analyze-new using the same identity contract.
- [x] Add tests proving SSE complete includes `repoPath` and UI prefers it over `repoName`.
- [x] Add tests proving Web analyze/load wiring does not use repo list or repo name after analyze completion to decide the graph target.
- [x] Add tests proving at least one representative follow-up repo-scoped call still targets the loaded path after graph load.
- [x] Add tests proving analyze/load failures do not display a stale graph or fall back to another repo.
- [x] Add validation proving post-analyze metadata/registry/storage paths belong to the selected repo; add tests only if the implementation touches this area.
- [x] Run full launcher build before browser validation.
- [x] Before browser validation, stop stale local servers if needed, start the freshly built launcher/backend/UI, and use a clean browser session or hard reload with cache cleared.
- [x] Validate manually in a real browser with a clean session/cache:
  - prepare repo B with a small canary source change before clicking;
  - click landing repo card for repo B;
  - confirm POST `/api/analyze` body path is repo B;
  - confirm SSE complete includes repo B path;
  - confirm `/api/repo` and `/api/graph` load repo B;
  - confirm repo B `meta.json.repoPath`, registry `path`, registry `storagePath`, and graph-load `lbugPath` point to repo B;
  - confirm the canary change appears in graph/search after load;
  - confirm repo B indexed timestamp changes;
  - confirm UI renders repo B graph and no related UI, console, or network error appears.
- [x] Repeat browser validation for:
  - landing "Analyze Another Repository";
  - header re-analyze;
  - header "Analyze a new repository...";
  - header dropdown repo switch if it is intentionally load-only.
- [x] For every browser validation flow, confirm no related UI, console, or network error appears.
- [x] After each graph load validation, confirm at least one follow-up repo-scoped action still targets the same repo path.
- [x] Only mark this plan complete after real browser validation and tests both prove the same repoPath contract.

## Acceptance Criteria

- CLI analyze behavior is unchanged.
- Web UI repo-card click cannot show an old graph without first running analyze.
- Web UI path analyze cannot early-return as up-to-date through a non-force request.
- Web UI re-analyze remains a full analyze.
- Graph loading is visibly and structurally a post-analyze step.
- Tests lock the Web/API contract so future edits cannot reintroduce cached-load-as-analyze behavior.

Additional reopened acceptance criteria:

- The physical repository path selected by the user is the same path analyzed by the backend and the same path used to load graph data.
- A repo name returned by analyze completion cannot redirect graph loading away from the selected path.
- Landing repo-card flow no longer has weaker post-analyze loading semantics than header/auto-connect flows.
- Real browser validation confirms the complete flow, not just mocked unit tests.
- If absolute-path graph loading requires backend routing changes, the backend resolves that selected absolute path before basename/name matching for the graph-load path.
- A representative follow-up repo-scoped action remains path-correct after graph load.
- Browser validation uses fresh built assets and verifies a source canary appears in the newly generated graph.
- Header dropdown switch is explicitly treated as switch/open-existing-graph if it remains fast, not as analyze, and is allowed to stay load-only.
- Analyze/load failures never fall back to stale or unrelated graphs.
- Post-analyze metadata, registry, storage, and LadybugDB paths all belong to the selected repository path.
- The implementation remains a narrow Web UI wiring fix; CLI analyze behavior and broad repo resolution semantics are not changed.

## Validation Log

- Full launcher build passed after the final code/test changes:
  - `powershell -ExecutionPolicy Bypass -File .\avmatrix-launcher\build.ps1`
- Backend targeted tests passed:
  - `cd avmatrix && npx vitest run test/unit/analyze-api.test.ts test/unit/repo-resolver.test.ts`
- Web targeted tests passed:
  - `cd avmatrix-web && npm test -- test/unit/DropZone.full-analyze-flow.test.tsx test/unit/Header.reanalyze-flow.test.tsx test/unit/RepoAnalyzer.local-only.test.tsx test/unit/repo-list.test.ts test/unit/server-connection.test.ts`
  - Result: 5 files, 30 tests passed.
- Browser validation with freshly built launcher passed for landing repo-card `AVmatrix`:
  - POST `/api/analyze` body was `{"path":"F:\\AVmatrix-main"}`.
  - SSE job status completed with `repoPath: "F:\\AVmatrix-main"`.
  - `/api/repo` loaded `repo=F%3A%5CAVmatrix-main&awaitAnalysis=true`.
  - `/api/graph` loaded `repo=F%3A%5CAVmatrix-main&stream=true`.
  - Graph stream had no `type:error` records.
  - Follow-up graph query found canary `buildAnalyzeCompleteEventPayload` in `avmatrix/src/server/api.ts`.
  - No related UI, console, or network error appeared.
- Browser repeat validation passed for the remaining flow family:
  - landing "Analyze Another Repository" by path: analyze body and graph load used `F:\AVmatrix-main`.
  - header re-analyze: analyze body and graph load used `F:\AVmatrix-main`.
  - header "Analyze a new repository...": analyze body and graph load used `F:\AVmatrix-main`.
  - header dropdown switch: loaded `F:\owner-tool\ui-ux-pro-max-skill-main` by path, made zero `/api/analyze` calls, and follow-up `/api/query` succeeded for that same path.
- Runtime was stopped after browser validation so the launcher bundle is not left locking `server-bundle\node.exe`.

## Re-review Validation Gap

Resolved. The final validation included real browser execution, network/request inspection, full-path analyze/load assertions, graph stream error checks, and a follow-up repo-scoped query.
