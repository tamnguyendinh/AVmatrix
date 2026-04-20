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
  - Không cắt feature nào theo mặc định; riêng các surface gây tranh cãi như wiki sẽ được chốt rõ ở pha scope tương ứng thay vì tự động loại bỏ
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
  - Nếu Codex CLI không ổn định trên Windows native, phải chốt ngay một trong hai hướng:
  - Dùng WSL2 bridge như execution environment chính
  - Hoặc dừng plan, không tự phát sinh workaround nửa vời trong các pha sau

 # Pha 1: Shared Local Runtime + Session Bridge

  - Repo đụng: gitnexus, gitnexus-shared.
  - Mục tiêu:
  - Tạo shared local runtime làm lõi dùng chung cho web UI, CLI, và MCP surface
  - Tách session bridge ra khỏi web/browser để auth và tool execution nằm hoàn toàn local
  - Tạo abstraction đủ trung tính để hỗ trợ cả Codex và Claude Code session adapters, không hard-code toàn bộ runtime vào một vendor duy nhất
  - Tạo mới:
  - gitnexus/src/runtime/runtime-controller.ts hoặc tên tương đương để quản lý lifecycle của local runtime
  - gitnexus/src/runtime/session-adapter.ts hoặc tên tương đương cho contract session chung
  - gitnexus/src/server/codex-bridge.ts
  - gitnexus/src/server/codex-session.ts hoặc codex-job.ts
  - Có thể thêm gitnexus/src/server/claude-code-bridge.ts hoặc claude-code-session.ts nếu cần adapter riêng
  - shared types trong gitnexus-shared cho SessionStatus, SessionChatRequest, SessionStreamEvent
  - Sửa:
  - /F:/GitNexus-main/gitnexus/src/server/api.ts:1 để thêm local bridge endpoints như /api/codex/status
  - /F:/GitNexus-main/gitnexus/src/server/api.ts:1 để thêm /api/codex/chat stream SSE hoặc NDJSON
  - /F:/GitNexus-main/gitnexus/src/server/api.ts:1 để thêm cancel endpoint cho chat session
  - Contract tối thiểu cần chốt trước khi code:
  - Request chat phải mang repo binding rõ ràng: repoName hoặc repoPath; không ngầm dùng repo cuối cùng trên server
  - Response stream phải map được sang step/content/tool-call shape hiện tại hoặc có adapter rõ ràng ở web
  - Cancel phải có session identifier rõ ràng; switch repo phải abort chat đang chạy
  - Web UI và CLI/MCP nếu đụng cùng một session hoặc repo phải có semantics rõ ràng: attach, isolate, hoặc steal session
  - Quy tắc bridge:
  - Chỉ cho phép chạy trong repo local đã index hoặc repo path hợp lệ
  - Không giới hạn vào workspace của GitNexus; được phép dùng repo local bất kỳ do người dùng chỉ định, miễn qua local-only path policy
  - Log stderr nội bộ, trả lỗi an toàn kiểu “Codex not installed” hoặc “Codex not signed in”
  - Ưu tiên map event stream về gần shape AgentStreamChunk hiện có để giảm sửa UI
  - Bridge local không được forward dữ liệu qua bất kỳ GitNexus-hosted service nào
  - Chính sách local-only cho path ở backend:
  - Reject hoàn toàn mọi URL/git URL
  - Reject UNC/network share paths kiểu \\server\share\repo
  - Normalize + resolve + fs.realpath trước khi dùng
  - Chỉ chấp nhận thư mục local tồn tại thật trên máy
  - Nếu path resolve sang network mount hoặc outside policy thì reject sớm với lỗi rõ ràng
  - Test cần thêm/sửa:
  - gitnexus/test/unit/codex-bridge.test.ts
  - test API contract mới cho /api/codex/status và /api/codex/chat
  - test local-path policy: absolute path, traversal, UNC path, missing folder, repo binding mismatch

 # Pha 2: Di chuyển web UI sang shared local runtime

  - Repo đụng: gitnexus-web.
  - Khuyến nghị an toàn: tạo file mới trước, không đập ngay agent.ts.
  - Tạo mới:
  - gitnexus-web/src/core/llm/codex-client.ts
  - Có thể thêm gitnexus-web/src/core/llm/codex-types.ts nếu chưa đưa vào gitnexus-shared
  - Sửa:
  - /F:/GitNexus-main/gitnexus-web/src/hooks/useAppState.tsx:1 để dùng backend Codex stream thay cho
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
  - Chỉ còn trạng thái Codex found / not found
  - Chỉ còn trạng thái Signed in / not signed in
  - Có nút Check connection
  - Migration strategy cần chốt:
  - Nếu sessionStorage/localStorage đang chứa provider cũ, hoặc migrate sang local session bridge mode, hoặc clear one-shot có kiểm soát
  - Nếu clear, phải có fallback UX rõ ràng thay vì app boot vào trạng thái lỗi mơ hồ
  - Khi switch repo, chat mới phải bind đúng repo đang active; chat cũ phải bị hủy hoặc tách session rõ ràng
  - Nếu có feature nào đang dựa nặng vào provider-specific config, phải chỉ ra compatibility shim tương ứng thay vì xóa feature đó khỏi web
  - Test cần thêm/sửa:
  - gitnexus-web/test/unit/settings-service.test.ts
  - test stream client/backend-client mới cho Codex
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
  - Nếu cần thêm runtime management commands, chỉ thêm những lệnh local như status/doctor/restart
  - Test cần thêm/sửa:
  - gitnexus/test/unit/setup-codex.test.ts
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
  - /F:/GitNexus-main/gitnexus/src/cli/serve.ts:1 update help/comment/copy để không còn assumption về hosted frontend
  - Xóa mọi copy “remote provider”, “API key”, “hosted UI”, “GitHub URL”, “cloud” trong onboarding/settings/help
  - Acceptance:
  - UI không còn gợi ý remote
  - Setup/runtime không còn tự kéo remote package
  - Không còn đường dữ liệu nào đi qua GitNexus-hosted/proxy backend
  - Backend local không còn mở cho Vercel/LAN nếu bạn muốn khóa tuyệt đối

