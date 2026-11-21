#!/usr/bin/env python3
"""Isolated tests of staging-only M6 XHTTP backend configuration."""
import unittest
from stage_m6_xhttp_backend import candidate


def current():
    xray = {"inbounds": [{
        "tag": "xhttp", "protocol": "vless", "port": 30443,
        "listen": "0.0.0.0", "streamSettings": {"network": "xhttp", "security": "tls"},
    }]}
    bridge = {
        "mode": "server",
        "routes": {"vpn": [{"Name": "tcp", "Address": "host.docker.internal:30000"}]},
        "authorized": {"a" * 64: ["vpn"], "b" * 64: ["other"]},
        "padding_enabled": True,
    }
    return xray, bridge


class TestM6XHTTPBackend(unittest.TestCase):
    def test_preserves_original_and_upgrades_existing_device_acl(self):
        xray, bridge = current()
        result = candidate(xray, bridge)
        self.assertEqual(result["routes"]["vpn-xhttp"][0]["Address"],
                         "host.docker.internal:30443")
        self.assertEqual(result["routes"]["vpn"], bridge["routes"]["vpn"])
        self.assertEqual(result["authorized"]["a" * 64], ["vpn", "vpn-xhttp"])
        self.assertEqual(result["authorized"]["b" * 64], ["other"])
        self.assertEqual(bridge["authorized"]["a" * 64], ["vpn"])
        self.assertTrue(result["padding_enabled"])
        self.assertNotIn("vpn-xhttp", bridge["routes"])

    def test_no_guess_if_ambiguous_or_loopback(self):
        xray, bridge = current()
        xray["inbounds"].append(xray["inbounds"][0].copy())
        with self.assertRaises(ValueError):
            candidate(xray, bridge)
        xray, bridge = current()
        xray["inbounds"][0]["listen"] = "127.0.0.1"
        with self.assertRaises(ValueError):
            candidate(xray, bridge)
        xray, bridge = current()
        bridge["routes"]["vpn-xhttp"] = []
        with self.assertRaises(ValueError):
            candidate(xray, bridge)

    def test_no_route_without_verified_tcp_backend(self):
        xray, bridge = current()
        del bridge["routes"]["vpn"]
        with self.assertRaises(ValueError):
            candidate(xray, bridge)


if __name__ == "__main__":
    unittest.main()
