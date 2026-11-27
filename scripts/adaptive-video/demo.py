#!/usr/bin/env python3
"""Exercise real Croc binaries over a loopback transmission-rate proxy."""
import argparse
from concurrent.futures import ThreadPoolExecutor
from contextlib import ExitStack
import csv
import hashlib
import os
from pathlib import Path
import re
import secrets
import shutil
import signal
import socket
import subprocess
import tempfile
import threading
import time

RELAY = "127.0.0.2"
PROXY = "127.0.0.1"
RATES = {"fast": 25, "medium": 5, "slow": 1.2}
PROBE = re.compile(rb"Adaptive video probe bytes=(\d+) elapsed=(\S+) raw_Bps=(\S+) safe_Bps=(\S+)")
CHOICE = re.compile(rb"Adaptive video selected height=(\d+) bytes=(\d+) eta=(\S+) late=(true|false)")
DURATION = re.compile(r"(\d+(?:\.\d+)?)(ns|[µμ]s|ms|s|m|h)")
PORT_LOCK = threading.Lock()
USED_PORTS = set()


def seconds(go_duration):
    units = {"ns": 1e-9, "µs": 1e-6, "μs": 1e-6, "ms": 1e-3, "s": 1, "m": 60, "h": 3600}
    return sum(float(n) * units[u] for n, u in DURATION.findall(go_duration.decode()))


def ports():
    with PORT_LOCK:
        for _ in range(200):
            port = 20000 + 2 * secrets.randbelow(19000)
            pair = (port, port + 1)
            if port in USED_PORTS:
                continue
            try:
                with ExitStack() as stack:
                    for address in (PROXY, RELAY):
                        for p in pair:
                            s = stack.enter_context(socket.socket())
                            s.bind((address, p))
            except OSError:
                continue
            USED_PORTS.add(port)
            return pair
    raise RuntimeError("cannot reserve loopback ports")


class Proxy:
    # Shape upload on the sender leg, on both control and single data port.
    # TCP segments still travel over real sockets to the unmodified relay.
    def __init__(self, pair, mbps, fault):
        self.rate = mbps * 1_000_000 / 8
        self.fault = fault
        self.control_bytes = 0
        self.data_bytes = 0
        self.fault_event = threading.Event()
        self.stop = threading.Event()
        self.listeners = []
        self.threads = []
        for index, port in enumerate(pair):
            listener = socket.socket()
            listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            listener.bind((PROXY, port))
            listener.listen()
            listener.settimeout(.1)
            self.listeners.append(listener)
            thread = threading.Thread(target=self.accept, args=(listener, port, index == 1), daemon=True)
            self.threads.append(thread)
            thread.start()

    def accept(self, listener, port, is_data):
        while not self.stop.is_set():
            try:
                left, _ = listener.accept()
            except socket.timeout:
                continue
            except OSError:
                return
            threading.Thread(target=self.bridge, args=(left, port, is_data), daemon=True).start()

    def bridge(self, left, port, is_data):
        try:
            right = socket.create_connection((RELAY, port), timeout=2)
        except OSError:
            left.close()
            return
        with left, right:
            left.settimeout(.25)
            right.settimeout(.25)
            reverse = threading.Thread(target=self.pump, args=(right, left, False, is_data), daemon=True)
            reverse.start()
            self.pump(left, right, True, is_data)
            reverse.join(timeout=1)

    def pump(self, src, dst, upload, is_data):
        sent = 0
        next_send = time.monotonic()
        try:
            while not self.stop.is_set():
                try:
                    data = src.recv(16384)
                except socket.timeout:
                    continue
                if not data:
                    break
                if upload:
                    if not is_data and self.fault == "probe-timeout" and sent >= 16384:
                        self.fault_event.set()
                        self.stop.wait()
                        break
                    if is_data and sent >= 131072:
                        if self.fault == "disconnect":
                            self.fault_event.set()
                            src.shutdown(socket.SHUT_RDWR)
                            dst.shutdown(socket.SHUT_RDWR)
                            return
                        if self.fault in ("cancel", "slowdown"):
                            self.fault_event.set()
                    rate = self.rate
                    if is_data and self.fault == "slowdown" and self.fault_event.is_set():
                        rate = 1.2 * 1_000_000 / 8
                    now = time.monotonic()
                    next_send = max(next_send, now) + len(data) / rate
                    if self.stop.wait(next_send - now):
                        break
                dst.sendall(data)
                if upload:
                    sent += len(data)
                    if is_data:
                        self.data_bytes += len(data)
                    else:
                        self.control_bytes += len(data)
        except OSError:
            pass
        finally:
            try:
                dst.shutdown(socket.SHUT_WR)
            except OSError:
                pass

    def __enter__(self):
        return self

    def __exit__(self, *_):
        self.stop.set()
        for listener in self.listeners:
            listener.close()
        for thread in self.threads:
            thread.join(timeout=1)


