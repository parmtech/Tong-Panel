package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/rbac"
)

func TestObservedExpiryRevokeHonorsWriterCancellation(t *testing.T) {
	for _, absolute := range []bool{true, false} {
		t.Run(map[bool]string{true: "absolute", false: "idle"}[absolute], func(t *testing.T) {
			for _, deadline := range []bool{true, false} {
				t.Run(map[bool]string{true: "deadline", false: "cancel"}[deadline], func(t *testing.T) {
					path, db, store := openSessionDB(t)
					now := time.Date(2026, 1, 1, 0, 0, 0, 500000000, time.UTC)
					store.now = func() time.Time { return now }
					token, _, expires, err := store.Issue(rbac.Principal{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Owner}, time.Hour)
					if err != nil {
						t.Fatal(err)
					}
					proof, ok := store.ReadProof(token)
					if !ok {
						t.Fatal("missing proof")
					}
					observed := now.Add(sqliteSessionIdle)
					if absolute {
						observed = expires
					}
					peer, err := database.Open(context.Background(), path)
					if err != nil {
						t.Fatal(err)
					}
					defer peer.Close()
					writer, err := peer.Conn(context.Background())
					if err != nil {
						t.Fatal(err)
					}
					defer writer.Close()
					if _, err := writer.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
						t.Fatal(err)
					}
					defer writer.ExecContext(context.Background(), "ROLLBACK")

					ctx, cancel := context.WithCancel(context.Background())
					want := context.Canceled
					if deadline {
						cancel()
						ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
						want = context.DeadlineExceeded
					}
					defer cancel()
					result := make(chan error, 1)
					started := time.Now()
					go func() {
						result <- proof.RevokeObservedExpiry(ctx, expires.Format(time.RFC3339Nano), observed, absolute)
					}()
					waitUntil := time.Now().Add(2 * time.Second)
					for db.Stats().InUse == 0 && time.Now().Before(waitUntil) {
						time.Sleep(time.Millisecond)
					}
					if db.Stats().InUse == 0 {
						t.Fatal("expiry revocation did not acquire a pooled connection")
					}
					if !deadline {
						cancel()
					}
					select {
					case err := <-result:
						if !errors.Is(err, want) {
							t.Fatalf("expiry revocation error=%v, want %v", err, want)
						}
					case <-time.After(2 * time.Second):
						t.Fatal("expiry revocation ignored cancellation during writer wait")
					}
					if elapsed := time.Since(started); elapsed > time.Second {
						t.Fatalf("expiry revocation cancellation took too long: %v", elapsed)
					}
					digest := proof.Digest()
					var revoked bool
					if err := db.QueryRow("SELECT revoked_at IS NOT NULL FROM sessions WHERE token_sha256=?", digest[:]).Scan(&revoked); err != nil || revoked {
						t.Fatalf("canceled expiry revocation changed session: revoked=%t err=%v", revoked, err)
					}
					if got := readBusyTimeout(t, db); got != 5000 {
						t.Fatalf("busy_timeout after canceled revocation=%d, want 5000", got)
					}
					if _, err := writer.ExecContext(context.Background(), "ROLLBACK"); err != nil {
						t.Fatal(err)
					}
					if err := proof.RevokeObservedExpiry(context.Background(), expires.Format(time.RFC3339Nano), observed, absolute); err != nil {
						t.Fatalf("expiry revocation retry after writer release: %v", err)
					}
					var revokedAt string
					if err := db.QueryRow("SELECT revoked_at FROM sessions WHERE token_sha256=?", digest[:]).Scan(&revokedAt); err != nil || revokedAt != observed.Format(time.RFC3339Nano) {
						t.Fatalf("retried revocation lost observation time: revoked_at=%q err=%v", revokedAt, err)
					}
					if got := readBusyTimeout(t, db); got != 5000 {
						t.Fatalf("busy_timeout after successful revocation=%d, want 5000", got)
					}
				})
			}
		})
	}
}

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
