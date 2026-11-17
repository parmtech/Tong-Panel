"""Synthetic non-root Linux export-to-restore process test under /tmp only."""
import os
import io
from pathlib import Path
import random
import subprocess
import sys
import stat
import tarfile
import tempfile


def main():
    if len(sys.argv) != 2 or not sys.platform.startswith("linux") or os.geteuid() == 0:
        raise SystemExit("requires one tongbackup executable on non-root Linux")
    binary = str(Path(sys.argv[1]).resolve(strict=True))
    marker = b"synthetic-private-site-marker"
    marker_b = b"synthetic-private-site-marker-tenant-b"
    key_bytes = b"K" * 32
    key_b_bytes = b"B" * 32
    with tempfile.TemporaryDirectory(prefix="tong-restore-site-cli-", dir="/tmp") as temporary:
        root = Path(temporary)
        source, source_b, output = root / "source", root / "source-b", root / "output"
        (source / "site" / "nested").mkdir(parents=True, mode=0o700)
        (source / "site" / "empty").mkdir(mode=0o700)
        (source_b / "site" / "nested").mkdir(parents=True, mode=0o700)
        (source_b / "site" / "empty").mkdir(mode=0o700)
        output.mkdir(mode=0o700)
        data = random.Random(72).randbytes(140000)
        data_b = random.Random(73).randbytes(140000)
        (source / "site" / "nested" / "data.bin").write_bytes(data)
        (source / "site" / "index.html").write_bytes(marker)
        (source_b / "site" / "nested" / "data.bin").write_bytes(data_b)
        (source_b / "site" / "index.html").write_bytes(marker_b)
        key, key_b, wrong = root / "key", root / "key-b", root / "wrong"
        key.write_bytes(key_bytes); key.chmod(0o600)
        key_b.write_bytes(key_b_bytes); key_b.chmod(0o600)
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
            if any(secret in result.stderr for secret in (marker, marker_b, key_bytes, key_b_bytes,
                                                          str(root).encode())):
                raise AssertionError("sensitive fixture data appeared in CLI output")

        artifact = output / "site.tpbk"
        run(["export-site", "--tenant", "tenant-a", "--key-file", str(key), "--root", str(source),
             "--site", "site", "--output", str(artifact), "--source-quiesced"])
        sealed = artifact.read_bytes()
        if marker in sealed or len(sealed) <= 65536:
            raise AssertionError("fixture must be multi-record ciphertext without plaintext marker")
        artifact_b = output / "site-tenant-b.tpbk"
        run(["export-site", "--tenant", "tenant-b", "--key-file", str(key_b), "--root", str(source_b),
             "--site", "site", "--output", str(artifact_b), "--source-quiesced"])
        sealed_b = artifact_b.read_bytes()
        if marker in sealed_b or marker_b in sealed_b or len(sealed_b) <= 65536:
            raise AssertionError("tenant-b fixture must be multi-record ciphertext without plaintext markers")

        def lab(name):
            # The command must atomically create this absent, named root.
            return root / ("tong-restore-lab-" + name)

        def args(directory, tenant="tenant-a", key_path=key):
            return ["restore-site-lab", "--tenant", tenant, "--key-file", str(key_path), "--root", str(directory),
                    "--destination", "restored", "--disposable-only"]

        def tree_snapshot(directory):
            entries = [(".", directory.lstat().st_mode, None)]
            for path in sorted(directory.rglob("*")):
                mode = path.lstat().st_mode
                if stat.S_ISLNK(mode):
                    value = os.readlink(path)
                elif stat.S_ISREG(mode):
                    value = path.read_bytes()
                else:
                    value = None
                entries.append((path.relative_to(directory).as_posix(), mode, value))
            return tuple(entries)

        def inspect_empty_inventory(directory):
            env = dict(os.environ)
            env["TONG_DISPOSABLE_SITE_RESTORE"] = "1"
            result = subprocess.run([binary, "inspect-restore-lab", "--root", str(directory), "--disposable-only"],
                                    stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=30, check=False, env=env)
            expected = (b"Archive stages observed: 0\n"
                        b"Snapshot includes active stages; count does not establish publication or permit deletion or quota release.\n")
            if result.returncode != 0 or result.stdout != expected or result.stderr:
                raise AssertionError("unexpected restore inventory response")

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
                if name == "wrong-key":
                    inspect_empty_inventory(directory)
                    if list(directory.iterdir()):
                        raise AssertionError("inventory modified denied restore root")
            elif directory.exists():
                raise AssertionError("missing opt-in created a lab root")
        nonempty = lab("nonempty")
        nonempty.mkdir(mode=0o700)
        (nonempty / "sentinel").write_bytes(b"unchanged")
        run(args(nonempty), sealed, success=False)
        if sorted(p.name for p in nonempty.iterdir()) != ["sentinel"] or (nonempty / "sentinel").read_bytes() != b"unchanged":
            raise AssertionError("nonempty root changed")

        # These archives authenticate successfully but contain unsafe entries.
        # The actual CLI must reject them without publishing or modifying a
        # sibling; no assumption is made about failed-stage cleanup.
        sibling = root / "outside-sentinel"
        sibling.write_bytes(b"untouched")
        for kind in ("traversal", "symlink"):
            raw = io.BytesIO()
            with tarfile.open(fileobj=raw, mode="w:gz") as tar:
                if kind == "traversal":
                    payload = b"unexpected"
                    entry = tarfile.TarInfo("../escape")
                    entry.size = len(payload)
                    tar.addfile(entry, io.BytesIO(payload))
                else:
                    entry = tarfile.TarInfo("link")
                    entry.type = tarfile.SYMTYPE
                    entry.linkname = "../outside-sentinel"
                    tar.addfile(entry)
            bad_artifact = output / (kind + ".tpbk")
            run(["encrypt", "--tenant", "tenant-a", "--key-file", str(key), "--output", str(bad_artifact)], raw.getvalue())
            directory = lab(kind)
            run(args(directory), bad_artifact.read_bytes(), success=False)
            if not directory.is_dir() or (directory / "restored").exists():
                raise AssertionError("unsafe archive published a destination or removed its lab root")
            if (root / "escape").exists() or (directory / "escape").exists() or sibling.read_bytes() != b"untouched":
                raise AssertionError("unsafe archive modified an outside path")

        target = lab("valid")
        run(args(target), sealed)
        restored = target / "restored"
        if (restored / "index.html").read_bytes() != marker or (restored / "nested" / "data.bin").read_bytes() != data:
            raise AssertionError("restored site differs from source")
        if not (restored / "empty").is_dir() or sorted(p.name for p in target.iterdir()) != ["restored"]:
            raise AssertionError("unexpected restored tree or stage")
        inspect_empty_inventory(target)
        if (restored / "index.html").read_bytes() != marker or (restored / "nested" / "data.bin").read_bytes() != data:
            raise AssertionError("inventory changed restored site")
        if artifact.read_bytes() != sealed:
            raise AssertionError("source ciphertext changed")

        target_b = lab("valid-tenant-b")
        run(args(target_b, "tenant-b", key_b), sealed_b)
        restored_b = target_b / "restored"
        if ((restored_b / "index.html").read_bytes() != marker_b or
                (restored_b / "nested" / "data.bin").read_bytes() != data_b):
            raise AssertionError("tenant-b restored site differs from source")
        if not (restored_b / "empty").is_dir() or sorted(p.name for p in target_b.iterdir()) != ["restored"]:
            raise AssertionError("unexpected tenant-b restored tree or stage")
        if artifact_b.read_bytes() != sealed_b:
            raise AssertionError("tenant-b source ciphertext changed")

        # A tenant-b archive paired with tenant-a's key must fail while leaving
        # both previously restored tenant roots byte-for-byte and mode unchanged.
        snapshot_a, snapshot_b = tree_snapshot(target), tree_snapshot(target_b)
        denied = lab("paired-swapped-key")
        run(args(denied, "tenant-b", key), sealed_b, success=False)
        if not denied.is_dir() or list(denied.iterdir()):
            raise AssertionError("swapped-key denial did not retain an empty lab root")
        if tree_snapshot(target) != snapshot_a or tree_snapshot(target_b) != snapshot_b:
            raise AssertionError("swapped-key denial changed a valid tenant restore root")

        # A tenant-b archive and its valid key must still be denied when the
        # request names tenant-a. This isolates tenant binding from key failure.
        denied_cross_tenant = lab("tenant-b-requested-as-a")
        run(args(denied_cross_tenant, "tenant-a", key_b), sealed_b, success=False)
        if not denied_cross_tenant.is_dir() or list(denied_cross_tenant.iterdir()):
            raise AssertionError("cross-tenant denial did not retain an empty lab root")
        if (denied_cross_tenant / "restored").exists():
            raise AssertionError("cross-tenant denial published a destination")
        if tree_snapshot(target) != snapshot_a or tree_snapshot(target_b) != snapshot_b:
            raise AssertionError("cross-tenant denial changed a valid tenant restore root")
        if artifact.read_bytes() != sealed or artifact_b.read_bytes() != sealed_b:
            raise AssertionError("cross-tenant denial changed source ciphertext")
    print("synthetic export-to-restore-site-lab CLI checks passed")


if __name__ == "__main__":
    main()
