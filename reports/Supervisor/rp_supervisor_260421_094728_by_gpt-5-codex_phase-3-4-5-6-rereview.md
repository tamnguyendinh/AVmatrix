# Supervisor Re-review — Phase 3 / 4 / 5 / 6

Plan baseline: `docs/plans/2026-04-20-convert-all-to-local.md`

Interpretation note for this round:

- `cleanup` is evaluated as technical cleanup and stable active behavior, not literal deletion of every historical file.
- Compatibility wrappers and retired modules are not blockers by themselves if the active local-only path is correct and they are no longer wired into the product surface.

## Findings

1. `HIGH` Phase 4 still has a stale integration test that locks the pre-hardening setup contract.
   `gitnexus/src/cli/setup.ts` now writes a local MCP entry through `getMcpEntry()` at [setup.ts](F:\GitNexus-main\gitnexus\src\cli\setup.ts:55), returning `command = "gitnexus"` / `args = ["mcp"]` when no resolved binary is found at [setup.ts](F:\GitNexus-main\gitnexus\src\cli\setup.ts:62). But [setup-skills.test.ts](F:\GitNexus-main\gitnexus\test\integration\setup-skills.test.ts:99) still asserts that fallback setup must contain `gitnexus@latest`. This is a same-scope stale test in the setup/local-runtime hardening area, and it fails in targeted validation.

2. `HIGH` Phase 4 also has a stale/brittle unit test in the onboarding local-only surface.
   [OnboardingGuide.local-only.test.tsx](F:\GitNexus-main\gitnexus-web\test\unit\OnboardingGuide.local-only.test.tsx:12) uses `screen.getByText(/local bridge/i)`, but the updated component now renders that text in two places: the descriptive copy at [OnboardingGuide.tsx](F:\GitNexus-main\gitnexus-web\src\components\OnboardingGuide.tsx:233) and the terminal label at [OnboardingGuide.tsx](F:\GitNexus-main\gitnexus-web\src\components\OnboardingGuide.tsx:205). The product copy itself is fine; the test is now stale/ambiguous and fails in targeted validation. Under the hard rule for this review lane, that is a blocker for Phase 4.

## Verdict

- Phase 3: `APPROVED`
- Phase 4: `NOT APPROVED`
- Phase 5: `APPROVED`
- Phase 6: `APPROVED`

Overall batch verdict: `NOT APPROVED` because Phase 4 is still blocked by stale tests in the exact scope that was changed.

## Notes

- Phase 3 blocker from the previous round is closed:
  - `gitnexus/src/server/git-clone.ts` is gone.
  - The remaining cleanup helper is narrow and local-only in [legacy-repo-cache.ts](F:\GitNexus-main\gitnexus\src\server\legacy-repo-cache.ts:5).
  - `server-analyze` integration was updated to the legacy-cache helper in [server-analyze.test.ts](F:\GitNexus-main\gitnexus\test\integration\server-analyze.test.ts:86).
- Phase 5 blocker from the previous round is closed for the active CLI path:
  - [wiki.ts](F:\GitNexus-main\gitnexus\src\cli\wiki.ts:1) is now just a compatibility wrapper to the gate.
  - [cli-e2e.test.ts](F:\GitNexus-main\gitnexus\test\integration\cli-e2e.test.ts:471) now tests the local-only gate instead of remote provider flags.
  - [wiki.compat.test.ts](F:\GitNexus-main\gitnexus\test\unit\wiki.compat.test.ts:29) covers the legacy import path correctly.
- Phase 6 looks acceptable on the active path under the corrected `cleanup` interpretation:
  - compatibility wrappers remain, but they preserve import stability while routing to the local-runtime implementation
  - provider migration tests now assert normalization back to Codex rather than preserving old provider behavior
  - the stale `GitHub URL` e2e expectations appear to have been removed from `gitnexus-web/e2e/onboarding.spec.ts`
- I am not using parked compatibility modules or retired provider literals as blockers in this round.

## Validation

- `cd gitnexus && npx tsc --noEmit`
- `cd gitnexus && npx vitest run test/unit/analyze-api.test.ts test/unit/analyze-job.test.ts test/unit/legacy-repo-cache.test.ts test/unit/wiki-gated.test.ts test/unit/wiki.compat.test.ts test/integration/server-analyze.test.ts test/integration/cli-e2e.test.ts test/integration/setup-skills.test.ts`
- `cd gitnexus-web && npx tsc -b --noEmit`
- `cd gitnexus-web && npx vitest run test/unit/analyze-contract.local-only.test.tsx test/unit/RepoAnalyzer.local-only.test.tsx test/unit/useBackend.local-only.test.tsx test/unit/OnboardingGuide.local-only.test.tsx test/unit/SettingsPanel.local-runtime.test.tsx test/unit/SettingsPanel.compat-local-runtime.test.tsx test/unit/session-client.test.ts test/unit/package-deps.local-runtime.test.ts test/unit/settings-service-local-runtime.phase6.test.ts test/unit/settings-service.compat-local-runtime.test.ts test/unit/settings-service.test.ts test/unit/types.compat.test.ts test/unit/types.local-runtime.test.ts test/unit/legacy-llm-modules.retired-local-runtime.test.ts test/unit/useAppState.compat.test.tsx test/unit/useAppState.local-runtime.test.tsx`

Result on this workspace snapshot:

- `gitnexus` typecheck: pass
- `gitnexus` targeted tests: `51/52` pass
  - failing test: `test/integration/setup-skills.test.ts`
- `gitnexus-web` typecheck: pass
- `gitnexus-web` targeted tests: `52/53` pass
  - failing test: `test/unit/OnboardingGuide.local-only.test.tsx`

Playwright e2e was not run in this review pass.
