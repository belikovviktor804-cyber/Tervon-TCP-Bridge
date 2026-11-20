"""Isolated unit tests. No real keys, APK build, network, or VPN processes."""
import base64
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import install_catalog_trust_asset as installer


class CatalogAssetTests(unittest.TestCase):
    def setUp(self):
        self.raw_public = bytes(range(32))  # Test-only public bytes, not a signing key.
        self.digest = hashlib.sha256(self.raw_public).hexdigest()
        self.asset = json.dumps({
            "format": installer.FORMAT,
            "keys": {"unit_test_only": base64.b64encode(self.raw_public).decode("ascii")},
        }).encode("utf-8")

    def test_valid_public_asset_and_mismatch(self):
        self.assertEqual(installer.validate(self.asset, self.digest), ("unit_test_only",))
        with self.assertRaisesRegex(ValueError, "fingerprint mismatch"):
            installer.validate(self.asset, "0" * 64)

    def test_reject_extra_key_unsupported_format_and_json_duplicates(self):
        malicious = [
            self.asset.replace(b"trust.v1", b"trust.v0"),
            b'{"format":"tervon.bridge.catalog.trust.v1","format":"tervon.bridge.catalog.trust.v1","keys":{}}',
            json.dumps({
                "format": installer.FORMAT,
                "keys": {
                    "unit_test_only": base64.b64encode(self.raw_public).decode("ascii"),
                    "unverified_second": base64.b64encode(bytes(range(1, 33))).decode("ascii"),
                },
            }).encode("utf-8"),
        ]
        for payload in malicious:
            with self.subTest(payload=payload[:80]):
                with self.assertRaises(ValueError):
                    installer.validate(payload, self.digest)

    def test_check_is_read_only_apply_is_no_clobber(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            main = root / "android/app/src/main"
            main.mkdir(parents=True)
            public_source = root / "trust.json"
            public_source.write_bytes(self.asset)
            script = Path(installer.__file__).resolve()
            base = [
                sys.executable, str(script), "--app-root", str(root),
                "--public-asset", str(public_source),
                "--expected-public-sha256", self.digest,
            ]
            check = subprocess.run(base + ["--check"], capture_output=True, text=True)
            self.assertEqual(check.returncode, 0, check.stderr)
            self.assertIn("M6_CATALOG_TRUST_ASSET=READY", check.stdout)
            self.assertFalse((main / "assets").exists())

            apply = subprocess.run(base + ["--apply"], capture_output=True, text=True)
            self.assertEqual(apply.returncode, 0, apply.stderr)
            dest = main / "assets" / installer.NAME
            self.assertEqual(dest.read_bytes(), self.asset)

            repeat = subprocess.run(base + ["--apply"], capture_output=True, text=True)
            self.assertEqual(repeat.returncode, 0, repeat.stderr)
            self.assertIn("ALREADY_PRESENT_IDENTICAL", repeat.stdout)

            altered = self.asset + b"\n"
            public_source.write_bytes(altered)
            rejected = subprocess.run(base + ["--apply"], capture_output=True, text=True)
            self.assertEqual(rejected.returncode, 2)
            self.assertEqual(dest.read_bytes(), self.asset)

    def test_no_secret_file_installation(self):
        self.assertEqual(
            installer.TARGET_REL,
            Path("android/app/src/main/assets/bridge_catalog_trust_v1.json"),
        )


if __name__ == "__main__":
    unittest.main()
