# Mục đích plan này

- Plan này dùng để đổi tên toàn bộ bản local hiện tại từ `GitNexus` sang `AVmatrix`.
- Mục tiêu chính không phải là rebrand cho đẹp, mà là **tách ngữ cảnh vận hành** giữa:
  - môi trường local/fork tùy biến của bạn
  - môi trường upstream / GitHub / npm mang tên `GitNexus`
- Sau khi đổi xong, agent làm việc với bạn sẽ không còn nhầm giữa:
  - CLI local bạn đang dùng
  - MCP local bạn đang cấu hình cho Codex/Claude Code
  - docs/resources/storage/config của môi trường local
  - tên upstream `GitNexus` còn tồn tại trên GitHub, npm, báo cáo cũ, hoặc tài liệu cũ

# Mục tiêu thành công

- Tên hiển thị trên web UI đổi sang `AVmatrix`.
- CLI người dùng gọi bằng `avmatrix`.
- MCP server trong Codex/Claude Code hiện là `avmatrix`, không còn `gitnexus`.
- Resource scheme và docs vận hành local đổi sang `avmatrix://...`.
- Namespace local đổi sang:
  - `.avmatrix`
  - `~/.avmatrix`
- Không làm mất bất kỳ feature nào của web UI, CLI, MCP, graph, query, impact, detect-changes, analyze, multi-repo, chat, hoặc local-only flow.
- Không được đổi hành vi sản phẩm ngoài phạm vi rename/namespace migration trừ khi cần để tương thích namespace mới.
- Có migration path rõ ràng để dữ liệu/index cũ từ `.gitnexus` và `~/.gitnexus` được chuyển sang namespace mới, nhưng runtime sau migration chỉ dùng namespace mới.

# Scope V1 được chốt

- **V1 chỉ đổi:**
  - surface user-facing
  - command alias / CLI entrypoint
  - MCP server name / setup output
  - storage namespace
  - resource scheme
  - docs / tests / setup helpers liên quan
- **V1 không đổi ngay:**
  - tên thư mục package `gitnexus/`, `gitnexus-web/`, `gitnexus-shared`
  - package npm names công bố ra ngoài
  - import paths nội bộ hàng loạt
  - full monorepo repo/folder rename
- Mục tiêu của V1 là giải quyết **nhầm lẫn ngữ cảnh vận hành local**, không phải làm full internal rename.

# Kết quả mong muốn sau cùng

- Khi làm việc local, cả bạn và agent chỉ nhìn thấy:
  - `AVmatrix`
  - `avmatrix`
  - `.avmatrix`
  - `~/.avmatrix`
  - `avmatrix://...`
- Cái tên `GitNexus` chỉ còn được giữ ở những nơi thật sự cần thiết:
  - lịch sử git
  - tên repo tạm thời nếu chưa đổi tên thư mục
  - tài liệu migration/compatibility
  - package compatibility shim nếu cần giữ một thời gian

# Nguyên tắc cứng

- Đây là **rename + namespace migration**, không phải feature redesign.
- Cấm làm mất tính năng.
- Cấm làm mất giao diện.
- Cấm đổi UX theo kiểu rút gọn cho dễ rename.
- **Docs/spec phải đổi trước code.**
- Tên, wording, command canonical, namespace canonical phải được chốt trong docs trước, rồi code mới đổi theo đúng spec đó.
- Không được rename code trước khi docs đã xác định rõ:
  - brand mới
  - command canonical
  - MCP name canonical
  - storage namespace canonical
  - resource scheme canonical
- Cấm xóa trực tiếp surface cũ trước khi surface mới đạt parity.
- Với mọi surface lớn hoặc có nhiều trạng thái:
  - tạo file mới song song nếu cần
  - đạt parity giao diện + tính năng
  - có behavioral tests
  - chỉ sau đó mới thay thế file cũ
