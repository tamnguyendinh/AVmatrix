# AVmatrix-GO Remaining TypeScript/JavaScript Conversion Files

Source plan: `docs/plans/2026-05-08-avmatrix-go-typescript-node-to-go-conversion-plan.md`.

Snapshot date: 2026-05-14. This is a tracked-file inventory of non-Web TypeScript/JavaScript/CJS/MJS files still needing one of these outcomes before final cutover: port to Go, remove from the final package, or explicitly reclassify as allowed non-runtime data.

Final cutover update, 2026-05-16: the legacy `avmatrix/src`, `avmatrix-shared`, `avmatrix/scripts`, and `avmatrix/vendor` JavaScript/TypeScript authority was retired in one package/source cutover batch after Go build, Go tests, Web build/tests, package dry-run, MCP/CLI smokes, benchmark refresh, and evidence. The only remaining file in the final non-Web TS/JS-family inventory is `eslint.config.mjs`, classified as root Web/dev lint config rather than CLI/backend runtime authority.

This checklist is a translation-and-retirement ledger. Legacy means the TypeScript/JavaScript file
is old authority from before the Go cutover, but implementation legacy still has to be translated
1:1 into Go before it can be retired. The intended flow is: translate the behavior to Go, switch
entrypoints/package scripts/tests/runtime authority to the Go path, prove the Go path with build,
tests, E2E, benchmark, and evidence, then delete or remove the old TypeScript/JavaScript file from
package/runtime/source authority. Reclassification is reserved for non-executed fixture or baseline
data only.

Validation is behavior-cluster based, not per-file. Translate the coherent cluster first, switch the
cluster to Go authority, then run one required validation package for that cluster: full build
before tests, tests including E2E, benchmark, and evidence. Do not run a full validation loop after
each individual file inside the same cluster. Individual file checkboxes are ticked only after the
cluster-level Go path is proven and the old authority for those files is retired.

Excluded from this checklist:

- `avmatrix-web/`: allowed TypeScript/React Web UI display/build surface.
- `avmatrix/test/fixtures/`: analyzer fixture input data, tracked separately from runtime/test-harness conversion.
- Generated/build/dependency output such as `node_modules/`, `dist/`, `build/`, and `coverage/`.

Tick a file only after the implementation slice has evidence showing the file is ported,
removed/excluded, or reclassified.

## Checklist Outcome Rules

- Port implementation behavior 1:1 into Go before retiring the TypeScript/JavaScript source.
  Preserve contracts, flags, output text, generated files, path layout, side effects, error
  behavior, and edge cases unless the plan records an explicit compatibility correction.
- After the Go translation exists, switch the behavior cluster's entrypoints, package metadata,
  scripts, tests, and runtime path to the Go implementation. The old TypeScript/JavaScript source
  must not remain as active authority.
- Tick a legacy implementation file only after the Go path is validated and the old file is deleted
  or removed from package/runtime/source authority.
- Reclassification is only for non-executed fixture, sample, snapshot, or baseline comparison data.
  Obsolete implementation source is not allowed to remain merely because it is "legacy".
- For behavior clusters with multiple files, keep the legacy files until the whole Go-translated
  cluster runs correctly; then retire the legacy files together so the repo does not keep dead
  implementation code.
- Do not run full build/test/E2E per individual file inside a cluster. Run the full validation
  package once per completed behavior cluster, then update all affected file checkboxes immediately.

## High-Risk Conversion Notes

Treat content-generating and agent-instruction files as higher risk than ordinary runtime code.
When converting these files, preserve exact generated semantics, section markers, idempotent
upsert behavior, skip flags, and skill paths before ticking them:

- `avmatrix/src/cli/ai-context.ts`: generates and upserts AVmatrix sections in `AGENTS.md` and
  `CLAUDE.md`, installs base skills, and must preserve `--skip-agents-md` and `--no-stats`.
- `avmatrix/src/cli/skill-gen.ts`: generates `.claude/skills/generated/*/SKILL.md` content from
  communities; conversion needs snapshot/e2e coverage for generated markdown and path layout.
- `avmatrix/src/core/run-analyze.ts` and `avmatrix/src/cli/analyze.ts`: orchestrate `--skills`
  and AI-context generation side effects.
- `avmatrix/src/cli/setup.ts`: writes editor MCP/skills/hooks config and must remain idempotent,
  preserving user-owned settings.
- Related tests `avmatrix/test/unit/ai-context.test.ts`, `avmatrix/test/unit/skill-gen.test.ts`,
  and `avmatrix/test/integration/setup-skills.test.ts` have moved to Go coverage; `avmatrix/test/integration/skills-e2e.test.ts`
  is explicitly retained as broad e2e evidence until a Go e2e replacement is selected.

## Summary

| Category | Files |
| --- | ---: |
| Legacy CLI/core runtime source | 339 |
| Legacy TypeScript contract authority | 34 |
| Node/Vitest test harness | 252 |
| Package and build scripts | 8 |
| Claude hook runtime support | 1 |
| Bundled JS/CJS vendor support | 4 |
| Legacy package config | 1 |
| Root JS/TS config | 1 |
| Total | 640 |

## Legacy CLI/core runtime source (339)

