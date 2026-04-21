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
- Vẫn có compatibility path đủ tốt để dữ liệu/index cũ từ `.gitnexus` và `~/.gitnexus` không bị mất ngay lập tức.

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
- Cấm xóa trực tiếp surface cũ trước khi surface mới đạt parity.
- Với mọi surface lớn hoặc có nhiều trạng thái:
  - tạo file mới song song nếu cần
  - đạt parity giao diện + tính năng
  - có behavioral tests
  - chỉ sau đó mới thay thế file cũ
- Với các rename có thể gây đứt dữ liệu local:
  - phải có migration path hoặc compatibility shim
  - không đổi một bước kiểu làm mất index/config đang dùng
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

## Compatibility philosophy

- V1 của migration nên là:
  - **AVmatrix-first**
  - nhưng vẫn đọc được dữ liệu/config/index từ namespace cũ
- Nghĩa là:
  - ưu tiên ghi mới vào `.avmatrix` / `~/.avmatrix`
  - nhưng nếu chưa có dữ liệu mới, có thể đọc fallback từ `.gitnexus` / `~/.gitnexus`
- Chỉ xóa compatibility path khi đã xác nhận:
  - CLI
  - MCP
  - web UI
  - setup docs
  - config migration
  - repo registry
  đều ổn định

## Rollout philosophy

- Ưu tiên rename theo thứ tự:
  1. user-facing brand
  2. command/MCP/config namespace
  3. storage namespace
  4. docs/tests/setup
  5. package/import internals nếu thật sự cần
- Không đổi package/import quá sớm nếu chỉ rename surface là đủ để giải quyết nhầm lẫn context.

# Rủi ro chính cần kiểm soát

## 1. Đứt command đang dùng

- Nếu đổi thẳng từ `gitnexus` sang `avmatrix` mà không có shim:
  - Codex/Claude config cũ sẽ gãy
  - docs cũ sẽ sai
  - script local cũ sẽ fail

## 2. Mất index/config local

- Nếu đổi `.gitnexus` sang `.avmatrix` mà không migrate/fallback:
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

## Giai đoạn A: Dual-brand compatibility

- `avmatrix` trở thành đường chính cho local usage
- `gitnexus` vẫn còn tồn tại như compatibility shim
- MCP config mới dùng `avmatrix mcp`
- docs mới chỉ nói `avmatrix`
- code vẫn có khả năng đọc namespace cũ để migrate mềm

## Giai đoạn B: AVmatrix-first storage/config

- ghi mới vào `.avmatrix` và `~/.avmatrix`
- đọc fallback từ `.gitnexus` và `~/.gitnexus`
- có cơ chế copy/migrate registry/config/index khi cần

## Giai đoạn C: Compatibility cleanup

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
  - fallback đọc `.gitnexus`
- User-global storage:
  - `~/.avmatrix` là primary
  - fallback đọc `~/.gitnexus`
- Registry migration:
  - nếu `~/.avmatrix/registry.json` chưa có mà `~/.gitnexus/registry.json` có
  - copy hoặc migrate sang namespace mới
- Config migration:
  - tương tự cho global config/runtime config
- Analyze mới:
  - ghi index vào `.avmatrix`
- Repo cũ đã index:
  - vẫn đọc được hoặc có migration step rõ ràng

## Acceptance

- Sau rename, repo đã index trước đó vẫn hiện trong UI/MCP/CLI
- Không bắt người dùng phải re-analyze toàn bộ chỉ vì đổi tên brand

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
- nếu còn compatibility:
  - parse cả `gitnexus://` và `avmatrix://` trong giai đoạn đầu
  - nhưng chỉ generate `avmatrix://`

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
- compatibility path được khóa bằng tests

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

- repo cũ chỉ có `.gitnexus` vẫn load được
- user cũ chỉ có `~/.gitnexus/registry.json` vẫn hiện repo
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
- `docs/local-usage.md`
- `docs/plans/*.md` liên quan
- `gitnexus/skills/*.md`
- `AGENTS.md`
- `CLAUDE.md`

## Tests

- CLI help/setup tests
- MCP runtime alignment tests
- onboarding/local-only tests
- package dep tests
- repo-manager migration tests
- new compatibility tests cho `.gitnexus -> .avmatrix`

