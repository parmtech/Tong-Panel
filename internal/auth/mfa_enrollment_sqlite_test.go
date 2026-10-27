package auth

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
)

func testMFAEnrollment(t *testing.T) (*MFAEnrollmentStore, *sql.DB, *time.Time) {
	t.Helper()
	_, db, _ := openSessionDB(t)
	box := newTestMFABox(t)
	store, err := NewMFAEnrollmentStore(db, box)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	return store, db, &now
}

func TestMFAEnrollmentEncryptedPendingConfirmationAndReplay(t *testing.T) {
	store, db, clock := testMFAEnrollment(t)
	ctx := context.Background()
	secret, err := store.BeginTrusted(ctx, "tenant-a", "user-a")
	if err != nil || len(secret) != 20 {
		t.Fatalf("begin enrollment: %d bytes err=%v", len(secret), err)
	}
	var envelope, status string
	if err := db.QueryRow("SELECT secret_envelope,status FROM mfa_totp_enrollments WHERE identity_id='user-a'").
		Scan(&envelope, &status); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || !strings.HasPrefix(envelope, "v1:key_2026:") || bytes.Contains([]byte(envelope), secret) {
		t.Fatal("plaintext MFA material leaked to SQL or pending state invalid")
	}
	if _, err := store.BeginTrusted(ctx, "tenant-a", "user-a"); !errors.Is(err, ErrMFAEnrollmentUnavailable) {
		t.Fatalf("double enrollment accepted: %v", err)
	}
	if ok, err := store.VerifyActive(ctx, "tenant-a", "user-a", "123456"); ok || err != nil {
		t.Fatalf("pending enrollment accepted as active: %v %v", ok, err)
	}
	if ok, err := store.ConfirmTrusted(ctx, "tenant-a", "user-a", "nototp"); ok || err != nil {
		t.Fatalf("invalid code activated pending MFA: %v %v", ok, err)
	}
	code, _ := TOTPCode(secret, *clock)
	ok, err := store.ConfirmTrusted(ctx, "tenant-a", "user-a", code)
	if err != nil || !ok {
		t.Fatalf("valid confirmation: %v %v", ok, err)
	}
	if ok, err := store.ConfirmTrusted(ctx, "tenant-a", "user-a", code); ok || err != nil {
		t.Fatalf("confirmation replay allowed: %v %v", ok, err)
	}
	if ok, err := store.VerifyActive(ctx, "tenant-a", "user-a", code); ok || err != nil {
		t.Fatalf("reused enrollment code accepted for login: %v %v", ok, err)
	}
	*clock = clock.Add(30 * time.Second)
	nextCode, _ := TOTPCode(secret, *clock)
	if ok, err := store.VerifyActive(ctx, "tenant-a", "user-a", nextCode); !ok || err != nil {
		t.Fatalf("next active MFA step failed: %v %v", ok, err)
	}
	if ok, err := store.VerifyActive(ctx, "tenant-a", "user-a", nextCode); ok || err != nil {
		t.Fatalf("reused active code succeeded: %v %v", ok, err)
	}
	var replayCount int
	if err := db.QueryRow("SELECT count(*) FROM totp_replay_state WHERE identity_id='user-a'").Scan(&replayCount); err != nil || replayCount != 1 {
		t.Fatalf("missing identity replay fence: %d %v", replayCount, err)
	}
}

func TestMFAEnrollmentRefusesSuspendedRevokedExpiredOrTampered(t *testing.T) {
	ctx := context.Background()
	store, db, clock := testMFAEnrollment(t)
	secret, err := store.BeginTrusted(ctx, "tenant-a", "user-a")
	if err != nil {
		t.Fatal(err)
	}
	code, _ := TOTPCode(secret, *clock)
	if _, err := db.Exec("UPDATE tenants SET status='suspended' WHERE id='tenant-a'"); err != nil {
		t.Fatal(err)
	}
	if ok, err := store.ConfirmTrusted(ctx, "tenant-a", "user-a", code); ok || err != nil {
		t.Fatalf("suspended tenant confirmed: %v %v", ok, err)
	}
	if _, err := db.Exec("UPDATE tenants SET status='active' WHERE id='tenant-a'; UPDATE identities SET disabled=1 WHERE id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if ok, err := store.ConfirmTrusted(ctx, "tenant-a", "user-a", code); ok || err != nil {
		t.Fatalf("disabled identity confirmed: %v %v", ok, err)
	}
	if _, err := db.Exec("UPDATE identities SET disabled=0 WHERE id='user-a'"); err != nil {
		t.Fatal(err)
	}
	*clock = clock.Add(11 * time.Minute)
	nextCode, _ := TOTPCode(secret, *clock)
	if ok, err := store.ConfirmTrusted(ctx, "tenant-a", "user-a", nextCode); ok || err != nil {
		t.Fatalf("expired enrollment confirmed: %v %v", ok, err)
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM totp_replay_state").Scan(&n); err != nil || n != 0 {
		t.Fatalf("denied enrollments consumed codes: %d %v", n, err)
	}
	// Tampering must never reveal the original seed or activate.
	if _, err := db.Exec("UPDATE mfa_totp_enrollments SET pending_expires_at='2099-12-31T00:00:00Z',secret_envelope='v1:key_2026:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA' WHERE identity_id='user-a'"); err != nil {
		t.Fatal(err)
	}
	if ok, err := store.ConfirmTrusted(ctx, "tenant-a", "user-a", nextCode); ok || err == nil {
		t.Fatalf("tampered seed activated: %v %v", ok, err)
	}
	if err := db.QueryRow("SELECT count(*) FROM totp_replay_state").Scan(&n); err != nil || n != 0 {
		t.Fatalf("tamper wrote replay state: %d %v", n, err)
	}
}

func TestMFAEnrollmentNoDefaultKeyOrUntrustedUserCreation(t *testing.T) {
	if _, err := NewMFAEnrollmentStore(nil, newTestMFABox(t)); err == nil {
		t.Fatal("accepted missing db")
	}
	_, db, _ := openSessionDB(t)
	if _, err := NewMFAEnrollmentStore(db, nil); err == nil {
		t.Fatal("accepted missing encryption key")
	}
	store, _ := NewMFAEnrollmentStore(db, newTestMFABox(t))
	for _, tc := range []struct{ tenant, identity string }{
		{"tenant-a", "unknown"}, {"tenant-b", "user-a"}, {"", "user-a"},
		{"tenant-a", ""}, {"tenant-a", "user-a\nother"},
	} {
		secret, err := store.BeginTrusted(context.Background(), tc.tenant, tc.identity)
		if err == nil || len(secret) != 0 {
			t.Fatalf("untrusted identity was enrolled: %+v err=%v", tc, err)
		}
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM mfa_totp_enrollments").Scan(&n); err != nil || n != 0 {
		t.Fatalf("untrusted requests created MFA seeds: %d %v", n, err)
	}
}
