# AVmatrix Go Graph Accuracy 100% Plan

Date: 2026-05-16

Status: completed

Companion files:

- Benchmark ledger: [2026-05-16-avmatrix-go-graph-accuracy-100-benchmark.md](2026-05-16-avmatrix-go-graph-accuracy-100-benchmark.md)
- Evidence ledger: [2026-05-16-avmatrix-go-graph-accuracy-100-evidence.md](2026-05-16-avmatrix-go-graph-accuracy-100-evidence.md)

## Rules

1. Use AVmatrix for codebase analysis and impact checks while working on this plan.
2. As each task is completed, update the corresponding checklist item immediately.
3. Run a full build before testing; the test suite must include an e2e test.
4. Record benchmark results as each benchmarkable task is completed. Benchmarkable means measured product/runtime performance, capacity, package/startup size, graph/DB throughput, or conversion-inventory counts; build/test/e2e timings are validation evidence unless the slice changes those systems.
5. Record evidence as each evidenced task is completed.
6. commit doc only do not use avmatrix.
7. After each completed implementation slice, commit the work, then continue until the full plan is complete.


## Rule Clarifications

### Full Build and Test Suite

The rule "Run a full build before testing; the test suite must include an e2e test" means:

- first run the full build, including Go runtime, launcher, and Web UI build;
- then run the test suite for the completed slice;
- that test suite must include at least one e2e test;
- e2e is one required component of the test suite, not a replacement for focused, integration, or broader regression tests;
- do not treat a single easy Playwright/browser test as sufficient unless it exercises the slice's real end-to-end path.

### E2E Selection

AVmatrix has two mechanisms:

- Primary mechanism: CLI/MCP runtime for AI/codebase understanding.
- Secondary mechanism: Web UI as a viewer for human inspection of graph/runtime results.

E2E must be selected by the mechanism affected by the slice:

- For CLI, MCP, indexing, graph, query, context, impact, detect-changes, repo storage, setup, package/runtime, or core conversion slices, the e2e must be CLI/MCP/runtime e2e. It should exercise a real repo/index/graph/tool path through the built Go runtime.
- For Web viewer, HTTP bridge, frontend state, graph rendering, repo selection, analyze-from-UI, or session viewer slices, the e2e must be FE-backend e2e through the Go backend and browser UI.
- UI-only/browser tests that only click buttons or check availability are not valid as the plan's required e2e unless the slice itself is strictly UI-only and the test still proves the complete affected behavior.
- Web e2e must not be used as proof for CLI/MCP/setup behavior. CLI/MCP/setup behavior requires CLI/MCP/config tests or smoke evidence.

### Benchmark Meaning

Benchmark ledger entries must be real benchmark measurements:

- runtime/product performance;
- graph/database/index throughput;
- MCP/CLI latency or throughput;
- memory/capacity/package/startup size;
- conversion inventory counts when the task is inventory-measurable.

Build time, unit-test duration, e2e duration, command wall time for validation, and "how long the agent took" are evidence only, not benchmark entries.

### Batch Size

Work must proceed by meaningful clusters:

- convert/delete a full related group of files/tests before build/test;
- do not build/test/commit after each individual file unless the whole cluster is genuinely one file;
- after each cluster passes validation and is committed, continue immediately to the next open checklist item until the plan is complete or a real blocker is recorded.

## Phase Jump Rule

Every executable work item is a checklist item. If a task must jump to another phase, write the jump directly inside the same checklist item.

Use this format:

```md
- [ ] [P2-A] Convert package X. Phase jump: blocked by ownership uncertainty in Y; jump to [P5-C] to convert safe helper Z, then return to [P2-A].
- [ ] [P5-C] Convert helper Z. Phase jump return: after completion, return to [P2-A].
```

Do not create separate loose notes for phase jumps. The checklist is the ledger.

## Goal

Make the Go local AVmatrix graph for `E:\AVmatrix-GO` reach `100%` on the measured graph-accuracy gates for this repo.

Canonical target: local Go binary `avmatrix\bin\avmatrix.exe`.

Node/MCP AVmatrix is kept only as a comparison baseline. Do not spend implementation work on Node/MCP unless a later plan explicitly asks for parity.

## Plan-Specific Rules

1. Keep all plan, benchmark, and evidence updates in `docs/plans/`.
2. Update the benchmark ledger after every measurable graph-accuracy run.
3. Update the evidence ledger after every completed implementation cluster.
4. Work by accuracy cluster, not by isolated one-line edits.
5. Do not mark a cluster complete until the relevant accuracy gate is rerun and recorded.
6. A cluster that changes graph extraction or resolution must include focused tests and one full local analyze on `E:\AVmatrix-GO`.
7. The final gate is `100%` for all measured Go local accuracy categories in this plan.

## Baseline

Baseline raw accuracy artifact: `.tmp\graph-accuracy-node-vs-go-20260516.json`

Go local accuracy gaps:

