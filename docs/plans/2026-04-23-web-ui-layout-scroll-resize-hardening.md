# Web UI Layout / Scroll / Resize Hardening Plan

Date: 2026-04-23
Scope: `avmatrix-web/` only
Status: Proposed

## Goal

Sửa nhóm vấn đề UX nền của web UI AVmatrix:

- resize cửa sổ làm layout bị chật, tràn, hoặc mất vùng hiển thị
- scroll chuột không mượt vì shell và các panel đang chia vùng cuộn không hợp lý
- left dashboard đang hơi rộng
- right/chat panel chưa có khung hình tốt và chưa kéo rộng hẹp được

Plan này chỉ nói về cấu trúc layout và hành vi panel. Không đổi graph engine, session runtime, hay logic dữ liệu.

## Current Problems Confirmed In Code

### 1. Shell tổng đang bị khóa quá cứng theo viewport

Hiện tại exploring shell dùng:

- `h-screen overflow-hidden` ở `avmatrix-web/src/App.tsx`
- `main` là flex row full height

Hệ quả:

- khi cửa sổ bị kéo nhỏ, shell ngoài cùng không cho fallback scroll toàn cục
- người dùng bị kẹt trong nhiều vùng cuộn cục bộ
- cảm giác là UI bị tràn hoặc “mất màn hình”

### 2. Left dashboard đang quá cứng và hơi rộng

Hiện tại file tree panel:

- mở ra ở `w-64`
- thu gọn thì còn `w-12`
- không có resize
- không có mức trung gian

Hệ quả:

- panel trái chiếm ngang hơi nhiều
- không linh hoạt theo màn hình
- không có cách giảm nhẹ xuống khoảng 80% như mong muốn

### 3. Right/chat panel đang là vách cứng, chưa phải một panel UI hoàn chỉnh

Hiện tại right panel:

- `w-[40%] max-w-[600px] min-w-[400px]`
- chỉ có `border-l`
- không có bo góc ngoài
- không có resize handle

Hệ quả:

- panel chat nhìn như chèn cứng vào mép phải
- không ăn theo khung trình duyệt
- không thể kéo rộng hẹp theo nhu cầu giao tiếp

### 4. Nhiều vùng scroll đang chồng nhau

Hiện tại:

- shell ngoài cùng khóa overflow
- file tree có scroll riêng
- chat transcript có scroll riêng
- filters có scroll riêng
- code panel cũng có scroll và resize riêng

Hệ quả:

- wheel bị “giật ownership” giữa các panel
- resize cửa sổ làm trải nghiệm càng khó chịu hơn
- cảm giác tổng thể là không mượt

### 5. Code panel bên trái overlay đã có resize, nhưng chat panel thì chưa

Điều này làm UX thiếu nhất quán:

- code references panel có width state và drag resize
- right/chat panel không có cơ chế tương tự

Hệ quả:

- cùng là panel nghiệp vụ nhưng hành vi resize không đồng bộ

### 6. Một số chi tiết ngang đang bị hard-code thêm

Ví dụ:

- status bar có spacer `w-[220px] shrink-0`

Hệ quả:

- ở màn hình hẹp, các hard-coded spacer làm layout bí hơn mức cần thiết

## Design Direction

### 1. Shell phải co giãn theo viewport thực, không khóa chết

Mục tiêu:

- thay mô hình `h-screen + overflow-hidden` bằng cấu trúc chịu resize tốt hơn
- dùng `min-h-dvh` hoặc cấu trúc tương đương
- cho phép vùng cần cuộn được cuộn đúng chỗ
- tránh mất nội dung khi cửa sổ bị kéo nhỏ

### 2. Left dashboard giảm mặc định xuống khoảng 80% hiện tại

Mục tiêu:

- giảm width mặc định từ mức hiện tại xuống khoảng 80%
- vẫn giữ collapse mode
- ưu tiên width nhỏ hơn trước, chưa cần resize ngay nếu chưa thật sự cần

### 3. Right/chat panel phải thành panel thật, không phải vách cắt ngang

Mục tiêu:

- panel phải có outer frame rõ hơn
- bo góc ngoài hợp logic với shell trình duyệt
- nhìn như một khối UI hoàn chỉnh

### 4. Right/chat panel phải kéo rộng hẹp được

Mục tiêu:

- thêm resize handle cho right panel
- có `min`, `default`, `max` width rõ ràng
- lưu width vào local storage
- resize mượt và không làm vỡ graph center area

### 5. Workspace trung tâm phải luôn là vùng ưu tiên giữ lại

Mục tiêu:

