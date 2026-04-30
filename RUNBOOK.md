# Runbook - AVmatrix

Copy-paste operations for the current local AVmatrix runtime: CLI, MCP, Web UI, packaged launcher, repo indexes, and recovery.

Current CLI package version: `1.2.1` (`avmatrix/package.json`).

---

## Prerequisites

Required for normal local development:

- Node.js 20+
- npm
- Git

Required only for the packaged Windows launcher:

- Go
- PowerShell

Build and link the local CLI from source:

```powershell
cd avmatrix-shared
npm install
npm run build

cd ..\avmatrix
npm install
npm run build
npm link

avmatrix --version
```

If you are using Codex CLI or Claude Code inside this repository, you can also ask the agent:

```text
Install AVmatrix from this repository and configure its MCP integration.
```

---

## Index A Repo

From the target repo:

```powershell
avmatrix analyze .
```

Force a full rebuild:

```powershell
avmatrix analyze . --force
```

Analyze a folder without requiring `.git`:

```powershell
avmatrix analyze C:\path\to\repo --skip-git
```

Register the repo under a custom name:

```powershell
avmatrix analyze C:\path\to\repo --name MyRepo
```

Check what AVmatrix knows:

```powershell
avmatrix status
avmatrix list
```

Index data is stored in the repo:

```text
<repo>\.avmatrix\
  lbug
  lbug.wal
  lbug.lock
  meta.json
  settings.json
```

The global registry is stored at:

```text
~\.avmatrix\registry.json
```

---

## Stale Index

Symptom:

- MCP/resources warn the index is behind `HEAD`
- graph/query results do not match recent code

Fix:

```powershell
cd C:\path\to\repo
avmatrix analyze .
```

If the index may be corrupt or the schema/runtime changed:

```powershell
avmatrix analyze . --force
```

---

## MCP Setup And Recovery

Configure detected editors/agents:

```powershell
avmatrix setup
```

Manual examples:

```powershell
claude mcp add avmatrix -- avmatrix mcp
codex mcp add avmatrix -- avmatrix mcp
```

Codex TOML:

```toml
[mcp_servers.avmatrix]
command = "avmatrix"
args = ["mcp"]
```

Start MCP manually:

```powershell
avmatrix mcp
```

If MCP says no repos are indexed:

```powershell
cd C:\path\to\repo
avmatrix analyze .
```

If multiple repos are indexed, pass `repo` explicitly in MCP tool calls or call `list_repos` first.

---

## Web UI Local Backend

Start the HTTP backend:

```powershell
cd avmatrix
node dist\cli\index.js serve
```

Or, after `npm link`:

```powershell
avmatrix serve
```

Default backend:

```text
http://localhost:4747
```

Health checks:

```powershell
Invoke-WebRequest http://localhost:4747/api/info
Invoke-WebRequest http://localhost:4747/api/repos
```

Start the Web UI dev server:

```powershell
cd avmatrix-web
npm install
npm run dev
```

Open:

```text
http://localhost:5173
```

Important behavior:

- `avmatrix serve` is loopback-only by default.
- Web graph loading uses explicit repo-scoped read targets.
- The graph switch path is `/api/graph?repo=...&stream=true`.
- The Web UI is a frontend over the local backend, not a separate index/runtime.

---

## Web UI Troubleshooting

### Failed to fetch / backend disconnected

Check backend:

```powershell
Invoke-WebRequest http://localhost:4747/api/info
```

If the port is down, restart:

```powershell
cd avmatrix
node dist\cli\index.js serve
```

If port `4747` is already in use:

```powershell
avmatrix serve --port 4748
```

The default Web UI expects `4747`; use the packaged/default path unless you are deliberately testing another backend URL.

### Loading graph hangs

Check the target repo is registered:

```powershell
avmatrix list
```

Check the graph endpoint directly:

```powershell
Invoke-WebRequest "http://localhost:4747/api/graph?repo=MyRepo&stream=true"
```

If the response contains LadybugDB/WAL errors, stop active runtimes and rebuild the repo index.

### Newly analyzed repo does not appear

Check the registry response:

```powershell
Invoke-WebRequest http://localhost:4747/api/repos
```

If the repo exists in CLI but not the Web UI, refresh the browser. If needed, restart `avmatrix serve`.

---

## Packaged Windows Launcher

Build the full packaged local runtime:

```powershell
powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1
```

Artifacts:

```text
Start-AVmatrix.html
avmatrix-launcher\AVmatrixLauncher.exe
avmatrix-launcher\server-bundle\avmatrix-server.exe
avmatrix-launcher\server-bundle\node.exe
avmatrix-launcher\web-dist\
```

Start:

```text
Open Start-AVmatrix.html
```

Reset runtime:

```powershell
.\avmatrix-launcher\AVmatrixLauncher.exe reset
```

Stop runtime:

```powershell
.\avmatrix-launcher\AVmatrixLauncher.exe stop
```

Register protocol:

```powershell
.\avmatrix-launcher\AVmatrixLauncher.exe register
```

Logs:

```text
avmatrix-launcher\logs\launcher.log
avmatrix-launcher\logs\backend.log
avmatrix-launcher\logs\server-wrapper.log
```

The launcher is optional. It starts the same local backend that `avmatrix serve` starts directly.

---

## Session Chat Runtime

The Web chat does not run an AI model inside AVmatrix. It sends chat requests through the local session bridge and currently executes them with the Codex CLI session available on this machine.

Implemented chat provider:

```text
codex -> Codex CLI adapter
```

Reserved provider identifier:

```text
claude-code -> UI/status slot only until a backend adapter is added
```

Check Codex locally:

```powershell
codex --version
codex login status
```

If chat is unavailable:

- verify the local provider CLI is installed
- verify the provider account is signed in
- verify the selected repo is indexed
- check `http://localhost:4747/api/session/status`

AVmatrix does not store provider API keys in the browser and does not route chat through an AVmatrix cloud service.

---

## Embeddings

Generate embeddings during full analyze:

```powershell
avmatrix analyze . --embeddings
```

If you want to preserve/regenerate embeddings on future rebuilds, keep passing `--embeddings`.

Current behavior:

- embeddings are optional
- embeddings are stored in the `CodeEmbedding` table
- embedding generation is skipped for repos over `100000` nodes
- search still works without embeddings through BM25/FTS paths

Backend embedding endpoint:

```text
POST /api/embed
GET /api/embed/:jobId
GET /api/embed/:jobId/progress
DELETE /api/embed/:jobId
```

Analyze and embed jobs are locked per repo to avoid concurrent writes to the same LadybugDB store.

---

## Clean And Recover A Repo Index

Current repo:

```powershell
avmatrix clean --force
```

All indexed repos:

```powershell
avmatrix clean --all --force
```

Then rebuild:

```powershell
avmatrix analyze . --force
```

If you see WAL/checksum corruption:

1. Stop the packaged launcher or `avmatrix serve`.
2. Stop MCP sessions that may have the repo open.
3. Clean the repo index.
4. Re-run `avmatrix analyze . --force`.

Do not manually delete `.avmatrix\lbug*` while launcher/backend/MCP is running.

---

## Lock And Concurrent Job Errors

Symptom:

```text
Another job is already active for this repository
```

Meaning:

- analyze, delete, or embed is already active for that repo
- the backend lock is protecting the repo-local LadybugDB store

Fix:

- wait for the current job to finish
- cancel the Web UI job if available
- restart the local backend if the job is stale

---

## CLI Equivalents Of MCP Tools

Useful when debugging without an editor:

```powershell
avmatrix query "authentication flow" --repo MyRepo
avmatrix context SomeSymbol --repo MyRepo
avmatrix impact SomeSymbol --direction upstream --repo MyRepo
avmatrix cypher "MATCH (n) RETURN count(n) LIMIT 1" --repo MyRepo
avmatrix detect-changes --repo MyRepo
```

---

## Wiki Capability

The active local-only build has no remote wiki fallback.

```powershell
avmatrix wiki
avmatrix wiki-mode
avmatrix wiki-mode off
avmatrix wiki-mode local
```

`wiki-mode local` is reserved for a future local wiki engine.

---

## Docker

Docker is an advanced deployment path, separate from the primary local CLI/launcher flow.

Required files:

```text
Dockerfile.cli
Dockerfile.web
docker-compose.yaml
.env.example
```

Run:

```powershell
Copy-Item .env.example .env
# edit SERVER_IMAGE and WEB_IMAGE
docker compose --env-file .env up -d
```

Default ports:

```text
server: http://localhost:4747
web:    http://localhost:4173
```

To analyze host repos inside Docker, set `WORKSPACE_DIR` to a local folder that contains those repos. It is mounted read-only at `/workspace`.

---

## Build And Validation Commands

Core:

```powershell
cd avmatrix
npm run build
npm test
npx tsc --noEmit
```

Web:

```powershell
cd avmatrix-web
npm run build
npm test
```

Web e2e requires both backend and frontend running:

```powershell
cd avmatrix
avmatrix serve

cd ..\avmatrix-web
npm run dev
npm run test:e2e
```

Launcher package:

```powershell
powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1
```

---

## Where To Dig Deeper

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [README.md](README.md)
- [CHANGELOG.md](CHANGELOG.md)
- [GUARDRAILS.md](GUARDRAILS.md)
- [TESTING.md](TESTING.md)
