#!/usr/bin/env python3
"""Install a separately verified PUBLIC catalog key into an Android APK asset.

This is a guarded, opt-in, PUBLIC-key-only installer. It does not generate
keys, modify Gradle, build APKs, touch VPN services, or use the network.
"""
from __future__ import annotations

import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import re
import sys

NAME = "bridge_catalog_trust_v1.json"
FORMAT = "tervon.bridge.catalog.trust.v1"
KEY_ID = re.compile(r"[A-Za-z0-9_-]{1,64}\Z", re.ASCII)
DIGEST = re.compile(r"[a-fA-F0-9]{64}\Z", re.ASCII)
MAX_ASSET_SIZE = 4096
TARGET_REL = Path("android/app/src/main/assets") / NAME


def _no_duplicate_keys(pairs: list[tuple[str, object]]) -> dict:
    output = {}
    for key, value in pairs:
        if key in output:
            raise ValueError("duplicate JSON field")
        output[key] = value
    return output


def validate(data: bytes, expected_sha256: str) -> tuple[str, ...]:
    if not DIGEST.fullmatch(expected_sha256):
        raise ValueError("expected public-key SHA256 must be 64 hex characters")
    if not (1 <= len(data) <= MAX_ASSET_SIZE):
        raise ValueError("invalid public trust asset length")
    try:
        envelope = json.loads(
            data.decode("utf-8", errors="strict"),
            object_pairs_hook=_no_duplicate_keys,
        )
    except (UnicodeError, json.JSONDecodeError) as exc:
        raise ValueError("invalid public trust JSON") from exc
    if not isinstance(envelope, dict) or set(envelope) != {"format", "keys"}:
        raise ValueError("invalid public trust asset fields")
    if envelope["format"] != FORMAT:
        raise ValueError("unsupported trust asset format")
    keys = envelope["keys"]
    if not isinstance(keys, dict) or not (1 <= len(keys) <= 4):
        raise ValueError("must have 1 to 4 pinned public keys")

    # First publisher bootstrap is deliberately restricted to ONE key:
    # multi-key rotation requires explicit independently verified fingerprints
    # for every new root. Do not silently trust extra keys in the same asset.
    if len(keys) != 1:
        raise ValueError("initial trust installation requires one independently verified key")

    names = []
    for name, encoded in keys.items():
        if not isinstance(name, str) or not KEY_ID.fullmatch(name):
            raise ValueError("invalid signing key ID")
        if not isinstance(encoded, str) or len(encoded) > 64:
            raise ValueError("invalid public key encoding")
        try:
            decoded = base64.b64decode(encoded, validate=True)
        except (ValueError, base64.binascii.Error) as exc:
            raise ValueError("invalid public key Base64") from exc
        if len(decoded) != 32 or base64.b64encode(decoded).decode("ascii") != encoded:
            raise ValueError("invalid Ed25519 raw public key")
        if hashlib.sha256(decoded).hexdigest() != expected_sha256.lower():
            raise ValueError("public key fingerprint mismatch")
        names.append(name)
    return tuple(names)


def safe_asset_directory(app_root: Path, apply: bool) -> Path:
    android = app_root / "android"
    app = android / "app"
    src = app / "src"
    main = src / "main"
    assets = main / "assets"
    for directory in (app_root, android, app, src, main):
        if directory.is_symlink() or not directory.is_dir():
            raise ValueError("invalid Android source directory or symlink")
    if assets.is_symlink() or (assets.exists() and not assets.is_dir()):
        raise ValueError("unsafe Android assets path")
    if not assets.exists() and apply:
        assets.mkdir(mode=0o755)
    return assets


def run() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--app-root", required=True, type=Path)
    parser.add_argument("--public-asset", required=True, type=Path)
    parser.add_argument("--expected-public-sha256", required=True)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    try:
        app = args.app_root.resolve(strict=True)
        source = args.public_asset
        if source.is_symlink() or not source.is_file() or source.stat().st_size > MAX_ASSET_SIZE:
            raise ValueError("invalid public asset source")
        data = source.read_bytes()
        names = validate(data, args.expected_public_sha256)
        directory = safe_asset_directory(app, apply=False)
        target = directory / NAME
        if target.is_symlink():
            raise ValueError("refuse asset symlink")
        if target.exists():
            if not target.is_file() or target.read_bytes() != data:
                raise ValueError("refuse to overwrite different APK trust asset")
            print("M6_CATALOG_TRUST_ASSET=ALREADY_PRESENT_IDENTICAL")
            return 0
        if args.check:
            print("M6_CATALOG_TRUST_ASSET=READY")
            print("M6_PUBLIC_KEY_IDS=" + ",".join(names))
            print("M6_FILES_UNCHANGED=YES")
            return 0
        directory = safe_asset_directory(app, apply=True)
        target = directory / NAME
        fd = os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o644)
        try:
            with os.fdopen(fd, "wb") as f:
                f.write(data)
                f.flush()
                os.fsync(f.fileno())
        except BaseException:
            target.unlink(missing_ok=True)
            raise
        print("M6_CATALOG_TRUST_ASSET=APPLIED")
        print("M6_PUBLIC_KEY_IDS=" + ",".join(names))
        print("M6_APK_BUILD=NOT_RUN")
        return 0
    except (OSError, ValueError) as exc:
        print("M6_CATALOG_TRUST_ASSET=BLOCKED: " + str(exc), file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(run())
