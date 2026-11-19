#!/usr/bin/env bash
# M5.2+M5.3 short live observation of existing v1.0.1. Read-only; zero restarts/load.
set -Eeuo pipefail
SRC=/opt/tervon-tcp-bridge/m5/source
ROOT=/opt/tervon-tcp-bridge/m5/results
[[ -f "$SRC/tests/homeserver/m3_passive_live_evidence.py" ]] || { echo 'M5_LIVE=BLOCKED_SOURCE_MISSING'; exit 3; }
[[ "$(git -C "$SRC" branch --show-current)" == "work/m5-linux-release" ]] || { echo 'M5_LIVE=BLOCKED_WRONG_BRANCH'; exit 3; }
mkdir -p "$ROOT"
RUN="$(mktemp -d "$ROOT/m5.23-live-XXXXXXXX")"
chmod 700 "$RUN"
UNIT=tervon-bridge-client.service
PID="$(systemctl show "$UNIT" --value -p MainPID)"
STATE="$(systemctl show "$UNIT" --value -p ActiveState)"
if [[ "$STATE" != active || ! "$PID" =~ ^[0-9]+$ || "$PID" == 0 ]]; then
  echo 'M5_LIVE=BLOCKED_CLIENT_NOT_ACTIVE'; exit 3
fi
exe="$(readlink -f "/proc/$PID/exe")"
start_hash="$(sha256sum "/proc/$PID/exe" | cut -d' ' -f1)"
expected=5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90
[[ "$start_hash" == "$expected" ]] || { echo 'M5_LIVE=BLOCKED_UNEXPECTED_BINARY'; exit 3; }
rss_before="$(awk '/^VmRSS:/{print $2}' "/proc/$PID/status")"
cpu_before="$(ps -p "$PID" -o %cpu= | xargs)"
echo "M5_LIVE_UTC_START=$(date -u +%FT%TZ)"
echo "M5_LIVE_RUNNING_M4_SHA=PASS"
echo "M5_LIVE_RSS_BEFORE_KIB=${rss_before:-UNKNOWN}"
echo "M5_LIVE_CPU_BEFORE_PERCENT=${cpu_before:-UNKNOWN}"
set +e
python3 "$SRC/tests/homeserver/m3_passive_live_evidence.py" --duration 60 --interval 10 > "$RUN/evidence.jsonl" 2>&1
sampler_exit=$?
set -e
if ((sampler_exit != 0)); then
  echo "M5_LIVE_SAMPLER=FAIL exit=$sampler_exit"
  tail -n 2 "$RUN/evidence.jsonl"
  echo "M5_LIVE_REPORT=$RUN/evidence.jsonl"
  exit 1
fi
end_pid="$(systemctl show "$UNIT" --value -p MainPID)"
end_hash="$(sha256sum "/proc/$PID/exe" | cut -d' ' -f1)"
[[ "$end_pid" == "$PID" && "$end_hash" == "$expected" ]] || { echo 'M5_LIVE=FAIL_PROCESS_CHANGED'; exit 1; }
rss_after="$(awk '/^VmRSS:/{print $2}' "/proc/$PID/status")"
cpu_after="$(ps -p "$PID" -o %cpu= | xargs)"
echo "M5_LIVE_RSS_AFTER_KIB=${rss_after:-UNKNOWN}"
echo "M5_LIVE_CPU_AFTER_PERCENT=${cpu_after:-UNKNOWN}"
python3 - "$RUN/evidence.jsonl" <<'PY'
import json,sys
items=[json.loads(line) for line in open(sys.argv[1],encoding="utf-8")]
s=next((x for x in reversed(items) if x.get("kind")=="summary"),None)
if not s: raise SystemExit("M5_LIVE=FAIL_NO_SUMMARY")
print("M5_LIVE_SAMPLER="+s["status"])
print("M5_LIVE_SAMPLES="+str(s["samples"]))
for name,node in s["nodes"].items():
    delta=node["counter_deltas"]
    print("M5_LIVE_"+name.upper()+"_ACTIVE_START="+str(node["active_at_start"]))
    print("M5_LIVE_"+name.upper()+"_ACTIVE_END="+str(node["active_at_end"]))
    print("M5_LIVE_"+name.upper()+"_SESSION_DELTA="+str(delta["sessions_started"]))
    print("M5_LIVE_"+name.upper()+"_ERROR_DELTA="+str(sum(node["error_deltas"].values())))
print("M5_LIVE_JITTER_FRAMES="+str(s["activation"]["jitter_frames_total"]))
print("M5_LIVE_PADDING_FRAMES="+str(s["activation"]["padding_frames_total"]))
print("M5_LIVE_RTT_FAIRNESS_FRAME_MAX=NOT_MEASURED")
PY
echo "M5_LIVE=PASS_PASSIVE_OBSERVATION"
echo "M5_LIVE_REPORT=$RUN/evidence.jsonl"
