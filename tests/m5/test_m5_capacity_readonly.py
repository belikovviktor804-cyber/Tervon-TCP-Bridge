#!/usr/bin/env python3
"""Offline, synthetic tests for M5 read-only limit reporter."""
import importlib.util
import pathlib
import unittest

source=pathlib.Path(__file__).with_name("m5_capacity_readonly.py")
spec=importlib.util.spec_from_file_location("m5_capacity_readonly",source)
module=importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

def fixture():
    def metrics(active, cap, rejected):
        return {"ready":True,"max_sessions":cap,"sessions_active":active,
            "sessions_started":424,"sessions_rejected":rejected,
            "goroutines":19,"memory_alloc_bytes":1048576}
    return {"schema":"tervon.bridge.dashboard.v1","nodes":{
        "homeserver":{"status":"ok","metrics":metrics(97,100,313)},
        "finserver":{"status":"ok","metrics":metrics(97,128,0)},
    }}

class TestCapacitySummary(unittest.TestCase):
    def test_accepted_limits(self):
        output="\n".join(module.summarize(fixture()))
        self.assertIn("M5_CAPACITY_HOMESERVER_CONFIGURED_MAX_SESSIONS=100",output)
        self.assertIn("M5_CAPACITY_HOMESERVER_HEADROOM_NOW=3",output)
        self.assertIn("M5_CAPACITY_FINSERVER_CONFIGURED_MAX_SESSIONS=128",output)
        self.assertIn("M5_CAPACITY_LOAD_HOMESERVER_REJECTS_180S=313",output)
        self.assertIn("LIMIT_REACHED_IS_SOURCE_PLAUSIBLE_NOT_TIMESTAMP_PROVEN",output)
    def test_saturated(self):
        raw=fixture()
        raw["nodes"]["homeserver"]["metrics"]["sessions_active"]=100
        self.assertIn("M5_CAPACITY_HOMESERVER_SATURATED_NOW=TRUE","\n".join(module.summarize(raw)))
    def test_not_available_and_untrusted_data(self):
        for bad in (None,"100",True,-1):
            raw=fixture()
            raw["nodes"]["homeserver"]["metrics"]["max_sessions"]=bad
            with self.subTest(bad=bad),self.assertRaises(ValueError):
                module.summarize(raw)
    def test_fails_closed_when_schema_missing(self):
        raw=fixture()
        raw["schema"]="unknown"
        with self.assertRaises(ValueError):
            module.summarize(raw)

if __name__=="__main__":
    unittest.main()
