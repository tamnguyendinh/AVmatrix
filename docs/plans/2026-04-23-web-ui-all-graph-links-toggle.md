# Web UI All Graph Links Toggle Plan

Date: 2026-04-23
Scope: `avmatrix-web/` only
Status: Proposed

## Goal

Code chức năng cho nút `all graph links` trong web UI để người dùng có thể:

- bật hiển thị toàn bộ mạng lưới cạnh của graph
- tắt toàn bộ mạng lưới cạnh của graph

Plan này chỉ nói về frontend rendering và state của web UI.

Không đổi:

- backend
- MCP
- graph data model trong index
- API `/api/graph`

## Product Meaning To Lock Before Coding

`All graph links` phải được hiểu là:

- toàn bộ **edges / relationships** đang được render trong graph canvas
- cụ thể là toàn bộ cạnh mà frontend graph renderer hiện đang quản lý và có thể vẽ
- không phải node
- không phải process list
- không phải AI highlight layer
- không phải code references panel

Nút này là một **master visibility toggle** cho cạnh graph.

Khi bật:

- graph được phép render cạnh như bình thường
- các filter theo loại cạnh vẫn tiếp tục áp dụng

Khi tắt:

- toàn bộ cạnh trong graph canvas bị ẩn
- node vẫn hiện
- selection, highlight, focus, query, process, blast radius vẫn hoạt động ở mức node
- chỉ riêng đường nối bị tắt

Nói ngắn:

- `visibleEdgeTypes` = filter theo loại cạnh
- `all graph links` = công tắc tổng cho mọi cạnh

Điểm cần khóa để tránh hiểu nhầm:

- nút này không có nghĩa là “hiện tất cả mọi loại relationship tồn tại trong AVmatrix”
- nó chỉ áp dụng cho tập edge mà graph canvas hiện tại đã nhận từ graph adapter và đang có khả năng render

## Why This Direction Is Correct For Current Code

Codebase hiện đã có đủ nền để làm theo hướng này:

- state loại cạnh ở [graph.tsx](/F:/AVmatrix-main/avmatrix-web/src/hooks/app-state/graph.tsx)
- danh sách edge types ở [constants.ts](/F:/AVmatrix-main/avmatrix-web/src/lib/constants.ts)
- edge rendering và hide logic ở [useSigma.ts](/F:/AVmatrix-main/avmatrix-web/src/hooks/useSigma.ts)
- nút UI hiện nằm ở [GraphCanvas.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/GraphCanvas.tsx)

Điều này có nghĩa là:

- không cần tạo cơ chế backend mới
- không cần tạo route mới
- không cần thay schema graph

Chỉ cần thêm một lớp state master ở frontend rồi để `edgeReducer` tôn trọng lớp state đó.

## Current Reality In Code

### 1. Hệ edge filter hiện tại đã tồn tại

Hiện tại graph có:

- `visibleEdgeTypes`
- `toggleEdgeVisibility(edgeType)`

ở [graph.tsx](/F:/AVmatrix-main/avmatrix-web/src/hooks/app-state/graph.tsx).

Sigma reducer đang kiểm tra:

- nếu `relationType` không nằm trong `visibleEdgeTypes`
- thì `res.hidden = true`

ở [useSigma.ts](/F:/AVmatrix-main/avmatrix-web/src/hooks/useSigma.ts).

### 2. Nút mới hiện chỉ là placeholder UI

Ở [GraphCanvas.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/GraphCanvas.tsx) hiện mới có:

- nút `AI-driven highlights` có logic thật
- nút `all graph links` mới chỉ là disabled placeholder

### 3. Query / process / AI logic hiện đang tác động mạnh lên edge reducer

`edgeReducer` hiện:

- làm đậm cạnh khi node nguồn/đích nằm trong highlight set
- dim cạnh không liên quan

Điều này đúng.

Nhưng nếu người dùng tắt `all graph links`, lớp logic này phải bị chặn ở đầu reducer.

Nếu không, cạnh sẽ vẫn còn hiện theo highlight state, làm nút mới mất nghĩa.

## UX Rules To Lock

### Rule 1. Button chỉ điều khiển cạnh, không điều khiển node

Tắt `all graph links`:

