# Mục đích plan này

  - Plan này dùng để chuyển GitNexus sang chế độ local-only theo đúng mong muốn ban đầu:
  - Giữ nguyên toàn bộ feature surface quan trọng của sản phẩm trên web UI, CLI, và MCP; không cắt tính năng chỉ để đơn giản hóa migration
  - Cái cần thay là đường đi auth/model execution: thay vì API key hoặc server/proxy bên thứ ba thì dùng session local đã đăng nhập của Codex hoặc Claude Code
  - Dùng tài khoản Codex hoặc Claude đã đăng nhập sẵn trên máy thay cho luồng API key và provider settings kiểu cloud
  - Chỉ cho phép phân tích repo từ đường dẫn local trên máy, không clone/pull từ GitHub hay URL bên ngoài
  - Không được đưa dữ liệu qua server của GitNexus hay proxy trung gian do GitNexus vận hành
  - Tool sau khi chuyển phải dùng hiệu quả ở cả giao diện web trong trình duyệt và ở CLI/MCP, vì cả hai đều quan trọng cho workflow agent viết code
  - Trình duyệt/web UI chỉ là lớp hiển thị local; mọi xử lý repo, tool, session, và orchestration phải chạy trong local runtime trên localhost
  - Xóa mọi gợi ý, copy, config, fallback, và dependency kéo tư duy API key/provider-setup để tránh đi nhầm luồng
  - Không cắt feature nào theo mặc định; riêng `wiki` sẽ được chuyển thành capability tùy chọn có thể bật/tắt, để hệ chạy ổn cả khi không có wiki và khi sau này có local wiki
  - Toàn bộ thay đổi sẽ được chia pha, có gate kỹ thuật và validation rõ ràng để triển khai rất cẩn thận, giảm rủi ro vỡ code

 # Ranh giới dữ liệu cần giữ

  - Được phép: dữ liệu đi từ máy local của người dùng lên OpenAI/Anthropic thông qua chính session local của Codex/Claude
  - Không được phép: dữ liệu repo đi qua server/proxy trung gian của GitNexus hoặc một backend cloud riêng được dựng thêm cho app này
  - Được phép: chạy process local trên máy người dùng như localhost bridge, daemon, CLI, hoặc native helper
  - Không được phép: biến kiến trúc thành mô hình “browser -> GitNexus-hosted backend -> model provider”

