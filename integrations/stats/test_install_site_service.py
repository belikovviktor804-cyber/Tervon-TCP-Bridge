"""Static safety contract for the opt-in isolated service installer.

The tests never execute systemctl or modify a host.
"""
import re
import unittest
from pathlib import Path

SCRIPT = (Path(__file__).resolve().parent / "install-site-service.sh").read_text(
    encoding="utf-8"
)


class IsolatedInstallerTests(unittest.TestCase):
    def test_preflight_and_rollback(self):
        for snippet in (
            'set -Eeuo pipefail',
            'preflight_homeserver.py',
            'trap rollback ERR INT TERM',
            'systemd-analyze verify "$UNIT_FILE"',
            'systemctl is-active --quiet tervon-bridge-stats.service',
            'LOOPBACK_STATUS=PASS',
            'METRICS_STATUS=UNAVAILABLE_EXPECTED',
        ):
            with self.subTest(snippet=snippet):
                self.assertIn(snippet, SCRIPT)

    def test_only_new_service_is_started(self):
        self.assertIn("systemctl start tervon-bridge-stats.service", SCRIPT)
        self.assertIn("systemctl enable tervon-bridge-stats.service", SCRIPT)
        for forbidden in (
            "systemctl restart nginx",
            "systemctl reload nginx",
            "systemctl restart tervon-bridge",
            "systemctl restart xray",
            "systemctl restart xray-blocker-dashboard",
            "nginx -s reload",
            "curl | bash",
        ):
            with self.subTest(forbidden=forbidden):
                self.assertNotIn(forbidden, SCRIPT)

    def test_only_new_files_are_installed(self):
        self.assertIn("APP_DIR=/opt/tervon/bridge-stats", SCRIPT)
        self.assertIn("UNIT_FILE=/etc/systemd/system/tervon-bridge-stats.service", SCRIPT)
        self.assertIn("APP_FILES=(bridge_site.py bridge_metrics.py", SCRIPT)
        self.assertNotRegex(SCRIPT, r"(?m)^\s*(?:rm|mv|cp|install)\s+[^\n]*(?:/etc/nginx|/opt/tervon-tcp-bridge|/opt/xray-blocker)")

    def test_no_remote_metrics(self):
        self.assertIn('set(nodes) == {"homeserver"}', SCRIPT)
        self.assertIn('status") == "unavailable"', SCRIPT)
        self.assertNotIn("finserver", SCRIPT.lower())


if __name__ == "__main__":
    unittest.main()
