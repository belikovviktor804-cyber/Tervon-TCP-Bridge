#!/usr/bin/env python3
"""Enable existing Homeserver Bridge metrics and attach the existing stats app.

One bounded change to the *currently deployed* bridge client and stats service.
No binaries are replaced. No nginx, Finserver, other services or site templates.
Must run on Homeserver as root from a clean SSH Git checkout.
"""
from __future__ import annotations

import json
import os
from pathlib import Path
import shutil
import socket
import stat
import subprocess
import sys
import tempfile
import time
from urllib.error import URLError
from urllib.request import ProxyHandler, build_opener

CONFIG = Path("/etc/tervon-tcp-bridge/client.json")
BACKUP = Path("/etc/tervon-tcp-bridge/client.json.before-metrics")
ENVFILE = Path("/etc/tervon/bridge-stats.env")
BRIDGE_BIN = Path("/opt/tervon-tcp-bridge/bridge")
BRIDGE_UNIT = "tervon-bridge-client.service"
STATS_UNIT = "tervon-bridge-stats.service"
PORT = 8895
METRICS_URL = f"http://127.0.0.1:{PORT}/metrics/v1"
STATUS_URL = "http://127.0.0.1:8894/bridge/api/status"
REMOTE = "git@github.com:tervon-software/Tervon-TCP-Bridge.git"
BRANCH = "work/m1-framing"


def command(*args: str) -> str:
    result = subprocess.run(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                            text=True, check=False, timeout=15)
    if result.returncode:
        raise RuntimeError("command failed: " + " ".join(args[:2]))
    return result.stdout.strip()


def active(unit: str) -> bool:
    return subprocess.run(["systemctl", "is-active", "--quiet", unit],
                          stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                          timeout=5, check=False).returncode == 0


def fetch(url: str) -> dict:
    opener = build_opener(ProxyHandler({}))
    with opener.open(url, timeout=1.5) as response:
        if response.status != 200:
            raise ValueError("HTTP status is not 200")
        raw = response.read(65537)
    if len(raw) > 65536:
        raise ValueError("response too large")
    return json.loads(raw)


def ready_metrics(payload: object) -> bool:
    return (isinstance(payload, dict)
            and payload.get("schema") == "tervon.bridge.telemetry.v1"
            and payload.get("ready") is True
            and type(payload.get("sessions_active")) is int
            and payload["sessions_active"] >= 0)


def ready_dashboard(payload: object) -> bool:
    if not isinstance(payload, dict):
        return False
    if payload.get("schema") != "tervon.bridge.dashboard.v1":
        return False
    nodes = payload.get("nodes")
    if not isinstance(nodes, dict) or set(nodes) != {"homeserver"}:
        return False
    node = nodes["homeserver"]
    return (isinstance(node, dict) and node.get("status") == "ok"
            and isinstance(node.get("metrics"), dict)
            and node["metrics"].get("ready") is True
            and type(node["metrics"].get("sessions_active")) is int)


def await_http(url: str, unit: str, validator, seconds: float = 20) -> dict:
    deadline = time.monotonic() + seconds
    while True:
        if not active(unit):
            raise RuntimeError(f"{unit} not active during readiness wait")
        try:
            payload = fetch(url)
            if validator(payload):
                return payload
        except (URLError, OSError, ValueError, TimeoutError, json.JSONDecodeError):
            pass
        if time.monotonic() >= deadline:
            raise RuntimeError(f"bounded readiness timeout for {unit}")
        time.sleep(min(0.5, max(0.0, deadline - time.monotonic())))


