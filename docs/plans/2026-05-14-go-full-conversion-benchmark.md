# AVmatrix Go Full Conversion Benchmark Ledger

Date: 2026-05-14

Source plan: [2026-05-14-go-full-conversion-plan.md](2026-05-14-go-full-conversion-plan.md)

This ledger records only benchmark data: measured runtime performance, throughput, capacity,
package size, startup/readiness, graph/DB metrics, and conversion inventory counts. Validation
command timings, build timings, unit-test timings, e2e timings, and AVmatrix impact/detect timings
belong in the evidence ledger, not here.

## Phase 0 - Inventory And Ownership

### [P0-A] Fresh Non-Web TS/JS Inventory

- Date: 2026-05-14
- Commit: pending
- Benchmark command: `rg --files -g '*.ts' -g '*.tsx' -g '*.js' -g '*.jsx' -g '*.mjs' -g '*.cjs' -g '!node_modules/**' -g '!dist/**' -g '!build/**' -g '!coverage/**' -g '!.git/**' -g '!.avmatrix/**' -g '!avmatrix-web/dist/**' -g '!avmatrix-launcher/server-bundle/**' -g '!avmatrix/vendor/**'`
- Artifact paths: `.tmp\phase0-inventory-refresh-20260514.json`

| Metric | Result |
| --- | ---: |
| Total TS/JS-family files | 1044 |
| Allowed Web UI files | 118 |
| Generated Web glue files | 1 |
| Non-Web runtime files | 339 |
| Contract authority files | 34 |
| Legacy test harness files | 252 |
| Fixture input files | 290 |
| Package/script glue files | 8 |
| Package test config files | 1 |
| Web dev config files | 1 |
| Unknown ownership files | 0 |
| AVmatrix graph nodes | 34007 |
| AVmatrix graph relationships | 67770 |
| AVmatrix scanned / parsed / failed | 1234 / 1058 / 0 |

Decision:
- Phase 0 inventory is sufficient to start package-script conversion. The first selected implementation slice is `avmatrix/scripts/clean-go-source-package.cjs` under [P6-A].

## Phase 6 - Packaging, Scripts, Hooks, Launcher Support

### [P6-A] Packaging/Script/Hook Slice

- Status: benchmark remeasured 2026-05-15.
- Commit: `b041a65`.
- Benchmark commands:
  - package size: `Get-ChildItem avmatrix-launcher\server-bundle -File`
  - readiness/latency: start `avmatrix-launcher\server-bundle\avmatrix.exe serve --host 127.0.0.1 --port 4848`, poll `/api/info`, then run 30 requests per measured endpoint.
- Artifact paths:
  - `.tmp\p6-packaged-http-benchmark.out.log`
  - `.tmp\p6-packaged-http-benchmark.err.log`

| Metric | Result | Notes |
| --- | ---: | --- |
| `AVmatrixLauncher.exe` size | 6,982,144 bytes | launcher binary |
| `server-bundle\avmatrix.exe` size | 49,482,240 bytes | packaged Go CLI/backend |
| `server-bundle\avmatrix-server.exe` size | 2,053,632 bytes | launcher server wrapper |
| `server-bundle\lbug_shared.dll` size | 18,938,880 bytes | native DB dependency |
| `server-bundle` file size total | 70,474,752 bytes | binary/DLL files only |
| `server-bundle\node.exe` | absent | Node runtime removed from normal package |
| Packaged backend readiness | 592.422ms | `/api/info` first 200 on `127.0.0.1:4848` |
| `/api/info` latency | avg 1.323ms / p95 2.175ms | 30 requests, 70-byte payload |
| `/api/repos` latency | avg 1.257ms / p95 1.676ms | 30 requests, 775-byte payload |

Decision:
- P6 package benchmark is now a product/package/runtime benchmark. Validation timings for the same slice remain only in the evidence ledger.

## Phase 7 - Legacy Test Harness Removal

### [P7-A] Legacy Test Cluster Inventory

- Status: completed.
- Date: 2026-05-14.
- Benchmark command:
  - corrected non-fixture file classification command over `avmatrix/test`.

