"""No-service unit tests for the M1 live config cutover helper."""
import importlib.util
import json
from pathlib import Path
import stat
import tempfile
import unittest

SOURCE = Path(__file__).with_name("activate-m1-framing.py")
spec = importlib.util.spec_from_file_location("m1_cutover", SOURCE)
cutover = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cutover)


class CutoverTests(unittest.TestCase):
    def test_change_only_url_path(self):
        data = {
            "mode": "client",
            "remote_url": "https://example.invalid:8443/v1/stream",
            "max_local_connections": 12,
            "service": "opaque",
        }
        changed = cutover.change_remote_url(data, "/v1/stream", "/v2/stream")
        self.assertEqual(changed["remote_url"], "https://example.invalid:8443/v2/stream")
        self.assertEqual(changed["service"], data["service"])
        self.assertEqual(data["remote_url"], "https://example.invalid:8443/v1/stream")

    def test_refuses_unexpected_config_and_endpoint(self):
        for data in (
            {"mode": "server", "remote_url": "https://example.invalid/v1/stream"},
            {"mode": "client", "remote_url": "http://example.invalid/v1/stream"},
            {"mode": "client", "remote_url": "https://example.invalid/v2/stream"},
            {"mode": "client", "remote_url": "https://example.invalid/v1/stream?token=secret"},
            {"mode": "client", "remote_url": "https://example.invalid/v1/stream#fragment"},
            {"mode": "client", "remote_url": "https://user:pass@example.invalid/v1/stream"},
            {"mode": "client"},
        ):
            with self.subTest(data=data):
                with self.assertRaises(ValueError):
                    cutover.change_remote_url(data, "/v1/stream", "/v2/stream")

    def test_atomic_write_preserves_file_mode(self):
        with tempfile.TemporaryDirectory() as folder:
            config = Path(folder) / "client.json"
            config.write_text('{"mode":"client"}')
            config.chmod(0o640)
            updated = b'{"mode":"client","remote_url":"https://example.invalid/v2/stream"}'
            cutover.atomic_write(config, updated, config)
            self.assertEqual(config.read_bytes(), updated)
            self.assertEqual(stat.S_IMODE(config.stat().st_mode), 0o640)
            self.assertEqual(list(Path(folder).iterdir()), [config])

    def test_atomic_write_refuses_symlink(self):
        with tempfile.TemporaryDirectory() as folder:
            config = Path(folder) / "client.json"
            config.write_text("{}")
            link = Path(folder) / "link.json"
            link.symlink_to(config)
            with self.assertRaises(ValueError):
                cutover.atomic_write(link, b'{"changed":true}', config)
            self.assertEqual(config.read_text(), "{}")


if __name__ == "__main__":
    unittest.main()
