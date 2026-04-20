# Supervisor Review — Phase 1 Local Runtime

- Plan: `docs/plans/2026-04-20-convert-all-to-local.md`
- Scope reviewed: `Pha 1: Shared Local Runtime + Session Bridge`
- Reviewed at: `2026-04-20 20:38:24 +07:00`
- Reviewer: `gpt-5-codex`
- Repo head during review: `898d813`
- Verdict: `NOT APPROVED`

## Applicable rules

- `AGENTS.md`: complex-task execution sequence, repo scope boundaries, required validation discipline.
- `GUARDRAILS.md`: least privilege, non-destructive review, surface high-risk architectural drift.
- Supervisor hard rule from user: nếu trong scope vừa đụng vẫn còn `dead code`, `dead path`, `stale handler`, `stale test`, hoặc `stale wiring` thì phải xem là blocker `HIGH`, không approve.

## Evidence collected

- Read plan and hard rules: `AGENTS.md`, `GUARDRAILS.md`, `docs/plans/2026-04-20-convert-all-to-local.md`
- Read core phase-1 files:
  - `gitnexus-shared/src/session.ts`
  - `gitnexus/src/runtime/session-adapter.ts`
  - `gitnexus/src/runtime/runtime-controller.ts`
  - `gitnexus/src/runtime/session-jobs/session-job.ts`
  - `gitnexus/src/runtime/session-adapters/codex.ts`
  - `gitnexus/src/server/session-bridge.ts`
  - `gitnexus/src/server/api.ts`
- Read current web wiring state:
  - `gitnexus-web/src/hooks/useAppState.tsx`
  - `gitnexus-web/src/core/llm/agent.ts`
  - `gitnexus-web/src/core/llm/settings-service.ts`
  - `gitnexus-web/src/components/SettingsPanel.tsx`
  - `gitnexus-web/src/components/RightPanel.tsx`
- Runtime probes executed:
  - `runtime.getStatus({ repoPath: 'F:/GitNexus-main/deploy' })` => `index_required`
  - `runtime.getStatus({ repoPath: 'https://github.com/openai/openai' })` => rejected as invalid remote URL
  - `runtime.getStatus({ repoPath: 'F:/this/path/does/not/exist' })` => raw `ENOENT`
  - `runtime.startChat({ message: 'hi', repoPath: 'F:/this/path/does/not/exist' })` => raw `ENOENT`
  - `codex exec --json` minimal probes to inspect event shape

## Validation run

- `cd gitnexus && npx tsc --noEmit` => pass
- `cd gitnexus-shared && npx tsc --noEmit` => pass
- `cd gitnexus && npx vitest run test/unit/analyze-api.test.ts test/unit/wiki-flags.test.ts` => pass, `21/21`
- `cd gitnexus && npm test` => `6582` passed, `98` skipped, but Vitest reported `3` unhandled worker-fork errors, so suite was not clean-green

## Findings

### HIGH-1 — Windows execution path contradicts the Phase 0 architectural decision

- Plan decision: Windows phải ưu tiên `WSL2 bridge` cho full agent mode sau spike, không tiếp tục dựa vào native sandbox path.
- Current implementation still launches native Codex on Windows via `codex.cmd` and simply flips to bypass mode:
  - `gitnexus/src/runtime/session-adapters/codex.ts:24`
  - `gitnexus/src/runtime/session-adapters/codex.ts:34`
  - `gitnexus/src/runtime/session-adapters/codex.ts:217`
  - `gitnexus/src/runtime/session-adapters/codex.ts:275`
  - `gitnexus/src/runtime/session-adapters/codex.ts:277`
- This is not a naming issue. It changes the actual execution environment and ignores the explicit `WSL2-first` decision already recorded in the plan.
- Result: phase-1 runtime is architecturally off-track on the platform that the spike explicitly flagged as risky.

### HIGH-2 — Missing-folder handling leaks raw `ENOENT`, so the local-path contract is not closed

