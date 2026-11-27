# Thiết lập và chạy

## Build hai bản

Ưu tiên Go 1.27.3 để tái lập đúng lần đo. Bản nộp đã có đầy đủ source adaptive. Thư mục Croc gốc bên cạnh dành cho relay và đối chứng.

```bash
git clone --branch v11.6.6 --single-branch https://github.com/schollz/croc.git ../baseline
test "$(git -C ../baseline rev-parse HEAD)" = 440ed73844fd02a8185e6e6d2d4cd772388587dc
(cd ../baseline && go build -o ../croc-baseline .)
go build -buildvcs=false -o ../croc-adaptive .
scripts/adaptive-video/fixtures.sh ../video-fixtures
```

Linux và Git Bash dùng lệnh theo cú pháp trên. Trên Windows, khi build trực tiếp hãy đặt đuôi `.exe`. Go CLI kiểm tra vỏ MP4 và có thể chạy trên máy không cài FFmpeg. Máy tạo fixture cần FFmpeg có libx264, FFprobe và Python 3.11 trở lên. Đặt `VIDEO_SECONDS` trước khi gọi script nếu cần clip ngắn hơn 60 giây và hiệu chỉnh lại profile dựa theo dung lượng mới.

## Một máy, ba tiến trình

Từ gốc source, mở ba terminal. Dùng một cụm mã phiên riêng dùng một lần gồm ít nhất 6 ký tự. Lệnh này cố định kết nối tới địa chỉ loopback và đặt cờ tắt kiểm tra phiên bản.

Terminal 1

```bash
../croc-baseline relay --host 127.0.0.1 --ports 9009,9010
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

1. Máy Linux A chạy `./croc-baseline relay --host 0.0.0.0 --ports 9009,9010`. Cho phép TCP 9009 và 9010 trong firewall cho mạng LAN demo.
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

Nếu LAN không kết nối được, dùng loopback và script Python trên máy Linux. Kết quả LAN và Windows native cần được đo trên thiết bị của nhóm.

## Git và patch

Nhánh cục bộ của bản triển khai là `feat/adaptive-video`. Tag cục bộ `baseline-v11.6.6` trỏ đúng commit gốc. Patch trong ZIP là diff từ tag cục bộ đến snapshot nộp.

```bash
git switch -c feat/adaptive-video
git tag baseline-v11.6.6 440ed73844fd02a8185e6e6d2d4cd772388587dc
git apply --check /path/to/croc-adaptive-video.patch
git apply /path/to/croc-adaptive-video.patch
git status --short
```

Chạy các lệnh đó từ checkout sạch đúng tag. Trong checkout có lịch sử feature đã commit, dùng `git log --oneline baseline-v11.6.6..feat/adaptive-video` và `git diff --stat baseline-v11.6.6 feat/adaptive-video`.

Khi nhóm xác định repo GitHub đích, xác minh repo và quyền bằng `gh repo view OWNER/REPO --json nameWithOwner,isPrivate,viewerPermission`. Thêm remote đích bằng `git remote add submission https://github.com/OWNER/REPO.git`. Xuất bản nhánh qua workflow được repo đích cấp quyền. Repo Croc gốc quy định helper `~/dotfiles/git-pr` cho thao tác push. Khi nhánh đã có trên GitHub, có thể mở PR nháp bằng `gh pr create --repo OWNER/REPO --head feat/adaptive-video --base main --draft`, với `main` là nhánh đích nếu repo dùng tên này.

Để lấy lại lịch sử các commit từ gói nộp thay cho áp patch, đứng ở checkout sạch đúng tag và chạy `git fetch /path/to/croc-adaptive-video.bundle feat/adaptive-video:feat/adaptive-video`, rồi `git switch feat/adaptive-video`. Bundle chứa các commit tính năng và cần commit baseline đã có trong repo.