- Với các rename có thể gây đứt dữ liệu local:
  - phải có migration path rõ ràng
  - không giữ fallback runtime lâu dài cho storage/env
  - sau migration, namespace cũ không còn là source of truth
- Với Codex/Claude Code:
  - không để lẫn lộn MCP name `gitnexus` và `avmatrix`
  - không để docs/setup hiển thị đồng thời cả hai tên nếu không có chú thích migration rõ ràng

# Phạm vi rename

## 1. User-facing brand

- Tên trên web UI
- Tên trên onboarding/help
- Tên trong CLI banner/help
- Tên trong log/error messages
- Tên MCP server trong config/setup/docs
- Tên resource scheme

## 2. Local namespace / storage

- `.gitnexus` -> `.avmatrix`
- `~/.gitnexus` -> `~/.avmatrix`
- `gitnexus://` -> `avmatrix://`
- `gitnexus mcp` -> `avmatrix mcp`
- `gitnexus analyze` -> `avmatrix analyze`
- `GITNEXUS_HOME` -> `AVMATRIX_HOME` bằng migration/cutover sạch, không giữ fallback runtime

## 3. Repo/package/codebase internals

- package names
- bin names
- exported constants/types có brand name
- docs/examples/tests/fixtures
- runtime config and registry loaders
- MCP setup templates

## 4. Những thứ có thể tạm giữ trong giai đoạn chuyển tiếp

- tên thư mục repo trên disk
- GitHub remote upstream
- import paths nội bộ nếu đổi ngay gây rủi ro quá lớn
- package names công bố ra ngoài nếu chưa chốt publish strategy

# Quyết định kiến trúc nên chốt trước

## Brand mới

- Brand local mới là: `AVmatrix`
- CLI command mục tiêu: `avmatrix`
- MCP server name mục tiêu: `avmatrix`
- Resource scheme mục tiêu: `avmatrix://`
- Repo-local hidden folder mục tiêu: `.avmatrix`
- User-global config dir mục tiêu: `~/.avmatrix`

## Migration philosophy

- V1 của migration nên là:
  - **AVmatrix-first**
  - **không đọc fallback namespace cũ trong runtime sau khi migration đã chạy**
- Nghĩa là:
  - runtime primary chỉ dùng `.avmatrix` / `~/.avmatrix`
  - dữ liệu cũ từ `.gitnexus` / `~/.gitnexus` chỉ được dùng làm **migration source**
  - migration diễn ra một lần, có thể theo command riêng hoặc explicit upgrade step
- Sau migration:
  - source of truth duy nhất là `.avmatrix` / `~/.avmatrix`
  - `GITNEXUS_HOME` không còn là env runtime chính

## Alias philosophy

- `avmatrix` là **canonical command mới** cho local usage.
- `gitnexus` được giữ làm **compatibility alias** trong giai đoạn đầu.
- Docs/setup/onboarding mới chỉ hướng dẫn `avmatrix`, không hướng dẫn `gitnexus` như đường chính.
- Resource parsing nên chấp nhận cả:
  - `gitnexus://...`
  - `avmatrix://...`
- Nhưng generator/output mới chỉ sinh:
  - `avmatrix://...`

## Rollout philosophy

- Ưu tiên rename theo thứ tự:
  1. docs/spec canonical
  2. user-facing brand
  3. command/MCP/config namespace
  4. storage namespace
  5. docs/tests/setup cleanup
  6. package/import internals nếu thật sự cần
- Không đổi package/import quá sớm nếu chỉ rename surface là đủ để giải quyết nhầm lẫn context.

## Install story phải chốt ngay

- V1 không yêu cầu đổi package name để có command mới.
- Cách triển khai nên là:
  - package hiện tại vẫn có thể giữ tên `gitnexus`
  - `bin` expose đồng thời:
    - `gitnexus`
    - `avmatrix`
