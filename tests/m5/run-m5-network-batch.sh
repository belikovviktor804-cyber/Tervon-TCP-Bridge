#!/usr/bin/env bash
# One isolated loopback RTT/fairness test. Exact M4 transport source; never touches running services.
set -Eeuo pipefail
SRC=/opt/tervon-tcp-bridge/m5/source
RESULTS=/opt/tervon-tcp-bridge/m5/results
M4=8d0aa3777c209d2825a1c11dfbae9487893cb7f5
[[ -f "$SRC/prototype/bridge.go" && -f "$SRC/tests/m5/m5_loopback_network_test.go" ]] || { echo 'M5_NETWORK=BLOCKED_MISSING_SOURCE'; exit 3; }
[[ "$(git -C "$SRC" branch --show-current)" == work/m5-linux-release ]] || { echo 'M5_NETWORK=BLOCKED_WRONG_BRANCH'; exit 3; }
[[ "$(git -C "$SRC" remote get-url origin)" == "git@github.com:tervon-software/Tervon-TCP-Bridge.git" ]] || { echo 'M5_NETWORK=BLOCKED_GIT_ORIGIN'; exit 3; }
[[ -z "$(git -C "$SRC" status --porcelain)" ]] || { echo 'M5_NETWORK=BLOCKED_DIRTY_WORKTREE'; exit 3; }
git -C "$SRC" diff --quiet "$M4" HEAD -- prototype/ || { echo 'M5_NETWORK=BLOCKED_M4_SOURCE_DRIFT'; exit 3; }
mkdir -p "$RESULTS"
RUN="$(mktemp -d "$RESULTS/m5-network-XXXXXXXX")"
chmod 700 "$RUN"
mkdir -p "$RUN/home" "$RUN/cache/build" "$RUN/cache/mod" "$RUN/tmp" "$RUN/source"
export HOME="$RUN/home" XDG_CACHE_HOME="$RUN/cache" GOCACHE="$RUN/cache/build"
export GOMODCACHE="$RUN/cache/mod" TMPDIR="$RUN/tmp" GOTOOLCHAIN=local
cp -a "$SRC/prototype/." "$RUN/source/"
cp "$SRC/tests/m5/m5_loopback_network_test.go" "$RUN/source/m5_loopback_network_test.go"
cd "$RUN/source"
echo "M5_NETWORK_START_UTC=$(date -u +%FT%TZ)"
echo "M5_NETWORK_M4_SOURCE=$M4"
echo "M5_NETWORK_BRANCH_HEAD=$(git -C "$SRC" rev-parse HEAD)"
if go test -count=1 -timeout=60s -run '^TestM5LoopbackRealTransportFairnessAndDelay$' -v . > "$RUN/test.log" 2>&1; then
  grep -E '^=== RUN|^--- PASS|M5_LOOPBACK_|^PASS$|^ok[[:space:]]' "$RUN/test.log" || true
  echo "M5_NETWORK_LOOPBACK=PASS"
  echo "M5_NETWORK_REAL_INTERSERVER_RTT=NOT_MEASURED"
  echo "M5_NETWORK_REPORT=$RUN/test.log"
else
  tail -n 80 "$RUN/test.log"
  echo "M5_NETWORK_LOOPBACK=FAIL"
  echo "M5_NETWORK_REPORT=$RUN/test.log"
  exit 1
fi