- [x] `avmatrix/src/cli/ai-context.ts`
- [x] `avmatrix/src/cli/analyze.ts`
- [x] `avmatrix/src/cli/augment.ts`
- [x] `avmatrix/src/cli/benchmark.ts`
- [x] `avmatrix/src/cli/clean.ts`
- [x] `avmatrix/src/cli/group.ts`
- [x] `avmatrix/src/cli/index-repo.ts`
- [x] `avmatrix/src/cli/index.ts`
- [x] `avmatrix/src/cli/lazy-action.ts`
- [x] `avmatrix/src/cli/list.ts`
- [x] `avmatrix/src/cli/mcp.ts`
- [x] `avmatrix/src/cli/serve.ts`
- [x] `avmatrix/src/cli/setup.ts`
- [x] `avmatrix/src/cli/skill-gen.ts`
- [x] `avmatrix/src/cli/status.ts`
- [x] `avmatrix/src/cli/tool.ts`
- [x] `avmatrix/src/cli/wiki-gated.ts`
- [x] `avmatrix/src/cli/wiki.ts`
- [x] `avmatrix/src/config/ignore-service.ts`
- [x] `avmatrix/src/config/supported-languages.ts`
- [x] `avmatrix/src/core/analyze/analyze-benchmark-snapshot.ts`
- [x] `avmatrix/src/core/analyze/analyze-metrics.ts`
- [x] `avmatrix/src/core/analyze/graph-correctness-snapshot.ts`
- [x] `avmatrix/src/core/augmentation/engine.ts`
- [x] `avmatrix/src/core/embeddings/ast-utils.ts`
- [x] `avmatrix/src/core/embeddings/character-chunk.ts`
- [x] `avmatrix/src/core/embeddings/chunker.ts`
- [x] `avmatrix/src/core/embeddings/embedder.ts`
- [x] `avmatrix/src/core/embeddings/embedding-pipeline.ts`
- [x] `avmatrix/src/core/embeddings/http-client.ts`
- [x] `avmatrix/src/core/embeddings/index.ts`
- [x] `avmatrix/src/core/embeddings/line-index.ts`
- [x] `avmatrix/src/core/embeddings/server-mapping.ts`
- [x] `avmatrix/src/core/embeddings/structural-extractor.ts`
- [x] `avmatrix/src/core/embeddings/text-generator.ts`
- [x] `avmatrix/src/core/embeddings/types.ts`
- [x] `avmatrix/src/core/git-staleness.ts`
- [x] `avmatrix/src/core/graph/graph.ts`
- [x] `avmatrix/src/core/graph/types.ts`
- [x] `avmatrix/src/core/group/bridge-db.ts`
- [x] `avmatrix/src/core/group/bridge-schema.ts`
- [x] `avmatrix/src/core/group/capabilities.ts`
- [x] `avmatrix/src/core/group/config-parser.ts`
- [x] `avmatrix/src/core/group/contract-extractor.ts`
- [x] `avmatrix/src/core/group/extractors/fs-utils.ts`
- [x] `avmatrix/src/core/group/extractors/grpc-extractor.ts`
- [x] `avmatrix/src/core/group/extractors/grpc-patterns/go.ts`
- [x] `avmatrix/src/core/group/extractors/grpc-patterns/index.ts`
- [x] `avmatrix/src/core/group/extractors/grpc-patterns/java.ts`
- [x] `avmatrix/src/core/group/extractors/grpc-patterns/node.ts`
- [x] `avmatrix/src/core/group/extractors/grpc-patterns/proto.ts`
- [x] `avmatrix/src/core/group/extractors/grpc-patterns/python.ts`
- [x] `avmatrix/src/core/group/extractors/grpc-patterns/types.ts`
- [x] `avmatrix/src/core/group/extractors/http-patterns/go.ts`
- [x] `avmatrix/src/core/group/extractors/http-patterns/index.ts`
- [x] `avmatrix/src/core/group/extractors/http-patterns/java.ts`
- [x] `avmatrix/src/core/group/extractors/http-patterns/node.ts`
- [x] `avmatrix/src/core/group/extractors/http-patterns/php.ts`
- [x] `avmatrix/src/core/group/extractors/http-patterns/python.ts`
- [x] `avmatrix/src/core/group/extractors/http-patterns/types.ts`
- [x] `avmatrix/src/core/group/extractors/http-route-extractor.ts`
- [x] `avmatrix/src/core/group/extractors/manifest-extractor.ts`
- [x] `avmatrix/src/core/group/extractors/topic-extractor.ts`
- [x] `avmatrix/src/core/group/extractors/topic-patterns/go.ts`
- [x] `avmatrix/src/core/group/extractors/topic-patterns/index.ts`
- [x] `avmatrix/src/core/group/extractors/topic-patterns/java.ts`
- [x] `avmatrix/src/core/group/extractors/topic-patterns/node.ts`
- [x] `avmatrix/src/core/group/extractors/topic-patterns/python.ts`
- [x] `avmatrix/src/core/group/extractors/topic-patterns/types.ts`
- [x] `avmatrix/src/core/group/extractors/tree-sitter-scanner.ts`
- [x] `avmatrix/src/core/group/matching.ts`
- [x] `avmatrix/src/core/group/normalization.ts`
- [x] `avmatrix/src/core/group/service-boundary-detector.ts`
- [x] `avmatrix/src/core/group/service.ts`
- [x] `avmatrix/src/core/group/storage.ts`
- [x] `avmatrix/src/core/group/sync.ts`
- [x] `avmatrix/src/core/group/types.ts`
- [x] `avmatrix/src/core/ingestion/ast-cache.ts`
- [x] `avmatrix/src/core/ingestion/binding-accumulator.ts`
- [x] `avmatrix/src/core/ingestion/call-extractors/configs/c-cpp.ts`
- [x] `avmatrix/src/core/ingestion/call-extractors/configs/csharp.ts`
- [x] `avmatrix/src/core/ingestion/call-extractors/configs/dart.ts`
- [x] `avmatrix/src/core/ingestion/call-extractors/configs/go.ts`
- [x] `avmatrix/src/core/ingestion/call-extractors/configs/jvm.ts`
- [x] `avmatrix/src/core/ingestion/call-extractors/configs/php.ts`
- [x] `avmatrix/src/core/ingestion/call-extractors/configs/python.ts`
- [x] `avmatrix/src/core/ingestion/call-extractors/configs/ruby.ts`
- [x] `avmatrix/src/core/ingestion/call-extractors/configs/rust.ts`
- [x] `avmatrix/src/core/ingestion/call-extractors/configs/swift.ts`
- [x] `avmatrix/src/core/ingestion/call-extractors/configs/typescript-javascript.ts`
- [x] `avmatrix/src/core/ingestion/call-extractors/generic.ts`
- [x] `avmatrix/src/core/ingestion/call-processor.ts`
- [x] `avmatrix/src/core/ingestion/call-routing.ts`
- [x] `avmatrix/src/core/ingestion/call-types.ts`
- [x] `avmatrix/src/core/ingestion/class-extractors/configs/c-cpp.ts`
- [x] `avmatrix/src/core/ingestion/class-extractors/configs/csharp.ts`
- [x] `avmatrix/src/core/ingestion/class-extractors/configs/dart.ts`
- [x] `avmatrix/src/core/ingestion/class-extractors/configs/go.ts`
- [x] `avmatrix/src/core/ingestion/class-extractors/configs/jvm.ts`
- [x] `avmatrix/src/core/ingestion/class-extractors/configs/php.ts`
- [x] `avmatrix/src/core/ingestion/class-extractors/configs/python.ts`
- [x] `avmatrix/src/core/ingestion/class-extractors/configs/ruby.ts`
- [x] `avmatrix/src/core/ingestion/class-extractors/configs/rust.ts`
- [x] `avmatrix/src/core/ingestion/class-extractors/configs/swift.ts`
- [x] `avmatrix/src/core/ingestion/class-extractors/configs/typescript-javascript.ts`
- [x] `avmatrix/src/core/ingestion/class-extractors/generic.ts`
- [x] `avmatrix/src/core/ingestion/class-types.ts`
- [x] `avmatrix/src/core/ingestion/cluster-enricher.ts`
- [x] `avmatrix/src/core/ingestion/cobol-processor.ts`
- [x] `avmatrix/src/core/ingestion/cobol/cobol-copy-expander.ts`
- [x] `avmatrix/src/core/ingestion/cobol/cobol-preprocessor.ts`
- [x] `avmatrix/src/core/ingestion/cobol/jcl-parser.ts`
- [x] `avmatrix/src/core/ingestion/cobol/jcl-processor.ts`
- [x] `avmatrix/src/core/ingestion/community-processor.ts`
- [x] `avmatrix/src/core/ingestion/constants.ts`
- [x] `avmatrix/src/core/ingestion/emit-references.ts`
- [x] `avmatrix/src/core/ingestion/entry-point-scoring.ts`
- [x] `avmatrix/src/core/ingestion/export-detection.ts`
- [x] `avmatrix/src/core/ingestion/field-extractor.ts`
- [x] `avmatrix/src/core/ingestion/field-extractors/configs/c-cpp.ts`
- [x] `avmatrix/src/core/ingestion/field-extractors/configs/csharp.ts`
- [x] `avmatrix/src/core/ingestion/field-extractors/configs/dart.ts`
- [x] `avmatrix/src/core/ingestion/field-extractors/configs/go.ts`
- [x] `avmatrix/src/core/ingestion/field-extractors/configs/helpers.ts`
- [x] `avmatrix/src/core/ingestion/field-extractors/configs/jvm.ts`
- [x] `avmatrix/src/core/ingestion/field-extractors/configs/php.ts`
- [x] `avmatrix/src/core/ingestion/field-extractors/configs/python.ts`
- [x] `avmatrix/src/core/ingestion/field-extractors/configs/ruby.ts`
- [x] `avmatrix/src/core/ingestion/field-extractors/configs/rust.ts`
- [x] `avmatrix/src/core/ingestion/field-extractors/configs/swift.ts`
- [x] `avmatrix/src/core/ingestion/field-extractors/configs/typescript-javascript.ts`
- [x] `avmatrix/src/core/ingestion/field-extractors/generic.ts`
- [x] `avmatrix/src/core/ingestion/field-extractors/typescript.ts`
- [x] `avmatrix/src/core/ingestion/field-types.ts`
- [x] `avmatrix/src/core/ingestion/filesystem-walker.ts`
- [x] `avmatrix/src/core/ingestion/finalize-orchestrator.ts`
- [x] `avmatrix/src/core/ingestion/framework-detection.ts`
- [x] `avmatrix/src/core/ingestion/heritage-extractors/configs/go.ts`
- [x] `avmatrix/src/core/ingestion/heritage-extractors/configs/ruby.ts`
- [x] `avmatrix/src/core/ingestion/heritage-extractors/generic.ts`
- [x] `avmatrix/src/core/ingestion/heritage-processor.ts`
- [x] `avmatrix/src/core/ingestion/heritage-types.ts`
- [x] `avmatrix/src/core/ingestion/import-processor.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/configs/c-cpp.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/configs/csharp.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/configs/dart.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/configs/go.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/configs/jvm.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/configs/php.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/configs/python.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/configs/ruby.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/configs/rust.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/configs/swift.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/configs/typescript-javascript.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/csharp.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/go.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/jvm.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/php.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/python.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/resolver-factory.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/ruby.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/rust.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/standard.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/types.ts`
- [x] `avmatrix/src/core/ingestion/import-resolvers/utils.ts`
- [x] `avmatrix/src/core/ingestion/import-target-adapter.ts`
- [x] `avmatrix/src/core/ingestion/language-config.ts`
- [x] `avmatrix/src/core/ingestion/language-provider.ts`
- [x] `avmatrix/src/core/ingestion/languages/c-cpp.ts`
- [x] `avmatrix/src/core/ingestion/languages/cobol.ts`
- [x] `avmatrix/src/core/ingestion/languages/csharp.ts`
- [x] `avmatrix/src/core/ingestion/languages/dart.ts`
- [x] `avmatrix/src/core/ingestion/languages/go.ts`
- [x] `avmatrix/src/core/ingestion/languages/index.ts`
- [x] `avmatrix/src/core/ingestion/languages/java.ts`
- [x] `avmatrix/src/core/ingestion/languages/kotlin.ts`
- [x] `avmatrix/src/core/ingestion/languages/php.ts`
- [x] `avmatrix/src/core/ingestion/languages/python.ts`
- [x] `avmatrix/src/core/ingestion/languages/ruby.ts`
- [x] `avmatrix/src/core/ingestion/languages/rust.ts`
- [x] `avmatrix/src/core/ingestion/languages/swift.ts`
- [x] `avmatrix/src/core/ingestion/languages/typescript.ts`
- [x] `avmatrix/src/core/ingestion/languages/vue.ts`
- [x] `avmatrix/src/core/ingestion/markdown-processor.ts`
- [x] `avmatrix/src/core/ingestion/method-extractors/configs/c-cpp.ts`
- [x] `avmatrix/src/core/ingestion/method-extractors/configs/csharp.ts`
- [x] `avmatrix/src/core/ingestion/method-extractors/configs/dart.ts`
- [x] `avmatrix/src/core/ingestion/method-extractors/configs/go.ts`
- [x] `avmatrix/src/core/ingestion/method-extractors/configs/jvm.ts`
- [x] `avmatrix/src/core/ingestion/method-extractors/configs/php.ts`
- [x] `avmatrix/src/core/ingestion/method-extractors/configs/python.ts`
- [x] `avmatrix/src/core/ingestion/method-extractors/configs/ruby.ts`
- [x] `avmatrix/src/core/ingestion/method-extractors/configs/rust.ts`
- [x] `avmatrix/src/core/ingestion/method-extractors/configs/swift.ts`
- [x] `avmatrix/src/core/ingestion/method-extractors/configs/typescript-javascript.ts`
- [x] `avmatrix/src/core/ingestion/method-extractors/generic.ts`
- [x] `avmatrix/src/core/ingestion/method-types.ts`
- [x] `avmatrix/src/core/ingestion/model/field-registry.ts`
- [x] `avmatrix/src/core/ingestion/model/heritage-map.ts`
- [x] `avmatrix/src/core/ingestion/model/index.ts`
- [x] `avmatrix/src/core/ingestion/model/method-registry.ts`
- [x] `avmatrix/src/core/ingestion/model/registration-table.ts`
- [x] `avmatrix/src/core/ingestion/model/resolution-context.ts`
- [x] `avmatrix/src/core/ingestion/model/resolve.ts`
- [x] `avmatrix/src/core/ingestion/model/scope-resolution-indexes.ts`
- [x] `avmatrix/src/core/ingestion/model/semantic-model.ts`
- [x] `avmatrix/src/core/ingestion/model/symbol-table.ts`
- [x] `avmatrix/src/core/ingestion/model/type-registry.ts`
- [x] `avmatrix/src/core/ingestion/mro-processor.ts`
- [x] `avmatrix/src/core/ingestion/named-bindings/csharp.ts`
- [x] `avmatrix/src/core/ingestion/named-bindings/java.ts`
- [x] `avmatrix/src/core/ingestion/named-bindings/kotlin.ts`
- [x] `avmatrix/src/core/ingestion/named-bindings/php.ts`
- [x] `avmatrix/src/core/ingestion/named-bindings/python.ts`
- [x] `avmatrix/src/core/ingestion/named-bindings/rust.ts`
- [x] `avmatrix/src/core/ingestion/named-bindings/types.ts`
- [x] `avmatrix/src/core/ingestion/named-bindings/typescript.ts`
- [x] `avmatrix/src/core/ingestion/parsing-processor.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/cobol.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/communities.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/cross-file-impl.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/cross-file.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/index.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/markdown.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/mro.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/orm.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/parse-impl.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/parse.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/processes.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/resolution.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/routes.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/runner.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/scan.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/structure.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/tools.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/types.ts`
- [x] `avmatrix/src/core/ingestion/pipeline-phases/wildcard-synthesis.ts`
- [x] `avmatrix/src/core/ingestion/pipeline.ts`
- [x] `avmatrix/src/core/ingestion/process-processor.ts`
- [x] `avmatrix/src/core/ingestion/registry-primary-flag.ts`
- [x] `avmatrix/src/core/ingestion/route-extractors/expo.ts`
- [x] `avmatrix/src/core/ingestion/route-extractors/middleware.ts`
- [x] `avmatrix/src/core/ingestion/route-extractors/nextjs.ts`
- [x] `avmatrix/src/core/ingestion/route-extractors/php.ts`
- [x] `avmatrix/src/core/ingestion/route-extractors/response-shapes.ts`
- [x] `avmatrix/src/core/ingestion/scope-captures/python.ts`
- [x] `avmatrix/src/core/ingestion/scope-captures/typescript-javascript.ts`
- [x] `avmatrix/src/core/ingestion/scope-extractor-bridge.ts`
- [x] `avmatrix/src/core/ingestion/scope-extractor.ts`
- [x] `avmatrix/src/core/ingestion/scope-reference-resolver.ts`
- [x] `avmatrix/src/core/ingestion/shadow-harness.ts`
- [x] `avmatrix/src/core/ingestion/structure-processor.ts`
- [x] `avmatrix/src/core/ingestion/tree-sitter-queries.ts`
- [x] `avmatrix/src/core/ingestion/type-env.ts`
- [x] `avmatrix/src/core/ingestion/type-extractors/c-cpp.ts`
- [x] `avmatrix/src/core/ingestion/type-extractors/csharp.ts`
- [x] `avmatrix/src/core/ingestion/type-extractors/dart.ts`
- [x] `avmatrix/src/core/ingestion/type-extractors/go.ts`
- [x] `avmatrix/src/core/ingestion/type-extractors/jvm.ts`
- [x] `avmatrix/src/core/ingestion/type-extractors/php.ts`
- [x] `avmatrix/src/core/ingestion/type-extractors/python.ts`
- [x] `avmatrix/src/core/ingestion/type-extractors/ruby.ts`
- [x] `avmatrix/src/core/ingestion/type-extractors/rust.ts`
- [x] `avmatrix/src/core/ingestion/type-extractors/shared.ts`
- [x] `avmatrix/src/core/ingestion/type-extractors/swift.ts`
- [x] `avmatrix/src/core/ingestion/type-extractors/types.ts`
- [x] `avmatrix/src/core/ingestion/type-extractors/typescript.ts`
- [x] `avmatrix/src/core/ingestion/utils/ast-helpers.ts`
- [x] `avmatrix/src/core/ingestion/utils/call-analysis.ts`
- [x] `avmatrix/src/core/ingestion/utils/env.ts`
- [x] `avmatrix/src/core/ingestion/utils/event-loop.ts`
- [x] `avmatrix/src/core/ingestion/utils/graph-sort.ts`
- [x] `avmatrix/src/core/ingestion/utils/method-props.ts`
- [x] `avmatrix/src/core/ingestion/utils/ruby-self-call.ts`
- [x] `avmatrix/src/core/ingestion/utils/verbose.ts`
- [x] `avmatrix/src/core/ingestion/variable-extractors/configs/c-cpp.ts`
- [x] `avmatrix/src/core/ingestion/variable-extractors/configs/csharp.ts`
- [x] `avmatrix/src/core/ingestion/variable-extractors/configs/dart.ts`
- [x] `avmatrix/src/core/ingestion/variable-extractors/configs/go.ts`
- [x] `avmatrix/src/core/ingestion/variable-extractors/configs/jvm.ts`
- [x] `avmatrix/src/core/ingestion/variable-extractors/configs/php.ts`
- [x] `avmatrix/src/core/ingestion/variable-extractors/configs/python.ts`
- [x] `avmatrix/src/core/ingestion/variable-extractors/configs/ruby.ts`
- [x] `avmatrix/src/core/ingestion/variable-extractors/configs/rust.ts`
- [x] `avmatrix/src/core/ingestion/variable-extractors/configs/swift.ts`
- [x] `avmatrix/src/core/ingestion/variable-extractors/configs/typescript-javascript.ts`
- [x] `avmatrix/src/core/ingestion/variable-extractors/generic.ts`
- [x] `avmatrix/src/core/ingestion/variable-types.ts`
- [x] `avmatrix/src/core/ingestion/vue-sfc-extractor.ts`
- [x] `avmatrix/src/core/ingestion/workers/parse-worker.ts`
- [x] `avmatrix/src/core/ingestion/workers/resolution-worker.ts`
- [x] `avmatrix/src/core/ingestion/workers/worker-pool.ts`
- [x] `avmatrix/src/core/lbug/csv-generator.ts`
- [x] `avmatrix/src/core/lbug/lbug-adapter.ts`
- [x] `avmatrix/src/core/lbug/pool-adapter.ts`
- [x] `avmatrix/src/core/lbug/schema.ts`
- [x] `avmatrix/src/core/run-analyze.ts`
- [x] `avmatrix/src/core/search/bm25-index.ts`
- [x] `avmatrix/src/core/search/hybrid-search.ts`
- [x] `avmatrix/src/core/search/phase-timer.ts`
- [x] `avmatrix/src/core/tree-sitter/parser-loader.ts`
- [x] `avmatrix/src/core/wiki/cursor-client.ts`
- [x] `avmatrix/src/core/wiki/generator.ts`
- [x] `avmatrix/src/core/wiki/graph-queries.ts`
- [x] `avmatrix/src/core/wiki/html-viewer.ts`
- [x] `avmatrix/src/core/wiki/llm-client.ts`
- [x] `avmatrix/src/core/wiki/prompts.ts`
- [x] `avmatrix/src/lib/utils.ts`
- [x] `avmatrix/src/mcp/compatible-stdio-transport.ts`
- [x] `avmatrix/src/mcp/contracts/impact.ts`
- [x] `avmatrix/src/mcp/core/embedder.ts`
- [x] `avmatrix/src/mcp/core/lbug-adapter.ts`
- [x] `avmatrix/src/mcp/local/local-backend.ts`
- [x] `avmatrix/src/mcp/resources.ts`
- [x] `avmatrix/src/mcp/server.ts`
- [x] `avmatrix/src/mcp/staleness.ts`
- [x] `avmatrix/src/mcp/tool-schema.ts`
- [x] `avmatrix/src/mcp/tools.ts`
- [x] `avmatrix/src/runtime/repo-resolver.ts`
- [x] `avmatrix/src/runtime/repo-runtime/graph-read-service.ts`
- [x] `avmatrix/src/runtime/repo-runtime/repo-read-executor.ts`
- [x] `avmatrix/src/runtime/runtime-controller.ts`
- [x] `avmatrix/src/runtime/session-adapter.ts`
- [x] `avmatrix/src/runtime/session-adapters/codex.ts`
- [x] `avmatrix/src/runtime/session-jobs/session-job.ts`
- [x] `avmatrix/src/server/analyze-job.ts`
- [x] `avmatrix/src/server/analyze-worker.ts`
- [x] `avmatrix/src/server/api.ts`
- [x] `avmatrix/src/server/compatibility-repo-cache.ts`
- [x] `avmatrix/src/server/graph-stream-http.ts`
- [x] `avmatrix/src/server/local-folder-picker.ts`
- [x] `avmatrix/src/server/local-path-policy.ts`
- [x] `avmatrix/src/server/mcp-http.ts`
- [x] `avmatrix/src/server/session-bridge.ts`
- [x] `avmatrix/src/storage/git.ts`
- [x] `avmatrix/src/storage/repo-manager.ts`
- [x] `avmatrix/src/storage/runtime-config.ts`
- [x] `avmatrix/src/storage/settings.ts`
- [x] `avmatrix/src/types/pipeline.ts`

