# Phase 1 Supervisor Re-review 3

- Plan: `docs/plans/2026-04-20-convert-all-to-local.md`
- Scope reviewed: Phase 1 only
- Head reviewed: `3c9e296` (`fix: harden phase 1 runtime repo resolution`)
- Verdict: `NOT APPROVED`

## Closed blockers

- The Windows-native dead path from the previous review has been removed. `CodexSessionAdapter` is now WSL2-only on Windows, with no native override branch left in Phase 1 code:
  - [`gitnexus/src/runtime/session-adapters/codex.ts:62`](F:\GitNexus-main\gitnexus\src\runtime\session-adapters\codex.ts:62)
  - [`gitnexus/src/runtime/session-adapters/codex.ts:285`](F:\GitNexus-main\gitnexus\src\runtime\session-adapters\codex.ts:285)
  - [`gitnexus/src/runtime/session-adapters/codex.ts:319`](F:\GitNexus-main\gitnexus\src\runtime\session-adapters\codex.ts:319)
- Status messaging is now aligned with the plan decision:
  - `runtimeEnvironment: "wsl2"`
  - `availability: "not_installed"`
  - message explicitly says Windows-native Codex execution is not supported in Phase 1
- Stale registry path handling for `repoName` bindings is now covered behaviorally and matches real controller logic:
  - controller logic at [`gitnexus/src/runtime/runtime-controller.ts:163`](F:\GitNexus-main\gitnexus\src\runtime\runtime-controller.ts:163)
  - status-path test at [`gitnexus/test/unit/runtime-controller.test.ts:144`](F:\GitNexus-main\gitnexus\test\unit\runtime-controller.test.ts:144)
  - chat-path test at [`gitnexus/test/unit/runtime-controller.test.ts:216`](F:\GitNexus-main\gitnexus\test\unit\runtime-controller.test.ts:216)
- The Phase 1 adapter tests no longer keep the stale native override branch alive. Coverage now matches the intended WSL2-only contract:
  - [`gitnexus/test/unit/codex-session-adapter.test.ts:56`](F:\GitNexus-main\gitnexus\test\unit\codex-session-adapter.test.ts:56)
  - [`gitnexus/test/unit/codex-session-adapter.test.ts:64`](F:\GitNexus-main\gitnexus\test\unit\codex-session-adapter.test.ts:64)

## Finding

### 1. HIGH — Full `gitnexus` validation gate is still not clean

Focused Phase 1 validation is green, but the package-level test gate remains dirty:

```text
cd gitnexus && npm test
Test Files  215 passed | 1 skipped (217)
Tests  6603 passed | 98 skipped (6713)
Errors  1 error

Vitest caught 1 unhandled error during the test run.
This might cause false positive tests.

Error: [vitest-pool]: Worker forks emitted error.
fatal: not a git repository (or any of the parent directories): .git
```

This is an improvement over the previous review, but it is still not clean-green. Until the unhandled worker error is removed or explicitly bounded as an accepted environment-only false alarm by maintainers, I do not approve the phase.

## Status vs plan

- Phase 1 code now matches the Windows execution decision from the plan: WSL2 is the only Windows path for the local session runtime.
- The session/runtime contract, adapter behavior, and stale registry handling are in much better shape and the previous Phase 1 code-level blockers appear closed.
- Real completion is still below sign-off because the declared validation command `cd gitnexus && npm test` is not clean yet.

## Validation run

- `cd gitnexus && npx vitest run test/unit/runtime-controller.test.ts test/unit/session-bridge.test.ts test/unit/codex-session-adapter.test.ts` — pass (`18/18`)
- `cd gitnexus && npx tsc --noEmit` — pass
- `cd gitnexus && npm test` — not clean (`1` unhandled worker-fork error)
- Real smoke: `CodexSessionAdapter.getStatus()` on this machine returns:
  - `runtimeEnvironment: "wsl2"`
  - `availability: "not_installed"`
  - message includes `Windows-native Codex execution is not supported in Phase 1`

## Recommendation to coder

- Treat Phase 1 runtime logic as functionally repaired.
- Close the remaining full-suite worker error before asking for final Phase 1 approval again.
