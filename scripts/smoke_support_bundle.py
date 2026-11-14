#!/usr/bin/env python3
"""Exercise the actual CLI's read-only structured support export on a lab runner."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def unique_members(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, "duplicate support JSON member")
        result[key] = value
    return result


def main(binary):
    binary = str(Path(binary).resolve())
    canary = "tong-support-secret-canary-DO-NOT-EXPORT"
    with tempfile.TemporaryDirectory(prefix="tong-support-smoke-") as folder:
        database = Path(folder) / "must-not-be-created.db"
        environment = dict(os.environ, TONG_SUPPORT_SECRET=canary,
                           TONG_SQLITE_PATH=str(database),
                           TONG_DEV_SQLITE_DB=str(database),
                           TONG_DEV_TOKEN=canary, DATABASE_URL=canary)
        result = subprocess.run([binary, "support-bundle"], env=environment,
                                capture_output=True, timeout=15)
        require(result.returncode == 0, "support export failed")
        require(0 < len(result.stdout) <= 4096 and not result.stderr,
                "support export has unexpected size or diagnostics")
        require(canary.encode() not in result.stdout, "support export leaked environment canary")
        require(str(database).encode() not in result.stdout, "support export leaked a configured path")
        require(not database.exists(), "support export created a database")
        document = json.loads(result.stdout, object_pairs_hook=unique_members)
        require(set(document) == {"schema_version", "notice", "collection", "platform", "diagnostics"},
                "unexpected support export fields")
        require(type(document["schema_version"]) is int and document["schema_version"] == 1,
                "unsupported export schema")
        require(document["notice"] == "Development-only read-only support facts; no installation or support certification.",
                "missing development scope notice")
        collection = document["collection"]
        require(set(collection) == {"status", "code"}, "unexpected collection fields")
        require((collection["status"], collection["code"]) in {
            ("complete", "facts_collected"), ("partial", "facts_incomplete")},
            "invalid collection state")
        platform = document["platform"]
        require(set(platform) == {"os", "arch", "distribution", "version"}, "unexpected platform fields")
        for key, values in {
            "os": {"linux", "other", "unknown"},
            "arch": {"amd64", "arm64", "other", "unknown"},
            "distribution": {"ubuntu", "debian", "other", "unknown"},
            "version": {"24.04", "12", "other", "unknown"},
        }.items():
            require(platform[key] in values, "unredacted platform value")
        if platform["version"] not in {"other", "unknown"}:
            require((platform["distribution"], platform["version"]) in {
                ("ubuntu", "24.04"), ("debian", "12")}, "inconsistent platform version")
        diagnostics = document["diagnostics"]
        require(set(diagnostics) == {"os", "memory", "disk", "web_ports"},
                "unexpected diagnostic fields")
        require(diagnostics["os"] in {"lab_candidate", "outside_lab_targets", "unknown"},
                "invalid OS diagnostic")
        if diagnostics["os"] == "lab_candidate":
            require(platform["os"] == "linux" and platform["arch"] == "amd64" and
                    (platform["distribution"], platform["version"]) in {
                        ("ubuntu", "24.04"), ("debian", "12")},
                    "lab-candidate diagnostic contradicts platform facts")
        require(all(diagnostics[key] in {"adequate", "low", "unknown"} for key in ["memory", "disk"]),
                "invalid resource diagnostic")
        require(diagnostics["web_ports"] in {"in_use", "not_seen", "unknown"},
                "invalid port diagnostic")
        rejected = subprocess.run([binary, "support-bundle", "--unexpected=" + canary],
                                  env=environment, capture_output=True, timeout=15)
        require(rejected.returncode == 2 and not rejected.stdout, "support export accepted extra arguments")
        require(canary.encode() not in rejected.stderr, "support usage error echoed secret argument")
        require(not database.exists(), "rejected command created a database")
    print("structured support export validated; canaries omitted, arguments rejected, no database created")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        raise SystemExit("usage: smoke_support_bundle.py BINARY")
    main(sys.argv[1])
