#!/usr/bin/env python3
"""Guarded installation of the existing M6 Bridge Monitoring screen.

Installs exactly one existing Dart screen and its existing Android diagnostics
provider. Does not touch network services, Xray, APK/version/signing or Gradle.
"""
import argparse
import hashlib
import os
from pathlib import Path
import sys
import tempfile

FILES = (
    (
        "lib/features/home/bridge_monitor_screen.dart",
        "bridge_monitor_screen.dart",
        "13e68dee54eae64f5e241680e54669f74a951e08",  # operator source archive
        "5cb6ecb82d9e9aacfe23d0a417448d12b8cdf371",  # M6 Dart nullable fix
    ),
    (
        "android/app/src/main/kotlin/com/tervonvpndpi/app/BridgeDiagnostics.kt",
        "BridgeDiagnostics.kt",
        "336668663fd70ad3d49ccadad1b8ea0b05ea5a75",  # operator source archive
        "e936f8ec4b57edcd7c1b48a3d2b1659dbbbef1cc",  # reviewed GitHub provider
    ),
)


def git_blob(data: bytes) -> str:
    return hashlib.sha1(b"blob " + str(len(data)).encode("ascii") + b"\0" + data).hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--app-root", required=True, type=Path)
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--check", action="store_true")
    group.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    root = args.app_root.resolve(strict=True)
    if not (root / "pubspec.yaml").is_file():
        print("M6_MONITOR=FAIL: NOT_A_FLUTTER_PROJECT")
        return 1
    staged = []
    for relative, name, original_sha, current_sha in FILES:
        target = root / relative
        source = Path(__file__).resolve().parent / name
        if target.is_symlink() or source.is_symlink() or not target.is_file() or not source.is_file():
            print("M6_MONITOR=FAIL: FILE_MISSING_OR_SYMLINK: " + name)
            return 1
        target_sha = git_blob(target.read_bytes())
        source_bytes = source.read_bytes()
        if git_blob(source_bytes) != current_sha:
            print("M6_MONITOR=FAIL: UNKNOWN_SOURCE_REVISION: " + name)
            return 1
        if target_sha == current_sha:
            continue
        if target_sha != original_sha:
            print("M6_MONITOR=FAIL: UNRECOGNIZED_APP_SOURCE: " + name)
            return 1
        staged.append((target, source_bytes))

    print("M6_MONITOR=READY")
    print("M6_MONITOR_CHANGED_FILES=" + str(len(staged)))
    if args.check:
        print("M6_MONITOR_APP_SOURCE_UNCHANGED=YES")
        return 0
    for target, data in staged:
        fd, temp = tempfile.mkstemp(prefix=".m6-monitor-", dir=str(target.parent))
        try:
            with os.fdopen(fd, "wb") as out:
                out.write(data)
                out.flush()
                os.fsync(out.fileno())
            os.chmod(temp, 0o644)
            os.replace(temp, target)
        finally:
            if os.path.exists(temp):
                os.unlink(temp)
    print("M6_MONITOR=SOURCE_APPLIED")
    print("M6_MONITOR_APK_BUILD=NOT_RUN")
    return 0


if __name__ == "__main__":
    sys.exit(main())
