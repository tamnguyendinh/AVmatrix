# Phase 1 Go-Owned Web Adapter Type Decision

Observed: 2026-05-08T21:41:27+07:00

Decision: after conversion, Go owns the Web UI-facing contract types. TypeScript contract files used
by `avmatrix-web` are generated browser-side adapter artifacts only.

## Authority

- Future Go authority package: `internal/contracts`.
- Go owns graph payload structs, schema constants, HTTP request/response payloads, analyze/embed
  job status/progress, session status/events, repo list/info payloads, and error envelopes.
- `avmatrix-shared` TypeScript is baseline input only. It must not remain the runtime contract
  authority after Go cutover.

## Generated Artifacts

- Schema manifest: `contracts/web-ui/avmatrix-web-contract.schema.json`.
- Web UI TypeScript adapter: `avmatrix-web/src/generated/avmatrix-contracts.ts`.
- Generated TypeScript exists only to compile the existing browser Web UI.

## Rules

- Go structs and constants are the source of truth.
- Generate the schema manifest from Go contracts.
- Generate the TypeScript adapter from the schema manifest.
- Generated files must include a generated-file header and must not be manually edited.
- CI must verify generated files are current before cutover.
- Generated Web UI types must not import Node-only packages.

## Compatibility

- Generated schema must remain compatible with
  `baseline/phase-1-contract-freeze/web-ui-shared-contract.json`.
- Field names, JSON casing, status strings, and SSE event names must stay stable unless a recorded
  compatibility migration exists.
- General HTTP errors remain `{ error: string }`.
- Session errors remain `{ code, error, details? }`.
- Graph JSON and NDJSON modes both remain supported.
