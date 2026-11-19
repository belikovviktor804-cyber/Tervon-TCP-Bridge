#!/usr/bin/env bash
# Prepare GitHub Release assets from SHA-verified M5 no-quota candidate; no prod mutation.
set -Eeuo pipefail
umask 077
SRC=/opt/tervon-tcp-bridge/m5/unbounded/stage-qSxWhELI/dist/tervon-bridge-linux-amd64
EXPECTED=4f997b60bd777516339512ac71ddbb89b96d5bf9af979286ff3bd63690ea525b
ROOT=/opt/tervon-tcp-bridge/m5/release-no-quota
NAME=tervon-bridge-linux-amd64
SOURCE_COMMIT=96c136702d10b6fb9a16aa2893f6b978e8b14e86
[[ -f "$SRC" && ! -L "$SRC" ]] || { echo 'RELEASE_PREP=BLOCKED_M5_CANDIDATE_MISSING';exit 3; }
ACTUAL="$(sha256sum "$SRC" | awk '{print $1}')"
[[ "$ACTUAL" == "$EXPECTED" ]] || { echo 'RELEASE_PREP=BLOCKED_M5_SHA_MISMATCH';exit 3; }
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
ARTIFACT_ORIGIN=M5_OPERATOR_CUTOVER_PASS_BINARY
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
echo "RELEASE_PREP=PASS_VERIFIED_DEPLOYED_M5"
echo "RELEASE_VERSION=v1.0.1"
echo "RELEASE_SHA256=$EXPECTED"
echo "RELEASE_SOURCE_COMMIT=$SOURCE_COMMIT"
echo "RELEASE_DIR=$ROOT"
echo "PUBLICATION_DONE=NO"
