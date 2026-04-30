# Contributing to AVmatrix

How to work on the current AVmatrix codebase, run the right checks, and open pull requests without changing the local-first runtime contract.

## Project Model

AVmatrix is a local code-intelligence tool:

- `avmatrix analyze` indexes a local repository into `<repo>/.avmatrix/`.
- `avmatrix mcp` exposes indexed repositories to MCP clients over stdio.
- `avmatrix serve` exposes the same local runtime to the Web UI over localhost.
- `avmatrix-web/` is a thin browser client. It does not own indexing or graph storage.
- `avmatrix-launcher/` is an optional Windows convenience launcher around the same backend and Web UI.
- Chat currently runs through the local session bridge and the Codex CLI adapter. `claude-code` exists as a shared provider identifier/UI slot, but there is no backend Claude Code adapter yet.

Keep these invariants intact:

- Do not add an AVmatrix-hosted cloud service to the runtime path.
- Do not require API keys to be stored in AVmatrix.
- Do not make the launcher mandatory. `avmatrix serve` must remain the direct backend entry point.
- Do not move indexed repositories into a managed workspace just to make repo switching work.
- Repo context should be explicit by repo name/path/session. Avoid new process-global "active repo" state for graph reads.

## Repository Layout

| Path | Purpose |
|------|---------|
| `avmatrix/` | npm package: CLI, MCP server, local HTTP API, ingestion pipeline, LadybugDB access, embeddings, session bridge. |
| `avmatrix-web/` | Vite + React Web UI for graph exploration, repo picker/analyze flows, and session chat. |
| `avmatrix-shared/` | Shared TypeScript contracts used by backend and Web UI. |
| `avmatrix-launcher/` | Windows launcher, protocol handler, server wrapper, bundled web/backend assets. |
| `docs/` | Plans, design notes, and investigation history. |
| `.github/` | CI, release, PR, and repository automation. |

## Development Setup

Requires Node.js 20+, npm, and Git.

Install root tooling first if you want Husky, lint-staged, ESLint, or Prettier available:

```powershell
npm install
```

Build the shared package and CLI:

```powershell
cd avmatrix-shared
npm install
npm run build

cd ..\avmatrix
npm install
npm run build
npm link

avmatrix --version
```

Install the Web UI when changing browser code:

```powershell
cd avmatrix-web
npm install
```

Package the Windows launcher only when launcher/runtime artifacts changed or when a user-facing packaged build is needed:

```powershell
powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1
```

## Local Runtime Workflow

Index a repository:

```powershell
cd C:\path\to\repo
avmatrix analyze .
```

Force a clean rebuild:

```powershell
avmatrix analyze . --force
```

Configure MCP/editor access:

```powershell
avmatrix setup
```

Run MCP manually:

```powershell
avmatrix mcp
```

Run the local Web backend:

```powershell
cd avmatrix
node dist\cli\index.js serve
```

Or, after `npm link`:

```powershell
avmatrix serve
```

Run the Web UI dev server:

```powershell
cd avmatrix-web
npm run dev
```

The default local endpoints are:

```text
backend: http://localhost:4747
web:     http://localhost:5173
```

## Validation

Run checks based on what you changed. Do not run the entire suite by habit for a docs-only change.

Docs only:

```powershell
git diff --check
```

Root formatting/linting:

```powershell
npm run format:check
npm run lint
```

CLI/core/MCP/backend changes:

```powershell
cd avmatrix
npm run build
npx tsc --noEmit
npm test
```

Narrower CLI/core loops:

```powershell
cd avmatrix
npm run test:unit
npm run test:integration
```

Web UI changes:

```powershell
cd avmatrix-web
npm run build
npx tsc -b --noEmit
npm test
```

Web E2E changes:

```powershell
cd avmatrix
avmatrix serve

cd ..\avmatrix-web
npm run dev
npm run test:e2e
```

Launcher changes:

```powershell
powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1
```

For graph/runtime changes, also manually verify repo switching across at least two indexed local repos through the Web UI.

## Pre-commit Hook

The repo includes `.husky/pre-commit` for staged formatting and package typechecks. Tests are intended to run in CI, not in the pre-commit hook.

