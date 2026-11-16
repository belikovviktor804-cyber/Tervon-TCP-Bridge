#!/usr/bin/env bash
set -euo pipefail

RUN="${BRIDGE_TEST_RUN:-/opt/tervon-tcp-bridge-lab/e2e-20261115-175152}"
PIDS=()

cleanup() {
    for pid in "${PIDS[@]}"; do
        kill -TERM "$pid" 2>/dev/null || true
    done
    for pid in "${PIDS[@]}"; do
        wait "$pid" 2>/dev/null || true
    done
}
trap cleanup EXIT

for port in 18443 19000 19001; do
    if ss -H -ltn "sport = :$port" | grep -q .; then
        echo "STOP: port $port occupied"
        exit 1
    fi
done

echo "===== START CONTROLLED BACKEND ====="

python3 -u - <<'PY' &
import socket
import threading

def handle(conn):
    with conn:
        conn.settimeout(15)
        data = conn.recv(65536)
        if data:
            conn.sendall(data)
        # Keep the connection open until the backend process is killed.
        threading.Event().wait(60)

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

"$RUN/bridge" -config "$RUN/server.json" &
SERVER_PID=$!
PIDS+=("$SERVER_PID")

"$RUN/bridge" -config "$RUN/client.json" &
CLIENT_PID=$!
PIDS+=("$CLIENT_PID")

for port in 18443 19000 19001; do
    ready=0
    for i in $(seq 1 40); do
        if ss -H -ltn "sport = :$port" | grep -q .; then
            ready=1
            break
        fi
        sleep 0.25
    done
    if [ "$ready" -ne 1 ]; then
        echo "FAIL: port $port unavailable"
        exit 1
    fi
done

echo "===== START ACTIVE TCP SESSION ====="

python3 -u - <<'PY' &
import socket
import time

with socket.create_connection(("127.0.0.1", 19001), timeout=5) as s:
    s.settimeout(10)
    s.sendall(b"ACTIVE-SESSION-TEST")
    received = s.recv(4096)

    if received != b"ACTIVE-SESSION-TEST":
        raise SystemExit("FAIL: initial exchange mismatch")

    print("PASS: active tunnel established", flush=True)

    started = time.monotonic()

    try:
        data = s.recv(4096)
        print(f"Connection closed: {len(data)} bytes", flush=True)
    except (ConnectionResetError, BrokenPipeError, TimeoutError) as exc:
        print(f"Connection ended: {type(exc).__name__}", flush=True)

    elapsed = time.monotonic() - started

    if elapsed >= 10:
        raise SystemExit(
            f"FAIL: active session did not terminate promptly: {elapsed:.3f}s"
        )

    print(f"PASS: active session terminated in {elapsed:.3f}s")
PY

TEST_PID=$!
PIDS+=("$TEST_PID")

sleep 2

if ! kill -0 "$TEST_PID" 2>/dev/null; then
    echo "FAIL: test client exited before backend interruption"
    exit 1
fi

echo "===== FORCE BACKEND FAILURE ====="
kill -KILL "$BACKEND_PID"
wait "$BACKEND_PID" 2>/dev/null || true

echo "===== WAIT FOR ACTIVE SESSION ====="
wait "$TEST_PID"

echo "===== CHECK BRIDGE SURVIVAL ====="
kill -0 "$SERVER_PID"
kill -0 "$CLIENT_PID"

echo "PASS: both bridge processes survived backend failure"
echo "===== ACTIVE DROP TEST PASSED ====="
