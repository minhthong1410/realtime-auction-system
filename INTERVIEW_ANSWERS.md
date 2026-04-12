# Interview Answers

## 1. Bạn thích Go ở điểm gì? So sánh với PHP?

Mình chọn Go làm ngôn ngữ chính cho backend vì mấy điểm sau:

**Tại sao thích Go:**

- **Concurrency là first-class citizen.** Goroutine + channel giúp xử lý concurrent rất tự nhiên. Ví dụ trong auction system, mình chạy WebSocket hub, auction closer worker, Redis subscriber — tất cả chỉ cần `go func()`. Trong production, Kafka consumer chạy nhiều partition song song, mỗi partition 1 goroutine, không cần lo thread pool hay async framework.

- **Single binary, zero runtime dependency.** `go build` ra 1 file, deploy lên server chạy luôn. Docker image chỉ cần `FROM alpine` + copy binary. Không cần runtime, interpreter, hay package manager trên production.

- **Type system + compile-time safety.** Kết hợp với sqlc (generate Go code từ SQL), cả query lẫn business logic đều type-safe lúc compile. Refactor lớn cũng tự tin vì compiler bắt hết.

- **Standard library đủ mạnh.** `net/http`, `encoding/json`, `crypto`, `database/sql` — viết production API mà gần như không cần dependency ngoài. Gin chỉ là thin wrapper cho routing + middleware.

- **Performance predictable.** Compiled, statically typed, GC tuned cho low-latency. Trong hệ thống gaming handle hàng triệu transaction/ngày, p99 response time rất stable.

- **Đơn giản, ít magic.** Không có decorator, annotation, DI container, hay hidden behavior. Code đọc từ trên xuống dưới là hiểu flow. Review code dễ, onboard người mới nhanh.

**So sánh với PHP:**

| | Go | PHP |
|---|---|---|
| **Execution** | Compiled binary | Interpreted (cần php-fpm + web server) |
| **Concurrency** | Goroutine native, hàng triệu concurrent | Mỗi request 1 process, cần Swoole/ReactPHP cho async |
| **Type safety** | Compile-time, strict | Runtime type hints (PHP 8+), vẫn loose |
| **Deploy** | Single binary | Cần PHP runtime + Composer + nginx/Apache |
| **WebSocket/long-running** | Native support | Không phải thế mạnh, share-nothing architecture |
| **Ecosystem** | Nhỏ hơn, tập trung backend/infra | Rất lớn cho web (Laravel, WordPress, Magento) |
| **Dev speed (CRUD)** | Chậm hơn, boilerplate nhiều | Nhanh hơn nhờ Laravel scaffold, Eloquent ORM |
| **Learning curve** | Trung bình, syntax đơn giản nhưng cần hiểu concurrency | Thấp, entry barrier dễ |

**Tóm lại:** PHP mạnh ở web ecosystem và tốc độ phát triển CRUD. Go mạnh ở performance, concurrency, và hệ thống phức tạp. Project mình làm — gaming platform cần WebSocket real-time, Kafka consumer, background worker, concurrent bet processing — Go là lựa chọn phù hợp hơn.

---

## 2. Bạn thích giải quyết những vấn đề gì nhất trong backend?

Mình thích nhất 2 loại vấn đề:

### a) Data consistency & transaction safety

Đảm bảo tiền không mất, không dư, không race condition. Ví dụ cụ thể:

- Trong auction system: khi 2 người bid cùng lúc, phải dùng `SELECT FOR UPDATE` lock row, deduct balance người bid mới, refund người bid cũ — tất cả trong 1 transaction. Nếu thiếu lock, 2 bid cùng thắng hoặc balance sai.

- Trong production: từng phát hiện bug `placeBetParlay` **cộng** tiền vào balance thay vì **trừ** (nghĩa là user bet mà lại được thêm tiền). Bug này chạy trên production gần 2 ngày trước khi phát hiện. Phải tính toán lại toàn bộ balance bị ảnh hưởng và tạo adjustment.

- Cũng từng audit phát hiện hàm `BetNSettleTxn` ghi sai `balance_after` trong lịch sử — tích lũy +3.177 tỷ sai lệch trên 4.25 triệu records. Phải viết reconciliation formula phức tạp để verify từng ngày, walk-back từ balance hiện tại.

### b) Performance optimization

Từ timeout 30s xuống dưới 1s — cảm giác rất sướng. Ví dụ:

