#!/usr/bin/env python3
"""M5 two-node cutover worker. Never logs or publishes private configuration."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time

OLD = "5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90"
NEW = "4f997b60bd777516339512ac71ddbb89b96d5bf9af979286ff3bd63690ea525b"
STAGE = Path("/opt/tervon-tcp-bridge/m5/cutover-no-quota")
ROLES = {
    "homeserver": ("tervon-bridge-client.service", Path("/opt/tervon-tcp-bridge/bridge"),
                   Path("/etc/tervon-tcp-bridge/client.json"), "client", "max_local_connections"),
    "finserver": ("tervon-bridge-server.service", Path("/opt/tervon-bridge-linux-amd64"),
                  Path("/etc/tervon-tcp-bridge/server.json"), "server", "max_streams"),
}

def result(label):
    print("M5_CUTOVER_" + label, flush=True)

def run(*args, timeout=45, check=True):
    return subprocess.run(args, capture_output=True, text=True, timeout=timeout, check=check)

def digest(path):
    h = hashlib.sha256()
    with path.open("rb") as f:
        while True:
            chunk = f.read(1024 * 1024)
            if not chunk:
                break
            h.update(chunk)
    return h.hexdigest()

def safe_file(path):
    if path.is_symlink() or not path.is_file():
        raise RuntimeError("missing_or_symlink_file")

def atomic_copy(src, dest, mode=None):
    tmp = dest.with_name(dest.name + ".m5-new-" + str(os.getpid()))
    if tmp.exists() or tmp.is_symlink():
        raise RuntimeError("unexpected_temp_path")
    try:
        shutil.copy2(src, tmp)
        if mode is not None:
            os.chmod(tmp, mode)
        os.replace(tmp, dest)
    finally:
        if tmp.exists():
            tmp.unlink()

def service_pid(unit):
    s = run("systemctl", "show", unit, "-p", "MainPID", "--value").stdout.strip()
    if not s.isdecimal() or int(s) < 1:
        raise RuntimeError("service_pid_invalid")
    return int(s)

def verify_service(unit, binary, expected):
    if run("systemctl", "is-active", unit, check=False).stdout.strip() != "active":
        return False
    try:
        pid = service_pid(unit)
        return digest(binary) == expected and digest(Path("/proc/" + str(pid) + "/exe")) == expected
    except (OSError, RuntimeError, subprocess.SubprocessError):
        return False

def validate_unit(unit, binary, config):
    # Structured systemd metadata only; never print ExecStart or file contents.
    s = run("systemctl", "show", unit, "-p", "ExecStart", "--value").stdout
    if "path=" + str(binary) not in s or " -config " + str(config) not in s:
        raise RuntimeError("unit_invocation_unexpected")
    if "path=" + str(binary) + " " not in s and "path=" + str(binary) + ";" not in s:
        raise RuntimeError("unit_binary_path_unexpected")

def conf_limit(path, mode, field):
    safe_file(path)
    with path.open(encoding="utf-8") as f:
        obj = json.load(f)
    if not isinstance(obj, dict) or obj.get("mode") != mode or type(obj.get(field)) is not int:
        raise RuntimeError("config_invalid")
    return obj, obj[field]

def backup_path(name):
    return STAGE / name

def require_backup(binary, config):
    for name in ("old-binary", "new-binary", "old-config", "new-config", "manifest.json"):
        safe_file(backup_path(name))
    meta = json.loads(backup_path("manifest.json").read_text(encoding="utf-8"))
    if meta.get("binary") != str(binary) or meta.get("config") != str(config):
        raise RuntimeError("backup_manifest_mismatch")
    if digest(backup_path("old-binary")) != OLD or digest(backup_path("new-binary")) != NEW:
        raise RuntimeError("backup_binary_hash_mismatch")
    if digest(backup_path("old-config")) != meta.get("old_config_sha"):
        raise RuntimeError("old_config_backup_hash_mismatch")
    if digest(backup_path("new-config")) != meta.get("new_config_sha"):
        raise RuntimeError("new_config_backup_hash_mismatch")
    return meta

def prepare(role, unit, binary, config, mode, field):
    if STAGE.exists():
        raise RuntimeError("stage_exists_refuse_to_overwrite")
    safe_file(binary)
    safe_file(config)
    candidate = Path(sys.argv[3])
    safe_file(candidate)
    if digest(candidate) != NEW or digest(binary) != OLD:
        raise RuntimeError("binary_hash_mismatch")
    validate_unit(unit, binary, config)
    if not verify_service(unit, binary, OLD):
        raise RuntimeError("old_service_not_verified")
    obj, limit = conf_limit(config, mode, field)
    if limit != 100:
        raise RuntimeError("expected_previous_100_quota_changed")
    STAGE.mkdir(mode=0o700, parents=True)
    os.chmod(STAGE, 0o700)
    shutil.copy2(binary, backup_path("old-binary"))
    shutil.copy2(candidate, backup_path("new-binary"))
    shutil.copy2(config, backup_path("old-config"))
    obj[field] = 0
    patched = backup_path("new-config")
    patched.write_text(json.dumps(obj, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    old_stat = config.stat()
    os.chmod(patched, old_stat.st_mode & 0o777)
    os.chown(patched, old_stat.st_uid, old_stat.st_gid)
    meta = {
        "role": role, "binary": str(binary), "config": str(config),
        "old_config_sha": digest(backup_path("old-config")),
        "new_config_sha": digest(patched),
        "old_binary_sha": OLD, "new_binary_sha": NEW
    }
    backup_path("manifest.json").write_text(json.dumps(meta, indent=2) + "\n", encoding="utf-8")
    os.chmod(backup_path("manifest.json"), 0o600)
    require_backup(binary, config)
    result(role.upper() + "_PREPARE=PASS_NO_LIVE_CHANGES")

def install_new(role, unit, binary, config, mode, field):
    meta = require_backup(binary, config)
    validate_unit(unit, binary, config)
    if not verify_service(unit, binary, OLD) or digest(config) != meta["old_config_sha"]:
        raise RuntimeError("live_source_changed_since_preparation")
    atomic_copy(backup_path("new-binary"), binary, 0o755)
    atomic_copy(backup_path("new-config"), config)
    try:
        run("systemctl", "restart", unit, timeout=45)
        for _ in range(15):
            if verify_service(unit, binary, NEW):
                _, limit = conf_limit(config, mode, field)
                if limit == 0:
                    result(role.upper() + "_SWITCH=PASS_RUNNING_NEW_SHA_AND_ZERO_CONFIG")
                    return
            time.sleep(1)
        raise RuntimeError("new_unit_health_failed")
    except Exception:
        # Always attempt rollback of this host, even if restart fails.
        result(role.upper() + "_AUTO_ROLLBACK=START")
        rollback(role, unit, binary, config, mode, field)
        raise

def rollback(role, unit, binary, config, mode, field):
    meta = require_backup(binary, config)
    if digest(binary) not in (OLD, NEW):
        raise RuntimeError("unexpected_live_binary_refuse_rollback")
    if digest(config) not in (meta["old_config_sha"], meta["new_config_sha"]):
        raise RuntimeError("unexpected_live_config_refuse_rollback")
    if verify_service(unit, binary, OLD) and digest(config) == meta["old_config_sha"]:
        result(role.upper() + "_ROLLBACK=ALREADY_OLD")
        return
    atomic_copy(backup_path("old-binary"), binary, 0o755)
    atomic_copy(backup_path("old-config"), config)
    run("systemctl", "restart", unit, timeout=45)
    for _ in range(15):
        if verify_service(unit, binary, OLD):
            _, limit = conf_limit(config, mode, field)
            if limit == 100:
                result(role.upper() + "_ROLLBACK=PASS_OLD_SHA_AND_CONFIG")
                return
        time.sleep(1)
    raise RuntimeError("rollback_health_failed")

def main():
    if os.geteuid() != 0 or len(sys.argv) not in (3, 4):
        result("ERROR=ROOT_ACTION_ROLE_REQUIRED")
        return 2
    action, role = sys.argv[1:3]
    if action not in ("prepare", "switch", "rollback") or role not in ROLES:
        result("ERROR=INVALID_ACTION_ROLE")
        return 2
    if action == "prepare" and len(sys.argv) != 4:
        result("ERROR=CANDIDATE_REQUIRED")
        return 2
    if action != "prepare" and len(sys.argv) != 3:
        result("ERROR=UNEXPECTED_CANDIDATE")
        return 2
    unit, binary, config, mode, field = ROLES[role]
    try:
        if action == "prepare":
            prepare(role, unit, binary, config, mode, field)
        elif action == "switch":
            install_new(role, unit, binary, config, mode, field)
        else:
            rollback(role, unit, binary, config, mode, field)
    except (RuntimeError, OSError, ValueError, subprocess.SubprocessError) as exc:
        # Errors must not dump config/service contents or expose PKI endpoints.
        result(role.upper() + "_" + action.upper() + "=FAIL_" + type(exc).__name__)
        return 1
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
