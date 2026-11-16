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
python3 -u - "$RUN" "$SERVER_PID" "$CLIENT_PID" <<'PY'
import os
import pathlib
import signal
import socket
import subprocess
import sys
import threading
import time

run, server_pid, client_pid = sys.argv[1], int(sys.argv[2]), int(sys.argv[3])
server_exe = str(pathlib.Path(run) / "bridge")
server_config = str(pathlib.Path(run) / "server.json")

def echo():
    payload = os.urandom(32768)
    with socket.create_connection(("127.0.0.1", 19001), timeout=3) as sock:
        sock.settimeout(5)
        sock.sendall(payload)
        sock.shutdown(socket.SHUT_WR)
        chunks = []
        while True:
            chunk = sock.recv(65536)
            if not chunk:
                break
            chunks.append(chunk)
        if b"".join(chunks) != payload:
            raise RuntimeError("echo mismatch")

def wait_server():
    for _ in range(50):
        try:
            with socket.create_connection(("127.0.0.1", 18443), timeout=0.2):
                return
        except OSError:
            time.sleep(0.1)
    raise RuntimeError("decapsulator did not restart")

echo()
print("PASS: healthy baseline echo", flush=True)
ready = threading.Event()
outcome = []
def active():
    try:
        with socket.create_connection(("127.0.0.1", 19001), timeout=3) as sock:
            sock.settimeout(8)
            sock.sendall(b"ACTIVE-REMOTE-FAILURE")
            data = sock.recv(4096)
            if data != b"ACTIVE-REMOTE-FAILURE":
                raise RuntimeError(f"unexpected initial response {data!r}")
            ready.set()
            started = time.monotonic()
            data = sock.recv(1)
            if data:
                raise RuntimeError("unexpected payload after decapsulator failure")
            outcome.append(("EOF", time.monotonic() - started))
    except (ConnectionResetError, BrokenPipeError) as exc:
        outcome.append((type(exc).__name__, 0))
    except Exception as exc:
        outcome.append(("FAIL", repr(exc)))
        ready.set()

thread = threading.Thread(target=active)
thread.start()
if not ready.wait(5):
    raise SystemExit("FAIL: active stream did not establish")
if outcome and outcome[0][0] == "FAIL":
    raise SystemExit(f"FAIL: active stream setup {outcome}")
print("PASS: active stream established", flush=True)
os.kill(server_pid, signal.SIGKILL)
thread.join(timeout=9)
if thread.is_alive():
    raise SystemExit("FAIL: active stream did not terminate within 9s")
if not outcome or outcome[0][0] == "FAIL":
    raise SystemExit(f"FAIL: active stream termination {outcome}")
print(f"PASS: active stream ended on remote failure: {outcome[0]}", flush=True)
try:
    echo()
    raise SystemExit("FAIL: new echo succeeded while decapsulator was stopped")
except (OSError, RuntimeError) as exc:
    print(f"PASS: new session failed while remote down: {type(exc).__name__}", flush=True)

with subprocess.Popen([server_exe, "-config", server_config], stdout=subprocess.DEVNULL, stderr=subprocess.PIPE) as restarted:
    try:
        wait_server()
        for attempt in range(20):
            try:
                echo()
                print(f"PASS: fresh session recovered after remote restart (attempt {attempt + 1})", flush=True)
                break
            except (OSError, RuntimeError):
                time.sleep(0.25)
        else:
            raise SystemExit("FAIL: no successful fresh session after remote restart")
        os.kill(client_pid, 0)
        os.kill(restarted.pid, 0)
        print("PASS: encapsulator and restarted decapsulator alive", flush=True)
    finally:
        restarted.terminate()
        try:
            restarted.wait(timeout=3)
        except subprocess.TimeoutExpired:
            restarted.kill()
            restarted.wait()
PY
kill -0 "$CLIENT_PID"
echo "PASS: remote failure/restart test"
