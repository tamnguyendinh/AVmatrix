# Supervisor Re-Review — Phase 1 Local Runtime

- Plan: `docs/plans/2026-04-20-convert-all-to-local.md`
- Scope reviewed: `Pha 1: Shared Local Runtime + Session Bridge`
- Reviewed at: `2026-04-20 21:11:03 +07:00`
- Reviewer: `gpt-5-codex`
- Repo head during review: `e2f3277`
- Previous supervisor artifact: `reports/Supervisor/rp_supervisor_260420_203824_by_gpt-5-codex_phase-1-local-runtime.md`
- Verdict: `NOT APPROVED`

## Delta since previous review

There were meaningful fixes after the previous supervisor report:

- `42c29bb` — `fix: close phase 1 session runtime gaps`
- `e2f3277` — `chore: clean up web tsconfig paths`

This re-review is based on current code, not on the older verdict.

## What improved

The following previous blockers are now materially closed:

1. Missing-folder path handling is now normalized into structured runtime errors.
   - `gitnexus/src/runtime/runtime-controller.ts:171`
   - `gitnexus/src/runtime/runtime-controller.ts:215`
   - `gitnexus/src/runtime/runtime-controller.ts:234`
   - Verified by runtime probe: `getStatus({ repoPath: 'F:/this/path/does/not/exist' })` now returns `repo.state = "not_found"` instead of raw `ENOENT`.

2. Stream contract now carries `reasoning` and `runtimeEnvironment`.
   - `gitnexus-shared/src/session.ts:7`
   - `gitnexus-shared/src/session.ts:52`
   - `gitnexus-shared/src/session.ts:85`
   - `gitnexus/src/runtime/session-adapter.ts:32`
   - `gitnexus/src/runtime/runtime-controller.ts:100`

3. Codex adapter now maps `aggregated_output` and emits `reasoning`.
   - `gitnexus/src/runtime/session-adapters/codex.ts:141`
   - `gitnexus/src/runtime/session-adapters/codex.ts:190`
   - `gitnexus/src/runtime/session-adapters/codex.ts:212`

4. Phase-1 behavioral tests were added and they pass.
   - `gitnexus/test/unit/session-bridge.test.ts`
   - `gitnexus/test/unit/runtime-controller.test.ts`
   - `gitnexus/test/unit/codex-session-adapter.test.ts`
   - `cd gitnexus && npx vitest run test/unit/session-bridge.test.ts test/unit/runtime-controller.test.ts test/unit/codex-session-adapter.test.ts` => `16/16` pass

## Validation run

- `cd gitnexus && npx tsc --noEmit` => pass
- `cd gitnexus-shared && npx tsc --noEmit` => pass
- `cd gitnexus-web && npx tsc -b --noEmit` => pass
- `cd gitnexus && npx vitest run test/unit/session-bridge.test.ts test/unit/runtime-controller.test.ts test/unit/codex-session-adapter.test.ts` => pass
- `cd gitnexus-web && npm test` => `220/220` pass
- `cd gitnexus && npm test` => `6598` passed, `98` skipped, but still `3` unhandled worker-fork errors; suite is not clean-green

## Remaining findings

### HIGH-1 — Windows runtime still violates the recorded execution decision, and the fallback fails on a normal local path with spaces

- The plan recorded a specific Windows decision after spike:
  - use `WSL2 bridge` as the primary execution environment for full agent mode
  - do not invent a half-workaround later
- Current adapter still resolves Windows strategy as `auto`, tries WSL, and then silently falls back to native:
  - `gitnexus/src/runtime/session-adapters/codex.ts:19`
  - `gitnexus/src/runtime/session-adapters/codex.ts:46`
  - `gitnexus/src/runtime/session-adapters/codex.ts:293`
  - `gitnexus/src/runtime/session-adapters/codex.ts:314`
- On this machine the real status probe now returns:
  - `runtimeEnvironment: "native"`
  - message: `WSL2 Codex unavailable; using native fallback. /mnt/c/Users/TAM PC/AppData/Roaming/npm/codex: 15: exec: node: not found`
- Worse: the native fallback is not reliable for ordinary Windows repo paths containing spaces.

Direct reproduction:

- `spawn('codex.cmd', ['--version'], { shell: true, cwd: 'F:/GitNexus-main' })` => works
- `spawn('codex.cmd', ['--version'], { shell: true, cwd: 'C:/Users/TAM PC/AppData/Local/Temp/skills-e2e-mixed-YHP0Mp' })` => `spawn C:\WINDOWS\system32\cmd.exe ENOENT`

End-to-end runtime probe against an actually indexed repo from registry:

- `RuntimeController.startChat({ repoName: 'skills-e2e-mixed-YHP0Mp', message: 'Reply with exactly OK.' })`
- Result:
  - `session_started`
  - then `error`
  - `SESSION_START_FAILED: spawn C:\WINDOWS\system32\cmd.exe ENOENT`

This is a real phase-1 blocker because:

- it sits inside the new session runtime itself
- it affects valid local repos
- the failing path shape is normal on Windows user machines
- it confirms the fallback path is not a safe substitute for the WSL2 execution decision

