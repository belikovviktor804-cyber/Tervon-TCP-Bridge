"""Offline readiness tests: no sockets, systemd or host changes."""
import importlib.util
import io
import json
import unittest
from contextlib import redirect_stdout
from pathlib import Path
from urllib.error import URLError

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("wait_for_site", HERE / "wait_for_site.py")
gate = importlib.util.module_from_spec(spec)
spec.loader.exec_module(gate)


class FakeClock:
    def __init__(self):
        self.value = 0.0

    def now(self):
        return self.value

    def sleep(self, seconds):
        self.value += seconds


class FakeResponse:
    status = 200

    def __init__(self, payload):
        self.payload = payload

    def __enter__(self):
        return self

    def __exit__(self, *args):
        return False

    def read(self, limit):
        return self.payload


class FakeOpener:
    def __init__(self, failures=0, payload=None, error=None):
        self.failures = failures
        self.calls = 0
        self.error = error
        self.payload = payload or json.dumps({
            "schema": "tervon.bridge.dashboard.v1",
            "nodes": {"homeserver": {"status": "unavailable", "metrics": None}},
        }).encode()

    def open(self, url, timeout):
        assert url == "http://127.0.0.1:8894/bridge/api/status"
        assert 0 < timeout <= 2
        self.calls += 1
        if self.error is not None:
            raise self.error
        if self.calls <= self.failures:
            raise URLError(ConnectionRefusedError(111, "connection refused"))
        return FakeResponse(self.payload)


class ReadinessTests(unittest.TestCase):
    def check(self, opener, active=lambda: True, timeout=2.0):
        clock = FakeClock()
        output = io.StringIO()
        with redirect_stdout(output):
            result = gate.wait_ready(
                timeout=timeout, interval=0.25, monotonic=clock.now,
                sleep=clock.sleep, service_active=active, opener=opener,
            )
        return result, output.getvalue(), clock.value

    def test_immediate_success(self):
        opener = FakeOpener()
        ok, out, elapsed = self.check(opener)
        self.assertTrue(ok)
        self.assertIn("LOOPBACK_STATUS=PASS", out)
        self.assertIn("METRICS_STATUS=UNAVAILABLE_EXPECTED", out)
        self.assertEqual(opener.calls, 1)
        self.assertEqual(elapsed, 0)

    def test_retries_connection_refused_then_passes(self):
        opener = FakeOpener(failures=3)
        ok, out, elapsed = self.check(opener)
        self.assertTrue(ok)
        self.assertEqual(opener.calls, 4)
        self.assertEqual(elapsed, 0.75)
        self.assertIn("LOOPBACK_STATUS=PASS", out)

    def test_timeout_is_bounded(self):
        opener = FakeOpener(failures=100)
        ok, out, elapsed = self.check(opener, timeout=1)
        self.assertFalse(ok)
        self.assertEqual(elapsed, 1)
        self.assertLessEqual(opener.calls, 5)
        self.assertIn("deadline exceeded", out)

    def test_inactive_service_fails_immediately(self):
        opener = FakeOpener()
        ok, out, elapsed = self.check(opener, active=lambda: False)
        self.assertFalse(ok)
        self.assertEqual(opener.calls, 0)
        self.assertEqual(elapsed, 0)
        self.assertIn("service not active", out)

    def test_other_transport_error_not_retried(self):
        opener = FakeOpener(error=URLError("permission denied"))
        ok, out, _ = self.check(opener)
        self.assertFalse(ok)
        self.assertEqual(opener.calls, 1)
        self.assertIn("unexpected transport error", out)

    def test_unexpected_metrics_or_nodes_fail_closed(self):
        for data in (
            {"schema": "tervon.bridge.dashboard.v1",
             "nodes": {"finserver": {"status": "unavailable", "metrics": None}}},
            {"schema": "tervon.bridge.dashboard.v1",
             "nodes": {"homeserver": {"status": "ok", "metrics": {}}}},
        ):
            with self.subTest(data=data):
                opener = FakeOpener(payload=json.dumps(data).encode())
                ok, out, _ = self.check(opener)
                self.assertFalse(ok)
                self.assertIn("invalid dashboard response", out)


if __name__ == "__main__":
    unittest.main()
