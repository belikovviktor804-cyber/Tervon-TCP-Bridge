"""Offline M5.1 release baseline verifier tests; synthetic data only."""
import importlib.util
import pathlib
import tempfile
import unittest

FILE = pathlib.Path(__file__).with_name("verify_release_baseline.py")
SPEC = importlib.util.spec_from_file_location("m5_baseline", FILE)
mod = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(mod)


class BaselineTests(unittest.TestCase):
    def meta(self, size):
        return {"tag_name": "v1.0.0", "assets": [{
            "name": mod.EXPECTED_ASSET, "size": size,
            "browser_download_url": (
                "https://github.com/tervon-software/Tervon-TCP-Bridge/"
                "releases/download/v1.0.0/tervon-bridge-linux-amd64"),
        }]}

    def test_select_known_asset(self):
        self.assertEqual(mod.select_binary_asset(self.meta(3))["size"], 3)

    def test_reject_duplicate_asset(self):
        meta = self.meta(3)
        meta["assets"].append(dict(meta["assets"][0]))
        with self.assertRaises(ValueError):
            mod.select_binary_asset(meta)

    def test_reject_malformed_tag_or_external_url(self):
        meta = self.meta(3)
        meta["tag_name"] = "v1.0.1"
        with self.assertRaises(ValueError):
            mod.select_binary_asset(meta)
        meta = self.meta(3)
        meta["assets"][0]["browser_download_url"] = "https://untrusted.test/file"
        with self.assertRaises(ValueError):
            mod.select_binary_asset(meta)

    def test_mismatch_never_looks_like_pass(self):
        with tempfile.TemporaryDirectory() as tmp:
            p = pathlib.Path(tmp) / "artifact"
            p.write_bytes(b"test")
            report = mod.build_report(self.meta(4), p)
            self.assertEqual(report["status"], "ARTIFACT_DIFFERS_FROM_HISTORICAL")
            self.assertFalse(report["historical_sha_matches"])

    def test_different_asset_size_never_passes(self):
        with tempfile.TemporaryDirectory() as tmp:
            p = pathlib.Path(tmp) / "artifact"
            p.write_bytes(b"abc")
            report = mod.build_report(self.meta(99), p)
            self.assertFalse(report["asset_size_matches"])
            self.assertNotEqual(report["status"], "PASS_ARTIFACT_MATCHES_HISTORICAL")


if __name__ == "__main__":
    unittest.main()
