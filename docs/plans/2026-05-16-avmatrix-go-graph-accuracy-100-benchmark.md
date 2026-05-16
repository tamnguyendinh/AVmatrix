# AVmatrix Go Graph Accuracy 100% Benchmark Ledger

Date: 2026-05-16

Source plan: [2026-05-16-avmatrix-go-graph-accuracy-100-plan.md](2026-05-16-avmatrix-go-graph-accuracy-100-plan.md)

This ledger records graph accuracy, graph size, and analyze performance measurements. Unit-test timing and build timing belong in the evidence ledger.

## Baseline - Node/MCP vs Go Local Analyze

- Status: completed.
- Date: 2026-05-16.
- Repo: `E:\AVmatrix-GO`
- Node/MCP benchmark artifact: `.tmp\compare2-node-analyze-20260516-r2.json`
- Go local benchmark artifact: `.tmp\compare2-go-analyze-20260516-r2.json`
- Combined summary artifact: `.tmp\compare2-avmatrix-node-vs-go-20260516.summary.json`

| Metric | Node/MCP | Go local | Delta |
|---|---:|---:|---:|
| Analyze time | 63,447.5 ms | 13,229.8 ms | Go local `4.80x` faster |
| API graph nodes | 17,321 | 18,969 | Go +1,648 |
| API graph relationships | 37,998 | 45,066 | Go +7,068 |
| Files scanned | 684 | 684 | 0 |

Decision:

- Go local is the canonical analyzer for this accuracy plan.
- Node/MCP remains a comparison baseline only.

## Baseline - Go Local Accuracy

- Status: completed.
- Date: 2026-05-16.
- Accuracy artifact: `.tmp\graph-accuracy-node-vs-go-20260516.json`
- Accuracy probe: `.tmp\accuracy_probe.go`
- Source files measured: `403` Go files common to both API graphs.

### Definition Accuracy

| Label | Expected | Go matched | Go recall | Go precision |
|---|---:|---:|---:|---:|
| Const | 320 | 320 | 100.00% | 100.00% |
| Function | 2,955 | 2,955 | 100.00% | 100.00% |
| Interface | 21 | 21 | 100.00% | 100.00% |
| Method | 787 | 787 | 100.00% | 100.00% |
| Struct | 472 | 472 | 100.00% | 100.00% |
| TypeAlias / named non-struct type | 44 | 41 | 93.18% | 100.00% |
| Variable | 7,074 | 6,223 | 87.97% | 100.00% |

Definition total:

| Analyzer | Matched | Expected | Recall | Precision proxy |
|---|---:|---:|---:|---:|
| Go local | 10,819 | 11,673 | 92.69% | 100.00% |

### Import Accuracy

| Analyzer | Expected local import edges | Matched | Recall | Graph candidates | Precision |
|---|---:|---:|---:|---:|---:|
| Go local | 3,456 | 3,456 | 100.00% | 3,456 | 100.00% |

### Direct CALLS Subset

| Analyzer | Expected direct call edges | Matched | Recall | Graph CALLS candidates |
|---|---:|---:|---:|---:|
| Go local | 5,495 | 4,696 | 85.46% | 6,956 |

## Target Accuracy Gates

| Gate | Baseline | Target | Status |
|---|---:|---:|---|
| TypeAlias recall | 93.18% | 100.00% | done |
| Variable recall | 87.97% | 100.00% | done |
| Direct CALLS subset recall | 85.46% | 100.00% | done |
| Local IMPORTS recall | 100.00% | 100.00% | hold |
| Local IMPORTS precision | 100.00% | 100.00% | hold |
| Core definitions recall | 100.00% | 100.00% | hold |
| Core definitions precision | 100.00% | 100.00% | hold |

## Measurement Slots

### Phase 1 - Accuracy Gate Ownership

- Status: completed.
- Benchmark artifact: `.tmp\p1-go-analyze-20260516.json`
- Fresh Go API graph artifact: `.tmp\p1-go-api-graph-20260516.json`
- Accuracy artifact: `.tmp\p1-tracked-gate-baseline-20260516.json`
- Date: 2026-05-16.
- Go local analyze time: `13,365.9 ms`.
- API graph nodes: `19,009`.
- API graph relationships: `45,385`.
- Files scanned: `676`.
- Files parsed: `520`.

| Metric | Before | After | Target |
|---|---:|---:|---:|
| Tracked gate baseline TypeAlias matched | 41 / 44 | 41 / 44 | 41 / 44 before fixes |
| Tracked gate baseline Variable matched | 6,223 / 7,074 | 6,223 / 7,074 | 6,223 / 7,074 before fixes |
| Tracked gate baseline direct CALLS matched | 4,696 / 5,495 | 4,696 / 5,495 | 4,696 / 5,495 before fixes |
| Tracked gate baseline local IMPORTS matched | 3,456 / 3,456 | 3,456 / 3,456 | 3,456 / 3,456 before fixes |

### Phase 2 - TypeAlias Fix

