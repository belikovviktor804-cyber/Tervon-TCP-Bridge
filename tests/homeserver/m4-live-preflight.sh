#!/usr/bin/env bash
# M4 read-only production preflight. NEVER reads configs, keys or secrets.
set -Eeuo pipefail
umask 077
ROLE="$1"
case "$ROLE" in homeserver|finserver) ;; *) echo 'M4_PREFLIGHT=INVALID_ROLE'; exit 2 ;; esac
if (( EUID != 0 )); then echo 'M4_PREFLIGHT=ROOT_REQUIRED'; exit 2; fi
UNIT=tervon-bridge-server.service
[[ "$ROLE" == homeserver ]] && UNIT=tervon-bridge-client.service
for name in systemctl sha256sum awk; do
  command -v "$name" >/dev/null || { echo 'M4_PREFLIGHT=MISSING_DEPENDENCY'; exit 2; }
done
unit_file="$(systemctl show "$UNIT" -p FragmentPath --value)"
exec_line="$(systemctl show "$UNIT" -p ExecStart --value)"
active="$(systemctl is-active "$UNIT" || true)"
pid="$(systemctl show "$UNIT" -p MainPID --value)"
[[ -f "$unit_file" && "$exec_line" =~ (^|[[:space:]])path=([^[:space:];]+) ]] || {
  echo 'M4_PREFLIGHT=UNEXPECTED_EXECSTART'; exit 2;
}
binary="$"{BASH_REMATCH[2]}
case "$UNIT:$binary" in
  tervon-bridge-client.service:/opt/tervon-tcp-bridge/bridge|\
  tervon-bridge-server.service:/opt/tervon-bridge-linux-amd64|\
  tervon-bridge-server.service:/opt/tervon-tcp-bridge/bridge) ;;
  *) echo 'M4_PREFLIGHT=UNKNOWN_BINARY_PATH'; exit 2 ;;
esac
[[ "$active" == active && "$pid" =~ ^[1-9][0-9]*$ && -e "/proc/$pid/exe" ]] || {
  echo 'M4_PREFLIGHT=SERVICE_NOT_ACTIVE'; exit 2;
}
[[ -f "$binary" && ! -L "$binary" ]] || {
  echo 'M4_PREFLIGHT=INVALID_BINARY_FILE'; exit 2;
}
disk_sha="$(sha256sum "$binary" | awk '{print $1}')"
running_sha="$(sha256sum "/proc/$pid/exe" | awk '{print $1}')"
[[ "$disk_sha" == "$running_sha" ]] || {
  echo 'M4_PREFLIGHT=RUNNING_DISK_MISMATCH'; exit 2;
}
echo 'M4_PREFLIGHT=PASS_READ_ONLY'
echo "ROLE=$ROLE"
echo "UNIT=$UNIT"
echo "UNIT_FILE=$unit_file"
echo "BINARY=$binary"
echo "ACTIVE_SHA256=$disk_sha"
echo "RUNNING_SHA256=$running_sha"
echo "MAIN_PID=$pid"
echo "ACTIVE_STATE=$active"
echo 'SERVICE_RESTARTED=NO'
echo 'CONFIG_READ=NO'
echo 'SECRETS_EXPOSED=NO'
