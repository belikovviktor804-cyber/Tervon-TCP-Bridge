"""Static safety constraints for the M3 dedicated SSH metrics unit."""
from pathlib import Path
import unittest

UNIT = Path(__file__).with_name("tervon-bridge-fin-metrics-tunnel.service").read_text()

class UnitTests(unittest.TestCase):
    def test_loopback_only_forward_and_strict_ssh(self):
        for item in (
            "127.0.0.1:8897:127.0.0.1:8896",
            "BatchMode=yes", "StrictHostKeyChecking=yes",
            "ExitOnForwardFailure=yes", "ClearAllForwardings=yes",
            "ServerAliveInterval=15", "ServerAliveCountMax=3",
            "Restart=always", "ProtectSystem=strict", "ProtectHome=read-only",
            "NoNewPrivileges=yes",
        ):
            self.assertIn(item, UNIT)
        self.assertNotIn("0.0.0.0:8897", UNIT)
        self.assertNotIn("GatewayPorts=yes", UNIT)

    def test_unit_does_not_restart_bridge_or_nginx(self):
        self.assertIn("ExecStart=/usr/bin/ssh ", UNIT)
        for item in ("systemctl restart", "nginx", "tervon-bridge-client.service",
                     "tervon-bridge-server.service"):
            self.assertNotIn(item, UNIT)

if __name__ == "__main__":
    unittest.main()
