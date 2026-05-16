# Phase 1 Web UI Shared Contract

Observed: 2026-05-08T21:39:40+07:00

This freezes the browser-facing payload contracts used by `avmatrix-web`. The Web UI remains
TypeScript/React. The future Go runtime must preserve these payloads or generate equivalent
TypeScript adapter types for the Web UI.

## Authorities

- Graph payloads: `avmatrix-shared/src/graph/types.ts` and
  `avmatrix-shared/src/lbug/schema-constants.ts`.
- Pipeline progress: `avmatrix-shared/src/pipeline.ts`.
- Session status/events: `avmatrix-shared/src/session.ts`.
- Web client surface: `avmatrix-web/src/services/backend-client.ts`.
- Runtime producers: `avmatrix/src/server/api.ts`, `avmatrix/src/server/session-bridge.ts`, and
  `avmatrix/src/server/analyze-job.ts`.

## Graph Payload

- `GET /api/graph` returns `{ nodes: GraphNode[], relationships: GraphRelationship[] }` in JSON
  mode.
- Stream mode returns NDJSON records:
  - `{ type: "node", data: GraphNode }`
  - `{ type: "relationship", data: GraphRelationship }`
  - `{ type: "error", error: string }`
- `GraphNode`: `id`, `label`, `properties`.
- `GraphRelationship`: `id`, `sourceId`, `targetId`, `type`, `confidence`, `reason`, optional
  `step`, `resolutionSource`, `fileHash`, and `evidence`.
- Runtime relationship contract includes `USES` and `INHERITS`; `DECORATES` remains type-only until
  a runtime emitter or fixture proves it.

## Repos And Jobs

- `GET /api/repos` returns `BackendRepo[]`.
- `GET /api/repo` returns one `BackendRepo` shape, with `repoPath` normalized by the Web client.
- `POST /api/analyze` and `POST /api/embed` return `{ jobId, status }` with HTTP `202`.
- Analyze/embed poll endpoints return `JobStatus`:
  `id`, `status`, `repoPath?`, `repoName?`, `progress`, `error?`, `startedAt`, `completedAt?`.
- Progress SSE sends `JobProgress` on normal messages and terminal `complete` or `failed` events
  with `{ repoName?, repoPath?, error? }`.

## Sessions

- `GET /api/session/status` returns `SessionStatusResponse`.
- `POST /api/session/chat` streams `SessionStreamEvent` as SSE, with event name equal to
  `event.type`.
- Session events are `session_started`, `reasoning`, `content`, `tool_call`, `tool_result`,
  `error`, `cancelled`, and `done`.
- `DELETE /api/session/:sessionId` returns `{ sessionId, status: "cancelled" }`.

## Errors

- General HTTP errors use `{ error: string }`.
- Session bridge errors use `{ code: SessionErrorCode, error: string, details?: unknown }`.
- Graph NDJSON stream errors use `{ type: "error", error: string }`.

## Go Port Rule

The Go backend must preserve these field names, status strings, and SSE event names for the existing
Web UI. Generated TypeScript types are allowed only as Web UI adapter artifacts; the non-Web UI
runtime authority must be Go-owned after cutover.
