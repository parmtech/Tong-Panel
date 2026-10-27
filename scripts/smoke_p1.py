#!/usr/bin/env python3
"""Non-destructive P1 daemon smoke test: real process, loopback only.

Run: python3 scripts/smoke_p1.py /path/to/built/tongd
Requires Python 3 stdlib only. Does not install services or modify the host.
"""
import json
import os
from pathlib import Path
import socket
import sqlite3
import subprocess
import sys
import tempfile
import time
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

BASE = "http://127.0.0.1:8090"
TOKEN = "p1-smoke-only-not-a-production-secret-0123456789"


def request(method, path, token=None, data=None, idem=None):
    headers = {"Accept": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    if data is not None:
        headers["Content-Type"] = "application/json"
    if idem:
        headers["Idempotency-Key"] = idem
    req = Request(BASE + path, data=data, headers=headers, method=method)
    try:
        with urlopen(req, timeout=3) as response:
            status, raw, hdrs = response.status, response.read(65536), response.headers
    except HTTPError as exc:
        status, raw, hdrs = exc.code, exc.read(65536), exc.headers
    try:
        body = json.loads(raw) if raw else {}
    except (ValueError, UnicodeDecodeError):
        body = raw.decode("utf-8", "replace")
    return status, body, hdrs


def assert_response(actual, expected, label):
    if actual != expected:
        raise AssertionError(f"{label}: expected HTTP {expected}, got {actual}")


def run_daemon(executable, env, check):
    with subprocess.Popen([executable], env=env, stdout=subprocess.DEVNULL,
                          stderr=subprocess.DEVNULL) as proc:
        try:
            deadline = time.monotonic() + 15
            while time.monotonic() < deadline:
                if proc.poll() is not None:
                    raise AssertionError(f"tongd exited unexpectedly ({proc.returncode})")
                try:
                    status, _, _ = request("GET", "/healthz")
                    if status == 200:
                        break
                except (OSError, URLError):
                    time.sleep(0.1)
            else:
                raise AssertionError("tongd did not answer loopback health check")
            check()
        finally:
            if proc.poll() is None:
                proc.terminate()
                try:
                    proc.wait(timeout=6)
                except subprocess.TimeoutExpired:
                    proc.kill()
                    proc.wait(timeout=6)


def main(executable):
    executable = str(Path(executable).resolve())
    if not Path(executable).is_file():
        raise SystemExit("supply the built tongd executable")
    # The smoke test must not disturb another local daemon occupying port 8090.
    with socket.socket() as sock:
        if sock.connect_ex(("127.0.0.1", 8090)) == 0:
            raise SystemExit("port 8090 already occupied: refusing to test over another daemon")

    with tempfile.TemporaryDirectory(prefix="tong-p1-smoke-") as folder:
        base_env = dict(os.environ)
        base_env.pop("TONG_DEV_TOKEN", None)
        base_env.pop("TONG_DEV_SQLITE_DB", None)
        base_env["TONG_DATA_DIR"] = folder

        def check_anonymous():
            status, body, hdrs = request("GET", "/healthz")
            assert_response(status, 200, "anonymous health")
            assert body.get("status") == "ok"
            assert hdrs.get("X-Content-Type-Options") == "nosniff"
            status, body, _ = request("GET", "/api/v1/version")
            assert_response(status, 200, "version")
            assert body.get("stage") == "experimental-core"
            status, _, hdrs = request("GET", "/")
            assert_response(status, 200, "embedded static UI")
            assert "frame-ancestors 'none'" in hdrs.get("Content-Security-Policy", "")
            status, js, headers = request("GET", "/ui/panel-status.mjs")
            assert_response(status, 200, "embedded panel status module")
            assert "javascript" in headers.get("Content-Type", "")
            assert isinstance(js, str) and "loadPanelStatus" in js
            status, body, _ = request("GET", "/api/v1/me")
            assert_response(status, 503, "unconfigured developer auth must deny")
            assert body["error"]["code"] == "auth_unconfigured"

        run_daemon(executable, base_env, check_anonymous)
        print("PASS: no-credential daemon startup, UI, version, deny-by-default")

        key_env = {**base_env, "TONG_DEV_TOKEN": TOKEN}

        def check_dev_journal():
            status, _, _ = request("GET", "/api/v1/me")
            assert_response(status, 401, "missing bearer")
            status, user, _ = request("GET", "/api/v1/me", TOKEN)
            assert_response(status, 200, "valid developer token")
            assert user.get("UserID") == "local-dev"
            unsafe = b'{"kind":"system.exec"}'
            status, _, _ = request("POST", "/api/v1/jobs", TOKEN, unsafe, "smoke-unsafe-01")
            assert_response(status, 400, "unsupported OS command denied")
            safe = b'{"kind":"validation.noop"}'
            status, job, _ = request("POST", "/api/v1/jobs", TOKEN, safe, "smoke-noop-0001")
            assert_response(status, 202, "no-op queued")
            job_id = job.get("id")
            assert isinstance(job_id, str) and len(job_id) == 32
            status, repeat, _ = request("POST", "/api/v1/jobs", TOKEN, safe, "smoke-noop-0001")
            assert_response(status, 202, "idempotent retry")
            assert repeat.get("id") == job_id
            status, fetched, _ = request("GET", "/api/v1/jobs/" + job_id, TOKEN)
            assert_response(status, 200, "read own queued job")
            assert fetched.get("state") == "queued"
            status, _, _ = request("DELETE", "/api/v1/jobs/" + job_id, TOKEN)
            assert_response(status, 501, "legacy journal cancellation safely disabled")

        run_daemon(executable, key_env, check_dev_journal)
        print("PASS: bearer-only journal, no-op idempotency, forbidden op, no fake cancellation")

        database = str(Path(folder) / "sqlite" / "panel.db")
        sql_env = {**key_env, "TONG_DEV_SQLITE_DB": database}

        def check_sqlite():
            # The daemon performs schema migrations, NEVER seeds default owners.
            with sqlite3.connect(database, timeout=5) as con:
                version = con.execute("PRAGMA user_version").fetchone()[0]
                assert version == 5, f"wrong migration version: {version}"
                assert con.execute("SELECT count(*) FROM identities").fetchone()[0] == 0
                status, _, _ = request("GET", "/api/v1/me", TOKEN)
                assert_response(status, 401, "token with no DB membership must be denied")
                con.executescript("""
                    INSERT INTO tenants(id,name) VALUES('dev-tenant','SMOKE ONLY');
                    INSERT INTO identities(id,username) VALUES('local-dev','smokedev');
                    INSERT INTO memberships(tenant_id,identity_id,role)
                      VALUES('dev-tenant','local-dev','owner');
                """)
            status, _, _ = request("GET", "/api/v1/me", TOKEN)
            assert_response(status, 200, "membership permits dev token")
            status, job, _ = request("POST", "/api/v1/jobs", TOKEN,
                                     b'{"kind":"validation.noop"}', "smoke-sql-noop-01")
            assert_response(status, 202, "transactional SQL no-op enqueue")
            job_id = job.get("id")
            assert isinstance(job_id, str) and len(job_id) == 32
            status, deleted, _ = request("DELETE", "/api/v1/jobs/" + job_id, TOKEN)
            assert_response(status, 200, "audited queued job cancellation")
            assert deleted.get("state") == "cancelled"
            with sqlite3.connect(database, timeout=5) as con:
                actor = con.execute(
                    "SELECT actor_kind,actor_identity_id FROM audit_events "
                    "WHERE subject_id=? AND action='job.cancelled'", (job_id,)
                ).fetchall()
                assert actor == [("user", "local-dev")], f"untrusted cancellation audit: {actor}"
                con.execute("UPDATE identities SET disabled=1 WHERE id='local-dev'")
                con.commit()
            status, _, _ = request("GET", "/api/v1/jobs/" + job_id, TOKEN)
            assert_response(status, 401, "revoked identity must lose HTTP access")

        run_daemon(executable, sql_env, check_sqlite)
        print("PASS: SQLite migrations, no default identity, membership, audit, revocation")
        print("PASS: P1 developer smoke tests. NOT a signed installer or certified VM test.")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        raise SystemExit("usage: python3 scripts/smoke_p1.py /path/to/tongd")
    main(sys.argv[1])
