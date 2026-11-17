"""Disposable-only, rootless synthetic dpkg interruption and reconciliation lab."""

import os
import shlex
import signal
import subprocess
import time
import unittest

from test_dpkg_failure_recovery_lab import DpkgFailureRecoveryLab, PACKAGE


class DpkgCrashRecoveryLab(DpkgFailureRecoveryLab):
    def build_initial_interruption_fixture(self, marker):
        stage = self.sandbox / "stage-initial-interruption"
        control = stage / "DEBIAN"
        control.mkdir(parents=True)
        (control / "control").write_text(
            f"Package: {PACKAGE}\nVersion: 1\nArchitecture: all\n"
            "Maintainer: Synthetic test only\nDescription: inert initial-install interruption fixture\n",
            encoding="ascii",
        )
        for hook in ("preinst", "postinst", "prerm", "postrm"):
            if hook == "postinst":
                content = (
                    "#!/bin/sh\n"
                    "if [ -e " + shlex.quote(str(marker)) + " ]; then\n"
                    "  exit 0\n"
                    "fi\n"
                    "printf 'entered\\n' > " + shlex.quote(str(marker)) + "\n"
                    "while :; do sleep 1; done\n"
                )
            else:
                content = "#!/bin/sh\n# inert, no host changes\nexit 0\n"
            hook_file = control / hook
            hook_file.write_text(content, encoding="ascii")
            hook_file.chmod(0o755)
        payload = stage / f"usr/share/{PACKAGE}/release.txt"
        payload.parent.mkdir(parents=True)
        payload.write_text("1\n", encoding="ascii")
        package = self.sandbox / "fixture-initial-interruption.deb"
        self.run_command(["dpkg-deb", "--build", "--root-owner-group", str(stage), str(package)], expected=0)
        return package

    def test_uninterrupted_upgrade_control(self):
        self.install(self.v1)
        self.install(self.v2)
        self.assertEqual(self.installed(), "install ok installed|2")
        self.assert_payload_version("2")
        self.assert_customer_state()

    def test_interrupted_initial_install_requires_explicit_reconciliation(self):
        marker = self.sandbox / "initial-postinst-entered"
        payload = self.instdir / f"usr/share/{PACKAGE}/release.txt"
        self.assertFalse(payload.exists())
        self.assert_customer_state()
        absent = self.run_command([
            "dpkg-query", f"--admindir={self.admindir}", "--show",
            "--showformat=${Status}|${Version}", PACKAGE,
        ], expected=1)
        self.assertEqual(absent.stdout, "")
        package = self.build_initial_interruption_fixture(marker)
        command = ["dpkg", "--force-not-root", "--force-script-chrootless",
                   f"--admindir={self.admindir}", f"--instdir={self.instdir}",
                   f"--log={self.log}", "--install", str(package)]
        env = {**os.environ, "LC_ALL": "C", "PATH": "/usr/sbin:/sbin:/usr/bin:/bin"}
        # Isolate cleanup to this lab process group; configured dpkg hooks may also
        # run, so execute only on disposable CI hosts/VM guests.
        process = subprocess.Popen(command, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                                   env=env, start_new_session=True)
        try:
            deadline = time.monotonic() + 30
            while not marker.exists() and process.poll() is None and time.monotonic() < deadline:
                time.sleep(0.05)
            self.assertTrue(marker.is_file(), "initial-install fixture did not reach the interruption point")
            self.assertIsNone(process.poll(), "dpkg exited before initial-install interruption")
        finally:
            # Kill the dedicated process group, including the paused fixture hook and sleep.
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            process.wait(timeout=10)
        self.assertNotEqual(process.returncode, 0)
        self.assertEqual(self.installed(), "install ok half-configured|1")
        self.assert_payload_version("1")
        self.assert_customer_state()

        # Explicitly finish configuration; the inert fixture hook succeeds on retry.
        self.run_command(["dpkg", "--force-not-root", "--force-script-chrootless",
                          f"--admindir={self.admindir}", f"--instdir={self.instdir}",
                          f"--log={self.log}", "--configure", PACKAGE], expected=0)
        self.assertEqual(self.installed(), "install ok installed|1")
        self.assert_payload_version("1")
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
        # Isolate cleanup to this lab process group; configured dpkg hooks may also
        # run, so execute only on disposable CI hosts/VM guests.
        process = subprocess.Popen(command, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                                   env=env, start_new_session=True)
        try:
            deadline = time.monotonic() + 30
            while not marker.exists() and process.poll() is None and time.monotonic() < deadline:
                time.sleep(0.05)
            self.assertTrue(marker.is_file(), "fixture did not reach the interruption point")
            self.assertIsNone(process.poll(), "dpkg exited before interruption")
        finally:
            # Kill the dedicated process group, including the paused fixture hook and sleep.
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
        "test_uninterrupted_upgrade_control", "test_interrupted_upgrade_requires_explicit_reconciliation",
        "test_interrupted_initial_install_requires_explicit_reconciliation"))
