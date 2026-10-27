"""Dependency-free SQLite schema validation (python3 -m unittest discover -s db -p 'test_*.py')."""
import pathlib
import sqlite3
import tempfile
import unittest

MIGRATIONS = sorted((pathlib.Path(__file__).parents[1] / "internal" / "database" / "migrations").glob("[0-9][0-9][0-9][0-9]_*.sql"))


class SchemaTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.path = str(pathlib.Path(self.tmp.name) / "panel.db")
        self.con = sqlite3.connect(self.path)
        self.addCleanup(lambda: self.con.close())
        self.con.execute("PRAGMA foreign_keys = ON")
        for version, migration in enumerate(MIGRATIONS, start=1):
            self.con.executescript(migration.read_text())
            self.con.execute(f"PRAGMA user_version={version}")
        self.con.execute("INSERT INTO tenants(id,name) VALUES('a','A'),('b','B')")
        self.con.execute("INSERT INTO identities(id,username) VALUES('u1','alice'),('u2','bob')")
        self.con.execute("INSERT INTO memberships(tenant_id,identity_id,role) VALUES('a','u1','owner'),('b','u2','customer')")
        self.con.commit()

    def test_schema_and_persistent_reopen(self):
        self.assertEqual(self.con.execute("PRAGMA user_version").fetchone()[0], len(MIGRATIONS))
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


    def test_audit_actor_provenance_guards(self):
        self.con.execute(
            "INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,actor_ref,action,subject_id,request_id) "
            "VALUES('human','a','u1','user',NULL,'job.created','job-1','request-1')"
        )
        self.con.execute(
            "INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,actor_ref,action,subject_id,request_id) "
            "VALUES('worker','a',NULL,'worker','worker-1','job.leased','job-1','request-2')"
        )
        self.con.execute(
            "INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,actor_ref,action,subject_id,request_id) "
            "VALUES('reconciler','a',NULL,'system','lease-reconciler','job.lease_expired','job-1','request-3')"
        )
        self.assertEqual(
            self.con.execute("SELECT actor_kind,actor_ref,actor_identity_id FROM audit_events WHERE id='worker'").fetchone(),
            ('worker', 'worker-1', None),
        )
        for values in [
            ('spoof-human', 'a', None, 'user', None, 'job.created', 'job-1', 'request-4'),
            ('spoof-worker', 'a', 'u1', 'worker', 'worker-1', 'job.leased', 'job-1', 'request-5'),
            ('spoof-system', 'a', None, 'system', None, 'job.failed', 'job-1', 'request-6'),
        ]:
            with self.assertRaises(sqlite3.IntegrityError):
                self.con.execute(
                    "INSERT INTO audit_events(id,tenant_id,actor_identity_id,actor_kind,actor_ref,action,subject_id,request_id) VALUES(?,?,?,?,?,?,?,?)",
                    values,
                )
        with self.assertRaises(sqlite3.IntegrityError):
            self.con.execute("UPDATE audit_events SET actor_kind='worker' WHERE id='human'")


if __name__ == '__main__':
    unittest.main()