| Gate | Current | Target |
|---|---:|---:|
| TypeAlias / named non-struct type recall | 41 / 44, `93.18%` | 44 / 44, `100.00%` |
| Variable recall | 6,223 / 7,074, `87.97%` | 7,074 / 7,074, `100.00%` |
| Direct CALLS subset recall | 4,696 / 5,495, `85.46%` | 5,495 / 5,495, `100.00%` |

Already at `100%` on Go local:

- `Const`
- `Function`
- `Interface`
- `Method`
- `Struct`
- local `IMPORTS` recall
- local `IMPORTS` precision

## Accuracy Method

The current accuracy probe builds source-derived ground truth from the Go standard parser over the `403` Go files present in both API graphs.

Measured layers:

- named definitions: `Function`, `Method`, `Struct`, `Interface`, `TypeAlias`, `Const`, `Variable`;
- local package `IMPORTS`, resolved through discovered `go.mod` modules and expanded to non-test Go files in the imported package directory;
- high-confidence direct `CALLS`: same-package direct function calls, dot-import function calls, and imported-package function calls.

Before final completion, move or rewrite the temporary probe into a tracked test/tool location so the gate is repeatable without relying on `.tmp`.

## Phase 1 - Accuracy Gate Ownership

- [x] [P1-A] Promote the temporary accuracy probe into a tracked repo-owned gate. Owner: `internal/graphaccuracy` for reusable gate logic and `cmd/graph-accuracy-probe` for the command wrapper.
- [x] [P1-B] Make the tracked gate regenerate or consume the Go local API graph deterministically. The report command runs `avmatrix\bin\avmatrix.exe analyze --force`, copies the fresh graph snapshot to `.tmp\p1-go-api-graph-20260516.json`, and compares that artifact instead of reading `.avmatrix\graph.json` as authority.
- [x] [P1-C] Add documented commands for the tracked gate with two modes. `report` exits `0` while known gates are below target; `enforce` exits `1` and emits missing examples when any measured Go-local gate is below target.
- [x] [P1-D] Record the tracked gate baseline in report mode in the benchmark and evidence ledgers. Result matches the current baseline before feature fixes begin: `TypeAlias 41/44`, `Variable 6223/7074`, direct `CALLS 4696/5495`, local `IMPORTS 3456/3456`.

## Phase 2 - TypeAlias Accuracy

- [x] [P2-A] Fix Go named non-struct type extraction for alias declarations. The Go provider now handles tree-sitter `type_alias` nodes, and the Go local graph emits the known `type Request = sfc.Request` declarations as `TypeAlias`.
- [x] [P2-B] Add focused TypeAlias coverage around Go provider extraction and graph emission. `TestExtractGoTypeAliasDeclarations` covers `type Request = sfc.Request`, ordinary named non-struct aliases, struct declarations, interface declarations, and interface methods.
- [x] [P2-C] Run the TypeAlias validation slice and record it. TypeAlias gate result: `44 / 44`, `100.00%`.

## Phase 3 - Variable Accuracy

- [x] [P3-A] Classify the missing Go local `Variable` examples by binding form and write the classification into the evidence ledger. The misses grouped into `range_clause` key/value bindings first, then the remaining `type_switch_statement` aliases and `receive_statement` aliases after the range fix.
- [x] [P3-B] Expand Go variable extraction by binding family. The Go provider now emits `Variable` definitions for `range` declarations, type-switch aliases, and receive-statement aliases while preserving the existing short-var, init-var, table-test, and helper-local coverage.
- [x] [P3-C] Add focused variable extraction tests for every binding family closed in [P3-B]. `TestExtractGoRangeVariableDeclarations` covers production and nested range forms; `TestExtractGoSwitchAndReceiveVariableDeclarations` covers type-switch and select receive forms.
- [x] [P3-D] Run the Variable validation slice and record it. Variable gate result: `7,080 / 7,080`, `100.00%`. Required evidence recorded: focused tests, full build before test, CLI/MCP runtime e2e, fresh Go local analyze, API graph export, tracked accuracy gate output, benchmark ledger update, evidence ledger update, and detect changes.

## Phase 4 - Direct CALLS Accuracy

