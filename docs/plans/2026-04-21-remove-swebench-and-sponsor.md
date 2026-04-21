# Remove SWE-bench and Sponsor Surface

Last updated: 2026-04-21
Status: completed

## Purpose

This plan removes the `SWE-bench` benchmark/evaluation feature from the local GitNexus product surface and removes the sponsor banner from the local web UI.

The goal is to keep the product focused on its actual local-first use case:

- analyze a local repository
- expose local MCP tools to Codex / Claude Code
- explore the code graph in the web UI
- use the local session runtime for chat and code understanding

`SWE-bench` is not part of that daily workflow. It currently introduces:

- API-key-oriented evaluation paths
- evaluation-only CLI surface
- confusing product copy in the local web UI
- a sponsor banner that links outside the app

After this change:

- the local web UI should not show `Sponsor need to buy some API credits to run SWE-bench`
- the product should not expose `SWE-bench` as an available user feature
- the repo should no longer present `SWE-bench` evaluation as part of the active local product

## Hard Rules

- This is an intentional feature removal.
- Do not accidentally remove or degrade any local-first feature that is still part of the core product:
  - analyze / re-analyze
  - local graph exploration
  - query / context / impact / detect-changes
  - MCP
  - Codex / Claude Code local-session flow
  - repo switching
  - code references / transcript grounding
- Do not redesign UI.
- Remove only the `SWE-bench` and sponsor surfaces.
- For any large file or visible UI surface, keep the repo rule:
  - file-new-first if a large refactor is needed
  - parity first
  - then swap
- Update tests so the removal is explicit and locked in behaviorally.

## Scope

### In scope

- `gitnexus-web/src/components/StatusBar.tsx`
- local UI/help/docs copy that implies `SWE-bench` is an active local-user feature
- CLI product surface for `eval-server`
- `eval/` package and its documentation/scripts/configs
- tests tied specifically to `SWE-bench` / `eval-server`
- README/docs references that advertise `SWE-bench` as a supported feature

### Out of scope

- local graph/chat/runtime behavior
- MCP protocol and local MCP server
- repo indexing pipeline
- code wiki replacement work
- AVmatrix rename work

## Product Decision

### What is being removed

1. `SWE-bench` benchmark/evaluation harness
2. `eval-server` CLI surface
3. sponsor banner in the local web UI

### What should remain

1. Status bar still shows:
   - ready/progress state
   - node count
   - edge count
   - primary language
2. No external sponsor CTA in the local product UI
3. No `Run SWE-bench` CTA unless a separate, real, local user-facing feature is implemented later

## Why remove instead of rename

Changing the sponsor link to `Run SWE-bench` would be misleading today because:

- the current implementation is not a normal local-user feature
- it is benchmark/evaluation infrastructure
- it currently relies on provider/API-key evaluation flows

So the correct move is:

- remove the banner
- remove the feature surface
- remove the benchmark package/surface itself

If a real local evaluation feature is desired later, it should come back as a new feature with its own dedicated plan.

## Current Surface Inventory

### Web UI

- `gitnexus-web/src/components/StatusBar.tsx`
  - external sponsor link
  - `need to buy some API credits to run SWE-bench 😅`

### CLI

- `gitnexus/src/cli/index.ts`
  - `eval-server` command registration
- `gitnexus/src/cli/eval-server.ts`
  - evaluation-only HTTP daemon

### Eval package

- `eval/README.md`
- `eval/run_eval.py`
- `eval/analysis/*`
- `eval/agents/*`
- `eval/environments/*`
- `eval/bridge/*`
- `eval/configs/*`
- `eval/tests/*`
- `eval/pyproject.toml`
- `eval/uv.lock`

### Tests / docs coupled to eval-server

- `gitnexus/test/unit/eval-formatters.test.ts`
- `gitnexus/test/integration/cli-e2e.test.ts` eval-server section
- any README/help/docs references discovered during implementation

## Implementation Phases

### Phase A — Remove local web sponsor surface

#### Goal