- Sau `npm link` hoặc local install, cả hai command phải cùng hoạt động.
- Docs local mới chỉ hướng dẫn:
  - `avmatrix`
- Nhưng alias `gitnexus` vẫn phải chạy được để không làm gãy config/script cũ.

# Rủi ro chính cần kiểm soát

## 1. Đứt command đang dùng

- Nếu đổi thẳng từ `gitnexus` sang `avmatrix` mà không có shim:
  - Codex/Claude config cũ sẽ gãy
  - docs cũ sẽ sai
  - script local cũ sẽ fail

## 2. Mất index/config local

- Nếu đổi `.gitnexus` sang `.avmatrix` mà không có migration path rõ ràng:
  - repo đã index sẽ biến mất khỏi UI
  - registry sẽ rỗng
  - MCP sẽ không thấy repo cũ

## 3. Lẫn brand mới/cũ

- Nếu rename nửa vời:
  - web hiện `AVmatrix`
  - CLI vẫn là `gitnexus`
  - storage vẫn là `.gitnexus`
  - MCP resource vẫn là `gitnexus://`
  => user và agent vẫn lẫn context

## 4. Vỡ test / e2e / setup

- Nhiều test và docs hiện hard-code `gitnexus`, `.gitnexus`, `gitnexus://`, `gitnexus mcp`
- Nếu không có plan rename theo phase, rất dễ gãy hàng loạt

# Chiến lược migration được đề xuất

## Giai đoạn A: Command/MCP alias transition

- `avmatrix` trở thành đường chính cho local usage
- `gitnexus` vẫn còn tồn tại như compatibility shim
- MCP config mới dùng `avmatrix mcp`
- docs mới chỉ nói `avmatrix`
- code vẫn có thể giữ alias command cũ trong giai đoạn đầu

## Giai đoạn B: AVmatrix-first storage/config

- ghi mới vào `.avmatrix` và `~/.avmatrix`
- chạy migration một lần từ `.gitnexus` và `~/.gitnexus`
- sau migration, runtime chỉ đọc `.avmatrix` và `~/.avmatrix`

## Giai đoạn C: Alias cleanup

- sau khi xác nhận toàn bộ flow local dùng ổn:
  - giảm dần docs cũ
  - đánh dấu `gitnexus` là legacy alias
  - về sau mới quyết định bỏ hẳn alias cũ hay không

# Pha 0: Audit rename surface

- Repo đụng: toàn repo ở mức read-only audit
- Mục tiêu:
  - thống kê tất cả nơi cần rename
  - phân loại cái gì là user-facing, cái gì là internal
  - xác định chỗ nào cần compatibility shim

## Checklist

- Audit toàn bộ text `GitNexus`, `gitnexus`, `.gitnexus`, `gitnexus://`
- Audit:
  - CLI command registrations
  - package.json `name`, `bin`, scripts
  - docs
  - web UI copy
  - MCP config/setup helpers
  - repo-manager storage paths
  - AI-context generator / skill installer / generated skill paths
  - env vars and home-dir loaders
  - tests/e2e/fixtures
  - README/skills/Onboarding
- Phân loại inventory thành:
  - user-facing
  - setup/config
  - storage/data
  - internal imports
  - tests/docs

## Deliverable

- Một inventory checklist rõ ràng để không rename sót surface nào
- File inventory thực tế: `docs/avmatrix-rename-audit.md`

# Pha 0.5: Docs/spec canonical trước khi đổi code

- Repo đụng:
  - `docs/`
  - `README.md`
  - setup docs / local usage docs / migration notes

## Mục tiêu

- Chốt spec rename bằng docs trước khi đổi code.
- Sau phase này, mọi quyết định tên gọi phải đã được viết rõ trong docs:
  - `AVmatrix`
  - `avmatrix`
  - `avmatrix mcp`
  - `.avmatrix`
  - `~/.avmatrix`
  - `avmatrix://`

## Checklist

