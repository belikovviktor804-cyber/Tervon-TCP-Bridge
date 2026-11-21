#!/usr/bin/env python3
"""M6 approved auto-discovery correction: three-file guarded source-only update.

No APK, running services, endpoint defaults, subscription values or live configuration.
"""
import argparse
import hashlib
import os
from pathlib import Path
import tempfile

FILES = {
    "BridgeAutoEnrollmentHttp.kt": (
        "3ea1259f96b528eee06e30c03430eb4448de87f2",
        "c2a3d6d0d29046ee5033dac23cbba580627448e1",
    ),
    "BridgeVlessRoutePlan.kt": (
        "79b39d6e0d5f7072695802c3133431dd55536bed",
        "271f0cf46bb27576cd8f43a3f702e3ad09c9391d",
    ),
    "BridgeVpnSession.kt": (
        "f49ee06c199c7aef4c907ac15e1165f1fe0d6bfc",
        "50d966aed9dd4311eb29e5f713a1d50181a53eed",
    ),
}
DEST = "android/app/src/main/kotlin/com/tervonvpndpi/app"
SRC = "integrations/android-ingress/stage4"


def blob(data):
    return hashlib.sha1(b"blob " + str(len(data)).encode() + b"\0" + data).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--app-root", required=True, type=Path)
    parser.add_argument("--repo-root", required=True, type=Path)
    actions = parser.add_mutually_exclusive_group(required=True)
    actions.add_argument("--check", action="store_true")
    actions.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    repo = args.repo_root.resolve()
    app = args.app_root.resolve()
    if not (app / "pubspec.yaml").is_file():
        print("M6_AUTODISCOVERY=BLOCKED: NOT_ANDROID_PROJECT")
        return 1
    old = {}
    new = {}
    states = []
    for name, (expected_before, expected_after) in FILES.items():
        target = app / DEST / name
        source = repo / SRC / name
        if not target.is_file() or not source.is_file() or target.is_symlink() or source.is_symlink():
            print("M6_AUTODISCOVERY=BLOCKED: FILE_UNAVAILABLE " + name)
            return 1
        current = target.read_bytes()
        wanted = source.read_bytes()
        if blob(wanted) != expected_after:
            print("M6_AUTODISCOVERY=BLOCKED: SOURCE_SHA_MISMATCH " + name)
            return 1
        actual = blob(current)
        if actual not in (expected_before, expected_after):
            print("M6_AUTODISCOVERY=BLOCKED: UNKNOWN_TARGET " + name)
            return 1
        old[name] = current
        new[name] = wanted
        states.append(actual == expected_after)
    if all(states):
        print("M6_AUTODISCOVERY=ALREADY_APPLIED")
        return 0
    if any(states):
        print("M6_AUTODISCOVERY=BLOCKED: PARTIALLY_UPDATED")
        return 1
    print("M6_AUTODISCOVERY=READY")
    print("M6_CHANGED_FILES=3")
    if args.check:
        print("M6_APP_SOURCE_UNCHANGED=YES")
        return 0
    completed = []
    try:
        for name in FILES:
            dest = app / DEST / name
            fd, staged = tempfile.mkstemp(prefix=".m6-autodiscovery-", dir=str(dest.parent))
            try:
                with os.fdopen(fd, "wb") as f:
                    f.write(new[name])
                    f.flush()
                    os.fsync(f.fileno())
                os.chmod(staged, dest.stat().st_mode & 0o777)
                os.replace(staged, dest)
                completed.append(name)
            finally:
                if os.path.exists(staged):
                    os.unlink(staged)
    except OSError:
        for name in reversed(completed):
            (app / DEST / name).write_bytes(old[name])
        print("M6_AUTODISCOVERY=BLOCKED: WRITE_FAILED_ROLLBACK_ATTEMPTED")
        return 1
    print("M6_AUTODISCOVERY=SOURCE_APPLIED")
    print("M6_APK_BUILD=NOT_RUN")
    print("M6_LIVE_SERVICES=UNCHANGED")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
