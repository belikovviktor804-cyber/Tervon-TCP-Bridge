#!/usr/bin/env bash
# M1 binary-only deployment. Keep /v1/stream until both hosts run M1.
# Usage: sudo bash deploy-m1-with-m0-backup.sh inspect SERVICE
#        sudo bash deploy-m1-with-m0-backup.sh install SERVICE CANDIDATE EXPECTED_OLD_SHA256
#        sudo bash deploy-m1-with-m0-backup.sh rollback SERVICE BACKUP_DIRECTORY
set -Eeuo pipefail
umask 077

# Resolve the binary from the selected live systemd unit. On Finserver
# M0 actually runs /opt/tervon-bridge-linux-amd64; the similarly named
# /opt/tervon-tcp-bridge/bridge is an inactive historical v1.0.0 artifact.
# Never overwrite a binary merely because it exists on disk.
backup_root=/opt/tervon-tcp-bridge/backups
mode=${1:-}
unit=${2:-}
case "$unit" in
  tervon-bridge-client.service|tervon-bridge-server.service) ;;
  *) echo "FAIL: explicit bridge service required" >&2; exit 2 ;;
esac
case "$mode" in inspect|install|rollback) ;; *) echo "FAIL: expected inspect/install/rollback" >&2; exit 2 ;; esac
if (( EUID != 0 )); then echo "FAIL: run as root" >&2; exit 2; fi
for cmd in systemctl sha256sum flock cp mv mktemp awk mkdir chmod readlink sleep; do
  command -v "$cmd" >/dev/null || { echo "FAIL: missing $cmd" >&2; exit 2; }
done
fragment=$(systemctl show "$unit" -p FragmentPath --value)
exec_line=$(systemctl show "$unit" -p ExecStart --value)
if [[ ! -f "$fragment" || ! "$exec_line" =~ (^|[[:space:]])path=([^[:space:];]+) ]]; then
  echo "FAIL: cannot identify the selected unit executable" >&2; exit 2
fi
binary=${BASH_REMATCH[2]}
case "$unit:$binary" in
  tervon-bridge-client.service:/opt/tervon-tcp-bridge/bridge|\
  tervon-bridge-server.service:/opt/tervon-bridge-linux-amd64|\
  tervon-bridge-server.service:/opt/tervon-tcp-bridge/bridge) ;;
  *) echo "FAIL: unexpected live executable path: $binary" >&2; exit 2 ;;
esac
if [[ ! -f "$binary" || -L "$binary" ]]; then
  echo "FAIL: selected live executable is missing or symlinked" >&2; exit 2
fi
installed_sha=$(sha256sum "$binary" | awk '{print $1}')
pid=$(systemctl show "$unit" -p MainPID --value)
running_sha=UNAVAILABLE
if [[ "$pid" =~ ^[0-9]+$ && "$pid" != 0 && -e "/proc/$pid/exe" ]]; then
  running_sha=$(sha256sum "/proc/$pid/exe" | awk '{print $1}')
fi
# Recovery must remain available even after a failed restart or an interrupted
# deployment, when the on-disk executable may differ from the running one.
if [[ "$mode" != rollback ]]; then
  if [[ "$(systemctl is-active "$unit")" != active || "$installed_sha" != "$running_sha" ]]; then
    echo "FAIL: service inactive or on-disk/running binary mismatch; inspect before install" >&2; exit 2
  fi
fi
if [[ "$mode" == inspect ]]; then
  printf 'SERVICE=%s\nUNIT=%s\nBINARY=%s\nRUNNING_PID=%s\nACTIVE_SHA256=%s\nRUNNING_SHA256=%s\n' "$unit" "$fragment" "$binary" "$pid" "$installed_sha" "$running_sha"
  exit 0
fi
mkdir -p "$backup_root"
chmod 0700 "$backup_root"
exec 9>"$backup_root/.deploy.lock"
flock -n 9 || { echo "FAIL: deployment already in progress" >&2; exit 2; }

