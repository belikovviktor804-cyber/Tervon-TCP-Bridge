#!/usr/bin/env python3
"""M6: guarded replacement of existing BridgeVpnSession failure reporting.

No changes to VPN route logic, keys, APK signing, running services or config.
"""
import argparse
import hashlib
import os
from pathlib import Path
import tempfile

SOURCE = "integrations/android-ingress/stage4/BridgeVpnSession.kt"
TARGET = "android/app/src/main/kotlin/com/tervonvpndpi/app/BridgeVpnSession.kt"
BEFORE = "50d966aed9dd4311eb29e5f713a1d50181a53eed"
AFTER = "bd0db652dfd44c70961df1b2297b968461f5587b"

def blob(data):
    return hashlib.sha1(b"blob " + str(len(data)).encode() + b"\0" + data).hexdigest()

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--app-root", type=Path, required=True)
    parser.add_argument("--repo-root", type=Path, required=True)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    app = args.app_root.resolve()
    src = args.repo_root.resolve() / SOURCE
    dest = app / TARGET
    if not (app / "pubspec.yaml").is_file() or not dest.is_file() or not src.is_file() or src.is_symlink() or dest.is_symlink():
        print("M6_FAILURE_RUNTIME=BLOCKED_FILES")
        return 1
    content = src.read_bytes()
    if blob(content) != AFTER:
        print("M6_FAILURE_RUNTIME=BLOCKED_SOURCE_SHA")
        return 1
    current = blob(dest.read_bytes())
    if current == AFTER:
        print("M6_FAILURE_RUNTIME=ALREADY_APPLIED")
        return 0
    if current != BEFORE:
        print("M6_FAILURE_RUNTIME=BLOCKED_UNEXPECTED_TARGET_SHA")
        return 1
    print("M6_FAILURE_RUNTIME=READY")
    if args.check:
        print("M6_ANDROID_SOURCE_UNCHANGED=YES")
        return 0
    fd, temporary = tempfile.mkstemp(prefix=".m6-failure-runtime-", dir=dest.parent)
    try:
        with os.fdopen(fd, "wb") as output:
            output.write(content)
            output.flush()
            os.fsync(output.fileno())
        os.chmod(temporary, dest.stat().st_mode & 0o777)
        os.replace(temporary, dest)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)
    print("M6_FAILURE_RUNTIME=SOURCE_APPLIED")
    print("M6_LIVE_SERVICES=UNCHANGED")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