# Pha 5: Xử lý wiki theo quyết định scope cuối cùng

  - Repo đụng: gitnexus.
  - Theo nguyên tắc feature parity, mặc định không xóa wiki chỉ để đơn giản hóa migration.
  - Có 2 khả năng, phải chốt rõ trước khi code:
  - Giữ wiki: chuyển mọi chỗ wiki đang dùng provider/API key sang local session path giống phần chat
  - Hoặc coi wiki là ngoại lệ bị loại khỏi scope do user xác nhận rõ
  - Nếu giữ wiki:
  - Giữ nguyên wiki command surface trong /F:/GitNexus-main/gitnexus/src/cli/index.ts:1
  - Chuyển config/help/runtime của wiki sang session-based flow
  - Cập nhật docs/tests tương ứng
  - Nếu loại wiki khỏi scope:
  - Chỉ hard-disable sau khi user xác nhận lại ngoại lệ này
  - Khi đó mới xóa src/cli/wiki.ts, src/core/wiki/*, config liên quan, và tests liên quan

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
  - Chat request luôn bind đúng repo đang active
  - UNC/network-share path bị reject ở analyze và chat bridge
  - settings cũ được migrate hoặc reset có kiểm soát, không để app boot lỗi vì provider state cũ
  - Browser chỉ làm local UI; không có auth flow model-provider chạy trong browser
  - Không còn đường dữ liệu nào đi qua GitNexus/proxy server trung gian
  - Web UI và CLI/MCP cùng dựa trên shared local runtime, không còn hai luồng session/tool khác nhau
  - Cancel chat và cancel analyze vẫn hoạt động
  - Graph browsing/search/query hiện có không bị regress
  - Nếu wiki còn trong scope thì wiki cũng phải chuyển xong sang local-session path; nếu không thì phải có xác nhận ngoại lệ rõ ràng

 # Test matrix cần chạy khi triển khai

  - gitnexus:
  - npx tsc --noEmit
  - npm test
  - unit trọng điểm: analyze-api.test.ts, analyze-job.test.ts, codex-bridge.test.ts, cli-index-help.test.ts, setup-codex.test.ts, tools.test.ts
  - integration/e2e trọng điểm: cli-e2e.test.ts cho help/runtime surface bị ảnh hưởng bởi local-session migration và quyết định scope wiki
  - gitnexus-web:
  - npx tsc -b --noEmit
  - npm test
  - unit trọng điểm: settings-service.test.ts, server-connection.test.ts, test mới cho codex stream client, test mới cho RepoAnalyzer local-only

 # Inventory file dự kiến

  - gitnexus: src/server/api.ts, src/server/analyze-job.ts, src/server/git-clone.ts, src/server/
    *codex*.ts mới, src/runtime/* mới nếu có, src/cli/index.ts, src/cli/mcp.ts, src/cli/tool.ts,
    src/cli/setup.ts, src/cli/serve.ts, src/cli/ai-context.ts, src/storage/repo-manager.ts, tests liên quan
  - gitnexus-web: src/core/llm/types.ts, src/core/llm/settings-service.ts, src/core/llm/agent.ts hoặc
    src/core/llm/codex-client.ts mới, src/core/llm/tools.ts, src/core/llm/context-builder.ts, src/core/llm/index.ts,
    src/hooks/useAppState.tsx, src/components/SettingsPanel.tsx, src/components/RepoAnalyzer.tsx,
    src/components/AnalyzeOnboarding.tsx, src/components/OnboardingGuide.tsx, src/components/Header.tsx,
    src/components/RepoLanding.tsx, src/components/RightPanel.tsx, src/components/HelpPanel.tsx,
    src/services/backend-client.ts, src/config/ui-constants.ts, package.json, package-lock.json, vercel.json,
    tests liên quan
  - gitnexus-shared: shared request/response/event types nếu chuẩn hóa contract giữa backend và web
  - docs/tests: README/CONTRIBUTING/help/unit tests bị ảnh hưởng
