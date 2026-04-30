# Testing — AVmatrix

How we structure tests and which commands to run locally and in CI.

## Packages

| Package | Path | Runner | Notes |
| ------- | ---- | ------ | ----- |
| Root tooling | `/` | Prettier / ESLint / Husky | Formatting, lint-staged, repository automation checks. |
| Shared contracts | `avmatrix-shared/` | TypeScript | Shared backend/Web types. Build before CLI/Web when contract files change. |
| CLI + MCP + HTTP backend | `avmatrix/` | Vitest | Primary core/runtime test surface. Includes ingestion, MCP, LadybugDB, repo runtime, embeddings, and server helpers. |
| Web UI unit/component | `avmatrix-web/` | Vitest + jsdom | Graph UI, repo picker/analyze flows, local runtime settings, chat UI/client behavior. |
| Web UI E2E | `avmatrix-web/` | Playwright | Browser flows against real `avmatrix serve` and Vite dev server. |
| Windows launcher | `avmatrix-launcher/` | Build/manual smoke | Build script produces launcher exe, backend wrapper, bundled web dist, and protocol registration. |

## Commands (local)

From repository root, unless noted:

**Root tooling**

```bash
npm install
npm run format:check
npm run lint
```

**`avmatrix-shared`**

```bash
cd avmatrix-shared
npm install
npm run build
```

**`avmatrix` (CLI / library)**

```bash
cd avmatrix
npm install
npm run build
npm test                    # full suite: vitest run
npm run test:unit           # unit only: vitest run test/unit
npm run test:integration    # integration suite
npm run test:coverage
npx tsc --noEmit            # typecheck (matches CI)
```

`avmatrix/vitest.config.ts` runs LadybugDB-heavy integration files in a dedicated sequential project to avoid native file-lock conflicts. Keep that grouping when adding new tests that open LadybugDB stores.

**`avmatrix-web`**

```bash
cd avmatrix-web
npm install
npm run build
npm test                    # unit tests (vitest)
npx tsc -b --noEmit         # typecheck (matches CI)
npm run test:coverage
```

**Web E2E**

Run these in separate terminals:

```bash
cd avmatrix
npm run build
node dist/cli/index.js serve
```

```bash
cd avmatrix-web
npm run dev
```

Then:

```bash
cd avmatrix-web
npm run test:e2e
```

Playwright ignores `manual-record.spec.ts` and `debug-issues.spec.ts` by default. Use `PLAYWRIGHT_INSECURE=1` only for explicit browser-security experiments.

**Windows launcher package**

```powershell
powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1
```

Use this when launcher code, bundled backend/web output, protocol behavior, or user-facing packaged runtime behavior changed.

## What To Run

Run the smallest useful validation for the change.

| Change area | Minimum useful validation |
| ----------- | ------------------------- |
| Docs only | `git diff --check` |
| Shared contracts | `cd avmatrix-shared && npm run build`, then build affected CLI/Web package. |
| CLI command, MCP tool, graph query, ingestion, LadybugDB | `cd avmatrix && npm run build && npx tsc --noEmit && npm test` |
| Narrow core logic | `cd avmatrix && npm run test:unit` plus targeted integration tests if storage/MCP is involved. |
| Web UI component/state only | `cd avmatrix-web && npm run build && npx tsc -b --noEmit && npm test` |
| Repo switching, graph loading, analyze from Web UI | Web build/tests plus manual or Playwright E2E against `avmatrix serve`. |
| Session chat runtime | Web unit tests plus manual `/api/session/status` and one chat request with the local Codex CLI session. |
| Launcher | Full launcher build plus start/reset/stop smoke check. |

Avoid running the full test matrix for docs-only or copy-only changes. Prefer targeted validation first, then broaden when the touched code crosses runtime boundaries.

## Pre-commit hook

A husky pre-commit hook (`.husky/pre-commit`) runs automatically on every `git commit`.

The intended behavior for the current package layout is:

1. **Formatting** — `lint-staged` runs prettier on staged files
2. **`avmatrix-web/` files staged** → `tsc -b --noEmit`
3. **`avmatrix/` files staged** → `tsc --noEmit`

Tests do **not** run in the pre-commit hook — they run in CI (`ci-tests.yml`) only.

