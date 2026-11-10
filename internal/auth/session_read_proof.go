package auth

import (
	"context"
	"database/sql"
	"time"
)

// SessionReadProof is an internal session identifier, not a substitute for
// authentication. Database reads must recheck revocation and expiry.
type SessionReadProof struct {
	store  *SQLiteSessionStore
	digest [32]byte
}

// ReadProof derives the stored digest; callers must authenticate the cookie
// separately before passing it to a guarded read.
func (s *SQLiteSessionStore) ReadProof(token string) (SessionReadProof, bool) {
	digest, ok := sessionKey(token)
	if !ok || s == nil || s.db == nil {
		return SessionReadProof{}, false
	}
	return SessionReadProof{store: s, digest: digest}, true
}

// MatchesDatabase requires the exact authoritative connection pool, not just
// matching file names or independently replicated session rows.
func (p SessionReadProof) MatchesDatabase(db *sql.DB) bool {
	return p.store != nil && db != nil && p.store.db == db
}
func (p SessionReadProof) Digest() [32]byte { return p.digest }
func (p SessionReadProof) Now() time.Time   { return p.store.now().UTC() }

// RevokeObservedExpiry makes a timeout observed by a guarded read permanent.
// A concurrent idle extension or expiry change must not be overwritten.
func (p SessionReadProof) RevokeObservedExpiry(ctx context.Context, expiry string, now time.Time, absolute bool) error {
	if p.store == nil || p.store.db == nil {
		return ErrInvalidSession
	}
	if absolute {
		_, err := p.store.db.ExecContext(ctx, `UPDATE sessions SET revoked_at=? WHERE token_sha256=? AND revoked_at IS NULL AND expires_at=?`, now.UTC().Format(time.RFC3339Nano), p.digest[:], expiry)
		return err
	}
	_, err := p.store.db.ExecContext(ctx, `UPDATE sessions SET revoked_at=? WHERE token_sha256=? AND revoked_at IS NULL AND idle_deadline_unix<=?`, now.UTC().Format(time.RFC3339Nano), p.digest[:], now.Unix())
	return err
}
