#!/usr/bin/env python3
"""Offline cutover test. No systemd calls and no server resources touched."""
import hashlib
import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import types
import unittest
from unittest import mock

SOURCE=Path(__file__).with_name("m5-dual-cutover-worker.py")
spec=importlib.util.spec_from_file_location("cutover_worker",SOURCE)
worker=importlib.util.module_from_spec(spec)
spec.loader.exec_module(worker)

class TestM5WorkerTransaction(unittest.TestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        root=Path(self.temp.name)
        self.bin=root/"bridge"
        self.conf=root/"bridge.json"
        self.next=root/"candidate"
        self.bin.write_bytes(b"old binary")
        self.next.write_bytes(b"new binary")
        self.conf.write_text(json.dumps({"mode":"client","max_local_connections":100,"key_file":"PRIVATE_PLACEHOLDER"})+"\n")
        self.old=hashlib.sha256(self.bin.read_bytes()).hexdigest()
        self.new=hashlib.sha256(self.next.read_bytes()).hexdigest()
        self.stage=root/"staging"
        self.role=("mock.service",self.bin,self.conf,"client","max_local_connections")
        self.active="old"
        for name,value in (("STAGE",self.stage),("OLD",self.old),("NEW",self.new)):
            patcher=mock.patch.object(worker,name,value)
            patcher.start()
            self.addCleanup(patcher.stop)
        patch=mock.patch.object(worker,"validate_unit",return_value=None)
        patch.start();self.addCleanup(patch.stop)
        def service_state(unit,binary,sha):
            return self.active==("new" if sha==self.new else "old") and worker.digest(binary)==sha
        patch=mock.patch.object(worker,"verify_service",side_effect=service_state)
        patch.start();self.addCleanup(patch.stop)
        self.fail_next_restart=False
        def run_cmd(*args,**kwargs):
            if args[:2]==("systemctl","restart"):
                if self.fail_next_restart:
                    self.fail_next_restart=False
                    raise RuntimeError("simulated_restart_failure")
                self.active="new" if worker.digest(self.bin)==self.new else "old"
            return types.SimpleNamespace(stdout="active",returncode=0)
        patch=mock.patch.object(worker,"run",side_effect=run_cmd)
        patch.start();self.addCleanup(patch.stop)

    def prepare(self):
        with mock.patch.object(sys,"argv",["worker.py","prepare","homeserver",str(self.next)]):
            worker.prepare("homeserver",*self.role)

    def test_prepare_is_read_only_for_live_files(self):
        before_bin=self.bin.read_bytes()
        before_conf=self.conf.read_bytes()
        self.prepare()
        self.assertEqual(self.bin.read_bytes(),before_bin)
        self.assertEqual(self.conf.read_bytes(),before_conf)
        self.assertEqual(worker.digest(self.stage/"old-binary"),self.old)
        self.assertEqual(worker.digest(self.stage/"new-binary"),self.new)
        self.assertEqual(json.loads((self.stage/"new-config").read_text())["max_local_connections"],0)
        self.assertEqual(json.loads((self.stage/"old-config").read_text())["max_local_connections"],100)
        with self.assertRaises(RuntimeError):
            self.prepare()

    def test_switch_and_rollback(self):
        self.prepare()
        worker.install_new("homeserver",*self.role)
        self.assertEqual(self.active,"new")
        self.assertEqual(worker.digest(self.bin),self.new)
        self.assertEqual(json.loads(self.conf.read_text())["max_local_connections"],0)
        worker.rollback("homeserver",*self.role)
        self.assertEqual(self.active,"old")
        self.assertEqual(worker.digest(self.bin),self.old)
        self.assertEqual(json.loads(self.conf.read_text())["max_local_connections"],100)

    def test_failed_restart_restores_old_binary_and_config(self):
        self.prepare()
        self.fail_next_restart=True
        with self.assertRaises(RuntimeError):
            worker.install_new("homeserver",*self.role)
        self.assertEqual(self.active,"old")
        self.assertEqual(worker.digest(self.bin),self.old)
        self.assertEqual(json.loads(self.conf.read_text())["max_local_connections"],100)

    def test_tampered_backup_fails_closed(self):
        self.prepare()
        (self.stage/"old-config").write_bytes(b"tampered")
        with self.assertRaises(RuntimeError):
            worker.install_new("homeserver",*self.role)
        self.assertEqual(self.bin.read_bytes(),b"old binary")

if __name__=="__main__":
    unittest.main()