Skip with `git commit --no-verify` (use sparingly).

Maintenance note: if `.husky/pre-commit` still contains legacy GitNexus path strings, it will not detect AVmatrix package changes correctly. If you touch the hook, update it to the current package paths: `avmatrix/`, `avmatrix-web/`, and `avmatrix-shared/`.

## Test categories

- **Unit** — Pure logic, parsers, graph/query helpers; fast; no network.
- **Integration** — Real combinations (filesystem, MCP wiring, larger pipelines) as already organized under `avmatrix/test/integration`.
- **Eval-style / golden sets** — For agent- or classification-style behavior, keep labeled inputs and expected outputs (JSON or table-driven tests) and run them in CI when relevant.
- **E2E (web)** — Critical user paths only; prefer `data-testid` attributes for stable selectors. Tests run against the real backend (`avmatrix serve` or `node dist/cli/index.js serve`) and Vite dev server.
- **Manual smoke** — Required for packaged launcher behavior, OS folder picker behavior, and any browser flow that depends on real local machine state.

## Performance metrics (targets)

Set targets to match team expectations, then tune to this repo’s CI reality:

| Metric | Target (initial) | Notes |
| ------ | ---------------- | ----- |
| Unit coverage | Align with CI | CI runs Vitest with coverage in `avmatrix`. |
| Unit wall time | Fast PR feedback | Use `vitest run test/unit` for tight loop. |
| Integration duration | &lt; few minutes | Guard heavy tests with env flags if needed. |
| Web graph interaction | No visible UI stall after graph load | Manually verify when graph loading, selection, or filtering changes. |
| Repo switching | Stable across repeated repo A -> B -> A switches | Required for backend/Web repo-runtime changes. |

## Regression testing

Re-run the full relevant suite when:

- Prompt or agent-behavior documentation changes (if tests encode behavior)
- Model or embedding-related code paths change
- Graph schema, query contracts, or MCP tool shapes change
- Dependencies with parsing or runtime impact upgrade
- Repo registry, repo resolver, graph streaming, or Web repo switching changes
- Session bridge, Codex adapter, or chat cancellation changes
- Launcher process management, reset, stop, or protocol registration changes

Manual regression matrix for local runtime changes:

1. Index at least two local repos with `avmatrix analyze .`.
2. Start `avmatrix serve` directly, without the launcher.
3. Start `avmatrix-web` and switch repo A -> B -> A from the dropdown.
4. Confirm graph loads through `/api/graph?repo=...&stream=true` and the UI does not fall back to the previous repo after a successful load.
5. Click a graph node, a dashboard file, and a search result; each should use the same visible graph selection path.
6. Start analyze from the Web UI and confirm the repo list/dropdown refreshes after success.
7. If chat changed, confirm `/api/session/status` reflects the local Codex CLI session and no AVmatrix API key is required.
8. If launcher changed, build the launcher and smoke-test start, reset, stop, and protocol registration.

## CI integration

GitHub Actions (`.github/workflows/ci.yml`) orchestrate:

- **`ci-quality.yml`** — prettier format check, eslint lint, `tsc --noEmit` for `avmatrix/`, `tsc -b --noEmit` for `avmatrix-web/`
- **`ci-tests.yml`** — `vitest run` with coverage (ubuntu) + cross-platform (macOS, Windows)
- **`ci-e2e.yml`** — Playwright E2E tests, gated on `avmatrix-web/**` changes

Local checks before pushing:

```bash
cd avmatrix-shared && npm run build
cd ../avmatrix && npx tsc --noEmit && npm test
cd ../avmatrix-web && npx tsc -b --noEmit && npm test
```

The pre-commit hook is useful for formatting/typecheck feedback, but do not treat it as a replacement for package-specific validation.

## User acceptance / beta (optional)

For staged releases or UI betas, use the packaged local runtime rather than a remote staging service:

1. Build with `avmatrix-launcher\build.ps1`.
2. Open `Start-AVmatrix.html`.
3. Verify backend health at `http://localhost:4747/api/info`.
4. Verify repo picking, repo switching, graph selection, analyze, reset runtime, and chat status on the target Windows machine.
5. Collect runtime logs from `avmatrix-launcher/logs/` when diagnosing failures.
