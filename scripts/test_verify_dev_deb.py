"""Real .deb fixtures: reject hidden install-time hooks, changed bytes and paths."""

import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

from verify_dev_deb import UnsafeDeveloperPackage, verify_package


@unittest.skipUnless(shutil.which("dpkg-deb") and os.uname().machine == "x86_64", "amd64 dpkg-deb fixture required")
class DeveloperPackageStructureTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.repo = self.root / "repo"
        control = self.repo / "packaging/debian"
        control.mkdir(parents=True)
        (control / "tong-panel-dev.control").write_text(
            "Package: tong-panel-dev\nVersion: @VERSION@\nArchitecture: amd64\nMaintainer: Fixture\nDescription: unsigned fixture\n",
            encoding="utf-8",
        )
        for name in ("postinst", "prerm", "postrm"):
            (control / name).write_text("#!/bin/sh\nexit 0\n", encoding="utf-8")
        (control / "tong-panel-dev.service").write_text("[Service]\nExecStart=/stub\n", encoding="utf-8")
        (control / "DO_NOT_INSTALL.txt").write_text("UNSIGNED DEVELOPER PACKAGE ONLY\n", encoding="utf-8")
        (self.repo / "LICENSE").write_text("fixture license", encoding="utf-8")
        self.tongd = self.root / "tongd"
        self.tongctl = self.root / "tongctl"
        self.tongd.write_bytes(b"synthetic tongd, not a real executable\n")
        self.tongctl.write_bytes(b"synthetic tongctl, not a real executable\n")
        self.staging = self.root / "staging"
        (self.staging / "DEBIAN").mkdir(parents=True)
        for name in ("postinst", "prerm", "postrm"):
            dest = self.staging / "DEBIAN" / name
            shutil.copyfile(control / name, dest)
            dest.chmod(0o755)
        text = (control / "tong-panel-dev.control").read_text().replace("@VERSION@", "0.0.0~dev.1")
        (self.staging / "DEBIAN/control").write_text(text, encoding="utf-8")
        self.data_paths = {
            "usr/libexec/tong-panel-dev/tongd": self.tongd,
            "usr/bin/tongctl": self.tongctl,
            "usr/lib/systemd/system/tong-panel-dev.service": control / "tong-panel-dev.service",
            "usr/share/doc/tong-panel-dev/DO_NOT_INSTALL.txt": control / "DO_NOT_INSTALL.txt",
            "usr/share/doc/tong-panel-dev/copyright": self.repo / "LICENSE",
        }
        for path, source in self.data_paths.items():
            dest = self.staging / path
            dest.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(source, dest)
            dest.chmod(0o755 if path.endswith(("/tongd", "/tongctl")) else 0o644)
        self.package = self.root / "fixture.deb"
        self.build()

    def build(self):
        subprocess.run(["dpkg-deb", "--build", "--root-owner-group", str(self.staging), str(self.package)],
                       check=True, capture_output=True, text=True, timeout=30)

    def verify(self):
        verify_package(self.package, "0.0.0~dev.1", self.tongd, self.tongctl, self.repo)

    def test_exact_developer_fixture_is_accepted(self):
        self.verify()

    def test_unreviewed_preinst_hook_is_rejected(self):
        (self.staging / "DEBIAN/preinst").write_text("#!/bin/sh\nexit 0\n")
        (self.staging / "DEBIAN/preinst").chmod(0o755)
        self.build()
        with self.assertRaises(UnsafeDeveloperPackage):
            self.verify()

    def test_tampered_lifecycle_script_is_rejected(self):
        (self.staging / "DEBIAN/postinst").write_text("#!/bin/sh\necho unexpected\n")
        self.build()
        with self.assertRaises(UnsafeDeveloperPackage):
            self.verify()

    def test_extra_payload_file_is_rejected(self):
        (self.staging / "usr/share/doc/tong-panel-dev/extra").write_text("unexpected")
        self.build()
        with self.assertRaises(UnsafeDeveloperPackage):
            self.verify()

    def test_replaced_binary_is_rejected(self):
        (self.staging / "usr/bin/tongctl").write_bytes(b"unexpected second executable")
        self.build()
        with self.assertRaises(UnsafeDeveloperPackage):
            self.verify()

    def test_symlink_payload_is_rejected(self):
        path = self.staging / "usr/bin/tongctl"
        path.unlink()
        path.symlink_to("/bin/sh")
        self.build()
        with self.assertRaises(UnsafeDeveloperPackage):
            self.verify()

    def test_setuid_payload_is_rejected(self):
        (self.staging / "usr/libexec/tong-panel-dev/tongd").chmod(0o4755)
        self.build()
        with self.assertRaises(UnsafeDeveloperPackage):
            self.verify()

    def test_wrong_expected_version_and_missing_file_are_rejected(self):
        with self.assertRaises(UnsafeDeveloperPackage):
            verify_package(self.package, "0.0.0~dev.2", self.tongd, self.tongctl, self.repo)
        (self.staging / "usr/share/doc/tong-panel-dev/copyright").unlink()
        self.build()
        with self.assertRaises(UnsafeDeveloperPackage):
            self.verify()

    def test_modified_source_expected_binary_is_rejected(self):
        self.tongd.write_bytes(b"different source binary")
        with self.assertRaises(UnsafeDeveloperPackage):
            self.verify()


if __name__ == "__main__":
    unittest.main()
