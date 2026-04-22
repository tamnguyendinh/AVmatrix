# AVmatrix MCP Startup Optimization

Last updated: 2026-04-22
Status: proposed

## Purpose

Plan nay chot cach toi uu cold start cua `avmatrix mcp` khi Codex spawn MCP server qua `stdio`.

Muc tieu cua plan:

- loai bo startup timeout khi Codex cho MCP handshake
- giam startup critical path truoc `server.connect()`
- giu nguyen tool surface, protocol, va hanh vi multi-repo
- khong can Docker, khong can auto-start service, khong doi transport trong pham vi plan nay

## Problem Statement

Loi van hanh hien tai:

- Codex bao `MCP startup incomplete (failed: avmatrix)` khi `avmatrix` khong san sang trong thoi gian cho startup

Nguyen nhan ky thuat hien tai trong code path:

1. `avmatrix/src/cli/index.ts` lazy-load `mcpCommand`
2. `avmatrix/src/cli/mcp.ts` tao `LocalBackend`
3. `mcpCommand()` `await backend.init()`
4. `mcpCommand()` `await backend.listRepos()`
5. Sau do moi goi `startMCPServer(backend)`
6. Trong `avmatrix/src/mcp/server.ts`, MCP chi san sang sau `server.connect(transport)`

He qua:

- handshake MCP bi dat sau phan bootstrap backend
- startup bi chan boi viec doc registry va warm context truoc khi client nhan duoc initialize/list_tools/list_resources

Ngoai ra con co duplicate work:

- `LocalBackend.init()` goi `refreshRepos()`
- `LocalBackend.listRepos()` cung goi `refreshRepos()`

Nghia la startup path hien tai vua muon, vua lap cong viec.

## Scope

Plan nay chi xu ly:

- startup path cua `avmatrix mcp` qua `stdio`
- cac thay doi trong `src/cli/mcp.ts`, `src/mcp/server.ts`, `src/mcp/local/local-backend.ts`, va cac helper lien quan
- logging/timing can thiet de do startup latency

Plan nay khong xu ly:

- Docker runtime
- auto-start `avmatrix serve`
- chuyen Codex sang dung `http://127.0.0.1:4747/api/mcp`
- toi uu query/impact/search sau khi MCP da startup xong

## Hard Goals

1. Startup critical path truoc `server.connect()` khong duoc chua:
- repo registry refresh
- repo listing de in banner
- LadybugDB init
- bat ky cong viec nao khong can cho MCP handshake

2. `list_tools` va `list_resources` phai phan hoi duoc ngay sau khi MCP connect.

3. `list_repos`, `resolveRepo`, va tool calls van phai dung trong ca 3 tinh huong:
- khong co repo nao
- co 1 repo
- co nhieu repo

4. Khong duoc ghi log vao stdout theo cach lam hong JSON-RPC stream.

5. Khong duoc danh doi startup nhanh bang cach lam sai ket qua tool.

## Non-Goals

- Khong dat muc tieu "startup = 0ms"
- Khong refactor toan bo runtime neu quick win da giai quyet duoc timeout
- Khong tach `serve` va `mcp` thanh hai kien truc khac nhau trong plan nay

## Guiding Principles

1. Handshake first, work later.
- Viec quan trong nhat cua startup path la de MCP client ket noi duoc som.

2. Static metadata must be cheap.
- `list_tools`, `list_resources`, va prompt metadata khong nen doi backend warm xong.

3. Backend should be lazy for repo-dependent work.
- Moi viec can registry, repo path, hoac DB thi chi lam khi thuc su can.

4. One refresh source of truth.
- Khong duoc de startup path refresh registry nhieu lan ma khong co gia tri tang them.

5. Measure on stderr only.
- Startup timing va diagnostic logs phai di stderr de khong pha protocol.

## Current Bottlenecks To Address

### A. Handshake bi dat qua muon

`mcpCommand()` hien dang warm backend truoc khi mo transport.

Day la bottleneck lon nhat vi no lam Codex khong nhan duoc MCP initialize trong som.

### B. Duplicate registry refresh

`init()` va `listRepos()` deu refresh registry.

Ngay ca khi refresh nhanh, phan lap nay van la startup tax vo ich.

### C. Import chain co the van nang

Ngay ca sau khi sua A va B, startup co the van chiu chi phi top-level import cua:

- `LocalBackend`
- search/group/storage modules
- cac module co side effects nhe nhung tich luy thanh chi phi boot

Phan nay chi duoc xu ly neu sau Phase 1-3 startup van chua dat muc tieu.

## Success Metrics

Chi so can theo doi:

- thoi gian tu process start den `server.connect()` xong
- thoi gian tu process start den MCP client initialize thanh cong
- thoi gian request dau tien cua `list_repos`
- ti le Codex startup timeout trong qua trinh dung that

Muc tieu v1:

- startup handshake khong con phu thuoc vao repo refresh
- startup binh thuong tren may dev Windows phai nhanh hon ro ret so voi hien tai
- timeout 30s khong con la duong loi chinh trong dieu kien thong thuong

Muc tieu tham vong:

- connect path o muc vai giay tro xuong, thay vi co the cham nguong timeout

## Proposed Implementation Plan

## Phase 1. Connect MCP first

Day la thay doi co ROI cao nhat va phai lam truoc.

### Changes

1. Sua `avmatrix/src/cli/mcp.ts`
- bo `await backend.init()` khoi startup critical path
- bo `await backend.listRepos()` khoi startup critical path
- tao `LocalBackend` xong thi goi `startMCPServer(backend)` ngay

2. Neu can warm backend, chi duoc warm theo kieu non-blocking
- sau khi `server.connect()` thanh cong, moi `void backend.init().catch(...)`
- warm-up nay la optional optimization, khong duoc tro thanh gate cua MCP connect

