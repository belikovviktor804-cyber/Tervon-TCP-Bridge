#!/usr/bin/env python3
"""M6 UI only: guarded replacement of the already installed screen."""
import argparse
import hashlib
import os
from pathlib import Path
import tempfile

SOURCE = "integrations/android-ingress/stage4/bridge_settings_screen.dart"
TARGET = "lib/features/home/bridge_settings_screen.dart"
BEFORE = "def9471988ff1cdb0e0ddec5986b379d6c62a88d"
AFTER = "a0ae6693795920da6ec948cce0d238f35585dd67"

def git_blob(content):
    return hashlib.sha1(b"blob " + str(len(content)).encode() + b"\0" + content).hexdigest()

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo-root", type=Path, required=True)
    parser.add_argument("--app-root", type=Path, required=True)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    src = args.repo_root.resolve() / SOURCE
    dest = args.app_root.resolve() / TARGET
    if not (args.app_root.resolve() / "pubspec.yaml").is_file() or not src.is_file() or not dest.is_file() or src.is_symlink() or dest.is_symlink():
        print("M6_UI=BLOCKED: SOURCE_OR_TARGET_UNAVAILABLE")
        return 1
    updated = src.read_bytes()
    old = dest.read_bytes()
    if git_blob(updated) != AFTER:
        print("M6_UI=BLOCKED: SOURCE_SHA_MISMATCH")
        return 1
    current = git_blob(old)
    if current == AFTER:
        print("M6_UI=ALREADY_APPLIED")
        return 0
    if current != BEFORE:
        print("M6_UI=BLOCKED: UNKNOWN_TARGET_SHA")
        return 1
    print("M6_UI=READY")
    if args.check:
        print("M6_UI_APP_UNCHANGED=YES")
        return 0
    name = None
    try:
        fd, name = tempfile.mkstemp(prefix=".m6-ui-", dir=str(dest.parent))
        with os.fdopen(fd, "wb") as stream:
            stream.write(updated)
            stream.flush()
            os.fsync(stream.fileno())
        os.chmod(name, dest.stat().st_mode & 0o777)
        os.replace(name, dest)
    except OSError:
        print("M6_UI=BLOCKED: WRITE_FAILED")
        return 1
    finally:
        if name is not None and os.path.exists(name):
            os.unlink(name)
    print("M6_UI=SOURCE_APPLIED")
    print("M6_APK_BUILD=NOT_RUN")
    print("M6_LIVE_SERVICES=UNCHANGED")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
