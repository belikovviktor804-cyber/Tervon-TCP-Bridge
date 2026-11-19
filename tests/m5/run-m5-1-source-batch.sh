#!/usr/bin/env bash
# M5.1 source baseline batch. Isolated build caches; never reads production configs or manages services.
set -Eeuo pipefail
SRC=/opt/tervon-tcp-bridge/m5/source
ROOT=/opt/tervon-tcp-bridge/m5/results
[[ -f "$SRC/prototype/go.mod" && -d "$SRC/.git" ]] || { echo "M5_1_BUILD=BLOCKED_SOURCE_MISSING"; exit 3; }
[[ "$(git -C "$SRC" branch --show-current)" == "work/m5-linux-release" ]] || { echo "M5_1_BUILD=FAIL_BRANCH"; exit 2; }
[[ "$(git -C "$SRC" remote get-url origin)" == "git@github.com:tervon-software/Tervon-TCP-Bridge.git" ]] || { echo "M5_1_BUILD=FAIL_ORIGIN"; exit 2; }
[[ -z "$(git -C "$SRC" status --porcelain)" ]] || { echo "M5_1_BUILD=FAIL_DIRTY_WORKTREE"; exit 2; }
mkdir -p "$ROOT"
RUN="$(mktemp -d "$ROOT/m5.1-build-XXXXXXXX")"
chmod 700 "$RUN"
mkdir -p "$RUN/home" "$RUN/cache/go-build" "$RUN/cache/go-mod" "$RUN/tmp" "$RUN/bin"
export HOME="$RUN/home" XDG_CACHE_HOME="$RUN/cache" GOCACHE="$RUN/cache/go-build"
export GOMODCACHE="$RUN/cache/go-mod" TMPDIR="$RUN/tmp" GOTOOLCHAIN=local
exec > >(tee "$RUN/result.log") 2>&1
cd "$SRC/prototype"
echo "M5_1_SOURCE_COMMIT=$(git -C "$SRC" rev-parse HEAD)"
echo "M5_1_GO_VERSION=$(go version)"
echo "M5_1_DATE_UTC=$(date -u +%FT%TZ)"
gate() {
 local name="$1"; shift
 echo "BEGIN_$name"
 if "$@" > "$RUN/$name.log" 2>&1; then
   echo "$name=PASS"
 else
   echo "$name=FAIL"
   tail -n 35 "$RUN/$name.log"
   echo "M5_1_BUILD=FAIL"
   echo "M5_1_REPORT=$RUN/result.log"
   exit 1
 fi
}
gate M5_1_GO_VET go vet ./...
gate M5_1_GO_BUILD go build -o "$RUN/bin/tervon-bridge" ./cmd/bridge
sha256sum "$RUN/bin/tervon-bridge" > "$RUN/bin/SHA256SUMS"
echo "M5_1_BUILD_SHA256=$(cut -d' ' -f1 "$RUN/bin/SHA256SUMS")"
gate M5_1_GO_UNIT go test -count=1 -timeout=120s ./...
gate M5_1_GO_RACE go test -race -count=1 -timeout=240s ./...
gate M5_1_JITTER_BENCH go test -run '^$' -bench '^Benchmark' -benchmem -benchtime=200ms ./jitter
gate M5_1_PADDING_BENCH go test -run '^$' -bench '^BenchmarkPaddingObserve$' -benchmem -benchtime=200ms ./padding
gate M5_1_FRAMING_BENCH go test -run '^$' -bench '^(BenchmarkShortFlowEncoding|BenchmarkSustainedFlowEncoding)$' -benchmem -benchtime=200ms ./framing
echo "M5_1_BENCHMARKS_BEGIN"
for name in M5_1_JITTER_BENCH M5_1_PADDING_BENCH M5_1_FRAMING_BENCH; do
  grep -E '^Benchmark|^PASS$|^ok[[:space:]]' "$RUN/$name.log" || true
done
echo "M5_1_BENCHMARKS_END"
echo "M5_1_BUILD=PASS_ISOLATED_SOURCE_DIAGNOSTICS"
echo "M5_1_COMPARISON=NOT_RUN"
echo "M5_1_REPORT=$RUN/result.log"