- không được làm node biến mất
- không được clear selection
- không được clear query result
- không được clear process focus
- không được clear AI highlight state

### Rule 2. Button là master override, nhưng không phá filter hiện có

Semantics đúng phải là:

`effectiveEdgeVisibility = areAllGraphLinksVisible && relationType in visibleEdgeTypes`

Nghĩa là:

- nếu master toggle tắt, mọi cạnh đều ẩn
- nếu master toggle bật, cạnh nào còn hiện vẫn còn phụ thuộc vào filter loại cạnh hiện có

### Rule 3. Tắt links không được gây side effects ẩn

Không được:

- reset `visibleEdgeTypes`
- reset node highlights
- reset camera
- reset selected node
- reset process modal

Đây chỉ là visibility toggle.

### Rule 4. Nút phải có trạng thái rõ ràng

UI phải cho người dùng biết:

- đang bật links
- hay đang tắt links

Không được dùng icon mơ hồ mà không có tooltip/aria/state rõ.

### Rule 5. Trạng thái nên được nhớ lại

Vì đây là preference hiển thị, nên nên persist bằng localStorage.

Ví dụ key:

- `avmatrix.graphLinksVisible`

Nếu chưa có key, default nên là `true`.

Semantics được chốt trong plan này:

- đây là **global UI preference** ở mức browser cho web UI hiện tại
- không phải repo-local preference
- khi user đổi repo, preference này vẫn giữ nguyên

## Proposed Implementation

## Workstream A. Add master state to graph app-state

Thêm vào [graph.tsx](/F:/AVmatrix-main/avmatrix-web/src/hooks/app-state/graph.tsx):

- `areGraphLinksVisible: boolean`
- `setGraphLinksVisible: (visible: boolean) => void`
- `toggleGraphLinksVisible: () => void`

Khuyến nghị:

- state này thuộc `GraphStateProvider`
- không nhét riêng vào `GraphCanvas`

Lý do:

- đây là graph rendering preference cấp app state
- các component khác có thể cần đọc trạng thái này về sau

### Persistence

Trong cùng workstream này:

- load initial state từ localStorage
- persist khi user toggle

Default:

- `true`

## Workstream B. Thread state through app state facade

`useAppState.local-runtime.tsx` đang re-export rất nhiều graph-related state.

Cần expose thêm:

- `areGraphLinksVisible`
- `toggleGraphLinksVisible`

để `GraphCanvas.tsx` chỉ việc dùng như các control graph khác.

Mục tiêu:

- tránh để `GraphCanvas` đọc localStorage trực tiếp
- giữ state topology sạch

## Workstream C. Wire GraphCanvas button to real state

Ở [GraphCanvas.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/GraphCanvas.tsx):

1. bỏ `disabled` khỏi nút `all graph links`
2. wire `onClick` vào `toggleGraphLinksVisible`
3. đổi class để có 2 trạng thái rõ:
- links visible
- links hidden

Tooltip / aria:

- khi bật: `Turn off all graph links`
- khi tắt: `Turn on all graph links`

Không cần đổi layout của toolbar trong đợt này ngoài state styling.

## Workstream D. Make useSigma respect the master toggle

Ở [useSigma.ts](/F:/AVmatrix-main/avmatrix-web/src/hooks/useSigma.ts):

thêm option mới:

- `areGraphLinksVisible?: boolean`

và lưu vào ref giống `visibleEdgeTypes`.

Trong `edgeReducer`, kiểm tra sớm:

- nếu `areGraphLinksVisible === false`
- thì `res.hidden = true`
- `return res`

Check này phải nằm trước:

- filter theo `visibleEdgeTypes`
- selected-node edge emphasis
- highlight edge emphasis
- blast-radius edge emphasis

Lý do:

- đây là master override

## Workstream E. Clarify interaction with existing edge filters

Ở `FileTreePanel` hiện có UI filter theo edge type.

Plan này không đổi UX filter đó, nhưng cần bảo đảm:

- khi links bị master toggle tắt, filter panel không bị reset
- khi bật links lại, edge type filters cũ vẫn còn nguyên

Nghĩa là:

- user tắt links
- user bật links lại
- graph quay lại đúng subset edge type trước đó

## Workstream F. Lock interaction with selected-node mode