## Legacy TypeScript contract authority (34)

- [x] `avmatrix-shared/src/graph/types.ts`
- [x] `avmatrix-shared/src/index.ts`
- [x] `avmatrix-shared/src/language-detection.ts`
- [x] `avmatrix-shared/src/languages.ts`
- [x] `avmatrix-shared/src/lbug/schema-constants.ts`
- [x] `avmatrix-shared/src/mro-strategy.ts`
- [x] `avmatrix-shared/src/pipeline.ts`
- [x] `avmatrix-shared/src/scope-resolution/def-index.ts`
- [x] `avmatrix-shared/src/scope-resolution/evidence-weights.ts`
- [x] `avmatrix-shared/src/scope-resolution/finalize-algorithm.ts`
- [x] `avmatrix-shared/src/scope-resolution/language-classification.ts`
- [x] `avmatrix-shared/src/scope-resolution/method-dispatch-index.ts`
- [x] `avmatrix-shared/src/scope-resolution/module-scope-index.ts`
- [x] `avmatrix-shared/src/scope-resolution/origin-priority.ts`
- [x] `avmatrix-shared/src/scope-resolution/parsed-file.ts`
- [x] `avmatrix-shared/src/scope-resolution/position-index.ts`
- [x] `avmatrix-shared/src/scope-resolution/qualified-name-index.ts`
- [x] `avmatrix-shared/src/scope-resolution/reference-site.ts`
- [x] `avmatrix-shared/src/scope-resolution/registries/class-registry.ts`
- [x] `avmatrix-shared/src/scope-resolution/registries/context.ts`
- [x] `avmatrix-shared/src/scope-resolution/registries/evidence.ts`
- [x] `avmatrix-shared/src/scope-resolution/registries/field-registry.ts`
- [x] `avmatrix-shared/src/scope-resolution/registries/lookup-core.ts`
- [x] `avmatrix-shared/src/scope-resolution/registries/lookup-qualified.ts`
- [x] `avmatrix-shared/src/scope-resolution/registries/method-registry.ts`
- [x] `avmatrix-shared/src/scope-resolution/registries/tie-breaks.ts`
- [x] `avmatrix-shared/src/scope-resolution/resolve-type-ref.ts`
- [x] `avmatrix-shared/src/scope-resolution/scope-id.ts`
- [x] `avmatrix-shared/src/scope-resolution/scope-tree.ts`
- [x] `avmatrix-shared/src/scope-resolution/shadow/aggregate.ts`
- [x] `avmatrix-shared/src/scope-resolution/shadow/diff.ts`
- [x] `avmatrix-shared/src/scope-resolution/symbol-definition.ts`
- [x] `avmatrix-shared/src/scope-resolution/types.ts`
- [x] `avmatrix-shared/src/session.ts`

