#!/usr/bin/env bash
set -euo pipefail
RUN="${BRIDGE_TEST_RUN:?run via run-all.sh}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PIDS=()
cleanup() {
  for pid in "${PIDS[@]}"; do kill -TERM "$pid" 2>/dev/null || true; done
  for pid in "${PIDS[@]}"; do wait "$pid" 2>/dev/null || true; done
}
trap cleanup EXIT
for port in 18443 18444 19000 19001; do
  if ss -H -ltn "sport = :$port" | grep -q .; then echo "STOP: port $port occupied" >&2; exit 1; fi
done
python3 -u - <<'PY' &
import socket, threading
def handle(c):
    with c:
        try:
            while data := c.recv(65536):
                c.sendall(data)
        except OSError:
            pass
with socket.socket() as s:
    s.bind(("127.0.0.1", 19000))
    s.listen(64)
    while True:
        c, _ = s.accept()
        threading.Thread(target=handle, args=(c,), daemon=True).start()
PY
PIDS+=("$!")
python3 -u "$ROOT/tests/homeserver/blackhole-proxy.py" "$RUN/blackhole" &
PIDS+=("$!")
python3 - "$RUN/client.json" <<'PY'
import json, pathlib, sys
p = pathlib.Path(sys.argv[1])
data = json.loads(p.read_text())
data["remote_url"] = "https://localhost:18444/v1/stream"
p.write_text(json.dumps(data))
PY
"$RUN/bridge" -config "$RUN/server.json" &
SERVER_PID=$!
PIDS+=("$SERVER_PID")
"$RUN/bridge" -config "$RUN/client.json" &
CLIENT_PID=$!
PIDS+=("$CLIENT_PID")
for port in 18443 18444 19000 19001; do
  ready=0
  for i in $(seq 1 40); do
    if ss -H -ltn "sport = :$port" | grep -q .; then ready=1; break; fi
    sleep 0.25
  done
  if (( !ready )); then echo "FAIL: port $port unavailable" >&2; exit 1; fi
done
python3 -u - "$RUN/blackhole" <<'PY'
import os, pathlib, socket, sys, threading, time
flag = pathlib.Path(sys.argv[1])
def echo():
    payload = os.urandom(16384)
    with socket.create_connection(("127.0.0.1", 19001), timeout=3) as s:
        s.settimeout(5)
        s.sendall(payload)
        s.shutdown(socket.SHUT_WR)
        out = bytearray()
        while True:
            data = s.recv(65536)
            if not data: break
            out.extend(data)
        if out != payload:
            raise RuntimeError("byte mismatch")
echo()
print("PASS: healthy echo through proxy", flush=True)
ready = threading.Event()
outcome = []
def active():
    try:
        with socket.create_connection(("127.0.0.1", 19001), timeout=3) as s:
            s.settimeout(3)
            s.sendall(b"BEFORE-LOSS")
            if s.recv(11) != b"BEFORE-LOSS":
                raise RuntimeError("initial echo mismatch")
            ready.set()
            while True:
                data = s.recv(1)
                if not data:
                    outcome.append("EOF")
                    return
    except socket.timeout:
        outcome.append("TIMEOUT")
    except Exception as exc:
        outcome.append(f"ERROR: {exc!r}")
        ready.set()
t = threading.Thread(target=active, daemon=True)
t.start()
if not ready.wait(5):
    raise SystemExit("FAIL: active session not established")
if outcome:
    raise SystemExit(f"FAIL: active session setup {outcome}")
flag.touch()
print("BLACKHOLE ON: packets discarded, processes remain running", flush=True)
t.join(4)
if not outcome or outcome[0] != "TIMEOUT":
    raise SystemExit(f"FAIL: expected active TCP stall, got {outcome}")
print("PASS: active stream stalled (no silent EOF or unexpected payload)", flush=True)
flag.unlink()
print("BLACKHOLE OFF: restoring packet forwarding", flush=True)
for attempt in range(10):
    try:
        echo()
        print(f"PASS: fresh TCP stream after restoration, attempt {attempt+1}", flush=True)
        break
    except (OSError, RuntimeError):
        time.sleep(0.3)
else:
    raise SystemExit("FAIL: fresh TCP stream did not recover")
PY
kill -0 "$SERVER_PID"
kill -0 "$CLIENT_PID"
echo "PASS: both bridge processes survived simulated packet loss"
