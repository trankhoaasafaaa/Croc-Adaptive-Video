# Báo cáo triển khai và thực nghiệm

## Baseline và bản đo

| Mục | Giá trị |
| --- | --- |
| Tag nguồn | `v11.6.6` |
| Commit upstream | `440ed73844fd02a8185e6e6d2d4cd772388587dc` |
| Module và Go | `github.com/schollz/croc/v11`, `go 1.27.3` trong `go.mod` |
| Môi trường chính | Linux x86-64, Go 1.27.3, Python 3.12.14, FFmpeg/FFprobe 7.1.5 |
| Commit tính năng | `7089031aaa0ada88a018c7e99d98e263e55d7c11` |
| Commit code ở thời điểm build đo | `6e5b933cfa7ff5c8b05f9d0541e88b8f21d5194e` |

Croc gốc đã build và gửi file qua relay loopback trước giai đoạn tích hợp. Bản feature đã build trên Linux, chạy test các package liên quan và test race có mục tiêu. Windows amd64 đã build chéo. Git có mốc baseline, nhánh feature và các commit theo nhóm thay đổi. Các bước và bằng chứng có trong [hướng dẫn kiểm thử](TESTING.md).

## Phương pháp

FFmpeg sinh ba bản từ một nguồn tổng hợp 60 giây. FFprobe xác nhận video stream, chiều cao, thời lượng và kích thước. Ba bản lần lượt có **12.023.775 B**, **3.384.051 B** và **717.803 B**. Profile pilot 25, 5 và 1,2 Mb/s chọn đủ ba mức nên được giữ nguyên cho ma trận chính.

Relay luôn là binary gốc và có một cổng control, một cổng data. Proxy TCP trong loopback điều tiết từng block tối đa 16 KiB từ sender đến relay trên cả hai cổng. Receiver nối trực tiếp đến relay. Không dùng cờ throttle ứng dụng Croc. Đối chiếu trên bản 1080p cố định, goodput tính từ byte file và thời gian pha nhận lần lượt xấp xỉ **23,5**, **4,9** và **1,19 Mb/s**. Chi phí relay, lập lịch và xác minh vẫn xuất hiện trong số đo. Các lượt chạy với tối đa 10 worker có relay và proxy riêng trên cùng máy.

Đồng hồ monotonic của harness ghi thời điểm có xác nhận nhận file và thời điểm receiver thoát. **Pha nhận** đo từ dòng `Receiving (...)` đến lúc receiver thoát với hash khớp. Đây là mốc giống nhau cho ba phương án, gồm khâu cài đặt data ở receiver và xác minh. **Tổng** của adaptive lấy thời điểm in dòng probe trừ thời lượng probe làm mốc bắt đầu, gồm thêm thời gian probe, chuẩn bị file và manifest trước pha nhận. Với hai phương án cố định, tổng bằng pha nhận. Script không tính thời gian chờ peer trước mốc đo. **Lệch ETA** bằng pha nhận thực tế trừ ETA mà sender công bố. Kịp hạn yêu cầu cả hai CLI thành công, tên và hash chính xác, tổng thời gian ≤ 20 giây. Mẫu lỗi vẫn tính vào mẫu số kịp hạn. Trung bình và độ lệch chuẩn mẫu của thời gian hoàn thành dùng các lượt truyền thành công.

## Kết quả ma trận chính

