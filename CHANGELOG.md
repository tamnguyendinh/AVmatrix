# Changelog

All notable changes to avmatrix will be documented in this file.

## [Unreleased]

### 2026-04-30

#### Changed

- Updated `ARCHITECTURE.md` to match the current codebase: CLI/MCP/HTTP backend, Web UI, shared contracts, Windows local launcher, repo-scoped graph reads, and the Codex/Claude Code session bridge contract.
- Polished the Web UI header, repo dropdown placement, graph loading copy, runtime controls, chat panel spacing, and status/footer presentation.
- Simplified the AI runtime settings panel around the actual local session state instead of legacy provider/model configuration.
- Added global button press states and refined button borders while keeping the repo tree and repo dropdown visually lighter.

#### Fixed

- Unified external graph node selection so choosing a file from the dashboard or selecting a result from search follows the same graph selection/display path as clicking a node in the graph canvas.

### 2026-04-29

#### Added

- Added the repo-scoped HTTP read engine for repo switching and graph loading.
- Added shared repo resolution/runtime ID helpers, a repo-scoped read executor, a graph read/stream service, and a graph streaming HTTP adapter.
- Added focused unit coverage for repo resolution, repo read execution, graph streaming, MCP alignment, and repeated dropdown repo-switching.
- Added an OS-backed local folder picker for analyzing local repositories from the Web UI.
- Added repo removal actions on the repository landing screen.
- Added a placeholder `user_guide.md` entry and removed the placeholder logo from active UI surfaces.

#### Changed

- Migrated `/api/graph`, `/api/query`, `/api/search`, and `/api/grep` away from the old process-global LadybugDB retargeting path for repo-switch graph-load scope.
- Kept AVmatrix local-only: no cloud control plane, no managed workspace import model, and no required always-on daemon.
- Refreshed the repository list after analyzing a repository from the dropdown flow.
- Reworked loading UI copy and removed the fake hard-coded graph download percentage.

#### Fixed

- Fixed the Web UI dropdown repo-switch failure path by replacing the broken global DB switch/read path with explicit repo-scoped graph reads.
- Fixed dropdown analyze flow so newly analyzed repositories appear in the repository list.

#### Release Notes

- Marked the repo-switch graph engine plan complete for the graph-load scope.
- Bumped the CLI package version to mark the repo-switch engine completion point.

### 2026-04-25

#### Added

- Added the packaged root HTML launcher flow for Windows.
- Added `Start-AVmatrix.html`, `AVmatrixLauncher.exe`, the Go launcher source, packaged Web UI output, backend server wrapper, and protocol registration for `avmatrix://start`.
- Added launcher reset/start/stop behavior around the same local backend runtime.

#### Changed

- Moved launcher sources and build output from hidden `.avmatrix-launcher/` to visible `avmatrix-launcher/`.
- Updated launcher path references, protocol registration, and build packaging after the directory move.

#### Documented

- Added local launcher entry and root HTML launcher plans.
- Documented reset/analyze lifecycle risks around stale runtime processes, repo locks, WAL corruption, and clean rebuild boundaries.
- Paused the full analyze optimization plan after the delivered speedup and measured LadybugDB/FTS bottleneck.

### 2026-04-24

#### Performance

- Added full analyze phase timing and deep persistence instrumentation.
- Made the parse worker path canonical for full analyze and removed the old silent whole-repo sequential fallback behavior from the optimized production path.
- Implemented a deterministic dynamic parse worker scheduler.
- Reduced large-repo parse time on the benchmark path from the prior sequential-fallback profile to the worker-canonical profile while preserving output counts.
- Cached cross-file parser query work after measuring crossFile internals.

#### Fixed

- Fixed Go receiver source attribution during crossFile reprocessing so receiver method calls are not incorrectly attributed to same-name interface methods.
- Made community detection deterministic by fixing randomization/input ordering behavior.

#### Documented

- Recorded full analyze Phase 0 through Phase 4 findings.
- Closed Phase 2 without resolver/linker optimization after measurement showed parse main-thread resolve was not a material bottleneck.
- Paused deeper LadybugDB/FTS optimization until there is a safer strategy that preserves stored graph/search behavior exactly.
- Added the Web graph load/render performance plan as a separate concern from full analyze performance.

### 2026-04-23

#### Added

- Added repo-local execution flow settings.
- Added a master graph links visibility toggle for the Web UI.

#### Changed

- Refined the Web UI shell and controls toward the `The Press` editorial design direction while keeping the graph workspace dark.
- Switched the graph viewport to black and refined graph rendering/selection behavior.
- Reordered graph guidance tools and aligned package metadata/docs with the current AVmatrix product surface.

#### Fixed

- Unblocked graph stream startup.
- Hardened the `impact` tool contract direction with schema/runtime alignment planning and implementation.

#### Documented

- Added plans for impact contract hardening, smart re-analyze delta graph constraints, graph links behavior, selected graph context, layout/scroll/resize hardening, click-node performance hardening, straight edges/thinner links, and the Web UI `The Press` migration.

### 2026-04-22

#### Performance

- Improved MCP startup performance for stdio clients.

#### Added

- Added configurable process flow caps.

#### Changed

- Updated AI context generation and README content for the local AVmatrix runtime direction.

#### Documented

- Added MCP startup optimization planning.
- Added Web UI Codex chat runtime optimization planning.

### 2026-04-21

#### Added

- Added a direct `detect-changes` CLI command.

#### Changed

- Completed the AVmatrix rename rollout across user-facing surfaces, CLI/package metadata, MCP integration, docs, setup, and active Web UI paths.
- Removed SWE-bench, sponsor, eval-server, and remote/eval product surfaces from the active local tool.
- Separated chat runtime state from graph state in the Web UI.
- Retired the legacy Web agent build path.
- Completed local runtime cleanup phases: provider/API-key paths were removed or wrapped out of the active build, and the Web runtime moved to the local session model.

#### Fixed

- Restored local Codex chat and portable onboarding.
- Hardened local runtime chat and WAL recovery.
- Removed stale clone states from local analyze flow.
- Fixed grounding links mock typing.

#### Documented

- Added and refined the local-only runtime migration plan.
- Added AVmatrix rename rollout planning.
- Added chat/graph separation reviews and supervisor review notes.
- Restored license notice and refreshed agent context/guidance.

### 2026-04-20

#### Added

- Added Phase 1 of the local session runtime bridge.
- Added HTTP session bridge support for local chat sessions, including status, streaming chat, cancellation, and indexed-repo binding.

#### Changed

- Migrated Web/local runtime behavior to the session bridge.
- Locked onboarding and CLI runtime behavior to the local-only flow.
- Enforced local path analyze semantics and began removing clone/remote URL assumptions from the active runtime path.

#### Fixed

- Hardened Phase 1 runtime repo resolution.
- Closed Phase 1 session runtime gaps.
- Enforced the WSL2 default recommendation for full Codex runtime mode.

#### Documented

- Refined the local runtime migration plan, preserved feature parity expectations, and gated wiki behavior for local runtime.
- Snapshotted the repository state before the local runtime migration.

### 2026-03-26

#### Planned

- Documented the complete COBOL language coverage plan, including high-value data-flow edges, EXEC SQL/DLI handling, DECLARATIVES support, CALL USING/RETURNING extraction, STRING/UNSTRING/INSPECT/SET/INITIALIZE access edges, and metadata completeness fixes.
