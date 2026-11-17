package auth

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
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

func TestMFABeginStartsPendingLifetimeAfterConnectionWait(t *testing.T) {
	store, db, clock := testMFAEnrollment(t)
	seed, err := store.BeginTrusted(context.Background(), "tenant-a", "user-a")
	if err != nil {
		t.Fatal(err)
	}
	defer clear(seed)
	expiredAt := clock.Add(-time.Second).Format(time.RFC3339Nano)
	if _, err := db.Exec("UPDATE mfa_totp_enrollments SET pending_expires_at=? WHERE identity_id='user-a'", expiredAt); err != nil {
		t.Fatal(err)
	}
	var current atomic.Int64
	current.Store(clock.Unix())
	store.now = func() time.Time { return time.Unix(current.Load(), 0).UTC() }
	db.SetMaxOpenConns(1)
	blocker, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	waitsBefore := db.Stats().WaitCount
	result := make(chan error, 1)
	go func() {
		replacement, beginErr := store.BeginTrusted(context.Background(), "tenant-a", "user-a")
		clear(replacement)
		result <- beginErr
	}()
	waitForDBPoolWait(t, db, waitsBefore)
	late := clock.Add(time.Minute)
	current.Store(late.Unix())
	if err := blocker.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("expired pending enrollment was not replaced: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("enrollment start did not finish after connection release")
	}
	var expiry string
	if err := db.QueryRow("SELECT pending_expires_at FROM mfa_totp_enrollments WHERE identity_id='user-a'").Scan(&expiry); err != nil {
		t.Fatal(err)
	}
	parsed, err := time.Parse(time.RFC3339Nano, expiry)
	if err != nil || !parsed.Equal(late.Add(10*time.Minute)) {
		t.Fatalf("pending expiry=%q err=%v, want %s", expiry, err, late.Add(10*time.Minute))
	}
}

func TestMFABeginHonorsCancellationDuringWriterWait(t *testing.T) {
	store, db, _ := testMFAEnrollment(t)
	var seq int
	var schema, path string
	if err := db.QueryRow("PRAGMA database_list").Scan(&seq, &schema, &path); err != nil {
		t.Fatal(err)
	}
	peerDB, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer peerDB.Close()
	writer, err := peerDB.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if _, err := writer.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	defer writer.ExecContext(context.Background(), "ROLLBACK")

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		secret, beginErr := store.BeginTrusted(ctx, "tenant-a", "user-a")
		clear(secret)
		result <- beginErr
	}()
	deadline := time.Now().Add(2 * time.Second)
	for db.Stats().InUse == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if db.Stats().InUse == 0 {
		cancel()
		t.Fatal("MFA enrollment did not acquire its connection while waiting for the writer")
	}
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("enrollment error=%v, want context cancellation", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("MFA enrollment did not stop after cancellation")
	}
	var pending int
	if err := db.QueryRow("SELECT count(*) FROM mfa_totp_enrollments WHERE identity_id='user-a'").Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if pending != 0 {
		t.Fatalf("canceled enrollment created pending state: count=%d", pending)
	}
	var busyTimeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil || busyTimeout != 5000 {
		t.Fatalf("busy_timeout=%d err=%v, want restored 5000ms", busyTimeout, err)
	}
}

func lockMFAWriterOnPeer(t *testing.T, db *sql.DB) {
	t.Helper()
	var seq int
	var schema, path string
	if err := db.QueryRow("PRAGMA database_list").Scan(&seq, &schema, &path); err != nil {
		t.Fatal(err)
	}
	peerDB, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	writer, err := peerDB.Conn(context.Background())
	if err != nil {
		peerDB.Close()
		t.Fatal(err)
	}
	if _, err := writer.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		writer.Close()
		peerDB.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = writer.ExecContext(context.Background(), "ROLLBACK")
		_ = writer.Close()
		_ = peerDB.Close()
	})
}

func cancelMFAWriteDuringLock(t *testing.T, db *sql.DB, write func(context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- write(ctx) }()
	deadline := time.Now().Add(2 * time.Second)
	for db.Stats().InUse == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if db.Stats().InUse == 0 {
		t.Fatal("MFA operation did not acquire a connection while waiting for the writer")
	}
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("MFA operation error=%v, want context cancellation", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("MFA operation did not stop after cancellation")
	}
	var busyTimeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil || busyTimeout != 5000 {
		t.Fatalf("busy_timeout=%d err=%v, want restored 5000ms", busyTimeout, err)
	}
}

func TestMFAConfirmHonorsCancellationDuringWriterWait(t *testing.T) {
	store, db, clock := testMFAEnrollment(t)
	secret, err := store.BeginTrusted(context.Background(), "tenant-a", "user-a")
	if err != nil {
		t.Fatal(err)
	}
	defer clear(secret)
	code, err := TOTPCode(secret, *clock)
	if err != nil {
		t.Fatal(err)
	}
	lockMFAWriterOnPeer(t, db)
	cancelMFAWriteDuringLock(t, db, func(ctx context.Context) error {
		_, err := store.ConfirmTrusted(ctx, "tenant-a", "user-a", code)
		return err
	})
	var status string
	if err := db.QueryRow("SELECT status FROM mfa_totp_enrollments WHERE identity_id='user-a'").Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "pending" {
		t.Fatalf("canceled confirmation changed enrollment status to %q", status)
	}
	var replay int
	if err := db.QueryRow("SELECT count(*) FROM totp_replay_state WHERE identity_id='user-a'").Scan(&replay); err != nil || replay != 0 {
		t.Fatalf("canceled confirmation changed replay rows: %d err=%v", replay, err)
	}
}

func TestMFAVerifyActiveHonorsCancellationDuringWriterWait(t *testing.T) {
	store, db, clock := testMFAEnrollment(t)
	secret, err := store.BeginTrusted(context.Background(), "tenant-a", "user-a")
	if err != nil {
		t.Fatal(err)
	}
	defer clear(secret)
	first, err := TOTPCode(secret, *clock)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := store.ConfirmTrusted(context.Background(), "tenant-a", "user-a", first); !ok || err != nil {
		t.Fatalf("confirm enrollment: ok=%t err=%v", ok, err)
	}
	*clock = clock.Add(30 * time.Second)
	next, err := TOTPCode(secret, *clock)
	if err != nil {
		t.Fatal(err)
	}
	var before int64
	if err := db.QueryRow("SELECT last_accepted_step FROM totp_replay_state WHERE identity_id='user-a'").Scan(&before); err != nil {
		t.Fatal(err)
	}
	lockMFAWriterOnPeer(t, db)
	cancelMFAWriteDuringLock(t, db, func(ctx context.Context) error {
		_, err := store.VerifyActive(ctx, "tenant-a", "user-a", next)
		return err
	})
	var after int64
	if err := db.QueryRow("SELECT last_accepted_step FROM totp_replay_state WHERE identity_id='user-a'").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("canceled active verification advanced replay step from %d to %d", before, after)
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
