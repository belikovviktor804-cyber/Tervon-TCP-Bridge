"""Static safety checks; operational verification on Homeserver is separate."""
import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent
UNIT = (ROOT / "tervon-bridge-stats.service.example").read_text(encoding="utf-8")
NGINX = (ROOT / "nginx-bridge.conf.example").read_text(encoding="utf-8")


class UnitTemplateTests(unittest.TestCase):
    def test_private_bind_and_existing_python_environment(self):
        self.assertRegex(UNIT, r"ExecStart=.*?/uvicorn\s+bridge_site:app\s+--host 127\.0\.0\.1\s+--port 8894")
        self.assertIn("/opt/xray-blocker/web-dashboard/venv/bin/uvicorn", UNIT)
        self.assertNotIn("--host 0.0.0.0", UNIT)
        self.assertNotIn("--reload", UNIT)
        self.assertNotIn("--workers 2", UNIT)

    def test_sandbox_and_network_isolation(self):
        for setting in (
            "DynamicUser=yes", "NoNewPrivileges=yes",
            "CapabilityBoundingSet=", "ProtectSystem=strict",
            "ProtectHome=yes", "PrivateDevices=yes",
            "IPAddressDeny=any", "IPAddressAllow=127.0.0.0/8",
            "IPAddressAllow=::1/128",
        ):
            with self.subTest(setting=setting):
                self.assertIn(setting, UNIT)
        self.assertNotRegex(UNIT, r"(?m)^User=root\s*$")

    def test_no_live_monitor_or_finserver_changes(self):
        self.assertNotIn("finserver", UNIT.lower())
        self.assertNotIn("unified_monitor:app", UNIT)
        self.assertNotIn("8893", UNIT)
        self.assertIn("proxy_pass http://127.0.0.1:8894;", NGINX)
        self.assertEqual(NGINX.count("auth_basic off;"), 2)
        self.assertNotIn("auth_basic_user_file", NGINX)


if __name__ == "__main__":
    unittest.main()