# Nguyên tắc sản phẩm sau khi chuyển

  - Local runtime là first-class product, không phải adapter tạm cho web
  - Web UI và CLI/MCP là hai client khác nhau của cùng một local runtime
  - Migration này là transport/auth replacement, không phải product reduction
  - Feature parity là yêu cầu cứng: search, graph, chat, multi-repo, MCP, CLI tools, và các capability hiện có phải còn dùng được sau migration, trừ khi có ngoại lệ được chốt rõ
  - Session model phải là session-based, không còn provider-based
  - Tool execution phải dùng chung runtime/tool contracts, không duplicate logic giữa web và CLI
  - Session state nếu cần giữ lâu phải nằm ở local runtime, không nằm riêng trong browser
  - Browser không được giữ quyền đặc biệt nào về auth, repo access, hay tool execution mà CLI không có
  - Các capability tùy chọn như `wiki` phải có feature flag rõ ràng; tắt capability không được làm vỡ các phần còn lại của sản phẩm

 # Khung làm việc

  - Áp dụng AGENTS.md + GUARDRAILS.md.
  - Scope dự kiến: gitnexus/, gitnexus-web/, gitnexus-shared/, docs/tests liên quan.
  - Bước này chỉ lên plan, chưa sửa code.
  - Trước khi edit từng pha, tôi sẽ chạy gitnexus_impact cho symbol chính bị đụng.
  - Validation khi triển khai:
      - cd gitnexus && npx tsc --noEmit && npm test
      - cd gitnexus-web && npx tsc -b --noEmit && npm test

 # Kiến trúc nên chốt

  - Không cố nhét luồng “login Codex/Claude account” vào browser.
  - Browser/web UI chỉ là local presentation layer; không giữ API key, không tự làm auth với model provider.
  - Kiến trúc mục tiêu là: web UI + CLI/MCP -> shared local runtime trên máy người dùng -> Codex/Claude local session + GitNexus local tools/index.
  - Shared local runtime ở đây là process local trên máy người dùng, có thể là daemon hoặc helper on-demand; không phải server cloud hay GitNexus-hosted backend.
  - Shared local runtime core phải tách khỏi HTTP adapter: runtime/core sống ở lớp riêng; `api.ts`, CLI, và MCP chỉ là các adapter/client vào cùng lõi này.
  - Web UI, direct CLI commands, và MCP surface phải hoặc gọi cùng runtime contracts, hoặc dùng cùng core modules; không duy trì hai hệ auth/session/tool riêng.
  - Hướng triển khai là compatibility-first: ưu tiên giữ nguyên contract UI/stream/tool hiện có, chỉ thay implementation phía sau bằng local session adapter.
  - Lý do: chat hiện tại đang là browser-side LangGraph + API-key providers trong /F:/GitNexus-main/
    gitnexus-web/src/core/llm/agent.ts:1, /F:/GitNexus-main/gitnexus-web/src/core/llm/types.ts:1, /F:/
    GitNexus-main/gitnexus-web/src/core/llm/settings-service.ts:1. Session account của Codex/Claude theo docs và thực tế công cụ
    kiểu desktop/CLI hiện nằm ở local app hoặc local CLI, không phải public browser auth flow cho web app này.
  - Tính đến ngày 2026-04-20, docs chính thức xác nhận Codex CLI hỗ trợ sign-in bằng ChatGPT account và
    có mode non-interactive codex exec; đồng thời Windows vẫn là experimental.
    Nguồn: https://developers.openai.com/codex/cli , https://developers.openai.com/codex/noninteractive ,
    https://help.openai.com/en/articles/11381614-codex-codex-andsign-in-with-chatgpt
  - Out of scope cho plan này:
  - Luồng API key cho OpenAI/Gemini/Anthropic/OpenRouter/Ollama
  - GitNexus-hosted backend hoặc proxy cloud
  - Browser-only auth flow cho Codex/Claude
  - Wiki chạy qua server bên thứ 3

 # Pha 0: Spike/Gate

  - Repo đụng: gitnexus.
  - Mục tiêu: chứng minh bridge khả thi trên máy bạn trước khi refactor UI.
  - Checklist:
  - Xác minh codex binary có sẵn và dùng được trên máy này.
  - Xác minh Codex CLI đã đăng nhập account của bạn.
  - Xác minh codex exec --json có stream/event shape đủ ổn định để map sang UI hiện tại.
  - Xác minh cancel được, timeout được, và chạy đúng theo cwd của repo local.
  - Xác minh Codex CLI có nối được GitNexus MCP local hay không.
  - Nếu MCP hookup tốt: dùng Codex CLI làm agent, GitNexus giữ vai trò tool server.
  - Nếu MCP hookup không ổn: fallback sang backend bridge gọi Codex CLI theo prompt orchestration hẹp
    hơn.
  - Go criteria: stream được partial output, cancel được, repo-local working dir đúng.
  - Deliverable bắt buộc sau spike:
  - Chốt duy nhất 1 nhánh kiến trúc trước khi vào Pha 1, không triển khai song song cả hai nhánh:
  - Track A: Codex/Claude CLI là full agent, localhost bridge chỉ làm relay + tool server
  - Track B: localhost bridge tự orchestration, Codex/Claude CLI chỉ làm model executor
  - Chốt process model của shared local runtime:
  - daemon sống lâu để web + CLI cùng attach
  - hoặc helper on-demand nhưng vẫn phải dùng chung contracts cho web + CLI
  - Chốt duy nhất 1 giao thức stream cho chat bridge để tránh tách đôi client/server:
  - Ưu tiên SSE nếu map tốt với UI hiện tại; nếu không thì chốt NDJSON và ghi rõ lý do
  - Ghi lại quyết định cho 4 điểm sau ngay trong plan hoặc ADR ngắn đi kèm:
  - Agent ownership nằm ở đâu
  - Tool execution nằm ở đâu
  - Stream protocol là gì
  - Repo binding được truyền theo repoName hay repoPath
  - CLI/MCP attach vào runtime kiểu nào
  - Rollout order của session adapters là gì: Codex-first rồi Claude Code sau, hay dual-adapter ngay từ v1
  - Ghi rõ feature parity strategy:
  - Những feature nào sẽ được giữ nguyên contract
  - Những feature nào cần compatibility shim tạm thời
  - Không được xóa surface nào trước khi local-session path đạt parity với surface đó
  - Quyết định mặc định của plan này, trừ khi spike bác bỏ:
  - V1 là Codex-first trên một session abstraction trung tính; Claude Code là adapter follow-up ngay sau khi runtime core ổn định
  - V1 ưu tiên Track A để tận dụng agent/runtime đã có của Codex/Claude CLI và giảm số lớp orchestration phải viết lại
  - CLI v1 giữ nguyên surface hiện có: direct tool commands, MCP, serve/runtime management; không mở thêm interactive chat command mới nếu không thật sự cần
  - Wiki remote bên thứ 3 bị đóng trong v1; thay vào đó tạo capability gate `wiki off / wiki local` để hệ chạy được cả khi không có wiki và khi sau này thêm local wiki
  - Nếu Codex CLI không ổn định trên Windows native, phải chốt ngay một trong hai hướng:
  - Dùng WSL2 bridge như execution environment chính
  - Hoặc dừng plan, không tự phát sinh workaround nửa vời trong các pha sau
  - Kết quả spike 2026-04-20:
  - Codex CLI 0.119.0 có sẵn, `codex login status` xác nhận đang đăng nhập bằng ChatGPT account
  - `codex exec --json` hoạt động và cho JSONL event stream đủ giàu để bridge lên web UI
  - Trên Windows native, shell execution trong sandbox mặc định fail với lỗi `CreateProcessAsUserW failed: 5`; chế độ bypass sandbox chạy được và tôn trọng `cwd`
  - WSL2 có sẵn trên máy; vì vậy quyết định cho Windows là ưu tiên WSL2 bridge cho full agent mode thay vì dựa vào Windows native sandbox path
  - MCP hookup với Codex được xác nhận ở mức config + resource access; cần giữ một validation task riêng cho tool-call path khi có repo đã index
  - Pha 0 chốt các quyết định v1 sau:
  - Track A
  - Giao thức stream cho chat bridge: SSE
  - Repo local hợp lệ nhưng chưa index => explicit analyze gate, không auto-analyze ngầm trong chat path
  - `serve` và `mcp` host cùng runtime core theo kiểu in-process; v1 không dựng daemon riêng để attach chéo process
  - `wikiMode` nằm trong global runtime config file và runtime là source of truth

 # Pha 1: Shared Local Runtime + Session Bridge

  - Repo đụng: gitnexus, gitnexus-shared.
  - Mục tiêu:
  - Tạo shared local runtime làm lõi dùng chung cho web UI, CLI, và MCP surface
  - Tách session bridge ra khỏi web/browser để auth và tool execution nằm hoàn toàn local
  - Tạo abstraction đủ trung tính để hỗ trợ cả Codex và Claude Code session adapters, không hard-code toàn bộ runtime vào một vendor duy nhất
  - Tạo mới:
  - gitnexus/src/runtime/runtime-controller.ts hoặc tên tương đương để quản lý lifecycle của local runtime
  - gitnexus/src/runtime/session-adapter.ts hoặc tên tương đương cho contract session chung
  - gitnexus/src/server/session-bridge.ts hoặc tên tương đương làm HTTP/localhost adapter chung
  - gitnexus/src/runtime/session-adapters/codex.ts hoặc tên tương đương cho Codex adapter đầu tiên
  - gitnexus/src/runtime/session-jobs/session-job.ts hoặc tên tương đương cho lifecycle/cancel/stream
  - Có thể thêm gitnexus/src/runtime/session-adapters/claude-code.ts sau khi core ổn định
  - shared types trong gitnexus-shared cho SessionStatus, SessionChatRequest, SessionStreamEvent
  - Sửa:
  - /F:/GitNexus-main/gitnexus/src/server/api.ts:1 để thêm local bridge endpoints trung tính như /api/session/status
  - /F:/GitNexus-main/gitnexus/src/server/api.ts:1 để thêm /api/session/chat stream SSE
  - /F:/GitNexus-main/gitnexus/src/server/api.ts:1 để thêm cancel endpoint cho chat session
  - V1 implementation đi qua Codex adapter trước, nhưng route/type/core naming giữ trung tính để không phải rename lớn khi thêm Claude Code
  - Contract tối thiểu cần chốt trước khi code:
  - Request chat phải mang repo binding rõ ràng: repoName hoặc repoPath; không ngầm dùng repo cuối cùng trên server
  - Response stream phải map được sang step/content/tool-call shape hiện tại hoặc có adapter rõ ràng ở web
  - Cancel phải có session identifier rõ ràng; switch repo phải abort chat đang chạy
  - Web UI và CLI/MCP nếu đụng cùng một session hoặc repo phải có semantics rõ ràng: attach, isolate, hoặc steal session
  - Repo local hợp lệ nhưng chưa index phải trả về trạng thái rõ ràng kiểu `INDEX_REQUIRED`; UI/CLI hiển thị CTA `Analyze now`, không tự chạy analyze ngầm từ chat request
  - Quy tắc bridge:
  - Chỉ cho phép chạy trong repo local đã index hoặc repo path hợp lệ
  - Không giới hạn vào workspace của GitNexus; được phép dùng repo local bất kỳ do người dùng chỉ định, miễn qua local-only path policy
  - Log stderr nội bộ, trả lỗi an toàn kiểu “Session runtime not installed” hoặc “Session not signed in”; adapter có thể chi tiết hóa thành Codex/Claude Code
  - Ưu tiên map event stream về gần shape AgentStreamChunk hiện có để giảm sửa UI
  - Bridge local không được forward dữ liệu qua bất kỳ GitNexus-hosted service nào
  - V1 process model:
  - `gitnexus serve` host runtime core trong chính process HTTP server
  - `gitnexus mcp` host runtime core trong chính process stdio MCP server
  - direct CLI commands reuse cùng runtime/core modules theo kiểu in-process hoặc ephemeral, không attach vào một daemon riêng
  - Chính sách local-only cho path ở backend:
  - Reject hoàn toàn mọi URL/git URL
  - Reject UNC/network share paths kiểu \\server\share\repo
  - Normalize + resolve + fs.realpath trước khi dùng
  - Chỉ chấp nhận thư mục local tồn tại thật trên máy
  - Nếu path resolve sang network mount hoặc outside policy thì reject sớm với lỗi rõ ràng
  - Test cần thêm/sửa:
  - gitnexus/test/unit/session-bridge.test.ts
  - test API contract mới cho /api/session/status và /api/session/chat
  - codex-adapter tests cho session lifecycle đầu tiên
  - test local-path policy: absolute path, traversal, UNC path, missing folder, repo binding mismatch

 # Pha 2: Di chuyển web UI sang shared local runtime

  - Repo đụng: gitnexus-web.
  - Khuyến nghị an toàn: tạo file mới trước, không đập ngay agent.ts.
  - Tạo mới:
  - gitnexus-web/src/core/llm/session-client.ts
  - Có thể thêm gitnexus-web/src/core/llm/session-types.ts nếu chưa đưa vào gitnexus-shared
  - Sửa:
  - /F:/GitNexus-main/gitnexus-web/src/hooks/useAppState.tsx:1 để dùng backend session stream thay cho
    createGraphRAGAgent()
  - /F:/GitNexus-main/gitnexus-web/src/core/llm/types.ts:1 để chuyển từ provider model sang session/backends model, hoặc thêm compatibility shim có thời hạn rõ ràng nếu cần giữ app ổn trong giai đoạn chuyển đổi
  - /F:/GitNexus-main/gitnexus-web/src/core/llm/settings-service.ts:1 để migrate settings cũ và bỏ logic
    API-key/provider setup, nhưng vẫn giữ được các capability UI vốn phụ thuộc vào settings
  - /F:/GitNexus-main/gitnexus-web/src/components/SettingsPanel.tsx:1 để chuyển từ provider form sang local session management UI như “Codex Account” / “Claude Code”
  - /F:/GitNexus-main/gitnexus-web/src/components/RightPanel.tsx:1 để bỏ message “Configure an LLM provider”
  - /F:/GitNexus-main/gitnexus-web/src/hooks/useAppState.tsx:1 để đổi error/init flow từ “provider” sang “local session bridge”
  - /F:/GitNexus-main/gitnexus-web/src/components/OnboardingGuide.tsx:1 để bỏ gợi ý npx gitnexus@latest serve
  - /F:/GitNexus-main/gitnexus-web/src/components/HelpPanel.tsx:1 để bỏ remote/provider copy không còn đúng
  - /F:/GitNexus-main/gitnexus-web/src/config/ui-constants.ts:1 để dọn constants provider cũ nếu không còn dùng
  - Giữ nguyên UI chat message/step nếu stream mới map được sang AgentStreamChunk
  - UX mục tiêu:
  - Không còn ô API key
  - V1 chỉ cần bật đầy đủ luồng Codex: Codex found / not found, Signed in / not signed in
  - Chừa chỗ trong UI/state cho Claude Code adapter follow-up mà không phải đổi model dữ liệu
  - Có nút Check connection
  - Nếu chat nhận `INDEX_REQUIRED`, hiển thị CTA `Analyze now` hoặc điều hướng thẳng sang flow analyze local path; không tự khởi chạy analyze âm thầm
  - Ở v1, CTA `Analyze now` sẽ reuse `RepoAnalyzer` sheet hiện có thay vì tạo flow phân tích mới riêng cho chat panel
  - Migration strategy cần chốt:
  - Nếu sessionStorage/localStorage đang chứa provider cũ, hoặc migrate sang local session bridge mode, hoặc clear one-shot có kiểm soát
  - Nếu clear, phải có fallback UX rõ ràng thay vì app boot vào trạng thái lỗi mơ hồ
  - Khi switch repo, chat mới phải bind đúng repo đang active; chat cũ phải bị hủy hoặc tách session rõ ràng
  - Nếu có feature nào đang dựa nặng vào provider-specific config, phải chỉ ra compatibility shim tương ứng thay vì xóa feature đó khỏi web
  - Với các surface UI/state nhiều trạng thái và nhiều tính năng như `SettingsPanel`, `useAppState`, và các panel onboarding/help:
  - Ưu tiên tạo file companion mới song song trước, đạt parity giao diện + tính năng với file cũ
  - Chỉ thay file cũ sau khi file mới đã đạt parity và có behavioral tests đi kèm
  - Không delete/rewrite rút gọn file cũ giữa chừng trong lúc parity chưa đạt
  - Test cần thêm/sửa:
  - gitnexus-web/test/unit/settings-service.test.ts
  - test stream client/backend-client mới cho session bridge
  - test init/send/cancel chat flow nếu cần
  - test settings migration/reset từ dữ liệu provider cũ

 # Pha 2B: Căn CLI/MCP vào shared local runtime

  - Repo đụng: gitnexus.
  - Mục tiêu:
  - Đảm bảo CLI và MCP vẫn là surface quan trọng cho agent coding, nhưng dùng cùng mô hình runtime/session local như web
  - Tránh việc web chạy một kiểu, CLI chạy một kiểu khác
  - Sửa:
  - /F:/GitNexus-main/gitnexus/src/cli/index.ts:1 để thêm hoặc chỉnh help/copy theo shared local runtime model
  - /F:/GitNexus-main/gitnexus/src/cli/mcp.ts:1 để chốt cách MCP attach vào runtime hoặc dùng cùng core modules
  - /F:/GitNexus-main/gitnexus/src/cli/tool.ts:1 để chốt cách direct tool commands reuse runtime/core contracts
  - /F:/GitNexus-main/gitnexus/src/cli/setup.ts:1 để wording không còn gợi ý provider/API-key flow
  - Quy tắc:
  - Không tạo một session/auth stack riêng cho CLI
  - Không tạo một tool execution path riêng cho web nếu CLI/MCP có thể dùng cùng core/runtime
  - Giữ nguyên toàn bộ direct tool commands và MCP capability hiện có; chỉ thay session/model execution path bên dưới khi có đụng tới LLM-backed capability
  - V1 không thêm interactive chat/session command mới cho CLI trừ khi spike chứng minh đó là bắt buộc để đạt parity
  - Nếu cần thêm runtime management commands, chỉ thêm những lệnh local như status/doctor/restart
  - `serve` và `mcp` không attach vào một daemon runtime riêng ở v1; mỗi surface host runtime core của chính nó và chia sẻ contract/module thay vì chia sẻ OS process
  - Test cần thêm/sửa:
  - gitnexus/test/unit/setup-session-runtime.test.ts
  - gitnexus/test/unit/tools.test.ts
  - smoke tests cho MCP/runtime attach path nếu thay đổi contract

 # Pha 3: Khóa analyze sang local path only

  - Repo đụng: gitnexus, gitnexus-web.
  - Sửa backend:
  - /F:/GitNexus-main/gitnexus/src/server/api.ts:1145 chỉ nhận path, reject url
  - /F:/GitNexus-main/gitnexus/src/server/analyze-job.ts:1 bỏ repoUrl, dedupe theo repoPath
  - Gỡ runtime use của /F:/GitNexus-main/gitnexus/src/server/git-clone.ts:1
  - Chốt path policy ở API/analyze:
  - Chỉ nhận absolute local paths
  - Reject UNC/network share
  - Resolve realpath trước khi enqueue job
  - Chỉ dedupe theo canonical repoPath sau normalize/realpath
  - Sửa web:
  - /F:/GitNexus-main/gitnexus-web/src/components/RepoAnalyzer.tsx:1 bỏ mode GitHub, bỏ URL validation,
    chỉ còn local path
  - /F:/GitNexus-main/gitnexus-web/src/components/AnalyzeOnboarding.tsx:1 sửa copy local-only
  - RepoLanding.tsx, Header.tsx, backend-client.ts cập nhật theo contract mới
  - Test cần thêm/sửa:
  - gitnexus/test/unit/analyze-api.test.ts
  - gitnexus/test/unit/analyze-job.test.ts

 # Pha 4: Hardening local-only

  - Repo đụng: gitnexus, gitnexus-web.
  - Sửa:
  - /F:/GitNexus-main/gitnexus/src/cli/setup.ts:1 bỏ fallback npx -y gitnexus@latest
  - /F:/GitNexus-main/gitnexus/src/server/api.ts:1 siết CORS còn localhost, 127.0.0.1, ::1, và no-origin
  - gitnexus-web/src/services/backend-client.ts và useBackend chỉ chấp nhận backend local như mặc định bắt buộc của chế độ local-only
  - Backend URL cũ trong localStorage nếu trỏ remote host phải bị reset an toàn về localhost mặc định, không được âm thầm tiếp tục dùng remote endpoint
  - Auto-connect bằng `?project=` phải mặc định về local backend URL chuẩn, không dùng `window.location.origin` kiểu hosted UI cũ
  - /F:/GitNexus-main/gitnexus/src/cli/serve.ts:1 update help/comment/copy để không còn assumption về hosted frontend
  - Xóa mọi copy “remote provider”, “API key”, “hosted UI”, “GitHub URL”, “cloud” trong onboarding/settings/help
  - Acceptance:
  - UI không còn gợi ý remote
  - Setup/runtime không còn tự kéo remote package
  - Không còn đường dữ liệu nào đi qua GitNexus-hosted/proxy backend
  - Backend local không còn mở cho Vercel/LAN nếu bạn muốn khóa tuyệt đối

