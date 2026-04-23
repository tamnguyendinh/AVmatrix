# Graph Links Toggle vs Selected Context Plan

Date: 2026-04-23
Scope: `avmatrix-web/` only
Status: Proposed

## Goal

Sửa bug UX hiện tại:

- khi tắt `all graph links`
- rồi bấm chọn một file / function / node trên graph
- UI chỉ còn các điểm sáng liên quan
- nhưng không hiện lại các đường nối ngữ cảnh của node đang chọn

Điều này không đúng với UX mong muốn.

Mục tiêu của plan:

- giữ `all graph links` như công tắc cho **link nền của toàn graph**
- nhưng vẫn cho phép hiện **link ngữ cảnh của node đang chọn**
- tách rõ 2 lớp semantics này trong code

Plan này **supersede semantics** đã mô tả trong
[2026-04-23-web-ui-all-graph-links-toggle.md](/F:/AVmatrix-main/docs/plans/2026-04-23-web-ui-all-graph-links-toggle.md)
ở đúng một điểm:

- `all graph links` không còn được hiểu là kill switch cho mọi edge
- nó chỉ còn là toggle cho **ambient graph links**

Plan này chỉ xử lý frontend graph rendering.

Không đổi:

- backend
- MCP
- `/api/graph`
- graph index

## Non-Goals

Plan này không xử lý:

- đổi tên nút `all graph links`
- thêm nút mới
- đổi key persistence `avmatrix.graphLinksVisible`
- đổi backend contract
- đổi node highlight semantics
- đổi query/process/AI highlight semantics ngoài phạm vi edge visibility

## Problem Statement

Bug hiện tại không nằm ở backend.

Nó nằm ở việc code đang gộp chung 2 loại edge visibility:

1. `ambient graph links`
- các link nền của toàn graph khi người dùng đang nhìn toàn cảnh

2. `selected-node contextual links`
- các link phục vụ việc đọc quan hệ của node/file/hàm đang chọn

Hiện tại `all graph links` đang được triển khai như:

- master kill switch cho **mọi edge**

Vì vậy khi tắt nút này:

- edge nền biến mất
- edge ngữ cảnh của node đang chọn cũng biến mất

Kết quả:

- node vẫn sáng
- nhưng người dùng không còn thấy “nó nối với ai”

Đó là UX sai.

## Current Code Reality

### 1. `GraphCanvas` chỉ truyền một cờ duy nhất

Ở [GraphCanvas.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/GraphCanvas.tsx):

- nút `all graph links` điều khiển `areGraphLinksVisible`
- cờ này được truyền nguyên xuống `useSigma`

### 2. `shouldHideGraphEdge()` đang quá tuyệt đối

Ở [graph-links-visibility.ts](/F:/AVmatrix-main/avmatrix-web/src/lib/graph-links-visibility.ts):

- nếu `areGraphLinksVisible === false`
- thì helper trả `true`
- nghĩa là mọi edge đều bị ẩn

### 3. `edgeReducer` return quá sớm

Ở [useSigma.ts](/F:/AVmatrix-main/avmatrix-web/src/hooks/useSigma.ts):

- `edgeReducer` gọi `shouldHideGraphEdge()` ngay đầu
- nếu helper nói hide thì reducer `return` luôn

Trong khi nhánh:

- selected node
- connected edge emphasis

lại nằm phía sau.

Kết quả:

- selected-node context không còn cơ hội override

## Product Rules To Lock Before Coding

### Rule 1. `All graph links` chỉ điều khiển link nền

Nút `all graph links` phải được hiểu là:

- bật/tắt các edge của chế độ toàn cảnh graph

Không được hiểu là:

- bật/tắt mọi edge trong mọi ngữ cảnh

### Rule 2. Selected node luôn có quyền hiện context links của nó

Khi user chọn một node:

- các edge nối trực tiếp tới node đó phải có thể hiện ra
- ngay cả khi `all graph links` đang tắt