| Metric | Result | Notes |
| --- | ---: | --- |
| Non-fixture TS test files before batch deletion | 252 | excludes `avmatrix/test/fixtures` |
| CLI/package candidates | 48 | first safe subset selected separately |
| DB/search/embed candidates | 16 | owner confirmation per batch |
| Graph/provider candidates | 98 | owner confirmation per batch |
| HTTP backend candidates | 11 | owner confirmation per batch |
| MCP/tool candidates | 17 | owner confirmation per batch |
| Repo/storage/config candidates | 12 | owner confirmation per batch |
| Setup/skills/editor candidates | 5 | owner confirmation per batch |
| Unclassified candidates | 45 | must be classified before conversion/deletion |

### [P7-TEST-INVENTORY] Legacy Test Harness Reduction

- Status: active.
- Benchmark command: non-fixture test inventory count after each completed test-conversion cluster.
- This is an inventory benchmark only. Per-batch validation timings are recorded in the evidence ledger.

| Point | Remaining Non-Fixture TS Tests | Delta From Prior | Notes |
| --- | ---: | ---: | --- |
| P7-A baseline | 252 | n/a | before batch deletion |
| P7-B1 | 249 | -3 | package/serve/CORS tests removed |
| P7-B2 | 245 | -4 | CLI help/benchmark/wiki/skip-git tests removed |
| P7-B3 | 242 | -3 | CLI clean/index/package metadata tests removed |
| P7-H1 | 239 | -3 | setup/Codex/skills tests removed |
| P7-H2 | 238 | -1 | AI context test removed |
| P7-D1 | 237 | -1 | group matching test removed |
| P7-D2 | 235 | -2 | group config/types tests removed |
| P7-D3 | 233 | -2 | group storage/tool registration tests removed |
| P7-D4 | 228 | -5 | group service/sync/CLI/impact tests removed |
| P7-C1 | 226 | -2 | HTTP graph stream / repo hold queue tests removed |
| P7-D5 | 225 | -1 | MCP tool surface schema test removed |
| P7-D6 | 224 | -1 | impact parser contract test removed |
| P7-D7 | 221 | -3 | MCP runtime dispatch / direct CLI tests removed |
| P7-D8 | 219 | -2 | impact confidence / traversal behavior tests removed |
| P7-G1 | 215 | -4 | repo git / ignore / runtime config tests removed |
| P7-B4 | 213 | -2 | CLI e2e / wiki compatibility tests removed |
| P7-D9 | 212 | -1 | MCP resources static/read test removed |
| P7-F1 | 211 | -1 | LadybugDB schema test removed |
| P7-F2 | 207 | -4 | embedding HTTP/cache/pipeline tests removed |
| P7-F3 | 204 | -3 | semantic chunk and lbug retry/vector tests removed |
| P7-F4 | 185 | -19 | LadybugDB runtime/CSV/security cluster removed; this row uses the exact current `rg --files avmatrix\test | rg "\.(test|spec)\.ts$" | rg -v "^avmatrix\\test\\fixtures\\" | Measure-Object` inventory and reconciles prior ledger drift while deleting 10 active TS test files in this batch |
| P7-C2/G2 | 173 | -12 | local backend, repo runtime/storage, and analyze server/job cluster removed |
| P7-E1 | 162 | -11 | graph/process/pipeline/parser/filesystem tests removed |
| P7-E2/E3/I1 | 138 | -24 | resolution/model, parser/provider/language, and Claude hook tests removed; orphan model helper removed outside the test count |
| P7-E4 | 130 | -8 | framework/ORM/Expo route/community/text/provider parity tests removed |
| P7-D10/P7-E5 | 123 | -7 | MCP impact/detect_changes, route fetch reason, provider qualified-name, and scope-id tests removed; stale native DB runner entries removed |
| P7-D11/P7-E6/P7-H3 | 120 | -3 | group manifest extractor, MRO processor, and skill generator tests removed |
| P7-D12/P7-E7/P7-F5 | 104 | -16 | search BM25/hybrid/core/pool, group extractor/monorepo, JCL/chunking, and call/cross-file tests removed |
| P7-D13/P7-E8/P7-F6/P7-G3/P7-H4/P7-I2 | 87 | -17 | scope index/import target, runtime/staleness/skills/wiki, provider/parser/process, COBOL helper, group bridge, and phase timer tests removed |
| P7-D14/P7-E9/P7-F7 | 61 | -26 | provider resolver/import, route/shape/API, Python MCP tools, and scope def/position/tree tests removed |
| P7-D15/P7-E10/P7-F8 | 44 | -17 | runtime graph/community/staleness/WAL, provider C#/Ruby/Vue, and scope/symbol/type-ref tests removed |
| P7-D16/P7-E11/P7-F9/P7-I3 | 29 | -15 | import/cross-file/query, runtime pipeline/worker, contract snapshot, and scope-audit tests removed |
| P7-D17/P7-E12/P7-F10/P7-I4 | 19 | -10 | resolution/heritage/suffix, shadow parity, enrichment, registry-primary, AST cache, and lazy action tests removed |
| P7-D18/P7-E13/P7-F11/P7-I5 | 0 | -19 | final augmentation, resolver, scope/type/call, runtime/session/stdio/Codex, and COBOL preprocessor tests removed |
| P7-J harness | 0 | n/a | non-fixture `avmatrix/test` files are now 0; only fixture input remains |

