#!/usr/bin/env bash
# Prepare an immutable M3 Linux candidate from the SSH-only GitHub source.
# No install, no service restart, no production binary replacement.
set -Eeuo pipefail
umask 077

REPO='git@github.com:tervon-software/Tervon-TCP-Bridge.git'
SOURCE='/opt/tervon-tcp-bridge/m3/source'
STAGE_ROOT='/opt/tervon-tcp-bridge/m3'
BRANCH='work/m3-adaptive-jitter'
EXPECTED_SHA="${1:-}"

if [[ "${EUID}" != 0 ]]; then
    echo 'STAGE_M3=REQUIRES_ROOT'
    exit 2
fi
if [[ ! "${EXPECTED_SHA}" =~ ^[[:xdigit:]]{40}$ ]]; then
    echo 'STAGE_M3=EXPECTED_COMMIT_SHA_REQUIRED'
    exit 2
fi
if [[ ! -d "${SOURCE}/.git" ]]; then
    echo 'STAGE_M3=SOURCE_CHECKOUT_MISSING'
    exit 2
fi
if [[ "$(git -C "${SOURCE}" remote get-url origin)" != "${REPO}" ]]; then
    echo 'STAGE_M3=WRONG_SSH_REMOTE'
    exit 2
fi
if [[ "$(git -C "${SOURCE}" branch --show-current)" != "${BRANCH}" ]]; then
    echo 'STAGE_M3=WRONG_BRANCH'
    exit 2
fi
if [[ -n "$(git -C "${SOURCE}" status --porcelain)" ]]; then
    echo 'STAGE_M3=SOURCE_DIRTY'
    exit 2
fi
git -C "${SOURCE}" fetch --no-tags origin "${BRANCH}"
ACTUAL_SHA="$(git -C "${SOURCE}" rev-parse FETCH_HEAD)"
if [[ "${ACTUAL_SHA}" != "${EXPECTED_SHA}" ]]; then
    echo "STAGE_M3=COMMIT_MISMATCH actual=${ACTUAL_SHA}"
    exit 3
fi

mkdir -p "${STAGE_ROOT}"
STAGE="$(mktemp -d "${STAGE_ROOT}/stage-XXXXXXXX")"
mkdir -p "${STAGE}/source" "${STAGE}/dist"
# Archive an exact source commit into a new private staging directory.
git -C "${SOURCE}" archive --format=tar "${ACTUAL_SHA}" |
    tar -xf - -C "${STAGE}/source"
(
    cd "${STAGE}/source/prototype"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
        -trimpath -buildvcs=false \
        -ldflags="-s -w -X main.buildVersion=v1.0.1-m3-candidate -X main.buildCommit=${ACTUAL_SHA}" \
        -o "${STAGE}/dist/tervon-bridge-m3-linux-amd64" ./cmd/bridge
)
test -s "${STAGE}/dist/tervon-bridge-m3-linux-amd64"
chmod 0700 "${STAGE}/dist/tervon-bridge-m3-linux-amd64"
(
    cd "${STAGE}/dist"
    sha256sum tervon-bridge-m3-linux-amd64 > SHA256SUMS
    sha256sum -c SHA256SUMS
    printf 'SOURCE_COMMIT=%s\nVERSION=v1.0.1-m3-candidate\nGOOS=linux\nGOARCH=amd64\nCGO_ENABLED=0\n' "${ACTUAL_SHA}" > BUILD_INFO
)

echo 'STAGE_M3=PASS'
echo "SOURCE_COMMIT=${ACTUAL_SHA}"
echo "STAGE_DIR=${STAGE}"
echo "BINARY_SHA256=$(sha256sum "${STAGE}/dist/tervon-bridge-m3-linux-amd64" | awk '{print $1}')"
echo 'SERVICES_RESTARTED=NONE'
echo 'PRODUCTION_BINARY_REPLACED=NO'
