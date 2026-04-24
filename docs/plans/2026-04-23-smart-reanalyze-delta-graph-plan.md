# Smart Re-Analyze Delta Graph Plan

Date: 2026-04-23  
Scope: `avmatrix/` primary, `avmatrix-web/` secondary  
Status: Proposed - revised fast-path constraints

## Goal

Khi user bấm `Re-analyze` trên web UI:

- không mặc định full rebuild toàn repo nếu có một delta path đúng về semantics
- tách `re-analyze` thành một orchestration path riêng để lifecycle của nó không bị trộn vào full analyze hiện tại
- vẫn giữ nguyên kiến trúc ingestion hiện tại của AVmatrix
- dùng Git delta + baseline index gần nhất để chọn giữa `No-op`, `Delta analyze`, và `Fallback to full analyze`
- delta fast path phải giảm work thật: parse / resolve trên `changedFiles` và `affectedFiles`, không rebuild toàn LadybugDB, không rewrite full snapshot
- ưu tiên correctness trước performance tuyệt đối ở V1

Plan này **không** được đổi nguyên lý build graph của tool.

Plan này cũng **không** được biến `re-analyze` thành "partial parse + full persist". Nếu fast path vẫn phải ghi lại toàn bộ query store hoặc toàn bộ snapshot thì đó là fallback/full behavior, không phải delta fast path.

## Current Behavior Confirmed In Code

### 1. Web `Re-analyze` hiện tại đang ép full analyze

Code hiện tại đang:

- web `Re-analyze` gọi `/api/analyze` với `force: true`
- backend chỉ forward `force` / `embeddings` sang analyze worker
- `runFullAnalysis()` chỉ early-return khi `existingMeta.lastCommit === currentCommit` và `force === false`

Nguồn:

- [api.ts](/F:/AVmatrix-main/avmatrix/src/server/api.ts:1346)
- [Header.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/Header.tsx:258)
- [run-analyze.ts](/F:/AVmatrix-main/avmatrix/src/core/run-analyze.ts:151)

Hệ quả:

- web `Re-analyze` hiện không có semantic riêng; nó chỉ là full analyze forced run
- modified, deleted, renamed, staged, và untracked đều chưa được phân loại thành delta / no-op / fallback

### 2. Full analyze hiện tại tạo graph mới từ đầu rồi thay on-disk query state

Code hiện tại đang:

- tạo `KnowledgeGraph` mới trong `runPipelineFromRepo()`
- có mutation API sẵn như `removeNodesByFile()`
- stream toàn bộ graph sang LadybugDB qua `loadGraphToLbug()`
- ghi `meta.json` ở cuối path thành công

Nguồn:

- [pipeline.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline.ts:94)
- [types.ts](/F:/AVmatrix-main/avmatrix/src/core/graph/types.ts:12)
- [graph.ts](/F:/AVmatrix-main/avmatrix/src/core/graph/graph.ts:48)
- [lbug-adapter.ts](/F:/AVmatrix-main/avmatrix/src/core/lbug/lbug-adapter.ts:376)
- [run-analyze.ts](/F:/AVmatrix-main/avmatrix/src/core/run-analyze.ts:323)

Hệ quả:

- delta merge có thể reuse shared mutable graph model hiện có
- suy ra từ code hiện tại: chưa có baseline artifact hạng nhất nào có thể load ngược lại thành `KnowledgeGraph`; V1 phải bổ sung đúng phần này

### 3. Các phase quan trọng phải giữ nguyên semantics hiện tại

Kiến trúc hiện tại theo code thật:

- runner chạy dependency-ordered phases trên shared mutable graph
- `parse` là orchestration lớn, không chỉ parse AST
- `routes` đọc repo-wide `allPaths`
- `crossFile` reprocess theo topological import order
- `mro` chạy sau `crossFile`
- `communities` và `processes` là whole-graph derived phases
- FTS và embeddings được rebuild / restored trong full analyze persistence path

Nguồn:

- [pipeline.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline.ts:72)
- [pipeline phase types](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline-phases/types.ts:23)
- [parse.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline-phases/parse.ts:68)
- [routes.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline-phases/routes.ts:57)
- [cross-file.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline-phases/cross-file.ts:44)
- [mro.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline-phases/mro.ts:25)
- [run-analyze.ts](/F:/AVmatrix-main/avmatrix/src/core/run-analyze.ts:217)

