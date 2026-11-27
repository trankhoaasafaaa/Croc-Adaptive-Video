# Croc Adaptive Video

Gửi đúng một phiên bản MP4 qua relay Croc. Sau khi hai đầu CLI xác thực, sender đo tốc độ bằng probe thật. Sender chọn độ phân giải cao nhất dự kiến kịp deadline và đạt ngưỡng phía nhận. Receiver có thể chủ động đồng ý nhận trễ nếu đã đặt ngưỡng.

Dự án dùng Croc **v11.6.6**, commit `440ed73844fd02a8185e6e6d2d4cd772388587dc`. Cờ adaptive chỉ có tác dụng khi người gửi bật. Luồng gửi tệp thông thường vẫn dùng giao thức Croc. Mã nguồn và nội dung gốc theo [README upstream](README.upstream.md), [MIT License](LICENSE) và [thông báo bên thứ ba](THIRD_PARTY_NOTICES.md).

## Chạy thử trên một máy Linux

Cần Go 1.27.3, Git, Python 3.11 trở lên, FFmpeg với libx264 và FFprobe. Gói nộp có source ở thư mục `croc-adaptive-video/` sau khi giải nén. Chạy tại gốc source và đặt `baseline` song song với source hiện tại.

```bash
git clone --branch v11.6.6 --single-branch https://github.com/schollz/croc.git ../baseline
git -C ../baseline rev-parse HEAD
(cd ../baseline && go build -o ../croc-baseline .)
go build -buildvcs=false -o ../croc-adaptive .
scripts/adaptive-video/fixtures.sh ../video-fixtures
python3 scripts/adaptive-video/demo.py --baseline ../croc-baseline \
  --adaptive ../croc-adaptive --fixtures ../video-fixtures --output demo.csv
```

Script chạy một relay Croc gốc và hai CLI riêng cho mỗi lượt. Ba profile dự kiến lần lượt chọn 1080p, 360p và 144p. CSV lưu ETA, thời gian nhận thực tế, tên tệp duy nhất và SHA-256. Dùng `--suite benchmark --profiles medium --repeats 1` để đối chiếu 1080p cố định, 144p cố định và adaptive trên cùng băng thông 5 Mb/s. Lệnh cho ba thiết bị và PowerShell có trong [hướng dẫn thiết lập](docs/SETUP_GUIDE.md).

## Cờ adaptive

| Cờ | Đặt ở đâu | Hành vi |
| --- | --- | --- |
| `--video-deadline 20s` | Sau `send` | Đặt ngân sách ước tính bắt đầu từ probe |
| `--video-variant '360=video 360.mp4'` | Sau `send`, lặp 2–16 lần | Khai báo chiều cao và đường dẫn đến bản MP4 đã tạo sẵn |
| `--video-min-height 360` | Cờ global ở CLI nhận | Giữ ngưỡng 360p. Nếu bản đủ ngưỡng đều trễ, bắt buộc nhập `y` hoặc `yes` |

Deadline phục vụ quyết định chọn bản. Tốc độ data có thể thay đổi sau probe. Xem [kiến trúc và giới hạn](docs/ARCHITECTURE.md), [lệnh kiểm thử](docs/TESTING.md), [báo cáo kết quả](docs/IMPLEMENTATION_REPORT.md) và [kịch bản demo](docs/DEMO_SCRIPT.md).
