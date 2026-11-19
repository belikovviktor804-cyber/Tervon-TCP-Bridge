#!/usr/bin/env bash
# Publish a SHA-verified EXISTING Linux v1.0.1 binary via official gh CLI/API.
# Does not compile software or touch running servers. Must run on Homeserver.
set -Eeuo pipefail
umask 077

REPO=tervon-software/Tervon-TCP-Bridge
SOURCE=/opt/tervon-tcp-bridge/m5/source
ASSETS=/opt/tervon-tcp-bridge/m5/release-no-quota
TAG=v1.0.1
EXPECTED=4f997b60bd777516339512ac71ddbb89b96d5bf9af979286ff3bd63690ea525b
ORIGIN=git@github.com:tervon-software/Tervon-TCP-Bridge.git
NOTES=docs/RELEASE_NOTES_v1.0.1.ru.md

for tool in git gh sha256sum cmp mktemp awk; do
  command -v "$tool" >/dev/null || { echo "M5_PUBLISH=BLOCKED_MISSING_TOOL_$tool"; exit 3; }
done
cd "$SOURCE"
[[ "$(git remote get-url origin)" == "$ORIGIN" ]] || { echo "M5_PUBLISH=BLOCKED_NOT_SSH_ORIGIN"; exit 3; }
[[ "$(git branch --show-current)" == work/m5-linux-release ]] || { echo "M5_PUBLISH=BLOCKED_WRONG_BRANCH"; exit 3; }
[[ -z "$(git status --porcelain)" ]] || { echo "M5_PUBLISH=BLOCKED_DIRTY_CHECKOUT"; exit 3; }
HEAD_SHA="$(git rev-parse HEAD)"
[[ "$HEAD_SHA" =~ ^[[:xdigit:]]{40}$ ]] || { echo "M5_PUBLISH=BLOCKED_INVALID_HEAD"; exit 3; }

for file in tervon-bridge-linux-amd64 SHA256SUMS BUILD_INFO; do
  [[ -f "$ASSETS/$file" && ! -L "$ASSETS/$file" ]] ||
    { echo "M5_PUBLISH=BLOCKED_MISSING_OR_SYMLINK_ASSET"; exit 3; }
done
[[ -f "$NOTES" ]] || { echo "M5_PUBLISH=BLOCKED_RELEASE_NOTES"; exit 3; }
[[ "$(sha256sum "$ASSETS/tervon-bridge-linux-amd64" | awk '{print $1}')" == "$EXPECTED" ]] ||
  { echo "M5_PUBLISH=BLOCKED_WRONG_BINARY_SHA"; exit 3; }
(
  cd "$ASSETS"
  sha256sum -c SHA256SUMS
)
grep -Fqx "BINARY_SHA256=$EXPECTED" "$ASSETS/BUILD_INFO" ||
  { echo "M5_PUBLISH=BLOCKED_BUILD_INFO"; exit 3; }
grep -Fqx "SOURCE_COMMIT=96c136702d10b6fb9a16aa2893f6b978e8b14e86" "$ASSETS/BUILD_INFO" ||
  { echo "M5_PUBLISH=BLOCKED_SOURCE_PROVENANCE"; exit 3; }

# GitHub API authorization is performed by gh with the operator's own
# pre-existing login/token. Never print or request a token in ChatGPT.
gh auth status >/dev/null 2>&1 || { echo "M5_PUBLISH=BLOCKED_GITHUB_AUTH"; exit 3; }
if gh release view "$TAG" --repo "$REPO" >/dev/null 2>&1; then
  echo "M5_PUBLISH=BLOCKED_RELEASE_ALREADY_EXISTS"
  exit 3
fi

# Bind tag to the exact source-and-public-documents branch HEAD.
# Git tag creation via the GitHub REST API, not HTTPS git push.
REMOTE_TAG="$(gh api "repos/$REPO/git/ref/tags/$TAG" --jq .object.sha 2>/dev/null || true)"
if [[ -n "$REMOTE_TAG" ]]; then
  [[ "$REMOTE_TAG" == "$HEAD_SHA" ]] ||
    { echo "M5_PUBLISH=BLOCKED_TAG_POINTS_TO_OTHER_COMMIT"; exit 3; }
else
  gh api --method POST "repos/$REPO/git/refs" \
    -f "ref=refs/tags/$TAG" -f "sha=$HEAD_SHA" >/dev/null
fi
echo "M5_PUBLISH_TAG_SHA=$HEAD_SHA"

# Keep it private as a GitHub draft until remote bytes verify.
gh release create "$TAG" \
  "$ASSETS/tervon-bridge-linux-amd64" \
  "$ASSETS/SHA256SUMS" \
  "$ASSETS/BUILD_INFO" \
  --repo "$REPO" \
  --verify-tag \
  --title "Tervon TCP Bridge v1.0.1 — Linux amd64" \
  --notes-file "$SOURCE/$NOTES" \
  --draft >/dev/null

CHECK="$(mktemp -d "$ASSETS/.verify-release-XXXXXXXX")"
cleanup() { rm -rf -- "$CHECK"; }
trap cleanup EXIT
for file in tervon-bridge-linux-amd64 SHA256SUMS BUILD_INFO; do
  gh release download "$TAG" --repo "$REPO" --dir "$CHECK" --pattern "$file" >/dev/null
  cmp -s "$ASSETS/$file" "$CHECK/$file" ||
    { echo "M5_PUBLISH=BLOCKED_REMOTE_ASSET_MISMATCH_DRAFT_RETAINED"; exit 3; }
done
(
  cd "$CHECK"
  sha256sum -c SHA256SUMS
)
[[ "$(gh release view "$TAG" --repo "$REPO" --json isDraft --jq .isDraft)" == true ]] ||
  { echo "M5_PUBLISH=BLOCKED_DRAFT_STATE"; exit 3; }
echo "M5_PUBLISH_DRAFT_REMOTE_VERIFY=PASS"

# The operator explicitly requested publication of the public binary.
gh release edit "$TAG" --repo "$REPO" --draft=false >/dev/null
[[ "$(gh release view "$TAG" --repo "$REPO" --json isDraft --jq .isDraft)" == false ]] ||
  { echo "M5_PUBLISH=BLOCKED_PUBLIC_STATE_NOT_CONFIRMED"; exit 3; }
echo "M5_PUBLISH_RELEASE_URL=https://github.com/$REPO/releases/tag/$TAG"
echo "M5_PUBLISH_ASSET_SHA256=$EXPECTED"
echo "M5_PUBLISH=PASS_PUBLIC_RELEASE"
