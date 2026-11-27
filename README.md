# Croc Adaptive Video

Gửi đúng một phiên bản MP4 qua relay Croc. Sau khi hai đầu CLI xác thực, sender đo tốc độ bằng probe thật. Sender chọn độ phân giải cao nhất dự kiến kịp deadline và đạt ngưỡng phía nhận. Receiver có thể chủ động đồng ý nhận trễ nếu đã đặt ngưỡng.

Dự án dùng Croc **v11.6.6**, commit `440ed73844fd02a8185e6e6d2d4cd772388587dc`. Cờ adaptive chỉ có tác dụng khi người gửi bật. Luồng gửi tệp thông thường vẫn dùng giao thức Croc. Mã nguồn và nội dung gốc theo [README upstream](README.upstream.md), [MIT License](LICENSE) và [thông báo bên thứ ba](THIRD_PARTY_NOTICES.md).

## Demo trên một máy Linux hoặc Windows qua WSL

Trong terminal Linux hoặc WSL, chuẩn bị Go 1.27.3, Python 3.11 trở lên, FFmpeg có encoder `libx264` và FFprobe. Sau khi clone, đứng tại thư mục có `go.mod` và chạy:

```bash
bash scripts/adaptive-video/run-demo.sh
```

Lệnh tự build CLI, tạo ba bản của cùng video, chạy relay và hai CLI riêng trên loopback. Proxy đặt ba mức băng thông **25 / 5 / 1,2 Mb/s** để thử việc tự chọn **1080p / 360p / 144p** với deadline 20 giây. Cả ba tiến trình dùng cùng bản build. Lệnh `relay` dùng relay Croc sẵn có.

Khi trình bày, chỉ ra dòng có băng thông, chất lượng được chọn, ETA, thời gian nhận thực tế và SHA-256. Mở thư mục `received` trong đường dẫn chương trình in ở cuối để phát tệp nhận được. Mỗi mức mạng có thư mục riêng và chỉ nhận một bản MP4. Thời gian chạy lần đầu phụ thuộc vào tải module Go và tốc độ build. Sau khi có dependency, cả ba phiên demo đi qua loopback.

Muốn thử lời nhắc nhận, dùng video tự chọn hoặc chạy giữa các máy, xem [hướng dẫn thiết lập](docs/SETUP_GUIDE.md). Các lệnh chạy đối chứng với Croc gốc nằm trong [hướng dẫn kiểm thử](docs/TESTING.md).

## Cờ adaptive

| Cờ | Đặt ở đâu | Hành vi |
| --- | --- | --- |
| `--video-deadline 20s` | Sau `send` | Đặt ngân sách ước tính bắt đầu từ probe |
| `--video-variant '360=video 360.mp4'` | Sau `send`, lặp 2–16 lần | Khai báo chiều cao và đường dẫn đến bản MP4 đã tạo sẵn |
| `--video-min-height 360` | Cờ global ở CLI nhận | Giữ ngưỡng 360p. Nếu bản đủ ngưỡng đều trễ, bắt buộc nhập `y` hoặc `yes` |

Deadline phục vụ quyết định chọn bản. Tốc độ data có thể thay đổi sau probe. Xem [kiến trúc và giới hạn](docs/ARCHITECTURE.md) và [cách chạy thử trên LAN](docs/SETUP_GUIDE.md).
