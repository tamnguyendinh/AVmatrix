# Mục đích plan này

- Plan này dùng để **tách hoàn toàn nhánh chat khỏi nhánh graph/render** trong `avmatrix-web`.
- Mục tiêu không phải redesign giao diện, không phải thay đổi tính năng, và không phải đổi hành vi agent.
- Mục tiêu chính là:
  - giảm lag khi gõ chat trong web local
  - ngăn graph state / Sigma render / highlight updates làm chat subtree rerender không cần thiết
  - chỉ gọi local session runtime / AVmatrix tool path khi **user thật sự gửi yêu cầu**

# Vấn đề hiện tại

- `RightPanel` đang là nơi chứa cùng lúc:
  - shell của panel
  - tab chat/processes
  - transcript render
  - composer/input
  - grounding handlers
  - session runtime state
- `useAppState.local-runtime.tsx` đang giữ:
  - graph state
  - AI highlight state
  - query state
  - code reference state
  - chat state
  - local session runtime state
- Hệ quả:
  - chat subtree có thể bị ảnh hưởng bởi graph/app-state rerender rộng
  - đường đi của chat chưa tách rõ khỏi graph
  - typing UX trong web local vẫn có thể lag dù transcript/composer đã tách component

# Kết quả mong muốn

- Chat có subtree và runtime riêng.
- `RightPanel` chỉ còn là shell/layout để chọn tab và mount child panels.
- `ChatPanel` là bề mặt chat hoàn chỉnh, nhưng:
  - không đọc graph trực tiếp
  - không import `GraphCanvas`
  - không import `useSigma`
  - không subscribe vào graph state rộng
- `useChatRuntime` / `ChatRuntimeContext` là runtime riêng cho chat:
  - quản lý session status
  - quản lý `chatMessages`
  - quản lý `isChatLoading`
  - quản lý `currentToolCalls`
  - quản lý `initializeAgent`, `sendChatMessage`, `stopChatResponse`, `clearChat`
- Chat chỉ gọi local runtime khi:
  - user `send`
  - hoặc app thật sự cần `initializeAgent`
- Mở panel chat và gõ input không được gọi query/tool path.

# Nguyên tắc cứng

1. **Không đổi giao diện.**
- Không đổi bố cục nhìn thấy của `RightPanel`.
- Không đổi cách tab `chat/processes` hiển thị.
- Không đổi visual của `Session offline`, `Analyze now`, transcript, composer.

2. **Không mất tính năng.**
- Phải giữ nguyên:
  - `Session offline`
  - `Analyze now`
  - `MarkdownRenderer`
  - `ToolCallCard`
  - Enter để gửi
  - Shift+Enter để xuống dòng
  - `stop response`
  - `clear chat`
  - suggestion chips
  - grounding click
  - tab `chat/processes`

3. **File mới trước.**
- Không đập thẳng file cũ để “refactor nhanh”.
- Phải tạo file mới song song trước:
  - runtime mới
  - panel mới
  - tests mới
- Chỉ khi parity đủ và tests pass mới swap wiring của file cũ.

4. **Không preload context graph vào chat path.**
- Mở chat panel: không query.
- Gõ input: không query.
- Chỉ khi `send` mới đi vào local session runtime.
- Chỉ khi agent/tool output yêu cầu grounding/highlight mới dùng bridge một chiều sang graph/code panel.

5. **Grounding là one-way bridge.**
- Chat không subscribe sâu vào graph để “chuẩn bị sẵn”.
- Chat chỉ gọi các callback bridge khi:
  - parse grounding từ content
  - parse highlight marker từ tool result
  - user click grounding link

# Kiến trúc mục tiêu

## 1. Shell panel

`RightPanel`
- chỉ giữ:
  - open/close
  - active tab
  - render `ChatPanel`
  - render `ProcessesPanel`

## 2. Chat runtime riêng

`ChatRuntimeProvider`
- giữ toàn bộ state runtime chat:
  - `llmSettings`
  - `isAgentReady`
  - `isAgentInitializing`
  - `agentError`
  - `chatMessages`
  - `isChatLoading`
  - `currentToolCalls`
- giữ methods:
  - `refreshLLMSettings`
  - `updateLLMSettings`
  - `initializeAgent`
  - `sendChatMessage`
  - `stopChatResponse`
  - `clearChat`

`useChatRuntime`
- hook duy nhất mà `ChatPanel` và `AppContent` dùng để chạm chat runtime

## 3. Bridge một chiều từ AppState sang ChatRuntime

`AppStateProviderInner` vẫn giữ:
- graph
- code references
- highlights
- selected node
- project/repo shell state

Nhưng chỉ cung cấp cho chat runtime một bridge tối thiểu:
- `getRepoName()`
- `getEmbeddingStatus()`
- `handleContentGrounding(fullText)`
- `handleToolResultMarkers(toolResult)`

Bridge này phải:
- ổn định về identity
- không kéo graph state vào `ChatPanel` trực tiếp
- không buộc `ChatPanel` phải subscribe vào graph context

# Files dự kiến tạo mới

- `avmatrix-web/src/hooks/chat-runtime/ChatRuntimeContext.tsx`
- `avmatrix-web/src/components/ChatPanel.tsx`
- Có thể giữ và tái sử dụng:
  - `avmatrix-web/src/components/right-panel/ChatTranscript.tsx`
  - `avmatrix-web/src/components/right-panel/ChatComposer.tsx`

# Files dự kiến sửa

