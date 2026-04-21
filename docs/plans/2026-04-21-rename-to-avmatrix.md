# Mục đích plan này

- Plan này dùng để đổi tên toàn bộ bản local hiện tại từ `AVmatrix` sang `AVmatrix`.
- Mục tiêu chính không phải là rebrand cho đẹp, mà là **tách ngữ cảnh vận hành** giữa:
  - môi trường local/fork tùy biến của bạn
  - môi trường upstream / GitHub / npm mang tên `AVmatrix`
- Sau khi đổi xong, agent làm việc với bạn sẽ không còn nhầm giữa:
  - CLI local bạn đang dùng
  - MCP local bạn đang cấu hình cho Codex/Claude Code
  - docs/resources/storage/config của môi trường local
  - tên upstream `AVmatrix` còn tồn tại trên GitHub, npm, báo cáo cũ, hoặc tài liệu cũ

# Mục tiêu thành công

- Tên hiển thị trên web UI đổi sang `AVmatrix`.
- CLI người dùng gọi bằng `avmatrix`.
- MCP server trong Codex/Claude Code hiện là `avmatrix`, không còn `avmatrix`.
- Resource scheme và docs vận hành local đổi sang `avmatrix://...`.
- Namespace local đổi sang:
  - `.avmatrix`
  - `~/.avmatrix`
- Không làm mất bất kỳ feature nào của web UI, CLI, MCP, graph, query, impact, detect-changes, analyze, multi-repo, chat, hoặc local-only flow.
- Không được đổi hành vi sản phẩm ngoài phạm vi rename/namespace migration trừ khi cần để tương thích namespace mới.
- Có migration path rõ ràng để dữ liệu/index cũ từ `.avmatrix` và `~/.avmatrix` được chuyển sang namespace mới, nhưng runtime sau migration chỉ dùng namespace mới.

# Scope được chốt cho rollout triệt để

- Rollout này **không dừng ở surface user-facing**.
- Tất cả những gì còn mang namespace `AVmatrix` / `avmatrix` đều phải được đổi sang `AVmatrix` / `avmatrix`, gồm:
  - surface user-facing
  - command CLI
  - MCP server name / setup output
  - storage namespace
  - resource scheme
  - package names nội bộ
  - tên file và tên thư mục
  - import paths nội bộ
  - generated files và generated skill paths
  - env vars
  - comments còn sống trong code
  - docs, tests, fixtures, snapshots
- Rollout này chỉ được coi là xong khi codebase active path đã sạch namespace cũ.

# Kết quả mong muốn sau cùng

- Khi làm việc local, cả bạn và agent chỉ nhìn thấy:
  - `AVmatrix`
  - `avmatrix`
  - `.avmatrix`
  - `~/.avmatrix`
  - `avmatrix://...`
- Cái tên `AVmatrix` chỉ còn được giữ ở những nơi thật sự cần thiết:
  - lịch sử git
  - tài liệu migration/historical note có chủ đích
  - tham chiếu upstream/GitHub/npm cũ khi cần nói về dự án upstream như một external reference

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
  - không để lẫn lộn MCP name `avmatrix` và `avmatrix`
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

- `.avmatrix` -> `.avmatrix`
- `~/.avmatrix` -> `~/.avmatrix`
- `avmatrix://` -> `avmatrix://`
- `avmatrix mcp` -> `avmatrix mcp`
- `avmatrix analyze` -> `avmatrix analyze`
- `AVMATRIX_HOME` -> `AVMATRIX_HOME` bằng migration/cutover sạch, không giữ fallback runtime

## 3. Repo/package/codebase internals

- package names
- bin names
- tên file và tên thư mục
- import paths nội bộ
- exported constants/types có brand name
- comments còn sống trong code
- docs/examples/tests/fixtures
- runtime config and registry loaders
- MCP setup templates

## 4. Những thứ chỉ được giữ trong giai đoạn chuyển tiếp ngắn

- migration notes nói về tên cũ
- shim command tạm thời nếu thật sự cần để bootstrap migration
- parser/chuyển đổi đọc namespace cũ trong migration step

Sau khi hoàn thành plan:
- không giữ `avmatrix` trong file names/thư mục/package/import/comment active path
- không giữ `avmatrix` là command canonical
- không giữ `avmatrix` trong generated outputs

# Quyết định kiến trúc nên chốt trước

## Brand mới

- Brand local mới là: `AVmatrix`
- CLI command mục tiêu: `avmatrix`
- MCP server name mục tiêu: `avmatrix`
- Resource scheme mục tiêu: `avmatrix://`
- Repo-local hidden folder mục tiêu: `.avmatrix`
- User-global config dir mục tiêu: `~/.avmatrix`

