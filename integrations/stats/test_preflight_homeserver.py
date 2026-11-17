"""Mocked preflight tests; never touch live services or auth files."""
import importlib.util
import io
import tempfile
import unittest
from contextlib import redirect_stdout
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location(
    "preflight_homeserver", ROOT / "preflight_homeserver.py"
)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class PreflightTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        base = Path(self.temp.name)
        self.nginx = base / "nginx.conf"
        self.auth = base / "secret-htpasswd"
        self.auth.write_text("test:fake-hash\n", encoding="utf-8")
        self.nginx.write_text(
            'location /admin/ {\n'
            '  auth_basic "Admin";\n'
            f'  auth_basic_user_file {self.auth};\n'
            '}\n', encoding="utf-8"
        )
        self.patches = [
            patch.object(module, "NGINX_CONFIG", self.nginx),
            patch.object(module, "UNIT", base / "not-installed.service"),
            patch.object(module, "APP_DIR", base / "not-installed"),
        ]
        for p in self.patches:
            p.start()
            self.addCleanup(p.stop)

    def command(self, *args):
        commands = {
            ("git", "remote", "get-url", "origin"):
                (0, module.EXPECTED_REMOTE),
            ("git", "branch", "--show-current"):
                (0, "work/m1-framing"),
            ("git", "rev-parse", "HEAD"):
                (0, "a" * 40),
            ("git", "status", "--porcelain"):
                (0, ""),
            ("ss", "-H", "-ltn", "( sport = :8894 )"):
                (0, ""),
            ("systemctl", "--version"):
                (0, "systemd 257 (257.5)"),
        }
        if args[1:] == ("-c", args[-1]) and args[0] == str(module.VENV_PYTHON):
            return 0, "0.139.0\n0.50.2"
        return commands.get(args, (127, ""))

    def test_preflight_clean_and_no_secret_output(self):
        with patch.object(module, "command", side_effect=self.command):
            out = io.StringIO()
            with redirect_stdout(out):
                result = module.main()
        self.assertEqual(result, 0)
        self.assertIn("ADMIN_HTPASSWD: PASS", out.getvalue())
        self.assertIn("DYNAMIC_USER_ACCESS: CHECK", out.getvalue())
        self.assertNotIn(str(self.auth), out.getvalue())
        self.assertNotIn("fake-hash", out.getvalue())

    def test_port_conflict_blocks(self):
        def busy(*args):
            if args[:2] == ("ss", "-H"):
                return 0, "LISTEN 0 128 127.0.0.1:8894"
            return self.command(*args)
        with patch.object(module, "command", side_effect=busy):
            rows = module.checks()
        self.assertIn(("PORT_8894", "BLOCKED", "listener exists or ss unavailable"), rows)

    def test_non_ssh_origin_blocks(self):
        def wrong(*args):
            if args == ("git", "remote", "get-url", "origin"):
                return 0, "https://github.com/tervon-software/Tervon-TCP-Bridge.git"
            return self.command(*args)
        with patch.object(module, "command", side_effect=wrong):
            rows = module.checks()
        self.assertTrue(any(k == "GIT_SSH_REMOTE" and status == "BLOCKED"
                            for k, status, _ in rows))

    def test_missing_htpasswd_blocks(self):
        self.auth.unlink()
        with patch.object(module, "command", side_effect=self.command):
            rows = module.checks()
        self.assertTrue(any(k == "ADMIN_HTPASSWD" and status == "BLOCKED"
                            for k, status, _ in rows))


if __name__ == "__main__":
    unittest.main()
