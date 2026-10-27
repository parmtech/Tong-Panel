package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"errors"
	"net"
	"strings"
	"time"
)

// SQLiteLoginLimiter is a shared single-node, persistent *admission* gate.
// No auth HTTP endpoint currently calls it. Source must be the actual trusted
// peer IP (never an arbitrary X-Forwarded-For header supplied by a browser).
// Keys are HMACed with a separately provisioned, stable 32-byte secret.
type SQLiteLoginLimiter struct {
	db     *sql.DB
	key    [32]byte
	now    func() time.Time
}

const loginWindow = 15 * time.Minute
const maxAccountAttempts = 5
const maxSourceAttempts = 25
const maxLoginBuckets = 50000

var ErrInvalidLoginLimiter = errors.New("login attempt limiter unavailable")

func NewSQLiteLoginLimiter(db *sql.DB, secret []byte) (*SQLiteLoginLimiter, error) {
	if db == nil || len(secret) != 32 {
		return nil, ErrInvalidLoginLimiter
	}
	var copied [32]byte
	copy(copied[:], secret)
	return &SQLiteLoginLimiter{db: db, key: copied, now: time.Now}, nil
}

func validLimitID(id string, min, max int) bool {
	if len(id) < min || len(id) > max {
		return false
	}
	for _, c := range id {
		if c < 0x21 || c > 0x7e {
			return false
		}
	}
	return true
}

func (l *SQLiteLoginLimiter) bucket(prefix byte, parts ...string) []byte {
	mac := hmac.New(sha256.New, l.key[:])
	_, _ = mac.Write([]byte{prefix})
	for _, v := range parts {
		_, _ = mac.Write([]byte{byte(len(v) >> 8), byte(len(v))})
		_, _ = mac.Write([]byte(v))
	}
	return mac.Sum(nil)
}

// Admit atomically records an attempt in BOTH account and source buckets.
// It NEVER validates credentials, creates sessions or reveals user existence.
// A separate reviewed HTTPS login handler must invoke this BEFORE Argon2id,
// not after a failed password. Denied attempts do not allocate new buckets.
// Time regression fails closed rather than silently resetting counts.
func (l *SQLiteLoginLimiter) Admit(ctx context.Context, tenant, username, trustedPeerIP string) (bool, error) {
	if l == nil || l.db == nil || !validLimitID(tenant, 1, 128) ||
		!validLimitID(username, 3, 120) {
		return false, ErrInvalidLoginLimiter
	}
	ip := net.ParseIP(trustedPeerIP)
	if ip == nil || ip.IsUnspecified() {
		return false, ErrInvalidLoginLimiter
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	now := l.now().UTC().Unix()
	if now < 0 {
		return false, ErrInvalidLoginLimiter
	}
	keys := [][]byte{
		l.bucket('a', tenant, strings.ToLower(username)),
		l.bucket('s', ip.String()),
	}
	limits := []int{maxAccountAttempts, maxSourceAttempts}
	conn, err := l.db.Conn(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return false, err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	// Prune expired windows with the indexed cutoff; never delete a future
	// record on clock regression. An absolute row cap fails closed under a
	// distributed attack rather than allowing SQLite to grow indefinitely.
	if _, err := conn.ExecContext(ctx, "DELETE FROM login_attempt_buckets WHERE window_start_unix <= ?", now-int64(loginWindow.Seconds())); err != nil {
		return false, err
	}
	var bucketCount int
	if err := conn.QueryRowContext(ctx, "SELECT count(*) FROM login_attempt_buckets").Scan(&bucketCount); err != nil {
		return false, err
	}
	if bucketCount > maxLoginBuckets {
		return false, nil
	}
	type current struct {
		attempts int
		start int64
	}
	state := []current{{start: now}, {start: now}}
	for i, key := range keys {
		var stored current
		err := conn.QueryRowContext(ctx,
			"SELECT attempts, window_start_unix FROM login_attempt_buckets WHERE bucket_sha256=?", key).
			Scan(&stored.attempts, &stored.start)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return false, err
		}
		if now < stored.start {
			return false, nil // suspicious clock rollback
		}
		if now-stored.start < int64(loginWindow.Seconds()) {
			state[i] = stored
		}
		if state[i].attempts >= limits[i] {
			return false, nil
		}
	}
	for i, key := range keys {
		_, err := conn.ExecContext(ctx,
			"INSERT INTO login_attempt_buckets(bucket_sha256,attempts,window_start_unix) VALUES(?,?,?) "+
				"ON CONFLICT(bucket_sha256) DO UPDATE SET attempts=excluded.attempts,window_start_unix=excluded.window_start_unix",
			key, state[i].attempts+1, state[i].start)
		if err != nil {
			return false, err
		}
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return false, err
	}
	committed = true
	return true, nil
}
