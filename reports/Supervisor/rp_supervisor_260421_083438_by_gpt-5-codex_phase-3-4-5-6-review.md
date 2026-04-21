# Supervisor Review — Phase 3 / 4 / 5 / 6

Plan baseline: `docs/plans/2026-04-20-convert-all-to-local.md`

Interpretation note for this round:

- `cleanup` is evaluated as technical cleanup and stable active behavior, not literal deletion of every historical file.
- Compatibility wrappers / excludes that keep the active local-runtime path stable are acceptable when the plan explicitly allows them.
- I am not blocking this round just because a `compat` or `retired` file exists.
- I am blocking only where the repo still leaves a same-scope dead execution path, stale test, or stale copy that conflicts with the phase contract.

## Findings

1. `HIGH` Phase 3 still leaves the old clone/pull execution path parked as dead code in the same scope, together with stale clone-era tests.
   The active analyze flow is local-only now: `/api/analyze` rejects URL input in `gitnexus/src/server/api.ts:1132`, the analyze job contract no longer carries clone-era state in `gitnexus/src/server/analyze-job.ts:24`, and the web contract is local-path-only in `gitnexus-web/src/services/backend-client.ts:59`, `gitnexus-web/src/services/backend-client.ts:662`, and `gitnexus-web/src/components/AnalyzeProgress.tsx:10`.
   But `gitnexus/src/server/git-clone.ts:15`, `gitnexus/src/server/git-clone.ts:39`, `gitnexus/src/server/git-clone.ts:161`, and `gitnexus/src/server/git-clone.ts:171` still ship URL extraction, URL validation, clone/pull progress, and `cloneOrPull()`. The matching tests are still clone-era in `gitnexus/test/unit/git-clone.test.ts:1` and `gitnexus/test/integration/server-analyze.test.ts:3`, `gitnexus/test/integration/server-analyze.test.ts:86`.
   This is not a “must delete the file” complaint. The problem is that the old remote clone path is still kept alive as a dead same-scope path after the phase moved analyze to local-path-only. Under the hard rule for this lane, that remains a blocker.

2. `HIGH` Phase 5 active wiki gating is correct, but the old remote wiki execution path, tests, and CLI-facing copy still remain in the same scope.
   The active CLI surface is gated correctly through `gitnexus/src/cli/index.ts:87` and `gitnexus/src/cli/index.ts:92`, backed by `gitnexus/src/cli/wiki-gated.ts:5` and `gitnexus/src/storage/runtime-config.ts:5`.
   However, `gitnexus/src/cli/wiki.ts:19`, `gitnexus/src/cli/wiki.ts:170`, `gitnexus/src/cli/wiki.ts:183`, `gitnexus/src/cli/wiki.ts:195`, and `gitnexus/src/cli/wiki.ts:296` still preserve the old provider / API-key / OpenRouter / Azure / custom wiki flow. The stale test surface is still present in `gitnexus/test/unit/wiki-flags.test.ts:86`, `gitnexus/test/unit/wiki-llm-client.test.ts:93`, and `gitnexus/test/integration/cli-e2e.test.ts:538`. The stale CLI-facing copy is still present in `gitnexus/skills/gitnexus-cli.md:48`.
   This is also not a literal-delete objection. The blocker is that Phase 5 was supposed to capability-gate wiki and shut down the remote path. Leaving the full old remote wiki path parked beside the gate, with tests and copy still targeting it, creates the same-scope drift your hard rule forbids.

3. `HIGH` Phase 6 still has stale web tests that lock the old GitHub URL onboarding/analyze flow instead of the local-only contract.
   The active local-only web flow is correct. `gitnexus-web/test/unit/RepoAnalyzer.local-only.test.tsx:32` explicitly verifies that `GitHub URL` is no longer shown, and the phase-6 compatibility/local-runtime tests pass.
   But `gitnexus-web/e2e/onboarding.spec.ts:170`, `gitnexus-web/e2e/onboarding.spec.ts:194`, `gitnexus-web/e2e/onboarding.spec.ts:198`, `gitnexus-web/e2e/onboarding.spec.ts:208`, `gitnexus-web/e2e/onboarding.spec.ts:229`, `gitnexus-web/e2e/onboarding.spec.ts:239`, and `gitnexus-web/e2e/onboarding.spec.ts:303` still expect a `GitHub URL` tab and URL input flow.
   Phase 6 explicitly says to update tests that still hard-code provider-era behavior, and the plan checklist explicitly says the UI must no longer expose GitHub URL input. These stale e2e expectations are enough to block Phase 6 even though the active runtime cleanup strategy itself is technically acceptable.

## Verdict

- Phase 3: `NOT APPROVED`
- Phase 4: `APPROVED`
- Phase 5: `NOT APPROVED`
- Phase 6: `NOT APPROVED`

## Notes

- I am treating the new compatibility wrappers in `gitnexus-web/src/components/SettingsPanel.tsx`, `gitnexus-web/src/core/llm/types.ts`, `gitnexus-web/src/core/llm/settings-service.ts`, and the retired fail-fast helper modules as acceptable by design for this round, because the plan explicitly allows compatibility wrappers / excludes while retiring the old active build path.
- I am not using those wrappers as blockers by themselves.
- The blockers above are narrower: same-scope dead remote execution paths and stale tests/copy that still encode the pre-local-only behavior.

## Validation

- `cd gitnexus && npx tsc --noEmit`
- `cd gitnexus && npx vitest run test/unit/analyze-api.test.ts test/unit/analyze-job.test.ts test/unit/runtime-config.test.ts test/unit/wiki-gated.test.ts test/unit/cors.test.ts test/unit/serve-command.test.ts test/unit/cli-index-help.test.ts`
- `cd gitnexus-web && npx tsc -b --noEmit`
- `cd gitnexus-web && npx vitest run test/unit/analyze-contract.local-only.test.tsx test/unit/RepoAnalyzer.local-only.test.tsx test/unit/useBackend.local-only.test.tsx test/unit/OnboardingGuide.local-only.test.tsx test/unit/SettingsPanel.local-runtime.test.tsx test/unit/SettingsPanel.compat-local-runtime.test.tsx test/unit/session-client.test.ts test/unit/package-deps.local-runtime.test.ts test/unit/settings-service-local-runtime.phase6.test.ts test/unit/settings-service.compat-local-runtime.test.ts test/unit/settings-service.test.ts test/unit/types.compat.test.ts test/unit/types.local-runtime.test.ts test/unit/legacy-llm-modules.retired-local-runtime.test.ts test/unit/useAppState.compat.test.tsx test/unit/useAppState.local-runtime.test.tsx`

Result on this snapshot:

- `gitnexus` typecheck: pass
- `gitnexus` targeted tests: `38/38` pass
- `gitnexus-web` typecheck: pass
- `gitnexus-web` targeted tests: `53/53` pass