## Scope

Plan này xử lý:

- analyze / re-analyze orchestration trong `avmatrix/`
- baseline snapshot persistence trong `.avmatrix/`
- Git delta classification cho web `Re-analyze`
- API / web runtime states tối thiểu để user thấy rõ `No-op`, delta, hay fallback

Plan này không xử lý:

- đổi call-resolution semantics
- phát minh graph architecture mới
- phát minh ownership engine / closure engine mới
- phát minh cách vẽ link graph mới hoặc relationship linker mới
- đổi nghĩa của `parse`, `crossFile`, `mro`, `communities`, hay `processes`
- tối ưu V1 theo hướng cực đoan nếu làm rủi ro correctness tăng
- rebuild toàn bộ LadybugDB hoặc rewrite full snapshot trong delta fast path

## Non-Negotiable Invariants

1. Delta path phải reuse phase code và phase order hiện tại.
2. `parse` vẫn là một orchestration boundary duy nhất.
3. `crossFile` vẫn là cross-file propagation mechanism; không thay bằng cơ chế tự nghĩ ra.
4. Relationship semantics chỉ thuộc các processor hiện có (`processImports*`, `processHeritage*`, `processCalls*`, `routesPhase`, `crossFile`, `mro`, `communities`, `processes`). Delta path không được tự tạo linker / resolver / graph-link builder mới.
5. Delta fast path chỉ được parse / resolve `changedFiles` và `affectedFiles`; không được quét rồi xử lý như full analyze trá hình.
6. Delta fast path không được gọi `finalizeAnalyzedGraph()` hoặc helper nào rebuild toàn LadybugDB / full snapshot. Full finalize chỉ thuộc `analyze` hoặc `re-analyze` fallback-full.
7. Delta persistence phải là delete/insert theo `filePath` hoặc graph section tương đương trong LadybugDB; không được tạo staged LadybugDB mới rồi swap toàn bộ.
8. Snapshot baseline phải là per-file shard có thể update theo delta; không được rewrite một `graph-snapshot.json` chứa toàn graph trong fast path.
9. `mro` không được silently stale. Nếu affected-scope update không chứng minh được correctness cho MRO inputs thì fast path phải fallback full hoặc mark derived state stale theo contract rõ.
10. Nếu chưa chứng minh được local scope là đúng, V1 được phép widen work scope bảo thủ, nhưng widen tới repo-wide đồng nghĩa fallback/full path, không còn là delta fast path.
11. Persistent baseline replacement phải atomic theo góc nhìn của user.
12. `re-analyze` phải tách thành orchestration path riêng; không được nhồi thành một chuỗi `if/else` lớn bên trong full analyze path.
13. Việc tách path không được kéo theo một ingestion engine thứ hai; phase semantics, graph primitives, và persistence helpers vẫn phải được chia sẻ.
14. Route, middleware, framework route registry, template route consumers, hoặc global configuration changes phải fallback full nếu không có local invalidation model đã test.

## Product Decisions To Lock Before Coding

### 1. `Re-analyze` phải có 3 kết quả rõ ràng

V1 phải chọn giữa:

- `No-op`
- `Delta analyze`
- `Fallback to full analyze`

Rule:

- không có indexed baseline => `Fallback to full analyze`
- không lấy được Git diff usable => `Fallback to full analyze`
- có Git diff nhưng tracked delta set rỗng trong supported scope => `No-op`
- có baseline usable + Git delta supported + scope đủ chắc + persistence có thể update incrementally => `Delta analyze`
- nếu delta parse được nhưng persistence bắt buộc rebuild toàn LadybugDB / full snapshot => `Fallback to full analyze`, không được report là delta fast path

### 2. Supported Git delta scope của V1

Delta entry chỉ bắt đầu từ:

- tracked `modified`
- tracked `deleted`
- tracked `renamed/moved`
- `staged-added`

V1 không coi các thứ này là delta input:

- untracked files chưa `git add`
- ignored files
- runtime artifacts ngoài Git scope

Target behavior so với hiện tại:

- untracked-only dirty state không còn được tự động ép thành full rebuild
- nếu repo chỉ có untracked relevant files thì web UI phải báo là chưa có tracked changes để `Re-analyze`

### 3. `re-analyze` phải là path riêng, không phải một mode cắm thêm vào full analyze

