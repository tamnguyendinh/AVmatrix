<!-- version: 1.3.0 -->
<!--
  Metadata: version, last reviewed, scope, model policy, reference docs, changelog.
  Last updated: 2026-03-22
-->

Last reviewed: 2026-04-13

**Project:** AVmatrix · **Environment:** dev · **Maintainer:** repository maintainers (see GitHub)

Follow **AGENTS.md** for the canonical rules; this file adds Claude Code–specific deltas. Cursor-specific notes live only in `AGENTS.md`.

## Scope

See the **Scope** table in [AGENTS.md](AGENTS.md) for read/write/execute/off-limits boundaries. Cursor-specific workflow notes also live only in AGENTS.md.

## Model Configuration

- **Primary:** Pin per **Claude Code** / Anthropic org policy (explicit model id). Do not rely on an unversioned `latest` alias for governed workflows.
- **Fallback:** As configured in Claude Code (organization default or user override).
- **Notes:** The AVmatrix CLI analyzer does not call an LLM.

## Execution Sequence (complex tasks)

Same discipline as [AGENTS.md](AGENTS.md): before large multi-step work, state which **AGENTS.md** / **GUARDRAILS.md** rules apply, current **Scope**, and planned validation commands (`npm test`, `tsc`, etc.). When pausing, summarize progress in the chat or a **local** scratch file (do not add `HANDOFF.md` to the repo), then `/clear` and resume with that summary.

## Claude Code hooks

Prefer **PreToolUse** hooks for hard gates (e.g. tests before `git_commit`). Adapt hook commands to `avmatrix/` npm scripts.

## Context budget

If always-on instructions grow, load deep conventions via conditional reads (e.g. *“When writing new code, read STANDARDS.md”*) instead of pasting long blocks here. In Cursor, prefer `.cursor/index.mdc` plus optional `.cursor/rules/*.mdc` globs (see [AGENTS.md](AGENTS.md) § Context budget).

## Reference Documentation

- **This repository:** [AGENTS.md](AGENTS.md) (Cursor + monorepo notes), [ARCHITECTURE.md](ARCHITECTURE.md), [CONTRIBUTING.md](CONTRIBUTING.md), [GUARDRAILS.md](GUARDRAILS.md).
- **Call-resolution DAG:** See ARCHITECTURE.md § Call-Resolution DAG. Shared pipeline code in `avmatrix/src/core/ingestion/` must not name languages — use `LanguageProvider` hooks instead (see AGENTS.md).
- **AVmatrix:** `.claude/skills/avmatrix/`; MCP and indexed-repo rules live only in [AGENTS.md](AGENTS.md) (`avmatrix:start` … `avmatrix:end`). See **AVmatrix rules** below.

## Changelog

| Date | Version | Change |
|------|---------|--------|
| 2026-04-13 | 1.3.0 | Updated AVmatrix index stats after DAG refactor. |
| 2026-03-24 | 1.2.0 | Removed duplicated avmatrix:start block and scope table; replaced with pointers to AGENTS.md. |
| 2026-03-23 | 1.1.0 | Updated agent instructions to match AGENTS.md. |
| 2026-03-22 | 1.0.0 | Added structured header and changelog. |

---

## AVmatrix rules

See the `<!-- avmatrix:start -->
# AVmatrix — Code Intelligence

This project is indexed by AVmatrix as **AVmatrix-main** (17353 symbols, 24959 relationships, 466 execution flows). Use the AVmatrix MCP tools to understand code, assess impact, and navigate safely.

> If any AVmatrix tool warns the index is stale, run `avmatrix analyze` in terminal first.

## Always Do

- **MUST refresh the graph before graph-based work.** Run `avmatrix analyze --force` before using `query`, `context`, `impact`, `detect_changes`, `rename`, or `cypher`.
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

<!-- avmatrix:end -->` block in **[AGENTS.md](AGENTS.md)** for the canonical MCP tools, impact analysis rules, and index instructions.
