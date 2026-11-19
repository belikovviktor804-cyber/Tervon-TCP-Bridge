#!/usr/bin/env bash
# Unified M5.2+M5.3 evidence batch: tested M4=v1.0.1 source, zero production mutations.
set -Eeuo pipefail
SRC=/opt/tervon-tcp-bridge/m5/source
OUT=/opt/tervon-tcp-bridge/m5/results
M4=8d0aa3777c209d2825a1c11dfbae9487893cb7f5
[[ -d "$SRC/.git" && -f "$SRC/prototype/go.mod" ]] || { echo 'M5_23_FINAL=BLOCKED_SOURCE'; exit 3; }
[[ "$(git -C "$SRC" branch --show-current)" == "work/m5-linux-release" ]] || { echo 'M5_23_FINAL=BLOCKED_WRONG_BRANCH'; exit 3; }
[[ "$(git -C "$SRC" remote get-url origin)" == "git@github.com:tervon-software/Tervon-TCP-Bridge.git" ]] || { echo 'M5_23_FINAL=BLOCKED_REMOTE'; exit 3; }
[[ -z "$(git -C "$SRC" status --porcelain)" ]] || { echo 'M5_23_FINAL=BLOCKED_DIRTY_SOURCE'; exit 3; }
git -C "$SRC" cat-file -e "$M4^{commit}" || { echo 'M5_23_FINAL=BLOCKED_M4_SOURCE'; exit 3; }
# Tests must exercise *exactly* M4 program source, not a later changed implementation.
git -C "$SRC" diff --quiet "$M4" HEAD -- prototype/ || { echo 'M5_23_FINAL=BLOCKED_SOURCE_DRIFT'; exit 3; }
mkdir -p "$OUT"
RUN=$(mktemp -d "$OUT/m5.23-XXXXXXXX")
chmod 700 "$RUN"
mkdir -p "$RUN/home" "$RUN/cache/build" "$RUN/cache/mod" "$RUN/tmp"
export HOME="$RUN/home" XDG_CACHE_HOME="$RUN/cache" GOCACHE="$RUN/cache/build"
export GOMODCACHE="$RUN/cache/mod" TMPDIR="$RUN/tmp" GOTOOLCHAIN=local
exec > >(tee "$RUN/summary.log") 2>&1
echo "M5_23_SOURCE_COMMIT=$(git -C "$SRC" rev-parse HEAD)"
echo "M5_23_M4_SOURCE=$M4"
echo "M5_23_TIME=$(date -u +%FT%TZ)"
echo "M5_23_GO=$(go version)"
cd "$SRC/prototype"
gate() {
  local gate_name="$1"; shift
  echo "BEGIN_$gate_name"
  if "$@" > "$RUN/$gate_name.log" 2>&1; then
    echo "$gate_name=PASS"
  else
    echo "$gate_name=FAIL"
    tail -n 35 "$RUN/$gate_name.log"
    echo "M5_23_OFFLINE=FAIL"
    echo "M5_23_FINAL=FAIL_OFFLINE"
    echo "M5_23_REPORT=$RUN/summary.log"
    exit 1
  fi
}
gate M5_23_VET go vet ./...
gate M5_23_BUILD go build ./...
gate M5_23_INTEGRATION go test -count=1 -timeout=120s ./...
gate M5_23_RACE go test -race -count=1 -timeout=240s ./...
gate M5_23_JITTER_REPEAT go test -race -count=5 -timeout=120s ./jitter
gate M5_23_FRAMING_REPEAT go test -race -count=5 -timeout=120s -run '^TestJitter' ./framing
echo "M5_23_OFFLINE=PASS"
echo "M5_23_LIVE_TCP=OPERATOR_M4_PASS_EXISTING"
echo "M5_23_NUMERIC_RTT_FAIRNESS_RESOURCES=NOT_MEASURED_BY_THIS_BATCH"
echo "M5_23_REAL_WORLD_JITTER_MAX=NOT_MEASURED_BY_THIS_BATCH"
echo "M5_23_FINAL=BLOCKED_LIVE_MEASUREMENTS"
echo "M5_23_REPORT=$RUN/summary.log"
exit 3