def cli_env(root, secret):
    env = os.environ.copy()
    for key in list(env):
        if key.startswith("CROC_") or key.lower().endswith("_proxy") or key.startswith(("UMAMI_", "SITE_URL")):
            env.pop(key)
    config = root / "config"
    config.mkdir()
    env.update({"CROC_SECRET": secret, "CROC_NO_UPDATE_CHECK": "1", "XDG_CONFIG_HOME": str(config),
                "APPDATA": str(config), "HOME": str(config), "NO_COLOR": "1"})
    return env


def wait_ports(pair, relay):
    for p in pair:
        until = time.monotonic() + 5
        while True:
            if relay.poll() is not None or time.monotonic() >= until:
                raise RuntimeError("loopback relay did not start")
            try:
                with socket.create_connection((RELAY, p), timeout=.2):
                    break
            except OSError:
                time.sleep(.03)


def digest(file):
    with file.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def stop_process(process):
    if process.poll() is None:
        process.kill()
    process.wait(timeout=5)


def observe(process, out, events):
    # CLI output is retained only in memory. Save timestamps and structured
    # evidence below, excluding the invitation and other local identifiers.
    while line := process.stdout.readline():
        now = time.monotonic()
        out.append(line)
        if b"Receiving (" in line:
            events.setdefault("accept", now)
        if PROBE.search(line):
            events["probe_log"] = now


