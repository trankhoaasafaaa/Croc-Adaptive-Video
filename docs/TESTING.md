# Kiểm thử và nghiệm thu

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

Cờ `-buildvcs=false` cho phép build từ source ZIP trong thư mục không có metadata Git. Baseline dùng cùng toolchain và ba package upstream ở trên. Kiểm tra build và gửi loopback trước khi sửa luồng gửi nhận. Dùng lệnh build baseline và lệnh demo ở [README](../README.md). Không thêm cờ `--debug` cho demo loopback. Debug upstream có thêm truy vấn public IP ngoài mạng LAN.

## Tái lập test CLI và benchmark

```bash
python3 scripts/adaptive-video/demo.py --baseline ../croc-baseline \
  --adaptive ../croc-adaptive --fixtures ../video-fixtures \
  --output integration.csv --suite integration --workers 3 --timeout 42

python3 scripts/adaptive-video/demo.py --baseline ../croc-baseline \
  --adaptive ../croc-adaptive --fixtures ../video-fixtures \
  --output benchmark.csv --suite benchmark --repeats 5 --workers 10 --timeout 125

python3 scripts/adaptive-video/summarize.py benchmark.csv
```

Mỗi lượt có một relay gốc, hai CLI, một thư mục đối chứng rỗng và một mã riêng. Script đợi cả hai cổng, giới hạn thời gian, dọn tiến trình, đối chiếu chính xác tên và SHA-256. Script chỉ lưu dữ liệu đã lọc. Các ca thất bại được giữ nguyên trong CSV.

| Hạng mục | Trạng thái | Bằng chứng |
| --- | --- | --- |
| Build baseline và feature trên Linux | Đạt | `adaptive-evidence/build-and-tests.txt` |
| Hai CLI độc lập qua relay Croc gốc | Đạt | `adaptive-evidence/baseline-loopback.txt`, `integration.csv`, `benchmark.csv` |
| Probe thật và chọn đúng bản trên ba profile | Đạt | Các dòng adaptive ở `adaptive-evidence/benchmark.csv`, đối chiếu `fixtures.csv` |
| Ngưỡng và xác nhận nhận trễ | Đạt | `no-quality`, `late-accept`, `late-refuse`, `late-empty` |
| Hash SHA-256 và duy nhất một file | Đạt | Các lượt kết thúc hợp lệ có tên file đúng và cột `sha_match=True` |
| Gửi thường giữa hai phiên bản | Đạt | `mixed` dùng sender mới và receiver gốc, adaptive tắt |
| Peer gốc khi bật adaptive, no-fit và probe timeout | Đạt | `old-peer`, `no-fit`, `probe-timeout` |
| Mất kết nối, hủy và băng thông tụt | Đạt | `disconnect`, `cancel`, `slowdown` |
| Test Go và race có mục tiêu | Đạt | `adaptive-evidence/build-and-tests.txt` |
| Patch và build source đã giải nén | Đạt | `adaptive-evidence/build-and-tests.txt` và kiểm tra gói ZIP |
| Chạy native Windows và LAN ba máy | Chưa kiểm chứng | Cross-build Windows amd64 thành công. Có hướng dẫn triển khai LAN |
| Browser, public relay qua Internet và DERP | Chưa kiểm chứng | Nằm ngoài MVP đã chạy |

## Dấu hiệu không có payload khi từ chối

Proxy đếm byte TCP ứng dụng từ sender đến relay qua cổng data riêng. Lượt từ chối chỉ có vài trăm byte điều khiển mở kênh của Croc, dưới ngưỡng 4 KiB của harness. Cổng control vẫn mang mẫu probe. Thư mục nhận rỗng và cả hai CLI kết thúc có lỗi. Ngưỡng quan sát này áp dụng cho các fixture đều lớn hơn 700 KB và kịch bản test hiện tại.
