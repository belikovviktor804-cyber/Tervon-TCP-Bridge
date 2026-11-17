"""Static safety regression only; not a substitute for nginx -t or live auth tests."""
import re
import unittest
from pathlib import Path

TEMPLATE = (Path(__file__).resolve().parent /
            "nginx-bridge.conf.example").read_text(encoding="utf-8")


class NginxTemplateTests(unittest.TestCase):
    def test_auth_is_inside_proxied_location(self):
        block = re.search(
            r"location\s+\^~\s+/bridge/\s*\{([^{}]*)\}",
            TEMPLATE,
        )
        self.assertIsNotNone(block)
        content = block.group(1)
        self.assertRegex(content, r'auth_basic\s+"[^"]+";')
        self.assertIn("auth_basic_user_file __EXISTING_ADMIN_HTPASSWD_FILE__;", content)
        self.assertIn("proxy_pass http://127.0.0.1:8894;", content)

    def test_exact_path_is_proxied_after_auth_not_rewritten(self):
        block = re.search(
            r"location\s*=\s*/bridge\s*\{([^{}]*)\}",
            TEMPLATE,
        )
        self.assertIsNotNone(block)
        content = block.group(1)
        self.assertIn("auth_basic_user_file __EXISTING_ADMIN_HTPASSWD_FILE__;", content)
        self.assertIn("proxy_pass http://127.0.0.1:8894/bridge/;", content)
        self.assertNotRegex(content, r"\\breturn\\s+30[1278]\\b")

    def test_no_external_endpoint_or_existing_monitor_rewrite(self):
        self.assertNotIn("finserver", TEMPLATE.lower())
        self.assertNotIn("proxy_pass http://127.0.0.1:8893", TEMPLATE)
        self.assertNotIn("proxy_pass http://127.0.0.1:8000", TEMPLATE)
        self.assertNotIn("location /infrastructure/", TEMPLATE)
        self.assertNotIn("location /admin/", TEMPLATE)

    def test_template_cannot_be_installed_unedited(self):
        self.assertIn("__EXISTING_ADMIN_HTPASSWD_FILE__", TEMPLATE)


if __name__ == "__main__":
    unittest.main()
