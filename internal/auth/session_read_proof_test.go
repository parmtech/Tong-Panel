package auth

import (
	"context"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/rbac"
)

func TestObservedExpiryRevokePreservesConcurrentRefresh(t *testing.T) {
	_, db, store := openSessionDB(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 500000000, time.UTC)
	store.now = func() time.Time { return now }
	p := rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Owner}
	token, _, expires, err := store.Issue(p, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	proof, ok := store.ReadProof(token)
	if !ok {
		t.Fatal("missing proof")
	}
	hash := proof.Digest()
	if err := proof.RevokeObservedExpiry(context.Background(), expires.Format(time.RFC3339Nano), now, false); err != nil {
		t.Fatal(err)
	}
	var revoked bool
	if err := db.QueryRow("SELECT revoked_at IS NOT NULL FROM sessions WHERE token_sha256=?", hash[:]).Scan(&revoked); err != nil || revoked {
		t.Fatalf("fresh idle clobbered: %v %v", revoked, err)
	}
	if _, err := db.Exec("UPDATE sessions SET idle_deadline_unix=? WHERE token_sha256=?", now.Unix(), hash[:]); err != nil {
		t.Fatal(err)
	}
	if err := proof.RevokeObservedExpiry(context.Background(), expires.Format(time.RFC3339Nano), now, false); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT revoked_at IS NOT NULL FROM sessions WHERE token_sha256=?", hash[:]).Scan(&revoked); err != nil || !revoked {
		t.Fatalf("idle expiry not revoked: %v %v", revoked, err)
	}
	store.now = func() time.Time { return now.Add(-time.Minute) }
	if _, ok := store.Check(token); ok {
		t.Fatal("idle session revived after clock rollback")
	}
	store.now = func() time.Time { return now }
	token, _, expires, err = store.Issue(p, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	proof, _ = store.ReadProof(token)
	hash = proof.Digest()
	if err := proof.RevokeObservedExpiry(context.Background(), "different", now, true); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT revoked_at IS NOT NULL FROM sessions WHERE token_sha256=?", hash[:]).Scan(&revoked); err != nil || revoked {
		t.Fatalf("changed expiry clobbered: %v %v", revoked, err)
	}
	if err := proof.RevokeObservedExpiry(context.Background(), expires.Format(time.RFC3339Nano), expires, true); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT revoked_at IS NOT NULL FROM sessions WHERE token_sha256=?", hash[:]).Scan(&revoked); err != nil || !revoked {
		t.Fatalf("absolute expiry not revoked: %v %v", revoked, err)
	}
	store.now = func() time.Time { return now }
	if _, ok := store.Check(token); ok {
		t.Fatal("absolute session revived after clock rollback")
	}
}
