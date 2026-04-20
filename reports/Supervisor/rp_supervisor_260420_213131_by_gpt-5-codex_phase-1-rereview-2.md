# Phase 1 Supervisor Re-review 2

- Plan: `docs/plans/2026-04-20-convert-all-to-local.md`
- Scope reviewed: Phase 1 only (`gitnexus/`, `gitnexus-shared/`, Phase 1 tests)
- Head reviewed: `d2588b9` (`fix: enforce wsl2 default for codex runtime`)
- Verdict: `NOT APPROVED`

## What improved since the last review

- Windows default now follows the plan decision to prefer `WSL2 bridge` instead of silently falling back to native execution. The default path is enforced in [`gitnexus/src/runtime/session-adapters/codex.ts:300`](F:\GitNexus-main\gitnexus\src\runtime\session-adapters\codex.ts:300) and reported through [`gitnexus/src/runtime/session-adapters/codex.ts:344`](F:\GitNexus-main\gitnexus\src\runtime\session-adapters\codex.ts:344).
- Missing local paths now fail with structured repo-state handling instead of leaking raw `ENOENT`, via [`gitnexus/src/runtime/runtime-controller.ts:69`](F:\GitNexus-main\gitnexus\src\runtime\runtime-controller.ts:69).
- The Codex bridge now carries `reasoning` and `aggregated_output` semantics that were missing in the first review.
- Phase-1-focused tests now exist and pass.

## Findings

### 1. HIGH — Windows native opt-in path is still a dead path in the exact scope that was just fixed

Code now defaults Windows to WSL2, but it still exposes a reachable native branch through `GITNEXUS_WINDOWS_SESSION_ENV=native` in [`gitnexus/src/runtime/session-adapters/codex.ts:305`](F:\GitNexus-main\gitnexus\src\runtime\session-adapters\codex.ts:305) and still launches native Windows shell execution through [`gitnexus/src/runtime/session-adapters/codex.ts:74`](F:\GitNexus-main\gitnexus\src\runtime\session-adapters\codex.ts:74).

I re-ran the real session path on an indexed repo whose absolute path contains spaces:

```text
repoName: skills-e2e-mixed-YHP0Mp
repoPath: C:\Users\TAM PC\AppData\Local\Temp\skills-e2e-mixed-YHP0Mp
env: GITNEXUS_WINDOWS_SESSION_ENV=native
```

Observed runtime result:

```json
[
  {
    "type": "session_started",
    "runtimeEnvironment": "native",
    "executionMode": "bypass"
  },
  {
    "type": "error",
    "code": "SESSION_START_FAILED",
    "error": "spawn C:\\WINDOWS\\system32\\cmd.exe ENOENT"
  }
]
```

That means the reachable native path is still dead on a common Windows path shape. Under the supervisor hard rule, dead path left inside the just-touched scope is a `HIGH` blocker.

The new test coverage does not close this risk. The native Windows test in [`gitnexus/test/unit/codex-session-adapter.test.ts:144`](F:\GitNexus-main\gitnexus\test\unit\codex-session-adapter.test.ts:144) only asserts mocked `spawn()` options at [`gitnexus/test/unit/codex-session-adapter.test.ts:152`](F:\GitNexus-main\gitnexus\test\unit\codex-session-adapter.test.ts:152), so the dead opt-in path escaped review. In this scope, that is also a stale-test problem.

### 2. HIGH — Phase 1 validation gate is still not clean-green

Focused validation is green, but the full `gitnexus` suite is not clean:

```text
cd gitnexus && npm test
6599 passed | 98 skipped
Errors 3 errors
Vitest caught 3 unhandled errors during the test run.
Error: [vitest-pool]: Worker forks emitted error.
fatal: not a git repository (or any of the parent directories): .git
```

This is not a cosmetic warning. Vitest explicitly says the run may contain false positives. A phase cannot be marked complete while the package-level gate still reports unhandled worker failures.

## Status vs plan

- The plan decision "Windows recommendation is `WSL2 bridge` for full agent mode" is now reflected in the default runtime path. That part is aligned.
- The structural Phase 1 checklist is mostly present in code: `runtime-controller`, `session-adapter`, `session-bridge`, Codex adapter, `/api/session/status`, `/api/session/chat`, cancel lifecycle, `INDEX_REQUIRED`, and focused behavioral tests.
- Real completion is still below the checklist claim because one reachable execution branch in the new runtime layer is dead, and the phase validation gate is still dirty.

## Validation run

- `cd gitnexus && npx tsc --noEmit` — pass
- `cd gitnexus-shared && npx tsc --noEmit` — pass
- `cd gitnexus-web && npx tsc -b --noEmit` — pass
- `cd gitnexus && npx vitest run test/unit/session-bridge.test.ts test/unit/runtime-controller.test.ts test/unit/codex-session-adapter.test.ts` — pass (`17/17`)
- `cd gitnexus-web && npm test` — pass (`220/220`)
- `cd gitnexus && npm test` — not clean (`3` unhandled worker-fork errors)

## Recommendation to coder

- Either remove the reachable Windows-native opt-in path from Phase 1 scope, or make it actually start a real session on Windows paths with spaces. Do not keep it half-alive.
- Add a non-mocked behavioral check for the Windows-native launch branch if that branch remains in the codebase.
- Clear the full `gitnexus` test gate before calling Phase 1 complete again.
