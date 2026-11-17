"""Process-level synthetic backup verification; Linux, non-root, private temp only."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile


def main():
    if len(sys.argv) != 2 or not sys.platform.startswith("linux") or os.geteuid() == 0:
        raise SystemExit("requires one tongbackup executable on non-root Linux")
    binary = str(Path(sys.argv[1]).resolve(strict=True))
    marker = b"synthetic-private-restore-marker"
    key_bytes = b"K" * 32
    with tempfile.TemporaryDirectory(prefix="tong-verify-cli-") as tmp:
        root = Path(tmp)
        key, wrong, artifact = root / "key", root / "wrong", root / "fixture.tpbk"
        key.write_bytes(key_bytes)
        wrong.write_bytes(b"W" * 32)
        key.chmod(0o600)
        wrong.chmod(0o600)

        def call(args, data, success):
            result = subprocess.run([binary, *args], input=data, stdout=subprocess.PIPE,
                                    stderr=subprocess.PIPE, timeout=30, check=False)
            if (result.returncode == 0) != success:
                raise AssertionError("unexpected CLI exit for " + args[0])
            if result.stdout or (success and result.stderr):
                raise AssertionError("unexpected output for " + args[0])
            for forbidden in (marker, key_bytes, str(root).encode()):
                if forbidden in result.stdout or forbidden in result.stderr:
                    raise AssertionError("sensitive fixture data appeared in CLI output")
            return result

        call(["encrypt", "--tenant", "tenant-a", "--key-file", str(key),
              "--output", str(artifact)], marker, True)
        sealed = artifact.read_bytes()
        if marker in sealed:
            raise AssertionError("sealed artifact exposed plaintext")
        args = ["verify", "--tenant", "tenant-a", "--key-file", str(key)]
        call(args, sealed, True)
        call(["verify", "--tenant", "tenant-b", "--key-file", str(key)], sealed, False)
        call(["verify", "--tenant", "tenant-a", "--key-file", str(wrong)], sealed, False)
        corrupt = bytearray(sealed)
        corrupt[-1] ^= 1
        for bad in (bytes(corrupt), sealed[:-1], sealed + b"x"):
            call(args, bad, False)
        call([*args, "--tenant", "tenant-b"], sealed, False)
        call(["verify", "--tenant", "tenant-a", "--key-file", str(root / "missing")], sealed, False)
        wrong.chmod(0o644)
        call(["verify", "--tenant", "tenant-a", "--key-file", str(wrong)], sealed, False)
        wrong.chmod(0o600)
        forbidden_output = root / "must-not-exist"
        call([*args, "--output", str(forbidden_output)], sealed, False)
        if forbidden_output.exists():
            raise AssertionError("verify created an output file")
        if sorted(p.name for p in root.iterdir()) != ["fixture.tpbk", "key", "wrong"]:
            raise AssertionError("verify created an unexpected file")
    print("synthetic backup verify CLI: success and negative cases passed")


if __name__ == "__main__":
    main()
