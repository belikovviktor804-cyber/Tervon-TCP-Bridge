#!/usr/bin/env python3
"""Install the staged M6 server leaf; restart ONLY the verified Android ingress.

--check is read-only. --apply retains the existing CA, keys, routes and ACL.
The old server leaf is restored and the one container restarted on failure.
No Nginx, Xray, RU-FI Bridge, or Android changes.
"""
import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile


def cmd(*args, timeout=20):
    p = subprocess.run(args, text=True, capture_output=True, timeout=timeout)
    if p.returncode:
        raise RuntimeError(f"{args[0]} {args[1]} failed (exit {p.returncode})")
    return p.stdout


def find_ingress(root):
    ids = cmd("docker", "ps", "--format", "{{.ID}}").splitlines()
    matches = []
    for cid in ids:
        state = json.loads(cmd("docker", "inspect", cid))[0]
        if state.get("Config", {}).get("Image") != "tervon-bridge-android-ingress:local" or not state.get("State", {}).get("Running"):
            continue
        mounts = state.get("Mounts", [])
        if any(x.get("Type") == "bind" and x.get("Source") == str(root / "config")
               and x.get("Destination") == "/config" and not x.get("RW", True)
               for x in mounts):
            matches.append(cid)
    if len(matches) != 1:
        raise RuntimeError("expected exactly one running, bind-mounted Android ingress container")
    return matches[0]


def validate(root, candidate, domain):
    cert = root / "config/pki/server.crt"
    key = root / "config/pki/server.key"
    ca = root / "private/server-ca.crt"
    if not re.fullmatch(r"[A-Za-z0-9.-]{4,253}", domain) or ".." in domain:
        raise RuntimeError("invalid requested domain")
    if (candidate.name != "server.crt" or
        not candidate.parent.name.startswith("m6-server-name-") or
        candidate.parent.parent != root / "private"):
        raise RuntimeError("candidate must be in the expected staging directory")
    for item in (candidate, cert, key, ca):
        if item.is_symlink() or not item.is_file():
            raise RuntimeError("required regular certificate/key file missing")
    cmd("openssl", "verify", "-purpose", "sslserver",
        "-CAfile", str(ca), str(candidate))
    check = cmd("openssl", "x509", "-in", str(candidate),
                "-noout", "-checkhost", domain)
    if "does match certificate" not in check:
        raise RuntimeError("staged certificate SAN mismatch")
    pub = cmd("openssl", "x509", "-in", str(candidate),
              "-noout", "-pubkey")
    key_pub = cmd("openssl", "pkey", "-in", str(key), "-pubout")
    if pub.strip() != key_pub.strip():
        raise RuntimeError("staged certificate does not match existing server key")
    return cert


def atomic_replace(path, data, stat):
    fd, temporary = tempfile.mkstemp(prefix=".m6-server-leaf-", dir=str(path.parent))
    try:
        with os.fdopen(fd, "wb") as stream:
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())
        os.chmod(temporary, stat.st_mode & 0o777)
        os.chown(temporary, stat.st_uid, stat.st_gid)
        os.replace(temporary, path)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path("/etc/tervon/bridge-android"))
    parser.add_argument("--candidate", required=True, type=Path)
    parser.add_argument("--domain", required=True)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--apply", action="store_true")
    a = parser.parse_args()
    root = a.root
    if not root.is_absolute() or root.is_symlink() or not root.is_dir():
        raise RuntimeError("invalid existing Bridge root")
    cert = validate(root, a.candidate, a.domain)
    container = find_ingress(root)
    print("M6_ANDROID_INGRESS_CONTAINER=" + container)
    print("M6_SERVER_LEAF_CANDIDATE=VERIFIED")
    if a.check:
        print("M6_DOMAIN_CUTOVER=READY_READ_ONLY")
        return
    if os.geteuid() != 0:
        raise RuntimeError("activation requires root")
    prior, stat = cert.read_bytes(), cert.stat()
    atomic_replace(cert, a.candidate.read_bytes(), stat)
    try:
        cmd("docker", "restart", "--time", "15", container, timeout=45)
        state = json.loads(cmd("docker", "inspect", container))[0]
        if not state.get("State", {}).get("Running"):
            raise RuntimeError("ingress not running after restart")
    except Exception:
        atomic_replace(cert, prior, stat)
        try:
            cmd("docker", "restart", "--time", "15", container, timeout=45)
        except Exception:
            pass
        raise RuntimeError("activation failed; previous certificate restored and ingress restart attempted")
    print("M6_SERVER_LEAF_INSTALLED=PASS")
    print("M6_INGRESS_ONLY_RESTARTED=PASS")
    print("M6_EXTERNAL_HTTPS_AND_ENROLLMENT=NOT_VERIFIED")


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print("M6_DOMAIN_CUTOVER=FAIL: " + str(e), file=sys.stderr)
        sys.exit(1)
