package auth

import (
	"bytes"
	"context"
	"crypto/cipher"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
)

func TestMFAEnrollmentExpiredPendingCanRestart(t *testing.T) {
	store, db, clock := testMFAEnrollment(t)
	ctx := context.Background()
	old, err := store.BeginTrusted(ctx, "tenant-a", "user-a")
	if err != nil {
		t.Fatal(err)
	}
	defer clear(old)
	*clock = clock.Add(10*time.Minute - time.Second)
	if _, err := store.BeginTrusted(ctx, "tenant-a", "user-a"); !errors.Is(err, ErrMFAEnrollmentUnavailable) {
		t.Fatal("live pending enrollment was replaced")
	}
	*clock = clock.Add(time.Second)
	fresh, err := store.BeginTrusted(ctx, "tenant-a", "user-a")
	if err != nil {
		t.Fatal("expired pending enrollment could not restart:", err)
	}
	defer clear(fresh)
	if len(fresh) != 20 || bytes.Equal(old, fresh) {
		t.Fatal("restart did not generate an independent seed")
	}
	var path string
	var sequence int
	var name string
	if err := db.QueryRow("PRAGMA database_list").Scan(&sequence, &name, &path); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	store, err = NewMFAEnrollmentStore(reopened, store.box)
	if err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return *clock }
	var envelope, expiry string
	if err := reopened.QueryRow("SELECT secret_envelope,pending_expires_at FROM mfa_totp_enrollments WHERE identity_id='user-a'").Scan(&envelope, &expiry); err != nil {
		t.Fatal(err)
	}
	opened, err := store.box.Open("user-a", envelope)
	if err != nil || !bytes.Equal(opened, fresh) {
		t.Fatal("replacement did not persist encrypted seed")
	}
	clear(opened)
	if expiry != clock.Add(10*time.Minute).Format(time.RFC3339Nano) {
		t.Fatal("replacement lifetime incorrect")
	}
	oldCode := distinctEnrollmentCode(t, old, fresh, *clock)
	newCode, _ := TOTPCode(fresh, *clock)
	if ok, err := store.ConfirmTrusted(ctx, "tenant-a", "user-a", oldCode); err != nil || ok {
		t.Fatal("old seed activated replacement")
	}
	assertMFAReplayCount(t, reopened, 0)
	if ok, err := store.ConfirmTrusted(ctx, "tenant-a", "user-a", newCode); err != nil || !ok {
		t.Fatal("replacement seed could not activate")
	}
	*clock = clock.Add(20 * time.Minute)
	if _, err := store.BeginTrusted(ctx, "tenant-a", "user-a"); !errors.Is(err, ErrMFAEnrollmentUnavailable) {
		t.Fatal("active enrollment was replaced")
	}
}

func assertMFAReplayCount(t *testing.T, db *sql.DB, want int) {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT count(*) FROM totp_replay_state WHERE identity_id='user-a'").Scan(&n); err != nil || n != want {
		t.Fatal("unexpected enrollment replay state")
	}
}

// The existing AEAD interface provides a deterministic interleaving after
// authenticated decryption and before confirmation's writer transaction.
type enrollmentInterleaveAEAD struct {
	cipher.AEAD
	once      sync.Once
	afterOpen func()
}

func (a *enrollmentInterleaveAEAD) Open(dst, nonce, ciphertext, additional []byte) ([]byte, error) {
	plaintext, err := a.AEAD.Open(dst, nonce, ciphertext, additional)
	if err == nil {
		a.once.Do(a.afterOpen)
	}
	return plaintext, err
}

func distinctEnrollmentCode(t *testing.T, old, fresh []byte, now time.Time) string {
	t.Helper()
	for _, offset := range []time.Duration{0, -30 * time.Second, 30 * time.Second} {
		code, _ := TOTPCode(old, now.Add(offset))
		if _, accepted := acceptedTOTPStep(fresh, code, now); !accepted {
			return code
		}
	}
	t.Fatal("independent seed fixtures have indistinguishable code windows")
	return ""
}

func TestMFAEnrollmentRestartConcurrentSingleWinner(t *testing.T) {
	store, db, clock := testMFAEnrollment(t)
	if _, err := store.BeginTrusted(context.Background(), "tenant-a", "user-a"); err != nil {
		t.Fatal(err)
	}
	*clock = clock.Add(10 * time.Minute)
	start := make(chan struct{})
	results := make(chan error, 8)
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			secret, err := store.BeginTrusted(context.Background(), "tenant-a", "user-a")
			clear(secret)
			results <- err
		}()
	}
	close(start)
	group.Wait()
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		} else if !errors.Is(err, ErrMFAEnrollmentUnavailable) {
			t.Fatal(err)
		}
	}
	if winners != 1 {
		t.Fatal("concurrent restarts did not produce exactly one winner")
	}
	assertMFAReplayCount(t, db, 0)
}

func TestMFAEnrollmentRestartRechecksCurrentAuthorization(t *testing.T) {
	for _, change := range []string{"UPDATE identities SET disabled=1 WHERE id='user-a'", "UPDATE tenants SET status='suspended' WHERE id='tenant-a'", "DELETE FROM memberships WHERE tenant_id='tenant-a' AND identity_id='user-a'"} {
		t.Run(change, func(t *testing.T) {
			store, db, clock := testMFAEnrollment(t)
			if _, err := store.BeginTrusted(context.Background(), "tenant-a", "user-a"); err != nil {
				t.Fatal(err)
			}
			var before string
			if err := db.QueryRow("SELECT secret_envelope FROM mfa_totp_enrollments WHERE identity_id='user-a'").Scan(&before); err != nil {
				t.Fatal(err)
			}
			*clock = clock.Add(10 * time.Minute)
			if _, err := db.Exec(change); err != nil {
				t.Fatal(err)
			}
			if _, err := store.BeginTrusted(context.Background(), "tenant-a", "user-a"); !errors.Is(err, ErrMFAEnrollmentUnavailable) {
				t.Fatal("unauthorized restart accepted")
			}
			var after string
			if err := db.QueryRow("SELECT secret_envelope FROM mfa_totp_enrollments WHERE identity_id='user-a'").Scan(&after); err != nil || before != after {
				t.Fatal("denied restart changed seed")
			}
		})
	}
}