- [x] [P4-A] Classify the missing direct-call edges by source/target package pattern and write the classification into the evidence ledger. The current full miss artifact had `803` misses: `803 / 803` same-directory Go cross-file direct function calls, with `0` imported-package and `0` dot-import misses.
- [x] [P4-B] Strengthen Go direct identifier call resolution. Direct identifier calls now resolve through lexical scope, same file, then a Go-only same-package function lookup before global fallback; this covers provider helpers, resolution helpers, test helpers, `Apply`, `Load`, `Run`, and variadic helpers.
- [x] [P4-C] Strengthen imported package and dot-import function call resolution. Phase jump: no implementation change was needed because the current full miss artifact showed `0` imported-package and `0` dot-import misses while local `IMPORTS` recall/precision stayed at `100.00%`; keep existing imported member and dot-import coverage in focused tests.
- [x] [P4-D] Add focused direct-call resolution tests for every miss family closed in [P4-B] and [P4-C]. `TestResolveGoSamePackageDirectCallAcrossFilesBeforeGlobalAmbiguity` covers same-package cross-file lookup, global ambiguity, and variadic/arity metadata mismatch; existing import tests continue to cover imported package member calls.
- [x] [P4-E] Run the direct `CALLS` validation slice and record it. Direct `CALLS` gate result: `5,520 / 5,520`, `100.00%`. Required evidence recorded: focused resolution tests, full build before test, CLI/MCP runtime e2e, fresh Go local analyze, API graph export, tracked accuracy gate output, benchmark ledger update, evidence ledger update, and detect changes.

## Phase 5 - Final Accuracy Cutover

- [x] [P5-A] Run the final Go local graph accuracy gate on `E:\AVmatrix-GO`. Final result is `100.00%` for `TypeAlias`, `Variable`, direct `CALLS` subset, local `IMPORTS` recall, local `IMPORTS` precision, and all core definition labels.
- [x] [P5-B] Record final graph size, analyze performance, and accuracy metrics in the benchmark ledger. Final Go local analyze time: `13,580.8 ms`; API graph: `19,984` nodes and `47,622` relationships.
- [x] [P5-C] Record final evidence. Commands, artifacts, focused tests, full build, e2e proof, fresh API graph export, tracked accuracy gate output, and implementation detect-changes evidence are recorded.
- [x] [P5-D] Close the plan only after benchmark and evidence ledgers agree with the final tracked accuracy artifact. Final tracked artifact `.tmp\p5-final-accuracy-20260516.json` agrees with both ledgers and the gate passed.

## Ledger

| ID | Area | Scope | Target | Benchmark | Evidence | Commit | Status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| P1-A | Accuracy gate | tracked probe owner | tracked gate owner selected | n/a | recorded | `288c093` | done |
| P1-B | Accuracy gate | API graph input | no stale `.avmatrix\graph.json` dependency | recorded | recorded | `288c093` | done |
| P1-C | Accuracy gate | report/enforce commands | report records baseline; enforce fails below target | n/a | recorded | `288c093` | done |
| P1-D | Accuracy gate | baseline rerun | report-mode baseline matches current artifact | recorded | recorded | `288c093` | done |
| P2-A | TypeAlias | Go provider extraction | `44 / 44` TypeAlias recall | recorded | recorded | `c6d2921` | done |
| P2-B | TypeAlias | focused tests | alias and adjacent label tests pass | n/a | recorded | `c6d2921` | done |
| P2-C | TypeAlias | validation slice | TypeAlias gate `100.00%` | recorded | recorded | `c6d2921` | done |
| P3-A | Variable | miss classification | variable miss families documented | recorded | recorded | `cffce45` | done |
| P3-B | Variable | extraction coverage | `7,080 / 7,080` Variable recall | recorded | recorded | `cffce45` | done |
| P3-C | Variable | focused tests | all variable binding family tests pass | n/a | recorded | `cffce45` | done |
| P3-D | Variable | validation slice | Variable gate `100.00%` | recorded | recorded | `cffce45` | done |
| P4-A | Direct CALLS | miss classification | `803` misses grouped by pattern | recorded | recorded | `7cefe83` | done |
| P4-B | Direct CALLS | same-package resolution | direct same-package calls resolved | recorded | recorded | `7cefe83` | done |
| P4-C | Direct CALLS | import/dot-import resolution | no current import/dot-import misses; imports held | recorded | recorded | `7cefe83` | done |
| P4-D | Direct CALLS | focused tests | direct-call miss family tests pass | n/a | recorded | `7cefe83` | done |
| P4-E | Direct CALLS | validation slice | direct `CALLS` gate `100.00%` | recorded | recorded | `7cefe83` | done |
| P5-A | Final gate | Go local graph accuracy | all measured gates `100.00%` | recorded | recorded | `3bca698` | done |
| P5-B | Final benchmark | graph size/performance/accuracy | final metrics recorded | recorded | recorded | `3bca698` | done |
| P5-C | Final evidence | commands/artifacts/tests/e2e | final proof recorded | recorded | recorded | `3bca698` | done |
| P5-D | Final closure | ledger consistency | plan closed | recorded | recorded | `3bca698` | done |

## Definition Of Done

- Every checklist item above is closed or has an inline blocker with a phase jump.
- The tracked accuracy gate exists outside `.tmp` and is documented.
- Go local graph accuracy on `E:\AVmatrix-GO` is `100.00%` for every measured gate in this plan.
- The benchmark ledger contains baseline and final accuracy, graph size, and analyze performance.
- The evidence ledger contains commands, artifacts, focused tests, full build, e2e proof, API graph export, detect changes, and commit hashes for completed implementation slices.