- Có bảng mapping canonical từ brand cũ sang brand mới
- Docs phải nói rõ phase đầu tiên là rename ở docs/spec, code chỉ được đổi sau khi mapping docs đã khóa
- Docs local usage / setup / migration giải thích rõ command canonical mới
- Docs nêu rõ chỗ nào còn là compatibility alias, chỗ nào là canonical mới
- Không để code rename đi trước docs rename

## Acceptance

- Khi bắt đầu đổi code, cả bạn và agent đều đã có một spec docs rõ ràng để bám theo
- Không còn phải “đoán” chỗ nào nên hiện `GitNexus`, chỗ nào nên hiện `AVmatrix`
- Có file spec canonical riêng: `docs/avmatrix-canonical-spec.md`

# Pha 1: Brand rename ở user-facing surfaces

- Repo đụng:
  - `gitnexus-web/`
  - `gitnexus/src/cli/`
  - docs/help/readme

## Mục tiêu

- Người dùng nhìn thấy `AVmatrix` thay vì `GitNexus`
- Chưa động sâu vào storage namespace nếu chưa cần

## Những nơi phải đổi

- Web title, onboarding, help, empty states, loading copy
- Header/app name/logo text
- CLI help banners
- MCP onboarding text
- session status messages
- setup instructions
- README / local usage docs / skills docs
- generated AI context copy (`AGENTS.md`, `CLAUDE.md`) trong cùng rollout đầu

## Acceptance

- Mọi surface người dùng hay nhìn thấy đều nói `AVmatrix`
- Không còn text `GitNexus` trên active local-only UI/CLI trừ chỗ giải thích migration

# Pha 2: Command rename và CLI alias strategy

- Repo đụng:
  - package metadata
  - CLI bootstrap
  - setup docs
  - scripts liên quan

## Mục tiêu

- Người dùng local gọi `avmatrix`
- `gitnexus` vẫn có thể tồn tại như alias/compatibility trong giai đoạn chuyển tiếp

## Quyết định cần chốt

- Có đổi package npm name ngay hay không
- Có đổi thư mục `gitnexus/` ngay hay không
- Có publish package mới hay chỉ dùng local bin alias

## Đề xuất tốt nhất cho giai đoạn đầu

- Chưa đổi tên thư mục package nội bộ ngay
- Thêm/localize bin name `avmatrix`
- Giữ `gitnexus` như alias tạm thời
- Docs/setup chỉ hướng dẫn `avmatrix`

## Checklist

- `package.json` bin có `avmatrix`
- local install/link tạo được command `avmatrix`
- `avmatrix --help` tương đương `gitnexus --help`
- `avmatrix mcp`, `avmatrix analyze`, `avmatrix query`, `avmatrix impact`, `avmatrix detect-changes` đều hoạt động

## Acceptance

- Local user có thể bỏ hẳn `gitnexus` và chỉ dùng `avmatrix`

# Pha 3: MCP rename và Codex/Claude integration

- Repo đụng:
  - setup code
  - docs
  - MCP help/config generation
  - compatibility tests

## Mục tiêu

- MCP server name hiển thị là `avmatrix`
- Canonical MCP command là `avmatrix mcp`
- Codex/Claude config không còn dùng `gitnexus mcp` như đường chính

## Checklist

- setup helpers sinh config:
  - `[mcp_servers.avmatrix]`
  - `command = "avmatrix"`
  - `args = ["mcp"]`
- docs local-only đổi sang `avmatrix mcp`
- bỏ `gitnexus mcp` khỏi docs chính, chỉ giữ nếu ghi rõ là alias cũ
- test end-to-end với Codex/Claude local config

## Acceptance

- Codex CLI và Claude Code CLI đều nhận MCP dưới tên `avmatrix`
- agent không còn bị lẫn giữa local `AVmatrix` và upstream `GitNexus`

# Pha 4: Storage namespace migration

