#!/usr/bin/env python3
"""M6: apply only the approved TCP/raw transport compatibility fix to Android.

All three source and destination Git blob IDs are checked BEFORE writing.
No APK build, VPN configuration, secrets, Docker, or live services are changed.
"""
import argparse
import hashlib
import os
from pathlib import Path
import tempfile

STAGE = "integrations/android-ingress/stage4"
DEST = "android/app/src/main/kotlin/com/tervonvpndpi/app"
SOURCES = {
    "BridgeVpnSession.kt": (
        "bd0db652dfd44c70961df1b2297b968461f5587b",
        "16eab42b49173f55e2d997f7c04d6edb2b58e527",
    ),
    "BridgeVlessRoutePlan.kt": (
        "271f0cf46bb27576cd8f43a3f702e3ad09c9391d",
        "cc2162eac04f456e75eb9a64b2b9466bb1512e97",
    ),
    "BridgeXrayConfigRoute.kt": (
        "79cca5bf957353a52327f07ff86af3a8c5984e01",
        "a0f4e404028b42a2efbbaa309bbbe9c18c57fd2e",
    ),
}


def blob(data: bytes) -> str:
    return hashlib.sha1(b"blob " + str(len(data)).encode() + b"\0" + data).hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo-root", required=True, type=Path)
    parser.add_argument("--app-root", required=True, type=Path)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--apply", action="store_true")
    opts = parser.parse_args()
    repo, app = opts.repo_root.resolve(), opts.app_root.resolve()
    if not (app / "pubspec.yaml").is_file():
        print("M6_TCP_RAW=BLOCKED_ANDROID_PROJECT_NOT_FOUND")
        return 1
    originals = {}
    replacements = {}
    states = []
    for name, (before, after) in SOURCES.items():
        src, target = repo / STAGE / name, app / DEST / name
        if not src.is_file() or not target.is_file() or src.is_symlink() or target.is_symlink():
            print("M6_TCP_RAW=BLOCKED_FILE " + name)
            return 1
        data = src.read_bytes()
        current = target.read_bytes()
        if blob(data) != after or blob(current) not in (before, after):
            print("M6_TCP_RAW=BLOCKED_SHA_MISMATCH " + name)
            return 1
        originals[name] = current
        replacements[name] = data
        states.append(blob(current) == after)
    if all(states):
        print("M6_TCP_RAW=ALREADY_APPLIED")
        return 0
    if any(states):
        print("M6_TCP_RAW=BLOCKED_PARTIAL_SOURCES")
        return 1
    print("M6_TCP_RAW=READY")
    print("M6_CHANGED_FILES=3")
    if opts.check:
        print("M6_APP_SOURCE_UNCHANGED=YES")
        return 0
    changed = []
    try:
        for name in SOURCES:
            target = app / DEST / name
            fd, temporary = tempfile.mkstemp(prefix=".m6-tcp-raw-", dir=target.parent)
            try:
                with os.fdopen(fd, "wb") as output:
                    output.write(replacements[name])
                    output.flush()
                    os.fsync(output.fileno())
                os.chmod(temporary, target.stat().st_mode & 0o777)
                os.replace(temporary, target)
                changed.append(name)
            finally:
                if os.path.exists(temporary):
                    os.unlink(temporary)
    except OSError:
        for name in reversed(changed):
            (app / DEST / name).write_bytes(originals[name])
        print("M6_TCP_RAW=WRITE_FAILED_ROLLBACK_ATTEMPTED")
        return 1
    print("M6_TCP_RAW=SOURCE_APPLIED")
    print("APK_BUILD=NOT_RUN")
    print("LIVE_SERVICES=UNCHANGED")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
