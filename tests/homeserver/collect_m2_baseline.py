#!/usr/bin/env python3
"""Read-only M2 baseline evidence for one existing Bridge service.

Never reads Bridge configuration, tokens, certificates, network endpoints or
customer identifiers. Does not start, stop, restart or reload any service.
Output contains only explicit allowlisted fields. Do not publish raw host
details to a public repository without operator review.
"""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import re
import subprocess
import sys
from pathlib import Path

UNIT_RE = re.compile(r"[A-Za-z0-9_.@-]+\.service\Z")
M1_REFERENCE_SHA256 = "e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5"
ALLOWED_SYSTEMD_KEYS = {"MainPID", "ActiveState", "SubState", "NRestarts"}


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def parse_systemd_show(text: str) -> dict[str, str]:
    result = {}
    for line in text.splitlines():
        key, sep, value = line.partition("=")
        if sep and key in ALLOWED_SYSTEMD_KEYS:
            result[key] = value
    if not ALLOWED_SYSTEMD_KEYS.issubset(result):
        raise ValueError("systemd status is incomplete")
    if not result["MainPID"].isdigit():
        raise ValueError("MainPID is invalid")
    return result


def rss_kib(status_text: str) -> int | None:
    for line in status_text.splitlines():
        if line.startswith("VmRSS:"):
            parts = line.split()
            if len(parts) == 3 and parts[1].isdigit() and parts[2] == "kB":
                return int(parts[1])
            raise ValueError("unrecognized VmRSS format")
    return None


def collect(role: str, unit: str, proc_root: Path = Path("/proc")) -> dict:
    if role not in {"homeserver", "finserver"}:
        raise ValueError("role must be homeserver or finserver")
    if not UNIT_RE.fullmatch(unit):
        raise ValueError("invalid systemd unit name")
    # Fixed command and argument list: no shell and no arbitrary user options.
    proc = subprocess.run(
        ["systemctl", "show", "--no-pager",
         "--property=MainPID,ActiveState,SubState,NRestarts", unit],
        check=True, capture_output=True, text=True, timeout=8,
    )
    status = parse_systemd_show(proc.stdout)
    pid = int(status["MainPID"])
    if pid <= 0 or status["ActiveState"] != "active":
        raise ValueError("Bridge service is not running; no PASS may be inferred")
    if not status["NRestarts"].isdigit():
        raise ValueError("NRestarts is invalid")
    runtime_exe = proc_root / str(pid) / "exe"
    # Never emit executable paths, command lines, service environment or configs.
    running_sha = sha256_file(runtime_exe)
    rss_text = (proc_root / str(pid) / "status").read_text(encoding="utf-8")
    return {
        "schema": "tervon.bridge.m2-readonly-baseline.v1",
        "role": role,
        "unit": unit,
        "observed_at_utc": dt.datetime.now(dt.timezone.utc).isoformat(),
        "active_state": status["ActiveState"],
        "sub_state": status["SubState"],
        "restart_count": int(status["NRestarts"]),
        "main_pid": pid,
        "running_executable_sha256": running_sha,
        "matches_last_operator_confirmed_m1_sha256": running_sha == M1_REFERENCE_SHA256,
        "process_rss_kib": rss_kib(rss_text),
        "padding_live_enabled": "NOT_MEASURED",
        "cpu_percent": "NOT_MEASURED",
        "route_rtt_ms": "NOT_MEASURED",
        "tcp_retransmissions": "NOT_MEASURED",
        "notes": "Read-only service snapshot; NOT a live M2 test or release approval.",
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--role", required=True, choices=("homeserver", "finserver"))
    parser.add_argument("--unit", required=True, help="existing bridge systemd .service unit")
    args = parser.parse_args(argv)
    try:
        result = collect(args.role, args.unit)
    except (OSError, subprocess.SubprocessError, ValueError) as exc:
        # Keep failure message generic; prevent raw systemctl or proc paths
        # (which could disclose host configuration) from appearing in reports.
        print(json.dumps({"status": "COLLECTION_FAILED", "reason": type(exc).__name__}), file=sys.stderr)
        return 1
    print(json.dumps(result, sort_keys=True, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
