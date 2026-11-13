"""Synthetic export-site process round trip on non-root Linux in private temp."""
import os
from pathlib import Path
import random
import subprocess
import sys
import tarfile
import tempfile


def main():
    if len(sys.argv) != 2 or not sys.platform.startswith("linux") or os.geteuid() == 0:
        raise SystemExit("requires one tongbackup executable on non-root Linux")
    binary = str(Path(sys.argv[1]).resolve(strict=True))
    marker = b"synthetic-site-secret-marker"
    key_bytes = b"K" * 32
    with tempfile.TemporaryDirectory(prefix="tong-export-cli-") as temporary:
        root = Path(temporary)
        source, output = root / "source", root / "output"
        (source / "site" / "nested").mkdir(parents=True, mode=0o700)
        (source / "site" / "empty").mkdir(mode=0o700)
        output.mkdir(mode=0o700)
        data = random.Random(42).randbytes(140000)
        (source / "site" / "nested" / "data.bin").write_bytes(data)
        (source / "site" / "index.html").write_bytes(marker)
        key, wrong = root / "key", root / "wrong"
        key.write_bytes(key_bytes); key.chmod(0o600)
        wrong.write_bytes(b"W" * 32); wrong.chmod(0o600)

        def run(args, data_in=b"", success=True):
            result = subprocess.run([binary, *args], input=data_in, stdout=subprocess.PIPE,
                                    stderr=subprocess.PIPE, timeout=30, check=False)
            if (result.returncode == 0) != success:
                raise AssertionError("unexpected CLI exit for " + args[0])
            if result.stdout or (success and result.stderr):
                raise AssertionError("unexpected CLI output for " + args[0])
            if any(secret in result.stderr for secret in (marker, key_bytes, str(root).encode())):
                raise AssertionError("sensitive fixture data appeared in CLI output")

        artifact = output / "site.tpbk"
        args = ["export-site", "--tenant", "tenant-a", "--key-file", str(key),
                "--root", str(source), "--site", "site", "--output", str(artifact), "--source-quiesced"]
        run(args)
        sealed = artifact.read_bytes()
        if len(sealed) <= 65536 or marker in sealed or artifact.stat().st_mode & 0o077:
            raise AssertionError("ciphertext fixture was not private or multi-record")
        run(["verify", "--tenant", "tenant-a", "--key-file", str(key)], sealed)
        archive = output / "restored.tar.gz"
        run(["decrypt", "--tenant", "tenant-a", "--key-file", str(key), "--output", str(archive)], sealed)
        with tarfile.open(archive, "r:gz") as tar:
            members = {item.name.rstrip("/"): item for item in tar.getmembers()}
            if set(members) != {"empty", "nested", "nested/data.bin", "index.html"}:
                raise AssertionError("exported archive inventory differs from source")
            if not members["empty"].isdir() or not members["nested"].isdir():
                raise AssertionError("exported directories missing")
            if tar.extractfile(members["nested/data.bin"]).read() != data or tar.extractfile(members["index.html"]).read() != marker:
                raise AssertionError("exported file contents differ from source")
        for tenant, key_path in (("tenant-b", key), ("tenant-a", wrong)):
            target = output / "denied.tar.gz"
            run(["decrypt", "--tenant", tenant, "--key-file", str(key_path), "--output", str(target)], sealed, False)
            if target.exists():
                raise AssertionError("denied decrypt created an output")
        damaged = bytearray(sealed); damaged[len(damaged) // 2] ^= 1
        run(["verify", "--tenant", "tenant-a", "--key-file", str(key)], bytes(damaged), False)
        for label, bad in (("damaged", bytes(damaged)), ("truncated", sealed[:-1])):
            target = output / (label + ".tar.gz")
            run(["decrypt", "--tenant", "tenant-a", "--key-file", str(key), "--output", str(target)], bad, False)
            if target.exists() or any(p.name.startswith(".tong-backup-stage-") for p in output.iterdir()):
                raise AssertionError("invalid decrypt published or left a plaintext stage")
            if artifact.read_bytes() != sealed:
                raise AssertionError("invalid decrypt changed the source artifact")
        restored_again = output / "restored-again.tar.gz"
        run(["decrypt", "--tenant", "tenant-a", "--key-file", str(key), "--output", str(restored_again)], sealed)
        if restored_again.read_bytes() != archive.read_bytes():
            raise AssertionError("valid decrypt after denial differed")
        run(args[:-1], success=False)  # missing explicit quiescence acknowledgement
        run(args, success=False)  # existing artifact must never be replaced
        if artifact.read_bytes() != sealed:
            raise AssertionError("existing artifact changed")
        inside = source / "site" / "bad.tpbk"
        run([*args[:-3], "--output", str(inside), "--source-quiesced"], success=False)
        if inside.exists():
            raise AssertionError("export created output inside source root")
        inner_key = source / "site" / "inner.key"
        inner_key.write_bytes(key_bytes); inner_key.chmod(0o600)
        inner_key_output = output / "inner-key.tpbk"
        run(["export-site", "--tenant", "tenant-a", "--key-file", str(inner_key),
             "--root", str(source), "--site", "site", "--output", str(inner_key_output),
             "--source-quiesced"], success=False)
        if inner_key_output.exists():
            raise AssertionError("export accepted a key inside source root")
        inner_key.unlink()
        (source / "site" / "unsafe-link").symlink_to(root / "key")
        link_output = output / "unsafe.tpbk"
        run([*args[:-3], "--output", str(link_output), "--source-quiesced"], success=False)
        if link_output.exists():
            raise AssertionError("export accepted a source symlink")
        if sorted(p.name for p in output.iterdir()) != ["restored-again.tar.gz", "restored.tar.gz", "site.tpbk"]:
            raise AssertionError("unexpected output or staging file remains")
    print("synthetic export-site CLI round trip and negative cases passed")


if __name__ == "__main__":
    main()
