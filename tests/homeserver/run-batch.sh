#!/usr/bin/env bash
# Run independent Go checks and isolated loopback integration batch concurrently.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WORK="$(mktemp -d "${TMPDIR:-/var/tmp}/tervon-bridge-batch.XXXXXXXX")"
PIDS=()
cleanup() {
  for pid in "${PIDS[@]}"; do
    kill -TERM "$pid" 2>/dev/null || true
  done
  for pid in "${PIDS[@]}"; do wait "$pid" 2>/dev/null || true; done
  rm -rf -- "$WORK"
}
trap cleanup EXIT

integration() {
  export BRIDGE_TEST_SUITE=batch
  export BRIDGE_STRESS_CONNECTIONS="${BRIDGE_STRESS_CONNECTIONS:-500}"
  export BRIDGE_HOLD_SECONDS="${BRIDGE_HOLD_SECONDS:-60}"
  export BRIDGE_HOLD_CONNECTIONS="${BRIDGE_HOLD_CONNECTIONS:-8}"
  bash "$ROOT/tests/homeserver/run-all.sh"
}
go_checks() {
  export GOCACHE="$WORK/go-cache"
  export GOTMPDIR="$WORK/go-tmp"
  mkdir -p "$GOCACHE" "$GOTMPDIR"
  cd "$ROOT/prototype"
  echo "===== GO BUILD ====="
  GOTOOLCHAIN=local go build ./...
  echo "===== GO VET ====="
  GOTOOLCHAIN=local go vet ./...
  echo "===== GO TEST ====="
  GOTOOLCHAIN=local go test -count=1 ./...
  echo "===== GO RACE ====="
  GOTOOLCHAIN=local go test -race -count=1 ./...
  echo "PASS: GO CHECKS"
}
echo "===== START PARALLEL LANES ====="
(integration) > "$WORK/integration.log" 2>&1 &
INTEGRATION_PID=$!
PIDS+=("$INTEGRATION_PID")
(go_checks) > "$WORK/go.log" 2>&1 &
GO_PID=$!
PIDS+=("$GO_PID")
echo "INTEGRATION_PID=$INTEGRATION_PID GO_PID=$GO_PID"
# Wait for both lanes even when one fails; report full logs and exit nonzero.
set +e
wait "$INTEGRATION_PID"
INTEGRATION_STATUS=$?
wait "$GO_PID"
GO_STATUS=$?
set -e
echo "===== INTEGRATION LANE (exit=$INTEGRATION_STATUS) ====="
cat "$WORK/integration.log"
echo "===== GO LANE (exit=$GO_STATUS) ====="
cat "$WORK/go.log"
if (( INTEGRATION_STATUS != 0 || GO_STATUS != 0 )); then
  echo "FAIL: batch integration=$INTEGRATION_STATUS go=$GO_STATUS" >&2
  exit 1
fi
echo "===== FULL PARALLEL BATCH PASSED ====="
