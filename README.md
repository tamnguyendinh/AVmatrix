# AVmatrix

Current CLI package version: `1.2.1` (`avmatrix/package.json`).

**Important notice:** AVmatrix has no official cryptocurrency, token, or coin. Any token/coin using the AVmatrix name is not affiliated with, endorsed by, or created by this project or its maintainers.

AVmatrix indexes a local codebase into a knowledge graph, then exposes that graph to AI coding agents, CLI commands, and a local Web UI.

The core product is still local code intelligence:

- `avmatrix analyze` builds a repo-local graph index in `<repo>/.avmatrix/`.
- `avmatrix mcp` exposes indexed repos to MCP clients such as Claude Code, Codex, Cursor, and OpenCode.
- `avmatrix serve` exposes the same local runtime over HTTP for the browser UI.
- `avmatrix-launcher/` packages the local backend and Web UI for a Windows `Start-AVmatrix.html` flow.

No AVmatrix-hosted cloud service is involved in the active local runtime path.

You do not need to put API keys into AVmatrix. Indexing, graph storage, repo switching, and graph queries run locally on your machine. For chat, AVmatrix uses the local Codex or Claude Code session/account you already use on this machine; AVmatrix does not store provider keys or route chat through an AVmatrix cloud service.

---

## Current Runtime Model

| Surface | Purpose | Entry point |
|---------|---------|-------------|
| CLI | Analyze repos, query the graph, inspect impact, manage indexes/groups | `avmatrix ...` |
| MCP stdio | Agent-facing graph tools and resources | `avmatrix mcp` |
| Local HTTP API | Web UI backend, graph streaming, analyze jobs, session bridge | `avmatrix serve` |
| Web UI | Browser graph explorer, repo picker/analyze UI, Codex/Claude Code style session chat | `avmatrix-web/` or packaged launcher |
| Windows launcher | Starts packaged Web UI and backend on localhost | `Start-AVmatrix.html`, `avmatrix://start` |

The Web UI is a frontend over the local HTTP backend. Repo switching and graph loading use explicit repo-scoped read targets; they do not depend on one mutable process-global active repo.

The Web chat does not run an AI model inside AVmatrix. The shared session contract supports `codex` and `claude-code`; the current backend mounts the Codex CLI adapter. AVmatrix keeps repo binding, streaming, cancellation, and UI state local.

---

## Quick Start: CLI + MCP

Requires Node.js 20+.

### Agent-assisted quick install

1. Download or clone AVmatrix from GitHub.
2. Open Codex CLI or Claude Code in the folder that contains the AVmatrix repo, or inside the AVmatrix repo itself.
3. Paste this prompt:

```text
"Install AVmatrix from this repository and configure its MCP integration."
```

The agent should install dependencies, build `avmatrix-shared`, install/link the local `avmatrix` CLI, run `avmatrix setup`, and verify `avmatrix --version`.

### Manual install

```bash
git clone https://github.com/tamnguyendinh/AVmatrix.git
cd AVmatrix

cd avmatrix-shared
npm install
npm run build

cd ../avmatrix
npm install
npm link

avmatrix --version
```

Index a local repository:

```bash
cd /path/to/your/repo
avmatrix analyze .
```

This creates `<repo>/.avmatrix/` and registers the repo in `~/.avmatrix/registry.json`.

Configure MCP/editor integration:

```bash
avmatrix setup
```

Manual MCP examples:

```bash
claude mcp add avmatrix -- avmatrix mcp
codex mcp add avmatrix -- avmatrix mcp
```

Codex TOML:

```toml
[mcp_servers.avmatrix]
command = "avmatrix"
args = ["mcp"]
```

---

## Quick Start: Web UI

Development flow:

```bash
# one-time source setup if you have not already built the CLI
cd AVmatrix
cd avmatrix-shared
npm install
npm run build

cd ../avmatrix
npm install
cd ..

# terminal 1, from the repo root
cd avmatrix
node dist/cli/index.js serve
# or, after npm link:
avmatrix serve

# terminal 2, from the repo root
cd avmatrix-web
npm install
npm run dev
```

Open:

```text
http://localhost:5173
```

The browser connects to:

```text
http://localhost:4747
```

From the Web UI you can:

- choose an indexed local repo
- analyze another local repo
- remove a repo from the landing list
- switch repos from the header dropdown
- browse graph nodes, links, files, processes, and search results
- use the local session bridge for Codex/Claude Code style chat

---

## Packaged Windows Launcher

The packaged launcher is a convenience layer around the same local backend and Web UI.

Build it:

```powershell
powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1
```

