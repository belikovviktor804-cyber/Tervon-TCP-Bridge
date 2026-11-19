#!/usr/bin/env python3
"""Read-only short M4 mixed-version telemetry diagnosis, no secrets or raw payload."""
from __future__ import annotations

import json
import sys
import time
from typing import Any

import m3_passive_live_evidence as sampler

ROLES = ("homeserver", "finserver")


def status(raw: Any) -> dict:
    if not isinstance(raw, dict):
        return {"status": "INVALID_RESPONSE", "reason": "NOT_JSON_OBJECT"}
    if raw.get("schema") != sampler.SCHEMA:
        return {"status": "INVALID_RESPONSE", "reason": "SCHEMA_MISMATCH"}
    nodes = raw.get("nodes")
    if not isinstance(nodes, dict):
        return {"status": "INVALID_RESPONSE", "reason": "NODES_MISSING"}
    for role in ROLES:
        node = nodes.get(role)
        if not isinstance(node, dict):
            return {"status": "INVALID_RESPONSE", "reason": "NODE_MISSING", "role": role}
        if node.get("status") != "ok":
            return {"status": "NOT_READY", "reason": "NODE_STATUS_NOT_OK", "role": role}
        metrics = node.get("metrics")
        if not isinstance(metrics, dict):
            return {"status": "INVALID_RESPONSE", "reason": "METRICS_MISSING", "role": role}
        if metrics.get("ready") is not True:
            return {"status": "NOT_READY", "reason": "BRIDGE_READY_NOT_TRUE", "role": role}
        fields = ("sessions_active",) + sampler.COUNTERS
        for field in fields:
            v = metrics.get(field)
            if type(v) is not int or v < 0:
                return {"status": "INVALID_RESPONSE", "reason": "COUNTER_INVALID",
                        "role": role, "field": field}
        errors = metrics.get("errors")
        if not isinstance(errors, dict):
            return {"status": "INVALID_RESPONSE", "reason": "ERRORS_MISSING", "role": role}
        for field in sampler.ERRORS:
            v = errors.get(field)
            if type(v) is not int or v < 0:
                return {"status": "INVALID_RESPONSE", "reason": "ERROR_COUNTER_INVALID",
                        "role": role, "field": field}
    return {"status": "VALID"}


def change(before: Any, after: Any) -> dict:
    for role in ROLES:
        a = before["nodes"][role]["metrics"]
        b = after["nodes"][role]["metrics"]
        for field in sampler.COUNTERS:
            if b[field] < a[field]:
                return {"status": "COUNTER_RESET", "reason": "COUNTER_DECREASED",
                        "role": role, "field": field}
        for field in sampler.ERRORS:
            if b["errors"][field] < a["errors"][field]:
                return {"status": "COUNTER_RESET", "reason": "ERROR_COUNTER_DECREASED",
                        "role": role, "field": field}
    deltas = {}
    for role in ROLES:
        a = before["nodes"][role]["metrics"]
        b = after["nodes"][role]["metrics"]
        deltas[role] = {
            "sessions_active": b["sessions_active"],
            "sessions_started_delta": b["sessions_started"] - a["sessions_started"],
            "upload_bytes_delta": b["upload_bytes"] - a["upload_bytes"],
            "download_bytes_delta": b["download_bytes"] - a["download_bytes"],
            "padding_download_frames_delta": b["padding_download_frames"] - a["padding_download_frames"],
            "jitter_download_frames_delta": b["jitter_download_frames"] - a["jitter_download_frames"],
            "error_delta": sum(b["errors"][k] - a["errors"][k] for k in sampler.ERRORS),
        }
    return {"status": "PASS_DIAGNOSTIC_COUNTER_CONTINUITY", "nodes": deltas,
            "vpn_dataplane_proven": False, "m4_module_pass": False}


def run(fetcher=sampler.remote_public_fetch, sleeper=time.sleep, interval: int = 10) -> dict:
    try:
        first = fetcher()
        checked = status(first)
        if checked["status"] != "VALID":
            return checked
        sleeper(interval)
        second = fetcher()
        checked = status(second)
        if checked["status"] != "VALID":
            return checked
        return change(first, second)
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        # Never print exception text: it can contain URLs/hosts or raw data.
        if isinstance(exc, ValueError) and str(exc) == "public fetch failed":
            reason = "PUBLIC_HTTPS_FETCH_FAILED"
        elif isinstance(exc, ValueError) and str(exc) == "public response too large":
            reason = "PUBLIC_RESPONSE_TOO_LARGE"
        else:
            reason = "FETCH_OR_PARSE_EXCEPTION"
        return {"status": "DIAGNOSTIC_FAIL", "reason": reason}


def main() -> int:
    result = run()
    print(json.dumps({"schema": "tervon.m4.mixed-diagnostic.v1", **result}, sort_keys=True),
          flush=True)
    return 0 if result["status"] == "PASS_DIAGNOSTIC_COUNTER_CONTINUITY" else 1


if __name__ == "__main__":
    sys.exit(main())
