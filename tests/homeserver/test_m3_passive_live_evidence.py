"""Isolated tests: no remote SSH, public requests, services or timers."""
from __future__ import annotations

import copy
import unittest

from m3_passive_live_evidence import COUNTERS, ERRORS, collect, parse, calculate


def fixture():
    base = dict(
        ready=True, sessions_active=2,
        **{k: 0 for k in COUNTERS},
    )
    base["errors"] = {k: 0 for k in ERRORS}
    return {"schema": "tervon.bridge.dashboard.v1", "nodes": {
        "homeserver": {"status": "ok", "metrics": copy.deepcopy(base)},
        "finserver": {"status": "ok", "metrics": copy.deepcopy(base)},
    }}


class PassiveEvidenceTests(unittest.TestCase):
    def test_allows_only_two_ready_nodes(self):
        data = fixture()
        self.assertEqual(set(parse(data)), {"homeserver", "finserver"})
        data["nodes"]["finserver"]["status"] = "unavailable"
        with self.assertRaises(ValueError):
            parse(data)

    def test_rejects_invalid_and_missing_counters(self):
        for val in (False, -1, 1.5, "2", None):
            d = fixture()
            d["nodes"]["homeserver"]["metrics"]["jitter_upload_frames"] = val
            with self.subTest(val=val), self.assertRaises(ValueError):
                parse(d)

    def test_rejects_reset(self):
        original = fixture()
        original["nodes"]["homeserver"]["metrics"]["padding_upload_frames"] = 7
        later = fixture()
        with self.assertRaisesRegex(ValueError, "counter reset"):
            calculate(parse(original), parse(later))

    def test_passive_collection_no_fabricated_rtt_or_pass(self):
        baseline = fixture()
        later = fixture()
        later["nodes"]["homeserver"]["metrics"]["sessions_started"] = 2
        later["nodes"]["homeserver"]["metrics"]["padding_download_frames"] = 5
        later["nodes"]["finserver"]["metrics"]["jitter_download_frames"] = 3
        later["nodes"]["finserver"]["metrics"]["jitter_download_delay_ns"] = 2300000
        incoming = iter((baseline, later))
        events = []
        outcome = collect(60, 60, fetcher=lambda: next(incoming),
                          sleep=lambda n: None, emit=events.append)
        self.assertEqual(outcome["samples"], 2)
        self.assertEqual(outcome["nodes"]["homeserver"]["counter_deltas"]["padding_download_frames"], 5)
        self.assertEqual(outcome["nodes"]["finserver"]["counter_deltas"]["jitter_download_frames"], 3)
        self.assertEqual(outcome["rtt"], "NOT_MEASURED")
        self.assertEqual(outcome["per_stream_fairness"], "NOT_MEASURED")
        self.assertEqual(outcome["full_M3_gate"], "NOT_ASSERTED")
        self.assertEqual([e["kind"] for e in events], ["start", "sample", "summary"])

    def test_activation_gate_requires_both_pad_and_jitter(self):
        first = fixture()
        second = fixture()
        second["nodes"]["finserver"]["metrics"]["padding_upload_frames"] = 7
        second["nodes"]["homeserver"]["metrics"]["padding_download_frames"] = 7
        second["nodes"]["finserver"]["metrics"]["jitter_upload_frames"] = 3
        second["nodes"]["finserver"]["metrics"]["jitter_upload_delay_ns"] = 2000000
        data = iter((first, second))
        result = collect(60, 60, fetcher=lambda: next(data),
                         sleep=lambda s: None, emit=lambda e: None,
                         require_activation=True)
        self.assertEqual(result["status"], "PASS_PAD_AND_JITTER_ACTIVATED")
        self.assertEqual(result["full_M3_gate"], "NOT_ASSERTED")
        self.assertFalse(result["activation"]["demand_generated_by_sampler"])

    def test_activation_gate_refuses_zero_eligible_traffic(self):
        data = iter((fixture(), fixture()))
        result = collect(60, 60, fetcher=lambda: next(data),
                         sleep=lambda s: None, emit=lambda e: None,
                         require_activation=True)
        self.assertEqual(result["status"], "NO_OBSERVED_PAD_OR_JITTER_ACTIVATION")
        self.assertEqual(result["full_M3_gate"], "NOT_ASSERTED")

    def test_activation_gate_refuses_pad_without_jitter(self):
        first = fixture()
        second = fixture()
        second["nodes"]["homeserver"]["metrics"]["padding_upload_frames"] = 10
        data = iter((first, second))
        result = collect(60, 60, fetcher=lambda: next(data),
                         sleep=lambda s: None, emit=lambda e: None,
                         require_activation=True)
        self.assertEqual(result["status"], "PAD_ONLY_JITTER_NOT_OBSERVED")

    def test_does_not_accept_missing_samples(self):
        with self.assertRaises(ValueError):
            collect(70, 60, fetcher=lambda: fixture(), sleep=lambda n: None)

    def test_status_refuses_false_readiness(self):
        d = fixture()
        d["nodes"]["finserver"]["metrics"]["ready"] = False
        with self.assertRaises(ValueError):
            parse(d)


if __name__ == "__main__":
    unittest.main()
