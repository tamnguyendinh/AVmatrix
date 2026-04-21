<!-- version: 1.4.0 -->
<!-- Last updated: 2026-04-16 -->

Last reviewed: 2026-04-16

**Project:** AVmatrix · **Environment:** dev · **Maintainer:** repository maintainers (see GitHub)

## Scope

| Boundary | Rule |
|----------|------|
| **Reads** | `gitnexus/`, `gitnexus-web/`, `eval/`, plugin packages, `.github/`, `.gitnexus/`, docs. |
| **Writes** | Only paths required for the change; keep diffs minimal. Update lockfiles when deps change. |
| **Executes** | `npm`, `npx`, `node` under `gitnexus/` and `gitnexus-web/`; `uv run` for Python under `eval/`; documented CI/dev workflows. |
| **Off-limits** | Real `.env` / secrets, production credentials, unrelated repos, destructive git ops without confirmation. |

## Model Configuration

- **Primary:** Use a named model (e.g. Claude Sonnet 4.x). Avoid `Auto` or unversioned `latest` when reproducibility matters.
- **Notes:** The AVmatrix CLI indexer does not call an LLM.

## Execution Sequence (complex tasks)

For multi-step work, state up front:
1. Which rules in this file and **[GUARDRAILS.md](GUARDRAILS.md)** apply (and any relevant Signs).
2. Current **Scope** boundaries.
3. Which **validation commands** you will run (`cd gitnexus && npm test`, `npx tsc --noEmit`).

On long threads, *"Remember: apply all AGENTS.md rules"* re-weights these instructions against context dilution.

## Claude Code hooks

**PreToolUse** hooks can block tools (e.g. `git_commit`) until checks pass. Adapt to this repo: `cd gitnexus && npm test` before commit.

## Context budget

Commands and gotchas live under **Repo reference** below and in **[CONTRIBUTING.md](CONTRIBUTING.md)**. If always-on rules grow, split into **`.cursor/rules/*.mdc`** (globs). **Cursor:** project-wide rules in `.cursor/index.mdc`. **Claude Code:** load `STANDARDS.md` only when needed.

## Reference docs

- **[ARCHITECTURE.md](ARCHITECTURE.md)**, **[CONTRIBUTING.md](CONTRIBUTING.md)**, **[GUARDRAILS.md](GUARDRAILS.md)**
- **Call-resolution DAG:** See ARCHITECTURE.md § Call-Resolution DAG. Typed 6-stage DAG inside the `parse` phase; language-specific behavior behind `inferImplicitReceiver` / `selectDispatch` hooks on `LanguageProvider`. Shared code in `gitnexus/src/core/ingestion/` must not name languages. Types: `gitnexus/src/core/ingestion/call-types.ts`.
- **Cursor:** `.cursor/index.mdc` (always-on); `.cursor/rules/*.mdc` (glob-scoped). Legacy `.cursorrules` deprecated.
- **AVmatrix:** skills in `.claude/skills/avmatrix/`; MCP rules in `avmatrix:start` block below.

## Changelog

| Date | Version | Change |
|------|---------|--------|
| 2026-04-16 | 1.4.0 | Fixed: web UI description, pre-commit behavior, MCP tools (7->16), added avmatrix-shared, removed stale vite-plugin-wasm gotcha. |
| 2026-04-13 | 1.3.0 | Updated AVmatrix index stats after DAG refactor. |
| 2026-03-24 | 1.2.0 | Fixed avmatrix:start block duplication. |
| 2026-03-23 | 1.1.0 | Updated agent instructions, references, Cursor layout. |
| 2026-03-22 | 1.0.0 | Initial structured header and changelog. |

---

<!-- avmatrix:start -->
# AVmatrix — Code Intelligence

This project is indexed by AVmatrix as **AVmatrix-main** (17899 symbols, 23379 relationships, 300 execution flows). Use the AVmatrix MCP tools to understand code, assess impact, and navigate safely.

> If any AVmatrix tool warns the index is stale, run `avmatrix analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `rename` which understands the call graph.
- NEVER commit changes without running `detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `avmatrix://repo/AVmatrix-main/context` | Codebase overview, check index freshness |
| `avmatrix://repo/AVmatrix-main/clusters` | All functional areas |
| `avmatrix://repo/AVmatrix-main/processes` | All execution flows |
| `avmatrix://repo/AVmatrix-main/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/avmatrix/avmatrix-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/avmatrix/avmatrix-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/avmatrix/avmatrix-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/avmatrix/avmatrix-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/avmatrix/avmatrix-guide/SKILL.md` |
| Index, status, clean, and wiki capability CLI commands | `.claude/skills/avmatrix/avmatrix-cli/SKILL.md` |

<!-- avmatrix:end -->

## Repo reference

### Packages

| Package | Path | Purpose |
|---------|------|---------|
| **CLI/Core** | `gitnexus/` | TypeScript CLI, indexing pipeline, MCP server. Published to npm. |
| **Web UI** | `gitnexus-web/` | React/Vite thin client. All queries via `gitnexus serve` HTTP API. |
| **Shared** | `gitnexus-shared/` | Shared TypeScript types and constants. |
| Claude Plugin | `avmatrix-claude-plugin/` | Static config for Claude marketplace. |
| Cursor Integration | `avmatrix-cursor-integration/` | Static config for Cursor editor. |
| Eval | `eval/` | Python evaluation harness (Docker + LLM API keys). |

### Running services

```bash
cd gitnexus && npm run dev                 # CLI: tsx watch mode
cd gitnexus-web && npm run dev             # Web UI: Vite on port 5173
npx gitnexus serve                         # HTTP API on port 4747 (from any indexed repo)
```

### Testing

**CLI / Core (`gitnexus/`)**
- `npm test` — full vitest suite (~2000 tests)
- `npm run test:unit` — unit tests only
- `npm run test:integration` — integration (~1850 tests). LadybugDB file-locking tests may fail in containers (known env issue).
- `npx tsc --noEmit` — typecheck

**Web UI (`gitnexus-web/`)**
- `npm test` — vitest (~200 tests)
- `npm run test:e2e` — Playwright (7 spec files; requires `gitnexus serve` + `npm run dev`)
- `npx tsc -b --noEmit` — typecheck

**Pre-commit hook** (`.husky/pre-commit`): formatting (prettier via lint-staged) + typecheck for staged packages. Tests do **not** run in pre-commit — CI only.

### Gotchas

- `npm install` in `gitnexus/` triggers `prepare` (builds via `tsc`) and `postinstall` (patches tree-sitter-swift, builds tree-sitter-proto). Native bindings need `python3`, `make`, `g++`.
- `tree-sitter-kotlin` and `tree-sitter-swift` are optional — install warnings expected.
- ESLint configured via `eslint.config.mjs` (TS, React Hooks, unused-imports). No `npm run lint` script; use `npx eslint .`. Prettier runs via lint-staged. CI checks both in `ci-quality.yml`.