### [P7] Runtime/Graph Benchmark Artifacts

These entries are benchmark artifacts tied to completed P7 conversion batches, separate from build/test timing evidence.

| Point | Artifact | Metric | Result |
| --- | --- | --- | ---: |
| P7-F2 | `.tmp\p7-f2-embedding-smoke-20260515221451.json` | embedded nodes | 3 |
| P7-F2 | `.tmp\p7-f2-embedding-smoke-20260515221451.json` | chunks | 3 |
| P7-F2 | `.tmp\p7-f2-embedding-smoke-20260515221451.json` | vector index created | 1 |
| P7-F3 | `.tmp\p7-f3-cli-smoke-20260515223038.json` | parsed files | 1 |
| P7-F3 | `.tmp\p7-f3-cli-smoke-20260515223038.json` | loaded nodes | 4 |
| P7-F3 | `.tmp\p7-f3-cli-smoke-20260515223038.json` | loaded relationships | 5 |
| P7-F4 | `.tmp\p7-lbug-runtime-cluster-refresh-20260515.json` | scanned files | 1184 |
| P7-F4 | `.tmp\p7-lbug-runtime-cluster-refresh-20260515.json` | parsed files | 1011 |
| P7-F4 | `.tmp\p7-lbug-runtime-cluster-refresh-20260515.json` | unsupported files | 173 |
| P7-F4 | `.tmp\p7-lbug-runtime-cluster-refresh-20260515.json` | failed files | 0 |
| P7-F4 | `.tmp\p7-lbug-runtime-cluster-refresh-20260515.json` | graph nodes | 33715 |
| P7-F4 | `.tmp\p7-lbug-runtime-cluster-refresh-20260515.json` | graph relationships | 66989 |
| P7-C2/G2 | `.tmp\p7-cg-local-backend-repo-cluster-refresh-20260515.json` | scanned files | 1174 |
| P7-C2/G2 | `.tmp\p7-cg-local-backend-repo-cluster-refresh-20260515.json` | parsed files | 1001 |
| P7-C2/G2 | `.tmp\p7-cg-local-backend-repo-cluster-refresh-20260515.json` | unsupported files | 173 |
| P7-C2/G2 | `.tmp\p7-cg-local-backend-repo-cluster-refresh-20260515.json` | failed files | 0 |
| P7-C2/G2 | `.tmp\p7-cg-local-backend-repo-cluster-refresh-20260515.json` | graph nodes | 33639 |
| P7-C2/G2 | `.tmp\p7-cg-local-backend-repo-cluster-refresh-20260515.json` | graph relationships | 66895 |
| P7-E1 | `.tmp\p7-e1-graph-process-pipeline-refresh-20260516.json` | scanned files | 1168 |
| P7-E1 | `.tmp\p7-e1-graph-process-pipeline-refresh-20260516.json` | parsed files | 995 |
| P7-E1 | `.tmp\p7-e1-graph-process-pipeline-refresh-20260516.json` | unsupported files | 173 |
| P7-E1 | `.tmp\p7-e1-graph-process-pipeline-refresh-20260516.json` | failed files | 0 |
| P7-E1 | `.tmp\p7-e1-graph-process-pipeline-refresh-20260516.json` | graph nodes | 33678 |
| P7-E1 | `.tmp\p7-e1-graph-process-pipeline-refresh-20260516.json` | graph relationships | 67091 |
| P7-E1 | `.tmp\p7-e1-graph-process-pipeline-refresh-20260516.json` | processes emitted | 560 |
| P7-E1 | `.tmp\p7-e1-graph-process-pipeline-refresh-20260516.json` | communities emitted | 1216 |
| P7-E2/E3/I1 | `.tmp\p7-e23-hook-resolution-provider-refresh-20260516.json` | scanned files | 1146 |
| P7-E2/E3/I1 | `.tmp\p7-e23-hook-resolution-provider-refresh-20260516.json` | parsed files | 973 |
| P7-E2/E3/I1 | `.tmp\p7-e23-hook-resolution-provider-refresh-20260516.json` | unsupported files | 173 |
| P7-E2/E3/I1 | `.tmp\p7-e23-hook-resolution-provider-refresh-20260516.json` | failed files | 0 |
| P7-E2/E3/I1 | `.tmp\p7-e23-hook-resolution-provider-refresh-20260516.json` | graph nodes | 33151 |
| P7-E2/E3/I1 | `.tmp\p7-e23-hook-resolution-provider-refresh-20260516.json` | graph relationships | 66305 |
| P7-E2/E3/I1 | `.tmp\p7-e23-hook-resolution-provider-refresh-20260516.json` | processes emitted | 567 |
| P7-E2/E3/I1 | `.tmp\p7-e23-hook-resolution-provider-refresh-20260516.json` | communities emitted | 1215 |
| P7-E2/E3/I1 | `.tmp\p7-e23-hook-resolution-provider-refresh-20260516.json` | total duration ms | 22718.1 |
| P7-E4 | `.tmp\p7-e4-framework-route-orm-provider-refresh-20260516.json` | scanned files | 1140 |
| P7-E4 | `.tmp\p7-e4-framework-route-orm-provider-refresh-20260516.json` | parsed files | 967 |
| P7-E4 | `.tmp\p7-e4-framework-route-orm-provider-refresh-20260516.json` | unsupported files | 173 |
| P7-E4 | `.tmp\p7-e4-framework-route-orm-provider-refresh-20260516.json` | failed files | 0 |
| P7-E4 | `.tmp\p7-e4-framework-route-orm-provider-refresh-20260516.json` | graph nodes | 33179 |
| P7-E4 | `.tmp\p7-e4-framework-route-orm-provider-refresh-20260516.json` | graph relationships | 66516 |
| P7-E4 | `.tmp\p7-e4-framework-route-orm-provider-refresh-20260516.json` | processes emitted | 563 |
| P7-E4 | `.tmp\p7-e4-framework-route-orm-provider-refresh-20260516.json` | communities emitted | 1220 |
| P7-E4 | `.tmp\p7-e4-framework-route-orm-provider-refresh-20260516.json` | total duration ms | 31876.4 |
| P7-D10/P7-E5 | `.tmp\p7-e5-mcp-provider-route-scope-refresh-20260516.json` | scanned files | 1138 |
| P7-D10/P7-E5 | `.tmp\p7-e5-mcp-provider-route-scope-refresh-20260516.json` | parsed files | 965 |
| P7-D10/P7-E5 | `.tmp\p7-e5-mcp-provider-route-scope-refresh-20260516.json` | unsupported files | 173 |
| P7-D10/P7-E5 | `.tmp\p7-e5-mcp-provider-route-scope-refresh-20260516.json` | failed files | 0 |
| P7-D10/P7-E5 | `.tmp\p7-e5-mcp-provider-route-scope-refresh-20260516.json` | graph nodes | 33222 |
| P7-D10/P7-E5 | `.tmp\p7-e5-mcp-provider-route-scope-refresh-20260516.json` | graph relationships | 66645 |
| P7-D10/P7-E5 | `.tmp\p7-e5-mcp-provider-route-scope-refresh-20260516.json` | processes emitted | 567 |
| P7-D10/P7-E5 | `.tmp\p7-e5-mcp-provider-route-scope-refresh-20260516.json` | communities emitted | 1223 |
| P7-D10/P7-E5 | `.tmp\p7-e5-mcp-provider-route-scope-refresh-20260516.json` | total duration ms | 29181.4 |
| P7-D11/P7-E6/P7-H3 | `.tmp\p7-e6-mro-group-skill-refresh-20260516.json` | scanned files | 1136 |
| P7-D11/P7-E6/P7-H3 | `.tmp\p7-e6-mro-group-skill-refresh-20260516.json` | parsed files | 963 |
| P7-D11/P7-E6/P7-H3 | `.tmp\p7-e6-mro-group-skill-refresh-20260516.json` | unsupported files | 173 |
| P7-D11/P7-E6/P7-H3 | `.tmp\p7-e6-mro-group-skill-refresh-20260516.json` | failed files | 0 |
| P7-D11/P7-E6/P7-H3 | `.tmp\p7-e6-mro-group-skill-refresh-20260516.json` | graph nodes | 33146 |
| P7-D11/P7-E6/P7-H3 | `.tmp\p7-e6-mro-group-skill-refresh-20260516.json` | graph relationships | 66648 |
| P7-D11/P7-E6/P7-H3 | `.tmp\p7-e6-mro-group-skill-refresh-20260516.json` | processes emitted | 567 |
| P7-D11/P7-E6/P7-H3 | `.tmp\p7-e6-mro-group-skill-refresh-20260516.json` | communities emitted | 1216 |
| P7-D11/P7-E6/P7-H3 | `.tmp\p7-e6-mro-group-skill-refresh-20260516.json` | total duration ms | 22886.4 |
| P7-D12/P7-E7/P7-F5 | `.tmp\p7-e7-search-group-jcl-chunk-refresh-20260516.json` | scanned files | 1134 |
| P7-D12/P7-E7/P7-F5 | `.tmp\p7-e7-search-group-jcl-chunk-refresh-20260516.json` | parsed files | 961 |
| P7-D12/P7-E7/P7-F5 | `.tmp\p7-e7-search-group-jcl-chunk-refresh-20260516.json` | unsupported files | 173 |
| P7-D12/P7-E7/P7-F5 | `.tmp\p7-e7-search-group-jcl-chunk-refresh-20260516.json` | failed files | 0 |
| P7-D12/P7-E7/P7-F5 | `.tmp\p7-e7-search-group-jcl-chunk-refresh-20260516.json` | graph nodes | 33522 |
| P7-D12/P7-E7/P7-F5 | `.tmp\p7-e7-search-group-jcl-chunk-refresh-20260516.json` | graph relationships | 67490 |
| P7-D12/P7-E7/P7-F5 | `.tmp\p7-e7-search-group-jcl-chunk-refresh-20260516.json` | processes emitted | 571 |
| P7-D12/P7-E7/P7-F5 | `.tmp\p7-e7-search-group-jcl-chunk-refresh-20260516.json` | communities emitted | 1253 |
| P7-D12/P7-E7/P7-F5 | `.tmp\p7-e7-search-group-jcl-chunk-refresh-20260516.json` | total duration ms | 26315.5 |
| P7-D13/P7-E8/P7-F6/P7-G3/P7-H4/P7-I2 | `.tmp\p7-e8-scope-runtime-provider-cobol-refresh-20260516.json` | scanned files | 1123 |
| P7-D13/P7-E8/P7-F6/P7-G3/P7-H4/P7-I2 | `.tmp\p7-e8-scope-runtime-provider-cobol-refresh-20260516.json` | parsed files | 950 |
| P7-D13/P7-E8/P7-F6/P7-G3/P7-H4/P7-I2 | `.tmp\p7-e8-scope-runtime-provider-cobol-refresh-20260516.json` | unsupported files | 173 |
| P7-D13/P7-E8/P7-F6/P7-G3/P7-H4/P7-I2 | `.tmp\p7-e8-scope-runtime-provider-cobol-refresh-20260516.json` | failed files | 0 |
| P7-D13/P7-E8/P7-F6/P7-G3/P7-H4/P7-I2 | `.tmp\p7-e8-scope-runtime-provider-cobol-refresh-20260516.json` | graph nodes | 33502 |
| P7-D13/P7-E8/P7-F6/P7-G3/P7-H4/P7-I2 | `.tmp\p7-e8-scope-runtime-provider-cobol-refresh-20260516.json` | graph relationships | 67632 |
| P7-D13/P7-E8/P7-F6/P7-G3/P7-H4/P7-I2 | `.tmp\p7-e8-scope-runtime-provider-cobol-refresh-20260516.json` | processes emitted | 567 |
| P7-D13/P7-E8/P7-F6/P7-G3/P7-H4/P7-I2 | `.tmp\p7-e8-scope-runtime-provider-cobol-refresh-20260516.json` | communities emitted | 1268 |
| P7-D13/P7-E8/P7-F6/P7-G3/P7-H4/P7-I2 | `.tmp\p7-e8-scope-runtime-provider-cobol-refresh-20260516.json` | total duration ms | 25479.8 |
| P7-D14/P7-E9/P7-F7 | `.tmp\p7-e9-provider-route-scope-refresh-20260516.json` | scanned files | 1102 |
| P7-D14/P7-E9/P7-F7 | `.tmp\p7-e9-provider-route-scope-refresh-20260516.json` | parsed files | 929 |
| P7-D14/P7-E9/P7-F7 | `.tmp\p7-e9-provider-route-scope-refresh-20260516.json` | unsupported files | 173 |
| P7-D14/P7-E9/P7-F7 | `.tmp\p7-e9-provider-route-scope-refresh-20260516.json` | failed files | 0 |
| P7-D14/P7-E9/P7-F7 | `.tmp\p7-e9-provider-route-scope-refresh-20260516.json` | graph nodes | 32814 |
| P7-D14/P7-E9/P7-F7 | `.tmp\p7-e9-provider-route-scope-refresh-20260516.json` | graph relationships | 67457 |
| P7-D14/P7-E9/P7-F7 | `.tmp\p7-e9-provider-route-scope-refresh-20260516.json` | processes emitted | 572 |
| P7-D14/P7-E9/P7-F7 | `.tmp\p7-e9-provider-route-scope-refresh-20260516.json` | communities emitted | 1284 |
| P7-D14/P7-E9/P7-F7 | `.tmp\p7-e9-provider-route-scope-refresh-20260516.json` | total duration ms | 20918.5 |
| P7-D15/P7-E10/P7-F8 | `.tmp\p7-e10-runtime-provider-scope-refresh-20260516.json` | scanned files | 1087 |
| P7-D15/P7-E10/P7-F8 | `.tmp\p7-e10-runtime-provider-scope-refresh-20260516.json` | parsed files | 914 |
| P7-D15/P7-E10/P7-F8 | `.tmp\p7-e10-runtime-provider-scope-refresh-20260516.json` | unsupported files | 173 |
| P7-D15/P7-E10/P7-F8 | `.tmp\p7-e10-runtime-provider-scope-refresh-20260516.json` | failed files | 0 |
| P7-D15/P7-E10/P7-F8 | `.tmp\p7-e10-runtime-provider-scope-refresh-20260516.json` | graph nodes | 32491 |
| P7-D15/P7-E10/P7-F8 | `.tmp\p7-e10-runtime-provider-scope-refresh-20260516.json` | graph relationships | 67038 |
| P7-D15/P7-E10/P7-F8 | `.tmp\p7-e10-runtime-provider-scope-refresh-20260516.json` | processes emitted | 572 |
| P7-D15/P7-E10/P7-F8 | `.tmp\p7-e10-runtime-provider-scope-refresh-20260516.json` | communities emitted | 1277 |
| P7-D15/P7-E10/P7-F8 | `.tmp\p7-e10-runtime-provider-scope-refresh-20260516.json` | total duration ms | 20621.7 |
| P7-D16/P7-E11/P7-F9/P7-I3 | `.tmp\p7-e11-import-runtime-contract-refresh-20260516.json` | scanned files | 1078 |
| P7-D16/P7-E11/P7-F9/P7-I3 | `.tmp\p7-e11-import-runtime-contract-refresh-20260516.json` | parsed files | 905 |
| P7-D16/P7-E11/P7-F9/P7-I3 | `.tmp\p7-e11-import-runtime-contract-refresh-20260516.json` | unsupported files | 173 |
| P7-D16/P7-E11/P7-F9/P7-I3 | `.tmp\p7-e11-import-runtime-contract-refresh-20260516.json` | failed files | 0 |
| P7-D16/P7-E11/P7-F9/P7-I3 | `.tmp\p7-e11-import-runtime-contract-refresh-20260516.json` | graph nodes | 32436 |
| P7-D16/P7-E11/P7-F9/P7-I3 | `.tmp\p7-e11-import-runtime-contract-refresh-20260516.json` | graph relationships | 67017 |
| P7-D16/P7-E11/P7-F9/P7-I3 | `.tmp\p7-e11-import-runtime-contract-refresh-20260516.json` | processes emitted | 573 |
| P7-D16/P7-E11/P7-F9/P7-I3 | `.tmp\p7-e11-import-runtime-contract-refresh-20260516.json` | communities emitted | 1284 |
| P7-D16/P7-E11/P7-F9/P7-I3 | `.tmp\p7-e11-import-runtime-contract-refresh-20260516.json` | total duration ms | 20068.6 |
| P7-D17/P7-E12/P7-F10/P7-I4 | `.tmp\p7-e12-resolution-shadow-enrichment-refresh-20260516.json` | scanned files | 1080 |
| P7-D17/P7-E12/P7-F10/P7-I4 | `.tmp\p7-e12-resolution-shadow-enrichment-refresh-20260516.json` | parsed files | 907 |
| P7-D17/P7-E12/P7-F10/P7-I4 | `.tmp\p7-e12-resolution-shadow-enrichment-refresh-20260516.json` | unsupported files | 173 |
| P7-D17/P7-E12/P7-F10/P7-I4 | `.tmp\p7-e12-resolution-shadow-enrichment-refresh-20260516.json` | failed files | 0 |
| P7-D17/P7-E12/P7-F10/P7-I4 | `.tmp\p7-e12-resolution-shadow-enrichment-refresh-20260516.json` | graph nodes | 32689 |
| P7-D17/P7-E12/P7-F10/P7-I4 | `.tmp\p7-e12-resolution-shadow-enrichment-refresh-20260516.json` | graph relationships | 67624 |
| P7-D17/P7-E12/P7-F10/P7-I4 | `.tmp\p7-e12-resolution-shadow-enrichment-refresh-20260516.json` | processes emitted | 573 |
| P7-D17/P7-E12/P7-F10/P7-I4 | `.tmp\p7-e12-resolution-shadow-enrichment-refresh-20260516.json` | communities emitted | 1313 |
| P7-D17/P7-E12/P7-F10/P7-I4 | `.tmp\p7-e12-resolution-shadow-enrichment-refresh-20260516.json` | total duration ms | 20720.2 |
| P7-D17/P7-E12/P7-F10/P7-I4 | `.tmp\p7-e12-resolution-shadow-enrichment-refresh-20260516.json` | max observed sys bytes | 529408248 |
| P7-D18/P7-E13/P7-F11/P7-I5 | `.tmp\p7-final-legacy-test-retirement-refresh-20260516.json` | scanned files | 1076 |
| P7-D18/P7-E13/P7-F11/P7-I5 | `.tmp\p7-final-legacy-test-retirement-refresh-20260516.json` | parsed files | 903 |
| P7-D18/P7-E13/P7-F11/P7-I5 | `.tmp\p7-final-legacy-test-retirement-refresh-20260516.json` | unsupported files | 173 |
| P7-D18/P7-E13/P7-F11/P7-I5 | `.tmp\p7-final-legacy-test-retirement-refresh-20260516.json` | failed files | 0 |
| P7-D18/P7-E13/P7-F11/P7-I5 | `.tmp\p7-final-legacy-test-retirement-refresh-20260516.json` | graph nodes | 32761 |
| P7-D18/P7-E13/P7-F11/P7-I5 | `.tmp\p7-final-legacy-test-retirement-refresh-20260516.json` | graph relationships | 68116 |
| P7-D18/P7-E13/P7-F11/P7-I5 | `.tmp\p7-final-legacy-test-retirement-refresh-20260516.json` | processes emitted | 585 |
| P7-D18/P7-E13/P7-F11/P7-I5 | `.tmp\p7-final-legacy-test-retirement-refresh-20260516.json` | communities emitted | 1328 |
| P7-D18/P7-E13/P7-F11/P7-I5 | `.tmp\p7-final-legacy-test-retirement-refresh-20260516.json` | total duration ms | 19586.8 |
| P7-J | `.tmp\p7-final-harness-retirement-refresh-20260516.json` | scanned files | 1066 |
| P7-J | `.tmp\p7-final-harness-retirement-refresh-20260516.json` | parsed files | 895 |
| P7-J | `.tmp\p7-final-harness-retirement-refresh-20260516.json` | unsupported files | 171 |
| P7-J | `.tmp\p7-final-harness-retirement-refresh-20260516.json` | failed files | 0 |
| P7-J | `.tmp\p7-final-harness-retirement-refresh-20260516.json` | graph nodes | 32639 |
| P7-J | `.tmp\p7-final-harness-retirement-refresh-20260516.json` | graph relationships | 67926 |
| P7-J | `.tmp\p7-final-harness-retirement-refresh-20260516.json` | processes emitted | 585 |
| P7-J | `.tmp\p7-final-harness-retirement-refresh-20260516.json` | communities emitted | 1327 |
| P7-J | `.tmp\p7-final-harness-retirement-refresh-20260516.json` | total duration ms | 20249.4 |
| P7-J | `.tmp\p7-final-harness-retirement-refresh-20260516.json` | max observed sys bytes | 576545016 |

