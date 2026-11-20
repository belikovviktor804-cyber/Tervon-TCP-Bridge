#!/usr/bin/env python3
"""Apply Stage 4 identity/CSR self-check to the reviewed TervonVPN DPI source.

Fail closed unless all target files byte-match the operator-provided 2026-11-20
source snapshot. Does not build/sign APK, bump version, or change VPN routing.
Run --check first, then --apply; never run on production service directories.
"""
from __future__ import annotations
import argparse
import hashlib
import os
from pathlib import Path
import tempfile

HERE = Path(__file__).resolve().parent
KOTLIN = Path("android/app/src/main/kotlin/com/tervonvpndpi/app")
GRADLE = Path("android/app/build.gradle.kts")
CHANNEL = KOTLIN / "BridgeControlChannel.kt"
DART = Path("lib/features/home/bridge_settings_screen.dart")
EXPECTED = {
    GRADLE: "228221df007c359636aa68a8f8c34869018e8b7fced5edfc0fc126282ece421b",
    CHANNEL: "c4f0ae10e857785e935cbac232c27e41a8ebf3747861cfe52c41c8e2a399133c",
    DART: "42b92244d6403067f9f04eb75b0af9447b66b6818aabfd690ce3252ccf9c0a67",
}
NEW_KOTLIN = (
    "BridgeDeviceIdentity.kt",
    "BridgeDeviceCsr.kt",
    "BridgeDeviceSelfCheck.kt",
)
DEPENDENCY = '    implementation("org.bouncycastle:bcpkix-jdk18on:1.87")'
GRADLE_ANCHOR = '    implementation("com.squareup.okhttp3:okhttp:4.12.0")'
PACKAGING_ANCHOR = '    packaging {\n        jniLibs {'
PACKAGING_REPLACEMENT = '    packaging {\n        resources {\n            pickFirsts += "META-INF/LICENSE.md"\n        }\n        jniLibs {'

CHANNEL_ANCHOR = '                else -> result.notImplemented()'
CHANNEL_INJECTION = '''                "stage4IdentitySelfCheck" -> Thread({
                    try {
                        val ok = BridgeDeviceSelfCheck.verify(app)
                        android.os.Handler(android.os.Looper.getMainLooper()).post {
                            result.success(mapOf("result" to if (ok) "PASS" else "FAIL"))
                        }
                    } catch (e: Exception) {
                        // Do not disclose key material, device identities or exception details.
                        android.os.Handler(android.os.Looper.getMainLooper()).post {
                            result.error("STAGE4_SELF_CHECK_FAILED", e.javaClass.simpleName, null)
                        }
                    }
                }, "bridge-stage4-selfcheck").start()
'''

DART_METHOD_ANCHOR = '  Future<void> _inspect() async {'
DART_METHOD = '''  Future<void> _checkStage4Identity() async {
    if (_busy) return;
    setState(() { _busy = true; _message = 'Проверка ключа устройства и CSR…'; });
    try {
      final data = await _channel.invokeMapMethod<String, dynamic>('stage4IdentitySelfCheck');
      if (mounted) setState(() {
        _message = data?['result'] == 'PASS'
            ? 'Stage 4: PASS — ключ устройства, подпись и CSR'
            : 'Stage 4: FAIL — самопроверка не пройдена';
      });
    } on PlatformException catch (e) {
      if (mounted) setState(() => _message = 'Stage 4: ${e.code}');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

'''
DART_BUTTON_ANCHOR = '''          OutlinedButton(onPressed: _busy ? null : _inspect,
              child: const Text('Показать параметры запуска ядра')),'''
DART_BUTTON = '''          OutlinedButton(
            onPressed: _busy ? null : _checkStage4Identity,
            child: const Text('Проверить ключ и CSR (Stage 4)'),
          ),'''


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def replace_exact(value: str, old: str, new: str, label: str) -> str:
    if value.count(old) != 1:
        raise RuntimeError(f"{label}: expected exactly one anchor, got {value.count(old)}")
    return value.replace(old, new, 1)


