---
name: avmatrix-cli
description: "Use when the user needs to run AVmatrix CLI commands like analyze/index a repo, check status, clean the index, generate a wiki, or list indexed repos. Examples: \"Index this repo\", \"Reanalyze the codebase\", \"Generate a wiki\""
---

# AVmatrix CLI Commands

Canonical commands use `avmatrix` once the local CLI is on `PATH`.

## Commands

### analyze — Build or refresh the index

```bash
avmatrix analyze
```

Run from the project root. This parses all source files, builds the knowledge graph, writes it to the local index directory, and generates CLAUDE.md / AGENTS.md context files.

| Flag           | Effect                                                           |
| -------------- | ---------------------------------------------------------------- |
| `--force`      | Force full re-index even if up to date                           |
| `--embeddings` | Enable embedding generation for semantic search (off by default) |

**When to run:** First time in a project, after major code changes, or when `avmatrix://repo/{name}/context` reports the index is stale. In Claude Code, a PostToolUse hook runs `analyze` automatically after `git commit` and `git merge`, preserving embeddings if previously generated.

### status — Check index freshness

```bash
avmatrix status
```

Shows whether the current repo has an AVmatrix index, when it was last updated, and symbol/relationship counts. Use this to check if re-indexing is needed.

### clean — Delete the index

```bash
avmatrix clean
```

Deletes the local index directory and unregisters the repo from the global registry. Use before re-indexing if the index is corrupt or after removing AVmatrix from a project.

| Flag      | Effect                                            |
| --------- | ------------------------------------------------- |
| `--force` | Skip confirmation prompt                          |
| `--all`   | Clean all indexed repos, not just the current one |

### wiki — Show wiki capability status

```bash
avmatrix wiki
```

Shows whether the optional wiki capability is `off` or `local`. Remote wiki generation is disabled in local-only mode.

### wiki-mode — Show or set wiki capability mode

```bash
avmatrix wiki-mode
avmatrix wiki-mode off
avmatrix wiki-mode local
```

Controls the optional wiki capability:

| Mode    | Effect |
| ------- | ------ |
| `off`   | Wiki generation is disabled |
| `local` | Reserved for a future local wiki engine; AVmatrix will not fall back to any remote wiki service |

### list — Show all indexed repos

```bash
avmatrix list
```

Lists all repositories registered in the global registry. The MCP `list_repos` tool provides the same information.

## After Indexing

1. **Read `avmatrix://repo/{name}/context`** to verify the index loaded
2. Use the other AVmatrix skills (`exploring`, `debugging`, `impact-analysis`, `refactoring`) for your task

## Troubleshooting

- **"Not inside a git repository"**: Run from a directory inside a git repo
- **Index is stale after re-analyzing**: Restart Claude Code to reload the MCP server
- **Embeddings slow**: Omit `--embeddings` (it's off by default) or set `OPENAI_API_KEY` for faster API-based embedding
