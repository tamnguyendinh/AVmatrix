# Phase 1 HTTP Contract

Observed: 2026-05-08T21:18:00+07:00

Source files:

- `avmatrix/src/server/api.ts`
- `avmatrix/src/server/session-bridge.ts`
- `avmatrix/src/server/mcp-http.ts`

## Server Defaults

- Serve CLI default: `localhost:4747`.
- `createServer` default host: `127.0.0.1`.
- Express JSON limit: `10mb`.
- `x-powered-by` is disabled.

## CORS And PNA

Allowed origins are no-origin requests, `localhost`, `127.0.0.1`, and `[::1]` over HTTP or HTTPS,
with or without a port. Disallowed browser origins receive no `Access-Control-Allow-Origin` header
instead of a 500. `Access-Control-Allow-Private-Network: true` is always set for Chromium Private
Network Access.

## SSE

- `GET /api/heartbeat`: sends `:ok` immediately and `:ping` every 15 seconds.
- `GET /api/analyze/:jobId/progress`: sends current progress immediately, `:heartbeat` every 30
  seconds, and terminal `complete` or `failed` events.
- `GET /api/embed/:jobId/progress`: same shared progress contract as analyze.
- `POST /api/session/chat`: streams `SessionStreamEvent` records, heartbeats every 15 seconds, and
  closes on `done`, `error`, or `cancelled`.

## Route Surface

The durable machine-readable route contract is `http-contract.json`. It covers all current HTTP
routes, methods, query/body contracts, success payload shapes, error status families, SSE event
behavior, MCP HTTP session behavior, repo locking, and local-only security guards.

Critical invariants for Go:

- `POST /api/query` must continue to reject write queries with `403`.
- `GET /api/file` and `GET /api/grep` must preserve repo-root path traversal guards.
- Analyze/embed/delete must share the same per-repo lock behavior.
- Remote analyze URLs remain unsupported; `POST /api/analyze` accepts local `path` only.
- Session chat remains SSE and cancellation-on-client-close behavior must be preserved.
- `/api/mcp` remains StreamableHTTP MCP with stateful sessions and 30-minute idle eviction.

