#!/usr/bin/env bash
# Reproducible loopback integration harness. Never uses production certificates.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
for cmd in go openssl python3 ss timeout; do
  command -v "$cmd" >/dev/null || { echo "Missing dependency: $cmd" >&2; exit 1; }
done
[[ "$(uname -s)" == Linux ]] || { echo "Linux required" >&2; exit 1; }
[[ "$#" -eq 0 ]] || { echo "Usage: $0 (no arguments)" >&2; exit 2; }
umask 077
RUN="$(mktemp -d "${TMPDIR:-/var/tmp}/tervon-bridge-integration.XXXXXXXX")"
cleanup() { rm -rf -- "$RUN"; }
trap cleanup EXIT
# Transient systemd units may have no HOME or XDG cache variables.
# Keep Go build cache private to this run; never write into production trees.
export GOCACHE="$RUN/go-cache"
export GOTMPDIR="$RUN/go-tmp"
mkdir -p "$GOCACHE" "$GOTMPDIR"
for port in 18443 19000 19001; do
  if ss -H -ltn "sport = :$port" | grep -q .; then
    echo "STOP: test port $port already occupied" >&2
    exit 1
  fi
done
echo "===== BUILD TEST BINARY ====="
(cd "$ROOT/prototype" && GOTOOLCHAIN=local go build -o "$RUN/bridge" ./cmd/bridge)
echo "===== GENERATE EPHEMERAL TEST PKI ====="
openssl req -x509 -newkey rsa:2048 -nodes -days 1 -sha256 \
  -subj "/CN=Tervon Test CA" -keyout "$RUN/ca.key" -out "$RUN/ca.pem" >/dev/null 2>&1
openssl req -newkey rsa:2048 -nodes -sha256 -subj "/CN=localhost" \
  -keyout "$RUN/server.key" -out "$RUN/server.csr" >/dev/null 2>&1
cat > "$RUN/server.ext" <<'EOF'
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth
subjectAltName=DNS:localhost,IP:127.0.0.1
EOF
openssl x509 -req -in "$RUN/server.csr" -CA "$RUN/ca.pem" -CAkey "$RUN/ca.key" \
  -CAcreateserial -days 1 -sha256 -extfile "$RUN/server.ext" -out "$RUN/server.pem" >/dev/null 2>&1
openssl req -newkey rsa:2048 -nodes -sha256 -subj "/CN=tervon-test-client" \
  -keyout "$RUN/client.key" -out "$RUN/client.csr" >/dev/null 2>&1
cat > "$RUN/client.ext" <<'EOF'
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature
extendedKeyUsage=clientAuth
EOF
openssl x509 -req -in "$RUN/client.csr" -CA "$RUN/ca.pem" -CAkey "$RUN/ca.key" \
  -CAcreateserial -days 1 -sha256 -extfile "$RUN/client.ext" -out "$RUN/client.pem" >/dev/null 2>&1
FINGERPRINT="$(openssl x509 -in "$RUN/client.pem" -outform DER | sha256sum | awk '{print $1}')"
python3 - "$RUN" "$FINGERPRINT" <<'PY'
import json
import pathlib
import sys
d = pathlib.Path(sys.argv[1])
fingerprint = sys.argv[2]
server = {
    "mode": "server",
    "listen": "127.0.0.1:18443",
    "cert_file": str(d / "server.pem"),
    "key_file": str(d / "server.key"),
    "ca_file": str(d / "ca.pem"),
    "routes": {"echo": [{"Name": "local-echo", "Address": "127.0.0.1:19000"}]},
    "authorized": {fingerprint: ["echo"]},
    "max_streams": 64,
}
client = {
    "mode": "client",
    "listen": "127.0.0.1:19001",
    "remote_url": "https://localhost:18443/v1/stream",
    "service": "echo",
    "cert_file": str(d / "client.pem"),
    "key_file": str(d / "client.key"),
    "ca_file": str(d / "ca.pem"),
    "server_name": "localhost",
    "max_local_connections": 64,
}
(d / "server.json").write_text(json.dumps(server))
(d / "client.json").write_text(json.dumps(client))
PY
if [[ "${BRIDGE_TEST_SUITE:-all}" == "hold" ]]; then
  tests=(hold)
elif [[ "${BRIDGE_TEST_SUITE:-all}" == "stress" ]]; then
  tests=(stress)
elif [[ "${BRIDGE_TEST_SUITE:-all}" == "all" ]]; then
  tests=(parallel recovery active-drop resource)
else
  echo "Invalid BRIDGE_TEST_SUITE (expected all, stress or hold)" >&2
  exit 2
fi
for test in "${tests[@]}"; do
  echo "===== TEST: $test ====="
  test_timeout=90
  if [[ "$test" == "hold" ]]; then
    test_timeout=$(( ${BRIDGE_HOLD_SECONDS:-60} + 45 ))
  fi
  BRIDGE_TEST_RUN="$RUN" timeout --signal=TERM --kill-after=5s "${test_timeout}s" \
    bash "$ROOT/tests/homeserver/$test-test.sh"
  echo "PASS: $test"
done
echo "===== LOOPBACK TEST SUITE PASSED: ${BRIDGE_TEST_SUITE:-all} ====="
