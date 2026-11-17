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


def _nginx_tokens(source: str) -> list[str]:
    """Minimal fail-closed nginx lexer for auth preflight, not a config editor.

    Handles quoted/escaped braces and comments so nested blocks cannot
    trick a flat regular expression. Rejects malformed syntax.
    """
    tokens: list[str] = []
    word: list[str] = []
    quote: str | None = None
    escaped = False
    comment = False

    def flush() -> None:
        if word:
            tokens.append("".join(word))
            word.clear()

    for char in source:
        if comment:
            if char == "\n":
                comment = False
            continue
        if escaped:
            word.append(char)
            escaped = False
            continue
        if char == "\\":
            escaped = True
            continue
        if quote is not None:
            if char == quote:
                quote = None
            else:
                word.append(char)
            continue
        if char in ('"', "'"):
            quote = char
        elif char == "#":
            flush()
            comment = True
        elif char in "{};":
            flush()
            tokens.append(char)
        elif char.isspace():
            flush()
        else:
            word.append(char)
    if escaped or quote is not None:
        raise ValueError("malformed nginx quoting")
    flush()
    return tokens


def _admin_auth_locations(source: str) -> list[dict]:
    """Collect direct auth directives from explicit /admin/ locations.

    Nested directives do not count as the parent's auth. Nested auth
    overrides or includes are treated as ambiguous and blocked.
    """
    stack: list[dict] = []
    statement: list[str] = []
    locations: list[dict] = []

    for token in _nginx_tokens(source):
        if token == "{":
            if not statement:
                raise ValueError("anonymous nginx block")
            is_location = statement[0] == "location"
            target = statement[1:]
            explicit_admin = (
                is_location
                and target in (["/admin/"], ["^~", "/admin/"],
                               ["=", "/admin/"])
                and not any(frame["is_location"] for frame in stack)
            )
            stack.append({
                "is_location": is_location,
                "candidate": explicit_admin,
                "auth": [],
                "files": [],
                "ambiguous": False,
            })
            statement = []
        elif token == ";":
            if not statement:
                raise ValueError("empty nginx directive")
            name, args = statement[0], statement[1:]
            for parent in stack[:-1]:
                if parent["candidate"] and name in (
                    "auth_basic", "auth_basic_user_file", "include"
                ):
                    parent["ambiguous"] = True
            if stack and stack[-1]["candidate"]:
                frame = stack[-1]
                if name == "auth_basic":
                    frame["auth"].append(args)
                elif name == "auth_basic_user_file":
                    frame["files"].append(args)
                elif name == "include":
                    frame["ambiguous"] = True
            statement = []
        elif token == "}":
            if statement or not stack:
                raise ValueError("malformed nginx block")
            frame = stack.pop()
            if frame["candidate"]:
                locations.append(frame)
        else:
            statement.append(token)

    if statement or stack:
        raise ValueError("incomplete nginx configuration")
    return locations


def _verified_admin_auth(source: str) -> bool:
    locations = _admin_auth_locations(source)
    if not locations:
        return False
    for location in locations:
        if location["ambiguous"]:
            return False
        if len(location["auth"]) != 1 or len(location["files"]) != 1:
            return False
        auth, auth_file = location["auth"][0], location["files"][0]
        if len(auth) != 1 or not auth[0] or auth[0].lower() == "off":
            return False
        if len(auth_file) != 1:
            return False
        path = auth_file[0]
        if not path or "$" in path or not Path(path).is_absolute():
            return False
        if not stat.S_ISREG(Path(path).stat().st_mode):
            return False
    return True


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

    # Check direct auth directives even when /admin/ contains nested blocks.
    # Any ambiguity fails closed; never expose auth file paths or contents.
    try:
        verified = _verified_admin_auth(
            NGINX_CONFIG.read_text(encoding="utf-8")
        )
        record("ADMIN_HTPASSWD", verified,
               "configured admin auth file exists" if verified
               else "explicit /admin/ auth and regular literal file not verified")
    except (OSError, UnicodeError, ValueError):
        record("ADMIN_HTPASSWD", False, "cannot verify admin auth safely")

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
