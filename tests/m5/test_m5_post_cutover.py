#!/usr/bin/env python3
"""Synthetic post-cutover observer tests; never invokes network or systemd."""
import importlib.util
import pathlib
import unittest

src=pathlib.Path(__file__).with_name("m5-post-cutover-readonly.py")
spec=importlib.util.spec_from_file_location("m5_post",src)
module=importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

def fixture():
    return {"schema":"tervon.bridge.dashboard.v1","nodes":{
        role:{"status":"ok","metrics":{
            "ready":True,"max_sessions":0,"sessions_active":8,
            "sessions_started":100,"sessions_rejected":0,
            "upload_bytes":500,"download_bytes":1000,
            "goroutines":20,"memory_alloc_bytes":1000000,
            "errors":{"timeout":0,"authorization":0}
        }} for role in ("homeserver","finserver")}}

class TestPostCutover(unittest.TestCase):
    def test_zero_quota_both_nodes(self):
        result=module.inspect(fixture())
        self.assertEqual(result["homeserver"]["max_sessions"],0)
        self.assertEqual(result["finserver"]["max_sessions"],0)
    def test_nonzero_quota_fails_closed(self):
        raw=fixture()
        raw["nodes"]["finserver"]["metrics"]["max_sessions"]=100
        with self.assertRaises(ValueError):
            module.inspect(raw)
    def test_bad_schema_fails_closed(self):
        raw=fixture()
        raw["schema"]="wrong"
        with self.assertRaises(ValueError):
            module.inspect(raw)
    def test_counter_traffic_observation(self):
        before=module.inspect(fixture())
        raw=fixture()
        raw["nodes"]["homeserver"]["metrics"]["upload_bytes"]+=100
        after=module.inspect(raw)
        self.assertTrue(module.compare(before,after))
        self.assertEqual(after["homeserver"]["sessions_rejected"],0)
    def test_counter_reset_fails(self):
        before=module.inspect(fixture())
        raw=fixture()
        raw["nodes"]["finserver"]["metrics"]["sessions_started"]=90
        with self.assertRaises(ValueError):
            module.compare(before,module.inspect(raw))
    def test_invalid_types_fails(self):
        raw=fixture()
        raw["nodes"]["finserver"]["metrics"]["max_sessions"]=False
        with self.assertRaises(ValueError):
            module.inspect(raw)

if __name__=="__main__":
    unittest.main()
