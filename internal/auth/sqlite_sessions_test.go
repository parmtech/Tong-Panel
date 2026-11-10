package auth

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/rbac"
)

func openSessionDB(t *testing.T) (string, *sql.DB, *SQLiteSessionStore) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "panel.db")
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`INSERT INTO tenants(id,name) VALUES('tenant-a','A'),('tenant-b','B');
		INSERT INTO identities(id,username) VALUES('user-a','alice'),('user-b','bob');
		INSERT INTO memberships(tenant_id,identity_id,role) VALUES
		('tenant-a','user-a','owner'),('tenant-b','user-b','customer');`)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLiteSessionStore(db)
	if err != nil {
		t.Fatal(err)
	}
	return path, db, store
}

func TestSQLiteSessionPersistenceRevocationAndMembershipRecheck(t *testing.T) {
	path, db, store := openSessionDB(t)
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	p := rbac.Principal{UserID: "user-a", TenantID: "tenant-a", Role: rbac.Owner}
	token, csrf, expires, err := store.Issue(p, time.Hour)
	if err != nil || token == csrf || expires != now.Add(time.Hour) {
		t.Fatalf("issue failed: expires=%s err=%v", expires, err)
	}
	tokenHash, _ := sessionKey(token)
	csrfHash, _ := sessionKey(csrf)
	var storedToken, storedCSRF []byte
	if err := db.QueryRow("SELECT token_sha256,csrf_sha256 FROM sessions WHERE tenant_id='tenant-a'").Scan(&storedToken, &storedCSRF); err != nil {
		t.Fatal(err)
	}
	if len(storedToken) != 32 || len(storedCSRF) != 32 {
		t.Fatal("session table did not store fixed-size hashes")
	}
	if !equalBytes(storedToken, tokenHash[:]) || !equalBytes(storedCSRF, csrfHash[:]) {
		t.Fatal("stored session values are not the expected token hashes")
	}
	if got, ok := store.Check(token); !ok || got.UserID != p.UserID || !store.CheckCSRF(token, csrf) || store.CheckCSRF(token, token) {
		t.Fatal("valid session or CSRF check failed")
	}
	if _, _, _, err := store.Issue(rbac.Principal{UserID: "user-a", TenantID: "tenant-b", Role: rbac.Owner}, time.Hour); err == nil {
		t.Fatal("cross-tenant membership accepted")
	}
	if _, err := db.Exec("UPDATE tenants SET status='suspended' WHERE id='tenant-a'"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := store.Issue(p, time.Hour); err == nil {
		t.Fatal("session issued for suspended tenant")
	}
	if _, ok := store.Check(token); ok || store.CheckCSRF(token, csrf) {
		t.Fatal("existing session remained valid after tenant suspension")
	}
	if _, err := db.Exec("UPDATE tenants SET status='active' WHERE id='tenant-a'"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE memberships SET role='admin' WHERE tenant_id='tenant-a' AND identity_id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if got, ok := store.Check(token); !ok || got.Role != rbac.Admin {
		t.Fatal("session did not re-read current membership role")
	}
	if _, err := db.Exec("UPDATE identities SET disabled=1 WHERE id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Check(token); ok {
		t.Fatal("disabled identity session remained valid")
	}
	if _, err := db.Exec("UPDATE identities SET disabled=0 WHERE id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	reopened, _ := NewSQLiteSessionStore(db)
	reopened.now = func() time.Time { return now }
	if _, ok := reopened.Check(token); !ok {
		t.Fatal("active session did not survive process-style reopen")
	}
	if err := reopened.Revoke(token); err != nil {
		t.Fatal(err)
	}
	if _, ok := reopened.Check(token); ok {
		t.Fatal("revoked session remained valid")
	}
}

func TestSQLiteSessionCheckSamplesExpiryAfterWriterWait(t *testing.T) {
	cases := []struct {
		name      string
		lifetime  time.Duration
		initial   time.Duration
		advance   time.Duration
		wantValid bool
	}{
		{name: "idle expiry during wait", lifetime: 4 * time.Hour, initial: 29 * time.Minute, advance: 31 * time.Minute},
		{name: "absolute expiry during wait", lifetime: 20 * time.Minute, initial: 19 * time.Minute, advance: 21 * time.Minute},
		{name: "still active after wait", lifetime: time.Hour, initial: 5 * time.Minute, advance: 5*time.Minute + 10*time.Second, wantValid: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, db, store := openSessionDB(t)
			start := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
			store.now = func() time.Time { return start }
			token, csrf, _, err := store.Issue(
				rbac.Principal{UserID: "user-a", TenantID: "tenant-a", Role: rbac.Owner},
				tc.lifetime,
			)
			if err != nil {
				t.Fatal(err)
			}
			var currentUnix atomic.Int64
			currentUnix.Store(start.Add(tc.initial).Unix())
			var writerHeld atomic.Bool
			writerHeld.Store(true)
			sampled := make(chan struct{}, 1)
			releaseClock := make(chan struct{})
			store.now = func() time.Time {
				sample := time.Unix(currentUnix.Load(), 0).UTC()
				select {
				case sampled <- struct{}{}:
				default:
				}
				if writerHeld.Load() {
					<-releaseClock
				}
				return sample
			}

			// A second handle holds SQLite's writer reservation. Reads can proceed
			// in WAL mode, but the session refresh must wait to write.
			lockDB, err := database.Open(context.Background(), path)
			if err != nil {
				t.Fatal(err)
			}
			defer lockDB.Close()
			lockConn, err := lockDB.Conn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer lockConn.Close()
			if _, err := lockConn.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
				t.Fatal(err)
			}

			checked := make(chan bool, 1)
			go func() { _, ok := store.Check(token); checked <- ok }()
			deadline := time.Now().Add(3 * time.Second)
			for db.Stats().InUse == 0 && time.Now().Before(deadline) {
				select {
				case <-sampled:
					goto operationStarted
				default:
				}
				time.Sleep(time.Millisecond)
			}
			if db.Stats().InUse == 0 {
				t.Fatal("session check did not acquire a pooled connection")
			}

			// The fixed path cannot sample until BEGIN IMMEDIATE succeeds. The old
			// read-then-update path samples after SELECT but before waiting to write.
			select {
			case <-sampled:
			case <-time.After(2 * time.Second):
			}
		operationStarted:
			currentUnix.Store(start.Add(tc.advance).Unix())
			writerHeld.Store(false)
			close(releaseClock)
			if _, err := lockConn.ExecContext(context.Background(), "ROLLBACK"); err != nil {
				t.Fatal(err)
			}
			select {
			case ok := <-checked:
				if ok != tc.wantValid {
					t.Fatalf("session validity after writer wait=%t, want %t", ok, tc.wantValid)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("session check did not finish after releasing the writer")
			}
			if !tc.wantValid {
				if store.CheckCSRF(token, csrf) {
					t.Fatal("CSRF check accepted a session expired during writer wait")
				}
				hash, ok := sessionKey(token)
				if !ok {
					t.Fatal("issued session token was not parseable")
				}
				var idleDeadline int64
				var revoked bool
				if err := db.QueryRow("SELECT idle_deadline_unix,revoked_at IS NOT NULL FROM sessions WHERE token_sha256=?", hash[:]).
					Scan(&idleDeadline, &revoked); err != nil {
					t.Fatal(err)
				}
				if idleDeadline != start.Add(min(sqliteSessionIdle, tc.lifetime)).Unix() || !revoked {
					t.Fatalf("expired session refresh/revocation state deadline=%d revoked=%t", idleDeadline, revoked)
				}
			}
		})
	}
}

func TestSQLiteSessionExpiryAndUserRevocation(t *testing.T) {
	_, _, store := openSessionDB(t)
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	p := rbac.Principal{UserID: "user-a", TenantID: "tenant-a", Role: rbac.Owner}
	one, _, _, err := store.Issue(p, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	two, _, _, err := store.Issue(p, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeUser("user-a"); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Check(one); ok {
		t.Fatal("first user session survived user-wide revocation")
	}
	if _, ok := store.Check(two); ok {
		t.Fatal("second user session survived user-wide revocation")
	}
	three, _, _, err := store.Issue(p, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	if _, ok := store.Check(three); ok {
		t.Fatal("expired session remained valid")
	}
	if err := store.Revoke(three); err != nil {
		t.Fatal(err)
	}
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
