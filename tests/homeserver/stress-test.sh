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
count = int(os.environ.get("BRIDGE_STRESS_CONNECTIONS", "500"))
if not 1 <= count <= 100000 or count % 50:
    raise SystemExit("BRIDGE_STRESS_CONNECTIONS must be a multiple of 50 in range 50..100000")
def stats(pid):
    proc = pathlib.Path(f"/proc/{pid}")
    fd = len(list((proc / "fd").iterdir()))
    status = (proc / "status").read_text()
    rss = int(next(line.split()[1] for line in status.splitlines() if line.startswith("VmRSS:")))
    return fd, rss

def echo(index):
    payload = os.urandom(65536)
    with socket.create_connection(("127.0.0.1", 19001), timeout=5) as conn:
        conn.settimeout(10)
        conn.sendall(payload)
        conn.shutdown(socket.SHUT_WR)
        received = bytearray()
        while True:
            chunk = conn.recv(65536)
            if not chunk:
                break
            received.extend(chunk)
            if len(received) > len(payload):
                raise RuntimeError(f"session {index}: oversized response")
    if received != payload:
        raise RuntimeError(f"session {index}: byte mismatch")
    return len(received)

print("===== WARMUP =====", flush=True)
for i in range(10):
    echo(i)
time.sleep(1)
baseline = {name: stats(pid) for name, pid in pids.items()}
print(f"BASELINE: {baseline}", flush=True)
total = 0
started = time.monotonic()
with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
    for batch in range(count // 50):
        sizes = list(pool.map(echo, range(batch * 50, (batch + 1) * 50)))
        total += sum(sizes)
        time.sleep(0.5)
        snapshot = {name: stats(pid) for name, pid in pids.items()}
        print(f"BATCH {batch + 1}/{count // 50}: sessions={(batch + 1)*50}, bytes={total}, resources={snapshot}", flush=True)
time.sleep(2)
after = {name: stats(pid) for name, pid in pids.items()}
for name in pids:
    delta = after[name][0] - baseline[name][0]
    if delta > 8:
        raise SystemExit(f"FAIL: {name} FD growth +{delta}")
    print(f"{name}: FD delta {delta}, RSS delta {after[name][1] - baseline[name][1]} KiB", flush=True)
if total != count * 65536:
    raise SystemExit(f"FAIL: wrong byte total {total}")
print(f"PASS: {count} sessions, {total} byte-exact bytes, elapsed {time.monotonic()-started:.2f}s", flush=True)
PY
kill -0 "$SERVER_PID"
kill -0 "$CLIENT_PID"
echo "PASS: bridge processes survived stress test"
