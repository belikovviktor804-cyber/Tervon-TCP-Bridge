"""Read-only, fail-closed adapter for private Tervon TCP Bridge JSON metrics.

This module does not register FastAPI routes or modify the running dashboard.
Each endpoint must be an explicit literal loopback address. Remote nodes
require a separately reviewed, authenticated local transport/relay.
"""

from __future__ import annotations

import ipaddress
import json
import math
from datetime import datetime, timezone
from urllib.error import HTTPError, URLError
from urllib.parse import urlsplit
from urllib.request import HTTPRedirectHandler, ProxyHandler, Request, build_opener

SCHEMA = "tervon.bridge.telemetry.v1"
MAX_RESPONSE_BYTES = 64 * 1024
ERROR_REASONS = (
    "backend_dial", "http_status", "timeout", "cancellation",
    "shutdown", "authorization", "tls_handshake",
)
STATES = frozenset({"NOT_IMPLEMENTED", "OFF", "ACTIVE", "ERROR"})


class _NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def _validated_endpoint(url: str) -> str:
    if not isinstance(url, str):
        raise ValueError("metrics URL must be a string")
    parsed = urlsplit(url)
    if (parsed.scheme != "http" or parsed.username is not None
            or parsed.password is not None or parsed.query or parsed.fragment
            or parsed.path != "/metrics/v1"):
        raise ValueError("metrics URL must be http loopback /metrics/v1")
    try:
        address = ipaddress.ip_address(parsed.hostname or "")
        port = parsed.port
    except ValueError as exc:
        raise ValueError("metrics URL requires a literal loopback IP and port") from exc
    if not address.is_loopback or port is None or port < 1:
        raise ValueError("metrics URL requires a literal loopback IP and port")
    return url


def _nonnegative_number(value: object) -> int | float:
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        raise ValueError("invalid numeric metric")
    if not math.isfinite(value) or value < 0:
        raise ValueError("invalid numeric metric")
    return value


def _snapshot(raw: object) -> dict:
    if not isinstance(raw, dict) or raw.get("schema") != SCHEMA:
        raise ValueError("unsupported telemetry schema")
    if type(raw.get("ready")) is not bool:
        raise ValueError("invalid readiness")
    values = {}
    for key in (
        "max_sessions", "sessions_started", "sessions_completed",
        "sessions_rejected", "upload_bytes", "download_bytes",
        "backend_dial_success", "backend_dial_failure",
        "connect_latency_p50_ms", "connect_latency_p95_ms",
        "connect_latency_p99_ms", "connect_latency_samples",
        "tls13_h2_mtls_requests", "uptime_seconds", "goroutines",
        "memory_alloc_bytes",
    ):
        values[key] = _nonnegative_number(raw.get(key))
    active = raw.get("sessions_active")
    if type(active) is not int or active < 0:
        raise ValueError("invalid active sessions")
    values["sessions_active"] = active
    errors = raw.get("errors")
    if not isinstance(errors, dict):
        raise ValueError("invalid error counters")
    values["errors"] = {
        key: _nonnegative_number(errors.get(key, 0)) for key in ERROR_REASONS
    }
    for key in ("padding_state", "jitter_state"):
        state = raw.get(key)
        if state not in STATES:
            raise ValueError("invalid feature state")
        values[key] = state
    values["ready"] = raw["ready"]
    return values


class BridgeMetricsCollector:
    """Fixed-node, loopback-only metrics reader; failures never block data plane."""

    def __init__(self, endpoints: dict[str, str], timeout: float = 1.0):
        if not isinstance(endpoints, dict) or not endpoints:
            raise ValueError("at least one endpoint required")
        if not (0 < timeout <= 2):
            raise ValueError("timeout must be in (0, 2] seconds")
        if any(name not in ("homeserver", "finserver") for name in endpoints):
            raise ValueError("unsupported node name")
        self.endpoints = {
            name: _validated_endpoint(url) for name, url in endpoints.items()
        }
        self.timeout = timeout
        self.opener = build_opener(ProxyHandler({}), _NoRedirect())

    def collect(self) -> dict:
        nodes = {}
        for name, endpoint in self.endpoints.items():
            try:
                req = Request(endpoint, method="GET", headers={
                    "Accept": "application/json",
                })
                with self.opener.open(req, timeout=self.timeout) as response:
                    if response.status != 200:
                        raise ValueError("unexpected HTTP status")
                    payload = response.read(MAX_RESPONSE_BYTES + 1)
                if len(payload) > MAX_RESPONSE_BYTES:
                    raise ValueError("metrics payload too large")
                raw = json.loads(payload)
                nodes[name] = {"status": "ok", "metrics": _snapshot(raw)}
            except (HTTPError, URLError, OSError, TimeoutError, ValueError,
                    TypeError, json.JSONDecodeError):
                # Never surface endpoint details or response bodies to website.
                nodes[name] = {"status": "unavailable", "metrics": None}
        return {
            "generated_at": datetime.now(timezone.utc).isoformat(),
            "schema": "tervon.bridge.dashboard.v1",
            "nodes": nodes,
        }
