"""Disposable-only, rootless synthetic dpkg interruption and reconciliation lab."""

import os
import shlex
import signal
import subprocess
import time
import unittest

from test_dpkg_failure_recovery_lab import DpkgFailureRecoveryLab


class DpkgCrashRecoveryLab(DpkgFailureRecoveryLab):
    def test_uninterrupted_upgrade_control(self):
        self.install(self.v1)
        self.install(self.v2)
        self.assertEqual(self.installed(), "install ok installed|2")
        self.assert_payload_version("2")
        self.assert_customer_state()

    def test_interrupted_upgrade_requires_explicit_reconciliation(self):
        self.install(self.v1)
        package = self.build_fixture("2", "postinst")
        stage = self.sandbox / "stage-2-postinst-0"
        marker = self.sandbox / "postinst-entered"
        hook = stage / "DEBIAN/postinst"
        hook.write_text("#!/bin/sh\nprintf 'entered\\n' > " + shlex.quote(str(marker)) +
                        "\nwhile :; do sleep 1; done\n", encoding="ascii")
        hook.chmod(0o755)
        self.run_command(["dpkg-deb", "--build", "--root-owner-group", str(stage), str(package)], expected=0)
        command = ["dpkg", "--force-not-root", "--force-script-chrootless",
                   f"--admindir={self.admindir}", f"--instdir={self.instdir}",
                   f"--log={self.log}", "--install", str(package)]
        env = {**os.environ, "LC_ALL": "C", "PATH": "/usr/sbin:/sbin:/usr/bin:/bin"}
        # A new session contains only the dpkg process and its owned fixture hook.
        process = subprocess.Popen(command, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                                   env=env, start_new_session=True)
        try:
            deadline = time.monotonic() + 30
            while not marker.exists() and process.poll() is None and time.monotonic() < deadline:
                time.sleep(0.05)
            self.assertTrue(marker.is_file(), "fixture did not reach the interruption point")
            self.assertIsNone(process.poll(), "dpkg exited before interruption")
        finally:
            # Kill the entire owned session, including the inert paused hook and sleep.
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            process.wait(timeout=10)
        self.assertNotEqual(process.returncode, 0)
        # Query afresh: unpacked bytes alone do not establish a completed upgrade.
        self.assertEqual(self.installed(), "install ok half-configured|2")
        self.assert_payload_version("2")
        self.assert_customer_state()
        # Explicit local recovery, followed by independent state and data checks.
        self.install(self.v1)
        self.assertEqual(self.installed(), "install ok installed|1")
        self.assert_payload_version("1")
        self.assert_customer_state()


def load_tests(loader, tests, pattern):
    # The original five regressions are executed by their own test module.
    return unittest.TestSuite(loader.loadTestsFromName(name, DpkgCrashRecoveryLab) for name in (
        "test_uninterrupted_upgrade_control", "test_interrupted_upgrade_requires_explicit_reconciliation"))
