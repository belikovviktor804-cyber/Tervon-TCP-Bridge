#!/usr/bin/env python3
"""M5 coordinated two-node cutover. Intended to run as root on Homeserver.

Prepares/restores only the two Tervon Bridge units. Source code is fetched from
GitHub over existing git SSH. Requires already-built exact-hash M5 candidate.
"""
from concurrent.futures import ThreadPoolExecutor
import hashlib
import os
from pathlib import Path
import subprocess
import sys

SOURCE = Path("/opt/tervon-tcp-bridge/m5/source")
WORKER = SOURCE / "tests/m5/m5-dual-cutover-worker.py"
CANDIDATE = Path("/opt/tervon-tcp-bridge/m5/unbounded/stage-qSxWhELI/dist/tervon-bridge-linux-amd64")
NEW = "4f997b60bd777516339512ac71ddbb89b96d5bf9af979286ff3bd63690ea525b"
REMOTE = "/opt/tervon-tcp-bridge/m5/dual-cutover"
SSH = ["ssh", "-T", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=yes", "-o", "ConnectTimeout=8", "finserver"]
SCP = ["scp", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=yes", "-o", "ConnectTimeout=8"]
REMOTE_WORKER = REMOTE + "/m5-dual-cutover-worker.py"
REMOTE_CANDIDATE = REMOTE + "/tervon-bridge-linux-amd64"

def output(line):
    print("M5_DUAL_" + line, flush=True)

def run(args, timeout=90, input_bytes=None):
    return subprocess.run(args, input=input_bytes, capture_output=True, timeout=timeout)

def checked(args, timeout=90):
    res = run(args, timeout)
    if res.returncode:
        raise RuntimeError("subprocess_failed")
    return res

def sha256(path):
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()

def worker(action, role):
    if role == "homeserver":
        args = ["python3", str(WORKER), action, role]
        if action == "prepare":
            args.append(str(CANDIDATE))
    else:
        command = "python3 " + REMOTE_WORKER + " " + action + " finserver"
        if action == "prepare":
            command += " " + REMOTE_CANDIDATE
        args = SSH + [command]
    result = run(args, timeout=125)
    if result.stdout:
        # Worker output is allowlisted status messages; config is never printed.
        print(result.stdout.decode(errors="replace"), end="", flush=True)
    if result.returncode:
        output(role.upper() + "_" + action.upper() + "=FAILED")
    return result.returncode == 0

def both(action):
    with ThreadPoolExecutor(max_workers=2) as pool:
        tasks = {role: pool.submit(worker, action, role) for role in ("homeserver", "finserver")}
        return {role: task.result() for role, task in tasks.items()}

def stage_remote():
    checked(SSH + ["install -d -m 0700 " + REMOTE])
    checked(SCP + [str(WORKER), "finserver:" + REMOTE_WORKER], timeout=50)
    checked(SCP + [str(CANDIDATE), "finserver:" + REMOTE_CANDIDATE], timeout=120)
    verify = checked(SSH + ["sha256sum " + REMOTE_CANDIDATE]).stdout.decode().split()[0]
    if verify != NEW:
        raise RuntimeError("remote_candidate_hash_mismatch")

def main():
    if os.geteuid() != 0 or len(sys.argv) != 2 or sys.argv[1] != "deploy":
        output("ERROR=ROOT_AND_DEPLOY_REQUIRED")
        return 2
    if not WORKER.is_file() or not CANDIDATE.is_file() or sha256(CANDIDATE) != NEW:
        output("ERROR=SOURCE_OR_CANDIDATE_HASH_MISMATCH")
        return 3
    if WORKER.is_symlink() or CANDIDATE.is_symlink():
        output("ERROR=SYMLINK_SOURCE_REFUSED")
        return 3
    branch = checked(["git", "-C", str(SOURCE), "branch", "--show-current"]).stdout.decode().strip()
    remote = checked(["git", "-C", str(SOURCE), "remote", "get-url", "origin"]).stdout.decode().strip()
    if branch != "work/m5-linux-release" or remote != "git@github.com:tervon-software/Tervon-TCP-Bridge.git":
        output("ERROR=WRONG_SOURCE_REPOSITORY")
        return 3

    output("PREPARE=START_NO_SERVICE_CHANGES")
    stage_remote()
    # Prepare requires both old versions active, snapshots binary/config,
    # and writes validated new copies without changing a running service.
    preps = both("prepare")
    if not all(preps.values()):
        output("PREPARE=BLOCKED_ONE_OR_BOTH_NODES")
        return 3
    output("PREPARE=PASS_BOTH_VERIFIED_OLD_VERSIONS_AND_BACKUPS")
    output("PARALLEL_SWITCH=START")
    results = both("switch")
    if all(results.values()):
        output("PARALLEL_SWITCH=PASS_BOTH_RUNNING_NEW_SHA_WITH_ZERO_QUOTA")
        output("RESULT=PASS_TWO_NODE_CUTOVER")
        return 0

    output("PARALLEL_SWITCH=FAIL_ROLLING_BACK_BOTH")
    restores = both("rollback")
    if all(restores.values()):
        output("RESULT=FAIL_CUTOVER_BOTH_ROLLED_BACK")
    else:
        output("RESULT=CRITICAL_ROLLBACK_INCOMPLETE_MANUAL_RECOVERY_REQUIRED")
    return 1

if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (OSError, RuntimeError, subprocess.TimeoutExpired, ValueError) as exc:
        # Conservative: no payload/config/SSH errors in public log.
        output("RESULT=BLOCKED_" + type(exc).__name__)
        raise SystemExit(3)