### HIGH-2 — Phase validation gate is still not met because `gitnexus` full test suite is not clean

- The plan’s implementation validation explicitly includes:
  - `cd gitnexus && npx tsc --noEmit && npm test`
- Current `cd gitnexus && npm test` still ends with:
  - `Errors  3 errors`
  - `Vitest caught 3 unhandled errors during the test run`
  - `Worker forks emitted error`
- This is not a clean validation pass.
- Even though targeted phase-1 tests are good, the phase gate itself is still red at package level.

### MEDIUM-1 — Phase 1 backend is in much better shape, but it is still backend-only in practice

- This is no longer a blocker against Phase 1 itself.
- However the product is not yet phase-2 wired:
  - web chat still does not use `/api/session/*`
  - provider/browser path still remains the active web chat path
- This stays out of Phase 1 approval logic, but it means the user-facing migration has not started yet.

## 19-task supervisor matrix

| # | Task | Current status | Verdict |
|---|------|----------------|---------|
| 1 | Theo dõi, không sửa tài liệu/code | Review-only, no project code changed by supervisor | OK |
| 2 | Rà soát code bằng grep verify | Done again on runtime, bridge, tests, and validation | OK |
| 3 | Đánh giá chất lượng code | Improved meaningfully from previous round | MIXED |
| 4 | Liệt kê file/module chưa đạt best practices | Listed below | NOT OK |
| 5 | Đánh giá tốc độ sinh code của coder | Fast response to blockers; closure speed improved | BETTER |
| 6 | Đánh giá style coder | Backend-first, type-first, now more disciplined on contract tests | BETTER |
| 7 | Theo dõi wiring end-to-end | Backend runtime wiring improved; Windows execution path still broken on space-containing repo path | NOT OK |
| 8 | Bindings wire chưa | Backend repo binding is good; runtime execution binding on Windows still unstable | PARTIAL |
| 9 | Frontend còn mock hay real API | Graph/search real backend; chat still old provider path | MIXED |
| 10 | Backend/VPS có vấn đề gì không | Local runtime still has a Windows launch blocker | ISSUE |
| 11 | DB của VPS đã wire chưa | N/A for this phase | N/A |
| 12 | DB của client đã wire chưa | Local index checks and storage lookup are wired | OK |
| 13 | Các nút UI đã wire backend/client chưa | Phase 2 work still pending | PENDING |
| 14 | Module có chạy đúng với Google Wire chưa | No `google/wire` usage found in repo | N/A |
| 15 | Có tuân thủ luật Hard trong AGENTS.md không | Better than previous round, but still cannot approve due runtime blocker + validation gate | FAIL |
| 16 | Vấn đề tiềm ẩn tương lai | Windows path/cwd launch fragility; package test instability | HIGH RISK |
| 17 | Đánh giá tổng quan theo Status -> SPEC -> Hard rules -> completion | Most old phase-1 gaps are fixed, but phase is still not complete | NOT COMPLETE |
| 18 | Đánh giá chất lượng project theo thời gian thực | Trending upward, but not ready to clear gate | IMPROVING |
| 19 | Khuyến nghị cho coder | Fix Windows execution path deterministically and clear full test gate before phase 2 | REQUIRED |

## Files/modules below best practice for the current Phase 1 state

- `gitnexus/src/runtime/session-adapters/codex.ts`
  - `auto -> native fallback` still conflicts with the spike decision
  - native spawn path is not robust on Windows local repos with spaces
- `gitnexus` package validation
  - full test suite still not clean-green

## Supervisor assessment of coder

### What got better

- The coder responded to the previous review with real closure work, not cosmetic edits.
- The new tests directly target the phase contract instead of hiding behind legacy test coverage.
- The runtime contract is now more explicit and easier to review.

### What is still weak

- The most sensitive platform decision was softened into a fallback path instead of being enforced.
- The phase still does not clear its stated package validation gate.

## Status vs spec vs hard rules

- `Status in code`: phase-1 backend runtime is substantially stronger than before.
- `Against spec`: still not complete because Windows runtime does not hold on a common local-path shape and package validation is not clean.
- `Against hard rules`: previous stale-test/stale-wiring blockers were largely fixed, but a new same-scope runtime blocker remains.
- `Real completion in code`: `near-complete backend phase`, but still not approvable.

## Recommendation to coder

1. Remove ambiguity from Windows execution policy.
   - Either enforce WSL2 as the only supported execution path for Windows phase-1 runtime
   - Or reopen the plan decision explicitly and justify a different architecture
2. Do not keep native fallback as the default “silent save” path while it still fails on user-profile paths with spaces.
3. Add a deterministic behavioral test for Windows/native launch with a `cwd` containing spaces, or block that path intentionally with a clear runtime error.
4. Clear the `gitnexus` full test-suite worker-fork errors before claiming phase-1 validation is done.
5. Only after those two gates are clean should the work move forward to Phase 2.

## Final supervisor verdict

- `Phase 1 is much closer than before, but still NOT APPROVED.`
- Main reason now is narrower and more concrete:
  - the Windows runtime execution path is still not trustworthy on this machine for normal local repo paths
  - the package-level validation gate is still not clean-green
