"""Isolated M3 collector deployment tests; never contacts systemd or Bridge."""
import importlib.util
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

SOURCE = Path(__file__).with_name("update-m3-collector.py")
spec = importlib.util.spec_from_file_location("update_m3_collector", SOURCE)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class CollectorUpdateTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.app = Path(self.tmp.name)
        (self.app / "bridge_metrics.py").write_bytes(b"legacy collector")
        self.mocked = [
            patch.object(module, "APP", self.app),
            patch.object(module, "TARGET", self.app / "bridge_metrics.py"),
            patch.object(module, "preflight", return_value=(
                {"sessions_active": 1},
                {"padding_upload_bytes": 100, "padding_upload_frames": 2,
                 "padding_upload_state": "OFF",
                 "padding_upload_wire_overhead_percent": 2.0},
            )),
            patch.object(module, "command", return_value="active"),
        ]
        for p in self.mocked:
            p.start()
            self.addCleanup(p.stop)

    def test_replaces_only_collector_and_keeps_backup(self):
        with patch.object(module, "wait_dashboard", return_value={"sessions_active": 1}):
            module.install()
        self.assertEqual((self.app / "bridge_metrics.py").read_bytes(),
                         SOURCE.with_name("bridge_metrics.py").read_bytes())
        backups = list((self.app / ".collector-backups").glob("m3-*"))
        self.assertEqual(len(backups), 1)
        self.assertEqual((backups[0] / "bridge_metrics.py").read_bytes(),
                         b"legacy collector")

    def test_restart_verification_failure_restores_original(self):
        with patch.object(module, "wait_dashboard", side_effect=RuntimeError("not ready")), \
             patch.object(module, "wait_ready", return_value=True):
            with self.assertRaisesRegex(RuntimeError, "not ready"):
                module.install()
        self.assertEqual((self.app / "bridge_metrics.py").read_bytes(),
                         b"legacy collector")

    def test_partial_forwarding_is_rejected(self):
        raw = {"padding_upload_bytes": 100, "padding_upload_frames": 2,
               "padding_upload_state": "OFF",
               "padding_upload_wire_overhead_percent": 2.0}
        with patch.object(module, "dashboard", return_value={
            "ready": True, "padding_upload_bytes": 100,
            "padding_upload_frames": 2, "padding_upload_state": "OFF",
        }):
            with self.assertRaisesRegex(RuntimeError, "timeout"):
                module.wait_dashboard(raw, timeout=0)

    def test_advanced_counters_may_increase_between_scrapes(self):
        raw = {"padding_upload_bytes": 100, "padding_upload_frames": 2,
               "padding_upload_state": "OFF",
               "padding_upload_wire_overhead_percent": 2.0}
        snap = {"ready": True, "padding_upload_bytes": 150,
                "padding_upload_frames": 3, "padding_upload_state": "ACTIVE",
                "padding_upload_wire_overhead_percent": 2.2}
        with patch.object(module, "dashboard", return_value=snap):
            self.assertEqual(module.wait_dashboard(raw, timeout=0), snap)

    def test_preflight_fails_closed_without_root(self):
        real = module.preflight
        with patch.object(module.os, "geteuid", return_value=1000):
            with self.assertRaisesRegex(RuntimeError, "root"):
                real(SOURCE.parents[2], SOURCE.with_name("bridge_metrics.py"))


if __name__ == "__main__":
    unittest.main()
