# Root HTML Packaged Launcher Plan

Date: 2026-04-25  
Status: Draft  
Scope: root HTML entrypoint, packaged local launcher, bundled backend/static Web UI startup UX

## Goal

Make Web UI startup simple:

```text
double-click Start-AVmatrix.html
-> click Start
-> run packaged AVmatrix launcher
-> open Web UI
```

Add a manual recovery path for stuck local runtime state:

```text
double-click Start-AVmatrix.html
-> click Reset
-> stop old launcher-owned runtime processes
-> clear launcher runtime state
-> start packaged AVmatrix launcher again
-> open Web UI
```

Root user-facing surface:

```text
Start-AVmatrix.html
```

User-facing runtime requirements:

```text
no Node.js install required
no npm command required
no Vite dev server
no visible terminal
no "Open Node.js runtime" browser prompt
```

## Flow

```text
Start-AVmatrix.html
-> Start button
-> avmatrix://start
-> .avmatrix-launcher/AVmatrixLauncher.exe
-> start packaged AVmatrix backend
-> serve bundled Web UI static build
-> open local Web UI URL
```

Reset flow:

```text
Start-AVmatrix.html
-> Reset button
-> avmatrix://reset
-> .avmatrix-launcher/AVmatrixLauncher.exe reset
-> stop owned backend/static runtime processes
-> remove launcher state file
-> start packaged AVmatrix backend again
-> serve bundled Web UI static build
-> open local Web UI URL
```

## Files

```text
Start-AVmatrix.html
.avmatrix-launcher/
  AVmatrixLauncher.exe
  build.ps1
  src/
  web-dist/
  server-bundle/
    avmatrix-server.exe
```

`Start-AVmatrix.html` is the only root file the user should need to see and click.

## Start-AVmatrix.html

Purpose:

- root entry file
- shows one primary `Start AVmatrix` button
- shows one secondary `Reset` button
- button calls `avmatrix://start`
- reset calls `avmatrix://reset`

Behavior:

```text
click Start
-> location.href = "avmatrix://start"

click Reset
-> location.href = "avmatrix://reset"
```

No command instructions as the primary UX.

## Protocol Registration

One-time setup registers:

```text
avmatrix://start
-> .avmatrix-launcher/AVmatrixLauncher.exe start

avmatrix://reset
-> .avmatrix-launcher/AVmatrixLauncher.exe reset
```

Windows first. Other OS support can be added later.

## AVmatrixLauncher.exe

Purpose:

- self-contained launcher, not `node launcher.js`
- no user-installed Node.js requirement
- no npm command requirement
- starts one local app runtime
- serves the Web UI static build
- starts or embeds the AVmatrix backend
- opens the Web UI when ready
- supports reset for stuck server/process state

Runtime shape:

```text
AVmatrixLauncher.exe
-> backend API on localhost
-> static Web UI on localhost
-> open browser
```

Reset behavior:

```text
AVmatrixLauncher.exe reset
-> read launcher state
-> stop recorded backend/runtime PIDs if alive
-> best-effort release owned web/static server
-> remove stale launcher state
-> start clean runtime
-> open browser
```

Dev commands are not user-mode commands:

```text
avmatrix/npm run serve      -> dev only
avmatrix-web/npm run dev    -> dev only
```

## Shutdown

```text
launcher exits
-> stop packaged backend/static server
```

If backend and static server are separate child processes, the launcher owns and kills them.

## Current Implementation Status

Done:

```text
Start-AVmatrix.html
.avmatrix-launcher/AVmatrixLauncher.exe
.avmatrix-launcher/src/
.avmatrix-launcher/build.ps1
.avmatrix-launcher/web-dist/
avmatrix://start protocol registration
```

Current blocker:

```text
none for local smoke
```

Previous observed failure:

```text
Start-AVmatrix.html
-> click Start
-> AVmatrixLauncher.exe starts
-> launcher cannot find server-bundle/avmatrix-server.exe
-> Web UI does not open
```

Implemented fix:

```text
.avmatrix-launcher/server-wrapper/
-> builds .avmatrix-launcher/server-bundle/avmatrix-server.exe
-> bundled node.exe is copied into server-bundle
-> avmatrix-server.exe starts avmatrix/dist/cli/index.js serve with hidden windows
```

Current smoke result:

```text
Start-AVmatrix.html
-> avmatrix://start
-> AVmatrixLauncher.exe
-> server-bundle/avmatrix-server.exe
-> backend ready at localhost:4747/api/info
-> web ready at 127.0.0.1:5173
```

Important note:

- The current backend bundle does not require user-installed Node.js and does not show the browser "Open Node.js runtime" prompt.
- It still ships and runs a bundled `node.exe` internally because the current AVmatrix backend is TypeScript/Node-based.
- A future lower-RAM backend would require a real native backend rewrite or a different runtime packaging strategy.

## Completion Sequence

1. Keep the existing root HTML and packaged Go launcher.
2. Build the launcher and backend wrapper:

```text
powershell -ExecutionPolicy Bypass -File .avmatrix-launcher/build.ps1
```

3. Generated runtime artifacts:

```text
.avmatrix-launcher/AVmatrixLauncher.exe
.avmatrix-launcher/server-bundle/avmatrix-server.exe
.avmatrix-launcher/server-bundle/node.exe
.avmatrix-launcher/web-dist/
```

4. Backend executable behavior:

```text
avmatrix-server.exe
-> starts AVmatrix HTTP API
-> listens on localhost:4747
-> exposes /api/info
```

5. Launcher behavior after backend bundle exists:

```text
Start-AVmatrix.html
-> avmatrix://start
-> AVmatrixLauncher.exe
-> start server-bundle/avmatrix-server.exe
-> serve web-dist on localhost:5173
-> wait for localhost:4747/api/info
-> wait for localhost:5173
-> open localhost:5173
```

6. If the project later requires zero Node runtime even internally, replace `server-wrapper` with a true native backend. Do not expose `node`, `npm run serve`, or `npm run dev` to the user-facing flow.

## Rules

- Do not change analyze logic.
- Do not change graph logic.
- Do not change Web UI graph behavior.
- Reset is a launcher/runtime recovery action only; it must not change analyze, graph, query, or repo-load semantics.
- Root user-facing surface remains `Start-AVmatrix.html`.
- Do not require user-installed Node.js.
- Do not require `npm run serve`.
- Do not require `npm run dev`.
- Do not use Vite dev server for user mode.
- Do not show terminal windows.
- Do not use Electron/Tauri unless the project explicitly decides to ship a desktop app.
- Current implementation may use bundled `node.exe` internally until a native backend exists.

## Validation

- Double-click `Start-AVmatrix.html`.
- Click `Start AVmatrix`.
- Browser calls `avmatrix://start`.
- `AVmatrixLauncher.exe` starts without Node.js prompt.
- Web UI opens.
- Backend API works from Web UI.
- Repeated click does not spawn duplicate app runtimes.
- `Reset` stops stuck launcher-owned runtime processes and starts a clean runtime.
- `Reset` does not impose a global one-repo or one-tab limit.
- Closing launcher stops owned runtime processes.

## Open Points

- Choose packaging technology for `AVmatrixLauncher.exe`.
- Choose whether backend is embedded in the launcher process or shipped as a separate bundled executable.
- Choose final local port strategy.
- Choose one-time protocol registration path for dev and packaged builds.
