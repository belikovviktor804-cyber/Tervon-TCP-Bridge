#!/usr/bin/env bash
# Isolated M4 testing on Homeserver. NO production service or binary changes.
set -Eeuo pipefail
umask 077
REMOTE='git@github.com:tervon-software/Tervon-TCP-Bridge.git'
BRANCH='work/m4-http2-reliability'
ROOT='/opt/tervon-tcp-bridge/m4'
SOURCE="$ROOT/source"
EXPECTED_SHA="${1:-}"
if [[ "$EUID" -ne 0 || ! "$EXPECTED_SHA" =~ ^[[:xdigit:]]{40}$ ]]; then
  echo 'STAGE_M4=ROOT_AND_EXACT_SHA_REQUIRED'
  exit 2
fi
mkdir -p "$ROOT"
if [[ ! -e "$SOURCE" ]]; then
  git clone --single-branch --branch "$BRANCH" "$REMOTE" "$SOURCE"
fi
if [[ ! -d "$SOURCE/.git" ]] ||
   [[ "$(git -C "$SOURCE" remote get-url origin)" != "$REMOTE" ]] ||
   [[ "$(git -C "$SOURCE" branch --show-current)" != "$BRANCH" ]]; then
  echo 'STAGE_M4=INVALID_SSH_CHECKOUT'
  exit 2
fi
if [[ -n "$(git -C "$SOURCE" status --porcelain)" ]]; then
  echo 'STAGE_M4=CHECKOUT_DIRTY'
  exit 2
fi
git -C "$SOURCE" fetch --no-tags origin "$BRANCH"
ACTUAL_SHA="$(git -C "$SOURCE" rev-parse FETCH_HEAD)"
if [[ "$ACTUAL_SHA" != "$EXPECTED_SHA" ]]; then
  echo "STAGE_M4=SOURCE_SHA_MISMATCH actual=$ACTUAL_SHA"
  exit 3
fi
STAGE="$(mktemp -d "$ROOT/stage-XXXXXXXX")"
mkdir -p "$STAGE/source" "$STAGE/dist" "$STAGE/evidence"
git -C "$SOURCE" archive --format=tar "$ACTUAL_SHA" | tar -xf - -C "$STAGE/source"
(
  cd "$STAGE/source/prototype"
  echo '===== M4 GO VET ====='
  GOTOOLCHAIN=local go vet ./... | tee "$STAGE/evidence/vet.log"
  echo '===== M4 TARGETED RACE TESTS (LOOPBACK ONLY) ====='
  GOTOOLCHAIN=local go test -race -count=1 -timeout=180s -run '^TestM4' ./... | tee "$STAGE/evidence/m4-race.log"
  echo '===== M4 STATIC LINUX CANDIDATE ====='
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOTOOLCHAIN=local go build \
    -trimpath -buildvcs=false \
    -ldflags="-s -w -X main.buildVersion=v1.0.1-m4-candidate -X main.buildCommit=$ACTUAL_SHA" \
    -o "$STAGE/dist/tervon-bridge-m4-linux-amd64" ./cmd/bridge
)
test -s "$STAGE/dist/tervon-bridge-m4-linux-amd64"
chmod 0700 "$STAGE/dist/tervon-bridge-m4-linux-amd64"
(
  cd "$STAGE/dist"
  sha256sum tervon-bridge-m4-linux-amd64 > SHA256SUMS
  sha256sum -c SHA256SUMS
)
printf 'SOURCE_COMMIT=%s\nVERSION=v1.0.1-m4-candidate\nGOOS=linux\nGOARCH=amd64\nCGO_ENABLED=0\n' \
  "$ACTUAL_SHA" > "$STAGE/dist/BUILD_INFO"
echo 'STAGE_M4=PASS'
echo "SOURCE_COMMIT=$ACTUAL_SHA"
echo "STAGE_DIR=$STAGE"
echo "BINARY_SHA256=$(sha256sum "$STAGE/dist/tervon-bridge-m4-linux-amd64" | awk '{print $1}')"
echo 'M4_TARGETED_RACE=PASS'
echo 'PRODUCTION_BINARY_REPLACED=NO'
echo 'BRIDGE_RESTART=NONE'
echo 'STATS_RESTART=NONE'
echo 'NGINX_RELOAD=NONE'