## Migration philosophy

- Migration phải theo kiểu:
  - **AVmatrix-first**
  - **migrate một lần**
  - **cutover sạch**
- Nghĩa là:
  - runtime primary chỉ dùng `.avmatrix` / `~/.avmatrix`
  - dữ liệu cũ từ `.avmatrix` / `~/.avmatrix` chỉ được dùng làm **migration source**
  - migration diễn ra một lần, có thể theo command riêng hoặc explicit upgrade step
- Sau migration:
  - source of truth duy nhất là `.avmatrix` / `~/.avmatrix`
  - `AVMATRIX_HOME` không còn là env runtime chính
  - `.avmatrix` / `~/.avmatrix` không còn là namespace active path

## Alias philosophy

- `avmatrix` là command canonical duy nhất sau khi rollout hoàn thành.
- Nếu cần giữ `avmatrix` làm shim tạm để bootstrap migration:
  - phải ghi rõ phase tồn tại của shim
  - phải có bước xóa shim ở cuối plan
- Docs/setup/onboarding chỉ hướng dẫn `avmatrix`.
- Generator/output mới chỉ sinh `avmatrix://...`.
- Runtime steady state không được còn alias `avmatrix` trên active path.

## Rollout philosophy

- Ưu tiên rename theo thứ tự:
  1. docs/spec canonical
  2. user-facing brand
  3. command/MCP/config namespace
  4. storage namespace
  5. generated surfaces
  6. package/folder/file rename
  7. import path rename
  8. comment/docs/tests/final scrub
- Không dừng ở “surface sạch”.
- Chỉ được coi là hoàn thành khi codebase active path đã sạch namespace cũ.

## Install story phải chốt ngay

- Command người dùng cuối phải là `avmatrix`.
- Nếu cần shim `avmatrix` trong migration window:
  - shim chỉ tồn tại tạm thời
  - cuối plan phải có bước loại bỏ hoặc đánh dấu legacy không còn dùng trong active docs/tests
- Sau `npm link` hoặc local install:
  - command chính phải là `avmatrix`
  - generated setup/config phải chỉ ghi `avmatrix`

# Rủi ro chính cần kiểm soát

## 1. Đứt command đang dùng

- Nếu đổi thẳng từ `avmatrix` sang `avmatrix` mà không có shim:
  - Codex/Claude config cũ sẽ gãy
  - docs cũ sẽ sai
  - script local cũ sẽ fail

## 2. Mất index/config local

- Nếu đổi `.avmatrix` sang `.avmatrix` mà không có migration path rõ ràng:
  - repo đã index sẽ biến mất khỏi UI
  - registry sẽ rỗng
  - MCP sẽ không thấy repo cũ

## 3. Lẫn brand mới/cũ

- Nếu rename nửa vời:
  - web hiện `AVmatrix`
  - CLI vẫn là `avmatrix`
  - storage vẫn là `.avmatrix`
  - MCP resource vẫn là `avmatrix://`
  => user và agent vẫn lẫn context

## 4. Vỡ test / e2e / setup

- Nhiều test và docs hiện hard-code `avmatrix`, `.avmatrix`, `avmatrix://`, `avmatrix mcp`
- Nếu không có plan rename theo phase, rất dễ gãy hàng loạt

# Chiến lược migration được đề xuất

## Giai đoạn A: Command/MCP alias transition

- `avmatrix` trở thành đường chính cho local usage
- `avmatrix` chỉ tồn tại như migration shim nếu thật sự cần
- MCP config mới dùng `avmatrix mcp`
- docs mới chỉ nói `avmatrix`
- code chỉ được giữ alias command cũ trong giai đoạn migration có kiểm soát

## Giai đoạn B: AVmatrix-first storage/config

- ghi mới vào `.avmatrix` và `~/.avmatrix`
- chạy migration một lần từ `.avmatrix` và `~/.avmatrix`
- sau migration, runtime chỉ đọc `.avmatrix` và `~/.avmatrix`

## Giai đoạn C: Alias cleanup

- sau khi xác nhận toàn bộ flow local dùng ổn:
  - xóa docs cũ khỏi active path
  - loại bỏ `avmatrix` khỏi command/MCP active path
  - giữ tên cũ chỉ trong migration/historical note có chủ đích

# Pha 0: Audit rename surface

