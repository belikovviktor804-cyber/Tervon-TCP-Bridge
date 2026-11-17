#!/usr/bin/env python3
"""Bounded, loopback-only readiness gate for the isolated Homeserver dashboard.

No network access except 127.0.0.1:8894; no credentials or metrics URLs.
"""
from __future__ import annotations

import json
import subprocess
import sys
import time
from urllib.error import URLError
from urllib.request import ProxyHandler, build_opener

URL = "http://127.0.0.1:8894/bridge/api/status"
SERVICE = "tervon-bridge-stats.service"
DEADLINE_SECONDS = 12.0
INTERVAL_SECONDS = 0.25


def check_payload(payload: object) -> None:
    if not isinstance(payload, dict) or payload.get("schema") != "tervon.bridge.dashboard.v1":
        raise ValueError("invalid dashboard schema")
    nodes = payload.get("nodes")
    if not isinstance(nodes, dict) or set(nodes) != {"homeserver"}:
        raise ValueError("unexpected node set")
    node = nodes["homeserver"]
    if not isinstance(node, dict) or node.get("status") != "unavailable":
        raise ValueError("expected disabled metrics status")
    if node.get("metrics") is not None:
        raise ValueError("unexpected metrics before activation")


def wait_ready(
    *,
    timeout: float = DEADLINE_SECONDS,
    interval: float = INTERVAL_SECONDS,
    monotonic=time.monotonic,
    sleep=time.sleep,
    service_active=None,
    opener=None,
) -> bool:
    if service_active is None:
        def service_active() -> bool:
            result = subprocess.run(
                ["systemctl", "is-active", "--quiet", SERVICE],
                stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                timeout=2, check=False,
            )
            return result.returncode == 0
    if opener is None:
        opener = build_opener(ProxyHandler({}))
    deadline = monotonic() + timeout
    while True:
        if not service_active():
            print("LOOPBACK_STATUS=BLOCKED (service not active)")
            return False
        remaining = deadline - monotonic()
        if remaining <= 0:
            print("LOOPBACK_STATUS=BLOCKED (readiness deadline exceeded)")
            return False
        try:
            with opener.open(URL, timeout=min(2.0, remaining)) as response:
                if response.status != 200:
                    raise ValueError("unexpected HTTP status")
                data = response.read(65537)
            if len(data) > 65536:
                raise ValueError("oversized dashboard response")
            check_payload(json.loads(data))
            print("LOOPBACK_STATUS=PASS")
            print("METRICS_STATUS=UNAVAILABLE_EXPECTED")
            return True
        except URLError as exc:
            # Only retry connection-refused while Uvicorn is binding.
            # Other transport failures are not assumed to be startup races.
            if not isinstance(exc.reason, ConnectionRefusedError):
                print("LOOPBACK_STATUS=BLOCKED (unexpected transport error)")
                return False
        except (OSError, ValueError, json.JSONDecodeError):
            print("LOOPBACK_STATUS=BLOCKED (invalid dashboard response)")
            return False
        remaining = deadline - monotonic()
        if remaining <= 0:
            print("LOOPBACK_STATUS=BLOCKED (readiness deadline exceeded)")
            return False
        sleep(min(interval, remaining))


if __name__ == "__main__":
    sys.exit(0 if wait_ready() else 1)
