#!/usr/bin/env python3
"""Guarded staging of pure offline M6 signed-catalog modules into Android.

Copies only the catalog reconciler, signed verifier and packaged trust loader.
Never overwrites different files; never touches VPN, APK or live services.
"""
from __future__ import annotations

import argparse
from pathlib import Path
import sys

FILES = ("BridgeServerReconciler.kt", "BridgeSignedServerCatalog.kt", "BridgeCatalogTrustAnchors.kt")
PACKAGE = b"package com.tervonvpndpi.app\n"
RELATIVE = Path("android/app/src/main/kotlin/com/tervonvpndpi/app")


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--app-root", type=Path, required=True)
    mode = p.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--apply", action="store_true")
    args = p.parse_args()

    app = args.app_root.resolve()
    source = Path(__file__).resolve().parent
    target_dir = app / RELATIVE
    if not (app / "android/gradlew").is_file() or not target_dir.is_dir():
        print("M6_CATALOG_STAGE=INVALID_APP_ROOT")
        return 2
    if target_dir.is_symlink():
        print("M6_CATALOG_STAGE=REFUSE_SYMLINK_DIRECTORY")
        return 2

    missing: list[tuple[Path, bytes]] = []
    for name in FILES:
        origin = source / name
        target = target_dir / name
        if not origin.is_file() or origin.is_symlink() or target.is_symlink():
            print("M6_CATALOG_STAGE=INVALID_OR_SYMLINK_SOURCE_OR_TARGET", name)
            return 2
        content = origin.read_bytes()
        if not content.startswith(PACKAGE) or b"internal " not in content:
            print("M6_CATALOG_STAGE=INVALID_KOTLIN_PACKAGE", name)
            return 2
        if target.exists():
            if not target.is_file() or target.read_bytes() != content:
                print("M6_CATALOG_STAGE=REFUSE_DIFFERENT_EXISTING_FILE", name)
                return 2
        else:
            missing.append((target, content))

    if not missing:
        print("M6_CATALOG_STAGE=ALREADY_PRESENT_IDENTICAL")
        return 0
    if args.check:
        print("M6_CATALOG_STAGE=READY")
        print("M6_CATALOG_NEW_FILES=" + str(len(missing)))
        print("M6_CATALOG_FILES_UNCHANGED=YES")
        return 0

    created: list[Path] = []
    try:
        for target, content in missing:
            with target.open("xb") as handle:
                handle.write(content)
            created.append(target)
    except (FileExistsError, OSError):
        for target in created:
            target.unlink()
        print("M6_CATALOG_STAGE=FAILED_AND_NEW_FILES_ROLLED_BACK")
        return 2

    print("M6_CATALOG_STAGE=APPLIED")
    print("M6_CATALOG_NEW_FILES=" + str(len(created)))
    print("M6_CATALOG_ANDROID_BUILD=NOT_RUN")
    return 0


if __name__ == "__main__":
    sys.exit(main())