- graph workspace không bị bóp quá mức khi hai panel cùng hiện
- panel phụ phải tôn trọng giới hạn của màn hình
- khi cần, panel phải co trước khi workspace bị phá

### 6. Scroll ownership phải rõ ràng

Mục tiêu:

- shell tổng không chặn vô lý
- từng panel cuộn đúng vùng của nó
- khi nội dung quá dài, người dùng vẫn luôn có đường cuộn hợp lý

## Implementation Workstreams

### Workstream A. Shell and viewport model

Làm:

- chỉnh root shell và main layout để chịu resize tốt hơn
- bỏ các chỗ khóa viewport quá cứng
- rà các container đang cần `min-w-0`, `min-h-0`, `overflow-auto`, `overflow-hidden`

Kết quả mong muốn:

- kéo nhỏ cửa sổ vẫn dùng được
- không bị mất panel hoặc mất phần chính

### Workstream B. Left panel width pass

Làm:

- giảm width mặc định của file tree panel xuống khoảng 80% hiện tại
- rà lại spacing, text truncation, tab header trong left rail

Kết quả mong muốn:

- bớt chiếm ngang
- vẫn đọc được tree
- không làm panel trái trở nên quá chật

### Workstream C. Right panel structural redesign

Làm:

- chuyển right panel từ “edge slab” sang panel có khung rõ ràng
- thêm radius phù hợp ở outer corners
- rà lại header, transcript body, composer footer để panel nhìn liền khối

Kết quả mong muốn:

- panel chat nhìn hợp với khung trình duyệt
- cảm giác hoàn thiện hơn

### Workstream D. Right panel resize system

Làm:

- thêm drag handle cho panel phải
- thêm width state
- thêm local storage key cho right panel width
- đặt min/default/max width thực tế
- kiểm tra interaction khi panel mở/đóng, đổi tab chat/processes, và khi code panel cũng đang mở

Kết quả mong muốn:

- người dùng kéo rộng hẹp được
- UI không giật
- workspace giữa vẫn an toàn

### Workstream E. Scroll hardening

Làm:

- rà lại file tree, chat transcript, processes list, code references, settings modal
- đảm bảo mỗi vùng chỉ cuộn khi nó thật sự là vùng nội dung
- loại bỏ cảm giác “scroll bị mắc kẹt”

Kết quả mong muốn:

- wheel behavior mượt hơn
- resize xong vẫn cuộn đúng

### Workstream F. Hard-coded width cleanup

Làm:

- bỏ hoặc làm mềm các spacer/width cứng không cần thiết
- rà status bar và các row header dễ vỡ trên màn hình hẹp

Kết quả mong muốn:

- shell bớt bí ngang
- layout đỡ gãy khi thu nhỏ

## Safety Rule For Refactors

Khi refactor file lớn:

- không đập trực tiếp file cũ nếu thay đổi cấu trúc lớn
- tạo file mới song song
- wire file mới vào từng bước
- validate xong mới bỏ file cũ

Mục tiêu:

- không mất logic của file cũ
- khi có lỗi còn đường quay lại rõ ràng

## Validation

### Manual validation is mandatory

Phải mở web UI thật và tự thao tác tay:

- landing
- chọn repo
- repo dropdown
- analyze flow
- settings
- file tree
- graph workspace
- code references panel
- desk chat
- processes
- process modal
- close/open panel
- resize cửa sổ trình duyệt nhiều mức
- kéo rộng hẹp right panel

### Resize matrix

Phải thử ít nhất:

- desktop rộng
- desktop trung bình
- chiều ngang hẹp
- chiều dọc thấp
- tình huống file tree + right panel + code panel cùng hiện

### Regression checks

Phải kiểm tra:

- chat vẫn gửi được
- process modal vẫn mở đúng
- code references panel vẫn resize đúng
- graph center không bị mất vùng tương tác
- header và status bar không gãy layout

### Automated validation

Sau khi sửa xong:

- `cd avmatrix-web && npx tsc -b --noEmit`
- `cd avmatrix-web && npm test`

Nếu có automation cho resize/panel behavior thì chạy thêm, nhưng manual walkthrough vẫn là bắt buộc.

## Done When

Đợt này chỉ được coi là xong khi:

- left dashboard nhỏ hơn rõ rệt và hợp lý hơn
- right/chat panel có hình khối tốt hơn
- right/chat panel kéo rộng hẹp được
- resize cửa sổ không còn gây tràn/mất vùng nhìn nghiêm trọng
- scroll chuột trong các panel chính cho cảm giác liền mạch hơn
- manual walkthrough qua toàn bộ vòng đời UI không còn lỗi layout đáng kể
