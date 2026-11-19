#!/usr/bin/env bash
# M5.2+M5.3 adaptive policy: isolated numerical tests against exact M4 policy source.
set -Eeuo pipefail
SRC=/opt/tervon-tcp-bridge/m5/source
OUT=/opt/tervon-tcp-bridge/m5/results
M4=8d0aa3777c209d2825a1c11dfbae9487893cb7f5
[[ -f "$SRC/prototype/jitter/policy.go" && -f "$SRC/tests/m5/m5_adaptive_policy_test.go" ]] || { echo 'M5_ADAPTIVE=BLOCKED_FILES'; exit 3; }
[[ "$(git -C "$SRC" branch --show-current)" == "work/m5-linux-release" ]] || { echo 'M5_ADAPTIVE=BLOCKED_BRANCH'; exit 3; }
git -C "$SRC" diff --quiet "$M4" HEAD -- prototype/ || { echo 'M5_ADAPTIVE=BLOCKED_M4_CODE_CHANGED'; exit 3; }
mkdir -p "$OUT"
RUN="$(mktemp -d "$OUT/m5-adaptive-XXXXXXXX")"
chmod 700 "$RUN"
mkdir -p "$RUN/home" "$RUN/cache/build" "$RUN/cache/mod" "$RUN/tmp" "$RUN/work"
export HOME="$RUN/home" XDG_CACHE_HOME="$RUN/cache" GOCACHE="$RUN/cache/build"
export GOMODCACHE="$RUN/cache/mod" TMPDIR="$RUN/tmp" GOTOOLCHAIN=local
cp "$SRC/prototype/jitter/policy.go" "$RUN/work/policy.go"
cp "$SRC/tests/m5/m5_adaptive_policy_test.go" "$RUN/work/m5_adaptive_policy_test.go"
cd "$RUN/work"
echo "M5_ADAPTIVE_SOURCE=$(git -C "$SRC" rev-parse HEAD)"
echo "M5_ADAPTIVE_POLICY_M4=$M4"
echo "M5_ADAPTIVE_TIME=$(date -u +%FT%TZ)"
if GO111MODULE=off go test -race -count=2 -timeout=120s -v policy.go m5_adaptive_policy_test.go > "$RUN/tests.log" 2>&1; then
  grep -E '^=== RUN|^--- PASS|^    m5_adaptive|^PASS$|^ok[[:space:]]' "$RUN/tests.log" || true
  echo "M5_ADAPTIVE_POLICY=PASS"
  echo "M5_ADAPTIVE_REAL_NETWORK=NOT_MEASURED"
  echo "M5_ADAPTIVE_REPORT=$RUN/tests.log"
else
  tail -n 60 "$RUN/tests.log"
  echo "M5_ADAPTIVE_POLICY=FAIL"
  echo "M5_ADAPTIVE_REPORT=$RUN/tests.log"
  exit 1
fi