- The phase-1 contract requires explicit handling for invalid path, missing folder, UNC path, traversal, and repo-binding mismatch.
- `runtime-controller` validates remote URL, UNC, and absolute-path shape, but then calls `fs.realpath()` without converting `ENOENT` into a structured session error:
  - `gitnexus/src/runtime/runtime-controller.ts:184`
  - `gitnexus/src/runtime/runtime-controller.ts:191`
  - `gitnexus/src/runtime/runtime-controller.ts:201`
  - `gitnexus/src/runtime/runtime-controller.ts:206`
- Direct probe evidence:
  - `runtime.getStatus({ repoPath: 'F:/this/path/does/not/exist' })` => `ENOENT: no such file or directory, realpath ...`
  - `runtime.startChat({ message: 'hi', repoPath: 'F:/this/path/does/not/exist' })` => same raw `ENOENT`
- This is a real contract break, not theoretical. It also directly matches a test item that phase 1 said must exist, but no such test was added.

### HIGH-3 — The new stream contract has stale wiring inside its own scope

- `SessionReasoningEvent` is defined in shared types:
  - `gitnexus-shared/src/session.ts:87`
- The current Codex adapter never emits a `reasoning` event. It emits pre-tool and post-tool agent messages as plain `content`:
  - `gitnexus/src/runtime/session-adapters/codex.ts:142`
  - `gitnexus/src/runtime/session-adapters/codex.ts:150`
  - `gitnexus/src/runtime/session-adapters/codex.ts:170`
  - `gitnexus/src/runtime/session-adapters/codex.ts:362`
- The current web chat renderer still has distinct reasoning-step semantics waiting on the stream shape:
  - `gitnexus-web/src/hooks/useAppState.tsx:732`
- This leaves a dead stream type already baked into the new contract and guarantees extra UI surgery later instead of the promised compatibility-first bridge.

### HIGH-4 — Tool result payload is currently lossy for real Codex command events

- The adapter only builds tool results from `stdout`, `stderr`, `output`, or `text`:
  - `gitnexus/src/runtime/session-adapters/codex.ts:107`
  - `gitnexus/src/runtime/session-adapters/codex.ts:118`
- Real Codex `command_execution` events in this environment return `aggregated_output`, not `stdout`:
  - direct probe: `codex exec --json --skip-git-repo-check --dangerously-bypass-approvals-and-sandbox "Run one command to list the top-level entries..."`
  - observed event shape included `item.completed` with `type: "command_execution"` and `aggregated_output: "..."`.
- Because `aggregated_output` is ignored, tool cards lose actual command output. That is stale wiring in the bridge itself, in the exact scope just added.

### HIGH-5 — Required phase-1 tests are missing, so same-scope stale-test blocker applies

- Plan-required tests:
  - `gitnexus/test/unit/session-bridge.test.ts`
  - `gitnexus/test/unit/setup-session-runtime.test.ts`
  - API contract tests for `/api/session/status` and `/api/session/chat`
  - Codex adapter lifecycle tests
  - local-path policy tests for absolute path, traversal, UNC, missing folder, repo mismatch
- Current repo does not contain those files. Grep inventory in `gitnexus/test/unit` only shows unrelated coverage such as:
  - `gitnexus/test/unit/analyze-api.test.ts`
  - `gitnexus/test/unit/wiki-flags.test.ts`
- Because the scope added a new runtime and a new HTTP bridge without closing its own tests, this is a hard blocker under the supervisor rule.

### MEDIUM-1 — Phase 1 landed only backend scaffolding; frontend chat is still on the old provider path

- Current UI still initializes browser-side provider agent:
  - `gitnexus-web/src/hooks/useAppState.tsx:573`
  - `gitnexus-web/src/hooks/useAppState.tsx:592`
  - `gitnexus-web/src/hooks/useAppState.tsx:606`
  - `gitnexus-web/src/hooks/useAppState.tsx:986`
  - `gitnexus-web/src/hooks/useAppState.tsx:987`
- Current UI copy still instructs provider/API-key setup:
  - `gitnexus-web/src/components/RightPanel.tsx:453`
  - `gitnexus-web/src/components/SettingsPanel.tsx:357`
  - `gitnexus-web/src/components/SettingsPanel.tsx:358`
  - `gitnexus-web/src/components/SettingsPanel.tsx:441`
- This is acceptable as “phase 2 not started yet”, but it means phase 1 completion is backend-only, not end-to-end usable.

