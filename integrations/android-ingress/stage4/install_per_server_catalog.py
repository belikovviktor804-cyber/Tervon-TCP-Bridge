#!/usr/bin/env python3
"""M6 per-server catalog migration, with strict on-disk source guards.

Staging only. No network, APK builds, key generation, VPN or service changes.
Existing BridgeSignedServerCatalog.kt may only be updated from the exact older
GitHub source at OLD_COMMIT. Other existing files are never overwritten.
"""
from __future__ import annotations

import argparse
import os
from pathlib import Path
import subprocess
import sys
import tempfile

OLD_COMMIT = "1db955240076b34ff43efafe9cbfeeeceaad15be"
REL = Path("android/app/src/main/kotlin/com/tervonvpndpi/app")
NAME = "BridgeSignedServerCatalog.kt"
ADDITIONAL = "BridgeServerCatalogTrust.kt"


def original_from_git(repo: Path) -> bytes:
    # Fixed, reviewed commit object, not branch/network state or user argument.
    result = subprocess.run(
        ["git", "-C", str(repo), "show",
         OLD_COMMIT + ":integrations/android-ingress/stage4/" + NAME],
        capture_output=True, check=False,
    )
    if result.returncode or not result.stdout:
        raise ValueError("cannot verify old Android source from Git history")
    return result.stdout


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--app-root", required=True, type=Path)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--apply", action="store_true")
    args = parser.parse_args()

    source_dir = Path(__file__).resolve().parent
    repo = source_dir.parents[2]
    target_dir = args.app_root / REL
    try:
        if not (args.app_root / "android/gradlew").is_file() or \
                target_dir.is_symlink() or not target_dir.is_dir():
            raise ValueError("invalid application source directory")
        for name in (NAME, ADDITIONAL):
            if (target_dir / name).is_symlink():
                raise ValueError("refuse symlink target")
        original = original_from_git(repo)
        current = (source_dir / NAME).read_bytes()
        pin_source = (source_dir / ADDITIONAL).read_bytes()
        if not all(x.startswith(b"package com.tervonvpndpi.app\n")
                   for x in (original, current, pin_source)):
            raise ValueError("unrecognized Kotlin package")

        to_write: list[tuple[Path, bytes, bytes | None]] = []
        main_target = target_dir / NAME
        if main_target.exists():
            if not main_target.is_file():
                raise ValueError("invalid catalog verifier target")
            installed = main_target.read_bytes()
            if installed == current:
                pass
            elif installed == original:
                to_write.append((main_target, current, installed))
            else:
                raise ValueError("refuse unexpected modifications to existing catalog verifier")
        else:
            to_write.append((main_target, current, None))

        trust_target = target_dir / ADDITIONAL
        if trust_target.exists():
            if not trust_target.is_file() or trust_target.read_bytes() != pin_source:
                raise ValueError("refuse different per-server trust source")
        else:
            to_write.append((trust_target, pin_source, None))
        # Create the independent pin source before updating the verifier so a
        # partial file-copy failure never leaves a verifier without its trust API.
        to_write.sort(key=lambda item: item[0].name == NAME)
        if not to_write:
            print("M6_SERVER_CATALOG_STAGE=ALREADY_PRESENT_IDENTICAL")
            return 0
        if args.check:
            print("M6_SERVER_CATALOG_STAGE=READY")
            print("M6_SERVER_CATALOG_CHANGED_FILES=" + str(len(to_write)))
            print("M6_SERVER_CATALOG_FILES_UNCHANGED=YES")
            return 0

        # All source states checked before any mutation.
        for target, new_data, old_data in to_write:
            if old_data is None:
                with target.open("xb") as f:
                    f.write(new_data)
                    f.flush()
                    os.fsync(f.fileno())
            else:
                if target.read_bytes() != old_data or target.is_symlink():
                    raise ValueError("Android catalog file changed since preflight")
                fd, temp = tempfile.mkstemp(prefix=".m6-catalog-", dir=target.parent)
                try:
                    with os.fdopen(fd, "wb") as f:
                        f.write(new_data)
                        f.flush()
                        os.fsync(f.fileno())
                    os.chmod(temp, target.stat().st_mode & 0o777)
                    if target.read_bytes() != old_data:
                        raise ValueError("concurrent catalog modification")
                    os.replace(temp, target)
                finally:
                    if os.path.exists(temp):
                        os.unlink(temp)
        print("M6_SERVER_CATALOG_STAGE=APPLIED")
        print("M6_SERVER_CATALOG_CHANGED_FILES=" + str(len(to_write)))
        print("M6_SERVER_CATALOG_BUILD=NOT_RUN")
        return 0
    except (OSError, ValueError) as error:
        print("M6_SERVER_CATALOG_STAGE=BLOCKED: " + str(error), file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
