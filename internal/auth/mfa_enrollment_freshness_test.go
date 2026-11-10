package auth

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func setBarrierMFAClock(store *MFAEnrollmentStore, initial time.Time) (advance func(time.Time), waitSample <-chan struct{}, release func()) {
	var current atomic.Int64
	current.Store(initial.Unix())
	sampled := make(chan struct{}, 1)
	resume := make(chan struct{})
	var calls atomic.Int64
	store.now = func() time.Time {
		sample := time.Unix(current.Load(), 0).UTC()
		if calls.Add(1) == 1 {
			sampled <- struct{}{}
			<-resume
		}
		return sample
	}
	return func(at time.Time) { current.Store(at.Unix()) }, sampled, func() { close(resume) }
}

func TestMFAEnrollmentConfirmationRevalidatesAfterWriterWait(t *testing.T) {
	cases := []struct {
		name       string
		initial    time.Duration
		advance    time.Duration
		kind       string
		wantActive bool
	}{
		{name: "TOTP expires while waiting", initial: 30 * time.Second, kind: "TOTP"},
		{name: "pending enrollment expires while waiting", initial: 9*time.Minute + 30*time.Second, advance: 10*time.Minute + time.Second, kind: "pending"},
		{name: "still valid after wait", initial: 30 * time.Second, advance: 40 * time.Second, kind: "valid", wantActive: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, db, clock := testMFAEnrollment(t)
			seed, err := store.BeginTrusted(context.Background(), "tenant-a", "user-a")
			if err != nil {
				t.Fatal(err)
			}
			defer clear(seed)
			start := *clock
			initialTime := start.Add(tc.initial)
			advance := tc.advance
			if advance == 0 {
				advance = initialTime.Sub(start) + 3*time.Minute
			}
			code, err := TOTPCode(seed, initialTime)
			if err != nil {
				t.Fatal(err)
			}
			late := start.Add(advance)
			if tc.kind == "pending" {
				if _, ok := acceptedTOTPStep(seed, code, late); !ok {
					t.Fatal("pending-expiry case must keep the TOTP valid after the wait")
				}
			} else if tc.kind == "TOTP" {
				late = rejectedTOTPLater(t, seed, code, initialTime)
			}
			advanceClock, sampled, releaseClock := setBarrierMFAClock(store, initialTime)
			waitsBefore := db.Stats().WaitCount
			result := make(chan struct {
				ok  bool
				err error
			}, 1)
			go func() {
				ok, confirmErr := store.ConfirmTrusted(context.Background(), "tenant-a", "user-a", code)
				result <- struct {
					ok  bool
					err error
				}{ok: ok, err: confirmErr}
			}()
			select {
			case <-sampled:
			case <-time.After(3 * time.Second):
				t.Fatal("confirmation did not sample its preflight time")
			}
			blocker, err := db.Conn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := blocker.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
				t.Fatal(err)
			}
			releaseClock()
			waitForDBPoolWait(t, db, waitsBefore)
			advanceClock(late)
			if _, err := blocker.ExecContext(context.Background(), "ROLLBACK"); err != nil {
				t.Fatal(err)
			}
			if err := blocker.Close(); err != nil {
				t.Fatal(err)
			}
			var got struct {
				ok  bool
				err error
			}
			select {
			case got = <-result:
			case <-time.After(5 * time.Second):
				t.Fatal("confirmation did not finish after releasing the writer")
			}
			if got.err != nil || got.ok != tc.wantActive {
				t.Fatalf("confirmation accepted=%t err=%v, want accepted=%t", got.ok, got.err, tc.wantActive)
			}
			if tc.wantActive {
				assertMFAReplayCount(t, db, 1)
				var status string
				if err := db.QueryRow("SELECT status FROM mfa_totp_enrollments WHERE identity_id='user-a'").Scan(&status); err != nil || status != "active" {
					t.Fatalf("valid waited confirmation status=%q err=%v", status, err)
				}
			} else {
				assertMFAReplayCount(t, db, 0)
				var status string
				if err := db.QueryRow("SELECT status FROM mfa_totp_enrollments WHERE identity_id='user-a'").Scan(&status); err != nil || status != "pending" {
					t.Fatalf("denied waited confirmation status=%q err=%v", status, err)
				}
			}
		})
	}
}

func TestMFAActiveVerificationRevalidatesAfterWriterWait(t *testing.T) {
	for _, expired := range []bool{true, false} {
		name := "still valid after wait"
		if expired {
			name = "TOTP expires while waiting"
		}
		t.Run(name, func(t *testing.T) {
			store, db, clock := testMFAEnrollment(t)
			seed, err := store.BeginTrusted(context.Background(), "tenant-a", "user-a")
			if err != nil {
				t.Fatal(err)
			}
			defer clear(seed)
			code, err := TOTPCode(seed, *clock)
			if err != nil {
				t.Fatal(err)
			}
			if ok, err := store.ConfirmTrusted(context.Background(), "tenant-a", "user-a", code); err != nil || !ok {
				t.Fatalf("MFA fixture confirmation: %v %v", ok, err)
			}
			initial := clock.Add(30 * time.Second)
			code, err = TOTPCode(seed, initial)
			if err != nil {
				t.Fatal(err)
			}
			late := initial.Add(10 * time.Second)
			if expired {
				late = rejectedTOTPLater(t, seed, code, initial)
			}
			advanceClock, _, releaseClock := setBarrierMFAClock(store, initial)
			var replayBefore int64
			if err := db.QueryRow("SELECT last_accepted_step FROM totp_replay_state WHERE identity_id='user-a'").Scan(&replayBefore); err != nil {
				t.Fatal(err)
			}
			waitsBefore := db.Stats().WaitCount
			result := make(chan struct {
				ok  bool
				err error
			}, 1)
			go func() {
				ok, verifyErr := store.VerifyActive(context.Background(), "tenant-a", "user-a", code)
				result <- struct {
					ok  bool
					err error
				}{ok: ok, err: verifyErr}
			}()
			// Hold a SQLite writer lock after preflight. The operation must wait
			// on BEGIN IMMEDIATE before it can sample the authoritative time.
			blocker, err := db.Conn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := blocker.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
				t.Fatal(err)
			}
			releaseClock()
			waitForDBPoolWait(t, db, waitsBefore)
			advanceClock(late)
			if _, err := blocker.ExecContext(context.Background(), "ROLLBACK"); err != nil {
				t.Fatal(err)
			}
			if err := blocker.Close(); err != nil {
				t.Fatal(err)
			}
			select {
			case got := <-result:
				if got.err != nil || got.ok == expired {
					t.Fatalf("active verification accepted=%t err=%v expired=%t", got.ok, got.err, expired)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("verification did not finish after releasing the writer")
			}
			var replayAfter int64
			if err := db.QueryRow("SELECT last_accepted_step FROM totp_replay_state WHERE identity_id='user-a'").Scan(&replayAfter); err != nil {
				t.Fatal(err)
			}
			if expired && replayAfter != replayBefore {
				t.Fatalf("expired code advanced replay state from %d to %d", replayBefore, replayAfter)
			}
			if !expired && replayAfter <= replayBefore {
				t.Fatalf("valid code did not advance replay state: before=%d after=%d", replayBefore, replayAfter)
			}
		})
	}
}
