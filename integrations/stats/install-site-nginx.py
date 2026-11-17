#!/usr/bin/env python3
"""Guarded, reversible integration of public /bridge/ into existing stats HTTPS vhost.

No changes to Bridge, admin auth, infrastructure dashboard, DNS, TLS or firewall.
Run as root on Homeserver only after the isolated stats service has passed.
"""
from __future__ import annotations

import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys
import tempfile
from urllib.request import ProxyHandler, build_opener

SITE = Path("/etc/nginx/sites-enabled/stats-temp")
SNIPPET_DIR = Path("/etc/nginx/tervon-bridge-stats")
SNIPPET = SNIPPET_DIR / "locations.conf"
BACKUP_DIR = Path("/var/lib/tervon/bridge-stats/nginx-backup")
BACKUP = BACKUP_DIR / "stats-temp.before-bridge"
TEMPLATE = Path(__file__).resolve().parent / "nginx-bridge.conf.example"
REMOTE = "git@github.com:tervon-software/Tervon-TCP-Bridge.git"
MARKER = "    include /etc/nginx/tervon-bridge-stats/locations.conf;\n"

SERVER = re.compile(r"^\s*server\s*\{\s*(?:#.*)?$")
HTTPS = re.compile(r"^\s*listen\s+[^;]*\b443\b[^;]*;")
NAME = re.compile(r"^\s*server_name\s+[^;]*\bstats\.xrayvless\.online\b")
BRIDGE = re.compile(r"^\s*location\s+(?:(?:=|\^~|~\*|~)\s+)?/bridge(?:/|\s|\{)")
OTHER_INCLUDES = re.compile(r"^\s*include\s+")


def run(args: list[str], *, cwd: Path | None = None) -> subprocess.CompletedProcess:
    return subprocess.run(
        args, cwd=cwd, text=True, stdout=subprocess.PIPE,
        stderr=subprocess.PIPE, timeout=20, check=False,
    )


def assert_git(repo: Path) -> None:
    checks = (
        (["git", "remote", "get-url", "origin"], REMOTE),
        (["git", "branch", "--show-current"], "work/m1-framing"),
    )
    for args, expected in checks:
        result = run(args, cwd=repo)
        if result.returncode != 0 or result.stdout.strip() != expected:
            raise RuntimeError("Git remote or branch does not match expected SSH checkout")
    result = run(["git", "--no-optional-locks", "status", "--porcelain"], cwd=repo)
    if result.returncode != 0 or result.stdout.strip():
        raise RuntimeError("Git checkout must be clean")


def nginx_test() -> None:
    result = run(["nginx", "-t"])
    if result.returncode != 0:
        raise RuntimeError("nginx -t failed (details retained on host)")


def effective_config() -> str:
    result = run(["nginx", "-T"])
    if result.returncode != 0:
        raise RuntimeError("nginx -T failed")
    return result.stdout


def ensure_no_existing_bridge(config: str) -> None:
    # Fail closed even for another vhost's Bridge location; never overwrite it.
    if any(BRIDGE.match(line) for line in config.splitlines()):
        raise RuntimeError("Bridge location already exists in effective nginx config")
    if str(SNIPPET) in config:
        raise RuntimeError("Bridge snippet is already referenced")


def render_site(original: str) -> str:
    """Insert exactly one include after HTTPS listen of stats vhost.

    The Homeserver site is known to have two server{} blocks, and one 443
    listen immediately after the second opener. Anything else is BLOCKED,
    rather than guessing where to edit.
    """
    lines = original.splitlines(keepends=True)
    starts = [i for i, line in enumerate(lines) if SERVER.match(line)]
    https = [i for i, line in enumerate(lines) if HTTPS.match(line)]
    names = [i for i, line in enumerate(lines) if NAME.match(line)]
    if len(starts) != 2 or len(https) != 1 or len(names) != 2:
        raise ValueError("unexpected stats vhost layout")
    if not (starts[1] < https[0] <= starts[1] + 4):
        raise ValueError("HTTPS listen is not near expected second server opener")
    if not (https[0] < names[1] <= starts[1] + 8):
        raise ValueError("HTTPS server_name does not match expected vhost")
    if MARKER.strip() in original:
        raise ValueError("Bridge include already present")
    if any(BRIDGE.match(line) for line in lines):
        raise ValueError("Bridge location already present")
    lines.insert(https[0] + 1, MARKER)
    return "".join(lines)


def atomic_copy_bytes(path: Path, data: bytes, reference: Path) -> None:
    fd, temporary = tempfile.mkstemp(prefix=".tervon-bridge-", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as stream:
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())
        # Preserve the live vhost's metadata, not the private 0600 backup.
        shutil.copystat(path if path.exists() else reference,
                        temporary, follow_symlinks=False)
        os.replace(temporary, path)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


