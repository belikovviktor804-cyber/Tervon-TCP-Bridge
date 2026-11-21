#!/usr/bin/env python3
"""Guarded source-only M6 patch against operator-supplied 2026-11-21 snapshot.

Does not change VPN routing, APK, certificates, signing or running services.
"""
import argparse
import hashlib
import os
from pathlib import Path
import sys
import tempfile

FILES = (
    'BridgeServerReconciler.kt',
    'BridgeVlessRoutePlan.kt',
    'BridgeXrayConfigRoute.kt',
    'BridgeClientConfigBuilder.kt',
    'BridgeClientPemHandoff.kt',
    'BridgeAutoCertificateEnrollment.kt',
    'BridgeAutoCertificateAndroidAdapter.kt',
    'BridgeAutoEnrollmentHttp.kt',
)
BASE_REL = Path('android/app/src/main/kotlin/com/tervonvpndpi/app')
RECONCILER_OLD_SHA = '882f81df9ed38a6448b77f58a00b3583614abfdb1507a78d7774e88641650bac'


def digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--app-root', type=Path, required=True)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument('--check', action='store_true')
    mode.add_argument('--apply', action='store_true')
    args = parser.parse_args()
    target_dir = args.app_root / BASE_REL
    source_dir = Path(__file__).resolve().parent
    if not target_dir.is_dir() or target_dir.is_symlink():
        print('M6_RECOVERY=FAIL: ANDROID_PACKAGE_NOT_FOUND')
        return 1
    for required in ('BridgeCoreManager.kt', 'BridgeRuntimeControl.kt', 'BridgeDeviceIdentity.kt', 'BridgeDeviceCsr.kt'):
        if not (target_dir / required).is_file():
            print('M6_RECOVERY=FAIL: REQUIRED_STAGE3_STAGE4_SOURCE_NOT_FOUND')
            return 1
    pending = []
    for name in FILES:
        source = source_dir / name
        target = target_dir / name
        if not source.is_file() or source.is_symlink():
            print('M6_RECOVERY=FAIL: BUNDLE_SOURCE_UNAVAILABLE')
            return 1
        content = source.read_bytes()
        if target.is_symlink():
            print('M6_RECOVERY=FAIL: REFUSE_SYMLINK')
            return 1
        if target.exists():
            if not target.is_file():
                print('M6_RECOVERY=FAIL: NON_REGULAR_TARGET')
                return 1
            previous = target.read_bytes()
            if previous == content:
                continue
            if name != 'BridgeServerReconciler.kt' or digest(previous) != RECONCILER_OLD_SHA:
                print('M6_RECOVERY=FAIL: UNRECOGNIZED_EXISTING_SOURCE')
                return 1
        pending.append((target, content))
    print('M6_RECOVERY=READY')
    print('M6_CHANGED_FILES=', len(pending))
    if args.check:
        print('M6_APP_SOURCE_UNCHANGED=YES')
        return 0
    for target, content in pending:
        fd, temporary = tempfile.mkstemp(prefix='.m6-recovery-', dir=str(target_dir))
        try:
            with os.fdopen(fd, 'wb') as stream:
                stream.write(content)
                stream.flush()
                os.fsync(stream.fileno())
            os.chmod(temporary, 0o644)
            os.replace(temporary, target)
        finally:
            if os.path.exists(temporary):
                os.unlink(temporary)
    print('M6_RECOVERY=SOURCE_APPLIED')
    print('M6_APK_BUILD=NOT_RUN')
    print('M6_ROUTE_WIRING=NOT_IMPLEMENTED')
    return 0


if __name__ == '__main__':
    sys.exit(main())