Lý do:

- lifecycle của `analyze` và `re-analyze` khác nhau
- `analyze` là build graph mới từ đầu rồi persist
- `re-analyze` phải đọc baseline, phân loại Git delta, widen affected scope, merge, verify, rồi mới swap

V1 phải đi theo hướng:

- có một orchestration entry riêng cho `re-analyze`
- orchestration đó chịu trách nhiệm mode selection, baseline loading, delta merge, verify, rollback, và persistence swap
- orchestration đó không copy phase logic hiện có sang một engine mới

Không chọn hướng:

- tiếp tục bơm thêm branch vào `runFullAnalysis()` cho tới khi nó vừa làm full analyze vừa làm delta orchestration
- clone một ingestion pipeline thứ hai chỉ để chạy `re-analyze`

Stance của plan này là:

- tách ở tầng flow
- chia sẻ ở tầng semantics và primitives

### 4. Baseline không thể chỉ là `meta.json`, và không thể là một full JSON phải rewrite mỗi lần

Để delta path là thật, V1 phải có baseline artifacts rõ ràng trong `.avmatrix/` có thể load phần cần thiết thành mutable `KnowledgeGraph` hoặc graph section.

Baseline contract:

- `meta.json` chỉ là metadata: commit, thời gian, stats
- LadybugDB vẫn là query store
- snapshot artifact là baseline cho các lần delta sau, nhưng fast path phải update theo per-file shard, không rewrite toàn graph
- sau `analyze` thành công có thể ghi full baseline mới
- sau `re-analyze` fast path thành công chỉ ghi affected snapshot shards cần thiết
- sau `re-analyze` fallback-full thành công được refresh full baseline

V1 chọn **per-file shard**, không chọn patch log.

Lý do:

- shard theo file map thẳng với Git delta và delete/insert theo `filePath`
- dễ test rollback vì mỗi affected file có staged shard riêng
- không cần replay / compact patch log trong V1

Snapshot format bắt buộc phải:

- round-trip được graph section cần update về `KnowledgeGraph` hoặc equivalent in-memory merge model
- có version và manifest
- update theo temp path + atomic rename ở shard level
- không rewrite toàn bộ graph trong delta fast path

Per-file shard contract:

- manifest lưu snapshot version, repoPath, lastCommit, trackedDeltaSignature, shard schema version, derived state versions, và map `filePath -> shardPath/hash`
- mỗi source-file shard lưu nodes thuộc file đó, relationships đã được emit bởi processors hiện có, parse metadata cần để tái tạo input cho processors hiện có, và content hash
- delete file phải xóa shard và DB graph section owned by old file
- rename/move phải xóa old shard, ghi new shard, và update manifest atomically
- full analyze có thể rebuild toàn bộ shard set và compact manifest

Đây là phần bổ sung về persistence, không phải một graph architecture mới.

### 4.1. Baseline parse metadata là bắt buộc cho delta correctness

Delta fast path không đủ nếu chỉ lưu graph nodes/relationships. Existing `crossFile` phụ thuộc vào parse output như `resolutionContext`, import maps, named import maps, exported type maps, binding accumulator output, route/fetch/tool/ORM extraction outputs.

Metadata shards không được trở thành một linker mới. Chúng chỉ được dùng để rehydrate / seed input cho phase code hiện có. Nếu cần tạo `IMPORTS`, `CALLS`, `EXTENDS`, `IMPLEMENTS`, `FETCHES_FROM`, hoặc route/tool links, code phải đi qua processor hiện tại tạo loại relationship đó.

V1 phải persist per-file metadata tối thiểu:

- imports / named imports / module dependencies
- exported type map entries
- declarations needed by cross-file call resolution
- inheritance / heritage inputs relevant to MRO
- extracted routes / decorator routes / fetch calls / tool defs / ORM queries
- file content hash used to detect stale shard reuse

Nếu metadata cần cho một phase không có hoặc không đủ để build local `phaseWorkSet`, fast path phải fallback-full.

Forbidden:

- tự dựng `CALLS` / `IMPORTS` / heritage edges từ metadata bằng code mới
- copy một phần logic resolve ra ngoài `parse` / `crossFile`
- suy luận relationship ownership để thay thế processor hiện tại

### 5. Phải tách `changedFiles` khỏi `affectedFiles`