- Repo đụng: toàn repo ở mức read-only audit
- Mục tiêu:
  - thống kê tất cả nơi cần rename
  - phân loại cái gì là user-facing, cái gì là internal
  - xác định chỗ nào cần compatibility shim

## Checklist

- Audit toàn bộ text `AVmatrix`, `avmatrix`, `.avmatrix`, `avmatrix://`
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
- Docs nêu rõ nếu có shim migration tạm thời thì shim đó sống ở phase nào và bị loại bỏ ở phase nào
- Không để code rename đi trước docs rename

## Acceptance

- Khi bắt đầu đổi code, cả bạn và agent đều đã có một spec docs rõ ràng để bám theo
- Không còn phải “đoán” chỗ nào nên hiện `AVmatrix`, chỗ nào nên hiện `AVmatrix`
- Có file spec canonical riêng: `docs/avmatrix-canonical-spec.md`

# Pha 1: Brand rename ở user-facing surfaces

- Repo đụng:
  - `avmatrix-web/`
  - `avmatrix/src/cli/`
  - docs/help/readme

## Mục tiêu

- Người dùng nhìn thấy `AVmatrix` thay vì `AVmatrix`
- Đồng thời tạo spec/copy chuẩn để các pha rename nội bộ bám theo

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
- Không còn text `AVmatrix` trên active local-only UI/CLI trừ chỗ giải thích migration

# Pha 2: Command rename, package metadata, và cutover CLI

- Repo đụng:
  - package metadata
  - CLI bootstrap
  - setup docs
  - scripts liên quan
  - package bins / package names

## Mục tiêu

- Người dùng local gọi `avmatrix`
- package metadata và generated setup đều canonical theo `avmatrix`
- nếu có shim `avmatrix` thì chỉ là bước migration tạm thời, không phải steady state

## Checklist

- `package.json` bin có `avmatrix`
- package metadata không còn coi `avmatrix` là brand canonical
- local install/link tạo được command `avmatrix`
- `avmatrix --help` hoạt động và phản ánh brand mới
- `avmatrix mcp`, `avmatrix analyze`, `avmatrix query`, `avmatrix impact`, `avmatrix detect-changes` đều hoạt động
- nếu còn shim `avmatrix`, plan phải có bước xóa shim ở phase cuối

## Acceptance

- Local user chỉ cần `avmatrix`
- `avmatrix` không còn là đường canonical trong docs, setup, package metadata, hoặc active CLI branding

# Pha 3: MCP rename và Codex/Claude integration

- Repo đụng:
  - setup code
  - docs
  - MCP help/config generation
  - compatibility tests

## Mục tiêu

- MCP server name hiển thị là `avmatrix`
- Canonical MCP command là `avmatrix mcp`
- Codex/Claude config không còn dùng `avmatrix mcp` như đường chính

## Checklist

- setup helpers sinh config:
  - `[mcp_servers.avmatrix]`
  - `command = "avmatrix"`
  - `args = ["mcp"]`
- docs local-only đổi sang `avmatrix mcp`
- bỏ `avmatrix mcp` khỏi docs chính; nếu còn shim thì phải nằm trong migration note riêng
- test end-to-end với Codex/Claude local config

## Acceptance

- Codex CLI và Claude Code CLI đều nhận MCP dưới tên `avmatrix`
- agent không còn bị lẫn giữa local `AVmatrix` và upstream `AVmatrix`
- setup/generated config không còn emit `avmatrix` trên active path

# Pha 4: Storage namespace migration

- Repo đụng:
  - `repo-manager`
  - runtime config
  - registry/config loaders
  - analyze/index paths

## Mục tiêu

- namespace local chuyển từ `.avmatrix` sang `.avmatrix`
- nhưng không làm mất dữ liệu/index cũ

## Checklist

- Repo-local storage:
  - `.avmatrix` là primary
- User-global storage:
  - `~/.avmatrix` là primary
- Env:
  - `AVMATRIX_HOME` là env var primary duy nhất sau cutover
- Registry migration:
  - nếu `~/.avmatrix/registry.json` chưa có mà `~/.avmatrix/registry.json` có
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
- Runtime sau migration không còn đọc `.avmatrix` / `~/.avmatrix` như fallback
- migration xong thì namespace cũ chỉ còn là historical source đã tiêu thụ xong

# Pha 5: Resource scheme và protocol namespace

- Repo đụng:
  - MCP resources
  - docs
  - AI context generation

## Mục tiêu

- `avmatrix://...` đổi sang `avmatrix://...`

## Checklist