| Mạng | Cách gửi | Lượt | Payload hợp lệ | Kịp hạn | Chất lượng đã nhận | Tổng giây, TB ± SD | Pha nhận giây, TB ± SD | Probe giây, TB ± SD | Lệch ETA giây, TB ± SD |
| --- | --- | ---: | ---: | ---: | --- | ---: | ---: | ---: | ---: | ---: |
| fast | fixed1080 | 5 | 5 | 5/5 | 1080p × 5 | 4.09 ± 0.05 | 4.09 ± 0.05 | — | — |
| fast | fixed144 | 5 | 5 | 5/5 | 144p × 5 | 0.30 ± 0.00 | 0.30 ± 0.00 | — | — |
| fast | adaptive | 5 | 5 | 5/5 | 1080p × 5 | 4.23 ± 0.06 | 4.07 ± 0.03 | 0.12 ± 0.02 | -2.87 ± 0.93 |
| medium | fixed1080 | 5 | 5 | 5/5 | 1080p × 5 | 19.54 ± 0.08 | 19.54 ± 0.08 | — | — |
| medium | fixed144 | 5 | 5 | 5/5 | 144p × 5 | 1.19 ± 0.02 | 1.19 ± 0.02 | — | — |
| medium | adaptive | 5 | 5 | 5/5 | 360p × 5 | 6.01 ± 0.04 | 5.52 ± 0.05 | 0.44 ± 0.02 | -1.60 ± 0.31 |
| slow | fixed1080 | 5 | 5 | 0/5 | 1080p × 5 | 80.73 ± 0.03 | 80.73 ± 0.03 | — | — |
| slow | fixed144 | 5 | 5 | 5/5 | 144p × 5 | 4.83 ± 0.02 | 4.83 ± 0.02 | — | — |
| slow | adaptive | 5 | 5 | 5/5 | 144p × 5 | 6.68 ± 0.03 | 4.86 ± 0.03 | 1.77 ± 0.01 | -1.20 ± 0.03 |

Toàn bộ **45/45** lượt hoàn thành và có SHA-256 khớp. Với deadline 20 giây, adaptive kịp **15/15** lượt, gửi cố định 1080p kịp **10/15** lượt và gửi cố định 144p kịp **15/15** lượt.

- Ở 25 Mb/s, adaptive chọn 1080p cả năm lượt. Tổng thời gian tăng khoảng 0,14 giây so với luôn gửi 1080p.
- Ở 5 Mb/s, 1080p cố định vẫn kịp trong cả năm lượt với trung bình 19,54 giây. Hệ số dự phòng làm adaptive chọn 360p. Kết quả là thời gian nhận giảm còn 6,01 giây và độ phân giải giảm so với 1080p.
- Ở 1,2 Mb/s, 1080p cố định mất khoảng 80,73 giây. Adaptive chọn 144p và hoàn thành trong 6,68 giây. Gửi 144p cố định đạt 4,83 giây với cùng chất lượng.
- Ca tụt tốc độ trong lúc gửi giữ nguyên bản 360p và vượt deadline. Dòng `slowdown` chứa ETA trước khi tụt và thời gian nhận thực tế.

[CSV ma trận chính](adaptive-evidence/benchmark.csv) lưu đủ 45 lượt từ binary có sửa lỗi trạng thái hủy. Một lượt ma trận tiền kiểm trên bản ngay trước đó có hai timeout lúc thiết lập, trước payload. [CSV tiền kiểm](adaptive-evidence/pre-final-benchmark.csv) giữ nguyên hai dòng timeout. [CSV pilot](adaptive-evidence/pilot.csv) ghi ba lượt dùng để kiểm tra profile.

## Chức năng và giới hạn

[CSV tích hợp](adaptive-evidence/integration.csv) ghi 11 ca, gồm hai cách từ chối nhận trễ, chấp nhận trễ có ngưỡng, ngưỡng không đạt, không có bản kịp, peer cũ, gửi thường giữa hai phiên bản, probe timeout, mất kết nối, Ctrl-C và mạng tụt. Ở các ca từ chối, cổng data chỉ chuyển byte mở kênh và thư mục nhận còn rỗng.

Kết quả áp dụng cho máy Linux, loopback, proxy có một lane và nguồn MP4 tổng hợp. Windows native, LAN ba thiết bị và các relay Internet chưa kiểm chứng trong lần triển khai này. Phiên adaptive dùng một phép đo tại thời điểm đầu và không đổi bản khi băng thông thay đổi. Phiên gián đoạn báo lỗi và cần bắt đầu lại. Chưa có repo đích được chỉ định nên bộ nộp sử dụng source, patch và lịch sử nhánh cục bộ.
