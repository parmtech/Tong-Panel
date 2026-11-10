#!/usr/bin/env python3
"""Run the opt-in development no-op worker against disposable SQLite only."""

from contextlib import contextmanager
from datetime import datetime, timedelta, timezone
import json
import os
from pathlib import Path
import secrets
import socket
import sqlite3
import subprocess
import sys
import tempfile
import time
from urllib.error import HTTPError, URLError
from urllib.request import ProxyHandler, Request, build_opener

BASE = "http://127.0.0.1:8090"
TOKEN = secrets.token_hex(32)
HTTP = build_opener(ProxyHandler({}))


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def check_port_available(port=8090):
    # Go's listener also uses SO_REUSEADDR on Linux. Match that behavior so
    # connections left in TIME_WAIT by the preceding smoke are not mistaken
    # for a live daemon. SO_REUSEPORT is deliberately not enabled.
    with socket.socket() as probe:
        probe.settimeout(0.25)
        if probe.connect_ex(("127.0.0.1", port)) == 0:
            raise RuntimeError("port occupied; refusing to test another daemon")
    with socket.socket() as probe:
        probe.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            probe.bind(("127.0.0.1", port))
        except OSError:
            raise RuntimeError("port unavailable; refusing to test another daemon")


def request(method, path, key=None):
    body = b'{"kind":"validation.noop"}' if method == "POST" else None
    headers = {"Authorization": "Bearer " + TOKEN, "Content-Type": "application/json"}
    if key:
        headers["Idempotency-Key"] = key
    try:
        with HTTP.open(Request(BASE + path, body, headers, method=method), timeout=3) as response:
            return response.status, json.loads(response.read(65536))
    except HTTPError as error:
        status = error.code
        error.close()
        return status, {}


def wait_for(check, process, label, timeout=15):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        require(process.poll() is None, "daemon exited while waiting for " + label)
        if check():
            return
        time.sleep(0.05)
    raise RuntimeError("timed out waiting for " + label)


@contextmanager
def daemon(binary, environment, fatal_expected=False):
    with subprocess.Popen([binary], env=environment, stdout=subprocess.DEVNULL,
                          stderr=subprocess.DEVNULL) as process:
        try:
            def ready():
                try:
                    return request("GET", "/healthz")[0] == 200
                except (OSError, URLError):
                    return False
            wait_for(ready, process, "loopback health")
            yield process
        finally:
            if process.poll() is None:
                process.terminate()
                try:
                    process.wait(timeout=7)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=5)
                    raise RuntimeError("daemon did not stop gracefully")
            if not fatal_expected:
                require(process.returncode == 0, "normal daemon shutdown failed")


@contextmanager
def connect(path):
    connection = sqlite3.connect(path, timeout=5)
    try:
        connection.execute("PRAGMA foreign_keys=ON")
        with connection:
            yield connection
    finally:
        connection.close()


def state(path, job):
    with connect(path) as connection:
        return connection.execute("SELECT state,attempt_count FROM jobs WHERE id=?", (job,)).fetchone()


def enqueue(key):
    status, job = request("POST", "/api/v1/jobs", key)
    require(status == 202 and job.get("kind") == "validation.noop" and len(job.get("id", "")) == 32,
            "HTTP enqueue failed")
    return job["id"]


def check_completed(path, job, expected_attempts):
    require(state(path, job) == ("completed", expected_attempts), "unexpected final job state")
    with connect(path) as connection:
        rows = connection.execute("SELECT status,error_code FROM job_attempts WHERE job_id=? ORDER BY started_at,id", (job,)).fetchall()
        require(len(rows) == expected_attempts and sum(row[0] == "completed" for row in rows) == 1,
                "unexpected persisted attempts")
        events = connection.execute("SELECT action,actor_kind,actor_identity_id,actor_ref FROM audit_events WHERE subject_id=?", (job,)).fetchall()
        for action in ["job.leased", "job.started", "job.completed"]:
            matching = [event for event in events if event[0] == action]
            require(len(matching) == (1 if action == "job.completed" else expected_attempts),
                    "missing or duplicate worker lifecycle audit")
            require(all(event[1:] == ("worker", None, "local-dev-noop") for event in matching),
                    "worker audit impersonated requester")
        require(sum(event[0] == "job.created" and event[1:3] == ("user", "local-dev") for event in events) == 1,
                "requester creation audit missing")
        if expected_attempts == 2:
            require(sum(row == ("failed", "lease_expired") for row in rows) == 1,
                    "expired attempt was not reconciled")
            require(sum(event[0] == "job.lease_expired" and event[1:] == ("system", None, "lease-reconciler") for event in events) == 1,
                    "expired lease provenance missing")


