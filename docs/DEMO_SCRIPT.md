# Kịch bản demo 6 phút 30 giây

| Thời gian | Thành viên | Nội dung nói và thao tác |
| --- | --- | --- |
| 0:00–0:45 | 1 | Một video có ba bản. Nếu luôn gửi 1080p sẽ dễ trễ trên mạng chậm. Nếu luôn gửi 144p sẽ bỏ phí chất lượng trên mạng nhanh. Mục tiêu là tự chọn đúng một file với deadline và ngưỡng của người nhận. |
| 0:45–1:35 | 1 | Mở sơ đồ kiến trúc. Chỉ PAKE, probe 256 KiB, ACK, bước chọn, rồi manifest một tệp. Cho thấy tag v11.6.6 và relay Croc nguyên bản. |
| 1:35–2:25 | 2 | Chạy `--suite demo`. Trong lúc chờ, mở CSV fixture để chỉ ra kích thước thật. Đọc các dòng 1080p, 360p, 144p khi script báo xong. |
| 2:25–3:20 | 2 | Dùng kết quả profile 5 Mb/s, đối chiếu deadline 20 giây, probe, ETA và SHA-256. Mở các dòng 1080p cố định, 144p cố định và adaptive trong CSV benchmark. |
| 3:20–4:35 | 3 | Trình chiếu lệnh receiver có ngưỡng 360p. Dùng ca `late-accept` và `late-refuse` trong script integration. Chỉ lời nhắc trễ, exit code, thư mục đích và số byte trên cổng data. |
| 4:35–5:25 | 3 | Giải thích ca giới hạn từ 5 xuống 1,2 Mb/s sau khi bắt đầu truyền. Bản đã chọn giữ nguyên. Thời gian hoàn thành có thể vượt deadline. |
| 5:25–6:30 | Cả nhóm | Cho xem checklist nghiệm thu, bản patch và lệnh build. Tóm tắt đánh đổi bằng số đo thực. Nếu LAN trục trặc, bật relay và cả hai CLI trên một máy. |

Chuẩn bị sẵn binary, fixture, file CSV và hai terminal đã mở trước phần trình bày. Dùng mã phiên dùng một lần và đóng relay khi demo xong.
