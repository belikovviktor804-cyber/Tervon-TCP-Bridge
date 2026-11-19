#!/usr/bin/env bash
# M5.2+M5.3 real-user load: observation only; no test traffic or service changes.
set -Eeuo pipefail
SRC=/opt/tervon-tcp-bridge/m5/source
ROOT=/opt/tervon-tcp-bridge/m5/results
UNIT=tervon-bridge-client.service
EXPECTED=5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90
[[ -f "$SRC/tests/homeserver/m3_passive_live_evidence.py" ]] || { echo 'M5_LOAD=BLOCKED_SOURCE'; exit 3; }
[[ "$(git -C "$SRC" branch --show-current)" == work/m5-linux-release ]] || { echo 'M5_LOAD=BLOCKED_BRANCH'; exit 3; }
PID="$(systemctl show "$UNIT" -p MainPID --value)"
[[ "$PID" =~ ^[0-9]+$ && "$PID" != 0 ]] || { echo 'M5_LOAD=BLOCKED_PROCESS'; exit 3; }
[[ "$(systemctl show "$UNIT" -p ActiveState --value)" == active ]] || { echo 'M5_LOAD=BLOCKED_INACTIVE'; exit 3; }
[[ "$(sha256sum "/proc/$PID/exe" | awk '{print $1}')" == "$EXPECTED" ]] || { echo 'M5_LOAD=BLOCKED_M4_SHA'; exit 3; }
mkdir -p "$ROOT"
RUN="$(mktemp -d "$ROOT/m5.23-userload-XXXXXXXX")"
chmod 700 "$RUN"
echo "M5_LOAD_START_UTC=$(date -u +%FT%TZ)"
echo "M5_LOAD_SOURCE_COMMIT=$(git -C "$SRC" rev-parse HEAD)"
echo "M5_LOAD_RUNNING_M4_SHA=PASS"
echo "M5_LOAD_DURATION_SECONDS=180"
echo "M5_LOAD_ACTION=PLAY_YOUTUBE_AND_RUN_SPEEDTEST_ON_PC_AND_TABLET_NOW"
(
 while kill -0 "$PID" 2>/dev/null; do
   ps -p "$PID" -o rss=,%cpu= 2>/dev/null || true
   sleep 5
 done
) > "$RUN/process.txt" &
MONITOR=$!
cleanup() { kill "$MONITOR" 2>/dev/null || true; wait "$MONITOR" 2>/dev/null || true; }
trap cleanup EXIT
set +e
python3 -u "$SRC/tests/homeserver/m3_passive_live_evidence.py" --duration 180 --interval 10 \
  | tee "$RUN/telemetry.jsonl" \
  | python3 -u -c 'import json,sys
for line in sys.stdin:
 try:
  event=json.loads(line)
  if event.get("kind")=="sample":
   n=event["nodes"]
   print("M5_LOAD_SAMPLE=%s HS_ACTIVE=%s FI_ACTIVE=%s HS_PAD=%s FI_PAD=%s HS_JITTER=%s FI_JITTER=%s" % (event["step"],n["homeserver"]["sessions_active"],n["finserver"]["sessions_active"],n["homeserver"]["padding_frames_delta"],n["finserver"]["padding_frames_delta"],n["homeserver"]["jitter_frames_delta"],n["finserver"]["jitter_frames_delta"]),flush=True)
 except (ValueError,KeyError,TypeError):
  print("M5_LOAD_SAMPLE_PARSE_ERROR",flush=True)'
COLLECT=$?
set -e
cleanup
trap - EXIT
if ((COLLECT != 0)); then
 echo "M5_LOAD=FAIL_COLLECTOR_$COLLECT"
 echo "M5_LOAD_REPORT=$RUN/telemetry.jsonl"
 exit 1
fi
END_PID="$(systemctl show "$UNIT" -p MainPID --value)"
[[ "$END_PID" == "$PID" ]] || { echo 'M5_LOAD=FAIL_PROCESS_RESTARTED'; exit 1; }
[[ "$(sha256sum "/proc/$PID/exe" | awk '{print $1}')" == "$EXPECTED" ]] || { echo 'M5_LOAD=FAIL_BINARY_CHANGED'; exit 1; }
python3 - "$RUN/telemetry.jsonl" "$RUN/process.txt" <<'PY'
import json,sys
events=[json.loads(line) for line in open(sys.argv[1],encoding="utf-8")]
summary=next((e for e in reversed(events) if e.get("kind")=="summary"),None)
if not summary:
    print("M5_LOAD=FAIL_NO_SUMMARY")
    sys.exit(1)
readings=[]
for line in open(sys.argv[2],encoding="utf-8"):
    v=line.split()
    if len(v)==2:
        try: readings.append((int(v[0]),float(v[1])))
        except ValueError: pass
print("M5_LOAD_SAMPLES="+str(summary["samples"]))
if readings:
    print("M5_LOAD_HOMESERVER_RSS_KIB_FIRST_LAST_PEAK="+",".join(str(x) for x in (readings[0][0],readings[-1][0],max(x[0] for x in readings))))
    print("M5_LOAD_HOMESERVER_CPU_LIFETIME_PERCENT_FIRST_LAST="+",".join(str(x) for x in (readings[0][1],readings[-1][1])))
print("M5_LOAD_OBSERVED_ACTIVATION="+summary["activation"]["status"])
print("M5_LOAD_JITTER_FRAME_MAX_WALLCLOCK=NOT_MEASURED_BY_COUNTERS")
print("M5_LOAD_INTERSERVER_RTT_AND_PER_STREAM_FAIRNESS=NOT_MEASURED")
total_bytes=0
total_errors=0
for role,node in summary["nodes"].items():
    key=role.upper()
    d=node["counter_deltas"]
    useful=d["upload_bytes"]+d["download_bytes"]
    total_bytes+=useful
    errors=sum(node["error_deltas"].values())
    total_errors+=errors
    pad=d["padding_upload_frames"]+d["padding_download_frames"]
    jitter=d["jitter_upload_frames"]+d["jitter_download_frames"]
    delay=d["jitter_upload_delay_ns"]+d["jitter_download_delay_ns"]
    print("M5_LOAD_"+key+"_ACTIVE_START_END="+str(node["active_at_start"])+","+str(node["active_at_end"]))
    print("M5_LOAD_"+key+"_USEFUL_MIB="+format(useful/1048576,".2f"))
    print("M5_LOAD_"+key+"_SESSIONS_STARTED="+str(d["sessions_started"]))
    print("M5_LOAD_"+key+"_ERRORS="+str(errors))
    print("M5_LOAD_"+key+"_PAD_FRAMES="+str(pad))
    print("M5_LOAD_"+key+"_JITTER_FRAMES="+str(jitter))
    print("M5_LOAD_"+key+"_JITTER_DELAY_MS_SUM="+format(delay/1e6,".3f"))
    if jitter: print("M5_LOAD_"+key+"_JITTER_DELAY_MS_AVG="+format(delay/jitter/1e6,".3f"))
if not total_bytes:
    print("M5_LOAD=NO_BRIDGE_TRAFFIC_OBSERVED")
elif total_errors:
    print("M5_LOAD=ERROR_COUNTERS_INCREASED")
else:
    print("M5_LOAD=PASS_TWO_NODE_TRAFFIC_AND_ERROR_FREE_WINDOW")
PY
echo "M5_LOAD_REPORT=$RUN/telemetry.jsonl"
echo "M5_LOAD_PROCESS_REPORT=$RUN/process.txt"
