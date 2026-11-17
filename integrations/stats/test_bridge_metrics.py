import importlib.util
import json
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

path = Path(__file__).with_name("bridge_metrics.py")
spec = importlib.util.spec_from_file_location("bridge_metrics", path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


def snapshot():
    values = {
        "schema": "tervon.bridge.telemetry.v1",
        "ready": True,
        "sessions_active": 2,
        "errors": {"timeout": 1},
        "padding_state": "NOT_IMPLEMENTED",
        "jitter_state": "NOT_IMPLEMENTED",
    }
    for key in (
        "max_sessions", "sessions_started", "sessions_completed",
        "sessions_rejected", "upload_bytes", "download_bytes",
        "backend_dial_success", "backend_dial_failure",
        "connect_latency_p50_ms", "connect_latency_p95_ms",
        "connect_latency_p99_ms", "connect_latency_samples",
        "tls13_h2_mtls_requests", "uptime_seconds", "goroutines",
        "memory_alloc_bytes",
    ):
        values[key] = 5
    return values


class Handler(BaseHTTPRequestHandler):
    payload = snapshot()
    status = 200

    def do_GET(self):
        body = json.dumps(self.payload).encode()
        self.send_response(self.status)
        if self.status == 302:
            self.send_header("Location", "http://example.com/")
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *_):
        pass


class BridgeMetricsTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.endpoint = "http://127.0.0.1:%d/metrics/v1" % cls.server.server_port

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join(timeout=2)

    def tearDown(self):
        Handler.payload = snapshot()
        Handler.status = 200

    def test_homeserver_only_and_sanitized(self):
        collector = module.BridgeMetricsCollector(self.endpoint)
        result = collector.collect()
        self.assertEqual(result["schema"], "tervon.bridge.dashboard.v1")
        self.assertEqual(set(result["nodes"]), {"homeserver"})
        node = result["nodes"]["homeserver"]
        self.assertEqual(node["status"], "ok")
        self.assertEqual(node["metrics"]["sessions_active"], 2)
        self.assertEqual(node["metrics"]["errors"]["timeout"], 1)
        self.assertEqual(node["metrics"]["padding_state"], "NOT_IMPLEMENTED")
        self.assertNotIn("schema", node["metrics"])
        self.assertNotIn("client_ip", node["metrics"])

    def test_unavailable_does_not_raise(self):
        Handler.status = 503
        node = module.BridgeMetricsCollector(self.endpoint).collect()
        self.assertEqual(node["nodes"]["homeserver"], {
            "status": "unavailable", "metrics": None
        })

    def test_invalid_payload_fails_closed(self):
        Handler.payload = {"schema": "wrong", "secret": "not-for-dashboard"}
        node = module.BridgeMetricsCollector(self.endpoint).collect()
        self.assertEqual(node["nodes"]["homeserver"]["status"], "unavailable")
        self.assertNotIn("secret", json.dumps(node))

    def test_redirect_rejected(self):
        Handler.status = 302
        node = module.BridgeMetricsCollector(self.endpoint).collect()
        self.assertEqual(node["nodes"]["homeserver"]["status"], "unavailable")

    def test_oversized_payload_rejected(self):
        Handler.payload = {"padding": "x" * (module.MAX_RESPONSE_BYTES + 1)}
        node = module.BridgeMetricsCollector(self.endpoint).collect()
        self.assertEqual(node["nodes"]["homeserver"]["status"], "unavailable")

    def test_non_loopback_rejected(self):
        for url in (
            "http://example.com:1234/metrics/v1",
            "http://0.0.0.0:1234/metrics/v1",
            "http://192.0.2.1:1234/metrics/v1",
            "https://127.0.0.1:1234/metrics/v1",
            "http://127.0.0.1:1234/metrics/v1?token=x",
            "http://127.0.0.1:1234/metrics",
            "http://127.0.0.1:1234@evil.example/metrics/v1",
        ):
            with self.subTest(url=url), self.assertRaises(ValueError):
                module.BridgeMetricsCollector(url)

    def test_remote_node_configuration_rejected(self):
        with self.assertRaises(ValueError):
            module.BridgeMetricsCollector({"finserver": self.endpoint})


if __name__ == "__main__":
    unittest.main()