Important maintenance note: some automation files may still contain legacy GitNexus path strings after the repo rename. If you touch `.husky/`, `.github/`, issue templates, or PR templates, audit those files and update paths to `avmatrix/`, `avmatrix-web/`, and `avmatrix-shared/` as part of the same change.

Do not rely on the hook as the only verification. Run the commands relevant to your touched area.

## Pull Requests

Use short-lived branches and keep diffs focused. A PR should explain:

- what changed
- why it changed
- how it was verified
- risk, rollback, or migration notes if relevant

PR titles must follow conventional-commit format:

```text
<type>[(scope)][!]: <subject>
```

Allowed types:

| Type | Label |
|------|-------|
| `feat` | `enhancement` |
| `fix` | `bug` |
| `perf` | `performance` |
| `refactor` | `refactor` |
| `docs` | `documentation` |
| `test` | `test` |
| `ci` | `ci` |
| `build` / `deps` | `dependencies` |
| `chore` / `revert` | `chore` |

Use `!` or `BREAKING CHANGE:` in the PR body for breaking changes.

Examples:

```text
feat(web): add repo-scoped graph loading
fix(server): keep graph reads isolated by repo
docs: refresh local runtime runbook
ci: update workflow paths for avmatrix packages
```

## Coding Guidelines

- Prefer existing package boundaries and local helper APIs.
- Keep write scopes small and avoid unrelated refactors.
- Use structured parsers or existing graph/runtime APIs instead of ad hoc string processing when available.
- Update `README.md`, `RUNBOOK.md`, `ARCHITECTURE.md`, or `TESTING.md` when public behavior changes.
- Never commit secrets, tokens, real `.env` files, or machine-specific credentials.
- Be careful with generated or repo-local index data under `.avmatrix/`; it is runtime data, not source code.

## Runtime-Sensitive Changes

Treat these areas as high-risk:

- `avmatrix/src/core/run-analyze.ts`
- `avmatrix/src/core/lbug/`
- `avmatrix/src/runtime/repo-runtime/`
- `avmatrix/src/server/api.ts`
- `avmatrix/src/server/session-bridge.ts`
- `avmatrix/src/mcp/`
- `avmatrix-web/src/hooks/useAppState.local-runtime.tsx`
- `avmatrix-web/src/services/backend-client.ts`
- `avmatrix-launcher/`

For these changes, document the before/after behavior and verify the affected runtime path directly.

## CI And Release Automation

Current workflow intent:

- `.github/workflows/ci.yml` orchestrates quality, tests, and E2E.
- `.github/workflows/ci-quality.yml` owns format, lint, and typecheck jobs.
- `.github/workflows/ci-tests.yml` owns unit/integration test jobs.
- `.github/workflows/ci-e2e.yml` owns Playwright browser flows.
- `.github/workflows/pr-labeler.yml` validates PR titles and applies labels.
- `.github/workflows/publish.yml` publishes stable `v*` tags to npm.
- `.github/workflows/release-candidate.yml` publishes RC builds from `main`.
- `.github/workflows/docker.yml` builds tagged Docker images.

Every entry-point workflow should have a top-level `concurrency:` block. Reusable workflows invoked through `workflow_call` should avoid `${{ github.workflow }}` in reusable group keys when that could collide with the caller. Use the existing workflow comments as the source of truth when editing CI.

Before relying on release automation, verify package paths and package names use AVmatrix names, not legacy GitNexus names.

## Release Notes

Update `CHANGELOG.md` for user-visible changes. Use `docs/plans/` as supporting history when a change is part of a planned migration or investigation.

Stable releases are tag-driven:

```powershell
git tag vX.Y.Z
git push origin vX.Y.Z
```

Before tagging, make sure `avmatrix/package.json`, `avmatrix/package-lock.json`, `README.md`, and `CHANGELOG.md` agree on the version and release notes.

## Related Docs

- [README.md](README.md)
- [RUNBOOK.md](RUNBOOK.md)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- [TESTING.md](TESTING.md)
- [GUARDRAILS.md](GUARDRAILS.md)
