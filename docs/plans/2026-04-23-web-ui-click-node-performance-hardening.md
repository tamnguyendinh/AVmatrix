# Web UI Click-Node Performance Hardening Plan

Date: 2026-04-23
Scope: `avmatrix-web/` only
Status: Proposed

## Goal

Sửa cảm giác chậm khi người dùng:

- click một node trên graph
- chờ node/edge response
- mở Code Inspector
- nhìn selected context trên graph

Mục tiêu là:

- click vào node phải có phản hồi thị giác gần như tức thì
- selected-node context edges phải hiện nhanh
- không để click một node kéo theo quá nhiều full-graph work không cần thiết

Plan này chỉ xử lý frontend runtime và rendering path.

Không đổi:

- backend
- MCP
- `/api/graph`
- graph index format
- cấu trúc graph data trả từ server
- graph data model
- graph payload từ server
- secondary data source riêng cho selected subgraph

## Core Product Principle

Graph đã được build đầy đủ sau khi connect/analyze:

- toàn bộ node đã có
- toàn bộ relationship đã có
- graphology graph đã được dựng sẵn trong memory

Vì vậy bài toán tối ưu ở đây **không phải**:

- tái tạo graph links
- gọi backend để tạo lại selected links

Mà là:

- giảm chi phí **lọc / quyết định hiển thị / refresh render**
- tái sử dụng graph links đã có
- chỉ render subset edge phù hợp với context người dùng đang xem

Nguyên lý cần giữ:

- **Graph is built once**
- **edge rendering must be context-scoped**
- **selection should reuse existing adjacency, not trigger broad recomputation**

## Problem Statement

Hiện tại click một node đang làm nhiều việc nặng cùng lúc:

1. đổi selected state trong Sigma
2. đổi selected state trong app state
3. refresh graph rendering
4. chạy lại graph filtering path
5. mở Code Inspector
6. fetch source code và render syntax highlight
7. kéo theo re-render của nhiều consumer trong `AppState`

Kết quả là:

- node không phản hồi ngay như mong muốn
- edge có cảm giác hiện chậm
- UX bị hiểu như “click rồi mà graph chưa ăn”

## Current Code Reality

## 1. Click node tự gây camera move giả

Ở [useSigma.ts](/F:/AVmatrix-main/avmatrix-web/src/hooks/useSigma.ts):

- `setSelectedNode()` gọi `camera.animate(...)`
- rồi `sigma.refresh()`

Trong khi Sigma config đang có:

- `hideEdgesOnMove: true`

Nghĩa là path selection hiện tại tự tạo ra một “mini move”, làm edge bị ẩn đúng lúc người dùng vừa click.

Đây là một nguồn gây cảm giác “không hiện tức thì” rất mạnh.

## 2. Selection đang đi qua nhiều refresh chồng nhau

Path hiện tại:

- `sigma.on('clickNode')`
- `GraphCanvas.handleNodeClick`
- `setSelectedNode(node)` ở app state
- `openCodePanel()`
- effect sync app selection -> Sigma selection
- effect filter graph -> `sigma.refresh()`

Tức là một click không phải một update, mà là nhiều update nối nhau.

## 3. Mỗi click vẫn chạm full-graph filter path

Ở [GraphCanvas.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/GraphCanvas.tsx):

- effect theo `appSelectedNode`, `depthFilter`, `visibleLabels`
- luôn gọi `filterGraphByDepth(...)`

Ở [graph-adapter.ts](/F:/AVmatrix-main/avmatrix-web/src/lib/graph-adapter.ts):

- nếu `depthFilter === null`
- code vẫn đi vào `filterGraphByLabels(...)`
- và loop toàn bộ node

Nghĩa là:

- dù user chỉ chọn một node
- và dù depth filter không bật
- app vẫn quét lại cả graph

Đây là full-graph work không cần thiết.

## 4. Selected-node reducer vẫn đang khá đắt

Ở [useSigma.ts](/F:/AVmatrix-main/avmatrix-web/src/hooks/useSigma.ts):