## Node/Vitest test harness (252)

- [x] `avmatrix/test/global-setup.ts`
- [x] `avmatrix/test/helpers/test-db.ts`
- [x] `avmatrix/test/helpers/test-graph.ts`
- [x] `avmatrix/test/helpers/test-indexed-db.ts`
- [x] `avmatrix/test/integration/api-impact-e2e.test.ts`
- [x] `avmatrix/test/integration/augmentation.test.ts`
- [x] `avmatrix/test/integration/class-impact-all-languages.test.ts`
- [x] `avmatrix/test/integration/cli-e2e.test.ts`
- [x] `avmatrix/test/integration/cross-file-binding.test.ts`
- [x] `avmatrix/test/integration/csv-pipeline.test.ts`
- [x] `avmatrix/test/integration/enrichment.test.ts`
- [x] `avmatrix/test/integration/expo-routes.test.ts`
- [x] `avmatrix/test/integration/filesystem-walker.test.ts`
- [x] `avmatrix/test/integration/group/group-cli.test.ts`
- [x] `avmatrix/test/integration/group/group-sync.test.ts`
- [x] `avmatrix/test/integration/group/monorepo-sync.test.ts`
- [x] `avmatrix/test/integration/has-method.test.ts`
- [x] `avmatrix/test/integration/heritage-extractor-wiring.test.ts`
- [x] `avmatrix/test/integration/hooks-e2e.test.ts`
- [x] `avmatrix/test/integration/ignore-and-skip-e2e.test.ts`
- [x] `avmatrix/test/integration/java-class-impact.test.ts`
- [x] `avmatrix/test/integration/lbug-core-adapter.test.ts`
- [x] `avmatrix/test/integration/lbug-lock-retry.test.ts`
- [x] `avmatrix/test/integration/lbug-pool-stability.test.ts`
- [x] `avmatrix/test/integration/lbug-pool.test.ts`
- [x] `avmatrix/test/integration/lbug-vector-extension.test.ts`
- [x] `avmatrix/test/integration/local-backend-calltool.test.ts`
- [x] `avmatrix/test/integration/local-backend.test.ts`
- [x] `avmatrix/test/integration/orm-dataflow.test.ts`
- [x] `avmatrix/test/integration/parsing.test.ts`
- [x] `avmatrix/test/integration/pipeline-graph-golden.test.ts`
- [x] `avmatrix/test/integration/pipeline.test.ts`
- [x] `avmatrix/test/integration/qualified-class-lookups.test.ts`
- [x] `avmatrix/test/integration/query-compilation.test.ts`
- [x] `avmatrix/test/integration/resolvers/api-deep-flow.test.ts`
- [x] `avmatrix/test/integration/resolvers/cobol.test.ts`
- [x] `avmatrix/test/integration/resolvers/cpp.test.ts`
- [x] `avmatrix/test/integration/resolvers/csharp.test.ts`
- [x] `avmatrix/test/integration/resolvers/dart.test.ts`
- [x] `avmatrix/test/integration/resolvers/express-routes.test.ts`
- [x] `avmatrix/test/integration/resolvers/go.test.ts`
- [x] `avmatrix/test/integration/resolvers/helpers.ts`
- [x] `avmatrix/test/integration/resolvers/java.test.ts`
- [x] `avmatrix/test/integration/resolvers/javascript.test.ts`
- [x] `avmatrix/test/integration/resolvers/kotlin.test.ts`
- [x] `avmatrix/test/integration/resolvers/php-response-shapes.test.ts`
- [x] `avmatrix/test/integration/resolvers/php.test.ts`
- [x] `avmatrix/test/integration/resolvers/python-mcp-tools.test.ts`
- [x] `avmatrix/test/integration/resolvers/python.test.ts`
- [x] `avmatrix/test/integration/resolvers/route-mapping.test.ts`
- [x] `avmatrix/test/integration/resolvers/ruby-mixin-worker.test.ts`
- [x] `avmatrix/test/integration/resolvers/ruby.test.ts`
- [x] `avmatrix/test/integration/resolvers/rust.test.ts`
- [x] `avmatrix/test/integration/resolvers/shape-check.test.ts`
- [x] `avmatrix/test/integration/resolvers/swift.test.ts`
- [x] `avmatrix/test/integration/resolvers/typescript.test.ts`
- [x] `avmatrix/test/integration/resolvers/vue.test.ts`
- [x] `avmatrix/test/integration/scope-audit-persistence.test.ts`
- [x] `avmatrix/test/integration/search-core.test.ts`
- [x] `avmatrix/test/integration/search-pool.test.ts`
- [x] `avmatrix/test/integration/server-analyze.test.ts`
- [x] `avmatrix/test/integration/setup-skills.test.ts`
- [x] `avmatrix/test/integration/shape-check-regression.test.ts`
- [x] `avmatrix/test/integration/skills-e2e.test.ts`
- [x] `avmatrix/test/integration/staleness-and-stability.test.ts`
- [x] `avmatrix/test/integration/tree-sitter-languages.test.ts`
- [x] `avmatrix/test/integration/worker-pool.test.ts`
- [x] `avmatrix/test/unit/ai-context.test.ts`
- [x] `avmatrix/test/unit/analyze-api.test.ts`
- [x] `avmatrix/test/unit/analyze-benchmark-snapshot.test.ts`
- [x] `avmatrix/test/unit/analyze-job.test.ts`
- [x] `avmatrix/test/unit/analyze-metrics.test.ts`
- [x] `avmatrix/test/unit/api-graph-streaming.test.ts`
- [x] `avmatrix/test/unit/ast-cache.test.ts`
- [x] `avmatrix/test/unit/ast-utils.test.ts`
- [x] `avmatrix/test/unit/benchmark-compare-command.test.ts`
- [x] `avmatrix/test/unit/binding-accumulator.test.ts`
- [x] `avmatrix/test/unit/bm25-search.test.ts`
- [x] `avmatrix/test/unit/call-extraction.test.ts`
- [x] `avmatrix/test/unit/call-form.test.ts`
- [x] `avmatrix/test/unit/call-processor.test.ts`
- [x] `avmatrix/test/unit/call-routing/ruby.test.ts`
- [x] `avmatrix/test/unit/calltool-dispatch.test.ts`
- [x] `avmatrix/test/unit/chunker.test.ts`
- [x] `avmatrix/test/unit/clean.test.ts`
- [x] `avmatrix/test/unit/cli-commands.test.ts`
- [x] `avmatrix/test/unit/cli-index-help.test.ts`
- [x] `avmatrix/test/unit/cobol-copy-expander.test.ts`
- [x] `avmatrix/test/unit/cobol-preprocessor.test.ts`
- [x] `avmatrix/test/unit/codex-session-adapter.test.ts`
- [x] `avmatrix/test/unit/cohesion-consistency.test.ts`
- [x] `avmatrix/test/unit/community-processor.test.ts`
- [x] `avmatrix/test/unit/compatibility-repo-cache.test.ts`
- [x] `avmatrix/test/unit/compatible-stdio-transport.test.ts`
- [x] `avmatrix/test/unit/contract-freeze/phase1-contract-snapshot.test.ts`
- [x] `avmatrix/test/unit/cors.test.ts`
- [x] `avmatrix/test/unit/cross-file-impl.test.ts`
- [x] `avmatrix/test/unit/cross-file.test.ts`
- [x] `avmatrix/test/unit/csv-escaping.test.ts`
- [x] `avmatrix/test/unit/dart-import-resolver.test.ts`
- [x] `avmatrix/test/unit/dart-type-extractor.test.ts`
- [x] `avmatrix/test/unit/embedder.test.ts`
- [x] `avmatrix/test/unit/embedding-chunking.test.ts`
- [x] `avmatrix/test/unit/embedding-pipeline.test.ts`
- [x] `avmatrix/test/unit/entry-point-scoring.test.ts`
- [x] `avmatrix/test/unit/expo-routes.test.ts`
- [x] `avmatrix/test/unit/extract-element-type-from-string.test.ts`
- [x] `avmatrix/test/unit/extract-generic-type-args.test.ts`
- [x] `avmatrix/test/unit/fetch-reason-parsing.test.ts`
- [x] `avmatrix/test/unit/field-extraction.test.ts`
- [x] `avmatrix/test/unit/framework-detection.test.ts`
- [x] `avmatrix/test/unit/git-utils.test.ts`
- [x] `avmatrix/test/unit/git.test.ts`
- [x] `avmatrix/test/unit/graph-correctness-snapshot.test.ts`
- [x] `avmatrix/test/unit/graph.test.ts`
- [x] `avmatrix/test/unit/group/bridge-db-edge.test.ts`
- [x] `avmatrix/test/unit/group/bridge-db.test.ts`
- [x] `avmatrix/test/unit/group/config-parser.test.ts`
- [x] `avmatrix/test/unit/group/fixtures.ts`
- [x] `avmatrix/test/unit/group/group-tools.test.ts`
- [x] `avmatrix/test/unit/group/grpc-extractor.test.ts`
- [x] `avmatrix/test/unit/group/http-route-extractor.test.ts`
- [x] `avmatrix/test/unit/group/http-route-multi-verb.test.ts`
- [x] `avmatrix/test/unit/group/impact-by-uid.test.ts`
- [x] `avmatrix/test/unit/group/manifest-extractor.test.ts`
- [x] `avmatrix/test/unit/group/matching.test.ts`
- [x] `avmatrix/test/unit/group/service-boundary-detector.test.ts`
- [x] `avmatrix/test/unit/group/service.test.ts`
- [x] `avmatrix/test/unit/group/storage.test.ts`
- [x] `avmatrix/test/unit/group/sync.test.ts`
- [x] `avmatrix/test/unit/group/topic-extractor.test.ts`
- [x] `avmatrix/test/unit/group/types.test.ts`
- [x] `avmatrix/test/unit/has-method.test.ts`
- [x] `avmatrix/test/unit/heritage-extraction.test.ts`
- [x] `avmatrix/test/unit/heritage-map.test.ts`
- [x] `avmatrix/test/unit/heritage-processor.test.ts`
- [x] `avmatrix/test/unit/hooks.test.ts`
- [x] `avmatrix/test/unit/http-embedder.test.ts`
- [x] `avmatrix/test/unit/hybrid-search.test.ts`
- [x] `avmatrix/test/unit/ignore-service.test.ts`
- [x] `avmatrix/test/unit/impact-batching-grouping.test.ts`
- [x] `avmatrix/test/unit/impact-confidence.test.ts`
- [x] `avmatrix/test/unit/impact-contract.test.ts`
- [x] `avmatrix/test/unit/import-processor.test.ts`
- [x] `avmatrix/test/unit/import-resolution/preprocessing.test.ts`
- [x] `avmatrix/test/unit/import-resolver-factory.test.ts`
- [x] `avmatrix/test/unit/index-repo-command.test.ts`
- [x] `avmatrix/test/unit/ingestion-utils.test.ts`
- [x] `avmatrix/test/unit/isWriteQuery.test.ts`
- [x] `avmatrix/test/unit/jcl-parser.test.ts`
- [x] `avmatrix/test/unit/language-skip.test.ts`
- [x] `avmatrix/test/unit/lazy-action.test.ts`
- [x] `avmatrix/test/unit/lbug-embedding-hashes.test.ts`
- [x] `avmatrix/test/unit/lbug-wal-recovery.test.ts`
- [x] `avmatrix/test/unit/local-backend-maxbuffer.test.ts`
- [x] `avmatrix/test/unit/mcp-runtime-alignment.test.ts`
- [x] `avmatrix/test/unit/method-extraction.test.ts`
- [x] `avmatrix/test/unit/method-props.test.ts`
- [x] `avmatrix/test/unit/model/field-registry.test.ts`
- [x] `avmatrix/test/unit/model/helpers.ts`
- [x] `avmatrix/test/unit/model/method-registry.test.ts`
- [x] `avmatrix/test/unit/model/registration-table.test.ts`
- [x] `avmatrix/test/unit/model/resolution-context.test.ts`
- [x] `avmatrix/test/unit/model/semantic-model.test.ts`
- [x] `avmatrix/test/unit/model/type-registry.test.ts`
- [x] `avmatrix/test/unit/mro-processor.test.ts`
- [x] `avmatrix/test/unit/named-bindings/csharp.test.ts`
- [x] `avmatrix/test/unit/noise-filter.test.ts`
- [x] `avmatrix/test/unit/package-bin.test.ts`
- [x] `avmatrix/test/unit/parse-diff-hunks.test.ts`
- [x] `avmatrix/test/unit/parse-impl-worker-canonical.test.ts`
- [x] `avmatrix/test/unit/parser-loader.test.ts`
- [x] `avmatrix/test/unit/phase-timer.test.ts`
- [x] `avmatrix/test/unit/pipeline-exports.test.ts`
- [x] `avmatrix/test/unit/pipeline-runner.test.ts`
- [x] `avmatrix/test/unit/process-processor.test.ts`
- [x] `avmatrix/test/unit/processes-phase.test.ts`
- [x] `avmatrix/test/unit/receiver-extraction.test.ts`
- [x] `avmatrix/test/unit/registry-primary-flag.test.ts`
- [x] `avmatrix/test/unit/rel-csv-split.test.ts`
- [x] `avmatrix/test/unit/repo-graph-read-service.test.ts`
- [x] `avmatrix/test/unit/repo-hold-queue-timeout.test.ts`
- [x] `avmatrix/test/unit/repo-manager.test.ts`
- [x] `avmatrix/test/unit/repo-read-executor.test.ts`
- [x] `avmatrix/test/unit/repo-resolver.test.ts`
- [x] `avmatrix/test/unit/resolve-enclosing-owner.test.ts`
- [x] `avmatrix/test/unit/resources.test.ts`
- [x] `avmatrix/test/unit/route-tool-detection.test.ts`
- [x] `avmatrix/test/unit/ruby-self-call.test.ts`
- [x] `avmatrix/test/unit/run-analyze.test.ts`
- [x] `avmatrix/test/unit/runtime-config.test.ts`
- [x] `avmatrix/test/unit/runtime-controller.test.ts`
- [x] `avmatrix/test/unit/schema.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/def-index.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/emit-references.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/finalize-algorithm.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/finalize-orchestrator.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/import-target-adapter.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/method-dispatch-index.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/module-scope-index.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/parse-worker-scope-integration.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/position-index.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/python-scope-captures.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/python-single-pass-parity.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/qualified-name-index.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/registries.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/resolution-phase.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/resolve-type-ref.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/scope-extractor.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/scope-id.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/scope-reference-resolver.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/scope-tree.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/shadow-harness.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/typescript-scope-captures.test.ts`
- [x] `avmatrix/test/unit/scope-resolution/typescript-single-pass-parity.test.ts`
- [x] `avmatrix/test/unit/security.test.ts`
- [x] `avmatrix/test/unit/semantic-chunk-search.test.ts`
- [x] `avmatrix/test/unit/sequential-language-availability.test.ts`
- [x] `avmatrix/test/unit/serve-command.test.ts`
- [x] `avmatrix/test/unit/server.test.ts`
- [x] `avmatrix/test/unit/session-bridge.test.ts`
- [x] `avmatrix/test/unit/settings.test.ts`
- [x] `avmatrix/test/unit/setup-codex.test.ts`
- [x] `avmatrix/test/unit/setup.test.ts`
- [x] `avmatrix/test/unit/shadow/aggregate.test.ts`
- [x] `avmatrix/test/unit/shadow/diff.test.ts`
- [x] `avmatrix/test/unit/shape-check.test.ts`
- [x] `avmatrix/test/unit/shared-type-extractors.test.ts`
- [x] `avmatrix/test/unit/skill-gen.test.ts`
- [x] `avmatrix/test/unit/skip-git-cli.test.ts`
- [x] `avmatrix/test/unit/staleness.test.ts`
- [x] `avmatrix/test/unit/stdout-silence.test.ts`
- [x] `avmatrix/test/unit/structure-processor.test.ts`
- [x] `avmatrix/test/unit/suffix-index-ambiguity.test.ts`
- [x] `avmatrix/test/unit/symbol-resolver.test.ts`
- [x] `avmatrix/test/unit/symbol-table.test.ts`
- [x] `avmatrix/test/unit/text-generator.test.ts`
- [x] `avmatrix/test/unit/tool-runtime-alignment.test.ts`
- [x] `avmatrix/test/unit/tools.test.ts`
- [x] `avmatrix/test/unit/topological-sort.test.ts`
- [x] `avmatrix/test/unit/transitive-include-closure.test.ts`
- [x] `avmatrix/test/unit/tree-sitter-queries.test.ts`
- [x] `avmatrix/test/unit/type-env.test.ts`
- [x] `avmatrix/test/unit/utils.test.ts`
- [x] `avmatrix/test/unit/variable-extraction.test.ts`
- [x] `avmatrix/test/unit/vue-sfc-extractor.test.ts`
- [x] `avmatrix/test/unit/wiki-gated.test.ts`
- [x] `avmatrix/test/unit/wiki-llm-client.test.ts`
- [x] `avmatrix/test/unit/wiki.compat.test.ts`
- [x] `avmatrix/test/unit/wildcard-synthesis.test.ts`
- [x] `avmatrix/test/utils/hook-test-helpers.ts`
- [x] `avmatrix/test/vitest.d.ts`