def run_case(args, case, repeat, profile, method, minimum=0, answer=None, deadline=None, fault=""):
    deadline = args.deadline if deadline is None else deadline
    pair = ports()
    with tempfile.TemporaryDirectory(prefix="croc-video-") as tmp, ExitStack() as stack:
        root = Path(tmp)
        destination = root / "received"
        destination.mkdir()
        env = cli_env(root, "video-test-" + secrets.token_hex(10))
        relay = subprocess.Popen([args.baseline, "relay", "--host", RELAY, "--ports",
                                  ",".join(map(str, pair))], stdout=subprocess.DEVNULL,
                                 stderr=subprocess.DEVNULL, env=env, cwd=root)
        stack.callback(stop_process, relay)
        wait_ports(pair, relay)
        proxy = stack.enter_context(Proxy(pair, RATES[profile], fault))
        sender_adaptive = method == "adaptive"
        fixed = 1080 if method == "fixed1080" else 144
        binary = args.adaptive if sender_adaptive or method == "mixed" else args.baseline
        receiver_bin = args.baseline if case == "old-peer" or method.startswith("fixed") else args.adaptive
        if method == "mixed":
            receiver_bin = args.baseline
        send = [binary, "--relay", f"{PROXY}:{pair[0]}", "--disable-clipboard", "--ignore-stdin",
                "--no-compress", "send", "--transport", "relay", "--no-local"]
        if sender_adaptive:
            send += ["--video-deadline", f"{deadline}s"]
            for h in (1080, 360, 144):
                send += ["--video-variant", f"{h}=video-{h}.mp4"]
        else:
            send += [f"video-{fixed}.mp4"]
        sender_refuses = case == "sender-refuse"
        if sender_refuses:
            send.insert(1, "--ask")
        receive = [receiver_bin, "--ip", f"{RELAY}:{pair[0]}", "--disable-clipboard", "--yes"]
        if minimum:
            receive += ["--video-min-height", str(minimum)]

        sender = subprocess.Popen(send, env=env, cwd=args.fixtures,
                                  stdin=subprocess.PIPE if sender_refuses else subprocess.DEVNULL,
                                  stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        stack.callback(stop_process, sender)
        if sender_refuses:
            sender.stdin.write(b"n\n")
            sender.stdin.close()
        receiver = subprocess.Popen(receive, env=env, cwd=destination, stdin=subprocess.PIPE,
                                    stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        stack.callback(stop_process, receiver)
        if answer is not None:
            receiver.stdin.write(answer)
        receiver.stdin.close()
        send_lines, receive_lines, events = [], [], {}
        observers = [
            threading.Thread(target=observe, args=(sender, send_lines, events), daemon=True),
            threading.Thread(target=observe, args=(receiver, receive_lines, events), daemon=True),
        ]
        for observer in observers:
            observer.start()
        if fault == "cancel":
            if proxy.fault_event.wait(args.timeout / 2):
                sender.send_signal(signal.SIGINT)
        timed_out = False
        try:
            receiver.wait(timeout=args.timeout)
            completed = time.monotonic()
            sender.wait(timeout=8)
        except subprocess.TimeoutExpired:
            timed_out = True
            stop_process(sender)
            stop_process(receiver)
            completed = time.monotonic()
        for observer in observers:
            observer.join(timeout=2)
        sender_output, receiver_output = b"".join(send_lines), b"".join(receive_lines)
        probe, decision = PROBE.search(sender_output), CHOICE.search(sender_output)
        height = int(decision[1]) if decision else (None if sender_adaptive else fixed)
        eta = seconds(decision[3]) if decision else None
        probe_s = seconds(probe[2]) if probe else None
        receive_phase = completed - events["accept"] if "accept" in events else None
        budget_start = events.get("accept")
        if probe_s is not None:
            budget_start = events["probe_log"] - probe_s
        elapsed = completed - budget_start if budget_start is not None else None
        filenames = sorted(f.name for f in destination.iterdir())
        filename = f"video-{height}.mp4"
        src_hash = digest(args.fixtures / filename) if height else ""
        dst_hash = digest(destination / filename) if filenames == [filename] else ""
        sha_match = bool(dst_hash and src_hash == dst_hash)
        success = not timed_out and sender.returncode == receiver.returncode == 0 and sha_match

        if success and args.keep_received:
            saved = args.keep_received / profile / method / str(repeat)
            saved.mkdir(parents=True, exist_ok=True)
            shutil.copy2(destination / filename, saved / filename)

        joined = sender_output + receiver_output
        error_signatures = (
            b"no variant fits the video deadline", b"no variant meets the receiver's minimum height",
            b"receiver declined the selected video", b"sender declined the selected video", b"incompatible peer", b"probe timed out",
            b"transfer canceled", b"context canceled", b"transfer disconnected", b"session interrupted",
        )
        status = "ok" if success else "timeout" if timed_out else next(
            (x.decode() for x in error_signatures if x in joined), "unexpected failure")
        success_expected = case in ("benchmark", "late-accept", "mixed", "slowdown")
        expected_errors = {
            "no-fit": "no variant fits the video deadline",
            "no-quality": "no variant meets the receiver's minimum height",
            "late-refuse": "receiver declined the selected video",
            "late-empty": "receiver declined the selected video",
            "old-peer": "incompatible peer",
            "probe-timeout": "probe timed out",
            "disconnect": "transfer disconnected",
        }
        expected_height = {"fast": 1080, "medium": 360, "slow": 144}[profile]
        if success_expected:
            passed = success and (case != "benchmark" or not sender_adaptive or height == expected_height)
            if case == "late-accept":
                passed = passed and height == 360 and decision[4] == b"true" and b"Accept after deadline" in receiver_output
        elif sender_refuses:
            # The receiver may observe disconnection before the refusal
            # diagnostic crosses the relay. It must still exit with an error.
            passed = (not timed_out and not sha_match and proxy.data_bytes < 4096
                      and sender.returncode == receiver.returncode == 1
                      and b"sender declined the selected video" in sender_output)
        elif fault in ("disconnect", "cancel"):
            passed = (not success and not timed_out and not sha_match and proxy.fault_event.is_set()
                      and (sender.returncode != 0 and receiver.returncode != 0))
            passed = passed and status == ("transfer canceled" if fault == "cancel" else "transfer disconnected")
        else:
            passed = (not success and not timed_out and not filenames and proxy.data_bytes < 4096
                      and status == expected_errors[case])
            if case.startswith("late-"):
                passed = passed and b"Accept after deadline" in receiver_output

        return {
            "case": case, "profile": profile, "mbps": RATES[profile], "method": method, "repeat": repeat,
            "deadline_s": deadline, "minimum": minimum, "height": height, "late_offer": decision[4].decode() if decision else "",
            "probe_s": probe_s, "raw_Bps": float(probe[3]) if probe else None, "safe_Bps": float(probe[4]) if probe else None,
            "eta_s": eta, "receive_phase_s": receive_phase, "budget_elapsed_s": elapsed,
            "eta_error_s": receive_phase - eta if success and eta is not None and receive_phase is not None else None,
            "ready_within_deadline": bool(success and elapsed is not None and elapsed <= deadline),
            "sha256": dst_hash if sha_match else "", "sha_match": sha_match,
            "files": "|".join(filenames), "control_upload_bytes": proxy.control_bytes, "data_upload_bytes": proxy.data_bytes,
            "sender_exit": sender.returncode, "receiver_exit": receiver.returncode, "status": status, "passed": bool(passed),
        }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for key in ("baseline", "adaptive", "fixtures", "output"):
        parser.add_argument("--" + key, required=True, type=lambda p: str(Path(p).resolve()))
    parser.add_argument("--keep-received", type=Path, help="save verified output videos under this directory")
    parser.add_argument("--suite", choices=("demo", "integration", "benchmark"), default="demo")
    parser.add_argument("--repeats", type=int, default=5)
    parser.add_argument("--workers", type=int, default=4)
    parser.add_argument("--timeout", type=int, default=110)
    parser.add_argument("--deadline", type=float, default=20)
    parser.add_argument("--profiles", nargs="+", choices=list(RATES), default=list(RATES))
    args = parser.parse_args()
    if args.repeats < 1 or args.workers < 1:
        parser.error("--repeats and --workers must be positive")
    args.fixtures = Path(args.fixtures)
    jobs = []
    if args.suite == "demo":
        jobs = [("benchmark", 1, p, "adaptive") for p in RATES]
    if args.suite == "benchmark":
        jobs = [("benchmark", n, p, m) for n in range(1, args.repeats + 1)
                for p in args.profiles for m in ("fixed1080", "fixed144", "adaptive")]
        jobs.sort(key=lambda job: (job[2] != "slow" or job[3] != "fixed1080", job[1]))
    if args.suite == "integration":
        jobs = [
            ("no-fit", 1, "slow", "adaptive", 0, None, 4),
            ("no-quality", 1, "fast", "adaptive", 2160),
            ("late-accept", 1, "fast", "adaptive", 360, b"y\n", 1),
            ("late-refuse", 1, "fast", "adaptive", 360, b"n\n", 1),
            ("late-empty", 1, "fast", "adaptive", 360, b"\n", 1),
            ("sender-refuse", 1, "fast", "adaptive", 0, b"y\n"),
            ("old-peer", 1, "fast", "adaptive"),
            ("mixed", 1, "fast", "mixed"),
            ("probe-timeout", 1, "fast", "adaptive", 0, None, 2, "probe-timeout"),
            ("disconnect", 1, "fast", "adaptive", 0, None, None, "disconnect"),
            ("cancel", 1, "fast", "adaptive", 0, None, None, "cancel"),
            ("slowdown", 1, "medium", "adaptive", 0, None, None, "slowdown"),
        ]
    with ThreadPoolExecutor(max_workers=args.workers) as pool:
        futures = [pool.submit(run_case, args, *job) for job in jobs]
        with open(args.output, "w", newline="") as stream:
            writer = None
            valid = True
            for future in futures:
                row = future.result()
                if writer is None:
                    writer = csv.DictWriter(stream, fieldnames=row.keys(), lineterminator="\n")
                    writer.writeheader()
                writer.writerow(row)
                stream.flush()
                if args.suite == "demo" and row["status"] == "ok":
                    print(f"{row['mbps']:g} Mb/s | {row['height']}p | "
                          f"ETA {row['eta_s']:.2f}s | nhận xong {row['budget_elapsed_s']:.2f}s / "
                          f"{row['deadline_s']:g}s | {row['files']} | SHA-256 khớp", flush=True)
                else:
                    print(f"{row['case']} {row['profile']} {row['method']} n={row['repeat']} "
                          f"height={row['height']} {row['status']} passed={row['passed']}", flush=True)
                valid = valid and row["passed"]
    return 0 if valid else 1


if __name__ == "__main__":
    raise SystemExit(main())
