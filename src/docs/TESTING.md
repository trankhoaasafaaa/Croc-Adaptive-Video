# Kiểm thử

## Lệnh build và test

Các lệnh dưới đây chạy bằng Go 1.27.3 trên Linux x86-64. Nếu môi trường có proxy cấu hình sẵn, bỏ các biến proxy khi chạy test loopback. Croc CLI có thể đọc `HTTP_PROXY` thành proxy riêng và ép test store gửi qua proxy dù mock server nằm ở localhost.

```bash
export CROC_NO_UPDATE_CHECK=1
unset HTTP_PROXY HTTPS_PROXY ALL_PROXY http_proxy https_proxy all_proxy SOCKS5_PROXY
go test -buildvcs=false -count=1 ./src/adaptivevideo ./src/croc ./src/cli ./src/message
go test -buildvcs=false -count=1 .
go test -buildvcs=false -race -count=1 ./src/adaptivevideo ./src/croc -run 'TestSelect|TestParseMP4|TestVideo'
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -o ../croc-adaptive.exe .
```

Cờ `-buildvcs=false` cho phép build từ source ZIP trong thư mục không có metadata Git. Dùng lệnh demo trong [README](../README.md) để kiểm tra việc chọn chất lượng. Không thêm cờ `--debug` cho demo loopback. Debug upstream có thêm truy vấn public IP ngoài mạng LAN.

## Thử tích hợp và đối chiếu với Croc gốc

Các lệnh dưới đây chạy tại gốc repo. Tạo binary đối chứng Croc v11.6.6 khi muốn thử khả năng tương thích với peer gốc hoặc so sánh với cách gửi cố định.

```bash
git clone --branch v11.6.6 --single-branch https://github.com/schollz/croc.git ../baseline
test "$(git -C ../baseline rev-parse HEAD)" = 440ed73844fd02a8185e6e6d2d4cd772388587dc
(cd ../baseline && go build -buildvcs=false -o ../croc-baseline .)
go build -buildvcs=false -o ../croc-adaptive .
bash scripts/adaptive-video/fixtures.sh ../video-fixtures

python3 scripts/adaptive-video/demo.py --baseline ../croc-baseline \
  --adaptive ../croc-adaptive --fixtures ../video-fixtures \
  --output integration.csv --suite integration --workers 3 --timeout 42

python3 scripts/adaptive-video/demo.py --baseline ../croc-baseline \
  --adaptive ../croc-adaptive --fixtures ../video-fixtures \
  --output benchmark.csv --suite benchmark --repeats 5 --workers 10 --timeout 125

python3 scripts/adaptive-video/summarize.py benchmark.csv
```

Mỗi lượt có một relay gốc, hai CLI, một thư mục đối chứng rỗng và một mã riêng. Script đợi cả hai cổng, giới hạn thời gian, dọn tiến trình, đối chiếu chính xác tên và SHA-256. Script chỉ lưu dữ liệu đã lọc. Các ca thất bại được giữ nguyên trong CSV.

Trong lượt demo, kiểm tra cột `height`, `sha_match` và `ready_within_deadline`. Trong lượt tích hợp, `passed=True` cho biết ca đạt đúng kỳ vọng, kể cả khi ca đó yêu cầu phiên báo lỗi. Ca `slowdown` cho thấy ETA ban đầu có thể lệch khi tốc độ thay đổi giữa phiên.

## Dấu hiệu không có payload khi từ chối

Proxy đếm byte TCP ứng dụng từ sender đến relay qua cổng data riêng. Lượt từ chối chỉ có vài trăm byte điều khiển mở kênh của Croc, dưới ngưỡng 4 KiB của harness. Cổng control vẫn mang mẫu probe. Thư mục nhận rỗng và cả hai CLI kết thúc có lỗi. Ngưỡng quan sát này áp dụng cho các fixture đều lớn hơn 700 KB và kịch bản test hiện tại.
