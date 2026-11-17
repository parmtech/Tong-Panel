"""Synthetic non-root Linux inventory process checks under private /tmp only."""
import fcntl
import os
from pathlib import Path
import subprocess
import sys
import tempfile


def main():
    if len(sys.argv) != 2 or not sys.platform.startswith("linux") or os.geteuid() == 0:
        raise SystemExit("requires one tongbackup executable on non-root Linux")
    binary = str(Path(sys.argv[1]).resolve(strict=True))
    marker = b"synthetic-stage-payload-secret"
    with tempfile.TemporaryDirectory(prefix="tong-inspect-cli-", dir="/tmp") as temporary:
        parent = Path(temporary)
        root = parent / "tong-restore-lab-inventory"
        root.mkdir(mode=0o700)
        args = ["inspect-restore-lab", "--root", str(root), "--disposable-only"]

        def run(command, count=None, enabled=True):
            env = dict(os.environ)
            env.pop("TONG_DISPOSABLE_SITE_RESTORE", None)
            if enabled:
                env["TONG_DISPOSABLE_SITE_RESTORE"] = "1"
            result = subprocess.run([binary, *command], stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                    timeout=30, check=False, env=env)
            if count is None:
                if result.returncode == 0 or result.stdout:
                    raise AssertionError("unsafe or unacknowledged inventory succeeded")
            else:
                expected = ("Archive stages observed: " + str(count) + "\n"
                            "Snapshot includes active stages; count does not establish publication or permit deletion or quota release.\n").encode()
                if result.returncode != 0 or result.stderr or result.stdout != expected:
                    raise AssertionError("unexpected inventory response")
            if any(secret in result.stdout or secret in result.stderr for secret in (marker, str(parent).encode())):
                raise AssertionError("inventory disclosed fixture content or path")

        run(args, 0)
        stage = root / ".tong-archive-stage-0123456789abcdef0123456789abcdef"
        stage.mkdir(mode=0o700)
        payload = stage / "payload"
        payload.write_bytes(marker)
        before = (stage.stat().st_mode, stage.stat().st_ino, payload.stat().st_ino, payload.stat().st_mtime_ns)
        fd = os.open(stage, os.O_RDONLY | os.O_DIRECTORY)
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
            run(args, 1)  # A count does not prove a stage is abandoned.
        finally:
            fcntl.flock(fd, fcntl.LOCK_UN)
            os.close(fd)
        run(args, 1)
        if payload.read_bytes() != marker or before != (stage.stat().st_mode, stage.stat().st_ino, payload.stat().st_ino, payload.stat().st_mtime_ns):
            raise AssertionError("inventory changed the retained stage")
        run(args, enabled=False)
        run(args[:-1])
        bad = root / ".tong-archive-stage-not-hex"
        bad.mkdir(mode=0o700)
        run(args)
        bad.rmdir()
        stage.chmod(0o755)
        run(args)
        stage.chmod(0o700)
        link = root / ".tong-archive-stage-1123456789abcdef0123456789abcdef"
        link.symlink_to(stage, target_is_directory=True)
        run(args)
        if payload.read_bytes() != marker or not link.is_symlink():
            raise AssertionError("failed inventory changed the fixture")
    print("synthetic inspect-restore-lab CLI checks passed")


if __name__ == "__main__":
    main()
