# Localhost Launcher Entry Plan

Date: 2026-04-25

## Goal

Make `http://127.0.0.1:5173/` the stable user entry point.

When a user types `127.0.0.1:5173` in the browser, they should see a simple Launcher UI with:

- `Start AVmatrix`
- `Reset`

The graph Web UI must move behind a separate route such as `/app`.

## Non-Goals

- Do not change analyze semantics.
- Do not change graph/query output.
- Do not introduce a large desktop app.
- Do not require Node.js to be visible to the user.
- Do not make `Reset` auto-start.
- Do not rely on users opening multiple terminals.

## Current Problem

The current root HTML flow is confusing:

- `Start-AVmatrix.html` is the visible root entry file.
- Launcher serves Web UI directly at `127.0.0.1:5173/`.
- If backend dies but web remains alive, the browser can show a UI that cannot actually operate.
- Users must remember a file entry point instead of a localhost entry point.

The desired behavior is simpler:

- `127.0.0.1:5173/` is always the launcher screen.
- The actual AVmatrix graph UI lives at `/app`.

## Target Routing

### `/`

Serve Launcher UI.

This page contains:

- `Start AVmatrix`
- `Reset`
- short status text only when useful

### `/app`

Serve the existing React graph Web UI.

All Vite assets must continue to load correctly.

### `/launcher/start`

Local launcher endpoint.

Behavior:

1. Ensure backend server is running.
2. Ensure web app is available.
3. Return success/failure as JSON.
4. Frontend then navigates to `/app`.

### `/launcher/reset`

Local launcher endpoint.

Behavior:

1. Stop AVmatrix runtime/backend/analyze/embedding processes cleanly.
2. Clean runtime/index artifacts according to the clean lifecycle plan.
3. Return success/failure as JSON.
4. Stay on `/`.

Reset must not start AVmatrix.

## UX Flow

### Start

1. User opens `127.0.0.1:5173/`.
2. Launcher UI appears.
3. User clicks `Start AVmatrix`.
4. UI shows `Starting AVmatrix...`.
5. Launcher starts backend if needed.
6. On success, browser navigates to `/app`.

### Reset

1. User opens `127.0.0.1:5173/`.
2. User clicks `Reset`.
3. UI shows `Resetting AVmatrix...`.
4. Launcher stops runtime and cleans artifacts.
5. UI returns to ready state.
6. User can click `Start AVmatrix` again.

## Implementation Plan

### Phase 1: Launcher Routing

Update `.avmatrix-launcher/src/main.go`:

- Serve a minimal launcher page at `/`.
- Serve existing Web UI under `/app`.
- Preserve static asset serving for Vite output.
- Avoid terminal windows by keeping GUI subsystem and hidden child processes.

Key rule:

- `/` must never directly serve the graph app.

### Phase 2: Local Launcher Endpoints

Add local-only endpoints to the launcher web server:

- `POST /launcher/start`
- `POST /launcher/reset`

These endpoints should call internal launcher functions directly instead of going through `avmatrix://`.

This avoids spawning extra protocol handler processes from the browser.

### Phase 3: Root HTML Simplification

Update `Start-AVmatrix.html`:

- It can simply open `http://127.0.0.1:5173/`.
- Or it can remain a fallback shortcut.

The main UX no longer depends on opening this file.

### Phase 4: Validation

Validate manually:

- Open `http://127.0.0.1:5173/` with backend stopped.
- Confirm Launcher UI appears.
- Click `Start AVmatrix`.
- Confirm browser enters `/app`.
- Confirm Web UI can choose/load repo.
- Click `Reset`.
- Confirm it stays at `/`.
- Confirm no terminal window flashes.
- Confirm backend is stopped after reset.
- Confirm clicking Start after reset works.

Validate with commands:

- `.avmatrix-launcher/build.ps1`
- `cd .avmatrix-launcher/src && go build ./...`
- `cd avmatrix-web && npx tsc -b --noEmit`

## Risks

- Vite asset paths may assume `/`.
- Browser refresh on `/app` must still serve React app.
- Reset must not kill the launcher web server handling the reset request before it sends a response.

## Guardrails

- Keep the launcher simple.
- Do not add persistent marker files.
- Do not introduce new analyze behavior.
- Do not change graph app behavior except its served route.
- Do not make Reset start anything.
