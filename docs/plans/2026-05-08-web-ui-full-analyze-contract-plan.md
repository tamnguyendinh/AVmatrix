# Web UI Full Analyze Contract Plan

Date: 2026-05-08

## Goal

Make the Web UI behave as a visual control surface for the same full analyze behavior users get from the CLI full analyze path. The Web UI should not introduce a separate analyze meaning, shortcut, mode, or cached-graph interpretation for actions that users understand as analyzing a repository.

The user-facing contract is:

- Clicking a repository card runs a full analyze for that repository, then loads the newly generated graph.
- Analyzing by local path runs a full analyze, then loads the newly generated graph.
- Re-analyze runs a full analyze, then loads the newly generated graph.

## Non-Goals

- Do not change CLI analyze semantics.
- Do not change the core analyze pipeline.
- Do not add new analyze modes such as `full` versus `ifChanged`.
- Do not add benchmark work for this fix.
- Do not compare graph-load time with analyze time.
- Do not make cached graph loading the default repo-card behavior.

## Problem

The current Web UI has multiple paths that can be interpreted as analyze:

- Repo-card click can connect to an already indexed repository and load the stored graph.
- Analyze by path can call `/api/analyze` without forcing a rebuild, allowing the up-to-date shortcut.
- Header re-analyze currently forces a rebuild.

That creates contract drift. The CLI full analyze path rebuilds the graph, while some Web UI paths can return quickly because they only load an existing graph or take the non-force shortcut. This makes the UI easy to misread and can show stale graph data after source code changes.

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

## Implementation Plan

- [ ] Keep CLI behavior unchanged.
- [ ] Keep `runFullAnalysis` and the pipeline behavior unchanged unless a direct Web API wrapper bug requires a minimal fix.
- [ ] Make backend `/api/analyze` always start a full analyze for Web/API analyze requests by passing `force: true` to the analyze worker.
- [ ] Remove Web UI reliance on client-provided `force` for analyze semantics.
- [ ] Keep any `force` request field only as legacy-tolerated input if needed, but do not let it make Web analyze non-full.
- [ ] Update local-path analyze flow to call `/api/analyze`, wait for completion, then load the newly generated graph.
- [ ] Update header re-analyze flow to call the same analyze path without adding special semantics beyond full analyze.
- [ ] Update repo-card click flow so it starts full analyze for `repo.path`, streams progress, and only connects/loads graph after completion.
- [ ] Ensure repo-card click no longer calls graph load directly as the primary action.
- [ ] Keep graph loading as a post-analyze rendering step, not the semantic action.
- [ ] Adjust user-visible progress wording so analyze and graph loading are distinct steps.
- [ ] Add or update backend tests proving `/api/analyze` sends worker options with `force: true` even when the request omits `force` or sends `force: false`.
- [ ] Add or update Web tests proving repo-card click starts analyze before graph loading.
- [ ] Add or update Web tests proving path analyze starts analyze and then loads the completed repo graph.
- [ ] Add or update Web tests proving re-analyze uses the same full-analyze flow.
- [ ] Run full launcher build before tests.
- [ ] Run targeted backend and Web tests for the analyze flows.
- [ ] Run broader relevant test suites if targeted tests pass.

## Acceptance Criteria

- CLI analyze behavior is unchanged.
- Web UI repo-card click cannot show an old graph without first running analyze.
- Web UI path analyze cannot early-return as up-to-date through a non-force request.
- Web UI re-analyze remains a full analyze.
- Graph loading is visibly and structurally a post-analyze step.
- Tests lock the Web/API contract so future edits cannot reintroduce cached-load-as-analyze behavior.

