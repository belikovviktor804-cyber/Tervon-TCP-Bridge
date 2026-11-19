#!/usr/bin/env python3
"""Offline-only coordinator safety tests: SSH/systemd/staging all mocked."""
import importlib.util
from pathlib import Path
from unittest import mock
import unittest

source=Path(__file__).with_name("m5-dual-cutover.py")
spec=importlib.util.spec_from_file_location("dual_coordinator",source)
coordinator=importlib.util.module_from_spec(spec)
spec.loader.exec_module(coordinator)

class TestOrchestrator(unittest.TestCase):
    def setUp(self):
        patches=[
            mock.patch.object(coordinator.os,"geteuid",return_value=0),
            mock.patch.object(coordinator.sys,"argv",["cutover.py","deploy"]),
            mock.patch.object(coordinator,"WORKER",mock.Mock(is_file=lambda: True, is_symlink=lambda: False)),
            mock.patch.object(coordinator,"CANDIDATE",mock.Mock(is_file=lambda: True, is_symlink=lambda: False)),
            mock.patch.object(coordinator,"sha256",return_value=coordinator.NEW),
            mock.patch.object(coordinator,"checked"),
            mock.patch.object(coordinator,"stage_remote"),
            mock.patch.object(coordinator,"both"),
            mock.patch.object(coordinator,"confirm_both_no_quota",return_value=True),
            mock.patch.object(coordinator,"output"),
        ]
        self.mocks=[p.start() for p in patches]
        for p in patches:
            self.addCleanup(p.stop)
        def check(args,**kw):
            if "branch" in args:
                return mock.Mock(stdout=b"work/m5-linux-release\n")
            return mock.Mock(stdout=b"git@github.com:tervon-software/Tervon-TCP-Bridge.git\n")
        coordinator.checked.side_effect=check

    def test_full_success(self):
        coordinator.both.side_effect=[
            {"homeserver":True,"finserver":True},
            {"homeserver":True,"finserver":True},
        ]
        self.assertEqual(coordinator.main(),0)
        self.assertEqual([c.args[0] for c in coordinator.both.call_args_list],
                         ["prepare","switch"])
        coordinator.confirm_both_no_quota.assert_called_once()

    def test_preparation_failure_cannot_restart(self):
        coordinator.both.side_effect=[
            {"homeserver":True,"finserver":False},
        ]
        self.assertEqual(coordinator.main(),3)
        self.assertEqual([c.args[0] for c in coordinator.both.call_args_list],
                         ["prepare"])

    def test_one_switch_failure_rolls_back_both(self):
        coordinator.both.side_effect=[
            {"homeserver":True,"finserver":True},
            {"homeserver":True,"finserver":False},
            {"homeserver":True,"finserver":True},
        ]
        self.assertEqual(coordinator.main(),1)
        self.assertEqual([c.args[0] for c in coordinator.both.call_args_list],
                         ["prepare","switch","rollback"])
        coordinator.confirm_both_no_quota.assert_not_called()

    def test_telemetry_failure_rolls_back_both(self):
        coordinator.both.side_effect=[
            {"homeserver":True,"finserver":True},
            {"homeserver":True,"finserver":True},
            {"homeserver":True,"finserver":True},
        ]
        coordinator.confirm_both_no_quota.side_effect=RuntimeError("mock telemetry")
        self.assertEqual(coordinator.main(),1)
        self.assertEqual([c.args[0] for c in coordinator.both.call_args_list],
                         ["prepare","switch","rollback"])

if __name__=="__main__":
    unittest.main()
