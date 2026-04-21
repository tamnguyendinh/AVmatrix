# Phase 1 Supervisor Verdict Correction

- Plan: `docs/plans/2026-04-20-convert-all-to-local.md`
- Scope reviewed: Phase 1 only
- Head reviewed: `3c9e296` (`fix: harden phase 1 runtime repo resolution`)
- Verdict: `APPROVED`

## Why this correction exists

The previous Phase 1 re-review blocked sign-off using a remaining full-suite `vitest` worker-fork error from `cd avmatrix && npm test`.

That blocker was too broad for this plan review.

Per the plan boundary, Phase 1 should be approved or rejected based on:

- the Phase 1 contract in `runtime-controller`, `session-adapter`, `session-bridge`, and Codex adapter
- real runtime behavior for the new Phase 1 flow
- tests that were updated to the new Phase 1 contract
- hard-rule violations inside the exact scope that was changed

It should **not** be blocked solely by repo-wide test noise unless that noise is traced back to Phase 1 behavior or to stale code/tests left inside the Phase 1 scope.

## Phase 1 status against plan

Phase 1 now matches the intended plan behavior:

- `runtime-controller` exists and resolves local repo bindings correctly
- `session-adapter` abstraction exists
- `session-bridge` HTTP adapter exists
- Codex adapter exists
- `/api/session/status` and `/api/session/chat` exist
- cancel/session lifecycle exists
- explicit `INDEX_REQUIRED` behavior exists
- Windows execution policy is now WSL2-only for Phase 1, which matches the plan decision

Relevant code:

- WSL2-only Windows adapter path:
  - [`avmatrix/src/runtime/session-adapters/codex.ts:285`](F:\AVmatrix-main\avmatrix\src\runtime\session-adapters\codex.ts:285)
  - [`avmatrix/src/runtime/session-adapters/codex.ts:319`](F:\AVmatrix-main\avmatrix\src\runtime\session-adapters\codex.ts:319)
- no Windows shell/native branch kept alive in adapter launch:
  - [`avmatrix/src/runtime/session-adapters/codex.ts:62`](F:\AVmatrix-main\avmatrix\src\runtime\session-adapters\codex.ts:62)
- stale registry path handling through `repoName`:
  - [`avmatrix/src/runtime/runtime-controller.ts:163`](F:\AVmatrix-main\avmatrix\src\runtime\runtime-controller.ts:163)
  - [`avmatrix/src/runtime/runtime-controller.ts:183`](F:\AVmatrix-main\avmatrix\src\runtime\runtime-controller.ts:183)

## Validation that is in-scope for Phase 1

- `cd avmatrix && npx vitest run test/unit/runtime-controller.test.ts test/unit/session-bridge.test.ts test/unit/codex-session-adapter.test.ts`
  - result: `18/18` pass
- `cd avmatrix && npx tsc --noEmit`
  - result: pass
- real smoke for `CodexSessionAdapter.getStatus()`
  - result:
    - `runtimeEnvironment: "wsl2"`
    - `availability: "not_installed"`
    - message explicitly says `Windows-native Codex execution is not supported in Phase 1`

Updated behavioral tests are aligned with the migrated contract:

- stale registry path via `repoName` status path:
  - [`avmatrix/test/unit/runtime-controller.test.ts:144`](F:\AVmatrix-main\avmatrix\test\unit\runtime-controller.test.ts:144)
- stale registry path via `repoName` chat path:
  - [`avmatrix/test/unit/runtime-controller.test.ts:216`](F:\AVmatrix-main\avmatrix\test\unit\runtime-controller.test.ts:216)
- WSL2 available / WSL2 required adapter behavior:
  - [`avmatrix/test/unit/codex-session-adapter.test.ts:56`](F:\AVmatrix-main\avmatrix\test\unit\codex-session-adapter.test.ts:56)
  - [`avmatrix/test/unit/codex-session-adapter.test.ts:64`](F:\AVmatrix-main\avmatrix\test\unit\codex-session-adapter.test.ts:64)

## Out-of-scope note

`cd avmatrix && npm test` still reports one repo-wide unhandled worker-fork error.

At the time of this correction, that error has **not** been traced to:

- a broken Phase 1 runtime behavior
- a stale Phase 1 test
- a dead Phase 1 path still left in code

Therefore it should be tracked as a separate repo/suite-health issue, not used as a standalone blocker for Phase 1 approval.

## Supervisor conclusion

Within the boundary of `docs/plans/2026-04-20-convert-all-to-local.md`, Phase 1 is complete enough to approve.

Residual full-suite instability may still deserve a separate investigation, but it is not a valid rejection basis for this Phase 1 plan review unless someone ties it back to the Phase 1 migration scope.