## Phase 8 - Final Cutover Audit

### [P8-A] Final Cutover Benchmark

- Status: completed before final commit.
- Date: 2026-05-16 local.
- Artifacts:
  - `.tmp\p8-final-non-web-tsjs-inventory-20260516.txt`
  - `.tmp\p8-npm-pack-dry-run-20260516.clean.json`
  - `.tmp\p8-smoke-latency-20260516.json`
  - `.tmp\p8-cli-smoke-analyze-20260516.json`
  - `.tmp\p8-e2e-fixture-analyze-20260516.json`
  - `.tmp\p8-package-source-cutover-refresh-20260516.json`
  - `.tmp\p8-package-source-cutover-detect-20260516.txt`

| Metric | Result | Notes |
| --- | ---: | --- |
| Final non-Web TS/JS inventory | 1 | only `eslint.config.mjs`; root Web/dev lint config |
| Non-Web CLI/backend/runtime TS/JS inventory | 0 | `avmatrix/src`, `avmatrix-shared`, `avmatrix/scripts`, legacy vendor glue deleted |
| npm package tarball size | 16,270,066 bytes | `npm pack --dry-run --json` |
| npm package unpacked size | 70,081,542 bytes | includes Go runtime binary, native DLL, generated `go-src`, skills |
| npm package entry count | 267 | no `dist`, `scripts`, `vendor`, or `avmatrix-shared` entries |
| MCP smoke latency | 51.9ms | initialize + tools/list through Go runtime |
| CLI query smoke latency | 40.1ms | isolated smoke repo |
| CLI context smoke latency | 39.7ms | isolated smoke repo |
| CLI impact smoke latency | 40.7ms | isolated smoke repo |
| Final staged detect_changes latency | 2,575.1ms | 408 staged files, 146 changed symbols, 7 affected processes, risk high |
| Small CLI smoke scanned / parsed / failed | 1 / 1 / 0 | `.tmp\p8-cli-smoke-analyze-20260516.json` |
| Small CLI smoke graph nodes / relationships | 5 / 6 | isolated one-file Go repo |
| Web e2e fixture scanned / parsed / failed | 5 / 3 / 0 | fixture used for accepted browser e2e |
| Web e2e fixture graph nodes / relationships | 21 / 40 | includes process-capable TS call chain and `package.json` |
| Web e2e fixture processes / communities | 3 / 2 | proves Process panel data path |
| Current repo scanned / parsed / failed | 684 / 517 / 0 | after deleting legacy TS/shared/script/vendor source |
| Current repo graph nodes / relationships | 18,969 / 45,066 | final refresh after source cutover |
| Current repo processes / communities | 579 / 872 | Go analyzer output |
| Current repo total analyze duration | 12,615.8ms | `totalDuration` from benchmark JSON |
| Current repo parser bytes | 3,294,109 | final refresh parser metric |
| Current repo DB fallback inserts | 0 | no fallback inserts |
| Current repo skipped relationships | 0 | no skipped relationships |
| Current repo max observed sys memory | 323,995,896 bytes | benchmark memory metric |

Decision:
- Final benchmark proves the package/source cutover removed non-Web TypeScript runtime authority, reduced indexed repo size accordingly, preserved Go analyze/DB load health, and kept MCP/CLI/Web e2e paths operational through the Go runtime.
