# Supervisor Review Correction — Phase 3 / 4 / 5

Supersedes: `reports/Supervisor/rp_supervisor_260421_080611_by_gpt-5-codex_phase-3-4-5-review.md`

Plan baseline: `docs/plans/2026-04-20-convert-all-to-local.md`

## Findings

1. `HIGH` Phase 3 still leaves clone-era implementation and stale tests inside the same scope that was migrated to local-path-only.
   Active `/api/analyze` is already local-only and only keeps `getCloneDir()` for legacy cleanup at [api.ts](F:\AVmatrix-main\avmatrix\src\server\api.ts:36) and [api.ts](F:\AVmatrix-main\avmatrix\src\server\api.ts:664). But [git-clone.ts](F:\AVmatrix-main\avmatrix\src\server\git-clone.ts:1) still ships URL extraction, URL validation, `CloneProgress` with `cloning|pulling`, and `cloneOrPull()` at [git-clone.ts](F:\AVmatrix-main\avmatrix\src\server\git-clone.ts:15), [git-clone.ts](F:\AVmatrix-main\avmatrix\src\server\git-clone.ts:39), [git-clone.ts](F:\AVmatrix-main\avmatrix\src\server\git-clone.ts:161), and [git-clone.ts](F:\AVmatrix-main\avmatrix\src\server\git-clone.ts:171). The matching tests are also still clone-era: [git-clone.test.ts](F:\AVmatrix-main\avmatrix\test\unit\git-clone.test.ts:1) and the clone-path assertions in [server-analyze.test.ts](F:\AVmatrix-main\avmatrix\test\integration\server-analyze.test.ts:3) and [server-analyze.test.ts](F:\AVmatrix-main\avmatrix\test\integration\server-analyze.test.ts:86). Under the hard rule for this review lane, these are dead/stale leftovers in the same phase scope, so Phase 3 cannot be approved.

2. `HIGH` Phase 4 currently includes an untracked dead compatibility wrapper in the settings surface.
   The workspace contains [SettingsPanel.compat-local-runtime.tsx](F:\AVmatrix-main\avmatrix-web\src\components\SettingsPanel.compat-local-runtime.tsx:1), which only re-exports `SettingsPanel` from the local-runtime file. A direct grep for `SettingsPanel.compat-local-runtime` / `compat-local-runtime` across `avmatrix-web/src` and `avmatrix-web/test` returned no references, so this is a dead path in the current workspace. Because the hardening phase already touched the settings/local-runtime surface, this counts as a same-scope stale artifact and blocks approval until it is either wired intentionally or removed.

3. `HIGH` Phase 5 active wiki gate is correct, but the old remote wiki implementation and its stale tests are still present in the same scope.
   The live CLI surface is correctly gated through [index.ts](F:\AVmatrix-main\avmatrix\src\cli\index.ts:87) and [index.ts](F:\AVmatrix-main\avmatrix\src\cli\index.ts:92), backed by runtime config in [runtime-config.ts](F:\AVmatrix-main\avmatrix\src\storage\runtime-config.ts:5) and fail-safe messaging in [wiki-gated.ts](F:\AVmatrix-main\avmatrix\src\cli\wiki-gated.ts:5). However, [wiki.ts](F:\AVmatrix-main\avmatrix\src\cli\wiki.ts:1) still contains the old remote/API-key/provider flow, including `WikiGenerator`, `resolveLLMConfig`, `detectCursorCLI`, OpenRouter/Azure/OpenAI/custom setup, and API-key prompts at [wiki.ts](F:\AVmatrix-main\avmatrix\src\cli\wiki.ts:19), [wiki.ts](F:\AVmatrix-main\avmatrix\src\cli\wiki.ts:127), [wiki.ts](F:\AVmatrix-main\avmatrix\src\cli\wiki.ts:183), [wiki.ts](F:\AVmatrix-main\avmatrix\src\cli\wiki.ts:195), and [wiki.ts](F:\AVmatrix-main\avmatrix\src\cli\wiki.ts:296). The remote LLM/client path also still exists in [llm-client.ts](F:\AVmatrix-main\avmatrix\src\core\wiki\llm-client.ts:10), and stale tests still lock that legacy behavior in [wiki-flags.test.ts](F:\AVmatrix-main\avmatrix\test\unit\wiki-flags.test.ts:86) and [wiki-llm-client.test.ts](F:\AVmatrix-main\avmatrix\test\unit\wiki-llm-client.test.ts:93). Because Phase 5 scope was specifically about capability-gating and remote shutdown, leaving the old remote wiki path parked here is a blocker under the no-dead-path/no-stale-test rule.

## Verdict

- Phase 3: `NOT APPROVED`
- Phase 4: `NOT APPROVED`
- Phase 5: `NOT APPROVED`

## Notes

- Active local-only analyze contract looks correct now:
  - `/api/analyze` rejects URL input and enters local analysis directly at [api.ts](F:\AVmatrix-main\avmatrix\src\server\api.ts:1132)
  - analyze job contract no longer carries `cloning` or `repoUrl` in [analyze-job.ts](F:\AVmatrix-main\avmatrix\src\server\analyze-job.ts:24)
  - web analyze contract and progress copy are local-only in [backend-client.ts](F:\AVmatrix-main\avmatrix-web\src\services\backend-client.ts:59), [backend-client.ts](F:\AVmatrix-main\avmatrix-web\src\services\backend-client.ts:662), and [AnalyzeProgress.tsx](F:\AVmatrix-main\avmatrix-web\src\components\AnalyzeProgress.tsx:10)
- `avmatrix-web/src` has no active wiki surface left, which is good; the remaining Phase 5 problem is the parked legacy CLI/core/test path, not UI wiring.
- If `SettingsPanel.compat-local-runtime.tsx` is scratch work and not part of the intended batch, it still needs to be removed before approval because the current workspace review lane treats same-scope dead files as blockers.

## Validation

- `cd avmatrix && npx tsc --noEmit`
- `cd avmatrix && npx vitest run test/unit/analyze-api.test.ts test/unit/analyze-job.test.ts test/unit/runtime-config.test.ts test/unit/wiki-gated.test.ts test/unit/cors.test.ts test/unit/serve-command.test.ts test/unit/cli-index-help.test.ts`
- `cd avmatrix-web && npx tsc -b --noEmit`
- `cd avmatrix-web && npx vitest run test/unit/analyze-contract.local-only.test.tsx test/unit/RepoAnalyzer.local-only.test.tsx test/unit/useBackend.local-only.test.tsx test/unit/OnboardingGuide.local-only.test.tsx test/unit/SettingsPanel.local-runtime.test.tsx test/unit/session-client.test.ts`

Result on this workspace snapshot:

- `avmatrix` typecheck: pass
- `avmatrix` targeted tests: `38/38` pass
- `avmatrix-web` typecheck: pass
- `avmatrix-web` targeted tests: `17/17` pass
