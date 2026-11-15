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

echo "===== START BACKEND ====="

python3 -u - <<'PY' &
import socket
import threading

def handle(conn):
    with conn:
        while True:
            data = conn.recv(65536)
            if not data:
                break
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

PIDS+=("$!")

echo "===== START BRIDGE ====="

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

fd_count() {
    find "/proc/$1/fd" -mindepth 1 -maxdepth 1 | wc -l
}

rss_kb() {
    awk '/^VmRSS:/ {print $2}' "/proc/$1/status"
}

run_sessions() {
    local count="$1"

    python3 - "$count" <<'PY'
import os
import socket
import sys

count = int(sys.argv[1])

for i in range(count):
    payload = os.urandom(32768)
    received = bytearray()

    with socket.create_connection(("127.0.0.1", 19001), timeout=5) as s:
        s.settimeout(10)
        s.sendall(payload)
        s.shutdown(socket.SHUT_WR)

        while True:
            chunk = s.recv(65536)
            if not chunk:
                break
            received.extend(chunk)

    if received != payload:
        raise SystemExit(f"FAIL: stream {i + 1} mismatch")

print(f"PASS: {count} byte-exact TCP sessions")
PY
}

echo "===== WARMUP ====="
run_sessions 10
sleep 2

SERVER_FD_BEFORE=$(fd_count "$SERVER_PID")
CLIENT_FD_BEFORE=$(fd_count "$CLIENT_PID")
SERVER_RSS_BEFORE=$(rss_kb "$SERVER_PID")
CLIENT_RSS_BEFORE=$(rss_kb "$CLIENT_PID")

echo "===== BEFORE ====="
echo "Server FD: $SERVER_FD_BEFORE"
echo "Client FD: $CLIENT_FD_BEFORE"
echo "Server RSS KB: $SERVER_RSS_BEFORE"
echo "Client RSS KB: $CLIENT_RSS_BEFORE"

echo "===== RUN 100 SESSIONS ====="
run_sessions 100

sleep 3

SERVER_FD_AFTER=$(fd_count "$SERVER_PID")
CLIENT_FD_AFTER=$(fd_count "$CLIENT_PID")
SERVER_RSS_AFTER=$(rss_kb "$SERVER_PID")
CLIENT_RSS_AFTER=$(rss_kb "$CLIENT_PID")

echo "===== AFTER ====="
echo "Server FD: $SERVER_FD_AFTER"
echo "Client FD: $CLIENT_FD_AFTER"
echo "Server RSS KB: $SERVER_RSS_AFTER"
echo "Client RSS KB: $CLIENT_RSS_AFTER"

SERVER_DELTA=$((SERVER_FD_AFTER - SERVER_FD_BEFORE))
CLIENT_DELTA=$((CLIENT_FD_AFTER - CLIENT_FD_BEFORE))

echo "===== FD DELTA ====="
echo "Server: $SERVER_DELTA"
echo "Client: $CLIENT_DELTA"

if (( SERVER_DELTA > 8 || CLIENT_DELTA > 8 )); then
    echo "FAIL: possible file descriptor leak"
    exit 1
fi

kill -0 "$SERVER_PID"
kill -0 "$CLIENT_PID"

echo "PASS: 100 SESSIONS AND RESOURCE CHECK"
