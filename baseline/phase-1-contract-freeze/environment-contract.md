# Phase 1 Environment Contract

Observed: 2026-05-08T20:55:00+07:00

This freezes runtime environment variables used by the current TypeScript/Node source so the Go
runtime preserves names, defaults, and parsing behavior.

## Key Runtime Rules

- `AVMATRIX_HOME` overrides the global home directory; default is `<home>/.avmatrix`.
- `AVMATRIX_NO_GITIGNORE` skips `.gitignore` parsing but still reads `.avmatrixignore`.
- `AVMATRIX_MAX_PROCESSES` is a positive integer cap override; fallback is repo settings then `700`.
- `AVMATRIX_SCOPE_RESOLUTION_WORKERS` accepts `auto`, force values `1/true/yes/on/force`, and off
  values `0/false/no/off`; unknown values fail closed to off.
- `AVMATRIX_SCOPE_RESOLUTION_WORKER_COUNT` is used only when it is a positive integer.
- `AVMATRIX_SHADOW_MODE` accepts `true/1/yes`; other values are false.
- `REGISTRY_PRIMARY_<LANG>` is a dynamic per-language flag and accepts `true/1/yes`; other values are
  false.
- HTTP embedding mode requires both `AVMATRIX_EMBEDDING_URL` and `AVMATRIX_EMBEDDING_MODEL`.
- `AVMATRIX_EMBEDDING_DIMS` must be a positive integer and defaults to `384`.
- Local embedding defaults to `Snowflake/snowflake-arctic-embed-xs`, `384` dimensions, and auto
  device selection.
- `ORT_LOG_LEVEL` is set to `3` when unset before loading ONNX Runtime.
- `AVMATRIX_SESSION_EXECUTION_MODE` defaults to `bypass` on Windows and `sandboxed` elsewhere.

## Variable Families

| Family | Variables |
| --- | --- |
| Repo/config | `AVMATRIX_HOME`, `AVMATRIX_NO_GITIGNORE` |
| Analyze/debug | `NODE_OPTIONS`, `AVMATRIX_VERBOSE`, `DEBUG`, `NODE_ENV`, `AVMATRIX_DEBUG` |
| Process detection | `AVMATRIX_MAX_PROCESSES` |
| Scope workers | `AVMATRIX_SCOPE_RESOLUTION_WORKERS`, `AVMATRIX_SCOPE_RESOLUTION_WORKER_COUNT` |
| Scope diagnostics/rollout | `INGESTION_EMIT_SCOPES`, `AVMATRIX_SHADOW_MODE`, `REGISTRY_PRIMARY_<LANG>` |
| Embeddings HTTP | `AVMATRIX_EMBEDDING_URL`, `AVMATRIX_EMBEDDING_MODEL`, `AVMATRIX_EMBEDDING_DIMS`, `AVMATRIX_EMBEDDING_API_KEY` |
| Embeddings local/cache/device | `HF_HOME`, `HOME`, `ORT_LOG_LEVEL`, `CUDA_PATH`, `LD_LIBRARY_PATH` |
| Local session/Codex | `AVMATRIX_CODEX_EXECUTABLE`, `AVMATRIX_SESSION_EXECUTION_MODE`, `ComSpec`, `SystemRoot`, `windir` |
| Wiki/LLM | `AVMATRIX_API_KEY`, `OPENAI_API_KEY`, `AVMATRIX_LLM_BASE_URL`, `AVMATRIX_MODEL`, `AVMATRIX_AZURE_API_VERSION` |
| MCP impact/server info | `IMPACT_MAX_CHUNKS`, `npm_execpath`, `npm_config_prefix` |

## Conversion Rule

The Go runtime must preserve these environment variable names and fail-close parsing behavior unless
a later explicit migration contract is written. Secret values are not captured; only variable names,
defaults, parsing rules, and runtime effects are frozen.

