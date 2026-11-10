package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
	"github.com/parmtech/tong-panel/internal/rbac"
	"golang.org/x/crypto/argon2"
)

var changedPasswordFixture = []byte("independent replacement password fixture")

type passwordChangeState struct {
	phc        string
	sessions   []string
	replay     int64
	audits     int
	enrollment string
	recovery   []string
}

func passwordChangeSnapshot(t *testing.T, db *sql.DB) passwordChangeState {
	t.Helper()
	var state passwordChangeState
	if err := db.QueryRow("SELECT password_phc FROM identities WHERE id='user-a'").Scan(&state.phc); err != nil {
		t.Fatal(err)
	}
	rows, err := db.Query("SELECT id||':'||quote(revoked_at) FROM sessions ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		state.sessions = append(state.sessions, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT last_accepted_step FROM totp_replay_state WHERE identity_id='user-a'").Scan(&state.replay); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM audit_events").Scan(&state.audits); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT quote(secret_envelope)||':'||quote(status)||':'||quote(pending_expires_at)||':'||quote(activated_at) FROM mfa_totp_enrollments WHERE identity_id='user-a'").Scan(&state.enrollment); err != nil {
		t.Fatal(err)
	}
	rows, err = db.Query(`SELECT 'sets:'||quote(tenant_id)||':'||quote(identity_id)||':'||quote(enrollment_sha256)||':'||quote(window_started_unix)||':'||quote(attempts) FROM mfa_recovery_sets WHERE identity_id='user-a'
		UNION ALL SELECT 'codes:'||quote(tenant_id)||':'||quote(identity_id)||':'||quote(code_sha256) FROM mfa_recovery_codes WHERE identity_id='user-a' ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		state.recovery = append(state.recovery, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	return state
}

func issuePasswordChangeSessions(t *testing.T, db *sql.DB, sessions *SQLiteSessionStore) (sameIdentity []string, other string) {
	t.Helper()
	if _, err := db.Exec("INSERT INTO memberships(tenant_id,identity_id,role) VALUES('tenant-b','user-a','customer')"); err != nil {
		t.Fatal(err)
	}
	for _, p := range []rbac.Principal{{TenantID: "tenant-a", UserID: "user-a", Role: rbac.Owner}, {TenantID: "tenant-b", UserID: "user-a", Role: rbac.Customer}} {
		token, _, _, err := sessions.Issue(p, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		sameIdentity = append(sameIdentity, token)
	}
	other, _, _, err := sessions.Issue(rbac.Principal{TenantID: "tenant-b", UserID: "user-b", Role: rbac.Customer}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return sameIdentity, other
}

func assertPasswordChangeDenied(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ErrLoginDenied) || err.Error() != ErrLoginDenied.Error() {
		t.Fatal("password change did not return generic authentication denial")
	}
}

func TestInternalPasswordChangePersistsCredentialAndRevokesAllIdentitySessions(t *testing.T) {
	service, db, sessions, old, clock, seed := setupTrustedLogin(t)
	tokens, other := issuePasswordChangeSessions(t, db, sessions)
	recovery, err := NewSQLiteRecoveryStore(db, bytes.Repeat([]byte{0x39}, 32))
	if err != nil {
		t.Fatal(err)
	}
	recovery.now = func() time.Time { return *clock }
	if codes, err := recovery.IssueTrusted(context.Background(), "tenant-a", "user-a"); err != nil || len(codes) != recoveryCodeCount {
		t.Fatal("could not seed synthetic recovery metadata")
	}
	before := passwordChangeSnapshot(t, db)
	code, _ := TOTPCode(seed, *clock)
	if err := service.ChangePasswordTrusted(context.Background(), "tenant-a", "alice", "192.0.2.71", old, changedPasswordFixture, code); err != nil {
		t.Fatal("valid password change rejected:", err)
	}
	after := passwordChangeSnapshot(t, db)
	if after.phc == before.phc || !VerifyPassword(changedPasswordFixture, after.phc) || VerifyPassword(old, after.phc) || PasswordHashNeedsRehash(after.phc) {
		t.Fatal("password change did not persist current-profile replacement verifier")
	}
	if len(before.recovery) != recoveryCodeCount+1 || !reflect.DeepEqual(before.recovery, after.recovery) || before.enrollment != after.enrollment {
		t.Fatal("password change modified MFA or recovery material")
	}
	for _, token := range tokens {
		if _, ok := sessions.Check(token); ok {
			t.Fatal("old identity session survived password change")
		}
	}
	if _, ok := sessions.Check(other); !ok {
		t.Fatal("password change revoked another identity session")
	}
	var revoked, total int
	if err := db.QueryRow("SELECT count(*),sum(revoked_at IS NOT NULL) FROM sessions WHERE identity_id='user-a'").Scan(&total, &revoked); err != nil || total != 2 || revoked != 2 {
		t.Fatal("password change failed identity-wide durable revocation")
	}
	step, ok := acceptedTOTPStep(seed, code, *clock)
	if !ok || after.replay != step || after.audits != before.audits+1 {
		t.Fatal("password change did not commit exactly one replay/audit transition")
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='auth.password_changed' AND tenant_id='tenant-a' AND actor_kind='user' AND actor_identity_id='user-a' AND actor_ref IS NULL").Scan(&count); err != nil || count != 1 {
		t.Fatal("password change audit attribution missing")
	}
	var audit string
	if err := db.QueryRow("SELECT action||'|'||subject_id||'|'||request_id FROM audit_events WHERE action='auth.password_changed'").Scan(&audit); err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{string(old), string(changedPasswordFixture), code, before.phc, after.phc} {
		if strings.Contains(audit, secret) {
			t.Fatal("password change audit contains credential material")
		}
	}
	assertPasswordChangeDenied(t, service.ChangePasswordTrusted(context.Background(), "tenant-a", "alice", "192.0.2.72", old, []byte("another independently chosen password"), code))
	if final := passwordChangeSnapshot(t, db); !reflect.DeepEqual(final, after) {
		t.Fatal("old proof retry changed password/session/replay/audit state")
	}
	assertPasswordChangeDenied(t, service.ChangePasswordTrusted(context.Background(), "tenant-a", "alice", "192.0.2.78", changedPasswordFixture, []byte("another independently chosen password"), code))
	if final := passwordChangeSnapshot(t, db); !reflect.DeepEqual(final, after) {
		t.Fatal("replayed MFA with current password changed security state")
	}
	var path, schema string
	var sequence int
	if err := db.QueryRow("PRAGMA database_list").Scan(&sequence, &schema, &path); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if persisted := passwordChangeSnapshot(t, reopened); !reflect.DeepEqual(persisted, after) {
		t.Fatal("password change did not persist across reopen")
	}
	reopenedSessions, err := NewSQLiteSessionStore(reopened)
	if err != nil {
		t.Fatal(err)
	}
	reopenedSessions.now = func() time.Time { return *clock }
	for _, token := range tokens {
		if _, ok := reopenedSessions.Check(token); ok {
			t.Fatal("identity session revocation did not survive reopen")
		}
	}
	if _, ok := reopenedSessions.Check(other); !ok {
		t.Fatal("other identity session did not survive reopen")
	}
}

func TestInternalPasswordChangeDenialsDoNotMutateSecurityState(t *testing.T) {
	for _, name := range []string{"wrong password", "wrong MFA", "new password too short", "new password too long", "unknown identity", "wrong tenant", "disabled", "suspended", "removed membership", "capacity", "hash capacity", "canceled"} {
		t.Run(name, func(t *testing.T) {
			service, db, sessions, old, clock, seed := setupTrustedLogin(t)
			issuePasswordChangeSessions(t, db, sessions)
			current, next := old, changedPasswordFixture
			tenant, user := "tenant-a", "alice"
			code, _ := TOTPCode(seed, *clock)
			ctx := context.Background()
			verifyCalls, hashCalls := 0, 0
			verify, hash := service.budget.verify, service.budget.hash
			service.budget.verify = func(password []byte, phc string) bool { verifyCalls++; return verify(password, phc) }
			service.budget.hash = func(password []byte) (string, error) { hashCalls++; return hash(password) }
			switch name {
			case "wrong password":
				current = []byte("incorrect current credential")
			case "wrong MFA":
				code = "not-a-totp"
			case "new password too short":
				next = []byte("short")
			case "new password too long":
				next = bytes.Repeat([]byte{'x'}, maxPasswordBytes+1)
			case "unknown identity":
				user = "unknown-user"
			case "wrong tenant":
				tenant = "missing-tenant"
			case "disabled":
				if _, err := db.Exec("UPDATE identities SET disabled=1 WHERE id='user-a'"); err != nil {
					t.Fatal(err)
				}
			case "suspended":
				if _, err := db.Exec("UPDATE tenants SET status='suspended' WHERE id='tenant-a'"); err != nil {
					t.Fatal(err)
				}
			case "removed membership":
				if _, err := db.Exec("DELETE FROM sessions WHERE tenant_id='tenant-a' AND identity_id='user-a';DELETE FROM memberships WHERE tenant_id='tenant-a' AND identity_id='user-a'"); err != nil {
					t.Fatal(err)
				}
			case "capacity":
				for range cap(service.budget.slots) {
					service.budget.slots <- struct{}{}
				}
				defer func() {
					for range cap(service.budget.slots) {
						<-service.budget.slots
					}
				}()
			case "hash capacity":
				service.box.aead = &enrollmentInterleaveAEAD{AEAD: service.box.aead, afterOpen: func() {
					for range cap(service.budget.slots) {
						service.budget.slots <- struct{}{}
					}
				}}
				defer func() {
					for range cap(service.budget.slots) {
						<-service.budget.slots
					}
				}()
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			before := passwordChangeSnapshot(t, db)
			err := service.ChangePasswordTrusted(ctx, tenant, user, "192.0.2.73", current, next, code)
			if name == "canceled" {
				if !errors.Is(err, context.Canceled) {
					t.Fatal("password change lost caller cancellation")
				}
			} else {
				assertPasswordChangeDenied(t, err)
			}
			if after := passwordChangeSnapshot(t, db); !reflect.DeepEqual(before, after) {
				t.Fatal("denied password change mutated credential/session/replay/audit state")
			}
			if hashCalls != 0 {
				t.Fatal("denied proof or unavailable capacity invoked new-password KDF")
			}
			if (name == "new password too short" || name == "new password too long" || name == "capacity" || name == "canceled") && verifyCalls != 0 {
				t.Fatal("invalid length/capacity/cancellation invoked old-password KDF")
			}
		})
	}
}

func TestInternalPasswordChangeRevalidatesTOTPAfterConnectionWait(t *testing.T) {
	for _, expired := range []bool{true, false} {
		name := "within window"
		if expired {
			name = "expired during connection wait"
		}
		t.Run(name, func(t *testing.T) {
			service, db, sessions, old, clock, seed := setupTrustedLogin(t)
			issuePasswordChangeSessions(t, db, sessions)
			before := passwordChangeSnapshot(t, db)
			code, err := TOTPCode(seed, *clock)
			if err != nil {
				t.Fatal(err)
			}
			newPHC, err := HashPassword(changedPasswordFixture)
			if err != nil {
				t.Fatal(err)
			}
			service.budget.verify = func(got []byte, _ string) bool { return bytes.Equal(got, old) }
			hashStarted := make(chan struct{})
			releaseHash := make(chan struct{})
			service.budget.hash = func([]byte) (string, error) {
				close(hashStarted)
				<-releaseHash
				return newPHC, nil
			}
			waitsBefore := db.Stats().WaitCount
			result := make(chan error, 1)
			go func() {
				result <- service.ChangePasswordTrusted(context.Background(), "tenant-a", "alice", "192.0.2.81", old, changedPasswordFixture, code)
			}()
			select {
			case <-hashStarted:
			case <-time.After(5 * time.Second):
				t.Fatal("password change did not reach new-verifier hashing")
			}
			blocker, err := db.Conn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			close(releaseHash)
			waitForDBPoolWait(t, db, waitsBefore)
			if expired {
				*clock = rejectedTOTPLater(t, seed, code, *clock)
			} else {
				*clock = clock.Add(10 * time.Second)
			}
			if err := blocker.Close(); err != nil {
				t.Fatal(err)
			}
			select {
			case got := <-result:
				if expired {
					assertPasswordChangeDenied(t, got)
				} else if got != nil {
					t.Fatalf("password change rejected a code still in its accepted window: %v", got)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("password change did not complete after the connection was released")
			}
			after := passwordChangeSnapshot(t, db)
			if expired {
				if !reflect.DeepEqual(before, after) {
					t.Fatal("expired code changed credential, sessions, replay, MFA, recovery, or audit state")
				}
			} else {
				if after.phc != newPHC || !reflect.DeepEqual(before.recovery, after.recovery) || before.enrollment != after.enrollment {
					t.Fatal("valid waited password change did not persist or unexpectedly changed MFA material")
				}
				if after.audits != before.audits+1 {
					t.Fatalf("valid waited password change audit count=%d, want %d", after.audits, before.audits+1)
				}
			}
		})
	}
}

func TestInternalPasswordChangeRechecksSnapshotAfterKDF(t *testing.T) {
	for _, name := range []string{"password", "MFA envelope", "role", "disabled", "suspended", "expired MFA", "canceled", "hash error"} {
		t.Run(name, func(t *testing.T) {
			service, db, sessions, old, clock, seed := setupTrustedLogin(t)
			issuePasswordChangeSessions(t, db, sessions)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			code, _ := TOTPCode(seed, *clock)
			originalHash := service.budget.hash
			var atTransaction passwordChangeState
			service.budget.hash = func(password []byte) (string, error) {
				phc, err := originalHash(password)
				if err != nil {
					return "", err
				}
				var statement string
				switch name {
				case "password":
					statement = "UPDATE identities SET password_phc=password_phc||'x' WHERE id='user-a'"
				case "MFA envelope":
					statement = "UPDATE mfa_totp_enrollments SET secret_envelope=secret_envelope||'x' WHERE identity_id='user-a'"
				case "role":
					statement = "UPDATE memberships SET role='admin' WHERE tenant_id='tenant-a' AND identity_id='user-a'"
				case "disabled":
					statement = "UPDATE identities SET disabled=1 WHERE id='user-a'"
				case "suspended":
					statement = "UPDATE tenants SET status='suspended' WHERE id='tenant-a'"
				case "canceled":
					cancel()
				case "expired MFA":
					later := clock.Add(2 * time.Minute)
					// Select a genuinely rejected future window; six-digit TOTP
					// collisions must not make this fixture depend on chance.
					for range 10 {
						if _, accepted := acceptedTOTPStep(seed, code, later); !accepted {
							break
						}
						later = later.Add(30 * time.Second)
					}
					if _, accepted := acceptedTOTPStep(seed, code, later); accepted {
						t.Fatal("could not select expired fixture code window")
					}
					*clock = later
				}
				if statement != "" {
					if _, err := db.Exec(statement); err != nil {
						t.Fatal(err)
					}
				}
				atTransaction = passwordChangeSnapshot(t, db)
				if name == "hash error" {
					return "", errors.New("synthetic hashing failure must stay private")
				}
				return phc, nil
			}
			err := service.ChangePasswordTrusted(ctx, "tenant-a", "alice", "192.0.2.74", old, changedPasswordFixture, code)
			if name == "canceled" {
				if !errors.Is(err, context.Canceled) {
					t.Fatal("post-KDF cancellation lost")
				}
			} else if name == "hash error" {
				if err != ErrPasswordChangeUnavailable {
					t.Fatal("hashing failure did not return static operational error")
				}
			} else {
				assertPasswordChangeDenied(t, err)
			}
			if after := passwordChangeSnapshot(t, db); !reflect.DeepEqual(atTransaction, after) {
				t.Fatal("changed credential/enrollment/authority snapshot committed password change")
			}
		})
	}
}

func TestInternalPasswordChangeAcceptsVerifiedShortLegacyOldPassword(t *testing.T) {
	service, db, _, _, clock, seed := setupTrustedLogin(t)
	legacy := []byte("short")
	salt := []byte("0123456789abcdef")
	hash := argon2.IDKey(legacy, salt, passwordHashTime, passwordHashMemoryKiB, passwordHashThreads, passwordHashLength)
	phc := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", passwordHashMemoryKiB, passwordHashTime, passwordHashThreads, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash))
	if _, err := db.Exec("UPDATE identities SET password_phc=? WHERE id='user-a'", phc); err != nil {
		t.Fatal(err)
	}
	code, _ := TOTPCode(seed, *clock)
	if err := service.ChangePasswordTrusted(context.Background(), "tenant-a", "alice", "192.0.2.79", legacy, changedPasswordFixture, code); err != nil {
		t.Fatal("verified short legacy password could not be changed")
	}
	state := passwordChangeSnapshot(t, db)
	if !VerifyPassword(changedPasswordFixture, state.phc) || VerifyPassword(legacy, state.phc) {
		t.Fatal("legacy change did not persist replacement verifier")
	}
	if _, err := HashPassword(legacy); err != ErrPasswordTooShort {
		t.Fatal("password change weakened new-password minimum")
	}
}

func TestInternalPasswordChangeAuditFailureRollsBackAllSecurityState(t *testing.T) {
	service, db, sessions, old, clock, seed := setupTrustedLogin(t)
	issuePasswordChangeSessions(t, db, sessions)
	if _, err := db.Exec("CREATE TRIGGER reject_password_change BEFORE INSERT ON audit_events WHEN NEW.action='auth.password_changed' BEGIN SELECT RAISE(ABORT,'synthetic audit failure'); END"); err != nil {
		t.Fatal(err)
	}
	before := passwordChangeSnapshot(t, db)
	code, _ := TOTPCode(seed, *clock)
	if err := service.ChangePasswordTrusted(context.Background(), "tenant-a", "alice", "192.0.2.75", old, changedPasswordFixture, code); err != ErrPasswordChangeUnavailable {
		t.Fatal("audit failure did not return static operational error")
	}
	if after := passwordChangeSnapshot(t, db); !reflect.DeepEqual(before, after) {
		t.Fatal("audit failure did not roll back password/session/replay state")
	}
	if _, err := db.Exec("DROP TRIGGER reject_password_change"); err != nil {
		t.Fatal(err)
	}
	if err := service.ChangePasswordTrusted(context.Background(), "tenant-a", "alice", "192.0.2.76", old, changedPasswordFixture, code); err != nil {
		t.Fatal("audit failure burned reusable proof")
	}
}

func TestInternalPasswordChangeConcurrentOldProofHasOneWinner(t *testing.T) {
	service, db, sessions, old, clock, seed := setupTrustedLogin(t)
	issuePasswordChangeSessions(t, db, sessions)
	originalHash := service.budget.hash
	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	service.budget.hash = func(password []byte) (string, error) {
		phc, err := originalHash(password)
		if err != nil {
			return "", err
		}
		ready <- struct{}{}
		<-release
		return phc, nil
	}
	code, _ := TOTPCode(seed, *clock)
	results := make(chan error, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for range 2 {
		go func() {
			results <- service.ChangePasswordTrusted(ctx, "tenant-a", "alice", "192.0.2.77", old, changedPasswordFixture, code)
		}()
	}
	for range 2 {
		select {
		case <-ready:
		case <-ctx.Done():
			t.Fatal("concurrent proofs did not reach post-KDF barrier")
		}
	}
	once.Do(func() { close(release) })
	winners := 0
	for range 2 {
		select {
		case err := <-results:
			if err == nil {
				winners++
			} else {
				assertPasswordChangeDenied(t, err)
			}
		case <-ctx.Done():
			t.Fatal("concurrent password changes did not complete")
		}
	}
	if winners != 1 {
		t.Fatal("concurrent old proof did not have exactly one winner")
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action='auth.password_changed'").Scan(&n); err != nil || n != 1 {
		t.Fatal("concurrent password changes duplicated audit")
	}
}