Remove the sponsor banner and all `SWE-bench` text from the local web UI while keeping the status bar layout intact.

#### Changes

- update `gitnexus-web/src/components/StatusBar.tsx`
- remove:
  - external sponsor link
  - sponsor heart icon block
  - `need to buy some API credits to run SWE-bench 😅`
- keep left status and right graph stats intact
- keep footer height and general layout stable

#### Tests

- add/update status-bar tests so:
  - sponsor text is absent
  - no external sponsor link is rendered
  - ready/progress and graph stats still render

### Phase B — Remove CLI eval-server surface

#### Goal

Remove `eval-server` from the public CLI surface.

#### Changes

- remove `eval-server` registration from `gitnexus/src/cli/index.ts`
- remove `gitnexus/src/cli/eval-server.ts`
- remove any imports/exports or help text tied to it

#### Tests

- update CLI help tests
- remove or replace eval-server integration tests
- ensure CLI help/output no longer advertises the command

### Phase C — Remove eval package

#### Goal

Delete the entire benchmark harness from the repository.

#### Changes

- remove `eval/` package contents
- remove packaging/config references to it from root docs or scripts if present
- remove `README` references that advertise benchmark support

#### Notes

This is intentional deletion, not a refactor.

### Phase D — Remove docs and product messaging

#### Goal

Make the documentation match the new product scope.

#### Changes

- update root `README.md`
- remove references to:
  - `SWE-bench`
  - benchmark claims
  - API-credit/sponsor messaging tied to evaluation
- update any local usage/help docs that still mention evaluation

#### Tests

- doc-grep checks in targeted tests if applicable
- otherwise explicit review of help surfaces

### Phase E — Cleanup and validation

#### Goal

Verify that no stale `SWE-bench` / sponsor product surface remains.

#### Required checks

- `rg -n "SWE-bench|Sponsor need to buy|eval-server" gitnexus gitnexus-web docs README.md`
- targeted test suite for changed surfaces
- typecheck:
  - `cd gitnexus && npx tsc --noEmit`
  - `cd gitnexus-web && npx tsc -b --noEmit`

## Validation Matrix

### Web

- status bar renders without sponsor CTA
- no link to GitHub Sponsors from local UI
- progress/ready state unchanged
- graph stats unchanged

### CLI

- `gitnexus --help` no longer lists `eval-server`
- no dead imports from removed eval-server code

### Repo surface

- no `eval/` package remains
- no public docs still present `SWE-bench` as a user feature

## Risks

### Risk 1 — Accidental removal of unrelated status bar behavior

Mitigation:

- keep the status bar shell/layout
- only remove the center sponsor block
- add targeted tests

### Risk 2 — CLI help/tests break because eval-server was coupled in hidden ways

Mitigation:

- remove the command cleanly from registration
- update help snapshots/behavior tests in the same batch

### Risk 3 — README still advertises removed capabilities

Mitigation:

- explicit grep audit in `Phase E`

## Completion Checklist

- [x] Local web UI no longer shows sponsor banner
- [x] Local web UI no longer mentions buying API credits for SWE-bench
- [x] `eval-server` no longer exists in CLI surface
- [x] `eval/` package is removed
- [x] README/docs no longer advertise SWE-bench
- [x] targeted tests updated and passing
- [x] typecheck passes in `gitnexus`
- [x] typecheck passes in `gitnexus-web`

## Validation Summary

- `cd gitnexus-web && npx vitest run test/unit/StatusBar.local-only.test.tsx`
- `cd gitnexus-web && npm test`
- `cd gitnexus-web && npx tsc -b --noEmit`
- `cd gitnexus && npx vitest run test/unit/cli-index-help.test.ts test/integration/cli-e2e.test.ts`
- `cd gitnexus && npx tsc --noEmit`
- `cd gitnexus && node dist/cli/index.js detect-changes --repo GitNexus-main --scope unstaged`

## Notes

- `AGENTS.md` and `CLAUDE.md` still contain legacy `eval/` references, but they were already dirty outside this task and were intentionally left untouched in this batch.