verify_active() {
  local wanted=$1 current_pid current_sha
  [[ "$(systemctl is-active "$unit")" == active ]] || return 1
  current_pid=$(systemctl show "$unit" -p MainPID --value)
  [[ "$current_pid" =~ ^[0-9]+$ && "$current_pid" != 0 ]] || return 1
  [[ -e "/proc/$current_pid/exe" ]] || return 1
  current_sha=$(sha256sum "/proc/$current_pid/exe" | awk '{print $1}')
  [[ "$current_sha" == "$wanted" ]] || return 1
  sleep 2
  [[ "$(systemctl is-active "$unit")" == active ]] || return 1
  current_pid=$(systemctl show "$unit" -p MainPID --value)
  [[ "$current_pid" =~ ^[0-9]+$ && "$current_pid" != 0 ]] || return 1
  [[ -e "/proc/$current_pid/exe" ]] || return 1
  [[ "$(sha256sum "/proc/$current_pid/exe" | awk '{print $1}')" == "$wanted" ]]
}
replace_binary() {
  local src=$1 temp
  temp=$(mktemp "${binary}.stage.XXXXXXXX")
  chmod 0755 "$temp"
  cp "$src" "$temp"
  chmod 0755 "$temp"
  mv -f "$temp" "$binary"
}
rollback_needed=0
rollback_on_exit() {
  if (( rollback_needed == 0 )); then return; fi
  echo "M1_INSTALL=FAIL; restoring backed-up M0" >&2
  if replace_binary "$backup/bridge" && systemctl restart "$unit" && verify_active "$installed_sha"; then
    echo "M0_AUTO_ROLLBACK=PASS" >&2
  else
    echo "M0_AUTO_ROLLBACK=FAIL; use rollback command and investigate" >&2
  fi
}
trap rollback_on_exit EXIT
trap 'exit 130' HUP INT TERM
if [[ "$mode" == install ]]; then
  candidate=${3:-}
  expected=${4:-}
  if [[ ! "$expected" =~ ^[a-f0-9]{64}$ || "$installed_sha" != "$expected" ]]; then
    echo "FAIL: current M0 checksum differs from explicit expected value" >&2; exit 2
  fi
  if [[ ! -f "$candidate" || -L "$candidate" || "$candidate" == "$binary" ]]; then
    echo "FAIL: candidate must be a regular staged file" >&2; exit 2
  fi
  new_sha=$(sha256sum "$candidate" | awk '{print $1}')
  if [[ "$new_sha" == "$installed_sha" ]]; then echo "FAIL: candidate equals active binary" >&2; exit 2; fi
  backup=$(mktemp -d "$backup_root/m0-XXXXXXXX")
  chmod 0700 "$backup"
  cp -p "$binary" "$backup/bridge"
  cp -p "$fragment" "$backup/service.unit"
  if [[ -d /etc/tervon-tcp-bridge ]]; then
    cp -a /etc/tervon-tcp-bridge "$backup/config"
  fi
  printf 'UNIT=%s\nBINARY=%s\nOLD_SHA256=%s\nNEW_SHA256=%s\n' "$unit" "$binary" "$installed_sha" "$new_sha" >"$backup/manifest"
  if [[ "$(sha256sum "$backup/bridge" | awk '{print $1}')" != "$installed_sha" ]]; then
    echo "FAIL: backup integrity check" >&2; exit 1
  fi
  echo "BACKUP_DIRECTORY=$backup"
  echo "OLD_SHA256=$installed_sha"
  echo "NEW_SHA256=$new_sha"
  rollback_needed=1
  replace_binary "$candidate"
  if systemctl restart "$unit" && verify_active "$new_sha"; then
    rollback_needed=0
    echo "M1_INSTALL=PASS"
    echo "CONFIG_CHANGED=NO"
    exit 0
  fi
  echo "FAIL: M1 restart/verification" >&2
  exit 1
fi
if [[ "$mode" == rollback ]]; then
  backup=${3:-}
  if [[ -z "$backup" || -L "$backup" || ! -d "$backup" ]]; then echo "FAIL: invalid backup" >&2; exit 2; fi
  backup=$(readlink -f -- "$backup")
  if [[ ! "$backup" =~ ^/opt/tervon-tcp-bridge/backups/m0-[[:alnum:]]{8}$ ]]; then
    echo "FAIL: backup outside controlled directory" >&2; exit 2
  fi
  [[ -f "$backup/bridge" && -f "$backup/manifest" ]] || { echo "FAIL: incomplete backup" >&2; exit 2; }
  saved_unit=$(awk -F= '$1=="UNIT"{print $2}' "$backup/manifest")
  old_sha=$(awk -F= '$1=="OLD_SHA256"{print $2}' "$backup/manifest")
  saved_binary=$(awk -F= '$1=="BINARY"{print $2}' "$backup/manifest")
  [[ "$saved_unit" == "$unit" && "$saved_binary" == "$binary" && "$old_sha" =~ ^[a-f0-9]{64}$ ]] || { echo "FAIL: backup manifest mismatch" >&2; exit 2; }
  [[ "$(sha256sum "$backup/bridge" | awk '{print $1}')" == "$old_sha" ]] || { echo "FAIL: backup checksum mismatch" >&2; exit 2; }
  replace_binary "$backup/bridge"
  systemctl restart "$unit"
  verify_active "$old_sha" || { echo "M0_ROLLBACK=FAIL" >&2; exit 1; }
  echo "M0_ROLLBACK=PASS"
fi
