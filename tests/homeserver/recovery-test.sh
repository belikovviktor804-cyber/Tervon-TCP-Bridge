#!/usr/bin/env bash
set -euo pipefail

RUN="${BRIDGE_TEST_RUN:-/opt/tervon-tcp-bridge-lab/e2e-20261115-175152}"
PIDS=()
BACKEND_PID=""

cleanup() {
    for pid in "${PIDS[@]}"; do
        kill -TERM "$pid" 2>/dev/null || true
    done
    for pid in "${PIDS[@]}"; do
        wait "$pid" 2>/dev/null || true
    done
}
trap cleanup EXIT

start_backend() {
    python3 -u - <<'PY' &
import socket
import threading

def handle(conn):
    with conn:
        while data := conn.recv(65536):
            conn.sendall(data)

with socket.socket() as listener:
    listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    listener.bind(("127.0.0.1", 19000))
    listener.listen(32)

    while True:
        conn, _ = listener.accept()
        threading.Thread(
            target=handle,
            args=(conn,),
            daemon=True
        ).start()
PY
    BACKEND_PID=$!
    PIDS+=("$BACKEND_PID")
}

wait_port() {
    local port="$1"
    for i in $(seq 1 40); do
        if ss -H -ltn "sport = :$port" | grep -q .; then
            return 0
        fi
        sleep 0.25
    done
    echo "FAIL: port $port unavailable"
    exit 1
}

check_echo() {
    python3 - <<'PY'
import os
import socket

payload = os.urandom(65536)
received = bytearray()

with socket.create_connection(("127.0.0.1", 19001), timeout=5) as s:
    s.settimeout(10)
    s.sendall(payload)
    s.shutdown(socket.SHUT_WR)

    while chunk := s.recv(65536):
        received.extend(chunk)

if received != payload:
    raise SystemExit("FAIL: data mismatch")

print("PASS: 65536 bytes transferred correctly")
PY
}

for port in 18443 19000 19001; do
    if ss -H -ltn "sport = :$port" | grep -q .; then
        echo "STOP: test port $port occupied"
        exit 1
    fi
done

echo "===== START BACKEND ====="
start_backend
wait_port 19000

echo "===== START BRIDGE ====="
"$RUN/bridge" -config "$RUN/server.json" &
PIDS+=("$!")

"$RUN/bridge" -config "$RUN/client.json" &
PIDS+=("$!")

wait_port 18443
wait_port 19001

echo "===== PHASE 1: HEALTHY ====="
check_echo

echo "===== PHASE 2: BACKEND FAILURE ====="
kill -TERM "$BACKEND_PID"
wait "$BACKEND_PID" 2>/dev/null || true

python3 - <<'PY'
import socket
import time

started = time.monotonic()

try:
    with socket.create_connection(("127.0.0.1", 19001), timeout=5) as s:
        s.settimeout(5)
        s.sendall(b"backend-failure-test")
        s.shutdown(socket.SHUT_WR)

        response = s.recv(4096)

        if response:
            raise SystemExit(
                "FAIL: unexpected data while backend unavailable"
            )

except (ConnectionResetError, BrokenPipeError, OSError):
    pass

elapsed = time.monotonic() - started

if elapsed > 6:
    raise SystemExit(
        f"FAIL: backend failure handling too slow: {elapsed:.2f}s"
    )

print(f"PASS: backend failure handled in {elapsed:.3f}s")
PY

echo "===== PHASE 3: BACKEND RECOVERY ====="
start_backend
wait_port 19000
check_echo

echo "===== RECOVERY TEST PASSED ====="