- `nodeReducer` dùng `graph.hasEdge(...)` để xác định neighbor của selected node
- việc đó xảy ra trong path reducer khi refresh

Điều này làm selection path nặng hơn mức cần thiết, nhất là khi có nhiều refresh chồng lên nhau.

## 5. Click graph đang bị gắn chặt với mở Code Inspector

Ở [GraphCanvas.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/GraphCanvas.tsx):

- click node gọi ngay `openCodePanel()`

Ở [CodeReferencesPanel.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/CodeReferencesPanel.tsx):

- panel mount lên
- đọc file bằng `readFile(...)`
- render `SyntaxHighlighter`

Điều này không phải nguyên nhân duy nhất, nhưng nó làm selection path nặng hơn rất rõ.

## 6. `AppState` hiện là một context lớn

Ở [useAppState.local-runtime.tsx](/F:/AVmatrix-main/avmatrix-web/src/hooks/useAppState.local-runtime.tsx):

- `selectedNode`
- `codeReferences`
- `isCodePanelOpen`
- nhiều state khác cùng nằm trong một `value`

Rất nhiều component cùng subscribe `useAppState()`.

Nên chỉ một thay đổi `selectedNode` cũng có thể kéo theo re-render rộng ở shell.

## Product Rules To Lock Before Coding

### Rule 1. Click selection phải ưu tiên feedback tức thì

Ngay khi user click node:

- selected visual state phải xuất hiện ngay
- selected context edges phải có đường hiện nhanh

Không được để việc mở panel/code loading chặn cảm giác phản hồi ban đầu.

### Rule 1a. Plain selection không được tự recenter camera

Path click chọn node bình thường:

- không được tự animate camera
- không được tự recenter graph

Camera movement chỉ được giữ cho:

- `focusNode()`
- hoặc action explicit kiểu “focus on selected”

### Rule 2. Không tái tạo graph links

Tuyệt đối không đi theo hướng:

- rebuild selected graph
- fetch lại edge từ backend
- generate lại link mới cho node được chọn

Chỉ được:

- reuse graph data đã có
- chọn subset edge cần render

### Rule 3. Selected-node context là local subset, không phải global recomputation

Khi chọn node:

- chỉ cần adjacency trực tiếp của node đó
- không cần chạy logic rộng cho toàn graph nếu user không bật depth filter

### Rule 3a. Selected adjacency cache phải là local/lazy cache

Nếu dùng cache cho selected-node context, cache đó phải là:

- local theo selected node
- hoặc lazy theo nhu cầu

Không được:

- duplicate toàn bộ graph adjacency sang một data structure mới chỉ để phục vụ selection
- thêm một selected-subgraph model độc lập với graph hiện có

### Rule 4. Code Inspector không được làm chậm graph response ban đầu

Graph response và code response là hai việc khác nhau:

- graph selection feedback phải nhanh
- code load có thể theo sau

### Rule 5. Ambient graph logic và selection logic phải tách

Không trộn:

- ambient graph rendering
- selected-node context rendering
- code-panel opening
- source loading

thành một path click duy nhất không có phân lớp.

### Rule 6. Selected context vẫn phải obey edge filters

Tối ưu selection performance không được làm đổi semantics edge filtering.

Nghĩa là:

- selected context reuse adjacency
- nhưng vẫn phải obey `visibleEdgeTypes`

## Desired UX

### Case A. Click node with no depth filter

- node highlight gần như xuất hiện ngay
- direct context edges hiện nhanh
- graph không có cảm giác “khựng”

### Case B. Click node while Code Inspector is closed

- graph highlight xuất hiện trước
- Code Inspector có thể mở sau rất ngắn
- nhưng không làm trễ selected graph response

### Case C. Click another node immediately after previous selection

- selection chuyển mượt
- không có cảm giác repaint nặng hoặc “đợi panel”

### Case D. Depth filter is null

- không có full-graph filtering pass chỉ vì đổi selection

## Design Direction

Hướng bền vững là:

1. tách selection response thành hai tầng:
- graph response
- side-panel/code response

2. tách edge rendering thành:
- ambient edges
- selected-node context edges

3. chuyển selected-node context sang reuse adjacency đã có

4. tránh full-graph work trong path click nếu user không bật feature cần nó

## Proposed Implementation

## Workstream A. Remove unnecessary full-graph filtering on plain selection

Ở [GraphCanvas.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/GraphCanvas.tsx),
effect hiện tại đang gọi `filterGraphByDepth(...)` cả khi:

- `depthFilter === null`
- user chỉ vừa đổi selected node

Điều này cần được chặn.

Hướng đúng:

- nếu `depthFilter === null`
- selection đổi không được trigger full-graph label pass nữa

Chỉ khi:

- `visibleLabels` đổi
- hoặc `depthFilter` thực sự có giá trị

thì mới cần chạy logic đó.

Đây là quick win lớn nhất.

## Workstream B. Remove the fake camera nudge from selection path

`setSelectedNode()` trong [useSigma.ts](/F:/AVmatrix-main/avmatrix-web/src/hooks/useSigma.ts)
đang dùng `camera.animate(...50ms)` như workaround để ép edge refresh.

Đây là điểm cần xử lý cẩn thận.

Mục tiêu:

- không dùng mini camera move để refresh selection nữa
- tránh đụng `hideEdgesOnMove` đúng lúc user vừa click
- giữ camera motion cho path focus explicit, không giữ cho plain selection

Các hướng cần đánh giá:

1. chỉ dùng `sigma.refresh()` nếu đã đủ
2. nếu Sigma edge cache còn issue, tìm trigger nhẹ hơn không đi qua camera motion
3. chỉ giữ motion thật sự cho `focusNode()`, không giữ cho plain selection

## Workstream C. Cache direct adjacency for selected node

Thay vì để reducer liên tục hỏi:

- `graph.hasEdge(node, currentSelected)`
- `graph.hasEdge(currentSelected, node)`

trong path refresh,

nên tính một lần khi selection đổi:

- `selectedNeighborNodeIds`
- `selectedDirectEdgeIds`

rồi reducer chỉ đọc các set đó.

Mục tiêu:

- selection cost trở thành local lookup
- không lặp lại graph queries rộng trong reducer
- không duplicate toàn bộ graph adjacency chỉ để phục vụ selection

Điểm cần khóa:

- selected adjacency cache chỉ là cache cục bộ/lazy
- selected context vẫn phải obey `visibleEdgeTypes`

## Workstream D. Decouple graph response from Code Inspector opening

Hiện tại click graph đang gắn ngay với:

- `openCodePanel()`
- fetch source
- syntax highlighting

Hướng đúng hơn:

- graph selection phản hồi trước
- panel/code work đi sau

Không bắt buộc phải đổi UX thành “không mở panel nữa”.

Nhưng implementation nên theo hướng:

- selection update được ưu tiên
- panel open/code load là lower-priority follow-up

Nếu phù hợp với codebase, có thể dùng:

- deferred state
- transition
- hoặc tách commit update để graph repaint trước

## Workstream E. Narrow context re-render blast radius

`AppState` đang khá rộng.

Không nhất thiết phải tách toàn bộ trong một lượt.

Nhưng nên khóa mục tiêu:

- giảm số component phải re-render chỉ vì `selectedNode` đổi

Các hướng khả thi:

1. chia nhỏ context
2. tách graph-selection state khỏi side-panel state
3. dùng selector-like pattern hoặc provider nhỏ hơn

Đây không phải quick win đầu tiên, nhưng là hướng bền vững.

## Workstream F. Add instrumentation before and after optimization

Không tối ưu mù.

Phải đo ít nhất các mốc:

1. click node event received
2. selected state committed
3. graph repaint complete
4. code panel open committed
5. source loaded

Mục tiêu là tách được:

- chậm ở graph response
- hay chậm ở panel/code response

## Refactor Safety Rule

Path này chạm vào logic trung tâm của:

- [useSigma.ts](/F:/AVmatrix-main/avmatrix-web/src/hooks/useSigma.ts)
- [GraphCanvas.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/GraphCanvas.tsx)
- [useAppState.local-runtime.tsx](/F:/AVmatrix-main/avmatrix-web/src/hooks/useAppState.local-runtime.tsx)

Vì vậy:

- nếu chỉ vá điều kiện nhỏ, có thể sửa trực tiếp
- nhưng nếu selection/render logic bắt đầu phình ra, phải tạo helper hoặc file mới song song trước

Đặc biệt:

- không được tiếp tục nhét thêm workaround selection vào cùng một callback
- không được trộn graph refresh workaround với panel opening logic

## Validation

## 1. Typecheck

Phải chạy:

- `cd avmatrix-web && npx tsc -b --noEmit`

## 2. Targeted tests

Ít nhất phải có regression coverage cho:

- click selection không làm mất selected context edges
- selection khi `depthFilter === null` không đi vào full-graph filtering path không cần thiết
- selected adjacency cache/helper trả đúng direct neighbors/direct edges
- selected context vẫn obey `visibleEdgeTypes`
- khi `depthFilter !== null`, selection vẫn cập nhật đúng phạm vi N-hop

## 3. Manual validation

Phải mở web UI thật và kiểm tra:

1. load repo lớn như `AVmatrix-main`
2. click một node giữa graph
3. quan sát:
- node selected phản hồi nhanh
- selected context edges hiện nhanh
- không có cảm giác delay rõ trước khi graph đổi state
4. click liên tiếp nhiều node
5. xác nhận selection vẫn mượt
6. mở/đóng Code Inspector trong lúc click node
7. xác nhận graph response không bị panel open làm chặn rõ rệt
8. bật `depthFilter`
9. click node
10. xác nhận phạm vi N-hop vẫn cập nhật đúng

## 4. Performance spot-check

Phải có số đo trước/sau cho ít nhất:

- click-to-selected-visual
- click-to-context-edges-visible
- click-to-code-panel-open

## Risks

### Risk 1. Tối ưu nhầm vào backend

Bug này chủ yếu nằm ở frontend interaction path.

Nếu đẩy sang backend hoặc stream graph lại, sẽ làm bài toán lệch hướng.

### Risk 2. Tối ưu graph nhưng phá Code Inspector UX

Nếu cố làm graph nhanh bằng cách cắt luôn code path mà không điều phối tốt:

- UX tổng thể có thể thành rời rạc

Nên phải tối ưu theo thứ tự ưu tiên, không phải bỏ bớt tính năng.

### Risk 3. Tối ưu một điểm nhưng giữ nguyên refresh chồng nhau

Nếu chỉ sửa một hotspot nhỏ mà vẫn giữ:

- camera nudge
- full-graph filter pass
- reducer lookup đắt
- panel open sync

thì hiệu quả thực tế sẽ không đủ rõ với user.

### Risk 4. Tối ưu selection nhưng vô tình phá focus semantics hoặc depth filter semantics

Nếu bỏ camera work hoặc rút gọn filter path mà không tách rõ:

- `plain selection`
- `focusNode`
- `depthFilter`

thì rất dễ:

- làm mất recenter ở chỗ cần recenter
- hoặc làm depth filter stale

## Recommended Execution Order

1. chặn full-graph filter pass khi `depthFilter === null`
2. bỏ fake camera nudge khỏi plain selection path
3. cache adjacency cho selected node
4. tách graph response khỏi Code Inspector open/load
5. nếu vẫn còn chậm, mới tiếp tục tách nhỏ `AppState`

## Done Means

Feature được coi là xong khi:

- click node cho cảm giác phản hồi gần như tức thì
- selected context edges hiện nhanh, ổn định
- selection khi `depthFilter === null` không còn kích hoạt full-graph work vô ích
- graph response không bị Code Inspector path kéo chậm rõ rệt
- typecheck pass
- regression tests pass
- manual UX pass trên repo lớn
