package auth

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/parmtech/tong-panel/internal/database"
)

func limiterFixture(t *testing.T) (string, *sql.DB, *SQLiteLoginLimiter, *time.Time) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "limiter.db")
	db, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	limiter, err := NewSQLiteLoginLimiter(db, bytes.Repeat([]byte{0x77}, 32))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }
	return path, db, limiter, &now
}

func TestSQLiteLoginLimiterBlocksAccountAcrossSourcesAndResetsWindow(t *testing.T) {
	_, db, limiter, clock := limiterFixture(t)
	ctx := context.Background()
	for i := 0; i < maxAccountAttempts; i++ {
		ip := "192.0.2." + string(rune('1'+i))
		if allowed, err := limiter.Admit(ctx, "tenant-1", "alice", ip); !allowed || err != nil {
			t.Fatalf("attempt %d denied: %v %v", i, allowed, err)
		}
	}
	if allowed, err := limiter.Admit(ctx, "tenant-1", "ALICE", "198.51.100.1"); allowed || err != nil {
		t.Fatalf("case-folded account budget bypassed: %v %v", allowed, err)
	}
	var buckets int
	if err := db.QueryRow("SELECT count(*) FROM login_attempt_buckets").Scan(&buckets); err != nil || buckets != 6 {
		t.Fatalf("unexpected account + source bucket count %d err=%v", buckets, err)
	}
	*clock = clock.Add(loginWindow)
	if allowed, err := limiter.Admit(ctx, "tenant-1", "alice", "192.0.2.1"); !allowed || err != nil {
		t.Fatalf("expired throttle window did not reset: %v %v", allowed, err)
	}
}

func TestSQLiteLoginLimiterSourceLimitAcrossDifferentAccounts(t *testing.T) {
	_, db, limiter, _ := limiterFixture(t)
	for i := 0; i < maxSourceAttempts; i++ {
		username := "user" + string(rune('a'+i))
		if allowed, err := limiter.Admit(context.Background(), "tenant-a", username, "203.0.113.20"); !allowed || err != nil {
			t.Fatalf("source attempt %d refused: %v %v", i, allowed, err)
		}
	}
	if allowed, err := limiter.Admit(context.Background(), "tenant-a", "extra", "203.0.113.20"); allowed || err != nil {
		t.Fatalf("source limit bypassed: %v %v", allowed, err)
	}
	// No raw names or IPs are permitted in the persisted throttle records.
	var raw string
	if err := db.QueryRow("SELECT hex(bucket_sha256) FROM login_attempt_buckets LIMIT 1").Scan(&raw); err != nil ||
		len(raw) != 64 || strings.Contains(raw, "203.0.113") {
		t.Fatalf("privacy-keyed bucket unavailable or leaked peer info: %q %v", raw, err)
	}
}

func TestSQLiteLoginLimiterConcurrentConnectionsShareBudget(t *testing.T) {
	path, db, first, clock := limiterFixture(t)
	secondDB, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer secondDB.Close()
	second, err := NewSQLiteLoginLimiter(secondDB, bytes.Repeat([]byte{0x77}, 32))
	if err != nil {
		t.Fatal(err)
	}
	second.now = func() time.Time { return *clock }
	const tries = 14
	results := make(chan struct {
		allowed bool
		err     error
	}, tries)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < tries; i++ {
		wg.Add(1)
		service := first
		if i%2 != 0 {
			service = second
		}
		go func(l *SQLiteLoginLimiter) {
			defer wg.Done()
			<-start
			allowed, err := l.Admit(context.Background(), "tenant-a", "victim", "192.0.2.90")
			results <- struct {
				allowed bool
				err     error
			}{allowed, err}
		}(service)
	}
	close(start)
	wg.Wait()
	close(results)
	winners := 0
	for outcome := range results {
		if outcome.err != nil {
			t.Fatalf("concurrent login throttling failed: %v", outcome.err)
		}
		if outcome.allowed {
			winners++
		}
	}
	if winners != maxAccountAttempts {
		t.Fatalf("cross-connection throttle oversubscribed: %d winners", winners)
	}
	var maxCount int
	if err := db.QueryRow("SELECT max(attempts) FROM login_attempt_buckets").Scan(&maxCount); err != nil ||
		maxCount != maxAccountAttempts {
		t.Fatalf("incorrect committed throttle count: %d %v", maxCount, err)
	}
}

func TestSQLiteLoginLimiterInvalidSourcesKeysAndClockRegression(t *testing.T) {
	_, db, limiter, clock := limiterFixture(t)
	if _, err := NewSQLiteLoginLimiter(nil, bytes.Repeat([]byte{0x45}, 32)); !errors.Is(err, ErrInvalidLoginLimiter) {
		t.Fatalf("nil limiter DB accepted: %v", err)
	}
	if _, err := NewSQLiteLoginLimiter(db, []byte("tiny")); !errors.Is(err, ErrInvalidLoginLimiter) {
		t.Fatalf("weak HMAC key accepted: %v", err)
	}
	for _, item := range []struct{ tenant, username, source string }{
		{"", "alice", "192.0.2.1"},
		{"tenant", "ab", "192.0.2.1"},
		{"tenant", "alice", "127.0.0.0.0"},
		{"tenant", "alice", "0.0.0.0"},
		{"tenant", "alice", "evil.example"},
		{"tenant", "alice", "1.2.3.4\n"},
	} {
		if ok, err := limiter.Admit(context.Background(), item.tenant, item.username, item.source); ok || !errors.Is(err, ErrInvalidLoginLimiter) {
			t.Fatalf("untrusted source/identity allowed: %+v %v %v", item, ok, err)
		}
	}
	ok, err := limiter.Admit(context.Background(), "tenant", "alice", "192.0.2.1")
	if !ok || err != nil {
		t.Fatal(err)
	}
	*clock = clock.Add(-time.Minute)
	if ok, err := limiter.Admit(context.Background(), "tenant", "alice", "192.0.2.1"); ok || err != nil {
		t.Fatalf("backward clock reset account limit: %v %v", ok, err)
	}
}