def atomic_write(path: Path, content: bytes, reference: Path) -> None:
    fd, tmp = tempfile.mkstemp(prefix=".bridge-metrics-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as f:
            f.write(content)
            f.flush()
            os.fsync(f.fileno())
        shutil.copystat(reference, tmp)
        os.replace(tmp, path)
    finally:
        if os.path.lexists(tmp):
            os.unlink(tmp)


def preflight() -> dict:
    if os.geteuid() != 0:
        raise RuntimeError("root required")
    repo = Path(__file__).resolve().parents[2]
    if command("git", "-C", str(repo), "remote", "get-url", "origin") != REMOTE:
        raise RuntimeError("unexpected Git remote; SSH required")
    if command("git", "-C", str(repo), "branch", "--show-current") != BRANCH:
        raise RuntimeError("unexpected Git branch")
    if command("git", "-C", str(repo), "--no-optional-locks", "status", "--porcelain"):
        raise RuntimeError("Git checkout is dirty")
    if not BRIDGE_BIN.is_file() or BRIDGE_BIN.is_symlink():
        raise RuntimeError("deployed bridge binary missing or symlink")
    if any(p.is_symlink() for p in (CONFIG, BACKUP, ENVFILE)):
        raise RuntimeError("symlinks forbidden")
    if not CONFIG.is_file() or BACKUP.exists() or ENVFILE.exists():
        raise RuntimeError("config missing or existing backup/env requires review")
    if not active(BRIDGE_UNIT) or not active(STATS_UNIT):
        raise RuntimeError("both existing services must be active")
    effective_exec = command("systemctl", "show", BRIDGE_UNIT, "-p", "ExecStart", "--value")
    if (f"path={BRIDGE_BIN} " not in effective_exec or
            f"-config {CONFIG}" not in effective_exec):
        raise RuntimeError("unexpected bridge ExecStart")
    with CONFIG.open(encoding="utf-8") as f:
        config = json.load(f)
    if not isinstance(config, dict) or config.get("mode") != "client":
        raise RuntimeError("client config expected")
    if config.get("metrics_listen"):
        raise RuntimeError("metrics already configured")
    if not config.get("listen") or config.get("listen") == f"127.0.0.1:{PORT}":
        raise RuntimeError("unexpected client listen")
    # Probe address availability without involving existing listeners/services.
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", PORT))
    if not Path("/opt/tervon/bridge-stats/bridge_site.py").is_file():
        raise RuntimeError("existing stats application missing")
    if not Path("/opt/tervon/bridge-stats/bridge_metrics.py").is_file():
        raise RuntimeError("existing collector missing")
    return config


def activate() -> None:
    config = preflight()
    print("PREFLIGHT=PASS", flush=True)
    original = CONFIG.read_bytes()
    updated = dict(config)
    updated["metrics_listen"] = f"127.0.0.1:{PORT}"
    # The envfile contains only this literal, non-secret URL. 0644 is required
    # for systemd DynamicUser; 0600 root-only caused the earlier issue.
    env_text = f"TERVON_BRIDGE_METRICS_URL={METRICS_URL}\n".encode()
    config_modified = False
    env_created = False
    bridge_restarted = False
    stats_restarted = False
    shutil.copy2(CONFIG, BACKUP)
    os.chmod(BACKUP, 0o600)
    try:
        atomic_write(CONFIG, (json.dumps(updated, ensure_ascii=False, indent=2) + "\n").encode(), BACKUP)
        config_modified = True
        print("CONFIG_UPDATED=PASS", flush=True)

        # Restart only the existing client; M2 padding keys remain unchanged.
        bridge_restarted = True
        command("systemctl", "restart", BRIDGE_UNIT)
        m = await_http(METRICS_URL, BRIDGE_UNIT, ready_metrics)
        print("M2_LOCAL_METRICS=PASS", flush=True)
        print("ACTIVE_SESSIONS=" + str(m["sessions_active"]), flush=True)

        fd = os.open(ENVFILE, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o644)
        env_created = True
        with os.fdopen(fd, "wb") as f:
            f.write(env_text)
        os.chmod(ENVFILE, 0o644)
        stats_restarted = True
        command("systemctl", "restart", STATS_UNIT)
        d = await_http(STATUS_URL, STATS_UNIT, ready_dashboard)
        print("DASHBOARD_METRICS=PASS", flush=True)
        print("DASHBOARD_SESSIONS=" + str(d["nodes"]["homeserver"]["metrics"]["sessions_active"]), flush=True)
    except Exception:
        print("ACTIVATION=FAILED; ROLLBACK STARTED", flush=True)
        if config_modified:
            atomic_write(CONFIG, original, BACKUP)
        if env_created:
            ENVFILE.unlink(missing_ok=True)
        # Restore only services actually restarted; report failure separately.
        errors = []
        if bridge_restarted:
            try:
                command("systemctl", "restart", BRIDGE_UNIT)
                if not active(BRIDGE_UNIT):
                    raise RuntimeError("Bridge not active after rollback")
            except Exception:
                errors.append("BRIDGE")
        if stats_restarted:
            try:
                command("systemctl", "restart", STATS_UNIT)
                if not active(STATS_UNIT):
                    raise RuntimeError("Stats not active after rollback")
            except Exception:
                errors.append("STATS")
        print("ROLLBACK_SERVICES=" + ("FAILED:" + ",".join(errors) if errors else "ACTIVE"), flush=True)
        print("ROLLBACK_CONFIG=" + ("PASS" if CONFIG.read_bytes() == original else "FAIL"), flush=True)
        raise
    print("METRICS_ACTIVATION=PASS", flush=True)
    print("M2_CONFIG_PRESERVED=PASS", flush=True)
    print("BACKUP=" + str(BACKUP), flush=True)


def main() -> int:
    if len(sys.argv) != 2 or sys.argv[1] != "activate":
        print("Usage: sudo python3 integrations/stats/activate-m2-metrics.py activate")
        return 2
    try:
        activate()
        return 0
    except Exception as exc:
        print("BLOCKED_OR_FAILED=" + str(exc), flush=True)
        return 1


if __name__ == "__main__":
    sys.exit(main())
