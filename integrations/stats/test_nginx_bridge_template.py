"""Static public read-only nginx template regression.

Not a substitute for nginx -t, safe backup and external TLS route checks.
"""
import unittest
from pathlib import Path

TEMPLATE = (Path(__file__).resolve().parent /
            "nginx-bridge.conf.example").read_text(encoding="utf-8")


class NginxTemplateTests(unittest.TestCase):
    def test_public_locations_are_explicitly_unauthenticated(self):
        self.assertEqual(TEMPLATE.count("location = /bridge {"), 1)
        self.assertEqual(TEMPLATE.count("location ^~ /bridge/ {"), 1)
        exact, prefix = TEMPLATE.split("location = /bridge {", 1)[1].split(
            "location ^~ /bridge/ {", 1
        )
        for location in (exact, prefix):
            self.assertIn("auth_basic off;", location)
            self.assertIn("limit_except GET { deny all; }", location)
            self.assertNotIn("auth_basic_user_file", location)
            self.assertNotIn("proxy_pass http://127.0.0.1:8893", location)
            self.assertNotIn("proxy_pass http://127.0.0.1:8000", location)
        self.assertIn("proxy_pass http://127.0.0.1:8894/bridge/;", exact)
        self.assertIn("proxy_pass http://127.0.0.1:8894;", prefix)
        self.assertNotIn("return 30", exact)

    def test_no_external_endpoint_or_existing_monitor_rewrite(self):
        self.assertNotIn("finserver", TEMPLATE.lower())
        self.assertNotIn("location /infrastructure/ {", TEMPLATE)
        self.assertNotIn("location /admin/ {", TEMPLATE)
        self.assertNotIn("proxy_pass http://127.0.0.1:8893", TEMPLATE)
        self.assertNotIn("proxy_pass http://127.0.0.1:8000", TEMPLATE)
        self.assertNotIn("proxy_pass http://0.0.0.0:", TEMPLATE)


if __name__ == "__main__":
    unittest.main()
