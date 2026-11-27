#!/usr/bin/env bash
set -euo pipefail

output=${1:-video-fixtures}
duration=${VIDEO_SECONDS:-60}
mkdir -p "$output"
for profile in 1080:1600k 360:450k 144:95k; do
  height=${profile%:*}
  rate=${profile#*:}
  ffmpeg -nostdin -hide_banner -loglevel error -y \
    -f lavfi -i "testsrc2=size=1920x1080:rate=12:duration=$duration" \
    -vf "scale=-2:$height" -an -c:v libx264 -preset ultrafast -threads 2 \
    -b:v "$rate" -minrate "$rate" -maxrate "$rate" -bufsize "$rate" \
    -x264-params nal-hrd=cbr -pix_fmt yuv420p -movflags +faststart \
    "$output/video-$height.mp4"
done
python3 - "$output" "$duration" <<'PY'
import csv
import json
from pathlib import Path
import subprocess
import sys

root = Path(sys.argv[1])
expected = float(sys.argv[2])
with (root / "fixtures.csv").open("w", newline="") as stream:
    writer = csv.writer(stream, lineterminator="\n")
    writer.writerow(["name", "height", "bytes", "bitrate_bps", "duration_s"])
    for height in (1080, 360, 144):
        file = root / f"video-{height}.mp4"
        raw = subprocess.check_output([
            "ffprobe", "-v", "error", "-select_streams", "v:0",
            "-show_entries", "stream=height,codec_type:format=size,bit_rate,duration",
            "-of", "json", str(file),
        ])
        info = json.loads(raw)
        track, = info["streams"]
        form = info["format"]
        assert track == {"codec_type": "video", "height": height}, info
        assert abs(float(form["duration"]) - expected) <= 0.1, form
        assert file.stat().st_size == int(form["size"]), form
        writer.writerow([file.name, height, form["size"], form["bit_rate"], form["duration"]])
PY
cat "$output/fixtures.csv"