# Pha 5: Capability gate cho wiki

  - Repo đụng: gitnexus, gitnexus-web.
  - Đóng hoàn toàn đường đi wiki qua server bên thứ 3.
  - Thêm cơ chế bật/tắt capability wiki, ví dụ `wikiMode: off | local` hoặc feature flag tương đương.
  - `wikiMode` được lưu trong global runtime config file, ví dụ `~/.gitnexus/runtime.json`; runtime là source of truth, không dùng localStorage làm nguồn chuẩn
  - Yêu cầu với `wiki off`:
  - Web UI, CLI, MCP, analyze, graph, chat vẫn hoạt động bình thường mà không cần wiki
  - Các menu/command/help text liên quan wiki phải ẩn hoặc disable có kiểm soát, không để dead path gọi nhầm
  - Không còn bất kỳ call nào ra remote wiki backend
  - Yêu cầu với `wiki local`:
  - Contract, routing, và settings phải chừa sẵn để sau này cắm local wiki mà không bẻ lại runtime core
  - Khi local wiki chưa được triển khai xong, mode này phải fail-safe với thông báo rõ ràng kiểu “local wiki chưa khả dụng”, không được fallback sang server bên thứ 3
  - Trong milestone hiện tại:
  - Hoàn thành capability gate và remote shutdown
  - Chưa cần triển khai local wiki engine đầy đủ; phần đó là follow-up sau khi tool core ổn định