3. Di chuyen startup banner/warning ra khoi pre-connect path
- hien tai `mcp.ts` in thong tin repo truoc khi start server
- sau refactor, banner neu con can thi phai in sau connect hoac in khi warm-up xong

### Expected outcome

- Codex thay MCP server san sang som hon
- initialize/list_tools/list_resources khong con bi block boi registry refresh

### Risks

- Neu client goi `list_repos` ngay sau initialize, backend phai tu refresh on-demand va van tra ket qua dung
- Neu startup banner doi vi tri, logs co the khac truoc

### Mitigation

- giu `listRepos()` va `resolveRepo()` lazy va blocking tai luc can thiet
- giu toan bo logs startup o stderr

## Phase 2. Remove duplicate refresh

Sau khi connect som, can xoa phan refresh lap de first real request cung gon.

### Changes

1. Chot mot nguon refresh duy nhat cho startup-era repo discovery
- `init()` chi la warm helper, khong phai bat buoc
- `listRepos()` va `resolveRepo()` moi la entry points can refresh khi can

2. Them co che single-flight cho `refreshRepos()`
- neu nhieu request dau tien cung den, chi cho mot lan refresh chay
- cac request khac await cung mot promise thay vi mo nhieu lan quet registry

3. Loai bo viec startup path goi `listRepos()` chi de hien thi thong tin
- neu can thong tin repo count, lay no sau warm-up, khong chen vao handshake

### Expected outcome

- bo duoc refresh vo ich truoc connect
- first repo-aware request nhe hon va on dinh hon
- tranh race condition khi nhieu tool call den som

### Risks

- neu cache/refresh contract khong ro, co the sinh bug repo list cu

### Mitigation

- quy uoc ro:
  - `refreshRepos()` la noi duy nhat sua in-memory repo map
  - `listRepos()` co the refresh on-demand
  - `resolveRepo()` refresh khi cache miss

## Phase 3. Measure timing again

Chi sau khi lam xong Phase 1-2 moi do lai de tranh toi uu mu.

### Required timing points

1. process start
2. vao `mcpCommand()`
3. tao `LocalBackend`
4. bat dau `server.connect()`
5. `server.connect()` xong
6. bat dau warm-up background neu co
7. warm-up xong neu co
8. `list_repos` dau tien
9. tool call dau tien can `resolveRepo`

### Logging rules

- chi log ra stderr
- format phai grep duoc
- khong spam qua nhieu trong normal path

### Validation scenarios

1. zero indexed repos
2. one indexed repo
3. many indexed repos
4. registry co stale path
5. storage co legacy cleanup path
6. Windows dev machine co disk/AV cham

### Decision gate after measurement

- neu startup da giam ro ret va khong con timeout thuc te, dung o day
- neu startup van con dang ke, moi chuyen Phase 4

## Phase 4. Lazy import / backend provider (only if still needed)

Phase nay chi mo khi timing sau Phase 3 cho thay startup van chua dat muc tieu.

### Candidate changes

1. Tach metadata MCP khoi backend-heavy path
- tool definitions va resource templates la static
- khong nen import them nhieu module chi de tra metadata

2. Dynamic import cho cac module nang
- search
- group service
- storage/DB helpers nao khong can cho startup

3. Can nhac backend provider lazy hon
- server nhan mot provider thay vi mot backend da warm
- backend duoc khoi tao va/hoac warm o request dau tien can den repo state

4. Can nhac dong bo lai startup strategy giua `mcp` va `serve`
- tranh de hai surfaces co hai logic lazy/eager khac nhau

### Tradeoffs

- code phuc tap hon
- de tao regression ve error handling va concurrency
- can test ky hon Phase 1-2

## Validation Checklist

Sau khi implement Phase 1-2, can verify:

- Codex khong con timeout khi spawn `avmatrix`
- `list_tools` van ra dung danh sach
- `list_resources` van ra dung templates/resources
- `list_repos` van dung khi:
  - khong co repo
  - 1 repo
  - nhieu repo
- `query`, `context`, `impact`, `detect_changes`, `rename`, `cypher` van resolve repo dung
- stderr logs khong lam hong MCP protocol

Neu vao Phase 4, can verify them:

- startup nhanh hon that, khong chi doi chi phi sang first request mot cach te hon
- khong co import cycle moi
- khong co race condition trong lazy backend initialization

## Rollout Strategy

1. Land Phase 1 va Phase 2 trong cung mot rollout nho
- day la cap thay doi co lien he truc tiep
- tach le tung phase se de tao half-state kho hieu

2. Giu `startup_timeout_sec = 120` trong thoi gian bake-in
- day la safety margin van hanh
- khong xem no la fix goc

3. Thu timing va startup behavior tren may Windows dung that
- day la moi truong dang gap van de

4. Chi mo Phase 4 neu du lieu timing cho thay can thiet

## Open Questions

1. Co nen bo han `backend.init()` khoi `mcp` path hay giu lai nhu mot background warm-up helper?

2. Co nen them single-flight cho `refreshRepos()` o muc backend chung de dung duoc cho ca `serve`?

3. Co nen doi startup banner hien repo count sang mot log diagnostic sau connect, thay vi coi no la phan cua startup?

4. Sau khi `mcp` startup path on, co nen ap dung cung pattern cho `serve` de tranh drift?

## Bottom Line

Thu tu hanh dong duoc chot:

1. Connect MCP truoc
2. Bo refresh lap
3. Do timing lai
4. Neu van cham, moi tach import va lam backend lazy hon

Plan nay co y bo qua Docker va auto-start. Muc tieu la giai quyet dung nut that cua `avmatrix mcp`: startup handshake dang bi dat sau cong viec backend khong can thiet.
