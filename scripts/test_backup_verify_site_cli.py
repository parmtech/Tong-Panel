"""Synthetic verify-site process checks on non-root Linux, in private temp."""
import io
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile


def main():
    if len(sys.argv) != 2 or not sys.platform.startswith("linux") or os.geteuid() == 0:
        raise SystemExit("requires one tongbackup executable on non-root Linux")
    binary = str(Path(sys.argv[1]).resolve(strict=True))
    marker = b"synthetic-site-plaintext-marker"
    key_bytes = b"K" * 32
    with tempfile.TemporaryDirectory(prefix="tong-verify-site-") as temporary:
        root = Path(temporary)
        key, wrong = root / "key", root / "wrong"
        key.write_bytes(key_bytes); key.chmod(0o600)
        wrong.write_bytes(b"W" * 32); wrong.chmod(0o600)
        sentinel = root / "sentinel"
        sentinel.write_bytes(b"unchanged")

        def run(args, data, success):
            result = subprocess.run([binary, *args], input=data, stdout=subprocess.PIPE,
                                    stderr=subprocess.PIPE, timeout=30, check=False)
            if (result.returncode == 0) != success:
                raise AssertionError("unexpected CLI exit for " + args[0])
            if result.stdout or (success and result.stderr):
                raise AssertionError("unexpected CLI output for " + args[0])
            if any(secret in result.stderr for secret in (marker, key_bytes, str(root).encode())):
                raise AssertionError("sensitive fixture data appeared in CLI output")

        def archive(kind):
            buf = io.BytesIO()
            with tarfile.open(fileobj=buf, mode="w:gz") as tar:
                if kind == "symlink":
                    item = tarfile.TarInfo("link")
                    item.type = tarfile.SYMTYPE
                    item.linkname = "../outside"
                    tar.addfile(item)
                else:
                    item = tarfile.TarInfo("../escape" if kind == "traversal" else "index.html")
                    item.size = len(marker)
                    tar.addfile(item, io.BytesIO(marker))
            return buf.getvalue()

        def encrypted(kind):
            target = root / (kind + ".tpbk")
            run(["encrypt", "--tenant", "tenant-a", "--key-file", str(key), "--output", str(target)], archive(kind), True)
            return target.read_bytes()

        valid = encrypted("valid")
        traversal = encrypted("traversal")
        symlink = encrypted("symlink")
        baseline = sorted(p.name for p in root.iterdir())
        args = ["verify-site", "--tenant", "tenant-a", "--key-file", str(key)]
        run(args, valid, True)
        run(["verify-site", "--tenant", "tenant-b", "--key-file", str(key)], valid, False)
        run(["verify-site", "--tenant", "tenant-a", "--key-file", str(wrong)], valid, False)
        damaged = bytearray(valid); damaged[-1] ^= 1
        for bad in (bytes(damaged), valid[:-1], valid + b"x", traversal, symlink):
            run(args, bad, False)
            if sorted(p.name for p in root.iterdir()) != baseline or sentinel.read_bytes() != b"unchanged":
                raise AssertionError("verify-site modified the fixture directory")
        run(args, valid, True)
        if sorted(p.name for p in root.iterdir()) != baseline or sentinel.read_bytes() != b"unchanged":
            raise AssertionError("valid verify-site modified the fixture directory")
    print("synthetic verify-site CLI checks passed")


if __name__ == "__main__":
    main()
