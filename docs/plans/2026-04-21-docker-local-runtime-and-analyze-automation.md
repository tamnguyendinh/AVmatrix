# AVmatrix Docker Local Runtime and Analyze Automation

Last updated: 2026-04-21
Status: proposed

## Purpose

Plan này chốt kiến trúc và rollout để chạy AVmatrix như một local service trong Docker trên Windows, tự lên cùng máy, cho client nối ổn định, và có automation để refresh index bằng `analyze`.

Mục tiêu cuối cùng:

- bật máy lên là Docker Desktop chạy
- AVmatrix backend tự lên
- Codex/MCP nối được ngay
- repo local được mount sẵn vào container
- index và registry không mất sau restart
- có một runner chủ động gọi `analyze`
- runner đó không chạy loạn khi repo thay đổi liên tục

## Hard Rules

1. Local-only là yêu cầu cứng.
- Dịch vụ chỉ được publish ra loopback của host (`127.0.0.1` / `localhost`), không mở LAN mặc định.
- Không biến bài toán này thành một remote service.

2. Chỉ có một runtime chuẩn cho container mode.
- Không duy trì hai cách dùng trái ngược nhau kiểu:
  - docs nói mount repo read-only rồi vẫn auto-analyze
  - code thật lại cần write vào repo

3. Không viết thêm orchestration trùng với cái đã có trong server nếu không cần.
- Nếu `serve` đã có `/api/analyze` và `/api/mcp`, automation phải ưu tiên dùng các endpoint đó.
- Không gọi raw `avmatrix analyze` từ nhiều nơi song song khi server đã có job manager và repo lock.

4. Phải tôn trọng constraint single-writer của LadybugDB.
- Không để watcher/analyzer mới đè lên `analyze` / embedding / MCP runtime theo cách tạo lock conflict.

5. V1 phải chốt một storage model rõ ràng.
- Nếu muốn auto-analyze repo từ host thì repo mount phải hỗ trợ ghi.
- Không hứa hẹn index ngoài repo khi code hiện tại chưa support mô hình đó.

6. Naming trong plan này phải bám canonical AVmatrix spec.
- Tên canonical trong plan này là:
  - `AVmatrix`
  - `avmatrix`
  - `.avmatrix`
  - `~/.avmatrix`
  - `AVMATRIX_HOME`
  - `avmatrix://`
- Nếu current implementation vẫn còn `gitnexus` / `.gitnexus` / `GITNEXUS_HOME`, đó chỉ là trạng thái hiện tại cần migrate, không phải target architecture.

## Naming Contract

Plan này follow trực tiếp `docs/avmatrix-canonical-spec.md`.

Nguyên tắc đọc file này:

- mọi quyết định kiến trúc đích phải dùng naming mới của AVmatrix
- mọi reference `gitnexus` chỉ còn dùng để mô tả:
  - file path / code path hiện tại
  - current implementation chưa rename
  - compatibility hoặc migration work cần xử lý

## Current Repo Facts

Các facts dưới đây là điểm tựa của plan này:

1. Current implementation của `gitnexus serve` hiện chỉ cho loopback host ở CLI path.
- `gitnexus/src/cli/serve.ts`

2. Current Docker image hiện lại chạy legacy command `gitnexus serve --host 0.0.0.0`.
- `Dockerfile.cli`

3. HTTP server đã mount MCP-over-StreamableHTTP tại `/api/mcp`.
- `gitnexus/src/server/mcp-http.ts`
- `gitnexus/src/server/api.ts`

4. Current implementation của `analyze` hiện ghi index vào `<repo>/.gitnexus`, không ghi vào `GITNEXUS_HOME`.
- `gitnexus/src/storage/repo-manager.ts`
- `gitnexus/src/core/run-analyze.ts`

5. Current implementation dùng `GITNEXUS_HOME` chỉ như global home cho registry/config/runtime metadata.
- `gitnexus/src/storage/repo-manager.ts`
- `Dockerfile.cli`

6. Legacy command `gitnexus index` không chạy phân tích mới.
- Nó chỉ register một `.gitnexus` đã tồn tại vào global registry.
- `gitnexus/src/cli/index-repo.ts`

7. `/api/analyze` trong server đã có sẵn:
- local-path validation
- same-repo dedup
- shared repo lock
- worker retry
- SSE progress
- nhưng hiện vẫn single-slot toàn cục cho các analyze job khác repo
- `gitnexus/src/server/api.ts`
- `gitnexus/src/server/analyze-job.ts`

