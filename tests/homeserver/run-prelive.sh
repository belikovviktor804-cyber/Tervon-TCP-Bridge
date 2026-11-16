#!/usr/bin/env bash
# Heavier regression batch before controlled two-host deployment.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export BRIDGE_STRESS_CONNECTIONS="${BRIDGE_STRESS_CONNECTIONS:-1000}"
export BRIDGE_HOLD_SECONDS="${BRIDGE_HOLD_SECONDS:-120}"
export BRIDGE_HOLD_CONNECTIONS="${BRIDGE_HOLD_CONNECTIONS:-16}"
echo "===== PRE-LIVE REGRESSION: stress=${BRIDGE_STRESS_CONNECTIONS} hold=${BRIDGE_HOLD_SECONDS}s connections=${BRIDGE_HOLD_CONNECTIONS} ====="
exec /bin/bash "$ROOT/tests/homeserver/run-batch.sh"
