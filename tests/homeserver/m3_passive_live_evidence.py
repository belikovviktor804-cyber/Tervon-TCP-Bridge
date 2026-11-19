#!/usr/bin/env python3
"""M3 passive real-network evidence sampler via Finserver's public HTTPS.

Read-only: never touches Bridge configs, systemd, client sockets, or keys.
Run from the existing Homeserver GitHub SSH checkout, preferably detached.
The script does NOT claim end-to-end RTT, per-stream fairness or jitter bounds
from aggregate counters. Its output contains only allowlisted numeric metrics.
"""
from __future__ import annotations

import argparse
import datetime as dt
import json
import subprocess
import sys
import time
from typing import Any, Callable

SCHEMA = "tervon.bridge.dashboard.v1"
URL = "https://stats.xrayvless.online/bridge/api/status"
PIN = "stats.xrayvless.online:443:87.103.198.219"
ROLES = ("homeserver", "finserver")
COUNTERS = (
    "sessions_started", "sessions_completed", "sessions_rejected",
    "upload_bytes", "download_bytes",
    "padding_upload_frames", "padding_download_frames",
    "padding_upload_bytes", "padding_download_bytes",
    "jitter_upload_frames", "jitter_download_frames",
    "jitter_upload_delay_ns", "jitter_download_delay_ns",
    "backend_dial_success", "backend_dial_failure",
    "tls13_h2_mtls_requests",
)
ERRORS = (
    "backend_dial", "http_status", "timeout", "cancellation",
    "shutdown", "authorization", "tls_handshake",
)
MAX_RESPONSE = 128 * 1024


def remote_public_fetch() -> dict[str, Any]:
    # Finserver is the independent external observer. TLS validation remains
    # ON, with SNI/domain unchanged and the public IPv4 explicitly pinned.
    proc = subprocess.run(
        [
            "ssh", "-F", "/root/.ssh/config", "-T",
            "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=yes",
            "-o", "ConnectTimeout=7", "finserver",
            "curl --noproxy '*' -fsS --max-time 8 "
            "--resolve stats.xrayvless.online:443:87.103.198.219 "
            "https://stats.xrayvless.online/bridge/api/status",
        ],
        capture_output=True, timeout=17,
    )
    if proc.returncode:
        raise ValueError("public fetch failed")
    if len(proc.stdout) > MAX_RESPONSE:
        raise ValueError("public response too large")
    return json.loads(proc.stdout)


def nonnegative_int(value: object) -> int:
    if type(value) is not int or value < 0:
        raise ValueError("invalid counter")
    return value


def parse(raw: object) -> dict[str, dict[str, Any]]:
    if not isinstance(raw, dict) or raw.get("schema") != SCHEMA:
        raise ValueError("invalid dashboard schema")
    nodes = raw.get("nodes")
    if not isinstance(nodes, dict):
        raise ValueError("missing nodes")
    out: dict[str, dict[str, Any]] = {}
    for role in ROLES:
        node = nodes.get(role)
        if not isinstance(node, dict) or node.get("status") != "ok":
            raise ValueError("node unavailable: " + role)
        metrics = node.get("metrics")
        if not isinstance(metrics, dict) or metrics.get("ready") is not True:
            raise ValueError("bridge not ready: " + role)
        errors = metrics.get("errors")
        if not isinstance(errors, dict):
            raise ValueError("missing error counters")
        out[role] = {
            "sessions_active": nonnegative_int(metrics.get("sessions_active")),
            "counters": {k: nonnegative_int(metrics.get(k)) for k in COUNTERS},
            "errors": {k: nonnegative_int(errors.get(k)) for k in ERRORS},
        }
    return out


def calculate(before: dict[str, dict], after: dict[str, dict]) -> dict:
    result = {}
    for role in ROLES:
        a, b = before[role], after[role]
        delta: dict[str, int] = {}
        for group in ("counters", "errors"):
            for key, current in b[group].items():
                diff = current - a[group][key]
                if diff < 0:
                    raise ValueError("counter reset or process restart: " + role)
                delta[key] = diff
        result[role] = {
            "active_at_start": a["sessions_active"],
            "active_at_end": b["sessions_active"],
            "counter_deltas": {k: delta[k] for k in COUNTERS},
            "error_deltas": {k: delta[k] for k in ERRORS},
        }
    return result


