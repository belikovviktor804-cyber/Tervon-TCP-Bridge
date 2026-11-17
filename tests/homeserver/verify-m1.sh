#!/usr/bin/env bash
# M1 code-only verification on an isolated checkout. No deployment or service changes.
set -Eeuo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: bash tests/homeserver/verify-m1.sh /path/to/isolated/Tervon-TCP-Bridge" >&2
  exit 2
fi

repo="$(realpath -- "$1")"
if [[ ! -d "$repo/.git" || ! -f "$repo/prototype/go.mod" ]]; then
  echo "FAIL: expected an isolated Git checkout containing prototype/go.mod" >&2
  exit 2
fi
case "$repo" in
  /opt/tervon/current|/opt/tervon/Release|/opt/tervon/Release/*|/opt/tervonvpn|/opt/tervonvpn/*)
    echo "FAIL: refusing a production/release directory" >&2
    exit 2 ;;
esac

cd "$repo"
remote="$(git remote get-url origin)"
if [[ "$remote" != 'git@github.com:tervon-software/Tervon-TCP-Bridge.git' ]]; then
  echo "FAIL: origin must use the approved GitHub SSH remote; got $remote" >&2
  exit 2
fi
branch="$(git branch --show-current)"
if [[ "$branch" != 'work/m1-framing' ]]; then
  echo "FAIL: expected work/m1-framing, got $branch" >&2
  exit 2
fi
if [[ -n "$(git status --porcelain)" ]]; then
  echo "FAIL: checkout is not clean; refusing to test an unrecorded tree" >&2
  exit 2
fi

# Verify the exact branch tip over SSH before testing: never certify stale code.
git fetch --quiet origin work/m1-framing
sha="$(git rev-parse HEAD)"
remote_sha="$(git rev-parse FETCH_HEAD)"
if [[ "$sha" != "$remote_sha" ]]; then
  echo "FAIL: local HEAD $sha differs from remote branch $remote_sha" >&2
  echo "Update this isolated checkout with git merge --ff-only FETCH_HEAD, then rerun." >&2
  exit 2
fi
echo "M1_VERIFY_BRANCH=$branch"
echo "M1_VERIFY_SHA=$sha"
echo "M1_VERIFY_GO=$(go version)"
echo "M1_VERIFY_ENV=isolated_checkout"
echo "M1_VERIFY_PRODUCTION_CHANGES=NONE"
cd prototype

run() {
  echo "===== RUN: $* ====="
  "$@"
  echo "===== PASS: $* ====="
}
run go build ./...
run go vet ./...
run go test -race -count=1 -timeout=120s ./...
run go test -race -count=3 -timeout=180s -run 'TestFramed|TestLegacy|TestStream' ./...
run go test -run '^$' -fuzz '^FuzzReadFrame$' -fuzztime=30s -parallel=2 -timeout=120s ./framing
run go test -run '^$' -fuzz '^FuzzStreamDecoder$' -fuzztime=30s -parallel=2 -timeout=120s ./framing

cd ..
if [[ -n "$(git status --porcelain)" ]]; then
  echo "FAIL: checkout changed during verification" >&2
  exit 1
fi
echo "M1_VERIFY_RESULT=PASS"
echo "M1_VERIFY_SHA=$sha"
echo "M1_VERIFY_NOTE=code_level_only_no_production_deployment"