func TestSQLiteLoginLimiterSamplesTimeAfterWriterWait(t *testing.T) {
	for _, expired := range []bool{true, false} {
		name := "still within window"
		if expired {
			name = "window expires during writer wait"
		}
		t.Run(name, func(t *testing.T) {
			_, db, limiter, clock := limiterFixture(t)
			if allowed, err := limiter.Admit(context.Background(), "tenant-lock", "alice", "192.0.2.45"); !allowed || err != nil {
				t.Fatalf("seed attempt: %v %v", allowed, err)
			}
			accountKey := limiter.bucket('a', "tenant-lock", "alice")
			for range maxAccountAttempts - 1 {
				if allowed, err := limiter.Admit(context.Background(), "tenant-lock", "alice", "192.0.2.45"); !allowed || err != nil {
					t.Fatalf("seed account budget: %v %v", allowed, err)
				}
			}
			var current atomic.Int64
			current.Store(clock.Unix())
			limiter.now = func() time.Time { return time.Unix(current.Load(), 0).UTC() }
			blocker, err := db.Conn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := blocker.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
				t.Fatal(err)
			}
			waitsBefore := db.Stats().WaitCount
			result := make(chan struct {
				allowed bool
				err     error
			}, 1)
			go func() {
				allowed, admitErr := limiter.Admit(context.Background(), "tenant-lock", "alice", "192.0.2.45")
				result <- struct {
					allowed bool
					err     error
				}{allowed, admitErr}
			}()
			waitForDBPoolWait(t, db, waitsBefore)
			late := clock.Add(loginWindow + time.Second)
			if !expired {
				late = clock.Add(time.Minute)
			}
			current.Store(late.Unix())
			if _, err := blocker.ExecContext(context.Background(), "ROLLBACK"); err != nil {
				t.Fatal(err)
			}
			if err := blocker.Close(); err != nil {
				t.Fatal(err)
			}
			select {
			case got := <-result:
				if got.err != nil || got.allowed != expired {
					t.Fatalf("admission=%t err=%v expired=%t", got.allowed, got.err, expired)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("limiter admission did not finish after releasing the writer")
			}
			var attempts, storedAt int64
			if err := db.QueryRow("SELECT attempts,window_start_unix FROM login_attempt_buckets WHERE bucket_sha256=?", accountKey).Scan(&attempts, &storedAt); err != nil {
				t.Fatal(err)
			}
			if expired && (attempts != 1 || storedAt != late.Unix()) {
				t.Fatalf("expired budget not reset at authoritative time: attempts=%d start=%d want=%d", attempts, storedAt, late.Unix())
			}
			if !expired && attempts != maxAccountAttempts {
				t.Fatalf("active budget changed: attempts=%d want=%d", attempts, maxAccountAttempts)
			}
		})
	}
}

func TestSQLiteLoginLimiterDoesNotExceedGlobalBucketCap(t *testing.T) {
	_, db, limiter, clock := limiterFixture(t)
	if allowed, err := limiter.Admit(context.Background(), "tenant-cap", "existing-user", "192.0.2.222"); !allowed || err != nil {
		t.Fatalf("initial bucket allocation failed: %v %v", allowed, err)
	}
	if _, err := db.Exec(`WITH RECURSIVE seq(n) AS (
		VALUES(1) UNION ALL SELECT n+1 FROM seq WHERE n < ?
	) INSERT INTO login_attempt_buckets(bucket_sha256,attempts,window_start_unix)
	SELECT randomblob(32),1,? FROM seq`, maxLoginBuckets-2, clock.Unix()); err != nil {
		t.Fatal(err)
	}
	var before int
	if err := db.QueryRow("SELECT count(*) FROM login_attempt_buckets").Scan(&before); err != nil {
		t.Fatal(err)
	}
	if before != maxLoginBuckets {
		t.Fatalf("seeded bucket count=%d, want exact cap %d", before, maxLoginBuckets)
	}
	if allowed, err := limiter.Admit(context.Background(), "tenant-cap", "new-user", "192.0.2.223"); allowed || err != nil {
		t.Fatalf("admission at capacity boundary returned %v %v, want closed denial", allowed, err)
	}
	if allowed, err := limiter.Admit(context.Background(), "tenant-cap", "existing-user", "192.0.2.222"); !allowed || err != nil {
		t.Fatalf("existing buckets were denied at capacity: %v %v", allowed, err)
	}
	var after int
	if err := db.QueryRow("SELECT count(*) FROM login_attempt_buckets").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != maxLoginBuckets {
		t.Fatalf("bucket cap exceeded: count=%d, cap=%d", after, maxLoginBuckets)
	}
}
