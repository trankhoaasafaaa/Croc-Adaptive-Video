#!/usr/bin/env bash
set -euo pipefail

for tool in go python3 ffmpeg ffprobe; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Thiếu $tool. Xem phần Chuẩn bị trong README.md." >&2
    exit 1
  fi
done
if ! python3 -c 'import sys; sys.exit(sys.version_info < (3, 11))'; then
  echo "Cần Python 3.11 trở lên." >&2
  exit 1
fi

repo="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo"
mkdir -p .croc-demo
run="$(mktemp -d "$repo/.croc-demo/run-XXXXXX")"

echo 'Đang build Croc Adaptive Video...'
go build -buildvcs=false -o "$run/croc-adaptive" .
echo 'Đang tạo ba bản của cùng video 60 giây...'
VIDEO_SECONDS=60 bash scripts/adaptive-video/fixtures.sh "$run/fixtures"
echo 'Đang thử ba mức băng thông qua relay Croc trên máy này...'
python3 scripts/adaptive-video/demo.py \
  --baseline "$run/croc-adaptive" --adaptive "$run/croc-adaptive" \
  --fixtures "$run/fixtures" --output "$run/demo.csv" \
  --keep-received "$run/received" --suite demo --workers 1 --timeout 40
echo "Video đã nhận và kết quả: $run"
