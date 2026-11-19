#!/usr/bin/env python3
"""M5 post-cutover read-only two-node check; no traffic generation or service changes.

Uses the already deployed dashboard observer and cross-verifies counters, source
identity, readiness, and zero session quotas of both endpoints. Does not claim
network RTT or fairness from aggregate metrics.
"""
from __future__ import annotations

import importlib.util
import json
import hashlib
from pathlib import Path
import subprocess
import time

ROOT = Path("/opt/tervon-tcp-bridge/m5/source")
COLLECTOR = ROOT / "tests/homeserver/m3_passive_live_evidence.py"
EXPECTED_SHA = "4f997b60bd777516339512ac71ddbb89b96d5bf9af979286ff3bd63690ea525b"
UNIT = "tervon-bridge-client.service"
ROLES = ("homeserver", "finserver")

def print_value(key: str, value: object) -> None:
    print(f"M5_POST_{key}={value}", flush=True)

def load_collector():
    spec = importlib.util.spec_from_file_location("m5_passive_existing", COLLECTOR)
    if spec is None or spec.loader is None:
        raise RuntimeError("collector_not_found")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module

def inspect(raw: object):
    if not isinstance(raw, dict) or raw.get("schema") != "tervon.bridge.dashboard.v1":
        raise ValueError("dashboard_schema")
    nodes = raw.get("nodes")
    if not isinstance(nodes, dict):
        raise ValueError("dashboard_nodes")
    result = {}
    for role in ROLES:
        node = nodes.get(role)
        if not isinstance(node, dict) or node.get("status") != "ok":
            raise ValueError("node_status")
        m = node.get("metrics")
        if not isinstance(m, dict) or m.get("ready") is not True:
            raise ValueError("node_not_ready")
        for key in ("max_sessions", "sessions_active", "sessions_started",
                    "sessions_rejected", "upload_bytes", "download_bytes", "goroutines",
                    "memory_alloc_bytes"):
            if type(m.get(key)) is not int or m[key] < 0:
                raise ValueError("invalid_metric")
        if m["max_sessions"] != 0:
            raise ValueError("unexpected_fixed_quota")
        result[role] = {key: m[key] for key in (
            "max_sessions", "sessions_active", "sessions_started", "sessions_rejected",
            "upload_bytes", "download_bytes", "goroutines", "memory_alloc_bytes")}
        errors = m.get("errors")
        if not isinstance(errors, dict) or not all(type(v) is int and v >= 0 for v in errors.values()):
            raise ValueError("invalid_error_counters")
        result[role]["errors"] = dict(errors)
    return result

def active_client_pid():
    active = subprocess.run(["systemctl","is-active",UNIT],capture_output=True,text=True,timeout=5)
    if active.stdout.strip() != "active":
        raise RuntimeError("client_unit_inactive")
    pid = subprocess.run(["systemctl","show",UNIT,"-p","MainPID","--value"],capture_output=True,text=True,timeout=5)
    value = pid.stdout.strip()
    if not value.isdecimal() or int(value) == 0:
        raise RuntimeError("client_pid_invalid")
    return int(value)

def process_hash(pid):
    h=hashlib.sha256()
    with open(f"/proc/{pid}/exe","rb") as f:
        for chunk in iter(lambda: f.read(1024*1024), b""):
            h.update(chunk)
    return h.hexdigest()

def compare(a, b):
    any_traffic = False
    for role in ROLES:
        for key in ("sessions_started", "sessions_rejected", "upload_bytes", "download_bytes"):
            delta = b[role][key] - a[role][key]
            if delta < 0:
                raise ValueError("counter_reset")
            print_value(role.upper()+"_"+key.upper()+"_DELTA", delta)
            if key in ("upload_bytes", "download_bytes") and delta:
                any_traffic = True
        old_err, new_err=a[role]["errors"],b[role]["errors"]
        if old_err.keys() != new_err.keys():
            raise ValueError("error_keys_changed")
        errors = sum(new_err[key]-old_err[key] for key in old_err)
        if errors < 0 or any(new_err[k] < old_err[k] for k in old_err):
            raise ValueError("error_counter_reset")
        print_value(role.upper()+"_ERROR_DELTA",errors)
        print_value(role.upper()+"_ACTIVE_END",b[role]["sessions_active"])
        print_value(role.upper()+"_GOROUTINES_END",b[role]["goroutines"])
        print_value(role.upper()+"_GO_HEAP_BYTES_END",b[role]["memory_alloc_bytes"])
    return any_traffic

def main():
    try:
        collector=load_collector()
        pid=active_client_pid()
        if process_hash(pid) != EXPECTED_SHA:
            raise ValueError("unexpected_live_binary_sha")
        before=inspect(collector.remote_public_fetch())
        print_value("START=PASS_BOTH_READY_AND_ZERO_QUOTA", "YES")
        time.sleep(30)
        after=inspect(collector.remote_public_fetch())
        if active_client_pid()!=pid or process_hash(pid)!=EXPECTED_SHA:
            raise RuntimeError("process_changed")
        observed=compare(before, after)
        print_value("END=PASS_BOTH_READY_AND_ZERO_QUOTA", "YES")
        print_value("TCP_TRAFFIC", "OBSERVED_COUNTER_INCREASE" if observed else "NOT_OBSERVED_IN_WINDOW")
        print_value("NETWORK_RTT_FAIRNESS", "NOT_MEASURED")
        print_value("RESULT", "PASS_READ_ONLY_TWO_NODE_CHECK")
        return 0
    except (Exception, SystemExit) as exc:
        # Never output SSH response, source config, credentials or tracebacks.
        print_value("RESULT", "BLOCKED_"+type(exc).__name__)
        return 3

if __name__=="__main__":
    raise SystemExit(main())
