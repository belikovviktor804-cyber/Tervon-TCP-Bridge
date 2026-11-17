"""Isolated read-only Homeserver Bridge dashboard ASGI app.

Run on a *separate* loopback listener. The operator explicitly chose
public read-only /bridge/ dashboard access (no Basic Auth). Only bounded,
non-sensitive Homeserver aggregates may be returned; never expose the
private /metrics/v1 endpoint. Do not mount in infrastructure monitor.
"""

from __future__ import annotations

import os
from pathlib import Path

from fastapi import FastAPI
from fastapi.responses import HTMLResponse, JSONResponse, Response

from bridge_metrics import BridgeMetricsCollector

ROOT = Path(__file__).resolve().parent
HEADERS = {
    "Cache-Control": "no-store",
    "X-Content-Type-Options": "nosniff",
    "X-Frame-Options": "DENY",
    "Referrer-Policy": "no-referrer",
    "Content-Security-Policy": (
        "default-src 'none'; script-src 'self'; style-src 'self'; "
        "connect-src 'self'; base-uri 'none'; form-action 'none'; "
        "frame-ancestors 'none'"
    ),
}
CSS_HEADERS = {"Cache-Control": "no-store", "X-Content-Type-Options": "nosniff"}


def create_app(metrics_endpoint: str | None = None) -> FastAPI:
    # Explicit endpoint required; never guess a port or connect to remote nodes.
    if metrics_endpoint is None:
        metrics_endpoint = os.environ.get("TERVON_BRIDGE_METRICS_URL", "")
    collector = (
        BridgeMetricsCollector(metrics_endpoint)
        if metrics_endpoint else None
    )

    app = FastAPI(
        title="Tervon Bridge Homeserver metrics",
        docs_url=None, redoc_url=None, openapi_url=None,
    )

    @app.get("/bridge/", response_class=HTMLResponse, include_in_schema=False)
    def index():
        fragment = (ROOT / "bridge_panel.html").read_text(encoding="utf-8")
        page = (
            '<!doctype html><html lang="ru"><head>'
            '<meta charset="utf-8">'
            '<meta name="viewport" content="width=device-width,initial-scale=1">'
            '<title>Tervon TCP Bridge — Homeserver</title>'
            '<link rel="stylesheet" href="/bridge/bridge_panel.css">'
            '</head><body>' + fragment +
            '<script defer src="/bridge/bridge_panel.js"></script>'
            '</body></html>'
        )
        return HTMLResponse(page, headers=HEADERS)

    @app.get("/bridge/api/status", include_in_schema=False)
    def status():
        result = (
            collector.collect() if collector is not None else
            {"schema": "tervon.bridge.dashboard.v1",
             "nodes": {"homeserver": {
                 "status": "unavailable", "metrics": None,
             }}}
        )
        return JSONResponse(result, headers=HEADERS)

    @app.get("/bridge/bridge_panel.js", include_in_schema=False)
    def javascript():
        return Response(
            (ROOT / "bridge_panel.js").read_text(encoding="utf-8"),
            media_type="text/javascript; charset=utf-8", headers=CSS_HEADERS,
        )

    @app.get("/bridge/bridge_panel.css", include_in_schema=False)
    def stylesheet():
        return Response(
            (ROOT / "bridge_panel.css").read_text(encoding="utf-8"),
            media_type="text/css; charset=utf-8", headers=CSS_HEADERS,
        )

    return app


app = create_app()