- resources/context/process URIs đổi brand
- docs/skills/AI context sections đổi theo
- migration tools phải rewrite toàn bộ reference cũ sang `avmatrix://`
- runtime active path và generator chỉ dùng `avmatrix://`

## Acceptance

- Agent, docs, and user prompts chỉ nhìn thấy `avmatrix://...` như chuẩn mới

# Pha 6: Web UI active path cleanup

- Repo đụng:
  - `avmatrix-web/`

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
- Không còn `AVmatrix` lộ ra trên active local flow trừ trang migration/debug

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

- Người mới đọc docs sẽ cài và dùng bằng `AVmatrix`, không đi theo `AVmatrix` nữa
- generated skills / AI context cũng không còn file names và headings mang `avmatrix`

# Pha 8: Test matrix và compatibility lock

- Repo đụng:
  - unit tests
  - integration tests
  - e2e tests

## Mục tiêu

- rename không làm đứt behavior
- migration path và active namespace mới được khóa bằng tests

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

- examples không còn hard-code `avmatrix` như đường chính

# Pha 9: Rename nội bộ triệt để package, folder, file, import

- Đây là phase bắt buộc để kết thúc rollout.
- Đây cũng là phase rủi ro cao nhất nên phải chia batch nhỏ, có test sau mỗi batch.

## Những thứ phải đổi ở pha này

- tên package `avmatrix`
- tên package `avmatrix-web`
- tên package `avmatrix-shared`
- thư mục source/package
- import paths nội bộ
- file names và directory names còn mang `avmatrix`
- generated skill source names còn mang `avmatrix`

## Acceptance

- Không còn package/file/folder/import active nào mang `avmatrix`
- Generated outputs và source skills đều mang `avmatrix`
- Alias/shim tạm thời, nếu có, đã được loại bỏ hoặc tách hẳn sang migration note không nằm trên active path

# Pha 10: Final scrub cho comments, docs, fixtures, tests, snapshots

- Đây là phase dọn sạch cuối cùng.

## Những thứ phải đổi ở pha này

- comments còn sống trong code
- fixture names
- test titles và snapshot text
- plan references / internal docs còn sót
- hidden paths / examples / help text còn lộ `avmatrix`

## Acceptance

- Không còn `avmatrix` trong active codebase trừ:
  - lịch sử git
  - migration/historical notes có chủ đích
  - external upstream references có chủ đích

# Inventory sơ bộ cần audit/đụng

## CLI / Core

- `avmatrix/package.json`
- `avmatrix/src/cli/index.ts`
- `avmatrix/src/cli/setup.ts`
- `avmatrix/src/cli/mcp.ts`
- `avmatrix/src/cli/status.ts`
- `avmatrix/src/cli/list.ts`
- `avmatrix/src/cli/ai-context.ts`
- `avmatrix/src/storage/repo-manager.ts`
- `avmatrix/src/storage/runtime-config.ts`
- `avmatrix/src/core/group/storage.ts`
- `avmatrix/src/server/api.ts`
- `avmatrix/src/mcp/resources.ts`
- `avmatrix/src/mcp/tools.ts`

## Web

- `avmatrix-web/src/App.tsx`
- `avmatrix-web/src/components/Header.tsx`
- `avmatrix-web/src/components/OnboardingGuide.tsx`
- `avmatrix-web/src/components/HelpPanel.tsx`
- `avmatrix-web/src/components/RepoLanding.tsx`
- `avmatrix-web/src/components/SettingsPanel*`
- `avmatrix-web/src/hooks/useAppState*`
- `avmatrix-web/src/services/backend-client.ts`

## Docs

- root `README.md`
- `avmatrix/README.md`
- `RUNBOOK.md`
- `TESTING.md`
- `CONTRIBUTING.md`
- `docs/local-usage.md`
- `docs/plans/*.md` liên quan
- `avmatrix/skills/*.md`
- `AGENTS.md`
- `CLAUDE.md`
- `.gitignore`
- active `.cursor` / plugin metadata / setup artifacts nếu còn lộ `avmatrix`

## Auto-generated local surfaces

- `AGENTS.md` / `CLAUDE.md` AVmatrix block generator
- `.claude/skills/avmatrix/*`
- `.claude/skills/generated/*` references nếu còn copy brand cũ vào generated content

## Tests

- CLI help/setup tests
- MCP runtime alignment tests
- onboarding/local-only tests
- package dep tests
- repo-manager migration tests
- migration tests cho `.avmatrix -> .avmatrix`

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
  - `AVMATRIX_HOME` không còn được runtime dùng như fallback sau cutover

## Repo-local