def assert_local_service() -> None:
    status = run(["systemctl", "is-active", "--quiet",
                  "tervon-bridge-stats.service"])
    if status.returncode != 0:
        raise RuntimeError("isolated stats service is not active")
    opener = build_opener(ProxyHandler({}))
    with opener.open("http://127.0.0.1:8894/bridge/api/status", timeout=3) as response:
        if response.status != 200:
            raise RuntimeError("isolated stats API did not return HTTP 200")


def assert_safe_paths() -> None:
    if SITE.is_symlink() or not SITE.is_file():
        raise RuntimeError("stats site missing or symlink")
    if SNIPPET_DIR.is_symlink() or SNIPPET.exists() or SNIPPET.is_symlink():
        raise RuntimeError("Bridge snippet already exists or directory unsafe")
    if BACKUP_DIR.is_symlink() or BACKUP.exists() or BACKUP.is_symlink():
        raise RuntimeError("Bridge backup already exists or directory unsafe")
    if not TEMPLATE.is_file() or TEMPLATE.is_symlink():
        raise RuntimeError("Bridge template missing or symlink")


def install(repo: Path) -> None:
    assert_git(repo)
    assert_safe_paths()
    nginx_test()
    ensure_no_existing_bridge(effective_config())
    assert_local_service()
    original = SITE.read_bytes()
    updated = render_site(original.decode("utf-8")).encode("utf-8")
    snippet = TEMPLATE.read_bytes()
    if b"auth_basic off;" not in snippet or snippet.count(b"location ") != 2:
        raise RuntimeError("unexpected Bridge nginx template")

    BACKUP_DIR.mkdir(mode=0o700, parents=True, exist_ok=True)
    os.chmod(BACKUP_DIR, 0o700)
    SNIPPET_DIR.mkdir(mode=0o755, parents=True, exist_ok=True)
    shutil.copy2(SITE, BACKUP)
    os.chmod(BACKUP, 0o600)
    site_modified = False
    snippet_written = False
    try:
        fd = os.open(SNIPPET, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o644)
        snippet_written = True
        with os.fdopen(fd, "wb") as stream:
            stream.write(snippet)
        atomic_copy_bytes(SITE, updated, BACKUP)
        site_modified = True
        nginx_test()
        result = run(["systemctl", "reload", "nginx"])
        if result.returncode != 0:
            raise RuntimeError("nginx reload failed")
        print("NGINX_SYNTAX=PASS")
        print("NGINX_RELOAD=PASS")
        print("PUBLIC_BRIDGE_LOCATIONS=INSTALLED")
        print("BACKUP=AVAILABLE_FOR_ROLLBACK")
        print("EXTERNAL_DNS_TLS_CHECK=NOT_YET_VERIFIED")
    except Exception:
        if site_modified:
            atomic_copy_bytes(SITE, original, BACKUP)
        if snippet_written:
            SNIPPET.unlink(missing_ok=True)
        # If reload failed, original nginx worker configuration remains active.
        # Validate restored files but do not change unrelated running services.
        try:
            nginx_test()
            print("NGINX_CONFIG_ROLLBACK=PASS")
        except Exception:
            print("NGINX_CONFIG_ROLLBACK=NEEDS_MANUAL_REVIEW")
        raise


def rollback(repo: Path) -> None:
    assert_git(repo)
    if not BACKUP.is_file() or BACKUP.is_symlink():
        raise RuntimeError("rollback backup not found")
    if not SITE.is_file() or SITE.is_symlink() or not SNIPPET.is_file():
        raise RuntimeError("expected installed files not found")
    current = SITE.read_bytes()
    if current.count(MARKER.encode()) != 1:
        raise RuntimeError("Bridge include marker missing or duplicated")
    original = BACKUP.read_bytes()
    atomic_copy_bytes(SITE, original, BACKUP)
    try:
        nginx_test()
        result = run(["systemctl", "reload", "nginx"])
        if result.returncode != 0:
            raise RuntimeError("nginx reload failed during rollback")
        SNIPPET.unlink()
        print("NGINX_ROLLBACK=PASS")
    except Exception:
        atomic_copy_bytes(SITE, current, BACKUP)
        nginx_test()
        print("NGINX_ROLLBACK=BLOCKED; restored pre-rollback files")
        raise


def main() -> int:
    if os.geteuid() != 0:
        print("BLOCKED: root required")
        return 2
    if len(sys.argv) != 2 or sys.argv[1] not in ("install", "rollback"):
        print("usage: install-site-nginx.py install|rollback")
        return 2
    try:
        repo = Path(__file__).resolve().parents[2]
        if sys.argv[1] == "install":
            install(repo)
        else:
            rollback(repo)
        return 0
    except Exception as exc:
        print("BLOCKED:", str(exc))
        return 1


if __name__ == "__main__":
    sys.exit(main())
