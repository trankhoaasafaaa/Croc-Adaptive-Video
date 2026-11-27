# Croc Adaptive Video

Croc Adaptive Video tự chọn chất lượng video trước khi gửi. Với Croc gốc, người gửi chọn sẵn một tệp. Với bản của nhóm, người gửi cung cấp nhiều bản MP4 của cùng một video và đặt thời hạn dự kiến. Sau khi hai CLI xác thực, chương trình đo tốc độ truyền qua relay và chọn bản có độ phân giải cao nhất dự kiến kịp thời hạn và đạt ngưỡng chất lượng người nhận đặt. Chương trình gửi đúng một tệp MP4 đã chọn.

Hai đầu gửi và nhận dùng CLI của dự án. Relay dùng cơ chế Croc sẵn có. Chế độ adaptive bật khi người gửi khai báo các cờ tương ứng. Lệnh gửi tệp thông thường vẫn dùng luồng Croc.

## Demo trên một máy Linux hoặc Windows qua WSL

Chuẩn bị Git, Go 1.27.3, Python 3.11 trở lên, FFmpeg có encoder `libx264` và FFprobe. Trên Windows, chạy các lệnh dưới đây trong terminal WSL. Lần build đầu có thể cần Internet để tải module Go.

```bash
git clone https://github.com/trankhoaasafaaa/Croc-Adaptive-Video.git
cd Croc-Adaptive-Video
bash scripts/adaptive-video/run-demo.sh
```

Script tự build CLI, tạo ba bản của cùng một video 60 giây và chạy relay cùng hai CLI trên máy. Proxy cục bộ giới hạn băng thông ở ba mức. Cả ba lượt dùng deadline 20 giây.

| Băng thông giới hạn | Bản dự kiến được chọn | Tệp nhận được |
| --- | --- | --- |
| 25 Mb/s | 1080p | `video-1080.mp4` |
| 5 Mb/s | 360p | `video-360.mp4` |
| 1,2 Mb/s | 144p | `video-144.mp4` |

Khi trình bày, chỉ vào băng thông, chất lượng được chọn, ETA và thời gian nhận thực tế trên từng dòng kết quả. Mỗi lượt thành công nhận một tệp và xác minh SHA-256. Mở thư mục `received` trong đường dẫn in ở cuối chương trình để phát các tệp đã nhận. Đây là demo loopback có giới hạn băng thông, nên các số liệu minh họa quyết định trong điều kiện thử này.

## Cờ adaptive

| Cờ | Đặt ở đâu | Hành vi |
| --- | --- | --- |
| `--video-deadline 20s` | Sau `send` | Đặt ngân sách thời gian tính từ lúc bắt đầu probe. |
| `--video-variant '360=video 360.mp4'` | Sau `send`, lặp 2–16 lần | Khai báo chiều cao và đường dẫn đến bản MP4 đã chuẩn bị. |
| `--video-min-height 360` | Cờ global ở CLI nhận | Chỉ chấp nhận đề xuất từ 360p trở lên. Nếu có bản đạt ngưỡng nhưng tất cả dự kiến trễ, người nhận phải nhập `y` hoặc `yes` để tiếp tục. |

ETA dựa trên tốc độ đo ở đầu phiên. Nếu băng thông thay đổi trong lúc gửi, thời gian nhận thực tế có thể khác ETA. Chất lượng được chọn một lần trước khi truyền tệp.

- [Thiết lập thủ công, video riêng và chạy qua LAN](docs/SETUP_GUIDE.md)
- [Kiến trúc, probe và quy tắc chọn bản](docs/ARCHITECTURE.md)
- [Kiểm thử và đối chứng với Croc gốc](docs/TESTING.md)

## Nguồn gốc và giấy phép

Dự án dựa trên Croc **v11.6.6**, commit `440ed73844fd02a8185e6e6d2d4cd772388587dc`. Xem [README upstream](README.upstream.md), [MIT License](LICENSE) và [thông báo bên thứ ba](THIRD_PARTY_NOTICES.md).
