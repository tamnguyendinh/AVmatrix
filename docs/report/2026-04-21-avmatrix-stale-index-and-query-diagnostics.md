# AVmatrix Report — Stale Index Snapshot and Weak Query Diagnostics

Date: 2026-04-21
Status: open
Priority: high

## Summary

Trong một phiên làm việc với repo `Website`, agent trả lời sai hướng khi dùng AVmatrix:

- kết quả `query(...)` gần như rỗng hoặc không hữu ích
- agent phải tự suy luận nguyên nhân
- sau đó fallback sang đọc source/spec trực tiếp

Sau khi kiểm tra, vấn đề không nằm hoàn toàn ở agent. Gốc vấn đề là AVmatrix đang dùng snapshot index cũ và query path hiện chưa chẩn đoán trạng thái đủ rõ cho agent.

## What Happened

Agent phản hồi theo kiểu:

- graph không cho kết quả hữu ích
- có thể index stale
- embeddings đang tắt
- chuyển sang đọc source/spec trực tiếp

Kết luận đó chỉ đúng một phần, nhưng vẫn phản ánh đúng một vấn đề thật của AVmatrix.

## Verified Facts

### 1. Repo `Website` đang stale thật

Registry/API của AVmatrix đang báo repo `Website` được index ở commit:

- `b4c046d401119e30fc03b19df7ab74d6733953e4`

Trong khi `HEAD` hiện tại của repo `F:\Website` là:

- `56f4c140233faf427e468a98cc8dfe24b646b273`

CLI cũng xác nhận:

- `Status: stale (re-run avmatrix analyze)`

Nghĩa là AVmatrix đang dùng snapshot cũ để trả kết quả.

### 2. Embeddings hiện đang tắt

`/api/repos` trả về:

- `embeddings: 0`

Nghĩa là semantic search hiện không có dữ liệu vector để hỗ trợ query.

### 3. Nhưng query path vẫn yếu hơn mức chấp nhận được

Ngay cả khi đã biết index stale, query hiện trả rỗng quá im lặng.

Các query thử trực tiếp trên `Website`:

- `query "login flow"`
- `query "auth login page"`
- `query "PublicLoginPage"`

đều trả:

- `processes: []`
- `process_symbols: []`
- `definitions: []`

Trong khi source hiện tại vẫn có symbol/thành phần liên quan, ví dụ:

- `F:\Website\app\(public)\auth\login\page.tsx`
- `PublicLoginPage`

Nghĩa là AVmatrix hiện chưa đưa ra được tín hiệu đủ rõ cho agent để biết chính xác:

- index đang stale tới mức nào
- BM25/FTS có đang match thất bại hay không
- semantic search đang bị vô hiệu vì không có embeddings
- nên re-analyze trước hay nên chuyển sang tool/path khác

## Root Cause

Nguyên nhân gốc là tổ hợp của 2 điểm:

1. AVmatrix đang dùng snapshot cũ vì repo chưa được re-analyze đúng lúc.
2. Query/retrieval diagnostics của AVmatrix chưa đủ tốt, nên khi kết quả yếu hoặc rỗng, tool không nói rõ nguyên nhân cho agent.

## Why This Matters

Nếu AVmatrix truy xuất đủ tốt và chẩn đoán rõ, agent sẽ tự dùng tool chính xác hơn.

Hiện tại, khi query path trả rỗng hoặc quá yếu:

- agent phải tự suy diễn
- dễ đọc nhầm tình trạng repo
- dễ fallback sang source/spec thủ công
- làm giảm giá trị thật của AVmatrix như một code-intelligence layer

Vấn đề này không nên được xem là “agent dùng sai hoàn toàn”. Đây là một hạn chế thật của AVmatrix hiện tại.

## Concrete Tooling Gaps

Các lỗ hổng hiện tại cần xử lý sau:

1. Query path không surfaced rõ trạng thái stale của repo ngay trong kết quả.
2. Query path không surfaced rõ semantic search đang tắt vì `embeddings = 0`.
3. BM25/FTS failure hoặc no-hit path đang quá im lặng.
4. Agent không được điều hướng đủ rõ:
   - re-analyze trước
   - sau đó query lại
   - hoặc chuyển sang `context(...)` nếu đã có symbol cụ thể

## Expected Future Fix Direction

Chưa xử lý trong batch này. Sẽ xử lý sau theo hướng:

1. Làm rõ diagnostics trong `query(...)`:
   - stale
   - embeddings off
   - BM25 no-hit / FTS unavailable

2. Ưu tiên thông báo actionable hơn cho agent:
   - `re-run avmatrix analyze`
   - query đang dùng snapshot cũ
   - current retrieval mode là exact/BM25-only

3. Cải thiện retrieval quality để khi index còn hợp lệ thì query các cụm phổ biến như:
   - `login flow`
   - `auth login page`
   - `PublicLoginPage`
   phải cho ra kết quả hữu ích hơn

## Decision

Vấn đề này được ghi nhận là lỗi/thiếu sót thật của AVmatrix.

Không sửa ngay trong batch hiện tại.

Sẽ xử lý sau trong một plan/fix batch riêng.
