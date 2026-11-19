#!/usr/bin/env python3
"""M3 Finserver Stats opt-in: local-only sources, backed-up Stats-only rollout."""
from __future__ import annotations

import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time
from urllib.request import ProxyHandler, build_opener

REPO_REMOTE = "git@github.com:tervon-software/Tervon-TCP-Bridge.git"
BRANCH = "work/m3-adaptive-jitter"
APP = Path("/opt/tervon/bridge-stats")
DROPIN = Path("/etc/systemd/system/tervon-bridge-stats.service.d/fin-metrics.conf")
SERVICE = "tervon-bridge-stats.service"
FILES = ("bridge_site.py", "bridge_panel.html", "bridge_panel.js")
FIN = "http://127.0.0.1:8897/metrics/v1"
API = "http://127.0.0.1:8894/bridge/api/status"

def run(*args):
    p = subprocess.run(args, text=True, capture_output=True, check=True, timeout=25)
    return p.stdout.strip()

def get(url):
    with build_opener(ProxyHandler({})).open(url, timeout=3) as resp:
        if resp.status != 200:
            raise RuntimeError("HTTP status not 200")
        data = resp.read(65537)
    if len(data) > 65536:
        raise RuntimeError("telemetry payload too large")
    return json.loads(data)

def verify(expect_fin):
    if run("systemctl", "is-active", SERVICE) != "active":
        raise RuntimeError("Stats is not active")
    data = get(API)
    if data.get("schema") != "tervon.bridge.dashboard.v1":
        raise RuntimeError("bad dashboard schema")
    nodes = data.get("nodes", {})
    if nodes.get("homeserver", {}).get("status") != "ok":
        raise RuntimeError("Homeserver dashboard unavailable")
    if expect_fin:
        fin = nodes.get("finserver", {})
        if fin.get("status") != "ok" or fin.get("metrics", {}).get("ready") is not True:
            raise RuntimeError("Finserver dashboard unavailable")
    return data

def atomic(path, contents, mode):
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, name = tempfile.mkstemp(prefix=".bridge-fin-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as out:
            out.write(contents)
            out.flush()
            os.fsync(out.fileno())
        os.chmod(name, mode)
        os.replace(name, path)
    finally:
        if os.path.exists(name):
            os.unlink(name)

def main():
    if os.geteuid() != 0:
        raise RuntimeError("root required")
    repo = Path(__file__).resolve().parents[2]
    if run("git", "-C", str(repo), "remote", "get-url", "origin") != REPO_REMOTE:
        raise RuntimeError("origin is not GitHub SSH")
    if run("git", "-C", str(repo), "branch", "--show-current") != BRANCH:
        raise RuntimeError("wrong branch")
    if run("git", "-C", str(repo), "status", "--porcelain"):
        raise RuntimeError("dirty source checkout")
    if any(not (APP / name).is_file() or (APP / name).is_symlink() for name in FILES):
        raise RuntimeError("installed Stats files missing or symlink")
    if DROPIN.exists() or DROPIN.is_symlink():
        raise RuntimeError("Finserver drop-in already exists; refuse overwrite")
    if any(not (repo / "integrations/stats" / name).is_file() for name in FILES):
        raise RuntimeError("source files missing")
    old = verify(False)
    raw = get(FIN)
    if raw.get("schema") != "tervon.bridge.telemetry.v1" or raw.get("ready") is not True:
        raise RuntimeError("SSH forwarded Finserver metrics not ready")

    backup_parent = APP / ".two-node-backups"
    if backup_parent.is_symlink():
        raise RuntimeError("backup parent symlink")
    backup_parent.mkdir(mode=0o700, exist_ok=True)
    backup = Path(tempfile.mkdtemp(prefix="m3-", dir=backup_parent))
    os.chmod(backup, 0o700)
    saved = {}
    for name in FILES:
        dst = APP / name
        saved[name] = (dst.read_bytes(), dst.stat().st_mode & 0o777)
        atomic(backup / name, saved[name][0], 0o600)
    print("STATS_BACKUP=" + str(backup), flush=True)

    # Only the isolated Stats app changes. No Bridge, nginx, or Finserver
    # services are modified by this deployment.
    changed = False
    try:
        for name in FILES:
            src = repo / "integrations/stats" / name
            atomic(APP / name, src.read_bytes(), saved[name][1])
        DROPIN.parent.mkdir(parents=True, exist_ok=True)
        atomic(DROPIN,
               ('[Service]\nEnvironment="TERVON_BRIDGE_FIN_METRICS_URL=' + FIN + '"\n').encode(),
               0o644)
        changed = True
        run("systemctl", "daemon-reload")
        subprocess.run(["systemctl", "restart", SERVICE], capture_output=True, timeout=20)
        deadline = time.monotonic() + 25
        while True:
            try:
                verify(True)
                break
            except Exception:
                if time.monotonic() >= deadline:
                    raise RuntimeError("Stats two-node readiness timeout")
                time.sleep(1)
    except BaseException:
        print("STATS_TWO_NODE_ROLLBACK=STARTED", flush=True)
        for name in FILES:
            if name in saved:
                atomic(APP / name, saved[name][0], saved[name][1])
        if DROPIN.exists():
            DROPIN.unlink()
        run("systemctl", "daemon-reload")
        subprocess.run(["systemctl", "restart", SERVICE], capture_output=True, timeout=20)
        # Previous one-node state must still work, even if tunnel is lost.
        deadline = time.monotonic() + 20
        while True:
            try:
                verify(False)
                print("STATS_TWO_NODE_ROLLBACK=PASS", flush=True)
                break
            except Exception:
                if time.monotonic() >= deadline:
                    print("STATS_TWO_NODE_ROLLBACK=FAILED", flush=True)
                    break
                time.sleep(1)
        raise

    print("M3_TWO_NODE_STATS_LOCAL=PASS", flush=True)
    print("FIN_SOURCE=SSH_LOOPBACK_8897", flush=True)
    print("BRIDGE_RESTART=NONE", flush=True)
    print("NGINX_RELOAD=NONE", flush=True)
    print("EXTERNAL_HTTPS=NOT_VERIFIED", flush=True)

if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print("STATS_TWO_NODE_INSTALL_FAILED=" + str(exc), flush=True)
        raise SystemExit(1) from None
