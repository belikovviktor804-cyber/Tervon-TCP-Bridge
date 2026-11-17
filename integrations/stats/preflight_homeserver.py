#!/usr/bin/env python3
"""Read-only preflight for a *new* Homeserver-only Bridge stats service.

No writes, no network requests, no service restarts, no secret output.
Run from an SSH git checkout of Tervon-TCP-Bridge on Homeserver.
"""
from __future__ import annotations

import os
import re
import stat
import subprocess
import sys
from pathlib import Path

EXPECTED_REMOTE = "git@github.com:tervon-software/Tervon-TCP-Bridge.git"
NGINX_CONFIG = Path("/etc/nginx/sites-enabled/stats-temp")
UNIT = Path("/etc/systemd/system/tervon-bridge-stats.service")
APP_DIR = Path("/opt/tervon/bridge-stats")
VENV_PYTHON = Path("/opt/xray-blocker/web-dashboard/venv/bin/python")
BRIDGE_PORT = "8894"


def command(*argv: str) -> tuple[int, str]:
    try:
        completed = subprocess.run(
            argv, capture_output=True, text=True, timeout=8, check=False,
            env={**os.environ, "PYTHONDONTWRITEBYTECODE": "1"},
        )
        return completed.returncode, completed.stdout.strip()
    except (OSError, subprocess.TimeoutExpired):
        return 127, ""


def checks() -> list[tuple[str, str, str]]:
    results: list[tuple[str, str, str]] = []

    def record(name: str, ok: bool, reason: str) -> None:
        results.append((name, "PASS" if ok else "BLOCKED", reason))

    code, remote = command("git", "remote", "get-url", "origin")
    record("GIT_SSH_REMOTE", code == 0 and remote == EXPECTED_REMOTE,
           "expected SSH origin" if code != 0 or remote != EXPECTED_REMOTE
           else "SSH origin verified")
    code, branch = command("git", "branch", "--show-current")
    record("GIT_BRANCH", code == 0 and branch == "work/m1-framing",
           "M1 work branch required" if branch != "work/m1-framing"
           else "M1 work branch")
    code, sha = command("git", "rev-parse", "HEAD")
    record("GIT_HEAD", code == 0 and bool(re.fullmatch(r"[0-9a-f]{40}", sha)),
           "commit unavailable" if code != 0 else "commit verified locally")
    code, status = command("git", "--no-optional-locks", "status", "--porcelain")
    record("GIT_CLEAN", code == 0 and not status,
           "uncommitted changes; do not overwrite" if status else "clean checkout")

    code, listeners = command("ss", "-H", "-ltn", f"( sport = :{BRIDGE_PORT} )")
    record("PORT_8894", code == 0 and not listeners,
           "listener exists or ss unavailable" if listeners or code != 0
           else "no TCP listener")

    record("NEW_UNIT", not UNIT.exists() and not UNIT.is_symlink(),
           "unit already exists" if UNIT.exists() or UNIT.is_symlink()
           else "no conflicting unit file")
    record("APP_DIRECTORY", not APP_DIR.exists() and not APP_DIR.is_symlink(),
           "directory already exists; inspect manually"
           if APP_DIR.exists() or APP_DIR.is_symlink()
           else "new isolated directory available")

    code, versions = command(
        str(VENV_PYTHON), "-c",
        "from importlib.metadata import version;"
        "print(version('fastapi'));print(version('uvicorn'))",
    )
    record("PYTHON_DEPS", code == 0 and len(versions.splitlines()) == 2,
           "existing FastAPI/Uvicorn unavailable" if code != 0
           else "existing dependencies importable")

    code, systemd = command("systemctl", "--version")
    version = re.search(r"(?m)^systemd\s+(\d+)", systemd)
    record("SYSTEMD_VERSION", code == 0 and version is not None
           and int(version.group(1)) >= 235,
           "systemd >=235 needed for IPAddressDeny; runtime sandbox still unverified"
           if not version or int(version.group(1)) < 235
           else "systemd version supports directive; runtime sandbox unverified")

    # Inspect the explicit /admin/ auth boundary. nginx permits optional
    # location modifiers ("^~" and "="); the old regex rejected both.
    # Fail closed on unknown location shapes, nested blocks, missing auth,
    # variables or a non-regular htpasswd. Never print the secret path.
    try:
        source = NGINX_CONFIG.read_text(encoding="utf-8")
        locations = re.findall(
            r"(?m)^[ \t]*location[ \t]+(?:(?:=|\^~)[ \t]+)?"
            r"/admin/[ \t]*\{([^{}]*)\}", source,
        )
        verified = False
        for body in locations:
            auth = re.search(
                r"(?m)^[ \t]*auth_basic[ \t]+([^;]+);", body,
            )
            auth_file = re.search(
                r"(?m)^[ \t]*auth_basic_user_file[ \t]+([^;]+);", body,
            )
            if not auth or auth.group(1).strip().strip('"\'').lower() == "off":
                continue
            if not auth_file:
                continue
            path = auth_file.group(1).strip().strip('"\'')
            if not path or "$" in path or not Path(path).is_absolute():
                continue
            if stat.S_ISREG(Path(path).stat().st_mode):
                verified = True
                break
        record("ADMIN_HTPASSWD", verified,
               "configured admin auth file exists" if verified
               else "explicit /admin/ auth and regular literal file not verified")
    except (OSError, UnicodeError):
        record("ADMIN_HTPASSWD", False, "cannot inspect admin auth file")

    # The application may need read access to the existing venv when run as
    # DynamicUser; this requires a separate on-host permission test.
    results.append(("DYNAMIC_USER_ACCESS", "CHECK",
                    "verify read/execute permissions and systemd sandbox before install"))
    return results


def main() -> int:
    rows = checks()
    for name, result, detail in rows:
        print(f"{name}: {result} — {detail}")
    print("NO_CHANGES: PASS — read-only preflight")
    return 2 if any(result == "BLOCKED" for _, result, _ in rows) else 0


if __name__ == "__main__":
    sys.exit(main())