## Package and build scripts (8)

- [x] `avmatrix/scripts/build-go-runtime.cjs`
- [x] `avmatrix/scripts/build-tree-sitter-proto.cjs`
- [x] `avmatrix/scripts/build.js`
- [x] `avmatrix/scripts/clean-go-source-package.cjs`
- [x] `avmatrix/scripts/patch-tree-sitter-swift.cjs`
- [x] `avmatrix/scripts/phase1-contract-snapshot.cjs`
- [x] `avmatrix/scripts/prepare-go-source-package.cjs`
- [x] `avmatrix/scripts/run-vitest-suite.cjs`

## Claude hook runtime support (1)

- [x] `avmatrix/hooks/claude/avmatrix-hook.cjs` - translated into hidden Go `avmatrix hook claude`
      command, setup switched to the Go hook command, package metadata stopped shipping `hooks`,
      and the legacy CJS hook was deleted with cluster-level validation/evidence.

## Bundled JS/CJS vendor support (4)

- [x] `avmatrix/vendor/leiden/index.cjs`
- [x] `avmatrix/vendor/leiden/utils.cjs`
- [x] `avmatrix/vendor/tree-sitter-proto/bindings/node/index.d.ts`
- [x] `avmatrix/vendor/tree-sitter-proto/bindings/node/index.js`

## Legacy package config (1)

- [x] `avmatrix/vitest.config.ts`

## Root JS/TS config (1)

- [x] `eslint.config.mjs` - classified as root Web/dev lint config; not CLI/backend runtime authority.
