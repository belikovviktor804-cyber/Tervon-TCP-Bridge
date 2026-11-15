#!/usr/bin/env bash
set -euo pipefail
RUN="${BRIDGE_TEST_RUN:?run via tests/homeserver/run-all.sh with BRIDGE_TEST_SUITE=stress}"
PIDS=()
cleanup() {
  for pid in "${PIDS[@]}"; do kill -TERM "$pid" 2>/dev/null || true; done
  for pid in "${PIDS[@]}"; do wait "$pid" 2>/dev/null || true; done
}
trap cleanup EXIT
for port in 18443 19000 19001; do
  if ss -H -ltn "sport = :$port" | grep -q .; then
    echo "STOP: test port $port occupied" >&2
    exit 1
  fi
done
python3 -u - <<'PY' &
import socket
import threading
def handle(conn):
    with conn:
        try:
            while True:
                chunk = conn.recv(65536)
                if not chunk:
                    break
                conn.sendall(chunk)
        except OSError:
            pass
with socket.socket() as listener:
    listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    listener.bind(("127.0.0.1", 19000))
    listener.listen(64)
    while True:
        conn, _ = listener.accept()
        threading.Thread(target=handle, args=(conn,), daemon=True).start()
PY
PIDS+=("$!")
"$RUN/bridge" -config "$RUN/server.json" &
SERVER_PID=$!
PIDS+=("$SERVER_PID")
"$RUN/bridge" -config "$RUN/client.json" &
CLIENT_PID=$!
PIDS+=("$CLIENT_PID")
for port in 18443 19000 19001; do
  ready=0
  for i in $(seq 1 40); do
    if ss -H -ltn "sport = :$port" | grep -q .; then ready=1; break; fi
    sleep 0.25
  done
  if (( !ready )); then echo "FAIL: listener $port unavailable" >&2; exit 1; fi
done
python3 -u - "$SERVER_PID" "$CLIENT_PID" <<'PY'
import concurrent.futures
import os
import pathlib
import socket
import sys
import time

pids = {"server": int(sys.argv[1]), "client": int(sys.argv[2])}
duration = int(os.environ.get("BRIDGE_HOLD_SECONDS", "60"))
workers = int(os.environ.get("BRIDGE_HOLD_CONNECTIONS", "8"))
if not (10 <= duration <= 600 and 1 <= workers <= 32):
    raise SystemExit("BRIDGE_HOLD_SECONDS must be 10..600 and BRIDGE_HOLD_CONNECTIONS 1..32")

def stats(pid):
    proc = pathlib.Path(f"/proc/{pid}")
    fd = len(list((proc / "fd").iterdir()))
    status = (proc / "status").read_text()
    rss = int(next(line.split()[1] for line in status.splitlines() if line.startswith("VmRSS:")))
    return fd, rss

start = time.monotonic()
deadline = start + duration
def hold(index):
    count = 0
    with socket.create_connection(("127.0.0.1", 19001), timeout=5) as conn:
        conn.settimeout(5)
        while time.monotonic() < deadline:
            payload = os.urandom(4096)
            conn.sendall(payload)
            received = bytearray()
            while len(received) < len(payload):
                chunk = conn.recv(len(payload) - len(received))
                if not chunk:
                    raise RuntimeError(f"session {index}: unexpected EOF")
                received.extend(chunk)
            if received != payload:
                raise RuntimeError(f"session {index}: byte mismatch")
            count += len(payload)
            time.sleep(0.2)
        conn.shutdown(socket.SHUT_WR)
        if conn.recv(1):
            raise RuntimeError(f"session {index}: unexpected trailing data")
    return count

print(f"===== HOLD TEST: {workers} connections for {duration}s =====", flush=True)
with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as pool:
    futures = [pool.submit(hold, i) for i in range(workers)]
    time.sleep(2)
    baseline = {name: stats(pid) for name, pid in pids.items()}
    print(f"ACTIVE BASELINE: {baseline}", flush=True)
    while not all(f.done() for f in futures):
        time.sleep(min(5, max(0.1, deadline - time.monotonic())))
        snapshot = {name: stats(pid) for name, pid in pids.items()}
        print(f"ACTIVE elapsed={time.monotonic()-start:.1f}s resources={snapshot}", flush=True)
    total = sum(f.result() for f in futures)
time.sleep(2)
after = {name: stats(pid) for name, pid in pids.items()}
print(f"AFTER CLOSE: {after}", flush=True)
for name in pids:
    if after[name][0] > baseline[name][0]:
        raise SystemExit(f"FAIL: {name} retained FDs after closing sessions")
print(f"PASS: {workers} long-lived sessions, {total} byte-exact bytes, {duration}s target", flush=True)
PY
kill -0 "$SERVER_PID"
kill -0 "$CLIENT_PID"
echo "PASS: bridge processes survived hold test"