def collect(duration: int, interval: int,
            fetcher: Callable[[], dict[str, Any]] = remote_public_fetch,
            sleep: Callable[[float], None] = time.sleep,
            emit: Callable[[dict], None] | None = None,
            require_activation: bool = False) -> dict:
    if not 60 <= duration <= 600 or not 5 <= interval <= 60 or duration % interval:
        raise ValueError("duration=60..600, interval=5..60, divisible")
    if emit is None:
        emit = lambda obj: print(json.dumps(obj, sort_keys=True), flush=True)
    first = parse(fetcher())
    prev = first
    sample_count = 1
    started_utc = dt.datetime.now(dt.timezone.utc).isoformat()
    emit({"kind": "start", "started_utc": started_utc,
          "duration_seconds": duration, "interval_seconds": interval,
          "source": "finserver_public_https", "schema": "tervon.m3.passive-evidence.v1"})
    for step in range(1, duration // interval + 1):
        sleep(interval)
        current = parse(fetcher())
        change = calculate(prev, current)
        # Each sample must be monotonic; do not conceal temporary failures,
        # counter resets, or missing remote telemetry.
        sample_count += 1
        emit({"kind": "sample", "step": step,
              "observed_utc": dt.datetime.now(dt.timezone.utc).isoformat(),
              "nodes": {role: {
                  "sessions_active": current[role]["sessions_active"],
                  "padding_frames_delta": (
                      change[role]["counter_deltas"]["padding_upload_frames"] +
                      change[role]["counter_deltas"]["padding_download_frames"]
                  ),
                  "jitter_frames_delta": (
                      change[role]["counter_deltas"]["jitter_upload_frames"] +
                      change[role]["counter_deltas"]["jitter_download_frames"]
                  ),
                  "error_delta": sum(change[role]["error_deltas"].values()),
              } for role in ROLES}})
        prev = current
    overall = calculate(first, prev)
    pad_frames = sum(
        overall[role]["counter_deltas"][direction]
        for role in ROLES
        for direction in ("padding_upload_frames", "padding_download_frames")
    )
    jitter_frames = sum(
        overall[role]["counter_deltas"][direction]
        for role in ROLES
        for direction in ("jitter_upload_frames", "jitter_download_frames")
    )
    error_count = sum(
        sum(overall[role]["error_deltas"].values()) for role in ROLES
    )
    pad_jitter_status = (
        "PASS_PAD_AND_JITTER_ACTIVATED" if pad_frames > 0 and jitter_frames > 0
        else "NO_OBSERVED_PAD_OR_JITTER_ACTIVATION"
    )
    if pad_frames > 0 and jitter_frames == 0:
        pad_jitter_status = "PAD_ONLY_JITTER_NOT_OBSERVED"
    if jitter_frames > 0 and pad_frames == 0:
        pad_jitter_status = "JITTER_ONLY_PAD_NOT_OBSERVED"
    if error_count:
        pad_jitter_status = "ERROR_COUNTERS_INCREASED"
    result = {
        "kind": "summary",
        "schema": "tervon.m3.passive-evidence.v1",
        "duration_seconds_target": duration,
        "samples": sample_count,
        "nodes": overall,
        "activation": {
            "padding_frames_total": pad_frames,
            "jitter_frames_total": jitter_frames,
            "error_delta_total": error_count,
            "status": pad_jitter_status,
            "demand_generated_by_sampler": False,
        },
        "status": (
            pad_jitter_status if require_activation
            else "PASS_PASSIVE_TWO_NODE_COUNTER_CONTINUITY"
        ),
        "rtt": "NOT_MEASURED",
        "per_stream_fairness": "NOT_MEASURED",
        "jitter_frame_and_window_max": "NOT_MEASURED_FROM_AGGREGATES",
        "full_M3_gate": "NOT_ASSERTED",
    }
    emit(result)
    return result


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--duration", type=int, default=180)
    parser.add_argument("--interval", type=int, default=10)
    parser.add_argument("--require-activation", action="store_true",
                        help="require observed PAD and jitter frames, without generating test traffic")
    args = parser.parse_args(argv)
    try:
        result = collect(args.duration, args.interval,
                         require_activation=args.require_activation)
        if args.require_activation and result["status"] != "PASS_PAD_AND_JITTER_ACTIVATED":
            return 3
    except (OSError, ValueError, json.JSONDecodeError, subprocess.SubprocessError) as exc:
        # Do not print stderr from SSH/curl, network endpoints or configs.
        print(json.dumps({"kind": "summary", "status": "M3_PASSIVE_EVIDENCE_FAIL",
                          "failure_type": type(exc).__name__}), flush=True)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
