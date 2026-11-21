#!/usr/bin/env python3
"""Remove only the deprecated user-facing manual server-key confirmation entry.

Approved M6 UX is automatic enrollment and a single manual 'Обход DPI' switch.
Retain unused Kotlin/Ed25519 sources pending dependency review: this does not
activate Bridge or change VPN runtime.
"""
import argparse
import hashlib
import os
from pathlib import Path
import tempfile

REL = Path('lib/features/home/bridge_settings_screen.dart')
SNAPSHOT_SHA = '8122a30980b3421d747d8563106c58a9d44f6cd796c74ccba35517273953c76a'
OLD_IMPORT = "import 'bridge_runtime_screen.dart';import 'bridge_server_trust_screen.dart';"
NEW_IMPORT = "import 'bridge_runtime_screen.dart';"
OLD_BUTTON = """        OutlinedButton.icon(
          icon: const Icon(Icons.verified_user_outlined),
          onPressed: () => Navigator.push(context, MaterialPageRoute(
            builder: (_) => const BridgeServerTrustScreen(),
          )),
          label: const Text('Подтвердить ключ Bridge-сервера'),
        ),
"""

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--app-root', required=True, type=Path)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument('--check', action='store_true')
    mode.add_argument('--apply', action='store_true')
    args = parser.parse_args()
    target = args.app_root / REL
    if not target.is_file() or target.is_symlink():
        raise SystemExit('M6_UI=FAIL: SOURCE_MISSING_OR_SYMLINK')
    raw = target.read_bytes()
    original = raw.decode('utf-8')
    if original.count(OLD_IMPORT) != 1 or original.count(OLD_BUTTON) != 1:
        raise SystemExit('M6_UI=FAIL: ANCHOR_MISMATCH')
    if hashlib.sha256(raw).hexdigest() != SNAPSHOT_SHA:
        raise SystemExit('M6_UI=FAIL: UNRECOGNIZED_SOURCE_SHA')
    revised = original.replace(OLD_IMPORT, NEW_IMPORT).replace(OLD_BUTTON, '')
    assert 'BridgeServerTrustScreen' not in revised
    print('M6_UI=READY')
    if args.check:
        print('M6_UI_SOURCE_UNCHANGED=YES')
        return
    fd, temporary = tempfile.mkstemp(prefix='.m6-ui-', dir=str(target.parent))
    try:
        with os.fdopen(fd, 'wb') as stream:
            stream.write(revised.encode('utf-8'))
            stream.flush()
            os.fsync(stream.fileno())
        os.chmod(temporary, target.stat().st_mode & 0o777)
        os.replace(temporary, target)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)
    print('M6_UI=SOURCE_APPLIED')
    print('M6_APK_BUILD=NOT_RUN')

if __name__ == '__main__':
    main()
