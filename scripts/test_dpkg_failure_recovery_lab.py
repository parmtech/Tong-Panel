"""Unprivileged dpkg failure/recovery lab with an isolated package database.

Only constructed, inert fixture packages are installed inside a private
TemporaryDirectory root. System-level dpkg hooks may still run: use this only
on disposable CI runners/guests, never on a real server.
"""

import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch


PACKAGE = "tong-dpkg-failure-fixture"


@unittest.skipUnless(sys.platform.startswith("linux") and shutil.which("dpkg") and shutil.which("dpkg-deb") and shutil.which("dpkg-query"),
                     "requires Linux dpkg and dpkg-deb")
class DpkgFailureRecoveryLab(unittest.TestCase):
    def setUp(self):
        self.assertNotEqual(os.geteuid(), 0, "the dpkg fixture must run without root privileges")
        self.assertEqual(os.environ.get("TONG_DISPOSABLE_DPKG_LAB"), "1",
                         "explicit disposable-lab opt-in is required")
        self.directory = tempfile.TemporaryDirectory(prefix="tong-dpkg-recovery-")
        self.addCleanup(self.directory.cleanup)
        self.sandbox = Path(self.directory.name)
        self.admindir = self.sandbox / "private-dpkg"
        self.admindir.mkdir()
        (self.admindir / "updates").mkdir()
        (self.admindir / "status").write_text("", encoding="ascii")
        self.instdir = self.sandbox / "private-root"
        self.instdir.mkdir()
        self.log = self.sandbox / "private-dpkg.log"
        self.data = self.instdir / "var/lib/tong-fault-fixture/customer-marker"
        self.data.parent.mkdir(parents=True)
        self.data.write_text("preserve-this-private-test-state\n", encoding="ascii")
        self.v1 = self.build_fixture("1", fault=None)
        self.v2 = self.build_fixture("2", fault=None)

    def build_fixture(self, version, fault, *, bad_prerm=False):
        stage = self.sandbox / f"stage-{version}-{fault or 'normal'}-{int(bad_prerm)}"
        control = stage / "DEBIAN"
        control.mkdir(parents=True)
        (control / "control").write_text(
            f"Package: {PACKAGE}\nVersion: {version}\nArchitecture: all\n"
            "Maintainer: Synthetic test only\nDescription: inert upgrade interruption fixture\n",
            encoding="ascii",
        )
        for hook in ("preinst", "postinst", "prerm", "postrm"):
            should_fail = (hook == fault or (hook == "prerm" and bad_prerm))
            content = "#!/bin/sh\n# inert, no host changes\nexit " + ("73" if should_fail else "0") + "\n"
            hook_file = control / hook
            hook_file.write_text(content, encoding="ascii")
            hook_file.chmod(0o755)
        payload = stage / f"usr/share/{PACKAGE}/release.txt"
        payload.parent.mkdir(parents=True)
        payload.write_text(version + "\n", encoding="ascii")
        package = self.sandbox / f"fixture-{version}-{fault or 'normal'}-{int(bad_prerm)}.deb"
        self.run_command(["dpkg-deb", "--build", "--root-owner-group", str(stage), str(package)], expected=0)
        return package

    def run_command(self, arguments, expected):
        result = subprocess.run(arguments, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                text=True, timeout=30, check=False,
                                env={**os.environ, "LC_ALL": "C",
                                     # dpkg requires these standard host tools even when
                                     # an unprivileged guest PATH omits sbin directories.
                                     "PATH": "/usr/sbin:/sbin:/usr/bin:/bin"})
        if expected == 0:
            self.assertEqual(result.returncode, 0, (arguments, result.stdout, result.stderr))
        else:
            self.assertNotEqual(result.returncode, 0, (arguments, result.stdout, result.stderr))
        return result

    def install(self, package, *, expected=0):
        return self.run_command([
            "dpkg", "--force-not-root", "--force-script-chrootless",
            f"--admindir={self.admindir}", f"--instdir={self.instdir}",
            f"--log={self.log}", "--install", str(package),
        ], expected=expected)

    def installed(self):
        result = self.run_command([
            "dpkg-query", f"--admindir={self.admindir}", "--show",
            "--showformat=${Status}|${Version}", PACKAGE,
        ], expected=0)
        return result.stdout.strip()

    def assert_customer_state(self):
        self.assertEqual(self.data.read_text(encoding="ascii"), "preserve-this-private-test-state\n")

    def assert_payload_version(self, version):
        self.assertEqual((self.instdir / f"usr/share/{PACKAGE}/release.txt").read_text(encoding="ascii"), version + "\n")

    def test_guest_path_without_sbin_still_uses_required_dpkg_helpers(self):
        # Reproduce clean-guest ci account PATH without sbin directories.
        # Each dpkg child gets a known system-tool PATH; no real host install.
        with patch.dict(os.environ, {"PATH": "/usr/bin:/bin"}):
            self.install(self.v1)
            self.assertEqual(self.installed(), "install ok installed|1")
            self.assert_payload_version("1")
            self.assert_customer_state()

    def test_failing_preinst_leaves_previous_version_installed(self):
        self.install(self.v1)
        self.install(self.build_fixture("2", "preinst"), expected=1)
        self.assertEqual(self.installed(), "install ok installed|1")
        self.assert_payload_version("1")
        self.assert_customer_state()
        # A failed upgrade must not be interpreted as a successful version 2 install.

    def test_failing_postinst_requires_explicit_recovery(self):
        self.install(self.v1)
        self.install(self.build_fixture("2", "postinst"), expected=1)
        self.assertEqual(self.installed(), "install ok half-configured|2")
        self.assert_payload_version("2")  # unpacked is NOT configured or healthy
        self.assert_customer_state()
        self.install(self.v1)  # explicit, local recovery; not an automatic rollback
        self.assertEqual(self.installed(), "install ok installed|1")
        self.assert_payload_version("1")
        self.assert_customer_state()

    def test_untrusted_or_corrupt_archive_fails_before_unpack(self):
        self.install(self.v1)
        corrupt = self.sandbox / "not-an-archive.deb"
        corrupt.write_bytes(b"not a signed or valid Debian package\n")
        self.install(corrupt, expected=1)
        self.assertEqual(self.installed(), "install ok installed|1")
        self.assert_payload_version("1")
        self.assert_customer_state()

    def test_old_prerm_failure_may_fall_back_to_new_hook(self):
        initial = self.build_fixture("1", None, bad_prerm=True)
        self.install(initial)
        result = self.install(self.v2)
        self.assertRegex(result.stderr, r"dpkg: trying (?:prerm maintainer )?script from the new ")
        self.assertEqual(self.installed(), "install ok installed|2")
        self.assert_payload_version("2")
        self.assert_customer_state()
        # dpkg did not fail closed: an approved installer must verify actual
        # hook/service effects, not infer safety from exit status zero.


if __name__ == "__main__":
    unittest.main()
