# AVmatrix Web UI Chat Runtime Optimization

Last updated: 2026-04-22
Status: proposed

## Purpose

Plan nay chot huong toi uu trai nghiem chat trong `avmatrix-web` khi dung local Codex runtime.

Muc tieu cua plan:

- giam do tre tu luc user gui prompt den luc Codex bat dau stream
- giam giat/lag trong transcript va composer khi stream content/tool events
- giu MCP va graph engine nguyen ven, chi toi uu lop session runtime + web UI
- giu kha nang dung `Codex CLI + AVmatrix MCP` va khong keo chat path vao startup/indexing path

## Problem Statement

Sau khi tach ro graph layer khoi Codex integration, van de hieu nang con lai tap trung o web UI chat path.

Cac van de hien tai:

1. Moi tin nhan chat hien tai spawn mot `codex exec` moi
- `avmatrix/src/runtime/session-adapters/codex.ts`
- Moi request phai launch process moi, probe runtime, stream lai tu dau
- Day la startup tax lap lai tren tung luot chat

2. Runtime status bi kiem tra lap
- frontend goi `fetchSessionStatus()` trong `initializeAgent()`
- backend `runChat()` lai goi `getStatus()` mot lan nua truoc khi spawn
- viec lap nay khong tao gia tri UX tuong ung

3. Frontend xu ly qua nhieu viec trong stream loop
- `ChatRuntimeContext` cap nhat transcript lien tuc theo chunk
- moi `content` chunk lai co the keo grounding parse qua `bridge.handleContentGrounding(fullText)`
- neu chunk nho va den dan, UI de bi rerender day dac

4. Session lifecycle chua toi uu cho chat lien tuc
- moi request la session/process rieng
- chua co reuse theo repo/tab
- khi user hoi lien tiep, chi phi spin-up lap lai rat de cam nhan thay

5. Web UI va chat runtime van co nhieu diem can thiet ke ky de tranh rerender rong
- `ChatRuntimeContext` da tach rieng, nhung van can tiep tuc khoa ro boundaries
- grounding, tool highlights, repo binding, va transcript updates can la one-way va duoc throttle hop ly

## Scope

Plan nay chi xu ly:

- `avmatrix/src/runtime/session-adapters/codex.ts`
- `avmatrix/src/runtime/runtime-controller.ts`
- `avmatrix/src/server/session-bridge.ts`
- `avmatrix-web/src/hooks/chat-runtime/*`
- `avmatrix-web/src/components/ChatPanel.tsx`
- `avmatrix-web/src/components/right-panel/*`
- cac test cho chat runtime / web UI streaming path

Plan nay khong xu ly:

- `analyze` pipeline
- MCP startup
- graph construction
- LadybugDB query engine
- thay doi core MCP transport cho Codex CLI ngoai AVmatrix

## Current Architecture Notes

Kien truc hien tai da co mot diem dung quan trong:

- graph duoc AVmatrix tu tao truoc
- web UI chat chi duoc phep chay khi repo da index
- session runtime la downstream consumer cua graph, khong can thay doi nguyen ly nay

Nen toi uu o day phai giu nguyen nguyen tac:

- graph/indexing va chat/session la hai path rieng
- toi uu chat khong duoc pha vo tinh on dinh cua MCP va indexer

## Hard Goals

1. Khong duoc tao process Codex moi cho moi prompt neu co the reuse session an toan hon.

2. Time-to-first-token phai giam ro ret so voi mo hinh `codex exec` moi request.

3. Khi Codex dang stream:
- typing, scroll, va transcript cap nhat phai muot
- khong duoc co grounding parse/blocking nang tren moi chunk nho

4. Repo switch va session cancel van phai dung:
- doi repo thi session cu bi huy sach
- user bam stop thi session dung nhanh
- reload trang khong de lai zombie session logic

5. Khong duoc lam MCP/CLI path phuc tap hon chi vi web UI.

## Non-Goals

- Khong thay doi noi dung prompt cua user
- Khong redesign giao dien chat
- Khong chuyen AVmatrix thanh hosted remote runtime
- Khong dua Codex vao graph creation path

## Guiding Principles

1. Persistent session over spawn-per-message.
- Chi phi launch process lap lai la bottleneck lon nhat can uu tien giai quyet.

2. Fast path first.
- User send message phai di qua duong ngan nhat co the.

3. Stream lightly.
- Chi update UI voi muc tan suat hop ly; grounding/highlight phai debounce hoac hoan lai.

4. Chat path must stay downstream.
- Chat chi duoc doc graph/tool output; khong chen nguoc vao indexing path.

5. Repo-scoped isolation.
- Session thuoc ve mot repo binding ro rang; doi repo la doi session.

## Concrete Problems To Address

### A. Spawn-per-message runtime

Hien tai `runChat()` dung:

- `codex exec --json ...`
- spawn process moi
- doc ket qua tu temp file

He qua:

- moi prompt deu co cold start rieng
- user cam thay do tre o dau moi luot chat
- kho dat duoc trai nghiem chat "lien mach"

### B. Duplicate readiness checks

Frontend va backend deu xac minh runtime status truoc chat.

He qua:

- co them 1-2 network/process hop khong can thiet
- tang do tre ma user khong nhan them gia tri

### C. Content grounding tren moi chunk

`handleContentGrounding(fullText)` dang co the duoc goi lien tuc theo stream.

He qua:

- parse lai noi dung nhieu lan
- de lam transcript va code-reference path giat khi response dai

### D. Transcript update frequency

