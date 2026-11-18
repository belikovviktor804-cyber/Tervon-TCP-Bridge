"""Read-only, fail-closed adapter for private Tervon TCP Bridge JSON metrics.

This module does not register FastAPI routes or modify the running dashboard.
The only source is the Homeserver Bridge on a literal loopback address.
No remote-node collection or transport is implemented.
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
    # Optional M2/M3 extension fields: older Bridge binaries do not emit
    # them. Preserve absence rather than fabricating zero-valued traffic.
    for key in (
        "padding_upload_bytes", "padding_download_bytes",
        "padding_upload_frames", "padding_download_frames",
        "padding_upload_wire_overhead_bytes",
        "padding_download_wire_overhead_bytes",
        "padding_upload_overhead_percent",
        "padding_download_overhead_percent",
        "padding_upload_wire_overhead_percent",
        "padding_download_wire_overhead_percent",
        "padding_upload_transitions", "padding_download_transitions",
    ):
        if key in raw:
            values[key] = _nonnegative_number(raw[key])
    for key in ("padding_upload_state", "padding_download_state"):
        if key in raw:
            if raw[key] not in STATES:
                raise ValueError("invalid directional feature state")
            values[key] = raw[key]
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
    """Homeserver-only loopback metrics reader; never blocks the data plane."""

    def __init__(self, endpoint: str, timeout: float = 1.0):
        if not (0 < timeout <= 2):
            raise ValueError("timeout must be in (0, 2] seconds")
        self.endpoint = _validated_endpoint(endpoint)
        self.timeout = timeout
        self.opener = build_opener(ProxyHandler({}), _NoRedirect())

    def collect(self) -> dict:
        try:
            req = Request(self.endpoint, method="GET", headers={
                "Accept": "application/json",
            })
            with self.opener.open(req, timeout=self.timeout) as response:
                if response.status != 200:
                    raise ValueError("unexpected HTTP status")
                payload = response.read(MAX_RESPONSE_BYTES + 1)
            if len(payload) > MAX_RESPONSE_BYTES:
                raise ValueError("metrics payload too large")
            raw = json.loads(payload)
            node = {"status": "ok", "metrics": _snapshot(raw)}
        except (HTTPError, URLError, OSError, TimeoutError, ValueError,
                TypeError, json.JSONDecodeError):
            # Do not surface endpoint details or response bodies to the site.
            node = {"status": "unavailable", "metrics": None}
        return {
            "generated_at": datetime.now(timezone.utc).isoformat(),
            "schema": "tervon.bridge.dashboard.v1",
            "nodes": {"homeserver": node},
        }