Plan này không được tiếp tục nói như thể "Git changed files" và "tất cả file có thể stale graph" là cùng một tập.

Định nghĩa:

- `changedFiles`: tập file trực tiếp lấy từ Git diff
- `affectedFiles`: `changedFiles` cộng với các file có graph có thể stale vì imports, exports, inheritance, route registry, middleware, hoặc cross-file binding propagation
- `phaseWorkSet`: tập file / phạm vi thật mà từng phase rerun; có thể rộng hơn `affectedFiles` nếu semantics hiện tại của phase cần repo-wide context

Rule:

- `changedFiles` là entry point
- `affectedFiles` là minimum correctness set
- từng phase được phép widen tới repo-wide nếu local scope chưa được chứng minh là đúng
- nếu `phaseWorkSet` widen tới repo-wide thì execution mode phải chuyển sang fallback-full hoặc explicitly report non-fast-path; không được gọi đó là delta fast path

### 6. Rollback phải giữ nguyên baseline tốt gần nhất

`Fallback` trong V1 không được có nghĩa là:

- mutate live baseline trước
- fail giữa chừng
- rồi cố vá lại sau

Required behavior:

- current baseline snapshot và current LadybugDB phải giữ nguyên cho tới khi delta result được verify
- delta persistence phải ghi staged shard / patch và staged DB operations trước khi publish
- replacement cuối cùng phải dùng transaction hoặc rename semantics ở đúng granularity delta
- `meta.json` chỉ được update sau khi LadybugDB delta và snapshot delta đã durable

Nếu delta build fail trước swap:

- baseline cũ vẫn là active baseline

Nếu delta build xong nhưng swap fail:

- baseline cũ vẫn phải là active baseline
- lỗi phải được surface rõ

### 7. LadybugDB delta writer phải có ownership rules rõ ràng

`delete/insert theo filePath` không được hiểu là chỉ xóa node `File`.

V1 phải định nghĩa ownership:

- file-owned nodes: `File` node và các symbol / route-independent extracted nodes có `properties.filePath === filePath`
- file-owned relationships: relationship có source hoặc target là file-owned node của affected file
- cross-file relationships: relationship nối affected file-owned node với node ngoài affected set; delta writer được xóa stale rows nếu source hoặc target thuộc affected file-owned set, nhưng relationship mới chỉ được insert nếu đã được emit bởi processor hiện có
- route / middleware / template / framework-global relationships: nếu thay đổi có thể ảnh hưởng mà chưa có rule local đã test thì fallback-full
- derived relationships như `METHOD_OVERRIDES`, `METHOD_IMPLEMENTS`, `MEMBER_OF`, `STEP_IN_PROCESS`, `ENTRY_POINT_OF`: không update như fresh trong fast path trừ khi có derived-state policy rõ

Delta writer contract:

- chạy trong một DB transaction hoặc staged operation có rollback rõ
- xóa stale file-owned nodes / relationships trước, insert rebuilt sections sau
- không tự quyết định target/source của relationship mới
- chỉ persist relationship objects có sẵn trong delta `KnowledgeGraph` output từ phase code hiện có
- update FTS entries cho nodes bị xóa / insert
- xóa embeddings cho removed or content-changed nodes; preserve embeddings cho node id + content hash không đổi
- không gọi `loadGraphToLbug()` toàn graph
- không tạo staged LadybugDB mới rồi swap toàn bộ

### 8. `communities` và `processes` không được làm delta fast path thành full analyze trá hình

`communities` và `processes` hiện tại là whole-graph derived phases.

V1 chốt policy mới:

- delta fast path không rerun `communities` và `processes` whole graph
- nếu user / API yêu cầu derived graph coherence tuyệt đối ngay lập tức thì route sang `Fallback to full analyze`
- nếu chạy fast path, derived `MRO` / `Community` / `Process` state phải được mark stale / partial / deferred bằng metadata rõ ràng để UI và MCP không hiểu nhầm là freshly coherent
- nếu chưa có stale/deferred contract cho derived phases thì V1 phải fallback full cho changes có thể ảnh hưởng tới communities/processes, thay vì silently giữ stale output

V1 không chọn policy "biết là stale nhưng vẫn giả vờ fresh" cho persistent state.

Derived state metadata:

