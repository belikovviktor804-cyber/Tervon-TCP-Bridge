"""Offline M4 mixed-version diagnostics: test safe failure attribution only."""
import copy
import unittest

from m4_mixed_diagnose import status, change, run
from test_m3_passive_live_evidence import fixture


class MixedDiagnosticTests(unittest.TestCase):
    def test_valid_pair(self):
        a = fixture()
        b = copy.deepcopy(a)
        b["nodes"]["homeserver"]["metrics"]["sessions_started"] += 1
        b["nodes"]["finserver"]["metrics"]["download_bytes"] += 4096
        result = change(a, b)
        self.assertEqual(result["status"], "PASS_DIAGNOSTIC_COUNTER_CONTINUITY")
        self.assertEqual(result["nodes"]["finserver"]["download_bytes_delta"], 4096)
        self.assertFalse(result["vpn_dataplane_proven"])

    def test_node_unavailable_is_identified(self):
        raw = fixture()
        raw["nodes"]["finserver"]["status"] = "unavailable"
        self.assertEqual(status(raw), {"status": "NOT_READY",
            "reason": "NODE_STATUS_NOT_OK", "role": "finserver"})

    def test_missing_metric_identifies_role_and_field(self):
        raw = fixture()
        raw["nodes"]["finserver"]["metrics"].pop("tls13_h2_mtls_requests")
        self.assertEqual(status(raw), {"status": "INVALID_RESPONSE",
            "reason": "COUNTER_INVALID", "role": "finserver",
            "field": "tls13_h2_mtls_requests"})

    def test_rejects_reset_without_auto_pass(self):
        a = fixture()
        b = copy.deepcopy(a)
        a["nodes"]["finserver"]["metrics"]["sessions_started"] = 9
        self.assertEqual(change(a,b), {"status": "COUNTER_RESET",
            "reason": "COUNTER_DECREASED", "role": "finserver",
            "field": "sessions_started"})

    def test_fetch_error_does_not_leak_source_text(self):
        def fetch():
            raise ValueError("private-url?secret=do-not-print")
        result = run(fetcher=fetch, sleeper=lambda _: None)
        self.assertEqual(result, {"status": "DIAGNOSTIC_FAIL",
                                  "reason": "FETCH_OR_PARSE_EXCEPTION"})
        self.assertNotIn("do-not-print", str(result))

    def test_public_fetch_failure_code(self):
        def fetch():
            raise ValueError("public fetch failed")
        self.assertEqual(run(fetcher=fetch, sleeper=lambda _: None)["reason"],
                         "PUBLIC_HTTPS_FETCH_FAILED")

    def test_not_ready_stops_without_sleep(self):
        bad = fixture()
        bad["nodes"]["finserver"]["metrics"]["ready"] = False
        result = run(fetcher=lambda: bad, sleeper=lambda _: self.fail("slept"))
        self.assertEqual(result["reason"], "BRIDGE_READY_NOT_TRUE")


if __name__ == "__main__":
    unittest.main()
