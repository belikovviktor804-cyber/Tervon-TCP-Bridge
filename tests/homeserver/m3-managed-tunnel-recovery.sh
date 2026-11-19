#!/usr/bin/env bash
# M3 tunnel-only recovery fault injection. Run detached using systemd-run.
# Never modifies Bridge, Stats, nginx, or unrelated services.
set -Eeuo pipefail
umask 077
UNIT='tervon-bridge-fin-metrics-tunnel.service'
SOURCE='/opt/tervon-tcp-bridge/m3/source'
EVIDENCE='/opt/tervon-tcp-bridge/m3/evidence/m3-tunnel-recovery-result.txt'
ARMED=0
report() { printf '%s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" | tee -a "$EVIDENCE"; }
cleanup() {
  result=$?
  trap - EXIT
  if (( ARMED == 1 )) && [[ "$(systemctl is-active "$UNIT" 2>/dev/null || :)" != active ]]; then
    systemctl start "$UNIT" || true
    report 'TUNNEL_RESTORE_ATTEMPTED=YES'
  fi
  if ((result != 0)) && ((ARMED == 1)); then report "M3_TUNNEL_RECOVERY=FAIL exit=$result"; fi
  exit "$result"
}
trap cleanup EXIT
[[ $EUID == 0 ]] || { echo ROOT_REQUIRED; exit 2; }
[[ -n "$INVOCATION_ID" ]] || { echo DETACHED_SYSTEMD_RUN_REQUIRED; exit 2; }
[[ -f "$SOURCE/.git/config" ]] || { echo SOURCE_MISSING; exit 2; }
[[ "$(git -C "$SOURCE" remote get-url origin)" == 'git@github.com:tervon-software/Tervon-TCP-Bridge.git' ]] || { echo SSH_REMOTE_MISMATCH; exit 2; }
[[ "$(git -C "$SOURCE" branch --show-current)" == work/m3-adaptive-jitter ]] || { echo BRANCH_MISMATCH; exit 2; }
[[ "$(systemctl is-active "$UNIT")" == active ]] || { echo TUNNEL_NOT_ACTIVE; exit 2; }
[[ "$(systemctl is-enabled "$UNIT")" == enabled ]] || { echo TUNNEL_NOT_ENABLED; exit 2; }
[[ "$(systemctl show "$UNIT" -p Restart --value)" == always ]] || { echo WRONG_RESTART_POLICY; exit 2; }
old_pid="$(systemctl show "$UNIT" -p MainPID --value)"
[[ "$old_pid" =~ ^[1-9][0-9]*$ ]] || { echo MAINPID_INVALID; exit 2; }
[[ "$(readlink -f "/proc/$old_pid/exe")" == /usr/bin/ssh ]] || { echo MAINPID_NOT_SSH; exit 2; }
[[ "$(systemctl show "$UNIT" -p ActiveState --value)" == active ]] || exit 2
[[ -d "$(dirname "$EVIDENCE")" ]] || { echo EVIDENCE_DIR_MISSING; exit 2; }
ARMED=1
: > "$EVIDENCE"
report "SOURCE_COMMIT=$(git -C "$SOURCE" rev-parse HEAD)"
report "TUNNEL_PID_BEFORE=$old_pid"
report INJECT=SIGTERM_MAINPID_ONLY
systemctl kill --kill-whom=main --signal=TERM "$UNIT"
passed=0
for index in $(seq 1 45); do
  sleep 1
  new_pid="$(systemctl show "$UNIT" -p MainPID --value)"
  state="$(systemctl show "$UNIT" -p ActiveState --value)"
  if [[ "$state" == active && "$new_pid" =~ ^[1-9][0-9]*$ && "$new_pid" != "$old_pid" ]] &&
     [[ "$(readlink -f "/proc/$new_pid/exe" 2>/dev/null || :)" == /usr/bin/ssh ]]; then
    passed=1
    report "TUNNEL_PID_AFTER=$new_pid"
    break
  fi
done
((passed == 1)) || { report RESTART_NOT_CONFIRMED; exit 1; }
report M3_TUNNEL_RESTART=PASS
report END_TO_END_PUBLIC_API=REQUIRES_EXTERNAL_FINSERVER_CHECK
report BRIDGE_RESTART=NONE
report STATS_RESTART=NONE
report NGINX_RELOAD=NONE