- Primary mới:
  - `<repo>/.avmatrix/`
- Migration:
  - migrate metadata/index sang `.avmatrix`
  - sau khi migrate xong, runtime chỉ đọc `.avmatrix`
  - `.avmatrix/` chỉ còn là source để migrate một lần, không phải fallback runtime

## Khuyến nghị

- Dùng **migration một lần + cutover sạch**
- Không giữ fallback runtime lâu dài cho storage/env

# Rollback strategy

- Nếu rename user-facing surface gây regression:
  - rollback phase đó trước, không rollback toàn bộ repo nếu các phase khác vẫn ổn
- Nếu namespace storage migration gây mất repo/index:
  - rollback phase migration
  - restore lại code/runtime của phase trước, không duy trì fallback song song trong steady state
- Nếu command `avmatrix` chưa ổn trong một batch:
  - rollback đúng batch đó
  - không quay lại triết lý giữ `avmatrix` làm canonical lâu dài

# Quyết định đã chốt cho rollout triệt để

- `avmatrix` là canonical command mới.
- Nếu có shim `avmatrix`, shim chỉ được tồn tại trong migration window và phải bị loại bỏ trước khi plan kết thúc.
- `AVMATRIX_HOME` là env var primary mới.
- Không giữ fallback runtime cho `.avmatrix` / `~/.avmatrix` / `AVMATRIX_HOME`.
- Docs/spec là phase đầu tiên; code chỉ được đổi sau khi docs đã khóa canonical names.
- Dữ liệu namespace cũ chỉ được dùng làm migration source một lần.
- Generator của `AGENTS.md` / `CLAUDE.md` / skills local phải được đổi cùng rollout đầu.
- Package names, folder names, import paths, file names, comments, generated outputs đều nằm trong scope bắt buộc của rollout.

# Đề xuất triển khai tốt nhất

- Rollout nên làm theo thứ tự:
  1. Pha 0 audit
  2. Pha 0.5 docs/spec canonical
  3. Pha 1 brand rename
  4. Pha 2 + Pha 3 trong cùng rollout để command/MCP/docs không lệch nhau
  5. Pha 4 storage/config migration một lần
  6. Pha 5 resource scheme rename
  7. Pha 6 web active path cleanup
  8. Pha 7 docs/setup cleanup cuối
  9. Pha 8 tests/compatibility lock
  10. Pha 9 rename nội bộ triệt để
  11. Pha 10 final scrub

# Checklist tổng

- [x] Audit đầy đủ mọi surface `AVmatrix` / `avmatrix` / `.avmatrix` / `avmatrix://`
- [x] Tạo inventory audit thực tế: `docs/avmatrix-rename-audit.md`
- [x] Chốt scope rollout triệt để: package/folder/import/file/comment đều nằm trong scope
- [x] Chốt docs/spec canonical trước khi đổi code
- [x] Đổi brand hiển thị sang `AVmatrix`
- [x] Thêm command `avmatrix`
- [x] Đổi MCP canonical command sang `avmatrix mcp`
- [x] Đổi MCP server name trong config/setup sang `avmatrix`
- [x] Đổi docs/setup sang `AVmatrix`
- [x] Đổi storage primary sang `.avmatrix` / `~/.avmatrix`
- [x] Thêm migration một lần từ namespace cũ
- [x] Đổi resource scheme sang `avmatrix://`
- [x] Web UI active path không còn lộ `AVmatrix`
- [x] Behavioral tests cho command/MCP/storage/web pass
- [x] Typecheck pass cho `avmatrix` và `avmatrix-web`
- [ ] Rename triệt để package names, folder names, file names, import paths
- [ ] Dọn sạch comments, fixtures, test titles, snapshots, generated skill source names còn `avmatrix`
- [ ] Loại bỏ shim/alias `avmatrix` khỏi active path khi phase rename nội bộ hoàn tất
- [ ] Rà lần cuối để `avmatrix` chỉ còn trong lịch sử git, migration notes, hoặc external upstream references có chủ đích

# Kết luận

- Rename này là hợp lý và có giá trị thật vì nó giải quyết **nhầm lẫn ngữ cảnh làm việc** giữa local toolchain của bạn và upstream `AVmatrix`.
- Cách đúng không phải là find/replace toàn repo một phát.
- Cách đúng là:
  - đổi docs/spec trước
  - đổi surface người dùng và command/MCP namespace
  - migrate storage/config an toàn
  - khóa lại bằng behavioral tests
  - rồi rename triệt để package/folder/import/file/comment cho đến khi codebase active path sạch namespace cũ.
