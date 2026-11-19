"""Isolated tests for the read-only M2 service baseline collector.

All fixtures are local files; systemctl is mocked. No systemd service, SSH
connection, running bridge or production config is accessed.
"""
import hashlib
import importlib.util
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

SCRIPT = Path(__file__).with_name("collect_m2_baseline.py")
spec = importlib.util.spec_from_file_location("collect_m2_baseline", SCRIPT)
collector = importlib.util.module_from_spec(spec)
spec.loader.exec_module(collector)

SERVICE = "tervon-bridge-client.service"
OUTPUT = "MainPID=123\nActiveState=active\nSubState=running\nNRestarts=0\n"


class M2ReadOnlyBaselineTests(unittest.TestCase):
    def make_proc(self, root: Path) -> bytes:
        proc = root / "123"
        proc.mkdir()
        binary = b"this is not the real production binary"
        (proc / "exe").write_bytes(binary)
        (proc / "status").write_text(
            "Name:\tbridge\nVmRSS:\t5120 kB\n", encoding="utf-8"
        )
        return binary

    def test_collect_only_allowlisted_fields_and_no_commands(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            binary = self.make_proc(root)
            with patch.object(
                collector.subprocess, "run",
                return_value=SimpleNamespace(stdout=OUTPUT),
            ) as run:
                data = collector.collect("homeserver", SERVICE, proc_root=root)
            run.assert_called_once()
            args = run.call_args.args[0]
            self.assertEqual(args[0:2], ["systemctl", "show"])
            self.assertNotIn("restart", args)
            self.assertNotIn("stop", args)
            self.assertNotIn("start", args)
            self.assertFalse(run.call_args.kwargs.get("shell", False))
            self.assertEqual(data["schema"], "tervon.bridge.m2-readonly-baseline.v1")
            self.assertEqual(data["running_executable_sha256"], hashlib.sha256(binary).hexdigest())
            self.assertFalse(data["matches_last_operator_confirmed_m1_sha256"])
            self.assertEqual(data["process_rss_kib"], 5120)
            self.assertEqual(data["restart_count"], 0)
            self.assertEqual(data["cpu_percent"], "NOT_MEASURED")
            self.assertEqual(data["route_rtt_ms"], "NOT_MEASURED")
            self.assertEqual(data["padding_live_enabled"], "NOT_MEASURED")
            self.assertNotIn("ExecStart", data)
            self.assertNotIn("Environment", data)
            self.assertNotIn("endpoint", data)

    def test_rejects_injected_unit_name_before_subprocess(self):
        with patch.object(collector.subprocess, "run") as run:
            with self.assertRaises(ValueError):
                collector.collect("finserver", "bridge.service;systemctl stop x", Path("/no-such-proc"))
            with self.assertRaises(ValueError):
                collector.collect("homeserver", "--property=Environment", Path("/no-such-proc"))
            with self.assertRaises(ValueError):
                collector.collect("unknown", SERVICE, Path("/no-such-proc"))
            run.assert_not_called()

    def test_rejects_inactive_or_invalid_pid(self):
        for stdout in (
            "MainPID=0\nActiveState=inactive\nSubState=dead\nNRestarts=0\n",
            "MainPID=bad\nActiveState=active\nSubState=running\nNRestarts=0\n",
            "MainPID=123\nActiveState=failed\nSubState=failed\nNRestarts=0\n",
            "MainPID=123\nActiveState=active\nSubState=running\n",
        ):
            with self.subTest(stdout=stdout):
                with patch.object(
                    collector.subprocess, "run",
                    return_value=SimpleNamespace(stdout=stdout),
                ):
                    with self.assertRaises(ValueError):
                        collector.collect("homeserver", SERVICE, Path("/no-such-proc"))

    def test_parser_ignores_all_sensitive_extra_fields(self):
        parsed = collector.parse_systemd_show(
            OUTPUT + "ExecStart={ path=/private/secret }\nEnvironment=TOKEN=secret\n"
        )
        self.assertEqual(set(parsed), collector.ALLOWED_SYSTEMD_KEYS)
        self.assertNotIn("secret", repr(parsed))

    def test_rss_missing_is_not_fabricated(self):
        self.assertIsNone(collector.rss_kib("Name:\tbridge\n"))
        with self.assertRaises(ValueError):
            collector.rss_kib("VmRSS:\t5 MB\n")


if __name__ == "__main__":
    unittest.main()
