#!/usr/bin/env python3
"""Safely update ONLY the existing Homeserver Bridge Stats UI assets.

No systemd restart, nginx reload, bridge config, metrics env or data-plane change.
Run explicitly as root from a clean SSH checkout of work/m3-adaptive-jitter.
"""
from __future__ import annotations

import os
from pathlib import Path
import shutil
import stat
import subprocess
import tempfile
from urllib.request import ProxyHandler, build_opener

APP = Path("/opt/tervon/bridge-stats")
FILES = ("bridge_panel.html", "bridge_panel.js")
BRANCH = "work/m3-adaptive-jitter"
REMOTE = "git@github.com:tervon-software/Tervon-TCP-Bridge.git"
URL = "http://127.0.0.1:8894/bridge/"


def command(*argv: str) -> str:
    result = subprocess.run(argv, capture_output=True, text=True, timeout=10, check=True)
    return result.stdout.strip()


def regular(path: Path) -> bool:
    return path.is_file() and not path.is_symlink()


def preflight(repo: Path, src: Path) -> None:
    if os.geteuid() != 0:
        raise RuntimeError("root required")
    if command("git", "-C", str(repo), "remote", "get-url", "origin") != REMOTE:
        raise RuntimeError("SSH remote mismatch")
    if command("git", "-C", str(repo), "branch", "--show-current") != BRANCH:
        raise RuntimeError("branch mismatch")
    if command("git", "-C", str(repo), "--no-optional-locks", "status", "--porcelain"):
        raise RuntimeError("checkout not clean")
    if not APP.is_dir() or APP.is_symlink():
        raise RuntimeError("stats app directory missing or symlink")
    if not regular(APP / "bridge_site.py") or not regular(APP / "bridge_metrics.py"):
        raise RuntimeError("existing stats app must be preserved")
    for name in FILES:
        if not regular(src / name) or not regular(APP / name):
            raise RuntimeError("missing or symlink asset: " + name)
    if command("systemctl", "is-active", "tervon-bridge-stats.service") != "active":
        raise RuntimeError("stats service inactive")


def write_atomic(path: Path, data: bytes) -> None:
    fd, name = tempfile.mkstemp(prefix=".bridge-ui-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as out:
            out.write(data)
            out.flush()
            os.fsync(out.fileno())
        os.chmod(name, 0o644)
        os.replace(name, path)
    finally:
        if os.path.lexists(name):
            os.unlink(name)


def live_ui_matches(expected: dict[str, bytes]) -> bool:
    opener = build_opener(ProxyHandler({}))
    with opener.open(URL, timeout=3) as response:
        page = response.read(131073)
        if response.status != 200 or len(page) > 131072:
            return False
    with opener.open(URL + "bridge_panel.js", timeout=3) as response:
        script = response.read(131073)
        if response.status != 200 or len(script) > 131072:
            return False
    return expected["bridge_panel.html"] in page and script == expected["bridge_panel.js"]


def install() -> None:
    src = Path(__file__).resolve().parent
    repo = src.parents[1]
    preflight(repo, src)
    expected = {name: (src / name).read_bytes() for name in FILES}
    previous = {name: (APP / name).read_bytes() for name in FILES}
    if expected == previous:
        print("UI_ALREADY_CURRENT=PASS", flush=True)
        return
    backup_root = APP / ".ui-backups"
    if backup_root.is_symlink():
        raise RuntimeError("backup root symlink")
    backup_root.mkdir(mode=0o700, exist_ok=True)
    if not backup_root.is_dir():
        raise RuntimeError("invalid backup root")
    backup = Path(tempfile.mkdtemp(prefix="m3-ui-", dir=backup_root))
    os.chmod(backup, 0o700)
    for name in FILES:
        (backup / name).write_bytes(previous[name])
        os.chmod(backup / name, 0o600)
    print("UI_BACKUP=" + str(backup), flush=True)
    try:
        for name in FILES:
            write_atomic(APP / name, expected[name])
        if not live_ui_matches(expected):
            raise RuntimeError("loopback UI content mismatch")
        if command("systemctl", "is-active", "tervon-bridge-stats.service") != "active":
            raise RuntimeError("stats service inactive after update")
    except BaseException:
        for name in FILES:
            try:
                write_atomic(APP / name, previous[name])
            except Exception:
                print("UI_ROLLBACK_FILE_FAILED=" + name, flush=True)
        print("UI_ROLLBACK=" + ("PASS" if all(
            regular(APP / name) and (APP / name).read_bytes() == previous[name]
            for name in FILES) else "FAIL"), flush=True)
        raise
    print("M3_STATS_UI_LOCAL=PASS", flush=True)
    print("BRIDGE_RESTART=NONE", flush=True)
    print("STATS_RESTART=NONE", flush=True)
    print("NGINX_RELOAD=NONE", flush=True)
    print("EXTERNAL_HTTPS=NOT_VERIFIED", flush=True)


if __name__ == "__main__":
    try:
        install()
    except Exception as exc:
        print("UI_UPDATE_BLOCKED_OR_FAILED=" + str(exc), flush=True)
        raise SystemExit(1) from None
