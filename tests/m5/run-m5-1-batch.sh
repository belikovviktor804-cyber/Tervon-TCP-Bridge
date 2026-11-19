#!/usr/bin/env bash
# M5.1 batch: isolated provenance and source/config baseline. No services, no production config.
set -Eeuo pipefail
REPO="/opt/tervon-tcp-bridge/m5/source"
ART="/opt/tervon-tcp-bridge/m5/artifacts/tervon-bridge-linux-amd64"
EXPECTED_OLD="5c813bae9cb0c68e3d5ac1e9afb0556f13cb270b369a86ae8ffc0d20adb086e2"
EXPECTED_M4="8d0aa3777c209d2825a1c11dfbae9487893cb7f5"
EXPECTED_BRANCH="work/m5-linux-release"
LOG_ROOT="/opt/tervon-tcp-bridge/m5/results"
[[ -d "$REPO/.git" && -f "$ART" ]] || { echo "M5_1_BATCH=BLOCKED_MISSING_INPUT"; exit 3; }
[[ "$(git -C "$REPO" remote get-url origin)" == "git@github.com:tervon-software/Tervon-TCP-Bridge.git" ]] || { echo "M5_1_BATCH=FAIL_ORIGIN"; exit 2; }
[[ "$(git -C "$REPO" branch --show-current)" == "$EXPECTED_BRANCH" ]] || { echo "M5_1_BATCH=FAIL_BRANCH"; exit 2; }
[[ -z "$(git -C "$REPO" status --porcelain)" ]] || { echo "M5_1_BATCH=FAIL_DIRTY"; exit 2; }
mkdir -p "$LOG_ROOT"
RUN_DIR="$(mktemp -d "$LOG_ROOT/m5.1-XXXXXXXX")"
chmod 700 "$RUN_DIR"
exec > >(tee "$RUN_DIR/report.log") 2>&1
echo "M5_1_SOURCE_COMMIT=$(git -C "$REPO" rev-parse HEAD)"
echo "M5_1_DATE_UTC=$(date -u +%FT%TZ)"
gate() {
  local label="$1"
  shift
  if "$@"; then echo "$label=PASS"; else echo "$label=FAIL"; return 1; fi
}
# Exact public artifact identity was already verified by operator. This is a checksum assertion, not another download.
gate "M5_1_ARTIFACT_SHA" bash -c 'printf "%s  %s\n" "$1" "$2" | sha256sum --check --status' _ "$EXPECTED_OLD" "$ART"
gate "M5_1_M4_SOURCE_OBJECT" git -C "$REPO" cat-file -e "$EXPECTED_M4^{commit}"
gate "M5_1_BASELINE_DOC" test -s "$REPO/docs/M5_BASELINE.md"
gate "M5_1_ACCEPTANCE_DOC" test -s "$REPO/docs/M5_ACCEPTANCE.md"
gate "M5_1_RELEASE_TAG_REF" git -C "$REPO" ls-remote --exit-code --tags origin refs/tags/v1.0.0
echo "M5_1_V100_GO_BUILD_INFO_BEGIN"
go version -m "$ART" | sed -E '/^[[:space:]]*build[[:space:]]/d' | head -25
echo "M5_1_V100_GO_BUILD_INFO_END"
# Offline documentation/example lint; no live configs are inspected.
gate "M5_1_JSON_EXAMPLES" python3 -c 'import json, pathlib, sys; root=pathlib.Path(sys.argv[1]); names=("examples/client.json","examples/server.json"); [json.loads((root/n).read_text()) for n in names]' "$REPO"
echo "M5_1_BATCH=PASS_PROVENANCE_AND_STATIC_BASELINE"
echo "M5_1_NOTE=NO_PERFORMANCE_COMPARISON_OR_RELEASE_PASS"
echo "M5_1_REPORT=$RUN_DIR/report.log"