def main(binary):
    binary = str(Path(binary).resolve())
    check_port_available()
    with tempfile.TemporaryDirectory(prefix="tong-noop-worker-") as folder:
        root = Path(folder)
        path = root / "private" / "panel.db"
        environment = {key: value for key, value in os.environ.items() if not key.startswith("TONG_")}
        environment.update(TONG_DEV_TOKEN=TOKEN, TONG_DEV_SQLITE_DB=str(path), TONG_DATA_DIR=str(root / "journal"))
        # Invalid opt-in must fail before creating either metadata or a journal.
        canary = "invalid-worker-opt-in-must-not-be-echoed"
        rejected = subprocess.run([binary], env={**environment, "TONG_DEV_RUN_NOOP_WORKER": canary},
                                  capture_output=True, timeout=10)
        require(rejected.returncode != 0 and canary.encode() not in rejected.stdout + rejected.stderr,
                "invalid worker opt-in was accepted or echoed")
        require(not path.exists() and not (root / "journal").exists(), "invalid opt-in created storage")
        no_sql = dict(environment, TONG_DEV_RUN_NOOP_WORKER="yes")
        no_sql.pop("TONG_DEV_SQLITE_DB")
        rejected = subprocess.run([binary], env=no_sql, capture_output=True, timeout=10)
        require(rejected.returncode != 0 and not (root / "journal").exists(),
                "worker was enabled without SQLite development mode")
        with daemon(binary, environment):
            with connect(path) as connection:
                connection.executescript("""INSERT INTO tenants(id,name) VALUES('dev-tenant','Disposable A'),('worker-peer','Disposable B');
                    INSERT INTO identities(id,username) VALUES('local-dev','dev_owner'),('peer-user','peer_owner');
                    INSERT INTO memberships(tenant_id,identity_id,role) VALUES('dev-tenant','local-dev','owner'),('worker-peer','peer-user','owner');""")
            blocked = enqueue("disabled-default-job-0001")
            require(state(path, blocked) == ("queued", 0), "default daemon advanced a job")
            with connect(path) as connection:
                connection.execute("UPDATE memberships SET role='auditor' WHERE tenant_id='dev-tenant' AND identity_id='local-dev'")
        # This peer fixture demonstrates progress while the HTTP requester's job
        # is parked by the current-role check. It is not a second public login.
        peer = "b" * 32
        with connect(path) as connection:
            connection.execute("INSERT INTO jobs(id,tenant_id,requested_by,kind,state,idempotency_key) VALUES(?,'worker-peer','peer-user','validation.noop','queued','eligible-peer-fixture-0001')", (peer,))
        enabled = dict(environment, TONG_DEV_RUN_NOOP_WORKER="yes")
        with daemon(binary, enabled) as process:
            wait_for(lambda: state(path, peer)[0] == "completed", process, "eligible peer job")
            require(state(path, blocked) == ("queued", 0), "role-ineligible job advanced")
            with connect(path) as connection:
                connection.execute("UPDATE memberships SET role='owner' WHERE tenant_id='dev-tenant' AND identity_id='local-dev'")
            successful = enqueue("worker-lifecycle-job-0001")
            wait_for(lambda: state(path, successful)[0] == "completed" and state(path, blocked)[0] == "completed",
                     process, "HTTP jobs completed")
            check_completed(path, successful, 1)
        # Force completion/audit rollback and require the supervisor to stop.
        with connect(path) as connection:
            connection.executescript("""CREATE TRIGGER worker_test_reject_completion BEFORE INSERT ON audit_events
                WHEN NEW.action='job.completed' BEGIN SELECT RAISE(ABORT,'disposable audit fault'); END;""")
        with daemon(binary, enabled, fatal_expected=True) as process:
            failed = enqueue("worker-restart-job-0001")
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                raise RuntimeError("audit failure did not stop daemon")
            require(process.returncode != 0, "audit failure reported successful shutdown")
        require(state(path, failed) == ("running", 1), "failed completion was not rolled back")
        with connect(path) as connection:
            require(connection.execute("SELECT count(*) FROM audit_events WHERE subject_id=? AND action='job.completed'", (failed,)).fetchone()[0] == 0,
                    "failed completion persisted an audit")
            connection.execute("DROP TRIGGER worker_test_reject_completion")
            # Explicitly expire the fixture lease; this is not a wall-clock or
            # physical-host crash test. Recovery does restart the real process.
            expired = (datetime.now(timezone.utc) - timedelta(minutes=1)).isoformat().replace("+00:00", "Z")
            connection.execute("UPDATE jobs SET lease_until=? WHERE id=?", (expired, failed))
        with daemon(binary, enabled) as process:
            wait_for(lambda: state(path, failed)[0] == "completed", process, "expired attempt recovery")
            check_completed(path, failed, 2)
            require(enqueue("worker-restart-job-0001") == failed and state(path, failed) == ("completed", 2),
                    "restart/idempotent retry duplicated work")
    print("PASS: opt-in no-op worker completed HTTP jobs, preserved role denial, audited transitions and recovered an expired attempt after restart")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        raise SystemExit("usage: smoke_noop_worker.py BINARY")
    main(sys.argv[1])
