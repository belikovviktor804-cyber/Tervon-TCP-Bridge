#!/usr/bin/env python3
"""Guarded, minimal M6 Android UI/channel integration for manual server pinning.

Requires existing per-server trust Kotlin source. Never touches VPN routing,
Xray, HEV, subscriptions, certificate issuance, signing, or running services.
--check only validates source anchors; --apply modifies two existing files and
creates one new Dart screen, without clobbering unrelated modifications.
"""
from __future__ import annotations
import argparse
import os
from pathlib import Path
import sys
import tempfile

ROOT = Path(__file__).resolve().parent
KOTLIN = Path("android/app/src/main/kotlin/com/tervonvpndpi/app")
CHANNEL = KOTLIN / "BridgeControlChannel.kt"
PIN = KOTLIN / "BridgeServerCatalogTrust.kt"
SCREEN = Path("lib/features/home/bridge_settings_screen.dart")
NEW_SCREEN = Path("lib/features/home/bridge_server_trust_screen.dart")
SOURCE_SCREEN = ROOT / NEW_SCREEN.name

CHANNEL_ANCHOR = "                else -> result.notImplemented()"
CHANNEL_INSERT = '''                "confirmServerCatalogKey" -> {
                    val confirmed = call.argument<Boolean>("explicitIndependentConfirmation") == true
                    if (!confirmed) {
                        result.error("INDEPENDENT_CONFIRMATION_REQUIRED", null, null)
                    } else {
                        try {
                            val serverId = requireNotNull(call.argument<String>("serverId"))
                            val keyId = requireNotNull(call.argument<String>("keyId"))
                            val publicKey = requireNotNull(call.argument<String>("publicKeyBase64"))
                            val digest = requireNotNull(call.argument<String>("verifiedFingerprintSha256"))
                            BridgeServerCatalogTrust(app).confirm(serverId, keyId, publicKey, digest)
                            result.success(mapOf("result" to "PINNED"))
                        } catch (_: Exception) {
                            // No key details, URLs or external fingerprints in app errors.
                            result.error("SERVER_CATALOG_KEY_REJECTED", null, null)
                        }
                    }
                }
'''
IMPORT_ANCHOR = "import 'bridge_runtime_screen.dart';"
IMPORT_INSERT = "import 'bridge_server_trust_screen.dart';\n"
DART_ANCHOR = """          label: const Text('Управление процессом Bridge'),
        ),"""
DART_INSERT = """
        OutlinedButton.icon(
          icon: const Icon(Icons.verified_user_outlined),
          onPressed: () => Navigator.push(context, MaterialPageRoute(
            builder: (_) => const BridgeServerTrustScreen(),
          )),
          label: const Text('Подтвердить ключ Bridge-сервера'),
        ),"""

def inject_once(data: bytes, anchor: str, addition: str, label: str) -> bytes:
    text = data.decode("utf-8")
    if text.count(anchor) != 1:
        raise ValueError(label + ": unexpected source anchor")
    if addition in text:
        raise ValueError(label + ": change already integrated")
    return text.replace(anchor, anchor + addition, 1).encode("utf-8")

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
            raise ValueError("not a recognized Android app")
        pin = app / PIN
        if pin.is_symlink() or not pin.is_file() or \
                b"class BridgeServerCatalogTrust" not in pin.read_bytes():
            raise ValueError("per-server catalog trust source not staged")
        locations = (app / CHANNEL, app / SCREEN, app / NEW_SCREEN)
        if any(path.is_symlink() or path.parent.is_symlink() for path in locations):
            raise ValueError("symlink target or parent refused")
        if any(not path.is_file() for path in locations[:2]):
            raise ValueError("current channel or Flutter screen not available")
        if not SOURCE_SCREEN.is_file() or locations[2].exists():
            raise ValueError("new trust UI screen missing or already exists")
        original_channel = locations[0].read_bytes()
        original_flutter = locations[1].read_bytes()
        if b"confirmServerCatalogKey" in original_channel or \
                b"BridgeServerTrustScreen" in original_flutter:
            raise ValueError("manual trust UI already modified")
        # Precheck every patch before any application.
        updated_channel = inject_once(
            original_channel, CHANNEL_ANCHOR, CHANNEL_INSERT, "Android channel"
        )
        updated_flutter = inject_once(
            original_flutter, IMPORT_ANCHOR, IMPORT_INSERT, "Flutter import"
        )
        updated_flutter = inject_once(
            updated_flutter, DART_ANCHOR, DART_INSERT, "Flutter screen link"
        )
        new_screen_bytes = SOURCE_SCREEN.read_bytes()
        if b"class BridgeServerTrustScreen" not in new_screen_bytes:
            raise ValueError("invalid new trust UI source")
        print("M6_TRUST_UI=READY")
        print("M6_TRUST_UI_CHANGED_FILES=3")
        if args.check:
            print("M6_TRUST_UI_FILES_UNCHANGED=YES")
            return 0
        # Prevent overwriting a source changed after the initial read.
        if locations[0].read_bytes() != original_channel or \
                locations[1].read_bytes() != original_flutter:
            raise ValueError("concurrent Android source modification")
        written = []
        try:
            for path, contents in (
                (locations[0], updated_channel),
                (locations[1], updated_flutter),
            ):
                fd, temp = tempfile.mkstemp(prefix=".m6-trust-", dir=path.parent)
                try:
                    with os.fdopen(fd, "wb") as handle:
                        handle.write(contents)
                        handle.flush()
                        os.fsync(handle.fileno())
                    os.chmod(temp, path.stat().st_mode & 0o777)
                    os.replace(temp, path)
                finally:
                    if os.path.exists(temp):
                        os.unlink(temp)
                written.append(path)
            with locations[2].open("xb") as handle:
                handle.write(new_screen_bytes)
                handle.flush()
                os.fsync(handle.fileno())
        except Exception:
            # Restore only files changed by this invocation, without creating
            # persistent backups or disturbing files outside this operation.
            for path in written:
                if path == locations[0]:
                    path.write_bytes(original_channel)
                elif path == locations[1]:
                    path.write_bytes(original_flutter)
            raise
        print("M6_TRUST_UI=APPLIED")
        print("M6_TRUST_UI_BUILD=NOT_RUN")
        return 0
    except (OSError, ValueError, UnicodeError) as error:
        print("M6_TRUST_UI=BLOCKED: " + str(error), file=sys.stderr)
        return 2

if __name__ == "__main__":
    sys.exit(main())