### MEDIUM-2 — `supportsMcp: true` is advertised, but runtime-owned MCP wiring is not visible in the adapter

- `supportsMcp` is reported as `true`:
  - `gitnexus/src/runtime/session-adapters/codex.ts:221`
  - `gitnexus/src/runtime/session-adapters/codex.ts:236`
- The adapter parses `mcp_tool_call` items if they appear:
  - `gitnexus/src/runtime/session-adapters/codex.ts:91`
  - `gitnexus/src/runtime/session-adapters/codex.ts:103`
- But the adapter itself does not show runtime-owned MCP attachment/config. In practice this means tool availability still depends on ambient Codex config on the machine instead of being clearly owned by the new runtime surface.
- I am not scoring this as a phase-1 blocker only because the plan left a dedicated validation task for Codex tool-call path after spike, but this is still incomplete closure.

### POSITIVE — A meaningful part of phase 1 is real, not fake

- Shared session/runtime types exist and are exported:
  - `gitnexus-shared/src/session.ts:1`
  - `gitnexus-shared/src/index.ts:25`
  - `gitnexus-shared/src/index.ts:28`
  - `gitnexus-shared/src/index.ts:29`
  - `gitnexus-shared/src/index.ts:30`
  - `gitnexus-shared/src/index.ts:31`
  - `gitnexus-shared/src/index.ts:40`
- Runtime controller exists and enforces several good rules:
  - explicit repo binding required
  - remote URL rejection
  - UNC rejection
  - absolute-path enforcement
  - same-repo chat supersede
  - `INDEX_REQUIRED` on existing local folder without index
- Session bridge routes are mounted in the HTTP server:
  - `gitnexus/src/server/api.ts:469`
  - `gitnexus/src/server/api.ts:471`
  - `gitnexus/src/server/session-bridge.ts:47`
  - `gitnexus/src/server/session-bridge.ts:56`
  - `gitnexus/src/server/session-bridge.ts:99`

## 19-task supervisor matrix

| # | Task | Phase 1 status | Verdict |
|---|------|----------------|---------|
| 1 | Theo dõi, không sửa tài liệu/code | Review-only, no project code changed by supervisor | OK |
| 2 | Rà soát code bằng grep verify | Done across runtime, server, web, tests | OK |
| 3 | Đánh giá chất lượng code | Có kiến trúc rõ nhưng closure chưa đủ | NOT OK |
| 4 | Liệt kê file/module chưa đạt best practices | Listed below | NOT OK |
| 5 | Đánh giá tốc độ sinh code của coder | Low-confidence: batch-drop style, closure/test lagging behind code | WATCH |
| 6 | Đánh giá style coder | Type-first, scaffold nhanh, backend-first; yếu ở edge-case closure và deterministic wiring | MIXED |
| 7 | Theo dõi wiring end-to-end | Backend bridge có; web chat chưa nối sang `/api/session/*` | NOT WIRED |
| 8 | Bindings wire chưa | `repoName/repoPath` binding ở backend có; frontend binding sang session bridge chưa có | PARTIAL |
| 9 | Frontend còn mock hay real API | Graph/search dùng real backend; chat vẫn browser-side provider flow | MIXED |
| 10 | Backend/VPS có vấn đề gì không | Runtime phase-1 có vấn đề Windows path + raw ENOENT; hosted/web-cloud copy còn tồn tại ngoài phase này | ISSUE |
| 11 | DB của VPS đã wire chưa | N/A cho phase 1 local runtime; không thấy DB/VPS path riêng | N/A |
| 12 | DB của client đã wire chưa | Local LadybugDB cho query/search/analyze đã wire; session bridge chỉ kiểm tra index/cwd | PARTIAL |
| 13 | Các nút UI đã wire backend/client chưa | Chat/settings vẫn chưa wire sang session runtime | NOT WIRED |
| 14 | Module có chạy đúng với Google Wire chưa | Không tìm thấy dấu vết `google/wire`; mục này không áp dụng cho repo hiện tại | N/A |
| 15 | Có tuân thủ luật Hard trong AGENTS.md không | Không thể approve. Thiếu test bắt buộc và còn stale wiring trong scope phase 1 | FAIL |
| 16 | Vấn đề tiềm ẩn tương lai | Ambient MCP config dependency, Windows native Codex path, lossy stream mapping, raw path errors | HIGH RISK |
| 17 | Đánh giá tổng quan theo Status -> SPEC -> Hard rules -> completion | Backend scaffold partial, chưa đạt phase-complete theo spec/hard rules | NOT COMPLETE |
| 18 | Đánh giá chất lượng project theo thời gian thực | Chất lượng pha này ở mức trung bình thấp: khung đúng, closure sai | MIXED |
| 19 | Khuyến nghị cho coder | Close blockers trước khi chạm phase 2 | REQUIRED |