- `avmatrix-web/src/hooks/useAppState.local-runtime.tsx`
- `avmatrix-web/src/components/RightPanel.tsx`
- `avmatrix-web/src/App.tsx`

# Files test dự kiến tạo/sửa

- `avmatrix-web/test/unit/ChatRuntimeContext.test.tsx`
- `avmatrix-web/test/unit/ChatPanel.test.tsx`
- `avmatrix-web/test/unit/RightPanel.local-runtime.test.tsx`
- `avmatrix-web/test/unit/ChatComposer.test.tsx`

# Blast radius đã kiểm tra

- `Function:avmatrix-web/src/components/RightPanel.tsx:RightPanel`
  - `risk: LOW`
  - `impactedCount: 0`
- `Function:avmatrix-web/src/hooks/useAppState.local-runtime.tsx:AppStateProviderInner`
  - `risk: LOW`
  - `impactedCount: 0`

# Pha triển khai

## Phase A — tạo runtime mới

Mục tiêu:
- tạo `ChatRuntimeProvider`
- chưa thay wiring cũ

Checklist:
- tạo file mới cho context/hook chat
- copy logic runtime chat từ `useAppState.local-runtime.tsx` sang file mới
- giữ nguyên behavior session hiện có
- chưa chạm `RightPanel` active path ngoài mức import chuẩn bị

## Phase B — bridge một chiều từ app state sang chat runtime

Mục tiêu:
- tách phần graph-dependent side effects ra khỏi runtime chat

Checklist:
- tạo callback bridge ổn định:
  - grounding từ content
  - highlight marker từ tool result
  - repo name getter
  - embedding status getter
- không để `ChatRuntimeProvider` phải đọc `graph` trực tiếp

## Phase C — tạo `ChatPanel`

Mục tiêu:
- gom chat UI vào file mới
- giữ nguyên transcript/composer hiện tại

Checklist:
- tạo `ChatPanel.tsx`
- `ChatPanel` dùng `useChatRuntime`
- `ChatPanel` render:
  - `ChatTranscript`
  - `ChatComposer`
- `ChatPanel` nhận vào tối thiểu:
  - `onRequestAnalyze`
  - `onGroundingClick` nếu cần
- chưa xóa logic cũ khỏi `RightPanel` cho tới khi parity xong

## Phase D — swap `RightPanel`

Mục tiêu:
- `RightPanel` chỉ còn shell

Checklist:
- `RightPanel` giữ:
  - active tab
  - close/open shell
  - mount `ProcessesPanel`
  - mount `ChatPanel`
- không còn đọc graph trực tiếp
- không còn giữ chat runtime state trực tiếp

## Phase E — migrate `AppContent`

Mục tiêu:
- `AppContent` không lấy `initializeAgent` / `refreshLLMSettings` từ app context cũ nữa

Checklist:
- `AppContent` lấy các chat-runtime methods từ `useChatRuntime`
- settings flow vẫn y nguyên
- onboarding / connect flow vẫn y nguyên

## Phase F — cleanup compatibility

Mục tiêu:
- dọn phần chat runtime cũ còn sót trong `useAppState.local-runtime.tsx`
- nhưng chỉ sau khi active path đã dùng runtime mới ổn định

Checklist:
- retire các state/method chat cũ khỏi `AppState` active path
- không xóa bừa nếu còn import path sống
- update tests theo active path mới

# Behavioral tests bắt buộc

1. **Typing không rerender transcript nặng**
- nhập vào composer
- transcript render count không tăng ngoài ý muốn

2. **Graph state đổi không kéo chat subtree rerender**
- thay graph/highlight/selection
- `ChatTranscript` không rerender nếu chat data không đổi

3. **Mở chat panel không gọi AVmatrix runtime**
- mount/open panel
- không có `streamSessionChat`
- không có tool/query call

4. **Chỉ `send` mới khởi động runtime**
- trước `send`: không có runtime activity
- sau `send`: mới có `initializeAgent` / `streamSessionChat`

5. **Grounding vẫn hoạt động**
- parse file refs
- parse node refs
- click grounding vẫn add code reference đúng

6. **Analyze CTA vẫn hoạt động**
- repo chưa index
- hiện `Analyze now`
- click vẫn gọi `requestRepoAnalyzeDialog`

7. **Composer behavior giữ nguyên**
- Enter gửi
- Shift+Enter xuống dòng
- clear
- stop
- suggestion prefill

# Validation commands

- `cd avmatrix-web && npx vitest run test/unit/ChatRuntimeContext.test.tsx test/unit/ChatPanel.test.tsx test/unit/RightPanel.local-runtime.test.tsx test/unit/ChatComposer.test.tsx`
- `cd avmatrix-web && npx tsc -b --noEmit`

# Điều kiện hoàn thành

Chỉ được gọi là xong khi:
- `RightPanel` không còn giữ runtime chat trực tiếp
- `ChatPanel` active path chạy bằng runtime riêng
- graph không còn đi trực tiếp vào chat subtree
- UI không đổi
- tính năng không mất
- toàn bộ behavioral tests mới pass

# Những thứ chưa được làm trong plan này

- Không tối ưu `MarkdownRenderer`
- Không tối ưu `GraphCanvas`
- Không đổi `Sigma` layout logic
- Không redesign `AppStateContext`

Lý do:
- plan này chỉ xử lý **tách chat khỏi graph**
- không mở rộng scope sang tối ưu khác trước khi nhánh chat sạch kiến trúc