8. Web UI và backend client đều đang mặc định local loopback.
- `gitnexus-web/src/services/backend-client.ts`

## V1 Decisions

### 1. Canonical topology

V1 dùng 3 process/container logic:

1. `avmatrix-server`
- chạy canonical command `avmatrix serve`
- expose HTTP API + `/api/mcp`

2. `avmatrix-web`
- optional
- chỉ là local UI

3. `avmatrix-watch`
- sidecar automation
- không trực tiếp chạm DB
- chỉ gọi `http://avmatrix-server:4747/api/analyze`

### 2. Canonical network model

- Bên trong container: server được bind `0.0.0.0`
- Trên host Windows: chỉ publish ra `127.0.0.1`
- Port chuẩn:
  - backend `127.0.0.1:4747`
  - web `127.0.0.1:4173`

Lý do:

- container cần bind non-loopback nội bộ để host map port vào được
- nhưng local-only contract vẫn giữ ở host boundary bằng publish loopback-only

### 3. Canonical repo/storage model

V1 chốt mô hình chuẩn như sau:

- mount một workspace root từ host vào container tại `/workspace`
- mount này là `read-write`
- mỗi repo được analyze sẽ giữ index ở chính repo đó:
  - `/workspace/<repo>/.avmatrix`
- global registry/config/runtime state nằm ở volume riêng:
  - `/data/avmatrix`

Hệ quả:

- restart container không làm mất global registry/config vì đã có volume
- restart container cũng không làm mất index vì `.avmatrix` nằm trên host bind mount
- không cần thêm một index-store abstraction mới trong v1

Ghi chú migration:

- current implementation vẫn còn dùng `.gitnexus` và `GITNEXUS_HOME`
- rollout code theo rename plan phải cutover sang:
  - `.avmatrix`
  - `~/.avmatrix`
  - `AVMATRIX_HOME`

### 4. Canonical repo coverage

V1 hỗ trợ:

- một repo:
  - mount repo đó dưới `/workspace/<repo>`
- nhiều repo:
  - mount một thư mục cha chung vào `/workspace`
  - mỗi repo là một child folder

V1 không lấy read-only workspace làm mode chuẩn cho auto-analyze.

Read-only chỉ là mode phụ, query-only, khi repo đã có `.avmatrix` sẵn từ trước.

### 5. Canonical client connectivity

Codex/MCP trong container mode sẽ nối bằng HTTP MCP:

- `http://localhost:4747/api/mcp`

Đây là đường chuẩn cho local background service, thay vì để Codex spawn một process `avmatrix mcp` riêng.

Web UI tiếp tục nói chuyện với:

- `http://localhost:4747`

### 6. Canonical automation owner

V1 chốt:

- sidecar `avmatrix-watch` là tiến trình chủ động quyết định khi nào gọi `analyze`
- `avmatrix-server` là nơi thực thi `analyze`

Nói cách khác:

- watcher quyết định khi nào
- server quyết định chạy như thế nào

### 7. Canonical trigger strategy

V1 dùng polling + debounce, không dùng filesystem event watcher làm nguồn chân lý chính.

Default:

- boot sweep ngay khi stack healthy
- poll mỗi `30s`
- debounce `20s`
- nếu repo tiếp tục đổi trong lúc analyze đang chạy thì rerun đúng 1 lần sau khi job hiện tại xong

### 8. Canonical scheduling model

V1 giữ đúng contract hiện tại của server:

- chỉ một analyze active tại một thời điểm trên toàn server

Nếu nhiều repo cùng dirty:

- watcher xếp hàng tuần tự
- không cố tạo analyze concurrency mới trong v1

## Out of Scope for V1

- index tập trung ngoài repo root
- watch bằng inotify/FSEvents làm trigger chính
- expose backend ra LAN
- auth/bearer cho local MCP HTTP
- multi-analyze concurrency
- migration sang một storage namespace hoàn toàn mới

## Problems This Plan Must Fix Explicitly

### Problem 1 — Docker backend contract đang tự mâu thuẫn

- docs/image kỳ vọng `0.0.0.0`
- CLI lại reject non-loopback host

Nếu không giải quyết điểm này thì backend container không thể là local service đúng nghĩa.

### Problem 2 — Docs hiện đang drift với hành vi thật của `index`

