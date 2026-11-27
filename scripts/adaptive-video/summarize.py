#!/usr/bin/env python3
"""Summarize benchmark CSV without dropping attempted transfers."""
from collections import Counter
import csv
import statistics
import sys


def describe(values):
    if not values:
        return "—"
    if len(values) == 1:
        return f"{values[0]:.2f}"
    return f"{statistics.mean(values):.2f} ± {statistics.stdev(values):.2f}"


with open(sys.argv[1], newline="") as stream:
    rows = list(csv.DictReader(stream))

print("| Mạng | Cách gửi | Lượt | Payload hợp lệ | Kịp hạn | Chất lượng đã nhận | Tổng giây, TB ± SD | Pha nhận giây, TB ± SD | Probe giây, TB ± SD | Lệch ETA giây, TB ± SD |")
print("| --- | --- | ---: | ---: | ---: | --- | ---: | ---: | ---: | ---: | ---: |")
for profile in ("fast", "medium", "slow"):
    for method in ("fixed1080", "fixed144", "adaptive"):
        group = [r for r in rows if r["profile"] == profile and r["method"] == method]
        completed = [r for r in group if r["sha_match"] == "True" and r["status"] == "ok"]
        ready = sum(r["ready_within_deadline"] == "True" for r in group)
        qualities = ", ".join(f"{h}p × {n}" for h, n in Counter(r["height"] for r in completed).items()) or "—"
        timing = describe([float(r["budget_elapsed_s"]) for r in completed if r["budget_elapsed_s"]])
        phase = describe([float(r["receive_phase_s"]) for r in completed if r["receive_phase_s"]])
        probe = describe([float(r["probe_s"]) for r in group if r["probe_s"]])
        error = describe([float(r["eta_error_s"]) for r in completed if r["eta_error_s"]])
        print(f"| {profile} | {method} | {len(group)} | {len(completed)} | {ready}/{len(group)} | {qualities} | {timing} | {phase} | {probe} | {error} |")
