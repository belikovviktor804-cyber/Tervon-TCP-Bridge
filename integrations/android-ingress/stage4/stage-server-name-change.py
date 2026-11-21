#!/usr/bin/env python3
"""Stage ONLY a new server leaf signed by the EXISTING M6 server CA.

--check is read-only. --stage creates only a candidate under existing private/.
No Docker changes, restarts, CA/key rotation, or production config modifications.
"""
import argparse
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile


def run(*cmd):
    p = subprocess.run(cmd, capture_output=True, text=True)
    if p.returncode:
        raise RuntimeError(f"{cmd[0]} {cmd[1]} failed: {p.stderr.strip()[:300]}")
    return p.stdout


def valid_dns(value):
    return len(value) <= 253 and len(value.split(".")) >= 2 and all(
        re.fullmatch(r"[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?", s)
        for s in value.split(".")
    )


def regular(file):
    if file.is_symlink() or not file.is_file():
        raise RuntimeError(f"required regular file missing: {file}")


def cert_pub(file):
    return run("openssl", "x509", "-in", str(file), "-pubkey", "-noout").strip()


def key_pub(file):
    return run("openssl", "pkey", "-in", str(file), "-pubout").strip()


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--root", type=Path, default=Path("/etc/tervon/bridge-android"))
    ap.add_argument("--domain", required=True)
    group = ap.add_mutually_exclusive_group(required=True)
    group.add_argument("--check", action="store_true")
    group.add_argument("--stage", action="store_true")
    args = ap.parse_args()
    if not valid_dns(args.domain):
        raise RuntimeError("invalid server DNS name")
    root = args.root
    if not root.is_absolute() or root.is_symlink() or not root.is_dir():
        raise RuntimeError("invalid existing Bridge production directory")
    private = root / "private"
    ca_crt, ca_key = private / "server-ca.crt", private / "server-ca.key"
    cert = root / "config/pki/server.crt"
    key = root / "config/pki/server.key"
    client_ca = root / "config/pki/client-ca.crt"
    for f in (ca_crt, ca_key, cert, key, client_ca):
        regular(f)
    if not private.is_dir() or private.is_symlink():
        raise RuntimeError("server private directory unavailable")
    run("openssl", "verify", "-purpose", "sslserver", "-CAfile", str(ca_crt), str(cert))
    if cert_pub(ca_crt) != key_pub(ca_key):
        raise RuntimeError("server CA key does not match CA certificate")
    if cert_pub(cert) != key_pub(key):
        raise RuntimeError("installed server key/certificate mismatch")
    print("M6_SERVER_NAME_CHECK=PASS")
    print("CURRENT_SAN=" + run(
        "openssl", "x509", "-in", str(cert), "-noout", "-ext", "subjectAltName"
    ).strip().replace("\n", " "))
    print("REQUESTED_FQDN=" + args.domain)
    if args.check:
        print("M6_LIVE_SERVER_CHANGED=NO")
        return
    if os.geteuid() != 0:
        raise RuntimeError("staging requires root to read the existing CA key")
    os.umask(0o077)
    stage = Path(tempfile.mkdtemp(prefix="m6-server-name-", dir=private))
    try:
        csr, ext, candidate = stage / "server.csr", stage / "server-ext.cnf", stage / "server.crt"
        ext.write_text(
            "basicConstraints=critical,CA:FALSE\n"
            "keyUsage=critical,digitalSignature\n"
            "extendedKeyUsage=serverAuth\n"
            "subjectAltName=DNS:" + args.domain + "\n"
        )
        run("openssl", "req", "-new", "-sha256", "-key", str(key),
            "-out", str(csr), "-subj", "/CN=" + args.domain)
        serial = "0x" + run("openssl", "rand", "-hex", "16").strip()
        run("openssl", "x509", "-req", "-sha256", "-days", "365",
            "-in", str(csr), "-CA", str(ca_crt), "-CAkey", str(ca_key),
            "-set_serial", serial, "-out", str(candidate), "-extfile", str(ext))
        run("openssl", "verify", "-purpose", "sslserver", "-CAfile",
            str(ca_crt), str(candidate))
        hostname = run("openssl", "x509", "-in", str(candidate),
                       "-noout", "-checkhost", args.domain)
        if "does match certificate" not in hostname or cert_pub(candidate) != key_pub(key):
            raise RuntimeError("new certificate hostname or existing server key mismatch")
        csr.unlink()
        ext.unlink()
        candidate.chmod(0o600)
        print("M6_NEW_SERVER_CERTIFICATE_STAGED=PASS")
        print("M6_STAGED_PEM=" + str(candidate))
        print("M6_LIVE_SERVER_CHANGED=NO")
    except Exception:
        shutil.rmtree(stage)
        raise


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print("M6_SERVER_NAME=FAIL: " + str(e), file=sys.stderr)
        sys.exit(1)