Nếu README/docker docs còn nói mount repo read-only rồi dùng legacy command `gitnexus index /workspace/my-repo`, hoặc canonical `avmatrix index /workspace/my-repo`, như cách index repo host, người dùng sẽ hiểu sai.

`index` không tạo index mới.

### Problem 3 — Read-only mount không khớp với auto-analyze

Vì current code hiện ghi `.gitnexus` vào repo root và target architecture sẽ ghi `.avmatrix` vào repo root, repo mount `:ro` không thể là mode chuẩn khi mục tiêu là tự refresh index.

### Problem 4 — Nhiều repo dirty cùng lúc có thể đụng single-slot analyze

Server hiện single-slot cho analyze jobs khác repo.
Watcher phải serialize thay vì assume parallelism.

### Problem 5 — Watcher có thể tự trigger loop vì index namespace thay đổi sau analyze

Fingerprint logic phải bỏ qua `.avmatrix` hoàn toàn, và trong giai đoạn migration phải bỏ qua cả `.gitnexus`.

## Target Runtime Behavior

Trạng thái vận hành cuối cùng của v1:

1. Windows login
2. Docker Desktop tự chạy
3. compose stack tự lên
4. `avmatrix-server` healthy tại `http://localhost:4747`
5. `avmatrix-watch` đợi healthcheck xong rồi scan `/workspace`
6. Codex nối `http://localhost:4747/api/mcp`
7. repo local được bind sẵn vào `/workspace`
8. watcher phát hiện repo dirty theo polling rule
9. watcher gọi `/api/analyze`
10. server chạy analyze với job manager/retry/lock sẵn có
11. nếu repo đổi tiếp trong lúc analyze chạy thì watcher đánh dấu `needs_rerun`
12. sau khi job xong và debounce đủ, watcher chạy lại đúng một lần

## Implementation Phases

### Phase A — Make `avmatrix serve` Container-safe Without Breaking Local-only

#### Goal

Cho backend chạy được trong container nhưng vẫn giữ local-only contract ở host boundary.

#### Required changes

- chỉnh `gitnexus/src/cli/serve.ts`
- chỉnh help text trong `gitnexus/src/cli/index.ts`
- chỉnh hoặc bổ sung tests cho `serve` host policy
- xác định một trong hai hướng implementation:

Option A:
- cho phép `0.0.0.0` khi có explicit container-mode signal

Option B:
- tách host validation theo execution mode, ví dụ:
  - local desktop mode: loopback-only
  - container mode: allow non-loopback bind nhưng docs bắt buộc publish loopback-only ở Docker layer

#### Acceptance

- server start được trong container
- host vẫn chỉ thấy service ở `127.0.0.1`
- local desktop mode không bị mở rộng LAN ngoài ý muốn
- docs/help canonical sau phase này phải dùng `avmatrix serve`, không dùng `gitnexus serve` làm đường chính

### Phase B — Align Compose, README, and Storage Contract

#### Goal

Đưa docs và compose về đúng storage model thực tế của code, nhưng wording/canonical output phải dùng naming mới của AVmatrix.

#### Required changes

- `docker-compose.yaml`
- `.env.example`
- `README.md`
- có thể thêm `docs/local-usage.md` hoặc docs docker riêng nếu cần

#### Required decisions

1. đổi mount workspace chuẩn từ `:ro` sang `:rw` cho managed mode
2. mô tả rõ hai mode:
- managed mode:
  - workspace `rw`
  - auto-analyze được
- query-only mode:
  - workspace `ro`
  - chỉ dùng khi repo đã có `.avmatrix`

3. sửa docs bị sai về `avmatrix index`
- tài liệu phải nói `avmatrix analyze` là lệnh tạo mới index
- `avmatrix index` chỉ là register existing `.avmatrix`

#### Acceptance

- docs không còn hứa hẹn read-only auto-analyze
- docs không còn dùng `index` như thể nó là `analyze`
- persistence story của:
  - repo source
  - `.avmatrix`
  - `AVMATRIX_HOME`
  đều rõ ràng

### Phase C — Define Canonical Client Connectivity

#### Goal

Chốt cách Codex và các client local nói chuyện với background AVmatrix trong Docker.

#### Canonical path

- Codex:
  - `codex mcp add avmatrix --url http://localhost:4747/api/mcp`
- web:
  - `http://localhost:4747`

#### Required changes

- docs setup cho Codex container mode
- docs setup cho Cursor/OpenCode nếu muốn cover thêm
- có thể thêm section riêng trong README cho:
  - stdio mode
  - docker-http mode

