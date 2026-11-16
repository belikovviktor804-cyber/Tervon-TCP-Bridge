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
        echo "FAIL: port $port occupied"
        exit 1
    fi
done

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
    listener.bind(("127.0.0.1", 19000))
    listener.listen(32)
    while True:
        conn, _ = listener.accept()
        threading.Thread(target=handle, args=(conn,), daemon=True).start()
PY
PIDS+=("$!")

"$RUN/bridge" -config "$RUN/server.json" &
PIDS+=("$!")

"$RUN/bridge" -config "$RUN/client.json" &
PIDS+=("$!")

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
        echo "FAIL: listener $port unavailable"
        exit 1
    fi
done

python3 - <<'PY'
import concurrent.futures
import os
import socket

COUNT = 8
SIZE = 128 * 1024

def test_stream(index):
    payload = os.urandom(SIZE)
    received = bytearray()

    with socket.create_connection(("127.0.0.1", 19001), timeout=5) as sock:
        sock.settimeout(20)
        sock.sendall(payload)
        sock.shutdown(socket.SHUT_WR)

        while True:
            chunk = sock.recv(65536)
            if not chunk:
                break
            received.extend(chunk)

    if received != payload:
        raise RuntimeError(f"Stream {index + 1}: data mismatch")

    return len(received)

with concurrent.futures.ThreadPoolExecutor(max_workers=COUNT) as pool:
    results = list(pool.map(test_stream, range(COUNT)))

for i, count in enumerate(results, 1):
    print(f"Stream {i}: PASS ({count} bytes)")

assert sum(results) == COUNT * SIZE
print(f"PASS: {COUNT} PARALLEL STREAMS, {sum(results)} BYTES")
PY