Important artifacts:

```text
Start-AVmatrix.html
avmatrix-launcher\AVmatrixLauncher.exe
avmatrix-launcher\server-bundle\avmatrix-server.exe
avmatrix-launcher\server-bundle\node.exe
avmatrix-launcher\web-dist\
```

Runtime behavior:

- `Start-AVmatrix.html` launches `avmatrix://start`.
- `AVmatrixLauncher.exe` serves the packaged Web UI on `127.0.0.1:5173`.
- `avmatrix-server.exe` starts the bundled Node backend and runs `avmatrix serve`.
- backend health is checked at `http://localhost:4747/api/info`.
- reset/stop use the launcher state file plus process path sweep for the packaged runtime.

The launcher must remain optional. `avmatrix serve` is still the direct backend entry point.

---

## Main CLI Commands

```bash
avmatrix setup                     # Configure local MCP/editor access
avmatrix analyze [path]            # Full local repo analysis
avmatrix analyze --force           # Force full re-index
avmatrix analyze --embeddings      # Generate semantic embeddings
avmatrix analyze --skills          # Generate repo-specific agent skills
avmatrix analyze --skip-agents-md  # Skip AGENTS.md / CLAUDE.md block updates
avmatrix analyze --no-stats        # Omit volatile stats from generated agent files
avmatrix analyze --skip-git        # Analyze a folder without requiring .git
avmatrix analyze --name <alias>    # Register repo under a custom name
avmatrix index [path...]           # Register an existing local index
avmatrix list                      # List indexed repos
avmatrix status                    # Show index status for current repo
avmatrix clean                     # Delete current repo index
avmatrix clean --all --force       # Delete all indexes
avmatrix mcp                       # Start MCP server over stdio
avmatrix serve                     # Start local HTTP backend on localhost:4747
avmatrix wiki                      # Show wiki capability status
avmatrix wiki-mode [off|local]     # Show or set local wiki capability mode
```

Direct graph tools:

```bash
avmatrix query <search_query>
avmatrix context [name]
avmatrix impact [target]
avmatrix cypher <query>
avmatrix detect-changes
```

Repository groups:

```bash
avmatrix group create <name>
avmatrix group add <group> <groupPath> <registryName>
avmatrix group remove <group> <path>
avmatrix group list [name]
avmatrix group sync <name>
avmatrix group contracts <name>
avmatrix group query <name> <query>
avmatrix group status <name>
```

Repo-local settings live in `.avmatrix/settings.json`. The current repo-local setting is `maxExecutionFlows`, which controls the cap used while materializing execution flows during `analyze`. `AVMATRIX_MAX_PROCESSES` remains available as a temporary override.

---

## MCP Tools And Resources

AVmatrix exposes 16 MCP tools:

| Tool | Purpose |
|------|---------|
| `list_repos` | Discover indexed repos |
| `query` | Hybrid BM25 + semantic search over execution flows and symbols |
| `cypher` | Raw Cypher against the graph |
| `context` | 360-degree symbol context |
| `detect_changes` | Map git diffs to affected symbols/processes |
| `rename` | Graph-assisted multi-file rename preview/application |
| `impact` | Upstream/downstream blast radius |
| `route_map` | API route to handler/consumer mapping |
| `tool_map` | MCP/RPC tool definition and handler mapping |
| `shape_check` | API response shape vs consumer access checks |
| `api_impact` | API route pre-change impact report |
| `group_list` | List repo groups |
| `group_sync` | Build cross-repo contract registry |
| `group_contracts` | Inspect group contracts and cross-links |
| `group_query` | Search across a repo group |
| `group_status` | Check group/repo staleness |

Common resources:

| Resource | Purpose |
|----------|---------|
| `avmatrix://repos` | All indexed repos |
| `avmatrix://setup` | Generated setup/onboarding content |
| `avmatrix://repo/{name}/context` | Repo overview and stats |
| `avmatrix://repo/{name}/clusters` | Functional clusters |
| `avmatrix://repo/{name}/cluster/{name}` | Cluster detail |
| `avmatrix://repo/{name}/processes` | Execution flows |
| `avmatrix://repo/{name}/process/{name}` | Process trace |
| `avmatrix://repo/{name}/schema` | Graph schema |

MCP prompts:

| Prompt | Purpose |
|--------|---------|
| `detect_impact` | Pre-change impact workflow |
| `generate_map` | Architecture map from graph context |

When only one repo is indexed, most repo-scoped tool calls can omit `repo`. With multiple indexed repos, pass the repo name or path explicitly.

---

## How Indexing Works

