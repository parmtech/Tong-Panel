#!/usr/bin/env python3
"""Read-only structural check of an *unsigned, disposable* Tong Panel .deb.

This checks the developer package built from known source files. It does NOT
authenticate the source, verify a publisher signature, or permit installation.
"""

from __future__ import annotations

import hashlib
import os
from pathlib import Path
import stat
import subprocess
import sys
import tarfile


class UnsafeDeveloperPackage(ValueError):
    pass


def _require(ok: bool, reason: str) -> None:
    if not ok:
        raise UnsafeDeveloperPackage(reason)


def _regular_file(path: Path) -> None:
    _require(stat.S_ISREG(path.lstat().st_mode), "expected input must be a regular file")


def _digest(path: Path) -> tuple[int, bytes]:
    _regular_file(path)
    digest = hashlib.sha256()
    total = 0
    with path.open("rb") as source:
        while block := source.read(1024 * 1024):
            total += len(block)
            digest.update(block)
    return total, digest.digest()


def _memory_digest(value: bytes) -> tuple[int, bytes]:
    return len(value), hashlib.sha256(value).digest()


def _clean_path(value: str) -> str:
    if value in (".", "./"):
        return ""
    if value.startswith("./"):
        value = value[2:]
    value = value.removesuffix("/")
    _require(
        bool(value) and not value.startswith("/") and "\\" not in value
        and all(part not in ("", ".", "..") for part in value.split("/"))
        and not any(ord(char) < 32 or ord(char) == 127 for char in value),
        "non-canonical archive path",
    )
    return value


def _archive_files(package: Path, option: str, expected: dict[str, tuple[int, tuple[int, bytes]]]) -> None:
    allowed_dirs: set[str] = set()
    for filename in expected:
        path = Path(filename)
        for ancestor in path.parents:
            if str(ancestor) != ".":
                allowed_dirs.add(ancestor.as_posix())
    seen: set[str] = set()
    proc = subprocess.Popen(
        ["dpkg-deb", option, str(package)],
        stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
    )
    try:
        assert proc.stdout is not None
        with tarfile.open(fileobj=proc.stdout, mode="r|*") as archive:
            for member in archive:
                key = _clean_path(member.name)
                if not key:
                    _require(member.isdir(), "archive root is not a directory")
                    continue
                _require(key not in seen, "duplicate archive entry")
                seen.add(key)
                _require(member.uid == 0 and member.gid == 0, "archive owner must be root")
                _require(member.mode & 0o7000 == 0, "setuid/setgid/sticky archive entry")
                if member.isdir():
                    _require(key in allowed_dirs and member.mode & 0o777 == 0o755,
                             "unexpected or unsafe directory")
                else:
                    _require(member.isfile(), "links, devices and special files are forbidden")
                    _require(key in expected, "unexpected executable or data file")
                    mode, (size, digest) = expected[key]
                    _require(member.mode & 0o777 == mode and member.size == size,
                             "wrong file mode or size")
                    source = archive.extractfile(member)
                    _require(source is not None, "missing archive file stream")
                    found = hashlib.sha256()
                    remaining = size
                    while remaining:
                        block = source.read(min(remaining, 1024 * 1024))
                        _require(bool(block), "truncated archive file")
                        found.update(block)
                        remaining -= len(block)
                    _require(found.digest() == digest, "file bytes do not match reviewed fixture")
        proc.stdout.close()
        _require(proc.wait(timeout=30) == 0, "dpkg-deb archive reader failed")
        _require(set(expected) <= seen, "required archive files missing")
    except BaseException:
        if proc.poll() is None:
            proc.kill()
        if proc.stdout is not None:
            proc.stdout.close()
        proc.wait()
        raise


def verify_package(package: Path, version: str, tongd: Path, tongctl: Path, repo_root: Path) -> None:
    _regular_file(package)
    _require(0 < package.stat().st_size <= 128 * 1024 * 1024, "invalid developer package size")
    _require(version in ("0.0.0~dev.1", "0.0.0~dev.2"), "unexpected developer version")
    controls = repo_root / "packaging/debian"
    source_template = (controls / "tong-panel-dev.control").read_bytes()
    _require(source_template.count(b"Version: @VERSION@") == 1,
             "invalid package control template")
    expected_control = source_template.replace(b"Version: @VERSION@", ("Version: " + version).encode())
    control = {
        "control": (0o644, _memory_digest(expected_control)),
        **{
            name: (0o755, _digest(controls / name))
            for name in ("postinst", "prerm", "postrm")
        },
    }
    data = {
        "usr/libexec/tong-panel-dev/tongd": (0o755, _digest(tongd)),
        "usr/bin/tongctl": (0o755, _digest(tongctl)),
        "usr/lib/systemd/system/tong-panel-dev.service": (0o644, _digest(controls / "tong-panel-dev.service")),
        "usr/share/doc/tong-panel-dev/DO_NOT_INSTALL.txt": (0o644, _digest(controls / "DO_NOT_INSTALL.txt")),
        "usr/share/doc/tong-panel-dev/copyright": (0o644, _digest(repo_root / "LICENSE")),
    }
    _archive_files(package, "--ctrl-tarfile", control)
    _archive_files(package, "--fsys-tarfile", data)


def main(argv: list[str]) -> int:
    if len(argv) != 5:
        print("usage: verify_dev_deb.py PACKAGE VERSION TONGD_BINARY TONGCTL_BINARY", file=sys.stderr)
        return 2
    try:
        root = Path(__file__).resolve().parent.parent
        verify_package(Path(argv[1]), argv[2], Path(argv[3]), Path(argv[4]), root)
    except (OSError, ValueError, tarfile.TarError, subprocess.SubprocessError) as exc:
        print(f"developer package structure rejected: {type(exc).__name__}", file=sys.stderr)
        return 1
    print("developer .deb has only reviewed files and lifecycle hooks (UNSIGNED; NOT INSTALL AUTHORIZATION)")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
