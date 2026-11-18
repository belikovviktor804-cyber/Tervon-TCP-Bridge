#!/usr/bin/env python3
"""Read-only M3 monitoring-chain diagnosis for existing Homeserver services.

Only emits allowlisted health flags. Never prints JSON configs, TLS keys,
URLs, endpoints, request contents, IPs, or connection metadata.
Does not access the public website: that check requires an external observer.
"""
from __future__ import annotations

import argparse
import ipaddress
import json
import subprocess
from pathlib import Path
from urllib.error import URLError
from urllib.request import HTTPRedirectHandler, ProxyHandler, Request, build_opener

BRIDGE_UNIT = "tervon-bridge-client.service"
STATS_UNIT = "tervon-bridge-stats.service"
CONFIG = Path("/etc/tervon-tcp-bridge/client.json")
STATS_LOCAL = "http://127.0.0.1:8894/bridge/api/status"
SCHEMA = "tervon.bridge.telemetry.v1"
DASH_SCHEMA = "tervon.bridge.dashboard.v1"


def unit_state(unit: str) -> str:
    result = subprocess.run(
        ["systemctl", "is-active", unit],
        capture_output=True, text=True, timeout=3, check=False,
    )
    return "ACTIVE" if result.returncode == 0 and result.stdout.strip() == "active" else "NOT_ACTIVE"


def safe_metrics_url(config: dict) -> str | None:
    addr = config.get("metrics_listen", "")
    if not isinstance(addr, str) or not addr:
        return None
    # Address must be a literal loopback IP and an explicit port.
    try:
        host, port_str = addr.rsplit(":", 1)
        host = host.strip("[]")
        if not ipaddress.ip_address(host).is_loopback:
            return None
        port = int(port_str)
        if not (1 <= port <= 65535):
            return None
    except (ValueError, TypeError):
        return None
    if ":" in host:
        return f"http://[{host}]:{port}/metrics/v1"
    return f"http://{host}:{port}/metrics/v1"


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def local_json(url: str, timeout: float = 1.0) -> dict | None:
    # No environment proxies or redirects to unexpected destinations.
    opener = build_opener(ProxyHandler({}), NoRedirect())
    try:
        request = Request(url, headers={"Accept": "application/json"})
        with opener.open(request, timeout=timeout) as response:
            if response.status != 200:
                return None
            body = response.read(65537)
        if len(body) > 65536:
            return None
        value = json.loads(body)
        return value if isinstance(value, dict) else None
    except (OSError, URLError, ValueError, TypeError, json.JSONDecodeError):
        return None


def diagnose(config_path: Path = CONFIG) -> dict:
    result = {
        "schema": "tervon.bridge.monitoring.preflight.v1",
        "bridge_service": unit_state(BRIDGE_UNIT),
        "stats_service": unit_state(STATS_UNIT),
        "bridge_metrics_config": "UNKNOWN",
        "bridge_metrics_local": "NOT_CHECKED",
        "stats_api_local": "UNAVAILABLE",
        "stats_bridge_node": "NOT_CHECKED",
        "public_site_external": "NOT_TESTED",
    }
    try:
        cfg = json.loads(config_path.read_text(encoding="utf-8"))
        if cfg.get("mode") != "client":
            raise ValueError("not client")
    except (OSError, ValueError, TypeError, json.JSONDecodeError):
        result["bridge_metrics_config"] = "UNKNOWN"
    else:
        url = safe_metrics_url(cfg)
        result["bridge_metrics_config"] = "ENABLED_LOOPBACK" if url else "DISABLED_OR_INVALID"
        if url:
            metrics = local_json(url)
            result["bridge_metrics_local"] = (
                "PASS" if metrics and metrics.get("schema") == SCHEMA else "UNAVAILABLE"
            )
    if result["stats_service"] == "ACTIVE":
        dash = local_json(STATS_LOCAL)
        if dash is not None and dash.get("schema") == DASH_SCHEMA:
            result["stats_api_local"] = "PASS"
            node = dash.get("nodes", {}).get("homeserver", {})
            result["stats_bridge_node"] = "OK" if node.get("status") == "ok" else "UNAVAILABLE"
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description="Read-only M3 Homeserver monitoring diagnosis")
    parser.add_argument("--config", type=Path, default=CONFIG)
    args = parser.parse_args()
    print(json.dumps(diagnose(args.config), indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