#### Acceptance

- user không phải `docker exec` vào container chỉ để dùng MCP
- background service và client config dùng cùng một URL ổn định

### Phase D — Add `avmatrix-watch` Sidecar

#### Goal

Tạo automation runner riêng, nhưng không duplicate analyze core.

#### Shape

- service mới: `avmatrix-watch`
- chạy trong cùng compose network với `avmatrix-server`
- mount cùng `/workspace`
- không cần mount write vào `/data/avmatrix` nếu chỉ gọi HTTP API

#### Candidate placement

- script Node mới trong repo, ví dụ:
  - `deploy/docker/watch-and-analyze.mjs`
  - hoặc `gitnexus/scripts/watch-and-analyze.mjs`

#### Why sidecar instead of in-process watcher in server

- tách responsibility rõ:
  - server = runtime/API
  - watcher = scheduling/polling policy
- giảm blast radius cho `serve`
- dễ restart/disable watcher độc lập

### Phase E — Repo Discovery Contract

#### Goal

Chốt watcher biết repo nào là managed repo.

#### V1 rule

Watcher scan direct children của `/workspace`.

Một child folder được coi là managed repo nếu:

- có `.git`, hoặc
- có `.avmatrix`, hoặc
- nằm trong allowlist explicit

#### Optional config

Hỗ trợ env:

- `WATCH_ROOT=/workspace`
- `WATCH_INCLUDE=repo-a,repo-b`
- `WATCH_EXCLUDE=node_modules,tmp`

#### Acceptance

- một workspace cha có thể chứa nhiều repo
- single-repo setup chỉ là trường hợp suy biến của cùng contract

### Phase F — Trigger and Fingerprint Design

#### Goal

Biết khi nào thực sự cần analyze, mà không bị loop hay spam.

#### V1 trigger

1. Initial sweep khi stack đã healthy
2. Poll mỗi `30s`
3. Debounce `20s`

#### Fingerprint rule

Ưu tiên:

- nếu repo có `.git`:
  - dùng fingerprint git-aware
- nếu repo không có `.git`:
  - fallback sang mtime/content snapshot nhẹ

#### Git-aware fingerprint needs to reflect

- `HEAD`
- staged/unstaged changes
- untracked files

#### Mandatory exclusions

- `.avmatrix`
- `.gitnexus`
- `.git`
- thư mục cache/runtime tạm của watcher

#### Acceptance

- analyze xong không tự kích hoạt lại chỉ vì `.avmatrix` đổi
- save file liên tục không tạo nhiều analyze job chồng nhau

### Phase G — Control Logic and State Machine

#### Goal

Đảm bảo automation ổn định khi repo đổi liên tục hoặc analyze lỗi.

#### Per-repo states

- `idle`
- `dirty`
- `debouncing`
- `queued`
- `running`
- `needs_rerun`
- `backoff`

#### Required rules

1. Không chạy chồng 2 analyze.
- watcher phải serialize theo đúng single-slot contract của server

2. Debounce trước khi enqueue.
- save liên tục trong cửa sổ debounce chỉ tạo một lần enqueue

3. Nếu repo đổi khi job đang chạy:
- set `needs_rerun = true`
- sau khi job complete:
  - nếu repo vẫn dirty sau debounce thì enqueue lại đúng 1 lần

4. Nếu analyze lỗi:
- retry với exponential backoff
- mặc định `30s`, `60s`, `120s`
- sau khi quá retry budget thì giữ repo ở trạng thái dirty/error để vòng poll sau thử lại khi có fingerprint mới

5. Nếu repo biến mất:
- mark unavailable
- không spam error log

#### Acceptance

- log watcher đọc được, không ồn vô ích
- một repo dirty liên tục không khóa cứng cả stack

### Phase H — Boot and Restart Behavior on Windows

#### Goal

Chốt hành vi vận hành khi bật máy.

#### Required operational decisions

1. Docker Desktop phải bật:
- `Start Docker Desktop when you sign in`

2. Compose services dùng:
- `restart: unless-stopped`

3. Nếu muốn luôn nóng sau idle:
- tắt hoặc điều chỉnh Resource Saver của Docker Desktop

4. Watcher chỉ start active loop sau khi:
- backend healthcheck pass

#### Acceptance

- sau login Windows, user không cần start tay từng container
- nếu Docker restart thì stack tự hồi phục

