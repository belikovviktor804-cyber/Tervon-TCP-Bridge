#!/usr/bin/env python3
"""M6: add safe reason code to existing BridgeControlChannel without changing VPN actions.

Pin the actual operator-installed M6 channel blob and replace only its
existing UI response helper. No secret data or exception text is exported.
"""
import argparse
import hashlib
import os
from pathlib import Path
import tempfile

REL = "android/app/src/main/kotlin/com/tervonvpndpi/app/BridgeControlChannel.kt"
BEFORE = "da2a9614e0c6afe9ba8abb35106b50ca921b7678"
OLD = '''    private fun postSuccess(result: MethodChannel.Result, state: String, enabled: Boolean, active: Boolean) {
        android.os.Handler(android.os.Looper.getMainLooper()).post {
            result.success(mapOf("state" to state, "enabled" to enabled, "routeActive" to active))
        }
    }'''
NEW = '''    private fun postSuccess(result: MethodChannel.Result, state: String, enabled: Boolean, active: Boolean) {
        val reason = when (state) {
            "DIRECT_FALLBACK" -> BridgeVpnSession.lastFailureCode().let {
                if (it == "READY" || it == "NOT_ATTEMPTED") "BRIDGE_NOT_ACTIVE" else it
            }
            "RESTART_FAILED" -> "VPN_RESTART_FAILED"
            "ENROLLMENT_FAILED" -> "ENROLLMENT_FAILED"
            else -> null
        }
        android.os.Handler(android.os.Looper.getMainLooper()).post {
            val payload = mutableMapOf<String, Any>(
                "state" to state, "enabled" to enabled, "routeActive" to active
            )
            if (reason != null) payload["reasonCode"] = reason
            result.success(payload)
        }
    }'''


def blob(data):
    return hashlib.sha1(b"blob " + str(len(data)).encode() + b"\0" + data).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--app-root", type=Path, required=True)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    app = args.app_root.resolve()
    target = app / REL
    if not (app / "pubspec.yaml").is_file() or not target.is_file() or target.is_symlink():
        print("M6_REASON_CHANNEL=BLOCKED_TARGET")
        return 1
    current = target.read_bytes()
    # Do not accept a changed channel with unrelated edits.
    old = OLD.encode("utf-8")
    new = NEW.encode("utf-8")
    if current.count(old) == 1 and blob(current) == BEFORE:
        output = current.replace(old, new)
        print("M6_REASON_CHANNEL=READY")
    elif current.count(new) == 1 and current.count(old) == 0:
        print("M6_REASON_CHANNEL=ALREADY_APPLIED")
        return 0
    else:
        print("M6_REASON_CHANNEL=BLOCKED_UNKNOWN_SHA_OR_CONTENT")
        return 1
    if args.check:
        print("M6_CHANNEL_UNCHANGED=YES")
        return 0
    fd, temporary = tempfile.mkstemp(prefix=".m6-reason-channel-", dir=target.parent)
    try:
        with os.fdopen(fd, "wb") as out:
            out.write(output)
            out.flush()
            os.fsync(out.fileno())
        os.chmod(temporary, target.stat().st_mode & 0o777)
        os.replace(temporary, target)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)
    print("M6_REASON_CHANNEL=SOURCE_APPLIED")
    print("LIVE_SERVICES=UNCHANGED")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
