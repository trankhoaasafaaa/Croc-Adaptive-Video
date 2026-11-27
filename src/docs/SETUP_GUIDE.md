# Thiết lập và chạy

## Chuẩn bị demo

Để thấy ba mức chất lượng trên một máy, dùng `bash scripts/adaptive-video/run-demo.sh` như trong [README](../README.md). Phần dưới dùng ba terminal để trực tiếp điều khiển hai đầu và thử lời nhắc xác nhận.

Chạy từ thư mục gốc repo trên Linux hoặc WSL với Go 1.27.3. Nếu muốn dùng video tự chuẩn bị, đặt các tệp MP4 với tên tương ứng hoặc sửa đường dẫn trong cờ `--video-variant`. Để tạo bộ mẫu, cần Python 3.11 trở lên, FFmpeg có encoder `libx264` và FFprobe.

```bash
go build -buildvcs=false -o ../croc-adaptive .
bash scripts/adaptive-video/fixtures.sh ../video-fixtures
```

Nếu build trực tiếp trên Windows, đặt đuôi `.exe`. Sau khi đã có ba tệp MP4, việc gửi và nhận không cần FFmpeg.

## Một máy, ba tiến trình

Từ gốc source, mở ba terminal. Dùng một cụm mã phiên riêng dùng một lần gồm ít nhất 6 ký tự. Lệnh này cố định kết nối tới địa chỉ loopback và đặt cờ tắt kiểm tra phiên bản.

Terminal 1

```bash
../croc-adaptive relay --host 127.0.0.1 --ports 9009,9010
```

Terminal 2, sender. Nhập mã phiên ở terminal này trước.

```bash
read -r -s -p 'Mã phiên dùng một lần: ' video_code; printf '\n'
CROC_SECRET="$video_code" ../croc-adaptive --no-update-check --disable-clipboard \
  --relay 127.0.0.1:9009 send --video-deadline 20s \
  --video-variant '1080=../video-fixtures/video-1080.mp4' \
  --video-variant '360=../video-fixtures/video-360.mp4' \
  --video-variant '144=../video-fixtures/video-144.mp4'
unset video_code
```

Terminal 3, receiver. Nhập cùng mã phiên. Observer có thể nhập `y` cho đề xuất hoặc thêm cờ global `--yes` để tự nhận đề xuất đúng hạn.

```bash
mkdir -p ../received
read -r -s -p 'Mã phiên từ sender: ' video_code; printf '\n'
CROC_SECRET="$video_code" ../croc-adaptive --no-update-check --disable-clipboard \
  --ip 127.0.0.1:9009 --out ../received --video-min-height 360
unset video_code
sha256sum ../received/*.mp4
```

Đối chiếu SHA-256 với tệp nguồn cùng tên trong `../video-fixtures`. Dùng thư mục nhận rỗng cho mỗi lượt. Nếu chọn bản 360p, đầu ra chỉ có `video-360.mp4`.

## Ba thiết bị trong LAN

1. Máy Linux A chạy `./croc-adaptive relay --host 0.0.0.0 --ports 9009,9010`. Cho phép TCP 9009 và 9010 trong firewall cho mạng LAN demo.
2. Máy B có ba file và `croc-adaptive`. Thay relay loopback bằng địa chỉ của A, ví dụ `--relay 192.168.1.20:9009`. Gửi mã phiên cho người điều khiển C qua kênh riêng.
3. Máy C nhận bằng `--ip 192.168.1.20:9009` và thư mục nhận trống. Ngưỡng tối thiểu được đặt trên C. Ví dụ receiver Windows PowerShell chạy trong thư mục có binary.

```powershell
New-Item -ItemType Directory -Force .\received | Out-Null
$env:CROC_SECRET = Read-Host 'Mã phiên từ sender'
try {
  .\croc-adaptive.exe --no-update-check --disable-clipboard --ip 192.168.1.20:9009 --out .\received --video-min-height 360
  Get-ChildItem .\received\*.mp4 | Get-FileHash -Algorithm SHA256
} finally {
  Remove-Item Env:CROC_SECRET -ErrorAction SilentlyContinue
}
```

Để thấy quyết định thay đổi qua ba mức băng thông, chạy demo tự động ở [README](../README.md) trên Linux hoặc WSL. Demo LAN dùng tốc độ thật của mạng nên có thể chọn 1080p ở cả nhiều lượt. Khi thử ngưỡng nhận trên LAN, dùng `--video-min-height 360` ở receiver. Để chủ động thử đề xuất trễ, bắt đầu với deadline ngắn, chẳng hạn `--video-deadline 1s`, rồi chỉnh theo ETA và thời gian probe đo được trên mạng đó. Khi đề xuất trễ xuất hiện, thử duyệt và từ chối ở hai phiên riêng.