- `derivedState.mro`: `fresh | stale | deferred`
- `derivedState.communities`: `fresh | stale | deferred`
- `derivedState.processes`: `fresh | stale | deferred`
- `derivedState.reason`: stable string for API/UI/MCP
- `derivedState.changedFiles`: count or list bounded by response-size policy

### 9. FTS và embeddings phải có delta policy

Fast path không được rebuild full FTS hoặc rerun embeddings toàn graph.

Policy:

- FTS: delete index rows for removed stale nodes and insert/update rows for rebuilt nodes in the same delta persistence flow
- embeddings: delete embeddings for removed nodes; for rebuilt nodes, preserve only when node id and content hash are unchanged; otherwise mark embedding missing/stale for later targeted generation
- if existing embedding pipeline cannot target affected nodes safely, fast path must skip embedding generation and surface `embeddings: stale-partial`, not run full embedding regeneration

### 10. Web UI phải phản ánh đúng path được chọn

Web runtime nên giữ graph cũ trên màn hình tới khi job hoàn tất rồi mới swap.

Các trạng thái user cần thấy:

- `No-op`
- `Delta analyze`
- `Fallback to full analyze`

Status message phải nói rõ path nào được chọn, để user không phải đoán.

API contract tối thiểu:

- request: `{ path, reanalyze?: boolean, force?: boolean, freshDerived?: boolean, embeddings?: boolean }`
- `reanalyze: true` chọn re-analyze orchestration; không dùng `force: true` để giả lập re-analyze
- `freshDerived: true` yêu cầu derived graph fresh, nên fallback-full nếu fast path không recompute derived phases
- response / job status: `executionMode`, `selectionReason`, `derivedState`, `alreadyUpToDate`

## Phase Constraints In The Current Architecture

### `scan` / `structure`

`structure` hiện build File/Folder graph từ repo-wide `allPaths`, nên V1 không được giả vờ đây là phase purely local.

Delta fast path được phép đọc lightweight path metadata để biết file nào còn tồn tại, nhưng không được rebuild toàn bộ structure graph rồi persist như full analyze. Nếu structural invalidation cần repo-wide graph rewrite thì phải fallback-full.

Nguồn:

- [structure.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline-phases/structure.ts:38)

### `parse`

`parse` hiện encapsulate:

- symbol parsing
- import resolution
- heritage resolution
- call resolution
- route extraction
- tool extraction
- fetch extraction
- ORM extraction
- `resolutionContext`
- `bindingAccumulator`
- `exportedTypeMap`

Nguồn:

- [parse.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline-phases/parse.ts:68)

Không plan nào ở đây được phép tách các phần này thành top-level architecture mới.

### `routes`

`routes` đọc `allPaths` và các extracted route / fetch outputs, nên invalidation route-related có thể fan out vượt ra ngoài changed file trực tiếp.

V1 fast path không cố incremental-route update nếu route registry, middleware, filesystem route, template consumer, hoặc framework config thay đổi. Những case đó phải fallback-full.

Nguồn:

- [routes.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline-phases/routes.ts:57)

### `crossFile`

`crossFile` hiện đã là cross-file propagation thật theo topological import order.

Nguồn:

- [cross-file.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline-phases/cross-file.ts:52)

V1 phải reuse logic này, không thay nó bằng closure engine khác.

### `mro`

`mro` hiện tạo derived edges từ graph sau `crossFile`.

Nguồn:

- [mro.ts](/F:/AVmatrix-main/avmatrix/src/core/ingestion/pipeline-phases/mro.ts:42)

V1 không được silently giữ stale MRO edges. Nếu affected-scope MRO update chưa được chứng minh đúng thì phải fallback-full hoặc mark derived state stale theo contract rõ.

## V1 Delta Execution Model

