"""Run disposable dpkg labs without allowing missing dependencies or skipped tests."""
import os
from pathlib import Path
import shutil
import sys
import unittest

EXPECTED = {"failure": ("test_dpkg_failure_recovery_lab.py", 5),
            "crash": ("test_dpkg_crash_recovery_lab.py", 2)}


def run(name):
    if name not in EXPECTED:
        raise ValueError("expected failure or crash lab")
    if (not sys.platform.startswith("linux") or not hasattr(os, "geteuid")
            or os.geteuid() == 0 or os.environ.get("TONG_DISPOSABLE_DPKG_LAB") != "1"):
        raise RuntimeError("dpkg labs require non-root Linux and disposable opt-in")
    missing = [tool for tool in ("dpkg", "dpkg-deb", "dpkg-query") if not shutil.which(tool)]
    if missing:
        raise RuntimeError("missing required dpkg tools: " + ", ".join(missing))
    pattern, expected = EXPECTED[name]
    suite = unittest.defaultTestLoader.discover(str(Path(__file__).resolve().parent), pattern=pattern)
    if suite.countTestCases() != expected:
        raise RuntimeError(f"expected {expected} {name} tests, discovered {suite.countTestCases()}")
    result = unittest.TextTestRunner(verbosity=2).run(suite)
    return result.wasSuccessful() and result.testsRun == expected and not result.skipped


if __name__ == "__main__":
    if len(sys.argv) != 2:
        sys.exit("usage: run_dpkg_labs.py failure|crash")
    sys.exit(0 if run(sys.argv[1]) else 1)
