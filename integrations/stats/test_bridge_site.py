"""ASGI route tests without httpx or a running production service."""
import asyncio
import importlib.util
import json
import sys
import unittest
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parent
sys.path.insert(0, str(ROOT))
spec = importlib.util.spec_from_file_location("bridge_site", ROOT / "bridge_site.py")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


async def request(app, path, method="GET"):
    messages = []
    scope = {
        "type": "http", "asgi": {"version": "3.0"},
        "http_version": "1.1", "method": method,
        "scheme": "http", "path": path,
        "raw_path": path.encode(), "query_string": b"",
        "root_path": "", "headers": [],
        "client": ("127.0.0.1", 10001),
        "server": ("127.0.0.1", 8894),
    }

    async def receive():
        return {"type": "http.request", "body": b"", "more_body": False}

    async def send(message):
        messages.append(message)

    await app(scope, receive, send)
    start = next(m for m in messages if m["type"] == "http.response.start")
    headers = {k.decode().lower(): v.decode() for k, v in start["headers"]}
    body = b"".join(m.get("body", b"") for m in messages
                    if m["type"] == "http.response.body")
    return start["status"], headers, body


class SiteTests(unittest.TestCase):
    def test_unconfigured_status_is_unavailable_not_zero(self):
        with patch.dict("os.environ", {}, clear=True):
            app = module.create_app()
        status, headers, body = asyncio.run(request(app, "/bridge/api/status"))
        self.assertEqual(status, 200)
        self.assertEqual(headers["cache-control"], "no-store")
        self.assertEqual(headers["x-content-type-options"], "nosniff")
        payload = json.loads(body)
        self.assertEqual(set(payload["nodes"]), {"homeserver"})
        self.assertEqual(payload["nodes"]["homeserver"], {
            "status": "unavailable", "metrics": None
        })

    def test_page_is_static_and_has_restrictive_csp(self):
        app = module.create_app("")
        status, headers, body = asyncio.run(request(app, "/bridge/"))
        self.assertEqual(status, 200)
        self.assertIn("frame-ancestors 'none'", headers["content-security-policy"])
        self.assertIn("default-src 'none'", headers["content-security-policy"])
        self.assertEqual(headers["x-frame-options"], "DENY")
        self.assertIn(b"/bridge/bridge_panel.js", body)
        self.assertIn(b"/bridge/bridge_panel.css", body)
        self.assertIn(b"Finserver", body)

    def test_reuses_existing_monitoring_site_theme_and_navigation(self):
        app = module.create_app("")
        status, headers, body = asyncio.run(request(app, "/bridge/"))
        self.assertEqual(status, 200)
        for required in (
            b'/static/style.css',
            b'/static/monitor-theme.css?v=20261020-1',
            b'class="site-nav"',
            b'href="/ai-prosecutor"',
            b'href="/infrastructure/"',
            b'href="/bridge/" class="active"',
            b'class="stats-grid"',
            b'class="stat-card"',
        ):
            with self.subTest(required=required):
                self.assertIn(required, body)
        self.assertNotIn(b'<style>', body)
        self.assertNotIn(b'https://fonts.googleapis.com', body)

    def test_static_assets_are_only_local(self):
        app = module.create_app("")
        for path, content_type in (
            ("/bridge/bridge_panel.js", "text/javascript"),
            ("/bridge/bridge_panel.css", "text/css"),
        ):
            with self.subTest(path=path):
                status, headers, body = asyncio.run(request(app, path))
                self.assertEqual(status, 200)
                self.assertIn(content_type, headers["content-type"])
                self.assertTrue(body)
                self.assertEqual(headers["cache-control"], "no-store")

    def test_no_existing_infrastructure_route_or_api_docs(self):
        app = module.create_app("")
        for path in ("/infrastructure/", "/infrastructure/api/infrastructure",
                     "/openapi.json", "/docs", "/bridge/api/finserver"):
            with self.subTest(path=path):
                status, _, _ = asyncio.run(request(app, path))
                self.assertEqual(status, 404)


    def test_finserver_loopback_and_independent_status(self):
        class FakeCollector:
            def __init__(self, endpoint):
                self.endpoint = endpoint
            def collect(self):
                state = {"status": "unavailable", "metrics": None} if "8897" in self.endpoint else {"status": "ok", "metrics": {"ready": True}}
                return {"schema": "tervon.bridge.dashboard.v1", "nodes": {"homeserver": state}}

        with patch.dict("os.environ", {"TERVON_BRIDGE_FIN_METRICS_URL": "http://127.0.0.1:8897/metrics/v1"}, clear=True):
            with patch.object(module, "BridgeMetricsCollector", FakeCollector):
                app = module.create_app("http://127.0.0.1:8895/metrics/v1")
                status, _, body = asyncio.run(request(app, "/bridge/api/status"))
        self.assertEqual(status, 200)
        nodes = json.loads(body)["nodes"]
        self.assertEqual(nodes["homeserver"]["status"], "ok")
        self.assertEqual(nodes["finserver"]["status"], "unavailable")

    def test_remote_finserver_url_rejected(self):
        with patch.dict("os.environ", {"TERVON_BRIDGE_FIN_METRICS_URL": "http://2.26.228.15:8896/metrics/v1"}, clear=True):
            with self.assertRaises(ValueError):
                module.create_app("")

    def test_bad_endpoint_fails_at_startup(self):
        with self.assertRaises(ValueError):
            module.create_app("http://example.com:9000/metrics/v1")

    def test_api_uses_only_homeserver_collector(self):
        class FakeCollector:
            def collect(self):
                return {
                    "schema": "tervon.bridge.dashboard.v1",
                    "nodes": {"homeserver": {
                        "status": "ok", "metrics": {"ready": True},
                    }},
                }

        with patch.object(module, "BridgeMetricsCollector", return_value=FakeCollector()) as mock:
            app = module.create_app("http://127.0.0.1:9876/metrics/v1")
            mock.assert_called_once_with("http://127.0.0.1:9876/metrics/v1")
        status, _, body = asyncio.run(request(app, "/bridge/api/status"))
        self.assertEqual(status, 200)
        self.assertEqual(json.loads(body)["nodes"]["homeserver"]["status"], "ok")


if __name__ == "__main__":
    unittest.main()
