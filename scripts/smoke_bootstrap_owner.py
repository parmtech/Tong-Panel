#!/usr/bin/env python3
"""Exercise actual local-only tongctl first-owner bootstrap on ephemeral SQLite."""
import os
from pathlib import Path
import sqlite3
import subprocess
import sys
import tempfile

PASSWORD = b"CI-only-high-entropy-example-password-do-not-use\n"


def command(binary, path, enable):
    args = [binary, "bootstrap-owner", "--db", str(path),
            "--username", "test_owner", "--tenant", "CI ephemeral tenant",
            "--password-stdin"]
    env = dict(os.environ)
    if enable:
        env["TONG_ALLOW_LOCAL_OWNER_BOOTSTRAP"] = "yes"
    else:
        env.pop("TONG_ALLOW_LOCAL_OWNER_BOOTSTRAP", None)
    return subprocess.run(args, input=PASSWORD, capture_output=True,
                          check=False, env=env, timeout=30)


def main(binary):
    binary = str(Path(binary).resolve())
    with tempfile.TemporaryDirectory(prefix="tong-bootstrap-lab-") as folder:
        db_file = Path(folder) / "private" / "panel.db"
        disabled = command(binary, db_file, False)
        assert disabled.returncode != 0, "bootstrap did not require explicit opt-in"
        assert not db_file.exists(), "disabled bootstrap created a database"

        first = command(binary, db_file, True)
        assert first.returncode == 0, (first.stderr.decode("utf-8", "replace"))
        assert b"local owner created" in first.stdout
        assert PASSWORD.strip() not in first.stdout + first.stderr
        assert db_file.stat().st_mode & 0o077 == 0, "database is group/world readable"
        assert db_file.parent.stat().st_mode & 0o077 == 0, "database directory not private"

        with sqlite3.connect(str(db_file)) as con:
            assert con.execute("PRAGMA user_version").fetchone()[0] == 7
            (user, phc, role) = con.execute(
                "SELECT i.username,i.password_phc,m.role FROM identities i "
                "JOIN memberships m ON m.identity_id=i.id").fetchone()
            assert user == "test_owner" and role == "owner"
            assert phc.startswith("$argon2id$v=19$"), "no Argon2id credential"
            assert con.execute(
                "SELECT count(*) FROM audit_events "
                "WHERE action='owner.bootstrap' AND actor_kind='user'").fetchone()[0] == 1
        second = command(binary, db_file, True)
        assert second.returncode != 0, "second privileged owner bootstrap succeeded"
        with sqlite3.connect(str(db_file)) as con:
            assert con.execute("SELECT count(*) FROM identities").fetchone()[0] == 1
            assert con.execute("SELECT count(*) FROM audit_events").fetchone()[0] == 1
        print("PASS: real tongctl local owner bootstrap, 0600 SQLite, Argon2id, "
              "audited single use and denial without opt-in")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        raise SystemExit("usage: python3 scripts/smoke_bootstrap_owner.py /path/to/tongctl")
    main(sys.argv[1])