1. đọc current baseline metadata và snapshot artifact từ `.avmatrix/`
2. đọc Git diff và phân loại direct `changedFiles`
3. chọn `No-op`, `Delta analyze`, hoặc `Fallback to full analyze`
4. nếu vào delta path thì derive `affectedFiles` theo semantics hiện tại của codebase
5. nếu route / middleware / framework registry / global config / unsupported scope xuất hiện thì `Fallback to full analyze`
6. load baseline file shards và parse metadata shards cần thiết, không load hoặc rewrite full snapshot nếu không cần
7. remove stale file-owned graph sections cho `changedFiles` / `affectedFiles`
8. rerun phase logic hiện tại chỉ trên `phaseWorkSet`; nếu phase cần repo-wide work thì chuyển sang fallback-full
9. update LadybugDB bằng delta transaction: delete stale nodes/relationships theo `filePath` rồi insert rebuilt sections
10. update snapshot bằng per-file shard + manifest update
11. không gọi `finalizeAnalyzedGraph()` trong delta fast path
12. mark / defer derived `mro`, `communities`, và `processes`, hoặc fallback-full nếu cần fresh derived graph
13. chỉ update `meta.json` và completion state sau khi LadybugDB delta + snapshot delta đã durable

Trong toàn bộ execution model này, delta orchestration chỉ điều phối workset và persistence. Nó không được tự tạo relationship semantics. Mọi relationship mới phải đến từ phase / processor hiện tại.

## Candidate Modules To Change

Khả năng cao implementation sẽ đụng các vùng này:

- `avmatrix/src/server/api.ts`
- `avmatrix/src/storage/git.ts`
- `avmatrix/src/core/run-analyze.ts`
- module orchestration mới, ví dụ `avmatrix/src/core/run-delta-reanalyze.ts`
- `avmatrix/src/core/graph/*`
- `avmatrix/src/core/ingestion/pipeline.ts`
- `avmatrix/src/core/ingestion/pipeline-phases/*`
- `avmatrix/src/core/lbug/lbug-adapter.ts`
- module LadybugDB delta persistence mới, ví dụ `avmatrix/src/core/lbug/lbug-delta-writer.ts`
- module FTS / embedding delta helpers nếu existing APIs chưa support targeted update
- module snapshot persistence mới trong `avmatrix/src/core/` hoặc `avmatrix/src/storage/`
- `avmatrix-web/src/components/Header.tsx` nếu cần update trạng thái hiển thị

## Implementation Workstreams

### A. Git delta classification

- thay gate dirty / not-dirty hiện tại bằng classified Git delta result
- tách `supported tracked changes` khỏi `untracked-only dirty state`
- trả explicit analyze mode selection lên job layer
- web `Re-analyze` phải gửi `reanalyze: true`, không gửi `force: true` như hiện tại

### B. Execution-path separation

- giữ `runFullAnalysis()` tập trung vào full analyze lifecycle
- tạo orchestration riêng cho `re-analyze`, ví dụ `runDeltaReanalyze()`
- extract shared helpers khi cần thay vì copy logic hoặc nhét branch vào một function quá tải trách nhiệm
- chốt ranh giới rõ giữa:
  - full analyze orchestration
  - delta re-analyze orchestration
  - shared ingestion / persistence primitives

### C. Snapshot persistence và baseline loading

- thay full `graph-snapshot.json` bằng per-file shard baseline
- version hóa snapshot format
- dùng temp write + atomic rename ở shard / patch granularity
- đảm bảo `analyze` full có thể compact / refresh baseline
- đảm bảo `re-analyze` fast path chỉ update affected snapshot shards / patches
- persist parse metadata shards cùng graph shards

### D. Affected-scope computation

- bắt đầu từ `changedFiles`
- widen thành `affectedFiles`
- chốt widening rules bảo thủ cho routes, cross-file binding, và MRO invalidation
- không cho delta execution quá hẹp nếu semantics hiện tại chưa được model đúng

### E. Delta graph merge

- load current baseline sections cần thiết vào `KnowledgeGraph`
- remove stale file-owned graph sections
- rerun current phases trên work sets tương ứng
- giữ merge logic bám sát mutation APIs hiện có của `KnowledgeGraph`
- không gọi shared full-finalize persistence từ delta path
- persist bằng delta DB writer: delete/insert theo `filePath`, không rebuild DB
- ownership rules phải cover file-owned nodes, file-owned relationships, cross-file relationships, and derived relationships
- không thêm manual `addRelationship()` trong delta orchestration / DB writer để thay thế logic của processors
- nếu existing processor không thể chạy với local workset mà vẫn giữ đúng link semantics thì fallback-full

### F. Derived phases

- không rerun `mro`, `communities`, và `processes` whole graph trong delta fast path
- mark stale / partial / deferred derived state trong metadata nếu fast path giữ graph cũ
- fallback full khi caller yêu cầu fresh derived graph hoặc khi stale marking chưa đủ an toàn
- không surface stale derived state như fresh result

