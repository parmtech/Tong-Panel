import unittest
from unittest.mock import MagicMock, patch
import run_dpkg_labs as guard


class LabGuardTests(unittest.TestCase):
    def setUp(self):
        self.patches = [patch.object(guard.sys, "platform", "linux"),
                        patch.object(guard.os, "geteuid", return_value=1000, create=True),
                        patch.dict(guard.os.environ, {"TONG_DISPOSABLE_DPKG_LAB": "1"}),
                        patch.object(guard.shutil, "which", return_value="/tool")]
        for p in self.patches:
            p.start()
            self.addCleanup(p.stop)

    def test_missing_tool_fails_closed(self):
        with patch.object(guard.shutil, "which", side_effect=lambda tool: None if tool == "dpkg-query" else "/tool"):
            with self.assertRaisesRegex(RuntimeError, "dpkg-query"):
                guard.run("crash")

    def test_root_and_missing_opt_in_fail_closed(self):
        with patch.object(guard.os, "geteuid", return_value=0):
            with self.assertRaises(RuntimeError):
                guard.run("crash")
        with patch.dict(guard.os.environ, {"TONG_DISPOSABLE_DPKG_LAB": "0"}):
            with self.assertRaises(RuntimeError):
                guard.run("crash")

    def test_wrong_count_fails_before_running(self):
        suite = MagicMock()
        suite.countTestCases.return_value = 0
        with patch.object(guard.unittest.defaultTestLoader, "discover", return_value=suite):
            with self.assertRaisesRegex(RuntimeError, "expected 2"):
                guard.run("crash")

    def test_skip_and_test_failure_are_not_success(self):
        suite = MagicMock()
        suite.countTestCases.return_value = 2
        result = MagicMock(testsRun=2, skipped=[("test", "missing")])
        result.wasSuccessful.return_value = True
        with patch.object(guard.unittest.defaultTestLoader, "discover", return_value=suite), \
             patch.object(guard.unittest, "TextTestRunner") as runner:
            runner.return_value.run.return_value = result
            self.assertFalse(guard.run("crash"))
            result.skipped = []
            result.wasSuccessful.return_value = False
            self.assertFalse(guard.run("crash"))
            result.wasSuccessful.return_value = True
            self.assertTrue(guard.run("crash"))
            result.testsRun = 1
            self.assertFalse(guard.run("crash"))


if __name__ == "__main__":
    unittest.main()