# Migration strategy cho config và data

## User-global

- Primary mới:
  - `~/.avmatrix/config.json`
  - `~/.avmatrix/registry.json`
  - `~/.avmatrix/runtime.json`
- Compatibility:
  - nếu file mới chưa tồn tại, thử đọc file cũ từ `~/.gitnexus`
- Migration:
  - lazy migration khi startup hoặc khi lần đầu save config

## Repo-local

- Primary mới:
  - `<repo>/.avmatrix/`
- Compatibility:
  - nếu `.avmatrix` chưa có nhưng `.gitnexus` có, cho phép đọc fallback
- Migration:
  - có thể:
    - copy metadata/index sang `.avmatrix`
    - hoặc dùng lazy read từ `.gitnexus` và ghi mới vào `.avmatrix` về sau

## Khuyến nghị

- Giai đoạn đầu nên dùng:
  - **lazy migration + fallback read**
- Không nên ép rename folder index ngay trên disk trong một bước

# Rollback strategy

- Nếu rename user-facing surface gây regression:
  - rollback phase đó trước, không rollback toàn bộ repo nếu các phase khác vẫn ổn
- Nếu namespace storage migration gây mất repo/index:
  - revert sang read `.gitnexus`
  - giữ write path tạm ở namespace cũ
- Nếu command `avmatrix` chưa ổn:
  - giữ `gitnexus` làm canonical tạm thời
  - nhưng vẫn để brand UI là `AVmatrix` nếu cần

# Open questions cần chốt trước khi code

- Có muốn đổi luôn:
  - tên package npm
  - tên thư mục `gitnexus/`, `gitnexus-web/`, `gitnexus-shared`
  hay chỉ đổi surface user-facing + command + namespace?
- Có muốn:
  - `gitnexus` là alias dài hạn
  - hay chỉ là alias tạm thời trong 1 giai đoạn migration?
- Có muốn auto-migrate `.gitnexus` -> `.avmatrix` ngay khi startup
  hay chỉ đọc fallback rồi chờ người dùng re-analyze/save?

# Đề xuất triển khai tốt nhất

- V1 nên làm theo thứ tự:
  1. Pha 0 audit
  2. Pha 1 brand rename
  3. Pha 2 command alias `avmatrix`
  4. Pha 3 MCP rename
  5. Pha 4 storage/config compatibility migration
  6. Pha 5 resource scheme rename
  7. Pha 6 web active path cleanup
  8. Pha 7 docs/setup
  9. Pha 8 tests/compatibility lock
- Pha 9 chỉ làm khi thật sự cần.

# Checklist tổng

- [ ] Audit đầy đủ mọi surface `GitNexus` / `gitnexus` / `.gitnexus` / `gitnexus://`
- [ ] Chốt scope rename: surface-only hay cả package/internal
- [ ] Đổi brand hiển thị sang `AVmatrix`
- [ ] Thêm command `avmatrix`
- [ ] Giữ hoặc chốt alias `gitnexus`
- [ ] Đổi MCP canonical command sang `avmatrix mcp`
- [ ] Đổi MCP server name trong config/setup sang `avmatrix`
- [ ] Đổi docs/setup sang `AVmatrix`
- [ ] Đổi storage primary sang `.avmatrix` / `~/.avmatrix`
- [ ] Thêm fallback/migration từ namespace cũ
- [ ] Đổi resource scheme sang `avmatrix://`
- [ ] Web UI active path không còn lộ `GitNexus`
- [ ] Behavioral tests cho command/MCP/storage/web pass
- [ ] Typecheck pass cho `gitnexus` và `gitnexus-web`

# Kết luận

- Rename này là hợp lý và có giá trị thật vì nó giải quyết **nhầm lẫn ngữ cảnh làm việc** giữa local toolchain của bạn và upstream `GitNexus`.
- Cách đúng không phải là find/replace toàn repo một phát.
- Cách đúng là:
  - đổi surface người dùng trước
  - đổi command/MCP namespace
  - migrate storage/config an toàn
  - khóa lại bằng behavioral tests
  - chỉ sau đó mới cân nhắc rename package/import nội bộ nếu cần.