Đây là phần **contextual inspection**, không phải ambient graph chrome.

### Rule 2a. Selected context vẫn phải obey edge-type filters

Exception của selected-node context chỉ bypass:

- ambient graph toggle

Nó **không bypass**:

- `visibleEdgeTypes`

Nếu edge type đó đang bị filter off thì edge vẫn phải ẩn.

### Rule 3. Hai lớp semantics phải tách riêng

Từ giờ phải có phân biệt rõ:

- `areAmbientGraphLinksVisible`
- `selectedNodeContextEdgesVisible`

Trong implementation có thể không cần giữ đúng tên này, nhưng semantics phải tách.

### Rule 4. Không reset selection chỉ vì toggle links

Tắt `all graph links`:

- không được clear selection
- không được clear query
- không được clear process focus

### Rule 5. Selected context chỉ hiện edge liên quan trực tiếp

Khi ambient links đang off nhưng có node được chọn:

- chỉ edge nối trực tiếp với node đó được hiện
- không bật lại toàn bộ graph

### Rule 6. Chỉ selected-node context mới có exception

Các lớp sau **không** được tự động có quyền hiện edge khi ambient links off:

- query highlights
- process highlights
- AI-driven highlights
- blast-radius highlights

Những lớp này vẫn phải đi theo luật edge visibility hiện hành.

## Desired UX

### Case A. No selected node, all graph links ON

- graph hiện link nền bình thường

### Case B. No selected node, all graph links OFF

- graph chỉ hiện node
- không hiện edge nền

### Case C. Selected node, all graph links ON

- graph hiện toàn bộ edge theo filter
- edge liên quan tới node đang chọn được emphasis như hiện nay

### Case D. Selected node, all graph links OFF

- edge nền của graph vẫn tắt
- nhưng edge nối trực tiếp tới node đang chọn phải hiện
- các edge đó được dùng như lớp “inspection context”

Đây là case hiện đang bị hỏng.

## Design Direction

Không vá kiểu:

- “nếu selected thì bỏ qua toggle”

theo một if chắp vá duy nhất mà không đặt lại semantics.

Hướng bền vững là:

1. tách decision layer cho edge visibility
2. tính riêng:
- ambient visibility
- selected-context visibility
- edge-type visibility
- highlight emphasis
3. rồi mới apply style/render

## Proposed Implementation

## Workstream A. Split edge visibility semantics in helper layer

Helper hiện tại ở [graph-links-visibility.ts](/F:/AVmatrix-main/avmatrix-web/src/lib/graph-links-visibility.ts)
đang làm quá ít và quá thô.

Cần đổi helper thành lớp quyết định rõ hơn:

- edge này bị ẩn hoàn toàn hay không
- edge này được hiện vì ambient graph
- hay được hiện vì selected-node context

Khuyến nghị:

- tạo helper mới song song trong cùng file hoặc file mới riêng
- không dồn toàn bộ logic ngữ cảnh vào thẳng `useSigma.ts`

Ví dụ hướng API:

- `getGraphEdgeVisibilityMode(...)`
- trả về một mode kiểu:
  - `hidden`
  - `ambient`
  - `selected-context`

Không bắt buộc đúng tên này, nhưng nên có lớp quyết định tương đương.

## Workstream B. Keep the UI toggle as ambient-only state

State hiện tại `areGraphLinksVisible` nên được reinterpret rõ thành:

- `areAmbientGraphLinksVisible`

Nếu chưa muốn rename toàn bộ code ngay, ít nhất phải:

- ghi comment rõ semantics mới
- tránh tiếp tục dùng nó như “mọi edge visible”

Nếu rename:

- phải rename xuyên suốt có kiểm soát

Nếu không rename trong lượt này:

- phải có helper layer bảo vệ semantics

## Workstream C. Update `edgeReducer` precedence

`edgeReducer` hiện phải được tách thành thứ tự đúng:

1. kiểm tra selected-node context trước
2. nếu edge nối trực tiếp tới selected node:
- edge này có quyền hiện dù ambient links đang off
- nhưng vẫn phải obey `visibleEdgeTypes`
3. nếu không thuộc selected context:
- mới áp dụng master ambient toggle
4. sau đó mới áp edge type filter / highlight styling tương ứng

Điểm cần khóa:

- selected-node context là một exception hợp lệ
- highlight/query/process không tự động có cùng quyền exception này

Tức là:

- selected node được exception
- ambient graph không được exception

## Workstream D. Decide styling for selected-context when ambient is off

Khi ambient links off nhưng selected context vẫn hiện:

- edge selected-context không nên bị hiểu nhầm là ambient graph vừa bật lại

Nên có 2 hướng:

1. giữ style selected-edge như hiện tại
2. hoặc cho style selected-context rõ hơn một chút

Khuyến nghị:

- giữ style selected-edge hiện tại trước
- không mở thêm bài toán visual mới nếu chưa cần

## Workstream E. Add tests for the broken UX case

Đây là phần bắt buộc.

Phải có test khóa case:

1. ambient links off
2. selected node exists
3. edge nối trực tiếp tới selected node vẫn visible
4. edge không liên quan vẫn hidden

Nếu không có test này, bug sẽ quay lại rất dễ.

## Refactor Safety Rule

Feature này chạm vào logic trung tâm của `useSigma.ts`.

Vì vậy:

- nếu chỉ patch rất nhỏ ở precedence, có thể sửa trực tiếp
- nhưng nếu helper logic phình ra hoặc reducer bắt đầu khó đọc, phải tạo file/helper mới song song trước

Không được tiếp tục dồn nhiều lớp semantics vào một nhánh `if` chắp vá trong `useSigma.ts`.

## Validation

## 1. Typecheck

Phải chạy:

- `cd avmatrix-web && npx tsc -b --noEmit`

## 2. Unit tests

Phải có ít nhất:

- test helper visibility mode
- test case ambient off + selected node vẫn hiện direct context edge
- test edge không liên quan vẫn hidden
- test case ambient off + selected node + edge type bị filter off => edge vẫn hidden
- test case ambient off + selected node + bỏ chọn => context edges biến mất lại

## 3. Manual validation

Phải mở web UI thật và kiểm tra:

1. load repo
2. tắt `all graph links`
3. xác nhận graph chỉ còn node
4. bấm chọn một file/hàm/node
5. xác nhận:
- node đang chọn vẫn nổi bật
- các edge nối trực tiếp tới nó hiện ra
- edge không liên quan vẫn không hiện
6. khi selected node có một edge type đang bị filter off:
- xác nhận edge đó vẫn không hiện
7. bỏ chọn node
8. xác nhận các edge context biến mất lại

## Risks

### Risk 1. Sửa nhanh nhưng không tách semantics

Nếu chỉ thêm một exception `if (currentSelected)` chắp vá mà không đặt lại helper layer:

- code sẽ lại khó đọc
- lần sau query/process highlight rất dễ đạp lên nhau

### Risk 2. Chọn exception quá rộng

Nếu cho mọi loại highlight đều override ambient toggle:

- graph sẽ lại rối
- nút `all graph links` mất ý nghĩa

### Risk 3. Rename quá lớn không cần thiết

Rename toàn bộ `areGraphLinksVisible` xuyên repo nếu không kiểm soát có thể tạo diff rộng hơn cần thiết.

Nên ưu tiên:

- khóa semantics bằng helper trước
- chỉ rename nếu thật sự mang lại clarity đủ lớn

## Done Means

Feature được coi là xong khi:

- tắt `all graph links` không còn giết luôn selected-node context links
- chọn node khi ambient links off vẫn thấy direct context edges
- edge không liên quan vẫn ẩn
- typecheck pass
- test khóa regression pass
- manual UX pass
