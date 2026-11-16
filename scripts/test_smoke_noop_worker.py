from contextlib import closing
from datetime import datetime, timedelta, timezone
from pathlib import Path
import sqlite3
import socket
import sys
import tempfile
import unittest
from unittest.mock import patch

from smoke_noop_worker import check_port_available, wait_for_lease_expiry


class WorkerPortGuardTests(unittest.TestCase):
    def test_rejects_live_listener(self):
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 0))
            listener.listen()
            with self.assertRaises(RuntimeError):
                check_port_available(listener.getsockname()[1])

    @unittest.skipUnless(sys.platform.startswith("linux"), "Linux listener reuse contract")
    def test_allows_reuse_after_server_closes_connection(self):
        with socket.socket() as listener:
            listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            listener.bind(("127.0.0.1", 0))
            port = listener.getsockname()[1]
            listener.listen()
            with socket.create_connection(("127.0.0.1", port), timeout=1) as client:
                accepted, _ = listener.accept()
                accepted.close()  # Server active-close creates TIME_WAIT.
                self.assertEqual(client.recv(1), b"")
        check_port_available(port)


class WorkerLeaseDeadlineTests(unittest.TestCase):
    def test_already_expired_lease_passes_without_sleep_or_rewrite(self):
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / "panel.db"
            expired = (datetime.now(timezone.utc) - timedelta(seconds=1)).isoformat().replace("+00:00", "Z")
            with closing(sqlite3.connect(path)) as connection:
                with connection:
                    connection.execute("CREATE TABLE jobs(id TEXT PRIMARY KEY,state TEXT,attempt_count INTEGER,lease_until TEXT)")
                    connection.execute("INSERT INTO jobs VALUES(?,?,?,?)", ("expired", "running", 1, expired))

            with patch("smoke_noop_worker.time.sleep") as sleep:
                wait_for_lease_expiry(path, "expired")
            sleep.assert_not_called()

            with closing(sqlite3.connect(path)) as connection:
                row = connection.execute(
                    "SELECT state,attempt_count,lease_until FROM jobs WHERE id=?", ("expired",)
                ).fetchone()
            self.assertEqual(row, ("running", 1, expired))


if __name__ == "__main__":
    unittest.main()
