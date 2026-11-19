#!/usr/bin/env bash
# Prepare GitHub Release assets from VERIFIED M4=v1.0.1 binary; no prod mutation.
set -Eeuo pipefail
umask 077
SRC=/opt/tervon-tcp-bridge/m4/cutover-homeserver/m4-bridge
EXPECTED=5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90
ROOT=/opt/tervon-tcp-bridge/m5/release
NAME=tervon-bridge-linux-amd64
SOURCE_COMMIT=8d0aa3777c209d2825a1c11dfbae9487893cb7f5
[[ -f "$SRC" && ! -L "$SRC" ]] || { echo 'RELEASE_PREP=BLOCKED_M4_CANDIDATE_MISSING';exit 3; }
ACTUAL="$(sha256sum "$SRC" | awk '{print $1}')"
[[ "$ACTUAL" == "$EXPECTED" ]] || { echo 'RELEASE_PREP=BLOCKED_M4_SHA_MISMATCH';exit 3; }
[[ -d "$ROOT" && ! -L "$ROOT" || ! -e "$ROOT" ]] || { echo 'RELEASE_PREP=BLOCKED_OUTPUT_PATH';exit 3; }
mkdir -p "$ROOT"
[[ ! -e "$ROOT/$NAME" && ! -e "$ROOT/SHA256SUMS" ]] || {
 echo 'RELEASE_PREP=EXISTING_ASSETS_REFUSE_OVERWRITE';exit 3;
}
STAGE="$(mktemp -d "$ROOT/.stage-XXXXXXXX")"
chmod 0700 "$STAGE"
cleanup() { rm -rf -- "$STAGE"; }
trap cleanup EXIT
install -m 0755 "$SRC" "$STAGE/$NAME"
[[ "$(sha256sum "$STAGE/$NAME" | awk '{print $1}')" == "$EXPECTED" ]] || {
 echo 'RELEASE_PREP=COPIED_SHA_MISMATCH';exit 1;
}
printf '%s  %s\n' "$EXPECTED" "$NAME" > "$STAGE/SHA256SUMS"
cat > "$STAGE/BUILD_INFO" <<EOF
VERSION=v1.0.1
ARTIFACT_ORIGIN=M4_OPERATOR_ACCEPTED_BINARY
SOURCE_COMMIT=$SOURCE_COMMIT
BINARY_SHA256=$EXPECTED
GOOS=linux
GOARCH=amd64
REPRODUCIBLE_BUILD_NOT_CLAIMED=YES
EOF
(
 cd "$STAGE"
 sha256sum -c SHA256SUMS >/dev/null
)
mv -- "$STAGE/$NAME" "$ROOT/$NAME"
mv -- "$STAGE/SHA256SUMS" "$ROOT/SHA256SUMS"
mv -- "$STAGE/BUILD_INFO" "$ROOT/BUILD_INFO"
echo "RELEASE_PREP=PASS_VERIFIED_EXISTING_M4"
echo "RELEASE_VERSION=v1.0.1"
echo "RELEASE_SHA256=$EXPECTED"
echo "RELEASE_SOURCE_COMMIT=$SOURCE_COMMIT"
echo "RELEASE_DIR=$ROOT"
echo "PUBLICATION_DONE=NO"
