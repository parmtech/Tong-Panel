package auth

import (
	"context"
	"database/sql"
	"path/filepath"
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