func TestMFAEnrollmentConfirmationRejectsChangedSnapshot(t *testing.T) {
	for _, change := range []string{"envelope", "expiry", "expired restart", "disabled", "suspended", "membership"} {
		t.Run(change, func(t *testing.T) {
			store, db, clock := testMFAEnrollment(t)
			ctx := context.Background()
			seed, err := store.BeginTrusted(ctx, "tenant-a", "user-a")
			if err != nil {
				t.Fatal(err)
			}
			defer clear(seed)
			code, _ := TOTPCode(seed, *clock)
			original := store.box.aead
			store.box.aead = &enrollmentInterleaveAEAD{AEAD: original, afterOpen: func() {
				var err error
				switch change {
				case "envelope":
					var replacement string
					replacement, err = store.box.Seal("user-a", bytes.Repeat([]byte{91}, 20))
					if err == nil {
						_, err = db.Exec("UPDATE mfa_totp_enrollments SET secret_envelope=? WHERE identity_id='user-a'", replacement)
					}
				case "expiry":
					_, err = db.Exec("UPDATE mfa_totp_enrollments SET pending_expires_at=? WHERE identity_id='user-a'", clock.Add(20*time.Minute).Format(time.RFC3339Nano))
				case "expired restart":
					restarter := *store
					later := clock.Add(10 * time.Minute)
					restarter.now = func() time.Time { return later }
					var replacement []byte
					replacement, err = restarter.BeginTrusted(ctx, "tenant-a", "user-a")
					clear(replacement)
				case "disabled":
					_, err = db.Exec("UPDATE identities SET disabled=1 WHERE id='user-a'")
				case "suspended":
					_, err = db.Exec("UPDATE tenants SET status='suspended' WHERE id='tenant-a'")
				case "membership":
					_, err = db.Exec("DELETE FROM memberships WHERE tenant_id='tenant-a' AND identity_id='user-a'")
				}
				if err != nil {
					t.Fatal(err)
				}
			}}
			if ok, err := store.ConfirmTrusted(ctx, "tenant-a", "user-a", code); err != nil || ok {
				t.Fatal("stale enrollment proof was accepted")
			}
			assertMFAReplayCount(t, db, 0)
			var status string
			if err := db.QueryRow("SELECT status FROM mfa_totp_enrollments WHERE identity_id='user-a'").Scan(&status); err != nil || status != "pending" {
				t.Fatal("stale proof activated enrollment")
			}
		})
	}
}

func TestMFAEnrollmentActivationMismatchRollsBackReplay(t *testing.T) {
	for _, mutation := range []string{"secret_envelope=secret_envelope||'x'", "pending_expires_at='2099-01-01T00:00:00Z'"} {
		t.Run(mutation, func(t *testing.T) {
			store, db, clock := testMFAEnrollment(t)
			ctx := context.Background()
			seed, err := store.BeginTrusted(ctx, "tenant-a", "user-a")
			if err != nil {
				t.Fatal(err)
			}
			defer clear(seed)
			if _, err := db.Exec("CREATE TRIGGER change_pending_snapshot AFTER INSERT ON totp_replay_state BEGIN UPDATE mfa_totp_enrollments SET " + mutation + " WHERE identity_id=NEW.identity_id; END"); err != nil {
				t.Fatal(err)
			}
			code, _ := TOTPCode(seed, *clock)
			if ok, err := store.ConfirmTrusted(ctx, "tenant-a", "user-a", code); err != nil || ok {
				t.Fatal("changed activation snapshot was accepted")
			}
			assertMFAReplayCount(t, db, 0)
			if _, err := db.Exec("DROP TRIGGER change_pending_snapshot"); err != nil {
				t.Fatal(err)
			}
			if ok, err := store.ConfirmTrusted(ctx, "tenant-a", "user-a", code); err != nil || !ok {
				t.Fatal("failed activation did not roll back replay and pending snapshot")
			}
		})
	}
}

func TestMFAEnrollmentRestartFailurePreservesPending(t *testing.T) {
	store, db, clock := testMFAEnrollment(t)
	seed, err := store.BeginTrusted(context.Background(), "tenant-a", "user-a")
	if err != nil {
		t.Fatal(err)
	}
	defer clear(seed)
	*clock = clock.Add(10 * time.Minute)
	var envelope, expiry string
	if err := db.QueryRow("SELECT secret_envelope,pending_expires_at FROM mfa_totp_enrollments WHERE identity_id='user-a'").Scan(&envelope, &expiry); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TRIGGER reject_pending_restart BEFORE UPDATE ON mfa_totp_enrollments BEGIN SELECT RAISE(ABORT,'fixture failure'); END"); err != nil {
		t.Fatal(err)
	}
	if fresh, err := store.BeginTrusted(context.Background(), "tenant-a", "user-a"); err == nil || fresh != nil {
		t.Fatal("failed restart delivered seed")
	}
	var currentEnvelope, currentExpiry string
	if err := db.QueryRow("SELECT secret_envelope,pending_expires_at FROM mfa_totp_enrollments WHERE identity_id='user-a'").Scan(&currentEnvelope, &currentExpiry); err != nil || envelope != currentEnvelope || expiry != currentExpiry {
		t.Fatal("failed restart changed pending enrollment")
	}
}
