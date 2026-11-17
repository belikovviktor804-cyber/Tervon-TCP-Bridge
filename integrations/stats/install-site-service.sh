#!/usr/bin/env bash
# Stage ONLY the isolated Bridge stats ASGI service on Homeserver.
# Does not edit/reload nginx or restart Bridge, Xray, or existing dashboard.
# Run as: sudo bash integrations/stats/install-site-service.sh
# Never source this script in an interactive shell.
set -Eeuo pipefail
umask 077

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
REPO_DIR="$(cd -- "$SCRIPT_DIR/../.." && pwd -P)"
APP_DIR=/opt/tervon/bridge-stats
UNIT_FILE=/etc/systemd/system/tervon-bridge-stats.service
ENV_FILE=/etc/tervon/bridge-stats.env
VENV_PYTHON=/opt/xray-blocker/web-dashboard/venv/bin/python
UNIT_TEMPLATE="$SCRIPT_DIR/tervon-bridge-stats.service.example"
APP_FILES=(bridge_site.py bridge_metrics.py bridge_panel.html bridge_panel.js bridge_panel.css)
created_app=0
created_unit=0
complete=0

rollback() {
  local rc=$?
  trap - ERR INT TERM
  if (( complete == 0 )); then
    echo "INSTALL_STATUS=ROLLBACK (exit=$rc)"
    if (( created_unit == 1 )); then
      systemctl stop tervon-bridge-stats.service >/dev/null 2>&1 || true
      systemctl disable tervon-bridge-stats.service >/dev/null 2>&1 || true
      rm -f -- "$UNIT_FILE"
      systemctl daemon-reload >/dev/null 2>&1 || true
    fi
    if (( created_app == 1 )); then
      rm -f -- "${APP_FILES[@]/#/$APP_DIR/}"
      rmdir -- "$APP_DIR" 2>/dev/null || true
    fi
  fi
  exit "$rc"
}
trap rollback ERR INT TERM

if (( EUID != 0 )); then
  echo "BLOCKED: root required for new isolated service"
  exit 2
fi

cd -- "$REPO_DIR"
echo "===== GIT SHA ====="
git rev-parse HEAD
echo "===== READ-ONLY PREFLIGHT ====="
python3 "$SCRIPT_DIR/preflight_homeserver.py"
echo "===== EXPLICIT CONFLICT CHECK ====="
if [[ -e "$UNIT_FILE" || -L "$UNIT_FILE" || -e "$APP_DIR" || -L "$APP_DIR" ]]; then
  echo "BLOCKED: service or application path already exists"
  exit 2
fi
if [[ -e "$ENV_FILE" || -L "$ENV_FILE" ]]; then
  echo "BLOCKED: existing stats env file requires manual review"
  exit 2
fi
for file in "${APP_FILES[@]}"; do
  if [[ ! -f "$SCRIPT_DIR/$file" || -L "$SCRIPT_DIR/$file" ]]; then
    echo "BLOCKED: application source missing or symlink"
    exit 2
  fi
done
if [[ ! -f "$UNIT_TEMPLATE" || -L "$UNIT_TEMPLATE" ]]; then
  echo "BLOCKED: unit template missing or symlink"
  exit 2
fi

echo "===== INSTALL NEW ISOLATED APP ====="
install -d -m 0755 -- "$APP_DIR"
created_app=1
for file in "${APP_FILES[@]}"; do
  install -m 0644 -- "$SCRIPT_DIR/$file" "$APP_DIR/$file"
done
install -m 0644 -- "$UNIT_TEMPLATE" "$UNIT_FILE"
created_unit=1

echo "===== VERIFY NEW UNIT ====="
systemd-analyze verify "$UNIT_FILE"
systemctl daemon-reload
systemctl start tervon-bridge-stats.service
systemctl is-active --quiet tervon-bridge-stats.service

echo "===== VERIFY LOOPBACK STATUS (METRICS DISABLED) ====="
"$VENV_PYTHON" - <<'PY'
import json
import urllib.request

url = "http://127.0.0.1:8894/bridge/api/status"
opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
with opener.open(url, timeout=4) as response:
    assert response.status == 200, response.status
    payload = json.load(response)
assert payload.get("schema") == "tervon.bridge.dashboard.v1", payload
nodes = payload.get("nodes", {})
assert set(nodes) == {"homeserver"}, nodes
assert nodes["homeserver"].get("status") == "unavailable", nodes
print("LOOPBACK_STATUS=PASS")
print("METRICS_STATUS=UNAVAILABLE_EXPECTED")
PY

echo "===== VERIFY LISTENER IS LOCAL ONLY ====="
listener="$(ss -H -ltn '( sport = :8894 )')"
if [[ -z "$listener" ]]; then
  echo "BLOCKED: no listener on 8894"
  false
fi
if [[ "$listener" == *"0.0.0.0:8894"* || "$listener" == *"[::]:8894"* ]]; then
  echo "BLOCKED: public listener detected"
  false
fi
echo "LOOPBACK_LISTENER=PASS"
systemctl enable tervon-bridge-stats.service >/dev/null
complete=1
trap - ERR INT TERM
echo "NEW_SERVICE=PASS"
echo "NGINX=UNCHANGED"
echo "BRIDGE_DATA_PLANE=UNCHANGED"
echo "WEBSITE_INTEGRATION=NOT_YET_ENABLED"