## Files/modules below best practice for this phase

- `gitnexus/src/runtime/session-adapters/codex.ts`
  - Sai execution strategy trên Windows so với plan
  - Không map `aggregated_output`
  - Có `reasoning` contract nhưng không phát event tương ứng
- `gitnexus/src/runtime/runtime-controller.ts`
  - Raw `ENOENT` leak ra ngoài contract cho missing-folder case
- `gitnexus/src/server/session-bridge.ts`
  - Không có dedicated tests dù vừa mở API surface mới
- `gitnexus/test/unit/`
  - Thiếu hẳn bộ test cho `session-bridge`, `runtime-controller`, `codex adapter`, local-path policy
- `gitnexus-web/src/hooks/useAppState.tsx`
  - Chat runtime vẫn dùng `createGraphRAGAgent()` browser-side
- `gitnexus-web/src/core/llm/agent.ts`
  - Provider-based browser agent vẫn là đường chính
- `gitnexus-web/src/core/llm/settings-service.ts`
  - Vẫn là provider/API-key state model
- `gitnexus-web/src/components/SettingsPanel.tsx`
  - UI vẫn là provider picker + API-key forms
- `gitnexus-web/src/components/RightPanel.tsx`
  - Message “Configure an LLM provider” vẫn active

## Coder speed and style assessment

### Speed

- Code generation appears fast and top-down: shared types, runtime classes, adapter, bridge, and server mount all landed together.
- However, completion speed is outrunning closure speed:
  - edge cases not normalized
  - stream contract not fully honored
  - same-scope tests not added
- Scoped git history available during review was not granular enough to score precise throughput. The pattern visible in code is “large batch scaffold first, hardening later”.

### Style

- Good:
  - Type-first design
  - Explicit contracts in `gitnexus-shared`
  - Decent separation of `adapter` / `controller` / `bridge`
  - Local-path policy intent is visible
- Weak:
  - Ambient dependency assumptions are left implicit
  - Cross-phase compatibility promises are declared before event semantics are actually closed
  - Fast scaffold left stale test debt immediately in the touched scope
- Unusual:
  - The code advertises future-neutral abstractions (`claude-code`, `reasoning`, `supportsMcp`) before the first adapter has fully honored them

## Status vs spec vs hard rules

- `Status in code`: phase-1 backend scaffold exists and compiles.
- `Against spec`: not complete because Windows execution branch, missing-folder contract, stream compatibility, and test closure are not finished.
- `Against hard rules`: fails supervisor rule because touched scope still contains stale wiring and stale tests.
- `Real completion in code`: `partial`, not phase-complete.

## Recommendation to coder

1. Close phase 1 before starting phase 2.
2. Implement the Windows strategy exactly as decided in Phase 0, or explicitly reopen the architecture decision.
3. Normalize missing-folder and invalid-path failures into structured session errors.
4. Fix stream fidelity:
   - emit `reasoning` where the contract says it exists
   - capture `aggregated_output`
   - verify tool/result ordering with real Codex event fixtures
5. Add the missing phase-1 tests before any UI migration:
   - `session-bridge`
   - `runtime-controller`
   - `codex adapter`
   - local-path policy cases
6. Only after those blockers are closed should phase 2 start wiring the web UI to `/api/session/*`.

## Final supervisor verdict

- `Phase 1 is not approvable yet.`
- Reason is not “phase 2 chưa làm”.
- Reason is that phase 1 itself still contains `stale wiring`, `stale test debt`, and one direct contradiction with the recorded architecture decision.
