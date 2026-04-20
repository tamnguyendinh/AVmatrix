 # Mục đích plan này

  - Plan này dùng để chuyển GitNexus sang chế độ local-only theo đúng mong muốn ban đầu:
  - Dùng tài khoản Codex đã đăng nhập trên máy thay cho luồng API key và các provider remote hiện có
  - Chỉ cho phép phân tích repo từ đường dẫn local trên máy, không clone/pull từ GitHub hay URL bên ngoài
  - Xóa mọi gợi ý, copy, config, và fallback mang tính remote để tránh đi nhầm luồng
  - Tắt wiki hiện tại để giảm surface không cần thiết; sau này có thể làm một wiki local riêng
  - Toàn bộ thay đổi sẽ được chia pha, có gate kỹ thuật và validation rõ ràng để triển khai rất cẩn thận, giảm rủi ro vỡ code

 # Khung làm việc

  - Áp dụng AGENTS.md + GUARDRAILS.md.
  - Scope dự kiến: gitnexus/, gitnexus-web/, gitnexus-shared/, docs/tests liên quan.
  - Bước này chỉ lên plan, chưa sửa code.
  - Trước khi edit từng pha, tôi sẽ chạy gitnexus_impact cho symbol chính bị đụng.
  - Validation khi triển khai:
      - cd gitnexus && npx tsc --noEmit && npm test
      - cd gitnexus-web && npx tsc -b --noEmit && npm test

 # Kiến trúc nên chốt

  - Không cố nhét luồng “login Codex account” vào browser.
  - Hướng ít phá vỡ nhất là: gitnexus-web -> gitnexus serve -> Codex CLI local đã đăng nhập -> GitNexus
    tools/index local.
  - Lý do: chat hiện tại đang là browser-side LangGraph + API-key providers trong /F:/GitNexus-main/
    gitnexus-web/src/core/llm/agent.ts:1, /F:/GitNexus-main/gitnexus-web/src/core/llm/types.ts:1, /F:/
    GitNexus-main/gitnexus-web/src/core/llm/settings-service.ts:1. Codex account theo docs chính thức
    hiện nằm ở Codex CLI, không phải public browser auth flow.
  - Tính đến ngày 2026-04-20, docs chính thức xác nhận Codex CLI hỗ trợ sign-in bằng ChatGPT account và
    có mode non-interactive codex exec; đồng thời Windows vẫn là experimental.
    Nguồn: https://developers.openai.com/codex/cli , https://developers.openai.com/codex/noninteractive ,
    https://help.openai.com/en/articles/11381614-codex-codex-andsign-in-with-chatgpt

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
  - Track A: Codex CLI là full agent, GitNexus serve chỉ làm bridge + MCP/tool server
  - Track B: GitNexus backend tự orchestration, Codex CLI chỉ làm model executor
  - Chốt duy nhất 1 giao thức stream cho chat bridge để tránh tách đôi client/server:
  - Ưu tiên SSE nếu map tốt với UI hiện tại; nếu không thì chốt NDJSON và ghi rõ lý do
  - Ghi lại quyết định cho 4 điểm sau ngay trong plan hoặc ADR ngắn đi kèm:
  - Agent ownership nằm ở đâu
  - Tool execution nằm ở đâu
  - Stream protocol là gì
  - Repo binding được truyền theo repoName hay repoPath

 # Pha 1: Backend Codex Bridge

  - Repo đụng: gitnexus, gitnexus-shared.
  - Tạo mới:
  - gitnexus/src/server/codex-bridge.ts
  - gitnexus/src/server/codex-session.ts hoặc codex-job.ts
  - shared types trong gitnexus-shared cho CodexStatus, CodexChatRequest, CodexStreamEvent
  - Sửa:
  - /F:/GitNexus-main/gitnexus/src/server/api.ts:1 để thêm /api/codex/status
  - /F:/GitNexus-main/gitnexus/src/server/api.ts:1 để thêm /api/codex/chat stream SSE hoặc NDJSON
  - /F:/GitNexus-main/gitnexus/src/server/api.ts:1 để thêm cancel endpoint cho chat session
  - Contract tối thiểu cần chốt trước khi code:
  - Request chat phải mang repo binding rõ ràng: repoName hoặc repoPath; không ngầm dùng repo cuối cùng trên server
  - Response stream phải map được sang step/content/tool-call shape hiện tại hoặc có adapter rõ ràng ở web
  - Cancel phải có session identifier rõ ràng; switch repo phải abort chat đang chạy
  - Quy tắc bridge:
  - Chỉ cho phép chạy trong repo local đã index hoặc repo path hợp lệ
  - Không nhận path tùy ý ngoài workspace repo
  - Log stderr nội bộ, trả lỗi an toàn kiểu “Codex not installed” hoặc “Codex not signed in”
  - Ưu tiên map event stream về gần shape AgentStreamChunk hiện có để giảm sửa UI
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

 # Pha 2: Di chuyển chat của web sang bridge

  - Repo đụng: gitnexus-web.
  - Khuyến nghị an toàn: tạo file mới trước, không đập ngay agent.ts.
  - Tạo mới:
  - gitnexus-web/src/core/llm/codex-client.ts
  - Có thể thêm gitnexus-web/src/core/llm/codex-types.ts nếu chưa đưa vào gitnexus-shared
  - Sửa:
  - /F:/GitNexus-main/gitnexus-web/src/hooks/useAppState.tsx:1 để dùng backend Codex stream thay cho
    createGraphRAGAgent()
  - /F:/GitNexus-main/gitnexus-web/src/core/llm/types.ts:1 để đưa active provider mặc định sang codex-
    account hoặc rút còn một mode
  - /F:/GitNexus-main/gitnexus-web/src/core/llm/settings-service.ts:1 để migrate settings cũ và bỏ logic
    API-key
  - /F:/GitNexus-main/gitnexus-web/src/components/SettingsPanel.tsx:1 để chỉ còn “Codex Account” + local
    backend
  - /F:/GitNexus-main/gitnexus-web/src/components/RightPanel.tsx:1 để bỏ message “Configure an LLM provider”
  - /F:/GitNexus-main/gitnexus-web/src/hooks/useAppState.tsx:1 để đổi error/init flow từ “provider” sang “Codex account bridge”
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
  - Nếu sessionStorage/localStorage đang chứa provider cũ, hoặc migrate sang codex-account, hoặc clear one-shot có kiểm soát
  - Nếu clear, phải có fallback UX rõ ràng thay vì app boot vào trạng thái lỗi mơ hồ
  - Khi switch repo, chat mới phải bind đúng repo đang active; chat cũ phải bị hủy hoặc tách session rõ ràng
  - Test cần thêm/sửa:
  - gitnexus-web/test/unit/settings-service.test.ts
  - test stream client/backend-client mới cho Codex
  - test init/send/cancel chat flow nếu cần
  - test settings migration/reset từ dữ liệu provider cũ

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
  - gitnexus-web/src/services/backend-client.ts và useBackend chỉ chấp nhận backend local nếu muốn khóa
    chặt hoàn toàn
  - /F:/GitNexus-main/gitnexus/src/cli/serve.ts:1 update help/comment/copy để không còn assumption về hosted frontend
  - Xóa mọi copy “remote provider”, “hosted UI”, “GitHub URL”, “cloud” trong onboarding/settings/help
  - Acceptance:
  - UI không còn gợi ý remote
  - Setup/runtime không còn tự kéo remote package
  - Backend không còn mở cho Vercel/LAN nếu bạn muốn khóa tuyệt đối

 # Pha 5: Tắt wiki an toàn

  - Repo đụng: gitnexus.
  - Khuyến nghị 2 bước để giảm rủi ro.
  - Bước 1, hard-disable public surface:
  - Bỏ wiki command trong /F:/GitNexus-main/gitnexus/src/cli/index.ts:1
  - Bỏ mention wiki trong src/cli/ai-context.ts
  - Update help/docs/tests để wiki không còn xuất hiện
  - Bước 2, cleanup sau khi app ổn:
  - Xóa src/cli/wiki.ts
  - Xóa src/core/wiki/*
  - Xóa/điều chỉnh các kiểu config chỉ còn phục vụ wiki trong /F:/GitNexus-main/gitnexus/src/storage/repo-manager.ts:468 nếu không còn ai dùng
  - Xóa wiki tests liên quan
  - Lý do tách 2 bước: đỡ tạo một diff xóa rất lớn cùng lúc với migration Codex/local-only

 # Pha 6: Dọn provider cũ sau khi bridge xanh

  - Repo đụng: gitnexus-web.
  - Sau khi Codex bridge ổn định end-to-end:
  - Xóa nhánh openai, azure-openai, gemini, anthropic, ollama, openrouter, minimax, glm
  - Dọn helper OpenRouter/Ollama trong /F:/GitNexus-main/gitnexus-web/src/components/SettingsPanel.tsx:1
  - Dọn builder cũ trong /F:/GitNexus-main/gitnexus-web/src/core/llm/settings-service.ts:1
  - Dọn model branches cũ trong /F:/GitNexus-main/gitnexus-web/src/core/llm/agent.ts:1 nếu file này còn
    tồn tại
  - Cập nhật toàn bộ test còn hard-code gemini hoặc provider cũ

 # Checklist hoàn thành

  - Chat chạy bằng tài khoản Codex đã login, không cần API key
  - UI không còn picker provider remote
  - UI không còn GitHub URL input
  - Backend không clone/pull từ URL
  - Backend chỉ nhận absolute local paths
  - setup không còn kéo gitnexus@latest
  - wiki không còn trong public CLI surface
  - CORS/backend flow không còn remote suggestion
  - Track A hoặc Track B đã được chốt sau spike, không giữ hai hướng song song
  - Chat request luôn bind đúng repo đang active
  - UNC/network-share path bị reject ở analyze và chat bridge
  - settings cũ được migrate hoặc reset có kiểm soát, không để app boot lỗi vì provider state cũ
  - Cancel chat và cancel analyze vẫn hoạt động
  - Graph browsing/search/query hiện có không bị regress

 # Test matrix cần chạy khi triển khai

  - gitnexus:
  - npx tsc --noEmit
  - npm test
  - unit trọng điểm: analyze-api.test.ts, analyze-job.test.ts, codex-bridge.test.ts, cli-index-help.test.ts
  - integration/e2e trọng điểm: cli-e2e.test.ts cho help surface bị ảnh hưởng bởi việc tắt wiki
  - gitnexus-web:
  - npx tsc -b --noEmit
  - npm test
  - unit trọng điểm: settings-service.test.ts, server-connection.test.ts, test mới cho codex stream client, test mới cho RepoAnalyzer local-only

 # Inventory file dự kiến

  - gitnexus: src/server/api.ts, src/server/analyze-job.ts, src/server/git-clone.ts, src/server/
    *codex*.ts mới, src/cli/index.ts, src/cli/setup.ts, src/cli/serve.ts, src/cli/ai-context.ts,
    src/storage/repo-manager.ts, tests liên quan
  - gitnexus-web: src/core/llm/types.ts, src/core/llm/settings-service.ts, src/core/llm/agent.ts hoặc
    src/core/llm/codex-client.ts mới, src/hooks/useAppState.tsx, src/components/SettingsPanel.tsx, src/
    components/RepoAnalyzer.tsx, src/components/AnalyzeOnboarding.tsx, src/components/OnboardingGuide.tsx,
    src/components/Header.tsx, src/components/RepoLanding.tsx, src/components/RightPanel.tsx, src/
    components/HelpPanel.tsx, src/services/backend-client.ts, src/config/ui-constants.ts, tests liên quan
  - gitnexus-shared: shared request/response/event types nếu chuẩn hóa contract giữa backend và web
  - docs/tests: README/CONTRIBUTING/help/unit tests bị ảnh hưởng
