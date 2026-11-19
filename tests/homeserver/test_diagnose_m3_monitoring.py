import importlib.util
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

SCRIPT = Path(__file__).with_name("diagnose_m3_monitoring.py")
spec = importlib.util.spec_from_file_location("diagnose_m3_monitoring", SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class MonitoringDiagnosisTests(unittest.TestCase):
    def test_stats_env_link_is_validated_without_exposing_values(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "stats.env"
            expected = "http://127.0.0.1:8895/metrics/v1"
            self.assertEqual(module.stats_source_env(path, expected), "NOT_CONFIGURED")
            path.write_text("TERVON_BRIDGE_METRICS_URL=" + expected + "\n")
            self.assertEqual(module.stats_source_env(path, expected), "MATCH")
            path.write_text("TERVON_BRIDGE_METRICS_URL=https://private.example/secret\\n")
            self.assertEqual(module.stats_source_env(path, expected), "INVALID_OR_REMOTE")
            path.write_text("TERVON_BRIDGE_METRICS_URL=" + expected + "\n" +
                            "TERVON_BRIDGE_METRICS_URL=" + expected + "\n")
            self.assertEqual(module.stats_source_env(path, expected), "AMBIGUOUS")

    def test_url_requires_loopback_explicit_port(self):
        self.assertEqual(
            module.safe_metrics_url({"metrics_listen": "127.0.0.1:8895"}),
            "http://127.0.0.1:8895/metrics/v1")
        for address in ("", "0.0.0.0:8895", "192.168.0.1:8895",
                        "somehost:8895", "localhost:8895", "127.0.0.1:0",
                        "127.0.0.1:65536"):
            self.assertIsNone(module.safe_metrics_url({"metrics_listen": address}))

    def test_disabled_metrics_are_explained_without_secrets(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "client.json"
            path.write_text(json.dumps({
                "mode": "client", "metrics_listen": "",
                "key_file": "/private/do-not-log-key.pem",
                "remote_url": "https://secret-host/v2/stream",
            }))
            with patch.object(module, "unit_state", side_effect=["ACTIVE", "ACTIVE"]), \
                 patch.object(module, "local_json", return_value={
                     "schema": module.DASH_SCHEMA,
                     "nodes": {"homeserver": {"status": "unavailable"}},
                 }) as local:
                got = module.diagnose(path)
            self.assertEqual(got["bridge_metrics_config"], "DISABLED_OR_INVALID")
            self.assertEqual(got["bridge_metrics_local"], "NOT_CHECKED")
            self.assertEqual(got["stats_api_local"], "PASS")
            self.assertEqual(got["stats_bridge_node"], "UNAVAILABLE")
            local.assert_called_once_with(module.STATS_LOCAL)
            self.assertNotIn("secret-host", json.dumps(got))
            self.assertNotIn("do-not-log-key", json.dumps(got))

    def test_metrics_enabled_and_dashboard_up(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "client.json"
            path.write_text(json.dumps({
                "mode": "client", "metrics_listen": "127.0.0.1:8895"
            }))
            def respond(url):
                if url.endswith("/metrics/v1"):
                    return {"schema": module.SCHEMA, "ready": True}
                return {
                    "schema": module.DASH_SCHEMA,
                    "nodes": {"homeserver": {"status": "ok"}},
                }
            with patch.object(module, "unit_state", return_value="ACTIVE"), \
                 patch.object(module, "local_json", side_effect=respond):
                got = module.diagnose(path)
            self.assertEqual(got["bridge_metrics_local"], "PASS")
            self.assertEqual(got["stats_bridge_node"], "OK")
            self.assertEqual(got["public_site_external"], "NOT_TESTED")

    def test_invalid_config_never_reports_healthy_source(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "client.json"
            path.write_text("{not json")
            with patch.object(module, "unit_state", return_value="ACTIVE"), \
                 patch.object(module, "local_json", return_value=None):
                got = module.diagnose(path)
            self.assertEqual(got["bridge_metrics_config"], "UNKNOWN")
            self.assertEqual(got["bridge_metrics_local"], "NOT_CHECKED")


if __name__ == "__main__":
    unittest.main()
