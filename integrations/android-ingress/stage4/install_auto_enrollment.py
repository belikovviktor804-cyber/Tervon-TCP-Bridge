#!/usr/bin/env python3
"""Install M6 enrollment SOURCES into the supplied Android snapshot only.

No APK build, no servers, no changes to Xray/VPN routes.
Refuses unrecognized code to protect the operator's existing app.
"""
import argparse
import hashlib
import os
from pathlib import Path
import tempfile

NATIVE = Path("android/app/src/main/kotlin/com/tervonvpndpi/app")
SCREEN = Path("lib/features/home/bridge_settings_screen.dart")
SOURCES = (
    "BridgeAutoCertificateEnrollment.kt",
    "BridgeAutoCertificateAndroidAdapter.kt",
    "BridgeAutoEnrollmentHttp.kt",
    "BridgeAutoEnrollmentChannel.kt",
)
EXPECTED = {
    NATIVE / "BridgeControlChannel.kt":
        "fde2bd903970bc6abc772605e7044f2ec6107f87acd8e4724953a02d1792ccb4",
    SCREEN: "8122a30980b3421d747d8563106c58a9d44f6cd796c74ccba35517273953c76a",
}


def edit_channel(text):
    anchor = "        BridgeRuntimeControl.register(messenger, app)\n"
    if text.count(anchor) != 1:
        raise ValueError("Bridge channel entry changed")
    return text.replace(anchor, anchor +
                        "        BridgeAutoEnrollmentChannel.register(messenger, app)\n")


def edit_screen(text):
    old = "import 'bridge_runtime_screen.dart';import 'bridge_server_trust_screen.dart';"
    new = "import 'bridge_runtime_screen.dart';\nimport '../../vpn/xray_service.dart';"
    if text.count(old) != 1:
        raise ValueError("settings imports changed")
    text = text.replace(old, new)
    before = "  Future<void> _save() async {\n"
    if text.count(before) != 1:
        raise ValueError("settings save function changed")
    text = text.replace(before,
        "  Future<void> _save({bool enrollmentTrigger = false}) async {\n")
    anchor = """      if (mounted) setState(() => _message = data == null
        ? 'Нет ответа от Android' : 'Настройки сохранены. Bridge не запущен.');
"""
    replacement = anchor + """      // The certificate request runs ONLY after the user's toggle ON.
      if (enrollmentTrigger && _enabled && data != null && data['enabled'] == true) {
        final source = await XrayService.getActiveSource();
        final subscriptions = await XrayService.getExternalSubscriptions();
        final index = source['externalIndex'] as int? ?? -1;
        final selected = source['source'] == 'external' &&
                index >= 0 && index < subscriptions.length
            ? subscriptions[index] : null;
        final parsed = selected == null ? null : Uri.tryParse(selected);
        if (parsed == null || parsed.scheme != 'https' || parsed.host.isEmpty) {
          if (mounted) setState(() =>
              _message = 'Для текущей подписки не указан HTTPS-сервер Bridge');
        } else {
          final response = await const MethodChannel(
            'com.tervonvpndpi/bridge_auto_enrollment',
          ).invokeMapMethod<String, dynamic>('onBypassToggled', {
            'enabled': true,
            'subscriptionUrl': selected,
          });
          if (mounted) setState(() {
            switch (response?['status']) {
              case 'CERTIFICATE_ENROLLED':
                _message = 'Сертификат Bridge получен';
                break;
              case 'CERTIFICATE_REUSED':
                _message = 'Сохранённый сертификат Bridge действителен';
                break;
              case 'NO_BRIDGE':
                _message = 'Tervon Bridge на сервере не настроен';
                break;
              default:
                _message = 'Регистрация Bridge недоступна';
            }
          });
        }
      }
"""
    if text.count(anchor) != 1:
        raise ValueError("settings result anchor changed")
    text = text.replace(anchor, replacement)
    old_switch = "value: _enabled, onChanged: (v) => setState(() => _enabled = v)),"
    new_switch = """value: _enabled, onChanged: _busy ? null : (v) {
              setState(() => _enabled = v);
              _save(enrollmentTrigger: true);
            }),"""
    if text.count(old_switch) != 1:
        raise ValueError("DPI toggle changed")
    text = text.replace(old_switch, new_switch)
    manual_button = """        OutlinedButton.icon(
          icon: const Icon(Icons.verified_user_outlined),
          onPressed: () => Navigator.push(context, MaterialPageRoute(
            builder: (_) => const BridgeServerTrustScreen(),
          )),
          label: const Text('Подтвердить ключ Bridge-сервера'),
        ),
"""
    if text.count(manual_button) != 1:
        raise ValueError("manual key button changed")
    return text.replace(manual_button, "")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--app-root", type=Path, required=True)
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--check", action="store_true")
    group.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    root = args.app_root
    source = Path(__file__).resolve().parent
    changed = {}
    for relative, expected_sha in EXPECTED.items():
        target = root / relative
        if not target.is_file() or target.is_symlink():
            raise SystemExit("M6_ENROLLMENT=FAIL:REQUIRED_APP_SOURCE_MISSING")
        raw = target.read_bytes()
        if hashlib.sha256(raw).hexdigest() != expected_sha:
            raise SystemExit("M6_ENROLLMENT=FAIL:UNRECOGNIZED_APP_SOURCE")
        updated = (edit_channel if relative == NATIVE / "BridgeControlChannel.kt"
                   else edit_screen)(raw.decode("utf-8"))
        changed[target] = updated.encode("utf-8")
    for name in SOURCES:
        incoming = source / name
        target = root / NATIVE / name
        if not incoming.is_file() or incoming.is_symlink():
            raise SystemExit("M6_ENROLLMENT=FAIL:SOURCE_MISSING")
        if target.exists() or target.is_symlink():
            raise SystemExit("M6_ENROLLMENT=FAIL:REFUSE_OVERWRITE")
        changed[target] = incoming.read_bytes()
    print("M6_ENROLLMENT=READY", len(changed))
    if args.check:
        print("M6_APP_UNCHANGED=YES")
        return
    for target, data in changed.items():
        fd, temp = tempfile.mkstemp(prefix=".m6-enrollment-", dir=str(target.parent))
        try:
            with os.fdopen(fd, "wb") as stream:
                stream.write(data)
                stream.flush()
                os.fsync(stream.fileno())
            os.chmod(temp, 0o644)
            os.replace(temp, target)
        finally:
            if os.path.exists(temp):
                os.unlink(temp)
    print("M6_ENROLLMENT=SOURCE_APPLIED")
    print("M6_APK=NOT_BUILT; M6_SERVER=UNCHANGED")


if __name__ == "__main__":
    main()
