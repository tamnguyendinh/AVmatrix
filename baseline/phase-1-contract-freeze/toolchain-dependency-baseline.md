# Phase 1 Toolchain And Dependency Baseline

Observed: 2026-05-08T20:48:28.4897561+07:00

Purpose: freeze the latest-stable starting point for the Go conversion before any Go module or
parser/persistence implementation is written.

## Selected Toolchain

- Go selected version: `go1.26.3`
- Local Go observed: `go version go1.25.7 windows/amd64`
- Status: `go1.26.3` was installed through `golang.org/dl/go1.26.3` for Phase 1 proof work.
- Source: `https://go.dev/VERSION?m=text`

## Selected Go Runtime Dependencies

| Area | Module | Selected | Evidence |
| --- | --- | --- | --- |
| tree-sitter core | `github.com/tree-sitter/go-tree-sitter` | `v0.25.0` | `go list -m -versions` |
| tree-sitter Go grammar | `github.com/tree-sitter/tree-sitter-go` | `v0.25.0` | GitHub latest release; `go get github.com/tree-sitter/tree-sitter-go/bindings/go@v0.25.0` |
| LadybugDB integration | `github.com/LadybugDB/go-ladybug` | `v0.13.1` | `go list -m -versions` |
| MCP protocol support | `github.com/modelcontextprotocol/go-sdk` | `v1.6.0` | `go list -m -versions` |
| HTTP router | `github.com/go-chi/chi/v5` | `v5.2.5` | `go list -m -versions` |
| SSE runtime | `github.com/r3labs/sse/v2` | `v2.10.0` | `go list -m -versions` |
| Text search runtime | `github.com/blevesearch/bleve/v2` | `v2.6.0` | `go list -m -versions` |
| CLI runtime | `github.com/spf13/cobra` | `v1.10.2` | `go list -m -versions` |
| Config/session runtime | `github.com/spf13/viper` | `v1.21.0` | `go list -m -versions` |
| Launcher/packaging | `github.com/goreleaser/goreleaser/v2` | `v2.15.4` | `go list -m -versions`; nightlies ignored |

## Tree-Sitter Grammar Latest Releases

These are the latest npm releases observed for the current grammar set. The Go parser proof pinned
the Go core binding and Go grammar module to `v0.25.0`.

| Package | Latest |
| --- | --- |
| `tree-sitter` | `0.25.0` |
| `tree-sitter-c` | `0.24.1` |
| `tree-sitter-c-sharp` | `0.23.5` |
| `tree-sitter-cpp` | `0.23.4` |
| `tree-sitter-go` | `0.25.0` |
| `tree-sitter-java` | `0.23.5` |
| `tree-sitter-javascript` | `0.25.0` |
| `tree-sitter-php` | `0.24.2` |
| `tree-sitter-python` | `0.25.0` |
| `tree-sitter-ruby` | `0.23.1` |
| `tree-sitter-rust` | `0.24.0` |
| `tree-sitter-typescript` | `0.23.2` |
| `tree-sitter-kotlin` | `0.3.8` |
| `tree-sitter-swift` | `0.7.1` |
| `tree-sitter-dart` | `1.0.0` |

## TypeScript Runtime Latest Versions For Reference

These are not selected as the future runtime stack, but they document the current source runtime's
latest upstream versions while contract freeze is still reading TypeScript/Node code.

| Package | Latest |
| --- | --- |
| `@ladybugdb/core` | `0.16.1` |
| `@modelcontextprotocol/sdk` | `1.29.0` |
| `express` | `5.2.1` |
| `cors` | `2.8.6` |
| `@huggingface/transformers` | `4.2.0` |
| `onnxruntime-node` | `1.25.1` |
| `graphology` | `0.26.0` |
| `glob` | `13.0.6` |
| `commander` | `14.0.3` |
| `typescript` | `6.0.3` |
| `vitest` | `4.1.5` |

## Exceptions And Mitigation

Checklist mapping: no selected Go runtime dependency is intentionally below the latest observed
stable/tagged release in this baseline. The records below capture the required tested-latest,
failure reason, fallback, and mitigation details for the remaining availability/stability gaps.

- Go toolchain: local `go1.25.7` is not the selected latest stable. Install or activate `go1.26.3`
  before creating the Go module and before running any Go proof/test. Tested latest:
  `go1.26.3`; selected version: `go1.26.3`; fallback: local `go1.25.7` only, not approved
  for Go module work; failure reason: local machine exposes the older toolchain.
- LadybugDB Go integration: latest observed Go binding is pre-v1. Pin `v0.13.1` for feasibility
  work and use the native LadybugDB `v0.16.1` Windows runtime path proven in
  `go-ladybugdb-windows-proof.json`.
  Tested latest: `v0.13.1`; selected version: `v0.13.1`; fallback: none selected; failure reason:
  latest tagged binding is pre-v1.
- tree-sitter grammar Go bindings: pin `github.com/tree-sitter/go-tree-sitter v0.25.0` and
  `github.com/tree-sitter/tree-sitter-go v0.25.0` for the first provider path. Tested latest:
  `v0.25.0`; selected version: `v0.25.0`; fallback: none selected; proof:
  `go-tree-sitter-windows-proof.json`.