- Repo đụng:
  - `repo-manager`
  - runtime config
  - registry/config loaders
  - analyze/index paths

## Mục tiêu

- namespace local chuyển từ `.gitnexus` sang `.avmatrix`
- nhưng không làm mất dữ liệu/index cũ

## Checklist

- Repo-local storage:
  - `.avmatrix` là primary
- User-global storage:
  - `~/.avmatrix` là primary
- Env:
  - `AVMATRIX_HOME` là env var primary duy nhất sau cutover
- Registry migration:
  - nếu `~/.avmatrix/registry.json` chưa có mà `~/.gitnexus/registry.json` có
  - migrate sang namespace mới
- Config migration:
  - tương tự cho global config/runtime config
- Analyze mới:
  - ghi index vào `.avmatrix`
- Repo cũ đã index:
  - phải có migration step rõ ràng trước khi dùng runtime mới

## Acceptance

- Sau rename, repo đã index trước đó vẫn hiện trong UI/MCP/CLI
- Không bắt người dùng phải re-analyze toàn bộ chỉ vì đổi tên brand
- Runtime sau migration không còn đọc `.gitnexus` / `~/.gitnexus` như fallback

# Pha 5: Resource scheme và protocol namespace

- Repo đụng:
  - MCP resources
  - docs
  - AI context generation

## Mục tiêu

- `gitnexus://...` đổi sang `avmatrix://...`

## Checklist

- resources/context/process URIs đổi brand
- docs/skills/AI context sections đổi theo
- migration tools phải rewrite toàn bộ reference cũ sang `avmatrix://`
- runtime active path và generator chỉ dùng `avmatrix://`

## Acceptance

- Agent, docs, and user prompts chỉ nhìn thấy `avmatrix://...` như chuẩn mới

# Pha 6: Web UI active path cleanup

- Repo đụng:
  - `gitnexus-web/`

## Mục tiêu

- Web UI không còn lộ brand/namespace cũ trên active path
- Không đổi thiết kế, không mất tính năng

## Checklist

- Header / onboarding / help / repo picker / settings / right panel / errors
- URLs/query params nếu có brand-specific wording
- empty states / loading states / progress copy
- local runtime settings labels
- e2e snapshots / expectations

## Acceptance

- Dùng web UI local chỉ thấy `AVmatrix`
- Không còn `GitNexus` lộ ra trên active local flow trừ trang migration/debug

# Pha 7: Docs, setup, examples, prompts, skills

- Repo đụng:
  - `README`
  - `docs/`
  - `skills/`
  - plan references
  - setup outputs/help text

## Checklist

- README chính
- local usage guide
- MCP setup docs
- Codex/Claude setup docs
- skills docs
- examples trong error messages/help banners
- báo cáo migration notes

## Acceptance

- Người mới đọc docs sẽ cài và dùng bằng `AVmatrix`, không đi theo `GitNexus` nữa

# Pha 8: Test matrix và compatibility lock

- Repo đụng:
  - unit tests
  - integration tests
  - e2e tests

## Mục tiêu

- rename không làm đứt behavior
- alias command/MCP và migration path được khóa bằng tests

## Behavioral tests bắt buộc

### CLI

- `avmatrix --help`
- `avmatrix analyze`
- `avmatrix query`
- `avmatrix context`
- `avmatrix impact`
- `avmatrix detect-changes`
- `avmatrix mcp`

### MCP

- config generation dùng `avmatrix`
- Codex/Claude đọc server `avmatrix`
- tool calls vẫn hoạt động như trước

### Storage migration

- migration một lần từ namespace cũ sang `.avmatrix` / `~/.avmatrix` thành công
- sau migration, runtime chỉ đọc namespace mới
- analyze mới ghi vào `.avmatrix`

### Web

- onboarding/help/header/repo picker không còn text cũ
- repo switch/re-analyze/chat/graph vẫn hoạt động

### Docs/setup

