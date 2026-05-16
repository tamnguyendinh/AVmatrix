# AVmatrix-Go vs AVmatrix-main Analyze Benchmarks

Date: 2026-05-16

This report compares both engines on two repos:

| Scenario | Repo | Profile |
|---|---|---|
| AVmatrix-GO self analyze | `E:\AVmatrix-GO` | Go-heavy / Go runtime home repo |
| Website analyze | `E:\Website` | TypeScript-heavy / Next.js-style app repo |

Raw benchmark payloads are embedded in `reports\benchmark\2026-05-16-120908-avmatrix-go-self-analyze.json` under `scenarios.<name>.runs`.

## Headline Summary

| Scenario | avmatrix-go | avmatrix-main | Go speedup | Go nodes | main nodes | Go relationships | main relationships |
|---|---:|---:|---:|---:|---:|---:|---:|
| AVmatrix-GO | 13,588.1 ms | 63,215.2 ms | 4.65x | 19,984 | 17,539 | 47,622 | 39,019 |
| Website | 20,763.6 ms | 70,428.1 ms | 3.39x | 26,081 | 18,607 | 48,163 | 34,055 |

## Commands

### AVmatrix-GO / avmatrix-go

~~~powershell
.\avmatrix-launcher\server-bundle\avmatrix.exe analyze E:\AVmatrix-GO --force --skip-agents-md --no-stats --benchmark-json reports\benchmark\2026-05-16-120908-avmatrix-go-self-analyze.json --benchmark-label 2026-05-16-120908-avmatrix-go-self-analyze
~~~

### AVmatrix-GO / avmatrix-main

~~~powershell
node E:\avmatrix-main\avmatrix\dist\cli\index.js analyze E:\AVmatrix-GO --force --skip-agents-md --no-stats --benchmark-json reports\benchmark\2026-05-16-120908-avmatrix-main-self-analyze.json --benchmark-label 2026-05-16-120908-avmatrix-main-on-avmatrix-go
~~~

### Website / avmatrix-main

~~~powershell
node E:\avmatrix-main\avmatrix\dist\cli\index.js analyze E:\Website --force --skip-agents-md --no-stats --benchmark-json reports\benchmark\2026-05-16-120908-avmatrix-main-on-website.json --benchmark-label 2026-05-16-120908-avmatrix-main-on-website
~~~

### Website / avmatrix-go

~~~powershell
.\avmatrix-launcher\server-bundle\avmatrix.exe analyze E:\Website --force --skip-agents-md --no-stats --benchmark-json reports\benchmark\2026-05-16-120908-avmatrix-go-on-website.json --benchmark-label 2026-05-16-120908-avmatrix-go-on-website
~~~

## AVmatrix-GO Detail

| Metric | avmatrix-go | avmatrix-main | Difference |
|---|---:|---:|---:|
| Total analyze time | 13,588.1 ms | 63,215.2 ms | avmatrix-go is 4.65x faster |
| API graph nodes | 19,984 | 17,539 | Go +2445 |
| API graph relationships | 47,622 | 39,019 | Go +8603 |
| Files scanned | 676 | 678 | Go -2 |
| Files parsed / parseable | 520 | 520 | 0 |
| Parse phase | 5,586.7 ms | 12,003.0 ms | main is 2.15x slower |
| DB / Ladybug load | 4,902.5 ms | 17,539.5 ms | main is 3.58x slower |
| Resolved calls | 12,792 | 585 | Go +12207 |
| Resolved references | 26,541 | 1,089 | Go +25452 |

### AVmatrix-GO Coverage Note

`avmatrix-main` reports Go scope extraction as unavailable for the Go files in `E:\AVmatrix-GO`:

| Field | Value |
|---|---:|
| Go parseable files | 406 |
| Go ScopeIR AST reused files | 0 |
| Go no-hook files | 406 |
| Go ScopeIR coverage | 0% |
| Go scope resolution reference sites | 0 |

So this scenario is useful for wall-clock and graph-size comparison, but it is biased toward `avmatrix-go` for Go resolution quality.

## Website Detail

| Metric | avmatrix-go | avmatrix-main | Difference |
|---|---:|---:|---:|
| Total analyze time | 20,763.6 ms | 70,428.1 ms | avmatrix-go is 3.39x faster |
| API graph nodes | 26,081 | 18,607 | Go +7474 |
| API graph relationships | 48,163 | 34,055 | Go +14108 |
| Files scanned | 1,870 | 1,870 | Go 0 |
| Files parsed / parseable | 998 | 998 | 0 |
| Parse phase | 6,316.1 ms | 27,611.0 ms | main is 4.37x slower |
| DB / Ladybug load | 4,057.5 ms | 14,604.8 ms | main is 3.60x slower |
| Resolved calls | 11,303 | 6,037 | Go +5266 |
| Resolved references | 20,849 | 10,232 | Go +10617 |
| Resolved accesses | 3 | 755 | Go -752 |
| Resolved type references | 9,543 | 3,440 | Go +6103 |

### Website Coverage Note

`avmatrix-main` reports full ScopeIR AST reuse for the JavaScript and TypeScript files in `E:\Website`:

| Language | Parseable files | AST reused | ScopeIR coverage | Reference sites |
|---|---:|---:|---:|---:|
| JavaScript | 7 | 7 | 100% | 523 |
| TypeScript | 991 | 991 | 100% | 73572 |

This makes the Website scenario a better cross-runtime comparison for a TypeScript-heavy repo. On this repo, `avmatrix-go` is still 3.39x faster and emits a larger graph.

## Worktree Note

`coder.md` was already untracked before this benchmark work and was not modified.
