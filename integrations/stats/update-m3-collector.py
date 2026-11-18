#!/usr/bin/env python3
"""Guarded update of the EXISTING Homeserver Stats collector only.

Only bridge_metrics.py is replaced; only tervon-bridge-stats.service restarts.
Never modifies Encapsulator, nginx, Finserver, config, or environment file.
Requires an explicitly invoked root command from a clean SSH M3 checkout.
"""
from __future__ import annotations

import json
import os
from pathlib import Path
import stat
import subprocess
import tempfile
import time
from urllib.request import ProxyHandler, build_opener

APP = Path("/opt/tervon/bridge-stats")
TARGET = APP / "bridge_metrics.py"
SERVICE = "tervon-bridge-stats.service"
BRANCH = "work/m3-adaptive-jitter"
REMOTE = "git@github.com:tervon-software/Tervon-TCP-Bridge.git"
STATUS_URL = "http://127.0.0.1:8894/bridge/api/status"
RAW_URL = "http://127.0.0.1:8895/metrics/v1"
PAD_FIELDS = (
    "padding_upload_bytes", "padding_download_bytes",
    "padding_upload_frames", "padding_download_frames",
    "padding_upload_transitions", "padding_download_transitions",
    "padding_upload_wire_overhead_percent",
    "padding_download_wire_overhead_percent",
    "padding_upload_state", "padding_download_state",
)


def command(*args: str) -> str:
    p = subprocess.run(args, capture_output=True, text=True, timeout=15,
                       check=True)
    return p.stdout.strip()


def regular(path: Path) -> bool:
    return path.is_file() and not path.is_symlink()


def json_get(url: str) -> dict:
    opener = build_opener(ProxyHandler({}))
    with opener.open(url, timeout=2) as response:
        payload = response.read(65537)
        if response.status != 200 or len(payload) > 65536:
            raise RuntimeError("unexpected local metrics response")
    data = json.loads(payload)
    if not isinstance(data, dict):
        raise RuntimeError("invalid local JSON")
    return data


def dashboard() -> dict:
    data = json_get(STATUS_URL)
    if data.get("schema") != "tervon.bridge.dashboard.v1":
        raise RuntimeError("unexpected dashboard schema")
    nodes = data.get("nodes")
    if not isinstance(nodes, dict) or set(nodes) != {"homeserver"}:
        raise RuntimeError("unexpected dashboard nodes")
    node = nodes["homeserver"]
    if not isinstance(node, dict) or node.get("status") != "ok":
        raise RuntimeError("dashboard source unavailable")
    metrics = node.get("metrics")
    if not isinstance(metrics, dict) or metrics.get("ready") is not True:
        raise RuntimeError("dashboard not ready")
    return metrics


def preflight(repo: Path, src: Path) -> tuple[dict, dict]:
    if os.geteuid() != 0:
        raise RuntimeError("root required")
    if command("git", "-C", str(repo), "remote", "get-url", "origin") != REMOTE:
        raise RuntimeError("SSH remote mismatch")
    if command("git", "-C", str(repo), "branch", "--show-current") != BRANCH:
        raise RuntimeError("branch mismatch")
    if command("git", "-C", str(repo), "--no-optional-locks", "status", "--porcelain"):
        raise RuntimeError("dirty checkout")
    if not APP.is_dir() or APP.is_symlink():
        raise RuntimeError("stats directory absent or symlink")
    for f in (src, TARGET, APP / "bridge_site.py"):
        if not regular(f):
            raise RuntimeError("collector source or application file invalid")
    if command("systemctl", "is-active", SERVICE) != "active":
        raise RuntimeError("stats service not active")
    old = dashboard()
    raw = json_get(RAW_URL)
    if raw.get("schema") != "tervon.bridge.telemetry.v1" or raw.get("ready") is not True:
        raise RuntimeError("private Bridge metrics not ready")
    # Never print raw payloads or local endpoint/config secrets.
    return old, raw


def replace_file(path: Path, data: bytes, mode: int) -> None:
    fd, tmp = tempfile.mkstemp(prefix=".m3-collector-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as out:
            out.write(data)
            out.flush()
            os.fsync(out.fileno())
        os.chmod(tmp, mode)
        os.replace(tmp, path)
    finally:
        if os.path.lexists(tmp):
            os.unlink(tmp)


def wait_dashboard(raw: dict, timeout: float = 20) -> dict:
    deadline = time.monotonic() + timeout
    while True:
        try:
            if command("systemctl", "is-active", SERVICE) != "active":
                raise RuntimeError("stats service stopped")
            metrics = dashboard()
            # Every supported M2 field present in private JSON must be forwarded
            # by the new allowlist, with matching values, not fabricated zeros.
            for key in PAD_FIELDS:
                if key in raw and metrics.get(key) != raw[key]:
                    raise RuntimeError("PAD field not forwarded: " + key)
            return metrics
        except (OSError, ValueError, RuntimeError, TimeoutError):
            if time.monotonic() >= deadline:
                raise RuntimeError("collector local readiness / schema timeout")
            time.sleep(min(0.5, max(0.0, deadline - time.monotonic())))


def install() -> None:
    src = Path(__file__).resolve().with_name("bridge_metrics.py")
    repo = src.parents[2]
    before, raw = preflight(repo, src)
    original = TARGET.read_bytes()
    desired = src.read_bytes()
    if desired == original:
        wait_dashboard(raw)
        print("M3_COLLECTOR_ALREADY_CURRENT=PASS")
        return

    backup_root = APP / ".collector-backups"
    if backup_root.is_symlink() or (backup_root.exists() and not backup_root.is_dir()):
        raise RuntimeError("invalid backup directory")
    backup_root.mkdir(mode=0o700, exist_ok=True)
    backup = Path(tempfile.mkdtemp(prefix="m3-", dir=backup_root))
    os.chmod(backup, 0o700)
    (backup / "bridge_metrics.py").write_bytes(original)
    os.chmod(backup / "bridge_metrics.py", 0o600)
    mode = stat.S_IMODE(TARGET.stat().st_mode)
    print("COLLECTOR_BACKUP=" + str(backup), flush=True)

    try:
        replace_file(TARGET, desired, mode)
        command("systemctl", "restart", SERVICE)
        current = wait_dashboard(raw)
        if current.get("sessions_active") is None:
            raise RuntimeError("active sessions metric absent")
    except BaseException:
        print("COLLECTOR_ROLLBACK_STARTED", flush=True)
        restored = False
        try:
            replace_file(TARGET, original, mode)
            command("systemctl", "restart", SERVICE)
            restored = (TARGET.read_bytes() == original and
                        dashboard().get("ready") is True)
        except Exception:
            pass
        print("COLLECTOR_ROLLBACK=" + ("PASS" if restored else "FAIL"), flush=True)
        raise
    print("M3_COLLECTOR_LOCAL=PASS", flush=True)
    print("STATS_RESTART=YES", flush=True)
    print("BRIDGE_RESTART=NONE", flush=True)
    print("NGINX_RELOAD=NONE", flush=True)
    print("EXTERNAL_HTTPS=NOT_VERIFIED", flush=True)


if __name__ == "__main__":
    try:
        install()
    except Exception as exc:
        print("COLLECTOR_UPDATE_BLOCKED_OR_FAILED=" + str(exc), flush=True)
        raise SystemExit(1) from None