- examples không còn hard-code `gitnexus` như đường chính

# Pha 9: Optional internal package/import rename

- Chỉ làm nếu thật sự cần.
- Đây là phần rủi ro cao nhất.

## Những thứ có thể đổi ở pha này

- tên package `gitnexus`
- `gitnexus-web`
- `gitnexus-shared`
- thư mục source/package
- import paths nội bộ

## Khuyến nghị

- Không làm pha này trong đợt đầu nếu mục tiêu chính chỉ là:
  - tránh nhầm lẫn context
  - đổi mặt dùng local sang `AVmatrix`
- Có thể giữ package/internal folder cũ thêm một thời gian nếu surface đã sạch

# Inventory sơ bộ cần audit/đụng

## CLI / Core

- `gitnexus/package.json`
- `gitnexus/src/cli/index.ts`
- `gitnexus/src/cli/setup.ts`
- `gitnexus/src/cli/mcp.ts`
- `gitnexus/src/cli/status.ts`
- `gitnexus/src/cli/list.ts`
- `gitnexus/src/cli/ai-context.ts`
- `gitnexus/src/storage/repo-manager.ts`
- `gitnexus/src/storage/runtime-config.ts`
- `gitnexus/src/core/group/storage.ts`
- `gitnexus/src/server/api.ts`
- `gitnexus/src/mcp/resources.ts`
- `gitnexus/src/mcp/tools.ts`

## Web

- `gitnexus-web/src/App.tsx`
- `gitnexus-web/src/components/Header.tsx`
- `gitnexus-web/src/components/OnboardingGuide.tsx`
- `gitnexus-web/src/components/HelpPanel.tsx`
- `gitnexus-web/src/components/RepoLanding.tsx`
- `gitnexus-web/src/components/SettingsPanel*`
- `gitnexus-web/src/hooks/useAppState*`
- `gitnexus-web/src/services/backend-client.ts`

## Docs

- root `README.md`
- `gitnexus/README.md`
- `RUNBOOK.md`
- `TESTING.md`
- `CONTRIBUTING.md`
- `docs/local-usage.md`
- `docs/plans/*.md` liên quan
- `gitnexus/skills/*.md`
- `AGENTS.md`
- `CLAUDE.md`
- `.gitignore`
- active `.cursor` / plugin metadata / setup artifacts nếu còn lộ `gitnexus`

## Auto-generated local surfaces

- `AGENTS.md` / `CLAUDE.md` GitNexus block generator
- `.claude/skills/gitnexus/*`
- `.claude/skills/generated/*` references nếu còn copy brand cũ vào generated content

## Tests

- CLI help/setup tests
- MCP runtime alignment tests
- onboarding/local-only tests
- package dep tests
- repo-manager migration tests
- migration tests cho `.gitnexus -> .avmatrix`

# Migration strategy cho config và data

## User-global

- Primary mới:
  - `~/.avmatrix/config.json`
  - `~/.avmatrix/registry.json`
  - `~/.avmatrix/runtime.json`
- Env var mới:
  - `AVMATRIX_HOME`
- Migration:
  - explicit migration step hoặc one-time startup migration
  - sau khi migrate xong, runtime chỉ đọc file mới
  - `GITNEXUS_HOME` không còn được runtime dùng như fallback sau cutover

## Repo-local

- Primary mới:
  - `<repo>/.avmatrix/`
- Migration:
  - migrate metadata/index sang `.avmatrix`
  - sau khi migrate xong, runtime chỉ đọc `.avmatrix`
  - `.gitnexus/` chỉ còn là source để migrate một lần, không phải fallback runtime

## Khuyến nghị

- Dùng **migration một lần + cutover sạch**
- Không giữ fallback runtime lâu dài cho storage/env

# Rollback strategy

- Nếu rename user-facing surface gây regression:
  - rollback phase đó trước, không rollback toàn bộ repo nếu các phase khác vẫn ổn
