package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"time"
)

// TOTPReplayGuard checks a caller-supplied, already enrolled TOTP secret and
// consumes the accepted 30-second step atomically in SQLite. The secret is
// NEVER saved by this component. There is no enrollment or login HTTP route.
// A production caller MUST obtain the secret from reviewed encrypted storage
// and provide trusted server time, not a user-supplied timestamp.
type TOTPReplayGuard struct {
	db *sql.DB
}

func NewTOTPReplayGuard(db *sql.DB) (*TOTPReplayGuard, error) {
	if db == nil {
		return nil, errors.New("missing SQLite store")
	}
	return &TOTPReplayGuard{db: db}, nil
}

// acceptedTOTPStep matches a six-digit code against +/- one 30-second step,
// returning the HIGHEST matching step to prevent accidental reuse of a
// colliding code from the same window as an older step.
func acceptedTOTPStep(secret []byte, code string, at time.Time) (int64, bool) {
	if len(secret) < 20 || len(code) != 6 || at.Unix() < 0 {
		return 0, false
	}
	for i := range code {
		if code[i] < '0' || code[i] > '9' {
			return 0, false
		}
	}
	current := at.Unix() / 30
	var accepted int64 = -1
	for _, delta := range []int64{-1, 0, 1} {
		step := current + delta
		if step < 0 {
			continue
		}
		expected, err := TOTPCode(secret, time.Unix(step*30, 0))
		if err != nil {
			return 0, false
		}
		if subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 && step > accepted {
			accepted = step
		}
	}
	return accepted, accepted >= 0
}

// VerifyAndConsume denies an already accepted step or any older step, even
// when a code remains in the allowed clock-skew window. ONE SQL statement
// performs the authorization recheck and conditional upsert atomically, so
// parallel requests or multiple API processes cannot both consume a code.
//
// The replay boundary is per identity, not per tenant. A user with membership
// in two tenants must not reuse one TOTP code to access both. This primitive
// does not create sessions and does not enroll/rotate/store MFA secrets.
func (g *TOTPReplayGuard) VerifyAndConsume(ctx context.Context, tenantID, identityID string, secret []byte, code string, at time.Time) (bool, error) {
	if g == nil || g.db == nil || len(tenantID) < 1 || len(tenantID) > 128 ||
		len(identityID) < 1 || len(identityID) > 128 {
		return false, ErrInvalidSession
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	step, ok := acceptedTOTPStep(secret, code, at)
	if !ok {
		return false, nil
	}
	// SQLite's UPSERT condition and identity-wide primary key fence enforce
	// monotonic consumption without a read-then-write window.
	res, err := g.db.ExecContext(ctx, `INSERT INTO totp_replay_state(identity_id,last_accepted_step)
		SELECT i.id,? FROM identities i
		JOIN memberships m ON m.identity_id=i.id
		JOIN tenants t ON t.id=m.tenant_id
		WHERE i.id=? AND m.tenant_id=? AND i.disabled=0 AND t.status='active'
		ON CONFLICT(identity_id) DO UPDATE SET
			last_accepted_step=excluded.last_accepted_step,
			updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE excluded.last_accepted_step > totp_replay_state.last_accepted_step`,
		step, identityID, tenantID)
	if err != nil {
		return false, err
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return changed == 1, nil
}
