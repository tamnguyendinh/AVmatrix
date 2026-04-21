# Supervisor Review — Phases 2, 2B, 3, 4

- Plan: `docs/plans/2026-04-20-convert-all-to-local.md`
- Scope reviewed: Phases 2, 2B, 3, 4 only
- Head reviewed: `3889218` (`test: lock web package to local runtime deps`)

## Verdict

- Phase 2: `APPROVED`
- Phase 2B: `APPROVED`
- Phase 3: `NOT APPROVED`
- Phase 4: `NOT APPROVED`

## Findings

### 1. HIGH — The active local-only analyze flow still emits a stale `cloning` state and remote-era copy

The backend and web contracts still model local-path analysis as if a clone step exists:

- backend analyze job status still includes `cloning` at [`avmatrix/src/server/analyze-job.ts:24`](F:\AVmatrix-main\avmatrix\src\server\analyze-job.ts:24)
- `/api/analyze` still marks every accepted local-path job as `cloning` before analysis starts at [`avmatrix/src/server/api.ts:1155`](F:\AVmatrix-main\avmatrix\src\server\api.ts:1155)
- the active web client status type still includes `cloning` at [`avmatrix-web/src/services/backend-client.ts:61`](F:\AVmatrix-main\avmatrix-web\src\services\backend-client.ts:61)
- the active progress UI still renders `cloning` as `Cloning repository` and `pulling` as `Pulling latest` at [`avmatrix-web/src/components/AnalyzeProgress.tsx:10`](F:\AVmatrix-main\avmatrix-web\src\components\AnalyzeProgress.tsx:10)

This is not just naming noise. It is a stale state machine in the exact Phase 3/4 scope that was migrated to local-path-only. Under the supervisor hard rule, leaving a stale handler/state/copy in the touched scope is a blocker.

### 2. HIGH — The shared web analyze contract still allows URL-shaped requests and responses after the local-only migration

The active client contract was not fully tightened:

- `startAnalyze()` still accepts `{ url?: string; path?: string; ... }` at [`avmatrix-web/src/services/backend-client.ts:664`](F:\AVmatrix-main\avmatrix-web\src\services\backend-client.ts:664)
- `JobStatus` still exposes `repoUrl?: string` at [`avmatrix-web/src/services/backend-client.ts:59`](F:\AVmatrix-main\avmatrix-web\src\services\backend-client.ts:59)

That means TypeScript consumers can still compile remote-analyze shaped calls even though the product contract is local-path-only. This is stale wiring in the exact Phase 3 scope. The UI currently calls it correctly with `path`, but the exported contract is still wider than the migrated product.

### 3. HIGH — The Phase 3 tests are too narrow and missed the stale analyze contract

The new local-only tests do verify the visible input path:

- `RepoAnalyzer` submits `{ path: ... }` at [`avmatrix-web/test/unit/RepoAnalyzer.local-only.test.tsx:36`](F:\AVmatrix-main\avmatrix-web\test\unit\RepoAnalyzer.local-only.test.tsx:36)

But the backend-side Phase 3 test coverage is still too narrow:

- `avmatrix/test/unit/analyze-api.test.ts` only tests `resolveAnalyzeRepoPath()` helper behavior at [`avmatrix/test/unit/analyze-api.test.ts:1`](F:\AVmatrix-main\avmatrix\test\unit\analyze-api.test.ts:1)

It does not assert:

- `/api/analyze` no longer exposes/uses `cloning`
- the shared client contract no longer accepts `url`
- the active analyze progress surface no longer shows clone/pull wording

Under the supervisor hard rule, stale tests in the same scope are blockers when they allow stale paths to survive a migration batch.

## What looks correct

### Phase 2

The active web product path is on the local-runtime/session bridge:

- app imports `useAppState.local-runtime` and `SettingsPanel.local-runtime` directly at [`avmatrix-web/src/App.tsx:1`](F:\AVmatrix-main\avmatrix-web\src\App.tsx:1)
- the old `useAppState.tsx` path is now a compatibility re-export only
- local-runtime chat flow, cancel flow, and session status are covered by targeted tests in `useAppState.local-runtime.test.tsx`, `session-client.test.ts`, `SettingsPanel.local-runtime.test.tsx`, and `RightPanel.local-runtime.test.tsx`

### Phase 2B

CLI/MCP/runtime alignment looks consistent with the plan:

- CLI help describes local runtime / MCP surfaces at [`avmatrix/src/cli/index.ts:15`](F:\AVmatrix-main\avmatrix\src\cli\index.ts:15)
- MCP boot path is covered by `mcp-runtime-alignment.test.ts`
- direct tool reuse of shared backend/runtime is covered by `tool-runtime-alignment.test.ts`
- setup behavior for MCP/Codex local runtime is covered in `setup.test.ts` and `setup-codex.test.ts`

### Phase 4 areas that look good

- onboarding now points to `avmatrix serve` without `avmatrix@latest` fallback, covered at [`avmatrix-web/test/unit/OnboardingGuide.local-only.test.tsx:5`](F:\AVmatrix-main\avmatrix-web\test\unit\OnboardingGuide.local-only.test.tsx:5)
- loopback-only backend URL enforcement is covered in `useBackend.local-only.test.tsx` and `server-connection.test.ts`
- loopback-only CORS policy is covered in `avmatrix/test/unit/cors.test.ts`

## Validation run

- `cd avmatrix && npx tsc --noEmit` — pass
- `cd avmatrix && npx vitest run test/unit/analyze-api.test.ts test/unit/analyze-job.test.ts test/unit/mcp-runtime-alignment.test.ts test/unit/tool-runtime-alignment.test.ts test/unit/serve-command.test.ts test/unit/wiki-gated.test.ts test/unit/setup.test.ts test/unit/setup-codex.test.ts test/unit/cors.test.ts` — pass (`43/43`)
- `cd avmatrix-web && npx tsc -b --noEmit` — pass
- `cd avmatrix-web && npm test` — pass (`247/247`)

## Supervisor conclusion

The migration is materially real in Phase 2 and 2B.

Phase 3 and 4 are close, but I do not approve them yet because the active local-only analyze flow still carries stale clone-era state/copy and the shared web contract still exposes URL-shaped analyze fields. Those are exactly the kind of stale path/stale test leftovers that the hard review rule says must block approval in the touched scope.