## Detailed Watcher Contract

### Inputs

- `WATCH_ROOT`
- `POLL_INTERVAL_SECS`
- `DEBOUNCE_SECS`
- `INITIAL_SWEEP=true|false`
- `WATCH_INCLUDE`
- `WATCH_EXCLUDE`
- `ANALYZE_EMBEDDINGS=true|false`

### Outputs

- structured logs:
  - repo discovered
  - repo dirty
  - debounce started
  - analyze requested
  - analyze completed
  - analyze failed
  - rerun scheduled

### HTTP contract used by watcher

- `GET /api/heartbeat`
- `POST /api/analyze`
- `GET /api/analyze/:jobId`
- `GET /api/analyze/:jobId/progress`

### Analyze request path

Watcher phải luôn gọi path theo filesystem nhìn từ trong container, ví dụ:

- `/workspace/my-repo`

Không dùng Windows host path kiểu:

- `F:\repo`

## Validation Matrix for Implementation

### Backend/container

- `cd gitnexus && npx vitest run test/unit/serve-command.test.ts test/unit/analyze-api.test.ts test/unit/analyze-job.test.ts`
- `cd gitnexus && npx tsc --noEmit`

### Web/docs/client path

- kiểm tra web vẫn auto-connect được `http://localhost:4747`
- kiểm tra Codex add MCP bằng URL hoạt động

### Compose/runtime

- `docker compose config`
- `docker compose up -d`
- `docker compose ps`
- `curl http://localhost:4747/api/heartbeat`

### Watcher behavior

- save 1 file một lần -> 1 analyze
- save liên tục 10 lần trong debounce window -> vẫn 1 analyze
- sửa tiếp khi analyze đang chạy -> job rerun đúng 1 lần
- 2 repo dirty cùng lúc -> chạy tuần tự
- restart backend/container -> watcher recover

## Risks

### Risk 1 — Mở host bind sai làm lộ service ra LAN

Mitigation:

- chỉ allow non-loopback ở container bind layer
- publish host port bằng `127.0.0.1:4747:4747`
- docs nhấn mạnh local-only publish

### Risk 2 — User nghĩ read-only mount vẫn auto-analyze được

Mitigation:

- docs phải ghi rõ read-only chỉ là query-only mode
- managed mode mặc định là read-write

### Risk 3 — Watcher loop vì `.avmatrix` thay đổi

Mitigation:

- exclude `.avmatrix` khỏi fingerprint bắt buộc
- trong giai đoạn migration transition, cũng exclude `.gitnexus`

### Risk 4 — Multi-repo backlog làm analyze chậm

Mitigation:

- v1 serialize có chủ đích
- expose queue/backlog log rõ ràng
- không giả vờ concurrent khi server chưa support

### Risk 5 — Windows bind mount polling có latency

Mitigation:

- dùng polling làm chuẩn ngay từ đầu
- không phụ thuộc hoàn toàn vào filesystem events

## Completion Criteria

Chỉ được coi là xong khi:

- backend chạy ổn trong Docker như local service
- host chỉ truy cập qua loopback
- docs không còn drift với code về:
  - `serve`
  - `index` vs `analyze`
  - read-only vs read-write mount
- Codex nối được vào `/api/mcp`
- repo mount vào `/workspace` được analyze thành công
- index không mất sau container restart
- watcher sidecar tự phát hiện thay đổi và gọi analyze
- watcher có debounce, lock-awareness, rerun, retry
- Windows boot flow không cần start tay từng thành phần
- canonical naming của rollout này khớp với:
  - `AVmatrix`
  - `avmatrix`
  - `.avmatrix`
  - `AVMATRIX_HOME`

## Explicit V1 Summary

V1 không cố giải một bài toán rộng hơn code hiện tại.

V1 chỉ chốt một đường đi thực dụng và đúng với repo hôm nay:

- Docker backend chạy nền
- host publish loopback-only
- repo bind mount `rw`
- index nằm trong repo `.avmatrix`
- registry/config nằm ở `AVMATRIX_HOME` volume
- Codex nối qua `http://localhost:4747/api/mcp`
- sidecar polling gọi `/api/analyze`
- tất cả analyze chạy tuần tự, không parallel

Nếu sau v1 muốn có:

- read-only source + separate index volume
- queue đa repo với concurrency
- watcher event-driven hoàn toàn

thì đó là các phase follow-up riêng, không được lẫn vào v1.
