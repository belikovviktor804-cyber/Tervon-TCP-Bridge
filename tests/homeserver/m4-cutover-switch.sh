#!/usr/bin/env bash
# M4 binary-only cutover, exact hashes and prepared rollback required.
# Explicitly limited to selected Bridge unit; NEVER touches unrelated services.
set -Eeuo pipefail
umask 077

MODE="${1:-}"
ROLE="${2:-}"
case "$MODE:$ROLE" in
 install:homeserver|install:finserver|rollback:homeserver|rollback:finserver) ;;
 *) echo 'M4_SWITCH=INVALID_MODE_OR_ROLE'; exit 2 ;;
esac
(( EUID == 0 )) || { echo 'M4_SWITCH=ROOT_REQUIRED'; exit 2; }

OLD='df3bcdfc70dfa8f238af454cd9fa1e852188b61233799972fd38bde48c8aa499'
NEW='5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90'
unit=tervon-bridge-server.service
binary=/opt/tervon-bridge-linux-amd64
if [[ "$ROLE" == homeserver ]]; then
  unit=tervon-bridge-client.service
  binary=/opt/tervon-tcp-bridge/bridge
fi
root="/opt/tervon-tcp-bridge/m4/cutover-$ROLE"
for dep in systemctl sha256sum cp mv mktemp chmod awk flock sleep timeout; do
  command -v "$dep" >/dev/null || { echo 'M4_SWITCH=MISSING_DEPENDENCY'; exit 2; }
done
[[ -d "$root" && ! -L "$root" && -f "$root/manifest" &&
   -f "$root/m3-bridge" && ! -L "$root/m3-bridge" &&
   -f "$root/m4-bridge" && ! -L "$root/m4-bridge" ]] || {
  echo 'M4_SWITCH=BACKUP_NOT_PREPARED'; exit 2;
}
manifest_field() {
  awk -F= -v key="$1" '$1==key { print $2 }' "$root/manifest"
}
[[ "$(manifest_field ROLE)" == "$ROLE" &&
   "$(manifest_field UNIT)" == "$unit" &&
   "$(manifest_field BINARY)" == "$binary" &&
   "$(manifest_field M3_SHA256)" == "$OLD" &&
   "$(manifest_field M4_SHA256)" == "$NEW" ]] || {
  echo 'M4_SWITCH=MANIFEST_INVALID'; exit 2;
}
hash() { sha256sum "$1" | awk '{print $1}'; }
[[ "$(hash "$root/m3-bridge")" == "$OLD" &&
   "$(hash "$root/m4-bridge")" == "$NEW" ]] || {
  echo 'M4_SWITCH=PREPARED_HASH_MISMATCH'; exit 2;
}

exec_line="$(systemctl show "$unit" -p ExecStart --value)"
[[ "$exec_line" =~ (^|[[:space:]])path=([^[:space:];]+) ]] || {
  echo 'M4_SWITCH=INVALID_EXECSTART'; exit 2;
}
[[ "${BASH_REMATCH[2]}" == "$binary" && -f "$binary" && ! -L "$binary" ]] || {
  echo 'M4_SWITCH=LIVE_BINARY_UNEXPECTED'; exit 2;
}
exec 9>"$root/.switch.lock"
flock -n 9 || { echo 'M4_SWITCH=CONCURRENT_SWITCH'; exit 2; }
current="$(hash "$binary")"
if [[ "$MODE" == install ]]; then
  [[ "$(systemctl is-active "$unit")" == active && "$current" == "$OLD" ]] || {
    echo 'M4_SWITCH=OLD_VERSION_NOT_ACTIVE'; exit 2;
  }
  pid="$(systemctl show "$unit" -p MainPID --value)"
  [[ "$pid" =~ ^[1-9][0-9]*$ && -e "/proc/$pid/exe" &&
     "$(hash "/proc/$pid/exe")" == "$OLD" ]] || {
    echo 'M4_SWITCH=OLD_RUNNING_SHA_MISMATCH'; exit 2;
  }
else
  # Deliberate rollback is permitted only when M4 is on disk, or when its
  # restart failed and the unit became inactive. Never overwrite unknown code.
  [[ "$current" == "$NEW" ]] || {
    echo 'M4_SWITCH=UNKNOWN_VERSION_REFUSE_ROLLBACK'; exit 2;
  }
fi

verify_running() {
  local expected="$1" pid i
  for ((i=0; i<12; i++)); do
    if [[ "$(systemctl is-active "$unit" || true)" == active ]]; then
      pid="$(systemctl show "$unit" -p MainPID --value)"
      if [[ "$pid" =~ ^[1-9][0-9]*$ && -e "/proc/$pid/exe" ]] &&
         [[ "$(hash "/proc/$pid/exe")" == "$expected" ]] &&
         [[ "$(hash "$binary")" == "$expected" ]]; then
        sleep 2
        [[ "$(systemctl is-active "$unit" || true)" == active &&
           "$(systemctl show "$unit" -p MainPID --value)" == "$pid" ]] && return 0
      fi
    fi
    sleep 1
  done
  return 1
}
atomic_binary_copy() {
  local source="$1" tmp
  tmp="$(mktemp "$binary.m4-stage-XXXXXXXX")"
  cp -p -- "$source" "$tmp"
  chmod 0755 "$tmp"
  [[ "$(hash "$tmp")" == "$(hash "$source")" ]] || return 1
  mv -f -- "$tmp" "$binary"
}
rollback_pending=0
automatic_restore() {
  (( rollback_pending == 1 )) || return 0
  rollback_pending=0
  echo 'M4_AUTO_ROLLBACK=START'
  if atomic_binary_copy "$root/m3-bridge"; then
    timeout 25 systemctl restart "$unit" || true
    if verify_running "$OLD"; then
      echo 'M4_AUTO_ROLLBACK=PASS'
    else
      echo 'M4_AUTO_ROLLBACK=FAIL'
    fi
  else
    echo 'M4_AUTO_ROLLBACK=RESTORE_FILE_FAILED'
  fi
}
trap automatic_restore EXIT
trap 'exit 130' HUP INT TERM

if [[ "$MODE" == install ]]; then
  rollback_pending=1
  atomic_binary_copy "$root/m4-bridge"
  timeout 25 systemctl restart "$unit" || echo 'M4_RESTART_NONZERO=YES; CHECK_RUNNING_SHA'
  if ! verify_running "$NEW"; then
    echo 'M4_INSTALL=FAIL; restoring saved M3'
    exit 1
  fi
  rollback_pending=0
  echo 'M4_INSTALL=PASS_PROCESS_AND_SHA'
  echo "ROLE=$ROLE"
  echo "UNIT=$unit"
  echo "RUNNING_SHA256=$NEW"
  echo 'CONFIG_CHANGED=NO'
  echo 'UNRELATED_SERVICES_RESTARTED=NO'
else
  atomic_binary_copy "$root/m3-bridge"
  timeout 25 systemctl restart "$unit" || echo 'M3_RESTART_NONZERO=YES; CHECK_RUNNING_SHA'
  verify_running "$OLD" || { echo 'M4_ROLLBACK=FAIL'; exit 1; }
  echo 'M4_ROLLBACK=PASS'
  echo "ROLE=$ROLE"
  echo "RUNNING_SHA256=$OLD"
  echo 'UNRELATED_SERVICES_RESTARTED=NO'
fi