- Nếu namespace storage migration gây mất repo/index:
  - rollback phase migration
  - restore lại code/runtime của phase trước, không duy trì fallback song song trong steady state
- Nếu command `avmatrix` chưa ổn:
  - giữ `gitnexus` làm canonical tạm thời
  - nhưng vẫn để brand UI là `AVmatrix` nếu cần

# Open questions cần chốt trước khi code

- Có muốn đổi luôn:
  - tên package npm
  - tên thư mục `gitnexus/`, `gitnexus-web/`, `gitnexus-shared`
  hay chỉ đổi surface user-facing + command + namespace?

# Quyết định đã chốt cho V1

- `V1` chỉ đổi surface user-facing + command + MCP + storage namespace + resource scheme.
- `V1` **không** đổi package/folder/import nội bộ hàng loạt.
- `avmatrix` là canonical command mới.
- `gitnexus` được giữ làm compatibility alias cho command/MCP trong giai đoạn đầu.
- `AVMATRIX_HOME` là env var primary mới.
- Không giữ fallback runtime cho `.gitnexus` / `~/.gitnexus` / `GITNEXUS_HOME`.
- Docs/spec là phase đầu tiên; code chỉ được đổi sau khi docs đã khóa canonical names.
- Dữ liệu namespace cũ chỉ được dùng làm migration source một lần.
- Generator của `AGENTS.md` / `CLAUDE.md` / skills local phải được đổi cùng rollout đầu.

# Đề xuất triển khai tốt nhất

- V1 nên làm theo thứ tự:
  1. Pha 0 audit
  2. Pha 0.5 docs/spec canonical
  3. Pha 1 brand rename
  4. Pha 2 + Pha 3 trong cùng rollout đầu để command/MCP/docs không lệch nhau
  5. Pha 4 storage/config migration một lần
  6. Pha 5 resource scheme rename
  7. Pha 6 web active path cleanup
  8. Pha 7 docs/setup cleanup cuối
  9. Pha 8 tests/compatibility lock
- Pha 9 chỉ làm khi thật sự cần.

# Checklist tổng

- [x] Audit đầy đủ mọi surface `GitNexus` / `gitnexus` / `.gitnexus` / `gitnexus://`
- [x] Tạo inventory audit thực tế: `docs/avmatrix-rename-audit.md`
- [x] Chốt scope V1: surface user-facing + command/MCP + storage/resource scheme, chưa đổi package/internal hàng loạt
- [x] Chốt docs/spec canonical trước khi đổi code
- [x] Đổi brand hiển thị sang `AVmatrix`
- [x] Thêm command `avmatrix`
- [x] Giữ alias `gitnexus` cho command/MCP trong giai đoạn đầu
- [x] Đổi MCP canonical command sang `avmatrix mcp`
- [x] Đổi MCP server name trong config/setup sang `avmatrix`
- [x] Đổi docs/setup sang `AVmatrix`
- [x] Đổi storage primary sang `.avmatrix` / `~/.avmatrix`
- [x] Thêm migration một lần từ namespace cũ
- [x] Đổi resource scheme sang `avmatrix://`
- [x] Web UI active path không còn lộ `GitNexus`
- [x] Behavioral tests cho command/MCP/storage/web pass
- [x] Typecheck pass cho `gitnexus` và `gitnexus-web`

# Kết luận

- Rename này là hợp lý và có giá trị thật vì nó giải quyết **nhầm lẫn ngữ cảnh làm việc** giữa local toolchain của bạn và upstream `GitNexus`.
- Cách đúng không phải là find/replace toàn repo một phát.
- Cách đúng là:
  - đổi surface người dùng trước
  - đổi command/MCP namespace
  - migrate storage/config an toàn
  - khóa lại bằng behavioral tests
  - chỉ sau đó mới cân nhắc rename package/import nội bộ nếu cần.
