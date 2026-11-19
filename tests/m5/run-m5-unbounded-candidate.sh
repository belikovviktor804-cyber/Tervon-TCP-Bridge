#!/usr/bin/env bash
# M5 no-fixed-session-quota candidate: isolated tests and build only.
# Requires a clean, up-to-date SSH checkout. NEVER modifies production.
set -Eeuo pipefail
umask 077
SOURCE=/opt/tervon-tcp-bridge/m5/source
ROOT=/opt/tervon-tcp-bridge/m5/unbounded
EXPECTED_REMOTE=git@github.com:tervon-software/Tervon-TCP-Bridge.git
[[ "$(git -C "$SOURCE" branch --show-current)" == work/m5-linux-release ]] ||
 { echo 'M5_UNBOUNDED=WRONG_BRANCH';exit 3; }
[[ "$(git -C "$SOURCE" remote get-url origin)" == "$EXPECTED_REMOTE" ]] ||
 { echo 'M5_UNBOUNDED=WRONG_REMOTE';exit 3; }
[[ -z "$(git -C "$SOURCE" status --porcelain)" ]] ||
 { echo 'M5_UNBOUNDED=DIRTY_CHECKOUT';exit 3; }
SHA="$(git -C "$SOURCE" rev-parse HEAD)"
[[ "$SHA" =~ ^[a-f0-9]{40}$ ]] || { echo 'M5_UNBOUNDED=INVALID_SHA';exit 3; }
mkdir -p "$ROOT"
STAGE="$(mktemp -d "$ROOT/stage-XXXXXXXX")"
mkdir -p "$STAGE"/{source,dist,evidence,home,cache,tmp}
chmod 0700 "$STAGE"/{home,cache,tmp}
export HOME="$STAGE/home"
export XDG_CACHE_HOME="$STAGE/cache"
export GOCACHE="$STAGE/cache/go-build"
export GOMODCACHE="$STAGE/cache/go-mod"
export TMPDIR="$STAGE/tmp"
mkdir -p "$GOCACHE" "$GOMODCACHE"
git -C "$SOURCE" archive --format=tar "$SHA" | tar -xf - -C "$STAGE/source"
echo "M5_UNBOUNDED_SOURCE_COMMIT=$SHA"
(
 cd "$STAGE/source/prototype"
 echo '===== GO VET ====='
 GOTOOLCHAIN=local go vet ./... > "$STAGE/evidence/vet.log" 2>&1
 echo '===== GO TEST ====='
 GOTOOLCHAIN=local go test -count=1 ./... > "$STAGE/evidence/test.log" 2>&1
 echo '===== GO RACE TEST ====='
 GOTOOLCHAIN=local go test -race -count=1 ./... > "$STAGE/evidence/race.log" 2>&1
 echo '===== BUILD ISOLATED CANDIDATE ====='
 CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOTOOLCHAIN=local go build \
   -trimpath -buildvcs=false \
   -ldflags="-s -w -X main.buildVersion=v1.0.1-m5-no-fixed-quota-candidate -X main.buildCommit=$SHA" \
   -o "$STAGE/dist/tervon-bridge-linux-amd64" ./cmd/bridge \
   > "$STAGE/evidence/build.log" 2>&1
)
(
 cd "$STAGE/dist"
 sha256sum tervon-bridge-linux-amd64 > SHA256SUMS
 sha256sum --check SHA256SUMS
)
printf 'SOURCE_COMMIT=%s\nVERSION=v1.0.1-m5-no-fixed-quota-candidate\nFIXED_SESSION_QUOTA_DEFAULT=0\n' "$SHA" > "$STAGE/dist/BUILD_INFO"
echo 'M5_UNBOUNDED_VET=PASS'
echo 'M5_UNBOUNDED_UNIT=PASS'
echo 'M5_UNBOUNDED_RACE=PASS'
echo 'M5_UNBOUNDED_BUILD=PASS'
echo 'M5_UNBOUNDED=PASS_STAGING_ONLY'
echo "M5_UNBOUNDED_BINARY_SHA256=$(sha256sum "$STAGE/dist/tervon-bridge-linux-amd64" | awk '{print $1}')"
echo "M5_UNBOUNDED_STAGE=$STAGE"
echo 'PRODUCTION_MODIFIED=NO'
