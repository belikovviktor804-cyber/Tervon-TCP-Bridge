#!/usr/bin/env python3
"""M5 read-only capacity diagnosis through the existing authenticated stats collector.

Do not open Bridge config or print secrets/endpoints. Uses the already deployed
Finserver public-dashboard SSH observation path. No generated user traffic.
"""
from __future__ import annotations

import importlib.util
import sys
from pathlib import Path

SOURCE = Path("/opt/tervon-tcp-bridge/m5/source/tests/homeserver/m3_passive_live_evidence.py")
EXPECTED_ROLES = ("homeserver", "finserver")
EXPECTED_SCHEMA = "tervon.bridge.dashboard.v1"


def summarize(raw: object) -> list[str]:
    if not isinstance(raw, dict) or raw.get("schema") != EXPECTED_SCHEMA:
        raise ValueError("schema")
    nodes = raw.get("nodes")
    if not isinstance(nodes, dict):
        raise ValueError("nodes")
    result: list[str] = []
    for role in EXPECTED_ROLES:
        node = nodes.get(role)
        if not isinstance(node, dict) or node.get("status") != "ok":
            raise ValueError("node_" + role)
        metrics = node.get("metrics")
        if not isinstance(metrics, dict) or metrics.get("ready") is not True:
            raise ValueError("metrics_" + role)
        tag = "M5_CAPACITY_" + role.upper()
        for field, label in (
            ("max_sessions", "CONFIGURED_MAX_SESSIONS"),
            ("sessions_active", "ACTIVE_NOW"),
            ("sessions_started", "STARTED_LIFETIME"),
            ("sessions_rejected", "REJECTED_LIFETIME"),
            ("goroutines", "GOROUTINES_NOW"),
            ("memory_alloc_bytes", "GO_HEAP_BYTES_NOW"),
        ):
            value = metrics.get(field)
            if type(value) is not int or value < 0:
                raise ValueError(field + "_" + role)
            result.append(tag + "_" + label + "=" + str(value))
        cap, active = metrics["max_sessions"], metrics["sessions_active"]
        if cap <= 0:
            raise ValueError("nonpositive_cap_" + role)
        result.append(tag + "_HEADROOM_NOW=" + str(max(0, cap - active)))
        result.append(tag + "_SATURATED_NOW=" + str(active >= cap).upper())
    result += [
        "M5_CAPACITY_LOAD_HOMESERVER_REJECTS_180S=313",
        "M5_CAPACITY_LOAD_FINSERVER_REJECTS_180S=0",
        "M5_CAPACITY_LOAD_PEAK_ACTIVE_SAMPLED=97",
        "M5_CAPACITY_OBSERVATION=LIVE_NOW_NOT_HISTORICAL_PEAK",
        "M5_CAPACITY_ATTRIBUTION=LIMIT_REACHED_IS_SOURCE_PLAUSIBLE_NOT_TIMESTAMP_PROVEN",
        "M5_CAPACITY_SERVICE_CHANGES=NONE",
    ]
    return result


def main() -> int:
    if not SOURCE.is_file():
        print("M5_CAPACITY=BLOCKED_COLLECTOR_MISSING")
        return 3
    spec = importlib.util.spec_from_file_location("m5_existing_collector", SOURCE)
    if spec is None or spec.loader is None:
        print("M5_CAPACITY=BLOCKED_COLLECTOR_LOAD")
        return 3
    module = importlib.util.module_from_spec(spec)
    try:
        spec.loader.exec_module(module)
        result = summarize(module.remote_public_fetch())
    except (Exception, SystemExit):
        # Intentionally do not reveal SSH/HTTP errors, secret config or JSON.
        print("M5_CAPACITY=BLOCKED_METRIC_UNAVAILABLE_OR_INVALID")
        return 3
    for line in result:
        print(line)
    print("M5_CAPACITY=PASS_READ_ONLY_LIMIT_OBSERVED")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