Hiện `useSigma` có nhánh selected-node emphasis cho edge.

Plan này phải khóa rõ:

- khi `all graph links` đang tắt
- user bấm chọn một node
- các edge neighbor của node đó **không được tự hiện lại**

Nghĩa là:

- selected-node mode không được override master edge toggle
- master edge toggle vẫn là check cao nhất trong `edgeReducer`

## Workstream G. Optional follow-up, not in this implementation

Không làm trong lượt này:

- hiển thị badge số lượng edges đang visible
- toolbar label text thay vì icon-only
- sync trạng thái nút với filter panel bằng status hint
- shortcut keyboard cho toggle links

Các ý này để sau.

## Refactor Safety Rule

Nếu thay đổi chỉ ở mức rất nhỏ, ví dụ:

- đổi label
- đổi tooltip
- thêm class hiển thị
- thêm prop rất cục bộ không làm đổi cấu trúc file

thì có thể patch trực tiếp.

Nhưng nếu đụng vào:

- cấu trúc file
- logic trung tâm của reducer
- state topology
- wiring lớn giữa nhiều component/hook

thì bắt buộc:

1. tạo file mới song song trước
2. wire file mới vào từng bước
3. chỉ xóa file cũ khi file mới chạy đúng

Tuy nhiên, với feature này, thay đổi dự kiến chủ yếu là:

- state nhỏ trong `graph.tsx`
- threading trong `useAppState.local-runtime.tsx`
- wiring trong `GraphCanvas.tsx`
- reducer change trong `useSigma.ts`

Do đó:

- patch nhỏ ở `graph.tsx` hoặc `GraphCanvas.tsx` có thể sửa trực tiếp
- nhưng nếu `useSigma.ts` bị refactor lớn hoặc tách reducer logic, phải tạo file mới song song trước

## Validation

## 1. Typecheck

Phải chạy:

- `cd avmatrix-web && npx tsc -b --noEmit`

## 2. Unit / component coverage

Nên thêm test cho:

- graph state default = visible
- toggle đổi state đúng
- localStorage load/save đúng
- `edgeReducer` ẩn toàn bộ cạnh khi master toggle = false
- bật lại thì edge visibility quay về phụ thuộc `visibleEdgeTypes`

## 3. Manual validation in real web UI

Phải mở web UI thật và kiểm tra:

1. load repo
2. nhìn graph với links đang bật
3. bấm `all graph links` để tắt
4. xác nhận:
- node vẫn còn
- selection vẫn còn
- highlight node vẫn còn
- chỉ cạnh biến mất
5. bấm lại để bật
6. xác nhận cạnh quay lại
7. đổi edge type filter ở panel trái
8. tắt links rồi bật lại
9. xác nhận edge type filter cũ vẫn còn hiệu lực
10. tắt links
11. chọn một node trên graph
12. xác nhận node vẫn được chọn nhưng edge neighbor không tự hiện lại

## 4. Interaction checks with existing features

Phải kiểm tra riêng các case:

- query highlight
- process highlight
- AI-driven highlights
- selected node focus
- code panel đang mở

Mục tiêu:

- nút mới không phá các flow cũ

## Risks

### Risk 1. Naming ambiguity

Nếu không khóa semantics rõ, rất dễ code nhầm thành:

- toggle AI edges
- toggle selected-node neighborhood
- toggle edge-type filters

Plan này chốt rõ: đây là **master edge visibility toggle**.

### Risk 2. Hiding edges may look like graph is broken

Nếu UI state không đủ rõ, người dùng có thể tưởng graph lỗi.

Vì vậy:

- tooltip
- aria
- visual pressed/unpressed state

phải rõ.

### Risk 3. Reducer precedence

Nếu chèn logic sai thứ tự trong `edgeReducer`, highlight logic có thể override toggle.

Vì vậy master check phải ở đầu reducer.

## Done Means

Feature được coi là xong khi:

- nút `all graph links` không còn là placeholder
- bấm tắt thì mọi cạnh trong graph canvas biến mất
- node vẫn giữ nguyên
- bấm bật lại thì cạnh quay lại theo filter loại cạnh hiện có
- trạng thái được nhớ lại qua reload
- typecheck pass
- manual walkthrough pass