- BO player list dùng 4 CTE MATERIALIZED quét full bảng `payment_transaction` (hàng triệu row) mỗi request. Đổi sang LATERAL join + dùng bảng tổng hợp có sẵn → response < 1s.

- Game bet history bị duplicate vì JOIN bảng `game` có duplicate data ngoài schema expectation. Fix bằng `LEFT JOIN LATERAL ... LIMIT 1`.

- Xóa `COUNT(*) OVER()` (scan toàn bộ result set) → tách thành count query riêng.

Mình thích những vấn đề này vì nó đòi hỏi phải hiểu sâu — không chỉ code mà còn database, index, execution plan, data flow.

---

## 3. Bạn thấy vấn đề gì khó nhất trong backend?

**Debugging production data issues khi không có đủ observability.**

Khó nhất không phải là code logic phức tạp — mà là khi production data đã sai rồi, phải tìm ra chính xác sai ở đâu, từ bao giờ, ảnh hưởng bao nhiêu record, và cách fix mà không làm hỏng thêm.

Ví dụ thực tế: balance reconciliation audit. Mình phải:

1. Decode công thức tính balance từ 7+ source tables khác nhau, mỗi table dùng timestamp khác nhau (created_at vs updated_at), timezone khác nhau (UTC vs WIB)
2. Phát hiện 3 bugs chồng lên nhau: parlay cộng thay trừ, BetNSettleTxn ghi sai history, WithdrawNDepositTxn thiếu history record
3. Xác định 10 "dirty users" có balance không khớp với bất kỳ formula nào — vì code path update balance mà không ghi history
4. Restore 691K records từ backup database sau khi partial fix script chạy sai

Bài học: **observability phải đi trước feature**. Nếu mỗi thay đổi balance đều có audit trail đầy đủ từ đầu, debug sẽ đơn giản hơn rất nhiều. Và luôn phải có reconciliation check — không tin tưởng 100% vào application code.

Một vấn đề khó nữa là **database incident dưới áp lực production**. Từng gặp case GIN trigram index trên bảng write-heavy gây write amplification → connection pool cạn kiệt → cascade failure. Lúc đó phải vừa giữ bình tĩnh phân tích root cause, vừa hotfix nhanh dưới áp lực hệ thống đang down.

---

## 4. Tình huống tâm đắc, tự hào nhất?

**Balance reconciliation audit — biến chaos thành con số chính xác.**

Context: Hệ thống gaming platform, handle tiền thật. Một ngày phát hiện tổng balance users không khớp với expected từ các transaction. Sai lệch hàng tỷ.

Mình tự hào vì:

**1. Không panic, tiếp cận có hệ thống.**
Thay vì cố fix ngay, mình dành thời gian decode hoàn toàn công thức reconciliation — 10 component, mỗi cái có timestamp rule riêng, edge case riêng. Viết ra thành tài liệu chi tiết để ai cũng verify được.

**2. Phát hiện 3 bugs lồng nhau mà team không ai biết.**
- Parlay bug: bet mà cộng tiền (đã disable nhưng data vẫn sai)
- BetNSettleTxn: 4.25 triệu record ghi sai balance history, tích lũy +3.177 tỷ sai lệch
- WithdrawNDeposit: update balance nhưng không ghi history

Mỗi bug riêng lẻ đã khó, nhưng 3 cái chồng lên nhau khiến reconciliation formula phải xử lý từng case riêng.

**3. Phát minh "walk-back method" để verify.**
Thay vì tin vào formula, mình verify ngược: lấy balance hiện tại → trừ ngược các delta đã corrected → so với expected. Kết quả khớp hoàn toàn (sai lệch 0 cho 15 ngày liên tiếp, trừ 134k Joker bug đã biết).

**4. Data restoration có kiểm soát.**
Khi phát hiện partial fix script đã chạy sai trên production, mình restore 691K records từ backup — không restore toàn bộ bảng (sẽ mất data mới) mà chỉ UPDATE các records bị modified, giữ nguyên records mới.

**Kết quả:** Từ "không biết sai bao nhiêu" → xác định chính xác từng đồng, từng ngày, từng user. Team có thể tự tin report cho stakeholders với con số có thể verify.

Tình huống này dạy mình rằng: **kỹ năng quan trọng nhất của backend engineer không phải viết code nhanh — mà là khả năng phân tích data, tìm root cause, và đưa ra giải pháp có thể verify được.**