- Status: completed.
- Benchmark artifact: `.tmp\p2-go-analyze-20260516.json`
- Fresh Go API graph artifact: `.tmp\p2-go-api-graph-20260516.json`
- Accuracy artifact: `.tmp\p2-typealias-accuracy-20260516.json`
- Date: 2026-05-16.
- Go local analyze time: `12,766.2 ms`.
- API graph nodes: `19,024`.
- API graph relationships: `45,410`.
- Files scanned: `676`.
- Files parsed: `520`.

| Metric | Before | After | Target |
|---|---:|---:|---:|
| TypeAlias matched | 41 / 44 | 44 / 44 | 44 / 44 |
| TypeAlias recall | 93.18% | 100.00% | 100.00% |

### Phase 3 - Variable Fix

- Status: completed.
- Benchmark artifact: `.tmp\p3-go-analyze-20260516-final.json`
- Fresh Go API graph artifact: `.tmp\p3-go-api-graph-20260516-final.json`
- Accuracy artifact: `.tmp\p3-variable-accuracy-20260516-final.json`
- Date: 2026-05-16.
- Go local analyze time: `13,268.4 ms`.
- API graph nodes: `19,916`.
- API graph relationships: `46,367`.
- Files scanned: `676`.
- Files parsed: `520`.

| Metric | Before | After | Target |
|---|---:|---:|---:|
| Variable matched | 6,225 / 7,076 | 7,080 / 7,080 | 100.00% current gate |
| Variable recall | 87.97% | 100.00% | 100.00% |
| TypeAlias matched | 44 / 44 | 44 / 44 | 44 / 44 |
| Local IMPORTS matched | 3,456 / 3,456 | 3,456 / 3,456 | 3,456 / 3,456 |
| Direct CALLS matched | 4,700 / 5,499 | 4,711 / 5,514 | tracked in Phase 4 |
| Direct CALLS recall | 85.47% | 85.44% | 100.00% in Phase 4 |

### Phase 4 - Direct CALLS Fix

- Status: completed.
- Classification artifact: `.tmp\p4-calls-full-missing-20260516.json`
- Benchmark artifact: `.tmp\p4-go-analyze-20260516-final.json`
- Fresh Go API graph artifact: `.tmp\p4-go-api-graph-20260516-final.json`
- Accuracy artifact: `.tmp\p4-calls-accuracy-20260516-final.json`
- Date: 2026-05-16.
- Go local analyze time: `13,160.5 ms`.
- API graph nodes: `19,984`.
- API graph relationships: `47,622`.
- Files scanned: `676`.
- Files parsed: `520`.

Miss inventory before fix:

| Miss group | Count |
|---|---:|
| Same-directory cross-file direct calls | 803 |
| Imported-package direct calls | 0 |
| Dot-import direct calls | 0 |
| Provider package helper calls | 658 |
| Resolution package helper calls | 51 |
| Enrichment `Apply` package calls | 36 |
| Analyze `Run` / test-helper calls | 22 |
| Package `Load` / storage helper calls | 11 |
| Repo path helper calls | 10 |
| Misc same-package helpers | 15 |

| Metric | Before | After | Target |
|---|---:|---:|---:|
| Direct CALLS matched | 4,711 / 5,514 | 5,520 / 5,520 | 100.00% current gate |
| Direct CALLS recall | 85.44% | 100.00% | 100.00% |
| TypeAlias matched | 44 / 44 | 44 / 44 | 44 / 44 |
| Variable matched | 7,080 / 7,080 | 7,088 / 7,088 | 100.00% current gate |
| Local IMPORTS matched | 3,456 / 3,456 | 3,456 / 3,456 | 3,456 / 3,456 |

### Phase 5 - Final Accuracy Cutover

- Status: completed.
- Benchmark artifact: `.tmp\p5-final-go-analyze-20260516.json`
- Fresh Go API graph artifact: `.tmp\p5-final-go-api-graph-20260516.json`
- Accuracy artifact: `.tmp\p5-final-accuracy-20260516.json`
- Date: 2026-05-16.
- Go local analyze time: `13,580.8 ms`.
- API graph nodes: `19,984`.
- API graph relationships: `47,622`.
- Files scanned: `676`.
- Files parsed: `520`.

| Metric | Result | Target |
|---|---:|---:|
| Function recall | 2,964 / 2,964, `100.00%` | 100.00% |
| Method recall | 791 / 791, `100.00%` | 100.00% |
| Struct recall | 472 / 472, `100.00%` | 100.00% |
| Interface recall | 21 / 21, `100.00%` | 100.00% |
| TypeAlias recall | 44 / 44, `100.00%` | 100.00% |
| Const recall | 321 / 321, `100.00%` | 100.00% |
| Variable recall | 7,088 / 7,088, `100.00%` | 100.00% |
| Local IMPORTS recall | 3,456 / 3,456, `100.00%` | 100.00% |
| Local IMPORTS precision | 3,456 / 3,456, `100.00%` | 100.00% |
| Direct CALLS subset recall | 5,520 / 5,520, `100.00%` | 100.00% |
| Go local analyze time | `13,580.8 ms` | record, no hard target |
| API graph nodes | 19,984 | record |
| API graph relationships | 47,622 | record |
