#!/usr/bin/env bash
# M4 cutover PREPARE ONLY. Does not install, replace or restart anything.
set -Eeuo pipefail
umask 077

ROLE="${1:-}"
CANDIDATE="${2:-}"
OLD_SHA="${3:-}"
NEW_SHA="${4:-}"
case "$ROLE" in homeserver|finserver) ;; *) echo 'M4_PREPARE=INVALID_ROLE'; exit 2 ;; esac
if (( EUID != 0 )) || [[ ! "$OLD_SHA" =~ ^[a-f0-9]{64}$ ]] ||
   [[ ! "$NEW_SHA" =~ ^[a-f0-9]{64}$ ]] || [[ "$OLD_SHA" == "$NEW_SHA" ]]; then
  echo 'M4_PREPARE=INVALID_INPUT'; exit 2
fi
unit=tervon-bridge-server.service
expected_binary=/opt/tervon-bridge-linux-amd64
[[ "$ROLE" == homeserver ]] && { unit=tervon-bridge-client.service; expected_binary=/opt/tervon-tcp-bridge/bridge; }
for dep in systemctl sha256sum cp mkdir chmod stat mktemp awk; do
  command -v "$dep" >/dev/null || { echo 'M4_PREPARE=MISSING_TOOL'; exit 2; }
done
exec_line="$(systemctl show "$unit" -p ExecStart --value)"
[[ "$exec_line" =~ (^|[[:space:]])path=([^[:space:];]+) ]] || {
  echo 'M4_PREPARE=INVALID_EXECSTART'; exit 2;
}
actual_binary="${BASH_REMATCH[2]}"
[[ "$actual_binary" == "$expected_binary" ]] || {
  echo 'M4_PREPARE=UNEXPECTED_LIVE_BINARY'; exit 2;
}
[[ "$(systemctl is-active "$unit")" == active ]] || {
  echo 'M4_PREPARE=LIVE_SERVICE_INACTIVE'; exit 2;
}
pid="$(systemctl show "$unit" -p MainPID --value)"
[[ "$pid" =~ ^[1-9][0-9]*$ && -e "/proc/$pid/exe" ]] || {
  echo 'M4_PREPARE=INVALID_LIVE_PID'; exit 2;
}
[[ -f "$actual_binary" && ! -L "$actual_binary" ]] || {
  echo 'M4_PREPARE=LIVE_BINARY_INVALID'; exit 2;
}
[[ "$(sha256sum "$actual_binary" | awk '{print $1}')" == "$OLD_SHA" &&
   "$(sha256sum "/proc/$pid/exe" | awk '{print $1}')" == "$OLD_SHA" ]] || {
  echo 'M4_PREPARE=LIVE_SHA_MISMATCH'; exit 2;
}
[[ -f "$CANDIDATE" && ! -L "$CANDIDATE" && "$CANDIDATE" != "$actual_binary" ]] || {
  echo 'M4_PREPARE=CANDIDATE_INVALID'; exit 2;
}
[[ "$(sha256sum "$CANDIDATE" | awk '{print $1}')" == "$NEW_SHA" ]] || {
  echo 'M4_PREPARE=CANDIDATE_SHA_MISMATCH'; exit 2;
}
# Only durable preparation in M4-specific directory; no live path writes.
root="/opt/tervon-tcp-bridge/m4/cutover-$ROLE"
mkdir -p "$root"
chmod 0700 "$root"
if [[ -e "$root/m3-bridge" || -e "$root/m4-bridge" ]]; then
  echo 'M4_PREPARE=EXISTING_BACKUP_OR_CANDIDATE; REFUSE_OVERWRITE'
  exit 3
fi
install_copy() {
  local from="$1" target="$2" temp
  temp="$(mktemp "$root/.copy-XXXXXXXX")"
  cp -p -- "$from" "$temp"
  chmod 0700 "$temp"
  mv -n -- "$temp" "$target"
  if [[ -e "$temp" ]]; then
    rm -f -- "$temp"
    echo 'M4_PREPARE=TARGET_ALREADY_EXISTS'
    return 1
  fi
}
install_copy "$actual_binary" "$root/m3-bridge"
install_copy "$CANDIDATE" "$root/m4-bridge"
[[ "$(sha256sum "$root/m3-bridge" | awk '{print $1}')" == "$OLD_SHA" &&
   "$(sha256sum "$root/m4-bridge" | awk '{print $1}')" == "$NEW_SHA" ]] || {
  echo 'M4_PREPARE=BACKUP_VERIFICATION_FAILED'; exit 1;
}
printf 'ROLE=%s\nUNIT=%s\nBINARY=%s\nM3_SHA256=%s\nM4_SHA256=%s\n' \
  "$ROLE" "$unit" "$actual_binary" "$OLD_SHA" "$NEW_SHA" > "$root/manifest"
chmod 0600 "$root/manifest"
echo 'M4_PREPARE=PASS'
echo "ROLE=$ROLE"
echo "BACKUP_ROOT=$root"
echo "M3_SHA256=$OLD_SHA"
echo "M4_SHA256=$NEW_SHA"
echo 'LIVE_BINARY_REPLACED=NO'
echo 'LIVE_SERVICE_RESTARTED=NO'
