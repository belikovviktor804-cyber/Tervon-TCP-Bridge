#!/usr/bin/env python3
"""Guarded M6 Android server-pin selfcheck source staging.

ONLY modifies BridgeServerCatalogTrust.kt, BridgeControlChannel.kt,
bridge_server_trust_screen.dart; creates BridgeServerCatalogTrustSelfCheck.kt.
The old Kotlin pin and Dart screen are verified against fixed Git commits
before modification. No network, production pins, APK installation or services.
"""
from __future__ import annotations

import argparse
import os
from pathlib import Path
import subprocess
import sys
import tempfile

SOURCE = Path(__file__).resolve().parent
REPO = SOURCE.parents[2]
KOTLIN = Path("android/app/src/main/kotlin/com/tervonvpndpi/app")
PIN = KOTLIN / "BridgeServerCatalogTrust.kt"
CHECK = KOTLIN / "BridgeServerCatalogTrustSelfCheck.kt"
CHANNEL = KOTLIN / "BridgeControlChannel.kt"
SCREEN = Path("lib/features/home/bridge_server_trust_screen.dart")
OLD_PIN_COMMIT = "4dba57d6c3670debee560e71e896dbd7e9936cd9"
OLD_SCREEN_COMMIT = "898873655d35d46a3454fd3287df76cfd4053e2a"
CHANNEL_ANCHOR = '                "confirmServerCatalogKey" -> {'
CHANNEL_BRANCH = '''                "serverCatalogPinSelfCheck" -> Thread({
                    try {
                        val passed = BridgeServerCatalogTrustSelfCheck.verify(app)
                        android.os.Handler(android.os.Looper.getMainLooper()).post {
                            result.success(mapOf("result" to if (passed) "PASS" else "FAIL"))
                        }
                    } catch (_: Exception) {
                        android.os.Handler(android.os.Looper.getMainLooper()).post {
                            result.error("SERVER_PIN_SELF_CHECK_FAILED", null, null)
                        }
                    }
                }, "m6-server-pin-selfcheck").start()
'''


def historical(path: Path, commit: str) -> bytes:
    command = ["git", "-C", str(REPO), "show",
               commit + ":integrations/android-ingress/stage4/" + path.name]
    proc = subprocess.run(command, capture_output=True)
    if proc.returncode or not proc.stdout:
        raise ValueError("cannot verify historical source: " + path.name)
    return proc.stdout


def guarded_source(target: Path, new: bytes, old: bytes) -> tuple[bool, bytes | None]:
    if target.is_symlink() or (target.exists() and not target.is_file()):
        raise ValueError("unsafe target: " + target.name)
    if not target.exists():
        raise ValueError("expected previously installed source missing: " + target.name)
    original = target.read_bytes()
    if original == new:
        return False, original
    if original != old:
        raise ValueError("refuse unexpected existing source: " + target.name)
    return True, original


def replace_atomic(target: Path, content: bytes) -> None:
    fd, tmp = tempfile.mkstemp(prefix=".m6-pin-", dir=target.parent)
    try:
        with os.fdopen(fd, "wb") as output:
            output.write(content)
            output.flush()
            os.fsync(output.fileno())
        os.chmod(tmp, target.stat().st_mode & 0o777)
        os.replace(tmp, target)
    finally:
        if os.path.exists(tmp):
            os.unlink(tmp)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--app-root", required=True, type=Path)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    app = args.app_root
    try:
        if not (app / "android/gradlew").is_file():
            raise ValueError("unknown Android application checkout")
        if not all((app / p).parent.is_dir() for p in (PIN, CHECK, CHANNEL, SCREEN)):
            raise ValueError("missing target source directory")
        if any((app / p).is_symlink() for p in (PIN, CHECK, CHANNEL, SCREEN)):
            raise ValueError("refuse symlink source")
        old_pin = historical(PIN, OLD_PIN_COMMIT)
        old_screen = historical(SCREEN, OLD_SCREEN_COMMIT)
        new_pin = (SOURCE / PIN.name).read_bytes()
        new_screen = (SOURCE / SCREEN.name).read_bytes()
        new_check = (SOURCE / CHECK.name).read_bytes()
        pin_changed, pin_original = guarded_source(app / PIN, new_pin, old_pin)
        screen_changed, screen_original = guarded_source(
            app / SCREEN, new_screen, old_screen
        )
        check_target = app / CHECK
        if check_target.exists() and check_target.read_bytes() != new_check:
            raise ValueError("refuse unexpected existing diagnostics Kotlin")
        check_changed = not check_target.exists()
        channel_target = app / CHANNEL
        channel_original = channel_target.read_bytes()
        channel_string = channel_original.decode("utf-8")
        if "serverCatalogPinSelfCheck" in channel_string:
            if CHANNEL_BRANCH not in channel_string:
                raise ValueError("unexpected existing channel diagnostics")
            channel_changed = False
            channel_new = channel_original
        else:
            if channel_string.count(CHANNEL_ANCHOR) != 1:
                raise ValueError("missing unique channel insertion anchor")
            channel_changed = True
            channel_new = channel_string.replace(
                CHANNEL_ANCHOR, CHANNEL_BRANCH + CHANNEL_ANCHOR, 1
            ).encode("utf-8")

        changed = sum((pin_changed, screen_changed, channel_changed, check_changed))
        if not changed:
            print("M6_SERVER_PIN_DIAGNOSTIC=ALREADY_PRESENT_IDENTICAL")
            return 0
        if args.check:
            print("M6_SERVER_PIN_DIAGNOSTIC=READY")
            print("M6_SERVER_PIN_CHANGED_FILES=" + str(changed))
            print("M6_SERVER_PIN_SOURCE_UNCHANGED=YES")
            return 0

        # Preflight all files before changing any. Pure Kotlin diagnostics file
        # is created first; existing files are then replaced atomically.
        snapshots = [
            (app / PIN, new_pin, pin_original, pin_changed),
            (app / SCREEN, new_screen, screen_original, screen_changed),
            (channel_target, channel_new, channel_original, channel_changed),
        ]
        for path, _, old, should_change in snapshots:
            if should_change and path.read_bytes() != old:
                raise ValueError("source changed between preflight and apply")
        created = False
        try:
            if check_changed:
                with check_target.open("xb") as output:
                    output.write(new_check)
                    output.flush()
                    os.fsync(output.fileno())
                created = True
            for path, new, _, should_change in snapshots:
                if should_change:
                    replace_atomic(path, new)
        except BaseException:
            # Best effort: changes are limited to these known files.
            for path, new, old, should_change in reversed(snapshots):
                if should_change and old is not None and path.read_bytes() == new:
                    replace_atomic(path, old)
            if created and check_target.read_bytes() == new_check:
                check_target.unlink()
            raise
        print("M6_SERVER_PIN_DIAGNOSTIC=APPLIED")
        print("M6_SERVER_PIN_CHANGED_FILES=" + str(changed))
        print("M6_ANDROID_BUILD=NOT_RUN")
        return 0
    except (OSError, ValueError, UnicodeError) as e:
        print("M6_SERVER_PIN_DIAGNOSTIC=BLOCKED: " + str(e), file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
