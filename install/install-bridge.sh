#!/usr/bin/env bash
# Tervon TCP Bridge v1.0.1, first installation only.
# sudo bash install-bridge.sh install client|server [--asset FILE --checksums FILE]
# sudo bash install-bridge.sh activate client|server
set -Eeuo pipefail
umask 027
VERSION=v1.0.1
SHA=5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90
ASSET=tervon-bridge-linux-amd64
RELEASE=https://github.com/tervon-software/Tervon-TCP-Bridge/releases/download/v1.0.1
PREFIX=/opt/tervon-tcp-bridge
CONFIG=/etc/tervon-tcp-bridge
ACCOUNT=tervon-bridge
MODE="${1:-}"
ROLE="${2:-}"
if [[ "$MODE" != install && "$MODE" != activate ]] || [[ "$ROLE" != client && "$ROLE" != server ]]; then
  echo "Usage: sudo bash install-bridge.sh install client|server [--asset FILE --checksums FILE]"
  echo "       sudo bash install-bridge.sh activate client|server"
  exit 2
fi
(( EUID == 0 )) || { echo "INSTALL=ROOT_REQUIRED";exit 2; }
[[ "$(uname -s)" == Linux && "$(uname -m)" == x86_64 ]] || { echo "INSTALL=LINUX_AMD64_REQUIRED";exit 2; }
for dep in sha256sum install systemctl mktemp id awk grep runuser useradd; do
  command -v "$dep" >/dev/null || { echo "INSTALL=MISSING_DEPENDENCY:$dep";exit 2; }
