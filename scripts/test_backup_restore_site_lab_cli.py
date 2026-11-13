"""Synthetic non-root Linux export-to-restore process test under /tmp only."""
import os
from pathlib import Path
import random
import subprocess
import sys
import tempfile


def main():
    if len(sys.argv) != 2 or not sys.platform.startswith("linux") or os.geteuid() == 0:
        raise SystemExit("requires one tongbackup executable on non-root Linux")
    binary = str(Path(sys.argv[1]).resolve(strict=True))
    marker = b"synthetic-private-site-marker"
    key_bytes = b"K" * 32
    with tempfile.TemporaryDirectory(prefix="tong-restore-site-cli-", dir="/tmp") as temporary:
        root = Path(temporary)
        source, output = root / "source", root / "output"
        (source / "site" / "nested").mkdir(parents=True, mode=0o700)
        (source / "site" / "empty").mkdir(mode=0o700)
        output.mkdir(mode=0o700)
        data = random.Random(72).randbytes(140000)
        (source / "site" / "nested" / "data.bin").write_bytes(data)
        (source / "site" / "index.html").write_bytes(marker)
        key, wrong = root / "key", root / "wrong"
        key.write_bytes(key_bytes); key.chmod(0o600)
        wrong.write_bytes(b"W" * 32); wrong.chmod(0o600)

        def run(args, data_in=b"", success=True, enabled=True):
            env = dict(os.environ)
            env.pop("TONG_DISPOSABLE_SITE_RESTORE", None)
            if enabled:
                env["TONG_DISPOSABLE_SITE_RESTORE"] = "1"
            result = subprocess.run([binary, *args], input=data_in, stdout=subprocess.PIPE,
                                    stderr=subprocess.PIPE, timeout=30, check=False, env=env)
            if (result.returncode == 0) != success:
                raise AssertionError("unexpected CLI exit for " + args[0])
            if result.stdout or (success and result.stderr):
                raise AssertionError("unexpected CLI output for " + args[0])
            if any(secret in result.stderr for secret in (marker, key_bytes, str(root).encode())):
                raise AssertionError("sensitive fixture data appeared in CLI output")

        artifact = output / "site.tpbk"
        run(["export-site", "--tenant", "tenant-a", "--key-file", str(key), "--root", str(source),
             "--site", "site", "--output", str(artifact), "--source-quiesced"])
        sealed = artifact.read_bytes()
        if marker in sealed or len(sealed) <= 65536:
            raise AssertionError("fixture must be multi-record ciphertext without plaintext marker")

        def lab(name):
            # The command must atomically create this absent, named root.
            return root / ("tong-restore-lab-" + name)

        def args(directory, tenant="tenant-a", key_path=key):
            return ["restore-site-lab", "--tenant", tenant, "--key-file", str(key_path), "--root", str(directory),
                    "--destination", "restored", "--disposable-only"]

        damaged = bytearray(sealed); damaged[len(damaged) // 2] ^= 1
        cases = [
            ("wrong-tenant", "tenant-b", key, sealed, True, True),
            ("wrong-key", "tenant-a", wrong, sealed, True, True),
            ("tampered", "tenant-a", key, bytes(damaged), True, True),
            ("truncated", "tenant-a", key, sealed[:-1], True, True),
            ("no-env", "tenant-a", key, sealed, False, True),
            ("no-flag", "tenant-a", key, sealed, True, False),
        ]
        for name, tenant, key_path, content, enabled, flag in cases:
            directory = lab(name)
            command = args(directory, tenant, key_path)
            if not flag:
                command = command[:-1]
            run(command, content, success=False, enabled=enabled)
            # Authentication failures retain the fresh root for inspection;
            # missing opt-in fails before root creation. No ambiguous rename is
            # injected or asserted by this process test.
            if enabled and flag:
                if not directory.is_dir() or list(directory.iterdir()):
                    raise AssertionError("authentication denial did not retain an empty lab root")
            elif directory.exists():
                raise AssertionError("missing opt-in created a lab root")
        nonempty = lab("nonempty")
        nonempty.mkdir(mode=0o700)
        (nonempty / "sentinel").write_bytes(b"unchanged")
        run(args(nonempty), sealed, success=False)
        if sorted(p.name for p in nonempty.iterdir()) != ["sentinel"] or (nonempty / "sentinel").read_bytes() != b"unchanged":
            raise AssertionError("nonempty root changed")

        target = lab("valid")
        run(args(target), sealed)
        restored = target / "restored"
        if (restored / "index.html").read_bytes() != marker or (restored / "nested" / "data.bin").read_bytes() != data:
            raise AssertionError("restored site differs from source")
        if not (restored / "empty").is_dir() or sorted(p.name for p in target.iterdir()) != ["restored"]:
            raise AssertionError("unexpected restored tree or stage")
        if artifact.read_bytes() != sealed:
            raise AssertionError("source ciphertext changed")
    print("synthetic export-to-restore-site-lab CLI checks passed")


if __name__ == "__main__":
    main()
