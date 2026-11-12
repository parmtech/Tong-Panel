"""Local synthetic-data restic drill. No daemon integration or remote backends."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import secrets
import subprocess
import tempfile

PINNED_EXE_SHA256 = "b0dd1fd21eea5d8fe1325f55f7118213c21f36de8a261e04c0624a5ab9fd7830"


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def run_drill(executable):
    executable = executable.resolve(strict=True)
    if digest(executable) != PINNED_EXE_SHA256:
        raise RuntimeError("executable does not match verified restic 0.19.1 Windows amd64 pin")
    # TemporaryDirectory owns every fixture, repository, key and restore path.
    with tempfile.TemporaryDirectory(prefix="tong-restic-fixture-") as temporary:
        root = Path(temporary)
        source = root / "source"
        tenant = source / "tenant-a"
        sibling = source / "tenant-b"
        (tenant / "nested").mkdir(parents=True)
        sibling.mkdir()
        (tenant / "hello.txt").write_bytes(b"Tong Panel synthetic restore fixture\n")
        (tenant / "nested" / "binary.bin").write_bytes(bytes(range(256)) * 256)
        (tenant / "empty.txt").write_bytes(b"")
        (sibling / "must-not-restore.txt").write_bytes(b"synthetic other-tenant fixture")
        expected = {str(p.relative_to(tenant)): digest(p) for p in tenant.rglob("*") if p.is_file()}
        # Child processes inherit no cloud/storage credentials or RESTIC_* settings.
        environment = {k: os.environ[k] for k in ("SystemRoot", "TEMP", "TMP") if k in os.environ}
        environment["RESTIC_PASSWORD"] = secrets.token_urlsafe(48)
        repository = root / "repository"
        target = root / "new-host-restore"

        def invoke(arguments, *, password=None, expect_success=True):
            child_env = environment.copy()
            if password is not None:
                child_env["RESTIC_PASSWORD"] = password
            result = subprocess.run(
                [str(executable), "--repo", str(repository), "--no-cache", *arguments],
                cwd=source, env=child_env, capture_output=True, text=True,
                timeout=90, check=False, shell=False,
            )
            if expect_success and result.returncode != 0:
                # Do not echo arbitrary tool output or credential-bearing environment.
                raise RuntimeError(f"restic {arguments[0]} failed with exit {result.returncode}")
            return result

        invoke(["init", "--repository-version", "2"])
        invoke(["backup", "--json", "--host", "tong-synthetic-drill", "--tag", "tenant-a-fixture", "--", "tenant-a"])
        snapshots = json.loads(invoke(["snapshots", "--json"]).stdout)
        if len(snapshots) != 1 or snapshots[0]["tags"] != ["tenant-a-fixture"]:
            raise RuntimeError("unexpected snapshot identity/count")
        snapshot = snapshots[0]["id"]
        if len(snapshot) != 64 or any(c not in "0123456789abcdef" for c in snapshot):
            raise RuntimeError("invalid snapshot identifier")
        # Wrong password must fail before restoring; no key is persisted by this script.
        wrong_password = invoke(["snapshots", "--json"], password=secrets.token_urlsafe(48), expect_success=False)
        if wrong_password.returncode != 12:
            raise RuntimeError(f"wrong password did not return documented exit 12: {wrong_password.returncode}")
        invoke(["check", "--read-data"])
        invoke(["restore", snapshot, "--target", str(target), "--verify"])
        restored_roots = [p for p in target.rglob("tenant-a") if p.is_dir()]
        if len(restored_roots) != 1:
            raise RuntimeError("unexpected restored tenant root count")
        restored = restored_roots[0]
        actual = {str(p.relative_to(restored)): digest(p) for p in restored.rglob("*") if p.is_file()}
        if actual != expected:
            raise RuntimeError("restored fixture file manifest/hash mismatch")
        if any(p.is_file() and not p.is_relative_to(restored) for p in target.rglob("*")):
            raise RuntimeError("restore included a file outside selected tenant fixture")
        if list(target.rglob("must-not-restore.txt")):
            raise RuntimeError("sibling tenant fixture was restored")
        print(json.dumps({
            "restic": "0.19.1", "platform": "windows/amd64",
            "result": "PASS", "verified_files": len(expected),
            "checks": ["pinned executable hash", "encrypted local repository init",
                       "one tagged snapshot", "wrong password rejected (exit 12)",
                       "full repository read-data check", "restore verification",
                       "exact file hashes", "sibling tenant excluded"],
        }, indent=2))
    # TemporaryDirectory removes synthetic data, repository and in-memory-only key scope.


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--restic", required=True, type=Path, help="verified pinned Windows amd64 executable")
    run_drill(parser.parse_args().restic)