# Pha 6: Dọn provider/API-key path cũ sau khi bridge đạt parity

  - Repo đụng: gitnexus-web.
  - Chỉ dọn sau khi local-session path đã đạt feature parity cho các surface tương ứng.
  - Sau khi local session bridge ổn định end-to-end:
  - Xóa nhánh openai, azure-openai, gemini, anthropic, ollama, openrouter, minimax, glm nếu và chỉ nếu không còn feature nào phụ thuộc chúng
  - Dọn helper OpenRouter/Ollama trong /F:/GitNexus-main/gitnexus-web/src/components/SettingsPanel.tsx:1
  - Dọn builder cũ trong /F:/GitNexus-main/gitnexus-web/src/core/llm/settings-service.ts:1
  - Dọn model branches cũ trong /F:/GitNexus-main/gitnexus-web/src/core/llm/agent.ts:1 nếu file này còn
    tồn tại
  - Dọn dependency packages chỉ còn phục vụ provider/API-key flow trong gitnexus-web/package.json và lockfile
  - Dọn language/copy “provider-based” còn sót trong cả web và CLI surfaces
  - Cập nhật toàn bộ test còn hard-code gemini hoặc provider cũ

# Checklist hoàn thành

  - Chat chạy bằng tài khoản Codex đã login, không cần API key
  - Hoặc Claude Code local session nếu user chọn flow đó
  - Feature surface quan trọng trên web UI, CLI, và MCP vẫn còn dùng được; không có regression do cắt feature
  - UI không còn picker provider remote
  - UI không còn GitHub URL input
  - Backend không clone/pull từ URL
  - Backend chỉ nhận absolute local paths
  - setup không còn kéo gitnexus@latest
  - CORS/backend flow không còn remote suggestion
  - Track A hoặc Track B đã được chốt sau spike, không giữ hai hướng song song
  - V1 rollout được chốt là Codex-first trên session abstraction trung tính; Claude Code follow-up không đòi rename lớn
  - Chat request luôn bind đúng repo đang active
  - UNC/network-share path bị reject ở analyze và chat bridge
  - settings cũ được migrate hoặc reset có kiểm soát, không để app boot lỗi vì provider state cũ
  - Browser chỉ làm local UI; không có auth flow model-provider chạy trong browser
  - Không còn đường dữ liệu nào đi qua GitNexus/proxy server trung gian
  - Web UI và CLI/MCP cùng dựa trên shared local runtime, không còn hai luồng session/tool khác nhau
  - Cancel chat và cancel analyze vẫn hoạt động
  - Graph browsing/search/query hiện có không bị regress
  - Remote wiki path đã bị tắt hoàn toàn
  - Hệ chạy ổn khi `wiki off`
  - Contract cho `wiki local` đã được chừa sẵn để phát triển tiếp mà không phải bẻ lại lõi

 # Test matrix cần chạy khi triển khai

  - Nguyên tắc kiểm thử theo phase:
  - Mỗi phase phải có behavioral tests mới bám đúng contract của phase đó; không chỉ dựa vào test legacy của kiến trúc provider/API-key cũ
  - Typecheck toàn package vẫn là kiểm tra tích hợp bắt buộc tối thiểu
  - Test legacy chỉ được dùng lại khi còn phản ánh đúng hành vi sản phẩm sau migration; test nào khóa vào provider/cloud path cũ không được coi là blocker cho phase mới
  - Ưu tiên thứ tự: behavioral tests của phase hiện tại -> targeted integration tests của phase hiện tại -> typecheck package liên quan -> legacy regression tests còn phù hợp
  - Không sang phase tiếp theo khi phase hiện tại chưa có test hành vi riêng xác nhận contract mới
  - gitnexus:
  - npx tsc --noEmit
  - npm test
  - unit trọng điểm: analyze-api.test.ts, analyze-job.test.ts, session-bridge.test.ts, cli-index-help.test.ts, setup-session-runtime.test.ts, tools.test.ts
  - integration/e2e trọng điểm: cli-e2e.test.ts cho help/runtime surface bị ảnh hưởng bởi local-session migration và capability gate của wiki
  - gitnexus-web:
  - npx tsc -b --noEmit
  - npm test
  - unit trọng điểm: settings-service.test.ts, server-connection.test.ts, test mới cho session stream client, test mới cho RepoAnalyzer local-only, test mới cho wiki off/on capability behavior

 # Inventory file dự kiến

  - gitnexus: src/server/api.ts, src/server/analyze-job.ts, src/server/git-clone.ts, src/server/
    session-bridge.ts mới, src/runtime/* mới nếu có, src/cli/index.ts, src/cli/mcp.ts, src/cli/tool.ts,
    src/cli/setup.ts, src/cli/serve.ts, src/cli/ai-context.ts, src/cli/wiki.ts, src/core/wiki/*,
    src/storage/repo-manager.ts, tests liên quan
  - gitnexus-web: src/core/llm/types.ts, src/core/llm/settings-service.ts, src/core/llm/agent.ts hoặc
    src/core/llm/session-client.ts mới, src/core/llm/tools.ts, src/core/llm/context-builder.ts, src/core/llm/index.ts,
    src/hooks/useAppState.tsx, src/components/SettingsPanel.tsx, src/components/RepoAnalyzer.tsx,
    src/components/AnalyzeOnboarding.tsx, src/components/OnboardingGuide.tsx, src/components/Header.tsx,
    src/components/RepoLanding.tsx, src/components/RightPanel.tsx, src/components/HelpPanel.tsx,
    src/services/backend-client.ts, src/config/ui-constants.ts, package.json, package-lock.json, vercel.json,
    tests liên quan
  - gitnexus-shared: shared request/response/event types nếu chuẩn hóa contract giữa backend và web
  - runtime config: `~/.gitnexus/runtime.json` hoặc helper tương đương để giữ mode/capability chung như `wikiMode`
  - docs/tests: README/CONTRIBUTING/help/unit tests bị ảnh hưởng

# Checklist theo dõi triển khai

## Pha 0 — Spike và quyết định

- [x] Xác minh `codex` CLI có sẵn trên máy
- [x] Xác minh `codex login status` dùng ChatGPT account
- [x] Xác minh `codex exec --json` cho JSONL event stream usable
- [x] Xác minh `cwd` binding hoạt động khi Codex chạy command thành công
- [x] Xác minh MCP config/resource hookup với GitNexus local
- [x] Chốt `Codex-first`
- [x] Chốt `Track A`
- [x] Chốt stream protocol là `SSE`
- [x] Chốt `repo chưa index => INDEX_REQUIRED + Analyze now`
- [x] Chốt `serve`/`mcp` host runtime core theo kiểu in-process ở v1
- [x] Chốt `wikiMode` nằm trong global runtime config
- [x] Chốt Windows recommendation là `WSL2 bridge` cho full agent mode

## Pha 1 — Shared runtime và session bridge

- [x] Tạo `runtime-controller`
- [x] Tạo `session-adapter` abstraction
- [x] Tạo `session-bridge` HTTP adapter
- [x] Tạo Codex adapter đầu tiên
- [x] Thêm `/api/session/status`
- [x] Thêm `/api/session/chat` qua `SSE`
- [x] Thêm cancel/session lifecycle
- [x] Trả `INDEX_REQUIRED` cho repo local chưa index
- [x] Chốt semantics khi web/CLI/MCP đụng cùng session
- [x] Viết behavioral tests cho `runtime-controller` và `session-bridge`

## Pha 2 — Web UI migrate sang session runtime

- [x] Tạo `session-client` cho web
- [x] Đổi `useAppState` sang session stream
- [x] Chuyển settings từ provider-based sang session-based
- [x] Đổi `SettingsPanel` sang local session management
- [x] Xử lý CTA `Analyze now` khi nhận `INDEX_REQUIRED`
- [x] Giữ nguyên chat steps/content UI
- [x] Migrate hoặc reset settings cũ an toàn
- [x] Viết behavioral tests cho web session flow

## Pha 2B — CLI/MCP alignment

- [x] Cập nhật help/copy của CLI theo shared runtime model
- [ ] Cho `mcp` reuse cùng runtime/core contracts
- [ ] Cho direct tool commands reuse cùng runtime/core contracts
- [ ] Không thêm interactive chat command vào CLI v1 nếu không bắt buộc
- [ ] Thêm runtime management commands tối thiểu nếu cần
- [ ] Viết behavioral tests cho CLI/MCP runtime alignment

## Pha 3 — Local path only

- [x] API analyze chỉ nhận local path
- [x] Bỏ `repoUrl` khỏi analyze flow
- [x] Gỡ clone/pull từ URL
- [x] Canonicalize path bằng `realpath`
- [x] Reject UNC/network share
- [x] Web chỉ còn local path analyze flow
- [x] Viết behavioral tests cho local path policy

## Pha 4 — Hardening local-only

- [x] Bỏ fallback `npx -y gitnexus@latest`
- [x] Siết CORS còn localhost/no-origin
- [x] Backend client chỉ chấp nhận local backend
- [ ] Bỏ toàn bộ wording remote/API key/cloud
- [x] Cập nhật `serve` help/copy theo local-only model
- [x] Viết behavioral tests cho hardening/local-only surface

## Pha 5 — Wiki capability gate

- [x] Tắt hoàn toàn remote wiki path
- [x] Thêm capability gate `wiki off | local`
- [x] Hệ chạy ổn khi `wiki off`
- [x] UI/CLI ẩn hoặc disable wiki dead paths đúng cách
- [x] Chừa contract/runtime/settings cho `wiki local`
- [x] `wiki local` fail-safe, không fallback remote
- [x] Viết behavioral tests cho wiki capability gate

## Pha 6 — Cleanup provider/API-key path cũ

- [ ] Chỉ cleanup sau khi session path đạt parity
- [ ] Xóa provider branches cũ không còn dùng
- [ ] Dọn dependency packages chỉ còn phục vụ provider/API-key flow
- [ ] Dọn copy/provider language còn sót
- [ ] Cập nhật test theo runtime/session model mới
- [ ] Dọn hoặc retire legacy tests không còn phản ánh kiến trúc mới
