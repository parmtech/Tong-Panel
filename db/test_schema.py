"""Dependency-free SQLite schema validation (python3 -m unittest discover -s db -p 'test_*.py')."""
import pathlib
import sqlite3
import tempfile
import unittest

MIGRATION = (pathlib.Path(__file__).parent / "migrations" / "0001_initial.sql").read_text()


class SchemaTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.path = str(pathlib.Path(self.tmp.name) / "panel.db")
        self.con = sqlite3.connect(self.path)
        self.addCleanup(lambda: self.con.close())
        self.con.execute("PRAGMA foreign_keys = ON")
        self.con.executescript(MIGRATION)
        self.con.execute("INSERT INTO tenants(id,name) VALUES('a','A'),('b','B')")
        self.con.execute("INSERT INTO identities(id,username) VALUES('u1','alice'),('u2','bob')")
        self.con.execute("INSERT INTO memberships(tenant_id,identity_id,role) VALUES('a','u1','owner'),('b','u2','customer')")
        self.con.commit()

    def test_schema_and_persistent_reopen(self):
        self.assertEqual(self.con.execute("PRAGMA user_version").fetchone()[0], 1)
        self.con.close()
        self.con = sqlite3.connect(self.path)
        self.con.execute("PRAGMA foreign_keys=ON")
        self.assertEqual(self.con.execute("SELECT count(*) FROM tenants").fetchone()[0], 2)

    def test_session_membership_enforced(self):
        sql = "INSERT INTO sessions(id,tenant_id,identity_id,token_sha256,csrf_sha256,expires_at) VALUES(?,?,?,?,?,?)"
        self.con.execute(sql, ('s1','a','u1',bytes(32), bytes([1])*32,'2026-10-27T00:00:00Z'))
        with self.assertRaises(sqlite3.IntegrityError):
            self.con.execute(sql, ('s2','b','u1',bytes([2])*32,bytes([3])*32,'2026-10-27T00:00:00Z'))
        with self.assertRaises(sqlite3.IntegrityError):
            self.con.execute(sql, ('s3','a','u1','plaintext',bytes(32),'2026-10-27T00:00:00Z'))

    def test_idempotency_is_tenant_scoped(self):
        def ins(id_, tenant, actor, key):
            self.con.execute("INSERT INTO jobs(id,tenant_id,requested_by,kind,idempotency_key) VALUES(?,?,?,?,?)",
                             (id_, tenant, actor, 'validation.noop', key))
        ins('j1','a','u1','request-0001')
        with self.assertRaises(sqlite3.IntegrityError):
            ins('j2','a','u1','request-0001')
        ins('j3','b','u2','request-0001')
        with self.assertRaises(sqlite3.IntegrityError):
            ins('j4','a','u2','request-2222')
        self.assertEqual(self.con.execute("SELECT count(*) FROM jobs").fetchone()[0], 2)

    def test_invalid_role_state_and_password_rejected(self):
        with self.assertRaises(sqlite3.IntegrityError):
            self.con.execute("INSERT INTO memberships VALUES('a','u2','root',CURRENT_TIMESTAMP)")
        with self.assertRaises(sqlite3.IntegrityError):
            self.con.execute("UPDATE identities SET password_phc='plaintext' WHERE id='u1'")
        with self.assertRaises(sqlite3.IntegrityError):
            self.con.execute("INSERT INTO jobs(id,tenant_id,requested_by,kind,state,idempotency_key) VALUES('j0','a','u1','validation.noop','exec','test-00001')")


if __name__ == '__main__':
    unittest.main()