`avmatrix analyze` runs a full local pipeline:

```text
scan -> structure -> [markdown, cobol] -> parse -> [routes, tools, orm]
  -> crossFile -> mro -> communities -> processes
  -> LadybugDB load -> FTS -> optional embeddings -> metadata/registry/agent files
```

The graph is stored in LadybugDB under `<repo>/.avmatrix/`.

Storage:

```text
<repo>/.avmatrix/
  lbug
  lbug.wal
  lbug.lock
  meta.json
  settings.json

~/.avmatrix/
  registry.json
```

Supported language detection currently covers:

```text
JavaScript, TypeScript, Python, Java, C, C++, C#, Go, Ruby, Rust,
PHP, Kotlin, Swift, Dart, Vue, COBOL
```

COBOL/JCL is handled through the dedicated COBOL phase rather than the normal tree-sitter worker path.

---

## Local HTTP API

`avmatrix serve` exposes the local backend used by the Web UI:

| Endpoint | Purpose |
|----------|---------|
| `/api/info`, `/api/heartbeat` | Backend liveness |
| `/api/repos`, `/api/repo` | List/select/remove indexed repos |
| `/api/graph` | Repo-scoped graph load/stream |
| `/api/query`, `/api/search`, `/api/file`, `/api/grep` | Repo-scoped read/search helpers |
| `/api/process*`, `/api/cluster*` | Derived graph views |
| `/api/local/folder-picker` | Native local folder picker bridge |
| `/api/analyze`, `/api/embed` | Background analyze/embed jobs |
| `/api/mcp` | MCP-over-HTTP bridge |
| `/api/session/*` | Session bridge for chat runtime |

The graph loading path uses:

```text
repo-resolver -> repo-read-executor -> graph-read-service -> graph-stream-http
```

That is the replacement for the older process-global DB retargeting path used during Web repo switching.

---

## Docker

Docker support remains available as an advanced deployment path. It is separate from the primary local CLI/launcher flow.

Files:

- `Dockerfile.cli`
- `Dockerfile.web`
- `docker-compose.yaml`
- `.env.example`

Compose expects exact image tags:

```bash
cp .env.example .env
# set SERVER_IMAGE and WEB_IMAGE
docker compose --env-file .env up -d
```

Default ports:

```text
server: http://localhost:4747
web:    http://localhost:4173
```

To make host repos visible to the container, set `WORKSPACE_DIR` to a local folder that contains the repos you want to analyze. It is mounted read-only at `/workspace`.

---

## Repository Layout

| Path | Role |
|------|------|
| `avmatrix/` | CLI, MCP server, HTTP API, ingestion, LadybugDB, embeddings, session bridge |
| `avmatrix-web/` | React/Vite Web UI and local runtime client |
| `avmatrix-shared/` | Shared types, graph constants, language detection, session contracts |
| `avmatrix-launcher/` | Windows launcher, server wrapper, packaged Web UI/backend assets |
| `.claude/`, `avmatrix-claude-plugin/`, `avmatrix-cursor-integration/` | Agent skills and plugin metadata |
| `docs/plans/` | Implementation plans and investigation records |
| `.github/` | CI workflows |

See [ARCHITECTURE.md](ARCHITECTURE.md) for the detailed system map.

---

## Development

Build core packages:

```bash
cd avmatrix-shared
npm install
npm run build

cd ../avmatrix
npm install
npm run build
```

Build Web UI:

```bash
cd avmatrix-web
npm install
npm run build
```

Build full Windows launcher package:

```powershell
powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1
```

Useful docs:

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [CHANGELOG.md](CHANGELOG.md)
- [RUNBOOK.md](RUNBOOK.md)
- [GUARDRAILS.md](GUARDRAILS.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [TESTING.md](TESTING.md)

---

## Security And Privacy

- Index data is stored locally in `<repo>/.avmatrix/`.
- The global registry is local under `~/.avmatrix/`.
- The Web UI talks to the local backend at `localhost:4747`.
- AVmatrix does not store AI provider API keys in the browser.
- AVmatrix does not route chat through an AVmatrix cloud service.
- Codex/Claude Code style chat depends on the local session/provider already available on the machine.

---

## Acknowledgments

- [Model Context Protocol](https://modelcontextprotocol.io/)
- [Tree-sitter](https://tree-sitter.github.io/)
- [LadybugDB](https://ladybugdb.com/)
- [Sigma.js](https://www.sigmajs.org/)
- [Graphology](https://graphology.github.io/)
- [Transformers.js](https://huggingface.co/docs/transformers.js)
