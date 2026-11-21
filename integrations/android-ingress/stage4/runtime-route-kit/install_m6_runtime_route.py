#!/usr/bin/env python3
"""SHA-guarded M6 full Android source routing installation; no APK or live services.

Source: reviewed actual operator archive, AFTER installed M6 8-source overlay
and 3-file automatic-toggle. All nine route patch targets are checked before
any modification; independently pin staged HTTPS discovery authority source.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

PATCH_SHA256 = "bc9d10cd71cffccb61e6eca443c1f6a85801e39a7a75c81adfe238b25bf1337d"
HTTP_REL = "android/app/src/main/kotlin/com/tervonvpndpi/app/BridgeAutoEnrollmentHttp.kt"
BUILDER_REL = "android/app/src/main/kotlin/com/tervonvpndpi/app/BridgeClientConfigBuilder.kt"
BUILDER_OLD_BLOB = "133a171d5521413ec66fda43f8abd49e8114363b"
BUILDER_NEW_BLOB = "1cce7812bc65cbecb45a121383adb09ce50dc1c3"
HTTP_OLD_BLOB = "c2a3d6d0d29046ee5033dac23cbba580627448e1"
HTTP_NEW_BLOB = "3ea1259f96b528eee06e30c03430eb4448de87f2"
STAGE = "integrations/android-ingress/stage4"


def blob(data: bytes) -> str:
    return hashlib.sha1(b"blob " + str(len(data)).encode() + b"\0" + data).hexdigest()


def fatal(reason: str):
    print("M6_RUNTIME=BLOCKED: " + reason, file=sys.stderr)
    return 1


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--app-root", type=Path, required=True)
    parser.add_argument("--repo-root", type=Path, required=True)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    app, repo = args.app_root.resolve(), args.repo_root.resolve()
    here = Path(__file__).resolve().parent
    patch = here / "m6-runtime-route.patch"
    manifest = here / "m6_manifest.json"
    if not (app / "pubspec.yaml").is_file() or not patch.is_file() or not manifest.is_file():
        return fatal("PROJECT_OR_KIT_UNAVAILABLE")
    if hashlib.sha256(patch.read_bytes()).hexdigest() != PATCH_SHA256:
        return fatal("PATCH_HASH_MISMATCH")
    try:
        entries = json.loads(manifest.read_text())
        if len(entries) != 9:
            return fatal("UNEXPECTED_PATCH_FILE_COUNT")
        http_src = repo / STAGE / "BridgeAutoEnrollmentHttp.kt"
        http_dest = app / HTTP_REL
        if not http_src.is_file() or http_src.is_symlink() or blob(http_src.read_bytes()) != HTTP_NEW_BLOB:
            return fatal("GITHUB_HTTP_SOURCE_NOT_CURRENT")
        states = []
        for rel, info in entries.items():
            target = app / rel
            if target.is_symlink() or target.parent.is_symlink():
                return fatal("SYMLINK_REJECTED " + rel)
            if target.is_file():
                actual = blob(target.read_bytes())
                if actual not in (info["before"], info["after"]):
                    return fatal("UNKNOWN_APP_SOURCE " + rel)
                states.append("NEW" if actual == info["after"] else "OLD")
            elif not target.exists() and info["before"] is None:
                states.append("OLD")
            else:
                return fatal("MISSING_OR_UNEXPECTED_TARGET " + rel)
        if http_dest.is_symlink() or not http_dest.is_file():
            return fatal("HTTP_DEST_UNAVAILABLE")
        current_http = blob(http_dest.read_bytes())
        if current_http not in (HTTP_OLD_BLOB, HTTP_NEW_BLOB):
            return fatal("UNKNOWN_HTTP_SOURCE")
        states.append("NEW" if current_http == HTTP_NEW_BLOB else "OLD")
        builder_src = repo / STAGE / "BridgeClientConfigBuilder.kt"
        builder_dest = app / BUILDER_REL
        if builder_src.is_symlink() or not builder_src.is_file() or blob(builder_src.read_bytes()) != BUILDER_NEW_BLOB:
            return fatal("GITHUB_BUILDER_SOURCE_NOT_CURRENT")
        if builder_dest.is_symlink() or not builder_dest.is_file():
            return fatal("BUILDER_DEST_UNAVAILABLE")
        current_builder = blob(builder_dest.read_bytes())
        if current_builder not in (BUILDER_OLD_BLOB, BUILDER_NEW_BLOB):
            return fatal("UNKNOWN_BUILDER_SOURCE")
        states.append("NEW" if current_builder == BUILDER_NEW_BLOB else "OLD")
        if all(x == "NEW" for x in states):
            print("M6_RUNTIME=ALREADY_APPLIED")
            return 0
        if not all(x == "OLD" for x in states):
            return fatal("PARTIAL_OR_MIXED_SOURCES")
        print("M6_RUNTIME=READY")
        print("M6_CHANGED_FILES=11")
        if args.check:
            print("M6_APP_SOURCE_UNCHANGED=YES")
            return 0

        # Validate entire patch in a separate staging tree BEFORE any write.
        with tempfile.TemporaryDirectory(prefix="m6-runtime-stage-") as tmp:
            stage = Path(tmp)
            for rel, info in entries.items():
                if info["before"] is None:
                    continue
                copy = stage / rel
                copy.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(app / rel, copy)
            run = subprocess.run(["patch", "--batch", "--fuzz=0", "--forward", "-p1", "-i", str(patch)],
                                 cwd=stage, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=30)
            if run.returncode:
                return fatal("PATCH_DRY_STAGING_FAILED")
            for rel, info in entries.items():
                if not (stage / rel).is_file() or blob((stage / rel).read_bytes()) != info["after"]:
                    return fatal("PATCH_OUTPUT_BLOB_MISMATCH " + rel)
            data = {rel: (stage / rel).read_bytes() for rel in entries}
            data[HTTP_REL] = http_src.read_bytes()
            data[BUILDER_REL] = builder_src.read_bytes()

        originals = {}
        completed = []
        try:
            for rel, content in data.items():
                dest = app / rel
                originals[rel] = dest.read_bytes() if dest.exists() else None
                fd, name = tempfile.mkstemp(prefix=".m6-route-", dir=str(dest.parent))
                try:
                    with os.fdopen(fd, "wb") as handle:
                        handle.write(content)
                        handle.flush()
                        os.fsync(handle.fileno())
                    os.chmod(name, (dest.stat().st_mode & 0o777) if dest.exists() else 0o600)
                    os.replace(name, dest)
                    completed.append(rel)
                finally:
                    if os.path.exists(name): os.unlink(name)
        except Exception:
            for rel in reversed(completed):
                dest = app / rel
                original = originals[rel]
                if original is None:
                    dest.unlink(missing_ok=True)
                else:
                    dest.write_bytes(original)
            return fatal("WRITE_FAILED_AND_ROLLBACK_ATTEMPTED")
        print("M6_RUNTIME=SOURCE_APPLIED")
        print("M6_APK_BUILD=NOT_RUN")
        print("M6_LIVE_SERVICES=UNCHANGED")
        return 0
    except (OSError, ValueError, json.JSONDecodeError, subprocess.TimeoutExpired) as exc:
        return fatal(exc.__class__.__name__)


if __name__ == "__main__":
    sys.exit(main())