def prepare(root: Path) -> dict[Path, bytes]:
    if root.is_symlink() or not root.is_dir():
        raise RuntimeError("app root must be an existing real directory")
    out: dict[Path, bytes] = {}
    source = {}
    for rel, digest in EXPECTED.items():
        target = root / rel
        if target.is_symlink() or not target.is_file():
            raise RuntimeError(f"missing or symlink source: {rel}")
        data = target.read_bytes()
        if sha256(data) != digest:
            raise RuntimeError(f"source differs from operator-reviewed snapshot: {rel}")
        source[rel] = data.decode("utf-8")

    gradle = replace_exact(source[GRADLE], GRADLE_ANCHOR,
                           GRADLE_ANCHOR + "\n" + DEPENDENCY, "Gradle")
    gradle = replace_exact(gradle, PACKAGING_ANCHOR, PACKAGING_REPLACEMENT, "Bouncy Castle license resource")
    channel = replace_exact(source[CHANNEL], CHANNEL_ANCHOR,
                            CHANNEL_INJECTION + CHANNEL_ANCHOR, "BridgeControlChannel")
    dart = replace_exact(source[DART], DART_METHOD_ANCHOR,
                         DART_METHOD + DART_METHOD_ANCHOR, "Bridge Settings method")
    dart = replace_exact(dart, DART_BUTTON_ANCHOR,
                         DART_BUTTON_ANCHOR + "\n" + DART_BUTTON, "Bridge Settings button")
    out[GRADLE] = gradle.encode("utf-8")
    out[CHANNEL] = channel.encode("utf-8")
    out[DART] = dart.encode("utf-8")

    for name in NEW_KOTLIN:
        rel = KOTLIN / name
        target = root / rel
        if target.exists() or target.is_symlink():
            raise RuntimeError(f"refuse to overwrite Stage 4 source: {rel}")
        data = (HERE / name).read_bytes()
        if not data or b"package com.tervonvpndpi.app" not in data:
            raise RuntimeError(f"invalid Stage 4 source: {name}")
        out[rel] = data
    return out


def apply(root: Path, changes: dict[Path, bytes]) -> None:
    # Preflight all paths before performing the first write.
    for rel in changes:
        parent = (root / rel).parent
        if not parent.is_dir() or parent.is_symlink():
            raise RuntimeError(f"invalid target parent: {rel}")
    for rel, data in changes.items():
        target = root / rel
        fd, temporary = tempfile.mkstemp(prefix=".stage4-", dir=str(target.parent))
        try:
            with os.fdopen(fd, "wb") as stream:
                stream.write(data)
                stream.flush()
                os.fsync(stream.fileno())
            os.chmod(temporary, 0o644)
            os.replace(temporary, target)
        finally:
            if os.path.exists(temporary):
                os.unlink(temporary)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--app-root", type=Path, required=True)
    action = parser.add_mutually_exclusive_group(required=True)
    action.add_argument("--check", action="store_true")
    action.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    try:
        root = args.app_root.resolve(strict=True)
        changes = prepare(root)
        print("STAGE4_SOURCE_SNAPSHOT=MATCH")
        print("STAGE4_FILES=" + str(len(changes)))
        if args.apply:
            apply(root, changes)
            print("STAGE4_SOURCE_INTEGRATION=APPLIED")
            print("STAGE4_APK_BUILD=NOT_RUN")
            print("STAGE4_ANDROID_TEST=NOT_RUN")
        else:
            print("STAGE4_SOURCE_INTEGRATION=READY_TO_APPLY")
            print("STAGE4_APP_UNCHANGED=YES")
    except (OSError, RuntimeError, UnicodeError) as exc:
        parser.exit(1, f"STAGE4_INTEGRATION=BLOCKED: {exc}\n")


if __name__ == "__main__":
    main()
