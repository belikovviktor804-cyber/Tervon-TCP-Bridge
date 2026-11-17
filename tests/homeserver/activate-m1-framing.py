#!/usr/bin/env python3
"""Controlled M1 /v1 -> /v2 cutover of the existing Homeserver client.

No new service, port or listener. Run as root. Never print config contents.
'Enable' checks the new process, NOT live Internet traffic. Operator must
verify live traffic and run 'rollback BACKUP_DIR' if it fails.
"""
import argparse
import fcntl
import hashlib
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
import tempfile
import time
from urllib.parse import urlsplit, urlunsplit

UNIT = "tervon-bridge-client.service"
CONFIG = Path("/etc/tervon-tcp-bridge/client.json")
BINARY = Path("/opt/tervon-tcp-bridge/bridge")
BACKUPS = Path("/opt/tervon-tcp-bridge/backups")
M1_SHA = "e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5"


def sha256(path):
    h = hashlib.sha256()
    with open(path, "rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            h.update(block)
    return h.hexdigest()


def change_remote_url(data, expected, new):
    if not isinstance(data, dict) or data.get("mode") != "client":
        raise ValueError("expected client configuration")
    url = data.get("remote_url")
    if not isinstance(url, str):
        raise ValueError("missing remote_url")
    parsed = urlsplit(url)
    if (parsed.scheme != "https" or not parsed.netloc or parsed.username is not None
            or parsed.query or parsed.fragment or parsed.path != expected):
        raise ValueError("remote_url is not the expected HTTPS endpoint")
    updated = dict(data)
    updated["remote_url"] = urlunsplit(parsed._replace(path=new))
    return updated


def atomic_write(destination, payload, reference):
    original = reference.stat(follow_symlinks=False)
    if not stat.S_ISREG(original.st_mode) or destination.is_symlink():
        raise ValueError("unsafe configuration file")
    fd, tmp = tempfile.mkstemp(prefix=".bridge-config-", dir=str(destination.parent))
    try:
        with os.fdopen(fd, "wb") as out:
            os.fchmod(out.fileno(), stat.S_IMODE(original.st_mode))
            os.fchown(out.fileno(), original.st_uid, original.st_gid)
            out.write(payload)
            out.flush()
            os.fsync(out.fileno())
        os.replace(tmp, destination)
        dirfd = os.open(str(destination.parent), os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(dirfd)
        finally:
            os.close(dirfd)
    finally:
        if os.path.exists(tmp):
            os.unlink(tmp)


def systemctl(*args):
    return subprocess.run(["systemctl", *args], capture_output=True, text=True)


def verify_process():
    for _ in range(15):
        if systemctl("is-active", "--quiet", UNIT).returncode == 0:
            pid_result = systemctl("show", UNIT, "-p", "MainPID", "--value")
            pid = pid_result.stdout.strip()
            if pid.isdigit() and int(pid) > 0:
                exe = Path("/proc") / pid / "exe"
                try:
                    if sha256(exe) == M1_SHA:
                        time.sleep(2)
                        if systemctl("is-active", "--quiet", UNIT).returncode == 0 and sha256(exe) == M1_SHA:
                            return pid
                except OSError:
                    pass
        time.sleep(1)
    raise RuntimeError("M1 client process did not become stable")


def restart_and_verify():
    result = systemctl("restart", UNIT)
    if result.returncode:
        print("RESTART_NONZERO=YES (checking running process)")
    return verify_process()


def assert_preflight():
    if CONFIG.is_symlink() or not CONFIG.is_file():
        raise RuntimeError("config is missing or symlinked")
    if BINARY.is_symlink() or sha256(BINARY) != M1_SHA:
        raise RuntimeError("running binary is not the expected M1 candidate")
    if systemctl("is-active", "--quiet", UNIT).returncode:
        raise RuntimeError("client service is inactive")
    pid = systemctl("show", UNIT, "-p", "MainPID", "--value").stdout.strip()
    if not pid.isdigit() or int(pid) == 0 or sha256(Path("/proc") / pid / "exe") != M1_SHA:
        raise RuntimeError("running M1 binary does not match disk")


def enable():
    assert_preflight()
    original = CONFIG.read_bytes()
    updated = change_remote_url(json.loads(original), "/v1/stream", "/v2/stream")
    payload = (json.dumps(updated, ensure_ascii=False, indent=2) + "\n").encode()
    backup = Path(tempfile.mkdtemp(prefix="m1-framing-", dir=str(BACKUPS)))
    os.chmod(backup, 0o700)
    saved = backup / "client.json"
    shutil.copy2(CONFIG, saved)
    if sha256(saved) != hashlib.sha256(original).hexdigest():
        raise RuntimeError("config backup verification failed")
    print("CONFIG_BACKUP_DIRECTORY=" + str(backup), flush=True)
    changed = False
    try:
        atomic_write(CONFIG, payload, CONFIG)
        changed = True
        pid = restart_and_verify()
    except BaseException:
        if changed:
            print("V2_ACTIVATION=FAIL; restoring /v1 config", flush=True)
            try:
                atomic_write(CONFIG, saved.read_bytes(), saved)
                restart_and_verify()
                print("V1_AUTO_ROLLBACK=PASS", flush=True)
            except Exception as error:
                print("V1_AUTO_ROLLBACK=FAIL: " + str(error), flush=True)
        raise
    print("V2_ACTIVATION=PASS_PROCESS_ONLY")
    print("RUNNING_PID=" + pid)
    print("LIVE_TRAFFIC_CHECK=PENDING")


def rollback(directory):
    assert_preflight()
    backup = Path(directory)
    if (backup.is_symlink() or backup.parent != BACKUPS
            or not backup.name.startswith("m1-framing-") or not backup.is_dir()):
        raise ValueError("backup directory is not in the expected location")
    saved = backup / "client.json"
    if saved.is_symlink() or not saved.is_file():
        raise ValueError("missing config backup")
    change_remote_url(json.loads(saved.read_bytes()), "/v1/stream", "/v2/stream")
    change_remote_url(json.loads(CONFIG.read_bytes()), "/v2/stream", "/v1/stream")
    atomic_write(CONFIG, saved.read_bytes(), saved)
    pid = restart_and_verify()
    print("V1_MANUAL_ROLLBACK=PASS_PROCESS_ONLY")
    print("RUNNING_PID=" + pid)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("mode", choices=["enable", "rollback"])
    parser.add_argument("backup", nargs="?")
    args = parser.parse_args()
    if os.geteuid() != 0:
        parser.error("run as root")
    if args.mode == "enable" and args.backup:
        parser.error("enable does not take a backup argument")
    if args.mode == "rollback" and not args.backup:
        parser.error("rollback requires backup directory")
    BACKUPS.mkdir(mode=0o700, parents=True, exist_ok=True)
    with open(BACKUPS / ".m1-framing.lock", "a+b") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        if args.mode == "enable":
            enable()
        else:
            rollback(args.backup)


if __name__ == "__main__":
    main()
