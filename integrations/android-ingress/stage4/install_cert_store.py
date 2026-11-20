#!/usr/bin/env python3
"""Install only the reviewed M6 per-server client certificate store.

Guarded, idempotent Android source stage. Does not build, start or stop anything.
Existing non-identical files are never overwritten.
"""
from __future__ import annotations

import argparse
from pathlib import Path
import sys

SOURCE = Path(__file__).resolve().parent / "BridgeServerCertificateStore.kt"
RELATIVE = Path(
    "android/app/src/main/kotlin/com/tervonvpndpi/app/BridgeServerCertificateStore.kt"
)


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--app-root", required=True, type=Path)
    modes = p.add_mutually_exclusive_group(required=True)
    modes.add_argument("--check", action="store_true")
    modes.add_argument("--apply", action="store_true")
    opts = p.parse_args()

    app = opts.app_root.resolve()
    destination = app / RELATIVE
    if not SOURCE.is_file() or not (app / "android/gradlew").is_file():
        print("M6_CERT_STORE_STAGE=INVALID_SOURCE_OR_APP")
        return 2
    if destination.is_symlink() or destination.parent.is_symlink():
        print("M6_CERT_STORE_STAGE=REFUSE_SYMLINK")
        return 2
    data = SOURCE.read_bytes()
    if (not data.startswith(b"package com.tervonvpndpi.app\n")
            or b"class BridgeServerCertificateStore" not in data):
        print("M6_CERT_STORE_STAGE=INVALID_PACKAGE")
        return 2

    if destination.exists():
        if destination.read_bytes() != data:
            print("M6_CERT_STORE_STAGE=REFUSE_DIFFERENT_EXISTING_FILE")
            return 2
        print("M6_CERT_STORE_STAGE=ALREADY_PRESENT_IDENTICAL")
        return 0

    if opts.check:
        print("M6_CERT_STORE_STAGE=READY")
        print("M6_CERT_STORE_CHANGED_FILES=1")
        return 0

    destination.parent.mkdir(parents=True, exist_ok=True)
    try:
        with destination.open("xb") as output:
            output.write(data)
    except FileExistsError:
        print("M6_CERT_STORE_STAGE=REFUSE_CONCURRENT_EXISTING_FILE")
        return 2

    print("M6_CERT_STORE_STAGE=APPLIED")
    print("M6_CERT_STORE_CHANGED_FILES=1")
    print("M6_CERT_STORE_BUILD=NOT_RUN")
    return 0


if __name__ == "__main__":
    sys.exit(main())
