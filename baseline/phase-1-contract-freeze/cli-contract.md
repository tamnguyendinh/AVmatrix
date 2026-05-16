# Phase 1 CLI Contract

Observed: 2026-05-08T21:05:00+07:00

Build gate before CLI snapshots: `cd avmatrix && npm run build` passed.

Temporary output files were written under `.tmp/phase1_cli_*` and are intentionally not committed.
The durable contract is this file plus `cli-contract.json`.

## Commands

Top-level command surface:

- `setup`
- `analyze [path]`
- `benchmark-compare <before> <after>`
- `index [path...]`
- `serve`
- `mcp`
- `list`
- `status`
- `clean`
- `wiki [path]`
- `wiki-mode [mode]`
- `augment <pattern>`
- `query <search_query>`
- `context [name]`
- `impact [target]`
- `cypher <query>`
- `detect-changes`
- `group`

Group subcommands:

- `create <name>`
- `add <group> <groupPath> <registryName>`
- `remove <group> <path>`
- `list [name]`
- `status <name>`
- `sync <name>`
- `query <name> <query>`
- `contracts <name>`

## Important Defaults

- `serve --port` default: `4747`.
- `serve --host` runtime default: `localhost`; non-loopback hosts are rejected.
- `impact --direction` default: `upstream`.
- `detect-changes --scope` default: `unstaged`.
- `group query --limit` default: `5`.
- `clean` and `clean --all` do not prompt interactively; without `--force` they print the planned
  deletion target and exit without deleting.
- `wiki` returns the local-only disabled status and sets exit code `1`.

## Expected Output Snapshots

Root help begins with:

```text
Usage: avmatrix [options] [command]
```

Analyze help includes:

```text
Environment variables:
  AVMATRIX_NO_GITIGNORE=1  Skip .gitignore parsing (still reads the local ignore override file)
  AVMATRIX_MAX_PROCESSES=700  Temporarily override maxExecutionFlows from .avmatrix/settings.json in the target repo during analyze
```

Bad serve host output:

```text
Failed to start AVmatrix server:

  Local-only mode only allows loopback hosts: localhost, 127.0.0.1, or ::1.
```

Missing impact target output:

```text
Usage: avmatrix impact [symbol_name] [--uid <uid>] [--direction upstream|downstream]
```

Wiki disabled output begins with:

```text
Wiki capability mode: off
```

Clean without force output ends with:

```text
Run with --force to confirm deletion.
```

## Exit Codes

- Help output: `0`.
- `avmatrix wiki`: `1` while local wiki is disabled.
- `avmatrix serve --host 0.0.0.0`: `1`.
- `avmatrix impact` with neither target nor `--uid`: `1`.
- `avmatrix clean` without `--force`: `0`, no deletion.
- Direct tool commands exit `1` after validation when no indexed repositories exist.

## Environment Reference

CLI env vars are frozen in `baseline/phase-1-contract-freeze/environment-contract.json`.

