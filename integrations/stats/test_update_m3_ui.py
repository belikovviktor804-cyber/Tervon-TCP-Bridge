"""No-systemd tests of the guarded M3 UI hot-update and rollback."""
import importlib.util
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

SOURCE = Path(__file__).with_name("update-m3-ui.py")
spec = importlib.util.spec_from_file_location("update_m3_ui", SOURCE)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class UIUpdateTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.app = Path(self.tmp.name)
        for name in module.FILES:
            (self.app / name).write_bytes(b"previous " + name.encode())
        self.patches = [
            patch.object(module, "APP", self.app),
            patch.object(module, "preflight"),
            patch.object(module, "command", return_value="active"),
        ]
        for ctx in self.patches:
            ctx.start()
            self.addCleanup(ctx.stop)

    def test_hot_update_without_restarting_services(self):
        with patch.object(module, "live_ui_matches", return_value=True) as live:
            module.install()
        live.assert_called_once()
        for name in module.FILES:
            self.assertEqual((self.app / name).read_bytes(),
                             (SOURCE.parent / name).read_bytes())
        backups = list((self.app / ".ui-backups").glob("m3-ui-*"))
        self.assertEqual(len(backups), 1)
        for name in module.FILES:
            self.assertEqual((backups[0] / name).read_bytes(),
                             b"previous " + name.encode())

    def test_failed_live_check_restores_all_assets(self):
        with patch.object(module, "live_ui_matches", return_value=False):
            with self.assertRaisesRegex(RuntimeError, "mismatch"):
                module.install()
        for name in module.FILES:
            self.assertEqual((self.app / name).read_bytes(),
                             b"previous " + name.encode())

    def test_atomic_write_does_not_leave_temp_files(self):
        p = self.app / module.FILES[0]
        module.write_atomic(p, b"new")
        self.assertEqual(p.read_bytes(), b"new")
        self.assertEqual(list(self.app.glob(".bridge-ui-*")), [])

    def test_preflight_requires_root_and_correct_ssh_branch(self):
        src = SOURCE.parent
        with patch.object(module.os, "geteuid", return_value=1000):
            with self.assertRaisesRegex(RuntimeError, "root"):
                module.preflight(src.parents[1], src)
        with patch.object(module.os, "geteuid", return_value=0), \
             patch.object(module, "command", return_value="https://github.com/wrong"):
            with self.assertRaisesRegex(RuntimeError, "SSH remote"):
                module.preflight(src.parents[1], src)


if __name__ == "__main__":
    unittest.main()