done
UNIT="tervon-bridge-$ROLE.service"
BINARY="$PREFIX/bridge"
UNITFILE="/etc/systemd/system/$UNIT"
USER_ASSET=''
USER_SUMS=''
if [[ "$MODE" == install ]]; then
  shift 2
  while (($#)); do
    case "$1" in
      --asset) (($#>=2)) || exit 2; USER_ASSET="$2";shift 2;;
      --checksums) (($#>=2)) || exit 2; USER_SUMS="$2";shift 2;;
      *) echo "INSTALL=INVALID_ARGUMENT";exit 2;;
    esac
  done
  [[ -z "$USER_ASSET" && -z "$USER_SUMS" || -n "$USER_ASSET" && -n "$USER_SUMS" ]] || {
    echo "INSTALL=ASSET_AND_CHECKSUMS_MUST_BE_PAIRED";exit 2;
  }
else
  (($#==2)) || { echo "ACTIVATE=INVALID_ARGUMENT";exit 2; }
fi
hash_file() { sha256sum -- "$1" | awk '{print $1}'; }
check_expected() {
  [[ -f "$1" && ! -L "$1" && "$(hash_file "$1")" == "$SHA" ]]
}
if [[ "$MODE" == activate ]]; then
  [[ -f "$UNITFILE" && -f "$BINARY" && -f "$CONFIG/$ROLE.json" ]] || {
    echo "ACTIVATE=MISSING_INSTALL_OR_CONFIGURATION";exit 3;
  }
  check_expected "$BINARY" || { echo "ACTIVATE=UNEXPECTED_BINARY";exit 3; }
  [[ "$(systemctl is-active "$UNIT" 2>/dev/null || true)" != active ]] || {
    echo "ACTIVATE=ALREADY_RUNNING_REFUSE";exit 3;
  }
  grep -Eq '"mode"[[:space:]]*:[[:space:]]*"'"$ROLE"'"' "$CONFIG/$ROLE.json" || {
    echo "ACTIVATE=WRONG_MODE";exit 3;
  }
  grep -Eq 'REPLACE_|ВСТАВЬТЕ_|bridge\.example\.org' "$CONFIG/$ROLE.json" && {
    echo "ACTIVATE=PLACEHOLDERS_PRESENT";exit 3;
  }
  for key in cert_file key_file ca_file; do
    value="$(python3 - "$CONFIG/$ROLE.json" "$key" <<'PY'
import json,sys
try:
    with open(sys.argv[1],encoding="utf-8") as f:
        cfg=json.load(f)
    value=cfg.get(sys.argv[2],"")
    print(value if isinstance(value,str) else "")
except (ValueError,OSError):
    print("")
PY
)"
    [[ -n "$value" && "$value" = /* && -f "$value" && ! -L "$value" ]] || {
      echo "ACTIVATE=MISSING_PKI:$key";exit 3;
    }
    runuser -u "$ACCOUNT" -- test -r "$value" || {
      echo "ACTIVATE=PKI_UNREADABLE_BY_SERVICE:$key";exit 3;
    }
  done
  echo "ACTIVATE=VALIDATION_PASSED_STARTING_ONLY_$UNIT"
  systemctl daemon-reload
  systemctl enable --now "$UNIT"
  [[ "$(systemctl is-active "$UNIT")" == active ]] || {
    echo "ACTIVATE=SERVICE_NOT_ACTIVE";exit 1;
  }
  echo "ACTIVATE=PASS_UNIT_ACTIVE"
  echo "UNIT=$UNIT"
  exit 0
fi

# Fail closed for existing installation, never replace existing binary/config/unit.
if [[ -e "$BINARY" || -L "$BINARY" || -e "$UNITFILE" || -L "$UNITFILE" ||
      -e "$CONFIG/client.json" || -e "$CONFIG/server.json" ]]; then
  echo "INSTALL=EXISTING_INSTALLATION_REFUSED";exit 3;
fi
if [[ -L "$PREFIX" || -L "$CONFIG" ||
      ( -e "$PREFIX" && ! -d "$PREFIX" ) ||
      ( -e "$CONFIG" && ! -d "$CONFIG" ) ]]; then
  echo "INSTALL=UNEXPECTED_INSTALL_PATH";exit 3;
fi
if [[ -d "$PREFIX" && -n "$(ls -A "$PREFIX")" ]] ||
   [[ -d "$CONFIG" && -n "$(ls -A "$CONFIG")" ]]; then
  echo "INSTALL=NONEMPTY_INSTALL_PATH_REFUSED";exit 3;
fi
if systemctl list-unit-files --no-legend "$UNIT" 2>/dev/null | grep -q "$UNIT"; then
  echo "INSTALL=UNIT_ALREADY_REGISTERED";exit 3;
fi
STAGE="$(mktemp -d /var/tmp/tervon-bridge-install-XXXXXXXX)"
chmod 0700 "$STAGE"
cleanup() { rm -rf -- "$STAGE"; }
trap cleanup EXIT
if [[ -n "$USER_ASSET" ]]; then
  [[ -f "$USER_ASSET" && ! -L "$USER_ASSET" &&
     -f "$USER_SUMS" && ! -L "$USER_SUMS" ]] || {
    echo "INSTALL=LOCAL_FILES_MISSING";exit 3;
  }
  cp -- "$USER_ASSET" "$STAGE/$ASSET"
  cp -- "$USER_SUMS" "$STAGE/SHA256SUMS"
else
  command -v curl >/dev/null || { echo "INSTALL=CURL_REQUIRED";exit 2; }
  curl --proto '=https' --tlsv1.2 --fail --location --show-error --silent \
    --max-time 120 --output "$STAGE/$ASSET" "$RELEASE/$ASSET"
  curl --proto '=https' --tlsv1.2 --fail --location --show-error --silent \
    --max-time 30 --output "$STAGE/SHA256SUMS" "$RELEASE/SHA256SUMS"
fi
check_expected "$STAGE/$ASSET" || { echo "INSTALL=BINARY_SHA256_MISMATCH";exit 3; }
(
  cd "$STAGE"
  grep -Fx "$SHA  $ASSET" SHA256SUMS >/dev/null || {
    echo "INSTALL=RELEASE_MANIFEST_MISMATCH";exit 3;
  }
  sha256sum --check --status SHA256SUMS || { echo "INSTALL=CHECKSUMS_FAILED";exit 3; }
)
if ! id -u "$ACCOUNT" >/dev/null 2>&1; then
  useradd --system --no-create-home --home-dir /nonexistent --shell /usr/sbin/nologin "$ACCOUNT"
fi
install -d -m 0755 "$PREFIX"
install -d -m 0750 -o root -g "$ACCOUNT" "$CONFIG" "$CONFIG/pki"
install -m 0755 "$STAGE/$ASSET" "$BINARY"
check_expected "$BINARY" || { echo "INSTALL=POST_INSTALL_HASH_MISMATCH";exit 1; }
# No keys, CA, or production configuration are ever generated.
cat > "$UNITFILE" <<EOF
[Unit]
Description=Tervon TCP Bridge $ROLE ($VERSION)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$ACCOUNT
Group=$ACCOUNT
ExecStart=$BINARY -config $CONFIG/$ROLE.json
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF
chmod 0644 "$UNITFILE"
echo "INSTALL=PASS_FILES_ONLY"
echo "VERSION=$VERSION"
echo "BINARY_SHA256=$SHA"
echo "SERVICE_NOT_STARTED=YES"
echo "NEXT=PROVISION_$CONFIG/$ROLE.json_AND_PKI_THEN_RUN_ACTIVATE"