Mac du da co `requestAnimationFrame`, state van phai ghep buoc, tao message, va sync tool calls rat thuong xuyen.

He qua:

- response dai va nhieu tool events co the gay render pressure
- UI de co cam giac "nhoi" khi response chay nhanh

### E. Session ownership va cleanup

Session hien tai hop le ve chuc nang, nhung chua toi uu cho su dung chat lien tuc trong cung mot repo/tab.

Can lam ro:

- session nao duoc reuse
- session nao phai dong
- khi nao can reset toan bo transcript/runtime state

## Proposed Direction

## Phase 1. Introduce a persistent Codex chat session model

Muc tieu:

- thay `codex exec` moi request bang session song lau hon theo repo/tab neu Codex CLI support du duong stream cho mo hinh nay
- neu Codex CLI khong support session persistent dung nghia, can co adapter layer giu process nong va gui request lien tiep theo protocol ho tro

Changes:

1. danh gia va chot session transport cho Codex runtime trong AVmatrix
2. tao abstraction ro giua:
- runtime bootstrap
- send user message
- stream agent events
- cancel current turn
- dispose session
3. giu backward-compatible fallback ve `codex exec` cho truong hop environment khong ho tro

Expected outcome:

- giam phan lon startup tax moi prompt
- time-to-first-token cai thien ro ret

## Phase 2. Remove duplicate readiness work

Muc tieu:

- khong check trang thai lap o ca frontend va backend tren moi turn

Changes:

1. cache `SessionStatus` theo thoi gian ngan va theo repo/runtime
2. frontend chi refresh khi:
- vao repo moi
- session bi loi
- user mo settings/refresh thu cong
3. backend chi re-probe khi cache stale hoac spawn/session reuse that bai

Expected outcome:

- giam do tre truoc stream
- giam command/process phu khong can thiet

## Phase 3. Debounce grounding and heavy post-processing

Muc tieu:

- transcript stream muot hon khi response dai

Changes:

1. tach chunk accumulation khoi grounding parse
2. chi grounding theo:
- moi nhip debounce ngan
- hoac khi ket thuc content block
- hoac khi turn done
3. tool result marker/highlight updates cung can co co che gop nhom hop ly

Expected outcome:

- giam render pressure
- code reference/highlight van dung nhung khong chen vao fast path moi chunk

## Phase 4. Tighten frontend render boundaries

Muc tieu:

- transcript/composer/tool call cards chi rerender khi du lieu cua chinh no doi

Changes:

1. ra soat `ChatRuntimeContext` value de tranh identity churn khong can thiet
2. giu `ChatPanel`, `ChatTranscript`, `ChatComposer`, `ToolCallCard` o boundary ro rang
3. tan dung incremental state update thay vi tai tao object rong neu khong can

Expected outcome:

- typing muot hon
- scroll on dinh hon
- response dai it gay khung hinh rung

## Phase 5. Session lifecycle and cancellation polish

Muc tieu:

- session dung dung luc, dung dung cach

Changes:

1. chot semantics:
- cung repo, cung tab: co reuse hay khong
- doi repo: bat buoc cancel + dispose
- prompt moi khi turn cu dang chay: cancel hay queue
2. bo sung telemetry cho:
- session start
- first token
- done
- cancel
- error

Expected outcome:

- UX chat nhat quan
- debug duoc cac case "lag", "tre", "treo", "stop cham"

## Success Metrics

Can do it nhat cac chi so sau:

- time-to-first-token
- time-to-first-tool-call
- tong thoi gian moi turn
- so process Codex duoc spawn cho 10 user turns
- tan suat rerender transcript trong mot response dai
- so lan grounding parse trong mot response dai

Muc tieu v1:

- moi prompt khong con bi cold-start nang nhu hien tai
- chat web UI cam nhan ro rang muot hon tren response dai
- cancel/doi repo on dinh

Muc tieu tham vong:

- trai nghiem chat trong web UI gan voi session local lien mach, khong con cam giac moi prompt la mot lan "chay lai tu dau"

## Validation Plan

Bat buoc co:

1. manual benchmark:
- 5 turns lien tiep cung repo
- 1 turn co tool call
- 1 turn dai co nhieu chunk content
- 1 turn dang stream roi bam stop
- 1 lan doi repo trong luc dang co session

2. frontend verification:
- transcript update dung
- tool cards dung
- code grounding van nhay dung
- khong bi memory leak/session leak ro rang

3. runtime verification:
- khong spawn process khong can thiet
- cleanup session dung luc
- fallback path van chay khi persistent mode khong available

## Rollout Strategy

1. them telemetry truoc
2. dat co feature flag cho persistent session mode
3. do benchmark A/B giua:
- `spawn-per-message`
- `persistent-session`
4. sau khi on dinh moi doi default

## Open Questions

1. Codex CLI hien tai ho tro muc nao cho session song lau hon va stream turn-by-turn?
2. Neu can fallback, nen fallback theo:
- toan cuc
- theo environment
- hay theo tung session loi
3. Grounding nen parse theo:
- debounce 100-200ms
- theo paragraph boundary
- hay chi parse khi turn done roi hien thi lan cuoi?

## Expected Outcome

Sau khi xong plan nay:

- AVmatrix van giu MCP/local graph architecture hien tai
- Codex van nam o lop session/chat, khong chen vao indexing
- web UI se muot hon ro ret trong tuong tac chat local
- va phan "Codex integration" se co gia tri dung tam: chat nhanh, on dinh, it lag
