# Phase 1 LadybugDB And Graph Contract

Observed: 2026-05-08T21:38:00+07:00

Verification:

- `powershell -ExecutionPolicy Bypass -File avmatrix-launcher\build.ps1` passed.
- `cd avmatrix && npx vitest run test/unit/schema.test.ts test/unit/security.test.ts test/unit/impact-confidence.test.ts test/unit/impact-contract.test.ts test/unit/contract-freeze/phase1-contract-snapshot.test.ts test/integration/pipeline-graph-golden.test.ts` passed.
- `cd avmatrix-web && npx vitest run test/unit/security-guards.test.ts` passed.

## LadybugDB Contract

- Node tables are the `NODE_TABLES` set from `avmatrix-shared/src/lbug/schema-constants.ts`.
- All graph edges are stored in one `CodeRelation` relationship table.
- `CodeRelation.type` is a `STRING` property; the database does not enforce relationship type enum
  membership. Schema constants are the application-level contract.
- `CodeRelation` columns: `type`, `confidence`, `reason`, `step`, `resolutionSource`, `evidence`,
  `fileHash`.
- Embeddings live in `CodeEmbedding` with chunk metadata and `FLOAT[AVMATRIX_EMBEDDING_DIMS]`;
  default dimensions are `384`.
- Vector index: `code_embedding_idx`, cosine metric.
- Analyze creates FTS indexes for `File`, `Function`, `Class`, `Method`, and `Interface`.

## Fixture-Generated Output

Mini-repo golden output currently emits:

- Node labels: `Class`, `Community`, `Const`, `File`, `Folder`, `Function`, `Interface`, `Method`,
  `Process`.
- Relationship types: `CALLS`, `CONTAINS`, `DEFINES`, `HAS_METHOD`, `IMPORTS`, `MEMBER_OF`,
  `STEP_IN_PROCESS`, `USES`.

`USES` is therefore not optional: it is emitted by current runtime output.

## Reconciliation Decisions

- `USES`: required Go relationship type. It is in graph types, emitted by `emit-references`, and
  present in fixture output. The TypeScript baseline drift is closed: `REL_TYPES`, MCP
  impact/context traversal, exposed MCP schema/help text, and contract tests now include it.
- `INHERITS`: required Go relationship capability. It is in graph types and emitted by
  `emit-references` for inheritance references, even though mini-repo does not hit it. The
  TypeScript baseline drift is closed across the same schema, MCP, and test surfaces.
- `DECORATES`: reserved/type-only in this baseline. Keep only if the generated Web UI/shared type
  contract exposes it; do not invent a Go emitter.
- `Project`, `Package`, `Decorator`, `Import`, `Type`: graph-type-only labels in the current
  baseline. Do not create LadybugDB tables in Go until a runtime emitter or fixture proves them.

The Go graph contract must use this reconciled set, not blindly copy either `REL_TYPES` or
`GraphRelationship` alone.
