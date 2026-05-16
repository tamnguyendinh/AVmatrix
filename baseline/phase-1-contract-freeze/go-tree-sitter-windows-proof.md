# Go Tree-Sitter Windows Proof

Date: 2026-05-08

Status: PASS

Purpose: prove that the Go rewrite can parse with tree-sitter on Windows before broad provider
work starts.

## Versions

- Go: `go version go1.26.3 windows/amd64`
- Core binding: `github.com/tree-sitter/go-tree-sitter v0.25.0`
- Go grammar: `github.com/tree-sitter/tree-sitter-go v0.25.0`
- Grammar import: `github.com/tree-sitter/tree-sitter-go/bindings/go`

## Proof

The proof program in `.tmp/phase1-go-proofs/tree-sitter` loaded the Go grammar, parsed a Go source
snippet, and validated the resulting AST.

Assertions verified:

- Parser creation succeeded.
- Go grammar loaded through `tree_sitter_go.Language()`.
- Root node is `source_file`.
- Parse tree has no errors.
- A `function_declaration` exists.
- The function name is `add`.
- A `return_statement` exists.

Observed output:

```text
tree-sitter proof ok: root=source_file function=add descendants=29 abi=15
```

Acceptance: Go can use tree-sitter on Windows for provider work.
