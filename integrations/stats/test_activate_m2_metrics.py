"""Offline tests for M2 metrics activation safety; never touches systemd."""
import importlib.util
import os
from pathlib import Path
import stat
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("m2_metrics", ROOT / "activate-m2-metrics.py")
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)


class MetricsActivationTests(unittest.TestCase):
    def test_only_correct_local_metrics_snapshot_is_ready(self):
        good = {"schema": "tervon.bridge.telemetry.v1", "ready": True,
                "sessions_active": 0, "padding_state": "OFF"}
        self.assertTrue(m.ready_metrics(good))
        for altered in (
            {**good, "schema": "unknown"},
            {**good, "ready": False},
            {**good, "sessions_active": -1},
            {**good, "sessions_active": True},
            {**good, "sessions_active": "0"},
        ):
            with self.subTest(altered=altered):
                self.assertFalse(m.ready_metrics(altered))

    def test_only_homeserver_dashboard_status_is_accepted(self):
        good = {"schema": "tervon.bridge.dashboard.v1",
                "nodes": {"homeserver": {"status": "ok",
                             "metrics": {"ready": True, "sessions_active": 0}}}}
        self.assertTrue(m.ready_dashboard(good))
        for altered in (
            {"nodes": good["nodes"]},
            {**good, "nodes": {"finserver": good["nodes"]["homeserver"]}},
            {**good, "nodes": {"homeserver": {"status": "unavailable", "metrics": None}}},
            {**good, "nodes": {"homeserver": good["nodes"]["homeserver"],
                               "finserver": good["nodes"]["homeserver"]}},
        ):
            with self.subTest(altered=altered):
                self.assertFalse(m.ready_dashboard(altered))

    def test_retry_on_connection_refused_until_http_works(self):
        calls = iter([OSError("refused"), OSError("refused"),
                      {"schema": "tervon.bridge.telemetry.v1",
                       "ready": True, "sessions_active": 1}])
        def next_call(_):
            value = next(calls)
            if isinstance(value, Exception):
                raise value
            return value
        with (patch.object(m, "active", return_value=True),
              patch.object(m, "fetch", side_effect=next_call),
              patch.object(m.time, "sleep")):
            got = m.await_http(m.METRICS_URL, m.BRIDGE_UNIT, m.ready_metrics, 20)
        self.assertEqual(got["sessions_active"], 1)

    def test_fail_fast_on_stopped_service(self):
        with (patch.object(m, "active", return_value=False),
              patch.object(m, "fetch") as fetch):
            with self.assertRaisesRegex(RuntimeError, "not active"):
                m.await_http(m.METRICS_URL, m.BRIDGE_UNIT, m.ready_metrics, 0.01)
        fetch.assert_not_called()

    def test_atomic_write_keeps_live_permissions_even_if_backup_private(self):
        with tempfile.TemporaryDirectory() as d:
            path = Path(d) / "client.json"
            path.write_bytes(b"old")
            os.chmod(path, 0o640)
            backup = Path(d) / "backup"
            backup.write_bytes(b"old")
            os.chmod(backup, 0o600)
            m.atomic_write(path, b"new", path)
            self.assertEqual(path.read_bytes(), b"new")
            self.assertEqual(stat.S_IMODE(path.stat().st_mode), 0o640)
            m.atomic_write(path, backup.read_bytes(), path)
            self.assertEqual(path.read_bytes(), b"old")


if __name__ == "__main__":
    unittest.main()
