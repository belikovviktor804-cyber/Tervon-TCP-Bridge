#!/usr/bin/env python3
"""One-file M6 repair: duplicate Bouncy Castle META-INF/LICENSE.md.

Run --check before --apply. No version/signing/plugin/dependency/VPN changes.
Refuse if Gradle deviates from the operator-reviewed Stage 4 integrated source.
"""
import argparse
import hashlib
import os
from pathlib import Path
import tempfile

GRADLE_PATH = "android/app/build.gradle.kts"
EXPECTED_SHA256 = "3b0dfefc5afef04f00bf65b8b5f92be80ff5a5b50d8c51625c69380c123d85e4"
ANCHOR = "    packaging {\n        jniLibs {"
REPLACEMENT = (
    "    packaging {\n"
    "        resources {\n"
    "            pickFirsts += \"META-INF/LICENSE.md\"\n"
    "        }\n"
    "        jniLibs {"
)

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--app-root", type=Path, required=True)
    action = parser.add_mutually_exclusive_group(required=True)
    action.add_argument("--check", action="store_true")
    action.add_argument("--apply", action="store_true")
    args = parser.parse_args()

    source = args.app_root / GRADLE_PATH
    if source.is_symlink() or not source.is_file():
        parser.exit(2, "BLOCKED: missing or symlinked Gradle file\n")
    original = source.read_bytes()
    digest = hashlib.sha256(original).hexdigest()
    if digest != EXPECTED_SHA256:
        parser.exit(2, "BLOCKED: Gradle file differs from the reviewed Stage 4 version; no changes made\n")
    try:
        text = original.decode("utf-8")
    except UnicodeDecodeError:
        parser.exit(2, "BLOCKED: Gradle file is not UTF-8\n")
    if text.count(ANCHOR) != 1 or 'bcpkix-jdk18on:1.87' not in text:
        parser.exit(2, "BLOCKED: expected Stage 4 dependency/packaging block missing\n")
    patched = text.replace(ANCHOR, REPLACEMENT, 1).encode("utf-8")

    print("STAGE4_BC_LICENSE_SOURCE=MATCH")
    print("STAGE4_BC_LICENSE_CHANGE=ONE_PACKAGING_RULE")
    if args.check:
        print("STAGE4_BC_LICENSE_PATCH=READY")
        return

    fd, tmp = tempfile.mkstemp(prefix=".stage4-bc-", dir=str(source.parent))
    try:
        with os.fdopen(fd, "wb") as stream:
            stream.write(patched)
            stream.flush()
            os.fsync(stream.fileno())
        os.chmod(tmp, source.stat().st_mode & 0o777)
        # A second read prevents replacing a file concurrently modified during prep.
        if source.read_bytes() != original:
            parser.exit(2, "BLOCKED: Gradle file changed concurrently; no replacement\n")
        os.replace(tmp, source)
        print("STAGE4_BC_LICENSE_PATCH=APPLIED")
        print("STAGE4_BUILD=NOT_RUN")
    finally:
        if os.path.exists(tmp):
            os.unlink(tmp)

if __name__ == "__main__":
    main()