### G. FTS / embeddings delta policy

- update FTS only for deleted / inserted / changed nodes
- delete embeddings for removed or content-changed nodes
- preserve embeddings when node id and content hash are unchanged
- nếu targeted embedding update chưa an toàn thì mark embeddings stale-partial, không rerun full embeddings trong fast path

### H. API / UI contract

- expose mode selection: `No-op`, `Delta analyze`, `Fallback to full analyze`
- giữ previous graph visible cho tới khi complete
- hiển thị rõ untracked-only state và fallback reason
- expose `derivedState`, `executionMode`, `selectionReason`, và `alreadyUpToDate`

## Validation Expectations

Implementation chưa được coi là xong nếu chưa cover tối thiểu:

- không có baseline snapshot => `Fallback to full analyze`
- có Git diff usable nhưng tracked delta set rỗng => `No-op`
- modified file remove + rebuild đúng file-owned graph
- deleted file remove stale graph đúng
- staged-added file insert graph đúng
- renamed / moved file remove old-path graph và insert new-path graph đúng
- affected dependents được widen đúng khi direct changed files là chưa đủ
- untracked-only dirty state không trigger full rebuild
- `analyze` thành công thì ghi snapshot mới
- `re-analyze` fast path thành công thì chỉ ghi affected snapshot shard, không rewrite full snapshot
- `re-analyze` fast path không gọi `finalizeAnalyzedGraph()`
- `re-analyze` fast path không tạo staged LadybugDB mới và không gọi `loadGraphToLbug()` toàn graph
- LadybugDB delta writer delete/insert đúng nodes và relationships theo `filePath`
- LadybugDB delta writer xóa stale cross-file relationships có source hoặc target thuộc affected file-owned nodes, rồi chỉ insert relationship mới đã được processors emit
- relationship mới trong delta path phải được tạo bởi processors hiện có; test không cho delta orchestration / DB writer tự gọi `addRelationship()` để vẽ link mới
- trên fixture nhỏ, delta output cho affected graph section phải match full analyze output về relationship type/id/source/target/confidence/reason cho cùng file set
- FTS chỉ update entries cho deleted / inserted / changed nodes
- embeddings không chạy full regeneration trong fast path
- embeddings bị xóa hoặc mark stale cho content-changed nodes
- per-file graph shard và parse metadata shard được ghi atomically cho affected files
- manifest update không rewrite full graph snapshot
- delta fail trước swap thì baseline cũ còn nguyên
- delta swap fail thì baseline cũ còn nguyên
- `mro`, `communities`, và `processes` không bị report là fresh nếu fast path không recompute chúng
- caller yêu cầu fresh derived graph thì route sang fallback-full
- graph sections không liên quan được preserve
- route / middleware / framework route registry / template route consumer changes fallback full
- web `Re-analyze` request dùng `reanalyze: true`, không dùng `force: true`
- job status trả `executionMode`, `selectionReason`, `derivedState`, `alreadyUpToDate`
- regression test mock đảm bảo delta fast path không gọi `runPipelineFromRepo`, `finalizeAnalyzedGraph`, `loadGraphToLbug`, hoặc full snapshot writer

Validation commands cho implementation:

- `cd avmatrix && npm test`
- `cd avmatrix && npx tsc --noEmit`
- `cd avmatrix-web && npm test` khi có đổi UI / runtime contract
- `cd avmatrix-web && npx tsc -b --noEmit` khi có đổi UI / runtime contract

## Non-Goals For V1

- minimal recomputation hoàn hảo cho mọi phase
- perfect incremental `communities` / `processes`
- background worker scheduling cho deferred derived phases
- support untracked source files trước `git add`
- đổi graph schema hoặc query model

## Next Step

Bước tiếp theo không phải là bàn thêm một kiến trúc mới.

Bước tiếp theo là sửa implementation theo fast-path contract này, quanh các ranh giới sau:

- delta path nào được phép parse local và persist local
- phase nào nếu cần repo-wide thì phải fallback-full
- LadybugDB delta writer delete/insert theo `filePath` hoạt động ra sao
- snapshot shard manifest format ra sao
- parse metadata shard format ra sao
- FTS / embeddings delta update hoạt động ra sao
- derived state stale/deferred metadata contract ra sao
