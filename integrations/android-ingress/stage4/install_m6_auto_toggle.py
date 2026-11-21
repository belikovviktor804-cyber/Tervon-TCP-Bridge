#!/usr/bin/env python3
"""Guarded source-only M6 automatic-toggle patch for the operator's verified archive.

No Android build, signing, installed APK, Docker, Xray or running services touched.
"""
import argparse
import hashlib
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

PATCH_SHA256 = 'cd898b4ebdb4b1c54dfb4b87b8de77268750bf43b9db6744643fbeb044590ef8'
FILES = {
    'android/app/src/main/kotlin/com/tervonvpndpi/app/BridgeRuntimeSettings.kt': (
        'f3621792048f1f8bb087513e7927b8328796efba0e9f83424ad65aa7477e314b',
        'c78048cd9228b5e89ceb3dc36d26448db7e28e389374e3e5d4d8382d1c906d33',
    ),
    'android/app/src/main/kotlin/com/tervonvpndpi/app/BridgeControlChannel.kt': (
        'fde2bd903970bc6abc772605e7044f2ec6107f87acd8e4724953a02d1792ccb4',
        'ee428928f6e5c1f0ae201d28ab3c33951edc2af33cdf990d65dbd24eb3dedf5c',
    ),
    'lib/features/home/bridge_settings_screen.dart': (
        '8122a30980b3421d747d8563106c58a9d44f6cd796c74ccba35517273953c76a',
        'b7675c871f9be19f12d9acdaad7796dbf54a26805b641cb83e565c554782e6f4',
    ),
}


def sha(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--app-root', type=Path, required=True)
    modes = parser.add_mutually_exclusive_group(required=True)
    modes.add_argument('--check', action='store_true')
    modes.add_argument('--apply', action='store_true')
    args = parser.parse_args()
    root = args.app_root
    patch = Path(__file__).resolve().with_name('m6-auto-toggle-source.patch')
    if not root.is_dir() or root.is_symlink() or not patch.is_file() or patch.is_symlink():
        print('M6_TOGGLE=BLOCKED: PATH_UNAVAILABLE')
        return 1
    if sha(patch.read_bytes()) != PATCH_SHA256:
        print('M6_TOGGLE=BLOCKED: PATCH_HASH_MISMATCH')
        return 1
    state = set()
    for rel, (before, after) in FILES.items():
        f = root / rel
        if not f.is_file() or f.is_symlink() or f.parent.is_symlink():
            print('M6_TOGGLE=BLOCKED: SOURCE_FILE_UNAVAILABLE')
            return 1
        current = sha(f.read_bytes())
        if current not in (before, after):
            print('M6_TOGGLE=BLOCKED: UNRECOGNIZED_SOURCE', rel)
            return 1
        state.add('OLD' if current == before else 'NEW')
    if state == {'NEW'}:
        print('M6_TOGGLE=ALREADY_APPLIED')
        return 0
    if state != {'OLD'}:
        print('M6_TOGGLE=BLOCKED: PARTIAL_OR_MIXED_SOURCES')
        return 1
    print('M6_TOGGLE=READY')
    if args.check:
        print('M6_SOURCE_UNCHANGED=YES')
        return 0
    with tempfile.TemporaryDirectory(prefix='.m6-toggle-stage-', dir=root) as staging:
        stage = Path(staging)
        for rel in FILES:
            destination = stage / rel
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(root / rel, destination)
        command = ['patch', '--batch', '--forward', '--fuzz=0', '-p1', '-i', str(patch)]
        try:
            finished = subprocess.run(command, cwd=stage, capture_output=True, text=True, timeout=20)
        except (OSError, subprocess.TimeoutExpired):
            print('M6_TOGGLE=BLOCKED: PATCH_TOOL_UNAVAILABLE')
            return 1
        if finished.returncode:
            print('M6_TOGGLE=BLOCKED: PATCH_DID_NOT_APPLY')
            return 1
        for rel, (_, expected) in FILES.items():
            if sha((stage / rel).read_bytes()) != expected:
                print('M6_TOGGLE=BLOCKED: PATCH_OUTPUT_HASH_MISMATCH', rel)
                return 1
        for rel in FILES:
            target = root / rel
            data = (stage / rel).read_bytes()
            fd, temporary = tempfile.mkstemp(prefix='.m6-toggle-', dir=target.parent)
            try:
                with os.fdopen(fd, 'wb') as stream:
                    stream.write(data)
                    stream.flush()
                    os.fsync(stream.fileno())
                os.chmod(temporary, target.stat().st_mode & 0o777)
                os.replace(temporary, target)
            finally:
                if os.path.exists(temporary):
                    os.unlink(temporary)
    print('M6_TOGGLE=SOURCE_APPLIED')
    print('M6_APK_BUILD=NOT_RUN')
    print('M6_LIVE_SERVICES=UNCHANGED')
    return 0


if __name__ == '__main__':
    sys.exit(main())
