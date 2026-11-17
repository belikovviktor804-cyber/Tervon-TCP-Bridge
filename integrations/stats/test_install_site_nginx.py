"""Offline regression tests for guarded nginx insertion; no root or nginx reload."""
from __future__ import annotations

import importlib.util
import os
from pathlib import Path
import stat
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("bridge_nginx", ROOT / "install-site-nginx.py")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

GOOD = """server {
    listen 80;
    server_name stats.xrayvless.online;
    return 301 https://stats.xrayvless.online$request_uri;
}
server {
    listen 443 ssl http2;
    server_name stats.xrayvless.online;
    include /etc/nginx/some-tls.conf;
    location /admin/ {
        auth_basic "Admin";
        if ($request_method = POST) { return 405; }
    }
    location /infrastructure/ { proxy_pass http://127.0.0.1:8893; }
    include /etc/nginx/some-site.conf;
}
"""


class NginxInsertionTests(unittest.TestCase):
    def test_inserts_only_in_https_server(self):
        result = module.render_site(GOOD)
        self.assertEqual(result.count(module.MARKER), 1)
        self.assertIn("listen 443 ssl http2;\n" + module.MARKER, result)
        self.assertIn("listen 80;", result)
        self.assertIn('auth_basic "Admin";', result)
        self.assertIn("location /infrastructure/", result)
        self.assertEqual(result.replace(module.MARKER, ""), GOOD)

    def test_fail_closed_on_unknown_vhost_layout(self):
        variants = (
            GOOD.replace("listen 443 ssl http2;", "listen 8443 ssl;"),
            GOOD.replace("server_name stats.xrayvless.online;",
                         "server_name wrong.example;", 1),
            GOOD.replace("listen 443 ssl http2;", "listen 443 ssl;\n    listen [::]:443 ssl;"),
            GOOD + "server { listen 444; }\n",
            GOOD.replace("    listen 443 ssl http2;",
                         "    location /other { return 404; }\n"
                         "    location /other2 { return 404; }\n"
                         "    location /other3 { return 404; }\n"
                         "    location /other4 { return 404; }\n"
                         "    listen 443 ssl http2;"),
            GOOD.replace("location /admin/", "location /bridge/"),
            GOOD.replace("listen 443 ssl http2;",
                         "listen 443 ssl http2;\n" + module.MARKER.strip()),
        )
        for config in variants:
            with self.subTest(config=config[-110:]):
                with self.assertRaises(ValueError):
                    module.render_site(config)

    def test_effective_conflict_detection(self):
        for config in (
            "server { \nlocation /bridge/ { proxy_pass http://127.0.0.1:1; }\n}",
            "location = /bridge { return 200; }",
            "location ^~ /bridge/ { return 200; }",
            "location ~ /bridge/ { return 200; }",
            "include /etc/nginx/tervon-bridge-stats/locations.conf;",
        ):
            with self.subTest(config=config):
                with self.assertRaises(RuntimeError):
                    module.ensure_no_existing_bridge(config)
        module.ensure_no_existing_bridge(GOOD)

    def test_atomic_update_preserves_vhost_permissions(self):
        with tempfile.TemporaryDirectory() as temp:
            directory = Path(temp)
            live = directory / "site"
            backup = directory / "backup"
            live.write_bytes(b"original")
            os.chmod(live, 0o640)
            backup.write_bytes(b"original")
            os.chmod(backup, 0o600)
            module.atomic_copy_bytes(live, b"new", backup)
            self.assertEqual(live.read_bytes(), b"new")
            self.assertEqual(stat.S_IMODE(live.stat().st_mode), 0o640)

    def test_template_only_adds_public_read_only_routes(self):
        snippet = module.TEMPLATE.read_text(encoding="utf-8")
        self.assertEqual(snippet.count("auth_basic off;"), 2)
        self.assertEqual(snippet.count("location "), 2)
        self.assertIn("limit_except GET { deny all; }", snippet)
        self.assertNotIn("auth_basic_user_file", snippet)


if __name__ == "__main__":
    unittest.main()
